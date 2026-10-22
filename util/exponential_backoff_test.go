package util

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteWithBackoff_ZeroConfigRunsOperationOnce(t *testing.T) {
	b := NewBackoff(BackoffConfig{})
	var calls atomic.Int32

	err := b.ExecuteWithBackoff(func() error {
		calls.Add(1)
		return errors.New("failed")
	})

	require.Error(t, err)
	assert.Equal(t, int32(1), calls.Load())
}

func TestExecuteWithBackoff_ZeroConfigSuccessRunsOnce(t *testing.T) {
	b := NewBackoff(BackoffConfig{})
	var calls atomic.Int32

	err := b.ExecuteWithBackoff(func() error {
		calls.Add(1)
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, int32(1), calls.Load())
}

func TestExecuteWithBackoff_SuccessOnAttemptNStopsRetrying(t *testing.T) {
	b := NewBackoff(BackoffConfig{MaxRetries: 5})
	b.sleep = func(time.Duration) {}

	var calls atomic.Int32
	err := b.ExecuteWithBackoff(func() error {
		n := calls.Add(1)
		if n < 3 {
			return errors.New("retry")
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, int32(3), calls.Load())
}

func TestExecuteWithBackoff_UnrecoverableErrorStopsImmediately(t *testing.T) {
	b := NewBackoff(BackoffConfig{MaxRetries: 10})
	b.sleep = func(time.Duration) {}

	unrecoverable := pkgerrors.New(fmt.Sprintf("upstream: %s", UnrecoverableErrorMsg))
	var calls atomic.Int32

	err := b.ExecuteWithBackoff(func() error {
		calls.Add(1)
		return unrecoverable
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), UnrecoverableErrorMsg)
	assert.Contains(t, err.Error(), "backoff canceled due to unrecoverable error")
	assert.Equal(t, int32(1), calls.Load())
}

func TestExecuteWithBackoff_NoPanicWithZeroOrTinyDelays(t *testing.T) {
	cases := []BackoffConfig{
		{InitialDelay: 0, MaxDelay: 0, Multiplier: 0, MaxRetries: 3},
		{InitialDelay: time.Nanosecond, MaxDelay: time.Nanosecond, Multiplier: 0.5, MaxRetries: 3},
		{InitialDelay: 0, MaxDelay: time.Second, Multiplier: 2, MaxRetries: 2},
	}
	for _, cfg := range cases {
		cfg := cfg
		t.Run(fmt.Sprintf("%+v", cfg), func(t *testing.T) {
			b := NewBackoff(cfg)
			b.sleep = func(time.Duration) {}
			assert.NotPanics(t, func() {
				_ = b.ExecuteWithBackoff(func() error {
					return errors.New("fail")
				})
			})
		})
	}
}

func TestExecuteWithBackoff_SleepDurationNeverExceedsMaxDelay(t *testing.T) {
	const maxDelay = 100 * time.Millisecond
	b := NewBackoff(BackoffConfig{
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     maxDelay,
		Multiplier:   10,
		MaxRetries:   20,
	})

	var sleeps []time.Duration
	b.sleep = func(d time.Duration) {
		sleeps = append(sleeps, d)
	}

	err := b.ExecuteWithBackoff(func() error {
		return errors.New("fail")
	})
	require.Error(t, err)

	for _, d := range sleeps {
		assert.LessOrEqual(t, d, maxDelay, "sleep duration %v exceeds MaxDelay", d)
	}
}

func TestExecuteWithBackoff_NoSleepAfterFinalAttempt(t *testing.T) {
	const attempts = 4
	b := NewBackoff(BackoffConfig{
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Second,
		Multiplier:   2,
		MaxRetries:   attempts,
	})

	var sleepCount atomic.Int32
	b.sleep = func(time.Duration) {
		sleepCount.Add(1)
	}

	err := b.ExecuteWithBackoff(func() error {
		return errors.New("fail")
	})
	require.Error(t, err)
	assert.Equal(t, int32(attempts-1), sleepCount.Load())
}

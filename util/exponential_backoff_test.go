package util

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteWithBackoff_ZeroRetriesRunsOnce(t *testing.T) {
	b := NewBackoff(BackoffConfig{MaxRetries: 0})
	calls := 0
	err := b.ExecuteWithBackoff(func() error {
		calls++
		return errors.New("fail once")
	})
	require.Error(t, err)
	assert.Equal(t, 1, calls)
	assert.False(t, errors.Is(err, ErrAllRetriesFailed))
}

func TestExecuteWithBackoff_ExhaustedReturnsSentinel(t *testing.T) {
	b := NewBackoff(BackoffConfig{
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Millisecond,
		Multiplier:   1,
		MaxRetries:   2,
	})

	err := b.ExecuteWithBackoff(func() error {
		return errors.New("upstream unavailable")
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAllRetriesFailed)
}

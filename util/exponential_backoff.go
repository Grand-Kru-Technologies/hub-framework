package util

import (
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var UnrecoverableErrorMsg = "error is unrecoverable - backoff cancelled"

type BackoffConfig struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	MaxRetries   int
}

type Backoff struct {
	config BackoffConfig
	sleep  func(time.Duration)
}

func NewBackoff(config BackoffConfig) *Backoff {
	return &Backoff{
		config: config,
		sleep:  time.Sleep,
	}
}

func (b *Backoff) maxAttempts() int {
	if b.config.MaxRetries <= 0 {
		return 1
	}
	return b.config.MaxRetries
}

func (b *Backoff) nextBaseDelay(current time.Duration) time.Duration {
	next := time.Duration(float64(current) * b.config.Multiplier)
	if next > b.config.MaxDelay {
		return b.config.MaxDelay
	}
	return next
}

func (b *Backoff) sleepDuration(base time.Duration) time.Duration {
	delay := base
	if half := delay / 2; half > 0 {
		delay += time.Duration(rand.Int63n(int64(half)))
	}
	if delay > b.config.MaxDelay {
		delay = b.config.MaxDelay
	}
	return delay
}

func (b *Backoff) ExecuteWithBackoff(operation func() error) error {
	attempts := b.maxAttempts()
	baseDelay := b.config.InitialDelay

	var err error
	for i := 0; i < attempts; i++ {
		err = operation()
		if err == nil {
			return nil
		}

		if strings.Contains(err.Error(), UnrecoverableErrorMsg) {
			slog.Warn("unrecoverable error encountered, canceling backoff", "error", err)
			return errors.Wrap(err, "backoff canceled due to unrecoverable error")
		}

		if i == attempts-1 {
			break
		}

		delay := b.sleepDuration(baseDelay)
		slog.Info(
			"backoff retry scheduled",
			"attempt", i+1,
			"max_attempts", attempts,
			"error", err,
			"delay", delay,
		)
		b.sleep(delay)
		baseDelay = b.nextBaseDelay(baseDelay)
	}

	return errors.Wrap(err, "all retry attempts failed")
}

package remoteforward

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QueerGlobal/hub-framework/service/logging"
	"github.com/QueerGlobal/hub-framework/util"
	"github.com/getsentry/sentry-go"
)

// ErrorReporter receives errors that should be reported after handling (for example
// to Sentry). Implementations must not log secrets or personal data from err.
type ErrorReporter interface {
	ReportError(err error, stage string)
	// Flush drains buffered events (for example before process exit). No-op for discard reporters.
	Flush(timeout time.Duration) bool
	// Close flushes pending events; safe to call multiple times.
	Close() error
}

type noopErrorReporter struct{}

func (noopErrorReporter) ReportError(err error, stage string) {}

func (noopErrorReporter) Flush(time.Duration) bool { return true }

func (noopErrorReporter) Close() error { return nil }

// NoopErrorReporter returns an ErrorReporter that discards all reports.
func NoopErrorReporter() ErrorReporter {
	return noopErrorReporter{}
}

var (
	sentryInitOnce            sync.Once
	sentryInitOK              bool
	sentryInitInvocationCount int64
)

func initSentryClient(dsn string) bool {
	sentryInitOnce.Do(func() {
		atomic.AddInt64(&sentryInitInvocationCount, 1)
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              dsn,
			AttachStacktrace: true,
		})
		if err != nil {
			logging.GetLogger().Error().
				Err(err).
				Str("stage", "sentry_init").
				Msg("remote forward Sentry init failed; error reporting disabled")
			sentryInitOK = false
			return
		}
		sentryInitOK = true
	})
	return sentryInitOK
}

// NewErrorReporter returns a Sentry-backed reporter when dsn is non-empty, otherwise a no-op.
func NewErrorReporter(dsn string) ErrorReporter {
	if dsn == "" {
		return NoopErrorReporter()
	}
	if !initSentryClient(dsn) {
		return NoopErrorReporter()
	}
	return &sentryReporter{}
}

type sentryReporter struct{}

func (s *sentryReporter) ReportError(err error, stage string) {
	if err == nil {
		return
	}
	safe := redactErrorForReport(err)
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("stage", stage)
		scope.SetTag("component", "remote_forwarder")
		sentry.CaptureException(safe)
	})
}

func (s *sentryReporter) Flush(timeout time.Duration) bool {
	return sentry.Flush(timeout)
}

func (s *sentryReporter) Close() error {
	sentry.Flush(2 * time.Second)
	return nil
}

// FlushInitializedSentry drains the global Sentry client when it was initialized for remote forwarders.
func FlushInitializedSentry(timeout time.Duration) bool {
	if !sentryInitOK {
		return true
	}
	return sentry.Flush(timeout)
}

// redactErrorForReport returns an error safe to send to external trackers (no upstream bodies).
func redactErrorForReport(err error) error {
	switch e := err.(type) {
	case *UpstreamHTTPError:
		return fmt.Errorf("upstream HTTP %d (body redacted)", e.StatusCode)
	case *NonJSONErrorResponse:
		return fmt.Errorf("upstream HTTP %d non-JSON body (redacted)", e.StatusCode)
	default:
		return errors.New(err.Error())
	}
}

// ReportIfExhausted invokes report when retries are exhausted for a retryable failure.
func ReportIfExhausted(err error, report ErrorReporter) {
	if err == nil || report == nil {
		return
	}
	if errors.Is(err, util.ErrAllRetriesFailed) {
		report.ReportError(err, "forward_exhausted")
	}
}

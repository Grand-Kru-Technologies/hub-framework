package remoteforward

import (
	"errors"
	"fmt"

	"github.com/QueerGlobal/hub-framework/service/logging"
	"github.com/QueerGlobal/hub-framework/util"
	"github.com/getsentry/sentry-go"
)

// ErrorReporter receives errors that should be reported after handling (for example
// to Sentry). Implementations must not log secrets or personal data from err.
type ErrorReporter interface {
	ReportError(err error, stage string)
}

type noopErrorReporter struct{}

func (noopErrorReporter) ReportError(err error, stage string) {}

// NoopErrorReporter returns an ErrorReporter that discards all reports.
func NoopErrorReporter() ErrorReporter {
	return noopErrorReporter{}
}

// NewErrorReporter returns a Sentry-backed reporter when dsn is non-empty, otherwise a no-op.
func NewErrorReporter(dsn string) ErrorReporter {
	if dsn == "" {
		return NoopErrorReporter()
	}
	return newSentryReporter(dsn)
}

type sentryReporter struct{}

func newSentryReporter(dsn string) ErrorReporter {
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		AttachStacktrace: true,
	})
	if err != nil {
		logging.GetLogger().Error().
			Err(err).
			Str("stage", "sentry_init").
			Msg("remote forward Sentry init failed; error reporting disabled")
		return NoopErrorReporter()
	}
	return &sentryReporter{}
}

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

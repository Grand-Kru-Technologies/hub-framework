package remoteforward

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

package remoteforward

import (
	"os"
	"time"

	"github.com/QueerGlobal/hub-framework/service/logging"
	"github.com/rs/zerolog"
)

const defaultClientTimeout = 30 * time.Second

// ServiceOptions holds shared wiring for remote HTTP forwarders.
type ServiceOptions struct {
	ClientTimeout time.Duration
	Logger        zerolog.Logger
	Reporter      ErrorReporter
}

// ComponentLogger returns a structured logger for a forwarder component without side effects.
func ComponentLogger(component string) zerolog.Logger {
	return logging.GetLogger().With().Str("component", component).Logger()
}

// BaseServiceOptions builds logger and timeout defaults without initializing Sentry.
func BaseServiceOptions(clientTimeout time.Duration, logger *zerolog.Logger, component string) ServiceOptions {
	opts := ServiceOptions{
		ClientTimeout: clientTimeout,
		Logger:        ComponentLogger(component),
	}
	if opts.ClientTimeout <= 0 {
		opts.ClientTimeout = defaultClientTimeout
	}
	if logger != nil {
		opts.Logger = *logger
	}
	return opts
}

// ServiceOptionsFromMap builds options from a task-style config map and env.
func ServiceOptionsFromMap(config map[string]interface{}, component string) ServiceOptions {
	opts := BaseServiceOptions(0, nil, component)
	if rawTimeout, ok := config["clientTimeout"].(float64); ok && rawTimeout > 0 {
		opts.ClientTimeout = time.Duration(rawTimeout * float64(time.Second))
	}
	opts.Reporter = NewErrorReporter(ResolveSentryDSN(config))
	return opts
}

// ResolveSentryDSN reads a Sentry DSN from config or SENTRY_DSN when unset in config.
func ResolveSentryDSN(config map[string]interface{}) string {
	if config != nil {
		for _, key := range []string{"sentryDsn", "sentryDSN", "sentry_dsn", "SentryDSN"} {
			if raw, ok := config[key].(string); ok && raw != "" {
				return raw
			}
		}
	}
	return os.Getenv("SENTRY_DSN")
}

// ResolveSentryDSNWithOverride prefers an explicit configured DSN, then SENTRY_DSN.
func ResolveSentryDSNWithOverride(configured string) string {
	if configured != "" {
		return configured
	}
	return os.Getenv("SENTRY_DSN")
}

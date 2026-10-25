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

// ServiceOptionsFromMap builds options from a task-style config map and env.
func ServiceOptionsFromMap(config map[string]interface{}, component string) ServiceOptions {
	opts := ServiceOptions{
		ClientTimeout: defaultClientTimeout,
		Logger:        logging.GetLogger().With().Str("component", component).Logger(),
		Reporter:      NewErrorReporter(ResolveSentryDSN(config)),
	}
	if rawTimeout, ok := config["clientTimeout"].(float64); ok && rawTimeout > 0 {
		opts.ClientTimeout = time.Duration(rawTimeout * float64(time.Second))
	}
	return opts
}

// ResolveSentryDSN reads a Sentry DSN from config or SENTRY_DSN when unset in config.
func ResolveSentryDSN(config map[string]interface{}) string {
	for _, key := range []string{"sentryDsn", "sentryDSN", "sentry_dsn"} {
		if raw, ok := config[key].(string); ok && raw != "" {
			return raw
		}
	}
	return os.Getenv("SENTRY_DSN")
}

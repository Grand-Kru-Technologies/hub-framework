package remotehttpservice

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewForwardingService_InitializesSentryFromConfigDSN(t *testing.T) {
	t.Setenv("SENTRY_DSN", "not-a-valid-dsn-format")
	validDSN := "https://01234567890123456789012345678901@example.com/1"

	svc := NewForwardingService(ForwardingServiceConfig{
		Host:          "http://example.com",
		BackoffConfig: testBackoff(),
		SentryDSN:     validDSN,
	})

	require.NotNil(t, svc.reporter)
	flushable, ok := svc.reporter.(interface {
		Flush(time.Duration) bool
	})
	require.True(t, ok)
	require.True(t, flushable.Flush(10 * time.Millisecond))
}

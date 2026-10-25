package remoteforward

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveSentryDSN_ConfigWinsOverEnv(t *testing.T) {
	t.Setenv("SENTRY_DSN", "https://env@example.com/1")
	fromConfig := map[string]interface{}{
		"sentryDsn": "https://config@example.com/2",
	}
	assert.Equal(t, "https://config@example.com/2", ResolveSentryDSN(fromConfig))
}

func TestResolveSentryDSNWithOverride_ExplicitWinsOverEnv(t *testing.T) {
	t.Setenv("SENTRY_DSN", "https://env@example.com/1")
	assert.Equal(t, "https://cfg@example.com/3", ResolveSentryDSNWithOverride("https://cfg@example.com/3"))
}

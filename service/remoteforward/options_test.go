package remoteforward

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResolveSentryDSN_ConfigWinsOverEnv(t *testing.T) {
	t.Setenv("SENTRY_DSN", "https://env@example.com/1")
	fromConfig := map[string]interface{}{
		"sentryDsn": "https://config@example.com/2",
	}
	assert.Equal(t, "https://config@example.com/2", ResolveSentryDSN(fromConfig))
}

func TestServiceOptionsFromMap_AcceptsIntegerClientTimeout(t *testing.T) {
	opts := ServiceOptionsFromMap(map[string]interface{}{
		"clientTimeout": 15,
	}, "test")
	assert.Equal(t, 15*time.Second, opts.ClientTimeout)
}

func TestResolveSentryDSNWithOverride_ExplicitWinsOverEnv(t *testing.T) {
	t.Setenv("SENTRY_DSN", "https://env@example.com/1")
	assert.Equal(t, "https://cfg@example.com/3", ResolveSentryDSNWithOverride("https://cfg@example.com/3"))
}

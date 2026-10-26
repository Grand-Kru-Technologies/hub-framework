package remoteforward_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/QueerGlobal/hub-framework/core/entity"
	"github.com/QueerGlobal/hub-framework/service/target/remotehttpservice"
	"github.com/QueerGlobal/hub-framework/service/task/remote"
	"github.com/QueerGlobal/hub-framework/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testBackoff() util.BackoffConfig {
	return util.BackoffConfig{
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Millisecond,
		Multiplier:   1,
		MaxRetries:   2,
	}
}

// TestForwardIntegration exercises task and target forwarders against a stub upstream.
func TestForwardIntegration(t *testing.T) {
	t.Run("task forwarder 200 JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"via":"task"}`))
		}))
		t.Cleanup(server.Close)

		task, err := remote.NewForwardingService(map[string]interface{}{
			"host": server.URL,
			"backoff": map[string]interface{}{
				"initialDelay": float64(0.001),
				"maxDelay":     float64(0.001),
				"multiplier":   float64(1),
				"maxRetries":   float64(2),
			},
		})
		require.NoError(t, err)

		req := &entity.HTTPServiceRequest{URL: &url.URL{Path: "/hook"}}
		require.NoError(t, task.Apply(context.Background(), req))
		stored := req.GetResponse()
		require.NotNil(t, stored)
		assert.JSONEq(t, `{"via":"task"}`, string(stored.GetBody()))
	})

	t.Run("target forwarder 400 JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"nope"}`))
		}))
		t.Cleanup(server.Close)

		svc := remotehttpservice.NewForwardingService(remotehttpservice.ForwardingServiceConfig{
			Host:          server.URL,
			BackoffConfig: testBackoff(),
		})

		req := &entity.HTTPServiceRequest{URL: &url.URL{Path: "/hook"}}
		_, err := svc.Apply(req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), util.UnrecoverableErrorMsg)
	})
}

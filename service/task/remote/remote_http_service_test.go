package remote

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/QueerGlobal/hub-framework/core/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApply_StoresResponseBodyOnce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"task":"done"}`))
	}))
	t.Cleanup(server.Close)

	task, err := NewForwardingService(map[string]interface{}{
		"host": server.URL,
		"backoff": map[string]interface{}{
			"initialDelay": float64(0.001),
			"maxDelay":     float64(0.001),
			"multiplier":   float64(1),
			"maxRetries":   float64(2),
		},
	})
	require.NoError(t, err)

	req := &entity.HTTPServiceRequest{URL: &url.URL{Path: "/work"}}
	err = task.Apply(context.Background(), req)
	require.NoError(t, err)

	stored := req.GetResponse()
	require.NotNil(t, stored)
	assert.Equal(t, http.StatusOK, stored.GetResponseMeta().GetStatusCode())
	assert.JSONEq(t, `{"task":"done"}`, string(stored.GetBody()))
}

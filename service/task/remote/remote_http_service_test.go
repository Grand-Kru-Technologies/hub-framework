package remote

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QueerGlobal/hub-framework/core/entity"
	"github.com/QueerGlobal/hub-framework/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTaskConfig(host string, extra map[string]interface{}) map[string]interface{} {
	cfg := map[string]interface{}{
		"host": host,
		"backoff": map[string]interface{}{
			"initialDelay": float64(0.001),
			"maxDelay":     float64(0.001),
			"multiplier":   float64(1),
			"maxRetries":   float64(3),
		},
	}
	for k, v := range extra {
		cfg[k] = v
	}
	return cfg
}

func TestNewForwardingService_RequiresHost(t *testing.T) {
	_, err := NewForwardingService(map[string]interface{}{
		"backoff": map[string]interface{}{
			"maxRetries": 3,
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "host")
}

func TestNewForwardingService_RejectsInvalidForwardHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		host string
	}{
		{name: "ftp", host: "ftp://upstream.example"},
		{name: "bare hostname", host: "upstream.example"},
		{name: "userinfo on invalid scheme", host: "ws://user:leaked-password@upstream.example"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewForwardingService(testTaskConfig(tt.host, nil))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "absolute http or https URL")
			assert.NotContains(t, err.Error(), "leaked-password")
		})
	}
}

func TestNewForwardingService_AcceptsIntegerMaxRetries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	task, err := NewForwardingService(map[string]interface{}{
		"host": server.URL,
		"backoff": map[string]interface{}{
			"initialDelay": 0,
			"maxDelay":     0,
			"multiplier":   1,
			"maxRetries":   3,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, task)
}

func TestApply_StoresResponseBodyOnce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"task":"done"}`))
	}))
	t.Cleanup(server.Close)

	task, err := NewForwardingService(testTaskConfig(server.URL, nil))
	require.NoError(t, err)

	req := &entity.HTTPServiceRequest{URL: &url.URL{Path: "/work"}}
	err = task.Apply(context.Background(), req)
	require.NoError(t, err)

	stored := req.GetResponse()
	require.NotNil(t, stored)
	assert.Equal(t, http.StatusOK, stored.GetResponseMeta().GetStatusCode())
	assert.JSONEq(t, `{"task":"done"}`, string(stored.GetBody()))
}

func TestApply_CancelledContextDoesNotRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(400 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	task, err := NewForwardingService(testTaskConfig(server.URL, map[string]interface{}{
		"clientTimeout": float64(10),
		"backoff": map[string]interface{}{
			"initialDelay": float64(0.001),
			"maxDelay":     float64(0.001),
			"multiplier":   float64(1),
			"maxRetries":   float64(5),
		},
	}))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := &entity.HTTPServiceRequest{URL: &url.URL{Path: "/work"}}
	err = task.Apply(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), util.UnrecoverableErrorMsg)
	assert.Equal(t, int32(1), calls.Load())
}

func TestApply_ClientTimeoutRetries(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)

	task, err := NewForwardingService(testTaskConfig(server.URL, map[string]interface{}{
		"clientTimeout": float64(0.05),
	}))
	require.NoError(t, err)

	req := &entity.HTTPServiceRequest{URL: &url.URL{Path: "/work"}}
	err = task.Apply(context.Background(), req)
	require.Error(t, err)
	assert.ErrorIs(t, err, util.ErrAllRetriesFailed)
	assert.GreaterOrEqual(t, calls.Load(), int32(2))
}

func TestApply_UpstreamJSON400IsUnrecoverable(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad"}`))
	}))
	t.Cleanup(server.Close)

	task, err := NewForwardingService(testTaskConfig(server.URL, nil))
	require.NoError(t, err)

	req := &entity.HTTPServiceRequest{URL: &url.URL{Path: "/work"}}
	err = task.Apply(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), util.UnrecoverableErrorMsg)
	assert.Equal(t, int32(1), calls.Load())
}

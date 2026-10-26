package remotehttpservice

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/QueerGlobal/hub-framework/core/entity"
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

func TestApply_JSONSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"recipe":"ok"}`))
	}))
	t.Cleanup(server.Close)

	svc := NewForwardingService(ForwardingServiceConfig{
		Host:          server.URL,
		PathPrefix:    "/api",
		BackoffConfig: testBackoff(),
	})

	req := &entity.HTTPServiceRequest{URL: &url.URL{Path: "/recipes"}}
	resp, err := svc.Apply(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.GetResponseMeta().GetStatusCode())
	assert.JSONEq(t, `{"recipe":"ok"}`, string(resp.GetBody()))
}

func TestApply_UpstreamJSONErrorIsUnrecoverable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad"}`))
	}))
	t.Cleanup(server.Close)

	svc := NewForwardingService(ForwardingServiceConfig{
		Host:          server.URL,
		BackoffConfig: testBackoff(),
	})

	req := &entity.HTTPServiceRequest{URL: &url.URL{Path: "/x"}}
	_, err := svc.Apply(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), util.UnrecoverableErrorMsg)
}

func TestApply_DoesNotDoubleSlashPath(t *testing.T) {
	var seen string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	svc := NewForwardingService(ForwardingServiceConfig{
		Host:          server.URL,
		PathPrefix:    "/api",
		BackoffConfig: testBackoff(),
	})

	req := &entity.HTTPServiceRequest{URL: &url.URL{Path: "/recipes"}}
	_, err := svc.Apply(req)
	require.NoError(t, err)
	assert.Equal(t, "/api/recipes", seen)
}

func TestApply_UsesContextTimeoutClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	svc := NewForwardingService(ForwardingServiceConfig{
		Host:          server.URL,
		BackoffConfig: testBackoff(),
		ClientTimeout: 50 * time.Millisecond,
	})

	req := &entity.HTTPServiceRequest{URL: &url.URL{Path: "/slow"}}
	_, err := svc.Apply(req)
	require.Error(t, err)
}

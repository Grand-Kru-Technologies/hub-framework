package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var smokeForwardPathVariants = []string{
	"/smokeApp/forward",
	"/smokeapp/forward",
	"/SmokeApp/forward",
	"//smokeApp/forward",
	"/internal/call/smokeApp/forward",
	"/internal/call//smokeApp/forward",
}

func TestRootHandler_BlocksSmokeForwardVariantsWithoutEnv(t *testing.T) {
	t.Setenv("ENABLE_SMOKE_FORWARD", "")

	hubCalled := false
	h := rootHandler{
		hub: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hubCalled = true
			w.WriteHeader(http.StatusOK)
		}),
	}

	for _, path := range smokeForwardPathVariants {
		hubCalled = false
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, nil)
		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code, "path %q", path)
		assert.False(t, hubCalled, "hub must not run for blocked path %q", path)
	}
}

func TestRootHandler_AllowsSmokeForwardVariantsWhenEnabled(t *testing.T) {
	t.Setenv("ENABLE_SMOKE_FORWARD", "true")

	h := rootHandler{
		hub: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}

	for _, path := range smokeForwardPathVariants {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, nil)
		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "path %q", path)
	}
}

func TestRequestTargetsSmokeForward_ParsesNormalizedPaths(t *testing.T) {
	for _, path := range smokeForwardPathVariants {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		require.True(t, requestTargetsSmokeForward(req), "expected smoke forward for %q", path)
	}

	other := httptest.NewRequest(http.MethodPost, "/recipe/create", nil)
	assert.False(t, requestTargetsSmokeForward(other))
}

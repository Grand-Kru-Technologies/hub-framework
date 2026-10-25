package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootHandler_BlocksSmokeAppWithoutEnv(t *testing.T) {
	t.Setenv("ENABLE_SMOKE_FORWARD", "")
	h := rootHandler{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/smokeApp/forward", nil)
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRootHandler_AllowsSmokeAppWhenEnabled(t *testing.T) {
	t.Setenv("ENABLE_SMOKE_FORWARD", "true")
	h := rootHandler{
		hub: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/smokeApp/forward", nil)
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRootHandler_BlocksCaseVariantWithoutEnv(t *testing.T) {
	unset := os.Getenv("ENABLE_SMOKE_FORWARD")
	os.Unsetenv("ENABLE_SMOKE_FORWARD")
	t.Cleanup(func() { os.Setenv("ENABLE_SMOKE_FORWARD", unset) })

	h := rootHandler{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/SmokeApp/forward", nil)
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

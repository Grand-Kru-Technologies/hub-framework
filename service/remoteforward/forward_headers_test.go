package remoteforward

import (
	"net/http"
	"testing"

	"github.com/QueerGlobal/hub-framework/core/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyForwardHeaders_AllowlistOnly(t *testing.T) {
	inbound := &entity.HTTPServiceRequest{
		Header: http.Header{
			"X-Request-Id":  []string{"abc"},
			"Traceparent":   []string{"00-abc-def-01"},
			"Authorization": []string{"Bearer secret"},
			"Cookie":        []string{"session=evil"},
			"X-Api-Key":     []string{"key-secret"},
			"Connection":    []string{"close"},
			"Content-Type":  []string{"application/json"},
			"X-Custom":      []string{"drop-me"},
		},
	}
	outReq, err := http.NewRequest(http.MethodPost, "http://upstream/hook", nil)
	require.NoError(t, err)
	outReq.Header.Set("Content-Type", "application/json")

	ApplyForwardHeaders(outReq, inbound)

	assert.Equal(t, "abc", outReq.Header.Get("X-Request-Id"))
	assert.Equal(t, "00-abc-def-01", outReq.Header.Get("Traceparent"))
	assert.Empty(t, outReq.Header.Get("Authorization"))
	assert.Empty(t, outReq.Header.Get("Cookie"))
	assert.Empty(t, outReq.Header.Get("X-Api-Key"))
	assert.Empty(t, outReq.Header.Get("Connection"))
	assert.Empty(t, outReq.Header.Get("X-Custom"))
	assert.Equal(t, "application/json", outReq.Header.Get("Content-Type"))
}

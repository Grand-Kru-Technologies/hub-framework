package remoteforward

import (
	"net/http"
	"testing"

	"github.com/QueerGlobal/hub-framework/core/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyForwardHeaders_CopiesAllowedSkipsHopByHop(t *testing.T) {
	inbound := &entity.HTTPServiceRequest{
		Header: http.Header{
			"X-Request-Id":  []string{"abc"},
			"Authorization": []string{"Bearer secret"},
			"Connection":    []string{"close"},
			"Content-Type":  []string{"application/json"},
		},
	}
	outReq, err := http.NewRequest(http.MethodPost, "http://upstream/hook", nil)
	require.NoError(t, err)
	outReq.Header.Set("Content-Type", "application/json")

	ApplyForwardHeaders(outReq, inbound)

	assert.Equal(t, "abc", outReq.Header.Get("X-Request-Id"))
	assert.Empty(t, outReq.Header.Get("Authorization"))
	assert.Empty(t, outReq.Header.Get("Connection"))
	assert.Equal(t, "application/json", outReq.Header.Get("Content-Type"))
}

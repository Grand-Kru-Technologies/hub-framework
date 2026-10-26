package remoteforward

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/QueerGlobal/hub-framework/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadAndParseResponse_JSONSuccess(t *testing.T) {
	body := strings.NewReader(`{"ok":true}`)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Body:       io.NopCloser(body),
	}

	parsed, err := ReadAndParseResponse(resp, DefaultLogger())
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Equal(t, http.StatusOK, parsed.GetResponseMeta().GetStatusCode())
	assert.JSONEq(t, `{"ok":true}`, string(parsed.GetBody()))
}

func TestReadAndParseResponse_EmptySuccessBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusNoContent,
		Status:     "204 No Content",
		Body:       io.NopCloser(bytes.NewReader(nil)),
	}

	parsed, err := ReadAndParseResponse(resp, DefaultLogger())
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Empty(t, parsed.GetBody())
}

func TestReadAndParseResponse_Non2xxJSONError(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Status:     "400 Bad Request",
		Body:       io.NopCloser(strings.NewReader(`{"error":"nope"}`)),
	}

	_, err := ReadAndParseResponse(resp, DefaultLogger())
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), util.UnrecoverableErrorMsg))

	var upstream *UpstreamHTTPError
	require.True(t, errors.As(err, &upstream))
	assert.Equal(t, http.StatusBadRequest, upstream.StatusCode)
	assert.Equal(t, "nope", upstream.Payload["error"])
}

func TestReadAndParseResponse_NonJSONErrorBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Status:     "500 Internal Server Error",
		Body:       io.NopCloser(strings.NewReader("<html>fail</html>")),
	}

	_, err := ReadAndParseResponse(resp, DefaultLogger())
	require.Error(t, err)
	assert.NotContains(t, err.Error(), util.UnrecoverableErrorMsg)

	var nonJSON *NonJSONErrorResponse
	require.True(t, errors.As(err, &nonJSON))
	assert.Equal(t, http.StatusInternalServerError, nonJSON.StatusCode)
}

func TestReadAndParseResponse_MalformedJSONSuccess(t *testing.T) {
	malformed := []byte("{\"ok\":")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Body:       io.NopCloser(bytes.NewReader(malformed)),
	}

	_, err := ReadAndParseResponse(resp, DefaultLogger())
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), util.UnrecoverableErrorMsg))
	assert.ErrorIs(t, err, ErrMalformedJSON)
}

package remoteforward

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/QueerGlobal/hub-framework/core/entity"
	"github.com/stretchr/testify/require"
)

func TestJSONPayloadForForward_OmitsHTTPRequestPointers(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "http://hub/smokeApp/forward", strings.NewReader(`{}`))
	require.NoError(t, err)
	svcReq, err := entity.GetRequestFromHttp(req)
	require.NoError(t, err)
	svcReq.SetBody([]byte(`{"name":"x"}`))

	raw, err := JSONPayloadForForward(svcReq)
	require.NoError(t, err)
	require.True(t, json.Valid(raw))

	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Equal(t, "smokeApp", decoded["ApiName"])
	require.NotContains(t, decoded, "OriginalRequest")
}

func TestJSONPayloadForForward_URLString(t *testing.T) {
	svcReq := &entity.HTTPServiceRequest{
		ApiName:      "a",
		ServiceName:  "b",
		Method:       entity.HTTPMethodPOST,
		URL:          &url.URL{Scheme: "http", Host: "hub", Path: "/a/b"},
		InternalPath: "/a/b",
		RequestMeta: entity.RequestMeta{
			Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1,
			ContentLength: 0, Host: "hub", RemoteAddr: "127.0.0.1", RequestURI: "/a/b",
		},
	}
	raw, err := JSONPayloadForForward(svcReq)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"URL":"http://hub/a/b"`)
}

package remoteforward

import (
	"encoding/json"

	"github.com/QueerGlobal/hub-framework/core/entity"
)

// JSONPayloadForForward builds a JSON document safe to POST to remote handlers
// (no http.Request pointers or other non-JSON types).
func JSONPayloadForForward(request entity.ServiceRequest) ([]byte, error) {
	payload := map[string]interface{}{
		"ApiName":      request.GetAPIName(),
		"ServiceName":  request.GetServiceName(),
		"Method":       string(request.GetMethod()),
		"InternalPath": request.GetInternalPath(),
	}

	if u := request.GetURL(); u != nil {
		payload["URL"] = u.String()
	}

	if body := request.GetBody(); len(body) > 0 {
		payload["Body"] = body
	}

	if meta := request.GetRequestMeta(); meta != nil {
		payload["RequestMeta"] = map[string]interface{}{
			"Proto":         meta.GetProto(),
			"ProtoMajor":    meta.GetProtoMajor(),
			"ProtoMinor":    meta.GetProtoMinor(),
			"ContentLength": meta.GetContentLength(),
			"Host":          meta.GetHost(),
			"RemoteAddr":    meta.GetRemoteAddr(),
			"RequestURI":    meta.GetRequestURI(),
		}
		if enc := meta.GetTransferEncoding(); len(enc) > 0 {
			payload["RequestMeta"].(map[string]interface{})["TransferEncoding"] = enc
		}
	}

	return json.Marshal(payload)
}

package remoteforward

import (
	"net/http"
	"strings"

	"github.com/QueerGlobal/hub-framework/core/entity"
)

var forwardHeaderDenylist = map[string]struct{}{
	"Authorization":       {},
	"Connection":          {},
	"Content-Length":      {},
	"Host":                {},
	"Keep-Alive":          {},
	"Proxy-Authenticate":  {},
	"Proxy-Authorization": {},
	"Te":                  {},
	"Trailer":             {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
}

// ApplyForwardHeaders copies inbound request headers onto an outbound forward request.
// Hop-by-hop and credential headers are skipped; Content-Type is left to the caller.
func ApplyForwardHeaders(dst *http.Request, src entity.ServiceRequest) {
	if dst == nil || src == nil {
		return
	}
	for name, values := range src.GetHeader() {
		canonical := http.CanonicalHeaderKey(name)
		if _, denied := forwardHeaderDenylist[canonical]; denied {
			continue
		}
		if strings.EqualFold(canonical, "Content-Type") {
			continue
		}
		for _, value := range values {
			dst.Header.Add(canonical, value)
		}
	}
}

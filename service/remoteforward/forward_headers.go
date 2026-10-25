package remoteforward

import (
	"net/http"

	"github.com/QueerGlobal/hub-framework/core/entity"
)

// Headers copied onto outbound forward requests (allowlist only).
var forwardHeaderAllowlist = map[string]struct{}{
	"Baggage":          {},
	"Traceparent":      {},
	"Tracestate":       {},
	"X-Amzn-Trace-Id":  {},
	"X-Correlation-Id": {},
	"X-Request-Id":     {},
}

// ApplyForwardHeaders copies allowlisted inbound request headers onto an outbound
// forward request. Credentials, cookies, and hop-by-hop headers are never forwarded.
// Content-Type is left to the caller.
func ApplyForwardHeaders(dst *http.Request, src entity.ServiceRequest) {
	if dst == nil || src == nil {
		return
	}
	for name, values := range src.GetHeader() {
		canonical := http.CanonicalHeaderKey(name)
		if _, allowed := forwardHeaderAllowlist[canonical]; !allowed {
			continue
		}
		for _, value := range values {
			dst.Header.Add(canonical, value)
		}
	}
}

package remoteforward

import (
	"fmt"
	"net/url"
	"strings"
)

// JoinForwardURL combines the upstream host, normalized path prefix, and request path.
func JoinForwardURL(host, pathPrefix, requestPath string) string {
	base := strings.TrimSuffix(host, "/")
	trimmedPath := strings.TrimPrefix(requestPath, "/")
	if pathPrefix == "" {
		if trimmedPath == "" {
			return base + "/"
		}
		return base + "/" + trimmedPath
	}
	if trimmedPath == "" {
		return base + pathPrefix
	}
	return base + pathPrefix + trimmedPath
}

// ValidateForwardHost reports whether host is an absolute http or https URL with a hostname.
func ValidateForwardHost(host string) error {
	parsed, err := url.Parse(host)
	if err != nil {
		return invalidForwardHostError(host)
	}
	if !parsed.IsAbs() {
		return invalidForwardHostError(host)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return invalidForwardHostError(host)
	}
	if parsed.Hostname() == "" {
		return invalidForwardHostError(host)
	}
	return nil
}

func invalidForwardHostError(host string) error {
	return fmt.Errorf(
		"HttpService config host must be an absolute http or https URL with a hostname; got %q",
		RedactHostConfigForError(host),
	)
}

// RedactHostConfigForError returns a display-safe form of a configured forward host
// without userinfo, query strings, or fragments.
func RedactHostConfigForError(host string) string {
	parsed, err := url.Parse(host)
	if err != nil {
		return redactHostConfigOpaque(host)
	}
	if parsed.User != nil {
		parsed.User = url.User("[REDACTED]")
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func redactHostConfigOpaque(host string) string {
	if at := strings.LastIndex(host, "@"); at >= 0 {
		return redactHostConfigOpaque(host[at+1:])
	}
	if q := strings.IndexAny(host, "?#"); q >= 0 {
		return host[:q]
	}
	return host
}

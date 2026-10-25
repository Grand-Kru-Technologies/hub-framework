package remoteforward

import "strings"

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
	return base + pathPrefix + trimmedPath
}

package remoteforward

import "strings"

// NormalizePathPrefix ensures pathPrefix starts and ends with a slash when non-empty.
func NormalizePathPrefix(pathPrefix string) string {
	if len(pathPrefix) == 0 {
		return pathPrefix
	}
	if !strings.HasPrefix(pathPrefix, "/") {
		pathPrefix = "/" + pathPrefix
	}
	if !strings.HasSuffix(pathPrefix, "/") {
		pathPrefix = pathPrefix + "/"
	}
	return pathPrefix
}

package common

import "strings"

// ClassicEmbedSPAPaths are classic frontend routes served under the default
// theme (e.g. embedded in an iframe from the new UI).
var ClassicEmbedSPAPaths = []string{
	"/console/task/embed",
}

// IsClassicEmbedSPAPath reports whether path should bootstrap the classic SPA
// while the global theme remains "default".
func IsClassicEmbedSPAPath(path string) bool {
	if path == "" {
		return false
	}
	if path != "/" && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}
	for _, p := range ClassicEmbedSPAPaths {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}

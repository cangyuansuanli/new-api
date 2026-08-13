package imagevendor

import "strings"

// CanonicalAspectRatio returns the public Image API ratio when size carries one.
// Exact pixel sizes stay available to vendors that support custom dimensions.
func CanonicalAspectRatio(size string) string {
	value := strings.TrimSpace(size)
	if value == "" || strings.EqualFold(value, "auto") {
		return ""
	}
	if strings.Contains(value, ":") {
		return value
	}
	switch strings.ToLower(value) {
	case "1024x1024", "2048x2048", "4096x4096":
		return "1:1"
	case "1536x1024":
		return "3:2"
	case "1024x1536":
		return "2:3"
	case "1792x1024", "1920x1080":
		return "16:9"
	case "1024x1792", "1080x1920":
		return "9:16"
	default:
		return ""
	}
}

// CanonicalResolution maps the public quality tier to the resolution labels
// used by Gemini-compatible image upstreams.
func CanonicalResolution(quality string) string {
	switch strings.ToLower(strings.TrimSpace(quality)) {
	case "high", "hd", "4k":
		return "4K"
	case "medium", "2k":
		return "2K"
	case "low", "standard", "1k", "1/2k", "auto":
		return "1K"
	default:
		return ""
	}
}

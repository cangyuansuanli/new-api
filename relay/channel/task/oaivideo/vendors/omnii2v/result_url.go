package omnii2v

import (
	"strings"

	oaivideo "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/shared"
)

// extractOmniI2VVideoURL selects the download URL for OAIREGBox omni i2v tasks.
// Upstream may return task_/content before the link is ready (HTTP 409) while
// vid-/content is already downloadable; prefer the ready link without changing
// global defaultvideo URL precedence for other vendors.
func extractOmniI2VVideoURL(res oaivideo.ResponseTask) string {
	if u := oaivideo.PickAbsoluteVideoURL(res.VideoURLSnake, res.VideoURL); u != "" && isOmniReadyVideoContentURL(u) {
		return u
	}
	return oaivideo.ExtractVideoURL(res)
}

func isOmniReadyVideoContentURL(raw string) bool {
	u := strings.ToLower(strings.TrimSpace(raw))
	return strings.Contains(u, "/v1/videos/vid-") && strings.HasSuffix(u, "/content")
}

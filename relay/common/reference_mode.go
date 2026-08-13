package common

import "strings"

// InferReferenceMode chooses frame vs media when the client omits reference_mode.
// Vendors may extend with model-specific allowlists after calling this helper.
func InferReferenceMode(req TaskSubmitReq, explicitMode string, allowMedia bool) string {
	mode := strings.ToLower(strings.TrimSpace(explicitMode))
	if mode != "" {
		return mode
	}
	first := strings.TrimSpace(req.FirstImageUrl)
	last := strings.TrimSpace(req.LastImageUrl)
	switch {
	case first != "" || last != "":
		return "frame"
	case len(req.Images) == 2 && first == "" && last == "":
		return "frame"
	case allowMedia && (len(req.Images) > 0 || len(req.ReferenceVideos) > 0 || len(req.ReferenceAudios) > 0):
		return "media"
	case len(req.Images) > 0:
		return "asset"
	default:
		return ""
	}
}

package common

import "strings"

// InferReferenceMode derives the upstream mode from the model contract and
// canonical reference fields. explicitMode is accepted only for callers that
// still serve legacy non-oaivideo task protocols.
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
	case allowMedia && (len(req.Images) > 0 || len(req.ReferenceVideos) > 0 || len(req.ReferenceAudios) > 0):
		return "media"
	case len(req.Images) > 0:
		return "asset"
	default:
		return ""
	}
}

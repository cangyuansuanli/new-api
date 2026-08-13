package omniv2v

import (
	"strings"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// mergeTaskSubmitIntoBodyMap overlays normalized TaskSubmitReq media fields onto the
// JSON body map so outbound mapping always reads canonical client fields.
func mergeTaskSubmitIntoBodyMap(body map[string]interface{}, req *relaycommon.TaskSubmitReq) {
	if body == nil || req == nil {
		return
	}
	if prompt := strings.TrimSpace(req.GetPrompt()); prompt != "" {
		body["prompt"] = prompt
	}
	if aspect := strings.TrimSpace(req.AspectRatio); aspect != "" {
		body["aspect_ratio"] = aspect
	}
	if len(req.ReferenceVideos) > 0 {
		body[flatKeyReferenceVideos] = stringSliceToInterface(req.ReferenceVideos)
	} else if videoURL := strings.TrimSpace(req.VideoURL); videoURL != "" {
		body["video_url"] = videoURL
	}
	if req.HasImage() {
		body[flatKeyReferenceImageURLs] = stringSliceToInterface(req.Images)
	}
}

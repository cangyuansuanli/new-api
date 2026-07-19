package geeknowgrok

import (
	"strconv"
	"strings"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func buildGeeknowGrokBody(req relaycommon.TaskSubmitReq, upstreamModel, originModel string) map[string]any {
	modelName := strings.TrimSpace(upstreamModel)
	if modelName == "" {
		modelName = strings.TrimSpace(req.Model)
	}

	out := map[string]any{
		"model":  modelName,
		"prompt": strings.TrimSpace(req.Prompt),
	}
	if seconds := req.RequestedDurationSeconds(); seconds > 0 {
		out["seconds"] = strconv.Itoa(seconds)
	}
	if ratio := strings.TrimSpace(req.AspectRatio); ratio != "" {
		out["aspect_ratio"] = ratio
	}
	if resolution := normalizeResolution(req.Resolution); resolution != "" {
		out["resolution"] = resolution
	}
	attachReferenceImages(out, req.Images, isImagine15Preview(originModel, modelName))
	return out
}

func attachReferenceImages(out map[string]any, images []string, singleImageOnly bool) {
	refs := make([]string, 0, len(images))
	for _, image := range images {
		if image = strings.TrimSpace(image); image != "" {
			refs = append(refs, image)
		}
	}
	if len(refs) == 0 {
		return
	}
	if singleImageOnly || len(refs) == 1 {
		out["image"] = refs[0]
		return
	}
	out["images"] = refs
}

func normalizeResolution(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	switch {
	case strings.Contains(lower, "480"):
		return "480P"
	case strings.Contains(lower, "720"):
		return "720P"
	default:
		return strings.ToUpper(value)
	}
}

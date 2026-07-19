package seedanceoairegbox

import (
	"strings"

	oaivideo "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/shared"
)

func buildUpstreamBody(body map[string]interface{}, upstreamModel string, duration int) map[string]interface{} {
	prompt := strings.TrimSpace(oaivideo.AsString(body["prompt"]))
	out := map[string]interface{}{
		"model":  strings.TrimSpace(upstreamModel),
		"prompt": prompt,
	}
	mergeFlatDuration(out, body, duration)

	for _, key := range []string{"aspect_ratio", "resolution"} {
		copyStringField(out, body, key)
	}
	if v, ok := body["generate_audio"]; ok {
		out["generate_audio"] = oaivideo.AsBool(v)
	}
	if v, ok := body["seed"]; ok {
		out["seed"] = v
	}

	copyStringField(out, body, flatKeyFirstImageURL)
	copyStringField(out, body, flatKeyLastImageURL)

	if imageURLs := collectReferenceImageURLs(body); len(imageURLs) > 0 {
		out[flatKeyReferenceImageURLs] = referenceImageURLsField(imageURLs)
	}
	copyPassthroughField(out, body, flatKeyReferenceVideos)
	copyPassthroughField(out, body, flatKeyReferenceAudios)

	return out
}

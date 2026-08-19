package seedanceleonardo

import (
	"strings"

	oaivideo "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/shared"
)

func buildUpstreamBody(body map[string]interface{}, upstreamModel string, duration int, referenceImages []string) map[string]interface{} {
	prompt := strings.TrimSpace(oaivideo.AsString(body["prompt"]))
	out := map[string]interface{}{
		"model":  strings.TrimSpace(upstreamModel),
		"prompt": prompt,
	}
	mergeFlatDuration(out, body, duration)

	for _, key := range []string{"aspect_ratio", "resolution"} {
		copyStringField(out, body, key)
	}
	if v, ok := body["audio"]; ok {
		out["audio"] = v
	} else if v, ok := body["generate_audio"]; ok {
		out["audio"] = oaivideo.AsBool(v)
	}

	// Keep the unified first/last-frame contract across the Leonardo gateway.
	// The Web2API server translates these URLs into StartImageURL/EndImageURL
	// before uploading them and constructing guidances.start_frame/end_frame.
	copyStringField(out, body, flatKeyFirstImageURL)
	copyStringField(out, body, flatKeyLastImageURL)

	if len(referenceImages) > 0 {
		out[flatKeyReferenceImageURLs] = referenceImageURLsField(referenceImages)
	}
	copyPassthroughField(out, body, flatKeyReferenceVideos)
	copyPassthroughField(out, body, flatKeyReferenceAudios)

	return out
}

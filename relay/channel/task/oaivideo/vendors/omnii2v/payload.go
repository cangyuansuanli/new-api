package omnii2v

import (
	"strings"

	oaivideo "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/shared"
)

const (
	flatKeyReferenceImageURLs = "reference_image_urls"
)

func buildUpstreamBody(body map[string]interface{}, upstreamModel string, duration int) map[string]interface{} {
	out := map[string]interface{}{
		"model":  strings.TrimSpace(upstreamModel),
		"prompt": strings.TrimSpace(oaivideo.AsString(body["prompt"])),
	}
	if aspectRatio := strings.TrimSpace(oaivideo.AsString(body["aspect_ratio"])); aspectRatio != "" {
		out["aspect_ratio"] = aspectRatio
	}
	if duration > 0 {
		out["seconds"] = duration
	}

	firstImage := strings.TrimSpace(oaivideo.AsString(body["first_image_url"]))
	lastImage := strings.TrimSpace(oaivideo.AsString(body["last_image_url"]))
	if firstImage != "" {
		out["first_image_url"] = firstImage
	}
	if lastImage != "" {
		out["last_image_url"] = lastImage
	}

	refImages := collectReferenceImages(body)
	switch len(refImages) {
	case 0:
	case 1:
		if firstImage == "" && lastImage == "" {
			out["image_url"] = refImages[0]
		} else {
			out["images"] = stringSliceToInterface(refImages)
		}
	default:
		if len(refImages) > 5 {
			refImages = refImages[:5]
		}
		out["images"] = stringSliceToInterface(refImages)
	}
	return out
}

func collectReferenceImages(body map[string]interface{}) []string {
	if body == nil {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, 5)
	add := func(url string) {
		url = strings.TrimSpace(url)
		if url == "" {
			return
		}
		if _, ok := seen[url]; ok {
			return
		}
		seen[url] = struct{}{}
		out = append(out, url)
	}
	for _, url := range oaivideo.CollectStringList(body[flatKeyReferenceImageURLs]) {
		add(url)
	}
	for _, url := range oaivideo.CollectStringList(body["images"]) {
		add(url)
	}
	for _, url := range oaivideo.CollectStringList(body["image_urls"]) {
		add(url)
	}
	if url := strings.TrimSpace(oaivideo.AsString(body["image_url"])); url != "" {
		add(url)
	}
	if url := strings.TrimSpace(oaivideo.AsString(body["image"])); url != "" {
		add(url)
	}
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}

func stringSliceToInterface(urls []string) []interface{} {
	out := make([]interface{}, len(urls))
	for i, url := range urls {
		out[i] = url
	}
	return out
}

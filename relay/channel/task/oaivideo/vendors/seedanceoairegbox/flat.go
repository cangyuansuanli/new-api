package seedanceoairegbox

import (
	"strings"

	oaivideo "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/shared"
)

const (
	flatKeyReferenceImageURLs = "reference_image_urls"
	flatKeyReferenceImages    = "reference_images"
	flatKeyReferenceVideos    = "reference_videos"
	flatKeyReferenceAudios    = "reference_audios"
	flatKeyFirstImageURL      = "first_image_url"
	flatKeyLastImageURL       = "last_image_url"
)

func collectReferenceImageURLs(body map[string]interface{}) []string {
	if body == nil {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, 9)
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
	switch refs := body[flatKeyReferenceImages].(type) {
	case []interface{}:
		for _, item := range refs {
			switch v := item.(type) {
			case string:
				add(v)
			case map[string]interface{}:
				add(oaivideo.AsString(v["url"]))
			}
		}
	}
	return out
}

func referenceImageURLsField(urls []string) interface{} {
	if len(urls) == 0 {
		return nil
	}
	out := make([]interface{}, len(urls))
	for i, url := range urls {
		out[i] = url
	}
	return out
}

func mergeFlatDuration(out map[string]interface{}, body map[string]interface{}, taskDuration int) {
	if out == nil {
		return
	}
	delete(out, "seconds")
	if taskDuration > 0 {
		out["duration"] = taskDuration
		return
	}
	if body != nil {
		if d, ok := body["duration"]; ok && !isEmptyValue(d) {
			out["duration"] = d
		}
	}
}

func copyStringField(out map[string]interface{}, body map[string]interface{}, key string) {
	if body == nil || out == nil {
		return
	}
	if value := strings.TrimSpace(oaivideo.AsString(body[key])); value != "" {
		out[key] = value
	}
}

func copyPassthroughField(out map[string]interface{}, body map[string]interface{}, key string) {
	if body == nil || out == nil {
		return
	}
	if value, ok := body[key]; ok && !isEmptyValue(value) {
		out[key] = value
	}
}

func isEmptyValue(value interface{}) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	case []interface{}:
		return len(v) == 0
	case []string:
		return len(v) == 0
	default:
		return false
	}
}

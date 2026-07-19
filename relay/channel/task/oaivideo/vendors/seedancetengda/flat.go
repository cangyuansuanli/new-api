package seedancetengda

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

func hasFlatSeedanceFields(body map[string]interface{}) bool {
	if body == nil {
		return false
	}
	for _, key := range []string{
		flatKeyReferenceImageURLs,
		flatKeyReferenceImages,
		flatKeyReferenceVideos,
		flatKeyReferenceAudios,
		flatKeyFirstImageURL,
		flatKeyLastImageURL,
	} {
		if _, ok := body[key]; ok {
			return true
		}
	}
	return false
}

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

package seedancetengda

import (
	"strings"
)

const (
	flatKeyReferenceVideos = "reference_videos"
	flatKeyReferenceAudios = "reference_audios"
	flatKeyFirstImageURL   = "first_image_url"
	flatKeyLastImageURL    = "last_image_url"
)

func hasFlatSeedanceFields(body map[string]interface{}) bool {
	if body == nil {
		return false
	}
	for _, key := range []string{
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

package seedance

import (
	"fmt"
	"strconv"
	"strings"

	oaivideo "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/shared"
)

const tengdaUpstreamModel = "manxue-2.0"

func maybeConvertTengdaBody(body map[string]interface{}, upstreamModel string) (map[string]interface{}, error) {
	if body == nil {
		return body, nil
	}
	if isGeeknowNativeSeedanceBody(body) {
		out := cloneBodyMap(body)
		out["model"] = upstreamModel
		return out, nil
	}
	if !needsFlatToGeeknowConversion(body) {
		out := cloneBodyMap(body)
		out["model"] = upstreamModel
		return out, nil
	}
	return convertFlatSeedanceToGeeknow(body, upstreamModel)
}

func needsFlatToGeeknowConversion(body map[string]interface{}) bool {
	if hasFlatSeedanceFields(body) {
		return true
	}
	if _, ok := body["aspect_ratio"]; ok {
		return true
	}
	if _, ok := body["duration"]; ok {
		return true
	}
	return false
}

func isGeeknowNativeSeedanceBody(body map[string]interface{}) bool {
	contentRaw, ok := body["content"]
	if !ok {
		return false
	}
	items, ok := contentRaw.([]interface{})
	if !ok || len(items) == 0 {
		return false
	}
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if role, _ := m["role"].(string); strings.TrimSpace(role) != "" {
			return true
		}
		typ, _ := m["type"].(string)
		switch typ {
		case "image_url", "audio_url", "video_url", "text":
			if typ != "text" {
				return true
			}
		}
	}
	return false
}

func hasFlatSeedanceFields(body map[string]interface{}) bool {
	for _, key := range []string{
		"image_url",
		"reference_image_urls",
		"reference_images",
		"reference_videos",
		"reference_audios",
		"first_image_url",
		"last_image_url",
	} {
		if _, ok := body[key]; ok {
			return true
		}
	}
	return false
}

func convertFlatSeedanceToGeeknow(body map[string]interface{}, upstreamModel string) (map[string]interface{}, error) {
	prompt := strings.TrimSpace(oaivideo.AsString(body["prompt"]))
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	out := map[string]interface{}{
		"model":  upstreamModel,
		"prompt": prompt,
	}

	if seconds := pickSeconds(body); seconds != "" {
		out["seconds"] = seconds
	}
	if ratio := pickRatio(body); ratio != "" {
		out["ratio"] = ratio
	}
	if resolution := pickGeeknowResolution(body); resolution != "" {
		out["resolution"] = resolution
	}

	firstURL := strings.TrimSpace(oaivideo.AsString(body["first_image_url"]))
	lastURL := strings.TrimSpace(oaivideo.AsString(body["last_image_url"]))
	content := make([]map[string]interface{}, 0, 8)

	if firstURL != "" || lastURL != "" {
		if firstURL != "" {
			content = append(content, map[string]interface{}{
				"type":      "image_url",
				"role":      "first_frame",
				"image_url": map[string]interface{}{"url": firstURL},
			})
		}
		if lastURL != "" {
			content = append(content, map[string]interface{}{
				"type":      "image_url",
				"role":      "last_frame",
				"image_url": map[string]interface{}{"url": lastURL},
			})
		}
	} else {
		imageURLs := collectImageURLs(body)
		videoURLs := oaivideo.CollectStringList(body["reference_videos"])
		audioURLs := oaivideo.CollectStringList(body["reference_audios"])

		for _, url := range imageURLs {
			content = append(content, map[string]interface{}{
				"type":      "image_url",
				"role":      "reference_image",
				"image_url": map[string]interface{}{"url": url},
			})
		}
		for _, url := range videoURLs {
			content = append(content, map[string]interface{}{
				"type":      "video_url",
				"role":      "reference_video",
				"video_url": map[string]interface{}{"url": url},
			})
		}
		for _, url := range audioURLs {
			content = append(content, map[string]interface{}{
				"type":      "audio_url",
				"role":      "reference_audio",
				"audio_url": map[string]interface{}{"url": url},
			})
		}
	}

	hasReferenceAudio := false
	for _, item := range content {
		if role, _ := item["role"].(string); role == "reference_audio" {
			hasReferenceAudio = true
			break
		}
	}
	if hasReferenceAudio {
		out["generate_audio"] = true
	} else if v, ok := body["generate_audio"]; ok {
		out["generate_audio"] = oaivideo.AsBool(v)
	}

	if len(content) > 0 {
		textItem := map[string]interface{}{"type": "text", "text": prompt}
		out["content"] = append([]map[string]interface{}{textItem}, content...)
	}

	return out, nil
}

func collectImageURLs(body map[string]interface{}) []string {
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
	add(oaivideo.AsString(body["image_url"]))
	for _, url := range oaivideo.CollectStringList(body["reference_image_urls"]) {
		add(url)
	}
	switch refs := body["reference_images"].(type) {
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

func pickSeconds(body map[string]interface{}) string {
	if d := body["duration"]; d != nil {
		switch v := d.(type) {
		case float64:
			return strconv.Itoa(int(v))
		case int:
			return strconv.Itoa(v)
		case int64:
			return strconv.Itoa(int(v))
		case string:
			if s := strings.TrimSpace(v); s != "" {
				return s
			}
		}
	}
	return ""
}

func pickRatio(body map[string]interface{}) string {
	if r := strings.TrimSpace(oaivideo.AsString(body["ratio"])); r != "" {
		return r
	}
	return strings.TrimSpace(oaivideo.AsString(body["aspect_ratio"]))
}

func pickGeeknowResolution(body map[string]interface{}) string {
	raw := strings.TrimSpace(oaivideo.AsString(body["resolution"]))
	if raw == "" {
		return "720P"
	}
	if strings.Contains(strings.ToLower(raw), "480") {
		return "480P"
	}
	return "720P"
}

func cloneBodyMap(body map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(body))
	for k, v := range body {
		out[k] = v
	}
	return out
}

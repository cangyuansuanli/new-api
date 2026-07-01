package sora

import (
	"fmt"
	"strconv"
	"strings"
)

const tengdaSeedanceUpstreamModel = "manxue-2.0"

// IsTengdaSeedanceRelay 是否走腾达 Geeknow Seedance（flat → content[] 转换）。
func IsTengdaSeedanceRelay(originModel, upstreamModel string) bool {
	if strings.EqualFold(strings.TrimSpace(upstreamModel), tengdaSeedanceUpstreamModel) {
		return true
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(originModel)), "tengd-seedance")
}

func maybeConvertTengdaSeedanceBody(body map[string]interface{}, upstreamModel string) (map[string]interface{}, error) {
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
	prompt := strings.TrimSpace(asString(body["prompt"]))
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	out := map[string]interface{}{
		"model": upstreamModel,
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

	firstURL := strings.TrimSpace(asString(body["first_image_url"]))
	lastURL := strings.TrimSpace(asString(body["last_image_url"]))
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
		videoURLs := collectStringList(body["reference_videos"])
		audioURLs := collectStringList(body["reference_audios"])

		if (len(videoURLs) > 0 || len(audioURLs) > 0) && len(imageURLs) == 0 {
			return nil, fmt.Errorf("带视频/音频参考时至少需要 1 张参考图")
		}

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
		out["generate_audio"] = asBool(v)
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

	add(asString(body["image_url"]))

	for _, url := range collectStringList(body["reference_image_urls"]) {
		add(url)
	}

	switch refs := body["reference_images"].(type) {
	case []interface{}:
		for _, item := range refs {
			add(extractRefURL(item))
		}
	}

	return out
}

func extractRefURL(item interface{}) string {
	switch v := item.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]interface{}:
		return strings.TrimSpace(asString(v["url"]))
	default:
		return ""
	}
}

func collectStringList(raw interface{}) []string {
	out := make([]string, 0)
	switch v := raw.(type) {
	case []interface{}:
		for _, item := range v {
			if s := extractRefURL(item); s != "" {
				out = append(out, s)
			}
		}
	case []string:
		for _, s := range v {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
	case string:
		if s := strings.TrimSpace(v); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func pickSeconds(body map[string]interface{}) string {
	if s := strings.TrimSpace(asString(body["seconds"])); s != "" {
		return s
	}
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
	if r := strings.TrimSpace(asString(body["ratio"])); r != "" {
		return r
	}
	return strings.TrimSpace(asString(body["aspect_ratio"]))
}

func pickGeeknowResolution(body map[string]interface{}) string {
	raw := strings.TrimSpace(asString(body["resolution"]))
	if raw == "" {
		return "720P"
	}
	if strings.Contains(strings.ToLower(raw), "480") {
		return "480P"
	}
	return "720P"
}

func asString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int(t)) {
			return strconv.Itoa(int(t))
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case bool:
		return strconv.FormatBool(t)
	default:
		return fmt.Sprint(v)
	}
}

func asBool(v interface{}) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(strings.TrimSpace(t), "true")
	case float64:
		return t != 0
	case int:
		return t != 0
	default:
		return false
	}
}

func cloneBodyMap(body map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(body))
	for k, v := range body {
		out[k] = v
	}
	return out
}

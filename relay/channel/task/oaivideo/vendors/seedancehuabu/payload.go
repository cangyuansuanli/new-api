package seedancehuabu

import (
	"strings"

	oaivideo "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/shared"
)

const (
	maxReferenceImages = 9
	maxReferenceVideos = 3
	maxReferenceAudios = 3
)

func isFastModel(originModel string) bool {
	return strings.EqualFold(strings.TrimSpace(originModel), ModelFast)
}

func buildUpstreamBody(body map[string]interface{}, originModel, upstreamModel string, duration int, images, refVideos, refAudios []string) map[string]interface{} {
	out := map[string]interface{}{
		"model":  strings.TrimSpace(upstreamModel),
		"prompt": strings.TrimSpace(oaivideo.AsString(body["prompt"])),
	}
	if duration > 0 {
		out["duration"] = duration
	}
	if size := resolveSize(body); size != "" {
		out["size"] = size
	}
	copyStringField(out, body, "first_image_url")
	copyStringField(out, body, "last_image_url")

	imgs := collectImages(body, images)
	if len(imgs) > maxReferenceImages {
		imgs = imgs[:maxReferenceImages]
	}
	// Huabu 上游：单图用 image 字符串，多图用 images 数组（见 sd2.0 接入文档）。
	switch len(imgs) {
	case 0:
	case 1:
		out["image"] = imgs[0]
	default:
		out["images"] = stringSliceToInterface(imgs)
	}

	if !isFastModel(originModel) {
		videos := append(oaivideo.CollectStringList(body["reference_videos"]), refVideos...)
		videos = dedupeStrings(videos)
		if len(videos) > maxReferenceVideos {
			videos = videos[:maxReferenceVideos]
		}
		if len(videos) > 0 {
			out["videos"] = stringSliceToInterface(videos)
		}

		audios := append(oaivideo.CollectStringList(body["reference_audios"]), refAudios...)
		audios = dedupeStrings(audios)
		if len(audios) > maxReferenceAudios {
			audios = audios[:maxReferenceAudios]
		}
		if len(audios) > 0 {
			out["audios"] = stringSliceToInterface(audios)
		}
	}
	return out
}

func copyStringField(out, body map[string]interface{}, key string) {
	if value := strings.TrimSpace(oaivideo.AsString(body[key])); value != "" {
		out[key] = value
	}
}

func resolveSize(body map[string]interface{}) string {
	if body == nil {
		return ""
	}
	if value := strings.TrimSpace(oaivideo.AsString(body["aspect_ratio"])); value != "" {
		return value
	}
	return strings.TrimSpace(oaivideo.AsString(body["size"]))
}

func collectImages(body map[string]interface{}, images []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(images)+4)
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
	for _, url := range images {
		add(url)
	}
	if body != nil {
		for _, url := range oaivideo.CollectStringList(body["reference_image_urls"]) {
			add(url)
		}
		for _, url := range oaivideo.CollectStringList(body["images"]) {
			add(url)
		}
		add(oaivideo.AsString(body["image"]))
		add(oaivideo.AsString(body["image_url"]))
	}
	return out
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
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

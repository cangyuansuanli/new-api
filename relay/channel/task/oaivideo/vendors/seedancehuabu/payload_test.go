package seedancehuabu

import "testing"

func TestBuildUpstreamBodySingleImageUsesImageField(t *testing.T) {
	got := buildUpstreamBody(map[string]any{
		"prompt": "hello",
		"size":   "16:9",
	}, ModelStandard, UpstreamStandard, 5, []string{"https://img/1.png"}, nil, nil)
	if got["image"] != "https://img/1.png" {
		t.Fatalf("expected single image field, got %#v", got)
	}
	if _, ok := got["images"]; ok {
		t.Fatal("upstream must not receive images field")
	}
}

func TestBuildUpstreamBodyMultiImageUsesImageArray(t *testing.T) {
	got := buildUpstreamBody(map[string]any{"prompt": "hello"}, ModelFast, UpstreamFast, 10, []string{"https://img/1.png", "https://img/2.png"}, nil, nil)
	images, ok := got["image"].([]interface{})
	if !ok || len(images) != 2 {
		t.Fatalf("expected image array, got %#v", got["image"])
	}
}

func TestBuildUpstreamBodyCapsReferenceCounts(t *testing.T) {
	images := make([]string, 0, 10)
	for i := 0; i < 10; i++ {
		images = append(images, "https://img/"+string(rune('a'+i))+".png")
	}
	videos := []string{"https://v/1.mp4", "https://v/2.mp4", "https://v/3.mp4", "https://v/4.mp4"}
	audios := []string{"https://a/1.mp3", "https://a/2.mp3", "https://a/3.mp3", "https://a/4.mp3"}
	got := buildUpstreamBody(map[string]any{"prompt": "hello"}, ModelStandard, UpstreamStandard, 10, images, videos, audios)
	if len(got["image"].([]interface{})) != maxReferenceImages {
		t.Fatalf("image cap = %d", len(got["image"].([]interface{})))
	}
	if len(got["videos"].([]interface{})) != maxReferenceVideos {
		t.Fatalf("videos cap = %d", len(got["videos"].([]interface{})))
	}
	if len(got["audios"].([]interface{})) != maxReferenceAudios {
		t.Fatalf("audios cap = %d", len(got["audios"].([]interface{})))
	}
}

func TestResolveSizePrefersAspectRatio(t *testing.T) {
	if got := resolveSize(map[string]any{"aspect_ratio": "9:16", "size": "16:9"}); got != "9:16" {
		t.Fatalf("size = %q", got)
	}
}

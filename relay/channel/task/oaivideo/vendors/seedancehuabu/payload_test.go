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
		t.Fatal("single-image upstream must not receive images field")
	}
}

func TestBuildUpstreamBodyMultiImageUsesImagesArray(t *testing.T) {
	got := buildUpstreamBody(map[string]any{"prompt": "hello @image1 @image2"}, ModelFast, UpstreamFast, 10, []string{"https://img/1.png", "https://img/2.png"}, nil, nil)
	if _, ok := got["image"]; ok {
		t.Fatalf("multi-image upstream must not receive image string, got %#v", got["image"])
	}
	images, ok := got["images"].([]interface{})
	if !ok || len(images) != 2 {
		t.Fatalf("expected images array, got %#v", got["images"])
	}
}

func TestBuildUpstreamBodyMultiImageFromClientReferenceField(t *testing.T) {
	got := buildUpstreamBody(map[string]any{
		"prompt":               "hello @image1 @image2",
		"reference_image_urls": []string{"https://img/1.png", "https://img/2.png"},
	}, ModelStandard, UpstreamStandard, 10, []string{"https://img/1.png", "https://img/2.png"}, nil, nil)
	images, ok := got["images"].([]interface{})
	if !ok || len(images) != 2 {
		t.Fatalf("expected images array from client reference_image_urls, got %#v", got["images"])
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
	imgs, ok := got["images"].([]interface{})
	if !ok || len(imgs) != maxReferenceImages {
		t.Fatalf("image cap = %d", len(imgs))
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

func TestBuildUpstreamBodyCopiesPairedFrames(t *testing.T) {
	got := buildUpstreamBody(map[string]any{
		"prompt":          "transition",
		"first_image_url": "https://img/first.png",
		"last_image_url":  "https://img/last.png",
	}, ModelFast, UpstreamFast, 10, nil, nil, nil)
	if got["first_image_url"] != "https://img/first.png" || got["last_image_url"] != "https://img/last.png" {
		t.Fatalf("paired frames not copied: %#v", got)
	}
}

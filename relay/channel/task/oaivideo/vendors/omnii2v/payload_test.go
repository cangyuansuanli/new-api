package omnii2v

import "testing"

func TestBuildUpstreamBodyMapsReferenceImages(t *testing.T) {
	out := buildUpstreamBody(map[string]interface{}{
		"prompt":               "test prompt",
		"aspect_ratio":         "16:9",
		"reference_image_urls": []interface{}{"https://cdn.example.com/a.jpg", "https://cdn.example.com/b.jpg"},
	}, "omni-fast", 0)
	images, ok := out["images"].([]interface{})
	if !ok || len(images) != 2 {
		t.Fatalf("expected images array with 2 entries, got %v", out["images"])
	}
	if out["model"] != "omni-fast" {
		t.Fatalf("expected upstream model omni-fast, got %v", out["model"])
	}
	if _, exists := out["reference_image_urls"]; exists {
		t.Fatal("reference_image_urls should not be forwarded upstream")
	}
}

func TestBuildUpstreamBodySingleImageUsesImageURL(t *testing.T) {
	out := buildUpstreamBody(map[string]interface{}{
		"prompt":    "test prompt",
		"image_url": "https://cdn.example.com/a.jpg",
	}, "omni-fast-no-water", 0)
	if out["image_url"] != "https://cdn.example.com/a.jpg" {
		t.Fatalf("expected image_url, got %v", out["image_url"])
	}
}

func TestBuildUpstreamBodyKeepsFrameURLs(t *testing.T) {
	out := buildUpstreamBody(map[string]interface{}{
		"prompt":           "test prompt",
		"first_image_url":  "https://cdn.example.com/first.jpg",
		"last_image_url":   "https://cdn.example.com/last.jpg",
		"reference_image_urls": []interface{}{"https://cdn.example.com/ref.jpg"},
	}, "omni-fast", 0)
	if out["first_image_url"] != "https://cdn.example.com/first.jpg" {
		t.Fatalf("expected first_image_url preserved, got %v", out["first_image_url"])
	}
	if images, ok := out["images"].([]interface{}); !ok || len(images) != 1 {
		t.Fatalf("expected single images entry with frame refs, got %v", out["images"])
	}
}

func TestIsRelay(t *testing.T) {
	if !IsRelay("cy-sd1-omni-fast", "omni-fast") {
		t.Fatal("expected omni i2v relay match")
	}
	if IsRelay("cy-sd1-omni-v2v", "omni-fast-v2v") {
		t.Fatal("omni v2v should not match omni i2v relay")
	}
	if IsRelay("omni-fast", "omni-fast-v2v") {
		t.Fatal("omni v2v upstream should not match omni i2v relay")
	}
}

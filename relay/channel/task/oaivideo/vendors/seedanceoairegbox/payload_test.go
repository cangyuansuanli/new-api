package seedanceoairegbox

import "testing"

func TestBuildUpstreamBody_ImageVideo933(t *testing.T) {
	in := map[string]interface{}{
		"prompt": "motion @image1 @video1",
		"reference_image_urls": []interface{}{
			"https://example.com/ref.jpg",
		},
		"reference_videos": []interface{}{
			"https://example.com/ref.mp4",
		},
		"duration":     8,
		"aspect_ratio": "16:9",
		"resolution":   "720p",
		"image_url":    "https://example.com/single.jpg",
	}
	out := buildUpstreamBody(in, "Seedance-2.0-720p", 0)
	if _, ok := out["image_url"]; ok {
		t.Fatal("legacy image_url must not be forwarded")
	}
	refs, ok := out["reference_image_urls"].([]interface{})
	if !ok || len(refs) != 2 || refs[0] != "https://example.com/ref.jpg" || refs[1] != "https://example.com/single.jpg" {
		t.Fatalf("expected canonical and image_url references, got %v", out["reference_image_urls"])
	}
	videos, ok := out["reference_videos"].([]interface{})
	if !ok || len(videos) != 1 {
		t.Fatalf("expected reference_videos passthrough, got %v", out["reference_videos"])
	}
	if out["model"] != "Seedance-2.0-720p" {
		t.Fatalf("expected upstream model, got %v", out["model"])
	}
}

func TestBuildUpstreamBody_TaskDurationOverridesBody(t *testing.T) {
	in := map[string]interface{}{
		"prompt":   "test",
		"duration": 6,
	}
	out := buildUpstreamBody(in, "Seedance-2.0-fast-720p", 10)
	if out["duration"] != 10 {
		t.Fatalf("expected task duration 10, got %v", out["duration"])
	}
}

func TestIsRelay(t *testing.T) {
	if !IsRelay("cy-sd1-seedance-2.0-fast-720p") {
		t.Fatal("expected oairegbox relay")
	}
	if IsRelay("cy-sd4-seedance-2.0") {
		t.Fatal("cy-sd4 must not match oairegbox")
	}
}

func TestCollectReferenceImageURLs_NormalizesPublicAliases(t *testing.T) {
	in := map[string]interface{}{
		"image_url": "https://example.com/single.jpg",
		"image":     map[string]interface{}{"url": "https://example.com/object.jpg"},
		"images":    []interface{}{"https://example.com/list.jpg", "https://example.com/single.jpg"},
		"reference_image_urls": []interface{}{
			"https://example.com/canonical.jpg",
		},
	}
	urls := collectReferenceImageURLs(in)
	if len(urls) != 4 {
		t.Fatalf("expected four deduplicated aliases, got %v", urls)
	}
}

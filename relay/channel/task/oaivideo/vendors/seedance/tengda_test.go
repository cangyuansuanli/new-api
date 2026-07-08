package seedance

import "testing"

func TestIsLeonardoRelay(t *testing.T) {
	if !IsLeonardoRelay("cy-sd4-seedance-2.0") {
		t.Fatal("expected leonardo relay")
	}
	if IsLeonardoRelay("sora-2") {
		t.Fatal("sora must not match leonardo")
	}
}

func TestIsOairegboxRelay(t *testing.T) {
	if !IsOairegboxRelay("cy-sd1-seedance-2.0-fast-720p") {
		t.Fatal("expected oairegbox relay")
	}
	if IsOairegboxRelay("cy-sd4-seedance-2.0") {
		t.Fatal("cy-sd4 must not match oairegbox prefix alone")
	}
	if !IsRelay("cy-sd1-seedance-2.0-mini-480p", "") {
		t.Fatal("cy-sd1 should match seedance IsRelay")
	}
}

func TestIsTengdaRelay(t *testing.T) {
	if !IsTengdaRelay("cy-sd2-Seedance-2.0", "manxue-2.0") {
		t.Fatal("expected tengda relay")
	}
	if !IsTengdaRelay("tengd-Seedance-2.0", "manxue-2.0") {
		t.Fatal("expected tengd relay")
	}
	if IsTengdaRelay("cy-sd1-seedance-2.0-720p", "Seedance-2.0-720p") {
		t.Fatal("cy-sd1 must not match tengda relay")
	}
	if IsTengdaRelay("cy-sd4-seedance-2.0", "seedance-2.0") {
		t.Fatal("cy-sd4 must not match tengda relay")
	}
}

func TestMaybeConvertTengdaBody_NativePassthrough(t *testing.T) {
	in := map[string]interface{}{
		"model": "manxue-2.0",
		"content": []interface{}{
			map[string]interface{}{
				"type": "image_url",
				"role": "reference_image",
				"image_url": map[string]interface{}{
					"url": "https://example.com/a.jpg",
				},
			},
		},
	}
	out, err := maybeConvertTengdaBody(in, "manxue-2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["model"] != "manxue-2.0" {
		t.Fatalf("expected model passthrough, got %v", out["model"])
	}
}

func TestMaybeConvertTengdaBody_AudioRequiresImage(t *testing.T) {
	in := map[string]interface{}{
		"prompt":            "test",
		"reference_audios":  []interface{}{"https://example.com/a.mp3"},
	}
	_, err := maybeConvertTengdaBody(in, "manxue-2.0")
	if err == nil {
		t.Fatal("expected error when audio without image")
	}
}

func TestMaybeConvertTengdaBody_FlatConversion(t *testing.T) {
	in := map[string]interface{}{
		"prompt":      "ocean waves",
		"duration":    8,
		"aspect_ratio": "16:9",
		"resolution":  "720p",
		"image_url":   "https://example.com/a.jpg",
	}
	out, err := maybeConvertTengdaBody(in, "manxue-2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["model"] != "manxue-2.0" {
		t.Fatalf("expected upstream model, got %v", out["model"])
	}
	if out["seconds"] != "8" {
		t.Fatalf("expected seconds 8, got %v", out["seconds"])
	}
	if out["ratio"] != "16:9" {
		t.Fatalf("expected ratio 16:9, got %v", out["ratio"])
	}
	content, ok := out["content"].([]map[string]interface{})
	if !ok || len(content) < 2 {
		t.Fatalf("expected converted content, got %v", out["content"])
	}
}

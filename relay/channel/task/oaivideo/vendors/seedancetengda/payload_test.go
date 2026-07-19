package seedancetengda

import "testing"

func TestIsRelay(t *testing.T) {
	if !IsRelay("cy-sd2-Seedance-2.0", "manxue-2.0") {
		t.Fatal("expected tengda relay")
	}
	if !IsRelay("tengd-Seedance-2.0", "manxue-2.0") {
		t.Fatal("expected tengd relay")
	}
	if IsRelay("cy-sd1-seedance-2.0-720p", "Seedance-2.0-720p") {
		t.Fatal("cy-sd1 must not match tengda relay")
	}
	if IsRelay("cy-sd4-seedance-2.0", "seedance-2.0") {
		t.Fatal("cy-sd4 must not match tengda relay")
	}
}

func TestConvertBody_NativePassthrough(t *testing.T) {
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
	out, err := convertBody(in, "manxue-2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["model"] != "manxue-2.0" {
		t.Fatalf("expected model passthrough, got %v", out["model"])
	}
}

func TestConvertBody_AudioWithoutImage(t *testing.T) {
	in := map[string]interface{}{
		"prompt":           "test",
		"reference_audios": []interface{}{"https://example.com/a.mp3"},
	}
	out, err := convertBody(in, "manxue-2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content, ok := out["content"].([]map[string]interface{})
	if !ok || len(content) != 2 || content[1]["role"] != "reference_audio" {
		t.Fatalf("expected standalone audio reference, got %v", out["content"])
	}
}

func TestConvertBody_FlatConversion(t *testing.T) {
	in := map[string]interface{}{
		"prompt":       "ocean waves",
		"duration":     8,
		"aspect_ratio": "16:9",
		"resolution":   "720p",
		"reference_image_urls": []interface{}{
			"https://example.com/a.jpg",
		},
	}
	out, err := convertBody(in, "manxue-2.0")
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

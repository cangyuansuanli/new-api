package sora

import (
	"encoding/json"
	"testing"
)

func TestConvertFlatSeedanceToGeeknow_TextToVideo(t *testing.T) {
	in := map[string]interface{}{
		"model":        "Seedance-2.0",
		"prompt":       "雨夜霓虹街道",
		"aspect_ratio": "16:9",
		"duration":     float64(8),
		"resolution":   "720p",
	}
	out, err := convertFlatSeedanceToGeeknow(in, "manxue-2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["model"] != "manxue-2.0" {
		t.Fatalf("model: %v", out["model"])
	}
	if out["seconds"] != "8" {
		t.Fatalf("seconds: %v", out["seconds"])
	}
	if out["ratio"] != "16:9" {
		t.Fatalf("ratio: %v", out["ratio"])
	}
	if out["resolution"] != "720P" {
		t.Fatalf("resolution: %v", out["resolution"])
	}
	if _, ok := out["content"]; ok {
		t.Fatalf("text-only should not include content")
	}
}

func TestConvertFlatSeedanceToGeeknow_MultiReference(t *testing.T) {
	in := map[string]interface{}{
		"prompt":                 "@image1 在 @image2 场景行走",
		"duration":               10,
		"aspect_ratio":           "9:16",
		"image_url":              "https://cdn.example.com/person.jpg",
		"reference_image_urls":   []interface{}{"https://cdn.example.com/scene.jpg"},
		"reference_videos":       []interface{}{"https://cdn.example.com/ref.mp4"},
		"reference_audios":       []interface{}{"https://cdn.example.com/ref.mp3"},
	}
	out, err := convertFlatSeedanceToGeeknow(in, "manxue-2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["generate_audio"] != true {
		t.Fatalf("generate_audio: %v", out["generate_audio"])
	}
	content, ok := out["content"].([]map[string]interface{})
	if !ok || len(content) != 5 {
		t.Fatalf("content len: %d", len(content))
	}
	if content[0]["type"] != "text" {
		t.Fatalf("first content item should be text")
	}
	if content[1]["role"] != "reference_image" {
		t.Fatalf("second item role: %v", content[1]["role"])
	}
}

func TestConvertFlatSeedanceToGeeknow_FrameTransition(t *testing.T) {
	in := map[string]interface{}{
		"prompt":          "平滑过渡",
		"duration":        5,
		"first_image_url": "https://cdn.example.com/start.jpg",
		"last_image_url":  "https://cdn.example.com/end.jpg",
	}
	out, err := convertFlatSeedanceToGeeknow(in, "manxue-2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := out["content"].([]map[string]interface{})
	if len(content) != 3 {
		t.Fatalf("content len: %d", len(content))
	}
	if content[1]["role"] != "first_frame" || content[2]["role"] != "last_frame" {
		t.Fatalf("unexpected frame roles: %+v", content)
	}
}

func TestMaybeConvertTengdaSeedanceBody_NativePassthrough(t *testing.T) {
	in := map[string]interface{}{
		"model":    "Seedance-2.0",
		"prompt":   "test",
		"seconds":  "8",
		"ratio":    "16:9",
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": "test",
			},
			map[string]interface{}{
				"type":      "image_url",
				"role":      "reference_image",
				"image_url": map[string]interface{}{"url": "https://example.com/a.png"},
			},
		},
	}
	out, err := maybeConvertTengdaSeedanceBody(in, "manxue-2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["seconds"] != "8" {
		t.Fatalf("native body should passthrough seconds")
	}
	content := out["content"].([]interface{})
	if len(content) != 2 {
		t.Fatalf("content should remain unchanged")
	}
}

func TestMaybeConvertTengdaSeedanceBody_AudioRequiresImage(t *testing.T) {
	in := map[string]interface{}{
		"prompt":           "test",
		"reference_audios": []interface{}{"https://cdn.example.com/a.mp3"},
	}
	_, err := maybeConvertTengdaSeedanceBody(in, "manxue-2.0")
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestIsTengdaSeedanceRelay(t *testing.T) {
	if !IsTengdaSeedanceRelay("tengd-Seedance-2.0", "manxue-2.0") {
		t.Fatal("expected tengd relay")
	}
	if IsTengdaSeedanceRelay("oairegbox-seedance-2.0-720p", "Seedance-2.0-720p") {
		t.Fatal("oairegbox should not use tengd relay")
	}
}

func TestConvertFlatSeedanceToGeeknow_JSONRoundTrip(t *testing.T) {
	raw := `{
		"model":"Seedance-2.0",
		"prompt":"清晨海边，电影感",
		"aspect_ratio":"16:9",
		"duration":8,
		"resolution":"480p"
	}`
	var in map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &in); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := convertFlatSeedanceToGeeknow(in, "manxue-2.0")
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !json.Valid(data) {
		t.Fatalf("invalid json: %s", string(data))
	}
}

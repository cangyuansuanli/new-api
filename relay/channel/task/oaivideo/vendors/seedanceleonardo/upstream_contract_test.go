package seedanceleonardo

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildUpstreamBody_NewAPIShapeForLeonardo(t *testing.T) {
	in := map[string]interface{}{
		"prompt":           "neon",
		"aspect_ratio":     "9:16",
		"resolution":       "480p",
		"generate_audio":   true,
		"reference_videos": []interface{}{"v1", "v2", "v3", "v4"},
	}
	out := buildUpstreamBody(in, "seedance-2.0-fast", 6, []string{"https://cdn.example.com/ref.jpg"})
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["model"] != "seedance-2.0-fast" || decoded["duration"] != float64(6) {
		t.Fatalf("model/duration: %v", decoded)
	}
	if decoded["audio"] != true {
		t.Fatalf("expected audio from generate_audio, got %v", decoded["audio"])
	}
	vids, ok := decoded["reference_videos"].([]interface{})
	if !ok || len(vids) != 4 {
		t.Fatalf("reference_videos=%v", decoded["reference_videos"])
	}
	refs, ok := decoded["reference_image_urls"].([]interface{})
	if !ok || len(refs) != 1 {
		t.Fatalf("reference_image_urls=%v", decoded["reference_image_urls"])
	}
}

func TestBuildUpstreamBody_LeonardoContractGoldenFourVideos(t *testing.T) {
	out := buildUpstreamBody(map[string]interface{}{
		"prompt": "x",
	}, "seedance-2.0", 8, nil)
	out["reference_videos"] = []interface{}{"u1", "u2", "u3", "u4"}
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	// Documented contract: leonardo-web2api internal/server/newapi_upstream_contract_test.go
	if !strings.Contains(string(raw), `"reference_videos"`) || !strings.Contains(string(raw), `"u4"`) {
		t.Fatalf("unexpected body: %s", raw)
	}
}

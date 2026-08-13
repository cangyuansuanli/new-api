package omniv2v

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestMergeTaskSubmitIntoBodyMap(t *testing.T) {
	body := map[string]interface{}{"prompt": "old"}
	req := relaycommon.TaskSubmitReq{
		Prompt:          "restyle",
		AspectRatio:     "9:16",
		ReferenceVideos: []string{"https://cdn.example.com/a.mp4", "https://cdn.example.com/b.mp4"},
		ReferenceImageUrls: []string{
			"https://cdn.example.com/ref.jpg",
		},
	}
	mergeTaskSubmitIntoBodyMap(body, &req)
	if body["prompt"] != "restyle" {
		t.Fatalf("prompt = %#v", body["prompt"])
	}
	if body["aspect_ratio"] != "9:16" {
		t.Fatalf("aspect_ratio = %#v", body["aspect_ratio"])
	}
	videos, ok := body["reference_videos"].([]interface{})
	if !ok || len(videos) != 2 {
		t.Fatalf("reference_videos = %#v", body["reference_videos"])
	}
	images, ok := body["reference_image_urls"].([]interface{})
	if !ok || len(images) != 1 {
		t.Fatalf("reference_image_urls = %#v", body["reference_image_urls"])
	}
}

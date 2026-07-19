package geeknowgrok

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func TestBuildRequestBodyUsesGeeknowContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Model:       "cy-gv1-grok-video",
		Prompt:      "rainy neon street",
		Duration:    6,
		AspectRatio: "16:9",
		Resolution:  "720p",
		Images:      []string{"https://example.com/a.png", "https://example.com/b.png"},
	})
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: upstreamImagineVideo}}

	reader, err := (&TaskAdaptor{}).BuildRequestBody(c, info)
	if err != nil {
		t.Fatalf("BuildRequestBody: %v", err)
	}
	body, _ := io.ReadAll(reader)
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if got["model"] != upstreamImagineVideo {
		t.Fatalf("model = %#v", got["model"])
	}
	if got["seconds"] != "6" {
		t.Fatalf("seconds must be string: %#v", got["seconds"])
	}
	if got["resolution"] != "720P" {
		t.Fatalf("resolution = %#v", got["resolution"])
	}
	images, ok := got["images"].([]any)
	if !ok || len(images) != 2 {
		t.Fatalf("images must be array: %#v", got["images"])
	}
	if _, exists := got["image_urls"]; exists {
		t.Fatalf("image_urls must not leak upstream: %#v", got)
	}
}

func TestBuildRequestBodyImagine15UsesSingleImageField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Model:       "cy-gv1-grok-video-1.5",
		Prompt:      "gentle push-in",
		Duration:    4,
		AspectRatio: "9:16",
		Resolution:  "480p",
		Images:      []string{"https://example.com/ref.png"},
	})
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: upstreamImagineVideo15Prev}}

	reader, err := (&TaskAdaptor{}).BuildRequestBody(c, info)
	if err != nil {
		t.Fatalf("BuildRequestBody: %v", err)
	}
	body, _ := io.ReadAll(reader)
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if got["image"] != "https://example.com/ref.png" {
		t.Fatalf("image = %#v", got["image"])
	}
	if _, exists := got["images"]; exists {
		t.Fatalf("1.5 preview must not send images array: %#v", got)
	}
}

func TestParseTaskResultUsesOpenAIVideoShape(t *testing.T) {
	body := []byte(`{"id":"task_x","status":"completed","progress":100,"video_url":"https://example.com/video.mp4"}`)
	result, err := (&TaskAdaptor{}).ParseTaskResult(body)
	if err != nil {
		t.Fatalf("ParseTaskResult: %v", err)
	}
	if result.Status != model.TaskStatusSuccess || result.Url != "https://example.com/video.mp4" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestFetchTaskUsesOpenAIVideosPath(t *testing.T) {
	service.InitHttpClient()

	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"task_x","status":"processing","progress":30}`))
	}))
	defer server.Close()

	resp, err := (&TaskAdaptor{}).FetchTask(server.URL, "test-key", map[string]any{"task_id": "task_x"}, "")
	if err != nil {
		t.Fatalf("FetchTask: %v", err)
	}
	_ = resp.Body.Close()
	if path != "/v1/videos/task_x" {
		t.Fatalf("path = %q", path)
	}
}

func TestIsRelayMatchesGeeknowUpstreamModels(t *testing.T) {
	if !IsRelay("cy-gv1-grok-video", upstreamImagineVideo) {
		t.Fatal("expected grok-imagine-video upstream to match")
	}
	if !IsRelay("cy-gv1-grok-video-1.5", upstreamImagineVideo15Prev) {
		t.Fatal("expected grok-imagine-video-1.5-preview upstream to match")
	}
	if IsRelay("cy-gv1-grok-video", "grok-image-video") {
		t.Fatal("119337 upstream must not match geeknow vendor")
	}
}

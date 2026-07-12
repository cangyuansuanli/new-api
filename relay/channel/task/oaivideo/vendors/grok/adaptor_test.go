package grok

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

func TestBuildRequestBodyUsesGenerationsContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Model:       "cy-gv1-grok-video-1.5",
		Prompt:      "animate",
		Images:      []string{"https://example.com/ref.png"},
		Duration:    4,
		AspectRatio: "9:16",
		Resolution:  "720p",
	})
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "grok-video-1.5"}}

	reader, err := (&TaskAdaptor{}).BuildRequestBody(c, info)
	if err != nil {
		t.Fatalf("BuildRequestBody: %v", err)
	}
	body, _ := io.ReadAll(reader)
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if got["model"] != "grok-video-1.5" || got["seconds"] != float64(4) {
		t.Fatalf("unexpected model/seconds: %#v", got)
	}
	images, ok := got["image_urls"].([]any)
	if !ok || len(images) != 1 || images[0] != "https://example.com/ref.png" {
		t.Fatalf("image_urls must be a string array: %#v", got["image_urls"])
	}
	if _, exists := got["input_reference"]; exists {
		t.Fatalf("input_reference must not leak upstream: %#v", got)
	}
}

func TestParseTaskResultUnwrapsGenerationsEnvelope(t *testing.T) {
	body := []byte(`{"code":"success","data":{"task_id":"task_x","status":"SUCCESS","progress":"100%","result_url":"https://example.com/video.mp4","fail_reason":""}}`)
	result, err := (&TaskAdaptor{}).ParseTaskResult(body)
	if err != nil {
		t.Fatalf("ParseTaskResult: %v", err)
	}
	if result.Status != model.TaskStatusSuccess || result.Url != "https://example.com/video.mp4" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestParseTaskResultPrefersTaskIDWhenEnvelopeHasNumericRecordID(t *testing.T) {
	body := []byte(`{"code":"success","data":{"id":70897,"task_id":"task_x","status":"SUCCESS","progress":"100%","result_url":"https://vidgen.x.ai/video.mp4","data":{"model":"grok-image-video","video":{"url":"https://vidgen.x.ai/video.mp4"}}}}`)
	result, err := (&TaskAdaptor{}).ParseTaskResult(body)
	if err != nil {
		t.Fatalf("ParseTaskResult: %v", err)
	}
	if result.Status != model.TaskStatusSuccess || result.Url != "https://vidgen.x.ai/video.mp4" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestParseTaskResultPreservesFailureReason(t *testing.T) {
	body := []byte(`{"code":"success","data":{"task_id":"task_x","status":"FAILURE","progress":"100%","fail_reason":"reference image rejected"}}`)
	result, err := (&TaskAdaptor{}).ParseTaskResult(body)
	if err != nil {
		t.Fatalf("ParseTaskResult: %v", err)
	}
	if result.Status != model.TaskStatusFailure || result.Reason != "reference image rejected" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestFetchTaskUsesGenerationsPath(t *testing.T) {
	service.InitHttpClient()

	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"success","data":{"task_id":"task_x","status":"IN_PROGRESS"}}`))
	}))
	defer server.Close()

	resp, err := (&TaskAdaptor{}).FetchTask(server.URL, "test-key", map[string]any{"task_id": "task_x"}, "")
	if err != nil {
		t.Fatalf("FetchTask: %v", err)
	}
	_ = resp.Body.Close()
	if path != "/v1/video/generations/task_x" {
		t.Fatalf("path = %q", path)
	}
}

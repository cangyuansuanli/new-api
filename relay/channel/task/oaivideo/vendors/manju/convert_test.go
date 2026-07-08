package manju

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestIsRelay(t *testing.T) {
	if !IsRelay("manju-openai-sora2", "sora2") {
		t.Fatal("expected manju sora2 relay")
	}
	if IsRelay("sora-2", "sora-2") {
		t.Fatal("expected standard sora not manju")
	}
	if IsRelay("cy-sd4-seedance-2.0", "sora2") {
		t.Fatal("leonardo seedance must not match manju relay")
	}
}

func TestConvertChatBody(t *testing.T) {
	out, err := ConvertChatBody(map[string]interface{}{
		"prompt":  "cat on beach",
		"seconds": "8",
		"size":    "1280x720",
	}, "sora2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["model"] != "sora2" {
		t.Fatalf("expected upstream model sora2, got %v", out["model"])
	}
	if out["stream"] != false {
		t.Fatalf("expected stream false, got %v", out["stream"])
	}
	if out["sora2_ratio"] != "16:9" {
		t.Fatalf("expected sora2_ratio 16:9, got %v", out["sora2_ratio"])
	}
	if out["sora2_duration"] != "8" {
		t.Fatalf("expected sora2_duration 8, got %v", out["sora2_duration"])
	}
	if out["sora2_output_resolution"] != "720p" {
		t.Fatalf("expected sora2_output_resolution 720p, got %v", out["sora2_output_resolution"])
	}
	msgs, ok := out["messages"].([]map[string]interface{})
	if !ok || len(msgs) == 0 || msgs[0]["content"] != "cat on beach" {
		t.Fatalf("expected messages with prompt, got %v", out["messages"])
	}
}

func TestConvertChatBody_InputReference(t *testing.T) {
	out, err := ConvertChatBody(map[string]interface{}{
		"prompt":          "cat",
		"seconds":         "8",
		"input_reference": "https://example.com/ref.png",
	}, "sora2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["input_reference"] != "https://example.com/ref.png" {
		t.Fatalf("expected input_reference, got %v", out["input_reference"])
	}
}

func TestParseTaskResult_Succeeded(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{
		"id":"sora2-e949ceaef92c",
		"platform":"sora2",
		"status":"succeeded",
		"progress":100,
		"properties":{"duration":"8","aspect_ratio":"16:9","output_resolution":"720p"},
		"raw_data":{"video_url":"https://dlff.manjuapi.com/files/demo.mp4","video_urls":["https://dlff.manjuapi.com/files/demo.mp4"]},
		"video":{"url":"https://dlff.manjuapi.com/files/demo.mp4"}
	}`)
	result, err := adaptor.ParseTaskResult(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != model.TaskStatusSuccess {
		t.Fatalf("expected SUCCESS, got %s", result.Status)
	}
	want := "https://dlff.manjuapi.com/files/demo.mp4"
	if result.Url != want {
		t.Fatalf("expected url %q, got %q", want, result.Url)
	}
	if result.CompletionTokens != 8 {
		t.Fatalf("expected 8 seconds, got %d", result.CompletionTokens)
	}
}

func TestParseTaskResult_FailedWithMessage(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{
		"id":"sora2-failed001",
		"platform":"sora2",
		"status":"failed",
		"message":"某张上传的参考图未通过平台内容审核（常见于含可识别真人肖像或敏感内容）；重试无效，请更换涉及的参考图后重试"
	}`)
	result, err := adaptor.ParseTaskResult(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != model.TaskStatusFailure {
		t.Fatalf("expected FAILURE, got %s", result.Status)
	}
	want := "某张上传的参考图未通过平台内容审核（常见于含可识别真人肖像或敏感内容）；重试无效，请更换涉及的参考图后重试"
	if result.Reason != want {
		t.Fatalf("expected reason %q, got %q", want, result.Reason)
	}
}

func TestBuildOpenAIErrorResponse(t *testing.T) {
	body := []byte(`{"id":"sora2-failed001","platform":"sora2","status":"failed","message":"审核失败"}`)
	out, ok := BuildOpenAIErrorResponse(body)
	if !ok {
		t.Fatal("expected error conversion")
	}
	if !strings.Contains(string(out), "审核失败") {
		t.Fatalf("expected message in output, got %s", string(out))
	}
}

func TestBuildOpenAIVideoCreateResponse(t *testing.T) {
	info := &relaycommon.RelayInfo{
		PublicTaskID:    "task_public",
		OriginModelName: "manju-openai-sora2",
	}
	out := buildOpenAIVideoCreateResponse(info, responseTaskFromGJSON([]byte(`{"status":"running","progress":13,"properties":{"duration":"8"}}`)), nil)
	if out["id"] != "task_public" {
		t.Fatal("expected public task id")
	}
	if out["status"] != "in_progress" {
		t.Fatalf("expected in_progress, got %v", out["status"])
	}
}

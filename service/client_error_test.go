package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func TestNormalizeTaskErrorMessageUsesRegisteredSD5Rule(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", nil)
	c.Request.Header.Set("Accept-Language", "zh-CN")
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "cy-sd5-seedance-2.0-fast")

	taskErr := &dto.TaskError{
		Code:       "fail_to_fetch_task",
		Message:    `{"error":"system under load","error_type":"submission_overloaded"}`,
		StatusCode: 408,
	}
	NormalizeTaskErrorMessage(c, taskErr)

	if got, want := taskErr.Message, "SD5 上游负载过高或提交超时，请稍后重试。"; got != want {
		t.Fatalf("NormalizeTaskErrorMessage() = %q, want %q", got, want)
	}
}

func TestNormalizeClientErrorMessageUsesRequestModelContext(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", nil)
	c.Request.Header.Set("Accept-Language", "zh-CN")
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "cy-sd5-seedance-2.0")

	raw := `{"error":"system under load","error_type":"submission_overloaded"}`
	if got, want := NormalizeClientErrorMessage(c, raw), "SD5 上游负载过高或提交超时，请稍后重试。"; got != want {
		t.Fatalf("NormalizeClientErrorMessage() = %q, want %q", got, want)
	}
}

func TestNormalizeOpenAIVideoTaskResponseUsesStoredSD5Payload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/v1/videos/task_1", nil)
	c.Request.Header.Set("Accept-Language", "zh-CN")
	task := &model.Task{
		FailReason: "system under load",
		Properties: model.Properties{OriginModelName: "cy-sd5-seedance-2.0"},
		Data:       []byte(`{"error":"system under load","error_type":"submission_overloaded","error_status":408}`),
	}

	out := NormalizeOpenAIVideoTaskResponse(c, task, []byte(`{"status":"failed","error":{"message":"system under load"},"fail_reason":"system under load"}`))
	var payload struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
		FailReason string `json:"fail_reason"`
	}
	if err := common.Unmarshal(out, &payload); err != nil {
		t.Fatal(err)
	}
	want := "SD5 上游负载过高或提交超时，请稍后重试。"
	if payload.Error.Message != want || payload.FailReason != want {
		t.Fatalf("NormalizeOpenAIVideoTaskResponse() = %#v, want message %q", payload, want)
	}
}

func TestNormalizeOpenAIImageTaskJobErrorUsesTaskContext(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/v1/images/task_1", nil)
	c.Request.Header.Set("Accept-Language", "zh-CN")
	task := &model.Task{Properties: model.Properties{OriginModelName: "adobe-gpt-image"}}
	job := &dto.OpenAIImageJob{Error: &dto.OpenAIImageError{Message: "reference images exceed 3"}}

	NormalizeOpenAIImageTaskJobError(c, task, job)

	if got, want := job.Error.Message, "参考图最多 3 张，请减少后重试。"; got != want {
		t.Fatalf("NormalizeOpenAIImageTaskJobError() = %q, want %q", got, want)
	}
}

func TestNormalizeTaskFailurePassthroughCySd4Leonardo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/v1/videos/task_1", nil)
	c.Request.Header.Set("Accept-Language", "zh-CN")

	upstream := "参考视频最多 3 段，当前 4 段，请减少后重试（MP4/MOV，单条 4–15 秒，最多 3 段总时长 ≤15 秒，宽高各 720–2160px，24–60 FPS）。"
	task := &model.Task{
		FailReason: upstream,
		Properties: model.Properties{OriginModelName: "cy-sd4-seedance-2.0"},
	}
	if got := NormalizeTaskFailure(c, task); got != upstream {
		t.Fatalf("NormalizeTaskFailure() = %q, want passthrough %q", got, upstream)
	}
}

func TestNormalizeOpenAIVideoTaskResponsePassthroughCySd4Leonardo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/v1/videos/task_1", nil)
	c.Request.Header.Set("Accept-Language", "zh-CN")

	upstream := "参考视频最多 3 段，当前 4 段，请减少后重试（MP4/MOV，单条 4–15 秒，最多 3 段总时长 ≤15 秒，宽高各 720–2160px，24–60 FPS）。"
	task := &model.Task{
		Properties: model.Properties{OriginModelName: "cy-sd4-seedance-2.0-mini"},
	}
	in := []byte(`{"status":"failed","error":{"message":"` + upstream + `"}}`)
	out := NormalizeOpenAIVideoTaskResponse(c, task, in)
	if string(out) != string(in) {
		t.Fatalf("expected passthrough body, got %s", out)
	}
}

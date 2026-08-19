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

func TestNormalizeMagicaErrorPassesThroughUpstreamGuidance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", nil)
	c.Request.Header.Set("Accept-Language", "zh-CN")
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "cy-sd7-seedance-2.0-720p")

	upstream := "提示词过长或结构不符合要求。请控制在 2500 字符以内，删除重复的角色设定、分镜解释和无关说明，只保留主体、动作、场景、镜头、风格及必要限制后重试"
	if got := NormalizeClientErrorMessage(c, upstream); got != upstream {
		t.Fatalf("NormalizeClientErrorMessage() = %q, want exact upstream message %q", got, upstream)
	}

	taskErr := &dto.TaskError{Code: "generation_failed", Message: upstream, StatusCode: 400}
	NormalizeTaskErrorMessage(c, taskErr)
	if taskErr.Message != upstream {
		t.Fatalf("NormalizeTaskErrorMessage() = %q, want exact upstream message %q", taskErr.Message, upstream)
	}
}

func TestNormalizeMagicaVideoResponsePassesThroughUpstreamPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/v1/videos/task_1", nil)
	c.Request.Header.Set("Accept-Language", "zh-CN")
	task := &model.Task{Properties: model.Properties{OriginModelName: "cy-sd7-seedance-2.0-720p"}}
	in := []byte(`{"status":"failed","error":{"message":"提示词过长或结构不符合要求。请控制在 2500 字符以内，删除重复的角色设定、分镜解释和无关说明，只保留主体、动作、场景、镜头、风格及必要限制后重试"}}`)
	if out := NormalizeOpenAIVideoTaskResponse(c, task, in); string(out) != string(in) {
		t.Fatalf("expected Magica upstream payload passthrough, got %s", out)
	}
}

func TestNormalizeOpenAIVideoTaskResponsePassesThroughSD5Payload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/v1/videos/task_1", nil)
	c.Request.Header.Set("Accept-Language", "zh-CN")
	task := &model.Task{
		FailReason: "system under load",
		Properties: model.Properties{OriginModelName: "cy-sd5-seedance-2.0"},
		Data:       []byte(`{"error":"system under load","error_type":"submission_overloaded","error_status":408}`),
	}

	in := []byte(`{"status":"failed","error":{"message":"当前服务负载较高，请稍后重试。"},"fail_reason":"当前服务负载较高，请稍后重试。"}`)
	out := NormalizeOpenAIVideoTaskResponse(c, task, in)
	if string(out) != string(in) {
		t.Fatalf("expected SD5 upstream payload passthrough, got %s", out)
	}
}

func TestNormalizeTaskFailurePassesThroughSD5Message(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	upstream := "参考素材包含可识别真人肖像，当前模型无法处理；请移除真人脸或改用不含真实身份的人物素材。"
	task := &model.Task{FailReason: upstream, Properties: model.Properties{OriginModelName: "cy-sd5-seedance-2.0"}}
	if got := NormalizeTaskFailure(c, task); got != upstream {
		t.Fatalf("NormalizeTaskFailure() = %q, want passthrough %q", got, upstream)
	}
}

func TestNormalizeTaskFailurePassesThroughSD5GenericFailureFaceHint(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	upstream := "视频生成失败，上游未提供具体原因。若使用了参考素材，可能因素材包含可识别真人脸而被当前渠道拒绝，也可能与内容安全、素材冲突或模型临时异常有关；请优先移除真人脸素材后重试。"
	task := &model.Task{FailReason: upstream, Properties: model.Properties{OriginModelName: "cy-sd5-seedance-2.0-fast"}}
	if got := NormalizeTaskFailure(c, task); got != upstream {
		t.Fatalf("NormalizeTaskFailure() = %q, want passthrough %q", got, upstream)
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

func TestNormalizeTaskFailurePassthroughCySd4HappyHouse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/v1/videos/task_1", nil)
	c.Request.Header.Set("Accept-Language", "zh-CN")

	upstream := "参考视频仅部分版本支持，请调整素材后重试。"
	task := &model.Task{
		FailReason: upstream,
		Properties: model.Properties{OriginModelName: "cy-sd4-happyhouse-1.1"},
	}
	if got := NormalizeTaskFailure(c, task); got != upstream {
		t.Fatalf("NormalizeTaskFailure() = %q, want passthrough %q", got, upstream)
	}
}

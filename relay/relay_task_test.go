package relay

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func TestTaskModel2DtoForClientNormalizesStructuredSD5Failure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/task", nil)
	c.Request.Header.Set("Accept-Language", "zh-CN")

	task := &model.Task{
		ChannelId:  86,
		Status:     model.TaskStatusFailure,
		FailReason: "system under load",
		Properties: model.Properties{OriginModelName: "cy-sd5-seedance-2.0-fast"},
		Data:       []byte(`{"error":"system under load","error_type":"submission_overloaded","error_status":408}`),
	}

	dto := TaskModel2DtoForClient(c, task)
	if got, want := dto.FailReason, "SD5 上游负载过高或提交超时，请稍后重试。"; got != want {
		t.Fatalf("TaskModel2DtoForClient() fail_reason = %q, want %q", got, want)
	}
}

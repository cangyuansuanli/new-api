package relay

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/registry"
	taskoairouter "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/router"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func TestPrepareRoutedTaskAdaptorMapsBeforeVendorInit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("POST", "/v1/videos", bytes.NewBufferString(`{"model":"seedance-2.0","prompt":"test"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("model_mapping", `{"cy-sd4-seedance-2.0":"seedance-2.0"}`)

	adaptor := taskoairouter.NewRouterAdaptor()
	info := &relaycommon.RelayInfo{
		OriginModelName: "cy-sd4-seedance-2.0",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://leonardo.example.test",
			ApiKey:         "gateway-key",
		},
	}

	resolver := adaptor.(interface {
		ResolveTaskVendor(*relaycommon.RelayInfo) string
	})
	if taskErr := prepareRoutedTaskAdaptor(c, adaptor, resolver, info, info.OriginModelName); taskErr != nil {
		t.Fatalf("prepareRoutedTaskAdaptor() error = %+v", taskErr)
	}

	if info.TaskVendor != string(registry.VendorSeedanceLeonardo) {
		t.Fatalf("task vendor = %q", info.TaskVendor)
	}
	url, err := adaptor.BuildRequestURL(info)
	if err != nil {
		t.Fatalf("BuildRequestURL() error = %v", err)
	}
	if url != "https://leonardo.example.test/v1/videos" {
		t.Fatalf("BuildRequestURL() = %q", url)
	}
	req := httptest.NewRequest("POST", url, nil)
	if err := adaptor.BuildRequestHeader(c, req, info); err != nil {
		t.Fatalf("BuildRequestHeader() error = %v", err)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer gateway-key" {
		t.Fatalf("Authorization = %q", got)
	}
}

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

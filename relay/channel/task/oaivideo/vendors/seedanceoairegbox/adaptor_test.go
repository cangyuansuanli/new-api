package seedanceoairegbox

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func multipartContext(t *testing.T, model, duration string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("model", model)
	_ = writer.WriteField("prompt", "test")
	if duration != "" {
		_ = writer.WriteField("duration", duration)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	return c
}

func TestValidateRequestRejectsMissingDuration(t *testing.T) {
	c := multipartContext(t, "cy-sd1-seedance-2.0-4k", "")
	info := &relaycommon.RelayInfo{OriginModelName: "cy-sd1-seedance-2.0-4k"}
	if taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info); taskErr == nil {
		t.Fatal("expected missing duration error")
	}
}

func TestValidateRequestRejectsOutOfRangeDuration(t *testing.T) {
	for _, duration := range []string{"1", "16"} {
		c := multipartContext(t, "cy-sd1-seedance-2.0-4k", duration)
		info := &relaycommon.RelayInfo{OriginModelName: "cy-sd1-seedance-2.0-4k"}
		if taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info); taskErr == nil {
			t.Fatalf("expected duration %s to be rejected", duration)
		}
	}
}

func TestValidateRequestAcceptsDurationRange(t *testing.T) {
	c := multipartContext(t, "cy-sd1-seedance-2.0-4k", "15")
	info := &relaycommon.RelayInfo{OriginModelName: "cy-sd1-seedance-2.0-4k"}
	if taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info); taskErr != nil {
		t.Fatalf("unexpected error: %v", taskErr)
	}
}

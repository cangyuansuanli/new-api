package common

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestValidateMultipartDirectStoresNormalizedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("model", "adobe-sora2")
	_ = writer.WriteField("prompt", "a city at night")
	_ = writer.WriteField("seconds", "8")
	_ = writer.WriteField("size", "1280x720")
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	info := &RelayInfo{}
	if taskErr := ValidateMultipartDirect(c, info); taskErr != nil {
		t.Fatalf("ValidateMultipartDirect: %v", taskErr)
	}
	req, err := GetTaskRequest(c)
	if err != nil {
		t.Fatalf("GetTaskRequest: %v", err)
	}
	if req.Model != "adobe-sora2" || req.Prompt != "a city at night" || req.Duration != 8 {
		t.Fatalf("normalized request = %#v", req)
	}
}

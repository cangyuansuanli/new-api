package shared

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func jsonVideoContext(t *testing.T, body string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	if taskErr := relaycommon.ValidateMultipartDirect(c, &relaycommon.RelayInfo{}); taskErr != nil {
		t.Fatalf("validate request: %v", taskErr)
	}
	t.Cleanup(func() { common.CleanupBodyStorage(c) })
	return c
}

func TestBuildNormalizedRequestBodyMapsDurationToSeconds(t *testing.T) {
	c := jsonVideoContext(t, `{"model":"grok-video","prompt":"test","duration":15,"aspect_ratio":"9:16","image_url":"https://example.com/ref.png","vendor_magic":"must-not-pass"}`)

	reader, err := BuildNormalizedRequestBody(c, "grok-image-video", DurationFieldSeconds)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read request: %v", err)
	}
	var got map[string]any
	if err := common.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if got["model"] != "grok-image-video" || got["seconds"] != float64(15) {
		t.Fatalf("normalized body = %#v", got)
	}
	if _, exists := got["duration"]; exists {
		t.Fatalf("duration alias leaked upstream: %#v", got)
	}
	if got["aspect_ratio"] != "9:16" {
		t.Fatalf("unrelated field was not preserved: %#v", got)
	}
	images, ok := got["reference_image_urls"].([]any)
	if !ok || len(images) != 1 || images[0] != "https://example.com/ref.png" {
		t.Fatalf("canonical images = %#v", got["reference_image_urls"])
	}
	if _, exists := got["image_url"]; exists {
		t.Fatalf("image alias leaked upstream: %#v", got)
	}
	if _, exists := got["vendor_magic"]; exists {
		t.Fatalf("unknown client field leaked upstream: %#v", got)
	}
}

func TestBuildNormalizedRequestBodyMapsSecondsToDuration(t *testing.T) {
	c := jsonVideoContext(t, `{"model":"seedance","prompt":"test","seconds":"12"}`)

	reader, err := BuildNormalizedRequestBody(c, "seedance-upstream", DurationFieldDuration)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	body, _ := io.ReadAll(reader)
	var got map[string]any
	if err := common.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if got["duration"] != float64(12) {
		t.Fatalf("normalized body = %#v", got)
	}
	if _, exists := got["seconds"]; exists {
		t.Fatalf("seconds alias leaked upstream: %#v", got)
	}
}

func TestBuildNormalizedRequestBodyPreservesMultipartFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var raw bytes.Buffer
	writer := multipart.NewWriter(&raw)
	_ = writer.WriteField("model", "grok-video")
	_ = writer.WriteField("prompt", "test")
	_ = writer.WriteField("duration", "15")
	part, err := writer.CreateFormFile("input_reference[]", "reference.png")
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	_, _ = part.Write([]byte("png-data"))
	_ = writer.Close()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", bytes.NewReader(raw.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	storage, err := common.CreateBodyStorage(raw.Bytes())
	if err != nil {
		t.Fatalf("cache request: %v", err)
	}
	c.Set(common.KeyBodyStorage, storage)
	c.Request.Body = io.NopCloser(storage)
	t.Cleanup(func() { common.CleanupBodyStorage(c) })
	if taskErr := relaycommon.ValidateMultipartDirect(c, &relaycommon.RelayInfo{}); taskErr != nil {
		t.Fatalf("validate request: %v", taskErr)
	}

	reader, err := BuildNormalizedRequestBody(c, "grok-image-video", DurationFieldSeconds)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	out, _ := io.ReadAll(reader)
	request := httptest.NewRequest("POST", "/v1/videos", bytes.NewReader(out))
	request.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	if err := request.ParseMultipartForm(1 << 20); err != nil {
		t.Fatalf("parse normalized multipart: %v", err)
	}
	if request.FormValue("seconds") != "15" || request.FormValue("duration") != "" {
		t.Fatalf("duration fields: seconds=%q duration=%q", request.FormValue("seconds"), request.FormValue("duration"))
	}
	files := request.MultipartForm.File["input_reference[]"]
	if len(files) != 1 || files[0].Filename != "reference.png" {
		t.Fatalf("multipart file not preserved: %#v", files)
	}
}

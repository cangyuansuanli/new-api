package openai

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
)

func TestIsChatImageModel(t *testing.T) {
	cases := []struct {
		model string
		want  bool
	}{
		{"manju-gemini-banana-pro-4k", false},
		{"0lll0-gemini-3.1-flash-lite-image", true},
		{"some-flash-image-model", true},
		{"byte-gemini-banana-2.0", true},
		{"gpt-image-2", false},
		{"Gulie-gpt-image-2", false},
	}
	for _, tc := range cases {
		if got := IsChatImageModel(tc.model); got != tc.want {
			t.Fatalf("IsChatImageModel(%q) = %v, want %v", tc.model, got, tc.want)
		}
	}
}

func TestConvertImageRequestForChatImageGeneration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		OriginModelName: "0lll0-gemini-3.1-flash-lite-image",
	}
	request := dto.ImageRequest{
		Model:   "0lll0-gemini-3.1-flash-lite-image",
		Prompt:  "a cat on windowsill",
		Size:    "16:9",
		Quality: "high",
	}
	out, err := ConvertImageRequestForChatImage(c, info, request)
	if err != nil {
		t.Fatalf("ConvertImageRequestForChatImage: %v", err)
	}
	chatReq, ok := out.(dto.GeneralOpenAIRequest)
	if !ok {
		t.Fatalf("expected GeneralOpenAIRequest, got %T", out)
	}
	if len(chatReq.Messages) != 1 || chatReq.Messages[0].Role != "user" {
		t.Fatalf("messages = %+v", chatReq.Messages)
	}
	content, ok := chatReq.Messages[0].Content.(string)
	if !ok || content != "a cat on windowsill" {
		t.Fatalf("content = %#v", chatReq.Messages[0].Content)
	}
	if chatReq.Stream == nil || *chatReq.Stream {
		t.Fatal("stream should be false")
	}
	var extra map[string]any
	if err := common.Unmarshal(chatReq.ExtraBody, &extra); err != nil {
		t.Fatalf("extra_body: %v", err)
	}
	google, _ := extra["google"].(map[string]any)
	cfg, _ := google["image_config"].(map[string]any)
	if cfg["aspect_ratio"] != "16:9" {
		t.Fatalf("aspect_ratio = %v", cfg["aspect_ratio"])
	}
	if cfg["image_size"] != "4K" {
		t.Fatalf("image_size = %v", cfg["image_size"])
	}
}

func TestConvertImageRequestForChatImageWithReferences(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		OriginModelName: "0lll0-gemini-3.1-flash-lite-image",
	}
	request := dto.ImageRequest{
		Model:  "0lll0-gemini-3.1-flash-lite-image",
		Prompt: "edit this",
		Image:  json.RawMessage(`"https://example.com/ref.png"`),
	}
	out, err := ConvertImageRequestForChatImage(c, info, request)
	if err != nil {
		t.Fatalf("ConvertImageRequestForChatImage: %v", err)
	}
	chatReq := out.(dto.GeneralOpenAIRequest)
	parts, ok := chatReq.Messages[0].Content.([]dto.MediaContent)
	if !ok || len(parts) < 2 {
		t.Fatalf("expected content parts, got %#v", chatReq.Messages[0].Content)
	}
	if parts[0].Type != "text" || parts[1].Type != "image_url" {
		t.Fatalf("parts = %+v", parts)
	}
	if img := parts[1].GetImageMedia(); img == nil || img.Url != "https://example.com/ref.png" {
		t.Fatalf("image url = %+v", parts[1].ImageUrl)
	}
}

func TestImageDataFromChatImageContent(t *testing.T) {
	images, err := imageDataFromChatImageContent("![image](data:image/png;base64,abc123)")
	if err != nil {
		t.Fatalf("imageDataFromChatImageContent: %v", err)
	}
	if len(images) != 1 || images[0].B64Json != "abc123" {
		t.Fatalf("images = %+v", images)
	}

	images, err = imageDataFromChatImageContent("![image](https://cdn.example.com/out.png)")
	if err != nil {
		t.Fatalf("http url: %v", err)
	}
	if images[0].Url != "https://cdn.example.com/out.png" {
		t.Fatalf("url = %q", images[0].Url)
	}
}

func TestParseLegacyChatImageResponse(t *testing.T) {
	body := []byte(`{"choices":[{"message":{"content":"![image](data:image/png;base64,xyz)"}}],"usage":{"total_tokens":10}}`)
	images, usage, err := ParseLegacyChatImageResponse(body)
	if err != nil {
		t.Fatalf("ParseLegacyChatImageResponse: %v", err)
	}
	if len(images) != 1 || images[0].B64Json != "xyz" {
		t.Fatalf("images = %+v", images)
	}
	if usage.TotalTokens != 10 {
		t.Fatalf("usage = %+v", usage)
	}
}

func TestNormalizeAsyncLegacyChatImageBody(t *testing.T) {
	out, err := NormalizeAsyncLegacyChatImageBody([]byte(`{"model":"gemini-banana-pro-4k","async":true,"stream":true}`))
	if err != nil {
		t.Fatalf("NormalizeAsyncLegacyChatImageBody: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := raw["async"]; ok {
		t.Fatal("async should be stripped")
	}
	if string(raw["stream"]) != "false" {
		t.Fatalf("stream = %s", raw["stream"])
	}
}

func TestIsAsyncChatImageRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gemini-banana-pro-4k","async":true,"stream":false,"messages":[{"role":"user","content":"cat"}]}`)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	storage, err := common.CreateBodyStorage(body)
	if err != nil {
		t.Fatalf("CreateBodyStorage: %v", err)
	}
	c.Set(common.KeyBodyStorage, storage)
	if !IsAsyncChatImageRequest(c) {
		t.Fatal("expected async chat image request")
	}
	if IsAsyncChatImageRequest(nil) {
		t.Fatal("nil context should be false")
	}
}

func TestIsLegacyChatImageRequestSync(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"0lll0-gemini-3.1-flash-lite-image","stream":false,"messages":[{"role":"user","content":"cat"}]}`)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	storage, err := common.CreateBodyStorage(body)
	if err != nil {
		t.Fatalf("CreateBodyStorage: %v", err)
	}
	c.Set(common.KeyBodyStorage, storage)
	if !IsLegacyChatImageRequest(c) {
		t.Fatal("expected legacy chat image request")
	}
}

func TestOpenaiChatImageHandlerMarkdownToImageResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	c.Set("id", 1)

	respBody := []byte(`{"choices":[{"message":{"content":"![image](data:image/png;base64,cGF5bG9hZA==)"}}],"usage":{"total_tokens":1}}`)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioNopCloser{bytes.NewReader(respBody)},
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "gemini-banana-2.0",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://example.com",
		},
	}
	usage, apiErr := OpenaiChatImageHandler(c, info, resp)
	if apiErr != nil {
		t.Fatalf("OpenaiChatImageHandler: %v", apiErr)
	}
	if usage == nil || usage.TotalTokens == 0 {
		t.Fatalf("usage = %+v", usage)
	}
	out := w.Body.Bytes()
	if !strings.Contains(string(out), `"b64_json"`) && !strings.Contains(string(out), `"url"`) {
		t.Fatalf("response missing image data: %s", out)
	}
}

type ioNopCloser struct{ *bytes.Reader }

func (ioNopCloser) Close() error { return nil }

package relay

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	openai "github.com/QuantumNous/new-api/relay/channel/openai"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
)

func TestNormalizeAsyncGenerationBodyUsesURLResponseFormatFor4K(t *testing.T) {
	out, err := normalizeAsyncGenerationBody([]byte(`{"model":"geek2-gpt-image-2-4k","prompt":"test","async":true,"response_format":"b64_json"}`), true)
	if err != nil {
		t.Fatalf("normalizeAsyncGenerationBody: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(raw["response_format"]) != `"url"` {
		t.Fatalf("response_format = %s, want url", raw["response_format"])
	}
	if _, ok := raw["async"]; ok {
		t.Fatalf("async should be stripped")
	}
}

func TestNormalizeAsyncGenerationBodyKeepsB64ForNon4K(t *testing.T) {
	out, err := normalizeAsyncGenerationBody([]byte(`{"model":"Gulie-gpt-image-2","prompt":"test","async":true}`), false)
	if err != nil {
		t.Fatalf("normalizeAsyncGenerationBody: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(raw["response_format"]) != `"b64_json"` {
		t.Fatalf("response_format = %s, want b64_json", raw["response_format"])
	}
}

func TestImageAsyncUsesURLResponseOnlyFor4K(t *testing.T) {
	if !imageAsyncUsesURLResponse("geek2-gpt-image-2-4k") {
		t.Fatal("expected 4k model to use url response")
	}
	if !imageAsyncUsesURLResponse("flux-pro-2") {
		t.Fatal("expected flux-pro-2 to use url response")
	}
	if imageAsyncUsesURLResponse("Gulie-gpt-image-2") {
		t.Fatal("non-4k model should not use url response")
	}
}

func TestNormalizeAsyncGenerationBodyUsesURLResponseFormatForFlux(t *testing.T) {
	out, err := normalizeAsyncGenerationBody([]byte(`{"model":"flux-pro-2","prompt":"test","async":true,"response_format":"b64_json"}`), true)
	if err != nil {
		t.Fatalf("normalizeAsyncGenerationBody: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(raw["response_format"]) != `"url"` {
		t.Fatalf("response_format = %s, want url", raw["response_format"])
	}
}

func TestDecodeImageDataItemDetectsJPEGFromB64(t *testing.T) {
	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46}
	data, mime, err := decodeImageDataItem(dto.ImageData{B64Json: base64.StdEncoding.EncodeToString(jpeg)})
	if err != nil {
		t.Fatalf("decodeImageDataItem: %v", err)
	}
	if mime != "image/jpeg" {
		t.Fatalf("mime = %q, want image/jpeg", mime)
	}
	if len(data) != len(jpeg) {
		t.Fatalf("data len = %d, want %d", len(data), len(jpeg))
	}
}

func TestIsAsyncChatImageRequestRelayWrapper(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gemini-banana-pro-4k","async":true,"stream":false,"messages":[{"role":"user","content":"cat"}]}`)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(string(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	storage, err := common.CreateBodyStorage(body)
	if err != nil {
		t.Fatalf("CreateBodyStorage: %v", err)
	}
	c.Set(common.KeyBodyStorage, storage)
	if !IsAsyncChatImageRequest(c) {
		t.Fatal("expected async chat image request via relay wrapper")
	}
}

func TestNormalizeAsyncLegacyChatImageBodyViaOpenAI(t *testing.T) {
	out, err := openai.NormalizeAsyncLegacyChatImageBody([]byte(`{"model":"gemini-banana-pro-4k","async":true,"stream":true}`))
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
}

func TestParseLegacyChatImageResponseViaOpenAI(t *testing.T) {
	body := []byte(`{"choices":[{"message":{"content":"![image](data:image/png;base64,abc123)"}}]}`)
	images, usage, err := openai.ParseLegacyChatImageResponse(body)
	if err != nil {
		t.Fatalf("ParseLegacyChatImageResponse: %v", err)
	}
	if len(images) != 1 || images[0].B64Json != "abc123" {
		t.Fatalf("images = %+v", images)
	}
	if usage.TotalTokens == 0 {
		t.Fatal("expected default usage")
	}
}

func TestImageJobObjectForPath(t *testing.T) {
	if imageJobObjectForPath("/v1/images/edits") != "image.edit" {
		t.Fatalf("edits object = %q", imageJobObjectForPath("/v1/images/edits"))
	}
	if imageJobObjectForPath("/v1/images/generations") != "image.generation" {
		t.Fatalf("generations object = %q", imageJobObjectForPath("/v1/images/generations"))
	}
}

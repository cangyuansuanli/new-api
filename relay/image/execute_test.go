package image

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	openai "github.com/QuantumNous/new-api/relay/channel/openai"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func TestQueuedEditHTTPSReferencesPassThroughWithoutR2Upload(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, pair := range [][2]string{
		{"model", "gpt-image-2"},
		{"prompt", "edit"},
		{"image", "https://cdn.example.com/a.png"},
		{"image", "https://cdn.example.com/b.png"},
		{"mask", "https://cdn.example.com/mask.png"},
	} {
		if err := writer.WriteField(pair[0], pair[1]); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	snapshot, err := SnapshotEditRequest(c, "task_url_refs")
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeRequestSnapshot(snapshot, "/v1/images/edits")
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Multipart.Files) != 3 {
		t.Fatalf("URL references = %d, want 3", len(decoded.Multipart.Files))
	}
	for _, file := range decoded.Multipart.Files {
		if file.URL == "" || file.ObjectKey != "" || len(file.Data) != 0 {
			t.Fatalf("unexpected URL snapshot file: %#v", file)
		}
	}

	task := &model.Task{PrivateData: model.TaskPrivateData{RequestSnapshot: snapshot, RequestPath: "/v1/images/edits"}}
	replayed, _, err := buildHTTPRequestForImageTask(context.Background(), task)
	if err != nil {
		t.Fatal(err)
	}
	if err := replayed.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	if got := replayed.MultipartForm.Value["image"]; len(got) != 2 || got[0] != "https://cdn.example.com/a.png" || got[1] != "https://cdn.example.com/b.png" {
		t.Fatalf("replayed images = %#v", got)
	}
	if got := replayed.MultipartForm.Value["mask"]; len(got) != 1 || got[0] != "https://cdn.example.com/mask.png" {
		t.Fatalf("replayed mask = %#v", got)
	}
	if len(replayed.MultipartForm.File) != 0 {
		t.Fatalf("URL references should not become files: %#v", replayed.MultipartForm.File)
	}
}

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
	out, err := normalizeAsyncGenerationBody([]byte(`{"model":"go2api-gpt-image-2-1k","prompt":"test","async":true,"response_format":"url"}`), false)
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

func TestImageAsyncUsesURLResponseForRehostModels(t *testing.T) {
	if !imageAsyncUsesURLResponse("geek2-gpt-image-2-4k") {
		t.Fatal("expected 4k model to use url response")
	}
	if !imageAsyncUsesURLResponse("flux-pro-2") {
		t.Fatal("expected flux-pro-2 to use url response")
	}
	if !imageAsyncUsesURLResponse("Gulie-gpt-image-2") {
		t.Fatal("gulie should use an internal upstream url before R2 rehost")
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
	data, mime, err := service.DecodeImageDataItem(dto.ImageData{B64Json: base64.StdEncoding.EncodeToString(jpeg)})
	if err != nil {
		t.Fatalf("DecodeImageDataItem: %v", err)
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

func TestIsAsyncRequestReadsJSONWithBareMultipartContentType(t *testing.T) {
	body := []byte(`{"model":"gpt-image-2-2k","prompt":"test","async":true}`)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "multipart/form-data")
	storage, err := common.CreateBodyStorage(body)
	if err != nil {
		t.Fatalf("CreateBodyStorage: %v", err)
	}
	c.Set(common.KeyBodyStorage, storage)

	if !IsAsyncRequest(c) {
		t.Fatal("expected JSON async flag to be detected despite bare multipart Content-Type")
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
	if JobObjectForPath("/v1/images/edits") != "image.edit" {
		t.Fatalf("edits object = %q", JobObjectForPath("/v1/images/edits"))
	}
	if JobObjectForPath("/v1/images/generations") != "image.generation" {
		t.Fatalf("generations object = %q", JobObjectForPath("/v1/images/generations"))
	}
}

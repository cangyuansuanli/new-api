package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
)

func TestBuildManjuBananaImageBodyJSONRequestSkipsMultipartParse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(
		`{"model":"gemini-banana-pro-4k","prompt":"test","size":"1:1","quality":"high","n":1,"stream":false}`,
	))
	c.Request.Header.Set("Content-Type", "application/json")

	n := uint(1)
	info := &relaycommon.RelayInfo{
		OriginModelName: "manju-gemini-banana-pro-4k",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gemini-3.0-pro-image 4K",
		},
	}
	body, err := buildManjuBananaImageBody(c, info, dto.ImageRequest{
		Model:   "gemini-banana-pro-4k",
		Prompt:  "test",
		Size:    "1:1",
		Quality: "high",
		N:       &n,
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if body["model"] != "gemini-3.0-pro-image 4K" {
		t.Fatalf("model = %v", body["model"])
	}
	if body["aspect_ratio"] != "1:1" {
		t.Fatalf("aspect_ratio = %v", body["aspect_ratio"])
	}
	if body["output_resolution"] != "4K" {
		t.Fatalf("output_resolution = %v", body["output_resolution"])
	}
}

func TestBuildManjuBananaImageGenerationBody(t *testing.T) {
	n := uint(1)
	body := BuildManjuBananaImageGenerationBody("manju-gemini-banana-pro-4k", dto.ImageRequest{
		Model:   "gemini-3.0-pro-image 4K",
		Prompt:  "a red apple",
		Size:    "1024x1024",
		Quality: "low",
		N:       &n,
	})
	if body["model"] != "gemini-3.0-pro-image 4K" {
		t.Fatalf("model = %v", body["model"])
	}
	if body["aspect_ratio"] != "1:1" {
		t.Fatalf("aspect_ratio = %v", body["aspect_ratio"])
	}
	if body["output_resolution"] != "4K" {
		t.Fatalf("output_resolution = %v", body["output_resolution"])
	}
	if body["stream"] != false {
		t.Fatalf("stream = %v", body["stream"])
	}
}

func TestBuildManjuBananaImageGenerationBodyWithReferenceImage(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "manju-gemini-banana-pro-1/2k",
		RelayMode:       relayconstant.RelayModeImagesEdits,
	}
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	out, err := ConvertManjuBananaImageRequest(c, info, dto.ImageRequest{
		Model:  "gemini-3.0-pro-image",
		Prompt: "edit style",
		Image:  json.RawMessage(`"https://example.com/ref.png"`),
		Size:   "16:9",
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	chatReq, ok := out.(dto.GeneralOpenAIRequest)
	if !ok {
		t.Fatalf("expected chat request, got %T", out)
	}
	if chatReq.Model != "gemini-3.0-pro-image" {
		t.Fatalf("model = %q", chatReq.Model)
	}
}

func TestBuildManjuBananaImageGenerationBodyWithMultipleImages(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "manju-gemini-banana-2.0-1/2k",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "Nano Banana 2",
		},
	}
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	out, err := ConvertManjuBananaImageRequest(c, info, dto.ImageRequest{
		Model:  "gemini-banana-2.0-1/2k",
		Prompt: "combine",
		Images: json.RawMessage(`["https://example.com/a.png","https://example.com/b.png"]`),
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	chatReq, ok := out.(dto.GeneralOpenAIRequest)
	if !ok {
		t.Fatalf("expected chat request, got %T", out)
	}
	if chatReq.Model != "Nano Banana 2" {
		t.Fatalf("model = %q", chatReq.Model)
	}
}

func TestResolveManjuBananaOutputResolutionDefault1K(t *testing.T) {
	for _, model := range []string{
		"manju-gemini-banana-pro-1/2k",
		"manju-gemini-banana-2.0-1/2k",
	} {
		if got := resolveManjuBananaOutputResolution(model, ""); got != "1K" {
			t.Fatalf("%s: output_resolution = %q, want 1K", model, got)
		}
	}
}

func TestManjuBananaUsesChatCompletionsUpstreamForEdits(t *testing.T) {
	info := &relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeImagesEdits}
	if !ManjuBananaUsesChatCompletionsUpstream(nil, info, dto.ImageRequest{Prompt: "x"}) {
		t.Fatal("edits should use chat/completions")
	}
}

func TestManjuBananaUsesChatCompletionsUpstreamForTextGeneration(t *testing.T) {
	info := &relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeImagesGenerations}
	if ManjuBananaUsesChatCompletionsUpstream(nil, info, dto.ImageRequest{Prompt: "x"}) {
		t.Fatal("text-only generation should use /v1/images/generations")
	}
}

func TestAdaptManjuBananaChatCompletionResponseAsyncPoll(t *testing.T) {
	var polls int
	imageBody := []byte("fakejpeg")
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/files/test.jpg"):
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write(imageBody)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/api/tasks/"):
			polls++
			if polls < 2 {
				_, _ = w.Write([]byte(`{"status":"running","task_id":"gemini-img-test"}`))
				return
			}
			_, _ = w.Write([]byte(`{"status":"succeeded","result_url":"` + srv.URL + `/files/test.jpg"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	createBody := []byte(`{
		"status":"running",
		"task_id":"gemini-img-test",
		"poll_url":"` + srv.URL + `/api/tasks/gemini-img-test",
		"choices":[{"message":{"role":"assistant","content":""}}]
	}`)
	info := &relaycommon.RelayInfo{
		OriginModelName: "manju-gemini-banana-pro-4k",
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:         "sk-test",
			ChannelBaseUrl: srv.URL,
		},
	}

	out, adaptErr := AdaptManjuBananaChatCompletionResponse(context.Background(), info, createBody)
	if adaptErr != nil {
		t.Fatalf("adapt: %v", adaptErr)
	}
	content := string(out)
	if !strings.Contains(content, "data:image/jpeg;base64,") {
		t.Fatalf("expected data uri markdown, got %s", content)
	}
	if strings.Contains(content, "poll_url") || strings.Contains(content, "task_id") {
		t.Fatalf("upstream task fields should be stripped: %s", content)
	}
	if polls < 2 {
		t.Fatalf("expected polling, got %d", polls)
	}
}

func TestAdaptManjuBananaChatCompletionResponseURLMarkdown(t *testing.T) {
	imageBody := []byte("pngbytes")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(imageBody)
	}))
	defer srv.Close()

	body := []byte(`{
		"choices":[{"message":{"role":"assistant","content":"![Generated Image](` + srv.URL + `/img.png)"}}],
		"task_id":"x",
		"poll_url":"` + srv.URL + `/poll"
	}`)
	info := &relaycommon.RelayInfo{OriginModelName: "manju-gemini-banana-2.0-1/2k"}
	out, err := AdaptManjuBananaChatCompletionResponse(context.Background(), info, body)
	if err != nil {
		t.Fatalf("adapt: %v", err)
	}
	if !strings.Contains(string(out), "data:image/png;base64,") {
		t.Fatalf("expected png data uri, got %s", string(out))
	}
}

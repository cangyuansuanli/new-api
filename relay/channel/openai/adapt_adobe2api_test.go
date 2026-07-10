package openai

import (
	"bytes"
	"encoding/json"
	"mime"
	"mime/multipart"
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

func TestConvertAdobe2APIImageRequestMapsGenerationParams(t *testing.T) {
	n := uint(2)
	request := dto.ImageRequest{
		Model:   "nano-banana-pro",
		Prompt:  "a blue icon",
		N:       &n,
		Size:    "16:9",
		Quality: "high",
	}
	request.Extra = map[string]json.RawMessage{
		"image_size": json.RawMessage(`"2K"`),
	}

	bodyAny, err := ConvertAdobe2APIImageRequest(nil, &relaycommon.RelayInfo{
		OriginModelName: "nano-banana-pro",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "nano-banana-pro",
		},
	}, request)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	body := bodyAny.(map[string]any)
	if body["model"] != "nano-banana-pro" {
		t.Fatalf("model = %v", body["model"])
	}
	if body["image_size"] != "2K" || body["output_resolution"] != "2K" {
		t.Fatalf("resolution fields = %v / %v", body["image_size"], body["output_resolution"])
	}
	if body["aspect_ratio"] != "16:9" {
		t.Fatalf("aspect_ratio = %v", body["aspect_ratio"])
	}
	if body["n"] != uint(2) {
		t.Fatalf("n = %v", body["n"])
	}
}

func TestConvertAdobe2APIImageRequestPreservesFrontendOpenAIParams(t *testing.T) {
	n := uint(1)
	request := dto.ImageRequest{
		Model:             "cy-img2-gpt-image-2-4k",
		Prompt:            "cinematic product photo",
		N:                 &n,
		Size:              "3840x2160",
		Quality:           "high",
		Background:        json.RawMessage(`"opaque"`),
		OutputFormat:      json.RawMessage(`"webp"`),
		OutputCompression: json.RawMessage(`80`),
		Moderation:        json.RawMessage(`"low"`),
		ResponseFormat:    "url",
	}

	bodyAny, err := ConvertAdobe2APIImageRequest(nil, &relaycommon.RelayInfo{
		OriginModelName: "cy-img2-gpt-image-2-4k",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:         75,
			UpstreamModelName: "gpt-image",
		},
	}, request)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	body := bodyAny.(map[string]any)
	assertAdobe2APIField(t, body, "size", "3840x2160")
	assertAdobe2APIField(t, body, "quality", "high")
	assertAdobe2APIField(t, body, "background", "opaque")
	assertAdobe2APIField(t, body, "output_format", "webp")
	assertAdobe2APIField(t, body, "output_compression", float64(80))
	assertAdobe2APIField(t, body, "moderation", "low")
	assertAdobe2APIField(t, body, "response_format", "url")
	assertAdobe2APIField(t, body, "aspect_ratio", "16:9")
	assertAdobe2APIField(t, body, "image_size", "4K")
	assertAdobe2APIField(t, body, "output_resolution", "4K")
}

func TestConvertAdobe2APIImageRequestReadsMetadataAndExtraBodyParams(t *testing.T) {
	request := dto.ImageRequest{
		Model:  "nano-banana-pro",
		Prompt: "a clean poster",
		Extra: map[string]json.RawMessage{
			"metadata":   json.RawMessage(`{"aspectRatio":"9:16","outputResolution":"2K","background":"opaque"}`),
			"extra_body": json.RawMessage(`{"output_format":"jpeg","output_compression":72,"google":{"image_config":{"image_size":"4K"}}}`),
		},
	}

	bodyAny, err := ConvertAdobe2APIImageRequest(nil, &relaycommon.RelayInfo{
		OriginModelName: "nano-banana-pro",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "http://45.67.221.45:6001",
		},
	}, request)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	body := bodyAny.(map[string]any)
	assertAdobe2APIField(t, body, "aspect_ratio", "9:16")
	assertAdobe2APIField(t, body, "image_size", "4K")
	assertAdobe2APIField(t, body, "output_resolution", "4K")
	assertAdobe2APIField(t, body, "background", "opaque")
	assertAdobe2APIField(t, body, "output_format", "jpeg")
	assertAdobe2APIField(t, body, "output_compression", float64(72))
}

func TestAdobe2APIImageRelayMatchesChannel75MappedModel(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "cy-img2-gpt-image-2-4k",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:         75,
			UpstreamModelName: "gpt-image",
		},
	}
	if !IsAdobe2APIImageRelay(info) {
		t.Fatal("mapped channel 75 image model should use Adobe2API relay")
	}
	bodyAny, err := ConvertAdobe2APIImageRequest(nil, info, dto.ImageRequest{
		Model:  "cy-img2-gpt-image-2-4k",
		Prompt: "a clean product render",
		Size:   "1024x1024",
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	body := bodyAny.(map[string]any)
	if body["model"] != "gpt-image" {
		t.Fatalf("model = %v", body["model"])
	}
}

func TestAdobe2APIImageRelayReusesManjuBananaModels(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "manju-gemini-banana-2.0-4k",
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:         75,
			ChannelBaseUrl:    "http://45.67.221.45:6001",
			UpstreamModelName: "nano-banana2",
		},
	}
	if !IsAdobe2APIImageRelay(info) {
		t.Fatal("channel 75 mapped manju banana model should use Adobe2API relay")
	}

	bodyAny, err := (&Adaptor{}).ConvertImageRequest(nil, info, dto.ImageRequest{
		Model:   "gemini-banana-2.0-4k",
		Prompt:  "a banana-shaped lamp",
		Size:    "16:9",
		Quality: "auto",
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	body := bodyAny.(map[string]any)
	assertAdobe2APIField(t, body, "model", "nano-banana2")
	assertAdobe2APIField(t, body, "aspect_ratio", "16:9")
	assertAdobe2APIField(t, body, "image_size", "4K")
	assertAdobe2APIField(t, body, "output_resolution", "4K")
	if _, exists := body["messages"]; exists {
		t.Fatalf("Adobe2API image relay should not use Manju chat body: %#v", body)
	}
}

func TestAdobe2APIImageRelayReusesManjuBananaProModel(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "manju-gemini-banana-pro-4k",
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:         75,
			UpstreamModelName: "nano-banana-pro",
		},
	}
	bodyAny, err := ConvertAdobe2APIImageRequest(nil, info, dto.ImageRequest{
		Model:  "gemini-banana-pro-4k",
		Prompt: "a clean product render",
		Size:   "1:1",
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	body := bodyAny.(map[string]any)
	assertAdobe2APIField(t, body, "model", "nano-banana-pro")
	assertAdobe2APIField(t, body, "image_size", "4K")
	assertAdobe2APIField(t, body, "output_resolution", "4K")
}

func TestAdobe2APIImageRelayMatchesChannelBaseURLWithoutChannel75(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "nano-banana-pro",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "http://45.67.221.45:6001",
		},
	}
	if !IsChatImageModel(info.OriginModelName) {
		t.Fatal("test sanity: banana model should normally be a chat image model")
	}
	if !IsAdobe2APIImageRelay(info) {
		t.Fatal("Adobe2API base URL should make banana use the image JSON relay")
	}
}

func TestAdobe2APIImageRelayDoesNotMatchRegularOpenAIModel(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-image-1",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl:    "https://api.openai.com",
			UpstreamModelName: "gpt-image-1",
		},
	}
	if IsAdobe2APIImageRelay(info) {
		t.Fatal("regular OpenAI image model should not use Adobe2API relay")
	}
}

func TestConvertAdobe2APIImageRequestNormalizesUIAspectSize(t *testing.T) {
	bodyAny, err := ConvertAdobe2APIImageRequest(nil, &relaycommon.RelayInfo{
		OriginModelName: "manju-gemini-banana-pro-4k",
	}, dto.ImageRequest{
		Model:   "gemini-banana-pro-4k",
		Prompt:  "poster",
		Size:    "16:9-4k",
		Quality: "high",
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	body := bodyAny.(map[string]any)
	assertAdobe2APIField(t, body, "aspect_ratio", "16:9")
	assertAdobe2APIField(t, body, "image_size", "4K")
	assertAdobe2APIField(t, body, "output_resolution", "4K")
}

func TestConvertAdobe2APIImageRequestIgnoresVideoResolutionOnImage(t *testing.T) {
	bodyAny, err := ConvertAdobe2APIImageRequest(nil, &relaycommon.RelayInfo{
		OriginModelName: "manju-gemini-banana-pro-4k",
	}, dto.ImageRequest{
		Model:  "gemini-banana-pro-4k",
		Prompt: "poster",
		Size:   "3840x2160",
		Extra: map[string]json.RawMessage{
			"aspect_ratio": json.RawMessage(`"16:9"`),
			"resolution":   json.RawMessage(`"720p"`),
		},
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	body := bodyAny.(map[string]any)
	assertAdobe2APIField(t, body, "aspect_ratio", "16:9")
	assertAdobe2APIField(t, body, "image_size", "4K")
}

func TestBuildAdobe2APIImageEditMultipartUsesRepeatedImageField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("model", "gemini-banana-pro-4k")
	_ = writer.WriteField("prompt", "make it cinematic")
	for _, name := range []string{"a.png", "b.png"} {
		part, err := writer.CreateFormFile("image", name)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		_, _ = part.Write([]byte("fakepng"))
	}
	_ = writer.Close()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	info := &relaycommon.RelayInfo{
		OriginModelName: "manju-gemini-banana-pro-4k",
		RelayMode:       relayconstant.RelayModeImagesEdits,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:         75,
			UpstreamModelName: "nano-banana-pro",
		},
	}
	out, err := BuildAdobe2APIImageEditMultipart(c, info, dto.ImageRequest{
		Model:  "gemini-banana-pro-4k",
		Prompt: "make it cinematic",
		Size:   "16:9-4k",
	})
	if err != nil {
		t.Fatalf("build multipart: %v", err)
	}
	if !info.Adobe2APIImageEditMultipart {
		t.Fatal("expected Adobe2APIImageEditMultipart flag")
	}
	contentType := c.Request.Header.Get("Content-Type")
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		t.Fatalf("parse content type: %v", err)
	}
	parsed, err := multipart.NewReader(out, params["boundary"]).ReadForm(1 << 20)
	if err != nil {
		t.Fatalf("parse multipart: %v", err)
	}
	if got := parsed.Value["model"]; len(got) != 1 || got[0] != "nano-banana-pro" {
		t.Fatalf("model = %#v", got)
	}
	if got := parsed.Value["aspect_ratio"]; len(got) != 1 || got[0] != "16:9" {
		t.Fatalf("aspect_ratio = %#v", got)
	}
	if got := parsed.Value["image_size"]; len(got) != 1 || got[0] != "4K" {
		t.Fatalf("image_size = %#v", got)
	}
	if files := parsed.File["image"]; len(files) != 2 {
		t.Fatalf("image files = %d, want 2", len(files))
	}
	if len(parsed.File["image[]"]) != 0 {
		t.Fatalf("unexpected image[] files: %#v", parsed.File["image[]"])
	}
}

func TestConvertAdobe2APIImageRequestAddsReferenceImagesForEdits(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-image",
		RelayMode:       relayconstant.RelayModeImagesEdits,
	}
	request := dto.ImageRequest{
		Model:  "gpt-image",
		Prompt: "make it cinematic",
		Image:  json.RawMessage(`"https://example.com/ref.png"`),
		Size:   "1024x1792",
	}
	bodyAny, err := ConvertAdobe2APIImageRequest(nil, info, request)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	body := bodyAny.(map[string]any)
	if body["aspect_ratio"] != "9:16" {
		t.Fatalf("aspect_ratio = %v", body["aspect_ratio"])
	}
	refs, ok := body["reference_images"].([]string)
	if !ok {
		t.Fatalf("reference_images type = %T", body["reference_images"])
	}
	if len(refs) != 1 || refs[0] != "https://example.com/ref.png" {
		t.Fatalf("reference_images = %#v", refs)
	}
}

func TestConvertAdobe2APIOpenAIChatRequestPreservesVideoExtensions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	rawBody := `{
		"model":"sora2",
		"messages":[{"role":"user","content":"city lights"}],
		"duration":4,
		"aspect_ratio":"16:9",
		"resolution":"720p",
		"image_urls":["https://example.com/ref.png"],
		"reference_mode":"frame"
	}`
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(rawBody))
	c.Request.Header.Set("Content-Type", "application/json")
	var req dto.GeneralOpenAIRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	bodyAny, err := ConvertAdobe2APIOpenAIChatRequest(c, &req, &relaycommon.RelayInfo{
		OriginModelName: "sora2",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "sora2",
		},
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	body := bodyAny.(map[string]any)
	if body["duration"].(float64) != 4 {
		t.Fatalf("duration = %v", body["duration"])
	}
	if body["aspect_ratio"] != "16:9" || body["resolution"] != "720p" {
		t.Fatalf("video options = %v / %v", body["aspect_ratio"], body["resolution"])
	}
	if body["video_reference_mode"] != "frame" {
		t.Fatalf("video_reference_mode = %v", body["video_reference_mode"])
	}
	messages := body["messages"].([]dto.Message)
	parts := messages[0].ParseContent()
	if len(parts) != 2 {
		t.Fatalf("parts len = %d, parts = %#v", len(parts), parts)
	}
	if media := parts[1].GetImageMedia(); media == nil || media.Url != "https://example.com/ref.png" {
		t.Fatalf("image media = %#v", parts[1].GetImageMedia())
	}
	if _, exists := body["image_urls"]; exists {
		t.Fatal("image_urls should not be forwarded after conversion")
	}
}

func assertAdobe2APIField(t *testing.T, body map[string]any, key string, want any) {
	t.Helper()
	if body[key] != want {
		t.Fatalf("%s = %#v, want %#v; body = %#v", key, body[key], want, body)
	}
}

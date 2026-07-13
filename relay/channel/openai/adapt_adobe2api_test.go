package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	n := uint(1)
	request := dto.ImageRequest{
		Model:   "nano-banana-pro",
		Prompt:  "a blue icon",
		N:       &n,
		Size:    "16:9",
		Quality: "medium",
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
	if body["image_size"] != "2K" {
		t.Fatalf("image_size = %v", body["image_size"])
	}
	if body["aspect_ratio"] != "16:9" {
		t.Fatalf("aspect_ratio = %v", body["aspect_ratio"])
	}
	if _, exists := body["n"]; exists {
		t.Fatalf("strict Adobe2API body must not contain n: %#v", body)
	}
}

func TestConvertAdobe2APIImageRequestStripsSellableSKUSuffixFromUpstreamModel(t *testing.T) {
	bodyAny, err := ConvertAdobe2APIImageRequest(nil, &relaycommon.RelayInfo{
		OriginModelName: "adobe-firefly-nano-banana-pro-2k",
	}, dto.ImageRequest{
		Model:  "adobe-firefly-nano-banana-pro-2k",
		Prompt: "a blue icon",
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	body := bodyAny.(map[string]any)
	assertAdobe2APIField(t, body, "model", "nano-banana-pro")
	assertAdobe2APIField(t, body, "image_size", "2K")
}

func TestConvertAdobe2APIGPTImageSKUFallsBackToPublicUpstreamModel(t *testing.T) {
	bodyAny, err := ConvertAdobe2APIImageRequest(nil, &relaycommon.RelayInfo{
		OriginModelName: "adobe-firefly-gpt-image-2-1k",
	}, dto.ImageRequest{
		Model:  "adobe-firefly-gpt-image-2-1k",
		Prompt: "a blue icon",
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	body := bodyAny.(map[string]any)
	assertAdobe2APIField(t, body, "model", "gpt-image")
	assertAdobe2APIField(t, body, "image_size", "1K")
}

func TestConvertAdobe2APIFixedSKUPreservesExactSizeAndBilledTier(t *testing.T) {
	bodyAny, err := ConvertAdobe2APIImageRequest(nil, &relaycommon.RelayInfo{
		OriginModelName: "adobe-firefly-gpt-image-2-4k",
	}, dto.ImageRequest{
		Model:   "adobe-firefly-gpt-image-2-4k",
		Prompt:  "a panoramic poster",
		Size:    "3840x2048",
		Quality: "high",
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	body := bodyAny.(map[string]any)
	assertAdobe2APIField(t, body, "model", "gpt-image")
	assertAdobe2APIField(t, body, "size", "3840x2048")
	assertAdobe2APIField(t, body, "image_size", "4K")
	assertAdobe2APIField(t, body, "quality", "high")
	for _, key := range []string{"aspect_ratio"} {
		if _, exists := body[key]; exists {
			t.Fatalf("exact-size request must not contain %q: %#v", key, body)
		}
	}
}

func TestConvertAdobe2APIExactSizePreservesAllBilledTiers(t *testing.T) {
	tests := []struct {
		model   string
		size    string
		quality string
	}{
		{"adobe-firefly-gpt-image-2-1k", "1024x1024", "low"},
		{"adobe-firefly-gpt-image-2-2k", "3072x1280", "medium"},
		{"adobe-firefly-gpt-image-2-4k", "3840x2048", "high"},
	}
	for _, test := range tests {
		bodyAny, err := ConvertAdobe2APIImageRequest(nil, &relaycommon.RelayInfo{
			OriginModelName: test.model,
		}, dto.ImageRequest{
			Model:   test.model,
			Prompt:  "test",
			Size:    test.size,
			Quality: test.quality,
		})
		if err != nil {
			t.Fatalf("%s: convert: %v", test.model, err)
		}
		body := bodyAny.(map[string]any)
		assertAdobe2APIField(t, body, "size", test.size)
		assertAdobe2APIField(t, body, "image_size", strings.ToUpper(test.model[len(test.model)-2:]))
		assertAdobe2APIField(t, body, "quality", test.quality)
	}
}

func TestConvertAdobe2APIGPTQualityDefaultsToMediumWithoutChangingTier(t *testing.T) {
	for _, quality := range []string{"", "auto", "medium"} {
		bodyAny, err := ConvertAdobe2APIImageRequest(nil, &relaycommon.RelayInfo{
			OriginModelName: "adobe-firefly-gpt-image-2-4k",
		}, dto.ImageRequest{
			Model:   "adobe-firefly-gpt-image-2-4k",
			Prompt:  "test",
			Size:    "3840x2048",
			Quality: quality,
		})
		if err != nil {
			t.Fatalf("quality %q: convert: %v", quality, err)
		}
		body := bodyAny.(map[string]any)
		assertAdobe2APIField(t, body, "image_size", "4K")
		assertAdobe2APIField(t, body, "quality", "medium")
	}
}

func TestAdobe2APIImageRelayMatchesDedicatedFireflySKU(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "adobe-firefly-gpt-image-2-4k",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:         75,
			UpstreamModelName: "gpt-image",
		},
	}
	if !IsAdobe2APIImageRelay(info) {
		t.Fatal("dedicated Adobe Firefly SKU should use Adobe2API image relay")
	}
}

func TestValidateAdobe2APIImageInputsRejectsOversizedMultipartBeforeQueue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = &http.Request{MultipartForm: &multipart.Form{
		File: map[string][]*multipart.FileHeader{
			"image": {{Filename: "oversized.png", Size: adobe2APIMaxImageBytes + 1}},
		},
	}}
	info := &relaycommon.RelayInfo{
		OriginModelName: "adobe-firefly-gpt-image-2-2k",
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: 75},
	}

	err := ValidateAdobe2APIImageInputs(c, info, dto.ImageRequest{})
	if err == nil || !strings.Contains(err.Error(), "max 10MB") {
		t.Fatalf("expected 10MB validation error, got %v", err)
	}
}

func TestValidateAdobe2APIImageInputsRejectsTooManyAndOversizedInlineReferences(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "adobe-firefly-gpt-image-2-2k",
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: 75},
	}
	maximum := make([]string, 9)
	for i := range maximum {
		maximum[i] = fmt.Sprintf("https://example.com/reference-%d.png", i)
	}
	maximumRaw, _ := json.Marshal(maximum)
	err := ValidateAdobe2APIImageInputs(nil, info, dto.ImageRequest{Images: maximumRaw})
	if err != nil {
		t.Fatalf("expected nine references to pass validation, got %v", err)
	}

	tooMany := make([]string, 10)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("https://example.com/reference-%d.png", i)
	}
	tooManyRaw, _ := json.Marshal(tooMany)
	err = ValidateAdobe2APIImageInputs(nil, info, dto.ImageRequest{Images: tooManyRaw})
	if err == nil || !strings.Contains(err.Error(), "too many images, max 9") {
		t.Fatalf("expected image count validation error, got %v", err)
	}

	encodedBytes := (adobe2APIMaxImageBytes+1)*4/3 + 8
	oversized := `"data:image/png;base64,` + strings.Repeat("A", int(encodedBytes)) + `"`
	err = ValidateAdobe2APIImageInputs(nil, info, dto.ImageRequest{Image: json.RawMessage(oversized)})
	if err == nil || !strings.Contains(err.Error(), "max 10MB") {
		t.Fatalf("expected inline size validation error, got %v", err)
	}
}

func TestAdobe2APIImageURLAliasesAreDeduplicatedBeforeValidationAndForwarding(t *testing.T) {
	seven := make([]string, 7)
	for i := range seven {
		seven[i] = fmt.Sprintf("https://example.com/reference-%d.png", i)
	}
	sevenRaw, _ := json.Marshal(seven)
	request := dto.ImageRequest{
		Model:  "gpt-image-2-2k",
		Prompt: "seven references",
		Size:   "2048x2048",
		Images: sevenRaw,
		Extra: map[string]json.RawMessage{
			"imageUrls": sevenRaw,
		},
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "adobe-firefly-gpt-image-2-2k",
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:         75,
			UpstreamModelName: "gpt-image",
		},
	}

	if err := ValidateAdobe2APIImageInputs(nil, info, request); err != nil {
		t.Fatalf("seven duplicated aliases should pass validation: %v", err)
	}
	bodyAny, err := ConvertAdobe2APIImageRequest(nil, info, request)
	if err != nil {
		t.Fatalf("convert aliases: %v", err)
	}
	refs, ok := bodyAny.(map[string]any)["images"].([]string)
	if !ok || len(refs) != 7 {
		t.Fatalf("forwarded references = %#v, want seven unique URLs", refs)
	}

	ten := make([]string, 10)
	for i := range ten {
		ten[i] = fmt.Sprintf("https://example.com/reference-%d.png", i)
	}
	tenRaw, _ := json.Marshal(ten)
	request.Images = nil
	request.Extra["imageUrls"] = tenRaw
	if err := ValidateAdobe2APIImageInputs(nil, info, request); err == nil || !strings.Contains(err.Error(), "too many images, max 9") {
		t.Fatalf("ten imageUrls references should be rejected, got %v", err)
	}
}

func TestValidateAdobe2APIImageInputsRejectsGPTContractBeforeQueue(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "adobe-firefly-gpt-image-2-1k",
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: 75},
	}
	for _, request := range []dto.ImageRequest{
		{Model: "adobe-firefly-gpt-image-2-1k", Size: "2048x1024", Quality: "low"},
		{Model: "adobe-firefly-gpt-image-2-1k", Size: "8:1", Quality: "low"},
		{Model: "adobe-firefly-gpt-image-2-1k", Size: "1024x1024", Quality: "ultra"},
	} {
		if err := ValidateAdobe2APIImageInputs(nil, info, request); err == nil {
			t.Fatalf("expected pre-queue contract rejection for %#v", request)
		}
	}
}

func TestConvertAdobe2APIImageRequestPreservesGPTSizeAndStripsUnsupportedParams(t *testing.T) {
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
	assertAdobe2APIField(t, body, "image_size", "4K")
	assertAdobe2APIField(t, body, "quality", "high")
	for _, key := range []string{"n", "aspect_ratio", "background", "output_format", "output_compression", "moderation", "response_format", "output_resolution"} {
		if _, exists := body[key]; exists {
			t.Fatalf("strict Adobe2API body contains unsupported field %q: %#v", key, body)
		}
	}
}

func TestConvertAdobe2APIGPTRatioStillUsesCalculatedSizeContract(t *testing.T) {
	bodyAny, err := ConvertAdobe2APIImageRequest(nil, &relaycommon.RelayInfo{
		OriginModelName: "adobe-firefly-gpt-image-2-4k",
	}, dto.ImageRequest{
		Model:   "adobe-firefly-gpt-image-2-4k",
		Prompt:  "a panoramic poster",
		Size:    "15:8",
		Quality: "high",
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	body := bodyAny.(map[string]any)
	assertAdobe2APIField(t, body, "image_size", "4K")
	assertAdobe2APIField(t, body, "aspect_ratio", "15:8")
	assertAdobe2APIField(t, body, "quality", "high")
	for _, key := range []string{"size"} {
		if _, exists := body[key]; exists {
			t.Fatalf("ratio request must not contain exact field %q: %#v", key, body)
		}
	}
}

func TestConvertAdobe2APIGPTRatioRejectsProviderLimit(t *testing.T) {
	_, err := ConvertAdobe2APIImageRequest(nil, &relaycommon.RelayInfo{
		OriginModelName: "adobe-firefly-gpt-image-2-4k",
	}, dto.ImageRequest{
		Model:  "adobe-firefly-gpt-image-2-4k",
		Prompt: "a panoramic poster",
		Size:   "8:1",
	})
	if err == nil || !strings.Contains(err.Error(), "3:1") {
		t.Fatalf("expected provider ratio rejection, got %v", err)
	}
}

func TestConvertAdobe2APIExactSizeRequiresBilledTier(t *testing.T) {
	_, err := ConvertAdobe2APIImageRequest(nil, &relaycommon.RelayInfo{
		OriginModelName: "adobe-firefly-gpt-image-2",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-image",
		},
	}, dto.ImageRequest{
		Model:   "adobe-firefly-gpt-image-2",
		Prompt:  "test",
		Size:    "2048x2048",
		Quality: "medium",
	})
	if err == nil || !strings.Contains(err.Error(), "fixed 1K, 2K, or 4K") {
		t.Fatalf("expected billed-tier rejection, got %v", err)
	}
}

func TestConvertAdobe2APIImageRequestReadsMetadataAndExtraBodyParams(t *testing.T) {
	request := dto.ImageRequest{
		Model:  "nano-banana-pro",
		Prompt: "a clean poster",
		Extra: map[string]json.RawMessage{
			"metadata":   json.RawMessage(`{"aspectRatio":"9:16","outputResolution":"4K","background":"opaque"}`),
			"extra_body": json.RawMessage(`{"output_format":"jpeg","output_compression":72,"google":{"image_config":{"image_size":"4K"}}}`),
		},
	}

	bodyAny, err := ConvertAdobe2APIImageRequest(nil, &relaycommon.RelayInfo{
		OriginModelName: "adobe-firefly-nano-banana-pro-4k",
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
	for _, key := range []string{"output_resolution", "background", "output_format", "output_compression"} {
		if _, exists := body[key]; exists {
			t.Fatalf("strict Adobe2API body contains unsupported field %q: %#v", key, body)
		}
	}
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
		Size:   "3840x2160",
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

func TestWriteAdobe2APIImageEditFormPreservesExactGPTSize(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	err := writeAdobe2APIImageEditFormFields(writer, &relaycommon.RelayInfo{
		OriginModelName: "adobe-firefly-gpt-image-2-4k",
	}, dto.ImageRequest{
		Prompt:  "edit",
		Size:    "3840x2048",
		Quality: "high",
	}, "gpt-image")
	if err != nil {
		t.Fatalf("write fields: %v", err)
	}
	_ = writer.Close()
	parsed, err := multipart.NewReader(&body, writer.Boundary()).ReadForm(1 << 20)
	if err != nil {
		t.Fatalf("parse multipart: %v", err)
	}
	if got := parsed.Value["size"]; len(got) != 1 || got[0] != "3840x2048" {
		t.Fatalf("size = %#v", got)
	}
	if got := parsed.Value["image_size"]; len(got) != 1 || got[0] != "4K" {
		t.Fatalf("image_size = %#v", got)
	}
	if got := parsed.Value["quality"]; len(got) != 1 || got[0] != "high" {
		t.Fatalf("quality = %#v", got)
	}
	if len(parsed.Value["aspect_ratio"]) != 0 {
		t.Fatalf("exact request must not include ratio fields: %#v", parsed.Value)
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
		Size:   "9:16",
	}
	bodyAny, err := ConvertAdobe2APIImageRequest(nil, info, request)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	body := bodyAny.(map[string]any)
	if body["aspect_ratio"] != "9:16" {
		t.Fatalf("aspect_ratio = %v", body["aspect_ratio"])
	}
	refs, ok := body["images"].([]string)
	if !ok {
		t.Fatalf("images type = %T", body["images"])
	}
	if len(refs) != 1 || refs[0] != "https://example.com/ref.png" {
		t.Fatalf("images = %#v", refs)
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

func TestAdobe2APIVideoRelayMatchesChannel75MappedModels(t *testing.T) {
	for _, model := range []string{"adobe-sora2", "adobe-sora2-pro", "adobe-veo31", "adobe-veo31-ref", "adobe-veo31-fast"} {
		info := &relaycommon.RelayInfo{
			OriginModelName: model,
			ChannelMeta: &relaycommon.ChannelMeta{
				ChannelId:         75,
				UpstreamModelName: strings.TrimPrefix(model, "adobe-"),
			},
		}
		if !IsAdobe2APIVideoChatRelay(info) {
			t.Fatalf("channel 75 model %q should use Adobe2API video chat relay", model)
		}
	}
}

func TestConvertAdobe2APIOpenAIChatRequestKeepsVeoParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	rawBody := `{
		"model":"veo31-fast",
		"messages":[{"role":"user","content":"a car crossing a rainy city"}],
		"duration":8,
		"aspect_ratio":"16:9",
		"resolution":"720p",
		"generate_audio":false
	}`
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(rawBody))
	c.Request.Header.Set("Content-Type", "application/json")
	var req dto.GeneralOpenAIRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	bodyAny, err := ConvertAdobe2APIOpenAIChatRequest(c, &req, &relaycommon.RelayInfo{
		OriginModelName: "adobe-veo31-fast",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:         75,
			UpstreamModelName: "veo31-fast",
		},
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	body := bodyAny.(map[string]any)
	assertAdobe2APIField(t, body, "model", "veo31-fast")
	assertAdobe2APIField(t, body, "duration", float64(8))
	assertAdobe2APIField(t, body, "aspect_ratio", "16:9")
	assertAdobe2APIField(t, body, "resolution", "720p")
	assertAdobe2APIField(t, body, "generate_audio", false)
}

func assertAdobe2APIField(t *testing.T, body map[string]any, key string, want any) {
	t.Helper()
	if body[key] != want {
		t.Fatalf("%s = %#v, want %#v; body = %#v", key, body[key], want, body)
	}
}

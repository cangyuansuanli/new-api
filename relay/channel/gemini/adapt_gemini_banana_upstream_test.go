package gemini

import (
	"bytes"
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
	"github.com/stretchr/testify/require"
)

func TestIsGeminiBananaUpstreamImage(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "manju-gemini-banana-pro-4k",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gemini-3-pro-image-preview",
		},
	}
	if !IsGeminiBananaUpstreamImage(info) {
		t.Fatal("expected true for manju banana on gemini imagine model")
	}
	info.OriginModelName = "gpt-image-1"
	if IsGeminiBananaUpstreamImage(info) {
		t.Fatal("expected false for non-banana model")
	}
}

func TestConvertGeminiBananaImageRequest4K(t *testing.T) {
	out, err := ConvertGeminiBananaImageRequest(nil, dto.ImageRequest{
		Prompt:  "a cat",
		Size:    "1:1",
		Quality: "high",
	})
	if err != nil {
		t.Fatalf("ConvertGeminiBananaImageRequest: %v", err)
	}
	if len(out.Contents) != 1 || out.Contents[0].Parts[0].Text != "a cat" {
		t.Fatalf("unexpected contents: %+v", out.Contents)
	}
	var imageConfig map[string]any
	if err := common.Unmarshal(out.GenerationConfig.ImageConfig, &imageConfig); err != nil {
		t.Fatalf("unmarshal image_config: %v", err)
	}
	if imageConfig["aspectRatio"] != "1:1" || imageConfig["imageSize"] != "4K" {
		t.Fatalf("imageConfig = %v", imageConfig)
	}
}

func TestConvertGeminiBananaImageRequestRequiresInput(t *testing.T) {
	_, err := ConvertGeminiBananaImageRequest(nil, dto.ImageRequest{Prompt: "   "})
	if err == nil || !strings.Contains(err.Error(), "prompt or reference image is required") {
		t.Fatalf("err = %v", err)
	}
}

func TestConvertGeminiBananaImageRequestWithReferenceImage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("prompt", "make it blue"))
	part, err := writer.CreateFormFile("image", "input.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("fakepng"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	out, err := ConvertGeminiBananaImageRequest(c, dto.ImageRequest{Prompt: "make it blue"})
	require.NoError(t, err)
	require.Len(t, out.Contents[0].Parts, 2)
	require.NotNil(t, out.Contents[0].Parts[0].InlineData)
	require.Equal(t, "make it blue", out.Contents[0].Parts[1].Text)
}

func TestGeminiBananaGetRequestURLUsesGenerateContent(t *testing.T) {
	adaptor := Adaptor{}
	info := &relaycommon.RelayInfo{
		OriginModelName: "manju-gemini-banana-pro-4k",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl:    "https://api.0lll0.cn",
			UpstreamModelName: "gemini-3-pro-image-preview",
		},
		RelayMode: relayconstant.RelayModeImagesGenerations,
	}
	url, err := adaptor.GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL: %v", err)
	}
	want := "https://api.0lll0.cn/v1beta/models/gemini-3-pro-image-preview:generateContent"
	if url != want {
		t.Fatalf("url = %q, want %q", url, want)
	}
}

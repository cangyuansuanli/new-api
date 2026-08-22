package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
)

func TestConvertAiclubImageRequestMapsGPTBody(t *testing.T) {
	bodyAny, err := ConvertAiclubImageRequest(nil, &relaycommon.RelayInfo{
		OriginModelName: "cy-ac-gpt-image-2-2k",
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-image2-2k-high",
		},
	}, dto.ImageRequest{
		Model:  "gpt-image2-2k-high",
		Prompt: "sunrise over mountains",
		Size:   "16:9",
	})
	if err != nil {
		t.Fatalf("ConvertAiclubImageRequest: %v", err)
	}
	body := bodyAny.(map[string]any)
	if body["model"] != "gpt-image2-2k-high" {
		t.Fatalf("model = %v", body["model"])
	}
	if body["aspect_ratio"] != "16:9" {
		t.Fatalf("aspect_ratio = %v", body["aspect_ratio"])
	}
	if _, ok := body["quality"]; ok {
		t.Fatal("quality must not be forwarded to aiclub")
	}
	if _, ok := body["resolution"]; ok {
		t.Fatal("resolution must not be sent for fixed SKU")
	}
}

func TestConvertAiclubImageRequestMapsBananaBody(t *testing.T) {
	bodyAny, err := ConvertAiclubImageRequest(nil, &relaycommon.RelayInfo{
		OriginModelName: "cy-ac-nano-banana-pro-1k",
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "Nano-Banana-Pro-1k",
		},
	}, dto.ImageRequest{
		Model:  "Nano-Banana-Pro-1k",
		Prompt: "product photo",
		Size:   "1:1",
	})
	if err != nil {
		t.Fatalf("ConvertAiclubImageRequest: %v", err)
	}
	body := bodyAny.(map[string]any)
	if body["model"] != "Nano-Banana-Pro-1k" {
		t.Fatalf("model = %v", body["model"])
	}
	if body["aspect_ratio"] != "1:1" {
		t.Fatalf("aspect_ratio = %v", body["aspect_ratio"])
	}
}

func TestConvertAiclubImageRequestRenamesSizeToAspectRatio(t *testing.T) {
	bodyAny, err := ConvertAiclubImageRequest(nil, &relaycommon.RelayInfo{
		OriginModelName: "cy-ac-gpt-image-2-2k",
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-image2-2k-high",
		},
	}, dto.ImageRequest{
		Model:  "gpt-image2-2k-high",
		Prompt: "portrait",
		Size:   "1536x2048",
	})
	if err != nil {
		t.Fatalf("ConvertAiclubImageRequest: %v", err)
	}
	body := bodyAny.(map[string]any)
	if body["aspect_ratio"] != "1536x2048" {
		t.Fatalf("aspect_ratio = %v", body["aspect_ratio"])
	}
	if _, ok := body["size"]; ok {
		t.Fatalf("size must not be forwarded: %#v", body)
	}
}

func TestConvertAiclubImageRequestPrefersExplicitAspectRatio(t *testing.T) {
	bodyAny, err := ConvertAiclubImageRequest(nil, &relaycommon.RelayInfo{
		OriginModelName: "cy-ac-gpt-image-2-2k",
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-image2-2k-high",
		},
	}, dto.ImageRequest{
		Model:  "gpt-image2-2k-high",
		Prompt: "portrait",
		Extra: map[string]json.RawMessage{
			"aspect_ratio": json.RawMessage(`"9:16"`),
		},
	})
	if err != nil {
		t.Fatalf("ConvertAiclubImageRequest: %v", err)
	}
	body := bodyAny.(map[string]any)
	if body["aspect_ratio"] != "9:16" {
		t.Fatalf("aspect_ratio = %v", body["aspect_ratio"])
	}
	if _, ok := body["size"]; ok {
		t.Fatalf("size must not be forwarded when aspect_ratio is explicit: %#v", body)
	}
}

func TestConvertAiclubImageRequestRejectsTooManyReferences(t *testing.T) {
	images := make([]string, 7)
	for i := range images {
		images[i] = fmt.Sprintf("https://example.com/ref-%d.png", i)
	}
	imagesRaw, _ := json.Marshal(images)
	_, err := ConvertAiclubImageRequest(nil, &relaycommon.RelayInfo{
		OriginModelName: "cy-ac-gpt-image-2-1k",
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-image2-1k-high",
		},
	}, dto.ImageRequest{
		Model:  "gpt-image2-1k-high",
		Prompt: "collage",
		Size:   "1:1",
		Images: imagesRaw,
	})
	if err == nil || !strings.Contains(err.Error(), "max 6") {
		t.Fatalf("expected max 6 error, got %v", err)
	}
}

func TestAdaptAiclubImageResponsePollsUntilCompleted(t *testing.T) {
	var polls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/v1/videos/"):
			n := atomic.AddInt32(&polls, 1)
			if n < 2 {
				_, _ = w.Write([]byte(`{"id":"image-test","status":"in_progress"}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":"image-test","status":"completed","metadata":{"result_url":"https://cdn.example.com/out.png"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	info := &relaycommon.RelayInfo{
		OriginModelName: "cy-ac-gpt-image-2-2k",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: srv.URL,
			ApiKey:         "test-key",
		},
	}
	t.Setenv("AICLUB_IMAGE_POLL_INTERVAL", "1")

	createBody := []byte(`{"id":"image-test","object":"image.generation","status":"queued"}`)
	out, err := adaptAiclubImageResponse(context.Background(), info, createBody)
	if err != nil {
		t.Fatalf("adaptAiclubImageResponse: %v", err)
	}
	var imageResp dto.ImageResponse
	if unmarshalErr := json.Unmarshal(out, &imageResp); unmarshalErr != nil {
		t.Fatalf("unmarshal: %v", unmarshalErr)
	}
	if len(imageResp.Data) != 1 || imageResp.Data[0].Url != "https://cdn.example.com/out.png" {
		t.Fatalf("unexpected image response: %+v", imageResp)
	}
	if atomic.LoadInt32(&polls) < 2 {
		t.Fatalf("expected polling, got %d polls", polls)
	}
}

func TestAdaptAiclubImageResponseFailsOnUpstreamError(t *testing.T) {
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: "https://example.com"}}
	_, err := adaptAiclubImageResponse(context.Background(), info, []byte(`{"status":"failed","error":{"message":"quota_exhausted"}}`))
	if err == nil || !strings.Contains(err.Error(), "quota_exhausted") {
		t.Fatalf("expected quota error, got %v", err)
	}
}

func TestIsAiclubImageRelay(t *testing.T) {
	if !IsAiclubImageRelay(&relaycommon.RelayInfo{OriginModelName: "cy-ac-gpt-image-2-4k"}) {
		t.Fatal("expected aiclub relay")
	}
	if IsAiclubImageRelay(&relaycommon.RelayInfo{OriginModelName: "adobe-firefly-gpt-image-2-4k"}) {
		t.Fatal("adobe should not match aiclub relay")
	}
}

func TestAiclubGetRequestURLUsesVideosEndpoint(t *testing.T) {
	adaptor := &Adaptor{}
	url, err := adaptor.GetRequestURL(&relaycommon.RelayInfo{
		OriginModelName: "cy-ac-gpt-image-2-1k",
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.aiclub.cv",
			ChannelType:    1,
		},
	})
	if err != nil {
		t.Fatalf("GetRequestURL: %v", err)
	}
	if url != "https://api.aiclub.cv/v1/videos" {
		t.Fatalf("url = %q", url)
	}
}

func TestAiclubPollTimeoutHonorsEnv(t *testing.T) {
	t.Setenv("AICLUB_IMAGE_POLL_TIMEOUT", "1")
	if got := aiclubImagePollTimeout(); got != time.Second {
		t.Fatalf("timeout = %v", got)
	}
}

package openai

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
)

func TestIsChatAudioModel(t *testing.T) {
	cases := []struct {
		model string
		want  bool
	}{
		{"gemini-music", true},
        {"cy-au1-gemini-music", true},
        {"oairegbox-gemini-music", true},
		{"gpt-4o", false},
		{"gemini-image", false},
	}
	for _, tc := range cases {
		if got := IsChatAudioModel(tc.model); got != tc.want {
			t.Fatalf("IsChatAudioModel(%q) = %v, want %v", tc.model, got, tc.want)
		}
	}
}

func TestConvertAudioGenerationRequestForChat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeAudioGenerations,
		OriginModelName: "cy-au1-gemini-music",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gemini-music",
		},
	}
	request := dto.AudioGenerationRequest{
		Model:  "cy-au1-gemini-music",
		Prompt: "upbeat electronic BGM for a product ad",
	}
	out, err := ConvertAudioGenerationRequestForChat(c, info, request)
	if err != nil {
		t.Fatalf("ConvertAudioGenerationRequestForChat: %v", err)
	}
	chatReq, ok := out.(dto.GeneralOpenAIRequest)
	if !ok {
		t.Fatalf("expected GeneralOpenAIRequest, got %T", out)
	}
	if chatReq.Model != "gemini-music" {
		t.Fatalf("model = %q, want gemini-music", chatReq.Model)
	}
	if len(chatReq.Messages) != 1 || chatReq.Messages[0].Content != request.Prompt {
		t.Fatalf("messages = %#v", chatReq.Messages)
	}
	if chatReq.Stream == nil || *chatReq.Stream {
		t.Fatalf("stream = %#v, want false", chatReq.Stream)
	}
}

func TestAudioURLFromChatContent(t *testing.T) {
	content := "✅ 音乐生成完成\n\n[⬇️ 点击下载音乐](https://download.oaibox.xyz/v1/audio/aud-123/content)\n\nhttps://download.oaibox.xyz/v1/audio/aud-123/content"
	url, err := audioURLFromChatContent(content)
	if err != nil {
		t.Fatalf("audioURLFromChatContent: %v", err)
	}
	want := "https://download.oaibox.xyz/v1/audio/aud-123/content"
	if url != want {
		t.Fatalf("url = %q, want %q", url, want)
	}
}

func TestAudioURLFromChatContentMissing(t *testing.T) {
	if _, err := audioURLFromChatContent("no url here"); err == nil {
		t.Fatal("expected error for missing url")
	}
}

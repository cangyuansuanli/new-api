package openai

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/audiovendor"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

var (
	markdownAudioURL = regexp.MustCompile(`\((https?://[^)\s]+)\)`)
	plainAudioURL    = regexp.MustCompile(`https?://[^\s<>"']+/v1/audio/[^\s<>"']+`)
)

// IsChatAudioModel reports models that generate music via upstream chat/completions.
func IsChatAudioModel(model string) bool {
	return audiovendor.IsGeminiMusicOriginModel(model)
}

func chatAudioGetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(info.ChannelBaseUrl), "/")
	if base == "" {
		return "/v1/chat/completions", nil
	}
	return base + "/v1/chat/completions", nil
}

func resolveChatAudioUpstreamModel(info *relaycommon.RelayInfo, request dto.AudioGenerationRequest) string {
	if info != nil && info.ChannelMeta != nil && strings.TrimSpace(info.UpstreamModelName) != "" {
		return strings.TrimSpace(info.UpstreamModelName)
	}
	if strings.TrimSpace(request.Model) != "" {
		return strings.TrimSpace(request.Model)
	}
	if info != nil && strings.TrimSpace(info.OriginModelName) != "" {
		return strings.TrimSpace(info.OriginModelName)
	}
	return ""
}

// ConvertAudioGenerationRequestForChat maps unified audio API to upstream chat/completions.
func ConvertAudioGenerationRequestForChat(_ *gin.Context, info *relaycommon.RelayInfo, request dto.AudioGenerationRequest) (any, error) {
	prompt := strings.TrimSpace(request.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	modelName := resolveChatAudioUpstreamModel(info, request)
	if modelName == "" {
		return nil, fmt.Errorf("model is required")
	}

	stream := false
	if request.Stream != nil {
		stream = *request.Stream
	}

	return dto.GeneralOpenAIRequest{
		Model: modelName,
		Messages: []dto.Message{
			{Role: "user", Content: prompt},
		},
		Stream: common.GetPointer(stream),
	}, nil
}

// OpenaiChatAudioHandler converts upstream chat music response to AudioGenerationResponse.
func OpenaiChatAudioHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	var simpleResponse dto.OpenAITextResponse
	if err := common.Unmarshal(responseBody, &simpleResponse); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if oaiError := simpleResponse.GetOpenAIError(); oaiError != nil && oaiError.Type != "" {
		return nil, types.WithOpenAIError(*oaiError, resp.StatusCode)
	}

	content, err := readChatAudioContent(resp, responseBody)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusBadGateway)
	}

	audioURL, err := audioURLFromChatContent(content)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusBadGateway)
	}

	usage := simpleResponse.Usage
	if usage.TotalTokens == 0 {
		usage = dto.Usage{TotalTokens: 1, PromptTokens: 1}
	}

	audioResp := dto.AudioGenerationResponse{
		Created: time.Now().Unix(),
		Data:    []dto.AudioGenerationData{{URL: audioURL}},
	}
	out, err := common.Marshal(audioResp)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	service.IOCopyBytesGracefully(c, resp, out)

	normalizeOpenAIUsage(&usage)
	applyUsagePostProcessing(info, &usage, responseBody)
	return &usage, nil
}

func readChatAudioContent(resp *http.Response, body []byte) (string, error) {
	if resp == nil {
		return extractChatAudioMessageContent(body), nil
	}
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if !strings.Contains(contentType, "text/event-stream") {
		return extractChatAudioMessageContent(body), nil
	}

	var content strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		part, err := contentFromChatAudioChunk([]byte(data))
		if err != nil {
			return "", err
		}
		content.WriteString(part)
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if content.Len() > 0 {
		return content.String(), nil
	}
	return extractChatAudioMessageContent(body), nil
}

type chatAudioChunk struct {
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
	Message string `json:"message"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func contentFromChatAudioChunk(data []byte) (string, error) {
	var chunk chatAudioChunk
	if err := common.Unmarshal(data, &chunk); err != nil {
		return "", err
	}
	if chunk.Error != nil && strings.TrimSpace(chunk.Error.Message) != "" {
		return "", fmt.Errorf("%s", strings.TrimSpace(chunk.Error.Message))
	}
	if len(chunk.Choices) == 0 {
		if strings.TrimSpace(chunk.Message) != "" {
			return "", fmt.Errorf("%s", strings.TrimSpace(chunk.Message))
		}
		return "", nil
	}
	return chunk.Choices[0].Delta.Content + chunk.Choices[0].Message.Content, nil
}

func extractChatAudioMessageContent(body []byte) string {
	var payload struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := common.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if len(payload.Choices) == 0 {
		return ""
	}
	return payload.Choices[0].Message.Content
}

func audioURLFromChatContent(content string) (string, error) {
	if match := markdownAudioURL.FindStringSubmatch(content); len(match) > 1 {
		return strings.TrimSpace(match[1]), nil
	}
	if match := plainAudioURL.FindString(content); match != "" {
		return strings.TrimRight(strings.TrimSpace(match), ").,;"), nil
	}
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return strings.TrimRight(trimmed, ").,;"), nil
	}
	return "", fmt.Errorf("chat audio response does not contain an audio url")
}

// IsLegacyChatAudioRequest detects deprecated POST /chat/completions music requests.
func IsLegacyChatAudioRequest(c *gin.Context) bool {
	if c == nil || c.Request == nil || c.Request.Method != http.MethodPost {
		return false
	}
	if !strings.HasSuffix(c.Request.URL.Path, "/chat/completions") {
		return false
	}
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return false
	}
	body, err := storage.Bytes()
	if err != nil || len(body) == 0 {
		return false
	}
	var probe struct {
		Model string `json:"model"`
	}
	if err := common.Unmarshal(body, &probe); err != nil {
		return false
	}
	return IsChatAudioModel(probe.Model)
}

// SetChatAudioDeprecationHeaders marks legacy chat music compatibility path.
func SetChatAudioDeprecationHeaders(c *gin.Context) {
	if c == nil {
		return
	}
	c.Header("Deprecation", "true")
	c.Header("Link", `</v1/audio/generations>; rel="successor-version"`)
}

func lastUserMessageText(messages []dto.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "user" {
			continue
		}
		switch content := messages[i].Content.(type) {
		case string:
			if v := strings.TrimSpace(content); v != "" {
				return v
			}
		case []any:
			var parts []string
			for _, item := range content {
				if m, ok := item.(map[string]any); ok && m["type"] == "text" {
					if text := strings.TrimSpace(common.Interface2String(m["text"])); text != "" {
						parts = append(parts, text)
					}
				}
			}
			if joined := strings.TrimSpace(strings.Join(parts, "\n")); joined != "" {
				return joined
			}
		}
	}
	return ""
}

// ConvertLegacyChatAudioToGenerationRequest maps chat/completions body to AudioGenerationRequest.
func ConvertLegacyChatAudioToGenerationRequest(c *gin.Context) (*dto.AudioGenerationRequest, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, err
	}
	body, err := storage.Bytes()
	if err != nil {
		return nil, err
	}
	var chatReq dto.GeneralOpenAIRequest
	if err := common.Unmarshal(body, &chatReq); err != nil {
		return nil, err
	}
	prompt := lastUserMessageText(chatReq.Messages)
	if prompt == "" {
		return nil, fmt.Errorf("messages content is required")
	}
	req := &dto.AudioGenerationRequest{
		Model:  strings.TrimSpace(chatReq.Model),
		Prompt: prompt,
	}
	if chatReq.Stream != nil {
		req.Stream = chatReq.Stream
	}
	return req, nil
}

package adobe

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/defaultvideo"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// TaskAdaptor uses the same OpenAI Video contract as every other standard
// video upstream. Adobe is a vendor identity and model mapping, not a second
// task protocol.
type TaskAdaptor struct {
	defaultvideo.TaskAdaptor
}

var ModelList = []string{
	"adobe-sora2",
	"adobe-sora2-pro",
	"adobe-veo31",
	"adobe-veo31-ref",
	"adobe-veo31-fast",
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return "adobe-video"
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if c == nil || c.Request == nil || !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("Adobe video requests must use application/json"),
			"invalid_request",
			http.StatusBadRequest,
		)
	}
	return a.TaskAdaptor.ValidateRequestAndSetAction(c, info)
}

// BuildRequestURL targets Adobe2API's typed video endpoint. NewAPI keeps its
// public endpoint as POST /v1/videos; only this vendor boundary knows the
// upstream path is /v1/videos/generations.
func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if info == nil || strings.TrimSpace(info.ChannelBaseUrl) == "" {
		return "", fmt.Errorf("adobe video base url is empty")
	}
	return strings.TrimRight(info.ChannelBaseUrl, "/") + "/v1/videos/generations", nil
}

// BuildRequestBody converts NewAPI's broad video request into Adobe2API's
// strict VideoGenerateRequest schema. In particular, size/seconds aliases and
// UI-only fields must not leak to Adobe2API, whose schema rejects unknown keys.
func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, fmt.Errorf("read adobe video request: %w", err)
	}
	body, err := storage.Bytes()
	if err != nil {
		return nil, fmt.Errorf("read adobe video request bytes: %w", err)
	}

	var raw map[string]any
	if err := common.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("adobe video request must be JSON: %w", err)
	}
	req, _ := relaycommon.GetTaskRequest(c)

	modelName := ""
	if info != nil {
		modelName = strings.TrimSpace(info.UpstreamModelName)
	}
	if modelName == "" {
		modelName = strings.TrimSpace(asString(raw["model"]))
	}
	prompt := strings.TrimSpace(asString(raw["prompt"]))
	if prompt == "" {
		prompt = strings.TrimSpace(req.Prompt)
	}
	if modelName == "" || prompt == "" {
		return nil, fmt.Errorf("model and prompt are required")
	}

	out := map[string]any{
		"model":  modelName,
		"prompt": prompt,
	}
	if duration := firstPositiveInt(raw["duration"], req.Duration, raw["seconds"], req.Seconds); duration > 0 {
		out["duration"] = duration
	}
	if ratio := normalizeAspectRatio(asString(raw["aspect_ratio"])); ratio != "" {
		out["aspect_ratio"] = ratio
	} else if ratio := normalizeAspectRatio(asString(raw["size"])); ratio != "" {
		out["aspect_ratio"] = ratio
	}
	for _, key := range []string{"resolution", "negative_prompt", "reference_mode"} {
		if value := strings.TrimSpace(asString(raw[key])); value != "" {
			out[key] = value
		}
	}
	if value, ok := raw["generate_audio"]; ok {
		if audio, valid := asBool(value); valid {
			out["generate_audio"] = audio
		}
	}
	if images := collectImages(raw, req.Images); len(images) > 0 {
		out["images"] = images
	}

	encoded, err := common.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal adobe video request: %w", err)
	}
	c.Request.Header.Set("Content-Type", "application/json")
	return bytes.NewReader(encoded), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	// Pass the outer adaptor so DoTaskApiRequest dispatches to this vendor's
	// BuildRequestURL instead of the embedded default video's URL.
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return ""
	}
}

func firstPositiveInt(values ...any) int {
	for _, value := range values {
		switch typed := value.(type) {
		case int:
			if typed > 0 {
				return typed
			}
		case float64:
			if typed > 0 {
				return int(typed)
			}
		case string:
			if n, err := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(typed, "s"))); err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}

func asBool(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1":
			return true, true
		case "false", "0":
			return false, true
		}
	}
	return false, false
}

func normalizeAspectRatio(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if strings.Contains(raw, "x") {
		raw = strings.Replace(raw, "x", ":", 1)
	}
	if raw == "16:9" || raw == "9:16" {
		return raw
	}
	return ""
}

func collectImages(raw map[string]any, normalized []string) []string {
	if len(normalized) > 0 {
		return normalized
	}
	for _, key := range []string{"images", "image_urls", "reference_image_urls"} {
		value, ok := raw[key]
		if !ok {
			continue
		}
		if single := strings.TrimSpace(asString(value)); single != "" {
			return []string{single}
		}
		if list, ok := value.([]any); ok {
			images := make([]string, 0, len(list))
			for _, item := range list {
				if image := strings.TrimSpace(asString(item)); image != "" {
					images = append(images, image)
				}
			}
			if len(images) > 0 {
				return images
			}
		}
	}
	return nil
}

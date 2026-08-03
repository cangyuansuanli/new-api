package sd5

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

type TaskAdaptor struct {
	defaultvideo.TaskAdaptor
}

var ModelList = []string{
	"cy-sd5-seedance-2.0",
	"cy-sd5-seedance-2.0-fast",
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return "sd5-seedance-video"
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if c == nil || c.Request == nil || !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("SD5 video requests must use application/json"),
			"invalid_request",
			http.StatusBadRequest,
		)
	}
	return a.TaskAdaptor.ValidateRequestAndSetAction(c, info)
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if info == nil || strings.TrimSpace(info.ChannelBaseUrl) == "" {
		return "", fmt.Errorf("SD5 video base url is empty")
	}
	return strings.TrimRight(info.ChannelBaseUrl, "/") + "/v1/videos/generations", nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, fmt.Errorf("read SD5 video request: %w", err)
	}
	body, err := storage.Bytes()
	if err != nil {
		return nil, fmt.Errorf("read SD5 video request bytes: %w", err)
	}

	var raw map[string]any
	if err := common.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("SD5 video request must be JSON: %w", err)
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
	if duration := req.RequestedDurationSeconds(); duration > 0 {
		out["duration"] = duration
	}
	if ratio := normalizeAspectRatio(asString(raw["aspect_ratio"])); ratio != "" {
		out["aspect_ratio"] = ratio
	} else if ratio := normalizeAspectRatio(asString(raw["size"])); ratio != "" {
		out["aspect_ratio"] = ratio
	}
	for _, key := range []string{"resolution", "negative_prompt", "reference_mode", "first_image_url", "last_image_url"} {
		if value := strings.TrimSpace(asString(raw[key])); value != "" {
			out[key] = value
		}
	}
	if value, ok := raw["generate_audio"]; ok {
		if audio, valid := asBool(value); valid {
			out["generate_audio"] = audio
		}
	} else if value, ok := raw["audio"]; ok {
		if audio, valid := asBool(value); valid {
			out["generate_audio"] = audio
		}
	}
	if req.Seed != nil {
		out["seed"] = *req.Seed
	}
	if images := collectMediaImages(raw, req.Images); len(images) > 0 {
		out["images"] = images
		if !hasExplicitFrameInput(raw) {
			out["reference_mode"] = "media"
		}
	}
	sourceCount := 0
	for _, key := range []string{"reference_videos", "reference_audios"} {
		if values := collectStringList(raw[key]); len(values) > 0 {
			out[key] = values
			sourceCount += len(values)
		}
	}
	if sourceCount > 3 {
		return nil, fmt.Errorf("reference_videos and reference_audios must contain at most 3 items combined")
	}

	encoded, err := common.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal SD5 video request: %w", err)
	}
	c.Request.Header.Set("Content-Type", "application/json")
	return bytes.NewReader(encoded), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
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
	if raw == "21:9" || raw == "16:9" || raw == "4:3" || raw == "1:1" || raw == "3:4" || raw == "9:16" {
		return raw
	}
	return ""
}

func hasExplicitFrameInput(raw map[string]any) bool {
	return strings.TrimSpace(asString(raw["first_image_url"])) != "" ||
		strings.TrimSpace(asString(raw["last_image_url"])) != ""
}

func collectMediaImages(raw map[string]any, normalized []string) []string {
	images := make([]string, 0, len(normalized)+1)
	seen := make(map[string]struct{})
	add := func(values ...string) {
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if _, exists := seen[value]; exists {
				continue
			}
			seen[value] = struct{}{}
			images = append(images, value)
		}
	}

	add(asString(raw["image_url"]))
	add(asString(raw["image"]))
	for _, key := range []string{"images", "image_urls", "reference_image_urls"} {
		add(collectStringList(raw[key])...)
	}
	add(normalized...)
	return images
}

func collectStringList(value any) []string {
	if single := strings.TrimSpace(asString(value)); single != "" {
		return []string{single}
	}
	list, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if entry := strings.TrimSpace(asString(item)); entry != "" {
			out = append(out, entry)
		}
	}
	return out
}

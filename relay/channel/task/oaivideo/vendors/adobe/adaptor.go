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
	"cy-adobe-veo-3.1",
	"cy-adobe-veo-3.1-fast",
	"cy-adobe-kling-3.0",
	"cy-adobe-kling-3.0-omni",
	"cy-adobe-gemini-omni-flash",
	"cy-sd5-seedance-2.0",
	"cy-sd5-seedance-2.0-fast",
}

var modelContracts = map[string]struct {
	maxImages        int
	minDuration      int
	maxDuration      int
	allowSeed        bool
	allowAudio       bool
	allowVideoAudio  bool
	allowAssets      bool
	allowStyles      bool
	allowFrames      bool
	allowSingleFrame bool
	maxWidth         int
}{
	"veo-3.1":           {maxImages: 3, minDuration: 4, maxDuration: 8, allowSeed: true, allowAudio: true, allowAssets: true, allowFrames: true, maxWidth: 1920},
	"veo-3.1-fast":      {maxImages: 2, minDuration: 4, maxDuration: 8, allowSeed: true, allowAudio: true, allowFrames: true, maxWidth: 1920},
	"kling-3.0":         {maxImages: 2, minDuration: 3, maxDuration: 15, allowSeed: true, allowAudio: true, allowFrames: true, maxWidth: 1920},
	"kling-3.0-omni":    {maxImages: 3, minDuration: 3, maxDuration: 15, allowSeed: true, allowAudio: true, allowStyles: true, allowFrames: true, maxWidth: 1920},
	"gemini-omni-flash": {maxImages: 4, minDuration: 3, maxDuration: 10, allowStyles: true, allowFrames: true, allowSingleFrame: true, maxWidth: 1280},
	"seedance-2.0":      {maxImages: 9, minDuration: 4, maxDuration: 15, allowSeed: true, allowAudio: true, allowVideoAudio: true, allowAssets: true, maxWidth: 1280},
	"seedance-2.0-fast": {maxImages: 9, minDuration: 4, maxDuration: 15, allowSeed: true, allowAudio: true, allowVideoAudio: true, allowAssets: true, maxWidth: 1280},
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
	if strings.HasPrefix(modelName, "cy-sd5-seedance-") {
		modelName = strings.TrimPrefix(modelName, "cy-sd5-")
	}
	prompt := strings.TrimSpace(asString(raw["prompt"]))
	if prompt == "" {
		prompt = strings.TrimSpace(req.Prompt)
	}
	if modelName == "" || prompt == "" {
		return nil, fmt.Errorf("model and prompt are required")
	}
	contract, ok := modelContracts[modelName]
	if !ok {
		return nil, fmt.Errorf("unsupported Adobe video model: %s", modelName)
	}
	duration := req.RequestedDurationSeconds()
	images := collectImages(raw, req.Images)
	if duration < contract.minDuration || duration > contract.maxDuration {
		return nil, fmt.Errorf("%s duration must be between %d and %d seconds", modelName, contract.minDuration, contract.maxDuration)
	}
	firstImage := strings.TrimSpace(req.FirstImageUrl)
	lastImage := strings.TrimSpace(req.LastImageUrl)
	if firstImage == "" {
		firstImage = strings.TrimSpace(asString(raw["first_image_url"]))
	}
	if lastImage == "" {
		lastImage = strings.TrimSpace(asString(raw["last_image_url"]))
	}
	mode := strings.ToLower(strings.TrimSpace(asString(raw["reference_mode"])))
	if mode == "" {
		if firstImage != "" || lastImage != "" || len(images) == 2 {
			mode = "frame"
		} else if len(images) > 0 {
			mode = "asset"
		}
	}
	if mode == "frame" {
		validPair := firstImage != "" && lastImage != ""
		validSingle := contract.allowSingleFrame && firstImage != "" && lastImage == ""
		if len(images) == 2 && firstImage == "" && lastImage == "" {
			firstImage, lastImage = images[0], images[1]
			validPair = true
		}
		if !contract.allowFrames || (!validPair && !validSingle) {
			return nil, fmt.Errorf("%s requires a first and last frame", modelName)
		}
		images = nil
	} else if mode == "asset" || mode == "image" || mode == "style" {
		if mode == "style" && !contract.allowStyles || (mode != "style" && !contract.allowAssets) {
			return nil, fmt.Errorf("%s does not support %s references", modelName, mode)
		}
		if len(images) > contract.maxImages {
			return nil, fmt.Errorf("%s supports at most %d reference images", modelName, contract.maxImages)
		}
	} else if len(images) > 0 {
		return nil, fmt.Errorf("unsupported reference_mode %q", mode)
	}
	if len(images) > contract.maxImages {
		return nil, fmt.Errorf("%s supports at most %d reference images", modelName, contract.maxImages)
	}
	if len(req.ReferenceVideos)+len(req.ReferenceAudios) > 0 && !contract.allowVideoAudio {
		return nil, fmt.Errorf("%s does not support video or audio references", modelName)
	}
	if req.Seed != nil && !contract.allowSeed {
		return nil, fmt.Errorf("%s does not support seed", modelName)
	}
	if _, present := raw["generate_audio"]; present && !contract.allowAudio {
		return nil, fmt.Errorf("%s does not support audio control", modelName)
	}
	if _, present := raw["audio"]; present && !contract.allowAudio {
		return nil, fmt.Errorf("%s does not support audio control", modelName)
	}

	out := map[string]any{
		"model":  modelName,
		"prompt": prompt,
	}
	if duration > 0 {
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
	if len(images) > 0 {
		out["images"] = images
	}
	if firstImage != "" {
		out["first_image_url"] = firstImage
	}
	if lastImage != "" {
		out["last_image_url"] = lastImage
	}
	for _, key := range []string{"reference_videos", "reference_audios"} {
		if values := collectStringList(raw[key]); len(values) > 0 {
			out[key] = values
		}
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

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

type modelContract struct {
	maxImages        int
	maxSourceMedia   int
	maxTotalMedia    int
	minDuration      int
	maxDuration      int
	allowSeed        bool
	allowAudio       bool
	allowVideoAudio  bool
	allowMedia       bool
	allowAssets      bool
	allowStyles      bool
	allowFrames      bool
	allowSingleFrame bool
	maxWidth         int
}

var modelContracts = map[string]modelContract{
	"veo-3.1":           {maxImages: 3, minDuration: 4, maxDuration: 8, allowSeed: true, allowAudio: true, allowAssets: true, allowFrames: true, maxWidth: 1920},
	"veo-3.1-fast":      {maxImages: 2, minDuration: 4, maxDuration: 8, allowSeed: true, allowAudio: true, allowFrames: true, maxWidth: 1920},
	"kling-3.0":         {maxImages: 2, minDuration: 3, maxDuration: 15, allowSeed: true, allowAudio: true, allowFrames: true, maxWidth: 1920},
	"kling-3.0-omni":    {maxImages: 3, minDuration: 3, maxDuration: 15, allowSeed: true, allowAudio: true, allowStyles: true, allowFrames: true, maxWidth: 1920},
	"gemini-omni-flash": {maxImages: 4, minDuration: 3, maxDuration: 10, allowStyles: true, allowFrames: true, allowSingleFrame: true, maxWidth: 1280},
	"seedance-2.0":      {maxImages: 9, maxSourceMedia: 3, maxTotalMedia: 12, minDuration: 4, maxDuration: 15, allowSeed: true, allowAudio: true, allowVideoAudio: true, allowMedia: true, allowFrames: true, maxWidth: 1280},
	"seedance-2.0-fast": {maxImages: 9, maxSourceMedia: 3, maxTotalMedia: 12, minDuration: 4, maxDuration: 15, allowSeed: true, allowAudio: true, allowVideoAudio: true, allowMedia: true, allowFrames: true, maxWidth: 1280},
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
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}

	modelName := ""
	if info != nil {
		modelName = strings.TrimSpace(info.UpstreamModelName)
	}
	if modelName == "" {
		modelName = strings.TrimSpace(req.Model)
	}
	if strings.HasPrefix(modelName, "cy-sd5-seedance-") {
		modelName = strings.TrimPrefix(modelName, "cy-sd5-")
	}
	prompt := strings.TrimSpace(req.Prompt)
	if modelName == "" || prompt == "" {
		return nil, fmt.Errorf("model and prompt are required")
	}
	contract, ok := modelContracts[modelName]
	if !ok {
		return nil, fmt.Errorf("unsupported Adobe video model: %s", modelName)
	}
	duration := req.RequestedDurationSeconds()
	images := append([]string(nil), req.Images...)
	referenceVideos := append([]string(nil), req.ReferenceVideos...)
	referenceAudios := append([]string(nil), req.ReferenceAudios...)
	if duration < contract.minDuration || duration > contract.maxDuration {
		return nil, fmt.Errorf("%s duration must be between %d and %d seconds", modelName, contract.minDuration, contract.maxDuration)
	}
	firstImage := strings.TrimSpace(req.FirstImageUrl)
	lastImage := strings.TrimSpace(req.LastImageUrl)
	mode := relaycommon.InferReferenceMode(req, "", contract.allowMedia)
	// The public contract only exposes canonical image references. Adobe's
	// Kling Omni lane calls that image group "style" upstream, so infer the
	// vendor mode at this boundary instead of requiring clients to send the
	// vendor-only reference_mode field.
	if mode == "asset" && contract.allowStyles && !contract.allowAssets {
		mode = "style"
	}
	if mode == "frame" {
		if len(referenceVideos)+len(referenceAudios) > 0 {
			return nil, fmt.Errorf("%s frame references cannot be combined with video or audio references", modelName)
		}
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
	} else if mode == "media" {
		if !contract.allowMedia {
			return nil, fmt.Errorf("%s does not support media references", modelName)
		}
		if firstImage != "" || lastImage != "" {
			return nil, fmt.Errorf("%s media references cannot be combined with first or last frames", modelName)
		}
		if len(referenceVideos)+len(referenceAudios) > 0 && len(images) == 0 {
			return nil, fmt.Errorf("%s video or audio references require at least one image reference", modelName)
		}
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
	sourceMediaCount := len(referenceVideos) + len(referenceAudios)
	if sourceMediaCount > 0 && !contract.allowVideoAudio {
		return nil, fmt.Errorf("%s does not support video or audio references", modelName)
	}
	if contract.maxTotalMedia > 0 && len(images)+sourceMediaCount > contract.maxTotalMedia {
		return nil, fmt.Errorf("%s supports at most %d total reference assets", modelName, contract.maxTotalMedia)
	}
	if contract.maxSourceMedia > 0 && sourceMediaCount > contract.maxSourceMedia {
		return nil, fmt.Errorf("%s reference videos and audios support at most %d items combined", modelName, contract.maxSourceMedia)
	}
	if req.Seed != nil && !contract.allowSeed {
		return nil, fmt.Errorf("%s does not support seed", modelName)
	}
	if req.GenerateAudio != nil && !contract.allowAudio {
		return nil, fmt.Errorf("%s does not support audio control", modelName)
	}

	out := map[string]any{
		"model":  modelName,
		"prompt": prompt,
	}
	if duration > 0 {
		out["duration"] = duration
	}
	if ratio := normalizeAspectRatio(req.AspectRatio); ratio != "" {
		out["aspect_ratio"] = ratio
	} else if ratio := normalizeAspectRatio(req.Size); ratio != "" {
		out["aspect_ratio"] = ratio
	}
	if value := strings.TrimSpace(req.Resolution); value != "" {
		out["resolution"] = value
	}
	if mode != "" {
		out["reference_mode"] = mode
	}
	if req.GenerateAudio != nil {
		out["generate_audio"] = *req.GenerateAudio
	}
	if req.Seed != nil {
		out["seed"] = *req.Seed
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
	if len(referenceVideos) > 0 {
		out["reference_videos"] = referenceVideos
	}
	if len(referenceAudios) > 0 {
		out["reference_audios"] = referenceAudios
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

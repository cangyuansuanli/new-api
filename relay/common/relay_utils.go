package common

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

type HasPrompt interface {
	GetPrompt() string
}

type HasImage interface {
	HasImage() bool
}

func GetFullRequestURL(baseURL string, requestURL string, channelType int) string {
	fullRequestURL := fmt.Sprintf("%s%s", baseURL, requestURL)

	if strings.HasPrefix(baseURL, "https://gateway.ai.cloudflare.com") {
		switch channelType {
		case constant.ChannelTypeOpenAI:
			fullRequestURL = fmt.Sprintf("%s%s", baseURL, strings.TrimPrefix(requestURL, "/v1"))
		case constant.ChannelTypeAzure:
			fullRequestURL = fmt.Sprintf("%s%s", baseURL, strings.TrimPrefix(requestURL, "/openai/deployments"))
		}
	}
	return fullRequestURL
}

func GetAPIVersion(c *gin.Context) string {
	query := c.Request.URL.Query()
	apiVersion := query.Get("api-version")
	if apiVersion == "" {
		apiVersion = c.GetString("api_version")
	}
	return apiVersion
}

func createTaskError(err error, code string, statusCode int, localError bool) *dto.TaskError {
	return &dto.TaskError{
		Code:       code,
		Message:    err.Error(),
		StatusCode: statusCode,
		LocalError: localError,
		Error:      err,
	}
}

const promptInputContextKey = "prompt_input"

func storeTaskRequest(c *gin.Context, info *RelayInfo, action string, requestObj TaskSubmitReq) {
	if info.TaskRelayInfo == nil {
		info.TaskRelayInfo = &TaskRelayInfo{}
	}
	info.Action = action
	c.Set("task_request", requestObj)
	StorePromptInput(c, requestObj.GetPrompt())
}

// StorePromptInput keeps the submitted prompt on the request context for task
// persistence and moderation audit logs.
func StorePromptInput(c *gin.Context, prompt string) {
	prompt = strings.TrimSpace(prompt)
	if prompt != "" {
		c.Set(promptInputContextKey, prompt)
	}
}

// PromptInputFromContext returns the prompt captured for the current relay request.
func PromptInputFromContext(c *gin.Context) string {
	if v, ok := c.Get(promptInputContextKey); ok {
		if prompt, ok := v.(string); ok {
			return strings.TrimSpace(prompt)
		}
	}
	if req, err := GetTaskRequest(c); err == nil {
		return strings.TrimSpace(req.GetPrompt())
	}
	return ""
}

func GetTaskRequest(c *gin.Context) (TaskSubmitReq, error) {
	v, exists := c.Get("task_request")
	if !exists {
		return TaskSubmitReq{}, fmt.Errorf("request not found in context")
	}
	req, ok := v.(TaskSubmitReq)
	if !ok {
		return TaskSubmitReq{}, fmt.Errorf("invalid task request type")
	}
	return req, nil
}

func validatePrompt(prompt string) *dto.TaskError {
	if strings.TrimSpace(prompt) == "" {
		return createTaskError(fmt.Errorf("prompt is required"), "invalid_request", http.StatusBadRequest, true)
	}
	return nil
}

func validateMultipartTaskRequest(c *gin.Context, info *RelayInfo, action string) (TaskSubmitReq, error) {
	var req TaskSubmitReq
	if _, err := c.MultipartForm(); err != nil {
		return req, err
	}

	formData := c.Request.PostForm
	req = TaskSubmitReq{
		Prompt:             formData.Get("prompt"),
		Model:              formData.Get("model"),
		Mode:               formData.Get("mode"),
		Image:              formData.Get("image"),
		ImageUrl:           formData.Get("image_url"),
		Size:               formData.Get("size"),
		AspectRatio:        formData.Get("aspect_ratio"),
		Resolution:         formData.Get("resolution"),
		Seconds:            formData.Get("seconds"),
		VideoURL:           formData.Get("video_url"),
		ReferenceVideos:    append([]string(nil), formData["reference_videos"]...),
		ImageUrls:          append([]string(nil), formData["image_urls"]...),
		ReferenceImages:    append([]string(nil), formData["reference_images"]...),
		ReferenceImageUrls: append([]string(nil), formData["reference_image_urls"]...),
		InputReference:     formData.Get("input_reference"),
		Metadata:           make(map[string]interface{}),
	}

	durationStr := formData.Get("duration")
	if durationStr == "" {
		durationStr = formData.Get("seconds")
	}
	if durationStr != "" {
		if duration, err := strconv.Atoi(durationStr); err == nil {
			req.Duration = duration
		}
	}
	if generateAudio := formData.Get("generate_audio"); generateAudio != "" {
		if value, err := strconv.ParseBool(generateAudio); err == nil {
			req.GenerateAudio = &value
		}
	}

	if images := formData["images"]; len(images) > 0 {
		req.Images = images
	}
	normalizeTaskSubmitImages(&req)

	for key, values := range formData {
		if len(values) > 0 && !isKnownTaskField(key) {
			if intVal, err := strconv.Atoi(values[0]); err == nil {
				req.Metadata[key] = intVal
			} else if floatVal, err := strconv.ParseFloat(values[0], 64); err == nil {
				req.Metadata[key] = floatVal
			} else {
				req.Metadata[key] = values[0]
			}
		}
	}
	return req, nil
}

func parseTaskSubmitRequest(c *gin.Context) (TaskSubmitReq, error) {
	if strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "multipart/form-data") {
		return validateMultipartTaskRequest(c, nil, "")
	}
	var req TaskSubmitReq
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return req, err
	}
	return req, nil
}

func ValidateMultipartDirect(c *gin.Context, info *RelayInfo) *dto.TaskError {
	var prompt string
	var model string
	var seconds int
	var size string
	var hasInputReference bool

	req, err := parseTaskSubmitRequest(c)
	if err != nil {
		return createTaskError(err, "invalid_json", http.StatusBadRequest, true)
	}
	if err := normalizeTaskSubmitDuration(&req); err != nil {
		return createTaskError(err, "invalid_duration", http.StatusBadRequest, true)
	}

	prompt = req.Prompt
	model = req.Model
	size = req.Size
	seconds = req.RequestedDurationSeconds()
	normalizeTaskSubmitImages(&req)

	if strings.TrimSpace(req.Model) == "" {
		return createTaskError(fmt.Errorf("model field is required"), "missing_model", http.StatusBadRequest, true)
	}

	if req.HasImage() || multipartTaskHasReferenceFile(c) {
		hasInputReference = true
	}

	if taskErr := validatePrompt(prompt); taskErr != nil {
		return taskErr
	}

	action := constant.TaskActionTextGenerate
	if hasInputReference {
		action = constant.TaskActionGenerate
	}
	if strings.HasPrefix(model, "sora-2") || model == "manju-openai-sora2" {

		if size == "" {
			size = "720x1280"
		}

		if seconds <= 0 {
			seconds = 4
		}

		if (model == "sora-2" || model == "manju-openai-sora2") && !lo.Contains([]string{"720x1280", "1280x720", "1024x1024"}, size) {
			return createTaskError(fmt.Errorf("sora-2 size is invalid"), "invalid_size", http.StatusBadRequest, true)
		}
		if model == "sora-2-pro" && !lo.Contains([]string{"720x1280", "1280x720", "1792x1024", "1024x1792"}, size) {
			return createTaskError(fmt.Errorf("sora-2 size is invalid"), "invalid_size", http.StatusBadRequest, true)
		}
		// OtherRatios 已移到 Sora adaptor 的 EstimateBilling 中设置
	}

	storeTaskRequest(c, info, action, req)

	return nil
}

func isKnownTaskField(field string) bool {
	knownFields := map[string]bool{
		"prompt":               true,
		"model":                true,
		"mode":                 true,
		"image":                true,
		"image_url":            true,
		"images":               true,
		"image_urls":           true,
		"reference_images":     true,
		"reference_image_urls": true,
		"size":                 true,
		"aspect_ratio":         true,
		"resolution":           true,
		"duration":             true,
		"seconds":              true,
		"generate_audio":       true,
		"reference_videos":     true,
		"video_url":            true,
		"input_video":          true,
		"input_video2":         true,
		"metadata":             true,
		"input_reference":      true, // Sora 特有字段
	}
	return knownFields[field]
}

func ValidateBasicTaskRequest(c *gin.Context, info *RelayInfo, action string) *dto.TaskError {
	req, err := parseTaskSubmitRequest(c)
	if err != nil {
		return createTaskError(err, "invalid_request", http.StatusBadRequest, true)
	}
	if err := normalizeTaskSubmitDuration(&req); err != nil {
		return createTaskError(err, "invalid_duration", http.StatusBadRequest, true)
	}

	if taskErr := validatePrompt(req.Prompt); taskErr != nil {
		return taskErr
	}

	normalizeTaskSubmitImages(&req)

	storeTaskRequest(c, info, action, req)
	return nil
}

// normalizeTaskSubmitDuration defines the public video duration contract:
// duration and seconds are aliases for the same integer-second value. Keeping
// both normalized in the internal request lets each vendor adaptor emit only
// the field its upstream protocol expects.
func normalizeTaskSubmitDuration(req *TaskSubmitReq) error {
	if req == nil {
		return nil
	}
	seconds, _ := strconv.Atoi(strings.TrimSpace(req.Seconds))
	if seconds != 0 && req.Duration != 0 && seconds != req.Duration {
		return fmt.Errorf("duration and seconds must have the same value")
	}
	if seconds != 0 {
		req.Duration = seconds
		return nil
	}
	if req.Duration != 0 {
		req.Seconds = strconv.Itoa(req.Duration)
	}
	return nil
}

func multipartTaskHasReferenceFile(c *gin.Context) bool {
	if c == nil || c.Request == nil || c.Request.MultipartForm == nil {
		return false
	}
	for _, field := range []string{"input_reference", "input_reference[]", "images", "images[]", "image"} {
		if len(c.Request.MultipartForm.File[field]) > 0 {
			return true
		}
	}
	return false
}

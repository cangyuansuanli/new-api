package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/imagevendor"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const (
	aiclubMaxInputImages           = 6
	defaultAiclubImagePollInterval = 5 * time.Second
	defaultAiclubImagePollTimeout  = 30 * time.Minute
)

func IsAiclubImageRelay(info *relaycommon.RelayInfo) bool {
	if info == nil {
		return false
	}
	return imagevendor.IsAiclubOriginModel(info.OriginModelName)
}

// ValidateAiclubImageInputs rejects inputs that Aiclub would reject before task enqueue.
func ValidateAiclubImageInputs(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) error {
	if !IsAiclubImageRelay(info) {
		return nil
	}
	if err := imagevendor.ValidateAiclubFixedResolutionSKU(c, info.OriginModelName, &request); err != nil {
		return err
	}
	modelName := resolveAiclubUpstreamModel(info, request.Model)
	if isAiclubGPTImageModelName(modelName) {
		if aspectRatio := aiclubAspectRatio(request); aspectRatio != "" {
			if err := imagevendor.ValidateGPTImageAspectRatio(aspectRatio); err != nil {
				return err
			}
		}
	} else {
		if err := imagevendor.ValidateAdobeBananaAspectRatio(info.OriginModelName, aiclubAspectRatio(request)); err != nil {
			return err
		}
	}
	files, err := collectAdobe2APIMultipartImageFiles(c)
	if err != nil {
		return err
	}
	refs := aiclubReferenceImageValuesForValidation(c, request)
	if err := validateAiclubReferenceURLs(refs); err != nil {
		return err
	}
	if len(files)+len(refs) > aiclubMaxInputImages {
		return fmt.Errorf("too many images, max %d", aiclubMaxInputImages)
	}
	for _, file := range files {
		if file != nil && file.Size > common.ReferenceImageMaxBytes {
			return fmt.Errorf("%s", common.ReferenceImageTooLargeDetail())
		}
	}
	for _, ref := range refs {
		if size, ok := inlineImageDecodedSize(ref); ok && size > common.ReferenceImageMaxBytes {
			return fmt.Errorf("%s", common.ReferenceImageTooLargeDetail())
		}
	}
	return nil
}

func ConvertAiclubImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	if request.N != nil && *request.N > 1 {
		return nil, fmt.Errorf("Aiclub image models only support n=1")
	}
	if info != nil {
		if err := imagevendor.ValidateAiclubFixedResolutionSKU(c, info.OriginModelName, &request); err != nil {
			return nil, err
		}
	}
	modelName := resolveAiclubUpstreamModel(info, request.Model)
	if modelName == "" {
		return nil, fmt.Errorf("model is required")
	}
	body := map[string]any{
		"model":  modelName,
		"prompt": request.Prompt,
	}
	if aspectRatio := aiclubAspectRatio(request); aspectRatio != "" {
		if isAiclubGPTImageModelName(modelName) {
			if err := imagevendor.ValidateGPTImageAspectRatio(aspectRatio); err != nil {
				return nil, err
			}
		} else if info != nil {
			if err := imagevendor.ValidateAdobeBananaAspectRatio(info.OriginModelName, aspectRatio); err != nil {
				return nil, err
			}
		}
		body["aspect_ratio"] = aspectRatio
	}
	refs, err := aiclubReferenceImages(c, request)
	if err != nil {
		return nil, err
	}
	if err := validateAiclubReferenceURLs(refs); err != nil {
		return nil, err
	}
	if len(refs) > 0 {
		body["images"] = refs
	}
	return body, nil
}

func validateAiclubReferenceURLs(refs []string) error {
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		lowerRef := strings.ToLower(ref)
		if !strings.HasPrefix(lowerRef, "http://") && !strings.HasPrefix(lowerRef, "https://") {
			return fmt.Errorf("Aiclub reference images must be HTTP or HTTPS URLs; base64 data URIs are not supported")
		}
	}
	return nil
}

func OpenaiAiclubImageHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, types.NewOpenAIError(fmt.Errorf("upstream HTTP %d: %s", resp.StatusCode, string(responseBody)), types.ErrorCodeBadResponse, resp.StatusCode)
	}

	normalized, adaptErr := adaptAiclubImageResponse(c.Request.Context(), info, responseBody)
	if adaptErr != nil {
		return nil, adaptErr
	}

	synthetic := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(normalized)),
	}
	synthetic.Header.Set("Content-Type", "application/json")
	return OpenaiImageHandler(c, info, synthetic)
}

func adaptAiclubImageResponse(ctx context.Context, info *relaycommon.RelayInfo, responseBody []byte) ([]byte, *types.NewAPIError) {
	if info == nil || len(responseBody) == 0 || !gjson.ValidBytes(responseBody) {
		return responseBody, nil
	}

	status := strings.ToLower(strings.TrimSpace(gjson.GetBytes(responseBody, "status").String()))
	if isAiclubUpstreamTaskPending(status) {
		polled, pollErr := pollAiclubImageTask(ctx, info, responseBody)
		if pollErr != nil {
			return nil, pollErr
		}
		responseBody = polled
		status = strings.ToLower(strings.TrimSpace(gjson.GetBytes(responseBody, "status").String()))
	}

	if isAiclubUpstreamTaskFailed(status) {
		message := strings.TrimSpace(gjson.GetBytes(responseBody, "error.message").String())
		if message == "" {
			message = "upstream image task failed"
		}
		return nil, types.NewOpenAIError(fmt.Errorf("%s", message), types.ErrorCodeBadResponse, http.StatusBadGateway)
	}

	resultURL := strings.TrimSpace(gjson.GetBytes(responseBody, "metadata.result_url").String())
	if resultURL == "" {
		return nil, types.NewOpenAIError(fmt.Errorf("upstream completed without result_url"), types.ErrorCodeBadResponse, http.StatusBadGateway)
	}

	imageResp := dto.ImageResponse{
		Created: time.Now().Unix(),
		Data: []dto.ImageData{
			{Url: resultURL},
		},
	}
	out, err := json.Marshal(imageResp)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	return out, nil
}

func resolveAiclubUpstreamModel(info *relaycommon.RelayInfo, fallback string) string {
	if info != nil && info.ChannelMeta != nil && strings.TrimSpace(info.UpstreamModelName) != "" {
		return strings.TrimSpace(info.UpstreamModelName)
	}
	return strings.TrimSpace(fallback)
}

func isAiclubGPTImageModelName(modelName string) bool {
	name := strings.ToLower(strings.TrimSpace(modelName))
	return strings.HasPrefix(name, "gpt-image")
}

func aiclubAspectRatio(request dto.ImageRequest) string {
	return adobe2APIAspectRatio(request)
}

func aiclubReferenceImages(c *gin.Context, request dto.ImageRequest) ([]string, error) {
	refs, err := adobe2APIReferenceImages(c, request)
	if err != nil {
		return nil, err
	}
	if len(refs) > aiclubMaxInputImages {
		return nil, fmt.Errorf("too many images, max %d", aiclubMaxInputImages)
	}
	return refs, nil
}

func aiclubReferenceImageValuesForValidation(c *gin.Context, request dto.ImageRequest) []string {
	refs, _ := aiclubReferenceImages(c, request)
	return refs
}

func isAiclubUpstreamTaskPending(status string) bool {
	switch status {
	case "queued", "in_progress", "submitting", "polling", "processing", "pending", "running":
		return true
	default:
		return false
	}
}

func isAiclubUpstreamTaskFailed(status string) bool {
	switch status {
	case "failed", "failure", "error", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func isAiclubUpstreamTaskCompleted(status string) bool {
	switch status {
	case "completed", "succeeded", "success", "done":
		return true
	default:
		return false
	}
}

func aiclubImagePollInterval() time.Duration {
	if v := strings.TrimSpace(os.Getenv("AICLUB_IMAGE_POLL_INTERVAL")); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	return defaultAiclubImagePollInterval
}

func aiclubImagePollTimeout() time.Duration {
	if v := strings.TrimSpace(os.Getenv("AICLUB_IMAGE_POLL_TIMEOUT")); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	return defaultAiclubImagePollTimeout
}

func pollAiclubImageTask(ctx context.Context, info *relaycommon.RelayInfo, createBody []byte) ([]byte, *types.NewAPIError) {
	taskID := strings.TrimSpace(gjson.GetBytes(createBody, "id").String())
	if taskID == "" {
		taskID = strings.TrimSpace(gjson.GetBytes(createBody, "task_id").String())
	}
	if taskID == "" {
		return nil, types.NewOpenAIError(fmt.Errorf("upstream returned async task without id"), types.ErrorCodeBadResponse, http.StatusBadGateway)
	}

	base := strings.TrimRight(strings.TrimSpace(info.ChannelBaseUrl), "/")
	if base == "" {
		return nil, types.NewOpenAIError(fmt.Errorf("channel base URL is required for polling"), types.ErrorCodeBadResponse, http.StatusBadGateway)
	}
	pollURL := base + "/v1/videos/" + taskID

	deadline := time.Now().Add(aiclubImagePollTimeout())
	interval := aiclubImagePollInterval()
	for {
		if ctx.Err() != nil {
			return nil, types.NewOpenAIError(ctx.Err(), types.ErrorCodeBadResponse, http.StatusGatewayTimeout)
		}
		if time.Now().After(deadline) {
			return nil, types.NewOpenAIError(fmt.Errorf("aiclub image task timed out after %s", aiclubImagePollTimeout()), types.ErrorCodeBadResponse, http.StatusGatewayTimeout)
		}

		body, err := fetchAiclubPollURL(ctx, info, pollURL)
		if err != nil {
			return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusBadGateway)
		}
		if !gjson.ValidBytes(body) {
			return nil, types.NewOpenAIError(fmt.Errorf("invalid aiclub poll response"), types.ErrorCodeBadResponse, http.StatusBadGateway)
		}

		status := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "status").String()))
		if isAiclubUpstreamTaskFailed(status) {
			message := strings.TrimSpace(gjson.GetBytes(body, "error.message").String())
			if message == "" {
				message = "upstream image task failed"
			}
			return nil, types.NewOpenAIError(fmt.Errorf("%s", message), types.ErrorCodeBadResponse, http.StatusBadGateway)
		}
		if isAiclubUpstreamTaskCompleted(status) {
			return body, nil
		}
		if !isAiclubUpstreamTaskPending(status) && strings.TrimSpace(gjson.GetBytes(body, "metadata.result_url").String()) != "" {
			return body, nil
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, types.NewOpenAIError(ctx.Err(), types.ErrorCodeBadResponse, http.StatusGatewayTimeout)
		case <-timer.C:
		}
	}
}

func fetchAiclubPollURL(ctx context.Context, info *relaycommon.RelayInfo, pollURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pollURL, nil)
	if err != nil {
		return nil, err
	}
	if info != nil && strings.TrimSpace(info.ApiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(info.ApiKey))
	}
	req.Header.Set("Accept", "application/json")

	client := service.GetHttpClient()
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("aiclub poll HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
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
	if err := validateAiclubAspectRatio(info, modelName, aiclubAspectRatio(request, modelName, info)); err != nil {
		return err
	}
	files, err := collectAdobe2APIMultipartImageFiles(c)
	if err != nil {
		return err
	}
	refs := aiclubReferenceImageValuesForValidation(c, request)
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
	if info != nil &&
		info.RelayMode == relayconstant.RelayModeImagesEdits &&
		hasAdobe2APIMultipartImageFiles(c, request) {
		return buildAiclubImageMultipart(c, info, request)
	}

	modelName := resolveAiclubUpstreamModel(info, request.Model)
	if modelName == "" {
		return nil, fmt.Errorf("model is required")
	}
	body := map[string]any{
		"model":  modelName,
		"prompt": request.Prompt,
	}
	if err := applyAiclubImageRatioFields(body, request, modelName, info); err != nil {
		return nil, err
	}
	refs, err := aiclubReferenceImages(c, request)
	if err != nil {
		return nil, err
	}
	if len(refs) > 0 {
		body["images"] = refs
	}
	return body, nil
}

func buildAiclubImageMultipart(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (*bytes.Buffer, error) {
	if info != nil {
		info.AiclubImageMultipart = true
	}
	imageFiles, err := collectAdobe2APIMultipartImageFiles(c)
	if err != nil {
		return nil, err
	}
	if len(imageFiles) == 0 {
		return nil, fmt.Errorf("image is required")
	}
	if len(imageFiles) > aiclubMaxInputImages {
		return nil, fmt.Errorf("too many images, max %d", aiclubMaxInputImages)
	}

	modelName := resolveAiclubUpstreamModel(info, request.Model)
	if modelName == "" {
		return nil, fmt.Errorf("model is required")
	}

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	_ = writer.WriteField("model", modelName)
	if prompt := strings.TrimSpace(request.Prompt); prompt != "" {
		_ = writer.WriteField("prompt", prompt)
	}
	if err := writeAiclubImageRatioFields(writer, request, modelName, info); err != nil {
		_ = writer.Close()
		return nil, err
	}
	for i, fileHeader := range imageFiles {
		if err := writeAdobe2APIMultipartFile(writer, "image", fileHeader); err != nil {
			return nil, fmt.Errorf("write image file %d: %w", i, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	if c != nil && c.Request != nil {
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	}
	return &requestBody, nil
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
	return strings.Contains(name, "gpt-image")
}

// Aiclub GPT Image 2 ratios from upstream docs section 11.2.
var aiclubGPTImageRatios = []string{
	"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3", "5:4", "4:5", "21:9",
}

// aiclubExactSizeRatios maps documented WIDTHxHEIGHT values to aspect_ratio.
// Includes the size-inference table plus GPT Image 2 / Nano-Banana output grids,
// because some official pixels (e.g. 5504x3072) do not gcd to 16:9.
var aiclubExactSizeRatios = map[string]string{
	"1024x1024": "1:1", "1536x1536": "1:1", "2048x2048": "1:1", "2880x2880": "1:1", "4096x4096": "1:1",
	"1024x1792": "9:16", "1536x2752": "9:16", "720x1280": "9:16", "1440x2560": "9:16", "2160x3840": "9:16",
	"768x1360": "9:16", "3072x5504": "9:16",
	"1792x1024": "16:9", "2752x1536": "16:9", "1280x720": "16:9", "2560x1440": "16:9", "3840x2160": "16:9",
	"1360x768": "16:9", "5504x3072": "16:9",
	"2048x1536": "4:3", "1152x864": "4:3", "2304x1728": "4:3", "3264x2448": "4:3", "4096x3072": "4:3",
	"1536x2048": "3:4", "864x1152": "3:4", "1728x2304": "3:4", "2448x3264": "3:4", "3072x4096": "3:4",
	"1248x832": "3:2", "2496x1664": "3:2", "3504x2336": "3:2",
	"832x1248": "2:3", "1664x2496": "2:3", "2336x3504": "2:3",
	"1120x896": "5:4", "2240x1792": "5:4", "3200x2560": "5:4",
	"896x1120": "4:5", "1792x2240": "4:5", "2560x3200": "4:5",
	"1456x624": "21:9", "3024x1296": "21:9", "3696x1584": "21:9",
	"512x2048": "1:4", "1024x4096": "1:4", "2048x8192": "1:4",
	"2048x512": "4:1", "4096x1024": "4:1", "8192x2048": "4:1",
	"384x3072": "1:8", "768x6144": "1:8", "1536x12288": "1:8",
	"3072x384": "8:1", "6144x768": "8:1", "12288x1536": "8:1",
}

// aiclubAspectRatio maps public Image API size/aspect_ratio to upstream aspect_ratio.
// Pixel sizes are inferred to a ratio; Aiclub does not accept WIDTHxHEIGHT as output size.
func aiclubAspectRatio(request dto.ImageRequest, modelName string, info *relaycommon.RelayInfo) string {
	if value := adobe2APIImageOptionString(request, "aspect_ratio", "aspectRatio", "ratio"); value != "" {
		if ratio := normalizePureAspectRatio(value); ratio != "" {
			return ratio
		}
	}
	size := strings.TrimSpace(request.Size)
	if ratio := normalizePureAspectRatio(size); ratio != "" {
		return ratio
	}
	ratio := aiclubRatioFromSize(size)
	if ratio == "" {
		return ""
	}
	return snapAiclubRatioForModel(ratio, modelName, info)
}

func aiclubRatioFromSize(size string) string {
	key := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(size), " ", ""))
	if ratio, ok := aiclubExactSizeRatios[key]; ok {
		return ratio
	}
	return aspectRatioFromImageDimensions(size)
}

func snapAiclubRatioForModel(ratio, modelName string, info *relaycommon.RelayInfo) string {
	if ratio == "" || !strings.Contains(ratio, ":") {
		return ratio
	}
	var allowed []string
	if isAiclubGPTImageModelName(modelName) || (info != nil && isAiclubGPTImageModelName(info.OriginModelName)) {
		allowed = aiclubGPTImageRatios
	} else if info != nil {
		allowed = aiclubBananaAllowedRatios(info.OriginModelName)
	}
	if len(allowed) == 0 {
		return ratio
	}
	for _, candidate := range allowed {
		if ratio == candidate {
			return ratio
		}
	}
	return snapToNearestAspectRatio(ratio, allowed)
}

func aiclubBananaAllowedRatios(originModel string) []string {
	name := strings.ToLower(originModel)
	allowed := []string{"1:1", "16:9", "9:16", "4:3", "3:4"}
	if strings.Contains(name, "nano-banana2") {
		return append(allowed, "1:4", "4:1", "1:8", "8:1")
	}
	if strings.Contains(name, "nano-banana-pro") {
		return append(allowed, "3:2", "2:3", "5:4", "4:5", "21:9")
	}
	return allowed
}

func snapToNearestAspectRatio(ratio string, allowed []string) string {
	width, height, ok := parseColonRatio(ratio)
	if !ok || len(allowed) == 0 {
		return ratio
	}
	target := float64(width) / float64(height)
	best := allowed[0]
	bestDiff := math.MaxFloat64
	for _, candidate := range allowed {
		cw, ch, cok := parseColonRatio(candidate)
		if !cok {
			continue
		}
		diff := math.Abs(target - float64(cw)/float64(ch))
		if diff < bestDiff {
			bestDiff = diff
			best = candidate
		}
	}
	return best
}

func parseColonRatio(ratio string) (int, int, bool) {
	parts := strings.Split(strings.TrimSpace(ratio), ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	width, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil || width <= 0 || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

func validateAiclubAspectRatio(info *relaycommon.RelayInfo, modelName, aspectRatio string) error {
	if aspectRatio == "" || !strings.Contains(aspectRatio, ":") {
		return nil
	}
	if isAiclubGPTImageModelName(modelName) {
		return imagevendor.ValidateGPTImageAspectRatio(aspectRatio)
	}
	if info != nil {
		return imagevendor.ValidateAdobeBananaAspectRatio(info.OriginModelName, aspectRatio)
	}
	return nil
}

func applyAiclubImageRatioFields(body map[string]any, request dto.ImageRequest, modelName string, info *relaycommon.RelayInfo) error {
	if aspectRatio := aiclubAspectRatio(request, modelName, info); aspectRatio != "" {
		if err := validateAiclubAspectRatio(info, modelName, aspectRatio); err != nil {
			return err
		}
		body["aspect_ratio"] = aspectRatio
	}
	return nil
}

func writeAiclubImageRatioFields(writer *multipart.Writer, request dto.ImageRequest, modelName string, info *relaycommon.RelayInfo) error {
	if aspectRatio := aiclubAspectRatio(request, modelName, info); aspectRatio != "" {
		if err := validateAiclubAspectRatio(info, modelName, aspectRatio); err != nil {
			return err
		}
		return writer.WriteField("aspect_ratio", aspectRatio)
	}
	return nil
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

func aiclubImageDoRequest(a *Adaptor, c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	if info != nil && info.AiclubImageMultipart {
		return channel.DoFormRequest(a, c, info, requestBody)
	}
	return channel.DoApiRequest(a, c, info, requestBody)
}

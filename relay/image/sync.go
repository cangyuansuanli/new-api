package image

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relay/imagevendor"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func Helper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *types.NewAPIError) {
	info.InitChannelMeta(c)

	imageReq, ok := info.Request.(*dto.ImageRequest)
	if !ok {
		return types.NewErrorWithStatusCode(fmt.Errorf("invalid request type, expected dto.ImageRequest, got %T", info.Request), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	request, err := common.DeepCopy(imageReq)
	if err != nil {
		return types.NewError(fmt.Errorf("failed to copy request to ImageRequest: %w", err), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	err = helper.ModelMappedHelper(c, info, request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	imagePatch, err := imagevendor.ApplyRequestPatch(info.OriginModelName, request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}
	if ApplyChannelImageSizeDowngrade(info.ChannelId, request) {
		syncImageSizeToForm(c, request.Size)
	}
	applySyncImageUpstreamB64Override(c, info, request)
	syncImageRequestStreamToForm(c, request)

	adaptor := getAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	var requestBody io.Reader

	if model_setting.GetGlobalSettings().PassThroughRequestEnabled || info.ChannelSetting.PassThroughBodyEnabled {
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			return types.NewErrorWithStatusCode(err, types.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		requestBody = common.ReaderOnly(storage)
	} else {
		convertedRequest, err := adaptor.ConvertImageRequest(c, info, *request)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed)
		}
		relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)

		switch convertedRequest.(type) {
		case *bytes.Buffer:
			requestBody = convertedRequest.(io.Reader)
		default:
			jsonData, err := common.Marshal(convertedRequest)
			if err != nil {
				return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			}

			// apply param override
			if len(info.ParamOverride) > 0 {
				jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
				if err != nil {
					return apiErrorFromParamOverride(err)
				}
			}

			logger.LogDebug(c, "image request body: %s", jsonData)
			body, size, closer, err := relaycommon.NewOutboundJSONBody(jsonData)
			if err != nil {
				return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			}
			defer closer.Close()
			jsonData = nil
			info.UpstreamRequestBodySize = size
			requestBody = body
		}
	}

	statusCodeMappingStr := c.GetString("status_code_mapping")

	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}
	var httpResp *http.Response
	if resp != nil {
		httpResp = resp.(*http.Response)
		info.IsStream = info.IsStream || strings.HasPrefix(httpResp.Header.Get("Content-Type"), "text/event-stream")
		if httpResp.StatusCode != http.StatusOK {
			if httpResp.StatusCode == http.StatusCreated && info.ApiType == constant.APITypeReplicate {
				// replicate channel returns 201 Created when using Prefer: wait, treat it as success.
				httpResp.StatusCode = http.StatusOK
			} else {
				newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
				// reset status code 重置状态码
				service.ResetStatusCode(newAPIError, statusCodeMappingStr)
				return newAPIError
			}
		}
	}

	usage, newAPIError := adaptor.DoResponse(c, httpResp, info)
	if newAPIError != nil {
		// reset status code 重置状态码
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return newAPIError
	}

	imageN := uint(1)
	if request.N != nil {
		imageN = *request.N
	}

	// n is handled via OtherRatio so it is applied exactly once in quota
	// calculation (both price-based and ratio-based paths).
	// Adaptors may have already set a more accurate count from the
	// upstream response; only set the default when they haven't.
	if info.PriceData.UsePrice { // only price model use N ratio
		if _, hasN := info.PriceData.OtherRatios["n"]; !hasN {
			info.PriceData.AddOtherRatio("n", float64(imageN))
		}
	}

	if usage.(*dto.Usage).TotalTokens == 0 {
		usage.(*dto.Usage).TotalTokens = 1
	}
	if usage.(*dto.Usage).PromptTokens == 0 {
		usage.(*dto.Usage).PromptTokens = 1
	}

	quality := request.Quality
	if quality == "" {
		quality = "standard"
	}

	var logContent []string

	logSize := request.Size
	if imagePatch.LogSize != "" {
		logSize = imagePatch.LogSize
	}
	if len(logSize) > 0 {
		logContent = append(logContent, fmt.Sprintf("大小 %s", logSize))
	}
	if len(quality) > 0 && !imagePatch.SuppressQualityLog {
		logContent = append(logContent, fmt.Sprintf("品质 %s", quality))
	}
	if imageN > 0 {
		logContent = append(logContent, fmt.Sprintf("生成数量 %d", imageN))
	}

	service.PostTextConsumeQuota(c, info, usage.(*dto.Usage), logContent)
	return nil
}

// applySyncImageUpstreamB64Override：Gulie 类模型客户要 url 时，对内请求上游 b64_json，响应再转 R2 公网 url。
func applySyncImageUpstreamB64Override(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ImageRequest) {
	if request == nil || info == nil {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(request.ResponseFormat), "url") {
		return
	}
	if !imagevendor.ImageSyncPreferUpstreamB64JSON(info.OriginModelName) {
		return
	}
	info.ImageClientWantsURL = true
	request.ResponseFormat = "b64_json"
	syncImageResponseFormatToForm(c, "b64_json")
}

func syncImageResponseFormatToForm(c *gin.Context, format string) {
	if c == nil || c.Request == nil {
		return
	}
	if c.Request.MultipartForm != nil {
		c.Request.MultipartForm.Value["response_format"] = []string{format}
	}
	if c.Request.PostForm != nil {
		c.Request.PostForm.Set("response_format", format)
	}
}

func syncImageRequestStreamToForm(c *gin.Context, request *dto.ImageRequest) {
	if request == nil || request.Stream == nil || c.Request.MultipartForm == nil {
		return
	}
	streamValue := "false"
	if *request.Stream {
		streamValue = "true"
	}
	c.Request.MultipartForm.Value["stream"] = []string{streamValue}
	if c.Request.PostForm == nil {
		c.Request.PostForm = url.Values{}
	}
	c.Request.PostForm.Set("stream", streamValue)
}

func syncImageSizeToForm(c *gin.Context, size string) {
	if c == nil || c.Request == nil || strings.TrimSpace(size) == "" {
		return
	}
	if c.Request.MultipartForm != nil {
		c.Request.MultipartForm.Value["size"] = []string{size}
	}
	if c.Request.PostForm == nil {
		c.Request.PostForm = url.Values{}
	}
	c.Request.PostForm.Set("size", size)
}

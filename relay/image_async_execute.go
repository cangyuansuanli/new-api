package relay

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

type AsyncImageEditFile struct {
	Field       string `json:"field"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Data        []byte `json:"data"`
}

type AsyncImageEditPayload struct {
	Fields map[string]string    `json:"fields"`
	Files  []AsyncImageEditFile `json:"files"`
}

func ExecuteImageTaskUpstream(task *model.Task) ([]dto.ImageData, *dto.Usage, error) {
	channel, err := model.GetChannelById(task.ChannelId, true)
	if err != nil {
		return nil, nil, err
	}
	cache, err := model.GetUserCache(task.UserId)
	if err != nil {
		return nil, nil, err
	}

	req, relayMode, err := buildHTTPRequestForImageTask(task)
	if err != nil {
		return nil, nil, err
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	cache.WriteContext(c)
	c.Set("id", task.UserId)

	group := task.Group
	if group == "" {
		group, _ = model.GetUserGroup(task.UserId, false)
	}
	c.Set("group", group)

	if apiErr := setupImageTaskChannelContext(c, channel, task.Properties.OriginModelName); apiErr != nil {
		return nil, nil, apiErr.Err
	}
	c.Set("relay_mode", relayMode)

	request, err := helper.GetAndValidateRequest(c, types.RelayFormatOpenAIImage)
	if err != nil {
		return nil, nil, err
	}

	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatOpenAIImage, request, nil)
	if err != nil {
		return nil, nil, err
	}
	relayInfo.InitChannelMeta(c)
	if relayInfo.TaskRelayInfo == nil {
		relayInfo.TaskRelayInfo = &relaycommon.TaskRelayInfo{}
	}
	relayInfo.TaskRelayInfo.PublicTaskID = task.TaskID
	relayInfo.IsStream = false
	relayInfo.SkipConsumeQuota = true

	useURLResponse := imageAsyncUsesURLResponse(task.Properties.OriginModelName)
	if imageReq, ok := relayInfo.Request.(*dto.ImageRequest); ok {
		imageReq.Stream = common.GetPointer(false)
		if useURLResponse {
			imageReq.ResponseFormat = "url"
		} else if strings.TrimSpace(imageReq.ResponseFormat) == "" {
			imageReq.ResponseFormat = "b64_json"
		}
	}

	apiErr := ImageHelper(c, relayInfo)
	if apiErr != nil {
		return nil, nil, apiErr.Err
	}

	images, usage, err := parseCapturedImageResponse(w)
	if err != nil {
		return nil, nil, err
	}
	return images, usage, nil
}

func buildHTTPRequestForImageTask(task *model.Task) (*http.Request, int, error) {
	path := strings.TrimSpace(task.PrivateData.RequestPath)
	if path == "" {
		path = "/v1/images/generations"
	}
	relayMode := relayconstant.RelayModeImagesGenerations
	if strings.Contains(path, "/edits") {
		relayMode = relayconstant.RelayModeImagesEdits
	}

	if relayMode == relayconstant.RelayModeImagesEdits {
		payload := AsyncImageEditPayload{}
		if err := common.Unmarshal(task.PrivateData.RequestSnapshot, &payload); err != nil {
			return nil, 0, fmt.Errorf("unmarshal edit payload: %w", err)
		}
		useURLResponse := imageAsyncUsesURLResponse(task.Properties.OriginModelName)
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		for key, value := range payload.Fields {
			if key == "async" {
				continue
			}
			if useURLResponse && key == "response_format" {
				continue
			}
			_ = writer.WriteField(key, value)
		}
		if useURLResponse {
			_ = writer.WriteField("response_format", "url")
		}
		for _, file := range payload.Files {
			part, err := writer.CreateFormFile(file.Field, file.Filename)
			if err != nil {
				writer.Close()
				return nil, 0, err
			}
			if _, err := part.Write(file.Data); err != nil {
				writer.Close()
				return nil, 0, err
			}
		}
		if err := writer.Close(); err != nil {
			return nil, 0, err
		}
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", writer.FormDataContentType())
		return req, relayMode, nil
	}

	body := task.PrivateData.RequestSnapshot
	if len(body) == 0 {
		return nil, 0, fmt.Errorf("empty request snapshot")
	}
	normalized, err := normalizeAsyncGenerationBody(body, imageAsyncUsesURLResponse(task.Properties.OriginModelName))
	if err != nil {
		return nil, 0, err
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(normalized))
	req.Header.Set("Content-Type", "application/json")
	return req, relayMode, nil
}

// imageAsyncUsesURLResponse：4K / Geek2 FLUX 等走 url 响应，避免超大 b64_json 被上游截断。
func imageAsyncUsesURLResponse(originModel string) bool {
	return service.ImageModelUsesURLRehost(originModel)
}

func normalizeAsyncGenerationBody(body []byte, useURLResponse bool) ([]byte, error) {
	var raw map[string]json.RawMessage
	if err := common.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	delete(raw, "async")
	raw["stream"] = json.RawMessage("false")
	if useURLResponse {
		raw["response_format"] = json.RawMessage("\"url\"")
	} else if _, ok := raw["response_format"]; !ok {
		raw["response_format"] = json.RawMessage("\"b64_json\"")
	}
	return common.Marshal(raw)
}

func parseCapturedImageResponse(w *httptest.ResponseRecorder) ([]dto.ImageData, *dto.Usage, error) {
	body := w.Body.Bytes()
	if len(body) == 0 {
		return nil, nil, fmt.Errorf("empty upstream image response")
	}
	contentType := strings.ToLower(w.Header().Get("Content-Type"))
	if strings.Contains(contentType, "text/event-stream") {
		return parseSSEImageResponse(body)
	}
	var imageResp dto.ImageResponse
	if err := common.Unmarshal(body, &imageResp); err != nil {
		return nil, nil, fmt.Errorf("parse image json: %w", err)
	}
	if len(imageResp.Data) == 0 {
		return nil, nil, fmt.Errorf("image response has no data")
	}
	usage := &dto.Usage{TotalTokens: 1, PromptTokens: 1}
	return imageResp.Data, usage, nil
}

func parseSSEImageResponse(body []byte) ([]dto.ImageData, *dto.Usage, error) {
	text := string(body)
	var images []dto.ImageData
	for _, block := range strings.Split(text, "\n\n") {
		data := extractSSEDataLine(block)
		if data == "" || data == "[DONE]" {
			continue
		}
		var event map[string]json.RawMessage
		if err := common.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		kind := readJSONStringField(event, "type")
		if kind == "" {
			kind = readJSONStringField(event, "object")
		}
		if strings.Contains(kind, "partial") {
			continue
		}
		if strings.Contains(kind, "completed") || strings.Contains(kind, "result") {
			if b64 := readJSONStringField(event, "b64_json"); b64 != "" {
				images = append(images, dto.ImageData{B64Json: b64})
			}
			if urlVal := readJSONStringField(event, "url"); urlVal != "" {
				images = append(images, dto.ImageData{Url: urlVal})
			}
			if rawData, ok := event["data"]; ok {
				var items []dto.ImageData
				if err := common.Unmarshal(rawData, &items); err == nil {
					images = append(images, items...)
				}
			}
		}
	}
	if len(images) == 0 {
		return nil, nil, fmt.Errorf("sse image response has no completed data")
	}
	usage := &dto.Usage{TotalTokens: 1, PromptTokens: 1}
	return images, usage, nil
}

func extractSSEDataLine(block string) string {
	var parts []string
	for _, line := range strings.Split(block, "\n") {
		if strings.HasPrefix(line, "data:") {
			parts = append(parts, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	return strings.Join(parts, "\n")
}

func readJSONStringField(event map[string]json.RawMessage, key string) string {
	raw, ok := event[key]
	if !ok || len(raw) == 0 {
		return ""
	}
	var s string
	if err := common.Unmarshal(raw, &s); err == nil {
		return s
	}
	return strings.Trim(string(raw), "\"")
}

func decodeImageDataItem(item dto.ImageData) ([]byte, string, error) {
	if item.B64Json != "" {
		data, err := base64.StdEncoding.DecodeString(item.B64Json)
		if err != nil {
			return nil, "", err
		}
		return data, detectImageBytesMimeType(data), nil
	}
	if item.Url == "" {
		return nil, "", fmt.Errorf("image item has no url or b64_json")
	}
	if strings.HasPrefix(item.Url, "data:") {
		data, mime, err := decodeDataURI(item.Url)
		return data, mime, err
	}
	return nil, item.Url, nil
}

func DecodeImageDataItemExported(item dto.ImageData) ([]byte, string, error) {
	return decodeImageDataItem(item)
}

func decodeDataURI(uri string) ([]byte, string, error) {
	comma := strings.Index(uri, ",")
	if comma < 0 {
		return nil, "", fmt.Errorf("invalid data uri")
	}
	meta := uri[5:comma]
	payload := uri[comma+1:]
	mimeType := "image/png"
	if semi := strings.Index(meta, ";"); semi > 0 {
		mimeType = meta[:semi]
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, "", err
	}
	if !strings.HasPrefix(mimeType, "image/") {
		mimeType = detectImageBytesMimeType(data)
	}
	return data, mimeType, nil
}

func detectImageBytesMimeType(data []byte) string {
	if len(data) == 0 {
		return "image/png"
	}
	mimeType := http.DetectContentType(data)
	if strings.HasPrefix(mimeType, "image/") {
		return mimeType
	}
	return "image/png"
}

func SnapshotAsyncImageEditRequest(c *gin.Context) ([]byte, error) {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		return nil, err
	}
	payload := AsyncImageEditPayload{
		Fields: make(map[string]string),
	}
	for key, values := range c.Request.MultipartForm.Value {
		if len(values) > 0 {
			payload.Fields[key] = values[0]
		}
	}
	for key, files := range c.Request.MultipartForm.File {
		for _, fh := range files {
			file, err := fh.Open()
			if err != nil {
				return nil, err
			}
			data, err := io.ReadAll(io.LimitReader(file, 20<<20))
			file.Close()
			if err != nil {
				return nil, err
			}
			field := key
			if strings.HasSuffix(key, "[]") {
				field = strings.TrimSuffix(key, "[]")
			}
			payload.Files = append(payload.Files, AsyncImageEditFile{
				Field:       field,
				Filename:    fh.Filename,
				ContentType: fh.Header.Get("Content-Type"),
				Data:        data,
			})
		}
	}
	return common.Marshal(payload)
}

func setupImageTaskChannelContext(c *gin.Context, channel *model.Channel, modelName string) *types.NewAPIError {
	if channel == nil {
		return types.NewError(fmt.Errorf("channel is nil"), types.ErrorCodeGetChannelFailed, types.ErrOptionWithSkipRetry())
	}
	c.Set("original_model", modelName)
	common.SetContextKey(c, constant.ContextKeyChannelId, channel.Id)
	common.SetContextKey(c, constant.ContextKeyChannelName, channel.Name)
	common.SetContextKey(c, constant.ContextKeyChannelType, channel.Type)
	common.SetContextKey(c, constant.ContextKeyChannelCreateTime, channel.CreatedTime)
	common.SetContextKey(c, constant.ContextKeyChannelSetting, channel.GetSetting())
	common.SetContextKey(c, constant.ContextKeyChannelOtherSetting, channel.GetOtherSettings())
	common.SetContextKey(c, constant.ContextKeyChannelParamOverride, channel.GetParamOverride())
	common.SetContextKey(c, constant.ContextKeyChannelHeaderOverride, channel.GetHeaderOverride())
	if channel.OpenAIOrganization != nil && *channel.OpenAIOrganization != "" {
		common.SetContextKey(c, constant.ContextKeyChannelOrganization, *channel.OpenAIOrganization)
	}
	common.SetContextKey(c, constant.ContextKeyChannelAutoBan, channel.GetAutoBan())
	common.SetContextKey(c, constant.ContextKeyChannelModelMapping, channel.GetModelMapping())
	common.SetContextKey(c, constant.ContextKeyChannelStatusCodeMapping, channel.GetStatusCodeMapping())
	key, index, newAPIError := channel.GetNextEnabledKey()
	if newAPIError != nil {
		return newAPIError
	}
	if channel.ChannelInfo.IsMultiKey {
		common.SetContextKey(c, constant.ContextKeyChannelIsMultiKey, true)
		common.SetContextKey(c, constant.ContextKeyChannelMultiKeyIndex, index)
	} else {
		common.SetContextKey(c, constant.ContextKeyChannelIsMultiKey, false)
	}
	common.SetContextKey(c, constant.ContextKeyChannelKey, key)
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, channel.GetBaseURL())
	common.SetContextKey(c, constant.ContextKeySystemPromptOverride, false)
	common.SetContextKey(c, constant.ContextKeyOriginalModel, modelName)
	return nil
}

func IsAsyncImageRequest(c *gin.Context) bool {
	if c.Request.Method != http.MethodPost {
		return false
	}
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		if c.Request.MultipartForm == nil {
			if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
				return false
			}
		}
		return strings.EqualFold(c.PostForm("async"), "true")
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
		Async *bool `json:"async"`
	}
	if err := common.Unmarshal(body, &probe); err != nil {
		return false
	}
	return probe.Async != nil && *probe.Async
}

func imageJobObjectForPath(path string) string {
	if strings.Contains(path, "/edits") {
		return "image.edit"
	}
	return "image.generation"
}

func ImageJobObjectForPathExported(path string) string {
	return imageJobObjectForPath(path)
}

// buildImageProxyURL 构建异步任务图片代理地址。
func buildImageProxyURL(taskID string) string {
	base := strings.TrimRight(system_setting.ServerAddress, "/")
	if base == "" {
		return fmt.Sprintf("/v1/images/%s/content", taskID)
	}
	return fmt.Sprintf("%s/v1/images/%s/content", base, taskID)
}

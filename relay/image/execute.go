package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	openai "github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relay/imagevendor"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func executeTaskUpstream(ctx context.Context, task *model.Task) ([]dto.ImageData, *dto.Usage, error) {
	channel, err := model.GetChannelById(task.ChannelId, true)
	if err != nil {
		return nil, nil, err
	}
	cache, err := model.GetUserCache(task.UserId)
	if err != nil {
		return nil, nil, err
	}

	req, relayMode, err := buildHTTPRequestForImageTask(ctx, task)
	if err != nil {
		return nil, nil, err
	}
	req = req.WithContext(ctx)
	defer req.Body.Close()

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

	if apiErr := setupImageTaskChannelContext(c, channel, task.Properties.OriginModelName, task.PrivateData.Key); apiErr != nil {
		return nil, nil, apiErr.Err
	}
	c.Set("relay_mode", relayMode)

	if relayMode == relayconstant.RelayModeChatCompletions {
		return executeLegacyAsyncChatImageTask(c, task, w)
	}

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

	apiErr := Helper(c, relayInfo)
	if apiErr != nil {
		return nil, nil, apiErr.Err
	}

	images, usage, err := parseCapturedImageResponse(w)
	if err != nil {
		return nil, nil, err
	}
	return images, usage, nil
}

func executeLegacyAsyncChatImageTask(c *gin.Context, task *model.Task, w *httptest.ResponseRecorder) ([]dto.ImageData, *dto.Usage, error) {
	request, err := helper.GetAndValidateTextRequest(c, relayconstant.RelayModeChatCompletions)
	if err != nil {
		return nil, nil, err
	}
	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatOpenAI, request, nil)
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
	if textReq, ok := relayInfo.Request.(*dto.GeneralOpenAIRequest); ok {
		textReq.Stream = common.GetPointer(false)
	}
	if textRelay == nil {
		return nil, nil, fmt.Errorf("image: text relay not configured")
	}
	apiErr := textRelay(c, relayInfo)
	if apiErr != nil {
		return nil, nil, apiErr.Err
	}
	return openai.ParseLegacyChatImageResponse(w.Body.Bytes())
}

func buildHTTPRequestForImageTask(ctx context.Context, task *model.Task) (*http.Request, int, error) {
	snapshot, err := DecodeRequestSnapshot(task.PrivateData.RequestSnapshot, task.PrivateData.RequestPath)
	if err != nil {
		return nil, 0, err
	}
	path := snapshot.Path
	relayMode := relayconstant.RelayModeImagesGenerations
	if snapshot.Kind == RequestSnapshotEditMultipart {
		relayMode = relayconstant.RelayModeImagesEdits
	} else if snapshot.Kind == RequestSnapshotLegacyChatJSON {
		relayMode = relayconstant.RelayModeChatCompletions
	}

	if relayMode == relayconstant.RelayModeImagesEdits {
		payload := *snapshot.Multipart
		useURLResponse := imageAsyncUsesURLResponse(task.Properties.OriginModelName)
		body, err := os.CreateTemp("", "new-api-image-edit-replay-*")
		if err != nil {
			return nil, 0, err
		}
		bodyName := body.Name()
		cleanup := func() {
			body.Close()
			os.Remove(bodyName)
		}
		writer := multipart.NewWriter(body)
		for key, value := range payload.Fields {
			if key == "async" || key == "stream" || key == "response_format" {
				continue
			}
			_ = writer.WriteField(key, value)
		}
		_ = writer.WriteField("stream", "false")
		if useURLResponse {
			_ = writer.WriteField("response_format", "url")
		} else {
			_ = writer.WriteField("response_format", "b64_json")
		}
		for _, file := range payload.Files {
			if strings.TrimSpace(file.URL) != "" {
				if err := writer.WriteField(file.Field, file.URL); err != nil {
					writer.Close()
					cleanup()
					return nil, 0, err
				}
				continue
			}
			part, err := createQueuedEditFormFile(writer, file)
			if err != nil {
				writer.Close()
				cleanup()
				return nil, 0, err
			}
			if err := writeQueuedEditFile(ctx, part, file); err != nil {
				writer.Close()
				cleanup()
				return nil, 0, err
			}
		}
		if err := writer.Close(); err != nil {
			cleanup()
			return nil, 0, err
		}
		if _, err := body.Seek(0, io.SeekStart); err != nil {
			cleanup()
			return nil, 0, err
		}
		os.Remove(bodyName)
		req := httptest.NewRequest(http.MethodPost, path, body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		return req, relayMode, nil
	}

	body := snapshot.Body
	if len(body) == 0 {
		return nil, 0, fmt.Errorf("empty request snapshot")
	}
	if relayMode == relayconstant.RelayModeChatCompletions {
		normalized, err := openai.NormalizeAsyncLegacyChatImageBody(body)
		if err != nil {
			return nil, 0, err
		}
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(normalized))
		req.Header.Set("Content-Type", "application/json")
		return req, relayMode, nil
	}
	normalized, err := normalizeAsyncGenerationBody(body, imageAsyncUsesURLResponse(task.Properties.OriginModelName))
	if err != nil {
		return nil, 0, err
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(normalized))
	req.Header.Set("Content-Type", "application/json")
	return req, relayMode, nil
}

func createQueuedEditFormFile(writer *multipart.Writer, file EditFile) (io.Writer, error) {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{
		"name":     file.Field,
		"filename": file.Filename,
	}))
	contentType := strings.TrimSpace(file.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	header.Set("Content-Type", contentType)
	return writer.CreatePart(header)
}

func writeQueuedEditFile(ctx context.Context, dst io.Writer, file EditFile) error {
	if len(file.Data) > 0 {
		_, err := dst.Write(file.Data)
		return err
	}
	if strings.TrimSpace(file.ObjectKey) == "" {
		return fmt.Errorf("queued edit file has no R2 object key")
	}
	source, err := service.OpenImageTaskInput(ctx, file.ObjectKey)
	if err != nil {
		return err
	}
	defer source.Close()
	written, err := io.Copy(dst, io.LimitReader(source, (20<<20)+1))
	if err != nil {
		return err
	}
	if written > 20<<20 {
		return fmt.Errorf("queued edit file exceeds 20 MiB")
	}
	return nil
}

// imageAsyncUsesURLResponse：4K / Geek2 FLUX 等走 url 响应，避免超大 b64_json 被上游截断。
func imageAsyncUsesURLResponse(originModel string) bool {
	return imagevendor.ImageModelUsesURLRehost(originModel)
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
	} else {
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

func SnapshotEditRequest(c *gin.Context, taskID string) ([]byte, error) {
	form, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		return nil, err
	}
	c.Request.MultipartForm = form
	c.Request.PostForm = form.Value
	payload := EditPayload{
		Fields: make(map[string]string),
	}
	uploadedObjectKeys := make([]string, 0)
	keepUploads := false
	defer func() {
		if keepUploads {
			return
		}
		for _, objectKey := range uploadedObjectKeys {
			_ = service.DeleteImageTaskInput(context.Background(), objectKey)
		}
	}()
	fileIndex := 0
	for key, values := range form.Value {
		for _, value := range values {
			trimmedValue := strings.TrimSpace(value)
			if isQueuedEditURLField(key, trimmedValue) {
				field := strings.TrimSuffix(key, "[]")
				payload.Files = append(payload.Files, EditFile{Field: field, URL: trimmedValue})
				continue
			}
			if _, exists := payload.Fields[key]; !exists {
				payload.Fields[key] = value
			}
		}
	}
	for key, files := range form.File {
		for _, fh := range files {
			file, err := fh.Open()
			if err != nil {
				return nil, err
			}
			uploaded, err := service.UploadImageTaskInput(
				c.Request.Context(), c.GetInt("id"), taskID, fileIndex,
				file, fh.Size, fh.Header.Get("Content-Type"),
			)
			file.Close()
			if err != nil {
				return nil, err
			}
			uploadedObjectKeys = append(uploadedObjectKeys, uploaded.ObjectKey)
			field := key
			if strings.HasSuffix(key, "[]") {
				field = strings.TrimSuffix(key, "[]")
			}
			payload.Files = append(payload.Files, EditFile{
				Field:       field,
				Filename:    fh.Filename,
				ContentType: fh.Header.Get("Content-Type"),
				ObjectKey:   uploaded.ObjectKey,
			})
			fileIndex++
		}
	}
	snapshot, err := NewEditRequestSnapshot(payload)
	if err == nil {
		keepUploads = true
	}
	return snapshot, err
}

func isQueuedEditURLField(field, value string) bool {
	switch strings.TrimSuffix(strings.TrimSpace(field), "[]") {
	case "image", "mask":
		return strings.HasPrefix(strings.ToLower(value), "https://")
	default:
		return false
	}
}

func setupImageTaskChannelContext(c *gin.Context, channel *model.Channel, modelName, keyOverride string) *types.NewAPIError {
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
	key := strings.TrimSpace(keyOverride)
	index := 0
	var newAPIError *types.NewAPIError
	if key == "" {
		key, index, newAPIError = channel.GetNextEnabledKey()
		if newAPIError != nil {
			return newAPIError
		}
	}
	if channel.ChannelInfo.IsMultiKey && keyOverride == "" {
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

func IsAsyncRequest(c *gin.Context) bool {
	if c.Request.Method != http.MethodPost {
		return false
	}
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		if c.Request.MultipartForm == nil {
			form, err := common.ParseMultipartFormReusable(c)
			if err == nil {
				c.Request.MultipartForm = form
				c.Request.PostForm = form.Value
			}
		}
		if c.Request.MultipartForm != nil {
			return strings.EqualFold(c.PostForm("async"), "true")
		}
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

// IsAsyncChatImageRequest 兼容期：POST /chat/completions + async 的 chat 出图（Banana / Flash Image 等）。
func IsAsyncChatImageRequest(c *gin.Context) bool {
	return openai.IsAsyncChatImageRequest(c)
}

func JobObjectForPath(path string) string {
	if strings.Contains(path, "/edits") {
		return "image.edit"
	}
	return "image.generation"
}

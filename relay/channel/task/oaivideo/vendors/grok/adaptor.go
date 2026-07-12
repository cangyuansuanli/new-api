package grok

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/defaultvideo"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// TaskAdaptor keeps NewAPI's public /v1/videos contract while translating the
// 119337 Grok family to its upstream /v1/video/generations protocol.
type TaskAdaptor struct {
	defaultvideo.TaskAdaptor
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{"grok-image-video", "grok-video-1.5"}
}

func (a *TaskAdaptor) GetChannelName() string { return "grok-generations" }

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if c == nil || c.Request == nil || !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return service.TaskErrorWrapperLocal(fmt.Errorf("Grok video requests must use application/json"), "invalid_request", http.StatusBadRequest)
	}
	if taskErr := a.TaskAdaptor.ValidateRequestAndSetAction(c, info); taskErr != nil {
		return taskErr
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if len([]rune(strings.TrimSpace(req.Prompt))) > 4096 {
		return service.TaskErrorWrapperLocal(fmt.Errorf("prompt exceeds 4096 characters"), "invalid_prompt", http.StatusBadRequest)
	}
	if len(req.Images) > 7 {
		return service.TaskErrorWrapperLocal(fmt.Errorf("Grok video supports at most 7 reference images"), "invalid_reference", http.StatusBadRequest)
	}
	if isGrok15(info.OriginModelName, info.UpstreamModelName) {
		if len(req.Images) != 1 {
			return service.TaskErrorWrapperLocal(fmt.Errorf("grok-video-1.5 requires exactly one reference image"), "invalid_reference", http.StatusBadRequest)
		}
		if strings.TrimSpace(req.VideoURL) != "" {
			return service.TaskErrorWrapperLocal(fmt.Errorf("grok-video-1.5 does not support video references"), "invalid_reference", http.StatusBadRequest)
		}
	}
	return nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if info == nil || strings.TrimSpace(info.ChannelBaseUrl) == "" {
		return "", fmt.Errorf("Grok video base url is empty")
	}
	return strings.TrimRight(info.ChannelBaseUrl, "/") + "/v1/video/generations", nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	modelName := strings.TrimSpace(info.UpstreamModelName)
	if modelName == "" {
		modelName = strings.TrimSpace(req.Model)
	}
	out := map[string]any{
		"model":  modelName,
		"prompt": strings.TrimSpace(req.Prompt),
	}
	if seconds := req.RequestedDurationSeconds(); seconds > 0 {
		out["seconds"] = seconds
	}
	if req.AspectRatio != "" {
		out["aspect_ratio"] = req.AspectRatio
	}
	if req.Resolution != "" {
		out["resolution"] = req.Resolution
	}
	if len(req.Images) > 0 {
		out["image_urls"] = append([]string(nil), req.Images...)
	}
	if req.VideoURL != "" {
		out["video_url"] = req.VideoURL
	}
	body, err := common.Marshal(out)
	if err != nil {
		return nil, err
	}
	c.Request.Header.Set("Content-Type", "application/json")
	return bytes.NewReader(body), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	_ = resp.Body.Close()
	normalized, err := normalizeResponse(body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "invalid_response", http.StatusBadGateway)
	}
	clone := *resp
	clone.Body = io.NopCloser(bytes.NewReader(normalized))
	return a.TaskAdaptor.DoResponse(c, &clone, info)
}

func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, _ := body["task_id"].(string)
	if strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/v1/video/generations/"+taskID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	normalized, err := normalizeResponse(respBody)
	if err != nil {
		return nil, err
	}
	return a.TaskAdaptor.ParseTaskResult(normalized)
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	if task == nil {
		return nil, fmt.Errorf("task is nil")
	}
	normalized, err := normalizeResponse(task.Data)
	if err != nil {
		return nil, err
	}
	clone := *task
	clone.Data = normalized
	return a.TaskAdaptor.ConvertToOpenAIVideo(&clone)
}

func normalizeResponse(body []byte) ([]byte, error) {
	var envelope map[string]json.RawMessage
	if err := common.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	data := body
	if rawCode, hasCode := envelope["code"]; hasCode {
		if !isSuccessCode(rawCode) {
			return nil, fmt.Errorf("%s", responseMessage(envelope))
		}
		if rawData, ok := envelope["data"]; ok && len(rawData) > 0 && string(rawData) != "null" {
			data = rawData
		}
	}

	var payload map[string]any
	if err := common.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if _, ok := payload["id"].(string); !ok {
		payload["id"] = payload["task_id"]
	}
	// Some Grok aggregators use data for provider diagnostics instead of the
	// OpenAI Video result array. Keep the raw response in task.Data, but omit
	// this incompatible object from the normalized parsing view.
	if _, ok := payload["data"].([]any); !ok {
		delete(payload, "data")
	}
	if resultURL := strings.TrimSpace(stringValue(payload["result_url"])); resultURL != "" {
		payload["video_url"] = resultURL
	}
	if reason := strings.TrimSpace(stringValue(payload["fail_reason"])); reason != "" && payload["error"] == nil {
		payload["error"] = map[string]any{"message": reason}
	}
	if progress, ok := payload["progress"].(string); ok {
		if value, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(progress, "%")), 64); err == nil {
			payload["progress"] = value
		}
	}
	return common.Marshal(payload)
}

func isSuccessCode(raw json.RawMessage) bool {
	var value string
	if err := common.Unmarshal(raw, &value); err == nil {
		return value == "" || strings.EqualFold(value, "success") || value == "0"
	}
	var number int
	return common.Unmarshal(raw, &number) == nil && number == 0
}

func responseMessage(envelope map[string]json.RawMessage) string {
	for _, key := range []string{"message", "msg"} {
		var value string
		if err := common.Unmarshal(envelope[key], &value); err == nil && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "Grok video upstream request failed"
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

package manju

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	oaivideo "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/shared"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const upstreamModel = "sora2"

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	return relaycommon.ValidateMultipartDirect(c, info)
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	modelName := info.OriginModelName
	if service.IsPerRequestTaskBilling(modelName) {
		return nil
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	seconds, _ := strconv.Atoi(req.Seconds)
	if seconds == 0 {
		seconds = req.Duration
	}
	if seconds <= 0 {
		seconds = 4
	}
	size := req.Size
	if size == "" {
		size = "720x1280"
	}
	ratios := map[string]float64{"seconds": float64(seconds), "size": 1}
	if size == "1792x1024" || size == "1024x1792" {
		ratios["size"] = 1.666667
	}
	return ratios
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s/v1/chat/completions", a.baseURL), nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	bodyMap, err := readRequestBodyMap(c)
	if err != nil {
		return nil, err
	}
	converted, convErr := ConvertChatBody(bodyMap, info.UpstreamModelName)
	if convErr != nil {
		return nil, convErr
	}
	newBody, err := common.Marshal(converted)
	if err != nil {
		return nil, err
	}
	c.Request.Header.Set("Content-Type", "application/json")
	return bytes.NewReader(newBody), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	dResp, err := parseResponseTask(responseBody)
	if err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	upstreamID := dResp.ID
	if upstreamID == "" {
		upstreamID = dResp.TaskID
	}
	if upstreamID == "" {
		if reason := ExtractFailReason(responseBody); reason != "" {
			taskErr = service.TaskErrorWrapperLocal(fmt.Errorf("%s", reason), "upstream_error", http.StatusBadRequest)
			return
		}
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, buildOpenAIVideoCreateResponse(info, dResp, responseBody))
	return upstreamID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}
	uri := fmt.Sprintf("%s/v1/videos/%s", baseUrl, taskID)
	req, err := http.NewRequest(http.MethodGet, uri, nil)
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

func (a *TaskAdaptor) GetModelList() []string {
	return []string{"manju-openai-sora2"}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "manju-sora2"
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	resTask, err := parseResponseTask(respBody)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskResult := relaycommon.TaskInfo{Code: 0}
	switch strings.ToLower(strings.TrimSpace(resTask.Status)) {
	case "queued", "pending":
		taskResult.Status = model.TaskStatusQueued
	case "processing", "in_progress", "running":
		taskResult.Status = model.TaskStatusInProgress
		if resTask.Progress <= 0 {
			taskResult.Progress = "50%"
		}
	case "completed", "succeeded", "success", "done":
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Progress = "100%"
		if videoURL := extractVideoURL(resTask, respBody); videoURL != "" {
			taskResult.Url = videoURL
		}
		if seconds := usageSecondsFromBody(respBody); seconds > 0 {
			taskResult.CompletionTokens = seconds
			taskResult.TotalTokens = seconds
		}
	case "failed", "failure", "cancelled", "canceled", "error":
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		if msg, _ := oaivideo.ParseErrorField(resTask.Error); msg != "" {
			taskResult.Reason = msg
		} else if reason := ExtractFailReason(respBody); reason != "" {
			taskResult.Reason = reason
		} else {
			taskResult.Reason = "task failed"
		}
	default:
		if IsResponse(respBody) {
			status := strings.ToLower(strings.TrimSpace(gjson.GetBytes(respBody, "status").String()))
			if status == "" || isFailedStatus(status) {
				if reason := ExtractFailReason(respBody); reason != "" {
					taskResult.Status = model.TaskStatusFailure
					taskResult.Progress = "100%"
					taskResult.Reason = reason
					break
				}
			}
		}
	}
	if resTask.Progress > 0 && resTask.Progress < 100 {
		taskResult.Progress = fmt.Sprintf("%d%%", resTask.Progress)
	}
	enrichTaskResult(&taskResult, respBody)
	return &taskResult, nil
}

func (a *TaskAdaptor) AdjustBillingOnComplete(task *model.Task, taskResult *relaycommon.TaskInfo) int {
	bc := task.PrivateData.BillingContext
	if bc == nil || bc.ModelPrice <= 0 {
		return 0
	}
	modelName := bc.OriginModelName
	if modelName == "" {
		modelName = task.Properties.OriginModelName
	}
	if service.IsPerRequestTaskBilling(modelName) || bc.PerCallBilling {
		return 0
	}
	if _, ok := bc.OtherRatios["seconds"]; !ok {
		return 0
	}
	actualSeconds := oaivideo.UsageSecondsFromTaskData(task.Data, usageSecondsFromBody)
	if actualSeconds <= 0 && taskResult != nil {
		if taskResult.CompletionTokens > 0 {
			actualSeconds = taskResult.CompletionTokens
		} else if taskResult.TotalTokens > 0 {
			actualSeconds = taskResult.TotalTokens
		}
	}
	if actualSeconds <= 0 && bc.OtherRatios != nil {
		if seconds, ok := bc.OtherRatios["seconds"]; ok && seconds > 0 {
			actualSeconds = int(seconds + 0.5)
		}
	}
	if actualSeconds <= 0 {
		return 0
	}
	groupRatio := bc.GroupRatio
	if groupRatio <= 0 {
		groupRatio = 1
	}
	multiplier := 1.0
	for key, ratio := range bc.OtherRatios {
		if key == "seconds" || ratio == 1.0 || ratio <= 0 {
			continue
		}
		multiplier *= ratio
	}
	return int(bc.ModelPrice * common.QuotaPerUnit * groupRatio * float64(actualSeconds) * multiplier)
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	dResp, err := parseResponseTask(task.Data)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal task data failed")
	}
	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = task.TaskID
	openAIVideo.TaskID = task.TaskID
	openAIVideo.Status = task.Status.ToVideoStatus()
	openAIVideo.SetProgressStr(task.Progress)
	openAIVideo.Model = task.Properties.OriginModelName
	openAIVideo.CreatedAt = task.CreatedAt
	if task.FinishTime > 0 {
		openAIVideo.CompletedAt = task.FinishTime
	} else if dResp.CompletedAt > 0 {
		openAIVideo.CompletedAt = dResp.CompletedAt
	}
	videoURL := extractVideoURL(dResp, task.Data)
	if videoURL == "" {
		videoURL = task.GetResultURL()
	}
	if videoURL != "" {
		openAIVideo.SetMetadata("url", videoURL)
	}
	if task.Status == model.TaskStatusFailure {
		reason := task.FailReason
		if reason == "" || oaivideo.IsGenericTaskFailureReason(reason) {
			if extracted := ExtractFailReason(task.Data); extracted != "" {
				reason = extracted
			}
		}
		if reason == "" {
			reason = oaivideo.ExtractErrorMessage(task.Data)
		}
		if reason == "" {
			reason, _ = oaivideo.ParseErrorField(dResp.Error)
		}
		if reason != "" {
			openAIVideo.Error = &dto.OpenAIVideoError{Message: reason}
		}
	}
	data, err := common.Marshal(openAIVideo)
	if err != nil {
		return nil, err
	}
	if videoURL != "" {
		if data, err = sjson.SetBytes(data, "video_url", videoURL); err != nil {
			return nil, errors.Wrap(err, "set video_url failed")
		}
	}
	if task.Status == model.TaskStatusFailure && openAIVideo.Error != nil && openAIVideo.Error.Message != "" {
		if data, err = sjson.SetBytes(data, "fail_reason", openAIVideo.Error.Message); err != nil {
			return nil, errors.Wrap(err, "set fail_reason failed")
		}
	}
	return data, nil
}

func readRequestBodyMap(c *gin.Context) (map[string]interface{}, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_request_body_failed")
	}
	cachedBody, err := storage.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "read_body_bytes_failed")
	}
	contentType := c.GetHeader("Content-Type")

	if strings.HasPrefix(contentType, "application/json") {
		var bodyMap map[string]interface{}
		if err := common.Unmarshal(cachedBody, &bodyMap); err != nil {
			return nil, err
		}
		return bodyMap, nil
	}

	if strings.Contains(contentType, "multipart/form-data") {
		formData, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return nil, err
		}
		bodyMap := map[string]interface{}{}
		for key, values := range formData.Value {
			if len(values) == 0 {
				continue
			}
			if len(values) == 1 {
				bodyMap[key] = values[0]
			} else {
				bodyMap[key] = values
			}
		}
		if ref, err := firstMultipartFileDataURI(formData, "input_reference", "image"); err != nil {
			return nil, err
		} else if ref != "" {
			if existing := strings.TrimSpace(oaivideo.AsString(bodyMap["input_reference"])); existing == "" {
				bodyMap["input_reference"] = ref
			}
		}
		return bodyMap, nil
	}

	var bodyMap map[string]interface{}
	if err := common.Unmarshal(cachedBody, &bodyMap); err != nil {
		return nil, err
	}
	return bodyMap, nil
}

func firstMultipartFileDataURI(formData *multipart.Form, fieldNames ...string) (string, error) {
	for _, name := range fieldNames {
		for _, fh := range formData.File[name] {
			f, err := fh.Open()
			if err != nil {
				return "", err
			}
			data, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				return "", err
			}
			ct := fh.Header.Get("Content-Type")
			if ct == "" || ct == "application/octet-stream" {
				ct = http.DetectContentType(data)
			}
			return fmt.Sprintf("data:%s;base64,%s", ct, base64.StdEncoding.EncodeToString(data)), nil
		}
	}
	return "", nil
}

// IsRelay 仅按 internal 模型名识别 Manju Sora2，避免上游 model 误匹配污染其他渠道。
func IsRelay(originModel, upstreamModel string) bool {
	origin := strings.ToLower(strings.TrimSpace(originModel))
	if origin != "manju-openai-sora2" && !strings.HasPrefix(origin, "manju-openai-sora") {
		return false
	}
	if strings.TrimSpace(upstreamModel) == "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(upstreamModel), upstreamModelName)
}

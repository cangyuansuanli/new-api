package seedance

import (
	"bytes"
	"fmt"
	"io"
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
)

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
	return fmt.Sprintf("%s/v1/videos", a.baseURL), nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	if IsTengdaRelay(info.OriginModelName, info.UpstreamModelName) {
		bodyMap, err := readJSONBodyMap(c)
		if err != nil {
			return nil, err
		}
		converted, convErr := maybeConvertTengdaBody(bodyMap, info.UpstreamModelName)
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
	return oaivideo.BuildPassthroughRequestBody(c, info.UpstreamModelName)
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

	dResp, err := oaivideo.ParseResponseTask(responseBody)
	if err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}
	upstreamID := dResp.ID
	if upstreamID == "" {
		upstreamID = dResp.TaskID
	}
	if upstreamID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}
	dResp.ID = info.PublicTaskID
	dResp.TaskID = info.PublicTaskID
	c.JSON(http.StatusOK, dResp)
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
	return nil
}

func (a *TaskAdaptor) GetChannelName() string {
	return "seedance"
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	resTask, err := oaivideo.ParseResponseTask(respBody)
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
		if videoURL := oaivideo.ExtractVideoURL(resTask); videoURL != "" {
			taskResult.Url = videoURL
		}
		if seconds := oaivideo.UsageSecondsFromResponseTask(resTask); seconds > 0 {
			taskResult.CompletionTokens = seconds
			taskResult.TotalTokens = seconds
		}
	case "failed", "failure", "cancelled", "canceled", "error":
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		if msg, _ := oaivideo.ParseErrorField(resTask.Error); msg != "" {
			taskResult.Reason = msg
		} else if reason := oaivideo.ExtractErrorMessage(respBody); reason != "" {
			taskResult.Reason = reason
		} else {
			taskResult.Reason = "task failed"
		}
	default:
		if taskResult.Status == "" {
			if reason := oaivideo.ExtractErrorMessage(respBody); reason != "" {
				taskResult.Status = model.TaskStatusFailure
				taskResult.Progress = "100%"
				taskResult.Reason = reason
			}
		}
	}
	if resTask.Progress > 0 && resTask.Progress < 100 {
		taskResult.Progress = fmt.Sprintf("%.0f%%", resTask.Progress)
	}
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
	actualSeconds := oaivideo.UsageSecondsFromTaskData(task.Data, nil)
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
	dResp, err := oaivideo.ParseResponseTask(task.Data)
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
	videoURL := oaivideo.ExtractVideoURL(dResp)
	if videoURL == "" {
		videoURL = task.GetResultURL()
	}
	if videoURL != "" {
		openAIVideo.SetMetadata("url", videoURL)
	}
	if task.Status == model.TaskStatusFailure {
		reason := task.FailReason
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
	return common.Marshal(openAIVideo)
}

func readJSONBodyMap(c *gin.Context) (map[string]interface{}, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_request_body_failed")
	}
	cachedBody, err := storage.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "read_body_bytes_failed")
	}
	var bodyMap map[string]interface{}
	if err := common.Unmarshal(cachedBody, &bodyMap); err != nil {
		return nil, err
	}
	return bodyMap, nil
}

// IsRelay Leonardo / oairegbox cy-sd1 / Tengda cy-sd2 模型。
func IsRelay(originModel, upstreamModel string) bool {
	return IsOairegboxRelay(originModel) || IsLeonardoRelay(originModel) || IsTengdaRelay(originModel, upstreamModel)
}

// IsOairegboxRelay oairegbox 主站 Seedance（cy-sd1-seedance-）。
func IsOairegboxRelay(originModel string) bool {
	origin := strings.ToLower(strings.TrimSpace(originModel))
	return strings.HasPrefix(origin, "cy-sd1-seedance")
}

// IsLeonardoRelay Leonardo 订阅号 Seedance（cy-sd4-）。
func IsLeonardoRelay(originModel string) bool {
	origin := strings.ToLower(strings.TrimSpace(originModel))
	return strings.HasPrefix(origin, "cy-sd4-seedance")
}

// IsTengdaRelay Seedance 特惠档（cy-sd2- / tengd-seedance）。
func IsTengdaRelay(originModel, upstreamModel string) bool {
	origin := strings.ToLower(strings.TrimSpace(originModel))
	if !strings.HasPrefix(origin, "cy-sd2-seedance") && !strings.HasPrefix(origin, "tengd-seedance") {
		return false
	}
	if strings.TrimSpace(upstreamModel) == "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(upstreamModel), tengdaUpstreamModel)
}

package defaultvideo

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	oaivideo "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/shared"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/tidwall/sjson"
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

func validateRemixRequest(c *gin.Context) *dto.TaskError {
	var req relaycommon.TaskSubmitReq
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return service.TaskErrorWrapperLocal(fmt.Errorf("field prompt is required"), "invalid_request", http.StatusBadRequest)
	}
	relaycommon.StorePromptInput(c, req.Prompt)
	c.Set("task_request", req)
	return nil
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	if info.Action == constant.TaskActionRemix {
		return validateRemixRequest(c)
	}
	return relaycommon.ValidateMultipartDirect(c, info)
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	if info.Action == constant.TaskActionRemix {
		return nil
	}
	modelName := info.OriginModelName
	if service.IsPerRequestTaskBilling(modelName) {
		return nil
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	seconds := req.RequestedDurationSeconds()
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
	if info.Action == constant.TaskActionRemix {
		return fmt.Sprintf("%s/v1/videos/%s/remix", a.baseURL, info.OriginTaskID), nil
	}
	return fmt.Sprintf("%s/v1/videos", a.baseURL), nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	return oaivideo.BuildNormalizedRequestBody(c, info.UpstreamModelName, oaivideo.DurationFieldSeconds)
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
	c.JSON(http.StatusOK, service.PatchClientFacingModelObjectFromContext(c, dResp))
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
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
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
	actualSeconds := usageSecondsFromTaskData(task.Data)
	if actualSeconds <= 0 && taskResult != nil {
		if taskResult.CompletionTokens > 0 {
			actualSeconds = taskResult.CompletionTokens
		} else if taskResult.TotalTokens > 0 {
			actualSeconds = taskResult.TotalTokens
		}
	}
	if actualSeconds <= 0 {
		actualSeconds = usageSecondsFromBillingContext(bc)
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

func usageSecondsFromTaskData(data []byte) int {
	return oaivideo.UsageSecondsFromTaskData(data, nil)
}

func usageSecondsFromBillingContext(bc *model.TaskBillingContext) int {
	if bc == nil || bc.OtherRatios == nil {
		return 0
	}
	seconds, ok := bc.OtherRatios["seconds"]
	if !ok || seconds <= 0 {
		return 0
	}
	return int(math.Round(seconds))
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
		openAIVideo.CompletedAt = int64(dResp.CompletedAt)
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

package seedancehuabu

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	oaivideo "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/shared"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/defaultvideo"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var allowedDurations = map[int]struct{}{
	5:  {},
	10: {},
	15: {},
}

type TaskAdaptor struct {
	defaultvideo.TaskAdaptor
}

func (*TaskAdaptor) GetModelList() []string {
	return []string{ModelStandard, ModelFast}
}

func (*TaskAdaptor) GetChannelName() string { return "seedance-huabu" }

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("SD8 Seedance requests must use application/json with public asset URLs"),
			"invalid_request",
			http.StatusBadRequest,
		)
	}
	if taskErr := relaycommon.ValidateMultipartDirect(c, info); taskErr != nil {
		return taskErr
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if seconds := req.RequestedDurationSeconds(); seconds != 0 {
		if _, ok := allowedDurations[seconds]; !ok {
			return service.TaskErrorWrapperLocal(
				fmt.Errorf("duration must be 5, 10, or 15 seconds"),
				"invalid_duration",
				http.StatusBadRequest,
			)
		}
	}
	if isFastModel(info.OriginModelName) {
		if len(req.ReferenceVideos) > 0 {
			return service.TaskErrorWrapperLocal(
				fmt.Errorf("sd8-seedance-2.0-fast does not support reference videos"),
				"invalid_reference_videos",
				http.StatusBadRequest,
			)
		}
		if len(req.ReferenceAudios) > 0 {
			return service.TaskErrorWrapperLocal(
				fmt.Errorf("sd8-seedance-2.0-fast does not support reference audios"),
				"invalid_reference_audios",
				http.StatusBadRequest,
			)
		}
	}
	return nil
}

func (*TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if info == nil || strings.TrimSpace(info.ChannelBaseUrl) == "" {
		return "", fmt.Errorf("SD8 Seedance base url is empty")
	}
	return strings.TrimRight(info.ChannelBaseUrl, "/") + "/v1/videos", nil
}

func (*TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	if info == nil || strings.TrimSpace(info.ApiKey) == "" {
		return fmt.Errorf("SD8 Seedance api key is empty")
	}
	req.Header.Set("Authorization", "Bearer "+info.ApiKey)
	req.Header.Set("Content-Type", "application/json")
	return nil
}

func (*TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	if info == nil {
		return nil, fmt.Errorf("SD8 Seedance relay info is nil")
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	bodyMap := req.CanonicalVideoBody(info.UpstreamModelName)
	out := buildUpstreamBody(bodyMap, info.OriginModelName, info.UpstreamModelName, req.RequestedDurationSeconds(), req.Images, req.ReferenceVideos, req.ReferenceAudios)
	newBody, err := common.Marshal(out)
	if err != nil {
		return nil, err
	}
	c.Request.Header.Set("Content-Type", "application/json")
	return bytes.NewReader(newBody), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, body io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, body)
}

func (*TaskAdaptor) ResolveTaskResultSource(baseURL, taskID, key string) *relaycommon.TaskResultSource {
	if strings.TrimSpace(taskID) == "" {
		return nil
	}
	headers := make(http.Header)
	if strings.TrimSpace(key) != "" {
		headers.Set("Authorization", "Bearer "+key)
	}
	rehostURL := strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/v1/videos/" + taskID + "/content"
	if _, keepDirect := consumeDirectRehostURL(taskID); keepDirect {
		rehostURL = ""
	}
	return &relaycommon.TaskResultSource{
		URL:     rehostURL,
		Headers: headers,
	}
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	result, err := a.TaskAdaptor.ParseTaskResult(respBody)
	if err != nil || result == nil || result.Status != model.TaskStatusSuccess {
		return result, err
	}
	resTask, parseErr := oaivideo.ParseResponseTask(respBody)
	if parseErr != nil {
		return result, err
	}
	upstreamTaskID := strings.TrimSpace(gjson.GetBytes(respBody, "id").String())
	if videoURL := extractSd8VideoURL(respBody, resTask); videoURL != "" {
		result.Url = videoURL
		noteDirectRehostURL(upstreamTaskID, videoURL)
	} else {
		noteDirectRehostURL(upstreamTaskID, "")
	}
	return result, err
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
	videoURL := task.GetResultURL()
	if videoURL == "" {
		videoURL = extractSd8VideoURL(task.Data, dResp)
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
		if data, err = sjson.SetBytes(data, "result_url", videoURL); err != nil {
			return nil, errors.Wrap(err, "set result_url failed")
		}
	}
	if task.Status == model.TaskStatusFailure && openAIVideo.Error != nil && openAIVideo.Error.Message != "" {
		if data, err = sjson.SetBytes(data, "fail_reason", openAIVideo.Error.Message); err != nil {
			return nil, errors.Wrap(err, "set fail_reason failed")
		}
	}
	return data, nil
}

package omnii2v

import (
	"bytes"
	"io"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	oaivideo "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/shared"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/defaultvideo"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/tidwall/sjson"
)

type TaskAdaptor struct {
	defaultvideo.TaskAdaptor
}

func (a *TaskAdaptor) GetChannelName() string {
	return "omni-i2v"
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	contentType := strings.ToLower(c.Request.Header.Get("Content-Type"))
	if strings.HasPrefix(contentType, "application/json") {
		req, err := relaycommon.GetTaskRequest(c)
		if err != nil {
			return nil, err
		}
		bodyMap := req.CanonicalVideoBody(info.UpstreamModelName)
		out := buildUpstreamBody(bodyMap, info.UpstreamModelName, req.Duration)
		newBody, err := common.Marshal(out)
		if err != nil {
			return nil, err
		}
		c.Request.Header.Set("Content-Type", "application/json")
		return bytes.NewReader(newBody), nil
	}
	return oaivideo.BuildNormalizedRequestBody(c, info.UpstreamModelName, oaivideo.DurationFieldSeconds)
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
	if videoURL := extractOmniI2VVideoURL(resTask); videoURL != "" {
		result.Url = videoURL
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
		videoURL = extractOmniI2VVideoURL(dResp)
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

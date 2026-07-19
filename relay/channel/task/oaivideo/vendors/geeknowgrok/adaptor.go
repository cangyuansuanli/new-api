package geeknowgrok

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/defaultvideo"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// TaskAdaptor keeps NewAPI's public /v1/videos contract while translating to
// Geeknow Grok's POST /v1/videos JSON protocol (grok-imagine-video family).
type TaskAdaptor struct {
	defaultvideo.TaskAdaptor
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{upstreamImagineVideo, upstreamImagineVideo15Prev}
}

func (a *TaskAdaptor) GetChannelName() string { return "geeknow-grok" }

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if c == nil || c.Request == nil || !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return service.TaskErrorWrapperLocal(fmt.Errorf("Geeknow Grok video requests must use application/json"), "invalid_request", http.StatusBadRequest)
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
		return service.TaskErrorWrapperLocal(fmt.Errorf("Geeknow Grok video supports at most 7 reference images"), "invalid_reference", http.StatusBadRequest)
	}
	if isImagine15Preview(info.OriginModelName, info.UpstreamModelName) {
		if len(req.Images) > 1 {
			return service.TaskErrorWrapperLocal(fmt.Errorf("grok-imagine-video-1.5-preview supports at most one reference image"), "invalid_reference", http.StatusBadRequest)
		}
		if strings.TrimSpace(req.VideoURL) != "" {
			return service.TaskErrorWrapperLocal(fmt.Errorf("grok-imagine-video-1.5-preview does not support video references"), "invalid_reference", http.StatusBadRequest)
		}
	}
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	body, err := common.Marshal(buildGeeknowGrokBody(req, info.UpstreamModelName, info.OriginModelName))
	if err != nil {
		return nil, err
	}
	c.Request.Header.Set("Content-Type", "application/json")
	return bytes.NewReader(body), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

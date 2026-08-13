package seedancetengda

import (
	"bytes"
	"io"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/defaultvideo"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
)

type TaskAdaptor struct {
	defaultvideo.TaskAdaptor
}

func (a *TaskAdaptor) GetChannelName() string {
	return "seedance-tengda"
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	return relaycommon.ValidateMultipartDirect(c, info)
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	bodyMap := req.CanonicalVideoBody(info.UpstreamModelName)
	mergeFlatDuration(bodyMap, bodyMap, req.Duration)
	converted, convErr := convertBody(bodyMap, info.UpstreamModelName, req.Images)
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

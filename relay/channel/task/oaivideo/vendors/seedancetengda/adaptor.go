package seedancetengda

import (
	"bytes"
	"io"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/defaultvideo"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
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
	bodyMap, err := readJSONBodyMap(c)
	if err != nil {
		return nil, err
	}
	duration := 0
	if req, err := relaycommon.GetTaskRequest(c); err == nil {
		duration = req.RequestedDurationSeconds()
	}
	mergeFlatDuration(bodyMap, bodyMap, duration)
	converted, convErr := convertBody(bodyMap, info.UpstreamModelName)
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

package omnii2v

import (
	"bytes"
	"io"
	"strings"

	"github.com/QuantumNous/new-api/common"
	oaivideo "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/shared"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/defaultvideo"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
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
		bodyMap, err := readJSONBodyMap(c)
		if err != nil {
			return nil, err
		}
		duration := 0
		if req, err := relaycommon.GetTaskRequest(c); err == nil {
			duration = req.RequestedDurationSeconds()
		}
		out := buildUpstreamBody(bodyMap, info.UpstreamModelName, duration)
		newBody, err := common.Marshal(out)
		if err != nil {
			return nil, err
		}
		c.Request.Header.Set("Content-Type", "application/json")
		return bytes.NewReader(newBody), nil
	}
	return oaivideo.BuildNormalizedRequestBody(c, info.UpstreamModelName, oaivideo.DurationFieldSeconds)
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

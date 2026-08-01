package seedanceleonardo

import (
	"bytes"
	"io"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
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
	return "seedance-leonardo"
}

// ValidateRequestAndSetAction only normalizes the OpenAI Video request shape.
// Reference limits, duration, and download-based checks live in leonardo-web2api.
func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	return relaycommon.ValidateMultipartDirect(c, info)
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	contentType := strings.ToLower(c.Request.Header.Get("Content-Type"))
	if strings.HasPrefix(contentType, "application/json") {
		bodyMap, err := readJSONBodyMap(c)
		if err != nil {
			return nil, err
		}
		req, err := relaycommon.GetTaskRequest(c)
		if err != nil {
			return nil, err
		}
		out := buildUpstreamBody(bodyMap, info.UpstreamModelName, req.RequestedDurationSeconds(), req.Images)
		newBody, err := common.Marshal(out)
		if err != nil {
			return nil, err
		}
		c.Request.Header.Set("Content-Type", "application/json")
		return bytes.NewReader(newBody), nil
	}
	return oaivideo.BuildNormalizedRequestBody(c, info.UpstreamModelName, oaivideo.DurationFieldDuration)
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

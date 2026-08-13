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
	if _, ok := seedance25Resolution(info.OriginModelName); !ok {
		return nil
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	seconds := req.RequestedDurationSeconds()
	if seconds <= 0 {
		seconds = 8
	}
	return map[string]float64{"seconds": float64(seconds)}
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	forcedResolution, forceResolution := seedance25Resolution(info.OriginModelName)
	contentType := strings.ToLower(c.Request.Header.Get("Content-Type"))
	if strings.HasPrefix(contentType, "application/json") {
		req, err := relaycommon.GetTaskRequest(c)
		if err != nil {
			return nil, err
		}
		bodyMap := req.CanonicalVideoBody(info.UpstreamModelName)
		out := buildUpstreamBody(bodyMap, info.UpstreamModelName, req.Duration, req.Images)
		if forceResolution {
			out["resolution"] = forcedResolution
		}
		newBody, err := common.Marshal(out)
		if err != nil {
			return nil, err
		}
		c.Request.Header.Set("Content-Type", "application/json")
		return bytes.NewReader(newBody), nil
	}
	if forceResolution {
		req, err := relaycommon.GetTaskRequest(c)
		if err != nil {
			return nil, errors.Wrap(err, "get_normalized_task_request_failed")
		}
		req.Resolution = forcedResolution
		c.Set("task_request", req)
	}
	return oaivideo.BuildNormalizedRequestBody(c, info.UpstreamModelName, oaivideo.DurationFieldDuration)
}

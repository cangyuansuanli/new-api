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
	resolution, ok := seedance25Resolution(info.OriginModelName)
	if !ok {
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
	ratios := map[string]float64{"seconds": float64(seconds)}
	if len(req.ReferenceVideos) > 0 {
		// Leonardo changes the output rate when a reference video is present.
		// Keep this model-specific multiplier in OtherRatios so settlement uses
		// the same rate as the web quote without affecting other video models.
		if resolution == "480p" {
			ratios["reference_video_rate"] = 258.0 / 180.0
		} else {
			ratios["reference_video_rate"] = 466.0 / 292.0
		}
	}
	return ratios
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	forcedResolution, forceResolution := fixedResolution(info.OriginModelName)
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

package seedanceoairegbox

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	oaivideo "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/shared"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/defaultvideo"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

type TaskAdaptor struct {
	defaultvideo.TaskAdaptor
}

func (a *TaskAdaptor) GetChannelName() string {
	return "seedance-oairegbox"
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if taskErr := relaycommon.ValidateMultipartDirect(c, info); taskErr != nil {
		return taskErr
	}
	if service.IsPerRequestTaskBilling(info.OriginModelName) {
		return nil
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	seconds := req.RequestedDurationSeconds()
	if seconds == 0 {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("duration is required for per-second Seedance models"),
			"missing_duration",
			http.StatusBadRequest,
		)
	}
	if seconds < 4 || seconds > 15 {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("duration must be an integer between 4 and 15 seconds"),
			"invalid_duration",
			http.StatusBadRequest,
		)
	}
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	contentType := strings.ToLower(c.Request.Header.Get("Content-Type"))
	if strings.HasPrefix(contentType, "application/json") {
		req, err := relaycommon.GetTaskRequest(c)
		if err != nil {
			return nil, err
		}
		bodyMap := req.CanonicalVideoBody(info.UpstreamModelName)
		out := buildUpstreamBody(bodyMap, info.UpstreamModelName, req.Duration, req.Images)
		newBody, err := common.Marshal(out)
		if err != nil {
			return nil, err
		}
		c.Request.Header.Set("Content-Type", "application/json")
		return bytes.NewReader(newBody), nil
	}
	return oaivideo.BuildNormalizedRequestBody(c, info.UpstreamModelName, oaivideo.DurationFieldDuration)
}

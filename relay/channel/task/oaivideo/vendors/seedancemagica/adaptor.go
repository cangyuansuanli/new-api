package seedancemagica

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

type TaskAdaptor struct{ defaultvideo.TaskAdaptor }

func (*TaskAdaptor) GetModelList() []string {
	return []string{Model720p, Model1080p}
}
func (*TaskAdaptor) GetChannelName() string { return "seedance-magica" }

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return service.TaskErrorWrapperLocal(fmt.Errorf("Magica Seedance requests must use application/json"), "invalid_request", http.StatusBadRequest)
	}
	if taskErr := a.TaskAdaptor.ValidateRequestAndSetAction(c, info); taskErr != nil {
		return taskErr
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if err := validateFrameInputs(req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_reference", http.StatusBadRequest)
	}
	return nil
}

func validateFrameInputs(req relaycommon.TaskSubmitReq) error {
	first := strings.TrimSpace(req.FirstImageUrl)
	last := strings.TrimSpace(req.LastImageUrl)
	if (first == "") != (last == "") {
		return fmt.Errorf("first_image_url and last_image_url must be provided together")
	}
	if first != "" && (len(req.Images) > 0 || len(req.ReferenceVideos) > 0 || len(req.ReferenceAudios) > 0) {
		return fmt.Errorf("first/last frame mode cannot be combined with reference images, videos, or audios")
	}
	return nil
}

func (*TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if info == nil || strings.TrimSpace(info.ChannelBaseUrl) == "" {
		return "", fmt.Errorf("Magica Seedance base url is empty")
	}
	return strings.TrimRight(info.ChannelBaseUrl, "/") + "/v1/videos", nil
}

func (*TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	if info == nil || strings.TrimSpace(info.ApiKey) == "" {
		return fmt.Errorf("Magica Seedance api key is empty")
	}
	req.Header.Set("Authorization", "Bearer "+info.ApiKey)
	req.Header.Set("Content-Type", "application/json")
	return nil
}

func (*TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	if info == nil {
		return nil, fmt.Errorf("Magica Seedance relay info is nil")
	}
	resolution, ok := resolutionForModel(info.OriginModelName)
	if !ok || !IsRelay(info.OriginModelName, info.UpstreamModelName) {
		return nil, fmt.Errorf("unsupported Magica Seedance model mapping")
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	upstream := resolveUpstreamModel(req)
	out := map[string]any{
		"model":      upstream,
		"prompt":     strings.TrimSpace(req.Prompt),
		"resolution": resolution,
	}
	if duration := req.RequestedDurationSeconds(); duration > 0 {
		out["duration"] = duration
	}
	if value := strings.TrimSpace(req.AspectRatio); value != "" {
		out["aspect_ratio"] = value
	}
	if len(req.Images) == 1 {
		out["image_url"] = req.Images[0]
	} else if len(req.Images) > 1 {
		out["reference_image_urls"] = append([]string(nil), req.Images...)
	}
	if len(req.ReferenceVideos) > 0 {
		out["reference_videos"] = append([]string(nil), req.ReferenceVideos...)
	}
	if len(req.ReferenceAudios) > 0 {
		out["reference_audios"] = append([]string(nil), req.ReferenceAudios...)
	}
	if value := strings.TrimSpace(req.FirstImageUrl); value != "" {
		out["first_image_url"] = value
	}
	if value := strings.TrimSpace(req.LastImageUrl); value != "" {
		out["last_image_url"] = value
	}
	body, err := common.Marshal(out)
	if err != nil {
		return nil, err
	}
	c.Request.Header.Set("Content-Type", "application/json")
	return bytes.NewReader(body), nil
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
	return &relaycommon.TaskResultSource{
		URL:     strings.TrimRight(baseURL, "/") + "/v1/videos/" + taskID + "/content",
		Headers: headers,
	}
}

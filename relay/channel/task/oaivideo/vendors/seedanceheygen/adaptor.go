package seedanceheygen

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

func (*TaskAdaptor) GetModelList() []string { return []string{Model720p, Model1080p} }
func (*TaskAdaptor) GetChannelName() string { return "seedance-heygen" }

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return service.TaskErrorWrapperLocal(fmt.Errorf("Seedance 2.0 requests must use application/json"), "invalid_request", http.StatusBadRequest)
	}
	return a.TaskAdaptor.ValidateRequestAndSetAction(c, info)
}

func (*TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if info == nil || strings.TrimSpace(info.ChannelBaseUrl) == "" {
		return "", fmt.Errorf("Seedance 2.0 base url is empty")
	}
	return strings.TrimRight(info.ChannelBaseUrl, "/") + "/v1/videos", nil
}

func (*TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	if info == nil || strings.TrimSpace(info.ApiKey) == "" {
		return fmt.Errorf("Seedance 2.0 api key is empty")
	}
	req.Header.Set("Authorization", "Bearer "+info.ApiKey)
	req.Header.Set("Content-Type", "application/json")
	return nil
}

func (*TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	if info == nil {
		return nil, fmt.Errorf("Seedance 2.0 relay info is nil")
	}
	resolution, ok := resolutionForModel(info.OriginModelName)
	if !ok || !IsRelay(info.OriginModelName, info.UpstreamModelName) {
		return nil, fmt.Errorf("unsupported Seedance 2.0 model mapping")
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"model":      UpstreamModel,
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

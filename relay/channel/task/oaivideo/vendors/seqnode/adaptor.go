package seqnode

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/defaultvideo"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type TaskAdaptor struct{ defaultvideo.TaskAdaptor }

func (*TaskAdaptor) GetModelList() []string {
	return []string{upstreamImagineVideo, upstreamImagineVideo15}
}
func (*TaskAdaptor) GetChannelName() string { return "seqnode" }
func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if !strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return service.TaskErrorWrapperLocal(fmt.Errorf("xAI Grok video requests must use application/json"), "invalid_request", http.StatusBadRequest)
	}
	return a.TaskAdaptor.ValidateRequestAndSetAction(c, info)
}
func (*TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if info == nil || strings.TrimSpace(info.ChannelBaseUrl) == "" {
		return "", fmt.Errorf("xAI Grok video base url is empty")
	}
	return strings.TrimRight(info.ChannelBaseUrl, "/") + "/v1/videos/generations", nil
}
func (*TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	if info == nil || info.ApiKey == "" {
		return fmt.Errorf("xAI Grok video api key is empty")
	}
	req.Header.Set("Authorization", "Bearer "+info.ApiKey)
	req.Header.Set("Content-Type", "application/json")
	return nil
}
func (*TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	name := info.UpstreamModelName
	out := map[string]any{"model": name, "prompt": strings.TrimSpace(req.Prompt)}
	if d := req.RequestedDurationSeconds(); d > 0 {
		out["duration"] = d
	}
	if req.AspectRatio != "" {
		out["aspect_ratio"] = req.AspectRatio
	}
	if req.Resolution != "" {
		out["resolution"] = req.Resolution
	}
	if len(req.Images) == 1 {
		out["image"] = map[string]any{"url": req.Images[0]}
	} else if len(req.Images) > 1 {
		refs := make([]map[string]any, 0, len(req.Images))
		for _, u := range req.Images {
			refs = append(refs, map[string]any{"url": u})
		}
		out["reference_images"] = refs
	}
	b, err := common.Marshal(out)
	return bytes.NewReader(b), err
}
func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, body io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, body)
}
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	b, e := io.ReadAll(resp.Body)
	if e != nil {
		return "", nil, service.TaskErrorWrapper(e, "read_response_body_failed", 500)
	}
	_ = resp.Body.Close()
	clone := *resp
	clone.Body = io.NopCloser(bytes.NewReader(normalize(b)))
	return a.TaskAdaptor.DoResponse(c, &clone, info)
}
func (a *TaskAdaptor) FetchTask(base, key string, body map[string]any, proxy string) (*http.Response, error) {
	id, _ := body["task_id"].(string)
	req, e := http.NewRequest(http.MethodGet, strings.TrimRight(base, "/")+"/v1/videos/"+id, nil)
	if e != nil {
		return nil, e
	}
	req.Header.Set("Authorization", "Bearer "+key)
	cl, e := service.GetHttpClientWithProxy(proxy)
	if e != nil {
		return nil, e
	}
	return cl.Do(req)
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
func (a *TaskAdaptor) ParseTaskResult(b []byte) (*relaycommon.TaskInfo, error) {
	return a.TaskAdaptor.ParseTaskResult(normalize(b))
}
func (a *TaskAdaptor) ConvertToOpenAIVideo(t *model.Task) ([]byte, error) {
	if t == nil {
		return nil, fmt.Errorf("task is nil")
	}
	c := *t
	c.Data = normalize(t.Data)
	return a.TaskAdaptor.ConvertToOpenAIVideo(&c)
}
func normalize(b []byte) []byte {
	var m map[string]any
	if common.Unmarshal(b, &m) != nil {
		return b
	}
	if id, ok := m["request_id"].(string); ok {
		m["id"] = id
	}
	if s, ok := m["status"].(string); ok && strings.EqualFold(s, "done") {
		m["status"] = "completed"
	}
	if v, ok := m["video"].(map[string]any); ok {
		if u, ok := v["url"].(string); ok {
			m["video_url"] = u
		}
	}
	o, e := common.Marshal(m)
	if e != nil {
		return b
	}
	return o
}

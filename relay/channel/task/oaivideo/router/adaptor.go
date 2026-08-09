package router

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/registry"
	oaivideo "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/shared"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/adobe"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/chatvideo"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/defaultvideo"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/geeknowgrok"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/grok"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/manju"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/omnii2v"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/omniv2v"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/sd5"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/seedanceheygen"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/seedanceleonardo"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/seedanceoairegbox"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/seedancetengda"
	seqnode "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/seqnode"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

type delegate interface {
	Init(info *relaycommon.RelayInfo)
	ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError
	EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64
	AdjustBillingOnSubmit(info *relaycommon.RelayInfo, taskData []byte) map[string]float64
	AdjustBillingOnComplete(task *model.Task, taskResult *relaycommon.TaskInfo) int
	BuildRequestURL(info *relaycommon.RelayInfo) (string, error)
	BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error
	BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error)
	DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error)
	DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, err *dto.TaskError)
	GetModelList() []string
	GetChannelName() string
	FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error)
	ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error)
}

type openAIVideoDelegate interface {
	ConvertToOpenAIVideo(task *model.Task) ([]byte, error)
}

type taskResultSourceDelegate interface {
	ResolveTaskResultSource(baseURL, taskID, key string) *relaycommon.TaskResultSource
}

// RouterAdaptor selects a vendor during submission, then reuses the persisted
// vendor for the complete task lifecycle.
type RouterAdaptor struct {
	native            delegate
	adobe             delegate
	chat              delegate
	grok              delegate
	geeknowGrok       delegate
	seqnode           delegate
	manju             delegate
	omniI2V           delegate
	omniV2V           delegate
	sd5               delegate
	seedanceOairegbox delegate
	seedanceLeonardo  delegate
	seedanceHeygen    delegate
	seedanceTengda    delegate
}

func NewRouterAdaptor() channel.TaskAdaptor {
	return &RouterAdaptor{
		native:            &defaultvideo.TaskAdaptor{},
		adobe:             &adobe.TaskAdaptor{},
		chat:              &chatvideo.TaskAdaptor{},
		grok:              &grok.TaskAdaptor{},
		geeknowGrok:       &geeknowgrok.TaskAdaptor{},
		seqnode:           &seqnode.TaskAdaptor{},
		manju:             &manju.TaskAdaptor{},
		omniI2V:           &omnii2v.TaskAdaptor{},
		omniV2V:           &omniv2v.TaskAdaptor{},
		sd5:               &sd5.TaskAdaptor{},
		seedanceOairegbox: &seedanceoairegbox.TaskAdaptor{},
		seedanceLeonardo:  &seedanceleonardo.TaskAdaptor{},
		seedanceHeygen:    &seedanceheygen.TaskAdaptor{},
		seedanceTengda:    &seedancetengda.TaskAdaptor{},
	}
}

func (r *RouterAdaptor) delegateFor(info *relaycommon.RelayInfo) delegate {
	if r == nil || info == nil {
		return nil
	}
	persisted := strings.TrimSpace(info.TaskVendor)
	vendor, ok := registry.ParseVendor(persisted)
	if persisted == "" {
		vendor = registry.ResolveSubmission(info.OriginModelName, info.UpstreamModelName, info.ChannelId, info.ChannelBaseUrl)
	} else if !ok {
		vendor = registry.VendorSora
	}
	switch vendor {
	case registry.VendorAdobe:
		return r.adobe
	case registry.VendorChat:
		return r.chat
	case registry.VendorGrok:
		return r.grok
	case registry.VendorGeeknowGrok:
		return r.geeknowGrok
	case registry.VendorSeqnode:
		return r.seqnode
	case registry.VendorManju:
		return r.manju
	case registry.VendorOmniI2V:
		return r.omniI2V
	case registry.VendorOmniV2V:
		return r.omniV2V
	case registry.VendorSD5:
		return r.sd5
	case registry.VendorSeedanceOairegbox:
		return r.seedanceOairegbox
	case registry.VendorSeedanceLeonardo:
		return r.seedanceLeonardo
	case registry.VendorSeedanceHeygen:
		return r.seedanceHeygen
	case registry.VendorSeedanceTengda:
		return r.seedanceTengda
	default:
		return r.native
	}
}

func (r *RouterAdaptor) delegateForTask(task *model.Task) delegate {
	if task == nil {
		return nil
	}
	info := registry.RelayInfoFromTask(task)
	info.TaskVendor = string(registry.ResolveTask(task))
	return r.delegateFor(info)
}

func (r *RouterAdaptor) ResolveTaskVendor(info *relaycommon.RelayInfo) string {
	if info == nil {
		return ""
	}
	return string(registry.ResolveSubmission(info.OriginModelName, info.UpstreamModelName, info.ChannelId, info.ChannelBaseUrl))
}

func (r *RouterAdaptor) Init(info *relaycommon.RelayInfo) {
	if d := r.delegateFor(info); d != nil {
		d.Init(info)
	}
}

func (r *RouterAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	d := r.delegateFor(info)
	if d == nil {
		return nil
	}
	return d.ValidateRequestAndSetAction(c, info)
}

func (r *RouterAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	d := r.delegateFor(info)
	if d == nil {
		return nil
	}
	return d.EstimateBilling(c, info)
}

func (r *RouterAdaptor) AdjustBillingOnSubmit(info *relaycommon.RelayInfo, taskData []byte) map[string]float64 {
	d := r.delegateFor(info)
	if d == nil {
		return nil
	}
	return d.AdjustBillingOnSubmit(info, taskData)
}

func (r *RouterAdaptor) AdjustBillingOnComplete(task *model.Task, taskResult *relaycommon.TaskInfo) int {
	if r == nil || task == nil {
		return 0
	}
	d := r.delegateForTask(task)
	if d == nil {
		return 0
	}
	return d.AdjustBillingOnComplete(task, taskResult)
}

func (r *RouterAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	d := r.delegateFor(info)
	if d == nil {
		return "", fmt.Errorf("video router delegate not available")
	}
	return d.BuildRequestURL(info)
}

func (r *RouterAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	d := r.delegateFor(info)
	if d == nil {
		return fmt.Errorf("video router delegate not available")
	}
	return d.BuildRequestHeader(c, req, info)
}

func (r *RouterAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	d := r.delegateFor(info)
	if d == nil {
		return nil, fmt.Errorf("video router delegate not available")
	}
	return d.BuildRequestBody(c, info)
}

func (r *RouterAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	d := r.delegateFor(info)
	if d == nil {
		return nil, fmt.Errorf("video router delegate not available")
	}
	return d.DoRequest(c, info, requestBody)
}

func (r *RouterAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, err *dto.TaskError) {
	d := r.delegateFor(info)
	if d == nil {
		return "", nil, service.TaskErrorWrapperLocal(fmt.Errorf("video router delegate not available"), "invalid_request", http.StatusInternalServerError)
	}
	return d.DoResponse(c, resp, info)
}

func (r *RouterAdaptor) GetModelList() []string {
	if r == nil {
		return nil
	}
	models := append([]string{}, r.native.GetModelList()...)
	models = append(models, r.adobe.GetModelList()...)
	models = append(models, r.chat.GetModelList()...)
	models = append(models, r.grok.GetModelList()...)
	models = append(models, r.geeknowGrok.GetModelList()...)
	models = append(models, r.seqnode.GetModelList()...)
	models = append(models, r.manju.GetModelList()...)
	models = append(models, r.sd5.GetModelList()...)
	models = append(models, r.seedanceOairegbox.GetModelList()...)
	models = append(models, r.seedanceLeonardo.GetModelList()...)
	models = append(models, r.seedanceHeygen.GetModelList()...)
	return append(models, r.seedanceTengda.GetModelList()...)
}

func (r *RouterAdaptor) GetChannelName() string {
	return "openai-video"
}

func (r *RouterAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	info := &relaycommon.RelayInfo{
		OriginModelName: stringFromBody(body, "origin_model"),
		TaskVendor:      stringFromBody(body, "task_vendor"),
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:         intFromBody(body, "channel_id"),
			ChannelBaseUrl:    baseUrl,
			UpstreamModelName: stringFromBody(body, "upstream_model"),
		},
	}
	if d := r.delegateFor(info); d != nil {
		return d.FetchTask(baseUrl, key, body, proxy)
	}
	return oaivideo.FetchVideoTask(baseUrl, key, body, proxy)
}

func intFromBody(body map[string]any, key string) int {
	switch value := body[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func stringFromBody(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func (r *RouterAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	return r.parseTaskResultBody(respBody, nil)
}

// ParseTaskResultForTask 轮询阶段按任务模型 + 响应形态解析（实现 channel.TaskAwareResultParser）。
func (r *RouterAdaptor) ParseTaskResultForTask(task *model.Task, respBody []byte) (*relaycommon.TaskInfo, error) {
	return r.parseTaskResultBody(respBody, task)
}

func (r *RouterAdaptor) ResolveTaskResultSourceForTask(task *model.Task, baseURL, key string) *relaycommon.TaskResultSource {
	if task == nil {
		return nil
	}
	d := r.delegateForTask(task)
	resolver, ok := d.(taskResultSourceDelegate)
	if !ok {
		return nil
	}
	return resolver.ResolveTaskResultSource(baseURL, task.GetUpstreamTaskID(), key)
}

func (r *RouterAdaptor) parseTaskResultBody(respBody []byte, task *model.Task) (*relaycommon.TaskInfo, error) {
	if r == nil {
		return nil, fmt.Errorf("video router adaptor not available")
	}
	if task != nil {
		if d := r.delegateForTask(task); d != nil {
			return d.ParseTaskResult(respBody)
		}
	}
	if manju.IsResponse(respBody) {
		return r.manju.ParseTaskResult(respBody)
	}
	return r.native.ParseTaskResult(respBody)
}

func (r *RouterAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	if r == nil || task == nil {
		return nil, fmt.Errorf("video router adaptor not available")
	}
	d := r.delegateForTask(task)
	if d == nil {
		return nil, fmt.Errorf("video router delegate not available")
	}
	if conv, ok := d.(openAIVideoDelegate); ok {
		return conv.ConvertToOpenAIVideo(task)
	}
	return nil, nil
}

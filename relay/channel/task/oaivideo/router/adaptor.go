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
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/sd5"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/seedance"
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

// RouterAdaptor 按模型路由到独立 vendor，避免上游协议互相污染。
type RouterAdaptor struct {
	native   delegate
	adobe    delegate
	chat     delegate
	grok        delegate
	geeknowGrok delegate
	manju    delegate
	sd5      delegate
	seedance delegate
}

func NewRouterAdaptor() channel.TaskAdaptor {
	return &RouterAdaptor{
		native:   &defaultvideo.TaskAdaptor{},
		adobe:    &adobe.TaskAdaptor{},
		chat:     &chatvideo.TaskAdaptor{},
		grok:        &grok.TaskAdaptor{},
		geeknowGrok: &geeknowgrok.TaskAdaptor{},
		manju:    &manju.TaskAdaptor{},
		sd5:      &sd5.TaskAdaptor{},
		seedance: &seedance.TaskAdaptor{},
	}
}

func (r *RouterAdaptor) delegateFor(info *relaycommon.RelayInfo) delegate {
	if r == nil {
		return nil
	}
	switch registry.ResolveWithChannel(info.OriginModelName, info.UpstreamModelName, info.ChannelId, info.ChannelBaseUrl) {
	case registry.VendorAdobe:
		return r.adobe
	case registry.VendorChat:
		return r.chat
	case registry.VendorGrok:
		return r.grok
	case registry.VendorGeeknowGrok:
		return r.geeknowGrok
	case registry.VendorManju:
		return r.manju
	case registry.VendorSD5:
		return r.sd5
	case registry.VendorSeedance:
		return r.seedance
	default:
		return r.native
	}
}

func (r *RouterAdaptor) delegateForTask(task *model.Task) delegate {
	return r.delegateFor(registry.RelayInfoFromTask(task))
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
	models = append(models, r.manju.GetModelList()...)
	models = append(models, r.sd5.GetModelList()...)
	return append(models, r.seedance.GetModelList()...)
}

func (r *RouterAdaptor) GetChannelName() string {
	return "openai-video"
}

func (r *RouterAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	info := &relaycommon.RelayInfo{
		OriginModelName: stringFromBody(body, "origin_model"),
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

func (r *RouterAdaptor) parseTaskResultBody(respBody []byte, task *model.Task) (*relaycommon.TaskInfo, error) {
	if r == nil {
		return nil, fmt.Errorf("video router adaptor not available")
	}
	if manju.IsResponse(respBody) {
		return r.manju.ParseTaskResult(respBody)
	}
	if task != nil {
		info := registry.RelayInfoFromTask(task)
		upstreamModel := ""
		if info.ChannelMeta != nil {
			upstreamModel = info.ChannelMeta.UpstreamModelName
		}
		switch registry.ResolveWithChannel(info.OriginModelName, upstreamModel, task.ChannelId, "") {
		case registry.VendorAdobe:
			return r.adobe.ParseTaskResult(respBody)
		case registry.VendorChat:
			return r.chat.ParseTaskResult(respBody)
		case registry.VendorGrok:
			return r.grok.ParseTaskResult(respBody)
		case registry.VendorGeeknowGrok:
			return r.geeknowGrok.ParseTaskResult(respBody)
		case registry.VendorManju:
			return r.manju.ParseTaskResult(respBody)
		case registry.VendorSD5:
			return r.sd5.ParseTaskResult(respBody)
		case registry.VendorSeedance:
			return r.seedance.ParseTaskResult(respBody)
		}
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

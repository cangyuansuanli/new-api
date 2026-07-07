package openaivideo

import (
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/manjusora"
	"github.com/QuantumNous/new-api/relay/channel/task/seedance"
	"github.com/QuantumNous/new-api/relay/channel/task/sora"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

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

// RouterAdaptor 按模型路由到独立适配器，避免 Sora / Manju / Seedance 互相污染。
type RouterAdaptor struct {
	sora    delegate
	manju   delegate
	seedance delegate
}

func NewRouterAdaptor() channel.TaskAdaptor {
	return &RouterAdaptor{
		sora:     &sora.TaskAdaptor{},
		manju:    &manjusora.TaskAdaptor{},
		seedance: &seedance.TaskAdaptor{},
	}
}

func (r *RouterAdaptor) pick(info *relaycommon.RelayInfo) delegate {
	if info == nil {
		return r.sora
	}
	if manjusora.IsRelay(info.OriginModelName, info.UpstreamModelName) {
		return r.manju
	}
	if seedance.IsRelay(info.OriginModelName, info.UpstreamModelName) {
		return r.seedance
	}
	return r.sora
}

func (r *RouterAdaptor) Init(info *relaycommon.RelayInfo) {
	r.pick(info).Init(info)
}

func (r *RouterAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	return r.pick(info).ValidateRequestAndSetAction(c, info)
}

func (r *RouterAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	return r.pick(info).EstimateBilling(c, info)
}

func (r *RouterAdaptor) AdjustBillingOnSubmit(info *relaycommon.RelayInfo, taskData []byte) map[string]float64 {
	return r.pick(info).AdjustBillingOnSubmit(info, taskData)
}

func (r *RouterAdaptor) AdjustBillingOnComplete(task *model.Task, taskResult *relaycommon.TaskInfo) int {
	info := &relaycommon.RelayInfo{
		OriginModelName: task.Properties.OriginModelName,
	}
	return r.pick(info).AdjustBillingOnComplete(task, taskResult)
}

func (r *RouterAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return r.pick(info).BuildRequestURL(info)
}

func (r *RouterAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	return r.pick(info).BuildRequestHeader(c, req, info)
}

func (r *RouterAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	return r.pick(info).BuildRequestBody(c, info)
}

func (r *RouterAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return r.pick(info).DoRequest(c, info, requestBody)
}

func (r *RouterAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, err *dto.TaskError) {
	return r.pick(info).DoResponse(c, resp, info)
}

func (r *RouterAdaptor) GetModelList() []string {
	return append(append(r.sora.GetModelList(), r.manju.GetModelList()...), r.seedance.GetModelList()...)
}

func (r *RouterAdaptor) GetChannelName() string {
	return "openai-video"
}

func (r *RouterAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	return r.sora.FetchTask(baseUrl, key, body, proxy)
}

func (r *RouterAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	if manjusora.IsResponse(respBody) {
		return r.manju.ParseTaskResult(respBody)
	}
	return r.sora.ParseTaskResult(respBody)
}

func (r *RouterAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	info := &relaycommon.RelayInfo{OriginModelName: task.Properties.OriginModelName}
	d := r.pick(info)
	if conv, ok := d.(openAIVideoDelegate); ok {
		return conv.ConvertToOpenAIVideo(task)
	}
	return nil, nil
}

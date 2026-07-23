package registry

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/adobe"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/chatvideo"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/geeknowgrok"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/grok"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/manju"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/omnii2v"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/omniv2v"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/sd5"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/seedanceleonardo"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/seedanceoairegbox"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/seedancetengda"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// Vendor 视频任务适配器族（提交阶段完整路由；轮询仅解析/计费分派）。
type Vendor string

const (
	VendorSora               Vendor = "sora"
	VendorAdobe              Vendor = "adobe"
	VendorChat               Vendor = "chat-video"
	VendorGrok               Vendor = "grok-generations"
	VendorGeeknowGrok        Vendor = "geeknow-grok"
	VendorManju              Vendor = "manju"
	VendorOmniI2V            Vendor = "omni-i2v"
	VendorOmniV2V            Vendor = "omni-v2v"
	VendorSD5                Vendor = "sd5-seedance"
	VendorSeedanceOairegbox  Vendor = "seedance-oairegbox"
	VendorSeedanceLeonardo   Vendor = "seedance-leonardo"
	VendorSeedanceTengda     Vendor = "seedance-tengda"
)

func IsOmniVideoModel(originModel, upstreamModel string) bool {
	return omniv2v.IsRelay(originModel, upstreamModel) || omnii2v.IsRelay(originModel, upstreamModel)
}

// Resolve 按 internal/upstream 模型名解析 Vendor；供应商专用协议优先于默认 OpenAI Video。
func Resolve(originModel, upstreamModel string) Vendor {
	return ResolveWithChannel(originModel, upstreamModel, 0, "")
}

// ResolveWithChannel resolves vendor-specific request and response behavior.
func ResolveWithChannel(originModel, upstreamModel string, channelID int, baseURL string) Vendor {
	if sd5.IsRelay(originModel, upstreamModel) {
		return VendorSD5
	}
	if adobe.IsRelay(originModel, upstreamModel, channelID, baseURL) {
		return VendorAdobe
	}
	if chatvideo.IsRelay(originModel) {
		return VendorChat
	}
	if geeknowgrok.IsRelay(originModel, upstreamModel) {
		return VendorGeeknowGrok
	}
	if grok.IsRelay(originModel, upstreamModel) {
		return VendorGrok
	}
	if manju.IsRelay(originModel, upstreamModel) {
		return VendorManju
	}
	if omniv2v.IsRelay(originModel, upstreamModel) {
		return VendorOmniV2V
	}
	if omnii2v.IsRelay(originModel, upstreamModel) {
		return VendorOmniI2V
	}
	if seedancetengda.IsRelay(originModel, upstreamModel) {
		return VendorSeedanceTengda
	}
	if seedanceleonardo.IsRelay(originModel) {
		return VendorSeedanceLeonardo
	}
	if seedanceoairegbox.IsRelay(originModel) {
		return VendorSeedanceOairegbox
	}
	return VendorSora
}

// RelayInfoFromTask 从任务记录还原路由所需的模型信息（轮询/查询阶段使用）。
func RelayInfoFromTask(task *model.Task) *relaycommon.RelayInfo {
	if task == nil {
		return &relaycommon.RelayInfo{}
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: task.Properties.OriginModelName,
	}
	upstream := task.Properties.UpstreamModelName
	if upstream == "" && task.PrivateData.BillingContext != nil {
		upstream = task.PrivateData.BillingContext.UpstreamModelName
	}
	if info.OriginModelName == "" && task.PrivateData.BillingContext != nil {
		info.OriginModelName = task.PrivateData.BillingContext.OriginModelName
	}
	if info.OriginModelName == "" {
		info.OriginModelName = upstreamModelFromTaskData(task.Data)
	}
	if task.ChannelId != 0 || upstream != "" {
		info.ChannelMeta = &relaycommon.ChannelMeta{
			ChannelId:         task.ChannelId,
			UpstreamModelName: upstream,
		}
	}
	return info
}

func upstreamModelFromTaskData(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	var m map[string]any
	if err := common.Unmarshal(data, &m); err != nil {
		return ""
	}
	if s, ok := m["model"].(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

package registry

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/manju"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/seedance"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// Vendor 视频任务适配器族（提交阶段完整路由；轮询仅解析/计费分派）。
type Vendor string

const (
	VendorSora     Vendor = "sora"
	VendorManju    Vendor = "manju"
	VendorSeedance Vendor = "seedance"
)

// Resolve 按 internal/upstream 模型名解析 Vendor（注册顺序：Manju → Seedance → Sora）。
func Resolve(originModel, upstreamModel string) Vendor {
	if manju.IsRelay(originModel, upstreamModel) {
		return VendorManju
	}
	if seedance.IsRelay(originModel, upstreamModel) {
		return VendorSeedance
	}
	return VendorSora
}

// RelayInfoFromTask 从任务记录还原路由所需的模型信息（轮询/查询阶段使用）。
func RelayInfoFromTask(task *model.Task) *relaycommon.RelayInfo {
	if task == nil {
		return &relaycommon.RelayInfo{}
	}
	info := &relaycommon.RelayInfo{
		OriginModelName:   task.Properties.OriginModelName,
		UpstreamModelName: task.Properties.UpstreamModelName,
	}
	if info.UpstreamModelName == "" && task.PrivateData.BillingContext != nil {
		info.UpstreamModelName = task.PrivateData.BillingContext.UpstreamModelName
	}
	if info.OriginModelName == "" && task.PrivateData.BillingContext != nil {
		info.OriginModelName = task.PrivateData.BillingContext.OriginModelName
	}
	if info.OriginModelName == "" {
		info.OriginModelName = upstreamModelFromTaskData(task.Data)
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

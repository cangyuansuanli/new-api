package imagevendor

import "github.com/QuantumNous/new-api/dto"

// RehostPolicy 描述某模型在上游出图后如何转存 R2。
type RehostPolicy struct {
	// AcceptUpstreamURL：上游可回 url，需下载并转存 R2（默认仅接受 b64_json）。
	AcceptUpstreamURL bool
	// PreferUpstreamB64JSON：仅供无法可靠下载上游 URL 的兼容渠道使用。
	PreferUpstreamB64JSON bool
	// AsyncPreferURLResponse：异步任务提交上游时使用 response_format=url。
	AsyncPreferURLResponse bool
}

// RequestPatchResult 请求补丁副作用：计费日志元数据。
type RequestPatchResult struct {
	// LogSize overrides request.Size in consume logs when non-empty (user-facing size).
	LogSize string
	// SuppressQualityLog omits the quality field from consume logs when the patcher
	// strips quality from the upstream request.
	SuppressQualityLog bool
}

// PatchRequestFunc 在发往上游前就地修改 ImageRequest；可按 originModel 决定是否生效。
type PatchRequestFunc func(originModel string, request *dto.ImageRequest) (RequestPatchResult, error)

// Descriptor 描述一类 image 渠道族：模型匹配、R2 转存策略、可选请求补丁。
// 协议级转换见 relay/channel/openai（如 Manju Image API body）。
type Descriptor struct {
	Name         string
	Match        func(originModel string) bool
	Rehost       RehostPolicy
	PatchRequest PatchRequestFunc
}

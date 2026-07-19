package service

import (
	"strings"

	"github.com/QuantumNous/new-api/model"
)

var clientFacingCopyReplacements = []struct {
	old string
	new string
}{
	{"Adobe2API Firefly 视频：", "OpenAI Video 兼容接口："},
	{"Adobe Firefly ", ""},
	{"Adobe2API ", ""},
	{"并发生成可能触发上游限流", "并发生成可能触发限流"},
	{"输出由上游固定为 PNG", "输出固定为 PNG"},
	{"省略时按上游默认值处理", "省略时按平台默认值处理"},
	{"为降低网页上游超时概率", "为降低异步超时概率"},
	{"网页线路仅保证", "平台仅保证"},
	{"网页线路", "平台"},
	{"Leonardo 订阅号 1300 积分号池，", ""},
	{"Leonardo Seedance", "Seedance"},
	{"Leonardo 1300 积分号池专用", "Mini 8 秒特惠专用"},
	{"cy-img1-gpt-image-2", "gpt-image-2"},
	{"固定传模型广场展示名 cy-img1-gpt-image-2。", "必填，传模型广场展示名（{{model}}）。"},
	{"勿传上游名 omni-fast-v2v-no-water", "请传 public 名 omni-v2v-no-water"},
	{"勿传上游名 omni-fast-v2v", "请传 public 名 omni-v2v"},
	{"（Gemini Veo）", ""},
	{"OAIREGBox ", ""},
	{
		"请求由上游网页生成能力执行，不等同于 OpenAI 官方 GPT Image API；仅保证下列基础参数生效。",
		"支持文生图和上传参考图后的图生图/编辑；下列参数为平台保证生效的基础项。",
	},
	{"video-tpl-cy-sd4-seedance-async", "video-tpl-seedance-subscription-async"},
	{"video-tpl-cy-sd5-seedance-933-async", "video-tpl-seedance-fullref-async"},
	{"video-tpl-cy-sd4-seedance-mini-8s", "video-tpl-seedance-mini-8s-async"},
	{"seedance-cy-sd4-mini-8s", "seedance-mini-8s"},
	{"image-tpl-adobe2api-nano-banana-pro-", "image-tpl-nano-banana-pro-"},
	{"image-tpl-adobe2api-nano-banana2-", "image-tpl-nano-banana2-"},
	{"image-tpl-adobe2api-gpt-image-2-", "image-tpl-gpt-image-2-"},
	{"image-tpl-adobe2api-1k", "image-tpl-nano-banana-tier-1k"},
	{"image-tpl-adobe2api-2k", "image-tpl-nano-banana-tier-2k"},
	{"image-tpl-adobe2api-4k", "image-tpl-nano-banana-tier-4k"},
}

func sanitizeClientFacingCopyString(value string) string {
	out := value
	for _, pair := range clientFacingCopyReplacements {
		if strings.Contains(out, pair.old) {
			out = strings.ReplaceAll(out, pair.old, pair.new)
		}
	}
	return out
}

func sanitizeClientFacingCopyMap(value map[string]interface{}) {
	if value == nil {
		return
	}
	for key, raw := range value {
		switch typed := raw.(type) {
		case string:
			value[key] = sanitizeClientFacingCopyString(typed)
		case map[string]interface{}:
			sanitizeClientFacingCopyMap(typed)
		case []interface{}:
			sanitizeClientFacingCopySlice(typed)
		}
	}
}

func sanitizeClientFacingCopySlice(items []interface{}) {
	for i, raw := range items {
		switch typed := raw.(type) {
		case string:
			items[i] = sanitizeClientFacingCopyString(typed)
		case map[string]interface{}:
			sanitizeClientFacingCopyMap(typed)
		case []interface{}:
			sanitizeClientFacingCopySlice(typed)
		}
	}
}

// SanitizeClientFacingPricing strips upstream/channel identifiers from pricing payloads
// served to the model marketplace and API docs.
func SanitizeClientFacingPricing(pricing *model.Pricing) {
	if pricing == nil {
		return
	}
	pricing.Description = sanitizeClientFacingCopyString(pricing.Description)
	if pricing.ApiDoc != nil {
		sanitizeClientFacingCopyMap(pricing.ApiDoc)
	}
	if pricing.VideoUiParams != nil {
		sanitizeClientFacingCopyMap(pricing.VideoUiParams)
	}
	if pricing.ImageUiParams != nil {
		sanitizeClientFacingCopyMap(pricing.ImageUiParams)
	}
}

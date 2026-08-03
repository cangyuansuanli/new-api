package model

import "strings"

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
	{"固定传模型广场展示名 cy-img1-gpt-image-2。", "传模型广场展示名（{{model}}）。"},
	{"cy-img1-gpt-image-2", "gpt-image-2"},
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

func cloneClientFacingCopyMap(value map[string]interface{}) map[string]interface{} {
	if value == nil {
		return nil
	}
	out := make(map[string]interface{}, len(value))
	for key, raw := range value {
		out[key] = cloneClientFacingCopyValue(raw)
	}
	return out
}

func cloneClientFacingCopySlice(items []interface{}) []interface{} {
	if items == nil {
		return nil
	}
	out := make([]interface{}, len(items))
	for i, raw := range items {
		out[i] = cloneClientFacingCopyValue(raw)
	}
	return out
}

func cloneClientFacingCopyValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case string:
		return sanitizeClientFacingCopyString(typed)
	case map[string]interface{}:
		return cloneClientFacingCopyMap(typed)
	case []interface{}:
		return cloneClientFacingCopySlice(typed)
	default:
		return value
	}
}

func cloneClientFacingSlice[T any](items []T) []T {
	if items == nil {
		return nil
	}
	out := make([]T, len(items))
	copy(out, items)
	return out
}

func cloneClientFacingPointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	out := *value
	return &out
}

func buildClientFacingPricingSnapshot(pricing []Pricing) []Pricing {
	if pricing == nil {
		return nil
	}

	snapshot := make([]Pricing, len(pricing))
	for i, item := range pricing {
		snapshot[i] = item
		snapshot[i].Description = sanitizeClientFacingCopyString(item.Description)
		snapshot[i].EnableGroup = cloneClientFacingSlice(item.EnableGroup)
		snapshot[i].SupportedEndpointTypes = cloneClientFacingSlice(item.SupportedEndpointTypes)
		snapshot[i].CacheRatio = cloneClientFacingPointer(item.CacheRatio)
		snapshot[i].CreateCacheRatio = cloneClientFacingPointer(item.CreateCacheRatio)
		snapshot[i].ImageRatio = cloneClientFacingPointer(item.ImageRatio)
		snapshot[i].AudioRatio = cloneClientFacingPointer(item.AudioRatio)
		snapshot[i].AudioCompletionRatio = cloneClientFacingPointer(item.AudioCompletionRatio)
		snapshot[i].ApiDoc = cloneClientFacingCopyMap(item.ApiDoc)
		snapshot[i].VideoUiParams = cloneClientFacingCopyMap(item.VideoUiParams)
		snapshot[i].ImageUiParams = cloneClientFacingCopyMap(item.ImageUiParams)
	}
	return snapshot
}

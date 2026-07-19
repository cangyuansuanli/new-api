package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestSanitizeClientFacingPricingRemovesUpstreamCopy(t *testing.T) {
	pricing := &model.Pricing{
		Description: "Seedance 2.0 Mini 8 秒特惠。Leonardo 订阅号 1300 积分号池，480p / 720p，支持 4–8 秒。",
		ApiDoc: map[string]interface{}{
			"intro": "Adobe2API Firefly 视频：POST /v1/videos 创建异步任务。",
			"params": []interface{}{
				map[string]interface{}{
					"name":        "model",
					"description": "必填，固定传模型广场展示名 cy-img1-gpt-image-2。",
				},
			},
		},
		VideoUiParams: map[string]interface{}{
			"id": "video-tpl-cy-sd4-seedance-async",
			"hints": []interface{}{
				map[string]interface{}{"text": "Leonardo Seedance 2.0 Mini 8 秒特惠"},
			},
		},
		ImageUiParams: map[string]interface{}{
			"id": "image-tpl-adobe2api-gpt-image-2-1k",
			"hints": []interface{}{
				map[string]interface{}{"text": "Adobe2API 1K 固定档位"},
				map[string]interface{}{"text": "并发生成可能触发上游限流，建议控制单次张数与并发。"},
			},
		},
	}

	SanitizeClientFacingPricing(pricing)

	require.NotContains(t, pricing.Description, "Leonardo")
	require.Contains(t, pricing.Description, "480p / 720p")
	require.Contains(t, pricing.ApiDoc["intro"], "OpenAI Video 兼容接口")
	require.NotContains(t, pricing.ApiDoc["intro"], "Adobe2API")
	params := pricing.ApiDoc["params"].([]interface{})
	param := params[0].(map[string]interface{})
	require.Contains(t, param["description"], "{{model}}")
	require.NotContains(t, param["description"], "cy-img1")
	require.Equal(t, "video-tpl-seedance-subscription-async", pricing.VideoUiParams["id"])
	require.Equal(t, "image-tpl-gpt-image-2-1k", pricing.ImageUiParams["id"])
	require.NotContains(t, pricing.ImageUiParams["hints"].([]interface{})[0].(map[string]interface{})["text"], "上游")
}

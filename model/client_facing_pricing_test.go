package model

import (
	"strings"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestBuildClientFacingPricingSnapshotSanitizesWithoutMutatingSource(t *testing.T) {
	source := []Pricing{{
		Description:            "Seedance 2.0 Mini 8 秒特惠。Leonardo 订阅号 1300 积分号池，480p / 720p，支持 4–8 秒。",
		EnableGroup:            []string{"default"},
		SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAIVideo},
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
	}}

	snapshot := buildClientFacingPricingSnapshot(source)

	require.Contains(t, source[0].Description, "Leonardo")
	require.Contains(t, source[0].ApiDoc["intro"], "Adobe2API")
	require.NotContains(t, snapshot[0].Description, "Leonardo")
	require.Contains(t, snapshot[0].Description, "480p / 720p")
	require.Contains(t, snapshot[0].ApiDoc["intro"], "OpenAI Video 兼容接口")
	require.NotContains(t, snapshot[0].ApiDoc["intro"], "Adobe2API")

	params := snapshot[0].ApiDoc["params"].([]interface{})
	param := params[0].(map[string]interface{})
	require.Contains(t, param["description"], "{{model}}")
	require.NotContains(t, param["description"], "cy-img1")
	require.Equal(t, "video-tpl-seedance-subscription-async", snapshot[0].VideoUiParams["id"])
	require.Equal(t, "image-tpl-gpt-image-2-1k", snapshot[0].ImageUiParams["id"])
	require.NotContains(t, snapshot[0].ImageUiParams["hints"].([]interface{})[0].(map[string]interface{})["text"], "上游")
}

func TestBuildClientFacingPricingSnapshotDoesNotShareMutableData(t *testing.T) {
	cacheRatio := 0.5
	source := []Pricing{{
		EnableGroup:            []string{"default"},
		SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAI},
		CacheRatio:             &cacheRatio,
		ApiDoc: map[string]interface{}{
			"nested": []interface{}{map[string]interface{}{"value": "original"}},
		},
	}}

	snapshot := buildClientFacingPricingSnapshot(source)
	snapshot[0].EnableGroup[0] = "changed"
	snapshot[0].SupportedEndpointTypes[0] = constant.EndpointTypeOpenAIResponse
	*snapshot[0].CacheRatio = 0.8
	snapshot[0].ApiDoc["nested"].([]interface{})[0].(map[string]interface{})["value"] = "changed"

	require.Equal(t, "default", source[0].EnableGroup[0])
	require.Equal(t, constant.EndpointTypeOpenAI, source[0].SupportedEndpointTypes[0])
	require.Equal(t, 0.5, *source[0].CacheRatio)
	require.Equal(t, "original", source[0].ApiDoc["nested"].([]interface{})[0].(map[string]interface{})["value"])
}

func TestBuildClientFacingPricingSnapshotSupportsConcurrentRefreshes(t *testing.T) {
	source := []Pricing{{
		Description: "Adobe2API 并发生成可能触发上游限流",
		ApiDoc: map[string]interface{}{
			"nested": []interface{}{map[string]interface{}{"value": "Adobe Firefly image"}},
		},
	}}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				snapshot := buildClientFacingPricingSnapshot(source)
				if strings.Contains(snapshot[0].Description, "上游") {
					t.Errorf("client snapshot contains upstream copy: %q", snapshot[0].Description)
				}
				if got := snapshot[0].ApiDoc["nested"].([]interface{})[0].(map[string]interface{})["value"]; got != "image" {
					t.Errorf("unexpected nested copy: %v", got)
				}
			}
		}()
	}
	wg.Wait()

	require.Contains(t, source[0].Description, "Adobe2API")
}

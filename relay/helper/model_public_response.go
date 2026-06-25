package helper

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/gin-gonic/gin"
)

func SanitizeResponseBodyForPublicModel(c *gin.Context, body []byte) []byte {
	publicName := service.GetClientModelName(c)
	if publicName == "" || len(body) == 0 {
		return body
	}
	if !gjson.ValidBytes(body) {
		return body
	}
	result := body
	if gjson.GetBytes(result, "model").Exists() {
		patched, err := sjson.SetBytes(result, "model", publicName)
		if err == nil {
			result = patched
		}
	}
	data := gjson.GetBytes(result, "data")
	if data.IsArray() {
		for i, item := range data.Array() {
			if item.Get("model").Exists() {
				path := "data." + fmt.Sprintf("%d", i) + ".model"
				patched, err := sjson.SetBytes(result, path, publicName)
				if err == nil {
					result = patched
				}
			}
			id := item.Get("id").String()
			if id != "" && item.Get("model").Exists() {
				_ = id
			}
		}
	}
	return result
}

func SanitizeStreamChunkForPublicModel(c *gin.Context, data string) string {
	publicName := service.GetClientModelName(c)
	if publicName == "" || data == "" {
		return data
	}
	trimmed := strings.TrimSpace(data)
	if trimmed == "[DONE]" || !gjson.Valid(trimmed) {
		return data
	}
	patched, err := sjson.Set(trimmed, "model", publicName)
	if err != nil {
		return data
	}
	return patched
}

func SanitizeObjectForPublicModel(c *gin.Context, object interface{}) interface{} {
	publicName := service.GetClientModelName(c)
	if publicName == "" || object == nil {
		return object
	}
	raw, err := common.Marshal(object)
	if err != nil {
		return object
	}
	patched := SanitizeResponseBodyForPublicModel(c, raw)
	if len(patched) == len(raw) && string(patched) == string(raw) {
		return object
	}
	var out map[string]interface{}
	if err := common.Unmarshal(patched, &out); err != nil {
		return object
	}
	return out
}

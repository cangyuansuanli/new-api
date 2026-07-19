package service

import (
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/gin-gonic/gin"
)

// ClientTaskProperties 是对外任务日志/列表可见的 properties 子集（不含 internal/upstream 模型名）。
type ClientTaskProperties struct {
	Input     string `json:"input,omitempty"`
	ModelName string `json:"model_name,omitempty"`
	TaskKind  string `json:"task_kind,omitempty"`
}

// ClientFacingModelFromContext 返回同步响应对客户端展示的 public 模型名。
func ClientFacingModelFromContext(c *gin.Context) string {
	return GetClientModelName(c)
}

// ClientFacingModelFromTask 返回异步 task 响应对客户端展示的 public 模型名。
// 优先持久化的 ClientModelName；legacy 任务回退前缀剥离。
func ClientFacingModelFromTask(task *model.Task) string {
	if task == nil {
		return ""
	}
	return clientFacingModelFromProperties(task.Properties)
}

func clientFacingModelFromProperties(props model.Properties) string {
	if name := strings.TrimSpace(props.ClientModelName); name != "" {
		return name
	}
	if name := strings.TrimSpace(props.OriginModelName); name != "" {
		return ToPublicModelName(name)
	}
	return ""
}

// PatchClientFacingModelJSON 将响应 JSON 中的 model 字段改为 public 名（同步/异步共用）。
func PatchClientFacingModelJSON(publicName string, body []byte) ([]byte, error) {
	publicName = strings.TrimSpace(publicName)
	if publicName == "" || len(body) == 0 {
		return body, nil
	}
	if !gjson.ValidBytes(body) {
		return body, nil
	}
	result := body
	if gjson.GetBytes(result, "model").Exists() {
		patched, err := sjson.SetBytes(result, "model", publicName)
		if err != nil {
			return body, err
		}
		result = patched
	}
	data := gjson.GetBytes(result, "data")
	if data.IsArray() {
		for i, item := range data.Array() {
			if item.Get("model").Exists() {
				path := "data." + fmt.Sprintf("%d", i) + ".model"
				patched, err := sjson.SetBytes(result, path, publicName)
				if err != nil {
					return body, err
				}
				result = patched
			}
		}
	}
	return result, nil
}

// PatchClientFacingModelStreamChunk 将 stream chunk 中的 model 字段改为 public 名。
func PatchClientFacingModelStreamChunk(publicName string, data string) string {
	publicName = strings.TrimSpace(publicName)
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

// PatchClientFacingModelObject 将 struct/map 响应中的 model 字段改为 public 名。
func PatchClientFacingModelObject(publicName string, object interface{}) interface{} {
	publicName = strings.TrimSpace(publicName)
	if publicName == "" || object == nil {
		return object
	}
	raw, err := common.Marshal(object)
	if err != nil {
		return object
	}
	patched, err := PatchClientFacingModelJSON(publicName, raw)
	if err != nil || (len(patched) == len(raw) && string(patched) == string(raw)) {
		return object
	}
	var out map[string]interface{}
	if err := common.Unmarshal(patched, &out); err != nil {
		return object
	}
	return out
}

// PatchClientFacingModelJSONFromContext 用 gin context 中的 public 名 patch 响应 JSON。
func PatchClientFacingModelJSONFromContext(c *gin.Context, body []byte) ([]byte, error) {
	return PatchClientFacingModelJSON(ClientFacingModelFromContext(c), body)
}

// PatchClientFacingModelStreamChunkFromContext 用 gin context 中的 public 名 patch stream chunk。
func PatchClientFacingModelStreamChunkFromContext(c *gin.Context, data string) string {
	return PatchClientFacingModelStreamChunk(ClientFacingModelFromContext(c), data)
}

// PatchClientFacingModelObjectFromContext 用 gin context 中的 public 名 patch 对象响应。
func PatchClientFacingModelObjectFromContext(c *gin.Context, object interface{}) interface{} {
	return PatchClientFacingModelObject(ClientFacingModelFromContext(c), object)
}

// PatchClientFacingModelJSONFromTask 用 task 持久化的 public 名 patch 响应 JSON。
func PatchClientFacingModelJSONFromTask(task *model.Task, data []byte) ([]byte, error) {
	return PatchClientFacingModelJSON(ClientFacingModelFromTask(task), data)
}

// ClientFacingTaskProperties 返回任务 DTO 对客户端可见的 properties。
func ClientFacingTaskProperties(task *model.Task) *ClientTaskProperties {
	if task == nil {
		return nil
	}
	public := ClientFacingModelFromTask(task)
	if public == "" && strings.TrimSpace(task.Properties.Input) == "" && strings.TrimSpace(task.Properties.TaskKind) == "" {
		return nil
	}
	return &ClientTaskProperties{
		Input:     task.Properties.Input,
		ModelName: public,
		TaskKind:  task.Properties.TaskKind,
	}
}

// ApplyClientFacingModelNamesToLogs 将消费日志的 model_name 转为 public 名。
func ApplyClientFacingModelNamesToLogs(logs []*model.Log) {
	for i := range logs {
		if logs[i] == nil {
			continue
		}
		logs[i].ModelName = ToPublicModelName(logs[i].ModelName)
	}
}

// PublicQuotaDataForClient 将 quota_data 的 model_name 转为 public 名，并合并同一时段的重复 public 名。
func PublicQuotaDataForClient(items []*model.QuotaData) []*model.QuotaData {
	if len(items) == 0 {
		return items
	}

	type aggKey struct {
		userID    int
		username  string
		modelName string
		createdAt int64
	}

	merged := make(map[aggKey]*model.QuotaData, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		public := ToPublicModelName(item.ModelName)
		key := aggKey{
			userID:    item.UserID,
			username:  item.Username,
			modelName: public,
			createdAt: item.CreatedAt,
		}
		if existing, ok := merged[key]; ok {
			existing.Count += item.Count
			existing.Quota += item.Quota
			existing.TokenUsed += item.TokenUsed
			continue
		}
		copy := *item
		copy.ModelName = public
		copy.Id = 0
		merged[key] = &copy
	}

	out := make([]*model.QuotaData, 0, len(merged))
	for _, item := range merged {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt < out[j].CreatedAt
		}
		return out[i].ModelName < out[j].ModelName
	})
	return out
}

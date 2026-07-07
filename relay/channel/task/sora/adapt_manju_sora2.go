package sora

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/tidwall/gjson"
)

const manjuSora2UpstreamModel = "sora2"

// IsManjuSora2Relay Manju 渠道 #70：internal manju-openai-sora2 → 上游 sora2。
func IsManjuSora2Relay(originModel, upstreamModel string) bool {
	if strings.EqualFold(strings.TrimSpace(upstreamModel), manjuSora2UpstreamModel) {
		return true
	}
	origin := strings.ToLower(strings.TrimSpace(originModel))
	return origin == "manju-openai-sora2" || strings.HasPrefix(origin, "manju-openai-sora")
}

// ConvertManjuSora2ChatBody 将客户端 /v1/videos 或 chat 请求转为 Manju 官方格式（Apifox sora2）。
// 上游：POST /v1/chat/completions，字段 sora2_duration / sora2_ratio / messages / stream:false。
func ConvertManjuSora2ChatBody(body map[string]interface{}, upstreamModel string) (map[string]interface{}, error) {
	if body == nil {
		return nil, fmt.Errorf("empty request body")
	}
	prompt := strings.TrimSpace(asString(body["prompt"]))
	if prompt == "" {
		prompt = manjuSoraPromptFromMessages(body["messages"])
	}
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	out := map[string]interface{}{
		"model":  upstreamModel,
		"stream": false,
		"messages": []map[string]interface{}{
			{"role": "user", "content": prompt},
		},
	}

	if duration := manjuSoraDurationFromBody(body); duration != "" {
		out["sora2_duration"] = duration
	}
	if ratio := manjuSoraRatioFromBody(body); ratio != "" {
		out["sora2_ratio"] = ratio
	}
	if ref := manjuSoraInputReferenceFromBody(body); ref != "" {
		out["input_reference"] = ref
	}
	return out, nil
}

func manjuSoraPromptFromMessages(raw interface{}) string {
	items, ok := raw.([]interface{})
	if !ok {
		return ""
	}
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if content := strings.TrimSpace(asString(m["content"])); content != "" {
			return content
		}
	}
	return ""
}

func manjuSoraDurationFromBody(body map[string]interface{}) string {
	if s := strings.TrimSpace(asString(body["sora2_duration"])); s != "" {
		return s
	}
	return manjuDurationFromBody(body)
}

func manjuSoraRatioFromBody(body map[string]interface{}) string {
	if r := strings.TrimSpace(asString(body["sora2_ratio"])); r != "" {
		return r
	}
	if r := strings.TrimSpace(asString(body["aspect_ratio"])); r != "" {
		return r
	}
	return manjuAspectRatioFromSize(asString(body["size"]))
}

func manjuSoraInputReferenceFromBody(body map[string]interface{}) string {
	for _, key := range []string{"input_reference", "image_url"} {
		if ref := strings.TrimSpace(asString(body[key])); ref != "" {
			return ref
		}
	}
	if urls := collectStringList(body["reference_image_urls"]); len(urls) > 0 {
		return urls[0]
	}
	if urls := collectStringList(body["images"]); len(urls) > 0 {
		return urls[0]
	}
	return ""
}

func manjuAspectRatioFromSize(size string) string {
	switch strings.TrimSpace(size) {
	case "1280x720", "1792x1024":
		return "16:9"
	case "720x1280", "1024x1792":
		return "9:16"
	case "1024x1024":
		return "1:1"
	default:
		return ""
	}
}

func manjuOutputResolutionFromSize(size string) string {
	size = strings.TrimSpace(size)
	switch size {
	case "1280x720", "720x1280", "1024x1024":
		return "720p"
	case "1792x1024", "1024x1792":
		return "1080p"
	default:
		return ""
	}
}

func manjuDurationFromBody(body map[string]interface{}) string {
	if s := strings.TrimSpace(asString(body["seconds"])); s != "" {
		return s
	}
	if d := body["duration"]; d != nil {
		return strings.TrimSpace(asString(d))
	}
	return ""
}

func extractManjuSoraVideoURL(respBody []byte) string {
	if len(respBody) == 0 {
		return ""
	}
	paths := []string{
		"video.url",
		"metadata.url",
		"raw_data.video_url",
		"task_data.video_url",
		"data.data.video_url",
		"data.video_url",
		"final_url",
		"download_url",
		"result_url",
	}
	for _, path := range paths {
		if u := pickAbsoluteVideoURL(gjson.GetBytes(respBody, path).String()); u != "" {
			return u
		}
	}
	for _, arrPath := range []string{"raw_data.video_urls", "data.data.video_urls"} {
		arr := gjson.GetBytes(respBody, arrPath)
		if !arr.IsArray() {
			continue
		}
		for _, item := range arr.Array() {
			if u := pickAbsoluteVideoURL(item.String()); u != "" {
				return u
			}
		}
	}
	return ""
}

func usageSecondsFromManjuSoraBody(respBody []byte) int {
	if len(respBody) == 0 {
		return 0
	}
	paths := []string{
		"properties.duration",
		"data.properties.duration",
		"usage.seconds",
		"seconds",
	}
	for _, path := range paths {
		if sec := parsePositiveIntString(gjson.GetBytes(respBody, path).String()); sec > 0 {
			return sec
		}
	}
	if v := gjson.GetBytes(respBody, "usage.seconds").Float(); v > 0 {
		return int(v)
	}
	return 0
}

func extractManjuSoraFailReason(respBody []byte) string {
	if len(respBody) == 0 {
		return ""
	}
	for _, path := range []string{
		"fail_reason",
		"data.fail_reason",
		"message",
		"error.message",
		"data.message",
	} {
		raw := strings.TrimSpace(gjson.GetBytes(respBody, path).String())
		if raw == "" || raw == "[object Object]" {
			continue
		}
		return raw
	}
	return extractErrorMessage(respBody)
}

func isManjuSoraFailedStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "failure", "error", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func isGenericTaskFailureReason(reason string) bool {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "task failed", "upstream returned empty status", "upstream returned unrecognized message":
		return true
	default:
		return false
	}
}

// IsManjuSora2Response 检测 Manju sora2 轮询/创建响应（platform=sora2 或 id 前缀 sora2-）。
func IsManjuSora2Response(respBody []byte) bool {
	if len(respBody) == 0 {
		return false
	}
	platform := strings.ToLower(strings.TrimSpace(gjson.GetBytes(respBody, "platform").String()))
	if platform == manjuSora2UpstreamModel {
		return true
	}
	id := strings.TrimSpace(gjson.GetBytes(respBody, "id").String())
	if id == "" {
		id = strings.TrimSpace(gjson.GetBytes(respBody, "task_id").String())
	}
	return strings.HasPrefix(id, "sora2-")
}

func enrichManjuSoraTaskResult(taskResult *relaycommon.TaskInfo, respBody []byte) {
	if taskResult == nil || len(respBody) == 0 || !IsManjuSora2Response(respBody) {
		return
	}
	if taskResult.Url == "" {
		if url := extractManjuSoraVideoURL(respBody); url != "" {
			taskResult.Url = url
		}
	}
	if taskResult.CompletionTokens <= 0 {
		if sec := usageSecondsFromManjuSoraBody(respBody); sec > 0 {
			taskResult.CompletionTokens = sec
			taskResult.TotalTokens = sec
		}
	}
	if taskResult.Status == model.TaskStatusFailure {
		if reason := extractManjuSoraFailReason(respBody); reason != "" &&
			(taskResult.Reason == "" || isGenericTaskFailureReason(taskResult.Reason)) {
			taskResult.Reason = reason
		}
	}
}

func extractVideoURLWithManjuFallback(res responseTask, respBody []byte) string {
	if url := extractVideoURL(res); url != "" {
		return url
	}
	if IsManjuSora2Response(respBody) {
		return extractManjuSoraVideoURL(respBody)
	}
	return ""
}

func parseResponseTask(respBody []byte) (responseTask, error) {
	var res responseTask
	err := common.Unmarshal(respBody, &res)
	if err == nil {
		return res, nil
	}
	if !IsManjuSora2Response(respBody) {
		return res, err
	}
	return manjuSoraResponseTaskFromGJSON(respBody), nil
}

func manjuSoraResponseTaskFromGJSON(respBody []byte) responseTask {
	res := responseTask{
		ID:     strings.TrimSpace(gjson.GetBytes(respBody, "id").String()),
		TaskID: strings.TrimSpace(gjson.GetBytes(respBody, "task_id").String()),
		Object: strings.TrimSpace(gjson.GetBytes(respBody, "object").String()),
		Model:  strings.TrimSpace(gjson.GetBytes(respBody, "platform").String()),
		Status: strings.TrimSpace(gjson.GetBytes(respBody, "status").String()),
	}
	if res.ID == "" {
		res.ID = res.TaskID
	}
	if p := gjson.GetBytes(respBody, "progress"); p.Exists() {
		switch {
		case p.Type == gjson.Number:
			res.Progress = int(p.Int())
		default:
			if pct := parsePositiveIntString(strings.TrimSuffix(strings.TrimSpace(p.String()), "%")); pct > 0 {
				res.Progress = pct
			}
		}
	}
	if res.Object == "" {
		res.Object = "video"
	}
	if url := extractManjuSoraVideoURL(respBody); url != "" {
		res.VideoURL = url
	}
	if sec := usageSecondsFromManjuSoraBody(respBody); sec > 0 {
		res.Seconds = strconv.Itoa(sec)
	}
	return res
}

func buildOpenAIVideoCreateResponse(info *relaycommon.RelayInfo, res responseTask, respBody []byte) map[string]any {
	out := map[string]any{
		"id":     info.PublicTaskID,
		"object": "video",
		"model":  info.OriginModelName,
		"status": normalizeManjuSoraStatusForClient(res.Status),
	}
	if res.Progress > 0 {
		out["progress"] = res.Progress
	}
	if res.Seconds != "" {
		out["seconds"] = res.Seconds
	}
	if res.Size != "" {
		out["size"] = res.Size
	}
	if isManjuSoraFailedStatus(res.Status) {
		if reason := extractManjuSoraFailReason(respBody); reason != "" {
			out["error"] = map[string]any{"message": reason}
			out["fail_reason"] = reason
		}
	}
	return out
}

func normalizeManjuSoraStatusForClient(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running", "processing", "in_progress":
		return "in_progress"
	case "succeeded", "success", "done":
		return "completed"
	case "queued", "pending":
		return "queued"
	case "failed", "failure", "error", "cancelled", "canceled":
		return "failed"
	default:
		return strings.TrimSpace(status)
	}
}

// BuildManjuSoraOpenAIErrorResponse 将 Manju 失败 task 转为 OpenAI error JSON，供 chat/completions 客户端识别。
func BuildManjuSoraOpenAIErrorResponse(respBody []byte) ([]byte, bool) {
	if len(respBody) == 0 || !IsManjuSora2Response(respBody) {
		return nil, false
	}
	status := strings.TrimSpace(gjson.GetBytes(respBody, "status").String())
	if !isManjuSoraFailedStatus(status) {
		return nil, false
	}
	reason := extractManjuSoraFailReason(respBody)
	if reason == "" {
		reason = "task failed"
	}
	out, err := common.Marshal(map[string]any{
		"error": map[string]any{
			"message": reason,
			"type":    "upstream_error",
		},
	})
	if err != nil {
		return nil, false
	}
	return out, true
}

// ExtractManjuSoraFailReasonForChat 提取非 task 形态的上游错误（如创建阶段仅返回 message）。
func ExtractManjuSoraFailReasonForChat(respBody []byte) string {
	if len(respBody) == 0 || IsManjuSora2Response(respBody) {
		return ""
	}
	return extractManjuSoraFailReason(respBody)
}

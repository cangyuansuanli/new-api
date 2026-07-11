package manju

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	oaivideo "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/shared"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/tidwall/gjson"
)

// ConvertChatBody 将客户端 /v1/videos 请求转为 Manju 官方 chat/completions 格式。
func ConvertChatBody(body map[string]interface{}, upstreamModel string) (map[string]interface{}, error) {
	if body == nil {
		return nil, fmt.Errorf("empty request body")
	}
	if strings.TrimSpace(upstreamModel) == "" {
		upstreamModel = upstreamModelName
	}

	prompt := strings.TrimSpace(oaivideo.AsString(body["prompt"]))
	if prompt == "" {
		prompt = promptFromMessages(body["messages"])
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

	if duration := durationFromBody(body); duration != "" {
		out["sora2_duration"] = duration
	}
	if ratio := ratioFromBody(body); ratio != "" {
		out["sora2_ratio"] = ratio
	}
	if resolution := outputResolutionFromBody(body); resolution != "" {
		out["sora2_output_resolution"] = resolution
	}
	if ref := inputReferenceFromBody(body); ref != "" {
		out["input_reference"] = ref
	}
	return out, nil
}

const upstreamModelName = "sora2"

func promptFromMessages(raw interface{}) string {
	items, ok := raw.([]interface{})
	if !ok {
		return ""
	}
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if content := strings.TrimSpace(oaivideo.AsString(m["content"])); content != "" {
			return content
		}
	}
	return ""
}

func durationFromBody(body map[string]interface{}) string {
	if s := strings.TrimSpace(oaivideo.AsString(body["sora2_duration"])); s != "" {
		return s
	}
	if s := strings.TrimSpace(oaivideo.AsString(body["seconds"])); s != "" {
		return s
	}
	if d := body["duration"]; d != nil {
		return strings.TrimSpace(oaivideo.AsString(d))
	}
	return ""
}

func ratioFromBody(body map[string]interface{}) string {
	if r := strings.TrimSpace(oaivideo.AsString(body["sora2_ratio"])); r != "" {
		return r
	}
	if r := strings.TrimSpace(oaivideo.AsString(body["aspect_ratio"])); r != "" {
		return r
	}
	return aspectRatioFromSize(oaivideo.AsString(body["size"]))
}

func outputResolutionFromBody(body map[string]interface{}) string {
	if r := strings.TrimSpace(oaivideo.AsString(body["sora2_output_resolution"])); r != "" {
		return r
	}
	if r := strings.TrimSpace(oaivideo.AsString(body["output_resolution"])); r != "" {
		return r
	}
	if r := strings.TrimSpace(oaivideo.AsString(body["resolution"])); r != "" {
		return normalizeOutputResolution(r)
	}
	return outputResolutionFromSize(oaivideo.AsString(body["size"]))
}

func inputReferenceFromBody(body map[string]interface{}) string {
	for _, key := range []string{"input_reference", "image_url"} {
		if ref := strings.TrimSpace(oaivideo.AsString(body[key])); ref != "" {
			return ref
		}
	}
	if urls := oaivideo.CollectStringList(body["reference_image_urls"]); len(urls) > 0 {
		return urls[0]
	}
	if urls := oaivideo.CollectStringList(body["images"]); len(urls) > 0 {
		return urls[0]
	}
	return ""
}

func aspectRatioFromSize(size string) string {
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

func outputResolutionFromSize(size string) string {
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

func normalizeOutputResolution(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(raw, "1080"), raw == "hd":
		return "1080p"
	case strings.Contains(raw, "720"):
		return "720p"
	default:
		return strings.TrimSpace(oaivideo.AsString(raw))
	}
}

func ExtractFailReason(respBody []byte) string {
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
	return oaivideo.ExtractErrorMessage(respBody)
}

func isFailedStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "failure", "error", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func IsResponse(respBody []byte) bool {
	if len(respBody) == 0 {
		return false
	}
	platform := strings.ToLower(strings.TrimSpace(gjson.GetBytes(respBody, "platform").String()))
	if platform == upstreamModelName {
		return true
	}
	id := strings.TrimSpace(gjson.GetBytes(respBody, "id").String())
	if id == "" {
		id = strings.TrimSpace(gjson.GetBytes(respBody, "task_id").String())
	}
	return strings.HasPrefix(id, "sora2-")
}

func enrichTaskResult(taskResult *relaycommon.TaskInfo, respBody []byte) {
	if taskResult == nil || len(respBody) == 0 || !IsResponse(respBody) {
		return
	}
	if taskResult.Url == "" {
		if url := extractManjuVideoURL(respBody); url != "" {
			taskResult.Url = url
		}
	}
	if taskResult.CompletionTokens <= 0 {
		if sec := usageSecondsFromBody(respBody); sec > 0 {
			taskResult.CompletionTokens = sec
			taskResult.TotalTokens = sec
		}
	}
	if taskResult.Status == model.TaskStatusFailure {
		if reason := ExtractFailReason(respBody); reason != "" &&
			(taskResult.Reason == "" || oaivideo.IsGenericTaskFailureReason(taskResult.Reason)) {
			taskResult.Reason = reason
		}
	}
}

func extractVideoURL(res oaivideo.ResponseTask, respBody []byte) string {
	if url := oaivideo.ExtractVideoURL(res); url != "" {
		return url
	}
	if IsResponse(respBody) {
		return extractManjuVideoURL(respBody)
	}
	return ""
}

func extractManjuVideoURL(respBody []byte) string {
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
		if u := oaivideo.PickAbsoluteVideoURL(gjson.GetBytes(respBody, path).String()); u != "" {
			return u
		}
	}
	for _, arrPath := range []string{"raw_data.video_urls", "data.data.video_urls"} {
		arr := gjson.GetBytes(respBody, arrPath)
		if !arr.IsArray() {
			continue
		}
		for _, item := range arr.Array() {
			if u := oaivideo.PickAbsoluteVideoURL(item.String()); u != "" {
				return u
			}
		}
	}
	return ""
}

func usageSecondsFromBody(respBody []byte) int {
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
		if sec := oaivideo.ParsePositiveIntString(gjson.GetBytes(respBody, path).String()); sec > 0 {
			return sec
		}
	}
	if v := gjson.GetBytes(respBody, "usage.seconds").Float(); v > 0 {
		return int(v + 0.5)
	}
	return 0
}

func parseResponseTask(respBody []byte) (oaivideo.ResponseTask, error) {
	var res oaivideo.ResponseTask
	err := common.Unmarshal(respBody, &res)
	if err == nil && (res.ID != "" || res.TaskID != "" || res.Status != "") {
		return res, nil
	}
	if !IsResponse(respBody) {
		return res, err
	}
	return responseTaskFromGJSON(respBody), nil
}

func responseTaskFromGJSON(respBody []byte) oaivideo.ResponseTask {
	res := oaivideo.ResponseTask{
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
			res.Progress = float64(p.Int())
		default:
			if pct := oaivideo.ParsePositiveIntString(strings.TrimSuffix(strings.TrimSpace(p.String()), "%")); pct > 0 {
				res.Progress = float64(pct)
			}
		}
	}
	if res.Object == "" {
		res.Object = "video"
	}
	if url := extractManjuVideoURL(respBody); url != "" {
		res.VideoURL = url
	}
	if sec := usageSecondsFromBody(respBody); sec > 0 {
		res.Seconds = strconv.Itoa(sec)
	}
	return res
}

func buildOpenAIVideoCreateResponse(info *relaycommon.RelayInfo, res oaivideo.ResponseTask, respBody []byte) map[string]any {
	out := map[string]any{
		"id":     info.PublicTaskID,
		"object": "video",
		"model":  info.OriginModelName,
		"status": normalizeStatusForClient(res.Status),
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
	if isFailedStatus(res.Status) {
		if reason := ExtractFailReason(respBody); reason != "" {
			out["error"] = map[string]any{"message": reason}
			out["fail_reason"] = reason
		}
	}
	return out
}

func normalizeStatusForClient(status string) string {
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

// BuildOpenAIErrorResponse 将 Manju 失败 task 转为 OpenAI error JSON。
func BuildOpenAIErrorResponse(respBody []byte) ([]byte, bool) {
	if len(respBody) == 0 || !IsResponse(respBody) {
		return nil, false
	}
	status := strings.TrimSpace(gjson.GetBytes(respBody, "status").String())
	if !isFailedStatus(status) {
		return nil, false
	}
	reason := ExtractFailReason(respBody)
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

// ExtractFailReasonForChat 提取非 task 形态的上游错误。
func ExtractFailReasonForChat(respBody []byte) string {
	if len(respBody) == 0 || IsResponse(respBody) {
		return ""
	}
	return ExtractFailReason(respBody)
}

package shared

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/tidwall/gjson"
)

type ResponseTask struct {
	ID                 string `json:"id"`
	TaskID             string `json:"task_id,omitempty"`
	Object             string `json:"object"`
	Model              string `json:"model"`
	Status             string `json:"status"`
	Progress           int    `json:"progress"`
	CreatedAt          int64  `json:"created_at"`
	CompletedAt        int64  `json:"completed_at,omitempty"`
	ExpiresAt          int64  `json:"expires_at,omitempty"`
	Seconds            string `json:"seconds,omitempty"`
	Size               string `json:"size,omitempty"`
	RemixedFromVideoID string `json:"remixed_from_video_id,omitempty"`
	VideoURL           string `json:"videoUrl,omitempty"`
	VideoURLSnake      string `json:"video_url,omitempty"`
	Data               []struct {
		URL      string `json:"url,omitempty"`
		VideoURL string `json:"video_url,omitempty"`
	} `json:"data,omitempty"`
	Usage *struct {
		Seconds    float64 `json:"seconds"`
		VideoCount int     `json:"video_count"`
	} `json:"usage,omitempty"`
	Error json.RawMessage `json:"error,omitempty"`
}

func ParseResponseTask(respBody []byte) (ResponseTask, error) {
	var res ResponseTask
	err := common.Unmarshal(respBody, &res)
	return res, err
}

func ExtractVideoURL(res ResponseTask) string {
	for _, item := range res.Data {
		if u := PickAbsoluteVideoURL(item.URL, item.VideoURL); u != "" {
			return u
		}
	}
	if u := PickAbsoluteVideoURL(res.VideoURL, res.VideoURLSnake); u != "" {
		return u
	}
	if res.VideoURL != "" {
		return res.VideoURL
	}
	return res.VideoURLSnake
}

func PickAbsoluteVideoURL(candidates ...string) string {
	for _, raw := range candidates {
		u := strings.TrimSpace(raw)
		if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
			return u
		}
	}
	return ""
}

func ExtractErrorMessage(respBody []byte) string {
	var raw map[string]any
	if err := common.Unmarshal(respBody, &raw); err != nil {
		return ""
	}
	errVal, ok := raw["error"]
	if !ok || errVal == nil {
		return ""
	}
	switch v := errVal.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		if msg, ok := v["message"].(string); ok {
			return strings.TrimSpace(msg)
		}
	}
	return ""
}

func ParseErrorField(raw json.RawMessage) (message, code string) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", ""
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strings.TrimSpace(asString), ""
	}
	var asObject struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	if err := json.Unmarshal(raw, &asObject); err == nil {
		return strings.TrimSpace(asObject.Message), strings.TrimSpace(asObject.Code)
	}
	return "", ""
}

func ParsePositiveIntString(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
		return seconds
	}
	if seconds, err := strconv.ParseFloat(raw, 64); err == nil && seconds > 0 {
		return int(seconds + 0.5)
	}
	return 0
}

func UsageSecondsFromResponseTask(res ResponseTask) int {
	if res.Usage != nil && res.Usage.Seconds > 0 {
		return int(res.Usage.Seconds + 0.5)
	}
	return ParsePositiveIntString(res.Seconds)
}

func UsageSecondsFromTaskData(data []byte, manjuFn func([]byte) int) int {
	if len(data) == 0 {
		return 0
	}
	if manjuFn != nil {
		if sec := manjuFn(data); sec > 0 {
			return sec
		}
	}
	var res ResponseTask
	if err := common.Unmarshal(data, &res); err == nil {
		if seconds := UsageSecondsFromResponseTask(res); seconds > 0 {
			return seconds
		}
	}
	if v := gjson.GetBytes(data, "usage.seconds").Float(); v > 0 {
		return int(v + 0.5)
	}
	if raw := strings.TrimSpace(gjson.GetBytes(data, "seconds").String()); raw != "" {
		if seconds := ParsePositiveIntString(raw); seconds > 0 {
			return seconds
		}
	}
	if v := gjson.GetBytes(data, "seconds").Int(); v > 0 {
		return int(v)
	}
	return 0
}

func AsString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int(t)) {
			return strconv.Itoa(int(t))
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case bool:
		return strconv.FormatBool(t)
	default:
		return fmt.Sprint(v)
	}
}

func AsBool(v interface{}) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(strings.TrimSpace(t), "true")
	case float64:
		return t != 0
	case int:
		return t != 0
	default:
		return false
	}
}

func CollectStringList(raw interface{}) []string {
	out := make([]string, 0)
	switch v := raw.(type) {
	case []interface{}:
		for _, item := range v {
			if s := extractRefURL(item); s != "" {
				out = append(out, s)
			}
		}
	case []string:
		for _, s := range v {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
	case string:
		if s := strings.TrimSpace(v); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func extractRefURL(item interface{}) string {
	switch v := item.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]interface{}:
		return strings.TrimSpace(AsString(v["url"]))
	default:
		return ""
	}
}

func IsGenericTaskFailureReason(reason string) bool {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "task failed", "upstream returned empty status", "upstream returned unrecognized message":
		return true
	default:
		return false
	}
}

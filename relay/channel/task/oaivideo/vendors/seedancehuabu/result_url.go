package seedancehuabu

import (
	oaivideo "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/shared"
	"github.com/tidwall/gjson"
)

func extractSd8VideoURL(respBody []byte, res oaivideo.ResponseTask) string {
	if u := oaivideo.PickAbsoluteVideoURL(gjson.GetBytes(respBody, "result_url").String()); u != "" {
		return u
	}
	return oaivideo.ExtractVideoURL(res)
}

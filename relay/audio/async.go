package audio

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	openai "github.com/QuantumNous/new-api/relay/channel/openai"
	"github.com/gin-gonic/gin"
)

// IsAsyncRequest reports whether POST /v1/audio/generations should enqueue a task.
// Music models default to async when the field is omitted; pass async:false for sync.
func IsAsyncRequest(c *gin.Context) bool {
	if c == nil || c.Request == nil || c.Request.Method != "POST" {
		return false
	}
	if !strings.HasSuffix(c.Request.URL.Path, "/audio/generations") {
		return false
	}
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return true
	}
	body, err := storage.Bytes()
	if err != nil || len(body) == 0 {
		return true
	}
	var probe struct {
		Async *bool  `json:"async"`
		Model string `json:"model"`
	}
	if err := common.Unmarshal(body, &probe); err != nil {
		return true
	}
	if probe.Async != nil {
		return *probe.Async
	}
	return openai.IsChatAudioModel(probe.Model)
}

// JobObjectForPath returns the OpenAI job object name for poll responses.
func JobObjectForPath(path string) string {
	return "audio.generation"
}

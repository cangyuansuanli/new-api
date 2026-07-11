package adobe

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	"github.com/gin-gonic/gin"
)

// IsDeprecatedChatRequest identifies the legacy video-over-chat endpoint.
// Endpoint policy belongs to the vendor boundary, while the OpenAI adaptor
// remains responsible only for translating a request that is actually relayed.
func IsDeprecatedChatRequest(c *gin.Context) bool {
	if c == nil || c.Request == nil || c.Request.Method != http.MethodPost {
		return false
	}
	if !strings.HasSuffix(c.Request.URL.Path, "/chat/completions") {
		return false
	}
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return false
	}
	body, err := storage.Bytes()
	if err != nil || len(body) == 0 {
		return false
	}
	var probe struct {
		Model string `json:"model"`
	}
	if err := common.Unmarshal(body, &probe); err != nil {
		return false
	}
	return openai.IsAdobe2APIVideoChatOriginModel(probe.Model)
}

func SetDeprecatedChatHeaders(c *gin.Context) {
	if c == nil {
		return
	}
	c.Header("X-Deprecated-Endpoint", "chat-completions-video")
	c.Header("X-Preferred-Endpoint", "POST /v1/videos")
}

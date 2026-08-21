package image

import (
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/imagevendor"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestYunfeiPatchSyncsOutboundMultipartSize(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = &http.Request{PostForm: url.Values{"size": {"16:9"}}}
	c.Request.MultipartForm = &multipart.Form{Value: map[string][]string{"size": {"16:9"}}}

	info := &relaycommon.RelayInfo{OriginModelName: "cy-yf-gpt-image-2-4k"}
	request := &dto.ImageRequest{Size: "16:9"}
	result, err := imagevendor.ApplyRequestPatch(info, request)
	require.NoError(t, err)
	require.True(t, result.SyncSizeToMultipart)

	syncImageSizeToForm(c, request.Size)
	require.Equal(t, "3840x2160", c.Request.PostForm.Get("size"))
	require.Equal(t, []string{"3840x2160"}, c.Request.MultipartForm.Value["size"])
}

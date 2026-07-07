package openai

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/gin-gonic/gin"
)

func TestConvertManjuBananaMultipartEditsMarshalsUpstreamModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("model", "manju-gemini-banana-pro-4k"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("prompt", "make background blue"); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("image", "ref.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("fakepng")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	imageReq, err := helper.GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
	if err != nil {
		t.Fatalf("parse multipart: %v", err)
	}

	c.Set("model_mapping", `{"manju-gemini-banana-pro-4k":"gemini-3.0-pro-image 4K"}`)
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "manju-gemini-banana-pro-4k")

	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeImagesEdits,
		OriginModelName: "manju-gemini-banana-pro-4k",
		Request:         imageReq,
	}
	info.InitChannelMeta(c)

	request, err := common.DeepCopy(imageReq)
	if err != nil {
		t.Fatalf("deep copy: %v", err)
	}
	if err := helper.ModelMappedHelper(c, info, request); err != nil {
		t.Fatalf("model mapped: %v", err)
	}

	converted, err := ConvertManjuBananaImageRequest(c, info, *request)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	upstreamBody, ok := converted.(map[string]any)
	if !ok {
		t.Fatalf("expected map body, got %T", converted)
	}
	if upstreamBody["model"] != "gemini-3.0-pro-image 4K" {
		t.Fatalf("body model = %q", upstreamBody["model"])
	}
	if upstreamBody["stream"] != false {
		t.Fatalf("stream = %v", upstreamBody["stream"])
	}
	if _, ok := upstreamBody["extra_body"]; ok {
		t.Fatalf("extra_body should not be present: %v", upstreamBody["extra_body"])
	}

	jsonData, err := common.Marshal(converted)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(jsonData, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	modelRaw, ok := payload["model"]
	if !ok || string(modelRaw) == `""` || string(modelRaw) == "null" {
		t.Fatalf("marshaled body missing model: %s", string(jsonData))
	}
	if got := string(modelRaw); got != `"gemini-3.0-pro-image 4K"` {
		t.Fatalf("marshaled model = %s", got)
	}

	messagesRaw, ok := payload["messages"]
	if !ok {
		t.Fatalf("missing messages: %s", string(jsonData))
	}
	var messages []struct {
		Role    string `json:"role"`
		Content []struct {
			Type     string `json:"type"`
			Text     string `json:"text,omitempty"`
			ImageURL *struct {
				URL string `json:"url"`
			} `json:"image_url,omitempty"`
		} `json:"content"`
	}
	if err := json.Unmarshal(messagesRaw, &messages); err != nil {
		t.Fatalf("messages: %v", err)
	}
	if len(messages) != 1 || len(messages[0].Content) < 2 {
		t.Fatalf("messages = %s", string(messagesRaw))
	}
	if messages[0].Content[1].Type != "image_url" || messages[0].Content[1].ImageURL == nil {
		t.Fatalf("expected image_url part, got %+v", messages[0].Content[1])
	}

	// body must stay replayable after conversion path touched multipart
	if _, err := io.ReadAll(c.Request.Body); err != nil {
		t.Fatalf("read body: %v", err)
	}
}

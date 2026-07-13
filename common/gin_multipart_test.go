package common

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

func TestIsMultipartContentTypeWithoutBoundary(t *testing.T) {
	if !IsMultipartContentTypeWithoutBoundary("multipart/form-data") {
		t.Fatal("expected bare multipart Content-Type to be detected")
	}
	if IsMultipartContentTypeWithoutBoundary("multipart/form-data; boundary=abc") {
		t.Fatal("valid multipart Content-Type must not be detected as missing boundary")
	}
	if IsMultipartContentTypeWithoutBoundary("application/json") {
		t.Fatal("JSON Content-Type must not be treated as multipart")
	}
}

func TestMultipartBodyStreamsFromDiskAndSpillsFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldConfig := GetDiskCacheConfig()
	oldMultipartMemoryMB := constant.MultipartMemoryMB
	t.Cleanup(func() {
		SetDiskCacheConfig(oldConfig)
		constant.MultipartMemoryMB = oldMultipartMemoryMB
	})

	SetDiskCacheConfig(DiskCacheConfig{
		Enabled:     true,
		ThresholdMB: 1,
		MaxSizeMB:   32,
		Path:        t.TempDir(),
	})
	constant.MultipartMemoryMB = 1

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("model", "gpt-image-1"); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("image", "large.png")
	if err != nil {
		t.Fatal(err)
	}
	image := bytes.Repeat([]byte("x"), 2<<20)
	if _, err = part.Write(image); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	var request struct {
		Model string `json:"model"`
	}
	if err = UnmarshalBodyReusable(c, &request); err != nil {
		t.Fatal(err)
	}
	if request.Model != "gpt-image-1" {
		t.Fatalf("unexpected model %q", request.Model)
	}
	storage, err := GetBodyStorage(c)
	if err != nil {
		t.Fatal(err)
	}
	if !storage.IsDisk() {
		t.Fatal("large request body should be disk-backed")
	}

	form, err := ParseMultipartFormReusable(c)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = form.RemoveAll() })
	file, err := form.File["image"][0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, ok := file.(*os.File); !ok {
		t.Fatalf("large multipart file should spill to disk, got %T", file)
	}
	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, image) {
		t.Fatal("spilled image content changed")
	}

	// Public model translation mutates the parsed scalar fields. Subsequent
	// validators must reuse that form instead of reparsing the original body.
	c.Request.MultipartForm = form
	form.Value["model"] = []string{"internal-gpt-image-1"}
	request.Model = ""
	if err = UnmarshalBodyReusable(c, &request); err != nil {
		t.Fatal(err)
	}
	if request.Model != "internal-gpt-image-1" {
		t.Fatalf("cached multipart form was not reused, got model %q", request.Model)
	}

	CleanupBodyStorage(c)
}

package shared

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// DurationField identifies the duration key required by an upstream video
// protocol. NewAPI accepts duration and seconds as public aliases, while each
// vendor boundary emits one canonical upstream field.
type DurationField string

const (
	DurationFieldSeconds  DurationField = "seconds"
	DurationFieldDuration DurationField = "duration"
)

// BuildNormalizedRequestBody preserves standard OpenAI video request fields
// and multipart files, replaces model, and translates the public duration
// aliases into the field required by the selected upstream vendor.
func BuildNormalizedRequestBody(c *gin.Context, upstreamModel string, durationField DurationField) (io.Reader, error) {
	contentType := c.GetHeader("Content-Type")
	taskReq, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_normalized_task_request_failed")
	}
	duration := taskReq.RequestedDurationSeconds()
	if durationField != DurationFieldSeconds && durationField != DurationFieldDuration {
		return nil, fmt.Errorf("unsupported upstream duration field %q", durationField)
	}

	if strings.HasPrefix(contentType, "application/json") {
		bodyMap := taskReq.CanonicalVideoBody(upstreamModel)
		delete(bodyMap, string(DurationFieldSeconds))
		delete(bodyMap, string(DurationFieldDuration))
		if duration != 0 {
			bodyMap[string(durationField)] = duration
		}
		newBody, err := common.Marshal(bodyMap)
		if err != nil {
			return nil, errors.Wrap(err, "marshal_video_request_failed")
		}
		return bytes.NewReader(newBody), nil
	}

	if strings.Contains(contentType, "multipart/form-data") {
		formData, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return nil, errors.Wrap(err, "parse_multipart_video_request_failed")
		}
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		canonical := taskReq.CanonicalVideoBody(upstreamModel)
		delete(canonical, "duration")
		for key, value := range canonical {
			if key == "metadata" {
				encoded, marshalErr := common.Marshal(value)
				if marshalErr != nil {
					return nil, errors.Wrap(marshalErr, "marshal_video_metadata_failed")
				}
				writer.WriteField(key, string(encoded))
				continue
			}
			switch typed := value.(type) {
			case string:
				writer.WriteField(key, typed)
			case []string:
				for _, item := range typed {
					writer.WriteField(key, item)
				}
			default:
				writer.WriteField(key, fmt.Sprint(typed))
			}
		}
		if duration != 0 {
			writer.WriteField(string(durationField), fmt.Sprintf("%d", duration))
		}
		for fieldName, fileHeaders := range formData.File {
			for _, fh := range fileHeaders {
				f, err := fh.Open()
				if err != nil {
					continue
				}
				ct := fh.Header.Get("Content-Type")
				if ct == "" || ct == "application/octet-stream" {
					buf512 := make([]byte, 512)
					n, _ := io.ReadFull(f, buf512)
					ct = http.DetectContentType(buf512[:n])
					f.Close()
					f, err = fh.Open()
					if err != nil {
						continue
					}
				}
				h := make(textproto.MIMEHeader)
				h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fh.Filename))
				h.Set("Content-Type", ct)
				part, err := writer.CreatePart(h)
				if err != nil {
					f.Close()
					continue
				}
				io.Copy(part, f)
				f.Close()
			}
		}
		writer.Close()
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		return &buf, nil
	}

	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_request_body_failed")
	}
	return common.ReaderOnly(storage), nil
}

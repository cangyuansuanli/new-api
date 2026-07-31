package seedanceleonardo

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	oaivideo "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/shared"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/defaultvideo"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type TaskAdaptor struct {
	defaultvideo.TaskAdaptor
}

const (
	maxReferenceImages = 4
	maxReferenceVideos = 3
	maxReferenceAudios = 1
)

func (a *TaskAdaptor) GetChannelName() string {
	return "seedance-leonardo"
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if taskErr := relaycommon.ValidateMultipartDirect(c, info); taskErr != nil {
		return taskErr
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if count := len(req.Images); count > maxReferenceImages {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("reference images exceed Leonardo limit (%d/%d)", count, maxReferenceImages),
			"reference_images_limit_exceeded",
			http.StatusBadRequest,
		)
	}
	videoCount, audioCount, err := referenceMediaCounts(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if videoCount > maxReferenceVideos {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("reference videos exceed Leonardo limit (%d/%d)", videoCount, maxReferenceVideos),
			"reference_videos_limit_exceeded",
			http.StatusBadRequest,
		)
	}
	if audioCount > maxReferenceAudios {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("reference audios exceed Leonardo limit (%d/%d)", audioCount, maxReferenceAudios),
			"reference_audios_limit_exceeded",
			http.StatusBadRequest,
		)
	}
	if info.OriginModelName != mini8sModel {
		return nil
	}
	seconds := req.RequestedDurationSeconds()
	if seconds != 0 && (seconds < 4 || seconds > 8) {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("duration must be an integer between 4 and 8 seconds for %s", mini8sModel),
			"invalid_duration",
			http.StatusBadRequest,
		)
	}
	return nil
}

func referenceMediaCounts(c *gin.Context) (int, int, error) {
	contentType := strings.ToLower(c.GetHeader("Content-Type"))
	if strings.HasPrefix(contentType, "application/json") {
		body, err := readJSONBodyMap(c)
		if err != nil {
			return 0, 0, err
		}
		return len(oaivideo.CollectStringList(body[flatKeyReferenceVideos])),
			len(oaivideo.CollectStringList(body[flatKeyReferenceAudios])), nil
	}
	if strings.Contains(contentType, "multipart/form-data") {
		form, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return 0, 0, err
		}
		videoCount, err := countMultipartReferenceValues(form.Value[flatKeyReferenceVideos])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid %s: %w", flatKeyReferenceVideos, err)
		}
		audioCount, err := countMultipartReferenceValues(form.Value[flatKeyReferenceAudios])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid %s: %w", flatKeyReferenceAudios, err)
		}
		return videoCount, audioCount, nil
	}
	return 0, 0, nil
}

func countMultipartReferenceValues(values []string) (int, error) {
	count := 0
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if !strings.HasPrefix(value, "[") && !strings.HasPrefix(value, "{") {
			count++
			continue
		}
		var parsed interface{}
		if err := common.Unmarshal([]byte(value), &parsed); err != nil {
			return 0, err
		}
		count += len(oaivideo.CollectStringList(parsed))
	}
	return count, nil
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	contentType := strings.ToLower(c.Request.Header.Get("Content-Type"))
	if strings.HasPrefix(contentType, "application/json") {
		bodyMap, err := readJSONBodyMap(c)
		if err != nil {
			return nil, err
		}
		req, err := relaycommon.GetTaskRequest(c)
		if err != nil {
			return nil, err
		}
		out := buildUpstreamBody(bodyMap, info.UpstreamModelName, req.RequestedDurationSeconds(), req.Images)
		newBody, err := common.Marshal(out)
		if err != nil {
			return nil, err
		}
		c.Request.Header.Set("Content-Type", "application/json")
		return bytes.NewReader(newBody), nil
	}
	return oaivideo.BuildNormalizedRequestBody(c, info.UpstreamModelName, oaivideo.DurationFieldDuration)
}

func readJSONBodyMap(c *gin.Context) (map[string]interface{}, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_request_body_failed")
	}
	cachedBody, err := storage.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "read_body_bytes_failed")
	}
	var bodyMap map[string]interface{}
	if err := common.Unmarshal(cachedBody, &bodyMap); err != nil {
		return nil, err
	}
	return bodyMap, nil
}

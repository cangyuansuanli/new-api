package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func shouldRunSyncImageViaQueue(c *gin.Context) bool {
	if c == nil || c.Request == nil || !common.GetEnvOrDefaultBool("IMAGE_SYNC_VIA_QUEUE", true) {
		return false
	}
	format := syncImageResponseFormat(c)
	if strings.EqualFold(format, "url") {
		return true
	}
	return format == "" && common.GetEnvOrDefaultBool("IMAGE_SYNC_QUEUE_DEFAULT_RESPONSE_IS_URL", false)
}

// syncImageRequestsB64JSON keeps explicit b64_json requests on the legacy
// synchronous path. URL/default responses can use the durable task pipeline
// without changing the public response shape.
func syncImageRequestsB64JSON(c *gin.Context) bool {
	return strings.EqualFold(syncImageResponseFormat(c), "b64_json")
}

func syncImageResponseFormat(c *gin.Context) string {
	if c.Request.MultipartForm != nil {
		values := c.Request.MultipartForm.Value["response_format"]
		if len(values) > 0 {
			return strings.TrimSpace(values[0])
		}
		return ""
	}
	if c.Request.PostForm != nil {
		if value := strings.TrimSpace(c.Request.PostForm.Get("response_format")); value != "" {
			return value
		}
	}
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return ""
	}
	body, err := storage.Bytes()
	if err != nil || len(body) == 0 {
		return ""
	}
	var request struct {
		ResponseFormat string `json:"response_format"`
	}
	if err := common.Unmarshal(body, &request); err != nil {
		return ""
	}
	return strings.TrimSpace(request.ResponseFormat)
}

func relaySyncImageViaQueue(c *gin.Context) {
	recorder := httptest.NewRecorder()
	submit, _ := gin.CreateTestContext(recorder)
	submit.Request = c.Request
	submit.Params = c.Params
	submit.Keys = make(map[string]any, len(c.Keys))
	for key, value := range c.Keys {
		submit.Keys[key] = value
	}
	submit.Set("image_sync_wait", true)

	RelayImageTaskSubmit(submit)
	if recorder.Code != http.StatusOK {
		c.Data(recorder.Code, recorder.Header().Get("Content-Type"), recorder.Body.Bytes())
		return
	}

	var job dto.OpenAIImageJob
	if err := common.Unmarshal(recorder.Body.Bytes(), &job); err != nil || job.ID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"message": "failed to create internal image task",
			"type":    "server_error",
		}})
		return
	}
	c.Header("X-Cangyuan-Task-Id", job.ID)
	waitForQueuedSyncImage(c, job.ID)
}

func waitForQueuedSyncImage(c *gin.Context, taskID string) {
	timeout := time.Duration(common.GetEnvOrDefault("IMAGE_SYNC_QUEUE_WAIT_SECONDS", 300)) * time.Second
	interval := time.Duration(common.GetEnvOrDefault("IMAGE_SYNC_QUEUE_POLL_INTERVAL_MS", 250)) * time.Millisecond
	if interval < 50*time.Millisecond {
		interval = 50 * time.Millisecond
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	userID := c.GetInt("id")
	for {
		task, exists, err := model.GetByTaskIdForFetch(userID, taskID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
				"message": "failed to read image task status",
				"type":    "server_error",
			}})
			return
		}
		if exists && task != nil {
			switch task.Status {
			case model.TaskStatusSuccess:
				writeQueuedSyncImageResponse(c, task)
				return
			case model.TaskStatusFailure:
				message := strings.TrimSpace(task.FailReason)
				if message == "" {
					message = "image generation failed"
				}
				c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{
					"message": message,
					"type":    "upstream_error",
				}})
				return
			}
		}

		select {
		case <-c.Request.Context().Done():
			return
		case <-timer.C:
			c.Header("Retry-After", "5")
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": gin.H{
				"message": "image task is still running; retry the request later",
				"type":    "timeout_error",
			}})
			return
		case <-ticker.C:
		}
	}
}

func writeQueuedSyncImageResponse(c *gin.Context, task *model.Task) {
	urls := task.PrivateData.ImageResultURLs
	if len(urls) == 0 && strings.TrimSpace(task.PrivateData.ResultURL) != "" {
		urls = []string{task.PrivateData.ResultURL}
	}
	data := make([]dto.ImageData, 0, len(urls))
	for _, resultURL := range urls {
		if strings.TrimSpace(resultURL) != "" {
			data = append(data, dto.ImageData{Url: resultURL})
		}
	}
	if len(data) == 0 {
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{
			"message": "image task completed without a result",
			"type":    "upstream_error",
		}})
		return
	}
	c.JSON(http.StatusOK, dto.ImageResponse{Created: time.Now().Unix(), Data: data})
}

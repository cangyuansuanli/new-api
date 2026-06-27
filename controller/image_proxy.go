package controller

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func imageProxyError(c *gin.Context, status int, errType, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"message": message,
			"type":    errType,
		},
	})
}

func ImageProxy(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		imageProxyError(c, http.StatusBadRequest, "invalid_request_error", "task_id is required")
		return
	}

	userID := c.GetInt("id")
	task, exists, err := model.GetByTaskId(userID, taskID)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to query image task %s: %s", taskID, err.Error()))
		imageProxyError(c, http.StatusInternalServerError, "server_error", "Failed to query task")
		return
	}
	if !exists || task == nil {
		imageProxyError(c, http.StatusNotFound, "invalid_request_error", "Task not found")
		return
	}
	if task.Properties.TaskKind != constant.TaskKindImage {
		imageProxyError(c, http.StatusBadRequest, "invalid_request_error", "Task is not an image task")
		return
	}
	if task.Status != model.TaskStatusSuccess {
		imageProxyError(c, http.StatusBadRequest, "invalid_request_error",
			fmt.Sprintf("Task is not completed yet, current status: %s", task.Status))
		return
	}

	imageURL := task.GetResultURL()
	if imageURL == "" && len(task.PrivateData.ImageResultURLs) > 0 {
		imageURL = task.PrivateData.ImageResultURLs[0]
	}
	if imageURL == "" {
		imageProxyError(c, http.StatusNotFound, "invalid_request_error", "Image URL not available")
		return
	}

	if strings.HasPrefix(imageURL, "data:") {
		data, mime, err := relayDecodeDataURI(imageURL)
		if err != nil {
			imageProxyError(c, http.StatusInternalServerError, "server_error", "Failed to decode image")
			return
		}
		c.Header("Content-Type", mime)
		c.Writer.WriteHeader(http.StatusOK)
		_, _ = c.Writer.Write(data)
		return
	}

	// 结果 URL 来自本系统异步任务落库，非代理接口入参；SSRF 端口白名单会误拦渠道方 URL（如 gulie:3001）。
	if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
		imageProxyError(c, http.StatusBadRequest, "invalid_request_error", "invalid image URL scheme")
		return
	}

	client := service.GetHttpClient()
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, imageURL, nil)
	if err != nil {
		imageProxyError(c, http.StatusInternalServerError, "server_error", "Failed to create request")
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		imageProxyError(c, http.StatusInternalServerError, "server_error", "Failed to fetch image")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		imageProxyError(c, resp.StatusCode, "invalid_request_error", "Failed to fetch image from upstream")
		return
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}
	c.Header("Content-Type", contentType)
	c.Writer.WriteHeader(http.StatusOK)
	_, _ = io.Copy(c.Writer, resp.Body)
}

func relayDecodeDataURI(uri string) ([]byte, string, error) {
	comma := strings.Index(uri, ",")
	if comma < 0 {
		return nil, "", fmt.Errorf("invalid data uri")
	}
	meta := uri[5:comma]
	payload := uri[comma+1:]
	mimeType := "image/png"
	if semi := strings.Index(meta, ";"); semi > 0 {
		mimeType = meta[:semi]
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, "", err
	}
	return data, mimeType, nil
}

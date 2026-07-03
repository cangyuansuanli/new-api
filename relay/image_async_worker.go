package relay

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	"github.com/QuantumNous/new-api/service"
)

var imageAsyncQueue chan string

func StartImageAsyncWorker() {
	maxConcurrent := 32
	if v := strings.TrimSpace(os.Getenv("IMAGE_ASYNC_MAX_CONCURRENT")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxConcurrent = n
		}
	}
	imageAsyncQueue = make(chan string, maxConcurrent*2)
	for i := 0; i < maxConcurrent; i++ {
		go imageAsyncWorkerLoop()
	}
	common.SysLog(fmt.Sprintf("image async worker started, concurrency=%d", maxConcurrent))
	go recoverPendingImageAsyncTasks()
}

func recoverPendingImageAsyncTasks() {
	tasks := model.GetPendingImageAsyncTasks(64)
	if len(tasks) == 0 {
		return
	}
	common.SysLog(fmt.Sprintf("image async recovering %d pending tasks", len(tasks)))
	for _, task := range tasks {
		if task != nil && task.TaskID != "" {
			EnqueueImageAsyncTask(task.TaskID)
		}
	}
}

func EnqueueImageAsyncTask(taskID string) {
	if imageAsyncQueue == nil {
		StartImageAsyncWorker()
	}
	select {
	case imageAsyncQueue <- taskID:
	default:
		common.SysLog("image async queue full, running inline for task " + taskID)
		go processImageAsyncTask(taskID)
	}
}

func imageAsyncWorkerLoop() {
	for taskID := range imageAsyncQueue {
		processImageAsyncTask(taskID)
	}
}

func processImageAsyncTask(taskID string) {
	ctx := context.Background()
	task, exist, err := model.GetByOnlyTaskId(taskID)
	if err != nil || !exist || task == nil {
		return
	}
	if task.Properties.TaskKind != constant.TaskKindImage {
		return
	}
	if task.Status == model.TaskStatusSuccess || task.Status == model.TaskStatusFailure {
		return
	}

	fromStatus := task.Snapshot().Status
	task.Status = model.TaskStatusInProgress
	task.Progress = taskcommon.ProgressInProgress
	if task.StartTime == 0 {
		task.StartTime = time.Now().Unix()
	}
	if !imageAsyncTransitionStatus(ctx, task, fromStatus, "in_progress") {
		return
	}

	images, _, execErr := ExecuteImageTaskUpstream(task)
	if execErr != nil {
		failImageAsyncTask(ctx, task, model.TaskStatusInProgress, execErr.Error())
		return
	}

	resultURLs, resolveErr := resolveTaskImageResultURLs(ctx, task, images)
	if resolveErr != nil {
		failImageAsyncTask(ctx, task, model.TaskStatusInProgress, resolveErr.Error())
		return
	}

	meta := map[string]any{"result_urls": resultURLs}
	task.SetData(meta)
	task.PrivateData.ImageResultURLs = resultURLs
	if len(resultURLs) > 0 {
		task.PrivateData.ResultURL = resultURLs[0]
	}
	task.Status = model.TaskStatusSuccess
	task.Progress = taskcommon.ProgressComplete
	task.FinishTime = time.Now().Unix()
	task.ReleaseRequestSnapshot()
	if !imageAsyncTransitionStatus(ctx, task, model.TaskStatusInProgress, "success") {
		return
	}

	service.RecalculateTaskQuota(ctx, task, task.Quota, "image async complete")
}

func imageAsyncTransitionStatus(ctx context.Context, task *model.Task, fromStatus model.TaskStatus, phase string) bool {
	won, err := task.UpdateWithStatus(fromStatus)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("image async %s CAS error for task %s: %v", phase, task.TaskID, err))
		return false
	}
	if won {
		return true
	}
	logger.LogInfo(ctx, fmt.Sprintf("image async %s CAS lost for task %s (from %s)", phase, task.TaskID, fromStatus))
	return false
}

// resolveTaskImageResultURLs：b64_json / data URI / Gulie·4K 上游 url 均转存 R2 后返回公网 URL。
func resolveTaskImageResultURLs(ctx context.Context, task *model.Task, images []dto.ImageData) ([]string, error) {
	acceptUpstreamURL := service.ImageAsyncAcceptsUpstreamURL(task.Properties.OriginModelName)
	resultURLs := make([]string, 0, len(images))
	for index, item := range images {
		data, mimeOrURL, err := DecodeImageDataItemExported(item)
		if err != nil {
			return nil, err
		}
		if len(data) > 0 {
			mimeType := mimeOrURL
			if !strings.HasPrefix(mimeType, "image/") {
				mimeType = "image/png"
			}
			uploaded, err := service.UploadGeneratedImageBytes(ctx, task.UserId, task.TaskID, index, data, mimeType)
			if err != nil {
				return nil, err
			}
			resultURLs = append(resultURLs, uploaded.PublicURL)
			continue
		}
		if mimeOrURL != "" {
			if acceptUpstreamURL {
				downloadURL := service.RewriteLoopbackUpstreamImageURL(taskUpstreamBaseURL(task), mimeOrURL)
				uploaded, err := service.UploadGeneratedImageFromURL(ctx, task.UserId, task.TaskID, index, downloadURL)
				if err != nil {
					return nil, fmt.Errorf("rehost upstream image url: %w", err)
				}
				resultURLs = append(resultURLs, uploaded.PublicURL)
				continue
			}
			return nil, fmt.Errorf("upstream returned url without b64_json; use response_format=b64_json")
		}
	}
	if len(resultURLs) == 0 {
		return nil, fmt.Errorf("no image results from upstream")
	}
	return resultURLs, nil
}

func failImageAsyncTask(ctx context.Context, task *model.Task, fromStatus model.TaskStatus, reason string) {
	task.Status = model.TaskStatusFailure
	task.Progress = taskcommon.ProgressComplete
	task.FailReason = reason
	task.FinishTime = time.Now().Unix()
	task.ReleaseRequestSnapshot()
	if !imageAsyncTransitionStatus(ctx, task, fromStatus, "failure") {
		if reloaded, exist, err := model.GetByOnlyTaskId(task.TaskID); err == nil && exist {
			if reloaded.Status == model.TaskStatusSuccess {
				return
			}
		}
		return
	}
	service.RefundTaskQuota(ctx, task, reason)
}

func taskUpstreamBaseURL(task *model.Task) string {
	if task == nil || task.ChannelId == 0 {
		return ""
	}
	channel, err := model.GetChannelById(task.ChannelId, true)
	if err != nil || channel == nil {
		return ""
	}
	return channel.GetBaseURL()
}

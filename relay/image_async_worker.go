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

	snap := task.Snapshot()
	task.Status = model.TaskStatusInProgress
	task.Progress = taskcommon.ProgressInProgress
	if task.StartTime == 0 {
		task.StartTime = time.Now().Unix()
	}
	if _, err := task.UpdateWithStatus(snap.Status); err != nil {
		logger.LogError(ctx, "image async update in_progress failed: "+err.Error())
		return
	}

	images, _, execErr := ExecuteImageTaskUpstream(task)
	if execErr != nil {
		failImageAsyncTask(ctx, task, execErr.Error())
		return
	}

	resultURLs, resolveErr := resolveTaskImageResultURLs(ctx, task, images)
	if resolveErr != nil {
		failImageAsyncTask(ctx, task, resolveErr.Error())
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
	if _, err := task.UpdateWithStatus(model.TaskStatusInProgress); err != nil {
		logger.LogError(ctx, "image async mark success failed: "+err.Error())
		return
	}

	service.RecalculateTaskQuota(ctx, task, task.Quota, "image async complete")
}

// resolveTaskImageResultURLs：异步生图结果统一由 b64_json / data URI 解码后上传 R2，对外只暴露 CDN URL。
func resolveTaskImageResultURLs(ctx context.Context, task *model.Task, images []dto.ImageData) ([]string, error) {
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
			return nil, fmt.Errorf("upstream returned url without b64_json; use response_format=b64_json")
		}
	}
	if len(resultURLs) == 0 {
		return nil, fmt.Errorf("no image results from upstream")
	}
	return resultURLs, nil
}

func failImageAsyncTask(ctx context.Context, task *model.Task, reason string) {
	snap := task.Snapshot()
	task.Status = model.TaskStatusFailure
	task.Progress = taskcommon.ProgressComplete
	task.FailReason = reason
	task.FinishTime = time.Now().Unix()
	if _, err := task.UpdateWithStatus(snap.Status); err != nil {
		common.SysLog("image async mark failure failed: " + err.Error())
	}
	service.RefundTaskQuota(ctx, task, reason)
}

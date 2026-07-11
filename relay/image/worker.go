package image

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
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	"github.com/QuantumNous/new-api/service"
)

var imageAsyncQueue chan string

func StartWorker() {
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
			EnqueueTask(task.TaskID)
		}
	}
}

func EnqueueTask(taskID string) {
	if imageAsyncQueue == nil {
		StartWorker()
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
	if !service.TransitionTaskStatus(ctx, task, fromStatus, "image in_progress") {
		return
	}

	images, _, execErr := executeTaskUpstream(task)
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
	if !service.TransitionTaskStatus(ctx, task, model.TaskStatusInProgress, "image success") {
		return
	}

	service.RecalculateTaskQuota(ctx, task, task.Quota, "image async complete")
}

// resolveTaskImageResultURLs：b64_json / data URI / Gulie·4K 上游 url 均转存 R2 后返回公网 URL。
func resolveTaskImageResultURLs(ctx context.Context, task *model.Task, images []dto.ImageData) ([]string, error) {
	return service.RehostTaskImageResultURLs(ctx, task.UserId, task.TaskID, taskUpstreamBaseURL(task), task.Properties.OriginModelName, images)
}

func failImageAsyncTask(ctx context.Context, task *model.Task, fromStatus model.TaskStatus, reason string) {
	task.Status = model.TaskStatusFailure
	task.Progress = taskcommon.ProgressComplete
	task.FailReason = reason
	task.FinishTime = time.Now().Unix()
	task.ReleaseRequestSnapshot()
	if !service.TransitionTaskStatus(ctx, task, fromStatus, "image failure") {
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

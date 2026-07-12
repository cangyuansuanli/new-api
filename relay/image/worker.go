package image

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	"github.com/QuantumNous/new-api/service"
	"github.com/go-redis/redis/v8"
)

const (
	imageTaskNotifyQueue = "new-api:image:task-notify"
	imageTaskNotifyDedup = "new-api:image:task-notify:"
)

type imageWorkerConfig struct {
	concurrency      int
	queueCapacity    int
	dispatchBatch    int
	dispatchInterval time.Duration
	leaseDuration    time.Duration
	maxAttempts      int
}

type imageTaskDispatcher struct {
	once      sync.Once
	queue     chan string
	owner     string
	config    imageWorkerConfig
	mu        sync.Mutex
	queued    map[string]struct{}
	enabled   bool
	active    atomic.Int64
	completed atomic.Int64
	failed    atomic.Int64
}

type WorkerStats struct {
	Enabled       bool   `json:"enabled"`
	Owner         string `json:"owner"`
	Concurrency   int    `json:"concurrency"`
	QueueCapacity int    `json:"queue_capacity"`
	QueueBuffered int    `json:"queue_buffered"`
	Active        int64  `json:"active"`
	Completed     int64  `json:"completed"`
	Failed        int64  `json:"failed"`
	GlobalBacklog int64  `json:"global_backlog"`
	RedisPending  int64  `json:"redis_pending"`
}

func GetWorkerStats() (WorkerStats, error) {
	stats := WorkerStats{
		Enabled:       imageDispatcher.enabled,
		Owner:         imageDispatcher.owner,
		Concurrency:   imageDispatcher.config.concurrency,
		QueueCapacity: imageDispatcher.config.queueCapacity,
		Active:        imageDispatcher.active.Load(),
		Completed:     imageDispatcher.completed.Load(),
		Failed:        imageDispatcher.failed.Load(),
	}
	if imageDispatcher.queue != nil {
		stats.QueueBuffered = len(imageDispatcher.queue)
	}
	global, _, err := model.CountActiveImageTasks(0)
	stats.GlobalBacklog = global
	if common.RedisEnabled && common.RDB != nil {
		stats.RedisPending, _ = common.RDB.LLen(context.Background(), imageTaskNotifyQueue).Result()
	}
	return stats, err
}

var imageDispatcher imageTaskDispatcher
var imageTaskURLPattern = regexp.MustCompile(`https?://[^\s"']+`)

func imageWorkerEnvInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func loadImageWorkerConfig() imageWorkerConfig {
	concurrency := imageWorkerEnvInt("IMAGE_ASYNC_MAX_CONCURRENT", 32)
	return imageWorkerConfig{
		concurrency:      concurrency,
		queueCapacity:    imageWorkerEnvInt("IMAGE_ASYNC_QUEUE_CAPACITY", concurrency*4),
		dispatchBatch:    imageWorkerEnvInt("IMAGE_ASYNC_DISPATCH_BATCH", concurrency*2),
		dispatchInterval: time.Duration(imageWorkerEnvInt("IMAGE_ASYNC_DISPATCH_INTERVAL_MS", 1000)) * time.Millisecond,
		leaseDuration:    time.Duration(imageWorkerEnvInt("IMAGE_ASYNC_LEASE_SECONDS", 180)) * time.Second,
		maxAttempts:      imageWorkerEnvInt("IMAGE_ASYNC_MAX_ATTEMPTS", 3),
	}
}

func imageWorkerOwner() string {
	hostname, _ := os.Hostname()
	parts := []string{strings.TrimSpace(common.NodeName), strings.TrimSpace(hostname), strconv.Itoa(os.Getpid())}
	nonEmpty := parts[:0]
	for _, part := range parts {
		if part != "" {
			nonEmpty = append(nonEmpty, part)
		}
	}
	return strings.Join(nonEmpty, "/")
}

// StartWorker starts a strictly bounded local worker pool. PostgreSQL remains
// the durable queue; every worker node continuously discovers claimable jobs.
func StartWorker() {
	imageDispatcher.once.Do(func() {
		if strings.EqualFold(strings.TrimSpace(os.Getenv("IMAGE_ASYNC_WORKER_ENABLED")), "false") {
			common.SysLog("image async worker disabled on this node")
			return
		}
		config := loadImageWorkerConfig()
		imageDispatcher.config = config
		imageDispatcher.owner = imageWorkerOwner()
		imageDispatcher.queue = make(chan string, config.queueCapacity)
		imageDispatcher.queued = make(map[string]struct{}, config.queueCapacity)
		imageDispatcher.enabled = true
		for i := 0; i < config.concurrency; i++ {
			go imageAsyncWorkerLoop()
		}
		if common.RedisEnabled && common.RDB != nil {
			go imageRedisDispatchLoop()
		}
		go imageAsyncDispatchLoop()
		common.SysLog(fmt.Sprintf(
			"image async worker started, owner=%s concurrency=%d queue_capacity=%d lease=%s",
			imageDispatcher.owner, config.concurrency, config.queueCapacity, config.leaseDuration,
		))
	})
}

// EnqueueTask is only a wake-up hint. If the bounded local buffer is full, the
// task stays QUEUED in PostgreSQL and a dispatcher picks it up later.
func EnqueueTask(taskID string) bool {
	if taskID == "" {
		return false
	}
	if common.RedisEnabled && common.RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		dedupKey := imageTaskNotifyDedup + taskID
		won, err := common.RDB.SetNX(ctx, dedupKey, "1", 30*time.Second).Result()
		if err == nil && !won {
			return true
		}
		if err == nil {
			if err = common.RDB.RPush(ctx, imageTaskNotifyQueue, taskID).Err(); err == nil {
				return true
			}
			_ = common.RDB.Del(ctx, dedupKey).Err()
		}
	}
	return enqueueLocalImageTask(taskID)
}

func enqueueLocalImageTask(taskID string) bool {
	if !imageDispatcher.enabled || imageDispatcher.queue == nil {
		return false
	}
	imageDispatcher.mu.Lock()
	if _, exists := imageDispatcher.queued[taskID]; exists {
		imageDispatcher.mu.Unlock()
		return true
	}
	imageDispatcher.queued[taskID] = struct{}{}
	imageDispatcher.mu.Unlock()

	select {
	case imageDispatcher.queue <- taskID:
		return true
	default:
		imageDispatcher.mu.Lock()
		delete(imageDispatcher.queued, taskID)
		imageDispatcher.mu.Unlock()
		return false
	}
}

func imageRedisDispatchLoop() {
	for {
		result, err := common.RDB.BLPop(context.Background(), 2*time.Second, imageTaskNotifyQueue).Result()
		if err != nil {
			if err != redis.Nil {
				common.SysError("image redis dispatcher: " + err.Error())
				time.Sleep(time.Second)
			}
			continue
		}
		if len(result) != 2 || result[1] == "" {
			continue
		}
		taskID := result[1]
		if !enqueueLocalImageTask(taskID) {
			_ = common.RDB.Del(context.Background(), imageTaskNotifyDedup+taskID).Err()
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func imageAsyncDispatchLoop() {
	ticker := time.NewTicker(imageDispatcher.config.dispatchInterval)
	defer ticker.Stop()
	for {
		dispatchClaimableImageTasks()
		<-ticker.C
	}
}

func dispatchClaimableImageTasks() {
	ids := model.GetClaimableImageAsyncTaskIDs(imageDispatcher.config.dispatchBatch, time.Now().Unix())
	for _, taskID := range ids {
		if !EnqueueTask(taskID) {
			return
		}
	}
}

func imageAsyncWorkerLoop() {
	for taskID := range imageDispatcher.queue {
		processImageAsyncTask(taskID)
		imageDispatcher.mu.Lock()
		delete(imageDispatcher.queued, taskID)
		imageDispatcher.mu.Unlock()
	}
}

func processImageAsyncTask(taskID string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	task, claimed, err := model.ClaimImageAsyncTask(taskID, imageDispatcher.owner, imageDispatcher.config.leaseDuration)
	if err != nil {
		common.SysError(fmt.Sprintf("image async claim failed for %s: %v", taskID, err))
		return
	}
	if !claimed || task == nil {
		return
	}
	if common.RedisEnabled && common.RDB != nil {
		_ = common.RDB.Del(context.Background(), imageTaskNotifyDedup+taskID).Err()
	}
	imageDispatcher.active.Add(1)
	defer imageDispatcher.active.Add(-1)
	if task.Attempt > imageDispatcher.config.maxAttempts {
		failImageAsyncTask(ctx, task, model.TaskStatusInProgress, "image task exceeded maximum attempts")
		return
	}

	heartbeatDone := make(chan struct{})
	go imageAsyncLeaseHeartbeat(task.TaskID, heartbeatDone, cancel)
	defer close(heartbeatDone)

	images, _, execErr := executeTaskUpstream(ctx, task)
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
	task.LeaseOwner = ""
	task.LeaseExpiresAt = 0
	inputObjectKeys := imageTaskInputObjectKeys(task)
	task.ReleaseRequestSnapshot()
	won, err := model.UpdateImageTaskWithLease(task, imageDispatcher.owner)
	if err != nil {
		common.SysError(fmt.Sprintf("image task %s success lease CAS error: %v", task.TaskID, err))
		return
	}
	if !won {
		common.SysLog("image task success lease lost for " + task.TaskID)
		return
	}
	cleanupImageTaskInputs(task.TaskID, inputObjectKeys)
	imageDispatcher.completed.Add(1)

	service.RecalculateTaskQuota(ctx, task, task.Quota, "image async complete")
}

func imageAsyncLeaseHeartbeat(taskID string, done <-chan struct{}, cancel context.CancelFunc) {
	interval := imageDispatcher.config.leaseDuration / 3
	if interval < time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			ok, err := model.RenewImageAsyncTaskLease(taskID, imageDispatcher.owner, imageDispatcher.config.leaseDuration)
			if err != nil {
				common.SysError(fmt.Sprintf("image async lease renewal failed for %s: %v", taskID, err))
				continue
			}
			if !ok {
				common.SysLog("image async lease lost for task " + taskID)
				cancel()
				return
			}
		}
	}
}

// resolveTaskImageResultURLs only publishes first-party R2 URLs. Upstream URLs
// remain process-local and are never stored in public task data.
func resolveTaskImageResultURLs(ctx context.Context, task *model.Task, images []dto.ImageData) ([]string, error) {
	return service.RehostTaskImageResultURLs(ctx, task.UserId, task.TaskID, taskUpstreamBaseURL(task), task.Properties.OriginModelName, images)
}

func failImageAsyncTask(ctx context.Context, task *model.Task, fromStatus model.TaskStatus, reason string) {
	reason = imageTaskURLPattern.ReplaceAllString(reason, "[upstream-url-redacted]")
	task.Status = model.TaskStatusFailure
	task.Progress = taskcommon.ProgressComplete
	task.FailReason = reason
	task.FinishTime = time.Now().Unix()
	task.LeaseOwner = ""
	task.LeaseExpiresAt = 0
	inputObjectKeys := imageTaskInputObjectKeys(task)
	task.ReleaseRequestSnapshot()
	won, err := model.UpdateImageTaskWithLease(task, imageDispatcher.owner)
	if err != nil {
		common.SysError(fmt.Sprintf("image task %s failure lease CAS error: %v", task.TaskID, err))
		return
	}
	if !won {
		if reloaded, exist, err := model.GetByOnlyTaskId(task.TaskID); err == nil && exist {
			if reloaded.Status == model.TaskStatusSuccess {
				return
			}
		}
		return
	}
	cleanupImageTaskInputs(task.TaskID, inputObjectKeys)
	imageDispatcher.failed.Add(1)
	service.RefundTaskQuota(ctx, task, reason)
}

func imageTaskInputObjectKeys(task *model.Task) []string {
	if task == nil || len(task.PrivateData.RequestSnapshot) == 0 || !strings.Contains(task.PrivateData.RequestPath, "/edits") {
		return nil
	}
	return EditSnapshotObjectKeys(task.PrivateData.RequestSnapshot)
}

func cleanupImageTaskInputs(taskID string, objectKeys []string) {
	for _, objectKey := range objectKeys {
		if err := service.DeleteImageTaskInput(context.Background(), objectKey); err != nil {
			common.SysError(fmt.Sprintf("image task input cleanup failed for %s: %v", taskID, err))
		}
	}
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

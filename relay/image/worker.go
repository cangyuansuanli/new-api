package image

import (
	"context"
	"fmt"
	"hash/fnv"
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
	// legacyAdobeTaskNotifyQueue drains stale notifications from the retired
	// adobe lane during rolling upgrades.
	legacyAdobeTaskNotifyQueue = "new-api:image:task-notify:adobe"
	legacyAdobeTaskNotifyDedup = "new-api:image:task-notify:adobe:"
	imageTaskDoneChannel       = "new-api:image:task-done:"
	defaultAdobeChannelIDList  = "75"
)

type imageWorkerConfig struct {
	concurrency    int
	queueCapacity  int
	dispatchBatch  int
	dbScanInterval time.Duration
	leaseDuration  time.Duration
	maxAttempts    int
}

type imageTaskDispatcher struct {
	once      sync.Once
	queue     chan string
	redis     *redis.Client
	owner     string
	config    imageWorkerConfig
	mu        sync.Mutex
	queued    map[string]struct{}
	enabled   bool
	active    atomic.Int64
	completed atomic.Int64
	failed    atomic.Int64
}

type WorkerLaneStats struct {
	Concurrency   int   `json:"concurrency"`
	QueueCapacity int   `json:"queue_capacity"`
	QueueBuffered int   `json:"queue_buffered"`
	Active        int64 `json:"active"`
	Completed     int64 `json:"completed"`
	Failed        int64 `json:"failed"`
	Backlog       int64 `json:"backlog"`
	RedisPending  int64 `json:"redis_pending"`
	DBScanMS      int64 `json:"db_scan_ms"`
}

type WorkerStats struct {
	Enabled       bool                       `json:"enabled"`
	Owner         string                     `json:"owner"`
	Concurrency   int                        `json:"concurrency"`
	QueueCapacity int                        `json:"queue_capacity"`
	QueueBuffered int                        `json:"queue_buffered"`
	Active        int64                      `json:"active"`
	Completed     int64                      `json:"completed"`
	Failed        int64                      `json:"failed"`
	GlobalBacklog int64                      `json:"global_backlog"`
	RedisPending  int64                      `json:"redis_pending"`
	DBScanMS      int64                      `json:"db_scan_ms"`
	Lanes         map[string]WorkerLaneStats `json:"lanes"`
}

func GetWorkerStats() (WorkerStats, error) {
	globalBacklog, _, err := model.CountActiveImageTasks(0)
	if err != nil {
		return WorkerStats{}, err
	}
	laneStats := workerPoolStats(&imageDispatcher, globalBacklog)
	legacyPending := legacyAdobeRedisPending()
	stats := WorkerStats{
		Enabled:       imageDispatcher.enabled,
		Owner:         imageDispatcher.owner,
		Concurrency:   laneStats.Concurrency,
		QueueCapacity: laneStats.QueueCapacity,
		QueueBuffered: laneStats.QueueBuffered,
		Active:        laneStats.Active,
		Completed:     laneStats.Completed,
		Failed:        laneStats.Failed,
		GlobalBacklog: globalBacklog,
		RedisPending:  laneStats.RedisPending + legacyPending,
		DBScanMS:      laneStats.DBScanMS,
		Lanes: map[string]WorkerLaneStats{
			"default": laneStats,
		},
	}
	if legacyPending > 0 {
		stats.Lanes["legacy_adobe"] = WorkerLaneStats{
			RedisPending: legacyPending,
			DBScanMS:     laneStats.DBScanMS,
		}
	}
	return stats, nil
}

var imageDispatcher imageTaskDispatcher
var imageTaskURLPattern = regexp.MustCompile(`https?://[^\s"']+`)

var imageTaskDoneNotifier struct {
	once    sync.Once
	mu      sync.Mutex
	ready   bool
	waiters map[string]map[chan struct{}]struct{}
}

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

func effectiveImageWorkerConcurrency() int {
	concurrency := imageWorkerEnvInt("IMAGE_ASYNC_MAX_CONCURRENT", 32)
	if extra := imageWorkerEnvInt("IMAGE_ASYNC_ADOBE_MAX_CONCURRENT", 0); extra > 0 {
		concurrency += extra
	}
	return concurrency
}

func effectiveImageWorkerQueueCapacity(concurrency int) int {
	if capacity := imageWorkerEnvInt("IMAGE_ASYNC_QUEUE_CAPACITY", 0); capacity > 0 {
		return capacity
	}
	if extra := imageWorkerEnvInt("IMAGE_ASYNC_ADOBE_QUEUE_CAPACITY", 0); extra > 0 {
		return imageWorkerEnvInt("IMAGE_ASYNC_QUEUE_CAPACITY", concurrency*4) + extra
	}
	return concurrency * 4
}

func effectiveImageWorkerDispatchBatch(concurrency int) int {
	if batch := imageWorkerEnvInt("IMAGE_ASYNC_DISPATCH_BATCH", 0); batch > 0 {
		return batch
	}
	if extra := imageWorkerEnvInt("IMAGE_ASYNC_ADOBE_DISPATCH_BATCH", 0); extra > 0 {
		return imageWorkerEnvInt("IMAGE_ASYNC_DISPATCH_BATCH", concurrency*2) + extra
	}
	return concurrency * 2
}

func loadImageWorkerConfig() imageWorkerConfig {
	concurrency := effectiveImageWorkerConcurrency()
	dbScanFallback := 1000
	if common.RedisEnabled && common.RDB != nil {
		dbScanFallback = 15000
	}
	return imageWorkerConfig{
		concurrency:    concurrency,
		queueCapacity:  effectiveImageWorkerQueueCapacity(concurrency),
		dispatchBatch:  effectiveImageWorkerDispatchBatch(concurrency),
		dbScanInterval: time.Duration(imageWorkerEnvInt("IMAGE_ASYNC_DB_SCAN_INTERVAL_MS", dbScanFallback)) * time.Millisecond,
		leaseDuration:  time.Duration(imageWorkerEnvInt("IMAGE_ASYNC_LEASE_SECONDS", 180)) * time.Second,
		maxAttempts:    imageWorkerEnvInt("IMAGE_ASYNC_MAX_ATTEMPTS", 3),
	}
}

// AdobeDirectChannelIDs remains for admission/env compatibility during rollout.
func AdobeDirectChannelIDs() []int {
	raw := strings.TrimSpace(os.Getenv("IMAGE_ASYNC_ADOBE_CHANNEL_IDS"))
	if raw == "" {
		raw = defaultAdobeChannelIDList
	}
	seen := make(map[int]struct{})
	ids := make([]int, 0)
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
		id, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func IsAdobeDirectChannel(channelID int) bool {
	for _, candidate := range AdobeDirectChannelIDs() {
		if channelID == candidate {
			return true
		}
	}
	return false
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

func workerPoolStats(dispatcher *imageTaskDispatcher, backlog int64) WorkerLaneStats {
	stats := WorkerLaneStats{
		Concurrency:   dispatcher.config.concurrency,
		QueueCapacity: dispatcher.config.queueCapacity,
		Active:        dispatcher.active.Load(),
		Completed:     dispatcher.completed.Load(),
		Failed:        dispatcher.failed.Load(),
		Backlog:       backlog,
		DBScanMS:      dispatcher.config.dbScanInterval.Milliseconds(),
	}
	if dispatcher.queue != nil {
		stats.QueueBuffered = len(dispatcher.queue)
	}
	if common.RedisEnabled && common.RDB != nil {
		stats.RedisPending, _ = common.RDB.LLen(context.Background(), imageTaskNotifyQueue).Result()
	}
	return stats
}

func legacyAdobeRedisPending() int64 {
	if !common.RedisEnabled || common.RDB == nil {
		return 0
	}
	pending, _ := common.RDB.LLen(context.Background(), legacyAdobeTaskNotifyQueue).Result()
	return pending
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
		owner := imageWorkerOwner()
		startImageTaskDispatcher(&imageDispatcher, owner, config)
	})
}

func startImageTaskDispatcher(dispatcher *imageTaskDispatcher, owner string, config imageWorkerConfig) {
	dispatcher.owner = owner
	dispatcher.config = config
	dispatcher.queue = make(chan string, config.queueCapacity)
	dispatcher.queued = make(map[string]struct{}, config.queueCapacity)
	if common.RedisEnabled && common.RDB != nil {
		options := *common.RDB.Options()
		if options.PoolSize < config.concurrency+2 {
			options.PoolSize = config.concurrency + 2
		}
		dispatcher.redis = redis.NewClient(&options)
	}
	dispatcher.enabled = true
	for i := 0; i < config.concurrency; i++ {
		go imageAsyncWorkerLoop(dispatcher)
	}
	go imageAsyncDispatchLoop(dispatcher)
	common.SysLog(fmt.Sprintf(
		"image async worker started, owner=%s concurrency=%d queue_capacity=%d db_scan=%s lease=%s",
		dispatcher.owner, config.concurrency, config.queueCapacity,
		config.dbScanInterval, config.leaseDuration,
	))
}

// EnqueueTask is only a wake-up hint. If the bounded local buffer is full, the
// task stays QUEUED in PostgreSQL and a dispatcher picks it up later.
func EnqueueTask(taskID string) bool {
	return enqueueImageTask(taskID)
}

// EnqueueTaskForChannel keeps the API surface but all channels share one pool.
func EnqueueTaskForChannel(taskID string, _ int) bool {
	return EnqueueTask(taskID)
}

func enqueueImageTask(taskID string) bool {
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

func imageAsyncDispatchLoop(dispatcher *imageTaskDispatcher) {
	for {
		leadership, acquired, err := model.TryAcquireImageTaskDispatchLeadership(context.Background())
		if err != nil {
			common.SysError("image async dispatch leader acquire failed: " + err.Error())
		} else if acquired {
			common.SysLog("image async dispatch leader acquired by " + dispatcher.owner)
			err = runImageAsyncDispatchLeader(dispatcher, leadership)
			leadership.Release()
			if err != nil {
				common.SysError("image async dispatch leadership lost: " + err.Error())
			}
		}
		time.Sleep(imageDispatchLeadershipRetryDelay(dispatcher.owner, dispatcher.config.dbScanInterval))
	}
}

func runImageAsyncDispatchLeader(dispatcher *imageTaskDispatcher, leadership *model.ImageTaskDispatchLeadership) error {
	ticker := time.NewTicker(dispatcher.config.dbScanInterval)
	defer ticker.Stop()
	for {
		checkCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := leadership.Check(checkCtx)
		cancel()
		if err != nil {
			return err
		}
		dispatchClaimableImageTasks(dispatcher)
		<-ticker.C
	}
}

func imageDispatchLeadershipRetryDelay(owner string, interval time.Duration) time.Duration {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	window := interval / 5
	if window < time.Second {
		window = time.Second
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(owner))
	return interval + time.Duration(uint64(hash.Sum32())%uint64(window))
}

func dispatchClaimableImageTasks(dispatcher *imageTaskDispatcher) {
	ids := model.GetClaimableImageAsyncTaskIDs(dispatcher.config.dispatchBatch, time.Now().Unix())
	for _, taskID := range ids {
		if !enqueueImageTask(taskID) {
			return
		}
	}
}

func imageAsyncWorkerLoop(dispatcher *imageTaskDispatcher) {
	for {
		taskID, ok := nextImageAsyncTaskID(dispatcher)
		if !ok {
			return
		}
		processImageAsyncTask(dispatcher, taskID)
		dispatcher.mu.Lock()
		delete(dispatcher.queued, taskID)
		dispatcher.mu.Unlock()
	}
}

// nextImageAsyncTaskID lets each idle worker compete for Redis notifications.
// Distribution therefore follows free execution slots instead of assigning an
// equal share to every node regardless of its configured concurrency.
func nextImageAsyncTaskID(dispatcher *imageTaskDispatcher) (string, bool) {
	for dispatcher.redis != nil {
		select {
		case taskID, ok := <-dispatcher.queue:
			return taskID, ok
		default:
		}
		result, err := dispatcher.redis.BLPop(
			context.Background(), 2*time.Second,
			imageTaskNotifyQueue, legacyAdobeTaskNotifyQueue,
		).Result()
		if err == nil && len(result) == 2 && result[1] != "" {
			return result[1], true
		}
		if err != nil && err != redis.Nil {
			common.SysError("image redis worker: " + err.Error())
			time.Sleep(time.Second)
		}
	}
	taskID, ok := <-dispatcher.queue
	return taskID, ok
}

func processImageAsyncTask(dispatcher *imageTaskDispatcher, taskID string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	task, claimed, err := model.ClaimImageAsyncTask(taskID, dispatcher.owner, dispatcher.config.leaseDuration)
	if err != nil {
		common.SysError(fmt.Sprintf("image async claim failed for %s: %v", taskID, err))
		return
	}
	if !claimed || task == nil {
		return
	}
	clearImageTaskNotifyDedup(taskID)
	dispatcher.active.Add(1)
	defer dispatcher.active.Add(-1)
	if task.Attempt > dispatcher.config.maxAttempts {
		failImageAsyncTask(dispatcher, ctx, task, model.TaskStatusInProgress, "image task exceeded maximum attempts")
		return
	}

	heartbeatDone := make(chan struct{})
	go imageAsyncLeaseHeartbeat(dispatcher, task.TaskID, heartbeatDone, cancel)
	defer close(heartbeatDone)

	images, _, execErr := executeTaskUpstream(ctx, task)
	if execErr != nil {
		failImageAsyncTask(dispatcher, ctx, task, model.TaskStatusInProgress, execErr.Error())
		return
	}

	resultURLs, resolveErr := resolveTaskImageResultURLs(ctx, task, images)
	if resolveErr != nil {
		failImageAsyncTask(dispatcher, ctx, task, model.TaskStatusInProgress, resolveErr.Error())
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
	won, err := model.UpdateImageTaskWithLease(task, dispatcher.owner)
	if err != nil {
		common.SysError(fmt.Sprintf("image task %s success lease CAS error: %v", task.TaskID, err))
		return
	}
	if !won {
		common.SysLog("image task success lease lost for " + task.TaskID)
		return
	}
	cleanupImageTaskInputs(task.TaskID, inputObjectKeys)
	dispatcher.completed.Add(1)
	publishImageTaskDone(task.TaskID)

	service.RecalculateTaskQuota(ctx, task, task.Quota, "image async complete")
}

func clearImageTaskNotifyDedup(taskID string) {
	if taskID == "" || !common.RedisEnabled || common.RDB == nil {
		return
	}
	ctx := context.Background()
	_ = common.RDB.Del(ctx, imageTaskNotifyDedup+taskID).Err()
	_ = common.RDB.Del(ctx, legacyAdobeTaskNotifyDedup+taskID).Err()
}

func imageAsyncLeaseHeartbeat(dispatcher *imageTaskDispatcher, taskID string, done <-chan struct{}, cancel context.CancelFunc) {
	interval := dispatcher.config.leaseDuration / 3
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
			ok, err := model.RenewImageAsyncTaskLease(taskID, dispatcher.owner, dispatcher.config.leaseDuration)
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

func failImageAsyncTask(dispatcher *imageTaskDispatcher, ctx context.Context, task *model.Task, fromStatus model.TaskStatus, reason string) {
	reason = imageTaskURLPattern.ReplaceAllString(reason, "[upstream-url-redacted]")
	task.Status = model.TaskStatusFailure
	task.Progress = taskcommon.ProgressComplete
	task.FailReason = reason
	task.FinishTime = time.Now().Unix()
	task.LeaseOwner = ""
	task.LeaseExpiresAt = 0
	inputObjectKeys := imageTaskInputObjectKeys(task)
	task.ReleaseRequestSnapshot()
	won, err := model.UpdateImageTaskWithLease(task, dispatcher.owner)
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
	dispatcher.failed.Add(1)
	publishImageTaskDone(task.TaskID)
	service.RefundTaskQuota(ctx, task, reason)
}

func publishImageTaskDone(taskID string) {
	if taskID == "" || !common.RedisEnabled || common.RDB == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := common.RDB.Publish(ctx, imageTaskDoneChannel+taskID, "1").Err(); err != nil {
		common.SysError(fmt.Sprintf("image task completion notify failed for %s: %v", taskID, err))
	}
}

// SubscribeTaskDone multiplexes all task completion events through one Redis
// pattern subscription per API process, avoiding one Redis connection per
// synchronous waiter.
func SubscribeTaskDone(taskID string) (<-chan struct{}, func()) {
	if taskID == "" || !common.RedisEnabled || common.RDB == nil {
		return nil, func() {}
	}
	imageTaskDoneNotifier.once.Do(startImageTaskDoneNotifier)
	imageTaskDoneNotifier.mu.Lock()
	defer imageTaskDoneNotifier.mu.Unlock()
	if !imageTaskDoneNotifier.ready {
		return nil, func() {}
	}
	if imageTaskDoneNotifier.waiters == nil {
		imageTaskDoneNotifier.waiters = make(map[string]map[chan struct{}]struct{})
	}
	waiter := make(chan struct{}, 1)
	if imageTaskDoneNotifier.waiters[taskID] == nil {
		imageTaskDoneNotifier.waiters[taskID] = make(map[chan struct{}]struct{})
	}
	imageTaskDoneNotifier.waiters[taskID][waiter] = struct{}{}
	return waiter, func() {
		imageTaskDoneNotifier.mu.Lock()
		defer imageTaskDoneNotifier.mu.Unlock()
		delete(imageTaskDoneNotifier.waiters[taskID], waiter)
		if len(imageTaskDoneNotifier.waiters[taskID]) == 0 {
			delete(imageTaskDoneNotifier.waiters, taskID)
		}
	}
}

func startImageTaskDoneNotifier() {
	pubsub := common.RDB.PSubscribe(context.Background(), imageTaskDoneChannel+"*")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	_, err := pubsub.Receive(ctx)
	cancel()
	if err != nil {
		_ = pubsub.Close()
		return
	}
	imageTaskDoneNotifier.mu.Lock()
	imageTaskDoneNotifier.ready = true
	imageTaskDoneNotifier.mu.Unlock()
	go func() {
		for message := range pubsub.Channel() {
			taskID := strings.TrimPrefix(message.Channel, imageTaskDoneChannel)
			imageTaskDoneNotifier.mu.Lock()
			for waiter := range imageTaskDoneNotifier.waiters[taskID] {
				select {
				case waiter <- struct{}{}:
				default:
				}
			}
			imageTaskDoneNotifier.mu.Unlock()
		}
	}()
}

func imageTaskInputObjectKeys(task *model.Task) []string {
	if task == nil || len(task.PrivateData.RequestSnapshot) == 0 {
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

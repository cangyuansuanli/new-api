package audio

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
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	"github.com/QuantumNous/new-api/service"
	"github.com/go-redis/redis/v8"
)

const (
	audioTaskNotifyQueue = "new-api:audio:task-notify"
	audioTaskNotifyDedup = "new-api:audio:task-notify:"
)

type audioWorkerConfig struct {
	concurrency    int
	queueCapacity  int
	dispatchBatch  int
	dbScanInterval time.Duration
	leaseDuration  time.Duration
	maxAttempts    int
}

type audioTaskDispatcher struct {
	once      sync.Once
	queue     chan string
	redis     *redis.Client
	owner     string
	config    audioWorkerConfig
	mu        sync.Mutex
	queued    map[string]struct{}
	enabled   bool
	active    atomic.Int64
	completed atomic.Int64
	failed    atomic.Int64
}

var audioDispatcher audioTaskDispatcher
var audioTaskURLPattern = regexp.MustCompile(`https?://[^\s"']+`)

func audioWorkerEnvInt(name string, fallback int) int {
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

func loadAudioWorkerConfig() audioWorkerConfig {
	concurrency := audioWorkerEnvInt("AUDIO_ASYNC_MAX_CONCURRENT", 8)
	dbScanFallback := 1000
	if common.RedisEnabled && common.RDB != nil {
		dbScanFallback = 15000
	}
	return audioWorkerConfig{
		concurrency:    concurrency,
		queueCapacity:  audioWorkerEnvInt("AUDIO_ASYNC_QUEUE_CAPACITY", concurrency*4),
		dispatchBatch:  audioWorkerEnvInt("AUDIO_ASYNC_DISPATCH_BATCH", concurrency*2),
		dbScanInterval: time.Duration(audioWorkerEnvInt("AUDIO_ASYNC_DB_SCAN_INTERVAL_MS", dbScanFallback)) * time.Millisecond,
		leaseDuration:  time.Duration(audioWorkerEnvInt("AUDIO_ASYNC_LEASE_SECONDS", 180)) * time.Second,
		maxAttempts:    audioWorkerEnvInt("AUDIO_ASYNC_MAX_ATTEMPTS", 3),
	}
}

func audioWorkerOwner() string {
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

// StartWorker starts the bounded audio async worker pool.
func StartWorker() {
	audioDispatcher.once.Do(func() {
		if strings.EqualFold(strings.TrimSpace(os.Getenv("AUDIO_ASYNC_WORKER_ENABLED")), "false") {
			common.SysLog("audio async worker disabled on this node")
			return
		}
		config := loadAudioWorkerConfig()
		owner := audioWorkerOwner()
		startAudioTaskDispatcher(&audioDispatcher, owner, config)
	})
}

func startAudioTaskDispatcher(dispatcher *audioTaskDispatcher, owner string, config audioWorkerConfig) {
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
		go audioAsyncWorkerLoop(dispatcher)
	}
	go audioAsyncDispatchLoop(dispatcher)
	common.SysLog(fmt.Sprintf(
		"audio async worker started, owner=%s concurrency=%d queue_capacity=%d db_scan=%s lease=%s",
		dispatcher.owner, config.concurrency, config.queueCapacity,
		config.dbScanInterval, config.leaseDuration,
	))
}

// EnqueueTask wakes the audio worker pool for one task id.
func EnqueueTask(taskID string) bool {
	return enqueueAudioTask(taskID)
}

func enqueueAudioTask(taskID string) bool {
	if taskID == "" {
		return false
	}
	if common.RedisEnabled && common.RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		dedupKey := audioTaskNotifyDedup + taskID
		won, err := common.RDB.SetNX(ctx, dedupKey, "1", 30*time.Second).Result()
		if err == nil && !won {
			return true
		}
		if err == nil {
			if err = common.RDB.RPush(ctx, audioTaskNotifyQueue, taskID).Err(); err == nil {
				return true
			}
			_ = common.RDB.Del(ctx, dedupKey).Err()
		}
	}
	return enqueueLocalAudioTask(taskID)
}

func enqueueLocalAudioTask(taskID string) bool {
	if !audioDispatcher.enabled || audioDispatcher.queue == nil {
		return false
	}
	audioDispatcher.mu.Lock()
	if _, exists := audioDispatcher.queued[taskID]; exists {
		audioDispatcher.mu.Unlock()
		return true
	}
	audioDispatcher.queued[taskID] = struct{}{}
	audioDispatcher.mu.Unlock()

	select {
	case audioDispatcher.queue <- taskID:
		return true
	default:
		audioDispatcher.mu.Lock()
		delete(audioDispatcher.queued, taskID)
		audioDispatcher.mu.Unlock()
		return false
	}
}

func audioAsyncDispatchLoop(dispatcher *audioTaskDispatcher) {
	for {
		leadership, acquired, err := model.TryAcquireAudioTaskDispatchLeadership(context.Background())
		if err != nil {
			common.SysError("audio async dispatch leader acquire failed: " + err.Error())
		} else if acquired {
			common.SysLog("audio async dispatch leader acquired by " + dispatcher.owner)
			err = runAudioAsyncDispatchLeader(dispatcher, leadership)
			leadership.Release()
			if err != nil {
				common.SysError("audio async dispatch leadership lost: " + err.Error())
			}
		}
		time.Sleep(audioDispatchLeadershipRetryDelay(dispatcher.owner, dispatcher.config.dbScanInterval))
	}
}

func runAudioAsyncDispatchLeader(dispatcher *audioTaskDispatcher, leadership *model.AudioTaskDispatchLeadership) error {
	ticker := time.NewTicker(dispatcher.config.dbScanInterval)
	defer ticker.Stop()
	for {
		checkCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := leadership.Check(checkCtx)
		cancel()
		if err != nil {
			return err
		}
		dispatchClaimableAudioTasks(dispatcher)
		<-ticker.C
	}
}

func audioDispatchLeadershipRetryDelay(owner string, interval time.Duration) time.Duration {
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

func dispatchClaimableAudioTasks(dispatcher *audioTaskDispatcher) {
	ids := model.GetClaimableAudioAsyncTaskIDs(dispatcher.config.dispatchBatch, time.Now().Unix())
	for _, taskID := range ids {
		if !enqueueAudioTask(taskID) {
			return
		}
	}
}

func audioAsyncWorkerLoop(dispatcher *audioTaskDispatcher) {
	for {
		taskID, ok := nextAudioAsyncTaskID(dispatcher)
		if !ok {
			return
		}
		processAudioAsyncTask(dispatcher, taskID)
		dispatcher.mu.Lock()
		delete(dispatcher.queued, taskID)
		dispatcher.mu.Unlock()
	}
}

func nextAudioAsyncTaskID(dispatcher *audioTaskDispatcher) (string, bool) {
	for dispatcher.redis != nil {
		select {
		case taskID, ok := <-dispatcher.queue:
			return taskID, ok
		default:
		}
		result, err := dispatcher.redis.BLPop(context.Background(), 2*time.Second, audioTaskNotifyQueue).Result()
		if err == nil && len(result) == 2 && result[1] != "" {
			return result[1], true
		}
		if err != nil && err != redis.Nil {
			common.SysError("audio redis worker: " + err.Error())
			time.Sleep(time.Second)
		}
	}
	taskID, ok := <-dispatcher.queue
	return taskID, ok
}

func processAudioAsyncTask(dispatcher *audioTaskDispatcher, taskID string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	task, claimed, err := model.ClaimAudioAsyncTask(taskID, dispatcher.owner, dispatcher.config.leaseDuration)
	if err != nil {
		common.SysError(fmt.Sprintf("audio async claim failed for %s: %v", taskID, err))
		return
	}
	if !claimed || task == nil {
		return
	}
	clearAudioTaskNotifyDedup(taskID)
	dispatcher.active.Add(1)
	defer dispatcher.active.Add(-1)
	if task.Attempt > dispatcher.config.maxAttempts {
		failAudioAsyncTask(dispatcher, ctx, task, model.TaskStatusInProgress, "audio task exceeded maximum attempts")
		return
	}

	heartbeatDone := make(chan struct{})
	go audioAsyncLeaseHeartbeat(dispatcher, task.TaskID, heartbeatDone, cancel)
	defer close(heartbeatDone)

	resultURL, execErr := ExecuteTaskUpstream(ctx, task)
	if execErr != nil {
		failAudioAsyncTask(dispatcher, ctx, task, model.TaskStatusInProgress, execErr.Error())
		return
	}

	taskData := []byte(`{}`)
	if len(task.Data) > 0 {
		taskData = task.Data
	}
	rehostedURL, patchedData, rehostErr := service.RehostAudioTaskResult(ctx, task.UserId, task.TaskID, resultURL, taskData, nil)
	if rehostErr != nil {
		failAudioAsyncTask(dispatcher, ctx, task, model.TaskStatusInProgress, fmt.Sprintf("audio result rehost failed: %v", rehostErr))
		return
	}

	task.SetData(patchedData)
	task.PrivateData.ResultURL = rehostedURL
	task.Status = model.TaskStatusSuccess
	task.Progress = taskcommon.ProgressComplete
	task.FinishTime = time.Now().Unix()
	task.LeaseOwner = ""
	task.LeaseExpiresAt = 0
	task.ReleaseRequestSnapshot()
	won, err := model.UpdateAudioTaskWithLease(task, dispatcher.owner)
	if err != nil {
		common.SysError(fmt.Sprintf("audio task %s success lease CAS error: %v", task.TaskID, err))
		return
	}
	if !won {
		common.SysLog("audio task success lease lost for " + task.TaskID)
		return
	}
	dispatcher.completed.Add(1)
	service.RecalculateTaskQuota(ctx, task, task.Quota, "audio async complete")
}

func clearAudioTaskNotifyDedup(taskID string) {
	if taskID == "" || !common.RedisEnabled || common.RDB == nil {
		return
	}
	ctx := context.Background()
	_ = common.RDB.Del(ctx, audioTaskNotifyDedup+taskID).Err()
}

func audioAsyncLeaseHeartbeat(dispatcher *audioTaskDispatcher, taskID string, done <-chan struct{}, cancel context.CancelFunc) {
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
			ok, err := model.RenewAudioAsyncTaskLease(taskID, dispatcher.owner, dispatcher.config.leaseDuration)
			if err != nil {
				common.SysError(fmt.Sprintf("audio async lease renewal failed for %s: %v", taskID, err))
				continue
			}
			if !ok {
				common.SysLog("audio async lease lost for task " + taskID)
				cancel()
				return
			}
		}
	}
}

func failAudioAsyncTask(dispatcher *audioTaskDispatcher, ctx context.Context, task *model.Task, fromStatus model.TaskStatus, reason string) {
	reason = audioTaskURLPattern.ReplaceAllString(reason, "[upstream-url-redacted]")
	task.Status = model.TaskStatusFailure
	task.Progress = taskcommon.ProgressComplete
	task.FailReason = reason
	task.FinishTime = time.Now().Unix()
	task.LeaseOwner = ""
	task.LeaseExpiresAt = 0
	task.ReleaseRequestSnapshot()
	won, err := model.UpdateAudioTaskWithLease(task, dispatcher.owner)
	if err != nil {
		common.SysError(fmt.Sprintf("audio task %s failure lease CAS error: %v", task.TaskID, err))
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
	dispatcher.failed.Add(1)
	service.RefundTaskQuota(ctx, task, reason)
}

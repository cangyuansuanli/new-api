package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	taskCleanupTickInterval = 1 * time.Hour
	taskCleanupBatchSize    = 100
	taskSnapshotStripBatch  = 50
)

var taskCleanupOnce sync.Once

func StartTaskCleanupTask() {
	taskCleanupOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("task cleanup task started: tick=%s", taskCleanupTickInterval))
			ticker := time.NewTicker(taskCleanupTickInterval)
			defer ticker.Stop()

			runTaskCleanupOnce()
			for range ticker.C {
				runTaskCleanupOnce()
			}
		})
	})
}

func runTaskCleanupOnce() {
	ctx := context.Background()
	retentionDays := operation_setting.GetTaskSetting().RetentionDays
	if retentionDays > 0 {
		cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour).Unix()
		total := int64(0)

		for {
			count, err := model.DeleteOldTasks(ctx, cutoff, taskCleanupBatchSize)
			if err != nil {
				logger.LogWarn(ctx, fmt.Sprintf("task cleanup failed: %v", err))
				return
			}
			total += count
			if count < int64(taskCleanupBatchSize) {
				break
			}
		}

		if total > 0 {
			logger.LogInfo(ctx, fmt.Sprintf("task cleanup removed %d finished tasks older than %d days", total, retentionDays))
		}
	}

	stripped, err := model.StripFinishedTaskRequestSnapshots(ctx, taskSnapshotStripBatch)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("task snapshot strip failed: %v", err))
		return
	}
	if stripped > 0 {
		logger.LogInfo(ctx, fmt.Sprintf("task cleanup stripped request_snapshot from %d finished tasks", stripped))
	}
}

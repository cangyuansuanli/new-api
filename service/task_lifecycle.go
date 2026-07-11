package service

import (
	"context"
	"fmt"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
)

// TransitionTaskStatus is the single CAS boundary for tasks stored in the
// shared tasks table. Billing must only run after this function returns true.
func TransitionTaskStatus(ctx context.Context, task *model.Task, fromStatus model.TaskStatus, phase string) bool {
	if task == nil {
		logger.LogError(ctx, fmt.Sprintf("task %s %s: nil task", "", phase))
		return false
	}
	won, err := task.UpdateWithStatus(fromStatus)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("task %s %s CAS error: %v", task.TaskID, phase, err))
		return false
	}
	if !won {
		logger.LogInfo(ctx, fmt.Sprintf("task %s %s CAS lost from %s", task.TaskID, phase, fromStatus))
	}
	return won
}

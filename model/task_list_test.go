package model

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskListSelectLoadsResultURLWithoutSnapshot(t *testing.T) {
	truncateTables(t)

	largeSnapshot := []byte(strings.Repeat("A", 1024*512))
	task := &Task{
		UserId:     1,
		TaskID:     "task_list_select",
		Platform:   "openai",
		Status:     TaskStatusSuccess,
		SubmitTime: time.Now().Unix(),
		PrivateData: TaskPrivateData{
			ResultURL:       "https://example.com/result.png",
			RequestSnapshot: largeSnapshot,
		},
		Data: json.RawMessage(`{"status":"ok"}`),
	}
	insertTask(t, task)

	items := TaskGetAllTasks(0, 10, SyncTaskQueryParams{})
	require.Len(t, items, 1)
	assert.Equal(t, "https://example.com/result.png", items[0].GetResultURL())
	assert.Empty(t, items[0].PrivateData.RequestSnapshot)
	assert.JSONEq(t, `{"status":"ok"}`, string(items[0].Data))
}

func TestTaskListSelectUserQueryOmitsChannelID(t *testing.T) {
	truncateTables(t)

	task := &Task{
		UserId:     42,
		ChannelId:  99,
		TaskID:     "task_user_list",
		Platform:   "openai",
		Status:     TaskStatusSuccess,
		SubmitTime: time.Now().Unix(),
		PrivateData: TaskPrivateData{
			ResultURL: "https://example.com/video.mp4",
		},
		Data: json.RawMessage(`{}`),
	}
	insertTask(t, task)

	items := TaskGetAllUserTask(42, 0, 10, SyncTaskQueryParams{})
	require.Len(t, items, 1)
	assert.Equal(t, 0, items[0].ChannelId)
	assert.Equal(t, "https://example.com/video.mp4", items[0].GetResultURL())
}

func TestDeleteOldTasksRemovesOnlyFinishedTasks(t *testing.T) {
	truncateTables(t)

	now := time.Now().Unix()
	oldSuccess := &Task{
		UserId:     1,
		TaskID:     "task_old_success",
		Platform:   "openai",
		Status:     TaskStatusSuccess,
		SubmitTime: now - 86400*40,
		Data:       json.RawMessage(`{}`),
	}
	oldFailure := &Task{
		UserId:     1,
		TaskID:     "task_old_failure",
		Platform:   "openai",
		Status:     TaskStatusFailure,
		SubmitTime: now - 86400*40,
		Data:       json.RawMessage(`{}`),
	}
	oldRunning := &Task{
		UserId:     1,
		TaskID:     "task_old_running",
		Platform:   "openai",
		Status:     TaskStatusInProgress,
		SubmitTime: now - 86400*40,
		Data:       json.RawMessage(`{}`),
	}
	recentSuccess := &Task{
		UserId:     1,
		TaskID:     "task_recent_success",
		Platform:   "openai",
		Status:     TaskStatusSuccess,
		SubmitTime: now - 86400,
		Data:       json.RawMessage(`{}`),
	}
	for _, task := range []*Task{oldSuccess, oldFailure, oldRunning, recentSuccess} {
		insertTask(t, task)
	}

	cutoff := now - 86400*30
	deleted, err := DeleteOldTasks(context.Background(), cutoff, 100)
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	var remaining []*Task
	require.NoError(t, DB.Order("task_id asc").Find(&remaining).Error)
	require.Len(t, remaining, 2)
	assert.Equal(t, "task_old_running", remaining[0].TaskID)
	assert.Equal(t, "task_recent_success", remaining[1].TaskID)
}

func TestStripFinishedTaskRequestSnapshots(t *testing.T) {
	truncateTables(t)

	largeSnapshot := []byte(strings.Repeat("A", 1024*512))
	task := &Task{
		UserId:     1,
		TaskID:     "task_strip_snapshot",
		Platform:   "image",
		Action:     "imageEdit",
		Status:     TaskStatusSuccess,
		SubmitTime: time.Now().Unix(),
		PrivateData: TaskPrivateData{
			ResultURL:       "https://example.com/result.png",
			RequestSnapshot: largeSnapshot,
		},
		Data: json.RawMessage(`{}`),
	}
	insertTask(t, task)

	stripped, err := StripFinishedTaskRequestSnapshots(context.Background(), 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stripped)

	reloaded, exist, err := GetByOnlyTaskId("task_strip_snapshot")
	require.NoError(t, err)
	require.True(t, exist)
	assert.Empty(t, reloaded.PrivateData.RequestSnapshot)
	assert.Equal(t, "https://example.com/result.png", reloaded.GetResultURL())
}

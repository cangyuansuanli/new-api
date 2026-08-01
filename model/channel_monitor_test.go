package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRecentMediaTasksMatchesPersistedModelBoundaries(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	channelID := 17

	fixtures := []*Task{
		{
			CreatedAt: now, UpdatedAt: now, TaskID: "client-model",
			ChannelId: channelID, Status: TaskStatusSuccess,
			Properties: Properties{ClientModelName: "public-video-model"},
		},
		{
			CreatedAt: now, UpdatedAt: now - 1, TaskID: "origin-model",
			ChannelId: channelID, Status: TaskStatusFailure,
			Properties: Properties{OriginModelName: "public-video-model"},
		},
		{
			CreatedAt: now, UpdatedAt: now - 2, TaskID: "upstream-model",
			ChannelId: channelID, Status: TaskStatusSuccess,
			Properties: Properties{UpstreamModelName: "public-video-model"},
		},
		{
			CreatedAt: now, UpdatedAt: now - 3, TaskID: "other-channel",
			ChannelId: channelID + 1, Status: TaskStatusSuccess,
			Properties: Properties{ClientModelName: "public-video-model"},
		},
		{
			CreatedAt: now, UpdatedAt: now - 4, TaskID: "non-terminal",
			ChannelId: channelID, Status: TaskStatusInProgress,
			Properties: Properties{ClientModelName: "public-video-model"},
		},
		{
			CreatedAt: now - 3600, UpdatedAt: now - 3600, TaskID: "stale",
			ChannelId: channelID, Status: TaskStatusSuccess,
			Properties: Properties{ClientModelName: "public-video-model"},
		},
	}
	for _, task := range fixtures {
		require.NoError(t, DB.Create(task).Error)
	}

	tasks, err := ListRecentMediaTasks(channelID, "public-video-model", now-60, 100)

	require.NoError(t, err)
	require.Len(t, tasks, 3)
	assert.Equal(t, []string{"client-model", "origin-model", "upstream-model"}, []string{
		tasks[0].TaskID,
		tasks[1].TaskID,
		tasks[2].TaskID,
	})
	for _, task := range tasks {
		assert.Empty(t, task.PrivateData)
		assert.Empty(t, task.Data)
	}
}

func TestListRecentMediaTasksRejectsInvalidScope(t *testing.T) {
	truncateTables(t)

	tasks, err := ListRecentMediaTasks(0, "model", 0, 10)
	require.NoError(t, err)
	assert.Empty(t, tasks)

	tasks, err = ListRecentMediaTasks(1, "  ", 0, 10)
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

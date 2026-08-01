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

func TestListRecentChannelsMediaTasksMatchesConfiguredModels(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	channelID := 29
	fixtures := []*Task{
		{
			CreatedAt: now, UpdatedAt: now, TaskID: "client-image",
			ChannelId: channelID, Status: TaskStatusSuccess,
			Properties: Properties{ClientModelName: "image-model"},
		},
		{
			CreatedAt: now, UpdatedAt: now - 1, TaskID: "origin-video",
			ChannelId: channelID, Status: TaskStatusFailure,
			Properties: Properties{OriginModelName: "video-model"},
		},
		{
			CreatedAt: now, UpdatedAt: now - 2, TaskID: "unconfigured",
			ChannelId: channelID, Status: TaskStatusSuccess,
			Properties: Properties{UpstreamModelName: "other-model"},
		},
	}
	for _, task := range fixtures {
		require.NoError(t, DB.Create(task).Error)
	}

	tasks, cursor, err := ListRecentChannelsMediaTasks(map[int][]string{
		channelID:     {"image-model", "video-model"},
		channelID + 1: {"unused-model"},
	}, now-60, 100)

	require.NoError(t, err)
	assert.Equal(t, now, cursor)
	require.Len(t, tasks, 2)
	assert.Equal(t, []string{"client-image", "origin-video"}, []string{tasks[0].TaskID, tasks[1].TaskID})
	for _, task := range tasks {
		assert.Empty(t, task.PrivateData)
		assert.Empty(t, task.Data)
		assert.Empty(t, task.Properties.Input)
	}
}

func TestListRecentChannelsMediaTasksSafelyReadsEscapedUnicode(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	task := &Task{
		CreatedAt: now, UpdatedAt: now, TaskID: "escaped-unicode",
		ChannelId: 31, Status: TaskStatusSuccess,
		Properties: Properties{Input: "prompt\x00suffix", ClientModelName: "image-model"},
	}
	require.NoError(t, DB.Create(task).Error)

	tasks, cursor, err := ListRecentChannelsMediaTasks(map[int][]string{31: {"image-model"}}, now-1, 100)

	require.NoError(t, err)
	assert.Equal(t, now, cursor)
	require.Len(t, tasks, 1)
	assert.Equal(t, "escaped-unicode", tasks[0].TaskID)
	assert.Empty(t, tasks[0].Properties.Input)
}

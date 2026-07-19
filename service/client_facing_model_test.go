package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestClientFacingModelFromTaskPrefersClientModelName(t *testing.T) {
	task := &model.Task{
		Properties: model.Properties{
			ClientModelName: "gemini-banana-pro-4k",
			OriginModelName: "manju-gemini-banana-pro-4k",
		},
	}
	require.Equal(t, "gemini-banana-pro-4k", ClientFacingModelFromTask(task))
}

func TestClientFacingModelFromTaskLegacyFallbackStripsPrefix(t *testing.T) {
	modelPublicRegistryMu.Lock()
	previousRegistry := modelPublicRegistryData
	modelPublicRegistryData.channelPrefixes = []string{"manju-"}
	modelPublicRegistryMu.Unlock()
	t.Cleanup(func() {
		modelPublicRegistryMu.Lock()
		modelPublicRegistryData = previousRegistry
		modelPublicRegistryMu.Unlock()
	})

	task := &model.Task{
		Properties: model.Properties{
			OriginModelName: "manju-gemini-banana-pro-4k",
		},
	}
	require.Equal(t, "gemini-banana-pro-4k", ClientFacingModelFromTask(task))
}

func TestClientFacingModelFromTaskLegacyFallbackEmpty(t *testing.T) {
	require.Equal(t, "", ClientFacingModelFromTask(&model.Task{}))
	require.Equal(t, "", ClientFacingModelFromTask(nil))
}

func TestPatchClientFacingModelJSON(t *testing.T) {
	in := []byte(`{"id":"task_x","model":"gz-seedance-pro-720p-k","status":"queued"}`)
	out, err := PatchClientFacingModelJSON("seedance-pro-720p-k", in)
	require.NoError(t, err)
	require.Contains(t, string(out), `"model":"seedance-pro-720p-k"`)
}

func TestPatchClientFacingModelJSONSkipsWhenNoModelField(t *testing.T) {
	in := []byte(`{"id":"task_x","status":"queued"}`)
	out, err := PatchClientFacingModelJSON("gpt-image-2", in)
	require.NoError(t, err)
	require.Equal(t, string(in), string(out))
}

func TestPatchClientFacingModelJSONFromTask(t *testing.T) {
	task := &model.Task{
		Properties: model.Properties{
			ClientModelName: "seedance-pro-720p-k",
			OriginModelName: "gz-seedance-pro-720p-k",
		},
	}
	in := []byte(`{"id":"task_x","model":"gz-seedance-pro-720p-k","status":"queued"}`)
	out, err := PatchClientFacingModelJSONFromTask(task, in)
	require.NoError(t, err)
	require.Contains(t, string(out), `"model":"seedance-pro-720p-k"`)
}

func TestPatchClientFacingModelJSONPatchesDataArray(t *testing.T) {
	in := []byte(`{"data":[{"id":"m1","model":"cy-gv1-grok-video"},{"id":"m2","model":"other"}]}`)
	out, err := PatchClientFacingModelJSON("grok-video", in)
	require.NoError(t, err)
	require.Contains(t, string(out), `"model":"grok-video"`)
}

func TestPatchClientFacingModelStreamChunk(t *testing.T) {
	out := PatchClientFacingModelStreamChunk("gpt-4", `{"id":"1","model":"internal-name","choices":[]}`)
	require.Contains(t, out, `"model":"gpt-4"`)
	require.Equal(t, "[DONE]", PatchClientFacingModelStreamChunk("gpt-4", "[DONE]"))
}

func TestClientFacingModelFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	SetClientModelNameContext(c, "gemini-banana-pro-4k")
	require.Equal(t, "gemini-banana-pro-4k", ClientFacingModelFromContext(c))
}

func TestPatchClientFacingModelJSONFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	SetClientModelNameContext(c, "public-model")
	in := []byte(`{"model":"internal-model"}`)
	out, err := PatchClientFacingModelJSONFromContext(c, in)
	require.NoError(t, err)
	require.Contains(t, string(out), `"model":"public-model"`)
}

func TestClientFacingTaskPropertiesHidesInternalNames(t *testing.T) {
	task := &model.Task{
		Properties: model.Properties{
			Input:             "prompt",
			OriginModelName:   "cy-img1-gpt-image-2",
			UpstreamModelName: "gpt-image-2-w",
			ClientModelName:   "gpt-image-2",
			TaskKind:          "image",
		},
	}
	props := ClientFacingTaskProperties(task)
	require.NotNil(t, props)
	require.Equal(t, "gpt-image-2", props.ModelName)
	require.Equal(t, "prompt", props.Input)
	require.Equal(t, "image", props.TaskKind)
}

func TestApplyClientFacingModelNamesToLogs(t *testing.T) {
	modelPublicRegistryMu.Lock()
	previousRegistry := modelPublicRegistryData
	modelPublicRegistryData.channelPrefixes = []string{"cy-img1-"}
	modelPublicRegistryMu.Unlock()
	t.Cleanup(func() {
		modelPublicRegistryMu.Lock()
		modelPublicRegistryData = previousRegistry
		modelPublicRegistryMu.Unlock()
	})

	logs := []*model.Log{{ModelName: "cy-img1-gpt-image-2"}}
	ApplyClientFacingModelNamesToLogs(logs)
	require.Equal(t, "gpt-image-2", logs[0].ModelName)
}

func TestPublicQuotaDataForClientAggregatesByPublicName(t *testing.T) {
	modelPublicRegistryMu.Lock()
	previousRegistry := modelPublicRegistryData
	modelPublicRegistryData.internalToPublic = map[string]string{
		"cy-img1-gpt-image-2":          "gpt-image-2",
		"adobe-firefly-gpt-image-2-1k": "gpt-image-2-1k",
	}
	modelPublicRegistryMu.Unlock()
	t.Cleanup(func() {
		modelPublicRegistryMu.Lock()
		modelPublicRegistryData = previousRegistry
		modelPublicRegistryMu.Unlock()
	})

	items := []*model.QuotaData{
		{UserID: 1, Username: "u1", ModelName: "cy-img1-gpt-image-2", CreatedAt: 100, Count: 2, Quota: 20, TokenUsed: 0},
		{UserID: 1, Username: "u1", ModelName: "adobe-firefly-gpt-image-2-1k", CreatedAt: 100, Count: 1, Quota: 10, TokenUsed: 0},
	}
	out := PublicQuotaDataForClient(items)
	require.Len(t, out, 2)
	require.Equal(t, "gpt-image-2", out[0].ModelName)
	require.Equal(t, 2, out[0].Count)
	require.Equal(t, "gpt-image-2-1k", out[1].ModelName)
}

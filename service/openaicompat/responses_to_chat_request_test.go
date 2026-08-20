package openaicompat

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestResponsesRequestToChatCompletionsRequestFullToolTurn(t *testing.T) {
	input := rawForTest(t, []any{
		map[string]any{"role": "user", "content": []any{
			map[string]any{"type": "input_text", "text": "inspect"},
			map[string]any{"type": "input_image", "image_url": "https://example.com/a.png", "detail": "low"},
			map[string]any{"type": "input_file", "file_data": "JVBERi0xLjQK", "filename": "spec.pdf"},
		}},
		map[string]any{"type": "function_call", "call_id": "call_1", "name": "lookup", "arguments": `{"q":"x"}`},
		map[string]any{"type": "function_call_output", "call_id": "call_1", "output": map[string]any{"ok": true}},
	})
	parallel := rawForTest(t, true)
	req := &dto.OpenAIResponsesRequest{
		Model:             "claude-sonnet-4-5",
		Instructions:      rawForTest(t, "be precise"),
		Input:             input,
		ParallelToolCalls: parallel,
		Tools: rawForTest(t, []any{
			map[string]any{"type": "function", "name": "lookup", "description": "lookup data"},
		}),
		ToolChoice: rawForTest(t, map[string]any{"type": "function", "name": "lookup"}),
		Text: rawForTest(t, map[string]any{"format": map[string]any{
			"type": "json_schema", "name": "answer", "schema": map[string]any{"type": "object"},
		}}),
	}

	out, err := ResponsesRequestToChatCompletionsRequest(req)
	require.NoError(t, err)
	require.Len(t, out.Messages, 4)
	require.Equal(t, "system", out.Messages[0].Role)
	parts := out.Messages[1].ParseContent()
	require.Len(t, parts, 3)
	require.Equal(t, "https://example.com/a.png", parts[1].GetImageMedia().Url)
	require.Equal(t, "low", parts[1].GetImageMedia().Detail)
	require.Equal(t, "spec.pdf", parts[2].GetFile().FileName)
	require.Equal(t, "call_1", out.Messages[2].ParseToolCalls()[0].ID)
	require.Equal(t, "call_1", out.Messages[3].ToolCallId)
	require.True(t, *out.ParallelTooCalls)
	require.Len(t, out.Tools, 1)
	params, ok := out.Tools[0].Function.Parameters.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "object", params["type"])
	require.NotNil(t, out.ResponseFormat)
	require.Equal(t, "json_schema", out.ResponseFormat.Type)
}

func TestResponsesRequestToChatCompletionsRequestRejectsUnsupportedStateAndTools(t *testing.T) {
	_, err := ResponsesRequestToChatCompletionsRequest(&dto.OpenAIResponsesRequest{Model: "claude", PreviousResponseID: "resp_old"})
	require.ErrorContains(t, err, "previous_response_id")

	_, err = ResponsesRequestToChatCompletionsRequest(&dto.OpenAIResponsesRequest{Model: "claude", Tools: rawForTest(t, []any{map[string]any{"type": "computer_use"}})})
	require.ErrorContains(t, err, "computer_use")
}

func rawForTest(t *testing.T, value any) []byte {
	t.Helper()
	data, err := common.Marshal(value)
	require.NoError(t, err)
	return data
}

package claude

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestConvertOpenAIResponsesRequestUsesCanonicalClaudePath(t *testing.T) {
	req := dto.OpenAIResponsesRequest{
		Model:           "claude-opus-4-7",
		Input:           claudeRawForTest(t, "hello"),
		MaxOutputTokens: common.GetPointer[uint](2048),
		Reasoning:       &dto.Reasoning{Effort: "high"},
		Tools: claudeRawForTest(t, []any{
			map[string]any{"type": "function", "name": "ping"},
		}),
		Text: claudeRawForTest(t, map[string]any{"format": map[string]any{
			"type": "json_schema", "name": "answer", "schema": map[string]any{"type": "object"},
		}}),
	}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, nil, req)
	require.NoError(t, err)
	claudeReq, ok := converted.(*dto.ClaudeRequest)
	require.True(t, ok)
	require.Equal(t, "adaptive", claudeReq.Thinking.Type)
	require.Equal(t, "summarized", claudeReq.Thinking.Display)
	require.JSONEq(t, `{"effort":"high","format":{"type":"json_schema","schema":{"type":"object"}}}`, string(claudeReq.OutputConfig))

	tools, ok := claudeReq.Tools.([]any)
	require.True(t, ok)
	require.Len(t, tools, 1)
	tool, ok := tools[0].(*dto.Tool)
	require.True(t, ok)
	require.Equal(t, "object", tool.InputSchema["type"])
	require.Equal(t, map[string]any{}, tool.InputSchema["properties"])
}

func TestResponseClaude2OpenAIKeepsTextReasoningAndTools(t *testing.T) {
	first, second, thinking := "first ", "second", "thought"
	resp := ResponseClaude2OpenAI(&dto.ClaudeResponse{
		Id: "msg_1", Model: "claude", StopReason: "tool_use",
		Content: []dto.ClaudeMediaMessage{
			{Type: "thinking", Thinking: &thinking},
			{Type: "text", Text: &first},
			{Type: "tool_use", Id: "call_1", Name: "lookup", Input: map[string]any{"q": "x"}},
			{Type: "text", Text: &second},
		},
	})

	require.Len(t, resp.Choices, 1)
	choice := resp.Choices[0]
	require.Equal(t, "first second", choice.Message.StringContent())
	require.Equal(t, "thought", choice.Message.GetReasoningContent())
	calls := choice.Message.ParseToolCalls()
	require.Len(t, calls, 1)
	require.Equal(t, "call_1", calls[0].ID)
	require.JSONEq(t, `{"q":"x"}`, calls[0].Function.Arguments)
}

func TestRequestOpenAI2ClaudeMessageRejectsInvalidToolArguments(t *testing.T) {
	message := dto.Message{Role: "assistant", Content: ""}
	message.SetToolCalls([]dto.ToolCallRequest{{ID: "call_bad", Type: "function", Function: dto.FunctionRequest{Name: "bad", Arguments: "{"}}})
	_, err := RequestOpenAI2ClaudeMessage(nil, dto.GeneralOpenAIRequest{Model: "claude", Messages: []dto.Message{message}})
	require.ErrorContains(t, err, "call_bad")
}

func TestRequestOpenAI2ClaudeMessagePreservesContentCacheControl(t *testing.T) {
	request := dto.GeneralOpenAIRequest{Model: "claude", Messages: []dto.Message{{
		Role: "user",
		Content: []any{map[string]any{
			"type":          "text",
			"text":          "cached context",
			"cache_control": map[string]any{"type": "ephemeral"},
		}},
	}}}
	converted, err := RequestOpenAI2ClaudeMessage(nil, request)
	require.NoError(t, err)
	content, ok := converted.Messages[0].Content.([]dto.ClaudeMediaMessage)
	require.True(t, ok)
	require.Len(t, content, 1)
	require.JSONEq(t, `{"type":"ephemeral"}`, string(content[0].CacheControl))
}

func claudeRawForTest(t *testing.T, value any) []byte {
	t.Helper()
	data, err := common.Marshal(value)
	require.NoError(t, err)
	return data
}

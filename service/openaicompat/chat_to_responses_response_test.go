package openaicompat

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestChatCompletionsResponseToResponsesResponsePreservesOutputAndUsage(t *testing.T) {
	reasoning := "reason"
	message := dto.Message{Role: "assistant", Content: "answer", ReasoningContent: &reasoning}
	message.SetToolCalls([]dto.ToolCallResponse{
		{ID: "call_1", Type: "function", Function: dto.FunctionResponse{Name: "first", Arguments: `{"a":1}`}},
		{ID: "call_2", Type: "function", Function: dto.FunctionResponse{Name: "second", Arguments: `{}`}},
	})
	resp := &dto.OpenAITextResponse{
		Id: "msg_abc", Model: "claude", Created: int64(123),
		Choices: []dto.OpenAITextResponseChoice{{Message: message, FinishReason: "length"}},
		Usage:   dto.Usage{PromptTokens: 10, CompletionTokens: 4, PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 3, CachedCreationTokens: 2}},
	}

	out, err := ChatCompletionsResponseToResponsesResponse(resp, "")
	require.NoError(t, err)
	require.Equal(t, "resp_abc", out.ID)
	require.JSONEq(t, `"incomplete"`, string(out.Status))
	require.Equal(t, "max_output_tokens", out.IncompleteDetails.Reason)
	require.Len(t, out.Output, 4)
	require.Equal(t, "reasoning", out.Output[0].Type)
	require.Equal(t, "message", out.Output[1].Type)
	require.Equal(t, "function_call", out.Output[2].Type)
	require.Equal(t, 10, out.Usage.InputTokens)
	require.Equal(t, 4, out.Usage.OutputTokens)
	require.Equal(t, 3, out.Usage.InputTokensDetails.CachedTokens)
	require.Equal(t, 2, out.Usage.InputTokensDetails.CachedCreationTokens)
}

func TestChatToResponsesStreamParallelToolsAndIncomplete(t *testing.T) {
	state := NewChatToResponsesStreamState("chatcmpl-stream", "claude")
	index0, index1 := 0, 1
	reasoning, text := "think", "answer"
	length := "length"
	chunks := []*dto.ChatCompletionsStreamResponse{
		{Choices: []dto.ChatCompletionsStreamResponseChoice{{
			Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ReasoningContent: &reasoning},
		}}},
		{Choices: []dto.ChatCompletionsStreamResponseChoice{{
			Delta: dto.ChatCompletionsStreamResponseChoiceDelta{Content: &text},
		}}},
		{Choices: []dto.ChatCompletionsStreamResponseChoice{{
			Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ToolCalls: []dto.ToolCallResponse{
				{Index: &index0, ID: "call_0", Type: "function", Function: dto.FunctionResponse{Name: "a", Arguments: `{"x":`}},
				{Index: &index1, ID: "call_1", Type: "function", Function: dto.FunctionResponse{Name: "b", Arguments: `{}`}},
			}},
		}}},
		{
			Choices: []dto.ChatCompletionsStreamResponseChoice{{
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ToolCalls: []dto.ToolCallResponse{
					{Index: &index0, Function: dto.FunctionResponse{Arguments: `1}`}},
				}},
				FinishReason: &length,
			}},
			Usage: &dto.Usage{PromptTokens: 8, CompletionTokens: 5},
		},
	}

	var eventTypes []string
	for _, chunk := range chunks {
		events, err := ChatCompletionsStreamChunkToResponsesEvents(chunk, state)
		require.NoError(t, err)
		for _, event := range events {
			eventTypes = append(eventTypes, event.Type)
		}
	}
	final := FinalizeChatCompletionsStreamToResponses(state)
	require.NotEmpty(t, final)
	require.Equal(t, "response.incomplete", final[len(final)-1].Type)
	response := final[len(final)-1].Payload.Response
	require.Equal(t, "resp_stream", response.ID)
	require.Len(t, response.Output, 4)
	require.Equal(t, `{"x":1}`, response.Output[2].ArgumentsString())
	require.Equal(t, `{}`, response.Output[3].ArgumentsString())
	require.Equal(t, 8, response.Usage.InputTokens)
	require.Contains(t, eventTypes, "response.reasoning_summary_text.delta")
	require.Contains(t, eventTypes, "response.reasoning_summary_part.added")
	require.Contains(t, eventTypes, "response.output_text.delta")
	require.Contains(t, eventTypes, "response.content_part.added")
	require.Equal(t, 2, countEvent(eventTypes, "response.output_item.added")-2)
}

func TestChatToResponsesTextStreamDoesNotEmitReasoningEvents(t *testing.T) {
	state := NewChatToResponsesStreamState("chatcmpl-text", "claude")
	text := "hello"
	events, err := ChatCompletionsStreamChunkToResponsesEvents(&dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{{Delta: dto.ChatCompletionsStreamResponseChoiceDelta{Content: &text}}},
	}, state)
	require.NoError(t, err)
	require.Equal(t, []string{
		"response.created",
		"response.output_item.added",
		"response.content_part.added",
		"response.output_text.delta",
	}, eventTypeList(events))
	require.Equal(t, 0, *events[1].Payload.OutputIndex)
	require.Equal(t, 0, *events[2].Payload.OutputIndex)
}

func countEvent(events []string, target string) int {
	count := 0
	for _, event := range events {
		if event == target {
			count++
		}
	}
	return count
}

func eventTypeList(events []ChatToResponsesStreamEvent) []string {
	result := make([]string, 0, len(events))
	for _, event := range events {
		result = append(result, event.Type)
	}
	return result
}

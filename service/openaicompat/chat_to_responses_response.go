package openaicompat

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

const (
	responsesEventCreated               = "response.created"
	responsesEventCompleted             = "response.completed"
	responsesEventIncomplete            = "response.incomplete"
	responsesEventOutputItemAdded       = "response.output_item.added"
	responsesEventOutputItemDone        = "response.output_item.done"
	responsesEventContentPartAdded      = "response.content_part.added"
	responsesEventContentPartDone       = "response.content_part.done"
	responsesEventOutputTextDelta       = "response.output_text.delta"
	responsesEventOutputTextDone        = "response.output_text.done"
	responsesEventReasoningSummaryDelta = "response.reasoning_summary_text.delta"
	responsesEventReasoningSummaryDone  = "response.reasoning_summary_text.done"
	responsesEventReasoningPartAdded    = "response.reasoning_summary_part.added"
	responsesEventReasoningPartDone     = "response.reasoning_summary_part.done"
	responsesEventFunctionArgsDelta     = "response.function_call_arguments.delta"
	responsesEventFunctionArgsDone      = "response.function_call_arguments.done"

	responsesOutputMessage      = "message"
	responsesOutputReasoning    = "reasoning"
	responsesOutputFunctionCall = "function_call"
)

// ChatCompletionsResponseToResponsesResponse converts canonical Chat output to
// a Responses API response. Provider-specific adaptors should convert to Chat
// first so Responses semantics stay identical across providers.
func ChatCompletionsResponseToResponsesResponse(resp *dto.OpenAITextResponse, responseID string) (*dto.OpenAIResponsesResponse, error) {
	if resp == nil {
		return nil, errors.New("response is nil")
	}
	if responseID == "" {
		responseID = resp.Id
	}
	responseID = normalizeResponsesID(responseID)
	status, incomplete := "completed", (*dto.IncompleteDetails)(nil)
	if len(resp.Choices) > 0 {
		status, incomplete = responsesStatusFromFinishReason(resp.Choices[0].FinishReason)
	}
	out := &dto.OpenAIResponsesResponse{
		ID:                responseID,
		Object:            "response",
		CreatedAt:         chatCreatedAt(resp.Created),
		Status:            rawJSONString(status),
		IncompleteDetails: incomplete,
		Model:             resp.Model,
		Output:            make([]dto.ResponsesOutput, 0),
		Usage:             ResponsesUsageFromChatUsage(&resp.Usage),
	}
	if len(resp.Choices) == 0 {
		return out, nil
	}

	choice := resp.Choices[0]
	if reasoning := choice.Message.GetReasoningContent(); reasoning != "" {
		out.Output = append(out.Output, reasoningResponsesOutput(responseID+"_reasoning_0", reasoning, status))
	}
	if text := choice.Message.StringContent(); text != "" {
		out.Output = append(out.Output, messageResponsesOutput(responseID+"_msg_0", text, status))
	}
	for index, toolCall := range choice.Message.ParseToolCalls() {
		if toolCall.Type != "" && toolCall.Type != "function" {
			return nil, fmt.Errorf("unsupported chat tool call type %q", toolCall.Type)
		}
		callID := strings.TrimSpace(toolCall.ID)
		if callID == "" {
			callID = fmt.Sprintf("%s_call_%d", responseID, index)
		}
		out.Output = append(out.Output, toolResponsesOutput(callID, toolCall.Function.Name, toolCall.Function.Arguments, status))
	}
	return out, nil
}

// ResponsesUsageFromChatUsage preserves cache and reasoning details while
// exposing the input/output token names expected by the Responses API.
func ResponsesUsageFromChatUsage(src *dto.Usage) *dto.Usage {
	usage := &dto.Usage{}
	if src == nil {
		return usage
	}
	*usage = *src
	usage.InputTokens = src.PromptTokens
	usage.OutputTokens = src.CompletionTokens
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.InputTokens + usage.OutputTokens
	}
	inputDetails := src.PromptTokensDetails
	usage.InputTokensDetails = &inputDetails
	return usage
}

func responsesStatusFromFinishReason(reason string) (string, *dto.IncompleteDetails) {
	switch strings.TrimSpace(reason) {
	case "length":
		return "incomplete", &dto.IncompleteDetails{Reason: "max_output_tokens"}
	case "content_filter":
		return "incomplete", &dto.IncompleteDetails{Reason: "content_filter"}
	default:
		return "completed", nil
	}
}

type ChatToResponsesStreamEvent struct {
	Type    string
	Payload dto.ResponsesStreamResponse
}

type ChatToResponsesStreamState struct {
	ID      string
	Model   string
	Created int64
	Usage   *dto.Usage

	status            string
	incompleteDetails *dto.IncompleteDetails
	sentCreated       bool
	finalized         bool
	nextOutputIndex   int
	text              streamOutput
	reasoning         streamOutput
	tools             map[int]*streamToolOutput
	order             []streamOutputRef
}

type streamOutput struct {
	Index   int
	Started bool
	Done    bool
	Text    strings.Builder
}

type streamToolOutput struct {
	ChatIndex   int
	OutputIndex int
	ID          string
	Name        string
	Arguments   strings.Builder
	Done        bool
}

type streamOutputRef struct {
	Kind      string
	ToolIndex int
}

func NewChatToResponsesStreamState(id, model string) *ChatToResponsesStreamState {
	return &ChatToResponsesStreamState{
		ID:        normalizeResponsesID(id),
		Model:     model,
		Created:   time.Now().Unix(),
		Usage:     &dto.Usage{},
		status:    "completed",
		tools:     make(map[int]*streamToolOutput),
		text:      streamOutput{Index: -1},
		reasoning: streamOutput{Index: -1},
	}
}

func ChatCompletionsStreamChunkToResponsesEvents(chunk *dto.ChatCompletionsStreamResponse, state *ChatToResponsesStreamState) ([]ChatToResponsesStreamEvent, error) {
	if chunk == nil || state == nil || state.finalized {
		return nil, nil
	}
	if state.ID == "" {
		state.ID = normalizeResponsesID(chunk.Id)
	}
	if state.Model == "" {
		state.Model = chunk.Model
	}
	if chunk.Created != 0 {
		state.Created = chunk.Created
	}
	if chunk.Usage != nil {
		state.Usage = ResponsesUsageFromChatUsage(chunk.Usage)
	}

	events := make([]ChatToResponsesStreamEvent, 0)
	if !state.sentCreated {
		state.sentCreated = true
		events = append(events, newResponsesStreamEvent(responsesEventCreated, dto.ResponsesStreamResponse{
			Response: state.response("in_progress", nil, nil),
		}))
	}
	for _, choice := range chunk.Choices {
		if delta := choice.Delta.GetReasoningContent(); delta != "" {
			events = append(events, state.appendReasoning(delta)...)
		}
		if delta := choice.Delta.GetContentString(); delta != "" {
			events = append(events, state.appendText(delta)...)
		}
		for _, call := range choice.Delta.ToolCalls {
			toolEvents, err := state.appendTool(call)
			if err != nil {
				return nil, err
			}
			events = append(events, toolEvents...)
		}
		if choice.FinishReason != nil && *choice.FinishReason != "" {
			state.status, state.incompleteDetails = responsesStatusFromFinishReason(*choice.FinishReason)
			events = append(events, state.finishOutputs()...)
		}
	}
	return events, nil
}

func FinalizeChatCompletionsStreamToResponses(state *ChatToResponsesStreamState) []ChatToResponsesStreamEvent {
	if state == nil || state.finalized {
		return nil
	}
	events := state.finishOutputs()
	state.finalized = true
	eventType := responsesEventCompleted
	if state.status == "incomplete" {
		eventType = responsesEventIncomplete
	}
	events = append(events, newResponsesStreamEvent(eventType, dto.ResponsesStreamResponse{
		Response: state.response(state.status, state.incompleteDetails, state.completedOutputs()),
	}))
	return events
}

func (s *ChatToResponsesStreamState) SetUsage(usage *dto.Usage) {
	if s != nil {
		s.Usage = ResponsesUsageFromChatUsage(usage)
	}
}

func (s *ChatToResponsesStreamState) appendText(delta string) []ChatToResponsesStreamEvent {
	events := make([]ChatToResponsesStreamEvent, 0, 2)
	if !s.text.Started {
		s.text.Started = true
		s.text.Index = s.nextIndex("message", -1)
		events = append(events, newResponsesStreamEvent(responsesEventOutputItemAdded, dto.ResponsesStreamResponse{
			OutputIndex: intPointer(s.text.Index), ItemID: s.messageID(),
			Item: &dto.ResponsesOutput{Type: responsesOutputMessage, ID: s.messageID(), Status: "in_progress", Role: "assistant", Content: []dto.ResponsesOutputContent{}},
		}))
		events = append(events, newResponsesStreamEvent(responsesEventContentPartAdded, dto.ResponsesStreamResponse{
			OutputIndex: intPointer(s.text.Index), ContentIndex: intPointer(0), ItemID: s.messageID(),
			Part: &dto.ResponsesReasoningSummaryPart{Type: "output_text", Text: "", Annotations: []interface{}{}},
		}))
	}
	s.text.Text.WriteString(delta)
	events = append(events, newResponsesStreamEvent(responsesEventOutputTextDelta, dto.ResponsesStreamResponse{
		OutputIndex: intPointer(s.text.Index), ContentIndex: intPointer(0), ItemID: s.messageID(), Delta: delta,
	}))
	return events
}

func (s *ChatToResponsesStreamState) appendReasoning(delta string) []ChatToResponsesStreamEvent {
	events := make([]ChatToResponsesStreamEvent, 0, 2)
	if !s.reasoning.Started {
		s.reasoning.Started = true
		s.reasoning.Index = s.nextIndex("reasoning", -1)
		events = append(events, newResponsesStreamEvent(responsesEventOutputItemAdded, dto.ResponsesStreamResponse{
			OutputIndex: intPointer(s.reasoning.Index), ItemID: s.reasoningID(),
			Item: &dto.ResponsesOutput{Type: responsesOutputReasoning, ID: s.reasoningID(), Status: "in_progress", Summary: []dto.ResponsesReasoningSummaryPart{}},
		}))
		events = append(events, newResponsesStreamEvent(responsesEventReasoningPartAdded, dto.ResponsesStreamResponse{
			OutputIndex: intPointer(s.reasoning.Index), SummaryIndex: intPointer(0), ItemID: s.reasoningID(),
			Part: &dto.ResponsesReasoningSummaryPart{Type: "summary_text", Text: ""},
		}))
	}
	s.reasoning.Text.WriteString(delta)
	events = append(events, newResponsesStreamEvent(responsesEventReasoningSummaryDelta, dto.ResponsesStreamResponse{
		OutputIndex: intPointer(s.reasoning.Index), SummaryIndex: intPointer(0), ItemID: s.reasoningID(), Delta: delta,
	}))
	return events
}

func (s *ChatToResponsesStreamState) appendTool(call dto.ToolCallResponse) ([]ChatToResponsesStreamEvent, error) {
	if call.Type != nil && common.Interface2String(call.Type) != "" && common.Interface2String(call.Type) != "function" {
		return nil, fmt.Errorf("unsupported chat tool call type %q", common.Interface2String(call.Type))
	}
	chatIndex := 0
	if call.Index != nil {
		chatIndex = *call.Index
	}
	tool := s.tools[chatIndex]
	events := make([]ChatToResponsesStreamEvent, 0, 2)
	if tool == nil {
		tool = &streamToolOutput{ChatIndex: chatIndex, OutputIndex: s.nextIndex("tool", chatIndex), ID: strings.TrimSpace(call.ID), Name: strings.TrimSpace(call.Function.Name)}
		if tool.ID == "" {
			tool.ID = fmt.Sprintf("%s_call_%d", s.ID, chatIndex)
		}
		s.tools[chatIndex] = tool
		events = append(events, newResponsesStreamEvent(responsesEventOutputItemAdded, dto.ResponsesStreamResponse{
			OutputIndex: intPointer(tool.OutputIndex), ItemID: tool.ID,
			Item: &dto.ResponsesOutput{Type: responsesOutputFunctionCall, ID: tool.ID, Status: "in_progress", CallId: tool.ID, Name: tool.Name, Arguments: rawJSONString("")},
		}))
	}
	if name := strings.TrimSpace(call.Function.Name); name != "" {
		tool.Name = name
	}
	if call.Function.Arguments != "" {
		tool.Arguments.WriteString(call.Function.Arguments)
		events = append(events, newResponsesStreamEvent(responsesEventFunctionArgsDelta, dto.ResponsesStreamResponse{
			OutputIndex: intPointer(tool.OutputIndex), ItemID: tool.ID, Delta: call.Function.Arguments,
		}))
	}
	return events, nil
}

func (s *ChatToResponsesStreamState) finishOutputs() []ChatToResponsesStreamEvent {
	events := make([]ChatToResponsesStreamEvent, 0)
	status := s.outputStatus()
	if s.reasoning.Started && !s.reasoning.Done {
		s.reasoning.Done = true
		events = append(events,
			newResponsesStreamEvent(responsesEventReasoningSummaryDone, dto.ResponsesStreamResponse{OutputIndex: intPointer(s.reasoning.Index), SummaryIndex: intPointer(0), ItemID: s.reasoningID(), Text: s.reasoning.Text.String(), Part: &dto.ResponsesReasoningSummaryPart{Type: "summary_text", Text: s.reasoning.Text.String()}}),
			newResponsesStreamEvent(responsesEventReasoningPartDone, dto.ResponsesStreamResponse{OutputIndex: intPointer(s.reasoning.Index), SummaryIndex: intPointer(0), ItemID: s.reasoningID(), Part: &dto.ResponsesReasoningSummaryPart{Type: "summary_text", Text: s.reasoning.Text.String()}}),
			newResponsesStreamEvent(responsesEventOutputItemDone, dto.ResponsesStreamResponse{OutputIndex: intPointer(s.reasoning.Index), Item: pointerOutput(reasoningResponsesOutput(s.reasoningID(), s.reasoning.Text.String(), status))}),
		)
	}
	if s.text.Started && !s.text.Done {
		s.text.Done = true
		events = append(events,
			newResponsesStreamEvent(responsesEventOutputTextDone, dto.ResponsesStreamResponse{OutputIndex: intPointer(s.text.Index), ContentIndex: intPointer(0), ItemID: s.messageID(), Text: s.text.Text.String()}),
			newResponsesStreamEvent(responsesEventContentPartDone, dto.ResponsesStreamResponse{OutputIndex: intPointer(s.text.Index), ContentIndex: intPointer(0), ItemID: s.messageID(), Part: &dto.ResponsesReasoningSummaryPart{Type: "output_text", Text: s.text.Text.String(), Annotations: []interface{}{}}}),
			newResponsesStreamEvent(responsesEventOutputItemDone, dto.ResponsesStreamResponse{OutputIndex: intPointer(s.text.Index), Item: pointerOutput(messageResponsesOutput(s.messageID(), s.text.Text.String(), status))}),
		)
	}
	indexes := make([]int, 0, len(s.tools))
	for index := range s.tools {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	for _, index := range indexes {
		tool := s.tools[index]
		if tool.Done {
			continue
		}
		tool.Done = true
		events = append(events,
			newResponsesStreamEvent(responsesEventFunctionArgsDone, dto.ResponsesStreamResponse{OutputIndex: intPointer(tool.OutputIndex), ItemID: tool.ID, Arguments: tool.Arguments.String()}),
			newResponsesStreamEvent(responsesEventOutputItemDone, dto.ResponsesStreamResponse{OutputIndex: intPointer(tool.OutputIndex), Item: pointerOutput(toolResponsesOutput(tool.ID, tool.Name, tool.Arguments.String(), status))}),
		)
	}
	return events
}

func (s *ChatToResponsesStreamState) completedOutputs() []dto.ResponsesOutput {
	outputs := make([]dto.ResponsesOutput, 0, len(s.order))
	for _, ref := range s.order {
		switch ref.Kind {
		case "message":
			outputs = append(outputs, messageResponsesOutput(s.messageID(), s.text.Text.String(), s.outputStatus()))
		case "reasoning":
			outputs = append(outputs, reasoningResponsesOutput(s.reasoningID(), s.reasoning.Text.String(), s.outputStatus()))
		case "tool":
			tool := s.tools[ref.ToolIndex]
			outputs = append(outputs, toolResponsesOutput(tool.ID, tool.Name, tool.Arguments.String(), s.outputStatus()))
		}
	}
	return outputs
}

func (s *ChatToResponsesStreamState) response(status string, incomplete *dto.IncompleteDetails, output []dto.ResponsesOutput) *dto.OpenAIResponsesResponse {
	if output == nil {
		output = []dto.ResponsesOutput{}
	}
	var usage *dto.Usage
	if status != "in_progress" {
		usage = s.Usage
	}
	return &dto.OpenAIResponsesResponse{ID: s.ID, Object: "response", CreatedAt: int(s.Created), Status: rawJSONString(status), IncompleteDetails: incomplete, Model: s.Model, Output: output, Usage: usage}
}

func (s *ChatToResponsesStreamState) nextIndex(kind string, toolIndex int) int {
	index := s.nextOutputIndex
	s.nextOutputIndex++
	s.order = append(s.order, streamOutputRef{Kind: kind, ToolIndex: toolIndex})
	return index
}

func (s *ChatToResponsesStreamState) outputStatus() string {
	if s.status == "incomplete" {
		return "incomplete"
	}
	return "completed"
}

func (s *ChatToResponsesStreamState) messageID() string   { return s.ID + "_msg_0" }
func (s *ChatToResponsesStreamState) reasoningID() string { return s.ID + "_reasoning_0" }

func messageResponsesOutput(id, text, status string) dto.ResponsesOutput {
	return dto.ResponsesOutput{Type: responsesOutputMessage, ID: id, Status: status, Role: "assistant", Content: []dto.ResponsesOutputContent{{Type: "output_text", Text: text, Annotations: []interface{}{}}}}
}

func reasoningResponsesOutput(id, text, status string) dto.ResponsesOutput {
	return dto.ResponsesOutput{Type: responsesOutputReasoning, ID: id, Status: status, Summary: []dto.ResponsesReasoningSummaryPart{{Type: "summary_text", Text: text}}}
}

func toolResponsesOutput(id, name, arguments, status string) dto.ResponsesOutput {
	return dto.ResponsesOutput{Type: responsesOutputFunctionCall, ID: id, Status: status, CallId: id, Name: name, Arguments: rawJSONString(arguments)}
}

func newResponsesStreamEvent(eventType string, payload dto.ResponsesStreamResponse) ChatToResponsesStreamEvent {
	payload.Type = eventType
	return ChatToResponsesStreamEvent{Type: eventType, Payload: payload}
}

func rawJSONString(value string) []byte {
	data, err := common.Marshal(value)
	if err != nil {
		return []byte(`""`)
	}
	return data
}

func chatCreatedAt(created any) int {
	switch value := created.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case string:
		parsed, _ := strconv.Atoi(value)
		if parsed != 0 {
			return parsed
		}
	}
	return int(time.Now().Unix())
}

func intPointer(value int) *int                                    { return &value }
func pointerOutput(value dto.ResponsesOutput) *dto.ResponsesOutput { return &value }

func normalizeResponsesID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" || strings.HasPrefix(id, "resp_") {
		return id
	}
	for _, prefix := range []string{"chatcmpl-", "msg_"} {
		id = strings.TrimPrefix(id, prefix)
	}
	return "resp_" + id
}

package openaicompat

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

const (
	responsesInputFunctionCall       = "function_call"
	responsesInputFunctionCallOutput = "function_call_output"
)

// ResponsesRequestToChatCompletionsRequest normalizes the stateless Responses
// subset into the gateway's canonical chat request. Provider adaptors can then
// reuse one media, tool, reasoning, and structured-output conversion path.
func ResponsesRequestToChatCompletionsRequest(req *dto.OpenAIResponsesRequest) (*dto.GeneralOpenAIRequest, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if req.Model == "" {
		return nil, errors.New("model is required")
	}
	if err := validateStatelessResponsesRequest(req); err != nil {
		return nil, err
	}

	messages, err := responsesMessagesToChat(req)
	if err != nil {
		return nil, err
	}
	tools, webSearch, err := responsesToolsToChat(req.Tools)
	if err != nil {
		return nil, err
	}
	toolChoice, err := responsesToolChoiceToChat(req.ToolChoice)
	if err != nil {
		return nil, err
	}
	responseFormat, err := responsesTextToChatResponseFormat(req.Text)
	if err != nil {
		return nil, err
	}

	out := &dto.GeneralOpenAIRequest{
		Model:               req.Model,
		Messages:            messages,
		Stream:              req.Stream,
		StreamOptions:       req.StreamOptions,
		MaxCompletionTokens: req.MaxOutputTokens,
		Temperature:         req.Temperature,
		TopP:                req.TopP,
		TopLogProbs:         req.TopLogProbs,
		ResponseFormat:      responseFormat,
		Tools:               tools,
		ToolChoice:          toolChoice,
		User:                req.User,
		Store:               req.Store,
		Metadata:            req.Metadata,
		SafetyIdentifier:    req.SafetyIdentifier,
		WebSearchOptions:    webSearch,
	}
	if req.Reasoning != nil {
		out.ReasoningEffort = req.Reasoning.Effort
	}
	if len(req.ParallelToolCalls) > 0 && common.GetJsonType(req.ParallelToolCalls) == "boolean" {
		var parallel bool
		if err := common.Unmarshal(req.ParallelToolCalls, &parallel); err != nil {
			return nil, fmt.Errorf("invalid parallel_tool_calls: %w", err)
		}
		out.ParallelTooCalls = &parallel
	} else if len(req.ParallelToolCalls) > 0 && string(req.ParallelToolCalls) != "null" {
		return nil, errors.New("parallel_tool_calls must be a boolean")
	}
	if len(req.PromptCacheKey) > 0 {
		if common.GetJsonType(req.PromptCacheKey) != "string" {
			return nil, errors.New("prompt_cache_key must be a string")
		}
		if err := common.Unmarshal(req.PromptCacheKey, &out.PromptCacheKey); err != nil {
			return nil, fmt.Errorf("invalid prompt_cache_key: %w", err)
		}
	}
	return out, nil
}

func validateStatelessResponsesRequest(req *dto.OpenAIResponsesRequest) error {
	unsupported := make([]string, 0, 4)
	if len(req.Conversation) > 0 && string(req.Conversation) != "null" {
		unsupported = append(unsupported, "conversation")
	}
	if req.PreviousResponseID != "" {
		unsupported = append(unsupported, "previous_response_id")
	}
	if len(req.Prompt) > 0 && string(req.Prompt) != "null" {
		unsupported = append(unsupported, "prompt")
	}
	if len(req.ContextManagement) > 0 && string(req.ContextManagement) != "null" {
		unsupported = append(unsupported, "context_management")
	}
	if len(unsupported) > 0 {
		return fmt.Errorf("responses to messages conversion does not support stateful fields: %s", strings.Join(unsupported, ", "))
	}
	return nil
}

func responsesMessagesToChat(req *dto.OpenAIResponsesRequest) ([]dto.Message, error) {
	messages := make([]dto.Message, 0)
	if len(req.Instructions) > 0 {
		var instructions string
		if err := common.Unmarshal(req.Instructions, &instructions); err != nil {
			return nil, fmt.Errorf("invalid instructions: %w", err)
		}
		if strings.TrimSpace(instructions) != "" {
			messages = append(messages, dto.Message{Role: "system", Content: instructions})
		}
	}
	if len(req.Input) == 0 {
		return messages, nil
	}
	if common.GetJsonType(req.Input) == "string" {
		var input string
		if err := common.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
		return append(messages, dto.Message{Role: "user", Content: input}), nil
	}
	if common.GetJsonType(req.Input) != "array" {
		return nil, fmt.Errorf("unsupported responses input type %q", common.GetJsonType(req.Input))
	}

	var items []map[string]any
	if err := common.Unmarshal(req.Input, &items); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	for _, item := range items {
		switch itemType := common.Interface2String(item["type"]); itemType {
		case responsesInputFunctionCall:
			call, err := responsesFunctionCallToChat(item)
			if err != nil {
				return nil, err
			}
			messages = appendToolCallToAssistant(messages, call)
		case responsesInputFunctionCallOutput:
			messages = append(messages, dto.Message{
				Role:       "tool",
				ToolCallId: responsesCallID(item),
				Content:    stringifyResponsesValue(item["output"]),
			})
		case "", "message":
			role := strings.TrimSpace(common.Interface2String(item["role"]))
			if role == "" {
				role = "user"
			}
			content, err := responsesContentToChat(item["content"])
			if err != nil {
				return nil, err
			}
			messages = append(messages, dto.Message{Role: role, Content: content})
		default:
			return nil, fmt.Errorf("unsupported responses input item type %q", itemType)
		}
	}
	return messages, nil
}

func responsesContentToChat(value any) (any, error) {
	if value == nil {
		return "", nil
	}
	if text, ok := value.(string); ok {
		return text, nil
	}
	parts, ok := value.([]any)
	if !ok {
		return value, nil
	}
	chatParts := make([]any, 0, len(parts))
	textOnly := true
	var text strings.Builder
	for _, raw := range parts {
		part, ok := raw.(map[string]any)
		if !ok {
			textOnly = false
			chatParts = append(chatParts, raw)
			continue
		}
		switch common.Interface2String(part["type"]) {
		case "input_text", "output_text", "text":
			value := common.Interface2String(part["text"])
			text.WriteString(value)
			chatPart := map[string]any{"type": dto.ContentTypeText, "text": value}
			if cacheControl := part["cache_control"]; cacheControl != nil {
				textOnly = false
				chatPart["cache_control"] = cacheControl
			}
			chatParts = append(chatParts, chatPart)
		case "input_image":
			textOnly = false
			imageURL := common.Interface2String(responsesPartValue(part, "image_url", "url"))
			if imageURL == "" {
				return nil, errors.New("input_image is missing image_url")
			}
			image := map[string]any{"url": imageURL}
			if detail := common.Interface2String(part["detail"]); detail != "" {
				image["detail"] = detail
			}
			chatPart := map[string]any{"type": dto.ContentTypeImageURL, "image_url": image}
			if cacheControl := part["cache_control"]; cacheControl != nil {
				chatPart["cache_control"] = cacheControl
			}
			chatParts = append(chatParts, chatPart)
		case "input_file":
			textOnly = false
			file, err := responsesFileValue(part)
			if err != nil {
				return nil, err
			}
			chatPart := map[string]any{"type": dto.ContentTypeFile, "file": file}
			if cacheControl := part["cache_control"]; cacheControl != nil {
				chatPart["cache_control"] = cacheControl
			}
			chatParts = append(chatParts, chatPart)
		default:
			return nil, fmt.Errorf("unsupported responses content type %q", common.Interface2String(part["type"]))
		}
	}
	if textOnly {
		return text.String(), nil
	}
	return chatParts, nil
}

func responsesPartValue(part map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := part[key]; ok && value != nil {
			return value
		}
	}
	return ""
}

func responsesFileValue(part map[string]any) (any, error) {
	if file, ok := part["file"]; ok {
		return file, nil
	}
	data := common.Interface2String(responsesPartValue(part, "file_data", "file_url", "url"))
	if data == "" {
		if common.Interface2String(part["file_id"]) != "" {
			return nil, errors.New("input_file.file_id cannot be resolved by Anthropic messages compatibility")
		}
		return nil, errors.New("input_file is missing file_data or file_url")
	}
	return map[string]any{"file_data": data, "filename": common.Interface2String(part["filename"])}, nil
}

func responsesToolsToChat(raw json.RawMessage) ([]dto.ToolCallRequest, *dto.WebSearchOptions, error) {
	if len(raw) == 0 {
		return nil, nil, nil
	}
	var tools []map[string]any
	if err := common.Unmarshal(raw, &tools); err != nil {
		return nil, nil, fmt.Errorf("invalid tools: %w", err)
	}
	out := make([]dto.ToolCallRequest, 0, len(tools))
	var webSearch *dto.WebSearchOptions
	for _, tool := range tools {
		switch common.Interface2String(tool["type"]) {
		case "function":
			name := strings.TrimSpace(common.Interface2String(tool["name"]))
			if name == "" {
				return nil, nil, errors.New("function tool is missing name")
			}
			parameters := tool["parameters"]
			if parameters == nil {
				parameters = map[string]any{"type": "object", "properties": map[string]any{}}
			}
			out = append(out, dto.ToolCallRequest{Type: "function", Function: dto.FunctionRequest{
				Name: name, Description: common.Interface2String(tool["description"]), Parameters: parameters,
			}})
		case "web_search", "web_search_preview":
			webSearch = &dto.WebSearchOptions{SearchContextSize: common.Interface2String(tool["search_context_size"])}
			if location, ok := tool["user_location"]; ok {
				webSearch.UserLocation, _ = common.Marshal(location)
			}
		default:
			return nil, nil, fmt.Errorf("unsupported responses tool type %q", common.Interface2String(tool["type"]))
		}
	}
	return out, webSearch, nil
}

func responsesToolChoiceToChat(raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	if common.GetJsonType(raw) == "string" {
		var choice string
		if err := common.Unmarshal(raw, &choice); err != nil {
			return nil, err
		}
		switch choice {
		case "auto", "none", "required":
			return choice, nil
		default:
			return nil, fmt.Errorf("unsupported tool_choice %q", choice)
		}
	}
	var choice map[string]any
	if err := common.Unmarshal(raw, &choice); err != nil {
		return nil, fmt.Errorf("invalid tool_choice: %w", err)
	}
	if common.Interface2String(choice["type"]) == "function" {
		return map[string]any{"type": "function", "function": map[string]any{"name": choice["name"]}}, nil
	}
	return nil, fmt.Errorf("unsupported tool_choice type %q", common.Interface2String(choice["type"]))
}

func responsesTextToChatResponseFormat(raw json.RawMessage) (*dto.ResponseFormat, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var text map[string]any
	if err := common.Unmarshal(raw, &text); err != nil {
		return nil, fmt.Errorf("invalid text format: %w", err)
	}
	format, _ := text["format"].(map[string]any)
	if len(format) == 0 {
		return nil, nil
	}
	typeName := common.Interface2String(format["type"])
	if typeName != "text" && typeName != "json_schema" {
		return nil, fmt.Errorf("unsupported responses text format type %q", typeName)
	}
	responseFormat := &dto.ResponseFormat{Type: typeName}
	if typeName == "json_schema" {
		responseFormat.JsonSchema, _ = common.Marshal(format)
	}
	return responseFormat, nil
}

func responsesFunctionCallToChat(item map[string]any) (dto.ToolCallRequest, error) {
	name := strings.TrimSpace(common.Interface2String(item["name"]))
	if name == "" {
		return dto.ToolCallRequest{}, errors.New("function_call item is missing name")
	}
	return dto.ToolCallRequest{ID: responsesCallID(item), Type: "function", Function: dto.FunctionRequest{
		Name: name, Arguments: stringifyResponsesValue(item["arguments"]),
	}}, nil
}

func appendToolCallToAssistant(messages []dto.Message, call dto.ToolCallRequest) []dto.Message {
	if len(messages) == 0 || messages[len(messages)-1].Role != "assistant" {
		messages = append(messages, dto.Message{Role: "assistant", Content: ""})
	}
	index := len(messages) - 1
	calls := messages[index].ParseToolCalls()
	calls = append(calls, call)
	messages[index].ToolCalls, _ = common.Marshal(calls)
	return messages
}

func responsesCallID(item map[string]any) string {
	if id := strings.TrimSpace(common.Interface2String(item["call_id"])); id != "" {
		return id
	}
	return strings.TrimSpace(common.Interface2String(item["id"]))
}

func stringifyResponsesValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	data, err := common.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}

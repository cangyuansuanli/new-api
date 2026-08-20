# Claude OpenAI Protocol Compatibility

The Anthropic channel accepts three client-facing protocols:

| Client endpoint | Anthropic upstream | Response format |
|---|---|---|
| `POST /v1/messages` | Native `POST /v1/messages` | Native Anthropic Messages JSON/SSE |
| `POST /v1/chat/completions` | Converted to `POST /v1/messages` | OpenAI Chat Completions JSON/SSE |
| `POST /v1/responses` | Normalized through canonical Chat, then converted to `POST /v1/messages` | OpenAI Responses JSON/SSE |

The compatibility boundary is channel-local. OpenAI and OpenAI-compatible channels keep their existing request conversion, routing, and response handling.

## Conversion Layers

The Responses compatibility path is intentionally split into reusable protocol layers:

```text
OpenAI Responses request
  -> canonical Chat request
  -> Anthropic Messages request
  -> canonical Chat response/chunks
  -> OpenAI Responses response/events
```

`service/openaicompat` owns protocol-generic normalization and response event generation. `relay/channel/claude` owns Anthropic-specific media, tool, reasoning, structured-output, usage, and stop-reason conversion.

## Supported Responses Subset

The stateless `/v1/responses` compatibility path supports:

- instructions and user/assistant message content;
- text, image, PDF, and text-file inputs supported by Anthropic Messages;
- function tools, tool choice, parallel tool calls, function calls, and function outputs;
- reasoning effort and JSON Schema structured output where the selected Claude model supports them;
- non-streaming responses and Responses SSE events;
- usage, cached input tokens, incomplete status, and client-facing model restoration.

Stateful OpenAI fields such as `conversation`, `previous_response_id`, hosted prompts, and context management are rejected explicitly. OpenAI built-in tools that have no Anthropic equivalent are also rejected instead of being silently discarded.

For the widest Claude-native feature coverage, clients should continue to use `/v1/messages`.

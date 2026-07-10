package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/packages/ssestream"

	openagent "github.com/yusheng-g/openagent-go"
)

// Model implements openagent.Model via openai-go v3.
type Model struct {
	client        openaisdk.Client
	modelID       string
	contextWindow int
}

// New creates a Model with the given API key, model ID, and base URL.
func New(apiKey, modelID, baseURL string) *Model {
	return &Model{
		client: openaisdk.NewClient(
			option.WithAPIKey(apiKey),
			option.WithBaseURL(baseURL),
		),
		modelID: modelID,
	}
}

func (m *Model) WithContextWindow(tokens int) *Model { m.contextWindow = tokens; return m }
func (m *Model) ContextWindow() int                  { return m.contextWindow }

func (m *Model) ChatCompletion(ctx context.Context, req openagent.ChatCompletionRequest) (*openagent.ChatCompletionResponse, error) {
	modelID := req.Model
	if modelID == "" {
		modelID = m.modelID
	}

	params := openaisdk.ChatCompletionNewParams{
		Model:    openaisdk.ChatModel(modelID),
		Messages: toSDKMessages(req.Messages),
		Tools:    toSDKTools(req.Tools),
	}
	if req.Temperature != 0 {
		params.Temperature = param.NewOpt(req.Temperature)
	}
	if req.MaxTokens != 0 {
		params.MaxTokens = param.NewOpt(int64(req.MaxTokens))
	}
	if req.TopP != 0 {
		params.TopP = param.NewOpt(req.TopP)
	}
	if len(req.Stop) > 0 {
		params.Stop = openaisdk.ChatCompletionNewParamsStopUnion{
			OfStringArray: req.Stop,
		}
	}

	completion, err := m.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, toRetryableError(err)
	}
	return toResponse(completion), nil
}

// ChatCompletionStream implements openagent.Model.
func (m *Model) ChatCompletionStream(ctx context.Context, req openagent.ChatCompletionRequest) (openagent.StreamReader, error) {
	modelID := req.Model
	if modelID == "" {
		modelID = m.modelID
	}

	params := openaisdk.ChatCompletionNewParams{
		Model:    openaisdk.ChatModel(modelID),
		Messages: toSDKMessages(req.Messages),
		Tools:    toSDKTools(req.Tools),
	}
	if req.Temperature != 0 {
		params.Temperature = param.NewOpt(req.Temperature)
	}
	if req.MaxTokens != 0 {
		params.MaxTokens = param.NewOpt(int64(req.MaxTokens))
	}
	if req.TopP != 0 {
		params.TopP = param.NewOpt(req.TopP)
	}
	if len(req.Stop) > 0 {
		params.Stop = openaisdk.ChatCompletionNewParamsStopUnion{
			OfStringArray: req.Stop,
		}
	}

	stream := m.client.Chat.Completions.NewStreaming(ctx, params)
	if err := stream.Err(); err != nil {
		return nil, toRetryableError(err)
	}
	return &streamReader{stream: stream}, nil
}

// toRetryableError wraps 429/503 errors so the Runner can retry.
func toRetryableError(err error) error {
	var apiErr *openaisdk.Error
	if errors.As(err, &apiErr) && (apiErr.StatusCode == 429 || apiErr.StatusCode == 503) {
		return &openagent.RetryableError{Err: err}
	}
	return fmt.Errorf("chat completion: %w", err)
}

// ── openagent → SDK ──

func toSDKMessages(msgs []openagent.Message) []openaisdk.ChatCompletionMessageParamUnion {
	out := make([]openaisdk.ChatCompletionMessageParamUnion, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, toSDKMessage(m))
	}
	return out
}

func toSDKMessage(m openagent.Message) openaisdk.ChatCompletionMessageParamUnion {
	switch m.Role {
	case openagent.RoleSystem:
		return openaisdk.SystemMessage(m.Content)

	case openagent.RoleUser:
		if m.IsMultimodal() {
			return openaisdk.UserMessage(toSDKContentParts(m.ContentParts))
		}
		return openaisdk.UserMessage(m.Content)

	case openagent.RoleAssistant:
		if len(m.ToolCalls) > 0 {
			return openaisdk.ChatCompletionMessageParamUnion{
				OfAssistant: &openaisdk.ChatCompletionAssistantMessageParam{
					ToolCalls: toSDKToolCallParams(m.ToolCalls),
				},
			}
		}
		return openaisdk.AssistantMessage(m.Content)

	case openagent.RoleTool:
		return openaisdk.ToolMessage(m.Content, m.ToolCallID)

	default:
		return openaisdk.UserMessage(m.Content)
	}
}

func toSDKContentParts(parts []openagent.ContentPart) []openaisdk.ChatCompletionContentPartUnionParam {
	out := make([]openaisdk.ChatCompletionContentPartUnionParam, len(parts))
	for i, p := range parts {
		switch p.Type {
		case "text":
			out[i] = openaisdk.TextContentPart(p.Text)
		case "image_url":
			out[i] = openaisdk.ImageContentPart(openaisdk.ChatCompletionContentPartImageImageURLParam{
				URL: p.ImageURL.URL,
			})
		}
	}
	return out
}

func toSDKToolCallParams(calls []openagent.ToolCall) []openaisdk.ChatCompletionMessageToolCallUnionParam {
	out := make([]openaisdk.ChatCompletionMessageToolCallUnionParam, len(calls))
	for i, c := range calls {
		out[i] = openaisdk.ChatCompletionMessageToolCallUnionParam{
			OfFunction: &openaisdk.ChatCompletionMessageFunctionToolCallParam{
				ID: c.ID,
				Function: openaisdk.ChatCompletionMessageFunctionToolCallFunctionParam{
					Name:      c.Function.Name,
					Arguments: c.Function.Arguments,
				},
			},
		}
	}
	return out
}

func toSDKTools(defs []openagent.FunctionDefinition) []openaisdk.ChatCompletionToolUnionParam {
	out := make([]openaisdk.ChatCompletionToolUnionParam, len(defs))
	for i, d := range defs {
		var params map[string]any
		if len(d.Parameters) > 0 {
			if err := json.Unmarshal(d.Parameters, &params); err != nil {
				// Invalid JSON Schema — fall back to empty params so the
				// tool is still sent to the model rather than dropped.
				params = map[string]any{}
			}
		}
		out[i] = openaisdk.ChatCompletionToolUnionParam{
			OfFunction: &openaisdk.ChatCompletionFunctionToolParam{
				Function: openaisdk.FunctionDefinitionParam{
					Name:        d.Name,
					Description: param.NewOpt(d.Description),
					Parameters:  openaisdk.FunctionParameters(params),
				},
			},
		}
	}
	return out
}

// ── SDK → openagent ──

func toResponse(c *openaisdk.ChatCompletion) *openagent.ChatCompletionResponse {
	resp := &openagent.ChatCompletionResponse{}
	for _, choice := range c.Choices {
		msg := openagent.Message{
			Role:             openagent.RoleAssistant,
			Content:          choice.Message.Content,
			ReasoningContent: extractReasoning(choice.Message.RawJSON()),
		}
		for _, tc := range choice.Message.ToolCalls {
			msg.ToolCalls = append(msg.ToolCalls, openagent.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: openagent.ToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		resp.Choices = append(resp.Choices, openagent.Choice{
			Index:        int(choice.Index),
			Message:      msg,
			FinishReason: choice.FinishReason,
		})
	}
	if c.Usage.TotalTokens > 0 {
		resp.Usage = openagent.Usage{
			PromptTokens:     int(c.Usage.PromptTokens),
			CompletionTokens: int(c.Usage.CompletionTokens),
			TotalTokens:      int(c.Usage.TotalTokens),
		}
	}
	return resp
}

// ── Stream wrapper ──

// streamReader adapts ssestream.Stream to openagent.StreamReader.
type streamReader struct {
	stream  *ssestream.Stream[openaisdk.ChatCompletionChunk]
	current openagent.StreamChunk
	done    bool
}

func (s *streamReader) Next() bool {
	if s.done {
		return false
	}
	if !s.stream.Next() {
		s.done = true
		return false
	}
	s.current = toStreamChunk(s.stream.Current())
	return true
}

func (s *streamReader) Current() openagent.StreamChunk { return s.current }
func (s *streamReader) Err() error {
	err := s.stream.Err()
	// Wrap 429/503 errors so the Runner can retry mid-stream failures.
	// Non-retryable errors are returned as-is to preserve error semantics.
	var apiErr *openaisdk.Error
	if errors.As(err, &apiErr) && (apiErr.StatusCode == 429 || apiErr.StatusCode == 503) {
		return &openagent.RetryableError{Err: err}
	}
	return err
}
func (s *streamReader) Close() error                     { return s.stream.Close() }

func toStreamChunk(c openaisdk.ChatCompletionChunk) openagent.StreamChunk {
	sc := openagent.StreamChunk{}
	if c.Usage.TotalTokens > 0 {
		sc.Usage = &openagent.Usage{
			PromptTokens:     int(c.Usage.PromptTokens),
			CompletionTokens: int(c.Usage.CompletionTokens),
			TotalTokens:      int(c.Usage.TotalTokens),
		}
	}
	for _, choice := range c.Choices {
		sd := openagent.StreamDelta{
			Content:          choice.Delta.Content,
			ReasoningContent: extractReasoning(choice.Delta.RawJSON()),
			FinishReason:     choice.FinishReason,
		}
		for _, tc := range choice.Delta.ToolCalls {
			sd.ToolCalls = append(sd.ToolCalls, openagent.ToolCallDelta{
				Index: int(tc.Index),
				ID:    tc.ID,
				Type:  tc.Type,
				Function: openagent.FunctionDelta{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		sc.Choices = append(sc.Choices, sd)
	}
	return sc
}

// extractReasoning extracts "reasoning_content" from raw JSON. The openai-go
// SDK doesn't have a typed field for it (as of v3.41), but reasoning models
// (o1, deepseek-r1) include it in delta chunks and message responses.
func extractReasoning(raw string) string {
	if raw == "" {
		return ""
	}
	// Fast path: look for "reasoning_content":"..." in the raw JSON.
	const key = `"reasoning_content":"`
	idx := strings.Index(raw, key)
	if idx < 0 {
		return ""
	}
	start := idx + len(key)
	// Scan until the closing unescaped quote.
	var buf strings.Builder
	for i := start; i < len(raw); i++ {
		if raw[i] == '\\' && i+1 < len(raw) {
			switch raw[i+1] {
			case '"':
				buf.WriteByte('"')
				i++
			case 'n':
				buf.WriteByte('\n')
				i++
			case 't':
				buf.WriteByte('\t')
				i++
			case '\\':
				buf.WriteByte('\\')
				i++
			default:
				buf.WriteByte(raw[i])
			}
		} else if raw[i] == '"' {
			break
		} else {
			buf.WriteByte(raw[i])
		}
	}
	return buf.String()
}

package openagent

import (
	"context"
	"time"
)

// ChatCompletionRequest is the input to a model call.
// Follows OpenAI Chat Completions API format.
type ChatCompletionRequest struct {
	Model       string               `json:"model"`
	Messages    []Message            `json:"messages"`
	Tools       []FunctionDefinition `json:"tools,omitempty"`
	Temperature float64              `json:"temperature,omitempty"`
	MaxTokens   int                  `json:"max_tokens,omitempty"`
	TopP        float64              `json:"top_p,omitempty"`
	Stop        []string             `json:"stop,omitempty"`
	Stream      bool                 `json:"stream,omitempty"`
}

// ChatCompletionResponse is the result of a model call.
type ChatCompletionResponse struct {
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage,omitempty"`
}

// Choice is a single completion choice.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage records token counts for a model call.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Model is the interface for LLM providers. Implementations handle API-specific
// details (OpenAI, Anthropic, etc.). The Runner calls ChatCompletionStream by
// default and falls back to ChatCompletion if streaming is not available.
type Model interface {
	ChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error)

	// ChatCompletionStream returns a stream of response deltas.
	// Return nil, nil if streaming is not supported (Runner falls back to ChatCompletion).
	ChatCompletionStream(ctx context.Context, req ChatCompletionRequest) (StreamReader, error)

	// ContextWindow returns the maximum input token count for this model.
	// Return 0 if unknown (Runner will not auto-compact).
	ContextWindow() int
}

// ── Streaming types ──

// StreamReader is an iterator over response deltas.
// Follows the pattern of openai-go v3 ssestream.Stream[T].
type StreamReader interface {
	Next() bool           // advance to next chunk
	Current() StreamChunk // current chunk (valid after Next() == true)
	Err() error           // stream error (check after Next() == false)
	Close() error
}

// StreamChunk is a single chunk in a streaming response.
type StreamChunk struct {
	Choices []StreamDelta `json:"choices"`
	Usage   *Usage        `json:"usage,omitempty"` // last chunk may include usage
}

// StreamDelta is the delta content within a chunk.
type StreamDelta struct {
	Content          string          `json:"content,omitempty"`           // text delta (empty if tool call)
	ReasoningContent string          `json:"reasoning_content,omitempty"` // reasoning/thinking (o1, deepseek-r1)
	ToolCalls        []ToolCallDelta `json:"tool_calls,omitempty"`        // incremental tool call fragments
	FinishReason     string          `json:"finish_reason,omitempty"`     // set in final chunk
}

// ToolCallDelta is an incremental tool call update in a stream chunk.
type ToolCallDelta struct {
	Index    int           `json:"index"`    // 0-based, fragments with same index belong together
	ID       string        `json:"id,omitempty"`
	Type     string        `json:"type,omitempty"`
	Function FunctionDelta `json:"function,omitempty"`
}

// FunctionDelta is an incremental function name/arguments update.
type FunctionDelta struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"` // JSON fragment, concatenated across chunks
}

// ── Embedding ──

// Embedder converts text to a dense vector. Implementations: OpenAI, local models.
// nil Embedder means semantic search is disabled; Memory falls back to keyword search.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
	Dimensions() int
}

// ── Summarizer ──

// Summarizer compresses old messages into a summary for context window management.
// nil means auto-compaction is disabled.
//
// When previous is non-nil, this is an incremental (rolling) compression —
// the implementation should incorporate the existing summary rather than
// re-summarizing from scratch. The returned CompressedContext.ThroughIndex
// should be set by the caller (memory backend), not the implementation.
type Summarizer interface {
	Summarize(ctx context.Context, messages []Message, previous *CompressedContext) (*CompressedContext, error)
}

// ── Retry ──

// RetryableError wraps a transient error (429, 503) so the Runner can retry.
type RetryableError struct {
	Err        error
	RetryAfter time.Duration // 0 = use default exponential backoff
}

func (e *RetryableError) Error() string { return e.Err.Error() }
func (e *RetryableError) Unwrap() error { return e.Err }

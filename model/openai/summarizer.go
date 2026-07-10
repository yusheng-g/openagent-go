package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"

	openagent "github.com/yusheng-g/openagent-go"
)

// Summarizer implements openagent.Summarizer via OpenAI ChatCompletion.
type Summarizer struct {
	client   openaisdk.Client
	modelID  string
	maxTokens int // 0 = default (1024)
}

// NewSummarizer creates a Summarizer. modelID defaults to the same model
// used for conversations (chat model, not embedding model).
func NewSummarizer(apiKey, modelID, baseURL string) *Summarizer {
	return &Summarizer{
		client: openaisdk.NewClient(
			option.WithAPIKey(apiKey),
			option.WithBaseURL(baseURL),
		),
		modelID: modelID,
	}
}

// WithMaxTokens sets the max output tokens for summarization.
// Default is 1024. Set this to match the agent's MaxCompressedTokens
// to keep the summary within the configured budget.
func (s *Summarizer) WithMaxTokens(n int) *Summarizer {
	s.maxTokens = n
	return s
}

// Summarize compresses messages into a summary with retrieval hints.
// When previous is non-nil, this is an incremental compression — the existing
// summary is prepended as context so the LLM can update it with the new messages.
func (s *Summarizer) Summarize(ctx context.Context, messages []openagent.Message, previous *openagent.CompressedContext) (*openagent.CompressedContext, error) {
	// Build conversation transcript
	var transcript strings.Builder
	for _, m := range messages {
		transcript.WriteString(string(m.Role))
		transcript.WriteString(": ")
		transcript.WriteString(m.Content)
		if len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				transcript.WriteString("\n  tool_call: ")
				transcript.WriteString(tc.Function.Name)
				transcript.WriteString("(")
				transcript.WriteString(tc.Function.Arguments)
				transcript.WriteString(")")
			}
		}
		if m.Role == openagent.RoleTool {
			transcript.WriteString(" // tool result")
		}
		transcript.WriteString("\n")
	}

	// Build the system prompt. For incremental compression, include the
	// previous summary so the LLM can preserve and update existing facts.
	systemPrompt := summarizerPrompt
	if previous != nil && previous.Summary != "" {
		systemPrompt = summarizerIncrementalPrompt(previous)
	}

	maxTok := s.maxTokens
	if maxTok <= 0 {
		maxTok = 1024
	}
	params := openaisdk.ChatCompletionNewParams{
		Model: openaisdk.ChatModel(s.modelID),
		Messages: []openaisdk.ChatCompletionMessageParamUnion{
			openaisdk.SystemMessage(systemPrompt),
			openaisdk.UserMessage(transcript.String()),
		},
		Temperature: param.NewOpt(0.3),
		MaxTokens:   param.NewOpt(int64(maxTok)),
	}

	completion, err := s.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("summarize: %w", err)
	}

	content := completion.Choices[0].Message.Content
	return parseSummaryResponse(content)
}

func parseSummaryResponse(content string) (*openagent.CompressedContext, error) {
	// Strip markdown code fences if present
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var cc openagent.CompressedContext
	if err := json.Unmarshal([]byte(content), &cc); err != nil {
		// Fallback: treat entire response as a plain summary
		return &openagent.CompressedContext{Summary: content}, nil
	}
	return &cc, nil
}

const summarizerPrompt = `You are a conversation summarizer. Compress the following conversation into:

1. A concise SUMMARY (2-4 sentences) capturing key facts, decisions, and context.
2. 0-5 RETRIEVAL HINTS — each a short label + a search query to find the original details.

Respond with ONLY valid JSON (no markdown, no commentary):
{
  "summary": "string",
  "hints": [
    {"description": "short label", "query": "search terms"}
  ]
}

Rules:
- Preserve user preferences, personal info, and factual claims verbatim.
- Hints should be specific search queries, not generic descriptions.
- If the conversation is trivial (greetings, chitchat), hints can be empty.`

// summarizerIncrementalPrompt builds a prompt that asks the LLM to update an
// existing summary with new messages, rather than starting from scratch.
// This enables rolling/incremental compression without information decay.
func summarizerIncrementalPrompt(previous *openagent.CompressedContext) string {
	hintsJSON, _ := json.Marshal(previous.Hints)
	return fmt.Sprintf(`You are a conversation summarizer updating an existing summary.

EXISTING SUMMARY (from earlier in the conversation):
%s

EXISTING RETRIEVAL HINTS:
%s

Below are NEW messages that occurred after the existing summary. Update the summary to incorporate the new information.

Rules:
- PRESERVE all key facts from the existing summary. Only add or modify to reflect new info.
- Merge related facts rather than duplicating.
- If new messages contradict old ones, prefer the newer information.
- Produce 0-5 retrieval hints total (merge old + new, keep the most useful).
- If the new messages are trivial, keep the existing summary unchanged.

Respond with ONLY valid JSON (no markdown, no commentary):
{
  "summary": "string",
  "hints": [
    {"description": "short label", "query": "search terms"}
  ]
}`, previous.Summary, string(hintsJSON))
}

var _ openagent.Summarizer = (*Summarizer)(nil)

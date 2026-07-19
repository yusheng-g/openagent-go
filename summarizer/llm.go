// Package summarizer provides LLM-based conversation compression
// implementing openagent.Summarizer.
//
// The Compressor uses the agent's Model to generate incremental,
// rolling summaries of older messages. Each summary subsumes the
// previous one so the context window stays compact without losing
// long-term history.
package summarizer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
)

// Compressor implements openagent.Summarizer by calling the configured
// Model to produce incremental summaries.
type Compressor struct {
	model openagent.Model
}

// New creates a Compressor backed by m.
func New(m openagent.Model) *Compressor {
	return &Compressor{model: m}
}

// Summarize implements openagent.Summarizer.
//
// When previous is nil this is the first compression pass — a fresh
// summary is generated. Otherwise the new messages are folded into the
// existing summary, producing an updated CompressedContext whose
// ThroughIndex is left at zero (the caller sets it).
func (c *Compressor) Summarize(ctx context.Context, messages []openagent.Message, previous *openagent.CompressedContext) (*openagent.CompressedContext, error) {
	if c.model == nil {
		return nil, fmt.Errorf("summarizer: no model configured")
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("summarizer: no messages to summarize")
	}

	prompt := buildSummarizePrompt(messages, previous)
	resp, err := c.model.ChatCompletion(ctx, openagent.ChatCompletionRequest{
		Messages: []openagent.Message{
			{Role: openagent.RoleSystem, Content: summarizeSystemPrompt},
			{Role: openagent.RoleUser, Content: prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("summarizer: model call: %w", err)
	}

	content := ""
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
	}
	return parseSummary(content)
}

// ── Prompt ──

const summarizeSystemPrompt = `You are a conversation summarizer. Your job is to produce a concise,
structured summary of a conversation so an AI assistant can resume the
thread without re-reading every message.

Rules:
- Capture key facts, decisions, user preferences, and ongoing tasks.
- Include up to 5 retrieval hints — short keyword queries the assistant
  could use to find specific details later.
- Be concise. The summary is injected into a system prompt.
- Output ONLY a JSON object. No markdown fences, no surrounding text.

Format:
{"summary": "<text>", "hints": [{"description": "<what to find>", "query": "<search keywords>"}]}`

func buildSummarizePrompt(messages []openagent.Message, prev *openagent.CompressedContext) string {
	var b strings.Builder
	if prev != nil && prev.Summary != "" {
		b.WriteString("## Existing Summary\n")
		b.WriteString(prev.Summary)
		b.WriteString("\n\n## New Messages (incorporate into the summary)\n\n")
	} else {
		b.WriteString("## Messages to Summarize\n\n")
	}
	for _, m := range messages {
		switch m.Role {
		case openagent.RoleUser:
			b.WriteString("User: ")
		case openagent.RoleAssistant:
			b.WriteString("Assistant: ")
		case openagent.RoleTool:
			if m.ToolCallID != "" {
				fmt.Fprintf(&b, "Tool result (%s): ", m.ToolCallID)
			} else {
				b.WriteString("System: ")
			}
		case openagent.RoleSystem:
			b.WriteString("System: ")
		}
		b.WriteString(truncateContent(m.Content, 300))
		b.WriteString("\n")
		if len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				fmt.Fprintf(&b, "  [called tool %s]\n", tc.Function.Name)
			}
		}
	}
	return b.String()
}

func truncateContent(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

// ── Parsing ──

func parseSummary(raw string) (*openagent.CompressedContext, error) {
	raw = strings.TrimSpace(raw)
	// Strip markdown fences.
	if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		if idx := strings.LastIndex(raw, "```"); idx >= 0 {
			raw = raw[:idx]
		}
		raw = strings.TrimSpace(raw)
	}
	// Find JSON object bounds.
	if idx := strings.Index(raw, "{"); idx > 0 {
		raw = raw[idx:]
	}
	if idx := strings.LastIndex(raw, "}"); idx >= 0 && idx < len(raw)-1 {
		raw = raw[:idx+1]
	}

	var out struct {
		Summary string `json:"summary"`
		Hints   []struct {
			Description string `json:"description"`
			Query       string `json:"query"`
		} `json:"hints"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("summarizer: parse JSON: %w (raw: %.200s)", err, raw)
	}
	if out.Summary == "" {
		return nil, fmt.Errorf("summarizer: model returned empty summary")
	}

	cc := &openagent.CompressedContext{Summary: out.Summary}
	for _, h := range out.Hints {
		if h.Description != "" && h.Query != "" {
			cc.Hints = append(cc.Hints, openagent.RetrievalHint{
				Description: h.Description,
				Query:       h.Query,
			})
		}
	}
	return cc, nil
}

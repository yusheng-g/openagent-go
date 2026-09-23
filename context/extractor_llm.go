package context

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	openagent "github.com/yusheng-g/openagent-go"
)

// Knowledge extraction: the self-evolution loop. After a session finishes,
// the extractor scans the conversation for durable knowledge (user
// preferences, project facts, successful approaches) and stores it via
// the MemoryProvider so future sessions recall it automatically.
//
// The LLM extractor follows the industry pattern (Mem0): one pass extracts
// candidate facts AND classifies each against existing knowledge — add
// (new topic), update (same topic, new/changed information → the store
// upserts by scope+topic), or skip (already covered / not durable). This
// prevents memory bloat: repeated sessions about the same topic converge
// on one entry instead of accumulating duplicates.

// Extractor scans finished conversations for durable knowledge.
//
// Extract is fire-and-forget: implementations may run synchronously
// (LLMExtractor) or enqueue to a background worker (AsyncExtractor).
// Errors are logged inside, never returned — extraction must not affect
// the agent run.
type Extractor interface {
	// Extract stores durable knowledge under scope. Best-effort: failures
	// never abort the run.
	Extract(ctx context.Context, scope ContextScope, messages []openagent.Message)
}

// ExtractionItem is one knowledge decision from the extraction pass.
type ExtractionItem struct {
	Op      string `json:"op"`      // "add" | "update" | "skip"
	Kind    string `json:"kind"`    // "preference" | "fact" | "lesson"
	Content string `json:"content"` // concise, self-contained statement
	Topic   string `json:"topic"`   // topic key; update reuses the existing entry's topic
}

// extractionPrompt instructs the model to extract durable knowledge and
// classify it against existing knowledge (Mem0-style ADD/UPDATE/SKIP).
const extractionPrompt = `You are a knowledge curator. From the conversation below, extract durable knowledge worth remembering across future sessions.

Extraction criteria:
- User preferences, project facts, lessons learned — things that will matter later
- Rewrite each as a concise, self-contained statement (never a verbatim quote)
- Skip: small talk, one-time requests, temporary state, anything not reusable

Existing knowledge (topic → content) — compare each candidate against it:
- Same topic with new or changed information → "update": reuse the SAME topic, provide the updated content
- New topic → "add": give a short topic key
- Already covered, nothing new → "skip"

Output at most 5 items. Keep each content under 200 characters.
If nothing is worth extracting, respond with [].

Respond with ONLY a JSON array, no markdown fences:
[{"op":"add","kind":"preference","content":"...","topic":"..."}]
"kind" is one of "preference", "fact", "lesson".`

// extractionBudget caps the transcript tokens fed to the model.
const extractionBudget = 4000

// maxKnowledgeItems caps how many items one extraction pass stores.
const maxKnowledgeItems = 10

// maxExistingText caps the existing-knowledge text fed to the model.
// OpenViking recall returns up to 6500 chars; capping at 1500 keeps the
// total input manageable and leaves room for the model's output.
const maxExistingText = 1500

// extractionRetryPrompt is sent after a non-JSON or truncated response,
// following the same corrective-retry pattern as guard/llm/guard.go.
const extractionRetryPrompt = `Your previous response was not valid JSON. Respond with ONLY a JSON array (no markdown, no prose), at most 3 items, each content under 100 characters. If nothing is worth extracting, respond with [].`

const extractionMaxTokens = 8192

const logRawTruncate = 300

// LLMExtractor extracts knowledge with the runtime model.
type LLMExtractor struct {
	mu sync.RWMutex
	// modelFn resolves the current model at extraction time (not held as
	// a snapshot). This ensures the extractor always uses the latest model
	// instance — when SetModel updates the registry (new api_key/base_url),
	// the next extraction picks it up without an explicit SetModel call.
	modelFn func() openagent.Model
	// Provider stores extracted knowledge.
	Provider MemoryProvider
	// MaxItems caps stored items per pass (default 10).
	MaxItems int
}

// NewLLMExtractor creates an extractor. modelFn resolves the model at
// extraction time; a nil function or nil model yields a no-op.
func NewLLMExtractor(modelFn func() openagent.Model, p MemoryProvider) *LLMExtractor {
	return &LLMExtractor{modelFn: modelFn, Provider: p, MaxItems: maxKnowledgeItems}
}

// SetModel updates the model resolver. Safe to call concurrently with
// Extract; the next call uses the new resolver.
func (e *LLMExtractor) SetModel(m openagent.Model) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.modelFn = func() openagent.Model { return m }
}

// SetModelFn updates the model resolver function. Use this when the
// model should be looked up dynamically (e.g. from a registry that may
// be updated at runtime).
func (e *LLMExtractor) SetModelFn(fn func() openagent.Model) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.modelFn = fn
}

// Extract implements Extractor.
func (e *LLMExtractor) Extract(ctx context.Context, scope ContextScope, messages []openagent.Message) {
	e.mu.RLock()
	modelFn := e.modelFn
	e.mu.RUnlock()
	var model openagent.Model
	if modelFn != nil {
		model = modelFn()
	}
	if e == nil || model == nil || e.Provider == nil || len(messages) == 0 {
		return
	}
	slog.Debug("knowledge extraction triggered", "user", scope.UserID, "messages", len(messages))
	max := e.MaxItems
	if max <= 0 {
		max = maxKnowledgeItems
	}

	// Build the transcript within the token budget (recent messages win).
	transcript := buildTranscript(messages)
	if openagent.CountTokens("gpt-4", transcript) > extractionBudget {
		transcript = trimTranscript(transcript, extractionBudget)
	}
	if transcript == "" {
		return
	}

	// Existing knowledge, so the model can classify add/update/skip.
	query := recallQuery(messages)
	existing, _ := e.Provider.Recall(ctx, scope, query, 20)
	var existingText strings.Builder
	for _, m := range existing {
		fmt.Fprintf(&existingText, "- %s → %s\n", m.Topic, m.Content)
	}
	if existingText.Len() == 0 {
		existingText.WriteString("(none)")
	}
	// Cap existing-knowledge text so it doesn't squeeze the model's output
	// budget — OpenViking recall can return up to 6500 chars.
	if existingText.Len() > maxExistingText {
		s := existingText.String()
		cut := maxExistingText
		if i := strings.LastIndex(s[:cut], "\n"); i > 0 {
			cut = i
		}
		existingText.Reset()
		existingText.WriteString(s[:cut])
		existingText.WriteString("\n... (truncated)")
	}

	input := "Conversation:\n" + transcript +
		"\n\nExisting knowledge:\n" + existingText.String()

	resp, err := model.ChatCompletion(ctx, openagent.ChatCompletionRequest{
		Messages: []openagent.Message{
			{Role: openagent.RoleSystem, Content: extractionPrompt},
			{Role: openagent.RoleUser, Content: input},
		},
		MaxTokens: extractionMaxTokens,
	})
	if err != nil || resp == nil || len(resp.Choices) == 0 {
		slog.Warn("knowledge extract failed", "error", err)
		return
	}

	choice := resp.Choices[0]
	raw := choice.Message.Content

	// An empty response is normal — the model found nothing worth
	// extracting this turn. Log finish_reason only when it indicates a
	// truncation or filter that may have suppressed content.
	if strings.TrimSpace(raw) == "" {
		if fr := choice.FinishReason; fr == "length" || fr == "content_filter" {
			slog.Warn("knowledge extract empty response",
				"finish_reason", fr)
		}
		return
	}

	items, err := parseExtractionItems(raw)
	if err != nil {
		// Corrective retry: the model returned non-JSON (e.g. a Chinese
		// refusal) or truncated JSON with zero salvageable items. Send
		// the failed response back with a format correction, following
		// the same pattern as guard/llm/guard.go retryJudge.
		slog.Warn("knowledge extract parse failed, retrying", "error", err, "raw", truncateForLog(raw))

		retryResp, retryErr := model.ChatCompletion(ctx, openagent.ChatCompletionRequest{
			Messages: []openagent.Message{
				{Role: openagent.RoleSystem, Content: extractionPrompt},
				{Role: openagent.RoleUser, Content: input},
				{Role: openagent.RoleAssistant, Content: raw},
				{Role: openagent.RoleUser, Content: extractionRetryPrompt},
			},
			MaxTokens: extractionMaxTokens,
		})
		if retryErr == nil && retryResp != nil && len(retryResp.Choices) > 0 {
			raw = retryResp.Choices[0].Message.Content
			items, err = parseExtractionItems(raw)
		}
	}
	if err != nil {
		slog.Warn("knowledge extract parse failed", "error", err, "raw", truncateForLog(raw))
		return
	}

	stored := 0
	seen := make(map[string]bool)
	for _, it := range items {
		if stored >= max {
			break
		}
		if it.Op == "skip" || strings.TrimSpace(it.Content) == "" {
			continue
		}
		content := strings.TrimSpace(it.Content)
		if len(content) < 12 || seen[content] {
			continue
		}
		seen[content] = true
		err := e.Provider.Store(ctx, scope, MemoryItem{
			Kind:    it.Kind,
			Content: content,
			Topic:   strings.TrimSpace(it.Topic),
			Meta:    map[string]any{"source": "extractor"},
		})
		if err != nil {
			// A failed store is invisible to the run (extraction is
			// fire-and-forget) — log it or the user silently loses the
			// extracted knowledge.
			slog.Warn("knowledge store failed", "error", err)
			continue
		}
		stored++
	}
	slog.Debug("knowledge extraction done", "user", scope.UserID, "stored", stored, "candidates", len(items))
}

// buildTranscript renders messages as a compact transcript (role: content).
func buildTranscript(messages []openagent.Message) string {
	var b strings.Builder
	for _, m := range messages {
		if strings.TrimSpace(m.Content) == "" {
			continue
		}
		b.WriteString(string(m.Role))
		b.WriteString(": ")
		b.WriteString(m.Content)
		b.WriteString("\n")
	}
	return b.String()
}

// trimTranscript keeps the most recent content within the token budget.
// The transcript is in chronological order, so walking from the END keeps
// the latest messages (the oldest overflow is dropped) — matching the
// "recent messages win" contract at the call site.
func trimTranscript(s string, budget int) string {
	lines := strings.Split(s, "\n")
	var keep []string
	total := 0
	for i := len(lines) - 1; i >= 0; i-- {
		n := openagent.CountTokens("gpt-4", lines[i])
		if total+n > budget {
			break // budget exhausted — drop everything older
		}
		total += n
		keep = append(keep, lines[i])
	}
	// Reverse back to chronological order.
	for i, j := 0, len(keep)-1; i < j; i, j = i+1, j-1 {
		keep[i], keep[j] = keep[j], keep[i]
	}
	return strings.Join(keep, "\n")
}

// recallQuery derives the recall query from the most recent user message.
func recallQuery(messages []openagent.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == openagent.RoleUser && strings.TrimSpace(messages[i].Content) != "" {
			c := messages[i].Content
			if len(c) > 200 {
				c = c[:200]
			}
			return c
		}
	}
	return ""
}

// truncateForLog truncates a string for log output, capped at logRawTruncate.
func truncateForLog(s string) string {
	if len(s) > logRawTruncate {
		return s[:logRawTruncate]
	}
	return s
}

// parseExtractionItems parses the model's JSON array response (markdown
// fences tolerated). An empty string returns nil (nothing to extract).
//
// When the JSON is truncated (finish_reason: "length" cut the output
// mid-array), the standard json.Unmarshal fails. The function then falls
// back to a lenient decoder that reads element-by-element and stops at
// the first incomplete element — salvaging all complete items before the
// truncation point.
func parseExtractionItems(raw string) ([]ExtractionItem, error) {
	content := strings.TrimSpace(raw)
	if content == "" {
		return nil, nil
	}
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	// Find the JSON array if the model wrapped it in prose. When the
	// output is truncated there may be no closing "]" — take from "[" to
	// end so the lenient decoder can salvage complete elements.
	if i := strings.Index(content, "["); i >= 0 {
		if j := strings.LastIndex(content, "]"); j > i {
			content = content[i : j+1]
		} else {
			content = content[i:]
		}
	}
	// Fast path: standard unmarshal works for well-formed JSON.
	var items []ExtractionItem
	if err := json.Unmarshal([]byte(content), &items); err == nil {
		return items, nil
	}
	// Fallback: lenient decoder salvages complete items from truncated JSON.
	items, err := parseExtractionItemsLenient(content)
	if err != nil {
		return nil, fmt.Errorf("parse extraction output: %w", err)
	}
	return items, nil
}

// parseExtractionItemsLenient reads a JSON array element-by-element using
// json.Decoder. When the input is truncated mid-element, Decode fails and
// the loop breaks — all items decoded before the truncation are returned.
// Returns an error when zero items are recovered (nothing useful).
func parseExtractionItemsLenient(content string) ([]ExtractionItem, error) {
	dec := json.NewDecoder(strings.NewReader(content))
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("expected '[': %w", err)
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '[' {
		return nil, fmt.Errorf("expected '[' got %v", tok)
	}
	var items []ExtractionItem
	for dec.More() {
		var item ExtractionItem
		if err := dec.Decode(&item); err != nil {
			break
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no complete items parsed")
	}
	return items, nil
}

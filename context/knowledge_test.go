package context

import (
	"context"
	"fmt"
	"strings"
	"testing"

	openagent "github.com/yusheng-g/openagent-go"
)

// fakeProvider is an in-memory MemoryProvider test double.
type fakeProvider struct {
	items []MemoryItem
}

func (f *fakeProvider) Recall(_ context.Context, scope ContextScope, query string, limit int) ([]MemoryEntry, error) {
	var out []MemoryEntry
	for _, it := range f.items {
		if query == "" || matchesQuery(it.Content, query) {
			out = append(out, MemoryEntry{Kind: MemoryKind(it.Kind), Content: it.Content, Score: 1.0})
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

// matchesQuery reports whether any query word appears in content.
func matchesQuery(content, query string) bool {
	for _, w := range strings.Fields(query) {
		if strings.Contains(content, w) {
			return true
		}
	}
	return false
}

func (f *fakeProvider) Store(_ context.Context, _ ContextScope, item MemoryItem) error {
	f.items = append(f.items, item)
	return nil
}

// TestKnowledgeLoop_RecallBuild verifies the recall half of the
// self-evolution chain: stored knowledge → next session's Build recalls
// and injects it into the AgentContext. (The extract half is the LLM
// extractor — covered end-to-end by the smoke test.)
func TestKnowledgeLoop_RecallBuild(t *testing.T) {
	prov := &fakeProvider{}
	prov.items = append(prov.items, MemoryItem{
		Kind:    "preference",
		Content: "I prefer terraform for infrastructure.",
	})

	rt := NewContextRuntime(Config{MemoryProvider: prov})
	ac, err := rt.Build(context.Background(), BuildRequest{
		Session:    openagent.Session{ID: "s2", UserID: "u1"},
		Goal:       "deploy with terraform",
		WorkingSet: []openagent.Message{openagent.UserMessage("deploy with terraform")},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(ac.Memories) == 0 {
		t.Fatal("expected recalled knowledge in AgentContext")
	}
	found := false
	for _, m := range ac.Memories {
		if strings.Contains(m.Content, "terraform") {
			found = true
		}
	}
	if !found {
		t.Fatalf("recalled memories missing the terraform preference: %+v", ac.Memories)
	}
}

// TestParseExtractionItems verifies the LLM output parser tolerates
// markdown fences and prose-wrapped JSON.
func TestParseExtractionItems(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{`[{"op":"add","kind":"preference","content":"prefers terraform","topic":"deployment"}]`, 1},
		{"```json\n[{\"op\":\"add\",\"kind\":\"fact\",\"content\":\"uses huawei cloud\",\"topic\":\"cloud\"}]\n```", 1},
		{`Here are the memories: [{"op":"skip","kind":"fact","content":"x","topic":"y"}]`, 1},
		{`not json`, 0},
	}
	for _, c := range cases {
		items, err := parseExtractionItems(c.raw)
		if c.want == 0 {
			if err == nil {
				t.Fatalf("expected error for %q", c.raw)
			}
			continue
		}
		if err != nil {
			t.Fatalf("parse %q: %v", c.raw, err)
		}
		if len(items) != c.want {
			t.Fatalf("parse %q: got %d items, want %d", c.raw, len(items), c.want)
		}
	}
}

// TestExtractor_DisabledNoOp verifies nil extractors are no-ops.
func TestExtractor_DisabledNoOp(t *testing.T) {
	prov := &fakeProvider{}
	ext := NewLLMExtractor(nil, prov) // nil model → disabled
	ext.Extract(context.Background(), ContextScope{}, []openagent.Message{
		openagent.UserMessage("I prefer nginx."),
	})
	if len(prov.items) != 0 {
		t.Fatal("disabled extractor wrote to provider")
	}
}

// fakeStore is a SessionStore test double.
type fakeStore struct {
	onAppend func(sid string, m openagent.Message)
}

func (f *fakeStore) Append(_ context.Context, sid string, m openagent.Message) error {
	if f.onAppend != nil {
		f.onAppend(sid, m)
	}
	return nil
}
func (f *fakeStore) Recent(context.Context, string, int, int) ([]openagent.Message, error) {
	return nil, nil
}
func (f *fakeStore) Count(context.Context, string) (int, error)  { return 0, nil }
func (f *fakeStore) DeleteSession(context.Context, string) error { return nil }

// mockModel returns canned responses in order, for testing retry logic.
type mockModel struct {
	responses []string
	idx       int
}

func (m *mockModel) ChatCompletion(_ context.Context, _ openagent.ChatCompletionRequest) (*openagent.ChatCompletionResponse, error) {
	if m.idx >= len(m.responses) {
		return nil, fmt.Errorf("mock: no more responses (idx=%d)", m.idx)
	}
	resp := &openagent.ChatCompletionResponse{
		Choices: []openagent.Choice{
			{Message: openagent.Message{Content: m.responses[m.idx]}},
		},
	}
	m.idx++
	return resp, nil
}

func (m *mockModel) ChatCompletionStream(_ context.Context, _ openagent.ChatCompletionRequest) (openagent.StreamReader, error) {
	return nil, nil
}

func (m *mockModel) ContextWindow() int { return 128000 }

// TestParseExtractionItems_Truncated verifies the lenient decoder salvages
// complete items from truncated JSON (finish_reason: "length" scenarios).
func TestParseExtractionItems_Truncated(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    int
		wantErr bool
	}{
		{
			name: "one complete + one truncated",
			raw:  `[{"op":"add","kind":"fact","content":"complete item here","topic":"t1"},{"op":"add","kind":"trunca`,
			want: 1,
		},
		{
			name:    "only truncated element",
			raw:     `[{"op":"add","kind":"fact","content":"trunca`,
			wantErr: true,
		},
		{
			name:    "non-JSON Chinese text",
			raw:     "关于这个问题，我没有相关信息",
			wantErr: true,
		},
		{
			name: "empty array",
			raw:  `[]`,
			want: 0,
		},
		{
			name: "two complete + truncated third",
			raw:  `[{"op":"add","kind":"fact","content":"first","topic":"a"},{"op":"add","kind":"lesson","content":"second","topic":"b"},{"op":"add","kind":"fact","content":"trun`,
			want: 2,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			items, err := parseExtractionItems(c.raw)
			if c.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(items) != c.want {
				t.Fatalf("got %d items, want %d", len(items), c.want)
			}
		})
	}
}

// TestExtractor_RetryOnParseFailure verifies the corrective retry: when
// the first response is non-JSON (e.g. a Chinese refusal), the extractor
// retries with a format correction and stores items from the retry.
func TestExtractor_RetryOnParseFailure(t *testing.T) {
	prov := &fakeProvider{}
	m := &mockModel{
		responses: []string{
			"关于这个问题，我没有相关信息",
			`[{"op":"add","kind":"fact","content":"user prefers terraform for deployment","topic":"deploy"}]`,
		},
	}
	ext := NewLLMExtractor(func() openagent.Model { return m }, prov)
	ext.Extract(context.Background(), ContextScope{}, []openagent.Message{
		openagent.UserMessage("I prefer terraform for deployment."),
	})
	if len(prov.items) != 1 {
		t.Fatalf("expected 1 stored item after retry, got %d", len(prov.items))
	}
	if m.idx != 2 {
		t.Fatalf("expected 2 model calls (initial + retry), got %d", m.idx)
	}
}

// TestExtractor_NoRetryOnValidJSON verifies no retry when the first
// response parses successfully.
func TestExtractor_NoRetryOnValidJSON(t *testing.T) {
	prov := &fakeProvider{}
	m := &mockModel{
		responses: []string{
			`[{"op":"add","kind":"fact","content":"user likes vim","topic":"editor"}]`,
		},
	}
	ext := NewLLMExtractor(func() openagent.Model { return m }, prov)
	ext.Extract(context.Background(), ContextScope{}, []openagent.Message{
		openagent.UserMessage("I use vim."),
	})
	if len(prov.items) != 1 {
		t.Fatalf("expected 1 stored item, got %d", len(prov.items))
	}
	if m.idx != 1 {
		t.Fatalf("expected 1 model call (no retry), got %d", m.idx)
	}
}

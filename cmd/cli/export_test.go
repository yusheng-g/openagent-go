package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/session"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello world", "hello world"},
		{"a/b\\c:d*e?f<g>h|i", "a_b_c_d_e_f_g_h_i"},
		{"double__underscore", "double_underscore"},
		{"  trim  ", "trim"},
		{"_leading_trailing_", "leading_trailing"},
		{"", ""},
		{"正常中文标题", "正常中文标题"},
		{"line1\nline2", "line1_line2"},
		{"tab\there", "tab_here"},
		{"carriage\rreturn", "carriage_return"},
		{"trailing.dot.", "trailing.dot"},
		{".leading.dot", "leading.dot"},
	}

	for _, tt := range tests {
		got := sanitizeFilename(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeFilenameTruncation(t *testing.T) {
	long := strings.Repeat("a", 150)
	got := sanitizeFilename(long)
	if utf8.RuneCountInString(got) > maxFilenameLen {
		t.Errorf("rune count = %d, want <= %d", utf8.RuneCountInString(got), maxFilenameLen)
	}
}

func TestSanitizeFilenameChineseTruncation(t *testing.T) {
	long := strings.Repeat("你好世界", 30)
	got := sanitizeFilename(long)
	if !utf8.ValidString(got) {
		t.Errorf("result is not valid UTF-8: %q", got)
	}
	if utf8.RuneCountInString(got) > maxFilenameLen {
		t.Errorf("rune count = %d, want <= %d", utf8.RuneCountInString(got), maxFilenameLen)
	}
}

func TestTimestampName(t *testing.T) {
	ts := time.Date(2026, 9, 18, 14, 30, 0, 0, time.UTC)
	got := timestampName(ts)
	want := "20260918-143000"
	if got != want {
		t.Errorf("timestampName() = %q, want %q", got, want)
	}
}

func TestTimestampNameZero(t *testing.T) {
	got := timestampName(time.Time{})
	if got == "" {
		t.Error("timestampName(zero) should not be empty")
	}
}

func TestBuildFilename(t *testing.T) {
	info := session.SessionInfo{
		Title:     "My Chat",
		CreatedAt: time.Date(2026, 9, 18, 14, 30, 0, 0, time.UTC),
	}
	got := buildFilename(info, ".json")
	want := "20260918-143000_My Chat.json"
	if got != want {
		t.Errorf("buildFilename() = %q, want %q", got, want)
	}
}

func TestBuildFilenameEmptyTitle(t *testing.T) {
	info := session.SessionInfo{
		CreatedAt: time.Date(2026, 9, 18, 14, 30, 0, 0, time.UTC),
	}
	got := buildFilename(info, ".md")
	want := "20260918-143000_untitled.md"
	if got != want {
		t.Errorf("buildFilename() = %q, want %q", got, want)
	}
}

func TestRenderJSON(t *testing.T) {
	info := session.SessionInfo{
		ID:        "sess-abc",
		Title:     "Test",
		CreatedAt: time.Date(2026, 9, 18, 14, 30, 0, 0, time.UTC),
	}
	messages := []openagent.Message{
		openagent.UserMessage("hello"),
		{Role: openagent.RoleAssistant, Content: "hi there"},
	}

	got, err := renderJSON(info, messages, nil)
	if err != nil {
		t.Fatalf("renderJSON: %v", err)
	}

	var payload exportPayload
	if err := json.Unmarshal([]byte(got), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.Session.ID != "sess-abc" {
		t.Errorf("session ID = %q", payload.Session.ID)
	}
	if len(payload.Messages) != 2 {
		t.Errorf("messages len = %d, want 2", len(payload.Messages))
	}
	if payload.Compressed != nil {
		t.Error("compressed should be nil")
	}
}

func TestRenderJSONWithCompressed(t *testing.T) {
	info := session.SessionInfo{ID: "sess-1"}
	messages := []openagent.Message{openagent.UserMessage("test")}
	cc := &openagent.CompressedContext{Summary: "prior context", ThroughIndex: 3}

	got, err := renderJSON(info, messages, cc)
	if err != nil {
		t.Fatalf("renderJSON: %v", err)
	}

	var payload exportPayload
	if err := json.Unmarshal([]byte(got), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.Compressed == nil {
		t.Fatal("compressed should not be nil")
	}
	if payload.Compressed.Summary != "prior context" {
		t.Errorf("summary = %q", payload.Compressed.Summary)
	}
	if payload.Compressed.ThroughIndex != 3 {
		t.Errorf("throughIndex = %d", payload.Compressed.ThroughIndex)
	}
}

func TestRenderMarkdown(t *testing.T) {
	info := session.SessionInfo{
		ID:        "sess-abc",
		Title:     "Test Session",
		CreatedAt: time.Date(2026, 9, 18, 14, 30, 0, 0, time.UTC),
	}
	messages := []openagent.Message{
		openagent.UserMessage("hello"),
		{Role: openagent.RoleAssistant, Content: "hi there"},
	}

	got := renderMarkdown(info, messages, nil)

	if !strings.Contains(got, "# Chat Transcript") {
		t.Error("missing title heading")
	}
	if !strings.Contains(got, "sess-abc") {
		t.Error("missing session ID")
	}
	if !strings.Contains(got, "Test Session") {
		t.Error("missing session title")
	}
	if !strings.Contains(got, "## 1. User") {
		t.Error("missing user heading")
	}
	if !strings.Contains(got, "hello") {
		t.Error("missing user content")
	}
	if !strings.Contains(got, "## 2. Assistant") {
		t.Error("missing assistant heading")
	}
	if !strings.Contains(got, "hi there") {
		t.Error("missing assistant content")
	}
}

func TestRenderMarkdownWithCompressed(t *testing.T) {
	info := session.SessionInfo{ID: "sess-1"}
	messages := []openagent.Message{openagent.UserMessage("test")}
	cc := &openagent.CompressedContext{Summary: "prior conversation summary", ThroughIndex: 3}

	got := renderMarkdown(info, messages, cc)

	if !strings.Contains(got, "## Summary") {
		t.Error("missing summary heading")
	}
	if !strings.Contains(got, "prior conversation summary") {
		t.Error("missing summary text")
	}
}

func TestRenderMarkdownWithToolCalls(t *testing.T) {
	info := session.SessionInfo{ID: "sess-1"}
	messages := []openagent.Message{
		openagent.UserMessage("run something"),
		{
			Role:    openagent.RoleAssistant,
			Content: "I'll run a tool",
			ToolCalls: []openagent.ToolCall{
				{ID: "tc1", Type: "function", Function: openagent.ToolCallFunction{Name: "read_file", Arguments: `{"path":"/tmp/foo"}`}},
			},
			Result: &openagent.ToolResult{Content: "file contents here"},
		},
	}

	got := renderMarkdown(info, messages, nil)

	if !strings.Contains(got, "Tool: read_file") {
		t.Error("missing tool name")
	}
	if !strings.Contains(got, `{"path":"/tmp/foo"}`) {
		t.Error("missing tool arguments")
	}
	if !strings.Contains(got, "file contents here") {
		t.Error("missing tool result")
	}
}

func TestRenderMarkdownWithReasoning(t *testing.T) {
	info := session.SessionInfo{ID: "sess-1"}
	messages := []openagent.Message{
		openagent.UserMessage("think"),
		{
			Role:             openagent.RoleAssistant,
			Content:          "answer",
			ReasoningContent: "let me think...",
		},
	}

	got := renderMarkdown(info, messages, nil)

	if !strings.Contains(got, "Thought") {
		t.Error("missing thought heading")
	}
	if !strings.Contains(got, "let me think...") {
		t.Error("missing reasoning content")
	}
}

func TestRenderMarkdownEmptyMessages(t *testing.T) {
	info := session.SessionInfo{ID: "sess-empty"}
	got := renderMarkdown(info, nil, nil)

	if !strings.Contains(got, "# Chat Transcript") {
		t.Error("missing title heading")
	}
	if strings.Contains(got, "## 1.") {
		t.Error("should not have message sections when empty")
	}
}

func TestIsOfflineCmd(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"openagent", "keyring", "set", "k", "v"}, true},
		{[]string{"openagent", "export"}, true},
		{[]string{"openagent", "export", "-f", "markdown"}, true},
		{[]string{"openagent", "export", "--output", "/tmp"}, true},
		{[]string{"openagent", "-q", "export"}, true},
		{[]string{"openagent", "serve"}, false},
		{[]string{"openagent", "run", "hello"}, false},
		{[]string{"openagent"}, false},
	}

	for _, tt := range tests {
		got := isOfflineCmd(tt.args)
		if got != tt.want {
			t.Errorf("isOfflineCmd(%v) = %v, want %v", tt.args, got, tt.want)
		}
	}
}

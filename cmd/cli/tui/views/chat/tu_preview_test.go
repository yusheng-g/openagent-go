package chat

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestTUPreview is a headless preview harness used by tu acceptance: it
// prints the rendered chat page with one message of every role so the
// per-role card styling can be inspected in a virtual terminal. It is
// skipped unless TU_PREVIEW=1, so it never runs in normal test passes.
//
//	TU_PREVIEW=1 go test ./cmd/cli/tui/views/chat/ -run TestTUPreview -v
func TestTUPreview(t *testing.T) {
	if os.Getenv("TU_PREVIEW") != "1" {
		t.Skip("preview harness; set TU_PREVIEW=1")
	}
	m := newTestModel()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.inChat = true
	m.needAutoScroll = false
	m.messages = []ChatMessage{
		{Role: "user", Content: "帮我看看这个项目的结构", TurnId: 1},
		{Role: "thought", Content: "让我先扫描一下仓库", TurnId: 1},
		{Role: "tool", ToolName: "bash", ToolStatus: toolDone, ToolInput: "ls -la", ToolOutput: "README.md  cmd/  go.mod\nMakefile  docs/", TurnId: 1},
		{Role: "assistant", Content: "这是一个 Go 项目,结构如下:\n\n- `cmd/` 命令入口\n- `docs/` 文档\n\n我建议先看 `cmd/cli/tui`。", TurnId: 1},
		{Role: "error", Content: "连接 ACP 服务超时(timeout after 30s)", TurnId: 1},
	}
	view := m.View().Content
	// Write the rendered screen to a file rather than stdout so a tu session
	// can display exactly the TUI frame without go-test log lines around it.
	if err := os.WriteFile("/tmp/tu_preview.ansi", []byte(view), 0o644); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(view, "┃") {
		t.Error("preview missing message rails")
	}
}

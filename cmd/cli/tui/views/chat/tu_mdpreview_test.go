package chat

import (
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestMDPreview is the markdown acceptance twin of TestTUPreview: it renders
// one assistant message with headings, inline code, a code fence, a list and
// a blockquote through the full view pipeline so tu can capture pixels and
// verify the backgrounds are uniform (no black or ANSI-256 patches inside
// the card). Skipped unless TU_PREVIEW=1.
//
//	TU_PREVIEW=1 go test ./cmd/cli/tui/views/chat/ -run TestMDPreview -v
func TestMDPreview(t *testing.T) {
	if os.Getenv("TU_PREVIEW") != "1" {
		t.Skip("preview harness; set TU_PREVIEW=1")
	}
	m := newTestModel()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.inChat = true
	m.needAutoScroll = false
	md := "## 项目结构\n\n主体是 `cmd/` 和 `docs/`,代码用 `main.go` 作为入口。\n\n```go\nfunc main() { fmt.Println(\"hi\") }\n```\n\n- 列表项一\n- 列表项二\n\n> 引用块内容"
	m.messages = []ChatMessage{{Role: "assistant", Content: md, TurnId: 1}}
	view := m.View().Content
	if err := os.WriteFile("/tmp/mdview.ansi", []byte(view), 0o644); err != nil {
		t.Fatal(err)
	}
}

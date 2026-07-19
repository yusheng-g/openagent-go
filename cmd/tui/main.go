// TUI chat client for openagent-go.
//
// Features:
//   - Streaming chat with real-time token output
//   - Tool call approval (Y/N) — human-in-the-loop
//   - Built-in calculator + echo tools
//
// Requires: bubbletea v2, bubbles v2, lipgloss v2
//
// Usage:
//
//	OPENAGENT_API_KEY=sk-xxx OPENAGENT_MODEL=gpt-4o go run ./cmd/tui/
package main

import (
	"fmt"
	"os"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/model/openai"
	tea "charm.land/bubbletea/v2"
)

func main() {
	apiKey := os.Getenv("OPENAGENT_API_KEY")
	modelID := os.Getenv("OPENAGENT_MODEL")
	baseURL := os.Getenv("OPENAGENT_BASE_URL")

	if apiKey == "" || modelID == "" {
		fmt.Fprintln(os.Stderr, "set OPENAGENT_API_KEY and OPENAGENT_MODEL")
		os.Exit(1)
	}

	llm := openai.New(apiKey, modelID, baseURL).WithContextWindow(128_000)

	approveCh := make(chan approveRequest, 8)

	agent := openagent.NewAgent("assistant",
		openagent.WithModel(llm),
		openagent.WithSystemPrompts("You are a capable assistant. Use tools when needed. Be concise and action-oriented."),
		openagent.WithTools(&calculatorTool{}, &echoTool{}),
		openagent.WithApprover(&TUIApprover{requests: approveCh}),
		openagent.WithMaxTurns(20),
	)

	session := openagent.Session{
		ID:        fmt.Sprintf("tui-%d", time.Now().Unix()),
		UserID:    "user",

		ModelID:   modelID,
		CreatedAt: time.Now(),
	}

	m := newModel(agent, session, approveCh)
	p := tea.NewProgram(&m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}

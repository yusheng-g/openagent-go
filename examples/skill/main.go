// Skill example: demonstrates automatic skill discovery and on-demand loading.
//
// Environment variables:
//
//	OPENAGENT_BASE_URL   — API base URL
//	OPENAGENT_MODEL      — model ID
//	OPENAGENT_API_KEY    — API key
//
//	go run ./examples/skill/
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/model/openai"
	"github.com/yusheng-g/openagent-go/skill/fs"
)

func main() {
	apiKey := os.Getenv("OPENAGENT_API_KEY")
	modelID := os.Getenv("OPENAGENT_MODEL")
	baseURL := os.Getenv("OPENAGENT_BASE_URL")

	model := openai.New(apiKey, modelID, baseURL).
		WithContextWindow(128_000)

	// Skill loader: skills directory is next to the binary
	skillRoot, _ := filepath.Abs("examples/skill/skills")
	loader := fs.New(skillRoot)

	agent := openagent.NewAgent("skill-demo",
		openagent.WithModel(model),
		openagent.WithSystemPrompts("You are a helpful assistant. Skills are available for loading — use use_skill to load one when you need detailed instructions."),
		openagent.WithSkillLoader(loader),
	)

	session := openagent.Session{
		ID:        "skill-demo-1",
		UserID:    "user-1",

		ModelID:   modelID,
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	result, err := agent.Run(ctx, session, openagent.UserMessage("请加载 example-skill 并按它的要求执行"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Run Result ===")
	fmt.Printf("Final output: %s\n", result.FinalOutput)
	fmt.Printf("Turns: %d\n", result.TurnCount)
	fmt.Printf("Tokens: prompt=%d completion=%d total=%d\n",
		result.Usage.PromptTokens, result.Usage.CompletionTokens, result.Usage.TotalTokens)
	fmt.Println("\n=== Messages ===")
	for i, msg := range result.Messages {
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				fmt.Printf("[%d] %s → tool_call: %s(%s)\n", i, msg.Role, tc.Function.Name, tc.Function.Arguments)
			}
		} else if msg.Role == openagent.RoleSystem {
			fmt.Printf("[%d] %s: %s\n", i, msg.Role, truncate(msg.Content, 120))
		} else {
			fmt.Printf("[%d] %s: %s\n", i, msg.Role, truncate(msg.Content, 200))
		}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

package server

import (
	"context"
	"fmt"
	"os"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/cmd/cli/config"
	"github.com/yusheng-g/openagent-go/sandbox/native"
)

// RunCLI runs a one-shot chat turn with streaming output to stdout.
// It creates a lightweight agent (model + system prompts + standard tools),
// sends the message, and streams text deltas to stdout in real time.
func RunCLI(ctx context.Context, cfg *config.Config, message string) error {
	// 1. Build model from config (unexported: buildModels, firstModel)
	models, _ := buildModels(cfg.Provider)
	m := firstModel(models)
	if m == nil {
		return fmt.Errorf("no models configured. Please add a provider in ~/.openagent/settings.json")
	}

	// 2. System prompts (unexported: resolveProfiles)
	prompts := resolveProfiles(cfg.Profiles)

	// 3. Sandbox + standard tools (unexported: sandboxPolicy, buildTools)
	workDir, _ := os.Getwd()
	policy := sandboxPolicy(cfg.Sandbox)
	sb, err := native.NewWithPolicy(workDir, policy)
	var tools []openagent.Tool
	if err == nil {
		tools = buildTools(sb, workDir, []string{"shell", "read", "write", "ls", "grep"})
	} else {
		fmt.Fprintf(os.Stderr, "sandbox unavailable, tools disabled: %v\n", err)
	}

	// 4. Construct agent
	opts := []openagent.AgentOption{
		openagent.WithModel(m),
		openagent.WithSystemPrompts(prompts...),
		openagent.WithMaxTurns(50),
	}
	if len(tools) > 0 {
		opts = append(opts, openagent.WithTools(tools...))
	}
	agent := openagent.NewAgent("openagent", opts...)

	// 5. Temporary session (one-shot, no persistence)
	session := openagent.Session{
		ID:        fmt.Sprintf("cli-%d", time.Now().UnixNano()),
		CreatedAt: time.Now(),
	}

	// 6. Run and stream events to terminal
	ch := agent.RunStream(ctx, session, openagent.UserMessage(message))
	for evt := range ch {
		switch evt.Type {
		case openagent.StreamTextDelta:
			fmt.Print(evt.Text)

		case openagent.StreamDone:
			fmt.Println()
			if evt.Result != nil {
				u := evt.Result.Usage
				fmt.Fprintf(os.Stderr, "─── %d prompt + %d completion = %d tokens, %d turns\n",
					u.PromptTokens, u.CompletionTokens, u.TotalTokens,
					evt.Result.TurnCount)
			}

		case openagent.StreamError:
			return evt.Error

		case openagent.StreamAborted:
			if evt.Error != nil {
				return evt.Error
			}
			return ctx.Err()
		}
	}
	return nil
}

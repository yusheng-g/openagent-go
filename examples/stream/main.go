// Stream example: streaming Agent with real-time output via RunStream.
// Demonstrates text_delta, tool_call, tool_result, retrying, and done events.
//
// Environment variables:
//
//	OPENAGENT_BASE_URL   — API base URL
//	OPENAGENT_MODEL      — model ID
//	OPENAGENT_API_KEY    — API key
//
//	go run ./examples/stream/
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/model/openai"
)

func main() {
	apiKey := os.Getenv("OPENAGENT_API_KEY")
	modelID := os.Getenv("OPENAGENT_MODEL")
	baseURL := os.Getenv("OPENAGENT_BASE_URL")

	model := openai.New(apiKey, modelID, baseURL).
		WithContextWindow(128_000)

	agent := openagent.NewAgent("calculator",
		openagent.WithModel(model),
		openagent.WithSystemPrompts("You are a precise calculator. Use the calculator tool for arithmetic. Answer concisely."),
		openagent.WithTools(&calculatorTool{}),
	)

	session := openagent.Session{
		ID:        "stream-session-1",
		UserID:    "user-1",

		ModelID:   modelID,
		CreatedAt: time.Now(),
	}

	fmt.Printf("Model: %s | Base URL: %s\n\n", modelID, baseURL)

	ctx := context.Background()
	events := agent.RunStream(ctx, session, openagent.UserMessage("what is 12 + 34?"))

	var inThought bool
	fmt.Print("Assistant: ")
	for event := range events {
		switch event.Type {
		case openagent.StreamThought:
			if !inThought {
				fmt.Println()
				fmt.Print("🧠 Thinking: ")
				inThought = true
			}
			fmt.Print(event.Text)

		case openagent.StreamTextDelta:
			if inThought {
				fmt.Println()
				fmt.Print("Assistant: ")
				inThought = false
			}
			fmt.Print(event.Text) // real-time character output

		case openagent.StreamToolCall:
			for _, tc := range event.Message.ToolCalls {
				fmt.Printf("\n🔧 calling %s(%s)...\n", tc.Function.Name, tc.Function.Arguments)
			}

		case openagent.StreamToolResult:
			fmt.Printf("📦 %s\n", truncate(event.Message.Content, 120))
			fmt.Print("Assistant: ")

		case openagent.StreamRetrying:
			fmt.Printf("\n⏳ retrying: %v\n", event.Error)
			fmt.Print("Assistant: ")

		case openagent.StreamDone:
			r := event.Result
			fmt.Printf("\n\n=== Done ===\n")
			fmt.Printf("Turns: %d | Tokens: prompt=%d completion=%d total=%d\n",
				r.TurnCount, r.Usage.PromptTokens, r.Usage.CompletionTokens, r.Usage.TotalTokens)

		case openagent.StreamError:
			fmt.Fprintf(os.Stderr, "\nERROR: %v\n", event.Error)
			os.Exit(1)
		}
	}
}

// ── Calculator Tool ──

type calculatorTool struct{}

func (t *calculatorTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "calculator",
		Description: "Evaluate a mathematical expression. Input is a valid arithmetic expression like '12+34' or '100/3'.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"expression": {"type": "string", "description": "The math expression to evaluate"}
			},
			"required": ["expression"]
		}`),
	}
}

func (t *calculatorTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Expression string `json:"expression"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}
	// Remove spaces — model may produce "12 + 34" or "12+34"
	expr := strings.ReplaceAll(params.Expression, " ", "")
	var a, b int
	var op rune
	fmt.Sscanf(expr, "%d%c%d", &a, &op, &b)
	switch op {
	case '+':
		return fmt.Sprintf("%d", a+b), nil
	case '-':
		return fmt.Sprintf("%d", a-b), nil
	case '*':
		return fmt.Sprintf("%d", a*b), nil
	case '/':
		if b == 0 {
			return "", fmt.Errorf("division by zero")
		}
		return fmt.Sprintf("%d", a/b), nil
	default:
		return "", fmt.Errorf("unsupported operator: %c", op)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

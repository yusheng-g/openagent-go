// Hooks example: demonstrates RunHooks with log/slog for structured observability.
//
//	go run ./examples/hooks/
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	sloghooks "github.com/yusheng-g/openagent-go/hooks/slog"
	"github.com/yusheng-g/openagent-go/model/openai"
)

func main() {
	apiKey := os.Getenv("OPENAGENT_API_KEY")
	modelID := os.Getenv("OPENAGENT_MODEL")
	baseURL := os.Getenv("OPENAGENT_BASE_URL")

	model := openai.New(apiKey, modelID, baseURL).
		WithContextWindow(128_000)

	// ── RunHooks: structured logging via log/slog ──
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	hooks := sloghooks.New(logger)

	agent := openagent.NewAgent("calculator",
		openagent.WithModel(model),
		openagent.WithSystemPrompts("You are a precise calculator. Use the calculator tool for arithmetic. Answer concisely."),
		openagent.WithTools(&calculatorTool{}),
		openagent.WithRunHooks(hooks),
	)

	session := openagent.Session{
		ID:        "hooks-session-1",
		UserID:    "user-1",

		ModelID:   modelID,
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	result, err := agent.Run(ctx, session, openagent.UserMessage("what is 12 + 34?"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Run Result ===")
	fmt.Printf("Final output: %s\n", result.FinalOutput)
	fmt.Printf("Turns: %d\n", result.TurnCount)
	fmt.Printf("Tokens: prompt=%d completion=%d total=%d\n",
		result.Usage.PromptTokens, result.Usage.CompletionTokens, result.Usage.TotalTokens)
}

// ── Calculator Tool ──

type calculatorTool struct{}

func (t *calculatorTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "calculator",
		Description: "Evaluate a mathematical expression like '12+34' or '100/3'.",
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

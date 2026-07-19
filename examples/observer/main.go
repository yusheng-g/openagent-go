// Observer example: demonstrates RunObserver for stage-level observability.
//
//	go run ./examples/observer/
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

	observer := &stageObserver{}

	agent := openagent.NewAgent("calculator",
		openagent.WithModel(model),
		openagent.WithSystemPrompts("You are a precise calculator. Use the calculator tool for arithmetic. Answer concisely."),
		openagent.WithTools(&calculatorTool{}),
		openagent.WithRunObserver(observer),
	)

	session := openagent.Session{
		ID:        "observer-session-1",
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

// ── StageObserver ──

type stageObserver struct{}

func (o *stageObserver) ObserveStage(ctx context.Context, event openagent.StageEvent) {
	switch event.Phase {
	case "enter":
		fmt.Printf("[%s] → entering\n", event.Name)
	case "leave":
		status := "ok"
		if event.Err != nil {
			status = event.Err.Error()
		}
		fmt.Printf("[%s] ← leaving (%v) %s\n", event.Name, event.Duration.Round(time.Microsecond), status)
	}
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

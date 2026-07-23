// Plugin example: demonstrates the WASM plugin system.
//
// Build plugins first (requires Rust + wasm32-unknown-unknown target):
//
//	make -C examples/plugin
//
// Then run:
//
//	go run ./examples/plugin/
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
	"github.com/yusheng-g/openagent-go/plugin/agent/wasm"
	"github.com/yusheng-g/openagent-go/plugin/wasmhost"
)

type stdLogger struct{}

func (l *stdLogger) Info(msg string)  { fmt.Println("[plugin]", msg) }
func (l *stdLogger) Warn(msg string)  { fmt.Println("[plugin] WARN:", msg) }
func (l *stdLogger) Error(msg string) { fmt.Println("[plugin] ERROR:", msg) }

func main() {
	apiKey := os.Getenv("OPENAGENT_API_KEY")
	modelID := os.Getenv("OPENAGENT_MODEL")
	baseURL := os.Getenv("OPENAGENT_BASE_URL")

	model := openai.New(apiKey, modelID, baseURL).
		WithContextWindow(128_000)

	// Plugin manager with host API so plugins can use log_* and keyring_*.
	hostAPI := &wasmhost.HostAPI{Logger: &stdLogger{}}
	mgr := wasm.NewManager("./plugins").WithHostAPI(hostAPI)
	if err := mgr.Discover(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Plugin discover error: %v\n", err)
		os.Exit(1)
	}
	defer mgr.Close()

	// Loaded tools (from .wasm plugins + built-in)
	tools := []openagent.Tool{&calculatorTool{}}
	tools = append(tools, mgr.Tools()...)

	fmt.Printf("Loaded %d tool plugin(s)\n", len(mgr.Tools()))

	// Observer: wrap stage plugin with a logger so we can see it fire.
	var observer openagent.RunObserver
	if raw := mgr.Observer(); raw != nil {
		fmt.Println("Stage plugins loaded")
		observer = raw
	} else {
		fmt.Println("No stage plugins")
	}

	agent := openagent.NewAgent("assistant",
		openagent.WithModel(model),
		openagent.WithSystemPrompts("You are a precise assistant. Use echo for testing tool calls, calculator for math. Be concise."),
		openagent.WithTools(tools...),
		openagent.WithRunObserver(observer),
	)

	ctx := context.Background()
	session := openagent.Session{
		ID:     "plugin-session-1",
		UserID: "user-1",

		ModelID:   modelID,
		CreatedAt: time.Now(),
	}

	fmt.Println("\n=== Running agent ===")
	result, err := agent.Run(ctx, session, openagent.UserMessage("Use the echo tool to echo 'hello plugin', then calculate 15+27"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Output: %s\n", result.FinalOutput)
	fmt.Printf("Turns: %d, Tokens: %d\n", result.TurnCount, result.Usage.TotalTokens)
}

// Calculator Tool (built-in, always available)

type calculatorTool struct{}

func (t *calculatorTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "calculator",
		Description: "Evaluate a math expression like '15+27' or '100/3'.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"expression": {"type": "string", "description": "The math expression to evaluate"}
			},
			"required": ["expression"]
		}`),
	}
}

func (t *calculatorTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Expression string `json:"expression"`
	}
	json.Unmarshal(args, &params)

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

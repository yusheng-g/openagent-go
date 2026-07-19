// MCP Client example — demo the iac-mcp server as an MCP client.
//
// This example spawns iac-mcp as a subprocess (like Claude Desktop does),
// imports its 6 IaC tools via MCP, and runs a demo pipeline:
//
//	intent_parse → architecture_design → generate_terraform → review_plan
//
//	go run ./examples/mcp-client/
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/mcp"
	"github.com/yusheng-g/openagent-go/model/openai"
)

type progressObserver struct{}

func (o *progressObserver) ObserveStage(ctx context.Context, event openagent.StageEvent) {
	if event.Name == openagent.StageToolExecute && event.Phase == "enter" {
		name, _ := event.Detail["tool"].(string)
		fmt.Printf("  → running %s...\n", name)
	}
	if event.Name == openagent.StageToolExecute && event.Phase == "leave" {
		fmt.Printf("    done (%.1fs)\n", event.Duration.Seconds())
	}
}

var _ openagent.RunObserver = (*progressObserver)(nil)

func main() {
	apiKey := os.Getenv("OPENAGENT_API_KEY")
	modelID := os.Getenv("OPENAGENT_MODEL")
	baseURL := os.Getenv("OPENAGENT_BASE_URL")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "OPENAGENT_API_KEY not set")
		os.Exit(1)
	}
	if modelID == "" {
		modelID = "claude-sonnet-5"
	}
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// ── Start iac-mcp as a subprocess ──
	// Same as Claude Desktop: spawn the binary over stdio.
	fmt.Println("Starting iac-mcp...")
	iacBin := os.Getenv("IAC_MCP_BIN")
	if iacBin == "" {
		iacBin = "./iac-mcp" // from go install or PATH
	}

	client := mcp.NewClient("mcp-client-demo", "1.0.0")
	session, err := client.ConnectStdio(ctx, iacBin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start iac-mcp (%s): %v\n", iacBin, err)
		fmt.Fprintln(os.Stderr, "\nBuild it first:  go build -o /tmp/iac-mcp ./cmd/iac-mcp/ && IAC_MCP_BIN=/tmp/iac-mcp go run ./examples/mcp-client/")
		os.Exit(1)
	}
	defer session.Close()

	tools, err := session.Tools(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list MCP tools: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d IaC tools via MCP:\n", len(tools))
	for _, t := range tools {
		fmt.Printf("  - %s: %s\n", t.Definition().Name, truncate(t.Definition().Description, 80))
	}

	// Build a coordinator agent that uses these MCP tools.
	model := openai.New(apiKey, modelID, baseURL).WithContextWindow(128_000)

	// Progress observer so the user sees live output.
	prog := &progressObserver{}

	coordinator := openagent.NewAgent("iac-coordinator",
		openagent.WithModel(model),
		openagent.WithTools(tools...),
		openagent.WithRunObserver(prog),
		openagent.WithSystemPrompts(`You are an IaC orchestrator controlling a 6-step deployment pipeline. You MUST call each tool IN ORDER — one at a time — and use the actual output of each step as input to the next. Never simulate or invent results.

Step 1: Call iac_intent_parse with the user's goal as the task. WAIT for the JSON result.
Step 2: Call iac_architecture_design. The task MUST include the full JSON from step 1: "Based on this ApplicationProfile: <paste the JSON output from step 1>, design 3 architecture options."
Step 3: Pick the recommended plan (option B). Call iac_generate_terraform. The task MUST include the selected option: "Generate Terraform for this architecture option: <paste option B from step 2>."
Step 4: Call iac_review_plan. Task: "Review the generated Terraform plan."
Step 5: Summarize what happened — which resources were planned, estimated cost, any issues.
Step 6: Do NOT call iac_apply unless the user explicitly asks for it.`),
		openagent.WithMaxTurns(12),
	)

	sess := openagent.Session{
		ID: "iac-mcp-demo", UserID: "user-1",

		CreatedAt: time.Now(),
	}

	goal := os.Getenv("GOAL")
	if goal == "" {
		goal = "deploy a wordpress blog"
	}
	fmt.Printf("\nGoal: %s\n\n", goal)

	result, err := coordinator.Run(ctx, sess, openagent.UserMessage(goal))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n── Final output ──\n%s\n", result.FinalOutput)
	fmt.Printf("Tokens: %d turns, %d+%d=%d\n",
		result.TurnCount,
		result.Usage.PromptTokens, result.Usage.CompletionTokens, result.Usage.TotalTokens)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

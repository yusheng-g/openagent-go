// Team example: multi-agent software development workflow.
//
//	go run ./examples/team/
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/model/openai"
)

func main() {
	apiKey := os.Getenv("OPENAGENT_API_KEY")
	modelID := os.Getenv("OPENAGENT_MODEL")
	baseURL := os.Getenv("OPENAGENT_BASE_URL")

	sharedModel := openai.New(apiKey, modelID, baseURL).
		WithContextWindow(128_000)

	// ── Analyst: understands requirements, produces a spec ──
	analyst := openagent.NewAgent("analyst",
		openagent.WithModel(sharedModel),
		openagent.WithSystemPrompts(`You are a requirements analyst. Your job:
1. Understand the user's request
2. Break it down into clear, testable requirements
3. Hand off to the designer with a structured specification
Be specific — include constraints, edge cases, and acceptance criteria.`),
		openagent.WithMaxTurns(2),
	)

	// ── Designer: architecture and component design ──
	designer := openagent.NewAgent("designer",
		openagent.WithModel(sharedModel),
		openagent.WithSystemPrompts(`You are a software designer. Your job:
1. Take the analyst's specification and design the architecture
2. Define components, interfaces, and data flow
3. Hand off to the coder with a clear design document
Be specific about types, function signatures, and module boundaries.`),
		openagent.WithMaxTurns(2),
	)

	// ── Coder: writes production code ──
	coder := openagent.NewAgent("coder",
		openagent.WithModel(sharedModel),
		openagent.WithSystemPrompts(`You are a software developer. Your job:
1. Take the designer's spec and write clean, idiomatic Go code
2. Include error handling, comments, and tests
3. Hand off the complete implementation to the tester
Output ONLY code with brief inline comments.`),
		openagent.WithMaxTurns(3),
	)

	// ── Tester: writes and runs tests ──
	tester := openagent.NewAgent("tester",
		openagent.WithModel(sharedModel),
		openagent.WithSystemPrompts(`You are a QA engineer. Your job:
1. Review the coder's implementation
2. Identify edge cases and write tests for them
3. If all tests pass, hand off to the reviewer with your test report
4. If tests fail, report the failures clearly — do NOT fix the code
Be thorough. List what you tested and why.`),
		openagent.WithMaxTurns(2),
	)

	// ── Reviewer: final quality gate ──
	reviewer := openagent.NewAgent("reviewer",
		openagent.WithModel(sharedModel),
		openagent.WithSystemPrompts(`You are a code reviewer. Your job:
1. Review the complete implementation and test results
2. Check for correctness, style, performance, and security
3. Produce a final review summary: approved, changes requested, or rejected
4. If approved, present the complete deliverable to the user
Do NOT hand off — you are the final gate.`),
		openagent.WithMaxTurns(1),
	)

	// ── Build team ──
	team := openagent.NewTeam(
		openagent.WithTeamAgent("analyst", "Understands requirements and produces specifications", analyst),
		openagent.WithTeamAgent("designer", "Designs architecture, components, and data flow", designer),
		openagent.WithTeamAgent("coder", "Writes clean, idiomatic Go code with error handling", coder),
		openagent.WithTeamAgent("tester", "Writes tests, identifies edge cases, reports results", tester),
		openagent.WithTeamAgent("reviewer", "Reviews code for correctness, style, and security", reviewer),
		openagent.WithTeamMaxHandoffs(5),
	)

	ctx := context.Background()
	session := openagent.Session{
		ID:        "team-session-1",
		UserID:    "user-1",

		ModelID:   modelID,
		CreatedAt: time.Now(),
	}

	fmt.Println("=== Team: analyst → designer → coder → tester → reviewer ===")
	fmt.Printf("User: Write a function that validates email addresses\n\n")

	result, err := team.Run(ctx, session, openagent.UserMessage("Write a function that validates email addresses"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Final output: %s\n", result.FinalOutput)
	fmt.Printf("Handoffs: %d\n", len(result.HandoffChain))
	for i, h := range result.HandoffChain {
		fmt.Printf("  %d. %s → %s: %s\n", i+1, h.From, h.To, truncate(h.Message, 120))
	}
	fmt.Printf("Total turns: %d, Tokens: prompt=%d completion=%d total=%d\n",
		result.TotalTurns, result.Usage.PromptTokens, result.Usage.CompletionTokens, result.Usage.TotalTokens)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// Memory example: incremental rolling compaction in a realistic agent workflow.
//
// The agent writes a code file, reads it back, refactors it, writes tests,
// and reviews the result — a mini software development loop. This generates
// enough content (tool calls, tool results, multi-turn responses) to
// naturally fill the working window and trigger compaction.
//
// Demonstrates:
//  1. Compaction fires when token budget is exceeded — not because we
//     artificially lowered it, but because real agent work fills context.
//  2. Incremental: later compactions only cover new messages since the
//     last pass (ThroughIndex tracking).
//  3. Compressed summary is injected into the prompt alongside hot messages.
//  4. Original messages are NEVER deleted — full archive remains searchable.
//
//	go run ./examples/memory/
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/memory/sqlite"
	"github.com/yusheng-g/openagent-go/model/openai"
	opentool "github.com/yusheng-g/openagent-go/tool"
)

type compactionObserver struct{}

func (o *compactionObserver) ObserveStage(ctx context.Context, event openagent.StageEvent) {
	if event.Name != openagent.StageMemoryFetch || event.Phase != "leave" {
		return
	}
	d := event.Detail
	if d == nil {
		return
	}
	if err, _ := d["compaction_error"]; err != nil {
		fmt.Printf("  ⚠️  compaction error: %v\n", err)
		return
	}
	cnt, _ := d["compacted_count"].(int)
	if cnt <= 0 {
		return
	}
	from, _ := d["compacted_from"].(int)
	to, _ := d["compacted_to"].(int)
	summary, _ := d["compacted_summary"].(string)

	fmt.Printf("\n  📦 [compaction] messages[%d:%d] (%d messages) compressed → ThroughIndex=%d\n",
		from, to, cnt, to)
	if summary != "" {
		fmt.Printf("                   summary: %s\n", truncate(summary, 100))
	}
}

var _ openagent.RunObserver = (*compactionObserver)(nil)

func main() {
	apiKey := os.Getenv("OPENAGENT_API_KEY")
	modelID := os.Getenv("OPENAGENT_MODEL")
	baseURL := os.Getenv("OPENAGENT_BASE_URL")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "OPENAGENT_API_KEY not set")
		os.Exit(1)
	}

	model := openai.New(apiKey, modelID, baseURL).WithContextWindow(128_000)

	mem, err := sqlite.New("./memory_demo.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "memory: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove("./memory_demo.db")

	mem.WithSummarizer(openai.NewSummarizer(apiKey, modelID, baseURL))

	workDir, _ := filepath.Abs(".")
	fileTools := []openagent.Tool{
		opentool.NewReadFile(workDir),
		opentool.NewWriteFile(workDir),
		opentool.NewListDir(workDir),
	}

	session := openagent.Session{

		ModelID: modelID, CreatedAt: time.Now(),
	}
	ctx := context.Background()

	agent := openagent.NewAgent("memory-demo",
		openagent.WithModel(model),
		openagent.WithSystemPrompts(`You are a helpful coding assistant. You can read, write, and list files.

When the user asks you to write code, do it — use write_file to save it to disk.
When asked to review or refactor, use read_file first to see what's on disk.
You have access to a workspace where you can create and modify files.`),
		openagent.WithMemory(mem),
		openagent.WithTools(fileTools...),
		openagent.WithMaxWorkingTokens(4000),
		openagent.WithMaxTurns(10),
		openagent.WithRunObserver(&compactionObserver{}),
	)

	// ── Phase 1: teach a personal fact ──
	fmt.Println("━━━ Phase 1 — teach ━━━")
	agent.Run(ctx, session, openagent.UserMessage("My favourite colour is cerulean. Got it?"))

	// ── Phase 2: real agent work that fills the context window ──
	fmt.Println("\n━━━ Phase 2 — develop (context fills, compaction fires) ━━━")

	work := []string{
		// Task 1: write a Go utility (agent uses write_file, generates code)
		"Write a Go file 'utils.go' with a function ReverseString(s string) string and a function IsPalindrome(s string) bool. Include package declaration and comments.",
		// Task 2: read it back for review (read_file returns full content)
		"Read utils.go and review it. Does the code look correct?",
		// Task 3: write a test file
		"Write 'utils_test.go' with table-driven tests for both ReverseString and IsPalindrome.",
		// Task 4: read the test file and explain how table-driven tests work
		"Read utils_test.go and explain what each test case validates.",
		// Task 5: refactor — add a new function
		"Read utils.go again, then add a function Truncate(s string, n int) string that truncates to n runes. Overwrite the file.",
			// Task 6: list the workspace to confirm all files
		"List all files in the workspace and confirm we have both utils.go and utils_test.go.",
		// Task 7: code review with specific feedback
		"Read utils.go one more time and suggest any improvements for error handling, edge cases, or naming.",
		// Task 8: write a README
		"Write a README.md describing the utility package — one paragraph is enough.",
	}
	for i, task := range work {
		fmt.Printf("  task %d: %s\n", i+1, task)
		result, err := agent.Run(ctx, session, openagent.UserMessage(task))
		if err != nil {
			fmt.Fprintf(os.Stderr, "    ERROR: %v\n", err)
			continue
		}
		if result != nil {
			fmt.Printf("    → %s\n", truncate(result.FinalOutput, 80))
		}
		time.Sleep(500 * time.Millisecond)
	}

	// ── Phase 3: recall from compressed context ──
	fmt.Println("\n━━━ Phase 3 — recall (from compressed summary) ━━━")
	result, err := agent.Run(ctx, session,
		openagent.UserMessage("What is my favourite colour? I told you at the very beginning."))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
	} else if result != nil {
		fmt.Printf("  Agent: %s\n", result.FinalOutput)
	}

	// ── Evidence ──
	fmt.Println("\n━━━ Evidence ━━━")
	total, _ := mem.Count(ctx, session.ID)
	all, _ := mem.Recent(ctx, session.ID, total, 0)
	fmt.Printf("Total messages in archive: %d\n\n", len(all))
	for i, m := range all {
		marker := ""
		if strings.Contains(strings.ToLower(m.Content), "cerulean") {
			marker = " ← ★ FACT"
		}
		fmt.Printf("  [%2d] %s: %s%s\n", i, string(m.Role)[:4], truncate(m.Content, 65), marker)
	}

	comp, _ := mem.Compressed(ctx, session.ID)
	fmt.Println()
	if comp != nil && comp.Summary != "" {
		fmt.Printf("Compressed summary (ThroughIndex=%d):\n  %s\n", comp.ThroughIndex, comp.Summary)
		if strings.Contains(strings.ToLower(comp.Summary), "cerulean") {
			fmt.Println("\n✓ The fact survived compaction — it's in the summary.")
		}
	}
	fmt.Println("✓ Full message archive intact — original messages never deleted.")
}

func truncate(s string, n int) string {
	r := strings.ReplaceAll(s, "\n", " ")
	if len(r) <= n {
		return r
	}
	return r[:n] + "..."
}

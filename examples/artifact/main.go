// Artifact hook example: save large tool results to disk so the model
// can inspect them with read/grep instead of consuming context window.
//
// Pattern:
//   1. RunHooks.OnToolEnd intercepts every tool result
//   2. If result > threshold, write to /tmp/openagent/<sessionID>/<tool>_<ts>.txt
//   3. Replace *result with a short message pointing to the file
//   4. Model calls read/grep on that path to inspect the output
//
// Run:
//
//	OPENAGENT_API_KEY=sk-... go run ./examples/artifact/
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/model/openai"
	opentool "github.com/yusheng-g/openagent-go/tool"
)

// ArtifactHook saves large tool results to disk.
// Threshold below is deliberate — set to 512 bytes so the demo triggers
// with a very small output. Real usage: 64 * 1024 (64 KB).
type ArtifactHook struct {
	Threshold int // bytes; 0 = always save
	Prefix    string
}

// No-op — artifact hook only cares about tool results.
func (h *ArtifactHook) OnAgentStart(ctx context.Context, req openagent.ChatCompletionRequest) (any, error) {
	return nil, nil
}
func (h *ArtifactHook) OnAgentEnd(ctx context.Context, req openagent.ChatCompletionRequest, resp *openagent.ChatCompletionResponse, runErr error, startState any) {
}

// No-op — artifacts are handled after execution.
func (h *ArtifactHook) OnToolStart(ctx context.Context, tool openagent.FunctionDefinition, args json.RawMessage) (any, error) {
	return nil, nil
}

// OnToolEnd checks result size and saves to disk when it exceeds the threshold.
// Reads from the artifact directory are excluded to prevent artifact-of-artifact
// recursion.
func (h *ArtifactHook) OnToolEnd(ctx context.Context, tool openagent.FunctionDefinition, args json.RawMessage, result *string, err *error, startState any) {
	if result == nil || *result == "" {
		return
	}
	if h.Threshold > 0 && len(*result) <= h.Threshold {
		return
	}

	// Don't re-save reads from the artifact directory itself.
	if h.isReadingArtifact(args) {
		return
	}

	session, ok := openagent.SessionFromContext(ctx)
	if !ok {
		return // no session context — passthrough
	}

	dir := filepath.Join(opentool.ArtifactRoot(), session.ID)
	_ = os.MkdirAll(dir, 0755)

	name := fmt.Sprintf("%s_%d.txt", tool.Name, time.Now().UnixNano())
	path := filepath.Join(dir, name)

	raw := *result
	_ = os.WriteFile(path, []byte(raw), 0644)

	// Replace the result with a terse pointer. The model is smart enough
	// to read/grep the file when it needs details. No AI summarisation.
	sizeKB := (len(raw) + 1023) / 1024
	*result = fmt.Sprintf("%s: Tool output saved to %s (%d KB, %d lines). Use read or grep to inspect.",
		h.Prefix, path, sizeKB, strings.Count(raw, "\n")+1)
}

// ── Helpers ──

// isReadingArtifact checks whether args contain a path inside the artifact
// root. Used to prevent artifact-of-artifact recursion when read/grep
// inspect a previously saved artifact.
func (h *ArtifactHook) isReadingArtifact(args json.RawMessage) bool {
	var params struct {
		Path string `json:"path"`
	}
	json.Unmarshal(args, &params)
	return params.Path != "" && strings.HasPrefix(params.Path, opentool.ArtifactRoot())
}

// RemoveSessionArtifacts deletes the artifact directory for a session.
// Call from WithCleanupDir or in shutdown logic.
func RemoveSessionArtifacts(sessionID string) error {
	return os.RemoveAll(filepath.Join(opentool.ArtifactRoot(), sessionID))
}

// ── Demo tool ──

// FakeLogGenerator produces configurable amounts of fake log output.
// Used here to demonstrate the artifact hook; in real usage any tool
// (shell, API client, database query) can trigger the same path.
type FakeLogGenerator struct{}

func (t *FakeLogGenerator) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "generate_logs",
		Description: "Generate fake application log output for testing. The lines parameter controls volume.",
		Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"lines":  {"type": "integer", "description": "Number of log lines to generate (default: 5)"},
					"errors": {"type": "integer", "description": "Number of lines that should be ERROR level (default: 0)"}
				}
			}`),
	}
}

func (t *FakeLogGenerator) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Lines  int `json:"lines"`
		Errors int `json:"errors"`
	}
	json.Unmarshal(args, &params)
	if params.Lines <= 0 {
		params.Lines = 10
	}
	if params.Errors > params.Lines {
		params.Errors = params.Lines
	}

	var b strings.Builder
	levels := make([]string, params.Lines)
	for i := 0; i < params.Errors; i++ {
		levels[i] = "ERROR"
	}
	for i := params.Errors; i < params.Lines; i++ {
		levels[i] = "INFO"
	}

	for i := 0; i < params.Lines; i++ {
		b.WriteString(fmt.Sprintf(
			"[2026-07-17 %02d:%02d:%02d.000] [%s] [worker-%d] %s\n",
			10+i/3600,
			(i/60)%60,
			i%60,
			levels[i],
			i%4,
			fakeMessage(i, levels[i]),
		))
	}
	return b.String(), nil
}

func fakeMessage(i int, level string) string {
	messages := map[string][]string{
		"INFO": {
			"Request processed successfully",
			"Cache hit for key user:session:%d",
			"Connection pool: 12 active, 3 idle",
			"Health check passed",
		},
		"ERROR": {
			"Connection refused to backend service",
			"Timeout waiting for database response",
			"Rate limit exceeded for API key",
			"Disk space below threshold on /data",
		},
	}
	pool := messages[level]
	return fmt.Sprintf(pool[i%len(pool)], i)
}

// ── Main ──

func main() {
	apiKey := os.Getenv("OPENAGENT_API_KEY")
	modelID := os.Getenv("OPENAGENT_MODEL")
	baseURL := os.Getenv("OPENAGENT_BASE_URL")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "OPENAGENT_API_KEY not set")
		os.Exit(1)
	}
	if modelID == "" {
		modelID = "gpt-4o"
	}

	model := openai.New(apiKey, modelID, baseURL).WithContextWindow(128_000)

	// Threshold set low so the demo triggers easily.
	// Real usage: 64 * 1024 (64 KB).
	hook := &ArtifactHook{Threshold: 512, Prefix: ""}

	agent := openagent.NewAgent("log-analyzer",
		openagent.WithModel(model),
		openagent.WithSystemPrompts("You are a log analyzer. When tool output is large and saved to disk, use read and grep to inspect it."),
		openagent.WithTools(
			&FakeLogGenerator{},
			opentool.NewReadFile("."),
			opentool.NewGrep("."),
		),
		openagent.WithRunHooks(hook),
		openagent.WithMaxTurns(6),
	)

	session := openagent.Session{
		ID:        "artifact-demo",
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	// Clean up artifact dir when done.
	defer RemoveSessionArtifacts(session.ID)

	fmt.Println("=== Artifact Hook Demo ===")
	fmt.Println()
	fmt.Println("Sending: 'generate 200 log lines with 5 errors, then tell me what went wrong'")
	fmt.Println()

	result, err := agent.Run(ctx, session, openagent.UserMessage(
		"Generate 200 lines of logs with 5 errors using the generate_logs tool. "+
			"Then tell me what went wrong by reading the saved output file.",
	))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Agent Response ===")
	fmt.Println(result.FinalOutput)
	fmt.Printf("\nTurns: %d, Tokens: %d\n\n", result.TurnCount, result.Usage.TotalTokens)

	// Show artifacts on disk.
	dir := filepath.Join(opentool.ArtifactRoot(), session.ID)
	entries, _ := os.ReadDir(dir)
	if len(entries) > 0 {
		fmt.Println("Artifacts on disk:")
		for _, e := range entries {
			info, _ := e.Info()
			fmt.Printf("  %s  (%d bytes)\n", e.Name(), info.Size())
		}
		fmt.Printf("Full path: %s\n", dir)
	}
}

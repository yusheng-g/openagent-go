// Package tool provides built-in Tool implementations for openagent.
package tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/process"
)

// Shell lets an agent execute shell commands inside an [openagent.Sandbox].
// If no sandbox is configured, commands are rejected.
//
// Commands start immediately and run in the background. The tool waits up to
// the configured timeout (default 30s) for the command to complete. When the
// command finishes before the timeout, full stdout/stderr/exit code is
// returned — exactly as before.
//
// When a command runs longer than the timeout, the process stays alive and
// the tool returns a summary with the process ID, PID, partial output
// snapshot, and paths to output files. The model can then:
//
//	read <stdout.log>      — check latest output
//	shell kill <PID>        — kill the process
//
// Process output files live under /tmp/openagent/sess-<session-id>/proc-<pid>/
// and are cleaned up on session deletion.
//
// Implements both [openagent.Tool] and [openagent.StreamExecutor].
type Shell struct {
	sandbox  openagent.Sandbox
	language string
}

func NewShell(sandbox openagent.Sandbox) *Shell {
	return &Shell{sandbox: sandbox}
}

func (t *Shell) WithLanguage(lang string) *Shell {
	t.language = lang
	return t
}

func (t *Shell) Definition() openagent.FunctionDefinition {
	desc := "Execute a shell command. The command runs in the background — the shell waits up to 30s for it to finish. If the command is still running after the timeout, you'll get a process ID, PID, and paths to stdout.log / stderr.log. Use `read` to check progress and `shell kill <PID>` to stop it. The shell starts in the workspace root — use relative paths."
	if t.language != "" {
		desc = fmt.Sprintf("Execute a shell command in a %s sandbox. CWD is the workspace root.", t.language)
	}
	if t.sandbox == nil {
		desc += " [UNAVAILABLE: no sandbox configured]"
	} else if cwd := t.sandbox.CWD(); cwd != "" {
		desc += fmt.Sprintf(" (CWD: %s)", cwd)
	}
	return openagent.FunctionDefinition{
		Name:        "shell",
		Description: desc,
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"command": {
					"type": "string",
					"description": "The shell command to execute"
				},
				"description": {
					"type": "string",
					"description": "A short description of what this command does (for audit/logging)"
				},
				"timeout": {
					"type": "integer",
					"description": "Timeout in milliseconds (default: 30000 = 30 seconds)"
				}
			},
			"required": ["command"]
		}`),
	}
}

func (t *Shell) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Command     string `json:"command"`
		Description string `json:"description"`
		Timeout     int    `json:"timeout"` // milliseconds, 0 = default (30s)
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("shell: %w", err)
	}
	if params.Command == "" {
		return "", fmt.Errorf("shell: command is required")
	}
	if t.sandbox == nil {
		return "", fmt.Errorf("shell: no sandbox configured")
	}
	timeout := params.Timeout
	if timeout <= 0 {
		timeout = 30000 // 30 seconds
	}
	shellCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	cmd := openagent.Command{
		Program: "/bin/bash",
		Args:    []string{"-c", params.Command},
		WorkDir: t.sandbox.CWD(),
	}

	// If a ProcessManager is in context, attach output writers so stdout/stderr
	// are persisted to disk for the model to read across turns.
	pm := process.FromContext(ctx)
	if pm != nil {
		proc, err := pm.Create(params.Command)
		if err != nil {
			return "", fmt.Errorf("shell: %w", err)
		}
		cmd.StdoutW = proc.StdoutW()
		cmd.StderrW = proc.StderrW()

		result, runErr := t.sandbox.Run(shellCtx, cmd)
		if errors.Is(runErr, openagent.ErrProcessRunning) {
			// Process still running — rename dir to proc-{PID} and return snapshot.
			proc.SetPID(result.PID)
			return formatProcessRunning(proc), nil
		}
		// Process finished — clean up and return result.
		proc.Close()
		pm.Remove(proc.ID)
		if runErr != nil {
			return "", runErr
		}
		return formatShellResult(result), nil
	}

	// No ProcessManager — use sandbox directly (preserves backward compat).
	result, err := t.sandbox.Run(shellCtx, cmd)
	if errors.Is(err, openagent.ErrProcessRunning) {
		return formatProcessRunningNoFiles(result), nil
	}
	if err != nil {
		return "", err
	}
	return formatShellResult(result), nil
}

func (t *Shell) ExecuteStream(ctx context.Context, args json.RawMessage) <-chan openagent.ToolStreamChunk {
	var params struct {
		Command     string `json:"command"`
		Description string `json:"description"`
		Timeout     int    `json:"timeout"`
	}
	if err := json.Unmarshal(args, &params); err != nil || params.Command == "" || t.sandbox == nil {
		ch := make(chan openagent.ToolStreamChunk, 1)
		if err != nil {
			ch <- openagent.ToolStreamChunk{Error: fmt.Errorf("shell: %w", err)}
		} else if params.Command == "" {
			ch <- openagent.ToolStreamChunk{Error: fmt.Errorf("shell: command is required")}
		} else {
			ch <- openagent.ToolStreamChunk{Error: fmt.Errorf("shell: no sandbox configured")}
		}
		close(ch)
		return ch
	}

	timeout := params.Timeout
	if timeout <= 0 {
		timeout = 30000
	}
	streamCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)

	cmd := openagent.Command{
		Program: "/bin/bash",
		Args:    []string{"-c", params.Command},
		WorkDir: t.sandbox.CWD(),
	}

	// Attach file writers so stdout/stderr are persisted to disk for
	// long-running processes (ProcessManager in context).
	pm := process.FromContext(ctx)
	var proc *process.Proc
	if pm != nil {
		var err error
		proc, err = pm.Create(params.Command)
		if err != nil {
			cancel()
			ch := make(chan openagent.ToolStreamChunk, 1)
			ch <- openagent.ToolStreamChunk{Error: fmt.Errorf("shell: %w", err)}
			close(ch)
			return ch
		}
		cmd.StdoutW = proc.StdoutW()
		cmd.StderrW = proc.StderrW()
	}

	type streamRunner interface {
		RunStream(ctx context.Context, cmd *openagent.Command) <-chan openagent.ToolStreamChunk
	}
	sr, ok := t.sandbox.(streamRunner)
	if !ok {
		cancel()
		// Fallback to blocking Execute.
		ch := make(chan openagent.ToolStreamChunk, 1)
		go func() {
			defer close(ch)
			output, err := t.Execute(ctx, args)
			if err != nil {
				ch <- openagent.ToolStreamChunk{Error: err}
			} else {
				ch <- openagent.ToolStreamChunk{Content: output}
			}
		}()
		return ch
	}

	src := sr.RunStream(streamCtx, &cmd)
	wrapped := make(chan openagent.ToolStreamChunk, cap(src))
	go func() {
		defer cancel()
		defer close(wrapped)
		for chunk := range src {
			wrapped <- chunk
		}
		// Stream ended. Two cases:
		// 1. ctx timed out → process still running, return info.
		// 2. process exited normally → clean up proc.
		if streamCtx.Err() != nil && proc != nil {
			proc.SetPID(cmd.PID)
			wrapped <- openagent.ToolStreamChunk{
				Content: formatProcessRunning(proc),
			}
		} else if proc != nil {
			proc.Close()
			pm.Remove(proc.ID)
		}
	}()
	return wrapped
}

func formatShellResult(result openagent.Result) string {
	var b strings.Builder
	if result.Stdout != "" {
		b.WriteString(result.Stdout)
	}
	if result.Stderr != "" {
		if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
			b.WriteString("\n")
		}
		b.WriteString(result.Stderr)
	}
	if result.ExitCode != 0 {
		b.WriteString(fmt.Sprintf("\n[exit code: %d]", result.ExitCode))
	}
	s := b.String()
	if s == "" {
		s = "(no output)"
	}
	return s
}

// formatProcessRunning returns a formatted message for a still-running process.
// Reads the partial output from the persisted files (written via sandbox MultiWriter).
func formatProcessRunning(proc *process.Proc) string {
	var b strings.Builder
	elapsed := time.Since(proc.StartedAt).Truncate(time.Second)

	b.WriteString(fmt.Sprintf("[process: %s] PID: %d — running for %v\n\n", proc.ID, proc.PID, elapsed))

	if stdout, err := os.ReadFile(proc.StdoutPath); err == nil && len(stdout) > 0 {
		b.WriteString("── stdout (partial) ──\n")
		b.WriteString(truncateStr(string(stdout), 2000))
		b.WriteString("\n")
	}
	if stderr, err := os.ReadFile(proc.StderrPath); err == nil && len(stderr) > 0 {
		b.WriteString("── stderr (partial) ──\n")
		b.WriteString(truncateStr(string(stderr), 500))
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("── output files ──\n%s\n%s\n",
		proc.StdoutPath, proc.StderrPath))
	return b.String()
}

// formatProcessRunningNoFiles returns a formatted message when no
// ProcessManager is in context (no files persisted).
func formatProcessRunningNoFiles(result openagent.Result) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("[process] PID: %d — still running\n\n", result.PID))

	if result.Stdout != "" {
		b.WriteString("── stdout (partial) ──\n")
		b.WriteString(truncateStr(result.Stdout, 2000))
		b.WriteString("\n")
	}
	if result.Stderr != "" {
		b.WriteString("── stderr (partial) ──\n")
		b.WriteString(truncateStr(result.Stderr, 500))
		b.WriteString("\n")
	}

	b.WriteString("\nNo output files — read /proc to monitor, or shell kill <PID> to stop.")
	return b.String()
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + fmt.Sprintf("\n... [truncated, %d total chars]", len(s))
}

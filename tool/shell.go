// Package tool provides built-in Tool implementations for openagent.
package tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
)

// Shell lets an agent execute shell commands inside an [openagent.Sandbox].
// If no sandbox is configured, commands are rejected.
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
	desc := "Execute a shell command. The shell blocks until the command exits. For servers, watchers, or long-lived processes, append & to run in the background. The shell starts in the workspace root — use relative paths."
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
					"description": "Timeout in milliseconds (default: 180000 = 3 minutes)"
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
		Timeout     int    `json:"timeout"` // milliseconds, 0 = use runner default
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
		timeout = 180000 // 3 minutes
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	result, err := t.sandbox.Run(ctx, openagent.Command{
		Program: "/bin/bash",
		Args:    []string{"-c", params.Command},
		WorkDir: t.sandbox.CWD(),
	})
	if errors.Is(err, context.DeadlineExceeded) {
		return "", fmt.Errorf("shell: command timed out after %v", time.Duration(timeout)*time.Millisecond)
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
		timeout = 180000
	}

	type streamRunner interface {
		RunStream(ctx context.Context, cmd openagent.Command) <-chan openagent.ToolStreamChunk
	}
	if sr, ok := t.sandbox.(streamRunner); ok {
		streamCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
		src := sr.RunStream(streamCtx, openagent.Command{
			Program: "/bin/bash",
			Args:    []string{"-c", params.Command},
			WorkDir: t.sandbox.CWD(),
		})
		wrapped := make(chan openagent.ToolStreamChunk, cap(src))
		go func() {
			defer cancel()
			defer close(wrapped)
			for chunk := range src {
				wrapped <- chunk
			}
		}()
		return wrapped
	}

	// Fallback: Execute handles its own timeout internally.
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

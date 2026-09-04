// Package tool provides built-in Tool implementations for openagent.
package tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
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
// Process output files live under <ArtifactRoot()>/sess-<session-id>/proc-<pid>/
// and are cleaned up on session deletion.
//
// Implements both [openagent.Tool] and [openagent.StreamExecutor].
type Shell struct {
	sandbox openagent.Sandbox
}

func NewShell(sandbox openagent.Sandbox) *Shell {
	return &Shell{sandbox: sandbox}
}

// platformShell returns the default shell program: /bin/sh on Unix
// (POSIX, present on every Unix including Alpine which ships no bash),
// cmd.exe on Windows.
func platformShell() string {
	if runtime.GOOS == "windows" {
		return "cmd.exe"
	}
	return "/bin/sh"
}

// platformShellArg returns the flag that runs a one-shot command string
// via the program from platformShell ("-c" on Unix, "/C" on Windows).
func platformShellArg() string {
	if runtime.GOOS == "windows" {
		return "/C"
	}
	return "-c"
}

func (t *Shell) Definition() openagent.FunctionDefinition {
	program := platformShell()
	desc := fmt.Sprintf("Execute a command via %s. If the command finishes quickly, stdout/stderr/exit code are returned directly. If it runs longer, you'll get file paths to monitor progress — use `read` to check stdout.log, stderr.log, and exit.code.", program)
	if t.sandbox == nil {
		desc += " [UNAVAILABLE: no sandbox configured]"
	} else if cwd := t.sandbox.CWD(); cwd != "" {
		desc += fmt.Sprintf(" (CWD: %s)", cwd)
	}
	return openagent.FunctionDefinition{
		Name:        "shell",
		Description: desc,
		Parameters:  openagent.SchemaOf[ShellParams](),
	}
}

func (t *Shell) Execute(ctx context.Context, args json.RawMessage) *openagent.ToolResult {
	params, err := openagent.ParseArgs[ShellParams](args)
	if err != nil {
		return openagent.ErrorResult(fmt.Errorf("shell: %w", err), false, "")
	}
	if t.sandbox == nil {
		return openagent.ErrorResult(fmt.Errorf("shell: no sandbox configured"), false, "")
	}
	shellCtx, cancel := context.WithTimeout(ctx, shellTimeout(params.Timeout))
	defer cancel()

	program, flag := platformShell(), platformShellArg()
	cmd := openagent.Command{
		Program: program,
		Args:    []string{flag, params.Command},
		WorkDir: t.sandbox.CWD(),
	}

	// If a ProcessManager is in context, attach output writers so stdout/stderr
	// are persisted to disk for the model to read across turns.
	pm := process.FromContext(ctx)
	if pm != nil {
		proc, err := pm.Create(params.Command)
		if err != nil {
			return openagent.ErrorResult(fmt.Errorf("shell: %w", err), false, "")
		}
		cmd.StdoutW = proc.StdoutW()
		cmd.StderrW = proc.StderrW()
		cmd.ExitCodeW = proc.ExitCodeW()

		result, runErr := t.sandbox.Run(shellCtx, cmd)
		if errors.Is(runErr, openagent.ErrProcessRunning) {
			// Process still running — rename dir to proc-{PID} and return snapshot.
			proc.SetPID(result.PID)
			return &openagent.ToolResult{Content: formatProcessRunning(proc)}
		}
		// Process finished — clean up and return result.
		proc.Close()
		pm.Remove(proc.ID)
		if runErr != nil {
			return openagent.ErrorResult(runErr, false, "")
		}
		return &openagent.ToolResult{Content: formatShellResult(result)}
	}

	// No ProcessManager — use sandbox directly (preserves backward compat).
	result, err := t.sandbox.Run(shellCtx, cmd)
	if errors.Is(err, openagent.ErrProcessRunning) {
		return &openagent.ToolResult{Content: formatProcessRunningNoFiles(result)}
	}
	if err != nil {
		return openagent.ErrorResult(err, false, "")
	}
	return &openagent.ToolResult{Content: formatShellResult(result)}
}

func (t *Shell) ExecuteStream(ctx context.Context, args json.RawMessage) <-chan openagent.ToolStreamChunk {
	params, err := openagent.ParseArgs[ShellParams](args)
	if err != nil {
		ch := make(chan openagent.ToolStreamChunk, 1)
		ch <- openagent.ToolStreamChunk{Error: fmt.Errorf("shell: %w", err)}
		close(ch)
		return ch
	}
	if t.sandbox == nil {
		ch := make(chan openagent.ToolStreamChunk, 1)
		ch <- openagent.ToolStreamChunk{Error: fmt.Errorf("shell: no sandbox configured")}
		close(ch)
		return ch
	}

	streamCtx, cancel := context.WithTimeout(ctx, shellTimeout(params.Timeout))

	program, flag := platformShell(), platformShellArg()
	cmd := openagent.Command{
		Program: program,
		Args:    []string{flag, params.Command},
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
		cmd.ExitCodeW = proc.ExitCodeW()
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
			output := t.Execute(ctx, args)
			if output.Error != nil {
				ch <- openagent.ToolStreamChunk{Error: output.AsError()}
			} else {
				ch <- openagent.ToolStreamChunk{Content: output.Content}
			}
		}()
		return ch
	}

	src := sr.RunStream(streamCtx, &cmd)
	wrapped := make(chan openagent.ToolStreamChunk, 16)
	go func() {
		defer cancel()
		defer close(wrapped)
		for {
			select {
			case chunk, ok := <-src:
				if !ok {
					if proc != nil {
						proc.Close()
						pm.Remove(proc.ID)
					}
					return
				}
				select {
				case wrapped <- chunk:
				case <-streamCtx.Done():
					if proc != nil && cmd.PID > 0 {
						proc.SetPID(cmd.PID)
						select {
						case wrapped <- openagent.ToolStreamChunk{Content: formatProcessRunning(proc)}:
						default:
						}
					}
					go func() {
						for range src {
						}
					}()
					return
				}
			case <-streamCtx.Done():
				if proc != nil && cmd.PID > 0 {
					proc.SetPID(cmd.PID)
				}
				if proc != nil {
					select {
					case wrapped <- openagent.ToolStreamChunk{Content: formatProcessRunning(proc)}:
					default:
					}
				}
				go func() {
					for range src {
					}
				}()
				return
			}
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
	pid := proc.PIDNow()
	stdoutPath, stderrPath, exitCodePath := proc.Paths()
	elapsed := time.Since(proc.StartedAt).Truncate(time.Second)

	// Check if exit code is available (process completed after timeout).
	var status string
	if code, err := os.ReadFile(exitCodePath); err == nil {
		status = fmt.Sprintf("exited (code: %s)", strings.TrimSpace(string(code)))
	} else {
		status = fmt.Sprintf("running for %v", elapsed)
	}
	b.WriteString(fmt.Sprintf("[process: %s] PID: %d — %s\n\n", proc.ID, pid, status))

	if stdout, err := os.ReadFile(stdoutPath); err == nil && len(stdout) > 0 {
		b.WriteString("── stdout ──\n")
		b.WriteString(truncateStr(string(stdout), 2000))
		b.WriteString("\n")
	}
	if stderr, err := os.ReadFile(stderrPath); err == nil && len(stderr) > 0 {
		b.WriteString("── stderr ──\n")
		b.WriteString(truncateStr(string(stderr), 500))
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("── output files ──\n%s\n%s\n%s\n",
		stdoutPath, stderrPath, exitCodePath))
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
	// Truncate by rune: byte-slicing can cut a multi-byte UTF-8 sequence
	// in half, producing invalid UTF-8 for the model (Chinese output is
	// 3 bytes per rune, so a byte cut almost always lands mid-rune).
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + fmt.Sprintf("\n... [truncated, %d total chars]", len(runes))
}

type ShellParams struct {
	Command     string `json:"command" jsonschema:"description=The shell command to execute"`
	Description string `json:"description,omitempty" jsonschema:"description=A short description of what this command does (for audit/logging)"`
	RiskNote    string `json:"risk_note,omitempty" jsonschema:"description=For destructive/irreversible commands (rm -rf, terraform apply, kubectl delete, git push --force, etc.), state the risk reason in a few words. Empty for safe commands."`
	Timeout     int    `json:"timeout,omitempty" jsonschema:"description=Seconds to wait before the command is backgrounded (default: 30, min: 1, max: 600)"`
}

// shellTimeout resolves the per-call timeout: explicit parameter wins,
// default 30s, clamped to [1, 600]s.
func shellTimeout(timeout int) time.Duration {
	if timeout <= 0 {
		return 30 * time.Second
	}
	d := time.Duration(timeout) * time.Second
	if d < 1*time.Second {
		d = 1 * time.Second
	}
	if d > 600*time.Second {
		d = 600 * time.Second
	}
	return d
}

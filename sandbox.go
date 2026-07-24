package openagent

import (
	"context"
	"errors"
	"io"
)

// ErrProcessRunning is returned by Sandbox.Run when the command is still
// running after ctx expires. The returned Result contains partial stdout/stderr
// and the process PID so the caller can monitor or kill it later.
var ErrProcessRunning = errors.New("process still running")

// Command represents a command to execute in a sandbox.
type Command struct {
	Program string   // executable name or path
	Args    []string // arguments
	Env     []string // environment variables (KEY=VALUE)
	WorkDir string   // working directory
	Stdin   string   // optional stdin content

	// StdoutW, if set, receives a copy of everything written to stdout.
	// Used by ProcessManager to persist output to disk for long-running commands.
	StdoutW io.Writer
	// StderrW is like StdoutW but for stderr.
	StderrW io.Writer

	// PID is set by the sandbox after the process starts. The caller can
	// read it after Run/RunStream returns (or after the first stream chunk
	// arrives) to get the OS process ID.
	PID int
}

// Result is the output of a command executed in a sandbox.
type Result struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`

	// PID is the OS process ID of the outermost sandbox process
	// (e.g. bwrap or bash). Valid even when ExitCode == -1 (still running).
	PID int `json:"pid,omitempty"`
}

// Sandbox isolates command execution. Tool implementations that execute
// code (ShellTool, etc.) use this interface. nil Sandbox = no sandbox
// configured — tools should refuse to execute or reject commands.
//
// Built-in implementation: sandbox/native (Linux: bwrap, macOS: Seatbelt).
// See tool/shell.go for the canonical consumer.
type Sandbox interface {
	Run(ctx context.Context, cmd Command) (Result, error)

	// CWD returns the working directory from the tool's perspective.
	// This may differ from the host path when the sandbox remaps
	// paths (e.g. bwrap maps host dir → /workspace).
	CWD() string
}

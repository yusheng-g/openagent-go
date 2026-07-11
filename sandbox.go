package openagent

import "context"

// Command represents a command to execute in a sandbox.
type Command struct {
	Program string   // executable name or path
	Args    []string // arguments
	Env     []string // environment variables (KEY=VALUE)
	WorkDir string   // working directory
	Stdin   string   // optional stdin content
}

// Result is the output of a command executed in a sandbox.
type Result struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// Sandbox isolates command execution. Tool implementations that execute
// code (ShellTool, etc.) use this interface. nil Sandbox = no sandbox
// configured — tools should refuse to execute or reject commands.
//
// Built-in implementation: sandbox/native (Linux: bwrap, macOS: Seatbelt).
// See tool/shell.go for the canonical consumer.
type Sandbox interface {
	Run(ctx context.Context, cmd Command) (Result, error)
}

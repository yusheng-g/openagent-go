// Package native provides OS-native sandbox implementations.
//
// Three-layer architecture:
//  1. Agent has no sandbox — tool execution forks a child process
//  2. OS security mechanism applied in child between fork and exec:
//     macOS   → sandbox-exec + Seatbelt profile
//     Linux   → Landlock LSM (filesystem) + seccomp (syscalls)
//     Windows → Restricted Token + Job Object
//  3. Filesystem sandbox: workspace r/w, system bins read-only, everything else denied
//
// Zero external dependencies — uses only OS built-in tools and kernel APIs.
package native

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
)

// Sandbox confines command execution using OS-native security mechanisms.
// Commands can only read/write within the workspace directory.
type Sandbox struct {
	workDir string // host path, the only writable directory
	policy  Policy
}

// Policy controls how strictly the sandbox confines the command.
//
// Enabled governs whether the sandbox is active at all:
//   - false (zero value) → commands run unconfined (no bwrap/seatbelt)
//   - true               → commands run inside the OS-native sandbox
//
// Network controls outbound network access (only effective when Enabled):
//   - "" or "host"     → share the host's network namespace (network allowed)
//   - "isolated"       → unshare the network namespace (no outbound network)
//
// WritablePaths are additional host paths bind-mounted read-write inside
// the sandbox (in addition to the workspace).
// ReadablePaths are additional host paths bind-mounted read-only inside
// the sandbox (in addition to the system paths already mounted).
type Policy struct {
	Enabled       bool
	Network       string
	WritablePaths []string
	ReadablePaths []string
}

// New creates a native sandbox rooted at workDir with the default policy
// (host network, no extra paths). workDir is created if it doesn't exist.
func New(workDir string) (*Sandbox, error) {
	return NewWithPolicy(workDir, Policy{})
}

// NewWithPolicy creates a native sandbox rooted at workDir with the given
// policy. workDir is created if it doesn't exist.
func NewWithPolicy(workDir string, p Policy) (*Sandbox, error) {
	abs, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("native sandbox: %w", err)
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return nil, fmt.Errorf("native sandbox: create workspace: %w", err)
	}
	return &Sandbox{workDir: abs, policy: p}, nil
}

// CWD implements openagent.Sandbox. Returns the path as seen from
// inside the sandbox, not the host path. When the sandbox is disabled
// (or bwrap is unavailable), commands run directly in s.workDir.
func (s *Sandbox) CWD() string {
	if !s.policy.Enabled {
		return s.workDir
	}
	if _, err := exec.LookPath("bwrap"); err == nil {
		return "/workspace"
	}
	return s.workDir
}

// Run executes cmd in a sandboxed child process. When Policy.Enabled
// is false, commands run unconfined (plain exec, no bwrap/seatbelt).
func (s *Sandbox) Run(ctx context.Context, cmd openagent.Command) (openagent.Result, error) {
	if !s.policy.Enabled {
		return s.unconfinedRun(ctx, cmd)
	}
	return s.confineAndRun(ctx, cmd)
}

// RunStream is like Run but streams stdout/stderr line by line.
func (s *Sandbox) RunStream(ctx context.Context, cmd openagent.Command) <-chan openagent.ToolStreamChunk {
	if !s.policy.Enabled {
		return s.unconfinedRunStream(ctx, cmd)
	}
	return s.confineAndRunStream(ctx, cmd)
}

// readLines reads lines from r and sends them as chunks to ch.
// Used by streaming implementations on all platforms.
func readLines(r io.Reader, ch chan<- openagent.ToolStreamChunk, done chan<- struct{}) {
	defer func() { done <- struct{}{} }()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 4096), 1024*1024)
	for sc.Scan() {
		ch <- openagent.ToolStreamChunk{Content: sc.Text() + "\n"}
	}
}

// ── Unconfined execution (platform-agnostic) ──

// unconfinedRun executes cmd directly on the host without any sandbox.
// A stderr warning is appended only when Policy.Enabled is true (i.e.
// the sandbox was requested but fell back to unconfined due to bwrap/
// seatbelt failure). When Enabled is false the user explicitly opted
// out, so no warning is added.
func (s *Sandbox) unconfinedRun(ctx context.Context, cmd openagent.Command) (openagent.Result, error) {
	c := exec.CommandContext(ctx, cmd.Program, cmd.Args...)
	c.Dir = s.workDir
	for _, e := range cmd.Env {
		c.Env = append(c.Env, e)
	}
	if cmd.Stdin != "" {
		c.Stdin = strings.NewReader(cmd.Stdin)
	}

	var stdout, stderr strings.Builder
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	result := openagent.Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}
	if s.policy.Enabled {
		result.Stderr += "\n[warning: running without sandbox]"
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return result, fmt.Errorf("native sandbox (unconfined): %w", err)
	}
	return result, nil
}

func (s *Sandbox) unconfinedRunStream(ctx context.Context, cmd openagent.Command) <-chan openagent.ToolStreamChunk {
	ch := make(chan openagent.ToolStreamChunk, 16)
	go func() {
		defer close(ch)
		c := exec.CommandContext(ctx, cmd.Program, cmd.Args...)
		c.Dir = s.workDir
		for _, e := range cmd.Env {
			c.Env = append(c.Env, e)
		}
		if cmd.Stdin != "" {
			c.Stdin = strings.NewReader(cmd.Stdin)
		}

		stdout, _ := c.StdoutPipe()
		stderr, _ := c.StderrPipe()
		if err := c.Start(); err != nil {
			ch <- openagent.ToolStreamChunk{Error: err}
			return
		}
		done := make(chan struct{}, 2)
		go readLines(stdout, ch, done)
		go readLines(stderr, ch, done)
		<-done
		<-done
		_ = c.Wait()
	}()
	return ch
}

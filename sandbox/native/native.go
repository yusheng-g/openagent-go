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
	"path/filepath"

	openagent "github.com/yusheng-g/openagent-go"
)

// Sandbox confines command execution using OS-native security mechanisms.
// Commands can only read/write within the workspace directory.
type Sandbox struct {
	workDir      string
	extraMounts  []bindMount
	bwrapFailed  bool
	selfTested   bool
}

type bindMount struct{ src, dst string }

// AddMount adds an additional read-only bind mount inside the sandbox.
// src is the host path, dst is the path visible inside the sandbox.
func (s *Sandbox) AddMount(src, dst string) *Sandbox {
	s.extraMounts = append(s.extraMounts, bindMount{src: src, dst: dst})
	return s
}

// New creates a native sandbox rooted at workDir.
// workDir is created if it doesn't exist.
func New(workDir string) (*Sandbox, error) {
	abs, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("native sandbox: %w", err)
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return nil, fmt.Errorf("native sandbox: create workspace: %w", err)
	}
	return &Sandbox{workDir: abs}, nil
}

// WorkDir returns the sandbox workspace path.
func (s *Sandbox) WorkDir() string { return s.workDir }

// Run executes cmd in a sandboxed child process.
// Calls the platform-specific confineAndRun (build-tagged per OS).
func (s *Sandbox) Run(ctx context.Context, cmd openagent.Command) (openagent.Result, error) {
	return s.confineAndRun(ctx, cmd)
}

// RunStream is like Run but streams stdout/stderr line by line.
func (s *Sandbox) RunStream(ctx context.Context, cmd openagent.Command) <-chan openagent.ToolStreamChunk {
	return s.confineAndRunStream(ctx, cmd)
}

// readLines reads lines from r and sends them as chunks to ch.
// Used by both darwin and linux streaming implementations.
func readLines(r io.Reader, ch chan<- openagent.ToolStreamChunk, done chan<- struct{}) {
	defer func() { done <- struct{}{} }()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 4096), 1024*1024)
	for sc.Scan() {
		ch <- openagent.ToolStreamChunk{Content: sc.Text() + "\n"}
	}
}

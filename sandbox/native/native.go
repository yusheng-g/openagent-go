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
// Network controls outbound network access:
//   - "" or "host"     → share the host's network namespace (network allowed)
//   - "isolated"       → unshare the network namespace (no outbound network)
//
// WritablePaths are additional host paths bind-mounted read-write inside
// the sandbox (in addition to the workspace).
// ReadablePaths are additional host paths bind-mounted read-only inside
// the sandbox (in addition to the system paths already mounted).
type Policy struct {
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
// inside the sandbox, not the host path. bwrap maps s.workDir to
// /workspace; unconfined mode preserves the original path.
// Callers should check for bwrap the same way confineAndRun does.
func (s *Sandbox) CWD() string {
	if _, err := exec.LookPath("bwrap"); err == nil {
		return "/workspace"
	}
	return s.workDir
}

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

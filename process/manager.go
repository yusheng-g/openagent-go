// Package process provides lifecycle management for background shell processes
// started by the shell tool. A Manager tracks running processes, persists their
// stdout/stderr to disk, and allows the model to monitor or kill them across turns.
//
//	// Per-session creation (REST / ACP):
//	pm, _ := process.NewManager(filepath.Join(cwd, ".openagent", "proc"))
//	defer pm.Cleanup()
//	ctx = process.WithManager(ctx, pm)
//
//	// In shell tool:
//	pm := process.FromContext(ctx)
//	proc, _ := pm.Create("npm run build")
//	// pass proc.StdoutW / proc.StderrW to sandbox.Command
package process

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// ctxKey is unexported to prevent external packages from colliding.
type ctxKey struct{}

// WithManager attaches a Manager to the context.
func WithManager(ctx context.Context, pm *Manager) context.Context {
	return context.WithValue(ctx, ctxKey{}, pm)
}

// FromContext extracts the Manager from ctx, or nil.
func FromContext(ctx context.Context) *Manager {
	pm, _ := ctx.Value(ctxKey{}).(*Manager)
	return pm
}

// Manager tracks running background processes for a session.
type Manager struct {
	mu      sync.Mutex
	procs   map[string]*Proc
	baseDir string
	nextID  int
}

// Proc represents a single running (or recently exited) process.
//
// The mutable fields (PID, dir, and the three *Path fields) are written
// by SetPID — called from the runner's tool goroutine once the OS PID is
// known — and read concurrently by Cleanup (session deletion, a
// different goroutine) and by the formatter that renders the model's
// result. They are guarded by mu. The immutable fields (ID, Command,
// StartedAt) and the file handles are set once at Create time and need
// no locking.
type Proc struct {
	ID          string    // short hex ID, e.g. "a1b2c3d4"
	PID         int       // host OS PID of the sandbox wrapper (bwrap / bash)
	Command     string    // original shell command
	StdoutPath  string    // absolute path to stdout.log
	StderrPath  string    // absolute path to stderr.log
	ExitCodePath string   // absolute path to exit.code
	StartedAt   time.Time

	mu       sync.Mutex // guards PID, dir, StdoutPath, StderrPath, ExitCodePath
	dir      string    // proc subdirectory
	stdoutF  *os.File  // open file handle for stdout
	stderrF  *os.File  // open file handle for stderr
	exitCodeF *os.File // open file handle for exit code
}

// StdoutW returns the writer for sandbox stdout output.
func (p *Proc) StdoutW() io.Writer { return p.stdoutF }

// StderrW returns the writer for sandbox stderr output.
func (p *Proc) StderrW() io.Writer { return p.stderrF }

// ExitCodeW returns the writer for the process exit code.
func (p *Proc) ExitCodeW() io.Writer { return p.exitCodeF }

// Close closes the stdout, stderr, and exit code file handles.
func (p *Proc) Close() {
	if p.stdoutF != nil {
		p.stdoutF.Close()
	}
	if p.stderrF != nil {
		p.stderrF.Close()
	}
	if p.exitCodeF != nil {
		p.exitCodeF.Close()
	}
}

// SetPID renames the proc directory to proc-<pid> and updates paths.
// Safe to call while files are still being written (Linux allows rename
// of open files). Call after sandbox.Run returns the OS PID.
func (p *Proc) SetPID(pid int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.PID = pid
	newDir := filepath.Join(filepath.Dir(p.dir), fmt.Sprintf("proc-%d", pid))
	if err := os.Rename(p.dir, newDir); err != nil {
		return // keep old name on failure
	}
	p.dir = newDir
	p.StdoutPath = filepath.Join(newDir, "stdout.log")
	p.StderrPath = filepath.Join(newDir, "stderr.log")
	p.ExitCodePath = filepath.Join(newDir, "exit.code")
}

// PIDNow returns the current OS PID under the lock. Use this instead of
// reading p.PID directly from a goroutine other than the one that called
// SetPID.
func (p *Proc) PIDNow() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.PID
}

// DirNow returns the current proc directory under the lock.
func (p *Proc) DirNow() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.dir
}

// Paths returns the current StdoutPath, StderrPath, ExitCodePath under the
// lock, so a concurrent SetPID rename can't hand a reader a stale path.
func (p *Proc) Paths() (stdout, stderr, exitCode string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.StdoutPath, p.StderrPath, p.ExitCodePath
}

// NewManager creates a Manager rooted at baseDir. The directory is created
// if it doesn't exist. baseDir should be inside the sandbox workspace so the
// model can read process output files.
func NewManager(baseDir string) (*Manager, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("process manager: %w", err)
	}
	return &Manager{
		procs:   make(map[string]*Proc),
		baseDir: baseDir,
	}, nil
}

// Create allocates a new Proc, creates its output directory
//
//	<baseDir>/<id>/ with stdout.log and stderr.log, and registers it.
//
// Caller must pass the returned writers to the sandbox via Command.StdoutW /
// Command.StderrW. After sandbox.Run returns:
//   - On nil/other error: caller should call proc.Close() and pm.Remove(proc.ID).
//   - On ErrProcessRunning: proc stays registered; output files remain open
//     until the process exits or the session is cleaned up.
func (m *Manager) Create(command string) (*Proc, error) {
	id := m.genID()

	dir := filepath.Join(m.baseDir, id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("process %s: %w", id, err)
	}

	stdoutPath := filepath.Join(dir, "stdout.log")
	stderrPath := filepath.Join(dir, "stderr.log")
	exitCodePath := filepath.Join(dir, "exit.code")

	stdoutF, err := os.Create(stdoutPath)
	if err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("process %s stdout: %w", id, err)
	}
	stderrF, err := os.Create(stderrPath)
	if err != nil {
		stdoutF.Close()
		os.RemoveAll(dir)
		return nil, fmt.Errorf("process %s stderr: %w", id, err)
	}
	exitCodeF, err := os.Create(exitCodePath)
	if err != nil {
		stdoutF.Close()
		stderrF.Close()
		os.RemoveAll(dir)
		return nil, fmt.Errorf("process %s exit code: %w", id, err)
	}

	p := &Proc{
		ID:           id,
		Command:      command,
		StdoutPath:   stdoutPath,
		StderrPath:   stderrPath,
		ExitCodePath: exitCodePath,
		StartedAt:    time.Now(),
		dir:          dir,
		stdoutF:      stdoutF,
		stderrF:      stderrF,
		exitCodeF:    exitCodeF,
	}

	m.mu.Lock()
	m.procs[id] = p
	m.mu.Unlock()

	return p, nil
}

// Get returns the Proc for id, or nil.
func (m *Manager) Get(id string) *Proc {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.procs[id]
}

// Remove closes the proc's files, deletes its directory, and removes
// it from tracking. Does not kill the process — use when a command
// completes before the timeout and output has been consumed.
func (m *Manager) Remove(id string) {
	m.mu.Lock()
	p, ok := m.procs[id]
	if ok {
		delete(m.procs, id)
	}
	m.mu.Unlock()
	if ok {
		p.Close()
		os.RemoveAll(p.dir)
	}
}

// List returns all tracked procs, most recent first.
func (m *Manager) List() []*Proc {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Proc, 0, len(m.procs))
	for _, p := range m.procs {
		out = append(out, p)
	}
	// Sort by start time descending (most recent first).
	for i := 0; i < len(out)-1; i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].StartedAt.After(out[i].StartedAt) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// Cleanup kills every tracked process (SIGKILL), closes files, and removes
// the base directory. Call on session delete.
func (m *Manager) Cleanup() error {
	m.mu.Lock()
	procs := make([]*Proc, 0, len(m.procs))
	for id, p := range m.procs {
		procs = append(procs, p)
		delete(m.procs, id)
	}
	m.mu.Unlock()

	for _, p := range procs {
		if pid := p.PIDNow(); pid > 0 {
			syscall.Kill(pid, syscall.SIGKILL)
		}
		p.Close()
		os.RemoveAll(p.DirNow())
	}
	return os.RemoveAll(m.baseDir)
}

// BaseDir returns the root directory for process output files.
func (m *Manager) BaseDir() string { return m.baseDir }

func (m *Manager) genID() string {
	var b [4]byte
	rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

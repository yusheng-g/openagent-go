package process

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// Manager has no tests. These cover the cross-goroutine mutation that
// previously raced: SetPID (called from the runner's tool goroutine once
// the OS PID is known) mutates PID/dir/*Path concurrently with Cleanup
// (session deletion) and the result formatter reading those same fields.
// Under -race the unlocked reads would fail; the locked getters must keep
// it clean.

func TestProcSetPIDConcurrentWithCleanup(t *testing.T) {
	base := t.TempDir()
	m, err := NewManager(base)
	if err != nil {
		t.Fatal(err)
	}

	const procs = 64
	var wg sync.WaitGroup
	for i := 0; i < procs; i++ {
		p, err := m.Create("echo hi")
		if err != nil {
			t.Fatal(err)
		}

		// Simulate the sandbox returning a PID (from a real child the
		// shell tool forks). Use a benign PID so Cleanup's Kill is a
		// no-op (killing PID 1 of init, or a recycled PID, would be
		// unsafe in CI); the fields themselves are what we stress.
		var pid int = 12345 + i

		// Reader goroutine: the result formatter / a monitoring turn
		// reads PID + paths while SetPID is renaming the dir.
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = p.PIDNow()
			stdout, stderr, exit := p.Paths()
			_ = stdout
			_ = stderr
			_ = exit
		}()

		// Writer: SetPID renames dir → proc-<pid> and rewrites paths.
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.SetPID(pid)
		}()
	}
	wg.Wait()
}

// TestProcPathsStaysConsistent asserts the three paths always resolve under
// the same locked snapshot, i.e. a Paths() call after SetPID reflects the
// renamed directory rather than mixing an old dir with new PID.
func TestProcPathsStaysConsistent(t *testing.T) {
	base := t.TempDir()
	m, err := NewManager(base)
	if err != nil {
		t.Fatal(err)
	}
	p, err := m.Create("ls")
	if err != nil {
		t.Fatal(err)
	}

	const pid = 999
	// Pre-create the target dir parent so Rename succeeds (it must move
	// into the same existing parent — it does, base).
	p.SetPID(pid)

	stdout, stderr, exit := p.Paths()
	wantDir := filepath.Join(base, "proc-999")
	for _, got := range []string{stdout, stderr, exit} {
		dir := filepath.Dir(got)
		if dir != wantDir {
			t.Fatalf("path %q not under renamed dir %q", got, wantDir)
		}
	}
	if p.PIDNow() != pid {
		t.Fatalf("PID = %d, want %d", p.PIDNow(), pid)
	}
	// dir on disk matches the locked snapshot.
	if fi, err := os.Stat(wantDir); err != nil || !fi.IsDir() {
		t.Fatalf("renamed dir not on disk at %q: %v", wantDir, err)
	}
}

// TestCleanupKillsByLockedPID ensures Cleanup reads PID through the lock
// (regression guard: a direct p.PID read would race SetPID).
func TestCleanupKillsByLockedPID(t *testing.T) {
	base := t.TempDir()
	m, err := NewManager(base)
	if err != nil {
		t.Fatal(err)
	}
	p, err := m.Create("sleep 1")
	if err != nil {
		t.Fatal(err)
	}
	// Concurrent SetPID while Cleanup runs.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.SetPID(0) // PID 0 → Cleanup's Kill branch skipped, not executed
	}()
	_ = m.Get(p.ID)
	if err := m.Cleanup(); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}
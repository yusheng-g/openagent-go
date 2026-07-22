package file

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yusheng-g/openagent-go"
)

// newTestMemory returns a Memory backed by a temp dir and a cleanup.
func newTestMemory(t *testing.T) (*Memory, string, func()) {
	t.Helper()
	dir := t.TempDir()
	m, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cleanup := func() { m.Close() }
	return m, dir, cleanup
}

// bigContent is a single token (no newlines) well over the 64KB default
// bufio.Scanner token cap, simulating an assistant message embedding a
// large artifact (a big read, a base64 screenshot, an SQL dump, …).
func bigContent(n int) string { return strings.Repeat("X", n) }

// TestCountOversizedLine verifies countLinesLocked no longer trips the
// bufio.Scanner 64KB token cap. A session containing a message whose body
// exceeds 64KB must still report the correct line count without error.
func TestCountOversizedLine(t *testing.T) {
	m, dir, cleanup := newTestMemory(t)
	defer cleanup()
	ctx := context.Background()
	sid := "s1"

	if err := m.Append(ctx, sid, openagent.Message{Role: "user", Content: "u1"}); err != nil {
		t.Fatalf("Append small: %v", err)
	}
	if err := m.Append(ctx, sid, openagent.Message{Role: "assistant", Content: bigContent(70 * 1024)}); err != nil {
		t.Fatalf("Append oversized: %v", err)
	}
	if err := m.Append(ctx, sid, openagent.Message{Role: "assistant", Content: "a1"}); err != nil {
		t.Fatalf("Append trailing: %v", err)
	}

	n, err := m.Count(ctx, sid)
	if err != nil {
		t.Fatalf("Count returned error on oversized line: %v", err)
	}
	if n != 3 {
		t.Fatalf("Count = %d, want 3", n)
	}
	_ = dir
}

// TestCountOversizedLineFirst ensures the count is correct even when the
// oversized line is the very first record (the position where stdlib
// bufio.Scanner returns 0 + ErrTooLong, causing callers to judge a
// populated session empty).
func TestCountOversizedLineFirst(t *testing.T) {
	m, _, cleanup := newTestMemory(t)
	defer cleanup()
	ctx := context.Background()
	sid := "s1"

	if err := m.Append(ctx, sid, openagent.Message{Role: "assistant", Content: bigContent(70 * 1024)}); err != nil {
		t.Fatalf("Append oversized-first: %v", err)
	}
	if err := m.Append(ctx, sid, openagent.Message{Role: "user", Content: "u1"}); err != nil {
		t.Fatalf("Append second: %v", err)
	}

	n, err := m.Count(ctx, sid)
	if err != nil {
		t.Fatalf("Count returned error on oversized-first line: %v", err)
	}
	if n != 2 {
		t.Fatalf("Count = %d, want 2", n)
	}
}

// TestAppendAfterRestartOversized reproduces the prior restart-append
// deadlock: a fresh Memory over a dir already holding a >64KB line seeds
// nextIdx via countLinesLocked. Previously that seed scan errored, so the
// first Append after a restart failed and nextIdx stayed at 0 — every
// subsequent Append re-entered the ==0 branch and failed again. The fix
// must let Append succeed and keep Indexes monotonic.
func TestAppendAfterRestartOversized(t *testing.T) {
	m, dir, cleanup := newTestMemory(t)
	defer cleanup()
	ctx := context.Background()
	sid := "s1"

	// Seed: 2 small + 1 oversized line.
	for _, c := range []string{"u1", "a1", bigContent(70 * 1024)} {
		if err := m.Append(ctx, sid, openagent.Message{Role: "user", Content: c}); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	// Simulate process restart: fresh Memory over the same dir.
	m2, err := New(dir)
	if err != nil {
		t.Fatalf("reopen New: %v", err)
	}
	defer m2.Close()

	post := openagent.Message{Role: "user", Content: "post-restart"}
	if err := m2.Append(ctx, sid, post); err != nil {
		t.Fatalf("Append after restart over oversized line errored: %v", err)
	}
	// Index assigned to the post-restart message must be count+1 = 4, not 1.
	// (Append reads msg.Index back only via Recent; verify via the file.)
	msgs, err := m2.Recent(ctx, sid, 100, 0)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(msgs) != 4 {
		t.Fatalf("Recent len = %d, want 4", len(msgs))
	}
	last := msgs[len(msgs)-1]
	if last.Index != 4 {
		t.Fatalf("post-restart message Index = %d, want 4 (monotonic, not reset to 1)", last.Index)
	}
}

// TestCountRecentNoSplitBrain guards the runner.go:540 invariant
// globalOffset := totalCount - len(recent) == 0. When the oversized line was
// readable by readAllLocked (1MB cap) but not countable by the old
// raw-scanner Count, globalOffset went negative. With the fix both paths
// agree.
func TestCountRecentNoSplitBrain(t *testing.T) {
	m, _, cleanup := newTestMemory(t)
	defer cleanup()
	ctx := context.Background()
	sid := "s1"

	extra := bigContent(70 * 1024)
	for _, msg := range []openagent.Message{
		{Role: "user", Content: "u1"},
		{Role: "assistant", Content: "a1"},
		{Role: "assistant", Content: extra},
	} {
		if err := m.Append(ctx, sid, msg); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	total, err := m.Count(ctx, sid)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	msgs, err := m.Recent(ctx, sid, 100, 0)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(msgs) != total {
		t.Fatalf("split-brain: Count=%d but Recent.len=%d (runner.go:540 globalOffset would be %d)",
			total, len(msgs), total-len(msgs))
	}
}

// TestCountEmptyAndPartialLine exercises edge cases the newline-counting
// ReadString loop must handle: an absent file (0), an empty file (0), and a
// corrupt trailing line with no terminating newline (not counted).
func TestCountEmptyAndPartialLine(t *testing.T) {
	m, dir, cleanup := newTestMemory(t)
	defer cleanup()
	ctx := context.Background()

	// Absent session file.
	if n, _ := m.Count(ctx, "nope"); n != 0 {
		t.Fatalf("Count absent = %d, want 0", n)
	}

	// Empty file on disk.
	sid := "s2"
	if err := os.WriteFile(filepath.Join(dir, sid+".jsonl"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	if n, _ := m.Count(ctx, sid); n != 0 {
		t.Fatalf("Count empty file = %d, want 0", n)
	}

	// N complete lines + 1 partial (no newline) tail.
	sid3 := "s3"
	body := "{\"role\":\"user\",\"content\":\"a\"}\n" +
		"{\"role\":\"assistant\",\"content\":\"b\"}\n" +
		"partial-with-no-newline"
	if err := os.WriteFile(filepath.Join(dir, sid3+".jsonl"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	if n, err := m.Count(ctx, sid3); err != nil {
		t.Fatalf("Count partial tail errored: %v", err)
	} else if n != 2 {
		t.Fatalf("Count partial tail = %d, want 2 (partial line must not be counted)", n)
	}
}

// TestCountVeryLargeLine ensures a >1MB line (beyond even readAllLocked's
// cap) still counts correctly via the newline-counting reader. This proves
// Count is no longer capped by any fixed buffer at all.
func TestCountVeryLargeLine(t *testing.T) {
	m, _, cleanup := newTestMemory(t)
	defer cleanup()
	ctx := context.Background()
	sid := "s1"

	big := bigContent(2 * 1024 * 1024) // 2MB, single line, >1MB readAllLocked cap
	if err := m.Append(ctx, sid, openagent.Message{Role: "assistant", Content: big}); err != nil {
		t.Fatalf("Append 2MB: %v", err)
	}
	if err := m.Append(ctx, sid, openagent.Message{Role: "user", Content: "tail"}); err != nil {
		t.Fatalf("Append tail: %v", err)
	}
	n, err := m.Count(ctx, sid)
	if err != nil {
		t.Fatalf("Count with 2MB line errored: %v", err)
	}
	if n != 2 {
		t.Fatalf("Count with 2MB line = %d, want 2", n)
	}
}

// TestOversizedContentIntegrity verifies that a message with >64KB content
// can be read back with the exact same content via Recent. This proves the
// write path (Append) and read path (readAllLocked, within its 1MB cap) agree
// on oversized lines — not just that Count returns a correct number.
func TestOversizedContentIntegrity(t *testing.T) {
	m, _, cleanup := newTestMemory(t)
	defer cleanup()
	ctx := context.Background()
	sid := "s1"

	want := bigContent(90 * 1024) // 90KB, within readAllLocked 1MB cap
	if err := m.Append(ctx, sid, openagent.Message{
		Role:    "assistant",
		Content: want,
	}); err != nil {
		t.Fatalf("Append oversized: %v", err)
	}

	msgs, err := m.Recent(ctx, sid, 10, 0)
	if err != nil {
		t.Fatalf("Recent errored: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("Recent len = %d, want 1", len(msgs))
	}
	if msgs[0].Content != want {
		t.Fatalf("Content mismatch: len(got)=%d, len(want)=%d", len(msgs[0].Content), len(want))
	}
}

// TestMultipleOversizedLines stresses the newline-counting loop with 3
// consecutive oversized lines. This verifies the loop correctly handles
// multiple long records without the counting, allocation, or EOF logic
// getting confused.
func TestMultipleOversizedLines(t *testing.T) {
	m, _, cleanup := newTestMemory(t)
	defer cleanup()
	ctx := context.Background()
	sid := "s1"

	contents := []string{
		bigContent(70 * 1024), // 70KB
		bigContent(90 * 1024), // 90KB
		bigContent(80 * 1024), // 80KB
	}
	for i, c := range contents {
		if err := m.Append(ctx, sid, openagent.Message{Role: "user", Content: c}); err != nil {
			t.Fatalf("Append[%d] errored: %v", i, err)
		}
	}

	n, err := m.Count(ctx, sid)
	if err != nil {
		t.Fatalf("Count errored: %v", err)
	}
	if n != 3 {
		t.Fatalf("Count = %d, want 3", n)
	}

	msgs, err := m.Recent(ctx, sid, 10, 0)
	if err != nil {
		t.Fatalf("Recent errored on oversized lines: %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("Recent len = %d, want 3", len(msgs))
	}
	for i := range contents {
		if msgs[i].Content != contents[i] {
			t.Fatalf("Content[%d] mismatch: len(got)=%d, len(want)=%d", i, len(msgs[i].Content), len(contents[i]))
		}
	}
}

// TestCountAndRecentLargeLine1000x runs 1000 small writes followed by one
// >64KB line, verifying Count and Recent agree. This simulates the real
// session pattern where an oversized message appears after many normal ones.
func TestCountAndRecentLargeLine1000x(t *testing.T) {
	m, _, cleanup := newTestMemory(t)
	defer cleanup()
	ctx := context.Background()
	sid := "s1"

	for i := 0; i < 1000; i++ {
		if err := m.Append(ctx, sid, openagent.Message{Role: "user", Content: "msg"}); err != nil {
			t.Fatalf("Append[%d] errored: %v", i, err)
		}
	}
	if err := m.Append(ctx, sid, openagent.Message{Role: "assistant", Content: bigContent(70 * 1024)}); err != nil {
		t.Fatalf("Append oversized errored: %v", err)
	}

	total, err := m.Count(ctx, sid)
	if err != nil {
		t.Fatalf("Count errored: %v", err)
	}
	if total != 1001 {
		t.Fatalf("Count = %d, want 1001", total)
	}

	// Recent with n=100 should still work and the sum of small msg + oversized
	// should not confuse the content readback.
	msgs, err := m.Recent(ctx, sid, 100, 0)
	if err != nil {
		t.Fatalf("Recent errored: %v", err)
	}
	if len(msgs) != 100 {
		t.Fatalf("Recent len = %d, want 100", len(msgs))
	}
}

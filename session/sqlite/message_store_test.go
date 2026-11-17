package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/session"
)

func newTestRuntime(t *testing.T) (*session.DefaultRuntime, *MessageStore) {
	t.Helper()
	ms, err := NewMessageStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("NewMessageStore: %v", err)
	}
	t.Cleanup(func() { ms.Close() })
	store, err := New(ms.DB())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return session.NewRuntime(store, ms), ms
}

// TestForkIntegration drives Runtime.Fork over the real SQLite backend:
// messages, the compressed summary, and metadata all copy under a new ID.
func TestForkIntegration(t *testing.T) {
	rt, ms := newTestRuntime(t)
	ctx := context.Background()

	src, err := rt.Create(ctx, "u1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	for _, c := range []string{"one", "two", "three"} {
		if err := ms.Append(ctx, src.ID, openagent.UserMessage(c)); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	if err := ms.storeCompressed(ctx, src.ID, &openagent.CompressedContext{
		Summary:      "earlier discussion",
		ThroughIndex: 2,
	}); err != nil {
		t.Fatalf("storeCompressed: %v", err)
	}

	dst, err := rt.Fork(ctx, src.ID)
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}

	if n, _ := ms.Count(ctx, dst.ID); n != 3 {
		t.Fatalf("fork message count = %d, want 3", n)
	}
	msgs, err := ms.Recent(ctx, dst.ID, 10, 0)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	for i, want := range []string{"one", "two", "three"} {
		if msgs[i].Content != want {
			t.Fatalf("forked[%d] = %q, want %q", i, msgs[i].Content, want)
		}
	}

	cc, err := ms.Compressed(ctx, dst.ID)
	if err != nil || cc == nil {
		t.Fatalf("fork compressed = %v, %v; want the copied summary", cc, err)
	}
	if cc.Summary != "earlier discussion" || cc.ThroughIndex != 2 {
		t.Fatalf("fork compressed = %+v", cc)
	}

	// Source keeps its own summary and messages.
	if scc, _ := ms.Compressed(ctx, src.ID); scc == nil {
		t.Fatal("source compressed summary lost after fork")
	}
	if n, _ := ms.Count(ctx, src.ID); n != 3 {
		t.Fatalf("source message count = %d, want 3", n)
	}
}

// TestForkWithoutSummary verifies the summary copy is a no-op when the
// source has no compressed context.
func TestForkWithoutSummary(t *testing.T) {
	rt, ms := newTestRuntime(t)
	ctx := context.Background()

	src, _ := rt.Create(ctx, "u1")
	_ = ms.Append(ctx, src.ID, openagent.UserMessage("hello"))

	dst, err := rt.Fork(ctx, src.ID)
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	if cc, _ := ms.Compressed(ctx, dst.ID); cc != nil {
		t.Fatalf("fork compressed = %+v, want nil", cc)
	}
}

// TestMigrateFTSBackfillOnReopen pins the count-gated backfill contract:
// a database whose FTS index is missing rows (the pre-FTS upgrade case,
// simulated by wiping the index) gets re-indexed on open, and reopening
// an in-sync database leaves exactly one FTS row per non-empty message.
func TestMigrateFTSBackfillOnReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")

	ms, err := NewMessageStore(path)
	if err != nil {
		t.Fatalf("NewMessageStore: %v", err)
	}
	for _, c := range []string{"one", "two", "three"} {
		if err := ms.Append(ctx, "s1", openagent.UserMessage(c)); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	// Simulate an out-of-sync index: rows exist, FTS entries gone.
	if _, err := ms.DB().Exec(`DELETE FROM messages_fts`); err != nil {
		t.Fatalf("wipe fts: %v", err)
	}
	ms.Close()

	// Reopen: count mismatch must trigger the backfill.
	ms, err = NewMessageStore(path)
	if err != nil {
		t.Fatalf("reopen out-of-sync: %v", err)
	}
	var msgCount, ftsCount int
	if err := ms.DB().QueryRow(
		`SELECT COUNT(*) FROM messages WHERE content != ''`,
	).Scan(&msgCount); err != nil {
		t.Fatalf("count messages: %v", err)
	}
	if err := ms.DB().QueryRow(
		`SELECT COUNT(*) FROM messages_fts`,
	).Scan(&ftsCount); err != nil {
		t.Fatalf("count fts: %v", err)
	}
	if msgCount != 3 || ftsCount != msgCount {
		t.Fatalf("after backfill: messages=%d fts=%d, want 3/3", msgCount, ftsCount)
	}
	ms.Close()

	// Reopen again: in-sync index must stay exactly one row per message.
	ms, err = NewMessageStore(path)
	if err != nil {
		t.Fatalf("reopen in-sync: %v", err)
	}
	defer ms.Close()
	if err := ms.DB().QueryRow(
		`SELECT COUNT(*) FROM messages_fts`,
	).Scan(&ftsCount); err != nil {
		t.Fatalf("count fts: %v", err)
	}
	if ftsCount != msgCount {
		t.Fatalf("in-sync reopen duplicated fts rows: fts=%d messages=%d", ftsCount, msgCount)
	}
}

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

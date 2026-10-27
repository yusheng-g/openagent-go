package session

import (
	"context"
	"strings"
	"testing"

	openagent "github.com/yusheng-g/openagent-go"
)

func TestSessionRuntime_Fork(t *testing.T) {
	meta := &fakeStore{}
	msgs := &fakeMessages{}
	rt := NewRuntime(meta, msgs)
	ctx := context.Background()

	src, err := rt.Create(ctx, "u1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	srcInfo := meta.items[src.ID]
	srcInfo.Title = "debugging"
	srcInfo.SetMeta("mode", "manual")
	srcInfo.SetMeta("checkpoint_msgs", int64(3))
	srcInfo.SetMeta("total_tokens", 1234)
	if err := rt.Save(ctx, srcInfo); err != nil {
		t.Fatalf("Save: %v", err)
	}
	for _, c := range []string{"one", "two", "three"} {
		if err := msgs.Append(ctx, src.ID, openagent.UserMessage(c)); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	dst, err := rt.Fork(ctx, src.ID)
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	if dst.ID == src.ID || !strings.HasPrefix(dst.ID, "sess-") {
		t.Fatalf("fork ID = %q, want a fresh sess-* ID", dst.ID)
	}
	if dst.Title != "debugging (forked)" {
		t.Fatalf("fork title = %q", dst.Title)
	}

	copied := msgs.bySession[dst.ID]
	if len(copied) != 3 {
		t.Fatalf("fork copied %d messages, want 3", len(copied))
	}
	for i, want := range []string{"one", "two", "three"} {
		if copied[i].Content != want {
			t.Fatalf("copied[%d] = %q, want %q", i, copied[i].Content, want)
		}
	}

	if got, _ := dst.Meta["forked_from"]; got != src.ID {
		t.Fatalf("forked_from = %v, want %q", got, src.ID)
	}
	if _, ok := dst.Meta["mode"]; !ok {
		t.Fatal("mode not carried over")
	}
	for _, k := range []string{"checkpoint_msgs", "checkpoint_at", "total_tokens", "created_at"} {
		if _, ok := dst.Meta[k]; ok {
			t.Fatalf("meta key %q must not carry over to a fork", k)
		}
	}

	if _, ok := meta.items[dst.ID]; !ok {
		t.Fatal("fork metadata not saved")
	}
	if _, ok := msgs.bySession[src.ID]; !ok || len(msgs.bySession[src.ID]) != 3 {
		t.Fatal("source messages were modified")
	}
	if meta.items[src.ID].Title != "debugging" {
		t.Fatalf("source title = %q", meta.items[src.ID].Title)
	}
}

func TestSessionRuntime_ForkMissingSession(t *testing.T) {
	rt := NewRuntime(&fakeStore{}, &fakeMessages{})
	_, err := rt.Fork(context.Background(), "nope")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Fork error = %v, want not-found", err)
	}
}

func TestSessionRuntime_ForkEmptyTitleStaysEmpty(t *testing.T) {
	rt := NewRuntime(&fakeStore{}, &fakeMessages{})
	ctx := context.Background()
	src, _ := rt.Create(ctx, "")

	dst, err := rt.Fork(ctx, src.ID)
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	if dst.Title != "" {
		t.Fatalf("fork title = %q, want empty", dst.Title)
	}
}

// TestSessionRuntime_ForkOfForkTitleNoAccumulation pins the title suffix
// rule: forking a fork must not stack " (forked)" markers.
func TestSessionRuntime_ForkOfForkTitleNoAccumulation(t *testing.T) {
	meta := &fakeStore{}
	msgs := &fakeMessages{}
	rt := NewRuntime(meta, msgs)
	ctx := context.Background()

	src, _ := rt.Create(ctx, "u1")
	srcInfo := meta.items[src.ID]
	srcInfo.Title = "debugging"
	_ = rt.Save(ctx, srcInfo)

	f1, err := rt.Fork(ctx, src.ID)
	if err != nil {
		t.Fatalf("first Fork: %v", err)
	}
	if f1.Title != "debugging (forked)" {
		t.Fatalf("first fork title = %q", f1.Title)
	}
	f2, err := rt.Fork(ctx, f1.ID)
	if err != nil {
		t.Fatalf("second Fork: %v", err)
	}
	if f2.Title != "debugging (forked)" {
		t.Fatalf("fork-of-fork title = %q, want single suffix", f2.Title)
	}
}

func TestSessionRuntime_ForkNoMessages(t *testing.T) {
	meta := &fakeStore{}
	msgs := &fakeMessages{}
	rt := NewRuntime(meta, msgs)
	ctx := context.Background()
	src, _ := rt.Create(ctx, "")

	dst, err := rt.Fork(ctx, src.ID)
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	if len(msgs.bySession[dst.ID]) != 0 {
		t.Fatalf("fork copied %d messages from an empty session", len(msgs.bySession[dst.ID]))
	}
}

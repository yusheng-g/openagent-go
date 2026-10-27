package acp

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	openagent "github.com/yusheng-g/openagent-go"
	openacp "github.com/yusheng-g/openagent-go/acp/sdk"
	"github.com/yusheng-g/openagent-go/session"
	sessionsqlite "github.com/yusheng-g/openagent-go/session/sqlite"
)

// newForkServer returns an AgentServer whose Runtime is backed by a real
// SQLite store in a temp dir.
func newForkServer(t *testing.T) (*AgentServer, *session.DefaultRuntime) {
	t.Helper()
	ms, err := sessionsqlite.NewMessageStore(filepath.Join(t.TempDir(), "fork.db"))
	if err != nil {
		t.Fatalf("NewMessageStore: %v", err)
	}
	t.Cleanup(func() { ms.Close() })
	store, err := sessionsqlite.New(ms.DB())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	rt := session.NewRuntime(store, ms)
	srv := newListMessagesServer(nil)
	srv.Runtime = rt
	return srv, rt
}

// TestOnForkSessionCopiesSession drives OnForkSession end to end over the
// SQLite backend and verifies the fork is a full copy the client can load.
func TestOnForkSessionCopiesSession(t *testing.T) {
	srv, rt := newForkServer(t)
	ctx := context.Background()

	src, err := rt.Create(ctx, "u1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	src.Title = "debugging"
	src.SetMeta("mode", "manual")
	if err := rt.Save(ctx, src); err != nil {
		t.Fatalf("Save: %v", err)
	}
	for _, c := range []string{"one", "two", "three"} {
		if err := rt.Messages.Append(ctx, src.ID, openagent.UserMessage(c)); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	resp, err := srv.OnForkSession(ctx, openacp.ForkSessionRequest{SessionID: openacp.SessionId(src.ID)})
	if err != nil {
		t.Fatalf("OnForkSession: %v", err)
	}
	if resp.SessionID == openacp.SessionId(src.ID) {
		t.Fatal("fork returned the source session ID")
	}
	if resp.Meta["forked_from"] != src.ID {
		t.Fatalf("forked_from = %v, want %q", resp.Meta["forked_from"], src.ID)
	}

	forkID := string(resp.SessionID)
	if n, err := rt.Messages.Count(ctx, forkID); err != nil || n != 3 {
		t.Fatalf("forked message count = %d, %v; want 3", n, err)
	}
	got, err := rt.Meta.Get(ctx, forkID)
	if err != nil || got == nil {
		t.Fatalf("forked session metadata missing: %v, %v", got, err)
	}
	if got.Title != "debugging (forked)" {
		t.Fatalf("forked title = %q", got.Title)
	}
	if got.Meta["forked_from"] != src.ID || got.Meta["mode"] != "manual" {
		t.Fatalf("forked meta = %#v", got.Meta)
	}

	// RFD semantics: the fork is active once the response returns — it sits
	// in the live sessions map with the source's mode restored from the
	// copied meta, and the response carries session/new's shape so the
	// client can prompt right away.
	ss := srv.getSession(resp.SessionID)
	if ss == nil {
		t.Fatal("forked session is not in the live sessions map")
	}
	if ss.Mode() != "manual" {
		t.Fatalf("forked session mode = %q, want manual (restored from copied meta)", ss.Mode())
	}
	if len(resp.ConfigOptions) == 0 {
		t.Fatal("fork response must carry configOptions (session/new shape)")
	}
	if resp.Modes == nil || string(resp.Modes.CurrentModeID) != "manual" {
		t.Fatalf("fork response modes = %+v, want current manual", resp.Modes)
	}

	// The source keeps its own state untouched.
	if n, _ := rt.Messages.Count(ctx, src.ID); n != 3 {
		t.Fatalf("source message count = %d, want 3", n)
	}
	if srcMeta, _ := rt.Meta.Get(ctx, src.ID); srcMeta == nil || srcMeta.Title != "debugging" {
		t.Fatalf("source metadata changed: %+v", srcMeta)
	}
}

// TestOnForkSessionUnknownSession verifies the unknown-session error
// surfaces to the client.
func TestOnForkSessionUnknownSession(t *testing.T) {
	srv, _ := newForkServer(t)
	_, err := srv.OnForkSession(context.Background(), openacp.ForkSessionRequest{SessionID: "nope"})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("OnForkSession error = %v, want not-found", err)
	}
}

// TestOnForkSessionCwdFallback pins the RFD cwd contract: a request without
// cwd must fall back to the FORK's stored cwd (copied from the source),
// never to the user's home directory — NormalizePath("") maps to home, so
// the empty-check must happen before normalization.
func TestOnForkSessionCwdFallback(t *testing.T) {
	srv, rt := newForkServer(t)
	ctx := context.Background()

	src, err := rt.Create(ctx, "u1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	src.Cwd = "/tmp/fork-cwd-source"
	if err := rt.Save(ctx, src); err != nil {
		t.Fatalf("Save: %v", err)
	}

	resp, err := srv.OnForkSession(ctx, openacp.ForkSessionRequest{SessionID: openacp.SessionId(src.ID)})
	if err != nil {
		t.Fatalf("OnForkSession: %v", err)
	}
	ss := srv.getSession(resp.SessionID)
	if ss == nil {
		t.Fatal("forked session is not in the live sessions map")
	}
	if ss.cwd != "/tmp/fork-cwd-source" {
		t.Fatalf("forked cwd = %q, want the source's stored cwd /tmp/fork-cwd-source", ss.cwd)
	}
}

// TestOnForkSessionKeepsAdditionalDirs verifies the live fork keeps the
// source session's additional workspace roots (persisted by Runtime.Fork,
// restored into the agentSession).
func TestOnForkSessionKeepsAdditionalDirs(t *testing.T) {
	srv, rt := newForkServer(t)
	ctx := context.Background()

	src, err := rt.Create(ctx, "u1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	src.AdditionalDirectories = []string{"/tmp/fork-extra-a", "/tmp/fork-extra-b"}
	if err := rt.Save(ctx, src); err != nil {
		t.Fatalf("Save: %v", err)
	}

	resp, err := srv.OnForkSession(ctx, openacp.ForkSessionRequest{SessionID: openacp.SessionId(src.ID)})
	if err != nil {
		t.Fatalf("OnForkSession: %v", err)
	}
	ss := srv.getSession(resp.SessionID)
	if ss == nil {
		t.Fatal("forked session is not in the live sessions map")
	}
	want := []string{"/tmp/fork-extra-a", "/tmp/fork-extra-b"}
	if len(ss.additionalDirectories) != len(want) {
		t.Fatalf("forked additionalDirectories = %v, want %v", ss.additionalDirectories, want)
	}
	for i, d := range want {
		if ss.additionalDirectories[i] != d {
			t.Fatalf("forked additionalDirectories = %v, want %v", ss.additionalDirectories, want)
		}
	}
}

package kernel

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/agent"
)

// TestChildRegistry_SpawnReturnsStableID verifies spawn registers a child
// with a non-empty agent_id and a "sub-" prefixed sessionID.
func TestChildRegistry_SpawnReturnsStableID(t *testing.T) {
	reg := newChildRegistry()
	cfg := agent.New("explorer")
	child := reg.spawn(Deps{}, cfg, nil)

	if child.id == "" {
		t.Fatal("spawn should return a non-empty agent_id")
	}
	if !strings.HasPrefix(child.sessionID, "sub-") {
		t.Errorf("sessionID = %q, want sub- prefix", child.sessionID)
	}
	if !strings.Contains(child.sessionID, child.id) {
		t.Errorf("sessionID %q should contain agent_id %q", child.sessionID, child.id)
	}
}

// TestChildRegistry_GetHitsSameChild verifies get returns the same child
// that spawn registered.
func TestChildRegistry_GetHitsSameChild(t *testing.T) {
	reg := newChildRegistry()
	cfg := agent.New("explorer")
	child := reg.spawn(Deps{}, cfg, nil)

	got, ok := reg.get(child.id)
	if !ok {
		t.Fatalf("get(%q) not found", child.id)
	}
	if got != child {
		t.Error("get should return the same child pointer")
	}
}

// TestChildRegistry_GetUnknownIDReturnsFalse verifies an unknown id is not
// found.
func TestChildRegistry_GetUnknownIDReturnsFalse(t *testing.T) {
	reg := newChildRegistry()
	if _, ok := reg.get("nope-999"); ok {
		t.Error("get on unknown id should return false")
	}
}

// TestChildRegistry_SpawnIncrementsCounter verifies multiple spawns get
// distinct ids with incrementing counters.
func TestChildRegistry_SpawnIncrementsCounter(t *testing.T) {
	reg := newChildRegistry()
	cfg := agent.New("explorer")
	c1 := reg.spawn(Deps{}, cfg, nil)
	c2 := reg.spawn(Deps{}, cfg, nil)
	if c1.id == c2.id {
		t.Errorf("two spawns should produce distinct ids, both %q", c1.id)
	}
}

// TestMemSessionStore_AppendAndRecent verifies the in-memory store round-trips
// messages through Append + Recent.
func TestMemSessionStore_AppendAndRecent(t *testing.T) {
	store := newMemSessionStore(nil)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_ = store.Append(ctx, "sub-explorer-1", openagent.UserMessage("msg"))
	}
	n, _ := store.Count(ctx, "sub-explorer-1")
	if n != 5 {
		t.Errorf("Count = %d, want 5", n)
	}
	recent, _ := store.Recent(ctx, "sub-explorer-1", 3, 0)
	if len(recent) != 3 {
		t.Errorf("Recent len = %d, want 3", len(recent))
	}
}

// TestMemSessionStore_RecentAfter verifies RecentAfter returns messages after
// a throughIndex, skipping the compressed prefix.
func TestMemSessionStore_RecentAfter(t *testing.T) {
	store := newMemSessionStore(nil)
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		_ = store.Append(ctx, "s", openagent.UserMessage("msg"))
	}
	// throughIndex=3 → return messages [3:] up to n.
	got, _ := store.RecentAfter(ctx, "s", 3, 100)
	if len(got) != 7 {
		t.Errorf("RecentAfter(3, 100) len = %d, want 7", len(got))
	}
	// n caps the result.
	got, _ = store.RecentAfter(ctx, "s", 3, 4)
	if len(got) != 4 {
		t.Errorf("RecentAfter(3, 4) len = %d, want 4", len(got))
	}
}

// TestMemSessionStore_IsolatedFromParent verifies the child's in-memory store
// does not share state with a parent store — the whole point of "sub-agent
// sessions distinct from normal sessions".
func TestMemSessionStore_IsolatedFromParent(t *testing.T) {
	childStore := newMemSessionStore(nil)
	ctx := context.Background()
	_ = childStore.Append(ctx, "sub-explorer-1", openagent.UserMessage("child msg"))

	// A fresh store (simulating the parent's sqlite store) should not see
	// the child's messages.
	other := newMemSessionStore(nil)
	n, _ := other.Count(ctx, "sub-explorer-1")
	if n != 0 {
		t.Errorf("parent store should not see child messages; Count = %d", n)
	}
}

// TestMemSessionStore_Compaction verifies that with a stub summarizer,
// Compact produces a CompressedContext and RecentAfter skips the compressed
// prefix. The stub summarizer returns a fixed summary without calling a model.
func TestMemSessionStore_Compaction(t *testing.T) {
	store := newMemSessionStore(stubSummarizer{})
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		_ = store.Append(ctx, "s", openagent.UserMessage("msg"))
	}

	// Compress the first 5 messages.
	if err := store.Compact(ctx, "s", 5, nil); err != nil {
		t.Fatalf("Compact: %v", err)
	}
	cc, err := store.Compressed(ctx, "s")
	if err != nil {
		t.Fatalf("Compressed: %v", err)
	}
	if cc == nil {
		t.Fatal("CompressedContext should be non-nil after Compact")
	}
	if cc.Summary == "" {
		t.Error("Summary should be non-empty")
	}
	if cc.ThroughIndex < 5 {
		t.Errorf("ThroughIndex = %d, want >= 5", cc.ThroughIndex)
	}

	// Messages are never deleted — Count stays 10.
	n, _ := store.Count(ctx, "s")
	if n != 10 {
		t.Errorf("Count after compact = %d, want 10 (messages never deleted)", n)
	}

	// RecentAfter(throughIndex) skips the compressed prefix.
	got, _ := store.RecentAfter(ctx, "s", cc.ThroughIndex, 100)
	if len(got) != 10-cc.ThroughIndex {
		t.Errorf("RecentAfter(throughIndex) len = %d, want %d", len(got), 10-cc.ThroughIndex)
	}
}

// TestMemSessionStore_CompactNoSummarizerIsNoop verifies Compact is a no-op
// (not an error) when no summarizer is configured.
func TestMemSessionStore_CompactNoSummarizerIsNoop(t *testing.T) {
	store := newMemSessionStore(nil)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_ = store.Append(ctx, "s", openagent.UserMessage("msg"))
	}
	if err := store.Compact(ctx, "s", 3, nil); err != nil {
		t.Errorf("Compact with nil summarizer should be no-op, got %v", err)
	}
	cc, _ := store.Compressed(ctx, "s")
	if cc != nil {
		t.Error("CompressedContext should be nil when no summarizer")
	}
}

// TestSendTool_UnknownAgentID verifies sub_agent_send returns a clear error
// for an unknown agent_id, steering the model to launch a fresh sub-agent.
func TestSendTool_UnknownAgentID(t *testing.T) {
	reg := newChildRegistry()
	st := newSendTool(reg)
	res := st.Execute(context.Background(), mustJSON(t, SendParams{
		AgentID: "nope-999",
		Message: "hello",
	}))
	if res.Error == nil {
		t.Fatal("expected error for unknown agent_id")
	}
	if !strings.Contains(res.Error.Message, "not found") {
		t.Errorf("error should mention 'not found', got %q", res.Error.Message)
	}
}

// TestSendTool_ConcurrentBusy verifies that two concurrent Execute calls on
// the same child don't interleave — the second reports "busy".
func TestSendTool_ConcurrentBusy(t *testing.T) {
	reg := newChildRegistry()
	// Pre-register a child and mark it running to simulate an in-flight call.
	child := reg.spawn(Deps{}, agent.New("explorer"), nil)
	child.running = true
	child.mu.Lock()
	child.running = true
	child.mu.Unlock()

	st := newSendTool(reg)
	res := st.Execute(context.Background(), mustJSON(t, SendParams{
		AgentID: child.id,
		Message: "follow-up",
	}))
	if res.Error == nil || !strings.Contains(res.Error.Message, "still processing") {
		t.Errorf("expected 'still processing' error, got %v", res.Error)
	}
}

// --- stubs ---

// stubSummarizer implements openagent.Summarizer without calling a model.
type stubSummarizer struct{}

func (stubSummarizer) Summarize(_ context.Context, _ []openagent.Message, _ *openagent.CompressedContext) (*openagent.CompressedContext, error) {
	return &openagent.CompressedContext{Summary: "stub summary"}, nil
}

// mustJSON marshals v to json.RawMessage, failing the test on error.
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Ensure runChild with a stable sessionID reuses history: this is verified
// structurally by TestMemSessionStore_RecentAfter (history persists) +
// TestChildRegistry_GetHitsSameChild (same child reused) — a full runChild
// round-trip needs a model and is covered by the CLI smoke test instead.

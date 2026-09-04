package acp

import (
	"context"
	"errors"
	"testing"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	openacp "github.com/yusheng-g/openagent-go/acp/sdk"
	"github.com/yusheng-g/openagent-go/agent"
	"github.com/yusheng-g/openagent-go/kernel"
	"github.com/yusheng-g/openagent-go/session"
)

// fakeMsgStore is an in-memory session.SessionStore that records the
// Recent parameters it is called with.
type fakeMsgStore struct {
	recentN, recentOff int
	msgs               []openagent.Message
	err                error
}

func (f *fakeMsgStore) Append(context.Context, string, openagent.Message) error { return nil }
func (f *fakeMsgStore) Recent(_ context.Context, _ string, n, offset int) ([]openagent.Message, error) {
	f.recentN, f.recentOff = n, offset
	if f.err != nil {
		return nil, f.err
	}
	return f.msgs, nil
}
func (f *fakeMsgStore) RecentAfter(context.Context, string, int, int) ([]openagent.Message, error) {
	return nil, nil
}
func (f *fakeMsgStore) Count(context.Context, string) (int, error)  { return len(f.msgs), nil }
func (f *fakeMsgStore) DeleteSession(context.Context, string) error { return nil }

// fakeMetaStore implements the session.Store metadata interface required by
// NewAgentServer.
type fakeMetaStore struct{}

func (f *fakeMetaStore) Save(context.Context, session.SessionInfo) error { return nil }
func (f *fakeMetaStore) Get(context.Context, string) (*session.SessionInfo, error) {
	return nil, nil
}
func (f *fakeMetaStore) List(context.Context) ([]session.SessionInfo, error) { return nil, nil }
func (f *fakeMetaStore) Delete(context.Context, string) error                { return nil }
func (f *fakeMetaStore) Close() error                                        { return nil }

func newListMessagesServer(ms *fakeMsgStore) *AgentServer {
	deps := kernel.Deps{}
	if ms != nil {
		deps.SessionStore = ms
	}
	return NewAgentServer(agent.New("t"), deps, &fakeMetaStore{}, nil)
}

func TestOnListMessagesMapsAndForwards(t *testing.T) {
	ts := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	ms := &fakeMsgStore{msgs: []openagent.Message{
		{Role: openagent.RoleUser, Content: "hello", CreatedAt: &ts},
		{Role: openagent.RoleAssistant, Content: "hi", ReasoningContent: "think",
			ToolCalls: []openagent.ToolCall{{ID: "tc1", Function: openagent.ToolCallFunction{Name: "bash", Arguments: `{"cmd":"ls"}`}}}},
	}}
	srv := newListMessagesServer(ms)

	resp, err := srv.OnListMessages(context.Background(), openacp.ListMessagesRequest{SessionID: "s1"})
	if err != nil {
		t.Fatalf("OnListMessages: %v", err)
	}
	// Default limit 50, no paging.
	if ms.recentN != 50 || ms.recentOff != 0 {
		t.Errorf("Recent called with (%d, %d), want (50, 0)", ms.recentN, ms.recentOff)
	}
	if len(resp.Messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(resp.Messages))
	}
	first := resp.Messages[0]
	if first.Role != "user" || first.Content != "hello" || first.CreatedAt != ts.UTC().Format(time.RFC3339Nano) {
		t.Errorf("user message mapped wrong: %+v", first)
	}
	second := resp.Messages[1]
	if second.Role != "assistant" || second.ReasoningContent != "think" {
		t.Errorf("assistant message mapped wrong: %+v", second)
	}
	if len(second.ToolCalls) != 1 || second.ToolCalls[0] != (openacp.ToolCallRef{
		ID: "tc1", Name: "bash", Args: `{"cmd":"ls"}`,
	}) {
		t.Errorf("tool call mapped wrong: %+v", second.ToolCalls)
	}
}

func TestOnListMessagesPagingAndLimitCap(t *testing.T) {
	ms := &fakeMsgStore{msgs: []openagent.Message{{Role: openagent.RoleUser, Content: "x"}}}
	srv := newListMessagesServer(ms)

	if _, err := srv.OnListMessages(context.Background(), openacp.ListMessagesRequest{
		SessionID: "s1", Limit: 999, Before: 5,
	}); err != nil {
		t.Fatalf("OnListMessages: %v", err)
	}
	if ms.recentN != 200 {
		t.Errorf("limit not capped to 200, got %d", ms.recentN)
	}
	if ms.recentOff != 5 {
		t.Errorf("before not forwarded, got %d", ms.recentOff)
	}
}

func TestOnListMessagesNilMemoryAndStoreError(t *testing.T) {
	// No SessionStore wired (--memory=off): empty list, non-nil slice.
	srv := newListMessagesServer(nil)
	resp, err := srv.OnListMessages(context.Background(), openacp.ListMessagesRequest{SessionID: "s1"})
	if err != nil {
		t.Fatalf("nil-memory OnListMessages: %v", err)
	}
	if resp.Messages == nil || len(resp.Messages) != 0 {
		t.Errorf("nil memory should return empty list, got %#v", resp.Messages)
	}

	// Store error propagates.
	ms := &fakeMsgStore{err: errors.New("boom")}
	srv2 := newListMessagesServer(ms)
	if _, err := srv2.OnListMessages(context.Background(), openacp.ListMessagesRequest{SessionID: "s1"}); err == nil {
		t.Error("store error should propagate")
	}
}

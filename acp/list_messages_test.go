package acp

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	openacp "github.com/yusheng-g/openagent-go/acp/sdk"
	"github.com/yusheng-g/openagent-go/agent"
	"github.com/yusheng-g/openagent-go/kernel"
	"github.com/yusheng-g/openagent-go/session"
)

// fakeMsgStore is an in-memory session.SessionStore with position-exact
// RecentAfter semantics (window = [start, start+n)), recording the window
// it was last asked for.
type fakeMsgStore struct {
	afterStart, afterN int
	msgs               []openagent.Message
	err                error
}

func (f *fakeMsgStore) Append(context.Context, string, openagent.Message) error { return nil }
func (f *fakeMsgStore) Recent(_ context.Context, _ string, _, _ int) ([]openagent.Message, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.msgs, nil
}
func (f *fakeMsgStore) RecentAfter(_ context.Context, _ string, start, n int) ([]openagent.Message, error) {
	f.afterStart, f.afterN = start, n
	if f.err != nil {
		return nil, f.err
	}
	if start >= len(f.msgs) || n <= 0 {
		return nil, nil
	}
	end := start + n
	if end > len(f.msgs) {
		end = len(f.msgs)
	}
	return f.msgs[start:end], nil
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

// numberedMsgs builds n user messages with content "m<i>", so a test can
// assert which session positions came back in a page.
func numberedMsgs(n int) []openagent.Message {
	msgs := make([]openagent.Message, n)
	for i := range msgs {
		msgs[i] = openagent.Message{Role: openagent.RoleUser, Content: fmt.Sprintf("m%d", i)}
	}
	return msgs
}

// pageIndexes returns the Index field of every message in the page.
func pageIndexes(msgs []openacp.Message) []int {
	out := make([]int, len(msgs))
	for i, m := range msgs {
		out[i] = m.Index
	}
	return out
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
	// Default limit 50 over a 2-message store: full window from position 0.
	if ms.afterStart != 0 || ms.afterN != 2 {
		t.Errorf("RecentAfter called with (%d, %d), want (0, 2)", ms.afterStart, ms.afterN)
	}
	if len(resp.Messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(resp.Messages))
	}
	if resp.Total != 2 || resp.HasMore {
		t.Errorf("total/hasMore = %d/%v, want 2/false", resp.Total, resp.HasMore)
	}
	first := resp.Messages[0]
	if first.Role != "user" || first.Content != "hello" || first.Index != 0 || first.CreatedAt != ts.UTC().Format(time.RFC3339Nano) {
		t.Errorf("user message mapped wrong: %+v", first)
	}
	second := resp.Messages[1]
	if second.Role != "assistant" || second.ReasoningContent != "think" || second.Index != 1 {
		t.Errorf("assistant message mapped wrong: %+v", second)
	}
	if len(second.ToolCalls) != 1 || second.ToolCalls[0] != (openacp.ToolCallRef{
		ID: "tc1", Name: "bash", Args: `{"cmd":"ls"}`,
	}) {
		t.Errorf("tool call mapped wrong: %+v", second.ToolCalls)
	}
}

func TestOnListMessagesPositionalPaging(t *testing.T) {
	srv := newListMessagesServer(&fakeMsgStore{msgs: numberedMsgs(10)})

	cases := []struct {
		name        string
		req         openacp.ListMessagesRequest
		wantIndexes []int
		wantHasMore bool
	}{
		{"tail page", openacp.ListMessagesRequest{SessionID: "s1", Limit: 3}, []int{7, 8, 9}, true},
		{"before cursor", openacp.ListMessagesRequest{SessionID: "s1", Limit: 3, Before: 7}, []int{4, 5, 6}, true},
		{"before first page", openacp.ListMessagesRequest{SessionID: "s1", Limit: 3, Before: 2}, []int{0, 1}, false},
		{"before beyond total clamps to tail", openacp.ListMessagesRequest{SessionID: "s1", Limit: 3, Before: 99}, []int{7, 8, 9}, true},
		{"before equals total is the tail page", openacp.ListMessagesRequest{SessionID: "s1", Limit: 3, Before: 10}, []int{7, 8, 9}, true},
	}
	for _, tc := range cases {
		resp, err := srv.OnListMessages(context.Background(), tc.req)
		if err != nil {
			t.Fatalf("%s: OnListMessages: %v", tc.name, err)
		}
		if resp.Total != 10 {
			t.Errorf("%s: total = %d, want 10", tc.name, resp.Total)
		}
		if got := pageIndexes(resp.Messages); !slices.Equal(got, tc.wantIndexes) {
			t.Errorf("%s: indexes = %v, want %v", tc.name, got, tc.wantIndexes)
		}
		if resp.HasMore != tc.wantHasMore {
			t.Errorf("%s: hasMore = %v, want %v", tc.name, resp.HasMore, tc.wantHasMore)
		}
	}
}

func TestOnListMessagesLimitCap(t *testing.T) {
	srv := newListMessagesServer(&fakeMsgStore{msgs: numberedMsgs(250)})

	resp, err := srv.OnListMessages(context.Background(), openacp.ListMessagesRequest{
		SessionID: "s1", Limit: 999,
	})
	if err != nil {
		t.Fatalf("OnListMessages: %v", err)
	}
	if len(resp.Messages) != 200 {
		t.Errorf("limit not capped to 200, got %d messages", len(resp.Messages))
	}
	if resp.Messages[0].Index != 50 {
		t.Errorf("capped page should start at position 50, got %d", resp.Messages[0].Index)
	}
	if !resp.HasMore {
		t.Error("capped page should report hasMore")
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

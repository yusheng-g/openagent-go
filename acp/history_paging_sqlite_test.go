package acp

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	openagent "github.com/yusheng-g/openagent-go"
	openacp "github.com/yusheng-g/openagent-go/acp/sdk"
	"github.com/yusheng-g/openagent-go/session/sqlite"
)

// TestHistoryPagingOverSQLiteStore exercises the paging paths against the
// real sqlite MessageStore (not the fake): the core risk this guards is
// the position space — session-relative positions must line up with what
// RecentAfter/Count return from the actual schema (AUTOINCREMENT rowid
// order), and paging must NOT exhibit the orphan-tool trimming that
// Recent applies to window starts (which would desync client cursors).
func TestHistoryPagingOverSQLiteStore(t *testing.T) {
	store, err := sqlite.NewMessageStore(filepath.Join(t.TempDir(), "memory.db"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	const sid = "sess-e2e"

	// 12 messages. Positions 0-1 form an assistant tool_call + tool
	// result pair — the pair straddling a window start is exactly where
	// Recent's orphan trimming would drop a message and shift positions.
	msgs := alternatingMsgs(12)
	msgs[0] = openagent.Message{Role: openagent.RoleAssistant, ToolCalls: []openagent.ToolCall{{
		ID: "call-1", Type: "function",
		Function: openagent.ToolCallFunction{Name: "bash", Arguments: `{"cmd":"ls"}`},
	}}}
	msgs[1] = openagent.Message{Role: openagent.RoleTool, ToolCallID: "call-1", Content: "tool output"}
	for _, m := range msgs {
		if err := store.Append(ctx, sid, m); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	srv := newListMessagesServer(nil)
	srv.Mem = store
	sender := &recordingSender{}

	// Window [1, 4) starts at the tool result. Recent would trim it as an
	// orphan; RecentAfter must keep the position space dense.
	resp, err := srv.OnListMessages(ctx, openacp.ListMessagesRequest{
		SessionID: sid, Limit: 3, Before: 4,
	})
	if err != nil {
		t.Fatalf("OnListMessages: %v", err)
	}
	if got := pageIndexes(resp.Messages); !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("indexes = %v, want [1 2 3] — orphan trimming must not shift positions", got)
	}
	if resp.Total != 12 || !resp.HasMore {
		t.Errorf("total/hasMore = %d/%v, want 12/true", resp.Total, resp.HasMore)
	}

	// Page all the way down: beforeIndex=2 keeps the tool pair intact.
	resp, err = srv.OnListMessages(ctx, openacp.ListMessagesRequest{
		SessionID: sid, Limit: 5, Before: 2,
	})
	if err != nil {
		t.Fatalf("OnListMessages: %v", err)
	}
	if got := pageIndexes(resp.Messages); !slices.Equal(got, []int{0, 1}) {
		t.Errorf("indexes = %v, want [0 1]", got)
	}
	if resp.HasMore {
		t.Error("first page should report hasMore=false")
	}

	// Tail replay over the real store: 3-message window, absolute ids,
	// bounds reported in _meta.
	lresp, err := srv.OnLoadSession(ctx, openacp.LoadSessionRequest{
		SessionID: sid, ReplayLimit: 3,
	}, sender)
	if err != nil {
		t.Fatalf("OnLoadSession: %v", err)
	}
	events := sender.historySnapshot()
	if len(events) == 0 || events[0].mid != "hist_9" || events[len(events)-1].mid != "hist_11" {
		t.Errorf("tail replay ids = %v, want hist_9..hist_11", events)
	}
	if got, ok := lresp.Meta["history_total"].(int); !ok || got != 12 {
		t.Errorf("meta history_total = %#v, want 12", lresp.Meta["history_total"])
	}
	if got, ok := lresp.Meta["replay_start_index"].(int); !ok || got != 9 {
		t.Errorf("meta replay_start_index = %#v, want 9", lresp.Meta["replay_start_index"])
	}

	// Full replay default: dense ids from hist_0, no paging meta keys.
	// Positions 0-1 emit tool_call updates (not history-message events),
	// so the first recorded history event is position 2.
	sender2 := &recordingSender{}
	lresp2, err := srv.OnLoadSession(ctx, openacp.LoadSessionRequest{SessionID: sid}, sender2)
	if err != nil {
		t.Fatalf("OnLoadSession: %v", err)
	}
	events2 := sender2.historySnapshot()
	if len(events2) == 0 || events2[0] != (historyEvent{"user_message_chunk", "u2", "hist_2"}) {
		t.Errorf("full replay first event = %v, want user u2 at hist_2", events2[:1])
	}
	if _, ok := lresp2.Meta["replay_start_index"]; ok {
		t.Error("full replay should not report paging meta")
	}

	// The tool pair at 0-1 replays as tool_call updates: pending (the
	// assistant's call) then completed (the result).
	var sawPending, sawCompleted bool
	for _, u := range sender2.snapshot() {
		if u.kind == "tool_call" && u.toolID == "call-1" {
			switch u.status {
			case "pending":
				sawPending = true
			case "completed":
				sawCompleted = true
			}
		}
	}
	if !sawPending || !sawCompleted {
		t.Errorf("tool pair replay: pending=%v completed=%v, want both", sawPending, sawCompleted)
	}
}

package acp

import (
	"context"
	"strconv"
	"testing"

	openagent "github.com/yusheng-g/openagent-go"
	openacp "github.com/yusheng-g/openagent-go/acp/sdk"
)

// alternatingMsgs builds n messages alternating user ("u<i>") and
// assistant ("a<i>") so a replay test can assert kind and position.
func alternatingMsgs(n int) []openagent.Message {
	msgs := make([]openagent.Message, n)
	for i := range msgs {
		if i%2 == 0 {
			msgs[i] = openagent.Message{Role: openagent.RoleUser, Content: "u" + strconv.Itoa(i)}
		} else {
			msgs[i] = openagent.Message{Role: openagent.RoleAssistant, Content: "a" + strconv.Itoa(i)}
		}
	}
	return msgs
}

func TestOnLoadSessionTailReplay(t *testing.T) {
	srv := newListMessagesServer(&fakeMsgStore{msgs: alternatingMsgs(6)})
	sender := &recordingSender{}

	resp, err := srv.OnLoadSession(context.Background(), openacp.LoadSessionRequest{
		SessionID: "s1", ReplayLimit: 2,
	}, sender)
	if err != nil {
		t.Fatalf("OnLoadSession: %v", err)
	}

	// Only the last two messages replay, addressed by absolute position.
	want := []historyEvent{
		{"user_message_chunk", "u4", "hist_4"},
		{"agent_message_chunk", "a5", "hist_5"},
	}
	events := sender.historySnapshot()
	if len(events) != len(want) {
		t.Fatalf("replayed %d events, want %d: %+v", len(events), len(want), events)
	}
	for i, ev := range want {
		if events[i] != ev {
			t.Errorf("event[%d] = %+v, want %+v", i, events[i], ev)
		}
	}

	// The response _meta reports the window bounds for client backfill.
	if got, ok := resp.Meta["history_total"].(int); !ok || got != 6 {
		t.Errorf("meta history_total = %#v, want 6", resp.Meta["history_total"])
	}
	if got, ok := resp.Meta["replay_start_index"].(int); !ok || got != 4 {
		t.Errorf("meta replay_start_index = %#v, want 4", resp.Meta["replay_start_index"])
	}
}

func TestOnLoadSessionFullReplayDefault(t *testing.T) {
	srv := newListMessagesServer(&fakeMsgStore{msgs: alternatingMsgs(6)})
	sender := &recordingSender{}

	resp, err := srv.OnLoadSession(context.Background(), openacp.LoadSessionRequest{
		SessionID: "s1",
	}, sender)
	if err != nil {
		t.Fatalf("OnLoadSession: %v", err)
	}

	// No replayLimit: full history replays, ids are the absolute
	// positions 0..5, and the legacy meta shape is unchanged — no
	// paging keys are added for clients that never opted in.
	events := sender.historySnapshot()
	if len(events) != 6 {
		t.Fatalf("replayed %d events, want 6", len(events))
	}
	for i, ev := range events {
		if want := "hist_" + strconv.Itoa(i); ev.mid != want {
			t.Errorf("event[%d].mid = %q, want %q", i, ev.mid, want)
		}
	}
	if _, ok := resp.Meta["history_total"]; ok {
		t.Error("full replay should not report history_total")
	}
	if _, ok := resp.Meta["replay_start_index"]; ok {
		t.Error("full replay should not report replay_start_index")
	}
}

func TestOnLoadSessionReplayLimitCoversAll(t *testing.T) {
	// replayLimit >= total degrades to a full replay: start position 0,
	// every id present, hasMore-style backfill not needed.
	srv := newListMessagesServer(&fakeMsgStore{msgs: alternatingMsgs(6)})
	sender := &recordingSender{}

	resp, err := srv.OnLoadSession(context.Background(), openacp.LoadSessionRequest{
		SessionID: "s1", ReplayLimit: 10,
	}, sender)
	if err != nil {
		t.Fatalf("OnLoadSession: %v", err)
	}
	if events := sender.historySnapshot(); len(events) != 6 {
		t.Fatalf("replayed %d events, want 6", len(events))
	}
	if got, ok := resp.Meta["replay_start_index"].(int); !ok || got != 0 {
		t.Errorf("meta replay_start_index = %#v, want 0", resp.Meta["replay_start_index"])
	}
}

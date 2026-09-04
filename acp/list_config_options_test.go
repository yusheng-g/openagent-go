package acp

import (
	"context"
	"testing"

	openacp "github.com/yusheng-g/openagent-go/acp/sdk"
)

// TestOnListConfigOptionsDefaults guards the session-less shape of the
// method: with no session state, the handler degrades to the defaults a
// fresh session would get (mode/thought_level selects, plus the model
// selector when the agent has models configured).
func TestOnListConfigOptionsDefaults(t *testing.T) {
	srv := newListMessagesServer(nil)

	resp, err := srv.OnListConfigOptions(context.Background(), openacp.ListConfigOptionsRequest{})
	if err != nil {
		t.Fatalf("OnListConfigOptions: %v", err)
	}

	byID := map[string]openacp.SessionConfigOption{}
	for _, o := range resp.ConfigOptions {
		byID[o.ID] = o
	}
	mode, ok := byID["mode"]
	if !ok {
		t.Fatal("config options must include the mode select")
	}
	if mode.CurrentValue != "auto" {
		t.Errorf("mode currentValue = %v, want auto (default)", mode.CurrentValue)
	}
	tl, ok := byID["thought_level"]
	if !ok {
		t.Fatal("config options must include the thought_level select")
	}
	if tl.CurrentValue != "medium" {
		t.Errorf("thought_level currentValue = %v, want medium (default)", tl.CurrentValue)
	}
	if model, ok := byID["model"]; ok {
		if len(model.Options) == 0 {
			t.Error("model select must carry its selectable values")
		}
		if model.CurrentValue == "" {
			t.Error("model currentValue must name the default model")
		}
	}
	// The request is session-less: the response must not be bound to any
	// session state (repeated calls stay identical).
	resp2, err := srv.OnListConfigOptions(context.Background(), openacp.ListConfigOptionsRequest{})
	if err != nil {
		t.Fatalf("second OnListConfigOptions: %v", err)
	}
	if len(resp2.ConfigOptions) != len(resp.ConfigOptions) {
		t.Errorf("repeated calls diverge: %d vs %d options",
			len(resp.ConfigOptions), len(resp2.ConfigOptions))
	}
}

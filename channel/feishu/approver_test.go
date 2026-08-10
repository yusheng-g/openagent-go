package feishu

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
)

func TestBuildApprovalCard(t *testing.T) {
	cardJSON, err := buildApprovalCard("abc123", "shell", `{"command":"rm -rf /tmp"}`)
	if err != nil {
		t.Fatal(err)
	}

	var card map[string]any
	if err := json.Unmarshal([]byte(cardJSON), &card); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Header
	assertPath(t, card, "header.title.content", "需要权限")
	assertPath(t, card, "header.template", "blue")
	assertPath(t, card, "schema", "2.0")

	// No banner image — first body element is markdown, not img.
	body, _ := card["body"].(map[string]any)
	elems, _ := body["elements"].([]any)
	first, _ := elems[0].(map[string]any)
	if first["tag"] != "markdown" {
		t.Errorf("first element tag = %v, want markdown (no banner image)", first["tag"])
	}

	// Markdown contains tool name and args.
	md, _ := first["content"].(string)
	if !strings.Contains(md, "shell") {
		t.Error("markdown should contain tool name")
	}
	if !strings.Contains(md, "rm -rf /tmp") {
		t.Error("markdown should contain args")
	}

	// Approval ID embedded in all button values.
	if !strings.Contains(cardJSON, `"approval_id":"abc123"`) {
		t.Error("approval ID should be embedded in button values")
	}

	// Count buttons with approval_id — should be 3 (Allow Once/Allow Always/Deny).
	count := strings.Count(cardJSON, `"approval_id":"abc123"`)
	if count != 3 {
		t.Errorf("expected 3 buttons with approval_id, got %d", count)
	}
}

func TestBuildApprovalCardButtonStructure(t *testing.T) {
	cardJSON, err := buildApprovalCard("x", "read", `{"path":"/etc"}`)
	if err != nil {
		t.Fatal(err)
	}

	var card map[string]any
	json.Unmarshal([]byte(cardJSON), &card)

	body, _ := card["body"].(map[string]any)
	elems, _ := body["elements"].([]any)

	// elements: [markdown, hr, column_set]
	if len(elems) != 3 {
		t.Fatalf("expected 3 body elements, got %d", len(elems))
	}
	if elems[1].(map[string]any)["tag"] != "hr" {
		t.Error("second element should be hr")
	}

	colSet, _ := elems[2].(map[string]any)
	if colSet["tag"] != "column_set" {
		t.Errorf("third element should be column_set, got %v", colSet["tag"])
	}

	cols, _ := colSet["columns"].([]any)
	if len(cols) != 3 {
		t.Fatalf("expected 3 columns (buttons), got %d", len(cols))
	}

	// No form or hint — Feishu schema V2 doesn't support them.
	if strings.Contains(cardJSON, `"tag":"form"`) {
		t.Error("card should not contain a form")
	}
	if strings.Contains(cardJSON, "/mode") {
		t.Error("card should not contain /mode hint")
	}
}

func TestBuildResolvedCard(t *testing.T) {
	cardJSON, err := buildResolvedCard("shell", `{"command":"ls"}`, "✅ **已同意**", "")
	if err != nil {
		t.Fatal(err)
	}

	var card map[string]any
	json.Unmarshal([]byte(cardJSON), &card)

	assertPath(t, card, "header.template", "green")
	body, _ := card["body"].(map[string]any)
	elems, _ := body["elements"].([]any)

	// First element should be a collapsible_panel (collapsed).
	panel, _ := elems[0].(map[string]any)
	if panel["tag"] != "collapsible_panel" {
		t.Errorf("expected collapsible_panel, got %v", panel["tag"])
	}
	if expanded, _ := panel["expanded"].(bool); expanded {
		t.Error("panel should start collapsed")
	}

	// Panel header title contains decision + tool name.
	hdr, _ := panel["header"].(map[string]any)
	title, _ := hdr["title"].(map[string]any)
	content, _ := title["content"].(string)
	if !strings.Contains(content, "已同意") {
		t.Error("panel header should contain decision text")
	}
	if !strings.Contains(content, "shell") {
		t.Error("panel header should contain tool name")
	}

	// Panel body (hidden) contains the details.
	panelElems, _ := panel["elements"].([]any)
	md, _ := panelElems[0].(map[string]any)
	mdContent, _ := md["content"].(string)
	if !strings.Contains(mdContent, "shell") {
		t.Error("panel body should contain tool name details")
	}

	// No form in resolved card.
	if strings.Contains(cardJSON, `"tag":"form"`) {
		t.Error("resolved card should not contain a form")
	}
}

func TestBuildResolvedCardDeny(t *testing.T) {
	cardJSON, err := buildResolvedCard("shell", `{}`, "❌ **已拒绝**", "too dangerous")
	if err != nil {
		t.Fatal(err)
	}

	var card map[string]any
	json.Unmarshal([]byte(cardJSON), &card)

	assertPath(t, card, "header.template", "red")

	// Panel body should contain the rejection reason.
	body, _ := card["body"].(map[string]any)
	elems, _ := body["elements"].([]any)
	panel, _ := elems[0].(map[string]any)
	panelElems, _ := panel["elements"].([]any)
	md, _ := panelElems[0].(map[string]any)
	mdContent, _ := md["content"].(string)
	if !strings.Contains(mdContent, "too dangerous") {
		t.Error("panel body should contain rejection reason")
	}
}

func TestChatIDFromSession(t *testing.T) {
	tests := []struct {
		sessionID string
		want      string
	}{
		{"feishu_oc_abcd1234", "oc_abcd1234"},
		{"feishu_", ""},
		{"other_prefix", "other_prefix"},
		{"", ""},
	}
	for _, tt := range tests {
		got := chatIDFromSession(tt.sessionID)
		if got != tt.want {
			t.Errorf("chatIDFromSession(%q) = %q, want %q", tt.sessionID, got, tt.want)
		}
	}
}

func TestHandleCardAction(t *testing.T) {
	tests := []struct {
		name       string
		buttonName string
		wantAction string
		wantReason string
	}{
		{"allow_once", "allow_once", "allow", ""},
		{"allow_always", "allow_always", "allow", ""},
		{"deny", "deny", "deny", "rejected by user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ap := newFeishuApprover()
			resultCh := make(chan approvalResult, 1)
			approvalID := "test-" + tt.name
			ap.mu.Lock()
			ap.pending[approvalID] = &pendingApproval{
				result:   resultCh,
				toolName: "shell",
				args:     `{}`,
			}
			ap.mu.Unlock()

			event := &callback.CardActionTriggerEvent{
				Event: &callback.CardActionTriggerRequest{
					Action: &callback.CallBackAction{
						Name:  tt.buttonName,
						Value: map[string]any{"approval_id": approvalID, "action": tt.buttonName},
					},
				},
			}

			resp, err := ap.handleCardAction(nil, event)
			if err != nil {
				t.Fatal(err)
			}
			if resp == nil || resp.Toast == nil {
				t.Fatal("expected toast response")
			}

			select {
			case r := <-resultCh:
				if r.action != tt.wantAction {
					t.Errorf("action = %q, want %q", r.action, tt.wantAction)
				}
				if tt.wantReason != "" && r.reason != tt.wantReason {
					t.Errorf("reason = %q, want %q", r.reason, tt.wantReason)
				}
			default:
				t.Fatal("pending approval was not resolved")
			}
		})
	}
}

func TestHandleCardActionExpired(t *testing.T) {
	ap := newFeishuApprover()

	event := &callback.CardActionTriggerEvent{
		Event: &callback.CardActionTriggerRequest{
			Action: &callback.CallBackAction{
				Name:  "allow_once",
				Value: map[string]any{"approval_id": "nonexistent", "action": "allow_once"},
			},
		},
	}

	resp, err := ap.handleCardAction(nil, event)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Toast.Type != "warning" {
		t.Errorf("expired approval should return warning, got %s", resp.Toast.Type)
	}
}

func TestHandleCardActionIntegrated(t *testing.T) {
	// When integrated=true, handleCardAction should resolve the result
	// channel and return a toast, but NOT call patchResolved (no separate
	// card to patch). We verify the result is delivered correctly.
	ap := newFeishuApprover()
	resultCh := make(chan approvalResult, 1)
	ap.mu.Lock()
	ap.pending["int-1"] = &pendingApproval{
		result:     resultCh,
		toolName:   "shell",
		args:       `{}`,
		integrated: true,
	}
	ap.mu.Unlock()

	event := &callback.CardActionTriggerEvent{
		Event: &callback.CardActionTriggerRequest{
			Action: &callback.CallBackAction{
				Name:  "allow_once",
				Value: map[string]any{"approval_id": "int-1", "action": "allow_once"},
			},
		},
	}

	resp, err := ap.handleCardAction(nil, event)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil || resp.Toast == nil {
		t.Fatal("expected toast response")
	}
	if resp.Toast.Type != "success" {
		t.Errorf("expected success toast, got %s", resp.Toast.Type)
	}

	select {
	case r := <-resultCh:
		if r.action != "allow" {
			t.Errorf("action = %q, want allow", r.action)
		}
	default:
		t.Fatal("pending approval was not resolved")
	}
}

func TestSetClearRunCardUpdater(t *testing.T) {
	ap := newFeishuApprover()

	called := false
	cb := func(toolName, args, approvalID string) {
		called = true
	}

	ap.setRunCardUpdater("session-1", cb)

	// Verify lookup.
	ap.updaterMu.Lock()
	_, ok := ap.updaters["session-1"]
	ap.updaterMu.Unlock()
	if !ok {
		t.Fatal("updater should be registered")
	}

	// Verify the callback is invocable.
	ap.updaterMu.Lock()
	updater := ap.updaters["session-1"]
	ap.updaterMu.Unlock()
	if updater == nil {
		t.Fatal("updater should not be nil")
	}
	updater("shell", "{}", "test-id")
	if !called {
		t.Error("updater callback should have been called")
	}

	// Clear.
	ap.clearRunCardUpdater("session-1")
	ap.updaterMu.Lock()
	_, ok = ap.updaters["session-1"]
	ap.updaterMu.Unlock()
	if ok {
		t.Error("updater should be cleared")
	}
}

func TestHandleCardActionNilEvent(t *testing.T) {
	ap := newFeishuApprover()
	resp, err := ap.handleCardAction(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestResolvedDisplay(t *testing.T) {
	tests := []struct {
		r            approvalResult
		wantDecision string
		wantToast    string
	}{
		{approvalResult{action: "allow"}, "✅ **已同意**", "已同意"},
		{approvalResult{action: "deny"}, "❌ **已拒绝**", "已拒绝"},
	}
	for _, tt := range tests {
		decision, _, toast := resolvedDisplay(tt.r)
		if decision != tt.wantDecision {
			t.Errorf("decision = %q, want %q", decision, tt.wantDecision)
		}
		if toast != tt.wantToast {
			t.Errorf("toast = %q, want %q", toast, tt.wantToast)
		}
	}
}

func TestBuildApprovalCardACPButtons(t *testing.T) {
	cardJSON, err := buildApprovalCard("abc", "shell", `{"command":"ls"}`)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(cardJSON, "Allow Once") {
		t.Error("card should contain Allow Once button")
	}
	if !strings.Contains(cardJSON, "Allow Always") {
		t.Error("card should contain Allow Always button")
	}
	if !strings.Contains(cardJSON, "Deny") {
		t.Error("card should contain Deny button")
	}
	if !strings.Contains(cardJSON, `"action":"allow_once"`) {
		t.Error("card should contain allow_once action")
	}
	if !strings.Contains(cardJSON, `"action":"allow_always"`) {
		t.Error("card should contain allow_always action")
	}
	if !strings.Contains(cardJSON, `"action":"deny"`) {
		t.Error("card should contain deny action")
	}
}

// ── helpers ──

func assertPath(t *testing.T, m map[string]any, path, want string) {
	t.Helper()
	parts := strings.Split(path, ".")
	cur := any(m)
	for i, p := range parts {
		mp, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("at %q: expected map, got %T", strings.Join(parts[:i], "."), cur)
		}
		v, ok := mp[p]
		if !ok {
			t.Fatalf("path %q not found", path)
		}
		if i == len(parts)-1 {
			if s, ok := v.(string); !ok || s != want {
				t.Errorf("%s = %q, want %q", path, v, want)
			}
			return
		}
		cur = v
	}
}

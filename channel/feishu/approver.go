package feishu

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/governance"
)

// feishuApprover implements governance.HumanApprover for the Feishu channel.
// When the policy engine routes a tool call to the human layer, Ask sends an
// interactive approval card to the Feishu chat and blocks until the user
// clicks a button (or the context is cancelled).
//
// Card action callbacks (button clicks) arrive via the WebSocket event loop
// and are dispatched to handleCardAction, which resolves the pending
// approval. The approver and the Channel share the same lark.Client
// (created in Channel.Start).
type feishuApprover struct {
	client *lark.Client
	memory governance.ApprovalMemory // set by Channel.Approver; used by "allow_always"

	mu      sync.Mutex
	pending map[string]*pendingApproval // approvalID → pending entry

	updaterMu sync.Mutex
	updaters  map[string]func(toolName, args, approvalID string) // sessionID → run-card update callback
}

// pendingApproval pairs a result channel with the tool info needed to build
// the resolved card after the user clicks.
type pendingApproval struct {
	result     chan approvalResult
	toolName   string
	args       string
	integrated bool // true = buttons embedded in run card (no separate card to patch)
}

// approvalResult is the outcome extracted from a card action click.
type approvalResult struct {
	action string // "allow" | "deny"
	reason string // populated for deny
}

// approvalSeq generates process-unique approval IDs.
var approvalSeq uint64

// newFeishuApprover creates an approver with no client (set later by
// Channel.Start) and no memory (set later by Approver).
func newFeishuApprover() *feishuApprover {
	return &feishuApprover{
		pending:  make(map[string]*pendingApproval),
		updaters: make(map[string]func(toolName, args, approvalID string)),
	}
}

// Ask implements governance.HumanApprover. It sends an approval card to the
// Feishu chat and blocks until the user responds (or ctx is cancelled).
func (a *feishuApprover) Ask(ctx context.Context, call openagent.ToolCall, def openagent.FunctionDefinition, session openagent.Session) (governance.Decision, error) {
	if a.client == nil {
		return governance.Decision{Action: governance.Deny, Reason: "approval channel not ready"}, nil
	}

	chatID := chatIDFromSession(session.ID)
	if chatID == "" {
		return governance.Decision{Action: governance.Deny, Reason: "cannot determine chat ID from session"}, nil
	}

	approvalID := strconv.FormatUint(atomic.AddUint64(&approvalSeq, 1), 36)
	toolName := def.Name
	args := call.Function.Arguments

	// Check for a run-card updater (integrated path). When registered,
	// embed the approval buttons in the existing run card instead of
	// sending a separate approval card.
	a.updaterMu.Lock()
	updater := a.updaters[session.ID]
	a.updaterMu.Unlock()

	integrated := false
	if updater != nil {
		updater(toolName, args, approvalID)
		integrated = true
	} else {
		cardJSON, err := buildApprovalCard(approvalID, toolName, args)
		if err != nil {
			return governance.Decision{Action: governance.Deny, Reason: "build approval card: " + err.Error()}, nil
		}
		if _, err := a.sendInteractive(ctx, chatID, cardJSON); err != nil {
			return governance.Decision{Action: governance.Deny, Reason: "send approval card: " + err.Error()}, nil
		}
	}

	resultCh := make(chan approvalResult, 1)
	entry := &pendingApproval{result: resultCh, toolName: toolName, args: args, integrated: integrated}

	a.mu.Lock()
	a.pending[approvalID] = entry
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		delete(a.pending, approvalID)
		a.mu.Unlock()
	}()

	select {
	case <-ctx.Done():
		return governance.Decision{Action: governance.Deny, Reason: "approval cancelled: " + ctx.Err().Error()}, nil
	case r := <-resultCh:
		switch r.action {
		case "allow":
			return governance.Decision{Action: governance.Allow, Reason: "approved by user"}, nil
		default:
			return governance.Decision{Action: governance.Deny, Reason: r.reason}, nil
		}
	}
}

// handleCardAction processes a Feishu card action trigger event (button click
// or form submit). It resolves the pending approval and returns a toast +
// updated card for the Feishu client to send back over the WebSocket.
func (a *feishuApprover) handleCardAction(_ context.Context, event *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error) {
	if event == nil || event.Event == nil || event.Event.Action == nil {
		return toastResponse("error", "无效的审批事件"), nil
	}

	action := event.Event.Action
	approvalID, _ := action.Value["approval_id"].(string)
	if approvalID == "" {
		return toastResponse("error", "无法识别审批ID"), nil
	}

	buttonAction, _ := action.Value["action"].(string)

	a.mu.Lock()
	entry, ok := a.pending[approvalID]
	if ok {
		delete(a.pending, approvalID)
	}
	a.mu.Unlock()

	if !ok {
		return toastResponse("warning", "审批已过期或已处理"), nil
	}

	var r approvalResult
	switch buttonAction {
	case "allow_once":
		r = approvalResult{action: "allow"}
	case "allow_always":
		r = approvalResult{action: "allow"}
		// Remember for the session — same tool+args won't ask again.
		a.rememberAllowAlways(event, entry)
	case "deny":
		r = approvalResult{action: "deny", reason: "rejected by user"}
	default:
		return toastResponse("warning", "未知操作: "+buttonAction), nil
	}

	// Non-blocking send: Ask is waiting on this channel.
	select {
	case entry.result <- r:
	default:
	}

	// Build the resolved card and patch the original card message — but
	// only for the fallback path (standalone approval card). For the
	// integrated path the buttons live in the run card; streamReply
	// removes them when the next stream event arrives.
	decision, toastType, toastText := resolvedDisplay(r)
	resp := toastResponse(toastType, toastText)

	if !entry.integrated {
		go a.patchResolved(event, entry, decision, r.reason)
	}

	return resp, nil
}

// rememberAllowAlways persists the approval decision for the session so
// the same tool+args won't ask again (ACP "Allow Always" semantics).
func (a *feishuApprover) rememberAllowAlways(event *callback.CardActionTriggerEvent, entry *pendingApproval) {
	if a.memory == nil {
		return
	}
	chatID := ""
	if event.Event.Context != nil {
		chatID = event.Event.Context.OpenChatID
	}
	if chatID == "" {
		return
	}
	sessionID := "feishu_" + chatID
	d := governance.Decision{Action: governance.Allow}
	keys := governance.MemoryKeys(entry.toolName, json.RawMessage(entry.args))
	if len(keys) == 0 {
		keys = []string{governance.ApprovalKey(entry.toolName, json.RawMessage(entry.args))}
	}
	for _, key := range keys {
		if err := a.memory.Remember(context.Background(), sessionID, key, d); err != nil {
			slog.Warn("feishu: approve always persistence failed", "session", sessionID, "error", err)
		}
	}
}

// patchResolved fire-and-forgets a PATCH to update the approval card with
// the final decision. Failures are non-critical (the toast already confirmed
// the action to the user).
func (a *feishuApprover) patchResolved(event *callback.CardActionTriggerEvent, entry *pendingApproval, decision, reason string) {
	if a.client == nil || event.Event.Context == nil {
		return
	}
	msgID := event.Event.Context.OpenMessageID
	if msgID == "" {
		return
	}
	cardJSON, err := buildResolvedCard(entry.toolName, entry.args, decision, reason)
	if err != nil {
		slog.Warn("feishu: build resolved card", "error", err)
		return
	}
	if err := patchMessageCard(context.Background(), a.client, msgID, cardJSON); err != nil {
		slog.Warn("feishu: patch resolved card", "error", err)
	}
}

// sendInteractive sends a raw interactive card JSON to a chat by chat_id.
func (a *feishuApprover) sendInteractive(ctx context.Context, chatID, cardJSON string) (string, error) {
	return createMessage(ctx, a.client, "chat_id", chatID, "interactive", cardJSON)
}

// chatIDFromSession extracts the Feishu chat ID from a session ID. Channel
// session IDs follow the convention "<channelName>_<chatID>" (e.g.
// "feishu_oc_xxx").
func chatIDFromSession(sessionID string) string {
	prefix := "feishu_"
	if strings.HasPrefix(sessionID, prefix) {
		return strings.TrimPrefix(sessionID, prefix)
	}
	return sessionID
}

// resolvedDisplay maps an approvalResult to the display text for the
// resolved card, plus the toast type and message.
func resolvedDisplay(r approvalResult) (decision, toastType, toastText string) {
	switch r.action {
	case "allow":
		return "✅ **已同意**", "success", "已同意"
	default:
		return "❌ **已拒绝**", "error", "已拒绝"
	}
}

// toastResponse builds a CardActionTriggerResponse containing only a toast.
func toastResponse(typ, content string) *callback.CardActionTriggerResponse {
	return &callback.CardActionTriggerResponse{
		Toast: &callback.Toast{Type: typ, Content: content},
	}
}

// setRunCardUpdater registers a callback invoked by Ask to embed approval
// buttons in the run card instead of sending a separate approval card.
// Keyed by session ID to support concurrent chats.
func (a *feishuApprover) setRunCardUpdater(sessionID string, cb func(toolName, args, approvalID string)) {
	a.updaterMu.Lock()
	a.updaters[sessionID] = cb
	a.updaterMu.Unlock()
}

// clearRunCardUpdater removes the callback registered for the given session.
func (a *feishuApprover) clearRunCardUpdater(sessionID string) {
	a.updaterMu.Lock()
	delete(a.updaters, sessionID)
	a.updaterMu.Unlock()
}

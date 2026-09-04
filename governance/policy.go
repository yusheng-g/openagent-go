// Package governance implements the policy layer: the layered approval
// engine (rules → safety → memory → human), guards, and the policy
// interface.
//
// This package holds only interfaces and the engine skeleton in P0; the
// full engine lands in P1 together with kernel.Runtime. The guard and
// approver interfaces stay in the root package until P1 moves Agent out of
// the root package (moving them here would create an import cycle: the
// root Agent struct references them, and this package references root
// types).
package governance

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
)

// ApprovalAction is the outcome of evaluating one tool call against the
// policy chain.
type ApprovalAction string

const (
	// Allow executes the call without human input.
	Allow ApprovalAction = "allow"
	// Deny blocks the call; the tool result reports the rejection reason.
	Deny ApprovalAction = "deny"
	// Ask routes the call to the human layer for a decision.
	Ask ApprovalAction = "ask"
)

// Decision is the policy engine's verdict on one tool call. Unlike the
// legacy boolean approver, a Decision carries semantics: ModifiedArgs lets
// a human approve with edited arguments (e.g. fixing a dangerous path),
// and Reason is always populated for audit.
type Decision struct {
	Action       ApprovalAction  `json:"action"`
	Reason       string          `json:"reason"`
	ModifiedArgs json.RawMessage `json:"modified_args,omitempty"`
}

// Rule is a settings-driven policy rule: when a call's tool name matches
// ToolPattern (glob) and its arguments match ArgsPattern, the rule's
// Action applies. Rules are evaluated in order; the first match wins.
// This is the Claude Code permissions shape.
type Rule struct {
	ToolPattern string         `json:"tool_pattern"`
	ArgsPattern map[string]any `json:"args_pattern,omitempty"`
	Action      ApprovalAction `json:"action"`
	Reason      string         `json:"reason"`
}

// SafetyClass classifies a tool's side-effect profile. The runtime derives
// it from a SafetyClassifier — read-only tools are auto-allowed without
// consulting the human.
type SafetyClass int

const (
	// ReadOnly tools never mutate external state.
	ReadOnly SafetyClass = iota
	// Mutating tools change state but stay within workspace boundaries.
	Mutating
	// Dangerous tools can affect the host (shell, network, keyring...).
	Dangerous
)

// SafetyClassifier decides a tool's safety class. Classification lives in
// the platform, not in the tools: the kernel injects ToolClassifier (a
// whitelist of read-only tools); everything else is Dangerous and
// consults the human layer. Tools never self-declare safety.
type SafetyClassifier interface {
	Classify(def openagent.FunctionDefinition) SafetyClass
}

// ApprovalMemory is the session-scoped approval memory: once a human
// answers "always allow" for a tool, subsequent calls to the same tool
// are auto-allowed within the session (persisted, not just in-memory —
// this fixes the legacy ACP "Allow Always" bug).
type ApprovalMemory interface {
	// Remember stores a decision keyed by tool name within a session.
	Remember(ctx context.Context, sessionID, key string, decision Decision) error
	// Recall returns the remembered decision for a key, if any.
	Recall(ctx context.Context, sessionID, key string) (Decision, bool)
}

// HumanApprover is the human layer of the policy chain. Ask is called when
// rules, safety, and memory all defer to a human. Implementations bridge
// to the application: ACP RequestPermission RPC, REST SSE dialog, TUI
// prompt.
type HumanApprover interface {
	Ask(ctx context.Context, call openagent.ToolCall, def openagent.FunctionDefinition, session openagent.Session) (Decision, error)
}

// Policy evaluates whether a tool call may execute. The layered engine
// (rules → safety → memory → human) implements it; the runtime consults
// Policy instead of calling an approver directly.
type Policy interface {
	Evaluate(ctx context.Context, call openagent.ToolCall, def openagent.FunctionDefinition, session openagent.Session) (Decision, error)
}

// ── Layered policy engine ──

// Engine implements Policy with the layered chain: Rules → Safety →
// Memory → Human. Each layer short-circuits when it reaches a decision.
type Engine struct {
	Rules  []Rule           // first match wins (nil = no rules)
	Safety SafetyClassifier // nil = no safety layer (all calls advance)
	Memory ApprovalMemory   // session-scoped approval memory (nil = no memory layer)
	Human  HumanApprover    // final layer (nil = Ask resolves to Deny)
	// DecObserver receives per-layer decision events (rule/safety/memory/human).
	// nil = silent. Set via WithDecisionObserver; not a NewEngine parameter so
	// the constructor signature stays stable. The kernel sets it only when the
	// configured RunObserver also implements DecisionObserver.
	DecObserver openagent.DecisionObserver
}

// NewEngine creates an engine. When human is nil, Ask decisions resolve to
// Deny (fail closed).
func NewEngine(rules []Rule, safety SafetyClassifier, mem ApprovalMemory, human HumanApprover) *Engine {
	return &Engine{Rules: rules, Safety: safety, Memory: mem, Human: human}
}

// WithDecisionObserver sets the observer for per-layer decision events. The
// kernel chains this after NewEngine when the configured RunObserver also
// implements DecisionObserver; nil-safe. Returns e for chaining.
func (e *Engine) WithDecisionObserver(obs openagent.DecisionObserver) *Engine {
	e.DecObserver = obs
	return e
}

// matchesRule reports whether a rule matches the tool name and args.
// Glob forms on the tool name: "*" (any), exact "name", prefix "name*",
// suffix "*name", and contains "*name*". An empty ArgsPattern matches any
// args; otherwise every key in ArgsPattern must exist in the call args
// with the same value.
func matchesRule(rule Rule, call openagent.ToolCall) bool {
	if rule.ToolPattern == "" {
		return false
	}
	p := rule.ToolPattern
	matched := p == "*" || p == call.Function.Name
	if !matched {
		switch {
		case strings.HasPrefix(p, "*") && strings.HasSuffix(p, "*"):
			core := strings.Trim(p, "*")
			matched = core != "" && strings.Contains(call.Function.Name, core)
		case strings.HasPrefix(p, "*"):
			matched = strings.HasSuffix(call.Function.Name, strings.TrimPrefix(p, "*"))
		case strings.HasSuffix(p, "*"):
			matched = strings.HasPrefix(call.Function.Name, strings.TrimSuffix(p, "*"))
		}
	}
	if !matched {
		return false
	}
	if len(rule.ArgsPattern) == 0 {
		return true
	}
	// Args pattern: every key must exist in the call args with the same value.
	var args map[string]any
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return false
	}
	for k, v := range rule.ArgsPattern {
		got, ok := args[k]
		if !ok || !reflect.DeepEqual(got, v) {
			return false
		}
	}
	return true
}

// Evaluate runs the layered chain. The resulting Decision is final: Allow
// executes the call (with ModifiedArgs if the human edited them), Deny
// blocks it, Ask (from Rules) routes to the human layer.
//
// Each layer emits one DecisionEvent (when DecObserver is set) showing what
// it decided — Allow/Deny/Ask on a short-circuit, Skipped when it defers to
// the next layer. Short-circuit means lower layers do not fire, so a single
// Evaluate call emits exactly the layers that ran.
func (e *Engine) Evaluate(ctx context.Context, call openagent.ToolCall, def openagent.FunctionDefinition, session openagent.Session) (Decision, error) {
	ri := openagent.RunInfoFromContext(ctx)
	// call (call.ID) and session (session.ID) are both Evaluate-scoped, so
	// the direct emit stamps the full four-tuple join key (session_id, run_id,
	// turn_id, call_id) explicitly — the ObserveDecision helper is bypassed
	// because the Engine talks to its DecObserver field directly.
	emit := func(layer, outcome string, detail map[string]any) {
		if e.DecObserver != nil {
			e.DecObserver.ObserveDecision(ctx, openagent.DecisionEvent{
				Layer: layer, Outcome: outcome, Subject: call.Function.Name,
				Detail: detail, RunID: ri.RunID, TurnID: ri.TurnID,
				ParentRunID: ri.ParentRunID, SessionID: session.ID, CallID: call.ID,
			})
		}
	}

	// 1) Rules layer.
	for _, rule := range e.Rules {
		if !matchesRule(rule, call) {
			continue
		}
		if rule.Action == Ask {
			emit(openagent.DecisionPolicyRule, openagent.OutcomeAsk, map[string]any{"reason": rule.Reason})
			return e.askHuman(ctx, call, def, session, rule.Reason)
		}
		emit(openagent.DecisionPolicyRule, string(rule.Action), map[string]any{"reason": rule.Reason})
		return Decision{Action: rule.Action, Reason: rule.Reason}, nil
	}
	// No rule matched: rules layer defers. (A matched rule always returns
	// inside the loop, so reaching here means no match.) Emit Skipped only
	// when rules were configured — an empty Rules slice is an absent layer.
	if len(e.Rules) > 0 {
		emit(openagent.DecisionPolicyRule, openagent.OutcomeSkipped, nil)
	}

	// 2) Safety layer: read-only tools auto-allow.
	if e.Safety != nil {
		if e.Safety.Classify(def) == ReadOnly {
			emit(openagent.DecisionPolicySafety, openagent.OutcomeAllow, map[string]any{"classifier": "readonly"})
			return Decision{Action: Allow, Reason: "read-only tool"}, nil
		}
		emit(openagent.DecisionPolicySafety, openagent.OutcomeSkipped, nil)
	}

	// 2.5) Risk-note bypass: if the tool call carries a non-empty risk_note
	// (shell tool's risk_note param for destructive commands), skip the
	// Memory layer and force human approval — even if the user previously
	// chose "allow always". A destructive command (rm -rf, terraform apply)
	// must not silently execute from memory just because a similar safe
	// command was remembered.
	if hasRiskNote(call) {
		emit(openagent.DecisionPolicyRule, openagent.OutcomeAsk, map[string]any{"reason": "risk_note present — forcing approval"})
		return e.askHuman(ctx, call, def, session, "risk_note present")
	}

	// 3) Memory layer: remembered decision for this call (tool + args —
	// a changed argument is a different operation and asks again). A
	// remembered Ask never short-circuits — it still routes to the human
	// layer (otherwise an Ask in memory would silently bypass approval).
	//
	// shell and write use multi-key ALL semantics: every command atom and
	// every file access must be remembered as Allow (see MemoryKeys), so
	// a new command in a chain or a new file target re-asks while reused
	// ones don't.
	if e.Memory != nil {
		if keys := MemoryKeys(call.Function.Name, json.RawMessage(call.Function.Arguments)); len(keys) > 0 {
			all := true
			for _, k := range keys {
				d, ok := e.Memory.Recall(ctx, session.ID, k)
				if !ok || d.Action != Allow {
					all = false
					break
				}
			}
			if all {
				emit(openagent.DecisionPolicyMemory, openagent.OutcomeHit, map[string]any{"mode": "multi-key", "keys": len(keys)})
				return Decision{Action: Allow, Reason: "remembered"}, nil
			}
			emit(openagent.DecisionPolicyMemory, openagent.OutcomeMiss, map[string]any{"mode": "multi-key", "keys": len(keys)})
		} else {
			key := ApprovalKey(call.Function.Name, json.RawMessage(call.Function.Arguments))
			if d, ok := e.Memory.Recall(ctx, session.ID, key); ok {
				if d.Action == Ask {
					emit(openagent.DecisionPolicyMemory, openagent.OutcomeAsk, map[string]any{"mode": "single"})
					return e.askHuman(ctx, call, def, session, "remembered ask")
				}
				emit(openagent.DecisionPolicyMemory, openagent.OutcomeHit, map[string]any{"mode": "single", "action": string(d.Action)})
				return d, nil
			}
			emit(openagent.DecisionPolicyMemory, openagent.OutcomeMiss, map[string]any{"mode": "single"})
		}
	}

	// 4) Human layer.
	return e.askHuman(ctx, call, def, session, "requires approval")
}

// askHuman routes to the human layer. A nil human fails closed (Deny).
// Emits one DecisionPolicyHuman event — Ask when escalating to a human,
// Deny when no approver is configured (fail closed).
func (e *Engine) askHuman(ctx context.Context, call openagent.ToolCall, def openagent.FunctionDefinition, session openagent.Session, reason string) (Decision, error) {
	if e.DecObserver != nil {
		ri := openagent.RunInfoFromContext(ctx)
		outcome := openagent.OutcomeAsk
		if e.Human == nil {
			outcome = openagent.OutcomeDeny
		}
		e.DecObserver.ObserveDecision(ctx, openagent.DecisionEvent{
			Layer: openagent.DecisionPolicyHuman, Outcome: outcome,
			Subject: call.Function.Name, Detail: map[string]any{"reason": reason},
			RunID: ri.RunID, TurnID: ri.TurnID,
			ParentRunID: ri.ParentRunID, SessionID: session.ID, CallID: call.ID,
		})
	}
	if e.Human == nil {
		return Decision{Action: Deny, Reason: reason + " (no approver configured)"}, nil
	}
	return e.Human.Ask(ctx, call, def, session)
}

// ToolClassifier is the default platform-side safety classification: a
// whitelist of read-only tool names. Read-only tools are auto-allowed by
// the policy chain (Safety layer); everything else is Dangerous and
// consults the human layer. The classification lives here in the
// platform, not in the tools themselves (tools never self-declare
// safety; that is the legacy CanSelfApprove pattern).
type ToolClassifier struct {
	ReadOnlyNames map[string]bool
}

// NewToolClassifier creates a classifier with the built-in read-only set.
func NewToolClassifier() *ToolClassifier {
	return &ToolClassifier{ReadOnlyNames: map[string]bool{
		"read": true, "ls": true, "grep": true,
		"webfetch": true, "websearch": true,
		"recall": true, "load_skill": true, "reload_skills": true,
		// Browser read-only tools: navigate/screenshot/evaluate/snapshot/tabs
		// only read page state. click/type/press/close_* mutate the page or
		// browser and stay Dangerous (require approval).
		"browser_navigate": true, "browser_screenshot": true, "browser_evaluate": true,
		"browser_use_snapshot": true, "browser_use_tabs": true,
		// Office read-only tools: word_read/excel_read/pptx_read only read.
		// write/template_fill create files and stay Dangerous (require approval).
		"word_read": true, "excel_read": true, "pptx_read": true,
		// Sub-agent delegation: the delegation tool itself is a read-only
		// dispatch (no side effect beyond what the child's own gated tools
		// do). The child inherits the parent's policy/approver, so its tool
		// calls are individually gated. sub_agent_send continues the same
		// child under the same gating.
		"sub_agent_send": true,
	}}
}

// Classify implements SafetyClassifier.
func (c *ToolClassifier) Classify(def openagent.FunctionDefinition) SafetyClass {
	if c.ReadOnlyNames[def.Name] {
		return ReadOnly
	}
	return Dangerous
}

// hasRiskNote reports whether the tool call's arguments include a non-empty
// "risk_note" field. Used to bypass the Memory layer and force human approval
// for destructive commands even when "allow always" was previously granted.
func hasRiskNote(call openagent.ToolCall) bool {
	var params struct {
		RiskNote string `json:"risk_note"`
	}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &params); err != nil {
		return false
	}
	return strings.TrimSpace(params.RiskNote) != ""
}

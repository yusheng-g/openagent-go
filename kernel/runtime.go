// Package kernel implements the Agent Runtime Kernel: the execution engine
// that drives an agent loop from a pure [agent.Agent] configuration plus
// runtime dependencies ([Deps]).
//
// Runtime owns the 8-node mainline loop (memory fetch → prompt build →
// guard.in → model call → guard.out → policy/approval → tool execution →
// memory store), decomposed per node into methods on [Runtime]. It replaces
// the former monolithic runner.go: the agent config lives in agent/, context
// assembly in context/, tool execution in execution/, and policy in
// governance/ — Runtime only orchestrates.
package kernel

import (
	"context"
	"sync"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/agent"
	ctxpkg "github.com/yusheng-g/openagent-go/context"
	"github.com/yusheng-g/openagent-go/eventbus"
	"github.com/yusheng-g/openagent-go/execution"
	"github.com/yusheng-g/openagent-go/governance"
	"github.com/yusheng-g/openagent-go/provider/resource"
	"github.com/yusheng-g/openagent-go/provider/skill"
	"github.com/yusheng-g/openagent-go/session"
)

// errNoModel is returned when neither the config nor the session provides
// a model. The message stays free of Go-API symbols so it reads cleanly to
// end users (CLI/TUI) and embedding developers alike — the caller layer
// knows how models are configured for its surface and can wrap if needed.
var errNoModel = &noModelError{}

type noModelError struct{}

func (*noModelError) Error() string {
	return "no model configured (configure a model on the agent or session)"
}

// Deps are the runtime dependencies injected at construction — everything
// the Agent config does NOT own. Nil fields mean the capability is absent
// (the loop skips nil modules).
type Deps struct {
	// Tools is the initial tool set. Runtime owns the tools slice from here
	// on (AppendTools/SnapshotTools under the tools lock); the acp layer
	// appends plan/execution tools per turn via rt.AppendTools.
	Tools []openagent.Tool

	// SubAgentExcludeTools removes these tool names from every sub-agent's
	// tool set. Mode tools (plan_create, enter_plan_mode, ...) are
	// session-bound — their callbacks would be nil in the child's isolated
	// runtime — so the application layer injects them here. The kernel
	// holds no mode-tool knowledge of its own.
	SubAgentExcludeTools []string

	// SessionStore persists the current conversation (short-term).
	SessionStore session.SessionStore
	// Compressor owns token-budget compression (summary layer).
	Compressor session.Compressor
	// Summarizer is the model-backed summarizer shared with sub-agent
	// children so their in-memory stores get compaction parity with the
	// parent. nil = sub-agents degrade to no-compaction. The parent's own
	// Compressor (a *sqlite.MessageStore) embeds its summarizer privately;
	// this field is the explicit, shareable handle.
	Summarizer openagent.Summarizer
	// SubAgentRegistry tracks resumable sub-agent children for the session.
	// nil = kernel.New creates a fresh one. Shared by subAgentTool (spawn)
	// and sendTool (continue) so both address the same live children.
	// Children use in-memory stores distinct from normal (on-disk) sessions.
	SubAgentRegistry *childRegistry
	// MemoryProvider stores/recalls durable knowledge (long-term).
	MemoryProvider ctxpkg.MemoryProvider

	// HumanApprover is the human layer of the policy chain. nil = allow
	// all (no approval step). When Policy is set, the layered engine
	// takes precedence and this field is ignored.
	HumanApprover governance.HumanApprover
	// Policy is the layered approval engine (rules → safety → memory →
	// human). nil = default engine: transfer_to_* auto-allowed +
	// Approver as the human layer.
	Policy governance.Policy
	// ApprovalMemory persists session-scoped approval decisions
	// ("allow always"). nil = an in-process per-runtime memory (decisions
	// live for the runtime's lifetime; the app can supply a persistent
	// one keyed by session).
	ApprovalMemory governance.ApprovalMemory

	// Hooks receive agent/tool lifecycle callbacks.
	Hooks openagent.RunHooks
	// Observer receives per-stage loop events.
	Observer openagent.RunObserver
	// ResultPolicy truncates oversized tool results (after hooks).
	ResultPolicy openagent.ResultPolicy

	// SkillProvider matches/discovers/loads skills (nil = no skills).
	SkillProvider skill.Provider
	// ResourceProvider supplies external reference material (nil = none).
	ResourceProvider resource.Provider
	// Context assembles the per-turn AgentContext (knowledge recall).
	// nil = default (built from the providers above). Interface so the
	// application can substitute its own context assembly.
	Context ctxpkg.Runtime
	// Extractor stores durable knowledge after a finished run. Wire one
	// shared AsyncExtractor per server (never per run — see
	// context.NewAsyncExtractor). nil = no self-evolution.
	Extractor ctxpkg.Extractor
	// EventLogger records audit events (user.input, tool.call, ...).
	// nil = no audit log.
	EventLogger eventbus.Logger
}

// Runtime drives one agent run. Create per run via New; it is not safe
// for concurrent use.
type Runtime struct {
	cfg  *agent.Agent
	deps Deps

	// mu guards the post-construction mutable state: cfg
	// (SetModel/SetReasoningEffort/SetSystemPrompts/SetMaxTurns),
	// humanApprover, and runModel. A single run is sequential, but the
	// acp layer reuses one Runtime across turns and applies session
	// config/mode changes (SetModel/SetHumanApprover) from the serve
	// loop and from tool callbacks while a run is in flight, so these
	// fields are concurrency-shared. Runs snapshot what they need under
	// the read lock; setters take the write lock.
	mu            sync.RWMutex
	humanApprover governance.HumanApprover // mutable — acp plan-mode transitions switch it mid-run

	// tools is the mutable tool set (toolsMu-guarded). Readers use
	// SnapshotTools; mutators use AppendTools — a tool callback
	// (exit_plan_mode via the acp layer) can append execution tools to the
	// SAME runtime during an executeTools batch, so the slice is a
	// concurrency-shared mutable field within a run.
	tools   []openagent.Tool
	toolsMu sync.RWMutex

	// Per-run state (fresh on every New).
	approvalMemory governance.ApprovalMemory
	runModel       openagent.Model
	builtinTools   []openagent.FunctionDefinition
	compressed     *openagent.CompressedContext
	execution      execution.Runtime
	context        ctxpkg.Runtime
	state          *ctxpkg.RuntimeState
}

// SubAgentRegistry returns the session's child registry, or nil when no
// sub-agents are configured. The ACP layer uses this to wire the onExit
// completion callback for async sub-agent notifications.
func (rt *Runtime) SubAgentRegistry() *childRegistry {
	return rt.deps.SubAgentRegistry
}

// New creates a Runtime from an agent config and dependencies.
func New(cfg *agent.Agent, deps Deps) *Runtime {
	rt := &Runtime{
		cfg:            cfg,
		deps:           deps,
		humanApprover:  deps.HumanApprover,
		approvalMemory: deps.ApprovalMemory,
		state:          &ctxpkg.RuntimeState{},
	}
	if rt.approvalMemory == nil {
		rt.approvalMemory = governance.NewSessionApprovalMemory()
	}
	if len(deps.Tools) > 0 {
		rt.tools = append(rt.tools, deps.Tools...)
	}
	// Builtin tools (load_skill/reload_skills, recall) mount once here —
	// NOT per Run() call: a runtime whose Run is retried (attempt loops)
	// would otherwise accumulate duplicate tool definitions, which
	// providers like DeepSeek reject ("Tool names must be unique").
	if deps.SkillProvider != nil {
		rt.builtinTools = execution.BuiltinSkillToolDefs()
	}
	if deps.MemoryProvider != nil {
		rt.builtinTools = append(rt.builtinTools, execution.BuiltinRecallDef())
	}
	// Pre-configured sub-agents become delegation tools: isolated context,
	// own system prompt, tools resolved at call time (see newSubAgentTool).
	// Registered in New so the model sees them from the first turn. A shared
	// childRegistry (one per session, lazily created here) lets sub_agent_send
	// resume a child spawned by a subAgentTool — both tools hold the same reg.
	if deps.SubAgentRegistry == nil {
		deps.SubAgentRegistry = newChildRegistry()
		rt.deps.SubAgentRegistry = deps.SubAgentRegistry
	}
	for _, sa := range cfg.SubAgents {
		rt.tools = append(rt.tools, rt.newSubAgentTool(sa, deps.SubAgentRegistry))
	}
	// sub_agent_send lets the model follow up on a spawned sub-agent with
	// history. Registered only when delegation tools exist, and alongside
	// them so plan-mode caching (subAgentToolNames) keeps them in lockstep.
	if len(cfg.SubAgents) > 0 {
		rt.tools = append(rt.tools, newSendTool(deps.SubAgentRegistry))
	}
	if deps.Context != nil {
		rt.context = deps.Context
	} else {
		rt.context = ctxpkg.NewContextRuntime(ctxpkg.Config{
			SessionStore:     deps.SessionStore,
			Compressor:       deps.Compressor,
			MemoryProvider:   deps.MemoryProvider,
			SkillProvider:    deps.SkillProvider,
			ResourceProvider: deps.ResourceProvider,
			Observer:         deps.Observer,
		})
	}
	// Default result policy: oversized tool output is truncated to disk
	// (FileRef) instead of flooding the model context. Applications can
	// substitute their own policy; nil means "no truncation" only when
	// explicitly desired.
	policy := deps.ResultPolicy
	if policy == nil {
		policy = &openagent.DefaultResultPolicy{}
	}
	rt.execution = execution.New(execution.Config{
		ToolSnapshot:   rt.SnapshotTools,
		SkillProvider:  deps.SkillProvider,
		MemoryProvider: deps.MemoryProvider,
		Hooks:          deps.Hooks,
		Observer:       deps.Observer,
		ResultPolicy:   policy,
	})
	return rt
}

// Config returns the agent configuration backing this runtime.
func (rt *Runtime) Config() *agent.Agent { return rt.cfg }

// SkillProvider returns the session's skill provider (nil if none).
func (rt *Runtime) SkillProvider() skill.Provider {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.deps.SkillProvider
}

// Model returns the resolved model for the current run (session override
// wins); nil until run() resolves it.
func (rt *Runtime) Model() openagent.Model {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.runModel
}

// SetSystemPrompts overrides the agent's system prompts for this runtime
// (used by wasm runtime_set host exports). Safe to call from any goroutine;
// prompt building snapshots the value under the lock.
func (rt *Runtime) SetSystemPrompts(p []string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.cfg.SystemPrompts = p
}

// SetMaxTurns overrides the max-turns limit for this runtime. Safe to
// call from any goroutine; the run loop reads it once at run start.
func (rt *Runtime) SetMaxTurns(n int) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.cfg.MaxTurns = n
}

// SetModel overrides the agent's model for this runtime (session config
// model changes). Safe to call from any goroutine; a running turn keeps
// the model it snapshotted at run start.
func (rt *Runtime) SetModel(m openagent.Model) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.cfg.Model = m
}

// SetReasoningEffort overrides the reasoning-effort pass-through for this
// runtime (session config thought_level changes). Safe to call from any
// goroutine.
func (rt *Runtime) SetReasoningEffort(e string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.cfg.ReasoningEffort = e
}

// SetHumanApprover replaces the human approval layer mid-run (used by
// acp plan-mode transitions). Safe to call from tool callbacks; the next
// executeTools batch reads the new value.
func (rt *Runtime) SetHumanApprover(ap governance.HumanApprover) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.humanApprover = ap
}

// policy returns the effective policy engine. The default engine
// auto-allows transfer_to_* handoffs and delegates the human layer to
// the configured approver. With NO approver configured, every call is
// allowed (no approval step — see Deps.HumanApprover). Applications
// that need fail-closed approval must wire a HumanApprover or Policy.
func (rt *Runtime) policy() governance.Policy {
	if rt.deps.Policy != nil {
		return rt.deps.Policy
	}
	// Snapshot the mutable human layer + sub-agents under the lock: a
	// concurrent SetHumanApprover (acp plan-mode transition from the
	// serve loop or a tool callback) must not tear the read.
	rt.mu.RLock()
	humanApprover := rt.humanApprover
	subAgents := rt.cfg.SubAgents
	rt.mu.RUnlock()
	if humanApprover == nil {
		// No human layer = no approval step = allow all.
		return allowAllPolicy{}
	}
	rules := []governance.Rule{{
		ToolPattern: "transfer_to_*",
		Action:      governance.Allow,
		Reason:      "handoff tools are always allowed",
	}}
	// Delegation to configured sub-agents is auto-allowed: the call itself
	// is control flow with no side effects — the child's tool calls are
	// governed by the inherited policy chain inside (v2.0 §22). Apps that
	// want to gate a specific sub-agent supply their own Policy via Deps.
	for _, sa := range subAgents {
		rules = append(rules, governance.Rule{
			ToolPattern: sa.Name,
			Action:      governance.Allow,
			Reason:      "delegation is governed inside the sub-agent",
		})
	}
	return governance.NewEngine(
		rules,
		governance.NewToolClassifier(), // platform-side read-only classification
		rt.approvalMemory,              // session-scoped approval memory ("allow always")
		humanApprover,
	).WithDecisionObserver(rt.decisionObserver())
}

// decisionObserver returns the configured RunObserver as a DecisionObserver,
// or nil when none is configured or it does not implement DecisionObserver.
// The Engine observes per-layer events only when this is non-nil; custom
// Policy impls are wrapped by observingPolicy in execute.go with the same
// value. Extracting it here keeps the type-assertion in one place.
func (rt *Runtime) decisionObserver() openagent.DecisionObserver {
	if rt.deps.Observer == nil {
		return nil
	}
	if d, ok := rt.deps.Observer.(openagent.DecisionObserver); ok {
		return d
	}
	return nil
}

// allowAllPolicy is the no-approver policy: every call executes, EXCEPT
// calls with a non-empty risk_note — those are denied (fail-closed) because
// there is no human to approve them. This prevents destructive commands
// (rm -rf, terraform apply) from silently executing in modes without an
// interactive approver (REST, CLI one-shot).
type allowAllPolicy struct{}

func (allowAllPolicy) Evaluate(_ context.Context, call openagent.ToolCall, _ openagent.FunctionDefinition, _ openagent.Session) (governance.Decision, error) {
	if governance.HasRiskNote(call) {
		return governance.Decision{Action: governance.Deny, Reason: "risk_note present and no approver configured — destructive command requires approval"}, nil
	}
	return governance.Decision{Action: governance.Allow, Reason: "no approver configured"}, nil
}

// AppendTools appends tools under the tools lock. Use this instead of
// mutating deps.Tools directly so concurrent SnapshotTools readers (from
// executeTools' parallel goroutines) see a consistent slice.
func (rt *Runtime) AppendTools(tools ...openagent.Tool) {
	rt.toolsMu.Lock()
	defer rt.toolsMu.Unlock()
	rt.tools = append(rt.tools, tools...)
}

// RemoveTools removes tools by name under the tools lock. Used by mode
// transitions (plan mode drops execution tools, execution modes drop
// read-only ones) and per-turn plan-tool rebinding. Running tool jobs
// hold their own Tool instance; removal only affects future snapshots.
func (rt *Runtime) RemoveTools(names ...string) {
	if len(names) == 0 {
		return
	}
	rt.toolsMu.Lock()
	defer rt.toolsMu.Unlock()
	drop := make(map[string]bool, len(names))
	for _, n := range names {
		drop[n] = true
	}
	out := rt.tools[:0]
	for _, t := range rt.tools {
		if !drop[t.Definition().Name] {
			out = append(out, t)
		}
	}
	rt.tools = out
}

// SnapshotTools returns a copy of the current tool set under the lock.
func (rt *Runtime) SnapshotTools() []openagent.Tool {
	rt.toolsMu.RLock()
	defer rt.toolsMu.RUnlock()
	out := make([]openagent.Tool, len(rt.tools))
	copy(out, rt.tools)
	return out
}

// Run runs one turn to completion.
func (rt *Runtime) Run(ctx context.Context, session openagent.Session, input openagent.Message) (*openagent.RunResult, error) {
	if !rt.hasConfigModel() && session.Model == nil {
		return nil, errNoModel
	}
	return rt.run(ctx, session, nil, input, nil)
}

// RunWithPrefix runs one turn with prefix messages (not persisted).
func (rt *Runtime) RunWithPrefix(ctx context.Context, session openagent.Session, prefix []openagent.Message, input openagent.Message) (*openagent.RunResult, error) {
	if !rt.hasConfigModel() && session.Model == nil {
		return nil, errNoModel
	}
	return rt.run(ctx, session, prefix, input, nil)
}

// RunStream runs one turn, streaming events to the returned channel.
func (rt *Runtime) RunStream(ctx context.Context, session openagent.Session, input openagent.Message) <-chan openagent.StreamEvent {
	return rt.RunStreamWithPrefix(ctx, session, nil, input)
}

// RunStreamWithPrefix runs one turn with prefix messages, streaming events.
func (rt *Runtime) RunStreamWithPrefix(ctx context.Context, session openagent.Session, prefix []openagent.Message, input openagent.Message) <-chan openagent.StreamEvent {
	ch := make(chan openagent.StreamEvent, 16)
	go func() {
		defer close(ch)
		if !rt.hasConfigModel() && session.Model == nil {
			ch <- openagent.StreamEvent{Type: openagent.StreamError, Error: errNoModel}
			return
		}
		rt.run(ctx, session, prefix, input, ch)
	}()
	return ch
}

// RunGoal runs an autonomous goal-mode turn.
func (rt *Runtime) RunGoal(ctx context.Context, session openagent.Session, goal string) (*openagent.RunResult, error) {
	if !rt.hasConfigModel() && session.Model == nil {
		return nil, errNoModel
	}
	cfg := rt.cfgWithGoal(goal)
	sub := New(cfg, rt.deps)
	return sub.run(ctx, session, nil, openagent.UserMessage(goal), nil)
}

// RunGoalStream runs an autonomous goal-mode turn, streaming events.
func (rt *Runtime) RunGoalStream(ctx context.Context, session openagent.Session, goal string) <-chan openagent.StreamEvent {
	ch := make(chan openagent.StreamEvent, 16)
	go func() {
		defer close(ch)
		if !rt.hasConfigModel() && session.Model == nil {
			ch <- openagent.StreamEvent{Type: openagent.StreamError, Error: errNoModel}
			return
		}
		cfg := rt.cfgWithGoal(goal)
		sub := New(cfg, rt.deps)
		sub.run(ctx, session, nil, openagent.UserMessage(goal), ch)
	}()
	return ch
}

// hasConfigModel reports whether the config carries a model, under mu
// (SetModel runs concurrently from the serve loop / wasm exports).
func (rt *Runtime) hasConfigModel() bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.cfg.Model != nil
}

// cfgWithGoal snapshots the config (with goal instructions applied) under
// mu — WithGoalInstructions clones the whole config, so the mutable fields
// (Model, SystemPrompts, ...) must be read under the lock.
func (rt *Runtime) cfgWithGoal(goal string) *agent.Agent {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.cfg.WithGoalInstructions(goal)
}

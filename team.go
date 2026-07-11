package openagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// ── Handoff ──

// HandoffEntry records one handoff in the chain.
type HandoffEntry struct {
	From    string // agent that handed off
	To      string // target agent
	Message string // truncated handoff message
}

// ── AgentRunner ──

// AgentRunner is the execution interface for any agent in a Team.
// Both in-process *Agent and external ACP agents implement this.
//
// Team calls RunWithPrefix (or RunStreamWithPrefix) to execute one turn.
// It does not care whether the agent runs in-process or externally —
// the interface abstracts the transport.
type AgentRunner interface {
	RunWithPrefix(ctx context.Context, session Session, prefix []Message, input Message) (*RunResult, error)
	RunStreamWithPrefix(ctx context.Context, session Session, prefix []Message, input Message) <-chan StreamEvent
}

// TeamPreparable is an optional interface for agents that need per-run
// team context injection. In-process *Agent implements this to receive
// cloned tools + team system prompt + handoff tools.
//
// External ACP agents typically do NOT implement TeamPreparable — they
// receive context through their own protocol (ACP session init, MCP tools).
type TeamPreparable interface {
	PrepareForTeam(tc TeamContext) AgentRunner
}

// Closer is an optional interface for agents that hold external resources.
// Team calls Close when the agent is removed via RemoveAgent, so that
// subprocesses, connections, and MCP servers are cleaned up.
type Closer interface {
	Close() error
}

// MembershipNotifiable is an optional interface for AgentRunners that need
// to be notified when team membership changes. Team calls it on every
// runner (including the newly-added one) after AddAgent or RemoveAgent.
//
// This is the trigger for external agents to synchronise their capabilities
// (e.g. registering transfer_to_* MCP tools for each teammate).
type MembershipNotifiable interface {
	OnMembersChanged(selfName string, members []AgentInfo)
}

// TeamContext carries per-run metadata for agent preparation.
type TeamContext struct {
	MyName         string           // agent's name within the team
	Members        []AgentInfo      // other team members (for handoff targets)
	HandoffHistory []HandoffEntry   // prior handoffs in this chain
	ForceFinal     bool             // if true, agent must produce final answer
	RecordHandoff  func(target, message string) // callback to record a handoff
	TeamPrompt    string           // "## Team Context" block produced by buildTeamPrompt
	HandoffTools   []Tool           // transfer_to_* tools to inject
}

// AgentType classifies an agent for UI display and capability checks.
type AgentType int

const (
	AgentInternal AgentType = iota // in-process *Agent
	AgentExternal                  // external ACP/MCP agent
)

func (t AgentType) String() string {
	switch t {
	case AgentInternal:
		return "internal"
	case AgentExternal:
		return "external"
	default:
		return "unknown"
	}
}

// ── Team ──

type agentEntry struct {
	runner      AgentRunner
	agentType   AgentType
	description string
}

// Team orchestrates multiple agents. It handles initial routing, handoff
// between agents, loop detection, and context injection.
//
// Create with NewTeam + TeamOption functions:
//
//	team := openagent.NewTeam(
//	    openagent.WithTeamAgent("planner", "breaks down tasks", plannerAgent),
//	    openagent.WithTeamAgent("coder", "writes code", coderAgent),
//	    openagent.WithTeamRouter(openagent.NewLLMRouter(model)),
//	    openagent.WithTeamMaxHandoffs(12),
//	)
//	result, err := team.Run(ctx, session, input)
type Team struct {
	agents map[string]*agentEntry
	order  []string // preserve insertion order for prompt

	router      Router
	maxHandoffs int
	observer    RunObserver

	// mu protects agents and order.
	mu sync.Mutex
}

type handoffRequest struct {
	target  string
	message string
}

// ── TeamOption ──

// TeamOption configures a Team.
type TeamOption func(*Team)

// WithTeamAgent adds an in-process agent to the team.
// name must be unique. description is shown to other agents so they know
// when to hand off.
func WithTeamAgent(name, description string, agent *Agent) TeamOption {
	return func(t *Team) {
		t.agents[name] = &agentEntry{runner: agent, agentType: AgentInternal, description: description}
		t.order = append(t.order, name)
	}
}

// WithTeamRouter sets the Router for initial message routing and handoff policy.
// nil (default) = FirstAgentRouter.
func WithTeamRouter(r Router) TeamOption {
	return func(t *Team) { t.router = r }
}

// WithTeamMaxHandoffs sets the maximum number of handoffs before force-stopping.
// Default is 10.
func WithTeamMaxHandoffs(n int) TeamOption {
	return func(t *Team) { t.maxHandoffs = n }
}

// WithTeamObserver sets a RunObserver that receives stage events during team
// execution (agent starts, handoffs, etc.).
func WithTeamObserver(o RunObserver) TeamOption {
	return func(t *Team) { t.observer = o }
}

// ── Constructor ──

// NewTeam creates a Team with the given options.
func NewTeam(opts ...TeamOption) *Team {
	t := &Team{
		agents:      make(map[string]*agentEntry),
		maxHandoffs: 10,
	}
	for _, o := range opts {
		o(t)
	}
	if t.router == nil {
		t.router = FirstAgentRouter{}
	}

	// Notify agents added via options of the initial membership.
	notify := t.collectObserverNotifications()
	for _, n := range notify {
		n.obs.OnMembersChanged(n.selfName, n.members)
	}

	return t
}

// ── Runtime membership ──

// AddAgent adds an agent to the team at runtime. It returns an error if an
// agent with the same name already exists.
//
// For in-process agents, pass *Agent directly (it implements AgentRunner).
// For external agents, pass an ACPRunner or any AgentRunner implementation.
//
// Safe to call concurrently with Run / RunStream. The new agent becomes
// visible to handoffs on the next agent transition.
func (t *Team) AddAgent(name, description string, runner AgentRunner) error {
	t.mu.Lock()
	if _, ok := t.agents[name]; ok {
		t.mu.Unlock()
		return fmt.Errorf("team: agent %q already exists", name)
	}

	at := AgentExternal
	if _, ok := runner.(*Agent); ok {
		at = AgentInternal
	}

	t.agents[name] = &agentEntry{runner: runner, agentType: at, description: description}
	t.order = append(t.order, name)

	// Collect observer notifications while holding the lock, then release
	// before calling external code to avoid deadlocks.
	notify := t.collectObserverNotifications()
	t.mu.Unlock()

	for _, n := range notify {
		n.obs.OnMembersChanged(n.selfName, n.members)
	}
	return nil
}

// RemoveAgent removes an agent from the team by name. No-op if the name
// doesn't exist.
//
// If the agent implements Closer (external ACP agents), Close is called
// to terminate subprocesses and clean up resources.
//
// Safe to call concurrently with Run / RunStream. The currently-running
// agent is not interrupted. Handoffs targeting the removed agent will fail
// with an "unknown agent" error.
func (t *Team) RemoveAgent(name string) {
	t.mu.Lock()
	entry, ok := t.agents[name]
	if !ok {
		t.mu.Unlock()
		return
	}
	delete(t.agents, name)
	for i, n := range t.order {
		if n == name {
			t.order = append(t.order[:i], t.order[i+1:]...)
			break
		}
	}

	// Notify remaining agents that membership changed.
	notify := t.collectObserverNotifications()
	t.mu.Unlock()

	for _, n := range notify {
		n.obs.OnMembersChanged(n.selfName, n.members)
	}

	// Clean up external resources outside the lock to avoid deadlock.
	if c, ok := entry.runner.(Closer); ok {
		c.Close()
	}
}

// observerNotification pairs an observer with the member list it should see
// (all agents except itself).
type observerNotification struct {
	obs      MembershipNotifiable
	selfName string
	members  []AgentInfo
}

// collectObserverNotifications builds the notification payloads for every
// agent that implements [MembershipNotifiable]. Caller must hold t.mu.
func (t *Team) collectObserverNotifications() []observerNotification {
	var out []observerNotification
	for name, entry := range t.agents {
		obs, ok := entry.runner.(MembershipNotifiable)
		if !ok {
			continue
		}
		members := make([]AgentInfo, 0, len(t.agents)-1)
		for _, n := range t.order {
			if n == name {
				continue
			}
			members = append(members, AgentInfo{
				Name:        n,
				Description: t.agents[n].description,
				Type:        t.agents[n].agentType,
			})
		}
		out = append(out, observerNotification{obs, name, members})
	}
	return out
}

// ── Run ──

// TeamResult holds the output of a Team.Run call.
type TeamResult struct {
	FinalOutput  string         // last agent's final text output
	HandoffChain []HandoffEntry // all handoffs that occurred
	TotalTurns   int            // total model calls across all agents
	Usage        Usage          // total token usage
}

// Run executes the team on a user input. It routes to the first agent, then
// follows handoff chains until an agent finishes without handing off.
func (t *Team) Run(ctx context.Context, session Session, input Message) (*TeamResult, error) {
	if len(t.agents) == 0 {
		return nil, fmt.Errorf("team has no agents")
	}
	tr := &teamRunner{team: t}
	return tr.run(ctx, session, input, nil)
}

// ── RunStream ──

// TeamEventType categorizes events emitted by Team.RunStream.
type TeamEventType string

const (
	TeamAgentStart TeamEventType = "agent_start" // agent begins execution
	TeamAgentEnd   TeamEventType = "agent_end"   // agent finished
	TeamHandoff    TeamEventType = "handoff"     // handoff between agents
	TeamThought    TeamEventType = "thought"     // reasoning content (o1, deepseek-r1)
	TeamTextDelta  TeamEventType = "text_delta"  // text token from current agent
	TeamToolCall     TeamEventType = "tool_call"     // tool call (non-handoff, for approval)
	TeamToolProgress TeamEventType = "tool_progress" // streaming tool output chunk
	TeamToolResult   TeamEventType = "tool_result"   // tool result (final)
	TeamRetrying   TeamEventType = "retrying"    // model call retrying after transient error
	TeamDone       TeamEventType = "done"        // team finished
	TeamError      TeamEventType = "error"       // error occurred
)

// TeamEvent is emitted by Team.RunStream.
type TeamEvent struct {
	Type       TeamEventType
	Agent      string    // current agent name
	Target     string    // handoff target agent (handoff)
	Text       string    // text_delta, tool_result content
	ToolCallID string    // tool_result, tool_progress (matches tool_call.id)
	Message    string    // handoff message (handoff)
	ToolCall   *ToolCall // tool_call
	Result     *TeamResult
	Error      error
}

// RunStream executes the team with streaming events.
func (t *Team) RunStream(ctx context.Context, session Session, input Message) <-chan TeamEvent {
	ch := make(chan TeamEvent, 32)
	go func() {
		defer close(ch)
		if len(t.agents) == 0 {
			ch <- TeamEvent{Type: TeamError, Error: fmt.Errorf("team has no agents")}
			return
		}
		tr := &teamRunner{team: t}
		if _, err := tr.run(ctx, session, input, ch); err != nil {
			ch <- TeamEvent{Type: TeamError, Error: err}
		}
	}()
	return ch
}

// ── teamRunner (internal, per-run state) ──

type teamRunner struct {
	team *Team

	chain       []HandoffEntry
	runMessages []Message // all messages from this team run (used as prefix for next agent)
	totalTurns  int
	totalUsage  Usage
	forceFinal       bool   // true = next agent gets no transfer tools
	currentName      string // currently executing agent
	handoffHintGiven bool   // true after one "please hand off" retry per agent turn

	// Per-run handoff queue — isolated from concurrent Team.Run() calls.
	handoffMu       sync.Mutex
	pendingHandoffs []handoffRequest
}

func (tr *teamRunner) run(ctx context.Context, session Session, input Message, ch chan<- TeamEvent) (*TeamResult, error) {
	// ── Resolve first agent ──
	agentInfos := tr.agentInfos()
	rr, err := tr.team.router.Route(ctx, input, agentInfos)
	if err != nil {
		return nil, fmt.Errorf("router: %w", err)
	}
	if rr.Fallback && tr.team.observer != nil {
		tr.team.observer.ObserveStage(ctx, StageEvent{
			Name: StageTeamRoute, Phase: "leave",
			Detail: map[string]any{"agent": rr.Agent, "fallback": rr.Reason},
		})
	}
	tr.team.mu.Lock()
	_, ok := tr.team.agents[rr.Agent]
	tr.team.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("router returned unknown agent %q", rr.Agent)
	}
	tr.currentName = rr.Agent
	currentInput := input

	for {
		// ── Hard limit ──
		if len(tr.chain) >= tr.team.maxHandoffs {
			return nil, fmt.Errorf("handoff limit reached (%d)", tr.team.maxHandoffs)
		}

		// ── Build runner for this run (may inject team context) ──
		runner := tr.prepareRunner(tr.currentName)
		if runner == nil {
			return nil, fmt.Errorf("agent %q was removed from the team during execution", tr.currentName)
		}

		// ── Emit agent_start ──
		if ch != nil {
			ch <- TeamEvent{Type: TeamAgentStart, Agent: tr.currentName}
		}
		if tr.team.observer != nil {
			tr.team.observer.ObserveStage(ctx, StageEvent{
				Name: StageTeamAgent, Phase: "enter",
				Detail: map[string]any{"agent": tr.currentName, "handoff_index": len(tr.chain)},
			})
		}

		// ── Run agent (full loop) ──
		var result *RunResult
		var runErr error

		if ch != nil {
			// Streaming path: forward text/tool events to the team channel.
			result, runErr = tr.runAgentStreaming(ctx, runner, session, currentInput, ch)
		} else {
			// Blocking path: fast path for Team.Run().
			result, runErr = runner.RunWithPrefix(ctx, session, tr.runMessages, currentInput)
		}

		if runErr != nil {
			// Agent run failed — emit agent_end with error and return.
			if tr.team.observer != nil {
				tr.team.observer.ObserveStage(ctx, StageEvent{
					Name: StageTeamAgent, Phase: "leave",
					Detail: map[string]any{"agent": tr.currentName},
					Err:    runErr,
				})
			}
			if ch != nil {
				ch <- TeamEvent{Type: TeamAgentEnd, Agent: tr.currentName, Error: runErr}
			}
			return nil, fmt.Errorf("agent %q: %w", tr.currentName, runErr)
		}

		tr.totalTurns += result.TurnCount
		tr.totalUsage.PromptTokens += result.Usage.PromptTokens
		tr.totalUsage.CompletionTokens += result.Usage.CompletionTokens
		tr.totalUsage.TotalTokens += result.Usage.TotalTokens

		// Accumulate messages for cross-agent context (retains multimodal content)
		tr.runMessages = append(tr.runMessages, result.Messages...)
		// Cap to prevent unbounded growth across handoffs.
		// Trim oldest, skipping orphaned tool results at the boundary.
		// Also handles the inverse: an orphaned assistant(tool_calls) whose
		// tool_results were trimmed away, which would violate API message rules.
		if len(tr.runMessages) > 128 {
			tr.runMessages = tr.runMessages[len(tr.runMessages)-128:]
			// Drop leading orphaned tool_results (no preceding tool_calls).
			for len(tr.runMessages) > 0 && tr.runMessages[0].Role == RoleTool {
				tr.runMessages = tr.runMessages[1:]
			}
			// Drop leading orphaned assistant(tool_calls) (their tool_results
			// were trimmed away and are now lost).
			for len(tr.runMessages) > 0 && tr.runMessages[0].Role == RoleAssistant && len(tr.runMessages[0].ToolCalls) > 0 {
				tr.runMessages = tr.runMessages[1:]
				// Also clean up any tool_results that now become orphaned.
				for len(tr.runMessages) > 0 && tr.runMessages[0].Role == RoleTool {
					tr.runMessages = tr.runMessages[1:]
				}
			}
		}

		// ── Check for handoff ──
		tr.handoffMu.Lock()
		handoffs := tr.pendingHandoffs
		tr.pendingHandoffs = nil
		tr.handoffMu.Unlock()

		if len(handoffs) == 0 {
			// Agent finished without handing off.
			// If the agent had handoff tools but didn't use them, retry
			// once with an explicit hint — silently, no SSE event.
			// The frontend sees a continuous stream from the same agent.
			if !tr.forceFinal && !tr.handoffHintGiven && len(tr.buildHandoffTools()) > 0 {
				tr.handoffHintGiven = true
				currentInput = Message{
					Role: RoleUser,
					Content: "⚠️ You must hand off. Your last response did not call transfer_to_*. " +
						"Use the appropriate transfer_to_<name> tool now to pass control. " +
						"If you are the final reviewer and the work is truly complete, " +
						"produce a short summary without handing off and the team will stop.",
				}
				continue
			}
			// forceFinal, already hinted, or no handoff tools — truly done.
			if tr.team.observer != nil {
				tr.team.observer.ObserveStage(ctx, StageEvent{
					Name: StageTeamAgent, Phase: "leave",
					Detail: map[string]any{"agent": tr.currentName},
				})
			}
			if ch != nil {
				ch <- TeamEvent{Type: TeamAgentEnd, Agent: tr.currentName}
			}
			return tr.finalize(result.FinalOutput, ch), nil
		}

		// Emit agent_end before transitioning to the next agent.
		if tr.team.observer != nil {
			tr.team.observer.ObserveStage(ctx, StageEvent{
				Name: StageTeamAgent, Phase: "leave",
				Detail: map[string]any{"agent": tr.currentName},
			})
		}
		if ch != nil {
			ch <- TeamEvent{Type: TeamAgentEnd, Agent: tr.currentName}
		}

		// Take the first handoff from this agent run.
		// Only the first handoff is honored — once the agent commits to a
		// transfer, that decision stands. Aligns with OpenAI Agents SDK.
		// If the agent also produced text output before handing off,
		// prepend it to the handoff message so the next agent sees it.
		req := handoffs[0]
		if result.FinalOutput != "" {
			if req.message != "" {
				req.message = result.FinalOutput + "\n" + req.message
			} else {
				req.message = result.FinalOutput
			}
		}

		// ── Validate target ──
		tr.team.mu.Lock()
		_, ok := tr.team.agents[req.target]
		tr.team.mu.Unlock()
		if !ok {
			return nil, fmt.Errorf("agent %q tried to hand off to unknown agent %q", tr.currentName, req.target)
		}

		// ── Build handoff entry ──
		entry := HandoffEntry{
			From:    tr.currentName,
			To:      req.target,
			Message: req.message,
		}

		// ── Policy check: Router.CanHandoff ──
		if vetoErr := tr.team.router.CanHandoff(ctx, entry, tr.chain, session); vetoErr != nil {
			// Veto: inject hint, stay with current agent, force final
			tr.forceFinal = true
			currentInput = Message{
				Role:    RoleUser,
				Content: fmt.Sprintf("⚠️ Handoff blocked: %v\n\nYour last request was: %s\nPlease handle this yourself.", vetoErr, req.message),
			}
			continue
		}

		tr.chain = append(tr.chain, entry)

		// ── Loop detection ──
		if hint := tr.detectLoop(); hint != "" {
			if tr.forceFinal {
				// Already gave a hint — hard stop
				return nil, fmt.Errorf("handoff loop detected: %s", hint)
			}
			tr.forceFinal = true
			currentInput = Message{
				Role:    RoleUser,
				Content: fmt.Sprintf("⚠️ %s\n\nLast handoff: %s\n\nYou must now produce a final answer yourself. Do not hand off again.", hint, req.message),
			}
		} else {
			tr.forceFinal = false
			currentInput = Message{
				Role:    RoleUser,
				Content: req.message,
			}
			// If the target agent previously handed off TO the current
			// agent, this is a return handoff (a follow-up question or
			// clarification). Add context so the target knows to answer
			// and hand off back — rather than treating it as a new
			// forward workflow task.
			if tr.isReturnHandoff(tr.currentName, req.target) {
				currentInput = Message{
					Role: RoleUser,
					Content: fmt.Sprintf("⚠️ Return handoff from %s (asking a follow-up). "+
						"Answer their question concisely, then hand off back to %s with your answer.\n\n"+
						"Message: %s", tr.currentName, tr.currentName, req.message),
				}
			}
		}

		// ── Emit handoff event ──
		if ch != nil {
			ch <- TeamEvent{Type: TeamHandoff, Agent: tr.currentName, Target: req.target, Message: req.message}
		}

		tr.currentName = req.target
		tr.handoffHintGiven = false // reset for next agent
	}
}

// runAgentStreaming runs the agent with streaming and forwards events to the
// team channel. Handoff tools (transfer_to_*) are not forwarded — they are
// internal team mechanism.
func (tr *teamRunner) runAgentStreaming(ctx context.Context, runner AgentRunner, session Session, input Message, ch chan<- TeamEvent) (*RunResult, error) {
	streamCh := runner.RunStreamWithPrefix(ctx, session, tr.runMessages, input)

	var result *RunResult
	// Map ToolCallID → function name for precise tool_result filtering.
	// When multiple tools execute in one turn and one is a handoff tool,
	// we need to filter only the handoff result, not all results.
	toolCallNames := make(map[string]string)
	for evt := range streamCh {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		switch evt.Type {
			case StreamThought:
				ch <- TeamEvent{Type: TeamThought, Agent: tr.currentName, Text: evt.Text}

		case StreamTextDelta:
			ch <- TeamEvent{Type: TeamTextDelta, Agent: tr.currentName, Text: evt.Text}

		case StreamToolCall:
			// Only forward non-handoff tool calls to the UI.
			if len(evt.Message.ToolCalls) > 0 {
				tc := &evt.Message.ToolCalls[0]
				toolCallNames[tc.ID] = tc.Function.Name
				if !strings.HasPrefix(tc.Function.Name, "transfer_to_") {
					ch <- TeamEvent{Type: TeamToolCall, Agent: tr.currentName, ToolCall: tc}
				}
			}

		case StreamToolProgress:
			ch <- TeamEvent{Type: TeamToolProgress, Agent: tr.currentName, ToolCallID: evt.ToolCallID, Text: evt.Text}

		case StreamToolResult:
			// Filter results of handoff tools by matching ToolCallID.
			if !strings.HasPrefix(toolCallNames[evt.Message.ToolCallID], "transfer_to_") {
				ch <- TeamEvent{Type: TeamToolResult, Agent: tr.currentName, ToolCallID: evt.Message.ToolCallID, Text: evt.Message.Content}
			}

		case StreamRetrying:
			ch <- TeamEvent{Type: TeamRetrying, Agent: tr.currentName, Error: evt.Error}

		case StreamDone:
			result = evt.Result

		case StreamError:
			return nil, evt.Error

		case StreamAborted:
			return nil, fmt.Errorf("agent %q aborted: %w", tr.currentName, evt.Error)
		}
	}
	if result == nil {
		return nil, fmt.Errorf("agent %q stream ended without result", tr.currentName)
	}
	return result, nil
}

// ── Loop detection ──

// buildHandoffTools returns the handoff tools for the current agent.
// Used to check whether the agent had any way to hand off.
func (tr *teamRunner) buildHandoffTools() []Tool {
	if tr.forceFinal {
		return nil
	}
	myName := tr.currentName
	tr.team.mu.Lock()
	defer tr.team.mu.Unlock()
	members := make([]AgentInfo, 0, len(tr.team.agents)-1)
	for _, name := range tr.team.order {
		if name == myName {
			continue
		}
		members = append(members, AgentInfo{
			Name:        name,
			Description: tr.team.agents[name].description,
			Type:        tr.team.agents[name].agentType,
		})
	}
	handoffTools := make([]Tool, 0, len(members))
	for _, m := range members {
		handoffTools = append(handoffTools, &handoffTool{
			record:   tr.recordHandoff,
			target:   m.Name,
			desc:     m.Description,
		})
	}
	return handoffTools
}

func (tr *teamRunner) detectLoop() string {
	if len(tr.chain) < 2 {
		return ""
	}

	n := len(tr.chain)

	// L1: Ping-pong: A→B→A→B
	if n >= 4 {
		a := tr.chain[n-4]
		b := tr.chain[n-3]
		c := tr.chain[n-2]
		d := tr.chain[n-1]
		if a.From == c.From && a.To == c.To && b.From == d.From && b.To == d.To {
			return fmt.Sprintf("Loop detected: %s → %s → %s → %s. You are stuck in a ping-pong.",
				a.From, a.To, c.From, c.To)
		}
	}

	// L2: Frequency: same agent appears ≥ 3 times in the chain.
	// Each handoff entry contributes both a From and a To —
	// both agents are involved in the handoff. The current
	// entry was already appended to the chain, so no extra
	// increment is needed.
	counts := make(map[string]int)
	for _, e := range tr.chain {
		counts[e.From]++
		counts[e.To]++
	}

	for name, c := range counts {
		if c >= 3 {
			return fmt.Sprintf("Agent %q has been called %d times. This may indicate a loop.", name, c)
		}
	}

	return ""
}

// isReturnHandoff reports whether target is handing off back to source
// after source previously handed off to target. Example: designer→coder,
// then coder→designer ("hey can you clarify X?").
//
// Return handoffs should be answered concisely and handed back — they are
// consultations, not forward workflow transitions.
func (tr *teamRunner) isReturnHandoff(from, to string) bool {
	for _, e := range tr.chain {
		if e.From == to && e.To == from {
			return true
		}
	}
	return false
}

// ── Agent preparation ──

// prepareRunner returns an AgentRunner ready for execution.
// For in-process agents (TeamPreparable), it injects team context
// (handoff tools + system prompt). For external agents, it returns as-is.
//
// Returns nil if the agent was removed between route validation and
// preparation (TOCTOU). Callers must handle nil.
func (tr *teamRunner) prepareRunner(name string) AgentRunner {
	tr.team.mu.Lock()
	entry := tr.team.agents[name]
	tr.team.mu.Unlock()

	if entry == nil {
		return nil
	}

	runner := entry.runner
	if tp, ok := runner.(TeamPreparable); ok {
		return tp.PrepareForTeam(tr.buildTeamContext(name))
	}
	return runner
}

// buildTeamContext assembles per-run team metadata for agent preparation.
func (tr *teamRunner) buildTeamContext(myName string) TeamContext {
	tr.team.mu.Lock()
	defer tr.team.mu.Unlock()

	// Collect other members (for handoff targets and prompt).
	members := make([]AgentInfo, 0, len(tr.team.agents)-1)
	for _, name := range tr.team.order {
		if name == myName {
			continue
		}
		members = append(members, AgentInfo{
			Name:        name,
			Description: tr.team.agents[name].description,
			Type:        tr.team.agents[name].agentType,
		})
	}

	// Build handoff tools unless forceFinal.
	var handoffTools []Tool
	if !tr.forceFinal {
		handoffTools = make([]Tool, 0, len(members))
		for _, m := range members {
			handoffTools = append(handoffTools, &handoffTool{
				record: tr.recordHandoff,
				target: m.Name,
				desc:   m.Description,
			})
		}
	}

	return TeamContext{
		MyName:         myName,
		Members:        members,
		HandoffHistory: tr.chain,
		ForceFinal:     tr.forceFinal,
		RecordHandoff:  tr.recordHandoff,
		TeamPrompt:    tr.buildTeamPrompt(myName, tr.forceFinal),
		HandoffTools:   handoffTools,
	}
}

// buildTeamPrompt builds the "## Team Context" system prompt block.
// Caller must hold tr.team.mu.
func (tr *teamRunner) buildTeamPrompt(myName string, forceFinal bool) string {
	var b strings.Builder
	b.WriteString("## Team Context\n")
	b.WriteString(fmt.Sprintf("You are %q in a multi-agent team.\n\n", myName))
	b.WriteString("Other team members you can hand off to:\n")

	for _, name := range tr.team.order {
		if name == myName {
			continue
		}
		entry := tr.team.agents[name]
		b.WriteString(fmt.Sprintf("- **%s** [%s]: %s\n", name, entry.agentType, entry.description))
	}

	if forceFinal {
		b.WriteString("\n⚠️ NO HANDOFFS: You have been instructed to produce a final answer yourself.\n")
	} else {
		b.WriteString("\nUse transfer_to_<name> to hand off. Include a clear message explaining what you need.\n")
		b.WriteString("When your part is done (no handoff needed), just respond directly.\n")
	}

	if len(tr.chain) > 0 {
		b.WriteString("\n## Handoff History\n")
		for i, e := range tr.chain {
			b.WriteString(fmt.Sprintf("%d. %s → %s: %s\n", i+1, e.From, e.To, truncateStr(e.Message, 120)))
		}
	}

	return b.String()
}

// agentInfos returns the list of agents for Route().
func (tr *teamRunner) agentInfos() []AgentInfo {
	tr.team.mu.Lock()
	defer tr.team.mu.Unlock()

	infos := make([]AgentInfo, 0, len(tr.team.agents))
	for _, name := range tr.team.order {
		e := tr.team.agents[name]
		infos = append(infos, AgentInfo{Name: name, Description: e.description, Type: e.agentType})
	}
	return infos
}

// finalize builds the final TeamResult.
func (tr *teamRunner) finalize(output string, ch chan<- TeamEvent) *TeamResult {
	r := &TeamResult{
		FinalOutput:  output,
		HandoffChain: tr.chain,
		TotalTurns:   tr.totalTurns,
		Usage:        tr.totalUsage,
	}
	if ch != nil {
		ch <- TeamEvent{Type: TeamDone, Result: r}
	}
	return r
}

// ── handoffTool ──

// handoffTool is a Tool implementation for transfer_to_<name>.
// When executed, it calls record to register the handoff request.
// The teamRunner provides the record callback which writes to
// Team.pendingHandoffs under mutex.
type handoffTool struct {
	record func(target, message string)
	target string
	desc   string
}

func (t *handoffTool) Definition() FunctionDefinition {
	return FunctionDefinition{
		Name:        "transfer_to_" + t.target,
		Description: fmt.Sprintf("Hand off to the %s agent. %s. Include a clear message about what you need.", t.target, t.desc),
		EndTurn:    true,
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"message": {
					"type": "string",
					"description": "What you need from the other agent. Be specific."
				}
			},
			"required": ["message"]
		}`),
	}
}

func (t *handoffTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("transfer_to_%s: %w", t.target, err)
	}

	t.record(t.target, params.Message)
	return fmt.Sprintf("Transferred to %s.", t.target), nil
}

// recordHandoff is the per-run callback that handoffTool invokes.
// Writes to the teamRunner's own pendingHandoffs (isolated per Run call).
func (tr *teamRunner) recordHandoff(target, message string) {
	tr.handoffMu.Lock()
	tr.pendingHandoffs = append(tr.pendingHandoffs, handoffRequest{
		target:  target,
		message: message,
	})
	tr.handoffMu.Unlock()
}

// ── Helpers ──

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

package kernel

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/agent"
)

// AsTool wraps an agent config + deps as a Tool so a coordinator can
// delegate sub-tasks in parallel. Unlike handoff (transfer_to_*), the
// sub-agent runs with isolated context and returns its output.
//
// The wrapped sub-agent runs with MaxTurns=3 and never spawns further
// sub-agents. Deps are used as given — pass MemoryProvider to share
// project knowledge (v2.0 §22), or leave empty for a fully isolated
// delegation. Unlike config-driven sub-agents (Agent.SubAgents), the
// default policy does NOT auto-allow delegation calls for AsTool tools:
// apps that gate them must supply rules via Deps.Policy. For
// config-driven sub-agents, the runtime builds the tool via
// newSubAgentTool with inherited deps.
func AsTool(cfg *agent.Agent, deps Deps) openagent.Tool {
	subCfg := cfg.Clone()
	subCfg.MaxTurns = 3
	subCfg.SubAgents = nil
	subCfg.Prompt = nil
	return &subAgentTool{cfg: subCfg, deps: deps}
}

// subAgentTool implements Tool by running a nested Runtime. The model
// supplies only the task; the agent identity (name, system prompt, tools,
// model) is fixed by config.
type subAgentTool struct {
	cfg  *agent.Agent
	deps Deps
	// toolFilter resolves the child's tool set at call time from the
	// parent's current tool snapshot (nil = use deps.Tools as-is).
	toolFilter func() []openagent.Tool
	// reg tracks spawned children so they're resumable via sub_agent_send.
	reg *childRegistry
}

// newSubAgentTool builds the delegation tool for a configured sub-agent.
// The child inherits the parent's runtime deps — knowledge, policy,
// approver, hooks, observer — but isolates its conversation: each spawn
// gets a fresh in-memory SessionStore+Compressor (memSessionStore) so the
// child's history accumulates across continue calls but never touches the
// parent's on-disk store. Its tool set is resolved from the parent snapshot
// at run time (per runChild), minus all sub-agent tools, narrowed to sa.Tools
// when an allowlist is configured.
func (rt *Runtime) newSubAgentTool(sa agent.SubAgent, reg *childRegistry) *subAgentTool {
	subCfg := rt.cfg.Clone()
	subCfg.Name = sa.Name
	subCfg.Description = sa.Description
	subCfg.SubAgents = nil // no nested delegation
	subCfg.MaxTurns = sa.MaxTurns
	if subCfg.MaxTurns == 0 {
		subCfg.MaxTurns = 30
	}
	if sa.Model != nil {
		subCfg.Model = sa.Model
	}
	subCfg.SystemPrompts = []string{sa.SystemPrompt}
	if strings.TrimSpace(sa.SystemPrompt) == "" {
		subCfg.SystemPrompts = []string{fmt.Sprintf(
			"You are the %s sub-agent. Complete the delegated task in an isolated context, using the available tools. Ask for clarification only if the task is ambiguous.", sa.Name)}
	}

	deps := rt.deps
	deps.Tools = nil                      // resolved at call time
	deps.SessionStore = nil               // isolated conversation
	deps.Compressor = nil                 // isolated: no shared summary
	deps.HumanApprover = rt.humanApprover // runtime-resolved (acp may swap mid-run)
	deps.ApprovalMemory = rt.approvalMemory

	names := rt.subAgentNames()
	allow := sa.Tools
	exclude := nameSet(rt.deps.SubAgentExcludeTools)
	return &subAgentTool{
		cfg:  subCfg,
		deps: deps,
		toolFilter: func() []openagent.Tool {
			return filterChildTools(rt.SnapshotTools(), names, exclude, allow)
		},
		reg: reg,
	}
}

// subAgentNames returns the names of all configured sub-agents (children
// never see other sub-agent tools — no recursion).
func (rt *Runtime) subAgentNames() map[string]bool {
	names := make(map[string]bool, len(rt.cfg.SubAgents))
	for _, sa := range rt.cfg.SubAgents {
		names[sa.Name] = true
	}
	return names
}

func nameSet(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	return set
}

// filterChildTools builds a sub-agent's tool set: all parent tools except
// other sub-agents and the caller-supplied exclusions (Deps.
// SubAgentExcludeTools — e.g. session-mode tools whose callbacks are
// session-bound and would be nil in the child), narrowed to the allowlist
// when one is configured.
func filterChildTools(snapshot []openagent.Tool, subAgentNames, exclude map[string]bool, allowlist []string) []openagent.Tool {
	var allow map[string]bool
	if allowlist != nil {
		allow = make(map[string]bool, len(allowlist))
		for _, n := range allowlist {
			allow[n] = true
		}
	}
	out := make([]openagent.Tool, 0, len(snapshot))
	for _, t := range snapshot {
		n := t.Definition().Name
		if subAgentNames[n] || exclude[n] {
			continue
		}
		if allow != nil && !allow[n] {
			continue
		}
		out = append(out, t)
	}
	return out
}

func (t *subAgentTool) Definition() openagent.FunctionDefinition {
	name := t.cfg.Name
	desc := t.cfg.Description
	if desc == "" {
		desc = "Handle a task delegated by the coordinator."
	}
	return openagent.FunctionDefinition{
		Name: name,
		Description: fmt.Sprintf("Delegate a focused investigation to the %s sub-agent. %s "+
			"Launch multiple sub-agents in one turn to cover different areas in parallel. "+
			"It runs in the background and notifies you on completion — you will receive a system-reminder with the result, "+
			"so do not duplicate its work while waiting. "+
			"The result includes an agent_id you can pass to sub_agent_send to send follow-up questions to the SAME sub-agent, "+
			"which keeps its conversation history so you don't need to re-explain context. "+
			"Up to %d sub-agents may run concurrently; excess calls are rejected — wait for some to finish first.",
			name, desc, maxConcurrentSubAgents),
		Parameters: openagent.SchemaOf[DelegateParams](),
	}
}

// resolveDeps materializes the child's tool set at call time (the parent
// snapshot may have grown since registration — plan tools, etc.).
func (t *subAgentTool) resolveDeps() Deps {
	deps := t.deps
	if t.toolFilter != nil {
		deps.Tools = t.toolFilter()
	}
	return deps
}

// sessionFromContext returns the current run's session (injected into the
// tool context by the execution runtime), or an empty one outside a run.
// Shared by subAgentTool and sendTool.
func sessionFromContext(ctx context.Context) openagent.Session {
	s, _ := openagent.SessionFromContext(ctx)
	return s
}

// formatWithAgentID prefixes a sub-agent reply with its agent_id so the model
// has a stable handle to pass to sub_agent_send for follow-up messages.
func formatWithAgentID(id, reply string) string {
	if id == "" {
		return reply
	}
	return fmt.Sprintf("[agent_id: %s]\n\n%s", id, reply)
}

// Execute runs the sub-agent and returns its final output.
// When a registry is wired with an onExit callback (ACP session mode), the
// child runs ASYNCHRONOUSLY: Execute returns immediately with the agent_id,
// and the result arrives later via the SDK mux's TriggerTurn (an idle turn
// fully serialized with user turns via sessionLocks). When reg is nil (AsTool)
// or onExit is not set (CLI one-shot), it runs synchronously — blocking.
func (t *subAgentTool) Execute(ctx context.Context, args json.RawMessage) *openagent.ToolResult {
	params, err := openagent.ParseArgs[DelegateParams](args)
	if err != nil {
		return openagent.ErrorResult(fmt.Errorf("agent tool %q: %w", t.cfg.Name, err), false, "")
	}
	if t.reg == nil {
		// One-shot: no registry, no persistence (AsTool path).
		output, err := runChild(ctx, t.cfg, t.resolveDeps(), sessionFromContext(ctx), params.Task, nil, "")
		if err != nil {
			return &openagent.ToolResult{
				Error: &openagent.ToolError{Message: fmt.Sprintf("agent tool %q: %v", t.cfg.Name, err)},
			}
		}
		return &openagent.ToolResult{Content: output}
	}
	child := t.reg.spawn(t.deps, t.cfg, t.toolFilter)

	// Async path: onExit wired → run in background, return immediately.
	if t.reg.hasOnExit() {
		if err := t.reg.startAsync(child, sessionFromContext(ctx), params.Task, params.Description); err != nil {
			return openagent.ErrorResult(fmt.Errorf("agent tool %q: %w", t.cfg.Name, err), false, "")
		}
		return &openagent.ToolResult{Content: fmt.Sprintf(
			"Sub-agent %s started in the background. You will be notified when it completes — do NOT duplicate its work while waiting. Use sub_agent_send to send follow-up messages.",
			child.id)}
	}

	// Sync path: run to completion, return the result.
	output, err := runChild(ctx, child.cfg, child.resolveDeps(), sessionFromContext(ctx), params.Task, nil, child.sessionID)
	if err != nil {
		return &openagent.ToolResult{
			Error: &openagent.ToolError{Message: fmt.Sprintf("agent tool %q: %v", t.cfg.Name, err)},
		}
	}
	return &openagent.ToolResult{Content: formatWithAgentID(child.id, output)}
}

// ExecuteStream runs the sub-agent with streaming. Text deltas and tool
// results are emitted as ToolStreamChunk events so the coordinator can
// show real-time progress; the stream ends with the final output.
func (t *subAgentTool) ExecuteStream(ctx context.Context, args json.RawMessage) <-chan openagent.ToolStreamChunk {
	params, err := openagent.ParseArgs[DelegateParams](args)
	if err != nil {
		ch := make(chan openagent.ToolStreamChunk, 1)
		ch <- openagent.ToolStreamChunk{Error: fmt.Errorf("agent tool %q: %w", t.cfg.Name, err)}
		close(ch)
		return ch
	}

	ch := make(chan openagent.ToolStreamChunk, 16)
	go func() {
		defer close(ch)
		if t.reg == nil {
			// One-shot: no registry, no persistence (AsTool path).
			output, err := runChild(ctx, t.cfg, t.resolveDeps(), sessionFromContext(ctx), params.Task, func(ev openagent.StreamEvent) {
				text := ev.Text
				if ev.Type == openagent.StreamToolResult {
					text = ev.Message.Content
				}
				if text != "" {
					ch <- openagent.ToolStreamChunk{Content: text}
				}
			}, "")
			if err != nil {
				ch <- openagent.ToolStreamChunk{Error: err}
				return
			}
			ch <- openagent.ToolStreamChunk{Content: output}
			return
		}
		spawned := t.reg.spawn(t.deps, t.cfg, t.toolFilter)
		// Async path: onExit wired → run in background, return immediately.
		if t.reg.hasOnExit() {
			if err := t.reg.startAsync(spawned, sessionFromContext(ctx), params.Task, params.Description); err != nil {
				ch <- openagent.ToolStreamChunk{Error: fmt.Errorf("agent tool %q: %w", t.cfg.Name, err)}
				return
			}
			ch <- openagent.ToolStreamChunk{Content: fmt.Sprintf(
				"Sub-agent %s started in the background. You will be notified when it completes — do NOT duplicate its work while waiting.", spawned.id)}
			return
		}
		// Sync path: stream the child's progress.
		output, err := runChild(ctx, spawned.cfg, spawned.resolveDeps(), sessionFromContext(ctx), params.Task, func(ev openagent.StreamEvent) {
			text := ev.Text
			if ev.Type == openagent.StreamToolResult {
				text = ev.Message.Content
			}
			if text != "" {
				ch <- openagent.ToolStreamChunk{Content: text}
			}
		}, spawned.sessionID)
		if err != nil {
			ch <- openagent.ToolStreamChunk{Error: err}
			return
		}
		ch <- openagent.ToolStreamChunk{Content: formatWithAgentID(spawned.id, output)}
	}()
	return ch
}

// DelegateParams are the arguments to a delegation tool (kernel.AsTool or
// a configured sub-agent): a short description for the tool card title +
// the task itself. The agent identity comes from config, not the args.
type DelegateParams struct {
	Description string `json:"description,omitempty" jsonschema:"description=Short label (3-7 words) for this delegation, shown in the progress UI"`
	Task        string `json:"task" jsonschema:"description=The task to complete"`
}

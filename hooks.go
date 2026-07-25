package openagent

import (
	"context"
	"encoding/json"
)

// RunHooks provides lifecycle callbacks in the Runner mainline.
// Naming follows OpenAI Agents SDK RunHooks conventions.
// nil RunHooks = no callbacks.
//
// OnAgentStart and OnToolStart return an opaque value that the Runner
// hands back to the corresponding End method. Implementations use this
// to carry state from start to finish: an OTEL span, a start timestamp,
// a WASM guest handle — the Runner never inspects it.
//
// OnToolEnd receives result and err as pointers so that hooks can
// mutate them (redaction, truncation, metadata injection) before the
// result is stored in memory.
type RunHooks interface {
	// OnAgentStart is called once when agent.Run() begins, before the loop.
	OnAgentStart(ctx context.Context, req ChatCompletionRequest) (any, error)
	// OnAgentEnd is called once when agent.Run() finishes (success, error, or cancel).
	OnAgentEnd(ctx context.Context, req ChatCompletionRequest, resp *ChatCompletionResponse, runErr error, startState any)
	// OnToolStart is called before each Tool.Execute.
	OnToolStart(ctx context.Context, tool FunctionDefinition, args json.RawMessage) (any, error)
	// OnToolEnd is called after each Tool.Execute finishes.
	// result and err are pointers — hooks may mutate them before memory storage.
	OnToolEnd(ctx context.Context, tool FunctionDefinition, args json.RawMessage, result *string, err *error, startState any)
}

// ── Multi-run-hooks combiner ──

// multiHookEntry pairs a hook with its index in the list, used to map
// start-state slices back to the correct hook on end calls.
type multiHookEntry struct {
	idx  int
	hook RunHooks
}

// multiHookState bundles the per-hook start states returned by
// OnAgentStart or OnToolStart.
type multiHookState struct {
	states []any // states[i] corresponds to list[i].hook
}

// MultiRunHooks combines multiple RunHooks into one.
// Each hook is called in order; one hook failing on start does not prevent
// subsequent hooks from running. Nil hooks are skipped. On end calls,
// result/err are passed through each hook in the same order, allowing
// chained mutations (e.g. artifact hook truncation followed by slog logging).
func MultiRunHooks(hooks ...RunHooks) RunHooks {
	var filtered []multiHookEntry
	for i, h := range hooks {
		if h != nil {
			filtered = append(filtered, multiHookEntry{idx: i, hook: h})
		}
	}
	switch len(filtered) {
	case 0:
		return nil
	case 1:
		return filtered[0].hook
	default:
		return &multiRunHooks{list: filtered}
	}
}

type multiRunHooks struct {
	list []multiHookEntry
}

func (m *multiRunHooks) OnAgentStart(ctx context.Context, req ChatCompletionRequest) (any, error) {
	states := make([]any, len(m.list))
	for i, e := range m.list {
		s, err := e.hook.OnAgentStart(ctx, req)
		states[i] = s
		if err != nil {
			return &multiHookState{states: states}, err
		}
	}
	return &multiHookState{states: states}, nil
}

func (m *multiRunHooks) OnAgentEnd(ctx context.Context, req ChatCompletionRequest, resp *ChatCompletionResponse, runErr error, startState any) {
	if startState == nil {
		return
	}
	ms, ok := startState.(*multiHookState)
	if !ok {
		return
	}
	if len(ms.states) != len(m.list) {
		return
	}
	for i, e := range m.list {
		e.hook.OnAgentEnd(ctx, req, resp, runErr, ms.states[i])
	}
}

func (m *multiRunHooks) OnToolStart(ctx context.Context, tool FunctionDefinition, args json.RawMessage) (any, error) {
	states := make([]any, len(m.list))
	for i, e := range m.list {
		s, err := e.hook.OnToolStart(ctx, tool, args)
		states[i] = s
		if err != nil {
			return &multiHookState{states: states}, err
		}
	}
	return &multiHookState{states: states}, nil
}

func (m *multiRunHooks) OnToolEnd(ctx context.Context, tool FunctionDefinition, args json.RawMessage, result *string, err *error, startState any) {
	if startState == nil {
		return
	}
	ms, ok := startState.(*multiHookState)
	if !ok {
		return
	}
	if len(ms.states) != len(m.list) {
		return
	}
	for i, e := range m.list {
		e.hook.OnToolEnd(ctx, tool, args, result, err, ms.states[i])
	}
}

// Package openagent provides a fully pluggable AI Agent, modeled after
// industry best practices and standard protocols.
//
// A minimal Agent needs only a Model. Every other capability (Memory, Guard,
// Approval, RunHooks, Observer, Skills, Sandbox) is a pluggable module — nil
// means the capability is absent, and the Runner skips the corresponding node.
//
// Basic usage:
//
//	agent := openagent.NewAgent("assistant",
//	    openagent.WithModel(openai.New("gpt-4o")),
//	    openagent.WithSystemPrompts("You are a helpful assistant."),
//	    openagent.WithTools(myTool1, myTool2),
//	)
//	result, err := agent.Run(ctx, session, openagent.UserMessage("hello"))
//
// Goal mode — the agent iterates autonomously until the goal is achieved:
//
//	result, err := agent.RunGoal(ctx, session, "Fix all failing tests")
//
// Multi-agent orchestration:
//
//   - Team: agents hand off to each other at runtime via transfer_to_* tools
//   - Plan: a Planner generates a DAG, then an Executor runs steps in
//     topological order with parallel batches and automatic replanning
//
// The Runner mainline executes 8 immutable nodes in order per turn:
//
//	① Memory → ② Prompt → ③ Guard.in → ④ Model → ⑤ Guard.out → ⑥ Approval → ⑦ Tools → ⑧ Memory
//
// Observability: inject a RunObserver to receive stage-level enter/leave
// events with wall-clock durations — useful for pipeline panels, tracing,
// and performance monitoring. Use RunHooks for agent/tool lifecycle events.
//
// See DESIGN.md for the full architecture.
package openagent

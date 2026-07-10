package openagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Agent is a configured agent ready to run. It holds a Model plus optional
// pluggable modules. All modules default to nil — the Runner skips nil modules.
//
// Create via NewAgent + WithXxx options:
//
//	agent := openagent.NewAgent("billing",
//	    openagent.WithModel(openaiModel),
//	    openagent.WithTools(sqlTool, httpTool),
//	    openagent.WithMemory(sqliteMem),
//	)
//	result, err := agent.Run(ctx, session, input)
type Agent struct {
	Name         string
	Description  string
	Instructions string

	Model Model
	Tools []Tool

	// Pluggable modules — nil means the capability is absent.
	Memory      Memory
	Prompt      PromptBuilder
	InGuard     InputGuard
	OutGuard    OutputGuard
	Approver    Approver
	Hooks       RunHooks
	Observer    RunObserver
	SkillLoader SkillLoader

	// Configuration
	MaxTurns             int // max loop iterations, 0 = default (20)
	MaxWorkingTokens     int // max tokens for working set before compaction; 0 = 70% of model context window
	MaxCompressedTokens  int // max tokens for compressed summary, 0 = no limit (default 2048)
}

// Clone returns a shallow copy of the Agent that is safe to mutate.
// Strings and ints are copied by value. Interface fields (Model, Memory,
// Approver, etc.) share the same underlying implementation — this is
// correct because you don't want a new DB connection or LLM client per clone.
// The Tools slice header is copied but gets its own backing array so the
// caller can append/remove tools without affecting the original.
func (a *Agent) Clone() *Agent {
	clone := *a
	if len(a.Tools) > 0 {
		clone.Tools = make([]Tool, len(a.Tools))
		copy(clone.Tools, a.Tools)
	}
	return &clone
}

// NewAgent creates an Agent with the given name and options.
func NewAgent(name string, opts ...AgentOption) *Agent {
	a := &Agent{
		Name:                name,
		MaxTurns:            20,
		MaxWorkingTokens:    0, // 0 = auto: 70% of model context window
		MaxCompressedTokens: 2048,
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

// Run executes one conversation turn. It creates a runner internally and
// blocks until the run completes or max turns is reached. Uses streaming
// internally for lower time-to-first-token, but returns the full result.
func (a *Agent) Run(ctx context.Context, session Session, input Message) (*RunResult, error) {
	if a.Model == nil {
		return nil, fmt.Errorf("agent %q has no Model", a.Name)
	}
	r := &runner{agent: a}
	return r.run(ctx, session, nil, input, nil)
}

// RunWithPrefix is like Run, but prefix messages are injected into the prompt
// after Memory.Recent() and before input. prefix messages are transient — they
// participate in this run only and are NOT persisted to Memory.
//
// Use this for cross-agent handoff context, multi-turn external context
// injection, or any scenario where you need to carry prior messages (including
// multimodal content) across agent invocations without side effects.
func (a *Agent) RunWithPrefix(ctx context.Context, session Session, prefix []Message, input Message) (*RunResult, error) {
	if a.Model == nil {
		return nil, fmt.Errorf("agent %q has no Model", a.Name)
	}
	r := &runner{agent: a}
	return r.run(ctx, session, prefix, input, nil)
}

// RunStream executes with a real-time event stream. Use for TUI or Web UI
// where individual text deltas, tool calls, and retry notifications should
// be rendered as they occur.
func (a *Agent) RunStream(ctx context.Context, session Session, input Message) <-chan StreamEvent {
	ch := make(chan StreamEvent, 16)
	go func() {
		defer close(ch)
		if a.Model == nil {
			ch <- StreamEvent{Type: StreamError, Error: fmt.Errorf("agent %q has no Model", a.Name)}
			return
		}
		r := &runner{agent: a}
		r.run(ctx, session, nil, input, ch)
	}()
	return ch
}

// RunStreamWithPrefix is like RunStream, but prefix messages are injected into
// the prompt after Memory.Recent() and before input. prefix messages are
// transient — they participate in this run only and are NOT persisted to Memory.
func (a *Agent) RunStreamWithPrefix(ctx context.Context, session Session, prefix []Message, input Message) <-chan StreamEvent {
	ch := make(chan StreamEvent, 16)
	go func() {
		defer close(ch)
		if a.Model == nil {
			ch <- StreamEvent{Type: StreamError, Error: fmt.Errorf("agent %q has no Model", a.Name)}
			return
		}
		r := &runner{agent: a}
		r.run(ctx, session, prefix, input, ch)
	}()
	return ch
}

// PrepareForTeam implements TeamPreparable. It returns a cloned *Agent
// with team context injected: handoff tools appended to Tools and the
// team system prompt prepended to Instructions.
//
// This is called by Team.prepareRunner before each agent turn.
// The original Agent is not modified.
func (a *Agent) PrepareForTeam(tc TeamContext) AgentRunner {
	clone := a.Clone()
	if !tc.ForceFinal {
		clone.Tools = append(clone.Tools, tc.HandoffTools...)
	}
	clone.Instructions = tc.TeamPrompt + "\n\n" + a.Instructions
	return clone
}

// ── Goal mode ──

// RunGoal executes the agent with a persistent goal. Unlike Run() where the
// input is a one-shot user message that scrolls out of context, the goal is
// injected into the system instructions — it persists across all turns,
// keeping the agent focused regardless of conversation length.
//
// The agent is instructed to iterate autonomously: plan, execute tools,
// observe results, and continue until the goal is achieved or determined
// impossible. The original Instructions are preserved as sub-instructions.
//
// Usage:
//
//	result, err := agent.RunGoal(ctx, session, "Set up a PostgreSQL database with a users table")
func (a *Agent) RunGoal(ctx context.Context, session Session, goal string) (*RunResult, error) {
	if a.Model == nil {
		return nil, fmt.Errorf("agent %q has no Model", a.Name)
	}
	clone := a.withGoalInstructions(goal)
	r := &runner{agent: clone}
	return r.run(ctx, session, nil, UserMessage(goal), nil)
}

// RunGoalStream is the streaming variant of RunGoal.
func (a *Agent) RunGoalStream(ctx context.Context, session Session, goal string) <-chan StreamEvent {
	ch := make(chan StreamEvent, 16)
	go func() {
		defer close(ch)
		if a.Model == nil {
			ch <- StreamEvent{Type: StreamError, Error: fmt.Errorf("agent %q has no Model", a.Name)}
			return
		}
		clone := a.withGoalInstructions(goal)
		r := &runner{agent: clone}
		r.run(ctx, session, nil, UserMessage(goal), ch)
	}()
	return ch
}

// withGoalInstructions returns a clone with goal-oriented instructions.
func (a *Agent) withGoalInstructions(goal string) *Agent {
	clone := a.Clone()
	clone.Instructions = buildGoalInstructions(a.Instructions, goal, a.MaxTurns)
	return clone
}

// buildGoalInstructions wraps the agent's instructions with goal-mode framing.
func buildGoalInstructions(original, goal string, maxTurns int) string {
	var b strings.Builder
	b.WriteString("## Goal\n\n")
	b.WriteString(goal)
	b.WriteString("\n\n---\n\n")
	b.WriteString("You are in autonomous goal mode. You must work toward this goal without further user input.\n\n")
	b.WriteString("**Rules:**\n")
	b.WriteString("- Plan your approach, then execute step by step\n")
	b.WriteString("- After each action, evaluate progress: what's done, what remains\n")
	b.WriteString("- If a step fails, diagnose and fix it yourself — do not give up\n")
	if maxTurns > 0 {
		b.WriteString(fmt.Sprintf("- You have up to %d iterations to complete this goal\n", maxTurns))
	}
	b.WriteString("- When the goal is fully achieved, respond with a summary of what was done\n")
	b.WriteString("- If you determine the goal is impossible, explain why and stop\n")
	b.WriteString("\n---\n\n")
	b.WriteString("## Instructions\n\n")
	b.WriteString(original)
	return b.String()
}

// ── Run result ──

// RunResult is the output of an Agent.Run call.
type RunResult struct {
	AgentName     string    // name of the agent that produced this result
	Messages      []Message // all messages from this run
	FinalOutput   string    // last assistant message content
	TurnCount     int
	Usage         Usage  // total tokens used
	ContextWindow int    // model's context window size (0 if unknown)
	StopReason    string // "end_turn", "refusal", "cancelled", etc. (ACP agents)
}

// ── Stream events ──

// StreamEventType categorizes events emitted by RunStream.
type StreamEventType string

const (
	StreamThought        StreamEventType = "thought"      // reasoning content (o1, deepseek-r1)
	StreamTextDelta      StreamEventType = "text_delta"
	StreamToolCall       StreamEventType = "tool_call"
	StreamToolProgress   StreamEventType = "tool_progress"  // streaming tool output chunk
	StreamToolResult     StreamEventType = "tool_result"
	StreamRetrying       StreamEventType = "retrying"
	StreamDone           StreamEventType = "done"
	StreamError          StreamEventType = "error"
	StreamAborted        StreamEventType = "aborted" // context cancelled, deadline exceeded
)

// StreamEvent is emitted by RunStream for real-time rendering.
type StreamEvent struct {
	Type       StreamEventType
	Text       string     // text_delta, tool_progress
	Message    Message    // tool_call, tool_result
	Result     *RunResult // done
	Error      error      // error, retrying
	ToolCallID string     // tool_progress — disambiguates concurrent streaming tools
}

// ── Agent as Tool (parallel delegation) ──

// AsTool wraps this agent as a Tool so a coordinator can delegate sub-tasks
// in parallel. Unlike handoff (transfer_to_*), the sub-agent runs with
// isolated context and returns its output to the coordinator — the coordinator
// continues running after receiving results.
//
// Context isolation:
//   - New session per call (invisible to coordinator's memory)
//   - No prefix (no coordinator conversation history leaked)
//   - Only the task string as input
//
// Usage:
//
//	coordinator := openagent.NewAgent("coordinator",
//	    openagent.WithTools(
//	        researcher.AsTool(),
//	        writer.AsTool(),
//	    ),
//	)
func (a *Agent) AsTool() Tool {
	return &agentTool{agent: a}
}

// agentTool wraps an Agent as a Tool for parallel delegation.
type agentTool struct {
	agent *Agent
}

func (t *agentTool) Definition() FunctionDefinition {
	name := t.agent.Name
	desc := t.agent.Description
	if desc == "" {
		desc = "Handle a task delegated by the coordinator."
	}
	return FunctionDefinition{
		Name: name,
		Description: fmt.Sprintf("Ask the %s agent to complete a task. %s "+
			"Provide a clear, self-contained task description — the agent "+
			"does not see the coordinator's conversation history.", name, desc),
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"task": {
					"type": "string",
					"description": "The task to complete. Be specific and self-contained."
				}
			},
			"required": ["task"]
		}`),
	}
}

func (t *agentTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Task string `json:"task"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("agent tool %q: %w", t.agent.Name, err)
	}
	if params.Task == "" {
		return "", fmt.Errorf("agent tool %q: task is required", t.agent.Name)
	}

	// New session — fully isolated from the coordinator.
	session := Session{
		ID:        fmt.Sprintf("%s-%d", t.agent.Name, time.Now().UnixNano()),
		AgentName: t.agent.Name,
		CreatedAt: time.Now(),
	}

	input := UserMessage(params.Task)
	// prefix is nil — the sub-agent sees only its instructions + the task.
	result, err := t.agent.Run(ctx, session, input)
	if err != nil {
		return "", fmt.Errorf("agent %q: %w", t.agent.Name, err)
	}
	return result.FinalOutput, nil
}

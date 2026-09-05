package kernel

import (
	"context"
	"encoding/json"
	"fmt"

	openagent "github.com/yusheng-g/openagent-go"
)

// sendTool implements the "sub_agent_send" tool: it delivers a follow-up
// message to a sub-agent launched earlier (via a subAgentTool like explorer),
// reusing the child's accumulated conversation history. The child keeps its
// in-memory SessionStore across calls, so the model can ask follow-up
// questions without re-explaining context from the original delegation.
//
// The child must have been spawned in this session (registry is session-scoped
// via Deps). An unknown agent_id — expired, evicted, or from another session —
// returns an error whose text steers the model to launch a fresh sub-agent.
type sendTool struct {
	reg *childRegistry
}

func newSendTool(reg *childRegistry) *sendTool {
	return &sendTool{reg: reg}
}

func (t *sendTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name: "sub_agent_send",
		Description: "Send a follow-up message to a sub-agent launched earlier (e.g. via explorer). " +
			"The sub-agent keeps its conversation history, so follow-up questions don't need " +
			"re-explained context. Pass the agent_id returned by the original sub-agent call.",
		Parameters: openagent.SchemaOf[SendParams](),
	}
}

func (t *sendTool) Execute(ctx context.Context, args json.RawMessage) *openagent.ToolResult {
	params, err := openagent.ParseArgs[SendParams](args)
	if err != nil {
		return openagent.ErrorResult(fmt.Errorf("sub_agent_send: %w", err), false, "")
	}
	if params.AgentID == "" {
		return openagent.ErrorResult(fmt.Errorf("sub_agent_send: agent_id is required"), false, "")
	}
	if params.Message == "" {
		return openagent.ErrorResult(fmt.Errorf("sub_agent_send: message is required"), false, "")
	}

	child, ok := t.reg.get(params.AgentID)
	if !ok {
		return openagent.ErrorResult(fmt.Errorf(
			"sub_agent_send: agent %q not found (it may have expired or was from another session); launch a fresh sub-agent instead",
			params.AgentID), false, "")
	}

	// Async path: onExit wired → run in background, return immediately.
	// startAsync atomically checks child.running + concurrency cap, so two
	// concurrent sends to the same child can't both start. The result arrives
	// via the same idle-turn mechanism as the initial spawn.
	if t.reg.hasOnExit() {
		if err := t.reg.startAsync(child, sessionFromContext(ctx), params.Message, params.Description); err != nil {
			return openagent.ErrorResult(fmt.Errorf("sub_agent_send: %w", err), false, "")
		}
		return &openagent.ToolResult{Content: fmt.Sprintf(
			"Message sent to sub-agent %s. You will be notified when it replies.", child.id)}
	}

	// Sync path: block until the child replies.
	child.mu.Lock()
	if child.running {
		child.mu.Unlock()
		return openagent.ErrorResult(fmt.Errorf(
			"sub_agent_send: agent %q is still processing a previous message; wait for it to finish before sending another",
			params.AgentID), false, "")
	}
	child.running = true
	child.mu.Unlock()
	defer func() {
		child.mu.Lock()
		child.running = false
		child.mu.Unlock()
	}()

	output, err := runChild(ctx, child.cfg, child.resolveDeps(), sessionFromContext(ctx), params.Message, nil, child.sessionID)
	if err != nil {
		return &openagent.ToolResult{
			Error: &openagent.ToolError{Message: fmt.Sprintf("sub_agent_send: %v", err)},
		}
	}
	return &openagent.ToolResult{Content: formatWithAgentID(child.id, output)}
}

// SendParams are the arguments to sub_agent_send.
type SendParams struct {
	Description string `json:"description,omitempty" jsonschema:"description=Short label (3-7 words) for this follow-up, shown in the progress UI"`
	AgentID     string `json:"agent_id" jsonschema:"description=The agent_id returned by the original sub-agent call (e.g. explorer-1)"`
	Message     string `json:"message" jsonschema:"description=The follow-up message or question for the sub-agent"`
}

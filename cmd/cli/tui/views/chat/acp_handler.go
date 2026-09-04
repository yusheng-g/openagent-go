package chat

import (
	"context"
	"encoding/json"
	"fmt"

	tea "charm.land/bubbletea/v2"

	openacp "github.com/yusheng-g/openagent-go/acp/sdk"
)

// acpEventHandler implements both openacp.EventHandler and
// openacp.ClientRequestHandler. The ACP SDK calls these methods from a
// background reader goroutine.
//
// Streaming events (EventHandler) are forwarded to the bubbletea model via
// Program.Send (one-way, fire-and-forget).
//
// Permission requests (ClientRequestHandler.HandleRequestPermission) need a
// response — a reply channel bridges the goroutine → TUI → goroutine round
// trip: the handler sends a msg with a channel, blocks on the channel; the
// TUI shows a dialog, the user picks an option, Update writes the response
// to the channel, the handler returns it to the server.
//
// Other ClientRequestHandler methods (fs, terminal) are not yet implemented.

// NewAcpEventHandler creates a handler that implements both EventHandler
// and ClientRequestHandler.
func NewAcpEventHandler(p *tea.Program) *acpEventHandler {
	return &acpEventHandler{program: p}
}

type acpEventHandler struct {
	program *tea.Program
}

// ── EventHandler ──

func (h *acpEventHandler) OnAgentMessage(text string) {
	h.program.Send(agentMessageMsg{text: text})
}

func (h *acpEventHandler) OnAgentThought(text string) {
	h.program.Send(agentThoughtMsg{text: text})
}

func (h *acpEventHandler) OnUserMessage(text string) {
	h.program.Send(userMessageMsg{text: text})
}

func (h *acpEventHandler) OnToolCall(tc openacp.ToolCallUpdate) {
	msg := toolCallMsg{id: tc.ToolCallID, title: tc.Title, status: toolRunning}
	switch tc.Status {
	case "completed":
		msg.status = toolDone
	case "failed":
		msg.status = toolFailed
	}
	if b, err := json.Marshal(tc.RawInput); err == nil && string(b) != "null" {
		msg.input = string(b)
	}
	if b, err := json.Marshal(tc.RawOutput); err == nil && string(b) != "null" {
		msg.output = string(b)
	}
	h.program.Send(msg)
}

func (h *acpEventHandler) OnPlan(plan openacp.Plan) {
	h.program.Send(planMsg{entries: plan.Entries})
}

func (h *acpEventHandler) OnAvailableCommandsUpdate(cmds []openacp.AvailableCommand) {
}

func (h *acpEventHandler) OnModeUpdate(modeID openacp.SessionModeId) {
	h.program.Send(modeUpdateMsg{mode: string(modeID)})
}

func (h *acpEventHandler) OnConfigOptionUpdate(opts []openacp.SessionConfigOption) {
	h.program.Send(configOptionsMsg{opts: opts})
}

func (h *acpEventHandler) OnUsageUpdate(used, total int, cost *openacp.Cost) {
	h.program.Send(usageUpdateMsg{used: used, total: total})
}

func (h *acpEventHandler) OnSessionInfo(title string, metadata map[string]any) {}

// ── ClientRequestHandler ──

// HandleRequestPermission sends the permission request to the TUI via
// Program.Send and blocks on a reply channel until the user selects an
// option. If the context is cancelled (e.g. user quits), returns the error.
func (h *acpEventHandler) HandleRequestPermission(ctx context.Context, req openacp.RequestPermissionRequest) (*openacp.RequestPermissionResponse, error) {
	replyCh := make(chan openacp.RequestPermissionResponse, 1)
	h.program.Send(permissionRequestMsg{req: req, replyCh: replyCh})
	select {
	case resp := <-replyCh:
		return &resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (h *acpEventHandler) HandleReadTextFile(ctx context.Context, req openacp.ReadTextFileRequest) (*openacp.ReadTextFileResponse, error) {
	return nil, fmt.Errorf("fs/read_text_file not implemented")
}

func (h *acpEventHandler) HandleWriteTextFile(ctx context.Context, req openacp.WriteTextFileRequest) (*openacp.WriteTextFileResponse, error) {
	return nil, fmt.Errorf("fs/write_text_file not implemented")
}

func (h *acpEventHandler) HandleCreateTerminal(ctx context.Context, req openacp.CreateTerminalRequest) (*openacp.CreateTerminalResponse, error) {
	return nil, fmt.Errorf("terminal/create not implemented")
}

func (h *acpEventHandler) HandleTerminalOutput(ctx context.Context, req openacp.TerminalOutputRequest) (*openacp.TerminalOutputResponse, error) {
	return nil, fmt.Errorf("terminal/output not implemented")
}

func (h *acpEventHandler) HandleWaitForTerminalExit(ctx context.Context, req openacp.WaitForTerminalExitRequest) (*openacp.WaitForTerminalExitResponse, error) {
	return nil, fmt.Errorf("terminal/wait not implemented")
}

func (h *acpEventHandler) HandleKillTerminal(ctx context.Context, req openacp.KillTerminalRequest) (*openacp.KillTerminalResponse, error) {
	return nil, fmt.Errorf("terminal/kill not implemented")
}

func (h *acpEventHandler) HandleReleaseTerminal(ctx context.Context, req openacp.ReleaseTerminalRequest) (*openacp.ReleaseTerminalResponse, error) {
	return nil, fmt.Errorf("terminal/release not implemented")
}

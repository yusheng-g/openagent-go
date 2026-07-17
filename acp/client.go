// Package acp provides an abstraction over the Agent Client Protocol (ACP).
//
// It defines openagent-go's own ACP types and a Client for connecting to
// external ACP agent processes. The default implementation uses
// coder/acp-go-sdk internally, but the interfaces are designed so that
// alternative implementations can be plugged in.
//
// Import as:
//
//	openacp "github.com/yusheng-g/openagent-go/acp"
package acp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"sync"

	acpsdk "github.com/coder/acp-go-sdk"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Client connects to external ACP agent processes.
//
// Create with [NewClient], then call [Client.ConnectStdio] to spawn
// an agent and communicate over stdin/stdout:
//
//	client := acp.NewClient("my-app", "1.0.0")
//	session, err := client.ConnectStdio(ctx, "my-acp-agent")
//	session.Initialize(ctx, acp.InitializeRequest{...})
//	session.NewSession(ctx, acp.NewSessionRequest{...})
//	session.SetEventHandler(myHandler)
//	resp, err := session.Prompt(ctx, acp.PromptRequest{...})
//	session.Close()
type Client struct {
	name    string
	version string
}

// NewClient creates an ACP [Client] with the given implementation identity.
func NewClient(name, version string) *Client {
	return &Client{name: name, version: version}
}

// ConnectStdio spawns an ACP agent as a subprocess and communicates over
// stdin/stdout. command is the path to the executable; args are its arguments.
//
// The context is used to start the process. Call [Session.Close] to terminate
// the process and clean up.
func (c *Client) ConnectStdio(ctx context.Context, command string, args ...string) (*Session, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("acp stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("acp stdout pipe: %w", err)
	}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("acp start %q: %w", command, err)
	}

	bridge := &sessionBridge{}
	conn := acpsdk.NewClientSideConnection(bridge, stdin, stdout)
	conn.SetLogger(slog.Default())

	return &Session{
		conn:      conn,
		cmd:       cmd,
		bridge:    bridge,
		stdin:     stdin,
		stderrBuf: &stderrBuf,
	}, nil
}

// ── Session ──

// Session is an active connection to an external ACP agent.
//
// Typical lifecycle:
//
//	session.Initialize(ctx, req)        // handshake
//	session.NewSession(ctx, req)        // create session (reusable)
//	session.SetEventHandler(handler)    // register streaming handler
//	session.Prompt(ctx, req)            // send prompt (blocks until done)
//	session.Close()                     // kill process + cleanup
//
// Session is safe for sequential use. For concurrent use, external
// synchronisation is required.
type Session struct {
	conn   *acpsdk.ClientSideConnection
	cmd    *exec.Cmd
	bridge *sessionBridge

	// stdin is the write end of the process's stdin pipe. Closing it
	// signals EOF to the process, allowing graceful shutdown before Kill.
	stdin io.WriteCloser

	// stderrBuf captures the process's stderr output for diagnostics.
	stderrBuf *bytes.Buffer

	mu        sync.Mutex
	sessionID string
}

// Initialize performs the ACP handshake. Must be called once after connect.
func (s *Session) Initialize(ctx context.Context, req InitializeRequest) (*InitializeResponse, error) {
	resp, err := s.conn.Initialize(ctx, acpsdk.InitializeRequest{
		ProtocolVersion: acpsdk.ProtocolVersion(req.ProtocolVersion),
		ClientInfo: &acpsdk.Implementation{
			Name: req.ClientName, Version: req.ClientVersion,
		},
		ClientCapabilities: acpsdk.ClientCapabilities{
			Terminal: req.ClientCapabilities.Terminal,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("acp initialize: %w", err)
	}

	agentName := ""
	agentVersion := ""
	if resp.AgentInfo != nil {
		agentName = resp.AgentInfo.Name
		agentVersion = resp.AgentInfo.Version
	}

	return &InitializeResponse{
		ProtocolVersion: int(resp.ProtocolVersion),
		AgentName:       agentName,
		AgentVersion:    agentVersion,
		Capabilities: AgentCapabilities{
			LoadSession: resp.AgentCapabilities.LoadSession,
			McpCapabilities: McpCapabilities{
				Acp:  resp.AgentCapabilities.McpCapabilities.Acp,
				Http: resp.AgentCapabilities.McpCapabilities.Http,
				Sse:  resp.AgentCapabilities.McpCapabilities.Sse,
			},
			PromptCapabilities: PromptCapabilities{
				Image:           resp.AgentCapabilities.PromptCapabilities.Image,
				Audio:           resp.AgentCapabilities.PromptCapabilities.Audio,
				EmbeddedContext: resp.AgentCapabilities.PromptCapabilities.EmbeddedContext,
			},
			SessionCapabilities: SessionCapabilities{
				List:   resp.AgentCapabilities.SessionCapabilities.List != nil,
				Delete: resp.AgentCapabilities.SessionCapabilities.Delete != nil,
				Resume: resp.AgentCapabilities.SessionCapabilities.Resume != nil,
				Close:  resp.AgentCapabilities.SessionCapabilities.Close != nil,
			},
		},
	}, nil
}

// NewSession creates a new ACP session. McpServers are passed to the agent
// so it can connect to them for tool calls (e.g. transfer_to_* for handoff).
//
// The returned session ID is stored internally and used in subsequent
// [Session.Prompt] calls. Only one session is active at a time.
func (s *Session) NewSession(ctx context.Context, req NewSessionRequest) (*NewSessionResponse, error) {
	mcpServers := make([]acpsdk.McpServer, 0, len(req.McpServers))
	for _, m := range req.McpServers {
		mcpServers = append(mcpServers, toSDKMcpServer(m))
	}

	resp, err := s.conn.NewSession(ctx, acpsdk.NewSessionRequest{
		Cwd:        req.Cwd,
		McpServers: mcpServers,
	})
	if err != nil {
		return nil, fmt.Errorf("acp new session: %w", err)
	}

	s.mu.Lock()
	s.sessionID = string(resp.SessionId)
	s.mu.Unlock()

	return &NewSessionResponse{
		SessionID:     string(resp.SessionId),
		ConfigOptions: fromSDKConfigOptions(resp.ConfigOptions),
		Modes:         fromSDKModeState(resp.Modes),
	}, nil
}

// SetEventHandler registers the handler that receives streaming events
// during [Session.Prompt].
func (s *Session) SetEventHandler(h EventHandler) {
	s.bridge.setHandler(h)
}

// Prompt sends a user prompt to the agent and blocks until the agent
// finishes processing. Streaming events are delivered to the registered
// [EventHandler] concurrently.
func (s *Session) Prompt(ctx context.Context, req PromptRequest) (*PromptResponse, error) {
	blocks := make([]acpsdk.ContentBlock, len(req.Blocks))
	for i, b := range req.Blocks {
		blocks[i] = acpsdk.ContentBlock{
			Text: &acpsdk.ContentBlockText{Text: b.Text},
		}
	}

	s.mu.Lock()
	sid := s.sessionID
	s.mu.Unlock()

	resp, err := s.conn.Prompt(ctx, acpsdk.PromptRequest{
		SessionId: acpsdk.SessionId(sid),
		Prompt:    blocks,
	})
	if err != nil {
		return nil, fmt.Errorf("acp prompt: %w", err)
	}

	return &PromptResponse{
		StopReason: string(resp.StopReason),
	}, nil
}

// DeleteSession permanently removes a session and all its data from the agent.
func (s *Session) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := s.conn.UnstableDeleteSession(ctx, acpsdk.UnstableDeleteSessionRequest{
		SessionId: acpsdk.SessionId(sessionID),
	})
	if err != nil {
		return fmt.Errorf("acp delete session: %w", err)
	}
	return nil
}

// CloseSession closes the current ACP session.
func (s *Session) CloseSession(ctx context.Context) error {
	s.mu.Lock()
	sid := s.sessionID
	s.mu.Unlock()

	if sid == "" {
		return nil
	}

	_, err := s.conn.CloseSession(ctx, acpsdk.CloseSessionRequest{
		SessionId: acpsdk.SessionId(sid),
	})
	return err
}

// ListSessions returns the sessions available on the agent.
func (s *Session) ListSessions(ctx context.Context, req ListSessionsRequest) (*ListSessionsResponse, error) {
	resp, err := s.conn.ListSessions(ctx, acpsdk.ListSessionsRequest{
		Cursor: req.Cursor,
		Cwd:    req.Cwd,
	})
	if err != nil {
		return nil, fmt.Errorf("acp list sessions: %w", err)
	}

	sessions := make([]SessionInfo, len(resp.Sessions))
	for i, info := range resp.Sessions {
		title := ""
		if info.Title != nil {
			title = *info.Title
		}
		updatedAt := ""
		if info.UpdatedAt != nil {
			updatedAt = *info.UpdatedAt
		}
		sessions[i] = SessionInfo{
			SessionID: string(info.SessionId),
			Cwd:       info.Cwd,
			Title:     title,
			UpdatedAt: updatedAt,
		}
	}

	return &ListSessionsResponse{
		NextCursor: resp.NextCursor,
		Sessions:   sessions,
	}, nil
}

// LoadSession resumes an existing session on the agent.
func (s *Session) LoadSession(ctx context.Context, req LoadSessionRequest) (*LoadSessionResponse, error) {
	mcpServers := make([]acpsdk.McpServer, 0, len(req.McpServers))
	for _, m := range req.McpServers {
		mcpServers = append(mcpServers, toSDKMcpServer(m))
	}

	_, err := s.conn.ResumeSession(ctx, acpsdk.ResumeSessionRequest{
		SessionId:            acpsdk.SessionId(req.SessionID),
		Cwd:                  req.Cwd,
		McpServers:           mcpServers,
		AdditionalDirectories: req.AdditionalDirectories,
	})
	if err != nil {
		return nil, fmt.Errorf("acp load session: %w", err)
	}

	s.mu.Lock()
	s.sessionID = req.SessionID
	s.mu.Unlock()

	return &LoadSessionResponse{}, nil
}

// SetSessionConfigOption sends a config option change to the agent.
func (s *Session) SetSessionConfigOption(ctx context.Context, sessionID string, opt SetConfigOption) error {
	var req acpsdk.SetSessionConfigOptionRequest
	switch opt.Value.(type) {
	case bool:
		v := opt.Value.(bool)
		req.Boolean = &acpsdk.SetSessionConfigOptionBoolean{
			SessionId: acpsdk.SessionId(sessionID),
			ConfigId:  acpsdk.SessionConfigId(opt.ID),
			Value:     v,
			Type:      "boolean",
		}
	default:
		v, _ := opt.Value.(string)
		req.ValueId = &acpsdk.SetSessionConfigOptionValueId{
			SessionId: acpsdk.SessionId(sessionID),
			ConfigId:  acpsdk.SessionConfigId(opt.ID),
			Value:     acpsdk.SessionConfigValueId(v),
		}
	}
	_, err := s.conn.SetSessionConfigOption(ctx, req)
	return err
}

// SetSessionMode changes the session mode on the agent.
func (s *Session) SetSessionMode(ctx context.Context, sessionID string, modeID string) error {
	_, err := s.conn.SetSessionMode(ctx, acpsdk.SetSessionModeRequest{
		SessionId: acpsdk.SessionId(sessionID),
		ModeId:    acpsdk.SessionModeId(modeID),
	})
	return err
}

// ForkSession creates a new session by forking an existing one on the agent.
func (s *Session) ForkSession(ctx context.Context, sessionID string) (*ForkSessionResponse, error) {
	resp, err := s.conn.UnstableForkSession(ctx, acpsdk.UnstableForkSessionRequest{
		SessionId: acpsdk.SessionId(sessionID),
	})
	if err != nil {
		return nil, fmt.Errorf("acp fork session: %w", err)
	}
	return &ForkSessionResponse{SessionID: string(resp.SessionId)}, nil
}

// Close terminates the ACP connection and cleans up the subprocess.
//
// It first closes stdin to signal EOF to the process, allowing it to exit
// gracefully. If the process doesn't exit on its own (it's not required to),
// it is killed. Stderr output is captured and available via [Session.Stderr].
func (s *Session) Close() error {
	// Close stdin first — signals EOF so the process can shut down gracefully.
	// The SDK connection reads from stdout until EOF; closing stdin triggers
	// the process to flush and exit.
	if s.stdin != nil {
		_ = s.stdin.Close()
	}

	// Force-kill the process as a safety net. If the process already exited
	// due to stdin closing, Kill is a no-op (Process is nil after Wait).
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
	}
	return nil
}

// Stderr returns the captured stderr output of the subprocess.
// Empty string if nothing was written to stderr. Use this for
// diagnostics when the process fails or behaves unexpectedly.
func (s *Session) Stderr() string {
	if s.stderrBuf == nil {
		return ""
	}
	return s.stderrBuf.String()
}

// ── sessionBridge ──

// sessionBridge implements acpsdk.Client to receive callbacks from the SDK
// and translate them to our EventHandler interface.
type sessionBridge struct {
	mu      sync.RWMutex
	handler EventHandler
}

func (b *sessionBridge) setHandler(h EventHandler) {
	b.mu.Lock()
	b.handler = h
	b.mu.Unlock()
}

func (b *sessionBridge) getHandler() EventHandler {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.handler
}

func (b *sessionBridge) SessionUpdate(ctx context.Context, params acpsdk.SessionNotification) error {
	h := b.getHandler()
	if h == nil {
		return nil
	}

	update := params.Update

	switch {
	case update.AgentMessageChunk != nil:
		h.OnAgentMessage(contentBlockTextSDK(update.AgentMessageChunk.Content))

	case update.AgentThoughtChunk != nil:
		h.OnAgentThought(contentBlockTextSDK(update.AgentThoughtChunk.Content))

	case update.ToolCall != nil:
		tc := update.ToolCall
		status := ""
		switch tc.Status {
		case acpsdk.ToolCallStatusCompleted:
			status = "completed"
		case acpsdk.ToolCallStatusInProgress:
			status = "in_progress"
		case acpsdk.ToolCallStatusFailed:
			status = "failed"
		}
		h.OnToolCall(ToolCallEvent{
			ID:        string(tc.ToolCallId),
			Title:     tc.Title,
			RawInput:  tc.RawInput,
			Status:    status,
			RawOutput: tc.RawOutput,
		})

	case update.ToolCallUpdate != nil:
		tu := update.ToolCallUpdate
		status := ""
		if tu.Status != nil {
			switch *tu.Status {
			case acpsdk.ToolCallStatusCompleted:
				status = "completed"
			case acpsdk.ToolCallStatusInProgress:
				status = "in_progress"
			case acpsdk.ToolCallStatusFailed:
				status = "failed"
			}
		}
		title := ""
		if tu.Title != nil {
			title = *tu.Title
		}
		h.OnToolCall(ToolCallEvent{
			ID:        string(tu.ToolCallId),
			Title:     title,
			RawInput:  tu.RawInput,
			Status:    status,
			RawOutput: tu.RawOutput,
		})

	case update.Plan != nil:
		if ph, ok := h.(PlanHandler); ok {
			entries := make([]PlanEntry, len(update.Plan.Entries))
			for i, e := range update.Plan.Entries {
				entries[i] = PlanEntry{
					Title:    e.Content,
					Priority: string(e.Priority),
					Status:   string(e.Status),
				}
			}
			ph.OnPlan(entries)
		}

	case update.AvailableCommandsUpdate != nil:
		if ch, ok := h.(CommandHandler); ok {
			cmds := make([]AvailableCommand, len(update.AvailableCommandsUpdate.AvailableCommands))
			for i, c := range update.AvailableCommandsUpdate.AvailableCommands {
				cmds[i] = AvailableCommand{Name: c.Name, Description: c.Description}
			}
			ch.OnAvailableCommands(cmds)
		}

	case update.CurrentModeUpdate != nil:
		if mh, ok := h.(ModeHandler); ok {
			mh.OnCurrentMode(string(update.CurrentModeUpdate.CurrentModeId))
		}

	case update.UserMessageChunk != nil:
		if mh, ok := h.(ModeHandler); ok {
			mh.OnUserMessage(contentBlockTextSDK(update.UserMessageChunk.Content))
		}

	case update.UsageUpdate != nil:
		if uh, ok := h.(UsageHandler); ok {
			uh.OnUsage(UsageInfo{
				Used: update.UsageUpdate.Used,
				Size: update.UsageUpdate.Size,
			})
		}
	}

	return nil
}

// Other Client methods — return "not supported".
func (b *sessionBridge) RequestPermission(ctx context.Context, params acpsdk.RequestPermissionRequest) (acpsdk.RequestPermissionResponse, error) {
	return acpsdk.RequestPermissionResponse{
		Outcome: acpsdk.RequestPermissionOutcome{
			Cancelled: &acpsdk.RequestPermissionOutcomeCancelled{Outcome: "cancelled"},
		},
	}, nil
}

func (b *sessionBridge) ReadTextFile(ctx context.Context, params acpsdk.ReadTextFileRequest) (acpsdk.ReadTextFileResponse, error) {
	return acpsdk.ReadTextFileResponse{}, fmt.Errorf("not supported")
}
func (b *sessionBridge) WriteTextFile(ctx context.Context, params acpsdk.WriteTextFileRequest) (acpsdk.WriteTextFileResponse, error) {
	return acpsdk.WriteTextFileResponse{}, fmt.Errorf("not supported")
}
func (b *sessionBridge) CreateTerminal(ctx context.Context, params acpsdk.CreateTerminalRequest) (acpsdk.CreateTerminalResponse, error) {
	return acpsdk.CreateTerminalResponse{}, fmt.Errorf("not supported")
}
func (b *sessionBridge) KillTerminal(ctx context.Context, params acpsdk.KillTerminalRequest) (acpsdk.KillTerminalResponse, error) {
	return acpsdk.KillTerminalResponse{}, fmt.Errorf("not supported")
}
func (b *sessionBridge) TerminalOutput(ctx context.Context, params acpsdk.TerminalOutputRequest) (acpsdk.TerminalOutputResponse, error) {
	return acpsdk.TerminalOutputResponse{}, fmt.Errorf("not supported")
}
func (b *sessionBridge) ReleaseTerminal(ctx context.Context, params acpsdk.ReleaseTerminalRequest) (acpsdk.ReleaseTerminalResponse, error) {
	return acpsdk.ReleaseTerminalResponse{}, fmt.Errorf("not supported")
}
func (b *sessionBridge) WaitForTerminalExit(ctx context.Context, params acpsdk.WaitForTerminalExitRequest) (acpsdk.WaitForTerminalExitResponse, error) {
	return acpsdk.WaitForTerminalExitResponse{}, fmt.Errorf("not supported")
}

// ── Helpers ──

// fromSDKConfigOptions converts SDK config options to our type via JSON round-trip.
func fromSDKConfigOptions(opts []acpsdk.SessionConfigOption) []SessionConfigOption {
	if len(opts) == 0 {
		return nil
	}
	data, err := json.Marshal(opts)
	if err != nil {
		return nil
	}
	var ours []SessionConfigOption
	json.Unmarshal(data, &ours)
	return ours
}

// fromSDKModeState converts SDK mode state to our type via JSON round-trip.
func fromSDKModeState(m *acpsdk.SessionModeState) *SessionModeState {
	if m == nil {
		return nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	var ours SessionModeState
	json.Unmarshal(data, &ours)
	return &ours
}

func toSDKMcpServer(m McpServer) acpsdk.McpServer {
	sdk := acpsdk.McpServer{}
	if m.URL != "" {
		sdk.Http = &acpsdk.McpServerHttpInline{Url: m.URL}
	}
	if m.Command != "" {
		sdk.Stdio = &acpsdk.McpServerStdio{
			Name:    m.Name,
			Command: m.Command,
			Args:    m.Args,
		}
	}
	return sdk
}

func contentBlockTextSDK(block acpsdk.ContentBlock) string {
	if block.Text != nil {
		return block.Text.Text
	}
	return ""
}

// Compile-time interface checks.
var _ acpsdk.Client = (*sessionBridge)(nil)
var _ *mcpsdk.Server = nil // ensure mcpsdk is available

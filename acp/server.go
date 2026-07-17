package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"

	acpsdk "github.com/coder/acp-go-sdk"
)

// AgentHandler receives ACP requests from a client. Implement this
// interface to expose application logic as an ACP agent.
//
// See [NewServer] and [Server.Run].
type AgentHandler interface {
	// OnInitialize is called for the ACP handshake. Return capabilities
	// and protocol version negotiation.
	OnInitialize(ctx context.Context, req InitializeRequest) (*InitializeResponse, error)

	// OnNewSession is called when the client creates a new session.
	// McpServers may be passed for the agent to connect to.
	OnNewSession(ctx context.Context, req NewSessionRequest) (*NewSessionResponse, error)

	// OnPrompt is called when the client sends a user prompt. The handler
	// processes the prompt and uses sender to stream events back to the
	// client. Return the final PromptResponse when done.
	OnPrompt(ctx context.Context, req PromptRequest, sender SessionEventSender) (*PromptResponse, error)

	// OnCancel is called when the client cancels an ongoing prompt.
	OnCancel(ctx context.Context, sessionID string) error

	// OnLoadSession is called when the client requests to resume an existing
	// session. The handler should restore the session state and use sender
	// to replay conversation history to the client.
	OnLoadSession(ctx context.Context, req LoadSessionRequest, sender SessionEventSender) (*LoadSessionResponse, error)

	// OnListSessions returns the sessions available for loading via OnLoadSession.
	OnListSessions(ctx context.Context, req ListSessionsRequest) (*ListSessionsResponse, error)

	// OnDeleteSession is called when the client deletes a session.
	// The handler should remove all session state permanently.
	OnDeleteSession(ctx context.Context, sessionID string) error

	// OnCloseSession is called when the client closes a session.
	OnCloseSession(ctx context.Context, sessionID string) error
}

// ConfigHandler is an optional extension of [AgentHandler]. Implement it
// to support session/config_set and session/mode changes.
type ConfigHandler interface {
	OnSetSessionConfigOption(ctx context.Context, sessionID string, opt SetConfigOption) error
	OnSetSessionMode(ctx context.Context, sessionID string, modeID string) error
}

// SessionAdminHandler is an optional extension of [AgentHandler]. Implement
// it to support session/fork.
type SessionAdminHandler interface {
	OnForkSession(ctx context.Context, req ForkSessionRequest) (*ForkSessionResponse, error)
}

// SessionEventSender sends streaming events from the agent back to the
// ACP client during [AgentHandler.OnPrompt].
type SessionEventSender interface {
	// SendAgentMessage sends a chunk of the agent's response text.
	SendAgentMessage(text string) error

	// SendAgentThought sends a chunk of the agent's internal reasoning.
	SendAgentThought(text string) error

	// SendToolCall notifies the client about a tool invocation.
	SendToolCall(tc ToolCallEvent) error

	// SendSessionInfo updates session metadata (title, timestamps).
	SendSessionInfo(title string, metadata map[string]any) error

	// SendUserMessage sends a chunk of the user's message (e.g. textified
	// image descriptions). Use this to echo back interpreted user input.
	SendUserMessage(text string) error

	// SendPlan sends or updates the agent's execution plan. Pass nil or
	// empty slice to clear the plan.
	SendPlan(entries []PlanEntry) error

	// SendAvailableCommands declares the commands the agent supports
	// (e.g. /help, /clear). Clients may render these as slash-commands.
	SendAvailableCommands(commands []AvailableCommand) error

	// SendCurrentMode notifies the client about the current session mode.
	SendCurrentMode(modeID string) error

	// SendUsage reports token counts and context window size.
	SendUsage(info UsageInfo) error
}

// ── Server ──

// Server exposes an [AgentHandler] as an ACP agent over a transport.
//
// Create with [NewServer], then call [Server.Run]:
//
//	handler := &myAgentHandler{}
//	server := acp.NewServer("my-agent", "1.0.0", handler)
//	if err := server.Run(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
// [Server.Run] blocks until the transport closes or ctx is cancelled.
// By default it uses os.Stdin/os.Stdout (stdio transport).
type Server struct {
	name    string
	version string

	handler AgentHandler
	logger  *slog.Logger
}

// NewServer creates an ACP [Server] with the given implementation identity.
// name and version are reported to ACP clients during initialization.
func NewServer(name, version string, handler AgentHandler) *Server {
	return &Server{
		name:    name,
		version: version,
		handler: handler,
	}
}

// SetLogger directs connection diagnostics to the provided logger.
func (s *Server) SetLogger(l *slog.Logger) { s.logger = l }

// Run starts the ACP server on stdio (os.Stdin / os.Stdout). It blocks
// until the transport closes or ctx is cancelled.
//
// For custom transports, use [Server.RunTransport].
func (s *Server) Run(ctx context.Context) error {
	// For a subprocess agent: we read ACP requests from stdin,
	// and write ACP responses/notifications to stdout.
	return s.RunTransport(ctx, os.Stdout, os.Stdin)
}

// RunTransport starts the ACP server on custom I/O streams.
//   - w: where the server writes responses (peer input, e.g. stdout pipe)
//   - r: where the server reads requests (peer output, e.g. stdin pipe)
func (s *Server) RunTransport(ctx context.Context, w io.Writer, r io.Reader) error {
	bridge := &agentBridge{
		server:  s,
		handler: s.handler,
	}

	conn := acpsdk.NewAgentSideConnection(bridge, w, r)
	bridge.conn = conn

	if s.logger != nil {
		conn.SetLogger(s.logger)
	}

	// Block until the peer disconnects.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-conn.Done():
		return nil
	}
}

// ── agentBridge ──

// agentBridge implements acpsdk.Agent to translate SDK calls to our AgentHandler.
type agentBridge struct {
	server  *Server
	handler AgentHandler
	conn    *acpsdk.AgentSideConnection // for pushing streaming events

	mu             sync.Mutex
	sessionSenders map[string]*agentSender // sessionID → sender
}

func (b *agentBridge) getSender(sessionID string) *agentSender {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.sessionSenders == nil {
		return nil
	}
	return b.sessionSenders[sessionID]
}

func (b *agentBridge) setSender(sessionID string, sender *agentSender) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.sessionSenders == nil {
		b.sessionSenders = make(map[string]*agentSender)
	}
	b.sessionSenders[sessionID] = sender
}

func (b *agentBridge) deleteSender(sessionID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.sessionSenders, sessionID)
}

// ── acpsdk.Agent implementation ──

func (b *agentBridge) Initialize(ctx context.Context, params acpsdk.InitializeRequest) (acpsdk.InitializeResponse, error) {
	resp, err := b.handler.OnInitialize(ctx, InitializeRequest{
		ProtocolVersion: int(params.ProtocolVersion),
		ClientName:      clientInfoName(params.ClientInfo),
		ClientVersion:   clientInfoVersion(params.ClientInfo),
		ClientCapabilities: ClientCapabilities{
			Terminal: params.ClientCapabilities.Terminal,
		},
	})
	if err != nil {
		return acpsdk.InitializeResponse{}, err
	}

	// Merge handler capabilities with bridge-level defaults.
	// The bridge always supports load, list, delete, resume, and close
	// because those are required AgentHandler methods. Prompt and MCP
	// capabilities come from the handler.
	caps := resp.Capabilities
	caps.LoadSession = true
	caps.SessionCapabilities.List = true
	caps.SessionCapabilities.Delete = true
	caps.SessionCapabilities.Resume = true
	caps.SessionCapabilities.Close = true

	return acpsdk.InitializeResponse{
		ProtocolVersion: acpsdk.ProtocolVersion(resp.ProtocolVersion),
		AgentCapabilities: acpsdk.AgentCapabilities{
			LoadSession: caps.LoadSession,
			McpCapabilities: acpsdk.McpCapabilities{
				Acp:  caps.McpCapabilities.Acp,
				Http: caps.McpCapabilities.Http,
				Sse:  caps.McpCapabilities.Sse,
			},
			PromptCapabilities: acpsdk.PromptCapabilities{
				Image:           caps.PromptCapabilities.Image,
				Audio:           caps.PromptCapabilities.Audio,
				EmbeddedContext: caps.PromptCapabilities.EmbeddedContext,
			},
			SessionCapabilities: acpsdk.SessionCapabilities{
				List:   sessionListCapPtr(caps.SessionCapabilities.List),
				Delete: sessionDeleteCapPtr(caps.SessionCapabilities.Delete),
				Resume: sessionResumeCapPtr(caps.SessionCapabilities.Resume),
				Close:  sessionCloseCapPtr(caps.SessionCapabilities.Close),
			},
		},
		AgentInfo: &acpsdk.Implementation{
			Name: b.server.name, Version: b.server.version,
		},
	}, nil
}

func (b *agentBridge) NewSession(ctx context.Context, params acpsdk.NewSessionRequest) (acpsdk.NewSessionResponse, error) {
	mcpServers := make([]McpServer, 0, len(params.McpServers))
	for _, m := range params.McpServers {
		mcpServers = append(mcpServers, fromSDKMcpServer(m))
	}

	resp, err := b.handler.OnNewSession(ctx, NewSessionRequest{
		Cwd:                   params.Cwd,
		McpServers:            mcpServers,
		AdditionalDirectories: params.AdditionalDirectories,
	})
	if err != nil {
		return acpsdk.NewSessionResponse{}, err
	}

	return acpsdk.NewSessionResponse{
		SessionId:     acpsdk.SessionId(resp.SessionID),
		ConfigOptions: convertConfigOptions(resp.ConfigOptions),
		Modes:         convertModeState(resp.Modes),
	}, nil
}

func (b *agentBridge) Prompt(ctx context.Context, params acpsdk.PromptRequest) (acpsdk.PromptResponse, error) {
	sessionID := string(params.SessionId)

	sender := &agentSender{
		sessionID: sessionID,
		bridge:    b,
		conn:      b.conn,
	}

	b.setSender(sessionID, sender)
	defer b.deleteSender(sessionID)

	blocks := make([]ContentBlock, len(params.Prompt))
	for i, pb := range params.Prompt {
		if pb.Text != nil {
			blocks[i] = ContentBlock{Text: pb.Text.Text}
		}
	}

	resp, err := b.handler.OnPrompt(ctx, PromptRequest{
		SessionID: sessionID,
		Blocks:    blocks,
	}, sender)
	if err != nil {
		return acpsdk.PromptResponse{}, err
	}

	return acpsdk.PromptResponse{
		StopReason: acpsdk.StopReason(resp.StopReason),
	}, nil
}

func (b *agentBridge) Cancel(ctx context.Context, params acpsdk.CancelNotification) error {
	return b.handler.OnCancel(ctx, string(params.SessionId))
}

func (b *agentBridge) CloseSession(ctx context.Context, params acpsdk.CloseSessionRequest) (acpsdk.CloseSessionResponse, error) {
	sessionID := string(params.SessionId)
	b.deleteSender(sessionID)
	if err := b.handler.OnCloseSession(ctx, sessionID); err != nil {
		return acpsdk.CloseSessionResponse{}, err
	}
	return acpsdk.CloseSessionResponse{}, nil
}

// Remaining Agent methods — return not supported for now.
func (b *agentBridge) Authenticate(ctx context.Context, params acpsdk.AuthenticateRequest) (acpsdk.AuthenticateResponse, error) {
	return acpsdk.AuthenticateResponse{}, fmt.Errorf("not supported")
}
func (b *agentBridge) Logout(ctx context.Context, params acpsdk.LogoutRequest) (acpsdk.LogoutResponse, error) {
	return acpsdk.LogoutResponse{}, nil
}
func (b *agentBridge) ListSessions(ctx context.Context, params acpsdk.ListSessionsRequest) (acpsdk.ListSessionsResponse, error) {
	resp, err := b.handler.OnListSessions(ctx, ListSessionsRequest{
		Cursor: params.Cursor,
		Cwd:    params.Cwd,
	})
	if err != nil {
		return acpsdk.ListSessionsResponse{}, err
	}

	sessions := make([]acpsdk.SessionInfo, 0, len(resp.Sessions))
	for _, s := range resp.Sessions {
		info := acpsdk.SessionInfo{
			SessionId: acpsdk.SessionId(s.SessionID),
			Cwd:       s.Cwd,
		}
		if s.Title != "" {
			info.Title = &s.Title
		}
		if s.UpdatedAt != "" {
			info.UpdatedAt = &s.UpdatedAt
		}
		sessions = append(sessions, info)
	}

	return acpsdk.ListSessionsResponse{
		Sessions:   sessions,
		NextCursor: resp.NextCursor,
	}, nil
}
func (b *agentBridge) ResumeSession(ctx context.Context, params acpsdk.ResumeSessionRequest) (acpsdk.ResumeSessionResponse, error) {
	mcpServers := make([]McpServer, 0, len(params.McpServers))
	for _, m := range params.McpServers {
		mcpServers = append(mcpServers, fromSDKMcpServer(m))
	}

	sessionID := string(params.SessionId)
	sender := &agentSender{sessionID: sessionID, bridge: b, conn: b.conn}
	_, err := b.handler.OnLoadSession(ctx, LoadSessionRequest{
		SessionID:            sessionID,
		Cwd:                  params.Cwd,
		McpServers:           mcpServers,
		AdditionalDirectories: params.AdditionalDirectories,
	}, sender)
	if err != nil {
		return acpsdk.ResumeSessionResponse{}, err
	}

	return acpsdk.ResumeSessionResponse{}, nil
}

// LoadSession implements acpsdk.AgentLoader so clients can restore a
// previously saved session via session/load.
func (b *agentBridge) LoadSession(ctx context.Context, params acpsdk.LoadSessionRequest) (acpsdk.LoadSessionResponse, error) {
	mcpServers := make([]McpServer, 0, len(params.McpServers))
	for _, m := range params.McpServers {
		mcpServers = append(mcpServers, fromSDKMcpServer(m))
	}

	sessionID := string(params.SessionId)
	sender := &agentSender{sessionID: sessionID, bridge: b, conn: b.conn}
	_, err := b.handler.OnLoadSession(ctx, LoadSessionRequest{
		SessionID:            sessionID,
		Cwd:                  params.Cwd,
		McpServers:           mcpServers,
		AdditionalDirectories: params.AdditionalDirectories,
	}, sender)
	if err != nil {
		return acpsdk.LoadSessionResponse{}, err
	}

	return acpsdk.LoadSessionResponse{}, nil
}

func (b *agentBridge) SetSessionConfigOption(ctx context.Context, params acpsdk.SetSessionConfigOptionRequest) (acpsdk.SetSessionConfigOptionResponse, error) {
	ch, ok := b.handler.(ConfigHandler)
	if !ok {
		return acpsdk.SetSessionConfigOptionResponse{}, nil
	}

	var sessionID string
	var opt SetConfigOption

	if params.Boolean != nil {
		sessionID = string(params.Boolean.SessionId)
		opt.ID = string(params.Boolean.ConfigId)
		opt.Value = params.Boolean.Value
	} else if params.ValueId != nil {
		sessionID = string(params.ValueId.SessionId)
		opt.ID = string(params.ValueId.ConfigId)
		opt.Value = string(params.ValueId.Value)
	}

	if err := ch.OnSetSessionConfigOption(ctx, sessionID, opt); err != nil {
		return acpsdk.SetSessionConfigOptionResponse{}, err
	}
	return acpsdk.SetSessionConfigOptionResponse{}, nil
}
func (b *agentBridge) SetSessionMode(ctx context.Context, params acpsdk.SetSessionModeRequest) (acpsdk.SetSessionModeResponse, error) {
	ch, ok := b.handler.(ConfigHandler)
	if !ok {
		return acpsdk.SetSessionModeResponse{}, nil
	}
	if err := ch.OnSetSessionMode(ctx, string(params.SessionId), string(params.ModeId)); err != nil {
		return acpsdk.SetSessionModeResponse{}, err
	}
	return acpsdk.SetSessionModeResponse{}, nil
}

// SessionDelete implements the SDK's optional session/delete interface.
// This is a protocol-defined method that the SDK marks as unstable.
func (b *agentBridge) UnstableDeleteSession(ctx context.Context, params acpsdk.UnstableDeleteSessionRequest) (acpsdk.UnstableDeleteSessionResponse, error) {
	sessionID := string(params.SessionId)
	b.deleteSender(sessionID)
	if err := b.handler.OnDeleteSession(ctx, sessionID); err != nil {
		return acpsdk.UnstableDeleteSessionResponse{}, err
	}
	return acpsdk.UnstableDeleteSessionResponse{}, nil
}

// UnstableForkSession implements the SDK's optional session/fork interface.
func (b *agentBridge) UnstableForkSession(ctx context.Context, params acpsdk.UnstableForkSessionRequest) (acpsdk.UnstableForkSessionResponse, error) {
	sh, ok := b.handler.(SessionAdminHandler)
	if !ok {
		return acpsdk.UnstableForkSessionResponse{}, fmt.Errorf("not supported")
	}
	resp, err := sh.OnForkSession(ctx, ForkSessionRequest{
		SessionID: string(params.SessionId),
	})
	if err != nil {
		return acpsdk.UnstableForkSessionResponse{}, err
	}
	return acpsdk.UnstableForkSessionResponse{
		SessionId: acpsdk.SessionId(resp.SessionID),
	}, nil
}

// ── agentSender ──

// agentSender implements SessionEventSender. It pushes streaming events
// to the ACP client via AgentSideConnection.SessionUpdate.
type agentSender struct {
	sessionID string
	bridge    *agentBridge
	conn      *acpsdk.AgentSideConnection
}

func (s *agentSender) SendAgentMessage(text string) error {
	if s.conn == nil {
		return nil
	}
	content := acpsdk.ContentBlock{Text: &acpsdk.ContentBlockText{Text: text}}
	return s.conn.SessionUpdate(context.Background(), acpsdk.SessionNotification{
		Update: acpsdk.UpdateAgentMessage(content),
	})
}

func (s *agentSender) SendAgentThought(text string) error {
	if s.conn == nil {
		return nil
	}
	content := acpsdk.ContentBlock{Text: &acpsdk.ContentBlockText{Text: text}}
	return s.conn.SessionUpdate(context.Background(), acpsdk.SessionNotification{
		Update: acpsdk.UpdateAgentThought(content),
	})
}

func (s *agentSender) SendToolCall(tc ToolCallEvent) error {
	if s.conn == nil {
		return nil
	}
	var update acpsdk.SessionUpdate
	switch tc.Status {
	case "completed":
		update = acpsdk.UpdateToolCall(acpsdk.ToolCallId(tc.ID),
			acpsdk.WithUpdateStatus(acpsdk.ToolCallStatusCompleted),
			acpsdk.WithUpdateRawOutput(tc.RawOutput))
	case "failed":
		update = acpsdk.UpdateToolCall(acpsdk.ToolCallId(tc.ID),
			acpsdk.WithUpdateStatus(acpsdk.ToolCallStatusFailed),
			acpsdk.WithUpdateRawOutput(tc.RawOutput))
	default:
		update = acpsdk.StartToolCall(acpsdk.ToolCallId(tc.ID), tc.Title,
			acpsdk.WithStartStatus(acpsdk.ToolCallStatusInProgress),
			acpsdk.WithStartRawInput(tc.RawInput))
	}
	return s.conn.SessionUpdate(context.Background(), acpsdk.SessionNotification{
		Update: update,
	})
}

func (s *agentSender) SendSessionInfo(title string, metadata map[string]any) error {
	if s.conn == nil {
		return nil
	}
	var titlePtr *string
	if title != "" {
		titlePtr = &title
	}
	return s.conn.SessionUpdate(context.Background(), acpsdk.SessionNotification{
		SessionId: acpsdk.SessionId(s.sessionID),
		Update: acpsdk.SessionUpdate{
			SessionInfoUpdate: &acpsdk.SessionSessionInfoUpdate{
				SessionUpdate: "sessionInfoUpdate",
				Title:         titlePtr,
				Meta:          metadata,
			},
		},
	})
}

func (s *agentSender) SendUserMessage(text string) error {
	if s.conn == nil {
		return nil
	}
	return s.conn.SessionUpdate(context.Background(), acpsdk.SessionNotification{
		SessionId: acpsdk.SessionId(s.sessionID),
		Update:    acpsdk.UpdateUserMessageText(text),
	})
}

func (s *agentSender) SendPlan(entries []PlanEntry) error {
	if s.conn == nil {
		return nil
	}
	sdkEntries := make([]acpsdk.PlanEntry, len(entries))
	for i, e := range entries {
		sdkEntries[i] = acpsdk.PlanEntry{
			Content:  e.Title,
			Priority: stringToPlanPriority(e.Priority),
			Status:   stringToPlanStatus(e.Status),
		}
	}
	return s.conn.SessionUpdate(context.Background(), acpsdk.SessionNotification{
		SessionId: acpsdk.SessionId(s.sessionID),
		Update:    acpsdk.UpdatePlan(sdkEntries...),
	})
}

func (s *agentSender) SendAvailableCommands(commands []AvailableCommand) error {
	if s.conn == nil {
		return nil
	}
	sdkCmds := make([]acpsdk.AvailableCommand, len(commands))
	for i, c := range commands {
		sdkCmds[i] = acpsdk.AvailableCommand{
			Name:        c.Name,
			Description: c.Description,
		}
	}
	return s.conn.SessionUpdate(context.Background(), acpsdk.SessionNotification{
		SessionId: acpsdk.SessionId(s.sessionID),
		Update: acpsdk.SessionUpdate{
			AvailableCommandsUpdate: &acpsdk.SessionAvailableCommandsUpdate{
				AvailableCommands: sdkCmds,
				SessionUpdate:     "availableCommandsUpdate",
			},
		},
	})
}

func (s *agentSender) SendCurrentMode(modeID string) error {
	if s.conn == nil {
		return nil
	}
	return s.conn.SessionUpdate(context.Background(), acpsdk.SessionNotification{
		SessionId: acpsdk.SessionId(s.sessionID),
		Update: acpsdk.SessionUpdate{
			CurrentModeUpdate: &acpsdk.SessionCurrentModeUpdate{
				CurrentModeId: acpsdk.SessionModeId(modeID),
				SessionUpdate: "currentModeUpdate",
			},
		},
	})
}

func (s *agentSender) SendUsage(info UsageInfo) error {
	if s.conn == nil {
		return nil
	}
	return s.conn.SessionUpdate(context.Background(), acpsdk.SessionNotification{
		SessionId: acpsdk.SessionId(s.sessionID),
		Update: acpsdk.SessionUpdate{
			UsageUpdate: &acpsdk.SessionUsageUpdate{
				Used:          info.Used,
				Size:          info.Size,
				SessionUpdate: "usageUpdate",
			},
		},
	})
}

// ── Helpers ──

func sessionListCapPtr(ok bool) *acpsdk.SessionListCapabilities {
	if !ok { return nil }
	return &acpsdk.SessionListCapabilities{}
}
func sessionResumeCapPtr(ok bool) *acpsdk.SessionResumeCapabilities {
	if !ok { return nil }
	return &acpsdk.SessionResumeCapabilities{}
}
func sessionDeleteCapPtr(ok bool) *acpsdk.SessionDeleteCapabilities {
	if !ok { return nil }
	return &acpsdk.SessionDeleteCapabilities{}
}
func sessionCloseCapPtr(ok bool) *acpsdk.SessionCloseCapabilities {
	if !ok { return nil }
	return &acpsdk.SessionCloseCapabilities{}
}

// convertConfigOptions converts our ConfigOptions to the SDK type via JSON round-trip.
func convertConfigOptions(opts []SessionConfigOption) []acpsdk.SessionConfigOption {
	if len(opts) == 0 {
		return nil
	}
	data, err := json.Marshal(opts)
	if err != nil {
		return nil
	}
	var sdk []acpsdk.SessionConfigOption
	json.Unmarshal(data, &sdk)
	return sdk
}

// convertModeState converts our SessionModeState to the SDK type via JSON round-trip.
func convertModeState(m *SessionModeState) *acpsdk.SessionModeState {
	if m == nil {
		return nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	var sdk acpsdk.SessionModeState
	json.Unmarshal(data, &sdk)
	return &sdk
}

func fromSDKMcpServer(m acpsdk.McpServer) McpServer {
	ms := McpServer{}
	if m.Http != nil {
		ms.URL = m.Http.Url
	}
	if m.Sse != nil {
		ms.URL = m.Sse.Url
	}
	if m.Stdio != nil {
		ms.Name = m.Stdio.Name
		ms.Command = m.Stdio.Command
		ms.Args = m.Stdio.Args
	}
	return ms
}

func clientInfoName(info *acpsdk.Implementation) string {
	if info == nil {
		return ""
	}
	return info.Name
}

func clientInfoVersion(info *acpsdk.Implementation) string {
	if info == nil {
		return ""
	}
	return info.Version
}

// Compile-time interface checks.
var _ acpsdk.Agent = (*agentBridge)(nil)
var _ acpsdk.AgentLoader = (*agentBridge)(nil)

func stringToPlanPriority(p string) acpsdk.PlanEntryPriority {
	switch p {
	case "high":
		return acpsdk.PlanEntryPriorityHigh
	case "medium":
		return acpsdk.PlanEntryPriorityMedium
	case "low":
		return acpsdk.PlanEntryPriorityLow
	default:
		return acpsdk.PlanEntryPriorityMedium
	}
}

func stringToPlanStatus(s string) acpsdk.PlanEntryStatus {
	switch s {
	case "pending":
		return acpsdk.PlanEntryStatusPending
	case "in_progress":
		return acpsdk.PlanEntryStatusInProgress
	case "completed":
		return acpsdk.PlanEntryStatusCompleted
	default:
		return acpsdk.PlanEntryStatusPending
	}
}

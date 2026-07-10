package acp

import (
	"context"
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

	// OnCloseSession is called when the client closes a session.
	OnCloseSession(ctx context.Context, sessionID string) error
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
	})
	if err != nil {
		return acpsdk.InitializeResponse{}, err
	}

	return acpsdk.InitializeResponse{
		ProtocolVersion: acpsdk.ProtocolVersion(resp.ProtocolVersion),
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
		Cwd:        params.Cwd,
		McpServers: mcpServers,
	})
	if err != nil {
		return acpsdk.NewSessionResponse{}, err
	}

	return acpsdk.NewSessionResponse{
		SessionId: acpsdk.SessionId(resp.SessionID),
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
	return acpsdk.ListSessionsResponse{Sessions: []acpsdk.SessionInfo{}}, nil
}
func (b *agentBridge) ResumeSession(ctx context.Context, params acpsdk.ResumeSessionRequest) (acpsdk.ResumeSessionResponse, error) {
	return acpsdk.ResumeSessionResponse{}, fmt.Errorf("not supported")
}
func (b *agentBridge) SetSessionConfigOption(ctx context.Context, params acpsdk.SetSessionConfigOptionRequest) (acpsdk.SetSessionConfigOptionResponse, error) {
	return acpsdk.SetSessionConfigOptionResponse{}, nil
}
func (b *agentBridge) SetSessionMode(ctx context.Context, params acpsdk.SetSessionModeRequest) (acpsdk.SetSessionModeResponse, error) {
	return acpsdk.SetSessionModeResponse{}, nil
}

// ── agentSender ──

// agentSender implements SessionEventSender. It pushes streaming events
// to the ACP client via AgentSideConnection.SessionUpdate.
// Every SessionNotification includes SessionId so the client can
// associate updates with the correct session.
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
		SessionId: acpsdk.SessionId(s.sessionID),
		Update:    acpsdk.UpdateAgentMessage(content),
	})
}

func (s *agentSender) SendAgentThought(text string) error {
	if s.conn == nil {
		return nil
	}
	content := acpsdk.ContentBlock{Text: &acpsdk.ContentBlockText{Text: text}}
	return s.conn.SessionUpdate(context.Background(), acpsdk.SessionNotification{
		SessionId: acpsdk.SessionId(s.sessionID),
		Update:    acpsdk.UpdateAgentThought(content),
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
			acpsdk.WithUpdateStatus(acpsdk.ToolCallStatusCompleted))
	case "failed":
		update = acpsdk.UpdateToolCall(acpsdk.ToolCallId(tc.ID),
			acpsdk.WithUpdateStatus(acpsdk.ToolCallStatusFailed))
	default:
		update = acpsdk.StartToolCall(acpsdk.ToolCallId(tc.ID), tc.Title)
	}
	return s.conn.SessionUpdate(context.Background(), acpsdk.SessionNotification{
		SessionId: acpsdk.SessionId(s.sessionID),
		Update:    update,
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

// ── Helpers ──

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

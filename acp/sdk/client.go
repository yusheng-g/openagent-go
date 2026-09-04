// ── Client ──
//
// Client spawns an external ACP agent process and communicates with it over
// stdin/stdout using JSON-RPC 2.0. Methods map 1:1 to the ACP v1 protocol:
//
//	https://agentclientprotocol.com/protocol/v1/schema
//
// Usage:
//
//	client := acp.NewClient("my-app", "1.0.0")
//	sess, _ := client.ConnectStdio(ctx, nil, "my-agent", "-v")
//	defer sess.Close()
//
//	sess.Initialize(ctx, acp.InitializeRequest{...})
//	sess.NewSession(ctx, acp.NewSessionRequest{...})
//	sess.SetEventHandler(handler)         // receive streaming notifications
//	sess.Prompt(ctx, acp.PromptRequest{...})
//
// The EventHandler receives session/update notifications (agent messages,
// tool calls, plan updates, config changes, etc.) from the agent in real
// time as they arrive on stdout. Notifications are dispatched asynchronously
// by a background reader goroutine started automatically by ConnectStdio.
//
// See [Session] for the full method set and [EventHandler] for notification
// callbacks.

package acp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// Client connects to external ACP agent processes over stdio.
type Client struct {
	name    string
	version string
}

// NewClient creates an ACP [Client] with the given implementation identity.
func NewClient(name, version string) *Client {
	return &Client{name: name, version: version}
}

// ConnectStdio spawns an ACP agent subprocess and communicates over
// stdin/stdout with newline-delimited JSON (JSON-RPC 2.0).
//
// env adds environment variables (name=value pairs) to the spawned process on
// top of the parent environment. Pass nil to inherit the parent env unchanged.
// command is the path to the executable; args are its CLI arguments.
// The background reader goroutine starts automatically and dispatches
// session/update notifications to the [EventHandler].
//
// Call [Session.Close] to kill the process and release resources.
func (c *Client) ConnectStdio(ctx context.Context, env []string, command string, args ...string) (*Session, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	if len(env) > 0 {
		cmd.Env = append(cmd.Environ(), env...)
	}
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

	return c.connectIO(ctx, stdin, stdout, subprocessCloser(cmd), &stderrBuf), nil
}

// ConnectIO wires a [Session] to arbitrary io streams and starts the reader
// goroutine. Use this for in-process connections (e.g. io.Pipe to a
// [Server.RunTransport] in the same process) instead of spawning a subprocess.
//
// stdin is where the client writes JSON-RPC requests (server reads from it);
// stdout is where the server writes responses (client reads from it).
func (c *Client) ConnectIO(ctx context.Context, stdin io.Writer, stdout io.Reader) *Session {
	return c.connectIO(ctx, stdin, stdout, nil, nil)
}

// connectIO wires a [Session] to arbitrary io streams and starts the reader
// goroutine. closer (optional) is invoked by [Session.Close] to tear the
// transport down; stderr (optional) is surfaced via [Session.Stderr].
//
// This is the seam that separates process-spawning ([ConnectStdio]) from the
// JSON-RPC transport, so a Session can also be driven in-process (e.g. over
// io.Pipe for tests or an already-spawned process's pipes).
func (c *Client) connectIO(ctx context.Context, stdin io.Writer, stdout io.Reader, closer io.Closer, stderr *bytes.Buffer) *Session {
	sess := &Session{
		ctx:       ctx,
		stdin:     stdin,
		stdout:    bufio.NewScanner(stdout),
		closer:    closer,
		stderrBuf: stderr,
		writeMu:   new(sync.Mutex),
		pending:   make(map[string]*pendingCall),
	}
	go sess.startReader(ctx)
	return sess
}

// subprocessCloser returns an io.Closer that kills and reaps cmd. Best-effort,
// mirroring the pre-refactor Close() behaviour.
func subprocessCloser(cmd *exec.Cmd) io.Closer {
	return closerFunc(func() error {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
		return nil
	})
}

// closerFunc lets a plain func satisfy io.Closer.
type closerFunc func() error

func (f closerFunc) Close() error { return f() }

// ── Session ──

// Session is an active connection to an external ACP agent. Public methods
// map 1:1 to ACP v1 protocol methods — each sends a JSON-RPC 2.0 request
// and blocks until the response arrives.
//
// Streaming notifications from the agent (agent messages, tool calls, plans,
// config changes, etc.) are delivered to the [EventHandler] set via
// [Session.SetEventHandler]. The handler must be set before [Session.Prompt]
// to receive streaming output.
type Session struct {
	// ctx is the connection lifecycle context (from ConnectStdio); inbound
	// agent->client RPC handlers run under it so they cancel on disconnect.
	ctx    context.Context
	stdin  io.Writer
	stdout *bufio.Scanner

	// closer optionally tears down the transport on Close. For a spawned
	// subprocess it kills and reaps the process; for an in-process io
	// connection it may be nil (caller owns lifecycle) or a pipe closer.
	closer io.Closer

	// stderrBuf captures the subprocess's stderr for diagnostics. Nil for
	// in-process connections that have no subprocess.
	stderrBuf *bytes.Buffer

	writeMu *sync.Mutex

	mu         sync.Mutex
	sessionID  string
	nextID     int64
	pending    map[string]*pendingCall
	eh         EventHandler
	clientReqH ClientRequestHandler
}

type pendingCall struct {
	resp rpcResponse
	done chan struct{}
}

// ── ACP Protocol Methods ──
//
// Each method sends the corresponding ACP v1 JSON-RPC request and blocks
// until the agent's response arrives. Protocols docs:
//
//	https://agentclientprotocol.com/protocol/v1/schema

// Initialize performs the ACP handshake: initialize.
// Must be called once after ConnectStdio before any session methods.
func (s *Session) Initialize(ctx context.Context, req InitializeRequest) (*InitializeResponse, error) {
	var resp InitializeResponse
	if err := s.call(ctx, "initialize", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// NewSession creates a new session: session/new.
// The returned session ID is stored internally and reused for subsequent
// [Session.Prompt], [Session.LoadSession], [Session.ResumeSession], and
// [Session.CloseSession] calls.
func (s *Session) NewSession(ctx context.Context, req NewSessionRequest) (*NewSessionResponse, error) {
	var resp NewSessionResponse
	if err := s.call(ctx, "session/new", req, &resp); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.sessionID = resp.SessionID
	s.mu.Unlock()
	return &resp, nil
}

// SetEventHandler registers the callback that receives session/update
// notifications from the agent. Must be set before [Session.Prompt] to
// capture streaming output.
func (s *Session) SetEventHandler(h EventHandler) {
	s.mu.Lock()
	s.eh = h
	s.mu.Unlock()
}

// SetClientRequestHandler registers the handler that responds to Agent→Client
// RPC requests (request_permission, fs/read_text_file, terminal/*, etc.).
// If not set, unrecognised agent→client requests return MethodNotFound.
func (s *Session) SetClientRequestHandler(h ClientRequestHandler) {
	s.mu.Lock()
	s.clientReqH = h
	s.mu.Unlock()
}

// Prompt sends a user prompt and blocks until the turn completes:
// session/prompt.
//
// Streaming output (agent messages, tool calls, plan updates, etc.) is
// delivered to the registered [EventHandler] concurrently via the
// background reader goroutine.
func (s *Session) Prompt(ctx context.Context, req PromptRequest) (*PromptResponse, error) {
	s.mu.Lock()
	req.SessionID = s.sessionID
	s.mu.Unlock()
	var resp PromptResponse
	if err := s.call(ctx, "session/prompt", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// LoadSession loads an existing session with history replay: session/load.
// The agent replays conversation history via session/update notifications
// before responding.
func (s *Session) LoadSession(ctx context.Context, req LoadSessionRequest) (*LoadSessionResponse, error) {
	var resp LoadSessionResponse
	if err := s.call(ctx, "session/load", req, &resp); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.sessionID = req.SessionID
	s.mu.Unlock()
	return &resp, nil
}

// ResumeSession resumes an existing session without history replay:
// session/resume.
func (s *Session) ResumeSession(ctx context.Context, req ResumeSessionRequest) (*ResumeSessionResponse, error) {
	var resp ResumeSessionResponse
	if err := s.call(ctx, "session/resume", req, &resp); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.sessionID = req.SessionID
	s.mu.Unlock()
	return &resp, nil
}

// CloseSession closes the active session: session/close.
func (s *Session) CloseSession(ctx context.Context) error {
	s.mu.Lock()
	sid := s.sessionID
	s.mu.Unlock()
	if sid == "" {
		return nil
	}
	_, err := s.request(ctx, "session/close", CloseSessionRequest{SessionID: sid})
	return err
}

// DeleteSession permanently removes a session and all its data:
// session/delete.
func (s *Session) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := s.request(ctx, "session/delete", DeleteSessionRequest{SessionID: sessionID})
	return err
}

// ListSessions returns the sessions available on the agent: session/list.
func (s *Session) ListSessions(ctx context.Context, req ListSessionsRequest) (*ListSessionsResponse, error) {
	var resp ListSessionsResponse
	if err := s.call(ctx, "session/list", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListMessages returns the most recent messages of a session without
// loading it: session/list_messages.
func (s *Session) ListMessages(ctx context.Context, req ListMessagesRequest) (*ListMessagesResponse, error) {
	var resp ListMessagesResponse
	if err := s.call(ctx, "session/list_messages", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListConfigOptions returns the config options a fresh session would
// receive (mode, thought level, model selector), without creating one:
// session/list_config_options.
func (s *Session) ListConfigOptions(ctx context.Context, req ListConfigOptionsRequest) (*ListConfigOptionsResponse, error) {
	var resp ListConfigOptionsResponse
	if err := s.call(ctx, "session/list_config_options", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetSessionMode changes the active session mode: session/set_mode.
func (s *Session) SetSessionMode(ctx context.Context, req SetSessionModeRequest) (*SetSessionModeResponse, error) {
	var resp SetSessionModeResponse
	if err := s.call(ctx, "session/set_mode", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetSessionConfigOption changes a config option: session/set_config_option.
// The agent's response contains the full configOptions array (not a delta).
func (s *Session) SetSessionConfigOption(ctx context.Context, req SetSessionConfigOptionRequest) (*SetSessionConfigOptionResponse, error) {
	var resp SetSessionConfigOptionResponse
	if err := s.call(ctx, "session/set_config_option", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Cancel sends a session/cancel notification to cancel an ongoing prompt.
// This is a notification (no response expected).
func (s *Session) Cancel(ctx context.Context, sessionID string) error {
	return s.notify(ctx, "session/cancel", CancelNotification{SessionID: sessionID})
}

// Logout ends the current authenticated state. Must only be called when the
// Agent advertised auth.logout capability.
func (s *Session) Logout(ctx context.Context) (*LogoutResponse, error) {
	var resp LogoutResponse
	if err := s.call(ctx, "logout", LogoutRequest{}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Close tears down the connection. It closes the write side of stdin (to
// signal EOF to the peer) and, if a closer was provided at connect time,
// invokes it (e.g. killing and reaping a spawned subprocess). Call when the
// client is done with the connection.
func (s *Session) Close() error {
	if s.stdin != nil {
		if wc, ok := s.stdin.(io.WriteCloser); ok {
			_ = wc.Close()
		}
	}
	if s.closer != nil {
		_ = s.closer.Close()
	}
	return nil
}

// Stderr returns the captured stderr output of the agent subprocess.
func (s *Session) Stderr() string {
	if s.stderrBuf == nil {
		return ""
	}
	return s.stderrBuf.String()
}

// ── JSON-RPC 2.0 transport ──

// nextReqID returns an auto-incrementing request id.
func (s *Session) nextReqID() string {
	s.mu.Lock()
	s.nextID++
	id := s.nextID
	s.mu.Unlock()
	return fmt.Sprintf("%d", id)
}

// call sends a JSON-RPC 2.0 request and unmarshals the result.
func (s *Session) call(ctx context.Context, method string, params, result any) error {
	resp, err := s.request(ctx, method, params)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("acp: %s: %s", method, resp.Error.Message)
	}
	return json.Unmarshal(resp.Result, result)
}

// request sends a JSON-RPC 2.0 request and returns the full response.
// It registers a pending call, writes the request to stdin, and blocks
// until the reader goroutine signals completion or ctx is cancelled.
func (s *Session) request(ctx context.Context, method string, params any) (rpcResponse, error) {
	id := s.nextReqID()
	req := struct {
		JSONRPC string `json:"jsonrpc"`
		ID      string `json:"id"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}{JSONRPC: "2.0", ID: id, Method: method, Params: params}

	reqBody, _ := json.Marshal(req)
	call := &pendingCall{done: make(chan struct{})}

	s.mu.Lock()
	s.pending[id] = call
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
	}()

	s.writeMu.Lock()
	_, err := s.stdin.Write(append(reqBody, '\n'))
	s.writeMu.Unlock()
	if err != nil {
		return rpcResponse{}, fmt.Errorf("acp write %s: %w", method, err)
	}

	select {
	case <-call.done:
	case <-ctx.Done():
		return rpcResponse{}, ctx.Err()
	}
	return call.resp, nil
}

// notify sends a JSON-RPC 2.0 notification (no id field, no response).
func (s *Session) notify(ctx context.Context, method string, params any) error {
	notif := struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}{JSONRPC: "2.0", Method: method, Params: params}
	body, _ := json.Marshal(notif)
	s.writeMu.Lock()
	_, err := s.stdin.Write(append(body, '\n'))
	s.writeMu.Unlock()
	return err
}

// ── EventHandler ──

// EventHandler receives session/update notifications from the agent.
// Each callback maps to a sessionUpdate discriminator value:
//
//	https://agentclientprotocol.com/protocol/v1/schema#session%2Fupdate
type EventHandler interface {
	// OnAgentMessage — sessionUpdate "agent_message_chunk".
	OnAgentMessage(text string)
	// OnAgentThought — sessionUpdate "agent_thought_chunk".
	OnAgentThought(text string)
	// OnUserMessage — sessionUpdate "user_message_chunk" (during session/load history replay).
	OnUserMessage(text string)
	// OnToolCall — sessionUpdate "tool_call" / "tool_call_update".
	OnToolCall(tc ToolCallUpdate)
	// OnPlan — sessionUpdate "plan".
	OnPlan(plan Plan)
	// OnAvailableCommandsUpdate — sessionUpdate "available_commands_update".
	OnAvailableCommandsUpdate(cmds []AvailableCommand)
	// OnModeUpdate — sessionUpdate "current_mode_update".
	OnModeUpdate(modeID SessionModeId)
	// OnConfigOptionUpdate — sessionUpdate "config_option_update".
	OnConfigOptionUpdate(opts []SessionConfigOption)
	// OnUsageUpdate — sessionUpdate "usage_update".
	OnUsageUpdate(used, total int, cost *Cost)
	// OnSessionInfo — sessionUpdate "session_info_update".
	OnSessionInfo(title string, metadata map[string]any)
}

// ── Reader goroutine ──

// startReader reads JSON-RPC messages from the agent's stdout. Three
// message shapes per JSON-RPC 2.0:
//
//	Request:      method + id present → agent→client RPC. Handle synchronously.
//	Notification: method present, no id → dispatch to EventHandler.
//	Response:     no method, id present → deliver to pending call.
func (s *Session) startReader(ctx context.Context) {
	s.stdout.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for s.stdout.Scan() {
		line := s.stdout.Bytes()
		if len(line) == 0 {
			continue
		}

		var env struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id,omitempty"`
			Method  string          `json:"method,omitempty"`
			Params  json.RawMessage `json:"params,omitempty"`
			Result  json.RawMessage `json:"result,omitempty"`
			Error   *Error          `json:"error,omitempty"`
		}
		if err := json.Unmarshal(line, &env); err != nil {
			continue
		}

		switch {
		case env.Method != "" && len(env.ID) > 0:
			// Agent→Client RPC request. Handle synchronously and write
			// the response back to the agent on stdin.
			s.handleAgentRequest(ctx, env.Method, env.ID, env.Params)

		case env.Method != "":
			// Notification — dispatch to EventHandler.
			s.dispatchNotif(env.Method, env.Params)

		case len(env.ID) > 0:
			// Response — deliver to pending call.
			callKey := idString(env.ID)
			s.mu.Lock()
			call := s.pending[callKey]
			s.mu.Unlock()
			if call != nil {
				if env.Error != nil {
					call.resp.Error = env.Error
				} else {
					call.resp.Result = env.Result
				}
				close(call.done)
			}
		}
	}
}

// handleAgentRequest dispatches an incoming Agent→Client RPC request
// and writes the JSON-RPC 2.0 response back to the agent on stdin.
func (s *Session) handleAgentRequest(ctx context.Context, method string, id json.RawMessage, params json.RawMessage) {
	s.mu.Lock()
	h := s.clientReqH
	s.mu.Unlock()

	if h == nil {
		s.writeRPCResponse(id, nil, fmt.Errorf("agent→client RPC not configured"))
		return
	}

	var resp any
	var err error

	switch method {
	case "session/request_permission":
		var req RequestPermissionRequest
		if err := json.Unmarshal(params, &req); err != nil {
			s.writeRPCResponse(id, nil, fmt.Errorf("invalid params for session/request_permission: %w", err))
			return
		}
		resp, err = h.HandleRequestPermission(ctx, req)
	case "fs/read_text_file":
		var req ReadTextFileRequest
		if err := json.Unmarshal(params, &req); err != nil {
			s.writeRPCResponse(id, nil, fmt.Errorf("invalid params for fs/read_text_file: %w", err))
			return
		}
		resp, err = h.HandleReadTextFile(ctx, req)
	case "fs/write_text_file":
		var req WriteTextFileRequest
		if err := json.Unmarshal(params, &req); err != nil {
			s.writeRPCResponse(id, nil, fmt.Errorf("invalid params for fs/write_text_file: %w", err))
			return
		}
		resp, err = h.HandleWriteTextFile(ctx, req)
	case "terminal/create":
		var req CreateTerminalRequest
		if err := json.Unmarshal(params, &req); err != nil {
			s.writeRPCResponse(id, nil, fmt.Errorf("invalid params for terminal/create: %w", err))
			return
		}
		resp, err = h.HandleCreateTerminal(ctx, req)
	case "terminal/output":
		var req TerminalOutputRequest
		if err := json.Unmarshal(params, &req); err != nil {
			s.writeRPCResponse(id, nil, fmt.Errorf("invalid params for terminal/output: %w", err))
			return
		}
		resp, err = h.HandleTerminalOutput(ctx, req)
	case "terminal/wait_for_exit":
		var req WaitForTerminalExitRequest
		if err := json.Unmarshal(params, &req); err != nil {
			s.writeRPCResponse(id, nil, fmt.Errorf("invalid params for terminal/wait_for_exit: %w", err))
			return
		}
		resp, err = h.HandleWaitForTerminalExit(ctx, req)
	case "terminal/kill":
		var req KillTerminalRequest
		if err := json.Unmarshal(params, &req); err != nil {
			s.writeRPCResponse(id, nil, fmt.Errorf("invalid params for terminal/kill: %w", err))
			return
		}
		resp, err = h.HandleKillTerminal(ctx, req)
	case "terminal/release":
		var req ReleaseTerminalRequest
		if err := json.Unmarshal(params, &req); err != nil {
			s.writeRPCResponse(id, nil, fmt.Errorf("invalid params for terminal/release: %w", err))
			return
		}
		resp, err = h.HandleReleaseTerminal(ctx, req)
	}

	if err != nil {
		s.writeRPCResponse(id, nil, err)
		return
	}
	s.writeRPCResponse(id, resp, nil)
}

// writeRPCResponse sends a JSON-RPC 2.0 response (or error) to the agent
// via stdin.
func (s *Session) writeRPCResponse(id json.RawMessage, result any, err error) {
	resp := struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Result  json.RawMessage `json:"result,omitempty"`
		Error   *Error          `json:"error,omitempty"`
	}{JSONRPC: "2.0", ID: id}

	if err != nil {
		resp.Error = &Error{Code: ErrorCodeInternal, Message: err.Error()}
	} else {
		resp.Result, _ = json.Marshal(result)
	}

	data, _ := json.Marshal(resp)
	s.writeMu.Lock()
	s.stdin.Write(append(data, '\n'))
	s.writeMu.Unlock()
}

// dispatchNotif routes a incoming notification to the EventHandler.
// Unrecognized notification methods are silently ignored per ACP spec.
func (s *Session) dispatchNotif(method string, params json.RawMessage) {
	switch method {
	case "session/update":
		s.dispatchSessionUpdate(params)
	case "$/cancel_request":
		s.handleCancelRequestNotif(params)
	}
}

// handleCancelRequestNotif handles an incoming $/cancel_request notification
// from the agent.  The agent is asking us to cancel an in-flight client→agent
// request.  We cancel the pending call so the blocked goroutine unblocks with
// a context-cancelled error.
func (s *Session) handleCancelRequestNotif(params json.RawMessage) {
	var notif CancelRequestNotification
	if json.Unmarshal(params, &notif) != nil {
		return
	}
	s.mu.Lock()
	call := s.pending[notif.RequestID]
	s.mu.Unlock()
	if call != nil {
		// Deliver a synthetic cancellation error so the blocked caller
		// returns promptly.
		call.resp.Error = &Error{Code: ErrorCodeRequestCancelled, Message: "request cancelled by agent"}
		close(call.done)
	}
}

func (s *Session) dispatchSessionUpdate(params json.RawMessage) {
	var notif SessionNotification
	if json.Unmarshal(params, &notif) != nil {
		return
	}
	s.mu.Lock()
	h := s.eh
	s.mu.Unlock()
	if h == nil {
		return
	}

	u := notif.Update
	switch u.SessionUpdate {
	case "agent_message_chunk":
		if cb := u.ContentAsBlock(); cb != nil {
			h.OnAgentMessage(cb.Text)
		}
	case "agent_thought_chunk":
		if cb := u.ContentAsBlock(); cb != nil {
			h.OnAgentThought(cb.Text)
		}
	case "user_message_chunk":
		if cb := u.ContentAsBlock(); cb != nil {
			h.OnUserMessage(cb.Text)
		}
	case "tool_call", "tool_call_update":
		title := ""
		if u.Title != nil {
			title = *u.Title
		}
		h.OnToolCall(ToolCallUpdate{
			ToolCallID: u.ToolCallID, Title: title,
			Kind: u.Kind, Status: u.Status,
			RawInput: u.RawInput, RawOutput: u.RawOutput,
			Content: u.ContentAsToolCallContent(), Locations: u.Locations,
		})
	case "plan":
		h.OnPlan(Plan{Entries: u.Entries})
	case "available_commands_update":
		h.OnAvailableCommandsUpdate(u.AvailableCommands)
	case "current_mode_update":
		h.OnModeUpdate(u.CurrentModeID)
	case "config_option_update":
		h.OnConfigOptionUpdate(u.ConfigOptions)
	case "usage_update":
		used, total := 0, 0
		if u.Used != nil {
			used = *u.Used
		}
		if u.Size != nil {
			total = *u.Size
		}
		h.OnUsageUpdate(used, total, u.Cost)
	case "session_info_update":
		title := ""
		if u.Title != nil {
			title = *u.Title
		}
		h.OnSessionInfo(title, u.Meta)
	}
}

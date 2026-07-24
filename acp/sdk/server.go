// Package acp implements the Agent Client Protocol v1.
//
// Server: expose an AgentHandler as an ACP-compliant agent over stdio.
// Client: connect to external ACP agents.
package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

// AgentHandler receives ACP requests from a client. Implement this
// interface to expose application logic as an ACP agent.
//
// Methods map 1:1 to the ACP v1 protocol:
//
//	https://agentclientprotocol.com/protocol/v1/schema
type AgentHandler interface {
	OnInitialize(ctx context.Context, req InitializeRequest) (*InitializeResponse, error)
	OnNewSession(ctx context.Context, req NewSessionRequest) (*NewSessionResponse, error)
	OnLoadSession(ctx context.Context, req LoadSessionRequest, sender SessionEventSender) (*LoadSessionResponse, error)
	OnResumeSession(ctx context.Context, req ResumeSessionRequest) (*ResumeSessionResponse, error)
	OnCloseSession(ctx context.Context, req CloseSessionRequest) (*CloseSessionResponse, error)
	OnDeleteSession(ctx context.Context, req DeleteSessionRequest) (*DeleteSessionResponse, error)
	OnListSessions(ctx context.Context, req ListSessionsRequest) (*ListSessionsResponse, error)
	OnSetSessionMode(ctx context.Context, req SetSessionModeRequest) (*SetSessionModeResponse, error)
	OnSetSessionConfigOption(ctx context.Context, req SetSessionConfigOptionRequest) (*SetSessionConfigOptionResponse, error)
	OnPrompt(ctx context.Context, req PromptRequest, sender SessionEventSender) (*PromptResponse, error)
	OnLogout(ctx context.Context, req LogoutRequest) (*LogoutResponse, error)
	OnCancel(ctx context.Context, sid SessionId) error
	OnAuthenticate(ctx context.Context, req AuthenticateRequest) (*AuthenticateResponse, error)
}

// SessionEventSender sends streaming events back to the ACP client during a
// prompt turn. Every method maps to a session/update notification variant.
type SessionEventSender interface {
	SendAgentMessage(text string) error
	SendAgentThought(text string) error
	SendToolCall(update ToolCallUpdate) error
	SendPlanUpdate(entries []PlanEntry) error
	SendAvailableCommands(cmds []AvailableCommand) error
	SendModeUpdate(modeID SessionModeId) error
	SendConfigOptionUpdate(opts []SessionConfigOption) error
	SendUsageUpdate(used, total int, cost *Cost) error
	SendSessionInfo(title string, metadata map[string]any) error

	// SendHistoryMessage replays a historical message during session/load.
	// sessionUpdate must be "user_message_chunk", "agent_message_chunk",
	// or "agent_thought_chunk". messageID identifies the message for
	// chunk grouping.
	SendHistoryMessage(sessionUpdate, text, messageID string) error
}

// ── Server ──

// Server exposes an [AgentHandler] as an ACP agent over a transport.
type Server struct {
	name    string
	version string
	handler AgentHandler
	logger  *slog.Logger
}

// NewServer creates an ACP [Server] with the given implementation identity.
func NewServer(name, version string, handler AgentHandler) *Server {
	return &Server{name: name, version: version, handler: handler}
}

// SetLogger directs diagnostics to the provided logger.
func (s *Server) SetLogger(l *slog.Logger) { s.logger = l }

// Run starts the ACP server on stdio. Blocks until the stream closes or ctx
// is cancelled.
func (s *Server) Run(ctx context.Context) error {
	return s.RunTransport(ctx, os.Stdout, os.Stdin)
}

// RunTransport starts the ACP server on custom I/O streams.
func (s *Server) RunTransport(ctx context.Context, w io.Writer, r io.Reader) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := &mux{
		handler:       s.handler,
		w:             w,
		queue:         newWriteQueue(),
		writeDone:     make(chan struct{}),
		cancelWrite:   cancel,
		cancelPending: make(map[string]context.CancelFunc),
		clientCalls:   make(map[string]*clientCall),
		logger:        s.logger,
	}
	if u, ok := s.handler.(ClientRPCUser); ok {
		u.SetClientRequester(mux)
	}
	go mux.writerLoop()
	return mux.serve(ctx, r)
}

// ── JSON-RPC 2.0 ──

// jsonrpcMessage is a JSON-RPC 2.0 message. Three shapes per the spec,
// distinguished by field presence:
//
//	Request:      {"jsonrpc":"2.0", "method":"...", "params":{...}, "id":"..."}
//	Notification: {"jsonrpc":"2.0", "method":"...", "params":{...}}
//	Response:     {"jsonrpc":"2.0", "result":{...}, "id":"..."}
//	Error:        {"jsonrpc":"2.0", "error":{...}, "id":"..."}
//
// JSON-RPC id is string, number, or null. json.RawMessage preserves
// the exact JSON representation — use [idString] to normalise for map keys.
type jsonrpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// idString normalises a JSON-RPC id value to a plain Go string suitable for
// map keying. Handles string ids ("foo" → "foo"), number ids (42 → "42"),
// and null/absent ids (→ "").
func idString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return fmt.Sprintf("%.0f", x)
	default:
		return fmt.Sprint(v)
	}
}

// clientCall tracks an in-flight agent→client JSON-RPC request.
type clientCall struct {
	resp rpcResponse
	done chan struct{}
}

// mux reads JSON-RPC 2.0 messages from stdin, routes them to handler
// methods, and writes responses (and notifications) to stdout.
//
// All writes to w go through a single writer goroutine fed by an
// unbounded FIFO queue ([writeQueue]). Handler code and tool goroutines
// never touch w directly — they only enqueue pre-marshaled frames via
// [mux.enqueue]. A blocked pipe (slow client) therefore blocks only the
// single writer goroutine, never any handler or tool goroutine, which is
// what prevents the prompt-turn deadlocks the prior dual-mutex + bufio
// transport produced under load.
type mux struct {
	handler AgentHandler
	w       io.Writer // written ONLY by writerLoop

	// Single-writer transport.
	queue       *writeQueue
	writeDone   chan struct{}    // closed when writerLoop exits
	writeErr    error            // first write error; set once via closeMu
	closeMu     sync.Mutex       // guards writeErr (set-once)
	cancelWrite context.CancelFunc // cancels serve ctx on transport failure

	// Prompt cancellation: client sends $/cancel_request with the
	// JSON-RPC id of the original session/prompt request. Keys are
	// normalised via idString. Guarded by cancelMu.
	cancelMu      sync.Mutex
	cancelPending map[string]context.CancelFunc

	// Agent→Client RPC state.
	clientNextID int64
	clientMu     sync.Mutex
	clientCalls  map[string]*clientCall

	// Per-session mutex: ACP requires sequential processing of prompts
	// within a session. LoadOrStore acquires the session lock.
	sessionLocks sync.Map // SessionId → *sync.Mutex

	// promptWG tracks in-flight session/prompt handlers (launched via
	// `go m.handlePrompt`). serve waits on it during shutdown so that a
	// turn's final response/notification enqueues land before the queue
	// is closed — preventing a response from being dropped when the
	// client disconnects (stdin EOF) or the context is cancelled mid-turn.
	promptWG sync.WaitGroup

	logger *slog.Logger
}

// errQueueClosed is returned by writeQueue.push after Close (transport
// has failed / shut down) and by writeQueue.pop once closed and drained.
var errQueueClosed = errors.New("acp: transport closed")

// writeQueue is an unbounded FIFO of pre-marshaled JSON-RPC frames (each
// []byte already contains its trailing '\n'). It never blocks producers
// except after Close, which transitions the queue to a fast-fail state.
// A single consumer (writerLoop) drains it.
//
// An unbounded queue is required: the stdout pipe can stall arbitrarily
// long when the client reads slowly, and any bounded buffer (channel or
// fixed bufio.Writer) would back-pressure producers and reintroduce the
// deadlock this transport was rewritten to eliminate.
type writeQueue struct {
	mu     sync.Mutex
	cond   *sync.Cond
	frames [][]byte
	closed bool
}

func newWriteQueue() *writeQueue {
	q := &writeQueue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// push appends a frame. It never blocks. Returns errQueueClosed if the
// queue is closed (transport failed); callers treat that as "client is
// gone". frame must be owned by the caller (no aliasing).
func (q *writeQueue) push(frame []byte) error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return errQueueClosed
	}
	q.frames = append(q.frames, frame)
	q.mu.Unlock()
	q.cond.Signal()
	return nil
}

// pop returns the next frame, blocking until one arrives. Returns
// errQueueClosed when the queue is closed AND drained. Used only by
// the single writer goroutine.
func (q *writeQueue) pop() ([]byte, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.frames) == 0 && !q.closed {
		q.cond.Wait()
	}
	if len(q.frames) == 0 { // closed && drained
		return nil, errQueueClosed
	}
	f := q.frames[0]
	q.frames[0] = nil
	q.frames = q.frames[1:]
	return f, nil
}

// close marks the queue closed. Pending frames remain drainable by the
// writer; new pushes fast-fail. Wakes the consumer.
func (q *writeQueue) close() {
	q.mu.Lock()
	q.closed = true
	q.mu.Unlock()
	q.cond.Broadcast()
}

// drainAndDrop discards all pending frames; used after the writer exits
// on a write error so no producer can linger on a dead transport.
func (q *writeQueue) drainAndDrop() {
	q.mu.Lock()
	q.frames = nil
	q.mu.Unlock()
}

func (m *mux) logf(format string, args ...any) {
	if m.logger != nil {
		m.logger.Debug(fmt.Sprintf("acp: "+format, args...))
	}
}

// serve reads newline-delimited JSON-RPC 2.0 messages from r until EOF
// or ctx cancellation. bufio.Scanner blocks on Read; by running the
// scanner in a goroutine and selecting on ctx.Done(), Ctrl+C exits
// immediately.
func (m *mux) serve(ctx context.Context, r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	// Writer lifecycle. Defer order is LIFO (last declared runs first):
	//   1. cancelInFlightPrompts() — cancel any still-running prompt
	//      handlers so they cannot block on a client RPC that the now-gone
	//      client will never answer.
	//   2. promptWG.Wait() — wait for in-flight prompt handlers to finish
	//      pushing their final response/notification frames BEFORE we close
	//      the queue, so a mid-turn disconnect or ctx cancel cannot drop a
	//      response.
	//   3. queue.close()   — signal the writer no more frames are coming
	//      so it drains the remaining ones.
	//   4. <-writeDone     — block until the writer goroutine has exited,
	//      guaranteeing serve() (and RunTransport) does not return before
	//      all queued frames are flushed.
	defer func() { <-m.writeDone }()
	defer m.queue.close()
	defer m.promptWG.Wait()
	defer m.cancelInFlightPrompts()

	lines := make(chan []byte, 8)
	errCh := make(chan error, 1)

	go func() {
		for scanner.Scan() {
			b := scanner.Bytes()
			if len(b) == 0 {
				continue
			}
			// Copy — scanner reuses buffer.
			line := make([]byte, len(b))
			copy(line, b)
			select {
			case lines <- line:
			case <-ctx.Done():
				return
			}
		}
		errCh <- scanner.Err()
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return err
		case line := <-lines:
			var msg jsonrpcMessage
			if err := json.Unmarshal(line, &msg); err != nil {
				m.logf("parse error: %v", err)
				continue
			}
			switch {
			case msg.Method != "":
				m.route(msg)
			case len(msg.ID) > 0:
				m.deliverResponse(msg)
			default:
				m.logf("unrecognised message (no method, no id)")
			}
		}
	}
}

// route dispatches a JSON-RPC 2.0 request or notification by method name.
func (m *mux) route(msg jsonrpcMessage) {
	isReq := len(msg.ID) > 0

	switch msg.Method {
	case "initialize":
		m.handleInit(msg)
	case "authenticate":
		if isReq {
			dispatch(m, msg, func(ctx context.Context, req AuthenticateRequest) (*AuthenticateResponse, error) {
				return m.handler.OnAuthenticate(ctx, req)
			})
		}
	case "session/new":
		if isReq {
			dispatch(m, msg, func(ctx context.Context, req NewSessionRequest) (*NewSessionResponse, error) {
				return m.handler.OnNewSession(ctx, req)
			})
		}
	case "session/load":
		if isReq {
			m.handleLoadSession(msg)
		}
	case "session/resume":
		if isReq {
			dispatch(m, msg, func(ctx context.Context, req ResumeSessionRequest) (*ResumeSessionResponse, error) {
				return m.handler.OnResumeSession(ctx, req)
			})
		}
	case "session/close":
		if isReq {
			dispatch(m, msg, func(ctx context.Context, req CloseSessionRequest) (*CloseSessionResponse, error) {
				return m.handler.OnCloseSession(ctx, req)
			})
		}
	case "session/delete":
		if isReq {
			dispatch(m, msg, func(ctx context.Context, req DeleteSessionRequest) (*DeleteSessionResponse, error) {
				return m.handler.OnDeleteSession(ctx, req)
			})
		}
	case "session/list":
		if isReq {
			dispatch(m, msg, func(ctx context.Context, req ListSessionsRequest) (*ListSessionsResponse, error) {
				return m.handler.OnListSessions(ctx, req)
			})
		}
	case "session/prompt":
		if isReq {
			// Run in a goroutine so the serve loop continues
			// reading stdin. Without this, Agent→Client RPC
			// (permission requests, fs reads, terminal) deadlocks:
			// the response arrives on stdin but serve() is
			// blocked in route() waiting for OnPrompt to return.
			// Tracked on promptWG so serve()'s shutdown drains the
			// turn's final response before the queue closes.
			m.promptWG.Add(1)
			go func() {
				defer m.promptWG.Done()
				m.handlePrompt(msg)
			}()
		}
	case "session/set_mode":
		if isReq {
			dispatch(m, msg, func(ctx context.Context, req SetSessionModeRequest) (*SetSessionModeResponse, error) {
				return m.handler.OnSetSessionMode(ctx, req)
			})
		}
	case "session/set_config_option":
		if isReq {
			dispatch(m, msg, func(ctx context.Context, req SetSessionConfigOptionRequest) (*SetSessionConfigOptionResponse, error) {
				return m.handler.OnSetSessionConfigOption(ctx, req)
			})
		}
	case "logout":
		if isReq {
			dispatch(m, msg, func(ctx context.Context, req LogoutRequest) (*LogoutResponse, error) {
				return m.handler.OnLogout(ctx, req)
			})
		}
	case "session/cancel":
		m.handleCancel(msg)
	case "$/cancel_request":
		m.handleCancelRequest(msg)
	default:
		if isReq {
			m.writeError(msg.ID, ErrorCodeMethodNotFound, fmt.Sprintf("method %q not found", msg.Method))
		}
		// Unrecognised notifications → silently ignored per spec.
	}
}

// ── Handlers ──

func (m *mux) handleInit(msg jsonrpcMessage) {
	dispatch(m, msg, func(ctx context.Context, req InitializeRequest) (*InitializeResponse, error) {
		return m.handler.OnInitialize(ctx, req)
	})
}

func (m *mux) handleLoadSession(msg jsonrpcMessage) {
	var req LoadSessionRequest
	if err := json.Unmarshal(msg.Params, &req); err != nil {
		m.writeError(msg.ID, ErrorCodeInvalidParams, err.Error())
		return
	}
	sender := &promptSender{m: m, sid: req.SessionID}
	resp, err := m.handler.OnLoadSession(context.Background(), req, sender)
	if err != nil {
		m.writeError(msg.ID, ErrorCodeInternal, err.Error())
		return
	}
	m.writeResult(msg.ID, resp)
}

// handlePrompt processes a session/prompt request. Per ACP spec, prompts on
// the same session are serialised.
func (m *mux) handlePrompt(msg jsonrpcMessage) {
	var req PromptRequest
	if err := json.Unmarshal(msg.Params, &req); err != nil {
		m.writeError(msg.ID, ErrorCodeInvalidParams, err.Error())
		return
	}

	// Serialise prompts per session. The protocol requires sequential
	// processing within a session.
	muI, _ := m.sessionLocks.LoadOrStore(req.SessionID, &sync.Mutex{})
	mu := muI.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	reqID := idString(msg.ID)
	m.cancelMu.Lock()
	m.cancelPending[reqID] = cancel
	m.cancelMu.Unlock()

	sender := &promptSender{m: m, sid: req.SessionID}
	resp, err := m.handler.OnPrompt(ctx, req, sender)

	// Check whether we were cancelled before the handler returned.
	// If so, report cancellation even if the handler returned an error.
	wasCancelled := ctx.Err() != nil
	cancel()

	m.cancelMu.Lock()
	delete(m.cancelPending, reqID)
	m.cancelMu.Unlock()

	if err != nil {
		// Only report cancellation if the handler was still running
		// when cancel fired (OnCancel → ctx cancel → StreamAborted).
		// Other errors must NOT be swallowed as cancelled.
		if wasCancelled {
			m.writeResult(msg.ID, PromptResponse{StopReason: StopReasonCancelled})
			return
		}
		m.writeError(msg.ID, ErrorCodeInternal, err.Error())
		return
	}
	m.writeResult(msg.ID, resp)
}

func (m *mux) handleCancel(msg jsonrpcMessage) {
	var notif CancelNotification
	if json.Unmarshal(msg.Params, &notif) != nil {
		return
	}
	_ = m.handler.OnCancel(context.Background(), notif.SessionID)
}

// handleCancelRequest processes $/cancel_request. The client sends the
// JSON-RPC id of the original request as a plain string (per ACP spec,
// CancelRequestNotification.RequestID is a string). We match it against
// cancelPending keys normalised via idString.
func (m *mux) handleCancelRequest(msg jsonrpcMessage) {
	var notif CancelRequestNotification
	if json.Unmarshal(msg.Params, &notif) != nil {
		return
	}
	m.cancelMu.Lock()
	cancel, ok := m.cancelPending[string(notif.RequestID)]
	m.cancelMu.Unlock()
	if ok {
		cancel()
	}
}

// cancelInFlightPrompts cancels every still-running prompt handler's
// context. Called during serve() shutdown (before promptWG.Wait) so a
// handler blocked on an Agent→Client RPC that the now-disconnected client
// will never answer is released instead of stranding shutdown.
func (m *mux) cancelInFlightPrompts() {
	m.cancelMu.Lock()
	for _, cancel := range m.cancelPending {
		cancel()
	}
	m.cancelMu.Unlock()
}

// ── Generic dispatcher ──

func dispatch[Req, Resp any](m *mux, msg jsonrpcMessage, fn func(context.Context, Req) (Resp, error)) {
	var req Req
	if err := json.Unmarshal(msg.Params, &req); err != nil {
		m.writeError(msg.ID, ErrorCodeInvalidParams, err.Error())
		return
	}
	resp, err := fn(context.Background(), req)
	if err != nil {
		m.writeError(msg.ID, ErrorCodeInternal, err.Error())
		return
	}
	m.writeResult(msg.ID, resp)
}

// ── Response writers ──

func (m *mux) writeResult(id json.RawMessage, result any) {
	body, _ := json.Marshal(result)
	resp := jsonrpcMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result:  body,
	}
	_ = m.enqueue(resp)
}

func (m *mux) writeError(id json.RawMessage, code ErrorCode, message string) {
	resp := jsonrpcMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &Error{Code: code, Message: message},
	}
	_ = m.enqueue(resp)
}

// enqueue marshals a JSON-RPC message and pushes it (with trailing '\n')
// to the single-writer queue. ALL writes to m.w go through here. The
// error is returned so callers that care (agent→client request) can
// react; response/notification callers discard it, but a failure still
// shuts the transport down centrally via writerLoop.
func (m *mux) enqueue(msg jsonrpcMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return m.queue.push(append(data, '\n'))
}

// writerLoop is the ONLY goroutine that writes to m.w. It drains the
// unbounded queue frame by frame. On the first write error it records
// the error, closes the queue (so producers fast-fail), cancels the
// serve context (unblocking serve and any pending agent→client request
// selects), and exits.
func (m *mux) writerLoop() {
	defer close(m.writeDone)
	for {
		frame, err := m.queue.pop()
		if err != nil {
			return // queue closed and drained
		}
		if _, werr := m.w.Write(frame); werr != nil {
			m.logf("transport write error: %v", werr)
			m.setWriteErr(werr)
			m.queue.close() // producers fast-fail from now on
			if m.cancelWrite != nil {
				m.cancelWrite() // unblock serve() + pending request() selects
			}
			m.queue.drainAndDrop()
			return
		}
	}
}

func (m *mux) setWriteErr(err error) {
	m.closeMu.Lock()
	if m.writeErr == nil {
		m.writeErr = err
	}
	m.closeMu.Unlock()
}

// ── Agent→Client RPC ──

// nextID returns the next request ID for agent→client calls. The return
// value is marshalled as a JSON-RPC string id.
func (m *mux) nextID() string {
	m.clientMu.Lock()
	m.clientNextID++
	id := m.clientNextID
	m.clientMu.Unlock()
	return fmt.Sprintf("ac-%d", id)
}

// request sends a JSON-RPC 2.0 request to the client, registers a pending
// call, and blocks until the response arrives or ctx is cancelled.
func (m *mux) request(ctx context.Context, method string, params any) (rpcResponse, error) {
	// If the caller supplied no deadline, cap the call so a non-responsive
	// client cannot hang the agent indefinitely (e.g. an abandoned manual-
	// mode approval dialog). Caller-supplied deadlines take precedence.
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
	}

	id := m.nextID()
	idJSON, _ := json.Marshal(id)

	req := jsonrpcMessage{
		JSONRPC: "2.0",
		ID:      idJSON,
		Method:  method,
	}
	if params != nil {
		p, _ := json.Marshal(params)
		req.Params = p
	}

	callKey := idString(idJSON)
	call := &clientCall{done: make(chan struct{})}
	m.clientMu.Lock()
	m.clientCalls[callKey] = call
	m.clientMu.Unlock()

	defer func() {
		m.clientMu.Lock()
		delete(m.clientCalls, callKey)
		m.clientMu.Unlock()
	}()

	if err := m.enqueue(req); err != nil {
		return rpcResponse{}, err // transport dead — give up before waiting
	}

	select {
	case <-call.done:
	case <-ctx.Done():
		return rpcResponse{}, ctx.Err()
	}
	return call.resp, nil
}

// deliverResponse routes an incoming JSON-RPC 2.0 response to the
// waiting agent→client call.
func (m *mux) deliverResponse(msg jsonrpcMessage) {
	callKey := idString(msg.ID)
	m.clientMu.Lock()
	call := m.clientCalls[callKey]
	m.clientMu.Unlock()
	if call == nil {
		return
	}
	if msg.Error != nil {
		call.resp.Error = msg.Error
	} else {
		call.resp.Result = msg.Result
	}
	close(call.done)
}

// doCall sends an agent→client request and unmarshals the response.
// JSON-RPC errors are returned as a Go error.
func doCall[Req, Resp any](m *mux, ctx context.Context, method string, req Req) (*Resp, error) {
	raw, err := m.request(ctx, method, req)
	if err != nil {
		return nil, err
	}
	if raw.Error != nil {
		return nil, fmt.Errorf("acp: %s call failed: %s", method, raw.Error.Message)
	}
	var out Resp
	if err := json.Unmarshal(raw.Result, &out); err != nil {
		return nil, fmt.Errorf("acp: unmarshal %s response: %w", method, err)
	}
	return &out, nil
}

// ── ClientRequester implementation ──

func (m *mux) RequestPermission(ctx context.Context, req RequestPermissionRequest) (*RequestPermissionResponse, error) {
	return doCall[RequestPermissionRequest, RequestPermissionResponse](m, ctx, "session/request_permission", req)
}
func (m *mux) ReadTextFile(ctx context.Context, req ReadTextFileRequest) (*ReadTextFileResponse, error) {
	return doCall[ReadTextFileRequest, ReadTextFileResponse](m, ctx, "fs/read_text_file", req)
}
func (m *mux) WriteTextFile(ctx context.Context, req WriteTextFileRequest) (*WriteTextFileResponse, error) {
	return doCall[WriteTextFileRequest, WriteTextFileResponse](m, ctx, "fs/write_text_file", req)
}
func (m *mux) CreateTerminal(ctx context.Context, req CreateTerminalRequest) (*CreateTerminalResponse, error) {
	return doCall[CreateTerminalRequest, CreateTerminalResponse](m, ctx, "terminal/create", req)
}
func (m *mux) TerminalOutput(ctx context.Context, req TerminalOutputRequest) (*TerminalOutputResponse, error) {
	return doCall[TerminalOutputRequest, TerminalOutputResponse](m, ctx, "terminal/output", req)
}
func (m *mux) WaitForTerminalExit(ctx context.Context, req WaitForTerminalExitRequest) (*WaitForTerminalExitResponse, error) {
	return doCall[WaitForTerminalExitRequest, WaitForTerminalExitResponse](m, ctx, "terminal/wait_for_exit", req)
}
func (m *mux) KillTerminal(ctx context.Context, req KillTerminalRequest) (*KillTerminalResponse, error) {
	return doCall[KillTerminalRequest, KillTerminalResponse](m, ctx, "terminal/kill", req)
}
func (m *mux) ReleaseTerminal(ctx context.Context, req ReleaseTerminalRequest) (*ReleaseTerminalResponse, error) {
	return doCall[ReleaseTerminalRequest, ReleaseTerminalResponse](m, ctx, "terminal/release", req)
}

// SendSessionUpdate writes a session/update notification (out of a
// prompt turn, e.g. current_mode_update after session/set_mode) through
// the single-writer queue.
func (m *mux) SendSessionUpdate(sid SessionId, update SessionUpdate) error {
	notif := jsonrpcMessage{
		JSONRPC: "2.0",
		Method:  "session/update",
	}
	params, _ := json.Marshal(SessionNotification{SessionID: sid, Update: update})
	notif.Params = params
	return m.enqueue(notif)
}

var _ ClientRequester = (*mux)(nil)
var _ SessionUpdateSender = (*mux)(nil)

// ── promptSender ──

type promptSender struct {
	m   *mux
	sid SessionId
}

// send enqueues a session/update notification for this turn. It never
// blocks — the unbounded queue absorbs bursts from concurrent tool
// goroutines; the single writer goroutine performs the actual pipe I/O.
// A transport failure is recorded centrally by writerLoop and surfaces
// as the turn unwinding.
func (s *promptSender) send(update SessionUpdate) {
	notif := jsonrpcMessage{
		JSONRPC: "2.0",
		Method:  "session/update",
	}
	params, _ := json.Marshal(SessionNotification{SessionID: s.sid, Update: update})
	notif.Params = params
	_ = s.m.enqueue(notif)
}

func (s *promptSender) SendAgentMessage(text string) error {
	su := SessionUpdate{SessionUpdate: "agent_message_chunk"}
	su.SetContentBlock(ContentBlock{Type: "text", Text: text})
	s.send(su)
	return nil
}

func (s *promptSender) SendAgentThought(text string) error {
	su := SessionUpdate{SessionUpdate: "agent_thought_chunk"}
	su.SetContentBlock(ContentBlock{Type: "text", Text: text})
	s.send(su)
	return nil
}

func (s *promptSender) SendToolCall(tc ToolCallUpdate) error {
	su := "tool_call"
	if tc.Status != "" && tc.Status != "pending" {
		su = "tool_call_update"
	}
	var t *string
	if tc.Title != "" {
		t = &tc.Title
	}
	u := SessionUpdate{
		SessionUpdate: su,
		ToolCallID:    tc.ToolCallID,
		Title:         t,
		Kind:          tc.Kind,
		Status:        tc.Status,
		RawInput:      tc.RawInput,
		RawOutput:     tc.RawOutput,
		Locations:     tc.Locations,
	}
	u.SetToolCallContent(tc.Content)
	s.send(u)
	return nil
}

func (s *promptSender) SendPlanUpdate(entries []PlanEntry) error {
	s.send(SessionUpdate{SessionUpdate: "plan", Entries: entries})
	return nil
}

func (s *promptSender) SendAvailableCommands(cmds []AvailableCommand) error {
	s.send(SessionUpdate{SessionUpdate: "available_commands_update", AvailableCommands: cmds})
	return nil
}

func (s *promptSender) SendModeUpdate(modeID SessionModeId) error {
	s.send(SessionUpdate{SessionUpdate: "current_mode_update", CurrentModeID: modeID})
	return nil
}

func (s *promptSender) SendConfigOptionUpdate(opts []SessionConfigOption) error {
	s.send(SessionUpdate{SessionUpdate: "config_option_update", ConfigOptions: opts})
	return nil
}

func (s *promptSender) SendUsageUpdate(used, total int, cost *Cost) error {
	s.send(SessionUpdate{SessionUpdate: "usage_update", Used: &used, Size: &total, Cost: cost})
	return nil
}

func (s *promptSender) SendSessionInfo(title string, metadata map[string]any) error {
	var t *string
	if title != "" {
		t = &title
	}
	su := SessionUpdate{
		SessionUpdate: "session_info_update",
		Title:         t,
		Meta:          metadata,
	}
	s.send(su)
	return nil
}

func (s *promptSender) SendHistoryMessage(sessionUpdate, text, messageID string) error {
	su := SessionUpdate{SessionUpdate: sessionUpdate}
	su.SetContentBlock(ContentBlock{Type: "text", Text: text})
	if messageID != "" {
		mid := MessageId(messageID)
		su.MessageID = &mid
	}
	s.send(su)
	return nil
}

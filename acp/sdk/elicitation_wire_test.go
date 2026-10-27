package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"
)

// elicitWireHandler is an AgentHandler that fires one elicitation/create
// Agent→Client RPC when prompted.
type elicitWireHandler struct {
	mu       sync.Mutex
	r        ClientRequester
	result   *ElicitResult
	err      error
	doneOnce sync.Once
	done     chan struct{}
}

func (h *elicitWireHandler) SetClientRequester(r ClientRequester) {
	h.mu.Lock()
	h.r = r
	h.mu.Unlock()
}

func (h *elicitWireHandler) OnInitialize(ctx context.Context, req InitializeRequest) (*InitializeResponse, error) {
	return &InitializeResponse{ProtocolVersion: 1}, nil
}

func (h *elicitWireHandler) OnPrompt(ctx context.Context, req PromptRequest, sender SessionEventSender) (*PromptResponse, error) {
	h.mu.Lock()
	r := h.r
	h.mu.Unlock()
	res, err := r.ElicitationCreate(ctx, ElicitCreateParams{
		SessionID: "s1",
		Mode:      "form",
		Message:   "wire test",
		RequestedSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
			},
			"required": []string{"name"},
		},
	})
	h.mu.Lock()
	h.result, h.err = res, err
	h.mu.Unlock()
	h.doneOnce.Do(func() { close(h.done) })
	return &PromptResponse{StopReason: StopReasonEndTurn}, nil
}

// Stubs for the rest of AgentHandler.
func (h *elicitWireHandler) OnNewSession(context.Context, NewSessionRequest) (*NewSessionResponse, error) {
	return &NewSessionResponse{SessionID: "s1"}, nil
}
func (h *elicitWireHandler) OnLoadSession(context.Context, LoadSessionRequest, SessionEventSender) (*LoadSessionResponse, error) {
	return &LoadSessionResponse{}, nil
}
func (h *elicitWireHandler) OnResumeSession(context.Context, ResumeSessionRequest) (*ResumeSessionResponse, error) {
	return &ResumeSessionResponse{}, nil
}
func (h *elicitWireHandler) OnCloseSession(context.Context, CloseSessionRequest) (*CloseSessionResponse, error) {
	return &CloseSessionResponse{}, nil
}
func (h *elicitWireHandler) OnDeleteSession(context.Context, DeleteSessionRequest) (*DeleteSessionResponse, error) {
	return &DeleteSessionResponse{}, nil
}
func (h *elicitWireHandler) OnListSessions(context.Context, ListSessionsRequest) (*ListSessionsResponse, error) {
	return &ListSessionsResponse{}, nil
}
func (h *elicitWireHandler) OnListMessages(context.Context, ListMessagesRequest) (*ListMessagesResponse, error) {
	return &ListMessagesResponse{}, nil
}
func (h *elicitWireHandler) OnForkSession(context.Context, ForkSessionRequest) (*ForkSessionResponse, error) {
	return &ForkSessionResponse{SessionID: "fork1"}, nil
}

func (h *elicitWireHandler) OnListConfigOptions(context.Context, ListConfigOptionsRequest) (*ListConfigOptionsResponse, error) {
	return &ListConfigOptionsResponse{}, nil
}
func (h *elicitWireHandler) OnSetSessionMode(context.Context, SetSessionModeRequest) (*SetSessionModeResponse, error) {
	return &SetSessionModeResponse{}, nil
}
func (h *elicitWireHandler) OnSetSessionConfigOption(context.Context, SetSessionConfigOptionRequest) (*SetSessionConfigOptionResponse, error) {
	return &SetSessionConfigOptionResponse{}, nil
}
func (h *elicitWireHandler) OnAuthenticate(context.Context, AuthenticateRequest) (*AuthenticateResponse, error) {
	return &AuthenticateResponse{}, nil
}
func (h *elicitWireHandler) OnLogout(context.Context, LogoutRequest) (*LogoutResponse, error) {
	return &LogoutResponse{}, nil
}
func (h *elicitWireHandler) OnCancel(context.Context, SessionId) error { return nil }

var _ AgentHandler = (*elicitWireHandler)(nil)

// readJSONRPC scans agent output for the next JSON-RPC frame.
func readJSONRPC(t *testing.T, sc *bufio.Scanner) map[string]any {
	t.Helper()
	for sc.Scan() {
		var m map[string]any
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			continue
		}
		return m
	}
	t.Fatal("no more agent output")
	return nil
}

// TestElicitationWireRoundtrip: a real elicitation/create Agent→Client
// request over RunTransport, answered by a raw JSON-RPC response line —
// proves request serialization (scope, mode, schema) and response decoding.
func TestElicitationWireRoundtrip(t *testing.T) {
	h := &elicitWireHandler{done: make(chan struct{})}
	srv := NewServer("wire-test", "1.0.0", h)
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go func() { _ = srv.RunTransport(ctx, stdoutW, stdinR) }()

	go func() {
		_, _ = io.WriteString(stdinW, `{"jsonrpc":"2.0","id":"1","method":"session/prompt","params":{"sessionId":"s1","prompt":[{"type":"text","text":"go"}]}}`+"\n")
	}()
	t.Cleanup(func() { stdinW.Close(); stdoutW.Close() })

	sc := bufio.NewScanner(stdoutR)

	// Read the elicitation/create request the handler fired.
	var elicit map[string]any
	deadline := time.After(5 * time.Second)
	gotCh := make(chan map[string]any, 1)
	go func() {
		for sc.Scan() {
			var m map[string]any
			if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
				continue
			}
			if m["method"] == "elicitation/create" {
				gotCh <- m
				return
			}
		}
	}()
	select {
	case elicit = <-gotCh:
	case <-deadline:
		t.Fatal("no elicitation/create received")
	}

	params, ok := elicit["params"].(map[string]any)
	if !ok {
		t.Fatalf("params = %#v", elicit["params"])
	}
	if params["sessionId"] != "s1" || params["mode"] != "form" || params["message"] != "wire test" {
		t.Errorf("params = %#v, want sessionId s1 / mode form / message wire test", params)
	}
	if params["requestedSchema"] == nil {
		t.Error("requestedSchema missing")
	}

	// Answer it like a real client would.
	id := elicit["id"]
	resp := `{"jsonrpc":"2.0","id":` + jsonNumber(id) + `,"result":{"action":"accept","content":{"name":"alice"}}}` + "\n"
	go func() { _, _ = io.WriteString(stdinW, resp) }()

	select {
	case <-h.done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler never returned")
	}
	if h.err != nil {
		t.Fatalf("ElicitationCreate: %v", h.err)
	}
	if h.result.Action != "accept" || h.result.Content["name"] != "alice" {
		t.Errorf("result = %+v, want accept {name: alice}", h.result)
	}
}

// TestElicitationCancelNotifies: cancelling the ElicitationCreate ctx makes
// the agent emit $/cancel_request so the client can dismiss its form.
func TestElicitationCancelNotifies(t *testing.T) {
	h := &elicitWireHandler{done: make(chan struct{})}
	srv := NewServer("wire-test", "1.0.0", h)
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go func() { _ = srv.RunTransport(ctx, stdoutW, stdinR) }()

	// Drive: read frames until elicitation/create shows, then cancel the
	// prompt via the real protocol path ($/cancel_request with the prompt's
	// JSON-RPC id) — the prompt ctx cancels, which cancels the in-flight
	// ElicitationCreate, which must emit $/cancel_request to the client.
	go func() {
		_, _ = io.WriteString(stdinW, `{"jsonrpc":"2.0","id":"1","method":"session/prompt","params":{"sessionId":"s1","prompt":[{"type":"text","text":"go"}]}}`+"\n")
	}()
	t.Cleanup(func() { stdinW.Close(); stdoutW.Close() })
	sc := bufio.NewScanner(stdoutR)
	gotCh := make(chan struct{})
	go func() {
		for sc.Scan() {
			var m map[string]any
			if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
				continue
			}
			if m["method"] == "elicitation/create" {
				close(gotCh)
				return
			}
		}
	}()
	select {
	case <-gotCh:
	case <-time.After(5 * time.Second):
		t.Fatal("no elicitation/create received")
	}
	_, _ = io.WriteString(stdinW, `{"jsonrpc":"2.0","method":"$/cancel_request","params":{"requestId":"1"}}`+"\n")

	// The agent must now emit $/cancel_request with the request id.
	sc2 := bufio.NewScanner(stdoutR)
	found := false
	deadline := time.Now().Add(5 * time.Second)
	for !found && time.Now().Before(deadline) {
		if !sc2.Scan() {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(sc2.Bytes(), &m); err != nil {
			continue
		}
		if m["method"] == "$/cancel_request" {
			p, _ := m["params"].(map[string]any)
			if p == nil || p["requestId"] == nil {
				t.Errorf("$/cancel_request params = %#v, want requestId", p)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("no $/cancel_request emitted after ctx cancel")
	}
}

// TestUnknownAgentClientMethodRejected: the client answers an unknown
// Agent→Client method with a JSON-RPC error instead of a null result.
func TestUnknownAgentClientMethodRejected(t *testing.T) {
	// Fake agent output: an unknown method with an id.
	agentOutR, agentOutW := io.Pipe()
	clientInR, clientInW := io.Pipe()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := NewClient("test-client", "1.0.0")
	sess := client.ConnectIO(ctx, clientInW, agentOutR)
	defer sess.Close()

	stub := &stubClientReqHandler{}
	sess.SetClientRequestHandler(stub)

	go func() {
		_, _ = io.WriteString(agentOutW, `{"jsonrpc":"2.0","id":"5","method":"bogus/method","params":{}}`+"\n")
	}()

	sc := bufio.NewScanner(clientInR)
	var resp map[string]any
	deadline := time.After(5 * time.Second)
	respCh := make(chan map[string]any, 1)
	go func() {
		for sc.Scan() {
			var m map[string]any
			if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
				continue
			}
			if m["id"] != nil && m["error"] != nil {
				respCh <- m
				return
			}
		}
	}()
	select {
	case resp = <-respCh:
	case <-deadline:
		t.Fatal("no error response for unknown method")
	}
	errObj, _ := resp["error"].(map[string]any)
	if errObj == nil || errObj["message"] == "" {
		t.Errorf("error = %#v, want a message naming the unknown method", resp["error"])
	}
}

// stubClientReqHandler satisfies ClientRequestHandler with no-op stubs.
type stubClientReqHandler struct{}

func (s *stubClientReqHandler) HandleRequestPermission(context.Context, RequestPermissionRequest) (*RequestPermissionResponse, error) {
	return nil, nil
}
func (s *stubClientReqHandler) HandleReadTextFile(context.Context, ReadTextFileRequest) (*ReadTextFileResponse, error) {
	return nil, nil
}
func (s *stubClientReqHandler) HandleWriteTextFile(context.Context, WriteTextFileRequest) (*WriteTextFileResponse, error) {
	return nil, nil
}
func (s *stubClientReqHandler) HandleCreateTerminal(context.Context, CreateTerminalRequest) (*CreateTerminalResponse, error) {
	return nil, nil
}
func (s *stubClientReqHandler) HandleTerminalOutput(context.Context, TerminalOutputRequest) (*TerminalOutputResponse, error) {
	return nil, nil
}
func (s *stubClientReqHandler) HandleWaitForTerminalExit(context.Context, WaitForTerminalExitRequest) (*WaitForTerminalExitResponse, error) {
	return nil, nil
}
func (s *stubClientReqHandler) HandleKillTerminal(context.Context, KillTerminalRequest) (*KillTerminalResponse, error) {
	return nil, nil
}
func (s *stubClientReqHandler) HandleReleaseTerminal(context.Context, ReleaseTerminalRequest) (*ReleaseTerminalResponse, error) {
	return nil, nil
}
func (s *stubClientReqHandler) HandleElicitationCreate(context.Context, ElicitCreateParams) (*ElicitResult, error) {
	return nil, nil
}

// jsonNumber marshals a JSON-RPC id (any) back to its raw form.
func jsonNumber(id any) string {
	b, _ := json.Marshal(id)
	return string(b)
}

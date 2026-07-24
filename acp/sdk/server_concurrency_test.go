package acp

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"
)

// This file guards against the concurrency deadlock that the prior
// transport (dual-mutex + bufio.Writer + 100ms flush goroutine) had: a
// slow client reader stalled the stdout pipe, the bufio auto-flush ran
// under a mutex held by every notification writer, so concurrent tool
// goroutines piled up against that mutex and the prompt turn deadlocked.
//
// The rewritten transport funnels every write through a single
// writer goroutine fed by an unbounded queue, so enqueues never block
// regardless of pipe back-pressure. These tests assert exactly that.

// slowWriter sleeps on every Write to simulate a client reading frames
// slower than the agent produces them.
type slowWriter struct {
	mu    sync.Mutex
	w     io.Writer
	delay time.Duration
}

func (s *slowWriter) Write(p []byte) (int, error) {
	time.Sleep(s.delay)
	return s.w.Write(p)
}

// floodHandler implements AgentHandler. Its OnPrompt spawns many
// goroutines that each flood notifications through the SessionEventSender
// — the concurrent tool-output pattern — then returns. With a blocking
// transport this deadlocks; with the single-writer queue it returns
// immediately.
type floodHandler struct {
	promptDone chan struct{}
}

var _ AgentHandler = (*floodHandler)(nil)

func (floodHandler) OnInitialize(context.Context, InitializeRequest) (*InitializeResponse, error) {
	return &InitializeResponse{ProtocolVersion: 1}, nil
}
func (floodHandler) OnNewSession(context.Context, NewSessionRequest) (*NewSessionResponse, error) {
	return &NewSessionResponse{SessionID: "s1"}, nil
}
func (floodHandler) OnLoadSession(context.Context, LoadSessionRequest, SessionEventSender) (*LoadSessionResponse, error) {
	return &LoadSessionResponse{}, nil
}
func (floodHandler) OnResumeSession(context.Context, ResumeSessionRequest) (*ResumeSessionResponse, error) {
	return &ResumeSessionResponse{}, nil
}
func (floodHandler) OnCloseSession(context.Context, CloseSessionRequest) (*CloseSessionResponse, error) {
	return &CloseSessionResponse{}, nil
}
func (floodHandler) OnDeleteSession(context.Context, DeleteSessionRequest) (*DeleteSessionResponse, error) {
	return &DeleteSessionResponse{}, nil
}
func (floodHandler) OnListSessions(context.Context, ListSessionsRequest) (*ListSessionsResponse, error) {
	return &ListSessionsResponse{Sessions: []SessionInfo{}}, nil
}
func (floodHandler) OnSetSessionMode(context.Context, SetSessionModeRequest) (*SetSessionModeResponse, error) {
	return &SetSessionModeResponse{}, nil
}
func (floodHandler) OnSetSessionConfigOption(context.Context, SetSessionConfigOptionRequest) (*SetSessionConfigOptionResponse, error) {
	return &SetSessionConfigOptionResponse{}, nil
}
func (floodHandler) OnLogout(context.Context, LogoutRequest) (*LogoutResponse, error) {
	return &LogoutResponse{}, nil
}
func (floodHandler) OnAuthenticate(context.Context, AuthenticateRequest) (*AuthenticateResponse, error) {
	return &AuthenticateResponse{}, nil
}

func (h *floodHandler) OnCancel(context.Context, SessionId) error { return nil }

func (h *floodHandler) OnPrompt(ctx context.Context, req PromptRequest, sender SessionEventSender) (*PromptResponse, error) {
	// Many concurrent producers, each flooding notifications — mirrors
	// runner.executeTools launching a goroutine per tool call that all
	// stream progress/outputs through the same sender.
	const goroutines, perG = 32, 200
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				// Must NEVER block, even when the stdout reader is slow.
				_ = sender.SendAgentMessage("chunk")
			}
		}()
	}
	wg.Wait()
	if h.promptDone != nil {
		close(h.promptDone)
	}
	return &PromptResponse{StopReason: StopReasonEndTurn}, nil
}

// TestPromptDoesNotDeadlockOnSlowReader drives a prompt turn whose
// notification volume vastly exceeds the slow reader's drain rate and
// asserts the handler returns promptly. The old transport would have
// blocked the producers on the pipe-backed mutex and timed out.
func TestPromptDoesNotDeadlockOnSlowReader(t *testing.T) {
	h := &floodHandler{promptDone: make(chan struct{})}
	srv := NewServer("test", "0", h)

	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	slow := &slowWriter{w: stdoutW, delay: 2 * time.Millisecond}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	go func() { _ = srv.RunTransport(ctx, slow, stdinR) }()

	// Drive a session/prompt request on stdin (the fake handler doesn't
	// require a real session; mux.handlePrompt calls OnPrompt directly).
	go func() {
		defer stdinW.Close()
		req := `{"jsonrpc":"2.0","id":"1","method":"session/prompt","params":{"sessionId":"s1","prompt":[{"type":"text","text":"hi"}]}}` + "\n"
		_, _ = io.WriteString(stdinW, req)
	}()

	// Slowly drain stdout in the background — slower than production.
	go func() {
		defer stdoutR.Close()
		b := make([]byte, 32*1024)
		for {
			if _, err := stdoutR.Read(b); err != nil {
				return
			}
		}
	}()

	// OnPrompt must return well before the overall 15s budget: producers
	// enqueue into an unbounded queue instead of blocking on the pipe.
	select {
	case <-h.promptDone:
		// success — turn completed without producer back-pressure
	case <-time.After(10 * time.Second):
		t.Fatal("deadlock: OnPrompt producers blocked on slow stdout reader")
	}

	// Allow the writer to drain remaining frames and the transport to
	// unwind; cancel the ctx so serve() returns cleanly.
	cancel()
}

// TestOrderingNotificationsBeforeResponse asserts the single FIFO queue
// preserves the invariant that all turn notifications reach the client
// before the final session/prompt response frame.
//
// The test captures all stdout into an in-memory buffer (no back-pressure)
// and, once OnPrompt has completed, cancels the transport and waits for it
// to unwind; the writer must have flushed the queued response before
// serve() returns, so the buffer ends with the response following every
// notification.
func TestOrderingNotificationsBeforeResponse(t *testing.T) {
	h := &floodHandler{promptDone: make(chan struct{})}
	srv := NewServer("test", "0", h)

	stdinR, stdinW := io.Pipe()

	var out muBuffer // thread-safe bytes capture
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	runDone := make(chan struct{})
	go func() {
		_ = srv.RunTransport(ctx, &out, stdinR)
		close(runDone)
	}()

	go func() {
		defer stdinW.Close()
		req := `{"jsonrpc":"2.0","id":"1","method":"session/prompt","params":{"sessionId":"s1","prompt":[{"type":"text","text":"hi"}]}}` + "\n"
		_, _ = io.WriteString(stdinW, req)
	}()

	// h.promptDone signals that OnPrompt returned. handlePrompt (running
	// in its own goroutine) then calls writeResult, enqueuing the final
	// response frame. Give that a moment to land before unwinding.
	select {
	case <-h.promptDone:
	case <-time.After(10 * time.Second):
		cancel()
		t.Fatal("deadlock: OnPrompt producers blocked")
	}
	// Let handlePrompt enqueue the response; poll the captured buffer.
	deadline := time.Now().Add(3 * time.Second)
	for {
		if bytesContainResponse(out.bytes(), "1") {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("response frame never flushed; captured %d bytes", len(out.bytes()))
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	<-runDone

	// Parse the captured stdout line by line in arrival order.
	var sawResponse bool
	for _, line := range splitLines(out.bytes()) {
		if len(line) == 0 {
			continue
		}
		var msg jsonrpcMessage
		if json.Unmarshal(line, &msg) != nil {
			continue
		}
		if idString(msg.ID) == "1" && msg.Method == "" {
			sawResponse = true
			continue
		}
		if msg.Method == "session/update" && sawResponse {
			t.Fatal("ordering violation: notification after response frame")
		}
	}
	if !sawResponse {
		t.Fatal("missing session/prompt response frame")
	}
}

// muBuffer is a goroutine-safe []byte sink implementing io.Writer.
type muBuffer struct {
	mu sync.Mutex
	b  []byte
}

func (m *muBuffer) Write(p []byte) (int, error) {
	m.mu.Lock()
	m.b = append(m.b, p...)
	m.mu.Unlock()
	return len(p), nil
}

func (m *muBuffer) bytes() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]byte, len(m.b))
	copy(out, m.b)
	return out
}

// splitLines splits newline-delimited frames, trimming the trailing '\n'.
func splitLines(b []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, c := range b {
		if c == '\n' {
			lines = append(lines, b[start:i])
			start = i + 1
		}
	}
	if start < len(b) {
		lines = append(lines, b[start:])
	}
	return lines
}

// bytesContainResponse reports whether b holds the JSON-RPC response
// frame with the given id (a message with that id and no method).
func bytesContainResponse(b []byte, wantID string) bool {
	for _, line := range splitLines(b) {
		if len(line) == 0 {
			continue
		}
		var msg jsonrpcMessage
		if json.Unmarshal(line, &msg) != nil {
			continue
		}
		if msg.Method == "" && idString(msg.ID) == wantID {
			return true
		}
	}
	return false
}
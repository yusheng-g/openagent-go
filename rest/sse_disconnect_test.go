package rest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
)

// ── test models ──

// blockingModel hangs in ChatCompletion until ctx is cancelled, then returns
// ctx.Err(). Records whether it exited via ctx cancel — the behaviour we want
// on client SSE disconnect.
type blockingModel struct {
	ctxCancelled atomic.Bool
}

func (m *blockingModel) ChatCompletion(ctx context.Context, req openagent.ChatCompletionRequest) (*openagent.ChatCompletionResponse, error) {
	<-ctx.Done()
	m.ctxCancelled.Store(true)
	return nil, ctx.Err()
}

func (m *blockingModel) ChatCompletionStream(ctx context.Context, req openagent.ChatCompletionRequest) (openagent.StreamReader, error) {
	<-ctx.Done()
	m.ctxCancelled.Store(true)
	return nil, ctx.Err()
}

func (m *blockingModel) ContextWindow() int { return 128_000 }

// promptModel returns a fixed completion after a short delay. It records how
// it exited: 0 = still running, 1 = returned normally, 2 = ctx cancelled.
type promptModel struct {
	exit atomic.Int32 // 0 pending, 1 normal, 2 cancelled
}

func (m *promptModel) ChatCompletion(ctx context.Context, req openagent.ChatCompletionRequest) (*openagent.ChatCompletionResponse, error) {
	// Race a tiny delay against ctx cancel so we can distinguish "returned
	// normally" from "cancelled prematurely".
	select {
	case <-time.After(50 * time.Millisecond):
		m.exit.Store(1)
		return &openagent.ChatCompletionResponse{
			Choices: []openagent.Choice{{
				Message:      openagent.Message{Role: openagent.RoleAssistant, Content: "done"},
				FinishReason: "stop",
			}},
		}, nil
	case <-ctx.Done():
		m.exit.Store(2)
		return nil, ctx.Err()
	}
}

func (m *promptModel) ChatCompletionStream(ctx context.Context, req openagent.ChatCompletionRequest) (openagent.StreamReader, error) {
	return nil, nil
}

func (m *promptModel) ContextWindow() int { return 128_000 }

// ── helpers ──

// newServerWithAgent wires a Handler for the given agent onto an httptest server.
func newServerWithAgent(t *testing.T, agent *openagent.Agent) *httptest.Server {
	t.Helper()
	h := NewHandler(agent)
	mux := http.NewServeMux()
	h.Register(mux)
	s := httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return s
}

// createSession POSTs /sessions and returns the new session id.
func createSession(t *testing.T, server *httptest.Server) string {
	t.Helper()
	req, _ := http.NewRequest("POST", server.URL+"/sessions", strings.NewReader(`{"title":"t"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	var info struct {
		ID string `json:"sessionId"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&info); err != nil {
		t.Fatalf("decode create resp: %v", err)
	}
	resp.Body.Close()
	return info.ID
}

// ── tests ──

// TestSSEDisconnectCancelsAgent verifies that closing the SSE client connection
// cancels the agent's run context, so the model call stops instead of running
// to the 5-minute timeout on a detached context.Background (credit leak).
func TestSSEDisconnectCancelsAgent(t *testing.T) {
	m := &blockingModel{}
	agent := openagent.NewAgent("assistant", openagent.WithModel(m), openagent.WithMaxTurns(1))
	server := newServerWithAgent(t, agent)
	sid := createSession(t, server)

	// Open the SSE chat stream and close it mid-stream.
	req, _ := http.NewRequest("POST", server.URL+"/sessions/"+sid+"/chat", strings.NewReader(`{"message":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	buf := make([]byte, 1)
	_, _ = resp.Body.Read(buf) // ensure the agent goroutine has started
	resp.Body.Close()          // simulate client disconnect

	// The model call should see ctx cancel within a short window. Before the
	// fix (context.Background-derived ctx) it would hang until the 5-minute
	// timeout — this wait would time out.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if m.ctxCancelled.Load() {
			return // pass
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("model ChatCompletion was not cancelled after client disconnect — " +
		"agent ctx is not derived from request ctx (credit leak on disconnect)")
}

// TestSSENormalCompletionNotPrematurelyCancelled verifies that deriving the
// agent ctx from r.Context() does NOT cancel the agent prematurely on the
// normal-completion path: the model returns its response (exit==1), not a
// ctx-cancel error (exit==2). Guards against the regression where a too-eager
// r.Context cancellation would kill the agent before it finishes.
func TestSSENormalCompletionNotPrematurelyCancelled(t *testing.T) {
	m := &promptModel{}
	agent := openagent.NewAgent("assistant", openagent.WithModel(m), openagent.WithMaxTurns(1))
	server := newServerWithAgent(t, agent)
	sid := createSession(t, server)

	// Run the chat to completion (drain the full SSE stream until server closes).
	req, _ := http.NewRequest("POST", server.URL+"/sessions/"+sid+"/chat", strings.NewReader(`{"message":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	_, _ = io.ReadAll(resp.Body)
	resp.Body.Close()

	if got := m.exit.Load(); got != 1 {
		t.Fatalf("model exit = %d, want 1 (normal); the r.Context-derived ctx "+
			"cancelled the agent before it finished", got)
	}
}

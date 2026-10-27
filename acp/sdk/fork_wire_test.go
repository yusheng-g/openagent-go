package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"
)

// blockingPromptHandler blocks inside OnPrompt until released, so the
// per-session prompt lock stays held — the state a fork must reject.
type blockingPromptHandler struct {
	floodHandler
	started chan struct{}
	release chan struct{}
}

func (h *blockingPromptHandler) OnPrompt(ctx context.Context, req PromptRequest, sender SessionEventSender) (*PromptResponse, error) {
	close(h.started)
	<-h.release
	return &PromptResponse{StopReason: StopReasonEndTurn}, nil
}

// startForkServer runs a server over in-memory pipes and returns its
// stdin writer and a channel of raw stdout lines.
func startForkServer(t *testing.T, handler AgentHandler) (*io.PipeWriter, chan string) {
	t.Helper()
	srv := NewServer("fork-wire-test", "1.0.0", handler)
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(func() {
		cancel()
		_ = stdinW.Close()
	})
	go func() { _ = srv.RunTransport(ctx, stdoutW, stdinR) }()

	lines := make(chan string, 64)
	go func() {
		defer stdoutR.Close()
		sc := bufio.NewScanner(stdoutR)
		for sc.Scan() {
			lines <- sc.Text()
		}
	}()
	return stdinW, lines
}

// readResponse scans stdout lines until the response with the given id
// arrives; notifications (no id) are skipped.
func readResponse(t *testing.T, lines chan string, id string) map[string]any {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case line := <-lines:
			var m map[string]any
			if err := json.Unmarshal([]byte(line), &m); err != nil {
				continue
			}
			if fmt.Sprint(m["id"]) == id {
				return m
			}
		case <-deadline:
			t.Fatalf("no response with id %s within timeout", id)
		}
	}
}

// TestForkSessionWireRoundtrip drives a session/fork request through
// RunTransport and checks the client-visible response.
func TestForkSessionWireRoundtrip(t *testing.T) {
	stdin, lines := startForkServer(t, &floodHandler{})

	go func() {
		_, _ = fmt.Fprintln(stdin, `{"jsonrpc":"2.0","id":"1","method":"session/fork","params":{"sessionId":"s1"}}`)
	}()

	got := readResponse(t, lines, "1")
	if got["error"] != nil {
		t.Fatalf("unexpected error: %v", got["error"])
	}
	res, ok := got["result"].(map[string]any)
	if !ok {
		t.Fatalf("result = %#v", got["result"])
	}
	if res["sessionId"] != "fork1" {
		t.Fatalf("sessionId = %v, want fork1", res["sessionId"])
	}
}

// TestForkSessionWireBusy verifies a fork is rejected while the source
// session is mid-turn: the prompt holds the per-session lock, and the
// fork must fail fast with ErrorCodeSessionBusy instead of queueing.
func TestForkSessionWireBusy(t *testing.T) {
	h := &blockingPromptHandler{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	stdin, lines := startForkServer(t, h)

	go func() {
		_, _ = fmt.Fprintln(stdin, `{"jsonrpc":"2.0","id":"1","method":"session/prompt","params":{"sessionId":"s1","prompt":[{"type":"text","text":"hi"}]}}`)
	}()
	select {
	case <-h.started:
	case <-time.After(5 * time.Second):
		t.Fatal("prompt did not start within timeout")
	}

	go func() {
		_, _ = fmt.Fprintln(stdin, `{"jsonrpc":"2.0","id":"2","method":"session/fork","params":{"sessionId":"s1"}}`)
	}()

	got := readResponse(t, lines, "2")
	errObj, ok := got["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected busy error, got %#v", got)
	}
	if code, _ := errObj["code"].(float64); int(code) != ErrorCodeSessionBusy {
		t.Fatalf("error code = %v, want %d (session busy)", errObj["code"], ErrorCodeSessionBusy)
	}

	close(h.release) // let the prompt turn finish and the test unwind
}

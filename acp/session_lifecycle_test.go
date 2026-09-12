package acp

import (
	"context"
	"sync"
	"testing"

	openagent "github.com/yusheng-g/openagent-go"
	openacp "github.com/yusheng-g/openagent-go/acp/sdk"

	"github.com/yusheng-g/openagent-go/agent"
	"github.com/yusheng-g/openagent-go/kernel"
)

// fakeSessionObserver captures lifecycle events for testing.
type fakeSessionObserver struct {
	mu      sync.Mutex
	creates []openagent.SessionLifecycleEvent
	closes  []openagent.SessionLifecycleEvent
	deletes []openagent.SessionLifecycleEvent
}

func (f *fakeSessionObserver) OnSessionCreate(_ context.Context, e openagent.SessionLifecycleEvent) {
	f.mu.Lock()
	f.creates = append(f.creates, e)
	f.mu.Unlock()
}

func (f *fakeSessionObserver) OnSessionClose(_ context.Context, e openagent.SessionLifecycleEvent) {
	f.mu.Lock()
	f.closes = append(f.closes, e)
	f.mu.Unlock()
}

func (f *fakeSessionObserver) OnSessionDelete(_ context.Context, e openagent.SessionLifecycleEvent) {
	f.mu.Lock()
	f.deletes = append(f.deletes, e)
	f.mu.Unlock()
}

func newTestServerWithObserver() (*AgentServer, *fakeSessionObserver) {
	srv := NewAgentServer(agent.New("test"), kernel.Deps{}, nil, nil)
	fake := &fakeSessionObserver{}
	srv.SetSessionObservers(fake)
	return srv, fake
}

func TestOnNewSession_EmitsCreateEvent(t *testing.T) {
	srv, fake := newTestServerWithObserver()

	resp, err := srv.OnNewSession(context.Background(), openacp.NewSessionRequest{Cwd: "/tmp"})
	if err != nil {
		t.Fatalf("OnNewSession: %v", err)
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.creates) != 1 {
		t.Fatalf("expected 1 create event, got %d", len(fake.creates))
	}
	evt := fake.creates[0]
	if evt.SessionID != string(resp.SessionID) {
		t.Errorf("SessionID = %q, want %q", evt.SessionID, resp.SessionID)
	}
	if evt.EntryPoint != "acp" {
		t.Errorf("EntryPoint = %q, want acp", evt.EntryPoint)
	}
	if evt.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestOnCloseSession_EmitsCloseEvent(t *testing.T) {
	srv, fake := newTestServerWithObserver()

	// Create a session first.
	resp, err := srv.OnNewSession(context.Background(), openacp.NewSessionRequest{Cwd: "/tmp"})
	if err != nil {
		t.Fatalf("OnNewSession: %v", err)
	}

	// Close it.
	_, err = srv.OnCloseSession(context.Background(), openacp.CloseSessionRequest{SessionID: resp.SessionID})
	if err != nil {
		t.Fatalf("OnCloseSession: %v", err)
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.closes) != 1 {
		t.Fatalf("expected 1 close event, got %d", len(fake.closes))
	}
	evt := fake.closes[0]
	if evt.SessionID != string(resp.SessionID) {
		t.Errorf("SessionID = %q, want %q", evt.SessionID, resp.SessionID)
	}
	if evt.EntryPoint != "acp" {
		t.Errorf("EntryPoint = %q, want acp", evt.EntryPoint)
	}
	if evt.DurationMs < 0 {
		t.Error("DurationMs should be non-negative for a session that was created")
	}
}

func TestOnDeleteSession_EmitsDeleteEvent(t *testing.T) {
	srv, fake := newTestServerWithObserver()

	// Create a session first.
	resp, err := srv.OnNewSession(context.Background(), openacp.NewSessionRequest{Cwd: "/tmp"})
	if err != nil {
		t.Fatalf("OnNewSession: %v", err)
	}

	// Delete it.
	_, err = srv.OnDeleteSession(context.Background(), openacp.DeleteSessionRequest{SessionID: resp.SessionID})
	if err != nil {
		t.Fatalf("OnDeleteSession: %v", err)
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.deletes) != 1 {
		t.Fatalf("expected 1 delete event, got %d", len(fake.deletes))
	}
	evt := fake.deletes[0]
	if evt.SessionID != string(resp.SessionID) {
		t.Errorf("SessionID = %q, want %q", evt.SessionID, resp.SessionID)
	}
	if evt.EntryPoint != "acp" {
		t.Errorf("EntryPoint = %q, want acp", evt.EntryPoint)
	}
}

func TestSessionObserver_NilIsNoOp(t *testing.T) {
	// No observer set — should not panic.
	srv := NewAgentServer(agent.New("test"), kernel.Deps{}, nil, nil)
	// srv.sessionObservers is nil.

	_, err := srv.OnNewSession(context.Background(), openacp.NewSessionRequest{Cwd: "/tmp"})
	if err != nil {
		t.Fatalf("OnNewSession with nil observer: %v", err)
	}
}

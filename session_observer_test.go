package openagent

import (
	"context"
	"sync"
	"testing"
)

// captureSessionObserver records every lifecycle event it receives.
type captureSessionObserver struct {
	mu      sync.Mutex
	creates []SessionLifecycleEvent
	closes  []SessionLifecycleEvent
	deletes []SessionLifecycleEvent
}

func (c *captureSessionObserver) OnSessionCreate(_ context.Context, e SessionLifecycleEvent) {
	c.mu.Lock()
	c.creates = append(c.creates, e)
	c.mu.Unlock()
}

func (c *captureSessionObserver) OnSessionClose(_ context.Context, e SessionLifecycleEvent) {
	c.mu.Lock()
	c.closes = append(c.closes, e)
	c.mu.Unlock()
}

func (c *captureSessionObserver) OnSessionDelete(_ context.Context, e SessionLifecycleEvent) {
	c.mu.Lock()
	c.deletes = append(c.deletes, e)
	c.mu.Unlock()
}

func (c *captureSessionObserver) createCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.creates)
}

func TestMultiSessionObserver_NilFiltering(t *testing.T) {
	// All nil → returns nil (caller can nil-check once).
	obs := MultiSessionObserver(nil, nil)
	if obs != nil {
		t.Fatal("MultiSessionObserver(nil, nil) should return nil")
	}
}

func TestMultiSessionObserver_SingleFlatten(t *testing.T) {
	// A single non-nil observer should be returned directly (not wrapped).
	c := &captureSessionObserver{}
	obs := MultiSessionObserver(nil, c, nil)
	if obs != c {
		t.Fatal("MultiSessionObserver with one non-nil should return it directly")
	}
}

func TestMultiSessionObserver_FanOut(t *testing.T) {
	a := &captureSessionObserver{}
	b := &captureSessionObserver{}
	obs := MultiSessionObserver(a, b)

	evt := SessionLifecycleEvent{SessionID: "s1", EntryPoint: "acp"}
	obs.OnSessionCreate(context.Background(), evt)

	if a.createCount() != 1 || b.createCount() != 1 {
		t.Fatalf("expected both observers to receive create; got a=%d b=%d",
			a.createCount(), b.createCount())
	}
}

func TestMultiSessionObserver_AllThreeMethods(t *testing.T) {
	c := &captureSessionObserver{}
	obs := MultiSessionObserver(c)

	obs.OnSessionCreate(context.Background(), SessionLifecycleEvent{SessionID: "s1"})
	obs.OnSessionClose(context.Background(), SessionLifecycleEvent{SessionID: "s1", DurationMs: 100})
	obs.OnSessionDelete(context.Background(), SessionLifecycleEvent{SessionID: "s1"})

	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.creates) != 1 || len(c.closes) != 1 || len(c.deletes) != 1 {
		t.Fatalf("expected 1/1/1 creates/closes/deletes; got %d/%d/%d",
			len(c.creates), len(c.closes), len(c.deletes))
	}
}

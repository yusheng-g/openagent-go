package openagent

import (
	"context"
	"time"
)

// SessionLifecycleEvent carries session-level lifecycle metadata.
// It is emitted by the ACP/REST/CLI servers at session create / close /
// delete boundaries and consumed by SessionObserver implementations
// (e.g. track.Observer for HTTP event reporting).
//
// This mirrors the RunObserver / StageEvent pattern but operates one level
// up: RunObserver observes stages *inside* a run loop; SessionObserver
// observes the session lifecycle *around* runs.
type SessionLifecycleEvent struct {
	SessionID   string    // session ID from Session.ID
	EntryPoint  string    // "acp" | "rest" | "cli"
	SessionMode string    // "manual" | "plan" | "auto" (empty if unknown)
	DurationMs  int64     // milliseconds since session create (close/delete only)
	Err         error     // non-nil if the session ended with an error
	CreatedAt   time.Time // session creation time (for duration calculation)
}

// SessionObserver receives session lifecycle events.  nil SessionObserver =
// events are silently dropped — callers MUST nil-check before invoking.
//
// This is a plain Go interface (no WASM, no plugin loader), compiled
// statically — same pattern as RunObserver.  Implementations include
// track.Observer (HTTP event reporting); future implementations may cover
// OpenTelemetry traces or agent evaluation hooks.
//
// Concurrency contract: implementations MUST be safe for concurrent use.
// OnCloseSession and OnDeleteSession may be called from different goroutines
// than OnNewSession (e.g. ACP session/delete handler vs. the original
// session/new handler goroutine).
type SessionObserver interface {
	OnSessionCreate(ctx context.Context, event SessionLifecycleEvent)
	OnSessionClose(ctx context.Context, event SessionLifecycleEvent)
	OnSessionDelete(ctx context.Context, event SessionLifecycleEvent)
}

// MultiSessionObserver combines multiple SessionObservers into one.
// Each observer is called in order; nil observers are skipped.  Returns nil
// when no observers remain after filtering, so the caller can store the
// result directly and nil-check once.
func MultiSessionObserver(observers ...SessionObserver) SessionObserver {
	var filtered []SessionObserver
	for _, o := range observers {
		if o != nil {
			filtered = append(filtered, o)
		}
	}
	switch len(filtered) {
	case 0:
		return nil
	case 1:
		return filtered[0]
	default:
		return &multiSessionObserver{list: filtered}
	}
}

type multiSessionObserver struct {
	list []SessionObserver
}

func (m *multiSessionObserver) OnSessionCreate(ctx context.Context, event SessionLifecycleEvent) {
	for _, o := range m.list {
		o.OnSessionCreate(ctx, event)
	}
}

func (m *multiSessionObserver) OnSessionClose(ctx context.Context, event SessionLifecycleEvent) {
	for _, o := range m.list {
		o.OnSessionClose(ctx, event)
	}
}

func (m *multiSessionObserver) OnSessionDelete(ctx context.Context, event SessionLifecycleEvent) {
	for _, o := range m.list {
		o.OnSessionDelete(ctx, event)
	}
}

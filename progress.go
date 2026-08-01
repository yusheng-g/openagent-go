package openagent

import "context"

// ProgressFunc reports progress to the caller during a long-running tool call.
//
// message is a human-readable status string (e.g. "Running terraform plan...").
// progress is the progress thus far; total is the total number of items or
// steps (0 means unknown). progress should increase monotonically.
//
// A nil ProgressFunc is a no-op — callers can invoke it unconditionally.
// Implementations must be safe to call from any goroutine that holds the
// context.
type ProgressFunc func(message string, progress, total float64)

type progressKey struct{}

// WithProgress returns a context that carries a [ProgressFunc].
// Tools retrieve it via [ProgressFromContext] and call it at key milestones.
// A nil f produces a context whose ProgressFromContext returns nil, so
// callers outside an MCP server (e.g. unit tests) are unaffected.
func WithProgress(ctx context.Context, f ProgressFunc) context.Context {
	return context.WithValue(ctx, progressKey{}, f)
}

// ProgressFromContext extracts the [ProgressFunc] from ctx, or nil if none
// was set. The nil ProgressFunc is safe to call — it is a no-op.
func ProgressFromContext(ctx context.Context) ProgressFunc {
	f, _ := ctx.Value(progressKey{}).(ProgressFunc)
	return f
}

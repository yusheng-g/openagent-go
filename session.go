package openagent

import (
	"context"
	"time"
)

// sessionCtxKey is the context key for Session propagation.
// Unexported to prevent external packages from colliding.
type sessionCtxKey struct{}

// WithSession returns a copy of ctx with the session attached.
// Tool implementations can extract it via SessionFromContext.
//
// The runner injects the session before calling Tool.Execute so that
// tools that need user/project context can access it without changing
// the Tool interface. Tools that don't need it are unaffected.
func WithSession(ctx context.Context, s Session) context.Context {
	return context.WithValue(ctx, sessionCtxKey{}, s)
}

// SessionFromContext extracts the Session that the runner injected into ctx.
// Returns ok=false if no session was set (e.g. when the tool is called
// outside the runner, or in tests).
func SessionFromContext(ctx context.Context) (Session, bool) {
	s, ok := ctx.Value(sessionCtxKey{}).(Session)
	return s, ok
}

// SessionStatus is the lifecycle state of a session.
type SessionStatus string

const (
	SessionActive    SessionStatus = "active"
	SessionDone      SessionStatus = "done"
	SessionCancelled SessionStatus = "cancelled"
	SessionError     SessionStatus = "error"
)

// Session carries the identity and context for a conversation thread.
// It is a plain data struct — the application layer owns session CRUD.
// openagent-go only reads from it during a run.
type Session struct {
	// Identity
	ID     string `json:"id"`
	UserID string `json:"user_id"`

	// Model selection (overrides Agent default if set)
	ModelID     string   `json:"model_id,omitempty"`
	Provider    string   `json:"provider,omitempty"`
	Model       Model    `json:"-"` // per-request model override; nil = use Agent.Model
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   int      `json:"max_tokens,omitempty"`

	// Context for Prompt
	UserProfile    string `json:"user_profile,omitempty"`
	ProjectContext string `json:"project_context,omitempty"`

	// DynamicContext is injected into the system prompt every turn.
	// It carries session-level state that changes between turns:
	// plan entries with status, active mode instructions, etc.
	// The ACP layer constructs it; the Runner simply passes it through.
	DynamicContext string `json:"-"`

	// Lifecycle
	Status    SessionStatus `json:"status"`
	Turn      int           `json:"turn"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`

	// Extension
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Package slash provides a registry for server-side slash commands.
// Commands are registered at startup and dispatched at the top of
// OnPrompt before the text reaches the LLM agent.
//
// Each command has a name, description, input hint, and a handler
// function that takes session context and raw args, returning a
// response string. If no command matches, the text falls through
// to normal agent processing.
//
// Usage from acp/server.go:
//
//	r := slash.NewRegistry()
//	r.Register("help", "Show available commands", nil,
//	    func(ctx slash.Context, args string) (string, error) {
//	        return "...", nil
//	    })
//
//	// In OnPrompt:
//	if resp, ok := s.reg.Handle(ctx, input.Content); ok {
//	    sender.SendAgentMessage(resp)
//	    return &PromptResponse{StopReason: EndTurn}, nil
//	}
package slash

import (
	"fmt"
	"strings"
	"time"
)

// ── Types ──

// InputHint describes the expected argument for a command.
type InputHint struct {
	Hint string
}

// Command is a registered slash command.
type Command struct {
	Name        string
	Description string
	Input       *InputHint
	Handler     Handler
}

// Handler receives the parsed context and raw argument string (after the
// command name). It returns the response text to send to the client.
type Handler func(ctx Context, args string) (string, error)

// Context is a read-only snapshot of the current session, plus callbacks
// so command handlers can mutate session state without importing
// server-level packages.
type Context struct {
	// Session identity.
	SessionID string
	Cwd       string
	Mode      string

	// Runtime stats.
	TotalTokens int

	// Timestamps.
	CreatedAt time.Time

	// ── Callbacks for state mutation ──
	// The server wires these to AgentServer methods.

	SetMode  func(mode string) error       // persists + sends current_mode_update notification
	Rename   func(title string) error      // persists + sends session_info_update notification
	Clear    func() error                  // deletes all messages for this session
	ListSess func() ([]SessionInfo, error) // returns all sessions from the store
}

// SessionInfo is a summary returned by /sessions.
type SessionInfo struct {
	ID        string
	Cwd       string
	Title     string
	UpdatedAt string
}

// ── Registry ──

// Registry holds registered slash commands and dispatches incoming text.
type Registry struct {
	cmds   []Command           // ordered list preserves insertion order
	byName map[string]int      // name → index in cmds
}

// NewRegistry creates an empty command registry.
func NewRegistry() *Registry {
	return &Registry{byName: make(map[string]int)}
}

// Register adds a command. If a command with the same name already
// exists, it is replaced in-place (preserving insertion order).
func (r *Registry) Register(name, description string, input *InputHint, h Handler) {
	name = strings.TrimPrefix(name, "/")
	cmd := Command{Name: name, Description: description, Input: input, Handler: h}
	if idx, ok := r.byName[name]; ok {
		r.cmds[idx] = cmd
		return
	}
	r.byName[name] = len(r.cmds)
	r.cmds = append(r.cmds, cmd)
}

// Available returns the command list in insertion order.
func (r *Registry) Available() []Command {
	out := make([]Command, len(r.cmds))
	copy(out, r.cmds)
	return out
}

// Handle parses text. If text starts with "/" and matches a registered
// command, the handler is called and returns the response with true.
// Otherwise returns ("", false).
func (r *Registry) Handle(ctx Context, text string) (string, bool) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return "", false
	}

	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", false
	}
	name := strings.TrimPrefix(fields[0], "/")
	args := ""
	if len(fields) > 1 {
		args = strings.Join(fields[1:], " ")
	}

	idx, ok := r.byName[name]
	if !ok {
		// Unknown command — let the agent handle it.
		return "", false
	}

	resp, err := r.cmds[idx].Handler(ctx, args)
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err), true
	}
	return resp, true
}

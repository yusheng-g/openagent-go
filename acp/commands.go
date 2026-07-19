package acp

import (
	"fmt"

	"github.com/yusheng-g/openagent-go/slash"
)

// buildCommandRegistry creates the slash command registry with all built-in
// commands. The server provides the callbacks by closing over AgentServer.
func (s *AgentServer) buildCommandRegistry() *slash.Registry {
	r := slash.NewRegistry()

	r.Register("help", "Show available commands and capabilities", nil,
		func(ctx slash.Context, args string) (string, error) {
			var out string
			out += "Built-in commands:\n\n"
			for _, c := range r.Available() {
				line := "  /" + c.Name + " — " + c.Description
				if c.Input != nil {
					line += " (e.g. /" + c.Name + " " + c.Input.Hint + ")"
				}
				out += line + "\n"
			}
			return out, nil
		})

	r.Register("mode", "Switch session mode", &slash.InputHint{Hint: "chat|plan"},
		func(ctx slash.Context, args string) (string, error) {
			switch args {
			case "chat", "plan":
				if err := ctx.SetMode(args); err != nil {
					return "", err
				}
				return "Switched to " + args + " mode.\n", nil
			default:
				return "Usage: /mode chat|plan (current: " + ctx.Mode + ")\n", nil
			}
		})

	r.Register("context", "Show context window usage", nil,
		func(ctx slash.Context, _ string) (string, error) {
			return "Context window: " + fmt.Sprintf("%d", ctx.TotalTokens) + " total tokens used.\n", nil
		})

	r.Register("cwd", "Show current working directory", nil,
		func(ctx slash.Context, _ string) (string, error) {
			return "Working directory: " + ctx.Cwd + "\n", nil
		})

	r.Register("clear", "Clear all session messages", nil,
		func(ctx slash.Context, _ string) (string, error) {
			if err := ctx.Clear(); err != nil {
				return "", err
			}
			return "Session cleared. All messages deleted.\n", nil
		})

	r.Register("rename", "Rename the current session", &slash.InputHint{Hint: "new title"},
		func(ctx slash.Context, args string) (string, error) {
			if args == "" {
				return "Usage: /rename <new title>\n", nil
			}
			if err := ctx.Rename(args); err != nil {
				return "", err
			}
			return "Session renamed to: " + args + "\n", nil
		})

	r.Register("sessions", "List all sessions", nil,
		func(ctx slash.Context, _ string) (string, error) {
			sessions, err := ctx.ListSess()
			if err != nil {
				return "", err
			}
			if len(sessions) == 0 {
				return "No sessions found.\n", nil
			}
			var out string
			current := ctx.SessionID
			for _, si := range sessions {
				marker := " "
				if si.ID == current {
					marker = "*"
				}
				title := si.Title
				if title == "" {
					title = "(untitled)"
				}
				out += marker + " " + si.ID + "  " + si.Cwd + "  " +
					title + "\n"
			}
			return out, nil
		})

	r.Register("new", "Instructions for creating a new session", &slash.InputHint{Hint: "cwd (optional)"},
		func(slash.Context, string) (string, error) {
			return "Create a new session via the client's session management UI or\n" +
				"AOP RPC method session/new.\n", nil
		})

	r.Register("switch", "Instructions for switching sessions", &slash.InputHint{Hint: "session ID"},
		func(slash.Context, string) (string, error) {
			return "Use /sessions to list available sessions, then use your\n" +
				"client's UI to switch or load a different session.\n", nil
		})

	return r
}

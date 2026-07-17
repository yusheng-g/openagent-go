// Package acp provides an abstraction over the Agent Client Protocol (ACP).
//
// It defines openagent-go's own ACP types and a Client for connecting to
// external ACP agent processes. The default implementation uses
// coder/acp-go-sdk internally, but the interfaces are designed so that
// alternative implementations can be plugged in.
//
// Import as:
//
//	openacp "github.com/yusheng-g/openagent-go/acp"
package acp

// ── Content / Prompt ──

// ContentBlock represents a piece of content in a prompt.
type ContentBlock struct {
	Text string // plain text or markdown
}

// PromptRequest is the input to an agent prompt turn.
type PromptRequest struct {
	SessionID string
	Blocks    []ContentBlock
}

// PromptResponse is the result of a prompt turn.
type PromptResponse struct {
	StopReason string // "end_turn", "cancelled", "refusal", etc.
}

// ── Session ──

// NewSessionRequest configures a new ACP session.
// See: https://agentclientprotocol.com/protocol/session-setup#creating-a-session
type NewSessionRequest struct {
	Cwd                  string      // working directory (required)
	McpServers           []McpServer // MCP servers the agent should connect to
	AdditionalDirectories []string   // additional workspace roots
}

// NewSessionResponse is the result of creating a session.
type NewSessionResponse struct {
	SessionID     string                 // unique session identifier
	ConfigOptions []SessionConfigOption  // initial config options (if supported)
	Modes         *SessionModeState      // initial mode state (if supported)
}

// LoadSessionRequest requests resuming an existing ACP session.
// See: https://agentclientprotocol.com/protocol/session-setup#loading-sessions
type LoadSessionRequest struct {
	SessionID             string      // session to resume (required)
	Cwd                   string      // working directory (required)
	McpServers            []McpServer // MCP servers to connect to
	AdditionalDirectories []string    // additional workspace roots
}

// LoadSessionResponse is the result of loading an existing session.
type LoadSessionResponse struct {
	ConfigOptions []SessionConfigOption // session config options (if supported)
	Modes         *SessionModeState     // session mode state (if supported)
}

// ListSessionsRequest requests the list of available sessions from the agent.
type ListSessionsRequest struct {
	Cursor *string // opaque cursor from previous response for pagination
	Cwd    *string // filter sessions by working directory
}

// SessionInfo describes an available ACP session.
type SessionInfo struct {
	SessionID             string   // unique session identifier
	Cwd                   string   // working directory
	Title                 string   // human-readable title
	UpdatedAt             string   // ISO 8601 timestamp of last activity
	AdditionalDirectories []string // additional workspace roots
}

// ListSessionsResponse is the result of listing available sessions.
type ListSessionsResponse struct {
	NextCursor *string        // present if there are more results
	Sessions   []SessionInfo
}

// McpServer describes an MCP server the agent should connect to.
// For HTTP/S: set URL. For stdio: set Command + Args.
type McpServer struct {
	Name    string
	URL     string   // HTTP endpoint (e.g. "http://localhost:PORT")
	Command string   // executable path (for stdio MCP servers)
	Args    []string // command arguments
}

// SessionConfigOption describes a configuration option presented to the user.
// Type is "select" or "boolean"; for "select", Options is populated.
type SessionConfigOption struct {
	Type         string                   `json:"type"`
	Name         string                   `json:"name,omitempty"`
	Label        string                   `json:"label,omitempty"`
	CurrentValue string                   `json:"currentValue,omitempty"`
	Options      []SessionConfigOptValue  `json:"options,omitempty"`
}

// SessionConfigOptValue is a single option value in a select config.
type SessionConfigOptValue struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// SessionModeState reports the agent's available modes and current mode.
type SessionModeState struct {
	AvailableModes []SessionMode // all modes the agent supports
	CurrentModeID  string        // currently active mode ID
}

// SessionMode describes a single agent mode.
type SessionMode struct {
	ID          string // unique mode identifier
	Name        string // human-readable name
	Description string // optional description
}

// ── Capabilities ──

// AgentCapabilities describes what the agent supports.
// Reported by the agent during [InitializeResponse].
type AgentCapabilities struct {
	LoadSession         bool                // supports session/resume
	McpCapabilities     McpCapabilities     // MCP transport support
	PromptCapabilities  PromptCapabilities  // prompt features
	SessionCapabilities SessionCapabilities // session methods
}

// McpCapabilities describes MCP transport methods the agent supports.
type McpCapabilities struct {
	Acp  bool // ACP-native transport
	Http bool // StreamableHTTP transport
	Sse  bool // SSE transport
}

// PromptCapabilities describes the prompt features an agent supports.
type PromptCapabilities struct {
	Image           bool // can process image inputs
	Audio           bool // can process audio inputs
	EmbeddedContext bool // can process embedded context blocks
}

// SessionCapabilities describes which session methods an agent supports.
type SessionCapabilities struct {
	List   bool // supports session/list
	Delete bool // supports session/delete
	Resume bool // supports session/resume
	Close  bool // supports session/close
}

// ClientCapabilities describes what the client supports.
// Reported by the client during [InitializeRequest].
type ClientCapabilities struct {
	Terminal bool // client supports terminal
}

// ── Initialize ──

// InitializeRequest is the ACP handshake request.
type InitializeRequest struct {
	ProtocolVersion    int
	ClientName         string
	ClientVersion      string
	ClientCapabilities ClientCapabilities
}

// InitializeResponse is the ACP handshake response.
type InitializeResponse struct {
	ProtocolVersion int
	AgentName       string
	AgentVersion    string
	Capabilities    AgentCapabilities
}

// ── Plan / Usage / Commands ──

// PlanEntry represents a single task in the agent's execution plan.
type PlanEntry struct {
	Title    string // human-readable task description
	Priority string // "high", "medium", "low"
	Status   string // "pending", "in_progress", "completed", "cancelled"
}

// UsageInfo reports token and context window usage for a session.
type UsageInfo struct {
	Used int // tokens currently in context
	Size int // total context window size in tokens
}

// AvailableCommand describes a command the agent can execute (e.g. /help, /clear).
type AvailableCommand struct {
	Name        string
	Description string
}

// ── Event handler ──

// EventHandler receives streaming events from the agent during Prompt.
// Implement this interface and register it via Session.SetEventHandler
// to receive real-time updates.
type EventHandler interface {
	OnAgentMessage(text string)
	OnAgentThought(text string)
	OnToolCall(tc ToolCallEvent)
}

// PlanHandler is an optional extension of [EventHandler]. Implement it
// to receive plan updates from the agent.
type PlanHandler interface {
	OnPlan(entries []PlanEntry)
}

// UsageHandler is an optional extension of [EventHandler]. Implement it
// to receive token usage information.
type UsageHandler interface {
	OnUsage(info UsageInfo)
}

// CommandHandler is an optional extension of [EventHandler]. Implement it
// to receive available command listings from the agent.
type CommandHandler interface {
	OnAvailableCommands(commands []AvailableCommand)
}

// ModeHandler is an optional extension of [EventHandler]. Implement it
// to receive mode changes and user message echoes.
type ModeHandler interface {
	OnCurrentMode(modeID string)
	OnUserMessage(text string)
}

// ToolCallEvent represents a tool invocation by the agent.
type ToolCallEvent struct {
	ID        string
	Title     string
	RawInput  any    // parameters sent to the tool
	Status    string // "in_progress", "completed", "failed"
	RawOutput any    // tool result (when Status == "completed")
}

// ── Config / Fork ──

// SetConfigOption describes a session configuration option change.
type SetConfigOption struct {
	ID    string // config option ID
	Value any    // bool for boolean options, string for select valueID
}

// ForkSessionRequest requests forking an existing session.
type ForkSessionRequest struct {
	SessionID string
}

// ForkSessionResponse is the result of forking a session.
type ForkSessionResponse struct {
	SessionID string
}

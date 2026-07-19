// Package acp provides the Agent Client Protocol types and abstractions.
//
// Types follow the ACP v1 spec:
//
//	https://agentclientprotocol.com/protocol/v1/schema
//
// Discriminated unions (McpServer, ContentBlock, SessionUpdate, etc.) use
// a Type field for routing; variants omit mutually exclusive fields.
//
// All types carry an optional _meta field for extensibility.
package acp

import (
	"context"
	"encoding/json"
)

// ── Core identity types ──

// SessionId is an opaque session identifier assigned by the Agent.
type SessionId = string

// MessageId is an opaque message identifier for grouping content chunks.
type MessageId = string

// TerminalId is an opaque terminal identifier assigned by the Client.
type TerminalId = string

// RequestId correlates requests to responses (JSON-RPC id).
type RequestId = string

// ProtocolVersion is the MAJOR protocol version as an integer.
type ProtocolVersion = int

// ── Primitive alias types ──

// SessionConfigId is a config option identifier.
type SessionConfigId = string

// SessionConfigValueId is a config option value identifier (select type).
type SessionConfigValueId = string

// SessionModeId is a session mode identifier.
type SessionModeId = string

// PermissionOptionId identifies a permission choice.
type PermissionOptionId = string

// AuthMethodId identifies an authentication method.
type AuthMethodId = string

// ── Implementation ──

// Implementation describes an Agent or Client implementation.
type Implementation struct {
	Meta    map[string]any `json:"_meta,omitempty"`
	Name    string         `json:"name"`
	Title   string         `json:"title,omitempty"`
	Version string         `json:"version"`
}

// ── Capabilities ──

// AgentCapabilities is returned in the initialize response.
// Per ACP v1 spec, session lifecycle capabilities (close, delete, list, resume,
// additionalDirectories) live inside SessionCapabilities, not at the top level.
type AgentCapabilities struct {
	Meta map[string]any `json:"_meta,omitempty"`

	LoadSession         bool                `json:"loadSession,omitempty"`
	McpCapabilities     McpCapabilities     `json:"mcpCapabilities,omitempty"`
	PromptCapabilities  PromptCapabilities  `json:"promptCapabilities,omitempty"`
	SessionCapabilities SessionCapabilities `json:"sessionCapabilities,omitempty"`
	Auth                AgentAuthCapabilities `json:"auth,omitempty"`
}

// ClientCapabilities is sent by the Client during initialize.
type ClientCapabilities struct {
	Meta     map[string]any              `json:"_meta,omitempty"`
	FS       FileSystemCapabilities      `json:"fs,omitempty"`
	Session  *ClientSessionCapabilities  `json:"session,omitempty"`
	Terminal bool                        `json:"terminal,omitempty"`
}

// PromptCapabilities advertises rich prompt content support.
type PromptCapabilities struct {
	Meta            map[string]any `json:"_meta,omitempty"`
	Image           bool           `json:"image,omitempty"`
	Audio           bool           `json:"audio,omitempty"`
	EmbeddedContext bool           `json:"embeddedContext,omitempty"`
}

// McpCapabilities advertises MCP transport support.
type McpCapabilities struct {
	Meta map[string]any `json:"_meta,omitempty"`
	HTTP bool           `json:"http,omitempty"`
	SSE  bool           `json:"sse,omitempty"`
}

// SessionCapabilities advertises session lifecycle support.
type SessionCapabilities struct {
	Meta                   map[string]any                          `json:"_meta,omitempty"`
	Close                  *SessionCloseCapabilities               `json:"close,omitempty"`
	Delete                 *SessionDeleteCapabilities              `json:"delete,omitempty"`
	List                   *SessionListCapabilities                `json:"list,omitempty"`
	Resume                 *SessionResumeCapabilities              `json:"resume,omitempty"`
	AdditionalDirectories  *SessionAdditionalDirectoriesCapabilities `json:"additionalDirectories,omitempty"`
}

// AgentAuthCapabilities advertises auth feature support.
type AgentAuthCapabilities struct {
	Meta   map[string]any      `json:"_meta,omitempty"`
	Logout *LogoutCapabilities `json:"logout,omitempty"` // null or absent = not supported, {} = supported
}

// ClientSessionCapabilities advertises client session support.
type ClientSessionCapabilities struct {
	Meta          map[string]any                        `json:"_meta,omitempty"`
	ConfigOptions *BooleanConfigOptionCapabilities `json:"configOptions,omitempty"`
}

// BooleanConfigOptionCapabilities advertises client support for boolean config options.
type BooleanConfigOptionCapabilities struct {
	Meta    map[string]any `json:"_meta,omitempty"`
	Boolean any            `json:"boolean,omitempty"` // {} = supported
}

// FileSystemCapabilities advertises file system RPC support.
type FileSystemCapabilities struct {
	Meta         map[string]any `json:"_meta,omitempty"`
	ReadTextFile bool           `json:"readTextFile,omitempty"`
	WriteTextFile bool          `json:"writeTextFile,omitempty"`
}

// ── Capability marker types (empty struct = supported) ──

// SessionCloseCapabilities is an empty object in JSON; presence signals support.
type SessionCloseCapabilities struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

// SessionDeleteCapabilities is an empty object in JSON; presence signals support.
type SessionDeleteCapabilities struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

// SessionListCapabilities is an empty object in JSON; presence signals support.
type SessionListCapabilities struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

// SessionResumeCapabilities is an empty object in JSON; presence signals support.
type SessionResumeCapabilities struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

// SessionAdditionalDirectoriesCapabilities is an empty object in JSON.
type SessionAdditionalDirectoriesCapabilities struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

// LogoutCapabilities is an empty object in JSON; presence signals logout support.
type LogoutCapabilities struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

// ── Initialize ──

// InitializeRequest is the ACP handshake request.
type InitializeRequest struct {
	Meta               map[string]any     `json:"_meta,omitempty"`
	ProtocolVersion    ProtocolVersion    `json:"protocolVersion"`
	ClientCapabilities ClientCapabilities `json:"clientCapabilities"`
	ClientInfo         *Implementation    `json:"clientInfo,omitempty"`
}

// InitializeResponse is the ACP handshake response.
type InitializeResponse struct {
	Meta              map[string]any     `json:"_meta,omitempty"`
	ProtocolVersion   ProtocolVersion    `json:"protocolVersion"`
	AgentCapabilities AgentCapabilities  `json:"agentCapabilities"`
	AgentInfo         *Implementation    `json:"agentInfo,omitempty"`
	AuthMethods       []AuthMethod       `json:"authMethods,omitempty"`
}

// ── Authentication ──

// AuthMethod describes an available authentication method.
type AuthMethod struct {
	Meta        map[string]any `json:"_meta,omitempty"`
	ID          AuthMethodId   `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Type        string         `json:"type,omitempty"` // "agent" (default when absent)
}

// AuthenticateRequest selects an advertised auth method.
type AuthenticateRequest struct {
	Meta     map[string]any `json:"_meta,omitempty"`
	MethodID AuthMethodId   `json:"methodId"`
}

// AuthenticateResponse is empty on success.
type AuthenticateResponse struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

// LogoutRequest ends the authenticated state.
type LogoutRequest struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

// LogoutResponse is empty on success.
type LogoutResponse struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

// ── Session lifecycle ──

// NewSessionRequest creates a new session.
type NewSessionRequest struct {
	Meta                   map[string]any `json:"_meta,omitempty"`
	Cwd                    string         `json:"cwd"` // absolute path
	McpServers             []McpServer    `json:"mcpServers"`
	AdditionalDirectories  []string       `json:"additionalDirectories,omitempty"`
}

// NewSessionResponse is returned after session creation.
type NewSessionResponse struct {
	Meta          map[string]any          `json:"_meta,omitempty"`
	SessionID     SessionId               `json:"sessionId"`
	ConfigOptions []SessionConfigOption   `json:"configOptions,omitempty"`
	Modes         *SessionModeState       `json:"modes,omitempty"`
}

// LoadSessionRequest loads a session and replays its history.
type LoadSessionRequest struct {
	Meta                   map[string]any `json:"_meta,omitempty"`
	SessionID              SessionId      `json:"sessionId"`
	Cwd                    string         `json:"cwd"` // absolute path
	McpServers             []McpServer    `json:"mcpServers"`
	AdditionalDirectories  []string       `json:"additionalDirectories,omitempty"`
}

// LoadSessionResponse is the result of loading a session.
type LoadSessionResponse struct {
	Meta          map[string]any          `json:"_meta,omitempty"`
	ConfigOptions []SessionConfigOption   `json:"configOptions,omitempty"`
	Modes         *SessionModeState       `json:"modes,omitempty"`
}

// ResumeSessionRequest resumes an existing session without history replay.
type ResumeSessionRequest struct {
	Meta                   map[string]any `json:"_meta,omitempty"`
	SessionID              SessionId      `json:"sessionId"`
	Cwd                    string         `json:"cwd"` // absolute path
	McpServers             []McpServer    `json:"mcpServers"`
	AdditionalDirectories  []string       `json:"additionalDirectories,omitempty"`
}

// ResumeSessionResponse is the result of resuming a session.
type ResumeSessionResponse struct {
	Meta          map[string]any          `json:"_meta,omitempty"`
	ConfigOptions []SessionConfigOption   `json:"configOptions,omitempty"`
	Modes         *SessionModeState       `json:"modes,omitempty"`
}

// CloseSessionRequest closes a session.
type CloseSessionRequest struct {
	Meta      map[string]any `json:"_meta,omitempty"`
	SessionID SessionId      `json:"sessionId"`
}

// CloseSessionResponse is empty on success.
type CloseSessionResponse struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

// DeleteSessionRequest permanently deletes a session.
type DeleteSessionRequest struct {
	Meta      map[string]any `json:"_meta,omitempty"`
	SessionID SessionId      `json:"sessionId"`
}

// DeleteSessionResponse is empty on success.
type DeleteSessionResponse struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

// ListSessionsRequest lists available sessions.
type ListSessionsRequest struct {
	Meta   map[string]any `json:"_meta,omitempty"`
	Cursor *string        `json:"cursor,omitempty"`
	Cwd    *string        `json:"cwd,omitempty"` // absolute path filter
}

// ListSessionsResponse is the result of listing sessions.
type ListSessionsResponse struct {
	Meta       map[string]any `json:"_meta,omitempty"`
	NextCursor *string        `json:"nextCursor,omitempty"`
	Sessions   []SessionInfo  `json:"sessions"`
}

// SessionInfo describes an available session.
type SessionInfo struct {
	Meta                  map[string]any `json:"_meta,omitempty"`
	SessionID             SessionId      `json:"sessionId"`
	Cwd                   string         `json:"cwd"`
	Title                 string         `json:"title,omitempty"`
	UpdatedAt             string         `json:"updatedAt,omitempty"` // ISO 8601
	AdditionalDirectories []string       `json:"additionalDirectories,omitempty"`
}

// ── Prompt ──

// PromptRequest is the input for a prompt turn.
type PromptRequest struct {
	Meta      map[string]any `json:"_meta,omitempty"`
	SessionID SessionId      `json:"sessionId"`
	Prompt    []ContentBlock `json:"prompt"`
}

// PromptResponse is the result of a prompt turn.
type PromptResponse struct {
	Meta       map[string]any `json:"_meta,omitempty"`
	StopReason StopReason     `json:"stopReason"`
}

// StopReason indicates why a prompt turn ended.
type StopReason string

const (
	StopReasonEndTurn        StopReason = "end_turn"
	StopReasonMaxTokens      StopReason = "max_tokens"
	StopReasonMaxTurnRequests StopReason = "max_turn_requests"
	StopReasonRefusal        StopReason = "refusal"
	StopReasonCancelled      StopReason = "cancelled"
)

// CancelNotification cancels an ongoing prompt turn.
type CancelNotification struct {
	Meta      map[string]any `json:"_meta,omitempty"`
	SessionID SessionId      `json:"sessionId"`
}

// ── Session modes ──

// SetSessionModeRequest changes the active session mode.
type SetSessionModeRequest struct {
	Meta      map[string]any `json:"_meta,omitempty"`
	SessionID SessionId      `json:"sessionId"`
	ModeID    SessionModeId  `json:"modeId"`
}

// SetSessionModeResponse is empty on success.
type SetSessionModeResponse struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

// SessionModeState reports available and current modes.
type SessionModeState struct {
	Meta           map[string]any `json:"_meta,omitempty"`
	CurrentModeID  SessionModeId  `json:"currentModeId"`
	AvailableModes []SessionMode  `json:"availableModes"`
}

// SessionMode describes a single agent mode.
type SessionMode struct {
	Meta        map[string]any `json:"_meta,omitempty"`
	ID          SessionModeId  `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
}

// CurrentModeUpdate is the payload for sessionUpdate "current_mode_update".
type CurrentModeUpdate struct {
	Meta          map[string]any `json:"_meta,omitempty"`
	CurrentModeID SessionModeId  `json:"currentModeId"`
}

// ── Session config options ──

// SessionConfigOption describes a configurable option.
type SessionConfigOption struct {
	Meta         map[string]any          `json:"_meta,omitempty"`
	ID           SessionConfigId         `json:"id"`
	Name         string                  `json:"name"`
	Description  string                  `json:"description,omitempty"`
	Category     string                  `json:"category,omitempty"` // "mode", "model", "model_config", "thought_level"
	Type         string                  `json:"type"`               // "select" or "boolean"
	CurrentValue any                     `json:"currentValue"`
	Options      []SessionConfigOptValue `json:"options,omitempty"` // required when type="select"
}

// SessionConfigOptValue is a single value in a select config option.
type SessionConfigOptValue struct {
	Meta        map[string]any `json:"_meta,omitempty"`
	Value       string         `json:"value"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
}

// SetSessionConfigOptionRequest changes a config option value.
// The Type field distinguishes boolean from select (value_id) variants.
type SetSessionConfigOptionRequest struct {
	Meta      map[string]any `json:"_meta,omitempty"`
	SessionID SessionId      `json:"sessionId"`
	ConfigID  SessionConfigId `json:"configId"`
	Type      string         `json:"type,omitempty"` // "boolean" selects the boolean variant
	Value     any            `json:"value"`          // string for select, bool for boolean
}

// SetSessionConfigOptionResponse always contains the full config state.
type SetSessionConfigOptionResponse struct {
	Meta          map[string]any        `json:"_meta,omitempty"`
	ConfigOptions []SessionConfigOption `json:"configOptions"`
}

// ConfigOptionUpdate is the payload for sessionUpdate "config_option_update".
type ConfigOptionUpdate struct {
	Meta          map[string]any        `json:"_meta,omitempty"`
	ConfigOptions []SessionConfigOption `json:"configOptions"`
}

// ── MCP Server ──

// McpServer describes an MCP server the Agent should connect to.
// Variants are distinguished by the Type field:
//
//	"" (absent)  → stdio (command + args required)
//	"http"       → HTTP (url + headers required)
//	"sse"        → SSE (url + headers required, deprecated per MCP spec)
type McpServer struct {
	Meta    map[string]any `json:"_meta,omitempty"`
	Type    string         `json:"type,omitempty"` // ""=stdio, "http", "sse"
	Name    string         `json:"name"`
	Command string         `json:"command,omitempty"` // stdio
	Args    []string       `json:"args,omitempty"`    // stdio
	Env     []EnvVariable  `json:"env,omitempty"`     // stdio
	URL     string         `json:"url,omitempty"`     // http / sse
	Headers []HttpHeader   `json:"headers,omitempty"` // http / sse
}

// HttpHeader is a name-value pair for HTTP requests.
type HttpHeader struct {
	Meta  map[string]any `json:"_meta,omitempty"`
	Name  string         `json:"name"`
	Value string         `json:"value"`
}

// EnvVariable is a name-value environment variable pair.
type EnvVariable struct {
	Meta  map[string]any `json:"_meta,omitempty"`
	Name  string         `json:"name"`
	Value string         `json:"value"`
}

// ── Content ──

// ContentBlock is a piece of content in a prompt or message.
// Follows MCP ContentBlock structure. Variants are distinguished by Type:
//
//	"text"          — plain text (baseline, always supported)
//	"image"         — base64-encoded image (requires image prompt capability)
//	"audio"         — base64-encoded audio (requires audio prompt capability)
//	"resource"      — embedded resource (requires embeddedContext prompt capability)
//	"resource_link" — URI reference to a resource (baseline, always supported)
type ContentBlock struct {
	Meta        map[string]any `json:"_meta,omitempty"`
	Type        string         `json:"type"`
	Text        string         `json:"text,omitempty"`        // "text"
	Data        string         `json:"data,omitempty"`        // "image" / "audio" (base64)
	MimeType    string         `json:"mimeType,omitempty"`    // "image" / "audio" / "resource" / "resource_link"
	URI         string         `json:"uri,omitempty"`         // "image" / "resource_link" / "resource"
	Name        string         `json:"name,omitempty"`        // "resource_link"
	Title       string         `json:"title,omitempty"`       // "resource_link"
	Description string         `json:"description,omitempty"` // "resource_link"
	Size        *int64         `json:"size,omitempty"`        // "resource_link"
	Resource    *EmbeddedResource `json:"resource,omitempty"`   // "resource"
	Annotations *Annotations   `json:"annotations,omitempty"`
}

// EmbeddedResource is the payload for "resource" content blocks.
type EmbeddedResource struct {
	Meta     map[string]any `json:"_meta,omitempty"`
	URI      string         `json:"uri"`
	MimeType string         `json:"mimeType,omitempty"`
	Text     string         `json:"text,omitempty"` // TextResourceContents
	Blob     string         `json:"blob,omitempty"` // BlobResourceContents (base64)
}

// Annotations provides metadata about a content block.
type Annotations struct {
	Meta         map[string]any `json:"_meta,omitempty"`
	Audience     []string       `json:"audience,omitempty"`
	LastModified *string        `json:"lastModified,omitempty"` // ISO 8601
	Priority     *float64       `json:"priority,omitempty"`
}

// Content wraps a ContentBlock in a container object.
type Content struct {
	Meta    map[string]any `json:"_meta,omitempty"`
	Content ContentBlock   `json:"content"`
}

// ContentChunk is a chunk of streamed agent output, grouped by messageId.
type ContentChunk struct {
	Meta      map[string]any `json:"_meta,omitempty"`
	Content   ContentBlock   `json:"content"`
	MessageID *MessageId     `json:"messageId,omitempty"`
}

// ── Tool Call ──

// ToolCallUpdate carries tool call state for streaming updates and permission
// requests. Only the fields that have changed need to be sent.
type ToolCallUpdate struct {
	Meta       map[string]any     `json:"_meta,omitempty"`
	ToolCallID string             `json:"toolCallId"`
	Title      string             `json:"title,omitempty"`
	Kind       string             `json:"kind,omitempty"`   // "read", "edit", "delete", "move", "search", "execute", "think", "fetch", "other"
	Status     string             `json:"status,omitempty"` // "pending", "in_progress", "completed", "failed"
	RawInput   any                `json:"rawInput,omitempty"`
	RawOutput  any                `json:"rawOutput,omitempty"`
	Content    []ToolCallContent  `json:"content,omitempty"`
	Locations  []ToolCallLocation `json:"locations,omitempty"`
}

// ToolCallContent carries output in a tool call update.
type ToolCallContent struct {
	Meta       map[string]any `json:"_meta,omitempty"`
	Type       string         `json:"type"`                // "content", "diff", "terminal"
	Content    *ContentBlock  `json:"content,omitempty"`   // "content"
	Diff       *Diff          `json:"diff,omitempty"`      // "diff"
	TerminalID *TerminalId    `json:"terminalId,omitempty"` // "terminal"
}

// ToolCallLocation associates a tool call with a file path and optional line.
type ToolCallLocation struct {
	Meta map[string]any `json:"_meta,omitempty"`
	Path string         `json:"path"`          // absolute
	Line *int           `json:"line,omitempty"` // 1-based
}

// ── Plan ──

// Plan is the payload for sessionUpdate "plan".
type Plan struct {
	Meta    map[string]any `json:"_meta,omitempty"`
	Entries []PlanEntry    `json:"entries"`
}

// PlanEntry is a single step in an agent's plan.
type PlanEntry struct {
	Meta     map[string]any    `json:"_meta,omitempty"`
	Content  string            `json:"content"`
	Priority PlanEntryPriority `json:"priority"`
	Status   string            `json:"status"` // "pending", "in_progress", "completed"
}

// PlanEntryPriority indicates how important a plan step is.
type PlanEntryPriority string

const (
	PlanPriorityHigh   PlanEntryPriority = "high"
	PlanPriorityMedium PlanEntryPriority = "medium"
	PlanPriorityLow    PlanEntryPriority = "low"
)

// ── Slash commands ──

// AvailableCommand describes a slash command the client can invoke.
type AvailableCommand struct {
	Meta        map[string]any         `json:"_meta,omitempty"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Input       *AvailableCommandInput `json:"input,omitempty"`
}

// AvailableCommandInput describes the expected input for a command.
type AvailableCommandInput struct {
	Meta map[string]any `json:"_meta,omitempty"`
	Hint string         `json:"hint"`
}

// AvailableCommandsUpdate is the payload for sessionUpdate "available_commands_update".
type AvailableCommandsUpdate struct {
	Meta              map[string]any     `json:"_meta,omitempty"`
	AvailableCommands []AvailableCommand `json:"availableCommands"`
}

// ── Permission / Approval ──

// PermissionOption describes a choice the user can make in response to a
// permission request.
type PermissionOption struct {
	Meta     map[string]any       `json:"_meta,omitempty"`
	OptionID PermissionOptionId   `json:"optionId"`
	Name     string               `json:"name"`
	Kind     PermissionOptionKind `json:"kind"`
}

// PermissionOptionKind categorizes a permission option.
type PermissionOptionKind string

const (
	PermissionAllowOnce     PermissionOptionKind = "allow_once"
	PermissionAllowAlways   PermissionOptionKind = "allow_always"
	PermissionRejectOnce    PermissionOptionKind = "reject_once"
	PermissionRejectAlways  PermissionOptionKind = "reject_always"
)

// RequestPermissionRequest asks the client for user approval.
type RequestPermissionRequest struct {
	Meta      map[string]any  `json:"_meta,omitempty"`
	SessionID SessionId       `json:"sessionId"`
	ToolCall  ToolCallUpdate  `json:"toolCall"`
	Options   []PermissionOption `json:"options"`
}

// RequestPermissionResponse is the client's decision.
type RequestPermissionResponse struct {
	Meta    map[string]any            `json:"_meta,omitempty"`
	Outcome RequestPermissionOutcome  `json:"outcome"`
}

// RequestPermissionOutcome is a union of possible permission outcomes.
type RequestPermissionOutcome struct {
	Meta     map[string]any        `json:"_meta,omitempty"`
	OptionID *PermissionOptionId   `json:"optionId,omitempty"` // present for "selected"
	Cancelled bool                 `json:"cancelled,omitempty"`
}

// ── File system (Agent → Client RPC) ──

// ReadTextFileRequest reads a file from the client's filesystem.
type ReadTextFileRequest struct {
	Meta      map[string]any `json:"_meta,omitempty"`
	SessionID SessionId      `json:"sessionId"`
	Path      string         `json:"path"` // absolute
	Line      *int           `json:"line,omitempty"`  // 1-based, min 0
	Limit     *int           `json:"limit,omitempty"` // max lines, min 0
}

// ReadTextFileResponse is the file content.
type ReadTextFileResponse struct {
	Meta    map[string]any `json:"_meta,omitempty"`
	Content string         `json:"content"`
}

// WriteTextFileRequest writes content to a file on the client's filesystem.
type WriteTextFileRequest struct {
	Meta      map[string]any `json:"_meta,omitempty"`
	SessionID SessionId      `json:"sessionId"`
	Path      string         `json:"path"` // absolute
	Content   string         `json:"content"`
}

// WriteTextFileResponse is empty on success.
type WriteTextFileResponse struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

// ── Terminal (Agent → Client RPC) ──

// CreateTerminalRequest spawns a command in a new terminal.
type CreateTerminalRequest struct {
	Meta            map[string]any `json:"_meta,omitempty"`
	SessionID       SessionId      `json:"sessionId"`
	Command         string         `json:"command"`
	Args            []string       `json:"args,omitempty"`
	Env             []EnvVariable  `json:"env,omitempty"`
	Cwd             *string        `json:"cwd,omitempty"` // absolute path
	OutputByteLimit *int           `json:"outputByteLimit,omitempty"` // min 0
}

// CreateTerminalResponse returns the new terminal's ID.
type CreateTerminalResponse struct {
	Meta       map[string]any `json:"_meta,omitempty"`
	TerminalID TerminalId     `json:"terminalId"`
}

// TerminalOutputRequest polls the current output of a terminal.
type TerminalOutputRequest struct {
	Meta       map[string]any `json:"_meta,omitempty"`
	SessionID  SessionId      `json:"sessionId"`
	TerminalID TerminalId     `json:"terminalId"`
}

// TerminalOutputResponse holds the captured output and optional exit status.
type TerminalOutputResponse struct {
	Meta       map[string]any       `json:"_meta,omitempty"`
	Output     string               `json:"output"`
	Truncated  bool                 `json:"truncated"`
	ExitStatus *TerminalExitStatus  `json:"exitStatus,omitempty"`
}

// TerminalExitStatus describes how a terminal command exited.
type TerminalExitStatus struct {
	Meta     map[string]any `json:"_meta,omitempty"`
	ExitCode *int           `json:"exitCode,omitempty"` // null if signaled
	Signal   *string        `json:"signal,omitempty"`   // null if exited normally
}

// WaitForTerminalExitRequest blocks until a terminal command finishes.
type WaitForTerminalExitRequest struct {
	Meta       map[string]any `json:"_meta,omitempty"`
	SessionID  SessionId      `json:"sessionId"`
	TerminalID TerminalId     `json:"terminalId"`
}

// WaitForTerminalExitResponse holds the exit code or signal.
type WaitForTerminalExitResponse struct {
	Meta     map[string]any `json:"_meta,omitempty"`
	ExitCode *int           `json:"exitCode,omitempty"`
	Signal   *string        `json:"signal,omitempty"`
}

// KillTerminalRequest terminates a running terminal command.
type KillTerminalRequest struct {
	Meta       map[string]any `json:"_meta,omitempty"`
	SessionID  SessionId      `json:"sessionId"`
	TerminalID TerminalId     `json:"terminalId"`
}

// KillTerminalResponse is empty on success.
type KillTerminalResponse struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

// ReleaseTerminalRequest releases terminal resources.
type ReleaseTerminalRequest struct {
	Meta       map[string]any `json:"_meta,omitempty"`
	SessionID  SessionId      `json:"sessionId"`
	TerminalID TerminalId     `json:"terminalId"`
}

// ReleaseTerminalResponse is empty on success.
type ReleaseTerminalResponse struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

// ── Session update notification ──

// SessionNotification is the envelope for session/update.
type SessionNotification struct {
	Meta      map[string]any `json:"_meta,omitempty"`
	SessionID SessionId      `json:"sessionId"`
	Update    SessionUpdate  `json:"update"`
}

// SessionUpdate is a discriminated union of update types.
// The SessionUpdate field determines which payload fields are populated:
//
//	"agent_message_chunk"       — content (ContentBlock), optional messageId
//	"agent_thought_chunk"       — content (ContentBlock), optional messageId
//	"user_message_chunk"        — content (ContentBlock), optional messageId (session/load replay)
//	"tool_call"                 — toolCallId🔴, title🔴, kind, status, rawInput, locations
//	"tool_call_update"          — toolCallId🔴, status, content, rawOutput, locations (all optional except id)
//	"plan"                      — entries🔴
//	"available_commands_update" — availableCommands🔴
//	"current_mode_update"       — currentModeId🔴
//	"config_option_update"      — configOptions🔴
//	"usage_update"              — used🔴, size🔴, cost
//	"session_info_update"       — title, updatedAt, _meta (flat, no wrapper; per schema def)
type SessionUpdate struct {
	Meta map[string]any `json:"_meta,omitempty"`

	SessionUpdate string `json:"sessionUpdate"`

	// agent_message_chunk / agent_thought_chunk / tool_call_update
	// The spec uses "content" in multiple update variants with different
	// underlying types. Senders populate it with the concrete value;
	// receivers unmarshal via accessor methods (ContentAsBlock, ContentAsToolCallContent).
	Content json.RawMessage `json:"content,omitempty"`

	MessageID *MessageId `json:"messageId,omitempty"`

	// tool_call / tool_call_update
	// Title is *string because session_info_update also uses the "title"
	// JSON key (as a nullable string). Mutually exclusive variants share
	// the same wire field name without conflict.
	ToolCallID string             `json:"toolCallId,omitempty"`
	Title      *string            `json:"title,omitempty"`
	Kind       string             `json:"kind,omitempty"`
	Status     string             `json:"status,omitempty"`
	RawInput   any                `json:"rawInput,omitempty"`
	RawOutput  any                `json:"rawOutput,omitempty"`
	Locations  []ToolCallLocation `json:"locations,omitempty"`

	// plan
	Entries []PlanEntry `json:"entries,omitempty"`

	// available_commands_update
	AvailableCommands []AvailableCommand `json:"availableCommands,omitempty"`

	// current_mode_update
	CurrentModeID SessionModeId `json:"currentModeId,omitempty"`

	// config_option_update
	ConfigOptions []SessionConfigOption `json:"configOptions,omitempty"`

	// usage_update
	Cost *Cost `json:"cost,omitempty"`

	// usage_update token counters
	Used *int `json:"used,omitempty"`
	Size *int `json:"size,omitempty"`

	// session_info_update — per ACP spec the fields are flat inside the
	// update object, not nested under a wrapper struct.
	UpdatedAt *string `json:"updatedAt,omitempty"`
}
// ContentAsBlock unmarshals Content as a ContentBlock. Returns nil if
// Content is empty or represents a different variant.
func (u SessionUpdate) ContentAsBlock() *ContentBlock {
	if len(u.Content) == 0 {
		return nil
	}
	var cb ContentBlock
	if err := json.Unmarshal(u.Content, &cb); err != nil {
		return nil
	}
	return &cb
}

// ContentAsToolCallContent unmarshals Content as []ToolCallContent.
func (u SessionUpdate) ContentAsToolCallContent() []ToolCallContent {
	if len(u.Content) == 0 {
		return nil
	}
	var tcc []ToolCallContent
	if err := json.Unmarshal(u.Content, &tcc); err != nil {
		return nil
	}
	return tcc
}

// SetContentBlock marshals cb into Content (agent_message_chunk / agent_thought_chunk).
func (u *SessionUpdate) SetContentBlock(cb ContentBlock) {
	u.Content, _ = json.Marshal(cb)
}

// SetToolCallContent marshals tcc into Content (tool_call / tool_call_update).
func (u *SessionUpdate) SetToolCallContent(tcc []ToolCallContent) {
	if tcc == nil {
		tcc = []ToolCallContent{}
	}
	u.Content, _ = json.Marshal(tcc)
}


// ── Cost / usage ──

// Cost reports monetary cost in an ISO 4217 currency.
type Cost struct {
	Meta     map[string]any `json:"_meta,omitempty"`
	Amount   float64        `json:"amount"`
	Currency string         `json:"currency"` // ISO 4217 e.g. "USD"
}

// ── Diff ──

// Diff represents a file modification in a tool call.
type Diff struct {
	Meta    map[string]any `json:"_meta,omitempty"`
	Path    string         `json:"path"` // absolute
	NewText string         `json:"newText"`
	OldText *string        `json:"oldText,omitempty"` // null = new file
}

// ── Error ──

// Error is a JSON-RPC 2.0 error object.
type Error struct {
	Code    ErrorCode   `json:"code"`
	Message string      `json:"message"`
	Data    any         `json:"data,omitempty"`
}

// ErrorCode is a JSON-RPC 2.0 error code.
type ErrorCode = int

const (
	ErrorCodeParse           ErrorCode = -32700
	ErrorCodeInvalidRequest  ErrorCode = -32600
	ErrorCodeMethodNotFound  ErrorCode = -32601
	ErrorCodeInvalidParams   ErrorCode = -32602
	ErrorCodeInternal        ErrorCode = -32603
	ErrorCodeRequestCancelled ErrorCode = -32800
	ErrorCodeAuthRequired    ErrorCode = -32000
	ErrorCodeResourceNotFound ErrorCode = -32002
)

// ── Cancel request (cross-cutting) ──

// CancelRequestNotification cancels a specific in-flight request by its JSON-RPC id.
type CancelRequestNotification struct {
	Meta      map[string]any `json:"_meta,omitempty"`
	RequestID RequestId      `json:"requestId"`
}

// ── Agent→Client RPC interfaces ──

// ClientRequester allows an AgentHandler to make Agent→Client RPC calls.
// The Server provides an implementation that writes JSON-RPC requests over
// the transport and blocks until the Client responds.
//
// Methods map 1:1 to the client-side ACP v1 methods:
//
//	https://agentclientprotocol.com/protocol/v1/schema
type ClientRequester interface {
	RequestPermission(ctx context.Context, req RequestPermissionRequest) (*RequestPermissionResponse, error)
	ReadTextFile(ctx context.Context, req ReadTextFileRequest) (*ReadTextFileResponse, error)
	WriteTextFile(ctx context.Context, req WriteTextFileRequest) (*WriteTextFileResponse, error)
	CreateTerminal(ctx context.Context, req CreateTerminalRequest) (*CreateTerminalResponse, error)
	TerminalOutput(ctx context.Context, req TerminalOutputRequest) (*TerminalOutputResponse, error)
	WaitForTerminalExit(ctx context.Context, req WaitForTerminalExitRequest) (*WaitForTerminalExitResponse, error)
	KillTerminal(ctx context.Context, req KillTerminalRequest) (*KillTerminalResponse, error)
	ReleaseTerminal(ctx context.Context, req ReleaseTerminalRequest) (*ReleaseTerminalResponse, error)
}

// ClientRequestHandler receives Agent→Client RPC requests from a connected
// agent. Clients register a handler via [Session.SetClientRequestHandler]
// and respond to the agent's calls synchronously.
//
// Each method maps 1:1 to a client-side ACP v1 method.
type ClientRequestHandler interface {
	HandleRequestPermission(ctx context.Context, req RequestPermissionRequest) (*RequestPermissionResponse, error)
	HandleReadTextFile(ctx context.Context, req ReadTextFileRequest) (*ReadTextFileResponse, error)
	HandleWriteTextFile(ctx context.Context, req WriteTextFileRequest) (*WriteTextFileResponse, error)
	HandleCreateTerminal(ctx context.Context, req CreateTerminalRequest) (*CreateTerminalResponse, error)
	HandleTerminalOutput(ctx context.Context, req TerminalOutputRequest) (*TerminalOutputResponse, error)
	HandleWaitForTerminalExit(ctx context.Context, req WaitForTerminalExitRequest) (*WaitForTerminalExitResponse, error)
	HandleKillTerminal(ctx context.Context, req KillTerminalRequest) (*KillTerminalResponse, error)
	HandleReleaseTerminal(ctx context.Context, req ReleaseTerminalRequest) (*ReleaseTerminalResponse, error)
}

// SessionUpdateSender writes a session/update notification to the transport.
// The mux implements this so handlers can send notifications outside of
// prompt turns (e.g. current_mode_update after set_mode).
type SessionUpdateSender interface {
	SendSessionUpdate(sid SessionId, update SessionUpdate) error
}

// ClientRPCUser is an optional interface that AgentHandler implementations
// may satisfy. The Server detects it at construction time and calls
// SetClientRequester before any other handler method.
type ClientRPCUser interface {
	SetClientRequester(r ClientRequester)
}

// rpcResponse is a JSON-RPC 2.0 response — either Result or Error is set.
// Used by both server-side (agent→client) and client-side (client→agent) pending call tracking.
type rpcResponse struct {
	Result json.RawMessage
	Error  *Error
}

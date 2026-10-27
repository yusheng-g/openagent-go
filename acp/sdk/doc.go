// Package acp implements the Agent Client Protocol v1 — an open, JSON-RPC 2.0
// based protocol for communication between AI agents and their clients (IDEs,
// terminals, chat UIs).
//
//	Official spec: https://agentclientprotocol.com/protocol/v1
//
// # Overview
//
// ACP defines a bidirectional protocol over newline-delimited JSON-RPC 2.0
// messages on stdio (stdin/stdout). It models the interaction as a
// client↔agent relationship:
//
//   - The Client is the user-facing application (IDE, CLI, chat UI).
//   - The Agent is the AI system (LLM-powered tool, MCP host, orchestrator).
//
// The protocol covers the full lifecycle: handshake → session management
// → prompt turns (streaming) → tool execution with permission → cancellation
// → terminal management → file system access.
//
// This package is a complete, spec-conformant SDK. It has zero external
// dependencies — only the Go standard library.
//
// # Architecture
//
// The package exposes two sides:
//
//	Server (Agent)              Client
//	─────────────────            ─────────────────
//	AgentHandler interface       Session struct
//	SessionEventSender           EventHandler interface
//	ClientRequester interface    ClientRequestHandler interface
//	Server struct                Client struct
//	NewServer(name, ver, h)      NewClient(name, ver)
//	server.Run(ctx)              client.ConnectStdio(ctx, nil, cmd, args...)
//
// Server implements the Agent side: it reads JSON-RPC 2.0 messages from
// stdin, routes them to AgentHandler methods, and writes responses plus
// session/update notifications to stdout. An internal mux handles method
// dispatch, prompt serialisation, $/cancel_request propagation, and
// agent→client RPC call management.
//
// Client implements the application side: it spawns an Agent subprocess
// over stdio, sends JSON-RPC 2.0 requests, and dispatches incoming
// notifications to an EventHandler via a background reader goroutine.
//
// # Protocol Methods
//
// Agent-side methods (Client → Agent, implemented via AgentHandler):
//
//	initialize                     handshake + capability negotiation
//	authenticate                   select auth method
//	logout                         end authenticated state
//	session/new                    create session
//	session/load                   load session (history replay via sender)
//	session/resume                 resume session (no replay)
//	session/close                  close session
//	session/delete                 permanently delete session
//	session/list                   list available sessions
//	session/prompt                 send user prompt (streaming output)
//	session/cancel                 cancel ongoing prompt
//	session/set_mode               change session mode
//	session/set_config_option      change config option
//	session/list_messages          list a session's stored messages (paginated)
//	session/list_config_options    list a fresh session's config options
//	session/fork                   copy a session under a new ID; active on response, prompts right away (not while the source is mid-turn)
//
// Session/update notification variants (Agent → Client, sent via
// SessionEventSender or received via EventHandler):
//
//	agent_message_chunk            stream LLM text response
//	agent_thought_chunk            stream LLM reasoning
//	user_message_chunk             replay user message (session/load)
//	tool_call                      announce tool invocation
//	tool_call_update               tool status transition
//	plan                           execution plan + progress
//	available_commands_update      advertise slash commands
//	current_mode_update            mode change notification
//	config_option_update           config change notification
//	usage_update                   token count + cost
//	session_info_update            session metadata (title, etc.)
//
// Agent→Client RPC methods (Agent calls Client, via ClientRequester or
// ClientRequestHandler):
//
//	session/request_permission     ask user for tool approval
//	fs/read_text_file              read file from client filesystem
//	fs/write_text_file             write file to client filesystem
//	terminal/create                spawn terminal command
//	terminal/output                poll terminal output
//	terminal/wait_for_exit         block until command finishes
//	terminal/kill                  terminate running command
//	terminal/release               release terminal resources
//
// # Lifecycle
//
// A typical session flows through these phases:
//
//  1. Client starts Agent subprocess (stdio pipes).
//  2. Client → Agent: initialize (version + capabilities).
//  3. Client → Agent: session/new (cwd, MCP servers).
//  4. Client → Agent: session/prompt → streaming output begins.
//     - Agent → Client: agent_message_chunk (text deltas).
//     - Agent → Client: tool_call (tool invocation announced).
//     - Agent → Client: request_permission (optional approval).
//     - Agent → Client: tool_call_update (in_progress → completed/failed).
//     - (tool results fed back to LLM; loop until end_turn).
//  5. Agent → Client: session/update (stopReason: end_turn).
//  6. Client → Agent: session/close
//
// # Capabilities
//
// Capabilities are negotiated during initialize. Each side advertises
// what it supports (terminal methods, file-system RPC, boolean config
// options, MCP transports, prompt content types, session lifecycle
// methods). Marker types (e.g. SessionCloseCapabilities) serialise as
// empty objects {}. Presence signals support; absence signals the feature
// is unavailable. The protocol requires clients to check capabilities
// before calling optional methods.
//
// # JSON-RPC 2.0 Details
//
// Every message is a single line of JSON with the shape:
//
//	{"jsonrpc":"2.0", "method":"...", "params":{...}, "id":"..."}   (request)
//	{"jsonrpc":"2.0", "method":"...", "params":{...}}               (notification, no id)
//	{"jsonrpc":"2.0", "result":{...}, "id":"..."}                   (response)
//	{"jsonrpc":"2.0", "error":{"code":-32600,"message":"..."}, "id":"..."}
//
// The id field is string, number, or null. idString() normalises all
// three for deterministic Go map keying. The mux distinguishes inbound
// messages by field presence: method present → request/notification;
// method absent, id present → response.
//
// # Usage: Building an Agent
//
//	import openacp "github.com/yusheng-g/openagent-go/acp"
//
//	type MyAgent struct {
//	    client openacp.ClientRequester // set via SetClientRequester
//	}
//
//	func (a *MyAgent) OnInitialize(ctx context.Context, req openacp.InitializeRequest) (*openacp.InitializeResponse, error) {
//	    return &openacp.InitializeResponse{
//	        ProtocolVersion: 1,
//	        AgentCapabilities: openacp.AgentCapabilities{
//	            LoadSession: true,
//	            SessionCapabilities: openacp.SessionCapabilities{
//	                List:   &openacp.SessionListCapabilities{},
//	                Delete: &openacp.SessionDeleteCapabilities{},
//	            },
//	        },
//	    }, nil
//	}
//
//	func (a *MyAgent) OnPrompt(ctx context.Context, req openacp.PromptRequest, sender openacp.SessionEventSender) (*openacp.PromptResponse, error) {
//	    sender.SendAgentMessage("Hello!\n")
//	    return &openacp.PromptResponse{StopReason: openacp.StopReasonEndTurn}, nil
//	}
//
//	func (a *MyAgent) SetClientRequester(r openacp.ClientRequester) { a.client = r }
//
//	func main() {
//	    srv := &MyAgent{}
//	    server := openacp.NewServer("my-agent", "1.0.0", srv)
//	    server.Run(context.Background()) // blocks on stdio
//	}
//
// # Usage: Building a Client
//
//	client := openacp.NewClient("my-client", "1.0.0")
//	sess, _ := client.ConnectStdio(ctx, nil, "./my-agent")
//	defer sess.Close()
//
//	sess.Initialize(ctx, openacp.InitializeRequest{ProtocolVersion: 1})
//	sess.NewSession(ctx, openacp.NewSessionRequest{Cwd: "/tmp"})
//	sess.SetEventHandler(myEventHandler)
//	sess.Prompt(ctx, openacp.PromptRequest{SessionID: "sess_1", Prompt: []openacp.ContentBlock{
//	    {Type: "text", Text: "Hello!"},
//	}})
//	sess.CloseSession(ctx)
//
// # Extensibility
//
// All types carry an optional _meta field (map[string]any) per the ACP
// extensibility spec. Custom methods must use the "_" name prefix.
// Unrecognised requests receive -32601 (Method not found); unrecognised
// notifications are silently ignored.
package acp

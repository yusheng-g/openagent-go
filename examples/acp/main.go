// ACP example: serves an openagent-go agent over the Agent Client Protocol.
//
// The agent is exposed as an ACP subprocess — an IDE or other ACP client
// spawns this binary and communicates over stdin/stdout using the ACP
// framing protocol (JSON-RPC over stdio). This allows any ACP-compatible
// client (VS Code extension, JetBrains plugin, CLI, etc.) to connect,
// create sessions, and send prompts that are handled by the openagent
// agent with full streaming support.
//
// Architecture:
//
//	ACP Client (IDE) ←─ stdin/stdout ─→ this binary (ACP Server)
//	                                         │
//	                                    acp.AgentHandler
//	                                         │
//	                                    openagent.Agent
//	                                      ├── Model (OpenAI-compatible LLM)
//	                                      └── Tools (calculator, etc.)
//
// The acp.AgentHandler interface bridges ACP protocol events to the
// openagent agent lifecycle:
//
//	OnInitialize  → handshake (protocol version, identity)
//	OnNewSession  → create a session context for a conversation thread
//	OnPrompt      → run the agent with streaming, forward events to the client
//	OnCancel      → cancel an in-flight prompt via context cancellation
//	OnCloseSession → tear down session state
//
// Environment variables:
//
//	OPENAGENT_API_KEY   — API key for the LLM
//	OPENAGENT_MODEL     — model ID (e.g. gpt-4o)
//	OPENAGENT_BASE_URL  — optional API base URL
//
// Run standalone (for testing with the acp-go-sdk client):
//
//	go run ./examples/acp/
//
// Or configure an IDE to launch this binary as an ACP agent.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	openacp "github.com/yusheng-g/openagent-go/acp"
	"github.com/yusheng-g/openagent-go/model/openai"
)

// logFile is the debug log file. All diagnostics are written here so they
// survive even when the IDE swallows stderr. The file is created in the
// system temp directory.
var logFile *os.File

func init() {
	tmp := os.TempDir()
	path := filepath.Join(tmp, "openagent-acp.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[acp] cannot create log file %s: %v\n", path, err)
		os.Exit(1)
	}
	logFile = f
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("[acp] === agent starting, log file: %s ===", path)
	log.Printf("[acp] OPENAGENT_API_KEY set: %v", os.Getenv("OPENAGENT_API_KEY") != "")
	log.Printf("[acp] OPENAGENT_MODEL=%s", os.Getenv("OPENAGENT_MODEL"))
	log.Printf("[acp] OPENAGENT_BASE_URL=%s", os.Getenv("OPENAGENT_BASE_URL"))
}

func main() {
	apiKey := os.Getenv("OPENAGENT_API_KEY")
	modelID := os.Getenv("OPENAGENT_MODEL")
	baseURL := os.Getenv("OPENAGENT_BASE_URL")

	if apiKey == "" || modelID == "" {
		fmt.Fprintln(os.Stderr, "set OPENAGENT_API_KEY and OPENAGENT_MODEL")
		log.Printf("[acp] FATAL: OPENAGENT_API_KEY or OPENAGENT_MODEL not set")
		os.Exit(1)
	}

	llm := openai.New(apiKey, modelID, baseURL).WithContextWindow(128_000)
	log.Printf("[acp] model created: %s", modelID)

	agent := openagent.NewAgent("assistant",
		openagent.WithModel(llm),
		openagent.WithInstructions("You are a helpful assistant. Use the calculator tool for arithmetic. Be concise."),
		openagent.WithTools(&calculatorTool{}),
	)
	log.Printf("[acp] agent created: name=%s tools=%d", agent.Name, len(agent.Tools))

	// acpHandler implements openacp.AgentHandler, bridging ACP protocol
	// events to the openagent agent. It manages per-session state and
	// forwards streaming events (text deltas, tool calls) back to the
	// ACP client via SessionEventSender.
	handler := &acpHandler{agent: agent}

	// NewServer creates an ACP server that reads requests from stdin
	// and writes responses to stdout. server.Run() blocks until the
	// client disconnects or the context is cancelled.
	server := openacp.NewServer("openagent-acp", "0.1.0", handler)
	log.Printf("[acp] server created, calling Run() — will block on stdio")
	if err := server.Run(context.Background()); err != nil {
		log.Printf("[acp] server exited with error: %v", err)
		log.Fatalf("ACP server error: %v", err)
	}
	log.Printf("[acp] server.Run() returned normally — client disconnected")
}

// ── ACP Handler ──

// acpHandler implements openacp.AgentHandler, translating ACP protocol
// callbacks into openagent agent operations. It maintains a map of
// active sessions, each with its own openagent.Session and a cancel
// function for in-flight prompts.
type acpHandler struct {
	agent *openagent.Agent

	mu       sync.Mutex
	sessions map[string]*sessionState
}

// sessionState holds per-session data: the openagent Session (identity,
// metadata) and a cancel function for the currently running prompt
// (if any). The cancel function is set in OnPrompt and cleared when
// the prompt completes or is cancelled.
type sessionState struct {
	session  openagent.Session
	cancelFn context.CancelFunc
}

// getOrCreateSession returns the session state for the given ID,
// creating a new one if it doesn't exist.
func (h *acpHandler) getOrCreateSession(id string) *sessionState {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.sessions == nil {
		h.sessions = make(map[string]*sessionState)
	}
	s, ok := h.sessions[id]
	if !ok {
		log.Printf("[acp] creating new session state for id=%s", id)
		s = &sessionState{
			session: openagent.Session{
				ID:        id,
				UserID:    "acp-user",
				AgentName: h.agent.Name,
				CreatedAt: time.Now(),
			},
		}
		h.sessions[id] = s
	}
	return s
}

// OnInitialize handles the ACP handshake. The client sends its protocol
// version and identity; we respond with our own. Echoing back the
// client's protocol version signals compatibility.
func (h *acpHandler) OnInitialize(_ context.Context, req openacp.InitializeRequest) (*openacp.InitializeResponse, error) {
	log.Printf("[acp] OnInitialize: client=%s/%s protocol=%d", req.ClientName, req.ClientVersion, req.ProtocolVersion)
	return &openacp.InitializeResponse{
		ProtocolVersion: req.ProtocolVersion,
		AgentName:       "openagent-acp",
		AgentVersion:    "0.1.0",
	}, nil
}

// OnNewSession creates a new conversation session. The ACP client may
// send MCP server configs (e.g. for tool handoff); this example ignores
// them since the agent has its own built-in tools.
func (h *acpHandler) OnNewSession(_ context.Context, req openacp.NewSessionRequest) (*openacp.NewSessionResponse, error) {
	id := fmt.Sprintf("acp-%d", time.Now().UnixNano())
	log.Printf("[acp] OnNewSession: id=%s cwd=%s mcpServers=%d", id, req.Cwd, len(req.McpServers))
	st := h.getOrCreateSession(id)
	st.session.CreatedAt = time.Now()
	log.Printf("[acp] OnNewSession: returning sessionID=%s", id)
	return &openacp.NewSessionResponse{SessionID: id}, nil
}

// OnPrompt is the main entry point for handling user input. It:
//  1. Concatenates ACP content blocks into a single user message
//  2. Creates a cancellable sub-context so OnCancel can abort the run
//  3. Calls agent.RunStream() to execute the agent with real-time streaming
//  4. Forwards stream events (text, tool calls/results) to the ACP client
//     via the SessionEventSender
//  5. Returns StopReason "end_turn" when the agent finishes
func (h *acpHandler) OnPrompt(ctx context.Context, req openacp.PromptRequest, sender openacp.SessionEventSender) (*openacp.PromptResponse, error) {
	log.Printf("[acp] OnPrompt: sessionID=%s blocks=%d", req.SessionID, len(req.Blocks))

	// Log each content block for debugging.
	for i, b := range req.Blocks {
		text := b.Text
		if len(text) > 200 {
			text = text[:200] + "..."
		}
		log.Printf("[acp] OnPrompt: block[%d] len=%d preview=%q", i, len(b.Text), text)
	}

	st := h.getOrCreateSession(req.SessionID)

	// Concatenate content blocks into a single user message.
	var sb strings.Builder
	for _, b := range req.Blocks {
		sb.WriteString(b.Text)
	}
	userText := sb.String()
	log.Printf("[acp] OnPrompt: combined user text len=%d", len(userText))

	if len(userText) == 0 {
		log.Printf("[acp] OnPrompt: WARNING — empty user text, returning immediately")
		return &openacp.PromptResponse{StopReason: "end_turn"}, nil
	}

	input := openagent.UserMessage(userText)

	// Cancellable context for this prompt so OnCancel can abort it.
	promptCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	h.mu.Lock()
	st.cancelFn = cancel
	h.mu.Unlock()

	log.Printf("[acp] OnPrompt: calling agent.RunStream for session=%s", req.SessionID)

	// Run the agent with streaming, forward events to the ACP client.
	ch := h.agent.RunStream(promptCtx, st.session, input)

	eventCount := 0
	for evt := range ch {
		eventCount++
		switch evt.Type {
		case openagent.StreamTextDelta:
			log.Printf("[acp] OnPrompt: event #%d StreamTextDelta len=%d", eventCount, len(evt.Text))
			if err := sender.SendAgentMessage(evt.Text); err != nil {
				log.Printf("[acp] OnPrompt: SendAgentMessage error: %v", err)
			}
		case openagent.StreamToolCall:
			if len(evt.Message.ToolCalls) > 0 {
				tc := evt.Message.ToolCalls[0]
				log.Printf("[acp] OnPrompt: event #%d StreamToolCall id=%s name=%s", eventCount, tc.ID, tc.Function.Name)
				if err := sender.SendToolCall(openacp.ToolCallEvent{
					ID:     tc.ID,
					Title:  tc.Function.Name,
					Status: "in_progress",
				}); err != nil {
					log.Printf("[acp] OnPrompt: SendToolCall error: %v", err)
				}
			} else {
				log.Printf("[acp] OnPrompt: event #%d StreamToolCall with no tool calls", eventCount)
			}
		case openagent.StreamToolResult:
			log.Printf("[acp] OnPrompt: event #%d StreamToolResult id=%s", eventCount, evt.Message.ToolCallID)
			if err := sender.SendToolCall(openacp.ToolCallEvent{
				ID:     evt.Message.ToolCallID,
				Status: "completed",
			}); err != nil {
				log.Printf("[acp] OnPrompt: SendToolCall (completed) error: %v", err)
			}
		case openagent.StreamDone:
			resultStr := ""
			if evt.Result != nil {
				resultStr = evt.Result.FinalOutput
				if len(resultStr) > 200 {
					resultStr = resultStr[:200] + "..."
				}
			}
			log.Printf("[acp] OnPrompt: event #%d StreamDone output=%q", eventCount, resultStr)
		case openagent.StreamError:
			log.Printf("[acp] OnPrompt: event #%d StreamError: %v", eventCount, evt.Error)
		default:
			log.Printf("[acp] OnPrompt: event #%d unknown type=%s", eventCount, evt.Type)
		}
	}

	log.Printf("[acp] OnPrompt: stream ended, total events=%d, returning end_turn", eventCount)
	return &openacp.PromptResponse{StopReason: "end_turn"}, nil
}

// OnCancel cancels an in-flight prompt by invoking the session's stored
// cancel function. The agent's RunStream will exit via context cancellation,
// and OnPrompt will return with the partial output collected so far.
func (h *acpHandler) OnCancel(_ context.Context, sessionID string) error {
	log.Printf("[acp] OnCancel: session=%s", sessionID)
	h.mu.Lock()
	st := h.sessions[sessionID]
	if st != nil && st.cancelFn != nil {
		st.cancelFn()
		st.cancelFn = nil
	}
	h.mu.Unlock()
	return nil
}

// OnCloseSession tears down a session: cancels any running prompt and
// removes the session state from the map.
func (h *acpHandler) OnCloseSession(_ context.Context, sessionID string) error {
	log.Printf("[acp] OnCloseSession: session=%s", sessionID)
	h.mu.Lock()
	if h.sessions != nil {
		st, ok := h.sessions[sessionID]
		if ok && st.cancelFn != nil {
			st.cancelFn()
		}
		delete(h.sessions, sessionID)
	}
	h.mu.Unlock()
	return nil
}

// ── Calculator Tool ──

type calculatorTool struct{}

func (t *calculatorTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "calculator",
		Description: "Evaluate a math expression like '15+27' or '100/3'.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"expression": {"type": "string", "description": "The math expression to evaluate"}
			},
			"required": ["expression"]
		}`),
	}
}

func (t *calculatorTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Expression string `json:"expression"`
	}
	json.Unmarshal(args, &params)
	log.Printf("[acp] calculator: expression=%q", params.Expression)
	expr := strings.ReplaceAll(params.Expression, " ", "")
	var a, b int
	var op rune
	fmt.Sscanf(expr, "%d%c%d", &a, &op, &b)
	switch op {
	case '+':
		return fmt.Sprintf("%d", a+b), nil
	case '-':
		return fmt.Sprintf("%d", a-b), nil
	case '*':
		return fmt.Sprintf("%d", a*b), nil
	case '/':
		if b == 0 {
			return "", fmt.Errorf("division by zero")
		}
		return fmt.Sprintf("%d", a/b), nil
	default:
		return "", fmt.Errorf("unsupported operator: %c", op)
	}
}

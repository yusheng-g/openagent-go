// Package acp provides an AgentRunner that communicates with an external
// ACP (Agent Client Protocol) agent process via stdio.
//
// Create a Runner and add it to a Team:
//
//	runner, err := acp.New("calculator", "calculates math", "my-acp-agent")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	team.AddAgent("calculator", "Performs arithmetic calculations", runner)
package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	openagent "github.com/yusheng-g/openagent-go"
	openacp "github.com/yusheng-g/openagent-go/acp"
	openmcp "github.com/yusheng-g/openagent-go/mcp"
)

// Runner wraps an external ACP agent process, implementing openagent.AgentRunner.
//
// It manages:
//   - The ACP agent subprocess (stdin/stdout ACP communication)
//   - An HTTP MCP server that exposes transfer_to_* tools for handoff
//
// Call Close() to terminate the process and free resources.
//
// Runner is safe for sequential use. Close() may be called concurrently with
// RunWithPrefix / RunStreamWithPrefix — in-flight runs will fail with an error
// rather than panic.
type Runner struct {
	name    string
	command string
	args    []string

	mu      sync.RWMutex
	session *openacp.Session
	closed  atomic.Bool

	// MCP server for transfer_to_* handoff tools.
	mcpServer     *mcpsdk.Server
	mcpListener   net.Listener
	mcpPort       int
	mcpHTTPServer *http.Server // stored for graceful Shutdown

	// onHandoff receives the handoff callback from TeamContext.RecordHandoff,
	// set by PrepareForTeam before each run. MCP transfer_to_* tool handlers
	// invoke it to write a handoff request to Team.pendingHandoffs.
	onHandoff atomic.Value // func(string, string)

	// teamPrompt is the "## Team Context" block from TeamContext.TeamPrompt,
	// set by PrepareForTeam. buildPromptRequest prepends it as the first
	// content block so the ACP agent knows its role, teammates, and history.
	teamPrompt atomic.Value // string

	// forceFinal mirrors TeamContext.ForceFinal and is checked by MCP tool
	// handlers at call time. When true, transfer_to_* tools return an error
	// rather than performing a handoff — no tool re-registration needed.
	forceFinal atomic.Bool

	// Track registered MCP tool names so we can remove stale ones on resync.
	mcpToolMu    sync.Mutex
	mcpToolNames map[string]bool

	// ACP session ID returned by NewSession, reused across runs.
	acpSessionID string
}

// New creates a Runner by spawning an external ACP agent process.
//
// command is the path to the executable; args are its arguments.
// The ACP agent must implement the ACP Agent interface and speak
// ACP over stdin/stdout.
//
// The agent is initialized immediately: the process starts, ACP
// handshake completes, and an MCP server for handoff tools starts.
func New(name, command string, args ...string) (*Runner, error) {
	r := &Runner{
		name:    name,
		command: command,
		args:    args,
	}

	// 1. Start MCP server for transfer_to_* tools.
	if err := r.startMCPServer(); err != nil {
		return nil, fmt.Errorf("acp runner %q: start MCP server: %w", name, err)
	}

	// 2. Spawn ACP agent process.
	// Use background context — process lifetime is tied to Runner.Close(),
	// NOT to New(). exec.CommandContext would kill the process when the
	// context is cancelled, so we must not use a timeout context here.
	client := openacp.NewClient("openagent-go", "0.1.0")
	sess, err := client.ConnectStdio(context.Background(), command, args...)
	if err != nil {
		r.stopMCPServer()
		return nil, fmt.Errorf("acp runner %q: connect: %w", name, err)
	}
	r.session = sess

	// 3-4. ACP handshake with timeout.
	// The process is already running; only the handshake is time-bounded.
	initCtx, initCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer initCancel()

	if _, err := sess.Initialize(initCtx, openacp.InitializeRequest{
		ProtocolVersion: 1,
		ClientName:      "openagent-go",
		ClientVersion:   "0.1.0",
	}); err != nil {
		r.Close()
		return nil, fmt.Errorf("acp runner %q: initialize: %w", name, err)
	}

	// 4. Create session with MCP server config (reused across runs).
	mcpURL := fmt.Sprintf("http://localhost:%d", r.mcpPort)
	resp, err := sess.NewSession(initCtx, openacp.NewSessionRequest{
		Cwd: "/",
		McpServers: []openacp.McpServer{{
			Name: name + "-handoff",
			URL:  mcpURL,
		}},
	})
	if err != nil {
		r.Close()
		return nil, fmt.Errorf("acp runner %q: new session: %w", name, err)
	}
	r.acpSessionID = resp.SessionID

	return r, nil
}

// Name returns the runner's name.
func (r *Runner) Name() string { return r.name }

// PrepareForTeam implements openagent.TeamPreparable.
// It stores per-run metadata: handoff callback, system prompt, and forceFinal
// flag. MCP tool registration is handled separately via OnMembersChanged.
func (r *Runner) PrepareForTeam(tc openagent.TeamContext) openagent.AgentRunner {
	r.onHandoff.Store(tc.RecordHandoff)
	r.teamPrompt.Store(tc.TeamPrompt)
	r.forceFinal.Store(tc.ForceFinal)
	return r
}

// OnMembersChanged implements openagent.MembershipNotifiable.
// It synchronises the MCP server's transfer_to_* tools to match the
// current team membership. Called by Team.AddAgent / Team.RemoveAgent.
func (r *Runner) OnMembersChanged(selfName string, members []openagent.AgentInfo) {
	r.syncMCPServerTools(selfName, members)
}

// ── AgentRunner implementation ──

// RunWithPrefix executes one turn via ACP Prompt.
func (r *Runner) RunWithPrefix(ctx context.Context, session openagent.Session, prefix []openagent.Message, input openagent.Message) (*openagent.RunResult, error) {
	if r.closed.Load() {
		return nil, fmt.Errorf("acp runner %q is closed", r.name)
	}

	r.mu.RLock()
	sess := r.session
	r.mu.RUnlock()

	if sess == nil {
		return nil, fmt.Errorf("acp runner %q has no session", r.name)
	}

	collector := &resultCollector{}
	sess.SetEventHandler(&runnerHandler{collector: collector})

	resp, err := sess.Prompt(ctx, r.buildPromptRequest(prefix, input))
	if err != nil {
		return nil, fmt.Errorf("acp prompt: %w", err)
	}

	result := collector.result(resp)
	if result.StopReason != "" && result.StopReason != "end_turn" {
		return nil, fmt.Errorf("acp agent stopped: %s", result.StopReason)
	}
	return result, nil
}

// RunStreamWithPrefix executes one turn via ACP Prompt with streaming.
func (r *Runner) RunStreamWithPrefix(ctx context.Context, session openagent.Session, prefix []openagent.Message, input openagent.Message) <-chan openagent.StreamEvent {
	ch := make(chan openagent.StreamEvent, 32)

	go func() {
		defer close(ch)

		if r.closed.Load() {
			ch <- openagent.StreamEvent{Type: openagent.StreamError,
				Error: fmt.Errorf("acp runner %q is closed", r.name)}
			return
		}

		r.mu.RLock()
		sess := r.session
		r.mu.RUnlock()

		if sess == nil {
			ch <- openagent.StreamEvent{Type: openagent.StreamError,
				Error: fmt.Errorf("acp runner %q has no session", r.name)}
			return
		}

		collector := &resultCollector{}
		sess.SetEventHandler(&runnerHandler{streamCh: ch, collector: collector})

		resp, err := sess.Prompt(ctx, r.buildPromptRequest(prefix, input))
		if err != nil {
			ch <- openagent.StreamEvent{Type: openagent.StreamError, Error: err}
			return
		}

		result := collector.result(resp)
		if result.StopReason != "" && result.StopReason != "end_turn" {
			ch <- openagent.StreamEvent{Type: openagent.StreamError,
				Error: fmt.Errorf("acp agent stopped: %s", result.StopReason)}
			return
		}
		ch <- openagent.StreamEvent{Type: openagent.StreamDone, Result: result}
	}()

	return ch
}

// ── Closer ──

// Close terminates the ACP agent process and stops the MCP server.
//
// It is safe to call Close concurrently with RunWithPrefix / RunStreamWithPrefix.
// In-flight runs will fail with an I/O error rather than panic. New runs after
// Close will fail immediately with "runner is closed".
func (r *Runner) Close() error {
	r.closed.Store(true)

	r.mu.Lock()
	sess := r.session
	r.session = nil
	r.mu.Unlock()

	if sess != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = sess.CloseSession(ctx)
		_ = sess.Close()
	}

	r.stopMCPServer()
	return nil
}

// ── MCP server management ──

func (r *Runner) startMCPServer() error {
	r.mcpServer = mcpsdk.NewServer(&mcpsdk.Implementation{
		Name: r.name + "-handoff", Version: "0.1.0",
	}, nil)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	r.mcpListener = listener
	r.mcpPort = listener.Addr().(*net.TCPAddr).Port

	runner := r // capture for closure
	mux := http.NewServeMux()
	mux.Handle("/", mcpsdk.NewStreamableHTTPHandler(
		func(_ *http.Request) *mcpsdk.Server { return runner.mcpServer },
		&mcpsdk.StreamableHTTPOptions{},
	))

	r.mcpHTTPServer = &http.Server{Handler: mux}
	go r.mcpHTTPServer.Serve(listener)

	return nil
}

func (r *Runner) stopMCPServer() {
	if r.mcpHTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = r.mcpHTTPServer.Shutdown(ctx)
		r.mcpHTTPServer = nil
	}
	if r.mcpListener != nil {
		r.mcpListener.Close()
		r.mcpListener = nil
	}
}

// SyncHandoffTargets is a manual trigger for MCP tool synchronisation.
// Normally called automatically via OnMembersChanged; use this only when
// the team membership changes outside of AddAgent/RemoveAgent.
func (r *Runner) SyncHandoffTargets(selfName string, members []openagent.AgentInfo) {
	r.syncMCPServerTools(selfName, members)
}

// handoffParamsJSON is the shared input schema for every transfer_to_* tool.
var handoffParamsJSON = json.RawMessage(`{
	"type": "object",
	"properties": {
		"message": {
			"type": "string",
			"description": "What you need from the other agent. Be specific."
		}
	},
	"required": ["message"]
}`)

// syncMCPServerTools removes stale transfer_to_* tools and registers new ones.
// It uses a diff-based approach: only tools that are actually added or removed
// are touched. Tools that exist in both the current and desired sets are left
// alone — there is no window where they disappear.
//
// Tools stay registered across runs; forceFinal is checked dynamically inside
// each handler instead of re-registering tools. Idempotent — no-op when the
// desired set matches the current set.
func (r *Runner) syncMCPServerTools(selfName string, members []openagent.AgentInfo) {
	if r.mcpServer == nil {
		return
	}

	r.mcpToolMu.Lock()
	defer r.mcpToolMu.Unlock()

	// Compute desired tool set.
	desired := make(map[string]bool)
	memberDescs := make(map[string]string)
	for _, m := range members {
		if m.Name != selfName {
			desired["transfer_to_"+m.Name] = true
			memberDescs[m.Name] = m.Description
		}
	}

	// No-op if unchanged.
	if mapStrEqual(r.mcpToolNames, desired) {
		return
	}

	// Compute diff: remove only tools that are no longer desired,
	// add only tools that are newly desired. Tools present in both
	// sets are NOT touched, eliminating the re-registration window.
	var toRemove []string
	var toAdd []string
	for n := range r.mcpToolNames {
		if !desired[n] {
			toRemove = append(toRemove, n)
		}
	}
	for n := range desired {
		if !r.mcpToolNames[n] {
			toAdd = append(toAdd, n)
		}
	}

	// Remove stale tools.
	if len(toRemove) > 0 {
		r.mcpServer.RemoveTools(toRemove...)
		for _, n := range toRemove {
			delete(r.mcpToolNames, n)
		}
	}

	// Add new tools.
	for _, toolName := range toAdd {
		targetName := toolName[len("transfer_to_"):]

		def := openagent.FunctionDefinition{
			Name:        toolName,
			Description: fmt.Sprintf("Hand off to the %s agent. %s Include a clear message about what you need.", targetName, memberDescs[targetName]),
			EndTurn:    true,
			Parameters:  handoffParamsJSON,
		}

		mcpTool := openmcp.ToMCPTool(def)

		handler := func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
			// Check forceFinal dynamically — tools stay registered.
			if r.forceFinal.Load() {
				return &mcpsdk.CallToolResult{
					IsError: true,
					Content: []mcpsdk.Content{&mcpsdk.TextContent{
						Text: "handoff blocked: this agent must produce a final answer",
					}},
				}, nil
			}

			record, _ := r.onHandoff.Load().(func(string, string))
			if record == nil {
				return &mcpsdk.CallToolResult{
					IsError: true,
					Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: "handoff not available"}},
				}, nil
			}

			var params struct {
				Message string `json:"message"`
			}
			if len(req.Params.Arguments) > 0 {
				json.Unmarshal(req.Params.Arguments, &params)
			}

			record(targetName, params.Message)
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: fmt.Sprintf("Transferred to %s.", targetName)}},
			}, nil
		}

		r.mcpServer.AddTool(mcpTool, handler)
		r.mcpToolNames[toolName] = true
	}
}

// mapStrEqual returns true when two map[string]bool have the same keys.
func mapStrEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// ── Prompt helpers ──

func (r *Runner) buildPromptRequest(prefix []openagent.Message, input openagent.Message) openacp.PromptRequest {
	var blocks []openacp.ContentBlock

	// Inject team context from PrepareForTeam. The block is self-contained:
	// it carries the agent's role, teammate list, handoff instructions, and
	// the handoff chain from previous turns.
	if tp, _ := r.teamPrompt.Load().(string); tp != "" {
		blocks = append(blocks, openacp.ContentBlock{Text: tp})
	}

	for _, msg := range prefix {
		text := formatPrefixMessage(msg)
		if text != "" {
			blocks = append(blocks, openacp.ContentBlock{Text: text})
		}
	}

	if input.Content != "" {
		blocks = append(blocks, openacp.ContentBlock{Text: input.Content})
	}

	return openacp.PromptRequest{
		SessionID: r.acpSessionID,
		Blocks:    blocks,
	}
}

// formatPrefixMessage serializes an openagent.Message into a text representation
// that preserves tool calls, tool call IDs, and multimodal content. This is used
// for cross-agent prefix messages sent to ACP agents, which only support text
// content blocks. The output is designed to be human-readable while retaining
// enough structure for the downstream agent to understand the conversation flow.
func formatPrefixMessage(msg openagent.Message) string {
	var b strings.Builder

	// Determine display role
	role := string(msg.Role)
	switch msg.Role {
	case openagent.RoleTool:
		if msg.ToolCallID != "" {
			role = fmt.Sprintf("tool_result (%s)", msg.ToolCallID)
		}
	case openagent.RoleAssistant:
		if len(msg.ToolCalls) > 0 {
			role = "assistant"
		}
	}

	b.WriteString("[")
	b.WriteString(role)
	b.WriteString("]")

	// Content: prefer ContentParts for multimodal, fall back to Content
	if msg.IsMultimodal() {
		for _, part := range msg.ContentParts {
			b.WriteString("\n")
			switch part.Type {
			case "text":
				if part.Text != "" {
					b.WriteString(part.Text)
				}
			case "image_url":
				if part.ImageURL != nil {
					b.WriteString("[image: ")
					b.WriteString(part.ImageURL.URL)
					b.WriteString("]")
				}
			case "input_audio":
				if part.InputAudio != nil {
					b.WriteString("[audio: ")
					b.WriteString(part.InputAudio.Format)
					b.WriteString("]")
				}
			default:
				b.WriteString("[")
				b.WriteString(part.Type)
				b.WriteString("]")
			}
		}
	} else if msg.Content != "" {
		b.WriteString(" ")
		b.WriteString(msg.Content)
	}

	// Tool calls (assistant messages)
	for _, tc := range msg.ToolCalls {
		b.WriteString("\n  > tool_call: ")
		b.WriteString(tc.Function.Name)
		b.WriteString("(")
		b.WriteString(tc.Function.Arguments)
		b.WriteString(")")
		if tc.ID != "" {
			b.WriteString(" [id: ")
			b.WriteString(tc.ID)
			b.WriteString("]")
		}
	}

	return b.String()
}

// ── runnerHandler ──

// runnerHandler implements openacp.EventHandler to receive streaming
// updates from the ACP agent and translate them to openagent events.
type runnerHandler struct {
	streamCh  chan<- openagent.StreamEvent
	collector *resultCollector
	seenTools map[string]bool // tool call IDs already emitted; prevents duplicates
	                          // when ACP sends in_progress → completed transitions
}

// sendStreamEvent enqueues an event without blocking. If the channel is full
// the event is dropped rather than blocking the ACP transport (which would
// deadlock the stdin/stdout pipe). The collector always captures all data
// for the final RunResult, so dropped events only affect UI display.
func (h *runnerHandler) sendStreamEvent(evt openagent.StreamEvent) {
	if h.streamCh == nil {
		return
	}
	select {
	case h.streamCh <- evt:
	default:
	}
}

func (h *runnerHandler) OnAgentMessage(text string) {
	if h.collector != nil {
		h.collector.addText(text)
	}
	h.sendStreamEvent(openagent.StreamEvent{
		Type: openagent.StreamTextDelta,
		Text: text,
	})
}

func (h *runnerHandler) OnAgentThought(text string) {
	h.sendStreamEvent(openagent.StreamEvent{
		Type: openagent.StreamTextDelta,
		Text: text,
	})
}

func (h *runnerHandler) OnToolCall(tc openacp.ToolCallEvent) {
	if h.seenTools == nil {
		h.seenTools = make(map[string]bool)
	}

	// Emit StreamToolCall only the first time we see this tool ID.
	// ACP agents send separate notifications for in_progress and completed
	// statuses; without dedup the assistant message (and its tool_calls)
	// would be recorded twice in the result.
	if !h.seenTools[tc.ID] {
		h.seenTools[tc.ID] = true
		msg := openagent.Message{
			Role: openagent.RoleAssistant,
			ToolCalls: []openagent.ToolCall{{
				ID:   tc.ID,
				Type: "function",
				Function: openagent.ToolCallFunction{
					Name:      tc.Title,
					Arguments: fmt.Sprintf("%v", tc.RawInput),
				},
			}},
		}
		if h.collector != nil {
			h.collector.addMessage(msg)
		}
		h.sendStreamEvent(openagent.StreamEvent{
			Type:    openagent.StreamToolCall,
			Message: msg,
		})
	}

	// Emit tool result when the tool finishes (completed or failed).
	if (tc.Status == "completed" || tc.Status == "failed") && tc.RawOutput != nil {
		resultMsg := openagent.Message{
			Role:       openagent.RoleTool,
			Content:    fmt.Sprintf("%v", tc.RawOutput),
			ToolCallID: tc.ID,
		}
		if h.collector != nil {
			h.collector.addMessage(resultMsg)
		}
		h.sendStreamEvent(openagent.StreamEvent{
			Type:    openagent.StreamToolResult,
			Message: resultMsg,
		})
	}
}

func (h *runnerHandler) OnPlan(entries []openacp.PlanEntry) {
	// Future: emit StreamPlan events. For now, plan entries are silently
	// tracked by the collector if needed.
}

func (h *runnerHandler) OnUsage(info openacp.UsageInfo) {
	// Future: emit StreamUsage events. For now, usage is silently absorbed.
}

// ── resultCollector ──

type resultCollector struct {
	mu       sync.Mutex
	text     strings.Builder
	messages []openagent.Message
}

func (c *resultCollector) addText(t string) {
	c.mu.Lock()
	c.text.WriteString(t)
	c.mu.Unlock()
}

func (c *resultCollector) addMessage(msg openagent.Message) {
	c.mu.Lock()
	c.messages = append(c.messages, msg)
	c.mu.Unlock()
}

func (c *resultCollector) result(resp *openacp.PromptResponse) *openagent.RunResult {
	c.mu.Lock()
	defer c.mu.Unlock()

	stopReason := ""
	if resp != nil {
		stopReason = resp.StopReason
	}

	return &openagent.RunResult{
		FinalOutput: c.text.String(),
		StopReason:  stopReason,
		Messages:    c.messages,
		TurnCount:   1,
	}
}

// ── Compile-time checks ──

var _ openagent.AgentRunner = (*Runner)(nil)
var _ openagent.TeamPreparable = (*Runner)(nil)
var _ openagent.MembershipNotifiable = (*Runner)(nil)
var _ openagent.Closer = (*Runner)(nil)
var _ io.Closer = (*Runner)(nil)

var _ openacp.EventHandler = (*runnerHandler)(nil)
var _ openacp.PlanHandler = (*runnerHandler)(nil)
var _ openacp.UsageHandler = (*runnerHandler)(nil)

// Package acp provides openagent Agent ↔ ACP protocol integration.
//
// AgentServer wraps an [openagent.Agent] as an [openacp.AgentHandler],
// implementing the full ACP v1 protocol lifecycle.
package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	openacp "github.com/yusheng-g/openagent-go/acp/sdk"
	"github.com/yusheng-g/openagent-go/mcp"
	"github.com/yusheng-g/openagent-go/plan"
	"github.com/yusheng-g/openagent-go/session"
	opentool "github.com/yusheng-g/openagent-go/tool"
	"github.com/yusheng-g/openagent-go/slash"
)

// AgentServer wraps an [openagent.Agent] as an [openacp.AgentHandler].
//
// Usage:
//
//	srv := acp.NewAgentServer(agent, mem, sessionStore)
//	server := openacpsdk.NewServer("my-agent", "1.0.0", srv)
//	server.Run(ctx)
type AgentServer struct {
	Agent        *openagent.Agent
	Mem          openagent.Memory
	SessionStore session.Store
	Models       map[string]openagent.Model // model id → Model

	mu       sync.Mutex
	sessions map[openacp.SessionId]*agentSession
	nextID   int64

	// clientRPC is set by the SDK mux via ClientRPCUser.
	clientRPC    openacp.ClientRequester
	updateSender openacp.SessionUpdateSender
	cmdRegistry   *slash.Registry // slash command dispatch

	// defaultModelID is used when the session config hasn't selected one.
	defaultModelID string
}

// agentSession holds per-session runtime state.
type agentSession struct {
	id        openacp.SessionId
	cwd       string
	createdAt time.Time
	mode      string                          // "chat" or "plan"
	config    map[openacp.SessionConfigId]any // config option values
	cancel    context.CancelFunc

	// Accumulated usage across turns.
	totalTokens int

	// Whether we have sent the first prompt yet — drives auto-title and
	// available_commands_update.
	firstPrompt bool

	// Additional directories from session creation/resume.
	additionalDirectories []string

	// MCP server configs from session creation.
	mcpServers []openacp.McpServer

	// Connected MCP sessions. Populated on session create/load/resume;
	// closed on session close/delete.
	mcpSessions []*mcp.Session

	// MCP tools imported from all connected servers. Populated once at
	// connect time; injected into every agentForTurn clone.
	mcpTools []openagent.Tool

	// Cached plan entries (mirrors SessionStore._meta["plan"]).
	planEntries []plan.Entry
}

// NewAgentServer creates an AgentServer wrapping the given agent.
// models maps model IDs to Model implementations; it is the single source
// of truth for model selection. The agent template's Model field is ignored.
func NewAgentServer(agent *openagent.Agent, mem openagent.Memory, store session.Store, models map[string]openagent.Model) *AgentServer {
	s := &AgentServer{
		Agent:        agent,
		Mem:          mem,
		SessionStore: store,
		Models:       models,
		sessions:     make(map[openacp.SessionId]*agentSession),
	}
	s.cmdRegistry = s.buildCommandRegistry()
	if s.Models == nil {
		s.Models = make(map[string]openagent.Model)
	}
	// Pick the first model as the default.
	for id := range s.Models {
		s.defaultModelID = id
		break
	}
	return s
}

// SetClientRequester implements [openacp.ClientRPCUser].
func (s *AgentServer) SetClientRequester(r openacp.ClientRequester) {
	s.clientRPC = r
	if sender, ok := r.(openacp.SessionUpdateSender); ok {
		s.updateSender = sender
	}
}

var _ openacp.ClientRPCUser = (*AgentServer)(nil)
var _ openacp.AgentHandler = (*AgentServer)(nil)

// ── Session helpers ──

func (s *AgentServer) newSessionID() openacp.SessionId {
	s.mu.Lock()
	s.nextID++
	id := s.nextID
	s.mu.Unlock()
	return openacp.SessionId(fmt.Sprintf("acp_%d_%d", time.Now().UnixNano(), id))
}

func (s *AgentServer) saveMeta(id string, cwd string, kind string) {
	if s.SessionStore == nil {
		return
	}
	now := time.Now()
	info := session.SessionInfo{
		ID:        id,
		Cwd:       cwd,
		CreatedAt: now,
		UpdatedAt: now,
	}
	info.SetMeta("kind", kind)
	_ = s.SessionStore.Save(context.Background(), info)
}

// savePlan persists plan entries to SessionStore._meta["plan"].
// This is called after plan_create tool execution.
func (s *AgentServer) savePlan(ctx context.Context, sessionID string, entries []plan.Entry) {
	if s.SessionStore == nil {
		return
	}
	info, err := s.SessionStore.Get(ctx, sessionID)
	if err != nil || info == nil {
		return
	}
	info.SetMeta("plan", entries)
	info.UpdatedAt = time.Now()
	_ = s.SessionStore.Save(ctx, *info)
}

// loadPlan reads persisted plan entries from SessionStore._meta["plan"].
// JSON unmarshaling turns []plan.Entry into []interface{}, so we
// cannot use GetMeta[[]plan.Entry] — instead, re-marshal+unmarshal.
func (s *AgentServer) loadPlan(ctx context.Context, sessionID string) []plan.Entry {
	if s.SessionStore == nil {
		return nil
	}
	info, err := s.SessionStore.Get(ctx, sessionID)
	if err != nil || info == nil || info.Meta == nil {
		return nil
	}
	raw, ok := info.Meta["plan"]
	if !ok {
		return nil
	}
	// Round-trip through JSON to recover typed struct.
	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var entries []plan.Entry
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil
	}
	return entries
}

// saveMode persists the session mode to SessionStore._meta["mode"].
func (s *AgentServer) saveMode(ctx context.Context, sessionID string, mode string) {
	if s.SessionStore == nil {
		return
	}
	info, err := s.SessionStore.Get(ctx, sessionID)
	if err != nil || info == nil {
		return
	}
	info.SetMeta("mode", mode)
	info.UpdatedAt = time.Now()
	_ = s.SessionStore.Save(ctx, *info)
}

// loadMode reads persisted session mode from SessionStore._meta["mode"].
func (s *AgentServer) loadMode(ctx context.Context, sessionID string) string {
	if s.SessionStore == nil {
		return ""
	}
	info, err := s.SessionStore.Get(ctx, sessionID)
	if err != nil || info == nil || info.Meta == nil {
		return ""
	}
	raw, ok := info.Meta["mode"]
	if !ok {
		return ""
	}
	// Meta values come back as string (unlike complex types like []plan.Entry
	// which become []interface{}).
	mode, _ := raw.(string)
	return mode
}

// connectMCP connects to all configured MCP servers and returns the sessions.
// Tools are listed once at connect time and cached — the connection is
// long-lived (one connection per session lifetime).
// Failed connections are logged but not fatal — MCP is an optional enhancement.
func (s *AgentServer) connectMCP(ctx context.Context, servers []openacp.McpServer) ([]*mcp.Session, []openagent.Tool) {
	client := mcp.NewClient("openagent-acp", "1.0.0")
	var sessions []*mcp.Session
	var tools []openagent.Tool
	for _, cfg := range servers {
		sess, err := s.connectOneMCP(ctx, client, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "acp: MCP connect %q failed: %v\n", cfg.Name, err)
			continue
		}
		sessions = append(sessions, sess)
		st, err := sess.Tools(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "acp: MCP list tools %q failed: %v\n", cfg.Name, err)
			continue
		}
		tools = append(tools, st...)
	}
	return sessions, tools
}

func (s *AgentServer) connectOneMCP(ctx context.Context, client *mcp.Client, cfg openacp.McpServer) (*mcp.Session, error) {
	switch cfg.Type {
	case "http":
		return client.ConnectHTTP(ctx, cfg.URL)
	case "sse":
		return client.ConnectHTTP(ctx, cfg.URL) // SSE deprecated by MCP spec; treat as HTTP
	default:
		// stdio (default when Type is empty).
		envVars := make([]string, len(cfg.Env))
		for i, ev := range cfg.Env {
			envVars[i] = ev.Name + "=" + ev.Value
		}
		return client.ConnectStdioWithEnv(ctx, cfg.Command, cfg.Args, envVars)
	}
}

// disconnectMCP closes all MCP connections.
func (s *AgentServer) disconnectMCP(sessions []*mcp.Session) {
	for _, sess := range sessions {
		_ = sess.Close()
	}
}

func (s *AgentServer) putSession(id openacp.SessionId, ss *agentSession) {
	s.mu.Lock()
	s.sessions[id] = ss
	s.mu.Unlock()
}

func (s *AgentServer) getSession(id openacp.SessionId) *agentSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[id]
}

func (s *AgentServer) removeSession(id openacp.SessionId) {
	s.mu.Lock()
	ss := s.sessions[id]
	delete(s.sessions, id)
	s.mu.Unlock()
	if ss != nil && ss.cancel != nil {
		ss.cancel()
	}
}

// ── openacp.AgentHandler ──

func (s *AgentServer) OnInitialize(ctx context.Context, req openacp.InitializeRequest) (*openacp.InitializeResponse, error) {
	caps := openacp.AgentCapabilities{
		LoadSession: true,
		PromptCapabilities: openacp.PromptCapabilities{
			Image:           true,
			Audio:           false,
			EmbeddedContext: true,
		},
		McpCapabilities: openacp.McpCapabilities{
			HTTP: false,
			SSE:  false,
		},
		SessionCapabilities: openacp.SessionCapabilities{
			Close:  &openacp.SessionCloseCapabilities{},
			Delete: &openacp.SessionDeleteCapabilities{},
			List:   &openacp.SessionListCapabilities{},
			Resume: &openacp.SessionResumeCapabilities{},
		},
		Auth: openacp.AgentAuthCapabilities{
			Logout: &openacp.LogoutCapabilities{},
		},
	}
	return &openacp.InitializeResponse{
		ProtocolVersion:   1,
		AgentCapabilities: caps,
		AgentInfo: &openacp.Implementation{
			Name:    "openagent-acp",
			Version: "1.0.0",
		},
	}, nil
}

// ── Session CRUD ──

func (s *AgentServer) OnNewSession(ctx context.Context, req openacp.NewSessionRequest) (*openacp.NewSessionResponse, error) {
	id := s.newSessionID()
	mcpSessions, mcpTools := s.connectMCP(ctx, req.McpServers)
	ss := &agentSession{
		id:                    id,
		cwd:                   req.Cwd,
		createdAt:             time.Now(),
		mode:                  "chat",
		config:                map[openacp.SessionConfigId]any{"thought_level": "medium", "model": s.defaultModelID},
		firstPrompt:           true,
		additionalDirectories: req.AdditionalDirectories,
		mcpServers:            req.McpServers,
		mcpSessions:           mcpSessions,
		mcpTools:              mcpTools,
	}
	s.putSession(id, ss)
	s.saveMeta(string(id), req.Cwd, "acp")

	return &openacp.NewSessionResponse{
		SessionID:     id,
		ConfigOptions: s.buildConfigOptions(id),
		Modes:         s.buildModeState(id),
	}, nil
}

func (s *AgentServer) OnLoadSession(ctx context.Context, req openacp.LoadSessionRequest, sender openacp.SessionEventSender) (*openacp.LoadSessionResponse, error) {
	ss := s.getSession(req.SessionID)
	if ss == nil {
		mode := s.loadMode(ctx, string(req.SessionID))
		if mode == "" {
			mode = "chat"
		}
		ss = &agentSession{
			id:                    req.SessionID,
			cwd:                   req.Cwd,
			createdAt:             time.Now(),
			mode:                  mode,
			config:                map[openacp.SessionConfigId]any{"thought_level": "medium", "model": s.defaultModelID},
			firstPrompt:           false,
			additionalDirectories: req.AdditionalDirectories,
			mcpServers:            req.McpServers,
		}
		// Reconnect MCP servers and inject tools for this session.
		ss.mcpSessions, ss.mcpTools = s.connectMCP(ctx, req.McpServers)
		s.putSession(req.SessionID, ss)
	}

	// Replay history from Memory if available.
	if s.Mem != nil {
		s.replayHistory(ctx, req.SessionID, sender)
	}

	// Replay persisted plan if present.
	if entries := s.loadPlan(ctx, string(req.SessionID)); len(entries) > 0 {
		ss.planEntries = entries
		s.replayPlan(sender, entries)
	}

	return &openacp.LoadSessionResponse{
		ConfigOptions: s.buildConfigOptions(req.SessionID),
		Modes:         s.buildModeState(req.SessionID),
	}, nil
}

// replayHistory replays stored conversation history as session/update
// notifications: user_message_chunk, agent_message_chunk, and tool call
// events so the client can reconstruct the full conversation.
func (s *AgentServer) replayHistory(ctx context.Context, sid openacp.SessionId, sender openacp.SessionEventSender) {
	n := 200
	if s.Mem != nil {
		if total, err := s.Mem.Count(ctx, string(sid)); err == nil && total > 0 {
			n = total
			if n > 2000 {
				n = 2000
			}
		}
	}
	msgs, err := s.Mem.Recent(ctx, string(sid), n, 0)
	if err != nil {
		return
	}
	for i, msg := range msgs {
		mid := fmt.Sprintf("hist_%d", i)
		switch msg.Role {
		case openagent.RoleUser:
			sender.SendHistoryMessage("user_message_chunk", msg.Content, mid)

		case openagent.RoleAssistant:
			if msg.Content != "" {
				sender.SendHistoryMessage("agent_message_chunk", msg.Content, mid)
			}
			// Replay tool calls made by this assistant message.
			for _, tc := range msg.ToolCalls {
				sender.SendToolCall(openacp.ToolCallUpdate{
					ToolCallID: tc.ID,
					Title:      tc.Function.Name,
					Kind:       "execute",
					Status:     "in_progress",
					RawInput:   json.RawMessage(tc.Function.Arguments),
				})
			}

		case openagent.RoleTool:
			// Tool results — send as completed tool call updates.
			sender.SendToolCall(openacp.ToolCallUpdate{
				ToolCallID: msg.ToolCallID,
				Status:     "completed",
				RawOutput:  map[string]any{"result": msg.Content},
			})

		case openagent.RoleSystem:
			// System messages are not rendered to clients; skip.
		}
	}
}

// replayPlan sends persisted plan entries as a session/update(plan) notification.
func (s *AgentServer) replayPlan(sender openacp.SessionEventSender, entries []plan.Entry) {
	sender.SendPlanUpdate(s.entriesToACP(entries))
}

// entriesToACP converts plan entries to ACP PlanEntry format.
func (s *AgentServer) entriesToACP(entries []plan.Entry) []openacp.PlanEntry {
	acpEntries := make([]openacp.PlanEntry, len(entries))
	for i, e := range entries {
		acpEntries[i] = openacp.PlanEntry{
			Content:  e.Content,
			Priority: openacp.PlanEntryPriority(e.Priority),
			Status:   string(e.Status),
		}
	}
	return acpEntries
}

// copyPlanEntries returns a deep copy of the entries slice.
func copyPlanEntries(src []plan.Entry) []plan.Entry {
	dst := make([]plan.Entry, len(src))
	copy(dst, src)
	return dst
}

func (s *AgentServer) OnResumeSession(ctx context.Context, req openacp.ResumeSessionRequest) (*openacp.ResumeSessionResponse, error) {
	ss := s.getSession(req.SessionID)
	if ss == nil {
		mode := s.loadMode(ctx, string(req.SessionID))
		if mode == "" {
			mode = "chat"
		}
		ss = &agentSession{
			id:                    req.SessionID,
			cwd:                   req.Cwd,
			createdAt:             time.Now(),
			mode:                  mode,
			config:                map[openacp.SessionConfigId]any{"thought_level": "medium", "model": s.defaultModelID},
			firstPrompt:           false,
			additionalDirectories: req.AdditionalDirectories,
			mcpServers:            req.McpServers,
		}
		// Reconnect MCP servers and inject tools for this session.
		ss.mcpSessions, ss.mcpTools = s.connectMCP(ctx, req.McpServers)
		s.putSession(req.SessionID, ss)
	}
	// Load persisted plan into memory (no replay per ACP spec:
	// session/resume MUST NOT replay history).
	if ss.planEntries == nil {
		ss.planEntries = s.loadPlan(ctx, string(req.SessionID))
	}
	return &openacp.ResumeSessionResponse{
		ConfigOptions: s.buildConfigOptions(req.SessionID),
		Modes:         s.buildModeState(req.SessionID),
	}, nil
}

func (s *AgentServer) OnCloseSession(ctx context.Context, req openacp.CloseSessionRequest) (*openacp.CloseSessionResponse, error) {
	ss := s.getSession(req.SessionID)
	if ss != nil {
		s.disconnectMCP(ss.mcpSessions)
	}
	s.removeSession(req.SessionID)
	if s.SessionStore != nil {
		_ = s.SessionStore.Delete(ctx, string(req.SessionID))
	}
	if s.Mem != nil {
		_ = s.Mem.DeleteSession(ctx, string(req.SessionID))
	}
	return &openacp.CloseSessionResponse{}, nil
}

func (s *AgentServer) OnDeleteSession(ctx context.Context, req openacp.DeleteSessionRequest) (*openacp.DeleteSessionResponse, error) {
	ss := s.getSession(req.SessionID)
	if ss != nil {
		s.disconnectMCP(ss.mcpSessions)
	}
	s.removeSession(req.SessionID)
	if s.SessionStore != nil {
		_ = s.SessionStore.Delete(ctx, string(req.SessionID))
	}
	if s.Mem != nil {
		_ = s.Mem.DeleteSession(ctx, string(req.SessionID))
	}
	return &openacp.DeleteSessionResponse{}, nil
}

func (s *AgentServer) OnListSessions(ctx context.Context, req openacp.ListSessionsRequest) (*openacp.ListSessionsResponse, error) {
	if s.SessionStore == nil {
		return &openacp.ListSessionsResponse{Sessions: []openacp.SessionInfo{}}, nil
	}
	list, err := s.SessionStore.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	out := make([]openacp.SessionInfo, 0, len(list))
	for _, si := range list {
		cwd := si.Cwd
		if cwd == "" {
			cwd = "/"
		}
		out = append(out, openacp.SessionInfo{
			SessionID: openacp.SessionId(si.ID),
			Cwd:       cwd,
			Title:     si.Title,
			UpdatedAt: si.UpdatedAt.Format(time.RFC3339),
		})
	}
	return &openacp.ListSessionsResponse{Sessions: out}, nil
}

// ── Config & modes ──

func (s *AgentServer) buildConfigOptions(sid openacp.SessionId) []openacp.SessionConfigOption {
	ss := s.getSession(sid)
	mode := "chat"
	thoughtLevel := "medium"
	modelID := s.defaultModelID
	if ss != nil {
		mode = ss.mode
		if v, ok := ss.config["thought_level"]; ok {
			if val, ok := v.(string); ok {
				thoughtLevel = val
			}
		}
		if v, ok := ss.config["model"]; ok {
			if val, ok := v.(string); ok {
				modelID = val
			}
		}
	}

	opts := []openacp.SessionConfigOption{
		{
			ID:           "mode",
			Name:         "Session Mode",
			Description:  "Chat mode for conversation, Plan mode for goal-driven execution",
			Category:     "mode",
			Type:         "select",
			CurrentValue: mode,
			Options: []openacp.SessionConfigOptValue{
				{Value: "chat", Name: "Chat", Description: "Conversational agent with tools"},
				{Value: "plan", Name: "Plan", Description: "Goal decomposition and DAG execution"},
			},
		},
		{
			ID:           "thought_level",
			Name:         "Reasoning Level",
			Description:  "Controls the amount of reasoning the model produces",
			Category:     "thought_level",
			Type:         "select",
			CurrentValue: thoughtLevel,
			Options: []openacp.SessionConfigOptValue{
				{Value: "low", Name: "Low"},
				{Value: "medium", Name: "Medium"},
				{Value: "high", Name: "High"},
			},
		},
	}

	// Model selector — only surfaced when multiple models are registered.
	if len(s.Models) > 1 {
		modelOpts := make([]openacp.SessionConfigOptValue, 0, len(s.Models))
		for id := range s.Models {
			modelOpts = append(modelOpts, openacp.SessionConfigOptValue{Value: id, Name: id})
		}
		opts = append(opts, openacp.SessionConfigOption{
			ID:           "model",
			Name:         "Model",
			Description:  "Select the LLM model to use",
			Category:     "model",
			Type:         "select",
			CurrentValue: modelID,
			Options:      modelOpts,
		})
	}

	return opts
}

func (s *AgentServer) buildModeState(sid openacp.SessionId) *openacp.SessionModeState {
	ss := s.getSession(sid)
	current := "chat"
	if ss != nil {
		current = ss.mode
	}
	return &openacp.SessionModeState{
		CurrentModeID: openacp.SessionModeId(current),
		AvailableModes: []openacp.SessionMode{
			{ID: "chat", Name: "Chat", Description: "Conversational agent with tools"},
			{ID: "plan", Name: "Plan", Description: "Goal decomposition and DAG execution"},
		},
	}
}

// availableCommands returns the slash commands this agent advertises.
func (s *AgentServer) availableCommands() []openacp.AvailableCommand {
	cmds := s.cmdRegistry.Available()
	out := make([]openacp.AvailableCommand, len(cmds))
	for i, c := range cmds {
		ac := openacp.AvailableCommand{
			Name:        c.Name,
			Description: c.Description,
		}
		if c.Input != nil {
			ac.Input = &openacp.AvailableCommandInput{Hint: c.Input.Hint}
		}
		out[i] = ac
	}
	return out
}

func (s *AgentServer) OnSetSessionMode(ctx context.Context, req openacp.SetSessionModeRequest) (*openacp.SetSessionModeResponse, error) {
	ss := s.getSession(req.SessionID)
	if ss == nil {
		return nil, fmt.Errorf("session %s not found", req.SessionID)
	}
	ss.mode = string(req.ModeID)
	s.saveMode(ctx, string(req.SessionID), ss.mode)

	// Notify the client of the mode change.
	if s.updateSender != nil {
		s.updateSender.SendSessionUpdate(req.SessionID, openacp.SessionUpdate{
			SessionUpdate: "current_mode_update",
			CurrentModeID: openacp.SessionModeId(ss.mode),
		})
	}
	return &openacp.SetSessionModeResponse{}, nil
}

func (s *AgentServer) OnSetSessionConfigOption(ctx context.Context, req openacp.SetSessionConfigOptionRequest) (*openacp.SetSessionConfigOptionResponse, error) {
	ss := s.getSession(req.SessionID)
	if ss == nil {
		return nil, fmt.Errorf("session %s not found", req.SessionID)
	}

	// Per ACP spec: Type "boolean" selects the boolean variant; absent/empty
	// defaults to select (value_id).  Value is bool for boolean, string for select.
	switch req.Type {
	case "boolean":
		if b, ok := req.Value.(bool); ok {
			ss.config[req.ConfigID] = b
		}
	default:
		if val, ok := req.Value.(string); ok {
			ss.config[req.ConfigID] = val
		}
	}

	// Notify clients of the config change.
	opts := s.buildConfigOptions(req.SessionID)
	if s.updateSender != nil {
		s.updateSender.SendSessionUpdate(req.SessionID, openacp.SessionUpdate{
			SessionUpdate: "config_option_update",
			ConfigOptions: opts,
		})
	}

	return &openacp.SetSessionConfigOptionResponse{
		ConfigOptions: s.buildConfigOptions(req.SessionID),
	}, nil
}

// ── Prompt ──

func (s *AgentServer) OnPrompt(ctx context.Context, req openacp.PromptRequest, sender openacp.SessionEventSender) (*openacp.PromptResponse, error) {
	ss := s.getSession(req.SessionID)
	if ss == nil {
		return nil, fmt.Errorf("session %s not found", req.SessionID)
	}

	// ── Build the input message from ACP content blocks ──
	input, err := s.contentBlocksToMessage(req.Prompt)
	if err != nil {
		return nil, fmt.Errorf("prompt: %w", err)
	}
	// ContentParts are populated by contentBlocksToMessage for image/resource
	// blocks. The model backend handles multimodal natively.  Fall through
	// to normal text path when ContentParts is empty.
	if input.Content == "" && !input.IsMultimodal() {
		return nil, fmt.Errorf("empty prompt")
	}

	// ── Per-prompt cancellable context ──
	ctx, cancel := context.WithCancel(ctx)
	ss.cancel = cancel
	defer func() {
		ss.cancel = nil
		cancel()
	}()

	// ── Intercept server-side slash commands ──
	// Must run BEFORE auto-title so slash commands don't get used as
	// the session title (e.g. "/mode plan" would become the title).
	if resp, handled := s.cmdRegistry.Handle(s.buildSlashContext(ctx, req.SessionID, ss), input.Content); handled {
		sender.SendAgentMessage(resp)
		return &openacp.PromptResponse{StopReason: openacp.StopReasonEndTurn}, nil
	}

	// ── Auto-title from first user message ──
	if ss.firstPrompt {
		ss.firstPrompt = false
		title := firstLine(input.Content, 80)
		s.updateTitle(ctx, req.SessionID, title)

		sender.SendSessionInfo(title, nil)
		sender.SendAvailableCommands(s.availableCommands())
	}

	// ── Build agent clone for this turn ──
	agent := s.agentForTurn(req.SessionID)

	oaSession := openagent.Session{
		ID:        string(req.SessionID),
		CreatedAt: ss.createdAt,
		Metadata: map[string]any{
			"cwd":                   ss.cwd,
			"additionalDirectories": ss.additionalDirectories,
			"mcpServers":            ss.mcpServers,
		},
	}

	// ── Register plan_create tool ──
	// The LLM outputs structured plan entries directly via function-calling
	// arguments. The tool validates, persists, and notifies — no internal
	// model calls needed.
	pt := plan.NewCreateTool(func(entries []plan.Entry) {
		ss.planEntries = entries
		s.savePlan(ctx, string(req.SessionID), entries)
		sender.SendPlanUpdate(s.entriesToACP(entries))
	})
	agent.Tools = append(agent.Tools, pt)

	// ── Register plan_update tool ──
	// Always present so the agent can update entry statuses as it works
	// through an existing plan. References entries by stable id.
	if len(ss.planEntries) > 0 {
		pu := plan.NewUpdateTool(func(updates []plan.Update) ([]plan.Entry, error) {
			// Build id → index lookup.
			idxByID := make(map[string]int, len(ss.planEntries))
			for i, e := range ss.planEntries {
				idxByID[e.ID] = i
			}
			for _, u := range updates {
				idx, ok := idxByID[u.ID]
				if !ok {
					return nil, fmt.Errorf("plan_update: unknown step id %q", u.ID)
				}
				ss.planEntries[idx].Status = plan.Status(u.Status)
			}
			s.savePlan(ctx, string(req.SessionID), ss.planEntries)
			sender.SendPlanUpdate(s.entriesToACP(ss.planEntries))
			return copyPlanEntries(ss.planEntries), nil
		})
		agent.Tools = append(agent.Tools, pu)
	}

	// ── Run the agent ──
	// Single code path for all modes. Mode differences are handled
	// upstream: system prompt (agentForTurn overlays mode-specific
	// instructions) and config options (thought_level).
	ch := agent.RunStream(ctx, oaSession, input)
	var usage openagent.Usage
	var stopReason openacp.StopReason

	for evt := range ch {
		switch evt.Type {

		case openagent.StreamThought:
			sender.SendAgentThought(evt.Text)

		case openagent.StreamTextDelta:
			sender.SendAgentMessage(evt.Text)

		// ACP 3-phase tool call lifecycle: pending → in_progress → completed/failed.
		case openagent.StreamToolCall:
			if len(evt.Message.ToolCalls) > 0 {
				for _, tc := range evt.Message.ToolCalls {
					sender.SendToolCall(openacp.ToolCallUpdate{
						ToolCallID: tc.ID,
						Title:      tc.Function.Name,
						Kind:       "execute",
						Status:     "pending",
						RawInput:   json.RawMessage(tc.Function.Arguments),
					})
				}
			}

		case openagent.StreamToolProgress:
			sender.SendToolCall(openacp.ToolCallUpdate{
				ToolCallID: evt.ToolCallID,
				Status:     "in_progress",
				RawOutput:  map[string]any{"chunk": evt.Text},
			})

		case openagent.StreamToolResult:
			status := "completed"
			if strings.HasPrefix(evt.Message.Content, "error: ") ||
				strings.HasPrefix(evt.Message.Content, "Error: ") {
				status = "failed"
			}
			sender.SendToolCall(openacp.ToolCallUpdate{
				ToolCallID: evt.Message.ToolCallID,
				Status:     status,
				RawOutput:  map[string]any{"result": evt.Message.Content},
			})

		case openagent.StreamRetrying:
			if evt.Error != nil {
				sender.SendAgentThought(fmt.Sprintf("[retrying: %v]", evt.Error))
			}

		case openagent.StreamDone:
			if evt.Result != nil {
				usage = evt.Result.Usage
				stopReason = finishReasonToACP(evt.Result.StopReason)
			}

		case openagent.StreamError:
			return nil, evt.Error

		case openagent.StreamAborted:
			return &openacp.PromptResponse{StopReason: openacp.StopReasonCancelled}, nil
		}
	}

	ss.totalTokens += usage.TotalTokens

	// Report *current* context usage (this turn's PromptTokens), not
	// accumulated total. Per ACP spec, `used` means "tokens currently
	// in context" — PromptTokens is the best proxy for that.
	if usage.PromptTokens > 0 {
		cw := 0
		if agent.Model != nil {
			cw = agent.Model.ContextWindow()
		}
		sender.SendUsageUpdate(usage.PromptTokens, cw, nil)
	}

	if ctx.Err() != nil {
		return &openacp.PromptResponse{StopReason: openacp.StopReasonCancelled}, nil
	}
	if stopReason == "" {
		stopReason = openacp.StopReasonEndTurn
	}
	return &openacp.PromptResponse{StopReason: stopReason}, nil
}

// ── Content block conversion ──

// contentBlocksToMessage converts ACP ContentBlocks to an openagent.Message.
// Text blocks become message content; images and resources become ContentParts
// so the model backend can handle them natively.
func (s *AgentServer) contentBlocksToMessage(blocks []openacp.ContentBlock) (openagent.Message, error) {
	var textParts []string
	var contentParts []openagent.ContentPart
	hasText := false

	for _, b := range blocks {
		switch b.Type {
		case "text":
			if b.Text != "" {
				textParts = append(textParts, b.Text)
				hasText = true
			}

		case "image":
			// Images require the image prompt capability (advertised).
			if b.Data != "" && b.MimeType != "" {
				contentParts = append(contentParts, openagent.ContentPart{
					Type: "image_url",
					ImageURL: &openagent.ImageURL{
						URL:    fmt.Sprintf("data:%s;base64,%s", b.MimeType, b.Data),
						Detail: "auto",
					},
				})
			}

		case "resource":
			if b.Resource != nil {
				// Inline the resource text if available; otherwise describe it.
				if b.Resource.Text != "" {
					textParts = append(textParts, b.Resource.Text)
					hasText = true
				} else if b.Resource.Blob != "" {
					textParts = append(textParts, fmt.Sprintf("[binary resource: %s (%s)]", b.Resource.URI, b.Resource.MimeType))
					hasText = true
				}
			}

		case "resource_link":
			textParts = append(textParts, fmt.Sprintf("[linked resource: %s (%s)]", b.URI, b.Name))
			hasText = true

		default:
			// Unknown content block types ignored per ACP extensibility rules.
		}
	}

	if !hasText && len(contentParts) > 0 {
		// Image-only prompt — prepend a context string.
		textParts = append([]string{"[image input]"}, textParts...)
	}

	msg := openagent.Message{
		Role:         openagent.RoleUser,
		Content:      strings.Join(textParts, "\n"),
		ContentParts: contentParts,
	}
	return msg, nil
}

// ── Cancel ──

func (s *AgentServer) OnCancel(ctx context.Context, sid openacp.SessionId) error {
	ss := s.getSession(sid)
	if ss != nil && ss.cancel != nil {
		ss.cancel()
	}
	return nil
}

// ── Auth ──

func (s *AgentServer) OnAuthenticate(ctx context.Context, req openacp.AuthenticateRequest) (*openacp.AuthenticateResponse, error) {
	// No authentication required for local agent.
	return &openacp.AuthenticateResponse{}, nil
}

func (s *AgentServer) OnLogout(ctx context.Context, req openacp.LogoutRequest) (*openacp.LogoutResponse, error) {
	return &openacp.LogoutResponse{}, nil
}

// ── Internal ──

// updateTitle sets the session title in the persistent store.
func (s *AgentServer) updateTitle(ctx context.Context, sessionID openacp.SessionId, title string) {
	if s.SessionStore == nil || title == "" {
		return
	}
	info, err := s.SessionStore.Get(ctx, string(sessionID))
	if err != nil || info == nil {
		return
	}
	if info.Title == "" {
		info.Title = title
		info.UpdatedAt = time.Now()
		_ = s.SessionStore.Save(ctx, *info)
	}
}

// agentForTurn prepares an Agent clone for a single prompt turn.
//
// Clone is necessary because s.Agent is a shared template — per-turn
// overrides (Approver sessionID binding, plan-mode instruction overlay,
// per-session MCP tools, ReasoningEffort from config) must mutate an
// isolated copy.  Agent.Clone() creates an independent Tools backing
// array so append() doesn't grow the template's slice.
func (s *AgentServer) agentForTurn(sid openacp.SessionId) *openagent.Agent {
	clone := s.Agent.Clone()

	// Inject ACP-based permission bridge for tool calls.
	if s.clientRPC != nil {
		clone.Approver = &acpApprover{client: s.clientRPC, sessionID: sid}
	}

	// Apply session-level config to the clone.
	if ss := s.getSession(sid); ss != nil {
		// Resolve model from the registry.
		modelID := s.defaultModelID
		if v, ok := ss.config["model"]; ok {
			if val, ok := v.(string); ok {
				modelID = val
			}
		}
		if m, ok := s.Models[modelID]; ok {
			clone.Model = m
		}

		if v, ok := ss.config["thought_level"]; ok {
			if val, ok := v.(string); ok && val != "" {
				clone.ReasoningEffort = val
			}
		}

		// Plan mode: overlay instructions to encourage structured planning.
		// The agent still decides when to call plan_create — the overlay
		// just makes it more likely to do so for complex tasks.
		if ss.mode == "plan" {
			clone.SystemPrompts = append(clone.SystemPrompts, planModeOverlay)
		}

		// Inject MCP tools from all connected servers.
		clone.Tools = append(clone.Tools, ss.mcpTools...)
	}

	// Inject Agent→Client RPC tools when the client supports them.
	if s.clientRPC != nil {
		clone.Tools = append(clone.Tools,
			opentool.NewACPReadFile(s.clientRPC, sid),
			opentool.NewACPWriteFile(s.clientRPC, sid),
			opentool.NewACPTerminalCreate(s.clientRPC, sid),
			opentool.NewACPTerminalOutput(s.clientRPC, sid),
			opentool.NewACPTerminalWait(s.clientRPC, sid),
			opentool.NewACPTerminalKill(s.clientRPC, sid),
			opentool.NewACPTerminalRelease(s.clientRPC, sid),
		)
	}

	return clone
}

// planModeOverlay is appended to the system instructions when the session
// is in plan mode. It nudges the agent to decompose complex goals explicitly.
const planModeOverlay = `## Plan Mode
You are in plan mode. For complex multi-step tasks, use the plan_create tool
to produce a structured execution plan before starting work. The plan will be
shown to the user so they can review the approach. After creating the plan,
proceed to execute each step in order.

For simple one-step tasks (reading a file, answering a question, a single edit),
you do not need to create a plan — just do the work directly.`

// buildSlashContext constructs the slash.Context for command dispatch.
func (s *AgentServer) buildSlashContext(ctx context.Context, sid openacp.SessionId, ss *agentSession) slash.Context {
	return slash.Context{
		SessionID:   string(sid),
		Cwd:         ss.cwd,
		Mode:        ss.mode,
		TotalTokens: ss.totalTokens,
		CreatedAt:   ss.createdAt,
		SetMode: func(mode string) error {
			ss.mode = mode
			s.saveMode(ctx, string(sid), mode)
			if s.updateSender != nil {
				s.updateSender.SendSessionUpdate(sid, openacp.SessionUpdate{
					SessionUpdate: "current_mode_update",
					CurrentModeID: openacp.SessionModeId(mode),
				})
			}
			return nil
		},
		Rename: func(title string) error {
			return s.renameSession(ctx, sid, title)
		},
		Clear: func() error {
			if s.Mem != nil {
				if err := s.Mem.DeleteSession(ctx, string(sid)); err != nil {
					return err
				}
			}
			ss.totalTokens = 0
			ss.planEntries = nil
			s.savePlan(ctx, string(sid), nil)
			return nil
		},
		ListSess: func() ([]slash.SessionInfo, error) {
			if s.SessionStore == nil {
				return nil, nil
			}
			list, err := s.SessionStore.List(ctx)
			if err != nil {
				return nil, err
			}
			out := make([]slash.SessionInfo, len(list))
			for i, si := range list {
				out[i] = slash.SessionInfo{
					ID:        si.ID,
					Cwd:       si.Cwd,
					Title:     si.Title,
					UpdatedAt: si.UpdatedAt.Format(time.RFC3339),
				}
			}
			return out, nil
		},
	}
}

// renameSession persists a new title and sends a session_info_update.
func (s *AgentServer) renameSession(ctx context.Context, sid openacp.SessionId, title string) error {
	if s.SessionStore == nil {
		return fmt.Errorf("session store not available")
	}
	info, err := s.SessionStore.Get(ctx, string(sid))
	if err != nil || info == nil {
		return fmt.Errorf("session not found")
	}
	info.Title = title
	info.UpdatedAt = time.Now()
	if err := s.SessionStore.Save(ctx, *info); err != nil {
		return err
	}
	// Notify the client of the title change.
	if s.updateSender != nil {
		s.updateSender.SendSessionUpdate(sid, openacp.SessionUpdate{
			SessionUpdate: "session_info_update",
			Title:         &title,
		})
	}
	return nil
}

// ── acpApprover ──

type acpApprover struct {
	client    openacp.ClientRequester
	sessionID openacp.SessionId
}

func (a *acpApprover) Approve(ctx context.Context, call openagent.ToolCall, def openagent.FunctionDefinition, session openagent.Session) (bool, string) {
	if a.client == nil {
		return true, ""
	}
	resp, err := a.client.RequestPermission(ctx, openacp.RequestPermissionRequest{
		SessionID: a.sessionID,
		ToolCall: openacp.ToolCallUpdate{
			ToolCallID: call.ID,
			Title:      def.Name,
			Kind:       "execute",
			Status:     "pending",
			RawInput:   json.RawMessage(call.Function.Arguments),
		},
		Options: []openacp.PermissionOption{
			{OptionID: "allow", Name: "Allow", Kind: openacp.PermissionAllowOnce},
			{OptionID: "always", Name: "Allow Always", Kind: openacp.PermissionAllowAlways},
			{OptionID: "reject", Name: "Reject", Kind: openacp.PermissionRejectOnce},
		},
	})
	if err != nil {
		return false, fmt.Sprintf("permission request failed: %v", err)
	}
	if resp.Outcome.Cancelled {
		return false, "cancelled"
	}
	if resp.Outcome.OptionID == nil {
		return false, "no option selected"
	}
	switch *resp.Outcome.OptionID {
	case "allow", "always":
		return true, ""
	case "reject":
		return false, "rejected by user"
	default:
		return false, fmt.Sprintf("unknown option: %s", *resp.Outcome.OptionID)
	}
}

// firstLine truncates s to the first line, up to maxLen characters.
func firstLine(s string, maxLen int) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[:idx]
	}
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// finishReasonToACP maps model finish reasons to ACP stop reasons.
func finishReasonToACP(finishReason string) openacp.StopReason {
	switch finishReason {
	case "length":
		return openacp.StopReasonMaxTokens
	case "content_filter", "safety":
		return openacp.StopReasonRefusal
	case "handoff":
		// Agent handed off to another agent — effectively end_turn for
		// this session (the handoff target continues elsewhere).
		return openacp.StopReasonEndTurn
	case "":
		return openacp.StopReasonEndTurn
	default:
		// Unknown finish reason — log but don't block.
		return openacp.StopReasonEndTurn
	}
}

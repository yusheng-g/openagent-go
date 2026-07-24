// Package acp provides openagent Agent ↔ ACP protocol integration.
//
// AgentServer wraps an [openagent.Agent] as an [openacp.AgentHandler],
// implementing the full ACP v1 protocol lifecycle.
package acp

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
	openacp "github.com/yusheng-g/openagent-go/acp/sdk"
	"github.com/yusheng-g/openagent-go/mcp"
	"github.com/yusheng-g/openagent-go/model/openai"
	"github.com/yusheng-g/openagent-go/plan"
	wasm "github.com/yusheng-g/openagent-go/plugin/agent/wasm"
	"github.com/yusheng-g/openagent-go/plugin/wasmhost"
	"github.com/yusheng-g/openagent-go/process"
	"github.com/yusheng-g/openagent-go/session"
	"github.com/yusheng-g/openagent-go/slash"
	opentool "github.com/yusheng-g/openagent-go/tool"
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
	cmdRegistry  *slash.Registry // slash command dispatch

	// clientCaps holds the capabilities advertised by the client during
	// initialize. Guarded by mu. Used to gate Agent→Client RPC tool
	// registration (fs/read_text_file, fs/write_text_file, terminal/*)
	// so the LLM is never offered a tool the client cannot handle.
	clientCaps openacp.ClientCapabilities

	// defaultModelID is used when the session config hasn't selected one.
	defaultModelID string

	// ToolFactory is called per-turn to create tools scoped to the
	// session cwd. If nil, only plan + MCP + client-RPC tools are used.
	ToolFactory func(cwd string) []openagent.Tool

	// MCPEnabled controls whether client-advertised MCP servers are
	// connected on session create/load/resume. Default true (enabled in
	// NewAgentServer); set false to disable MCP tool integration.
	MCPEnabled bool

	// Plugin manager and model config backup for runtime_set_model_config.
	PluginMgr    *wasm.Manager
	modelConfigs map[string]ModelConfig // "provider/modelID" → original config
	modelsMu     sync.Mutex
}

// ModelConfig stores the original apiKey/baseURL for a registered model,
// so SetModel can preserve values when only model_id changes.
type ModelConfig struct {
	Provider string
	ModelID  string
	APIKey   string
	BaseURL  string
}

// agentSession holds per-session runtime state.
type agentSession struct {
	id           openacp.SessionId
	cwd          string
	createdAt    time.Time
	mode         string                          // "auto", "manual", or "plan"
	previousMode string                          // mode before entering plan; used by exit_plan_mode
	config       map[openacp.SessionConfigId]any // config option values
	cancel       context.CancelFunc

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

	// planMu guards plan notification sends so that exit_plan_mode's
	// mode change + empty-plan notification is atomic with respect to
	// plan_create / plan_update notification sends. Without this, when
	// tools execute concurrently (runner.go executeTools goroutines),
	// plan entries can arrive at the client after the mode change,
	// causing the VS Code plugin to keep showing plan mode.
	planMu sync.Mutex

	// injectedPlanTools is set to true after enter_plan_mode injects
	// plan_create + exit_plan_mode into the agent clone. Prevents
	// duplicate injection on repeated enter_plan_mode calls within
	// the same turn.
	injectedPlanTools bool

	// processMgr tracks background processes started by the shell tool.
	// Created on session start, cleaned up on deletion.
	processMgr *process.Manager
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
		modelConfigs: make(map[string]ModelConfig),
		sessions:     make(map[openacp.SessionId]*agentSession),
		MCPEnabled:   true,
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

// SetModel replaces or inserts a model in the registry. Used by
// runtime_set_model_config. When the model already exists, empty apiKey
// or baseURL preserve the originals; when inserting a new model, values
// are used as-is.
func (s *AgentServer) SetModel(provider, modelID, apiKey, baseURL string) {
	s.modelsMu.Lock()
	defer s.modelsMu.Unlock()
	key := provider + "/" + modelID
	if old, ok := s.modelConfigs[key]; ok {
		if apiKey == "" {
			apiKey = old.APIKey
		}
		if baseURL == "" {
			baseURL = old.BaseURL
		}
		log.Printf("[acp] updating model: %s", key)
	} else {
		log.Printf("[acp] inserting model: %s", key)
	}
	s.Models[key] = openai.New(apiKey, modelID, baseURL)
	s.modelConfigs[key] = ModelConfig{Provider: provider, ModelID: modelID, APIKey: apiKey, BaseURL: baseURL}
}

// RegisterModel stores a model's original config for SetModel fallback.
func (s *AgentServer) RegisterModel(key, provider, modelID, apiKey, baseURL string) {
	s.modelsMu.Lock()
	defer s.modelsMu.Unlock()
	s.modelConfigs[key] = ModelConfig{Provider: provider, ModelID: modelID, APIKey: apiKey, BaseURL: baseURL}
}

// resolveModelConfig returns the provider and bare model ID for the current
// session's model selection. The session config stores the composite key
// "provider/modelID"; this looks up modelConfigs to extract both parts so
// they can be set on session.Provider / session.ModelID for runtime_* host
// exports and buildModelRequest.
func (s *AgentServer) resolveModelConfig(ss *agentSession) (provider, modelID string) {
	key := s.defaultModelID
	if v, ok := ss.config["model"]; ok {
		if val, ok := v.(string); ok {
			key = val
		}
	}
	s.modelsMu.Lock()
	defer s.modelsMu.Unlock()
	if mc, ok := s.modelConfigs[key]; ok {
		return mc.Provider, mc.ModelID
	}
	return "", key
}

// ── Client capability helpers ──
//
// These read the capabilities advertised by the client during OnInitialize.
// Each helper acquires s.mu internally — safe to call from any goroutine
// (agentForTurn, injectExecutionTools, buildDynamicContext, etc.).

// clientCanReadFile reports whether the client advertised fs.readTextFile.
func (s *AgentServer) clientCanReadFile() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clientCaps.FS.ReadTextFile
}

// clientCanWriteFile reports whether the client advertised fs.writeTextFile.
func (s *AgentServer) clientCanWriteFile() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clientCaps.FS.WriteTextFile
}

// clientCanTerminal reports whether the client advertised terminal support.
func (s *AgentServer) clientCanTerminal() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clientCaps.Terminal
}

// ── Session helpers ──

func (s *AgentServer) newSessionID() openacp.SessionId {
	s.mu.Lock()
	s.nextID++
	id := s.nextID
	s.mu.Unlock()
	return openacp.SessionId(fmt.Sprintf("acp_%d_%d", time.Now().UnixNano(), id))
}

func (s *AgentServer) saveMeta(id string, cwd string, kind string, meta map[string]any) {
	if s.SessionStore == nil {
		return
	}
	now := time.Now()
	info := session.SessionInfo{
		ID:        id,
		Cwd:       cwd,
		CreatedAt: now,
		UpdatedAt: now,
		Meta:      meta,
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
	if !s.MCPEnabled {
		return nil, nil
	}
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
	// Persist the client's advertised capabilities so that agentForTurn
	// can gate Agent→Client RPC tool registration (fs/read_text_file,
	// fs/write_text_file, terminal/*) on what the client actually
	// supports. Without this, the LLM may be offered a tool whose RPC
	// the client will reject with -32601.
	s.mu.Lock()
	s.clientCaps = req.ClientCapabilities
	s.mu.Unlock()

	caps := openacp.AgentCapabilities{
		LoadSession: true,
		PromptCapabilities: openacp.PromptCapabilities{
			Image:           true,
			Audio:           false,
			EmbeddedContext: true,
		},
		McpCapabilities: openacp.McpCapabilities{
			HTTP: true,
			SSE:  true,
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
		mode:                  "auto",
		config:                map[openacp.SessionConfigId]any{"thought_level": "medium", "model": s.defaultModelID},
		firstPrompt:           true,
		additionalDirectories: req.AdditionalDirectories,
		mcpServers:            req.McpServers,
		mcpSessions:           mcpSessions,
		mcpTools:              mcpTools,
	}

	// Create per-session process manager for long-running shell commands.
	if req.Cwd != "" { // require cwd but put files in /tmp
		pm, err := process.NewManager(filepath.Join(os.TempDir(), "openagent", "sess-"+string(id)))
		if err == nil {
			ss.processMgr = pm
		}
	}
	s.putSession(id, ss)
	s.saveMeta(string(id), req.Cwd, "acp", req.Meta)
	if s.PluginMgr != nil {
		s.PluginMgr.OnSessionInit(ctx, wasm.SessionCtx{SessionID: string(id)})
	}

	// Send available commands so the client can show them immediately.
	if s.updateSender != nil {
		s.updateSender.SendSessionUpdate(id, openacp.SessionUpdate{
			SessionUpdate:     "available_commands_update",
			AvailableCommands: s.availableCommands(),
		})
	}

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
			mode = "auto"
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

		// Create per-session process manager for long-running shell commands.
		if req.Cwd != "" {
			pm, err := process.NewManager(filepath.Join(os.TempDir(), "openagent", "sess-"+string(req.SessionID)))
			if err == nil {
				ss.processMgr = pm
			}
		}
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

	// Send available commands (same as session/new).
	if s.updateSender != nil {
		s.updateSender.SendSessionUpdate(req.SessionID, openacp.SessionUpdate{
			SessionUpdate:     "available_commands_update",
			AvailableCommands: s.availableCommands(),
		})
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
			// Replay reasoning content before the message body.
			if msg.ReasoningContent != "" {
				sender.SendHistoryMessage("agent_thought_chunk", msg.ReasoningContent, mid)
			}
			if msg.Content != "" {
				sender.SendHistoryMessage("agent_message_chunk", msg.Content, mid)
			}
			// Replay tool calls made by this assistant message.
			// Status "pending" → sessionUpdate "tool_call" variant.
			for _, tc := range msg.ToolCalls {
				sender.SendToolCall(openacp.ToolCallUpdate{
					ToolCallID: tc.ID,
					Title:      tc.Function.Name,
					Kind:       "execute",
					Status:     "pending",
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
			mode = "auto"
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

		// Create per-session process manager for shell tool background processes.
		if req.Cwd != "" {
			pm, err := process.NewManager(filepath.Join(os.TempDir(), "openagent", "sess-"+string(req.SessionID)))
			if err == nil {
				ss.processMgr = pm
			}
		}

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
	return &openacp.CloseSessionResponse{}, nil
}

func (s *AgentServer) OnDeleteSession(ctx context.Context, req openacp.DeleteSessionRequest) (*openacp.DeleteSessionResponse, error) {
	if s.PluginMgr != nil {
		s.PluginMgr.OnSessionDestroy(ctx, wasm.SessionCtx{SessionID: string(req.SessionID)})
	}
	ss := s.getSession(req.SessionID)
	if ss != nil {
		s.disconnectMCP(ss.mcpSessions)
		if ss.processMgr != nil {
			ss.processMgr.Cleanup()
		}
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
			Meta:      si.Meta,
		})
	}
	return &openacp.ListSessionsResponse{Sessions: out}, nil
}

// ── Config & modes ──

func (s *AgentServer) buildConfigOptions(sid openacp.SessionId) []openacp.SessionConfigOption {
	ss := s.getSession(sid)
	mode := "auto"
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
			Description:  "Auto: full access with LLM-judged approval. Manual: full access with per-tool user approval. Plan: read-only planning.",
			Category:     "mode",
			Type:         "select",
			CurrentValue: mode,
			Options: []openacp.SessionConfigOptValue{
				{Value: "auto", Name: "Auto", Description: "Full tool access with LLM-judged approval for writes"},
				{Value: "manual", Name: "Manual", Description: "Full tool access with per-tool user approval"},
				{Value: "plan", Name: "Plan", Description: "Read-only analysis and planning — no execution tools"},
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

	// Model selector.
	if len(s.Models) > 0 {
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
	current := "auto"
	if ss != nil {
		current = ss.mode
	}
	return &openacp.SessionModeState{
		CurrentModeID: openacp.SessionModeId(current),
		AvailableModes: []openacp.SessionMode{
			{ID: "auto", Name: "Auto", Description: "Full tool access with LLM-judged approval for writes"},
			{ID: "manual", Name: "Manual", Description: "Full tool access with per-tool user approval"},
			{ID: "plan", Name: "Plan", Description: "Read-only analysis and planning — no execution tools"},
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

// setSessionMode transitions the session to a new mode. When entering plan,
// the current mode is saved as previousMode so exit_plan_mode can restore it.
// Callers: OnSetSessionMode (ACP RPC), slash /mode, and OnSetSessionConfigOption.
func (s *AgentServer) setSessionMode(ctx context.Context, sid openacp.SessionId, mode string) error {
	ss := s.getSession(sid)
	if ss == nil {
		return fmt.Errorf("session %s not found", sid)
	}

	// Save previous mode when entering plan (unless already in plan).
	if mode == "plan" && ss.mode != "plan" {
		ss.previousMode = ss.mode
	}
	ss.mode = mode
	s.saveMode(ctx, string(sid), mode)

	if s.updateSender != nil {
		s.updateSender.SendSessionUpdate(sid, openacp.SessionUpdate{
			SessionUpdate: "current_mode_update",
			CurrentModeID: openacp.SessionModeId(mode),
		})
		// Also send config_option_update so the client's mode dropdown
		// (which reads the "mode" config option) stays in sync.
		s.updateSender.SendSessionUpdate(sid, openacp.SessionUpdate{
			SessionUpdate: "config_option_update",
			ConfigOptions: s.buildConfigOptions(sid),
		})
	}
	return nil
}

// enterPlanMode transitions the session into plan mode. Called by
// enter_plan_mode's onEnter callback. The mode change takes effect
// immediately (persisted + client notified), but the agent clone's tools
// are not mutated — the next OnPrompt turn picks up plan mode and
// registers plan_create + exit_plan_mode.
func (s *AgentServer) enterPlanMode(ctx context.Context, sid openacp.SessionId, ss *agentSession) error {
	if ss.mode == "plan" {
		return nil // already in plan mode, no-op
	}
	return s.setSessionMode(ctx, sid, "plan")
}

func (s *AgentServer) OnSetSessionMode(ctx context.Context, req openacp.SetSessionModeRequest) (*openacp.SetSessionModeResponse, error) {
	if err := s.setSessionMode(ctx, req.SessionID, string(req.ModeID)); err != nil {
		return nil, err
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

	// Sync session mode when the client sets the "mode" config option
	// (most clients use set_config_option rather than set_mode).
	// setSessionMode now sends both current_mode_update and
	// config_option_update, so skip the duplicate notification below.
	needsConfigUpdate := true
	if req.ConfigID == "mode" {
		if v, ok := ss.config["mode"].(string); ok {
			_ = s.setSessionMode(ctx, req.SessionID, v)
			needsConfigUpdate = false // already sent by setSessionMode
		}
	}

	// Notify clients of the config change.
	if needsConfigUpdate {
		opts := s.buildConfigOptions(req.SessionID)
		if s.updateSender != nil {
			s.updateSender.SendSessionUpdate(req.SessionID, openacp.SessionUpdate{
				SessionUpdate: "config_option_update",
				ConfigOptions: opts,
			})
		}
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
	// Track whether non-plan tools are used during this turn.
	// If mode is still "plan" at the end and execution tools
	// were used (meaning the system exited plan mode without
	// calling openagent-go's exit_plan_mode), auto-exit.
	var usedNonPlanTool bool
	// Reset per-turn injectedPlanTools flag so enter_plan_mode
	// can inject plan_create + exit_plan_mode again this turn.
	ss.injectedPlanTools = false
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

	providerID, modelID := s.resolveModelConfig(ss)
	oaSession := openagent.Session{
		ID:        string(req.SessionID),
		ModelID:   modelID,
		Provider:  providerID,
		CreatedAt: ss.createdAt,
		Metadata: map[string]any{
			"cwd":                   ss.cwd,
			"additionalDirectories": ss.additionalDirectories,
			"mcpServers":            ss.mcpServers,
		},
		DynamicContext: s.buildDynamicContext(ss),
	}

	// Inject AgentRuntime for runtime_* host exports.
	if s.PluginMgr != nil {
		rt := wasm.BuildAgentRuntime(agent, &oaSession, s.SetModel)
		ctx = wasmhost.WithAgentRuntime(ctx, rt)
	}

	// Inject ProcessManager so the shell tool can persist
	// long-running process output across turns.
	if ss.processMgr != nil {
		ctx = process.WithManager(ctx, ss.processMgr)
	}

	// ── Register mode-specific planning tools ──
	if ss.mode == "plan" {
		// plan_create: only available in plan mode.
		pt := plan.NewCreateTool(func(entries []plan.Entry) {
			ss.planEntries = entries
			s.savePlan(ctx, string(req.SessionID), entries)
			// planMu + mode check prevents a race with concurrent
			// exit_plan_mode (runner.go executes tools in goroutines).
			ss.planMu.Lock()
			if ss.mode == "plan" {
				sender.SendPlanUpdate(s.entriesToACP(entries))
			}
			ss.planMu.Unlock()
		})
		agent.Tools = append(agent.Tools, pt)

		// exit_plan_mode: only available in plan mode.
		// In plan mode the agent has no execution tools. exit_plan_mode
		// restores the mode that was active before entering plan mode and
		// unlocks the full tool set. If the session started in plan (no
		// previous mode), defaults to auto.
		et := plan.NewExitTool(func() error {
			target := ss.previousMode
			if target == "" || target == "plan" {
				target = "auto"
			}

			// Persist mode change and notify client (sends both
			// current_mode_update and config_option_update).
			if err := s.setSessionMode(ctx, req.SessionID, target); err != nil {
				return err
			}

			// Clear the client's plan panel. planMu ensures
			// atomicity with concurrent plan_create / plan_update
			// goroutines: either this empty-plan notification
			// arrives after the entries (correct order), or the
			// mode check in those callbacks skips the entries
			// (client never sees stale plan data after exit).
			ss.planMu.Lock()
			sender.SendPlanUpdate(nil)
			ss.planMu.Unlock()

			// Inject execution tools into the running agent clone
			// for subsequent model calls this turn.
			s.injectExecutionTools(agent, req.SessionID, ss)

			// Set approver based on target mode.
			if target == "manual" && s.clientRPC != nil {
				agent.Approver = &acpApprover{client: s.clientRPC, sessionID: req.SessionID}
			} else {
				agent.Approver = nil
			}

			return nil
		})
		agent.Tools = append(agent.Tools, et)
	} else {
		// enter_plan_mode: available in auto and manual mode.
		// Changes session mode to "plan" AND immediately injects
		// plan_create + exit_plan_mode into the agent clone so they
		// are available this same turn. Without immediate injection,
		// the model would use system-provided plan tools that don't
		// sync the ACP session mode.
		enterTool := plan.NewEnterTool(func() error {
			wasPlan := ss.mode == "plan"
			if err := s.enterPlanMode(ctx, req.SessionID, ss); err != nil {
				return err
			}

			// Inject plan_create + exit_plan_mode on the FIRST
			// transition into plan mode within this turn. Use
			// wasPlan to guard against re-injection on
			// subsequent enter_plan_mode calls (enter→exit→enter).
			if !wasPlan && !ss.injectedPlanTools {
				ss.injectedPlanTools = true
				pt := plan.NewCreateTool(func(entries []plan.Entry) {
					ss.planEntries = entries
					s.savePlan(ctx, string(req.SessionID), entries)
					ss.planMu.Lock()
					if ss.mode == "plan" {
						sender.SendPlanUpdate(s.entriesToACP(entries))
					}
					ss.planMu.Unlock()
				})
				agent.Tools = append(agent.Tools, pt)

				et := plan.NewExitTool(func() error {
					target := ss.previousMode
					if target == "" || target == "plan" {
						target = "auto"
					}
					if err := s.setSessionMode(ctx, req.SessionID, target); err != nil {
						return err
					}
					ss.planMu.Lock()
					sender.SendPlanUpdate(nil)
					ss.planMu.Unlock()
					return nil
				})
				agent.Tools = append(agent.Tools, et)
			}

			return nil
		})
		agent.Tools = append(agent.Tools, enterTool)
	}

	// ── Register plan_update tool (all modes) ──
	// Always registered so it can track plan progress. At execution time
	// the tool reads ss.planEntries to resolve IDs — no stale closures.
	pu := plan.NewUpdateTool(func(updates []plan.Update) ([]plan.Entry, error) {
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
		ss.planMu.Lock()
		sender.SendPlanUpdate(s.entriesToACP(ss.planEntries))
		ss.planMu.Unlock()
		return copyPlanEntries(ss.planEntries), nil
	})
	agent.Tools = append(agent.Tools, pu)

	// ── Run the agent ──
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
					// Detect execution tool usage in plan mode
					// for auto-exit fallback.
					if ss.mode == "plan" && !isPlanTool(tc.Function.Name) {
						usedNonPlanTool = true
					}
					sender.SendToolCall(openacp.ToolCallUpdate{
						ToolCallID: tc.ID,
						Title:      toolTitle(tc.Function.Name, tc.Function.Arguments),
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
			return &openacp.PromptResponse{StopReason: openacp.StopReasonCancelled, Meta: map[string]any{"mode": ss.mode}}, nil
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
		return &openacp.PromptResponse{StopReason: openacp.StopReasonCancelled, Meta: map[string]any{"mode": ss.mode}}, nil
	}
	if stopReason == "" {
		stopReason = openacp.StopReasonEndTurn
	}
	// Auto-exit plan mode if the turn started in plan mode,
	// execution tools were used (meaning the system already
	// exited plan mode), but openagent-go's exit_plan_mode
	// was not called (mode is still "plan").
	// Restore previousMode (same as exit_plan_mode) instead
	// of hardcoding "auto" — preserves manual mode.
	if ss.mode == "plan" && usedNonPlanTool {
		target := ss.previousMode
		if target == "" || target == "plan" {
			target = "auto"
		}
		_ = s.setSessionMode(ctx, req.SessionID, target)
	}
	return &openacp.PromptResponse{StopReason: stopReason, Meta: map[string]any{"mode": ss.mode}}, nil
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

		// ── Mode-gated tool injection ──
		switch ss.mode {
		case "plan":
			// Plan mode: read-only tools only. No execution tools, no
			// approver (no side effects to approve).
			clone.NoSpawn = true
			clone.Approver = nil

			// ACP read_file is safe (reads from client filesystem), but
			// only register it if the client advertised fs.readTextFile.
			if s.clientRPC != nil && s.clientCanReadFile() {
				clone.Tools = append(clone.Tools,
					opentool.NewACPReadFile(s.clientRPC, sid),
				)
			}

			// plan_create, plan_update, exit_plan_mode are injected in
			// OnPrompt — not here — because they need closures over the
			// session and sender.

		case "auto":
			// Auto mode: full tool set, no approval prompts.
			// Safety is handled by Guard.in/Guard.out (if configured).
			clone.Approver = nil
			if s.clientRPC != nil && s.clientCanReadFile() {
				clone.Tools = append(clone.Tools,
					opentool.NewACPReadFile(s.clientRPC, sid),
				)
			}
			s.injectExecutionTools(clone, sid, ss)

		case "manual":
			// Manual mode: full tool set, per-tool user approval via ACP.
			if s.clientRPC != nil {
				clone.Approver = &acpApprover{client: s.clientRPC, sessionID: sid}
				if s.clientCanReadFile() {
					clone.Tools = append(clone.Tools,
						opentool.NewACPReadFile(s.clientRPC, sid),
					)
				}
			}
			s.injectExecutionTools(clone, sid, ss)
		}
	}

	return clone
}

// injectExecutionTools appends all execution-capable tools to the agent
// clone. Called in manual mode and after exit_plan_mode transitions.
// Mirrors the original flat injection — MCP tools, ToolFactory tools,
// and Agent→Client RPC tools all go through here.
func (s *AgentServer) injectExecutionTools(clone *openagent.Agent, sid openacp.SessionId, ss *agentSession) {
	// MCP tools from connected servers.
	clone.Tools = append(clone.Tools, ss.mcpTools...)

	// Per-turn tools scoped to the session cwd.
	if s.ToolFactory != nil && ss.cwd != "" {
		if tools := s.ToolFactory(ss.cwd); len(tools) > 0 {
			clone.Tools = append(clone.Tools, tools...)
		}
	}

	// Agent→Client RPC tools (write/terminal only — read_client_file is
	// added by agentForTurn to avoid duplicate registration across modes).
	// Each tool is gated on the corresponding client capability so the
	// LLM is never offered a tool whose RPC the client will reject.
	if s.clientRPC != nil {
		if s.clientCanWriteFile() {
			clone.Tools = append(clone.Tools,
				opentool.NewACPWriteFile(s.clientRPC, sid),
			)
		}
		if s.clientCanTerminal() {
			clone.Tools = append(clone.Tools,
				opentool.NewACPTerminalCreate(s.clientRPC, sid),
				opentool.NewACPTerminalOutput(s.clientRPC, sid),
				opentool.NewACPTerminalWait(s.clientRPC, sid),
				opentool.NewACPTerminalKill(s.clientRPC, sid),
				opentool.NewACPTerminalRelease(s.clientRPC, sid),
			)
		}
	}
}

// buildDynamicContext assembles per-turn dynamic context from session
// runtime state — plan entries with status, mode instruction, etc.
// Called every turn in OnPrompt; injected into the system prompt via
// Session.DynamicContext → PromptInput → BuildPrompt.
func (s *AgentServer) buildDynamicContext(ss *agentSession) string {
	var b strings.Builder

	// ── Plan entries with current status ──
	if len(ss.planEntries) > 0 {
		b.WriteString("## Current Plan\n")
		for _, e := range ss.planEntries {
			fmt.Fprintf(&b, "- [%s] [%s] %s\n", e.Priority, e.Status, e.Content)
		}
		b.WriteString("\nUpdate plan status with plan_update when starting or completing each step.\n\n")
	}

	// ── Mode instruction ──
	if ss.mode == "plan" {
		b.WriteString("## Session Mode\n")
		b.WriteString("You are in **plan mode**. You have NO execution tools — you cannot modify files, run shell commands, or create terminals. ")
		if s.clientCanReadFile() {
			b.WriteString("Your only tools are read-only inspection (read_client_file) and planning (plan_create, plan_update, exit_plan_mode).\n\n")
			b.WriteString("**Workflow:**\n")
			b.WriteString("1. Read and analyze relevant files to understand the task\n")
		} else {
			b.WriteString("Your only tools are planning (plan_create, plan_update, exit_plan_mode). You have no file-reading tools in this mode — base your plan on the user's description and any context already provided.\n\n")
			b.WriteString("**Workflow:**\n")
			b.WriteString("1. Analyze the task from the user's description and available context\n")
		}
		b.WriteString("2. Call plan_create with concrete, actionable steps\n")
		b.WriteString("3. Wait for the user to review the plan\n")
		b.WriteString("4. Call exit_plan_mode to leave plan mode and begin execution\n\n")
		b.WriteString("Do NOT call exit_plan_mode until you have a complete plan that the user has reviewed.\n")
	} else if len(ss.planEntries) == 0 {
		// Auto/manual mode without a plan: hint about enter_plan_mode.
		b.WriteString("## Task Planning\n")
		b.WriteString("If this task is complex (involves multiple steps, multiple files, or requires careful sequencing), consider calling **enter_plan_mode** first. This will give you access to plan_create for structured planning. After creating a plan and having it reviewed, call exit_plan_mode to regain your execution tools and work through the plan.\n\n")
	}

	return b.String()
}

// buildSlashContext constructs the slash.Context for command dispatch.
func (s *AgentServer) buildSlashContext(ctx context.Context, sid openacp.SessionId, ss *agentSession) slash.Context {
	return slash.Context{
		SessionID:   string(sid),
		Cwd:         ss.cwd,
		Mode:        ss.mode,
		TotalTokens: ss.totalTokens,
		CreatedAt:   ss.createdAt,
		SetMode: func(mode string) error {
			return s.setSessionMode(ctx, sid, mode)
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
		SetModel: func(modelID string) error {
			if _, ok := s.Models[modelID]; !ok {
				return fmt.Errorf("unknown model: %s", modelID)
			}
			ss.config["model"] = modelID
			if s.updateSender != nil {
				opts := s.buildConfigOptions(sid)
				s.updateSender.SendSessionUpdate(sid, openacp.SessionUpdate{
					SessionUpdate: "config_option_update",
					ConfigOptions: opts,
				})
			}
			return nil
		},
		ListModels: func() []string {
			ids := make([]string, 0, len(s.Models))
			for id := range s.Models {
				ids = append(ids, id)
			}
			return ids
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
	case "allow":
		return true, ""
	case "reject":
		reason := "rejected by user"
		if fb, ok := resp.Outcome.Meta["feedback"].(string); ok && fb != "" {
			reason = fb
		}
		return false, reason
	default:
		return false, fmt.Sprintf("unknown option: %s", *resp.Outcome.OptionID)
	}
}

// isPlanTool reports whether the given tool name is a plan-mode-only tool
// (plan_create, plan_update, exit_plan_mode) or a read-only inspection tool
// (read_client_file). These tools are allowed in plan mode.
func isPlanTool(name string) bool {
	switch name {
	case "plan_create", "plan_update", "exit_plan_mode", "read_client_file":
		return true
	}
	return false
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

// toolTitle builds a human-readable title for an ACP tool_call update.
// Extracts the most informative field from the tool arguments JSON.
func toolTitle(name string, args string) string {
	var params struct {
		Path    string `json:"path"`
		Command string `json:"command"`
		Query   string `json:"query"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return name
	}
	switch name {
	case "read", "write", "ls":
		if params.Path != "" {
			return name + " " + params.Path
		}
	case "shell":
		if params.Command != "" {
			return name + ": " + truncateToolArg(params.Command, 60)
		}
	case "grep":
		if params.Query != "" {
			return name + " " + params.Query
		}
	case "recall":
		if params.Query != "" {
			return name + " " + params.Query
		}
	}
	return name
}

// truncateToolArg truncates s to n characters, adding "..." at the end.
func truncateToolArg(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
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

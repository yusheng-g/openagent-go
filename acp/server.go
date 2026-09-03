// Package acp provides openagent Agent ↔ ACP protocol integration.
//
// AgentServer wraps an [openagent.Agent] as an [openacp.AgentHandler],
// implementing the full ACP v1 protocol lifecycle.
package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	openacp "github.com/yusheng-g/openagent-go/acp/sdk"
	"github.com/yusheng-g/openagent-go/agent"
	ctxpkg "github.com/yusheng-g/openagent-go/context"
	"github.com/yusheng-g/openagent-go/governance"
	"github.com/yusheng-g/openagent-go/kernel"
	"github.com/yusheng-g/openagent-go/mcp"
	"github.com/yusheng-g/openagent-go/model/openai"
	"github.com/yusheng-g/openagent-go/plan"
	wasm "github.com/yusheng-g/openagent-go/plugin/agent/wasm"
	"github.com/yusheng-g/openagent-go/plugin/wasmhost"
	"github.com/yusheng-g/openagent-go/process"
	"github.com/yusheng-g/openagent-go/provider/skill"
	"github.com/yusheng-g/openagent-go/session"
	fs "github.com/yusheng-g/openagent-go/skill/fs"
	builtinskills "github.com/yusheng-g/openagent-go/skills"
	"github.com/yusheng-g/openagent-go/slash"
	"github.com/yusheng-g/openagent-go/summarizer"
	opentool "github.com/yusheng-g/openagent-go/tool"
	"github.com/yusheng-g/openagent-go/utils"
)

// AgentServer wraps an [openagent.Agent] as an [openacp.AgentHandler].
//
// Usage:
//
//	srv := acp.NewAgentServer(agent, mem, sessionStore)
//	server := openacpsdk.NewServer("my-agent", "1.0.0", srv)
//	server.Run(ctx)
type AgentServer struct {
	Cfg     *agent.Agent // template configuration (cloned per turn)
	Deps    kernel.Deps  // template runtime deps (derived per turn)
	Mem     session.SessionStore
	Runtime session.Runtime            // session lifecycle (meta + messages), nil-safe halves
	Models  map[string]openagent.Model // model id → Model

	mu       sync.Mutex
	sessions map[openacp.SessionId]*agentSession
	nextID   int64

	// clientRPC is set by the SDK mux via ClientRPCUser.
	clientRPC    openacp.ClientRequester
	updateSender openacp.SessionUpdateSender
	cmdRegistry  *slash.Registry // slash command dispatch

	// turnTrigger is set by the SDK mux via TurnTriggerUser. It lets the
	// server start an idle turn (no client prompt) to process async
	// sub-agent completions. nil when no trigger is wired (CLI one-shot).
	turnTriggerMu sync.RWMutex
	turnTrigger   openacp.TurnTrigger

	// clientCaps holds the capabilities advertised by the client during
	// initialize. Guarded by mu. Used to gate Agent→Client RPC tool
	// registration (fs/read_text_file, fs/write_text_file, terminal/*)
	// so the LLM is never offered a tool the client cannot handle.
	clientCaps openacp.ClientCapabilities

	// defaultModelID is used when the session config hasn't selected one.
	defaultModelID string

	// ToolFactory creates tools scoped to the
	// session cwd. If nil, only plan + MCP + client-RPC tools are used.
	ToolFactory func(cwd string) []openagent.Tool

	// AgentName and AgentVersion are reported to peers in ACP initialize
	// (agentInfo) and MCP client identity. They are populated by the
	// assembly layer from the version package; left empty they yield an
	// empty reported identity (a wiring signal, not a fatal error).
	AgentName    string
	AgentVersion string

	// MCPEnabled controls whether client-advertised MCP servers are
	// connected on session create/load/resume. Default true (enabled in
	// NewAgentServer); set false to disable MCP tool integration.
	MCPEnabled bool

	// settingsMcpServers are MCP servers declared in settings.json (the
	// global config). They are merged with client-advertised servers
	// (req.McpServers) at session create/load/resume — client wins on name
	// conflict. Guarded by mcpMu so the settings watcher can hot-swap them.
	settingsMcpServers []openacp.McpServer
	mcpMu              sync.RWMutex

	// Plugin manager and model config backup for runtime_set_model_config.
	PluginMgr    *wasm.Manager
	modelConfigs map[string]ModelConfig // "provider/modelID" → original config

	// Summarizer and extractor are updated alongside the session runtime
	// when the user switches models, so background compression and
	// knowledge extraction use the current provider instead of the
	// first-configured one.
	Summarizer *summarizer.Compressor
	Extractor  *ctxpkg.AsyncExtractor

	// approvalMemory persists session-scoped approval decisions
	// ("allow always" — the same tool + args no longer asks within this
	// session; "allow once" is deliberately NOT remembered, per the ACP
	// allow_once / allow_always semantics).
	approvalMemory governance.ApprovalMemory
	modelsMu       sync.Mutex

	// ProfileResolver resolves static context prompts (SOUL/SYSTEM/AGENTS)
	// for a given cwd. If set, buildRuntimeForSession calls it at session
	// creation to override
	// the clone's SystemPrompts with session-cwd-aware profiles. If nil,
	// the agent template's SystemPrompts are used as-is.
	ProfileResolver func(cwd string) []string

	// DefaultMode is the mode new sessions start in; "" = "manual"
	// (approval-based safe default). Configured via settings
	// "default_mode": "auto" | "manual" | "plan".
	DefaultMode string
}

// defaultMode resolves the configured default mode.
func (s *AgentServer) defaultMode() string {
	if s.DefaultMode == "auto" || s.DefaultMode == "plan" {
		return s.DefaultMode
	}
	return "manual"
}

// ModelConfig stores the original apiKey/baseURL for a registered model,
// so SetModel can preserve values when only model_id changes.
type ModelConfig struct {
	Provider                 string
	ModelID                  string
	APIKey                   string
	BaseURL                  string
	MaxInputTokens           int
	MaxOutputTokens          int
	InputCostPerMillion      float64
	InputCacheCostPerMillion float64
	OutputCostPerMillion     float64
}

// agentSession holds per-session runtime state.
type agentSession struct {
	id        openacp.SessionId
	cwd       string
	createdAt time.Time

	// modeMu guards ALL mutable session state: the mode state machine
	// (mode, previousMode, planEntries, injectedPlanTools), config,
	// cancel, totalTokens, modeTools, subAgentTools, and the session
	// runtime reference (rt). It supersedes the former planMu. Reads and
	// writes go through the accessors below
	// (Mode/PreviousMode/PlanEntries/SetPlanEntries/ConfigValue/
	// SetConfigValue/ConfigSnapshot/TotalTokens/setCancel/cancelPrompt/
	// getRuntime/setRuntime/setSubAgentTools/ApplyPlanUpdates/
	// ClearPlanEntries/transitionModeLocked/...), EXCEPT
	// transitionModeLocked and applyModeTools which callers invoke while
	// already holding the write lock.
	//
	// Three execution flows touch this state and MUST be guarded: the
	// serve loop (RPCs), the per-session prompt goroutine (serialized by
	// acp/sdk/server.go), and tool callbacks running in parallel
	// goroutines (executeTools) — plan_create/plan_update/
	// enter_plan_mode/exit_plan_mode closures, wasm runtime_set exports.
	// Notifications are sent to the ACP single-writer queue (non-blocking,
	// FIFO — see acp/sdk/server.go writeQueue), so holding modeMu across
	// SendPlanUpdate is safe and preserves wire ordering
	// (entries-before-empty-plan).
	//
	// Lock-order: modeMu is acquired on its own. getSession takes s.mu
	// and returns, releasing it before any modeMu use. modeMu is never
	// held across saveMode/savePlan SessionStore I/O (those run after
	// unlock; the snapshot is captured under the lock). kernel.Runtime
	// locks (rt.mu, rt.toolsMu) may be acquired while holding modeMu
	// (applyModeTools); Runtime never calls back into agentSession, so
	// there is no inversion.
	modeMu       sync.RWMutex
	mode         string                          // "auto", "manual", or "plan"
	previousMode string                          // mode saved when plan was entered; used by exit_plan_mode
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
	// connect time; injected into the session runtime.
	mcpTools []openagent.Tool

	// Cached plan entries (mirrors SessionStore._meta["plan"]). Guarded
	// by modeMu.
	planEntries []plan.Entry

	// injectedPlanTools is set to true after enter_plan_mode injects
	// plan_create + exit_plan_mode into the agent clone. Prevents
	// duplicate injection on repeated enter_plan_mode calls within the
	// same turn. Guarded by modeMu.
	injectedPlanTools bool

	// processMgr tracks background processes started by the shell tool.
	// Created on session start, cleaned up on deletion.
	processMgr *process.Manager

	// rt is the session-scoped Runtime, built once at session creation
	// (or load) and reused across turns. Per-turn changes are incremental:
	// plan tools (sender-bound closures) are rebound each prompt via
	// RemoveTools/AppendTools; mode transitions swap the tool set and the
	// approver via SetHumanApprover. Tools, skills cache, and sandbox
	// state survive across turns — the legacy per-turn Runtime rebuild is
	// gone.
	rt *kernel.Runtime

	// subAgentTools caches the delegation tools registered at runtime
	// build (kernel.New registers cfg.SubAgents as tools). Plan mode
	// removes them (no delegation); exiting plan re-appends them.
	subAgentTools []openagent.Tool

	// modeTools is the tool set injected for the CURRENT mode (read-only
	// for plan, execution set for auto/manual). applyModeTools removes
	// these by name (tracking the actual injected instances, not a
	// hard-coded whitelist) and re-injects for the new mode. Guarded by
	// modeMu (applyModeTools holds the write lock for its whole body).
	modeTools []openagent.Tool
}

// ── agentSession config/cancel/tokens/runtime accessors (modeMu-guarded) ──

// ConfigValue returns a session config option (nil when absent). Safe
// for concurrent hot-path readers.
func (ss *agentSession) ConfigValue(key openacp.SessionConfigId) any {
	ss.modeMu.RLock()
	defer ss.modeMu.RUnlock()
	return ss.config[key]
}

// ConfigString returns a config option's string value; ok is true only
// when the option exists AND holds a string.
func (ss *agentSession) ConfigString(key openacp.SessionConfigId) (string, bool) {
	v, ok := ss.ConfigValue(key).(string)
	return v, ok
}

// SetConfigValue stores a config option.
func (ss *agentSession) SetConfigValue(key openacp.SessionConfigId, val any) {
	ss.modeMu.Lock()
	defer ss.modeMu.Unlock()
	ss.config[key] = val
}

// ConfigSnapshot returns a copy of the config map for persistence. A
// copy is required: the live map is mutated by SetConfigValue while
// SessionStore.Save JSON-marshals the stored value.
func (ss *agentSession) ConfigSnapshot() map[openacp.SessionConfigId]any {
	ss.modeMu.RLock()
	defer ss.modeMu.RUnlock()
	out := make(map[openacp.SessionConfigId]any, len(ss.config))
	for k, v := range ss.config {
		out[k] = v
	}
	return out
}

// TotalTokens returns the accumulated token count.
func (ss *agentSession) TotalTokens() int {
	ss.modeMu.RLock()
	defer ss.modeMu.RUnlock()
	return ss.totalTokens
}

// setTotalTokens replaces the accumulated token count.
func (ss *agentSession) setTotalTokens(n int) {
	ss.modeMu.Lock()
	defer ss.modeMu.Unlock()
	ss.totalTokens = n
}

// addTotalTokens adds to the accumulated token count and returns the new
// total (the caller persists it to the store).
func (ss *agentSession) addTotalTokens(n int) int {
	ss.modeMu.Lock()
	defer ss.modeMu.Unlock()
	ss.totalTokens += n
	return ss.totalTokens
}

// setCancel stores the per-prompt cancel function (nil clears it).
func (ss *agentSession) setCancel(f context.CancelFunc) {
	ss.modeMu.Lock()
	defer ss.modeMu.Unlock()
	ss.cancel = f
}

// cancelPrompt invokes the current per-prompt cancel function, if any.
func (ss *agentSession) cancelPrompt() {
	ss.modeMu.RLock()
	f := ss.cancel
	ss.modeMu.RUnlock()
	if f != nil {
		f()
	}
}

// getRuntime returns the session-scoped Runtime (nil before first build).
func (ss *agentSession) getRuntime() *kernel.Runtime {
	ss.modeMu.RLock()
	defer ss.modeMu.RUnlock()
	return ss.rt
}

// setRuntime stores the session-scoped Runtime.
func (ss *agentSession) setRuntime(rt *kernel.Runtime) {
	ss.modeMu.Lock()
	defer ss.modeMu.Unlock()
	ss.rt = rt
}

// setSubAgentTools caches the delegation tool instances (kernel.New
// registers cfg.SubAgents as tools) for re-injection after plan mode.
// Guarded by modeMu: applyModeTools reads the slice.
func (ss *agentSession) setSubAgentTools(tools []openagent.Tool) {
	ss.modeMu.Lock()
	defer ss.modeMu.Unlock()
	ss.subAgentTools = tools
}

// ── agentSession plan/mode state accessors (modeMu-guarded) ──

// Mode returns the current session mode. Safe for concurrent hot-path
// readers (buildRuntimeForSession, buildDynamicContext, buildConfigOptions, ...).
func (ss *agentSession) Mode() string {
	ss.modeMu.RLock()
	defer ss.modeMu.RUnlock()
	return ss.mode
}

// PreviousMode returns the mode saved when plan was entered ("" if never).
func (ss *agentSession) PreviousMode() string {
	ss.modeMu.RLock()
	defer ss.modeMu.RUnlock()
	return ss.previousMode
}

// PlanEntries returns a deep-copy snapshot of current plan entries. Callers
// that need a snapshot consistent with the mode (e.g. gating a notification
// on mode=="plan") MUST use the WithLock helpers (SetPlanEntries/
// ApplyPlanUpdates/transitionModeLocked) rather than PlanEntries()+Mode()
// separately.
func (ss *agentSession) PlanEntries() []plan.Entry {
	ss.modeMu.RLock()
	defer ss.modeMu.RUnlock()
	return copyPlanEntries(ss.planEntries)
}

// SetPlanEntries replaces the whole planEntries slice. Returns a copy of
// the new entries (for the caller to persist/notify). Used by plan_create.
// Caller must NOT hold modeMu.
func (ss *agentSession) SetPlanEntries(entries []plan.Entry) []plan.Entry {
	ss.modeMu.Lock()
	defer ss.modeMu.Unlock()
	ss.planEntries = copyPlanEntries(entries)
	return copyPlanEntries(ss.planEntries)
}

// ApplyPlanUpdates validates-then-applies per-id status updates in place.
// If all ids are valid, the new statuses are committed and underLock is
// invoked while still holding modeMu (so the caller can send a
// mode-gated notification atomically with the mutation). Returns the
// resulting snapshot. On any unknown id, nothing is mutated.
func (ss *agentSession) ApplyPlanUpdates(updates []plan.Update, underLock func(snap []plan.Entry)) ([]plan.Entry, error) {
	ss.modeMu.Lock()
	defer ss.modeMu.Unlock()
	if len(ss.planEntries) == 0 {
		// The actionable exit depends on the mode: plan mode can create a
		// plan or leave; auto/manual cannot create (plan_create is
		// plan-only) — enter_plan_mode is the way in.
		if ss.mode == "plan" {
			return nil, fmt.Errorf("plan_update: there is no plan in the current session — create one with plan_create, or call exit_plan_mode to leave plan mode")
		}
		return nil, fmt.Errorf("plan_update: there is no plan in the current session — call enter_plan_mode to plan it first")
	}
	idxByID := make(map[string]int, len(ss.planEntries))
	for i, e := range ss.planEntries {
		idxByID[e.ID] = i
	}
	for _, u := range updates {
		if _, ok := idxByID[u.ID]; !ok {
			return nil, fmt.Errorf("plan_update: unknown step id %q", u.ID)
		}
	}
	next := copyPlanEntries(ss.planEntries)
	for _, u := range updates {
		next[idxByID[u.ID]].Status = plan.Status(u.Status)
	}
	ss.planEntries = next
	snap := copyPlanEntries(next)
	if underLock != nil {
		underLock(snap)
	}
	return snap, nil
}

// ClearPlanEntries empties the plan (used by slash /clear). Returns the
// previous snapshot.
func (ss *agentSession) ClearPlanEntries() []plan.Entry {
	ss.modeMu.Lock()
	defer ss.modeMu.Unlock()
	prev := ss.planEntries
	ss.planEntries = nil
	return prev
}

// PlanToolsInjected reports the per-turn injection gate.
func (ss *agentSession) PlanToolsInjected() bool {
	ss.modeMu.RLock()
	defer ss.modeMu.RUnlock()
	return ss.injectedPlanTools
}

// MarkPlanToolsInjected sets the injection gate. Used by enter_plan_mode
// once it has appended plan_create + exit_plan_mode to the clone.
func (ss *agentSession) MarkPlanToolsInjected() {
	ss.modeMu.Lock()
	defer ss.modeMu.Unlock()
	ss.injectedPlanTools = true
}

// ResetPlanToolsInjected clears the per-turn injection gate at turn start.
func (ss *agentSession) ResetPlanToolsInjected() {
	ss.modeMu.Lock()
	defer ss.modeMu.Unlock()
	ss.injectedPlanTools = false
}

// transitionModeLocked swaps the session mode, saving the current mode to
// previousMode when entering plan. Caller MUST hold the modeMu write lock.
func (ss *agentSession) transitionModeLocked(newMode string) {
	if newMode == "plan" && ss.mode != "plan" {
		ss.previousMode = ss.mode
	}
	ss.mode = newMode
}

// NewAgentServer creates an AgentServer wrapping the given agent.
// models maps model IDs to Model implementations; it is the single source
// of truth for model selection. The agent template's Model field is ignored.
func NewAgentServer(cfg *agent.Agent, deps kernel.Deps, store session.Store, models map[string]openagent.Model) *AgentServer {
	s := &AgentServer{
		Cfg:          cfg,
		Deps:         deps,
		Mem:          deps.SessionStore,
		Runtime:      session.NewRuntime(store, deps.SessionStore),
		Models:       models,
		modelConfigs: make(map[string]ModelConfig),
		sessions:     make(map[openacp.SessionId]*agentSession),
		MCPEnabled:   true,
	}
	s.approvalMemory = governance.NewPersistentApprovalMemory(s.Runtime)
	// One shared background extractor per server (never per run).
	if deps.Extractor == nil && deps.MemoryProvider != nil && cfg.Model != nil {
		s.Deps.Extractor = ctxpkg.NewAsyncExtractor(ctxpkg.NewLLMExtractor(func() openagent.Model { return cfg.Model }, deps.MemoryProvider))
	}
	s.cmdRegistry = s.buildCommandRegistry()
	if s.Models == nil {
		s.Models = make(map[string]openagent.Model)
	}
	// Pick the default model deterministically (sorted keys) so the fallback
	// is stable across runs, not a random map-iteration order. SetDefaultModelID
	// overrides this with settings "model" when configured.
	s.defaultModelID = firstModelIDLocked(s.Models)
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
var _ openacp.TurnTriggerUser = (*AgentServer)(nil)

// SetTurnTrigger implements openacp.TurnTriggerUser. The SDK mux injects a
// function the server calls to start an idle turn (no client prompt) — used
// when an async sub-agent completes and the model needs to process the result
// immediately, not "whenever the user comes back". The trigger acquires the
// same per-session lock as a client prompt, so idle turns and user turns are
// fully serialized.
func (s *AgentServer) SetTurnTrigger(trigger openacp.TurnTrigger) {
	s.turnTriggerMu.Lock()
	s.turnTrigger = trigger
	s.turnTriggerMu.Unlock()
}

// triggerIdleTurn calls the injected turn trigger if one is wired. Returns
// false when no trigger is available (CLI one-shot, or SDK not yet wired) —
// the caller falls back to synchronous execution in that case.
func (s *AgentServer) triggerIdleTurn(sid openacp.SessionId, text string) bool {
	s.turnTriggerMu.RLock()
	trigger := s.turnTrigger
	s.turnTriggerMu.RUnlock()
	if trigger == nil {
		return false
	}
	go trigger(sid, text)
	return true
}

// killSubAgents cancels every running async sub-agent for a session and
// clears the registry. Called on session close/delete so background goroutines
// don't outlive the session.
func (s *AgentServer) killSubAgents(ss *agentSession) {
	rt := ss.getRuntime()
	if rt == nil {
		return
	}
	if reg := rt.SubAgentRegistry(); reg != nil {
		reg.KillAll()
	}
}

// SetModel replaces or inserts a model in the registry. Used by
// runtime_set_model_config. When the model already exists, empty apiKey
// or baseURL preserve the originals; when inserting a new model, values
// are used as-is.
func (s *AgentServer) SetModel(provider, modelID, apiKey, baseURL string, maxInputTokens, maxOutputTokens int) {
	s.modelsMu.Lock()
	defer s.modelsMu.Unlock()
	key := provider + "/" + modelID
	old, exists := s.modelConfigs[key]
	if apiKey == "" {
		apiKey = old.APIKey
	}
	if baseURL == "" {
		baseURL = old.BaseURL
	}
	cw := maxInputTokens
	if cw == 0 {
		if oldModel, ok := s.Models[key]; ok {
			cw = oldModel.ContextWindow()
		}
	}
	mot := maxOutputTokens
	if mot == 0 {
		mot = old.MaxOutputTokens
	}
	if exists {
		slog.Info("acp updating model", "key", key)
	} else {
		slog.Info("acp inserting model", "key", key)
	}
	m := openai.New(apiKey, modelID, baseURL)
	if cw > 0 {
		m = m.WithContextWindow(cw)
	}
	s.Models[key] = m
	s.modelConfigs[key] = ModelConfig{
		Provider: provider, ModelID: modelID, APIKey: apiKey, BaseURL: baseURL,
		MaxInputTokens:           cw,
		MaxOutputTokens:          mot,
		InputCostPerMillion:      old.InputCostPerMillion,
		InputCacheCostPerMillion: old.InputCacheCostPerMillion,
		OutputCostPerMillion:     old.OutputCostPerMillion,
	}
}

// SetEmbedding refreshes the embedder's baseURL, apiKey, and model in
// place. Used by runtime_set_embedding_config. No-op when no memory
// provider is configured or the provider does not expose UpdateEmbedder.
func (s *AgentServer) SetEmbedding(baseURL, apiKey, model string) {
	if u, ok := s.Deps.MemoryProvider.(interface{ UpdateEmbedder(string, string, string) }); ok {
		u.UpdateEmbedder(baseURL, apiKey, model)
		slog.Info("acp embedding config updated")
	}
}

// modelIDs returns the registered model ids under modelsMu. SetModel
// (wasm runtime_set_model_config) can insert concurrently from a tool
// goroutine, so all iterations must go through this helper.
func (s *AgentServer) ModelIDs() []string {
	s.modelsMu.Lock()
	defer s.modelsMu.Unlock()
	ids := make([]string, 0, len(s.Models))
	for id := range s.Models {
		ids = append(ids, id)
	}
	return ids
}

// RemoveModel removes a model from the registry. Called by the settings
// watcher when a provider/model is removed from settings.json. Existing
// sessions referencing the removed model keep their runtime snapshot
// (rt.Model() returns the last-used model), but new sessions will not
// be able to select it.
//
// If the removed model was the default, the default falls back to the
// first remaining model (sorted keys, deterministic) so background
// components (guard/summarizer/extractor) that resolve via
// GetDefaultModelID don't get a stale key that LookupModel can't find.
func (s *AgentServer) RemoveModel(key string) {
	s.modelsMu.Lock()
	defer s.modelsMu.Unlock()
	delete(s.Models, key)
	delete(s.modelConfigs, key)
	if s.defaultModelID == key {
		s.defaultModelID = firstModelIDLocked(s.Models)
	}
}

// firstModelIDLocked returns the first model id by sorted key order, or
// "" when the registry is empty. Caller must hold modelsMu.
func firstModelIDLocked(models map[string]openagent.Model) string {
	ids := make([]string, 0, len(models))
	for id := range models {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) > 0 {
		return ids[0]
	}
	return ""
}

// lookupModel returns the model registered under id, under modelsMu.
func (s *AgentServer) LookupModel(id string) (openagent.Model, bool) {
	s.modelsMu.Lock()
	defer s.modelsMu.Unlock()
	m, ok := s.Models[id]
	return m, ok
}

// SetDefaultModelID sets the default model used when a session has not
// selected one (settings "model" wins over the first-registered fallback).
// Returns false when id is not a registered model.
func (s *AgentServer) SetDefaultModelID(id string) bool {
	s.modelsMu.Lock()
	defer s.modelsMu.Unlock()
	if _, ok := s.Models[id]; !ok {
		return false
	}
	s.defaultModelID = id
	return true
}

// getDefaultModelID returns the default model id under modelsMu
// (SetDefaultModelID runs concurrently from the serve loop).
func (s *AgentServer) GetDefaultModelID() string {
	s.modelsMu.Lock()
	defer s.modelsMu.Unlock()
	return s.defaultModelID
}

// RegisterModel stores a model's original config for SetModel fallback.
func (s *AgentServer) RegisterModel(key, provider, modelID, apiKey, baseURL string, pricing ModelPricing) {
	s.modelsMu.Lock()
	defer s.modelsMu.Unlock()
	s.modelConfigs[key] = ModelConfig{
		Provider: provider, ModelID: modelID, APIKey: apiKey, BaseURL: baseURL,
		MaxInputTokens:           pricing.MaxInputTokens,
		MaxOutputTokens:          pricing.MaxOutputTokens,
		InputCostPerMillion:      pricing.InputCostPerMillion,
		InputCacheCostPerMillion: pricing.InputCacheCostPerMillion,
		OutputCostPerMillion:     pricing.OutputCostPerMillion,
	}
}

// ModelPricing carries the per-model capability/cost metadata (from the
// settings models config) used for usage reporting. Costs are USD per 1M
// tokens (matching OpenRouter/litellm convention).
type ModelPricing struct {
	MaxInputTokens           int
	MaxOutputTokens          int
	InputCostPerMillion      float64
	InputCacheCostPerMillion float64
	OutputCostPerMillion     float64
}

// resolveModelConfig returns the provider and bare model ID for the current
// session's model selection. The session config stores the composite key
// "provider/modelID"; this looks up modelConfigs to extract both parts so
// they can be set on session.Provider / session.ModelID for runtime_* host
// exports and buildModelRequest.
func (s *AgentServer) resolveModelConfig(ss *agentSession) (provider, modelID string) {
	s.modelsMu.Lock()
	defer s.modelsMu.Unlock()
	key := s.defaultModelID
	if val, ok := ss.ConfigString("model"); ok {
		key = val
	}
	if mc, ok := s.modelConfigs[key]; ok {
		return mc.Provider, mc.ModelID
	}
	return "", key
}

// resolveSessionModel returns the model for the session's current config
// (session "model" config value wins over the server default). Unlike
// Runtime.Model() which returns the previous run's snapshot (runModel),
// this reads from the registry so it reflects SetModel updates —
// critical for oaSession.Model in OnPrompt so run()'s session.Model
// override does not defeat a mid-session model switch.
func (s *AgentServer) resolveSessionModel(ss *agentSession) openagent.Model {
	key := s.GetDefaultModelID()
	if val, ok := ss.ConfigString("model"); ok {
		key = val
	}
	m, _ := s.LookupModel(key)
	return m
}

// switchSessionModel updates the session runtime, summarizer, and
// extractor to the new model. Called from all model-switch entry points
// (set_config_option "model", slash /model) so background components
// stay in sync with the conversation model.
func (s *AgentServer) switchSessionModel(ss *agentSession, m openagent.Model) {
	if rt := ss.getRuntime(); rt != nil {
		rt.SetModel(m)
	}
	// Summarizer and extractor use dynamic model lookup (SetModelFn) in
	// ACP mode, so they pick up registry updates automatically — no need
	// to call SetModel here. rt.SetModel above updates the session runtime,
	// which is the per-session model switch.
}

// ── Client capability helpers ──
//
// These read the capabilities advertised by the client during OnInitialize.
// Each helper acquires s.mu internally — safe to call from any goroutine
// (buildRuntimeForSession, applyModeTools, buildDynamicContext, etc.).

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

func (s *AgentServer) saveMeta(ctx context.Context, id string, cwd string, kind string, meta map[string]any) {
	now := time.Now()
	info := session.SessionInfo{
		ID:        id,
		Cwd:       cwd,
		CreatedAt: now,
		UpdatedAt: now,
		Meta:      meta,
	}
	info.SetMeta("kind", kind)
	if err := s.Runtime.Save(ctx, info); err != nil {
		slog.Warn("session meta save failed", "error", err)
	}
}

// savePlan persists plan entries to SessionStore._meta["plan"].
// This is called after plan_create tool execution.
func (s *AgentServer) savePlan(ctx context.Context, sessionID string, entries []plan.Entry) {
	info, err := s.Runtime.Get(ctx, sessionID)
	if err != nil || info == nil {
		return
	}
	info.SetMeta("plan", entries)
	if err := s.Runtime.Save(ctx, *info); err != nil {
		slog.Warn("session meta save failed", "error", err)
	}
}

// loadPlan reads persisted plan entries from SessionStore._meta["plan"].
// JSON unmarshaling turns []plan.Entry into []interface{}, so we
// cannot use GetMeta[[]plan.Entry] — instead, re-marshal+unmarshal.
func (s *AgentServer) loadPlan(ctx context.Context, sessionID string) []plan.Entry {
	info, err := s.Runtime.Get(ctx, sessionID)
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
	info, err := s.Runtime.Get(ctx, sessionID)
	if err != nil || info == nil {
		return
	}
	info.SetMeta("mode", mode)
	// Persist the plan-exit target too, so a plan-mode session restored
	// after restart exits back to the pre-plan mode (not the default).
	if ss := s.getSession(openacp.SessionId(sessionID)); ss != nil {
		ss.modeMu.RLock()
		info.SetMeta("previous_mode", ss.previousMode)
		ss.modeMu.RUnlock()
	}
	if err := s.Runtime.Save(ctx, *info); err != nil {
		slog.Warn("session meta save failed", "error", err)
	}
}

// loadSessionState reads persisted mode + previousMode + createdAt +
// config options from the session metadata. Returns zero values when the
// session has no persisted state.
func (s *AgentServer) loadSessionState(ctx context.Context, sessionID string) (mode, prevMode string, createdAt time.Time, config map[openacp.SessionConfigId]any) {
	info, err := s.Runtime.Get(ctx, sessionID)
	if err != nil || info == nil || info.Meta == nil {
		return "", "", time.Time{}, nil
	}
	mode, _ = info.Meta["mode"].(string)
	prevMode, _ = info.Meta["previous_mode"].(string)
	createdAt = info.CreatedAt
	if raw, ok := info.Meta["config"].(map[string]any); ok {
		config = make(map[openacp.SessionConfigId]any, len(raw))
		for k, v := range raw {
			config[k] = v
		}
	}
	return mode, prevMode, createdAt, config
}

// saveConfig persists the session's config options to _meta["config"].
func (s *AgentServer) saveConfig(ctx context.Context, sessionID string) {
	ss := s.getSession(openacp.SessionId(sessionID))
	if ss == nil {
		return
	}
	info, err := s.Runtime.Get(ctx, sessionID)
	if err != nil || info == nil {
		return
	}
	// Snapshot under the lock: the live map mutates on every
	// SetConfigValue while Save marshals the stored value.
	info.SetMeta("config", ss.ConfigSnapshot())
	if err := s.Runtime.Save(ctx, *info); err != nil {
		slog.Warn("session config save failed", "error", err)
	}
}

// saveTotalTokens persists the accumulated total token count to
// SessionStore._meta["total_tokens"]. Called after each prompt completes
// so /context survives server restarts.
func (s *AgentServer) saveTotalTokens(ctx context.Context, sessionID string, n int) {
	info, err := s.Runtime.Get(ctx, sessionID)
	if err != nil || info == nil {
		return
	}
	info.SetMeta("total_tokens", n)
	if err := s.Runtime.Save(ctx, *info); err != nil {
		slog.Warn("session meta save failed", "error", err)
	}
}

// loadTotalTokens reads the persisted accumulated total token count from
// session meta. Returns 0 if unset or unavailable.
func (s *AgentServer) loadTotalTokens(ctx context.Context, sessionID string) int {
	info, err := s.Runtime.Get(ctx, sessionID)
	if err != nil || info == nil || info.Meta == nil {
		return 0
	}
	raw, ok := info.Meta["total_tokens"]
	if !ok {
		return 0
	}
	// JSON round-trip may yield float64; handle both int and float64.
	switch v := raw.(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}

// connectMCP connects to all configured MCP servers and returns the sessions.
// Tools are listed once at connect time and cached — the connection is
// long-lived (one connection per session lifetime).
// Failed connections are logged but not fatal — MCP is an optional enhancement.
func (s *AgentServer) connectMCP(ctx context.Context, servers []openacp.McpServer) ([]*mcp.Session, []openagent.Tool) {
	if !s.MCPEnabled {
		return nil, nil
	}
	servers = s.mergeMcpServers(servers)
	client := mcp.NewClient(s.AgentName, s.AgentVersion)
	var sessions []*mcp.Session
	var tools []openagent.Tool
	seen := make(map[string]string) // tool name → server (duplicate detection)
	for _, cfg := range servers {
		sess, err := s.connectOneMCP(ctx, client, cfg)
		if err != nil {
			mcpWarn("connect", cfg.Name, err)
			continue
		}
		sessions = append(sessions, sess)
		// Name the session so tools are "mcp__<server>__<tool>" — unique
		// across servers and self-describing to the model.
		st, err := sess.Named(cfg.Name).Tools(ctx)
		if err != nil {
			mcpWarn("list tools", cfg.Name, err)
			continue
		}
		for _, t := range st {
			name := t.Definition().Name
			if owner, dup := seen[name]; dup {
				// Two servers exposing the same tool name would make
				// tool lookup ambiguous — skip the later one.
				mcpWarnDup(name, cfg.Name, owner)
				continue
			}
			seen[name] = cfg.Name
			tools = append(tools, t)
		}
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

// mergeMcpServers merges settings-declared MCP servers with client-advertised
// ones. Client servers win on name conflict (the client is the more specific
// source — it knows the session's intent). Settings servers fill in the
// global defaults the client didn't override.
func (s *AgentServer) mergeMcpServers(client []openacp.McpServer) []openacp.McpServer {
	s.mcpMu.RLock()
	settings := s.settingsMcpServers
	s.mcpMu.RUnlock()
	if len(settings) == 0 {
		return client
	}
	seen := make(map[string]bool, len(client)+len(settings))
	merged := make([]openacp.McpServer, 0, len(client)+len(settings))
	// Client first so it wins on conflict.
	for _, m := range client {
		seen[m.Name] = true
		merged = append(merged, m)
	}
	for _, m := range settings {
		if !seen[m.Name] {
			merged = append(merged, m)
		}
	}
	return merged
}

// SetSettingsMcpServers replaces the settings-declared MCP servers. Called
// at startup and by the settings watcher on hot-reload. Safe to call
// concurrently with session creation (mergeMcpServers takes the read lock).
// Existing sessions keep their connected MCP tools; only new sessions pick
// up the change.
func (s *AgentServer) SetSettingsMcpServers(servers []openacp.McpServer) {
	s.mcpMu.Lock()
	s.settingsMcpServers = servers
	s.mcpMu.Unlock()
}

// disconnectMCP closes all MCP connections.
func (s *AgentServer) disconnectMCP(sessions []*mcp.Session) {
	for _, sess := range sessions {
		_ = sess.Close()
	}
}

// mcpWarn logs an MCP connection/setup failure to BOTH stderr (the ACP
// control pipe — surfaced to the client via Session.Stderr) and slog (the
// persisted log file). A connect failure is non-fatal (connectMCP skips
// the server and continues), but the user needs to see it in both places:
// stderr for immediate feedback in the client, slog for post-mortem in the
// server log.
func mcpWarn(op, name string, err error) {
	msg := fmt.Sprintf("acp: MCP %s %q failed: %v", op, name, err)
	fmt.Fprintln(os.Stderr, msg)
	slog.Warn("mcp setup failed", "op", op, "server", name, "error", err)
}

// mcpWarnDup logs a duplicate tool-name skip (same dual-write rationale).
func mcpWarnDup(tool, server, owner string) {
	msg := fmt.Sprintf("acp: MCP tool %q from server %q duplicates %q — skipped", tool, server, owner)
	fmt.Fprintln(os.Stderr, msg)
	slog.Warn("mcp tool duplicate skipped", "tool", tool, "server", server, "owner", owner)
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
	if ss != nil {
		ss.cancelPrompt()
	}
}

// ── openacp.AgentHandler ──

func (s *AgentServer) OnInitialize(ctx context.Context, req openacp.InitializeRequest) (*openacp.InitializeResponse, error) {
	// Persist the client's advertised capabilities so that buildRuntimeForSession
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
			Name:    s.AgentName,
			Version: s.AgentVersion,
		},
	}, nil
}

// ── Session CRUD ──

// resolveSessionCwd resolves the working directory for a session being loaded
// or resumed. If reqCwd is empty, it falls back to the persisted SessionInfo.Cwd
// from the store, then normalizes (tilde expansion + empty-cwd fallback).
func (s *AgentServer) resolveSessionCwd(ctx context.Context, sessionID, reqCwd string) string {
	cwd := reqCwd
	if cwd == "" {
		if info, err := s.Runtime.Get(ctx, sessionID); err == nil && info != nil && info.Cwd != "" {
			cwd = info.Cwd
		}
	}
	return utils.NormalizePath(cwd)
}

func (s *AgentServer) OnNewSession(ctx context.Context, req openacp.NewSessionRequest) (*openacp.NewSessionResponse, error) {
	id := s.newSessionID()
	mcpSessions, mcpTools := s.connectMCP(ctx, req.McpServers)
	cwd := utils.NormalizePath(req.Cwd)
	ss := &agentSession{
		id:                    id,
		cwd:                   cwd,
		createdAt:             time.Now(),
		mode:                  s.defaultMode(),
		config:                map[openacp.SessionConfigId]any{"thought_level": "medium", "model": s.defaultModelID},
		firstPrompt:           true,
		additionalDirectories: req.AdditionalDirectories,
		mcpServers:            req.McpServers,
		mcpSessions:           mcpSessions,
		mcpTools:              mcpTools,
	}

	// Create per-session process manager for long-running shell commands.
	if cwd != "" { // require cwd but put files in /tmp
		pm, err := process.NewManager(filepath.Join(opentool.ArtifactRoot(), "sess-"+string(id)))
		if err == nil {
			ss.processMgr = pm
		}
	}
	// Build the session-scoped Runtime once; reused for every prompt.
	ss.setRuntime(s.buildRuntimeForSession(id, ss))
	s.putSession(id, ss)
	s.saveMeta(ctx, string(id), cwd, "acp", req.Meta)

	// Send available commands and skills so the client can show them immediately.
	if s.updateSender != nil {
		s.updateSender.SendSessionUpdate(id, openacp.SessionUpdate{
			SessionUpdate:     "available_commands_update",
			AvailableCommands: s.availableCommands(),
		})
		s.updateSender.SendSessionUpdate(id, openacp.SessionUpdate{
			SessionUpdate:   "available_skills_update",
			AvailableSkills: s.availableSkills(ss),
		})
	}

	return &openacp.NewSessionResponse{
		Meta:          map[string]any{"created_at": time.Now().UTC().Format(time.RFC3339Nano)},
		SessionID:     id,
		ConfigOptions: s.buildConfigOptions(id),
		Modes:         s.buildModeState(id),
	}, nil
}

func (s *AgentServer) OnLoadSession(ctx context.Context, req openacp.LoadSessionRequest, sender openacp.SessionEventSender) (*openacp.LoadSessionResponse, error) {
	ss := s.getSession(req.SessionID)
	if ss == nil {
		mode, prevMode, createdAt, config := s.loadSessionState(ctx, string(req.SessionID))
		if mode == "" {
			mode = s.defaultMode()
		}
		if config == nil {
			config = map[openacp.SessionConfigId]any{"thought_level": "medium", "model": s.defaultModelID}
		}
		cwd := s.resolveSessionCwd(ctx, string(req.SessionID), req.Cwd)
		ss = &agentSession{
			id:                    req.SessionID,
			cwd:                   cwd,
			createdAt:             createdAt,
			mode:                  mode,
			previousMode:          prevMode,
			config:                config,
			firstPrompt:           false,
			additionalDirectories: req.AdditionalDirectories,
			mcpServers:            req.McpServers,
		}
		// Reconnect MCP servers and inject tools for this session.
		ss.mcpSessions, ss.mcpTools = s.connectMCP(ctx, req.McpServers)

		// Create per-session process manager for long-running shell commands.
		if cwd != "" {
			pm, err := process.NewManager(filepath.Join(opentool.ArtifactRoot(), "sess-"+string(req.SessionID)))
			if err == nil {
				ss.processMgr = pm
			}
		}
		// Build the session-scoped Runtime once; reused for every prompt.
		ss.setRuntime(s.buildRuntimeForSession(req.SessionID, ss))
		s.putSession(req.SessionID, ss)
	}

	// Restore accumulated token count so /context survives server restarts.
	ss.setTotalTokens(s.loadTotalTokens(ctx, string(req.SessionID)))

	// Replay history from Memory if available.
	if s.Mem != nil {
		s.replayHistory(ctx, req.SessionID, sender)
	}

	// Replay persisted plan if present.
	if entries := s.loadPlan(ctx, string(req.SessionID)); len(entries) > 0 {
		ss.SetPlanEntries(entries)
		s.replayPlan(sender, entries)
	}

	// Send available commands and skills (same as session/new).
	if s.updateSender != nil {
		s.updateSender.SendSessionUpdate(req.SessionID, openacp.SessionUpdate{
			SessionUpdate:     "available_commands_update",
			AvailableCommands: s.availableCommands(),
		})
		s.updateSender.SendSessionUpdate(req.SessionID, openacp.SessionUpdate{
			SessionUpdate:   "available_skills_update",
			AvailableSkills: s.availableSkills(ss),
		})
	}

	// Carry created_at in _meta so the frontend can display the session's
	// creation time on load (not just on live creation).
	meta := map[string]any{}
	if !ss.createdAt.IsZero() {
		meta["created_at"] = ss.createdAt.UTC().Format(time.RFC3339Nano)
	}

	return &openacp.LoadSessionResponse{
		Meta:          meta,
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
		// Carry the stored wall-clock as _meta.created_at so the frontend
		// can render per-message times from loadSession replay. nil for
		// legacy rows (pre-column or never stamped) keeps _meta off the
		// wire. Key matches the Go field / JSON tag / DB column end-to-end.
		// RFC3339Nano mirrors nowMeta so live and replay share one precision.
		var meta map[string]any
		if msg.CreatedAt != nil {
			meta = map[string]any{"created_at": msg.CreatedAt.UTC().Format(time.RFC3339Nano)}
		}
		switch msg.Role {
		case openagent.RoleUser:
			// Skip <system-reminder> messages on replay — they are injected
			// environment events (sub-agent completion notifications), not
			// user speech. Rendering them as user_message_chunk would show
			// raw <system-reminder> XML in the chat history on session load.
			trimmed := strings.TrimSpace(msg.Content)
			if strings.HasPrefix(trimmed, "<system-reminder>") && strings.HasSuffix(trimmed, "</system-reminder>") {
				continue
			}
			sender.SendHistoryMessageWithMeta("user_message_chunk", msg.Content, mid, meta)

		case openagent.RoleAssistant:
			// Replay reasoning content before the message body.
			if msg.ReasoningContent != "" {
				sender.SendHistoryMessageWithMeta("agent_thought_chunk", msg.ReasoningContent, mid, meta)
			}
			if msg.Content != "" {
				sender.SendHistoryMessageWithMeta("agent_message_chunk", msg.Content, mid, meta)
			}
			// Replay tool calls made by this assistant message.
			// Status "pending" → sessionUpdate "tool_call" variant.
			for _, tc := range msg.ToolCalls {
				sender.SendToolCallWithMeta(openacp.ToolCallUpdate{
					ToolCallID: tc.ID,
					Title:      opentool.ToolTitle(tc.Function.Name, tc.Function.Arguments),
					Kind:       "execute",
					Status:     "pending",
					RawInput:   json.RawMessage(tc.Function.Arguments),
				}, meta)
			}

		case openagent.RoleTool:
			// Tool results — send as completed tool call updates.
			sender.SendToolCallWithMeta(openacp.ToolCallUpdate{
				ToolCallID: msg.ToolCallID,
				Status:     "completed",
				RawOutput:  map[string]any{"result": msg.Content},
			}, meta)

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
		mode, prevMode, createdAt, config := s.loadSessionState(ctx, string(req.SessionID))
		if mode == "" {
			mode = s.defaultMode()
		}
		if config == nil {
			config = map[openacp.SessionConfigId]any{"thought_level": "medium", "model": s.defaultModelID}
		}
		cwd := s.resolveSessionCwd(ctx, string(req.SessionID), req.Cwd)
		ss = &agentSession{
			id:                    req.SessionID,
			cwd:                   cwd,
			createdAt:             createdAt,
			mode:                  mode,
			previousMode:          prevMode,
			config:                config,
			firstPrompt:           false,
			additionalDirectories: req.AdditionalDirectories,
			mcpServers:            req.McpServers,
		}
		// Reconnect MCP servers and inject tools for this session.
		ss.mcpSessions, ss.mcpTools = s.connectMCP(ctx, req.McpServers)

		// Create per-session process manager for shell tool background processes.
		if cwd != "" {
			pm, err := process.NewManager(filepath.Join(opentool.ArtifactRoot(), "sess-"+string(req.SessionID)))
			if err == nil {
				ss.processMgr = pm
			}
		}
		// Build the session-scoped Runtime once; reused for every prompt.
		ss.setRuntime(s.buildRuntimeForSession(req.SessionID, ss))
		s.putSession(req.SessionID, ss)
	}
	// Restore accumulated token count so /context survives server restarts.
	ss.setTotalTokens(s.loadTotalTokens(ctx, string(req.SessionID)))
	// Load persisted plan into memory (no replay per ACP spec:
	// session/resume MUST NOT replay history).
	if ss.PlanEntries() == nil {
		ss.SetPlanEntries(s.loadPlan(ctx, string(req.SessionID)))
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
		s.killSubAgents(ss)
	}
	s.removeSession(req.SessionID)
	return &openacp.CloseSessionResponse{}, nil
}

func (s *AgentServer) OnDeleteSession(ctx context.Context, req openacp.DeleteSessionRequest) (*openacp.DeleteSessionResponse, error) {
	ss := s.getSession(req.SessionID)
	if ss != nil {
		s.disconnectMCP(ss.mcpSessions)
		s.killSubAgents(ss)
		if ss.processMgr != nil {
			ss.processMgr.Cleanup()
		}
	}
	s.removeSession(req.SessionID)
	// Runtime.Delete removes metadata and messages together.
	if err := s.Runtime.Delete(ctx, string(req.SessionID)); err != nil {
		slog.Warn("session delete failed", "session", req.SessionID, "error", err)
	}
	return &openacp.DeleteSessionResponse{}, nil
}

func (s *AgentServer) OnListSessions(ctx context.Context, req openacp.ListSessionsRequest) (*openacp.ListSessionsResponse, error) {
	list, err := s.Runtime.List(ctx)
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
	modelID := s.GetDefaultModelID()
	if ss != nil {
		mode = ss.Mode()
		if val, ok := ss.ConfigString("thought_level"); ok {
			thoughtLevel = val
		}
		if val, ok := ss.ConfigString("model"); ok {
			modelID = val
		}
	}

	opts := []openacp.SessionConfigOption{
		{
			ID:           "mode",
			Name:         "Session Mode",
			Description:  "Controls the autonomy and safety boundaries of AI",
			Category:     "mode",
			Type:         "select",
			CurrentValue: mode,
			Options: []openacp.SessionConfigOptValue{
				{Value: "auto", Name: "Auto", Description: "Fully automated processing (HIGH RISK), AI will NOT seek your approval"},
				{Value: "manual", Name: "Manual", Description: "Your approval is required for AI to perform NONE-READ-ONLY operations"},
				{Value: "plan", Name: "Plan", Description: "Present the plan first, AI will execute it according to the plan"},
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
	if ids := s.ModelIDs(); len(ids) > 0 {
		modelOpts := make([]openacp.SessionConfigOptValue, 0, len(ids))
		for _, id := range ids {
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
	current := s.defaultMode()
	if ss != nil {
		current = ss.Mode()
	}
	return &openacp.SessionModeState{
		CurrentModeID: openacp.SessionModeId(current),
		AvailableModes: []openacp.SessionMode{
			{ID: "auto", Name: "Auto", Description: "Fully automated processing (HIGH RISK), AI will NOT seek your approval"},
			{ID: "manual", Name: "Manual", Description: "Your approval is required for AI to perform NONE-READ-ONLY operations"},
			{ID: "plan", Name: "Plan", Description: "Present the plan first, AI will execute it according to the plan"},
		},
	}
}

// BroadcastConfigOptions sends a config_option_update to every active
// session so the frontend picks up model list changes (e.g. after a
// settings reload added/removed models via SetModel). Sessions without
// an updateSender (CLI one-shot) are skipped.
func (s *AgentServer) BroadcastConfigOptions() {
	if s.updateSender == nil {
		return
	}
	s.mu.Lock()
	ids := make([]openacp.SessionId, 0, len(s.sessions))
	for sid := range s.sessions {
		ids = append(ids, sid)
	}
	s.mu.Unlock()
	for _, sid := range ids {
		s.updateSender.SendSessionUpdate(sid, openacp.SessionUpdate{
			SessionUpdate: "config_option_update",
			ConfigOptions: s.buildConfigOptions(sid),
		})
	}
}

// BroadcastSystemReminder sends a <system-reminder> to all active sessions
// via triggerIdleTurn. Used by the settings watcher to notify the model that
// settings.json was externally modified. triggerIdleTurn sends an
// "idle_turn_end" session/update after the turn ends so the frontend knows
// the idle turn is over and can re-enable user input.
func (s *AgentServer) BroadcastSystemReminder(text string) {
	s.mu.Lock()
	ids := make([]openacp.SessionId, 0, len(s.sessions))
	for sid := range s.sessions {
		ids = append(ids, sid)
	}
	s.mu.Unlock()
	for _, sid := range ids {
		s.triggerIdleTurn(sid, text)
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

// buildSessionSkillProvider creates a skill provider scoped to the session's
// cwd. Project-level skills resolve from <cwd>/.agents/skills instead of the
// server process's cwd. Global (~/.agents/skills) and builtin (embed) sources
// are unaffected by cwd.
func (s *AgentServer) buildSessionSkillProvider(cwd string) skill.Provider {
	var roots []fs.RootEntry
	// Always register this root (do NOT os.Stat it at session-creation
	// time). The directory may not exist yet — the user may install a
	// skill later via npx/CLI, which creates <cwd>/.agents/skills. If we
	// skip the root when the dir is absent at creation, reload_skills
	// (which calls Discover on the already-built roots) will never scan
	// the new directory. Discover handles a missing directory gracefully
	// (returns no skills from that root).

	// global: ~/.agents/skills
	if home, err := os.UserHomeDir(); err == nil {
		d := filepath.Join(home, ".agents", "skills")
		roots = append(roots, fs.RootEntry{Path: d, Type: "global"})
	}
	// project: <session-cwd>/.agents/skills
	if cwd != "" {
		d := filepath.Join(cwd, ".agents", "skills")
		roots = append(roots, fs.RootEntry{Path: d, Type: "project"})
	}
	embedFS := builtinskills.BuiltinFS()
	if len(roots) == 0 && embedFS == nil {
		return s.Deps.SkillProvider // fallback to server-level provider
	}
	loader := fs.NewWithSources(roots...)
	if embedFS != nil {
		loader = loader.WithEmbedFS(embedFS)
	}
	return skill.NewFSBridge(loader)
}

// availableSkills returns the skill catalog for the client to render a
// skill panel or @skill autocomplete. Discovers from the session's
// SkillProvider (per-session cwd) when available, falling back to the
// server-level provider.
func (s *AgentServer) availableSkills(ss *agentSession) []openacp.AvailableSkill {
	var sp skill.Provider
	if rt := ss.getRuntime(); rt != nil {
		sp = rt.SkillProvider()
	}
	if sp == nil {
		sp = s.Deps.SkillProvider
	}
	if sp == nil {
		return nil
	}
	skills, err := sp.Discover(context.Background())
	if err != nil || len(skills) == 0 {
		return nil
	}
	out := make([]openacp.AvailableSkill, len(skills))
	for i, sk := range skills {
		// Builtin skills have no disk path; leave Path empty so the
		// frontend can distinguish by Type alone.
		path := sk.Path
		if sk.Type == "builtin" {
			path = ""
		}
		out[i] = openacp.AvailableSkill{
			Name:        sk.Name,
			Description: sk.Description,
			Path:        path,
			Type:        sk.Type,
		}
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

	// Swap mode under the lock; the no-op "already in plan" early return
	// keeps previousMode from being clobbered on a redundant re-enter.
	ss.modeMu.Lock()
	if mode == "plan" && ss.mode == "plan" {
		ss.modeMu.Unlock()
		return nil
	}
	ss.transitionModeLocked(mode)
	ss.modeMu.Unlock()

	// Persist + notify OUTSIDE the lock. Notifications go to the ACP
	// single-writer queue (non-blocking), so total mode-change ordering
	// relative to plan notifications emitted by concurrent callbacks is
	// preserved by the FIFO queue — see exit_plan_mode helper.
	s.persistAndNotifyMode(ctx, sid, mode)

	// Swap the permanent tool set + approver on the session runtime
	// (plan ⇄ auto/manual). The skill cache and sandbox state survive.
	s.applyModeTools(sid, ss, ss.getRuntime())
	return nil
}

// persistAndNotifyMode persists the mode to the session store and sends
// the current_mode_update + config_option_update notifications. It does
// NOT swap the in-memory mode (already done by transitionModeLocked).
// Split out so exit_plan_mode can skip the swap (it flips mode itself
// while taking the empty-plan notification under the lock) and still
// persist + notify.
func (s *AgentServer) persistAndNotifyMode(ctx context.Context, sid openacp.SessionId, mode string) {
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
}

// enterPlanMode transitions the session into plan mode. Called by
// enter_plan_mode's onEnter callback. The mode change takes effect
// immediately (persisted + client notified), but the agent clone's tools
// are not mutated — the next OnPrompt turn picks up plan mode and
// registers plan_create + exit_plan_mode.
func (s *AgentServer) enterPlanMode(ctx context.Context, sid openacp.SessionId, ss *agentSession) error {
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
			ss.SetConfigValue(req.ConfigID, b)
		}
	default:
		if val, ok := req.Value.(string); ok {
			ss.SetConfigValue(req.ConfigID, val)
		}
	}

	// Persist the updated options so a restored session keeps them.
	s.saveConfig(ctx, string(req.SessionID))

	// Sync session mode when the client sets the "mode" config option
	// (most clients use set_config_option rather than set_mode).
	// setSessionMode now sends both current_mode_update and
	// config_option_update, so skip the duplicate notification below.
	needsConfigUpdate := true
	if req.ConfigID == "mode" {
		if v, ok := ss.ConfigString("mode"); ok {
			_ = s.setSessionMode(ctx, req.SessionID, v)
			needsConfigUpdate = false // already sent by setSessionMode
		}
	}

	// Live-sync model / reasoning-effort onto the session runtime so the
	// next prompt uses the new config without a runtime rebuild.
	if rt := ss.getRuntime(); rt != nil {
		switch req.ConfigID {
		case "model":
			if v, ok := ss.ConfigString("model"); ok {
				if m, ok := s.LookupModel(v); ok {
					s.switchSessionModel(ss, m)
				} else {
					// The requested model is not in the provider list —
					// keep the current model, but don't stay silent.
					slog.Warn("requested model not in provider list, keeping current", "session", req.SessionID, "model", v)
				}
			}
		case "thought_level":
			if v, ok := ss.ConfigString("thought_level"); ok && v != "" {
				rt.SetReasoningEffort(v)
			}
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
	// Reset per-turn injectedPlanTools flag so enter_plan_mode
	// can inject plan_create + exit_plan_mode again this turn.
	ss.ResetPlanToolsInjected()
	ctx, cancel := context.WithCancel(ctx)
	ss.setCancel(cancel)
	defer func() {
		ss.setCancel(nil)
		cancel()
	}()

	// ── Intercept server-side slash commands ──
	// Must run BEFORE auto-title so slash commands don't get used as
	// the session title (e.g. "/mode plan" would become the title).
	// The command round-trip NEVER enters the conversation store: slash
	// commands are control operations (industry: Claude Code keeps
	// /context-style commands out of the context budget), so they must
	// not consume working tokens nor leak into the compressed summary.
	if resp, handled := s.cmdRegistry.Handle(s.buildSlashContext(ctx, req.SessionID, ss), input.Content); handled {
		sender.SendAgentMessage(resp)
		return &openacp.PromptResponse{StopReason: openacp.StopReasonEndTurn}, nil
	}

	// ── Auto-title from first user message ──
	if ss.firstPrompt {
		ss.firstPrompt = false
		// Generate a concise title via LLM — it picks the right language
		// (matching the user's) and a short descriptive label, no truncation.
		// Falls back to firstLine if the model call fails or times out.
		// Uses a temporary title immediately so the UI isn't blank while
		// the LLM call runs; updates to the real title when it returns.
		fallback := firstLine(input.Content, 80)
		s.updateTitle(ctx, req.SessionID, fallback)
		sender.SendSessionInfo(fallback, nil)
		sender.SendAvailableCommands(s.availableCommands())
		sender.SendAvailableSkills(s.availableSkills(ss))

		// Async LLM title generation — don't block the turn for it.
		if m := s.resolveSessionModel(ss); m != nil {
			go s.generateTitle(req.SessionID, m, input.Content, fallback)
		} else {
			slog.Warn("title generation skipped: no model resolved", "session", req.SessionID)
		}
	}

	// ── Session-scoped Runtime, reused across turns ──
	// Built once at session creation; per-turn changes are incremental
	// (plan tools rebinding below, mode transitions via applyModeTools).
	// The skill cache, sandbox state, and tool set survive across turns.
	agent := ss.getRuntime()
	if agent == nil {
		agent = s.buildRuntimeForSession(req.SessionID, ss)
		ss.setRuntime(agent)
	}

	providerID, modelID := s.resolveModelConfig(ss)
	oaSession := openagent.Session{
		ID:       string(req.SessionID),
		ModelID:  modelID,
		Provider: providerID,
		// Carry the resolved Model instance so downstream consumers
		// (RunHooks via SessionFromContext, e.g. the artifact hook's
		// context-window threshold) can read ContextWindow() without
		// depending on every call site to re-resolve from ModelID.
		// Use resolveSessionModel (reads the registry) instead of
		// agent.Model() (returns the previous run's runModel snapshot)
		// so run()'s session.Model override does not defeat a
		// mid-session SetModel update.
		Model:     s.resolveSessionModel(ss),
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
		rt := wasm.BuildAgentRuntime(agent, &oaSession, s.SetModel, s.SetEmbedding)
		ctx = wasmhost.WithAgentRuntime(ctx, rt)
	}

	// Inject ProcessManager so the shell tool can persist
	// long-running process output across turns.
	if ss.processMgr != nil {
		ctx = process.WithManager(ctx, ss.processMgr)
	}

	// ── Rebind per-prompt plan tools ──
	// plan_create/plan_update/exit_plan_mode/enter_plan_mode close over the
	// per-prompt sender, so they are removed and re-appended each prompt.
	s.reconcilePlanTools(ctx, req.SessionID, ss, agent, sender)
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
					sender.SendToolCall(openacp.ToolCallUpdate{
						ToolCallID: tc.ID,
						Title:      opentool.ToolTitle(tc.Function.Name, tc.Function.Arguments),
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
			// Structured failure first; text-prefix kept as a fallback for
			// tools that don't populate Result.
			status := "completed"
			if (evt.Message.Result != nil && evt.Message.Result.Error != nil) ||
				strings.HasPrefix(evt.Message.Content, "error: ") ||
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

		case openagent.StreamSkillsUpdated:
			// reload_skills discovered a new skill set (install/uninstall
			// on disk). Push the updated catalog to the client so the
			// frontend skill panel refreshes in real time.
			skills := make([]openacp.AvailableSkill, len(evt.Skills))
			for i, sk := range evt.Skills {
				path := sk.Path
				if sk.Type == "builtin" {
					path = ""
				}
				skills[i] = openacp.AvailableSkill{
					Name:        sk.Name,
					Description: sk.Description,
					Path:        path,
					Type:        sk.Type,
				}
			}
			sender.SendAvailableSkills(skills)

		case openagent.StreamCompacting:
			// "context_compacting" — compaction started, client shows a status.
			if evt.Compaction != nil && s.updateSender != nil {
				s.updateSender.SendSessionUpdate(req.SessionID, openacp.SessionUpdate{
					SessionUpdate: "context_compacting",
					Meta: map[string]any{
						"overflow_tokens": evt.Compaction.OverflowTokens,
						"total_messages":  evt.Compaction.TotalMessages,
					},
				})
			}

		case openagent.StreamCompacted:
			// "context_compacted" — compaction finished, client updates usage.
			if evt.Compaction != nil && s.updateSender != nil {
				s.updateSender.SendSessionUpdate(req.SessionID, openacp.SessionUpdate{
					SessionUpdate: "context_compacted",
					Meta: map[string]any{
						"compressed_messages": evt.Compaction.CompressedMessages,
						"freed_tokens":        evt.Compaction.FreedTokens,
						"error":               evt.Compaction.Error,
					},
				})
			}

		case openagent.StreamDone:
			if evt.Result != nil {
				usage = evt.Result.Usage
				stopReason = finishReasonToACP(evt.Result.StopReason)
			}

		case openagent.StreamError:
			return nil, evt.Error

		case openagent.StreamAborted:
			return &openacp.PromptResponse{StopReason: openacp.StopReasonCancelled, Meta: map[string]any{"mode": ss.Mode()}}, nil
		}
	}

	// Checkpoint: recoverable restore point after a completed run.
	if s.Runtime != nil {
		cpCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := s.Runtime.Checkpoint(cpCtx, string(req.SessionID)); err != nil {
			slog.Warn("session checkpoint failed", "session", req.SessionID, "error", err)
		}
	}

	total := ss.addTotalTokens(usage.TotalTokens)
	s.saveTotalTokens(ctx, string(req.SessionID), total)

	// Report *current* context usage (this turn's PromptTokens), not
	// accumulated total. Per ACP spec, `used` means "tokens currently
	// in context" — PromptTokens is the best proxy for that.
	if usage.PromptTokens > 0 {
		cw := 0
		if m := agent.Model(); m != nil {
			cw = m.ContextWindow()
		}
		sender.SendUsageUpdate(usage.PromptTokens, cw, s.usageCost(ss, usage))
	}

	if ctx.Err() != nil {
		return &openacp.PromptResponse{StopReason: openacp.StopReasonCancelled, Meta: map[string]any{"mode": ss.Mode()}}, nil
	}
	if stopReason == "" {
		stopReason = openacp.StopReasonEndTurn
	}
	return &openacp.PromptResponse{StopReason: stopReason, Meta: map[string]any{"mode": ss.Mode()}}, nil
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
	if ss != nil {
		ss.cancelPrompt()
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
	if title == "" {
		return
	}
	info, err := s.Runtime.Get(ctx, string(sessionID))
	if err != nil || info == nil {
		return
	}
	if info.Title == "" {
		info.Title = title
		if err := s.Runtime.Save(ctx, *info); err != nil {
			slog.Warn("session meta save failed", "error", err)
		}
	}
}

// generateTitle calls the LLM to produce a concise session title from the
// first user message. It runs in a goroutine with a 1-minute timeout; on
// success it updates the session title and pushes sessionInfo to all
// subscribers. On failure or timeout it keeps the fallback title. The LLM
// picks the right language (matching the user's) and a short label — no
// truncation needed.
func (s *AgentServer) generateTitle(sid openacp.SessionId, model openagent.Model, userMessage, fallback string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	slog.Info("title generation started", "session", sid, "fallback", fallback)
	resp, err := model.ChatCompletion(ctx, openagent.ChatCompletionRequest{
		Messages: []openagent.Message{
			{Role: openagent.RoleSystem, Content: "You are a conversation title generator. " +
				"A user has just started a new chat session. Below is their first message. " +
				"Generate a concise title (2-10 words) that captures the topic or theme of this conversation. " +
				"The title should reflect what the user wants to discuss, not answer their message. " +
				"Use the same language as the user's message. " +
				"Output ONLY the title — no quotes, no explanation, no trailing punctuation."},
			{Role: openagent.RoleUser, Content: userMessage},
		},
	})
	if err != nil {
		slog.Warn("title generation failed", "session", sid, "error", err)
		return
	}
	if len(resp.Choices) == 0 {
		slog.Warn("title generation no choices", "session", sid, "usage", resp.Usage)
		return
	}
	content := resp.Choices[0].Message.Content
	if content == "" {
		slog.Warn("title generation empty content", "session", sid, "finish_reason", resp.Choices[0].FinishReason, "usage", resp.Usage)
		return
	}
	title := strings.TrimSpace(content)
	// Strip quotes if the model wrapped the title in them.
	title = strings.Trim(title, `"'`)
	if title == "" || title == fallback {
		return
	}
	slog.Info("title generation success", "session", sid, "title", title)

	// Update the persisted title and notify subscribers.
	updateCtx, updateCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer updateCancel()
	s.renameSession(updateCtx, sid, title)
	if s.updateSender != nil {
		s.updateSender.SendSessionUpdate(sid, openacp.SessionUpdate{
			SessionUpdate: "session_info_update",
			Title:         &title,
		})
	}
}

// buildRuntimeForSession builds the session-scoped Runtime once, at session
// creation or load. It carries the permanent tool set (execution tools for
// auto/manual, read-only tools for plan) plus the session's model, prompts,
// and approval wiring. Plan tools (plan_create/plan_update/exit_plan_mode/
// enter_plan_mode) are NOT injected here — they close over the per-prompt
// sender and are rebound each turn by reconcilePlanTools.
func (s *AgentServer) buildRuntimeForSession(sid openacp.SessionId, ss *agentSession) *kernel.Runtime {
	cfg := s.Cfg.Clone()
	deps := s.Deps
	deps.Tools = nil // tools are injected per mode via applyModeTools
	deps.HumanApprover = nil
	// Session-mode tools are excluded from sub-agent tool sets: their
	// callbacks are session-bound and would be nil in the child runtime.
	// settings is excluded too — settings.json is global config (may carry
	// apikeys and other secrets); only the parent agent (in direct contact
	// with the user) should read or modify it, never a delegated sub-agent.
	deps.SubAgentExcludeTools = []string{
		"plan_create", "plan_update",
		"enter_plan_mode", "exit_plan_mode",
		"settings",
	}
	// Share the server-level persisted approval memory so the policy
	// chain's Memory layer recalls "always allow" decisions across turns
	// (written by acpApprover.always into the same instance).
	deps.ApprovalMemory = s.approvalMemory

	// Build a per-session skill provider so project-level skills
	// (<cwd>/.agents/skills) resolve against the session's working
	// directory, not the server process's cwd. Global and builtin
	// sources are unaffected by cwd.
	deps.SkillProvider = s.buildSessionSkillProvider(ss.cwd)

	// Resolve model from the session config registry.
	modelID := s.GetDefaultModelID()
	if val, ok := ss.ConfigString("model"); ok {
		modelID = val
	}
	if m, ok := s.LookupModel(modelID); ok {
		cfg.Model = m
	} else if m, ok := s.LookupModel(s.GetDefaultModelID()); ok {
		// The session's saved model is no longer in the provider list
		// (removed/renamed after the session was saved) — fall back to the
		// default instead of leaving cfg.Model nil, which would make a
		// restored session fail every turn with "no model configured".
		if v, _ := ss.ConfigString("model"); v != "" && v != s.GetDefaultModelID() {
			slog.Warn("session model not in provider list, falling back to default", "session", sid, "model", v, "default", s.GetDefaultModelID())
		}
		cfg.Model = m
	}
	if v, ok := ss.ConfigString("thought_level"); ok && v != "" {
		cfg.ReasoningEffort = v
	}

	// Override system prompts with session-cwd-aware profiles so each
	// session picks up SOUL/SYSTEM/AGENTS from its own project directory.
	if s.ProfileResolver != nil && ss.cwd != "" {
		if prompts := s.ProfileResolver(ss.cwd); len(prompts) > 0 {
			cfg.SystemPrompts = prompts
		}
	}

	// Sub-agents are always registered (kernel.New turns cfg.SubAgents into
	// delegation tools); applyModeTools removes them in plan mode and
	// re-appends the cached instances in execution modes. Plan mode never
	// clears them from the config, so a plan→auto transition can restore
	// delegation.
	rt := kernel.New(cfg, deps)

	// Cache delegation tools for re-injection after plan mode.
	// Guarded by modeMu: applyModeTools reads the slice during a
	// concurrent mode transition.
	if len(cfg.SubAgents) > 0 {
		want := subAgentToolNames(cfg)
		var cached []openagent.Tool
		for _, t := range rt.SnapshotTools() {
			for _, n := range want {
				if t.Definition().Name == n {
					cached = append(cached, t)
				}
			}
		}
		ss.setSubAgentTools(cached)
		// Wire async sub-agent completion: when a background sub-agent
		// finishes, trigger an idle turn via the SDK mux's TriggerTurn
		// (fully serialized with user turns via sessionLocks). The note
		// becomes the prompt input so the model processes the result
		// immediately, not "whenever the user comes back".
		if reg := rt.SubAgentRegistry(); reg != nil {
			reg.SetOnExit(func(note string) {
				s.triggerIdleTurn(sid, note)
			})
		}
	}

	// Initial mode tool set + approver.
	s.applyModeTools(sid, ss, rt)
	return rt
}

// applyModeTools swaps the permanent tool set and approver for the
// session's CURRENT mode. It removes the previously injected mode tools
// (tracked in ss.modeTools — the actual instances, not a hard-coded
// whitelist) plus the cached sub-agent tools (plan has no delegation),
// then injects the mode-appropriate set and records it for the next
// transition. The runtime's skill cache and sandbox state survive.
//
// Called at runtime build and on every mode transition. Plan tools are
// NOT touched here — they close over the per-prompt sender and are
// rebound each prompt by reconcilePlanTools.
func (s *AgentServer) applyModeTools(sid openacp.SessionId, ss *agentSession, rt *kernel.Runtime) {
	// Hold modeMu for the whole body: applyModeTools runs from the serve
	// loop (set_mode / set_config_option "mode"), the prompt goroutine
	// (slash /mode), and tool callbacks (enter/exit_plan_mode), all of
	// which can interleave with a concurrent mode transition. modeTools
	// and subAgentTools are read/written under the lock; ss.mode is read
	// directly (Mode() would re-lock and self-deadlock). No I/O happens
	// under the lock — rt.RemoveTools/AppendTools/SetHumanApprover are
	// in-memory, so the lock-order comment on modeMu holds.
	ss.modeMu.Lock()
	defer ss.modeMu.Unlock()

	// Drop the previous mode's tools by the names we actually injected.
	drop := toolNames(ss.modeTools)
	drop = append(drop, subAgentToolNames(rt.Config())...)
	rt.RemoveTools(drop...)
	ss.modeTools = nil

	var add []openagent.Tool
	switch ss.mode {
	case "plan":
		// Read-only tools only — local read/ls/grep plus client
		// read_client_file, so the model can inspect the workspace while
		// planning. No execution tools, no approver (no side effects to
		// approve), no delegation.
		//
		// Non-read-only tools are NOT dropped: they are replaced by
		// stubs that explain the situation. A dropped tool surfaces as a
		// bare `tool "shell" not found`, which leaves the model guessing
		// why shell vanished; the stub tells it to exit plan mode.
		if s.ToolFactory != nil && ss.cwd != "" {
			for _, t := range s.ToolFactory(ss.cwd) {
				if readOnlyToolNames[t.Definition().Name] {
					add = append(add, t)
				} else {
					add = append(add, planModeStub{def: t.Definition()})
				}
			}
		}
		if s.clientRPC != nil && s.clientCanReadFile() {
			add = append(add, opentool.NewACPReadFile(s.clientRPC, sid))
		}
		rt.SetHumanApprover(nil)

	default:
		// Auto/manual: full tool set. Auto has no approval prompts (safety
		// is handled by Guard.in/Guard.out if configured); manual routes
		// EVERY tool call through the ACP approver — including read-only
		// tools (no Safety layer, so nothing auto-approves). "Always allow"
		// decisions still shortcut through the approval memory; handoffs
		// stay free.
		if s.clientRPC != nil && s.clientCanReadFile() {
			add = append(add, opentool.NewACPReadFile(s.clientRPC, sid))
		}
		add = append(add, s.executionTools(sid, ss)...)
		add = append(add, ss.subAgentTools...)

		if ss.mode == "manual" && s.clientRPC != nil {
			rt.SetHumanApprover(&acpApprover{client: s.clientRPC, sessionID: sid, memory: s.approvalMemory})
		} else {
			rt.SetHumanApprover(nil)
		}
	}

	ss.modeTools = add
	rt.AppendTools(add...)
}

// toolNames extracts the Definition().Name of each tool in the slice.
func toolNames(tools []openagent.Tool) []string {
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		names = append(names, t.Definition().Name)
	}
	return names
}

// subAgentToolNames returns the delegation tool names for a config's
// sub-agents (registered as tools by kernel.New), plus "sub_agent_send" when
// delegation tools exist — the follow-up tool is registered alongside them
// (kernel.New) and must be cached/dropped/re-injected in lockstep with them
// across plan-mode transitions (applyModeTools).
func subAgentToolNames(cfg *agent.Agent) []string {
	var names []string
	for _, sa := range cfg.SubAgents {
		names = append(names, sa.Name)
	}
	if len(cfg.SubAgents) > 0 {
		names = append(names, "sub_agent_send")
	}
	return names
}

// reconcilePlanTools rebinds the per-prompt plan tools (plan_create,
// plan_update, exit_plan_mode, enter_plan_mode) to the current sender.
// Called at the top of every prompt. The tools close over the sender and
// session state, so they cannot live on the session-scoped runtime.
func (s *AgentServer) reconcilePlanTools(ctx context.Context, sid openacp.SessionId, ss *agentSession, rt *kernel.Runtime, sender openacp.SessionEventSender) {
	rt.RemoveTools("plan_create", "plan_update", "enter_plan_mode", "exit_plan_mode")

	// plan_update is always registered so it can track plan progress in
	// all modes (plan/auto/manual) — including the same prompt that just
	// ran plan_create (reconcile runs once per OnPrompt, not per turn).
	// ApplyPlanUpdates validates-then-mutates under modeMu and runs the
	// notification in the same critical section, so the notified snapshot
	// is consistent with the mutation and ordered relative to a concurrent
	// exit_plan_mode's empty-plan notification. A call with no plan yet
	// gets an actionable error ("create one first"), never a confusing
	// "unknown step id".
	pu := plan.NewUpdateTool(func(updates []plan.Update) ([]plan.Entry, error) {
		snap, err := ss.ApplyPlanUpdates(updates, func(snap []plan.Entry) {
			// Called with modeMu held (ApplyPlanUpdates) — read the field
			// directly, Mode() would re-lock and self-deadlock.
			sender.SendPlanUpdate(s.entriesToACP(snap))
		})
		if err != nil {
			return nil, err
		}
		s.savePlan(ctx, string(sid), snap)
		return snap, nil
	})
	rt.AppendTools(pu)

	switch ss.Mode() {
	case "plan":
		// plan_create + exit_plan_mode are injected in OnPrompt — not in
		// buildRuntimeForSession — because they need closures over the
		// session and sender.
		rt.AppendTools(plan.NewCreateTool(s.makeCreateCallback(ctx, sid, ss, sender)))
		rt.AppendTools(plan.NewExitTool(s.makeExitCallback(ctx, sid, ss, rt, sender)))

	default:
		// enter_plan_mode: available in auto and manual mode. Changes the
		// session mode to "plan" and immediately injects plan_create +
		// exit_plan_mode into the agent clone so they are available this
		// same turn.
		rt.AppendTools(plan.NewEnterTool(func() error {
			wasPlan := ss.Mode() == "plan"
			if err := s.setSessionMode(ctx, sid, "plan"); err != nil {
				return err
			}
			// Inject plan_create + exit_plan_mode on the FIRST transition
			// into plan mode within this turn. The injection block is
			// serialized under modeMu so two concurrent enter_plan_mode
			// calls in one model response make the second a no-op (the
			// flag is already set) and only one injection happens.
			ss.modeMu.Lock()
			if !wasPlan && !ss.injectedPlanTools {
				ss.injectedPlanTools = true
				ss.modeMu.Unlock()
				rt.AppendTools(
					plan.NewCreateTool(s.makeCreateCallback(ctx, sid, ss, sender)))
				rt.AppendTools(
					plan.NewExitTool(s.makeExitCallback(ctx, sid, ss, rt, sender)))
			} else {
				ss.modeMu.Unlock()
			}
			return nil
		}))
	}
}

// readOnlyToolNames is the plan-mode tool whitelist — a single source:
// the platform ToolClassifier's read-only set (the same list the policy
// chain's Safety layer auto-allows). A second copy here drifted from
// governance's list, so plan mode could offer a tool the policy chain
// treats as dangerous (or vice versa).
var readOnlyToolNames = governance.NewToolClassifier().ReadOnlyNames

// injectExecutionTools appends all execution-capable tools to the agent
// clone. Called in manual mode and after exit_plan_mode transitions.
// Mirrors the original flat injection — MCP tools, ToolFactory tools,
// and Agent→Client RPC tools all go through here. The injection goes
// through AppendTools (toolsMu-guarded): this runs inside exit_plan_mode's
// callback, which executes within an executeTools parallel-goroutine
// batch, so sibling tool goroutines read the clone's Tools via the
// runner's SnapshotTools/findTool concurrently with this append.
func (s *AgentServer) executionTools(sid openacp.SessionId, ss *agentSession) []openagent.Tool {
	var add []openagent.Tool

	// MCP tools from connected servers.
	add = append(add, ss.mcpTools...)

	// Per-turn tools scoped to the session cwd.
	if s.ToolFactory != nil && ss.cwd != "" {
		if tools := s.ToolFactory(ss.cwd); len(tools) > 0 {
			add = append(add, tools...)
		}
	}

	// Agent→Client RPC tools (write/terminal only — read_client_file is
	// added by buildRuntimeForSession to avoid duplicate registration across modes).
	// Each tool is gated on the corresponding client capability so the
	// LLM is never offered a tool whose RPC the client will reject.
	if s.clientRPC != nil {
		if s.clientCanWriteFile() {
			add = append(add,
				opentool.NewACPWriteFile(s.clientRPC, sid),
			)
		}
		if s.clientCanTerminal() {
			add = append(add,
				opentool.NewACPTerminalCreate(s.clientRPC, sid),
				opentool.NewACPTerminalOutput(s.clientRPC, sid),
				opentool.NewACPTerminalWait(s.clientRPC, sid),
				opentool.NewACPTerminalKill(s.clientRPC, sid),
				opentool.NewACPTerminalRelease(s.clientRPC, sid),
			)
		}
	}

	return add
}

// makeCreateCallback builds the OnPlan callback shared by the plan-mode
// plan_create tool and the enter_plan_mode-injected plan_create tool.
// It atomically swaps ss.planEntries under modeMu, sends the plan update
// notification (so the panel refreshes even in auto/manual mode after an
// enter_plan_mode-triggered plan_create), then persists outside the lock.
// Shared by both injection sites so there is one canonical create path.
func (s *AgentServer) makeCreateCallback(
	ctx context.Context, sid openacp.SessionId, ss *agentSession,
	sender openacp.SessionEventSender,
) plan.OnPlan {
	return func(entries []plan.Entry) {
		ss.modeMu.Lock()
		ss.planEntries = copyPlanEntries(entries)
		snap := copyPlanEntries(ss.planEntries)
		sender.SendPlanUpdate(s.entriesToACP(snap))
		ss.modeMu.Unlock()
		s.savePlan(ctx, string(sid), snap)
	}
}

// makeExitCallback builds the onExit callback shared by the plan-mode
// exit_plan_mode tool and the enter_plan_mode-injected exit_plan_mode
// tool. Idempotent under concurrent exit calls: a second concurrent exit
// sees mode != "plan" (already flipped by the first) and no-ops. The
// mode flip + empty-plan notification happen under modeMu so concurrent
// plan_create/plan_update closures either fully see mode=="plan" (and
// notify their entries, which the FIFO writer emits first) or fully see
// mode!=plan (and skip). The mode-change notifications + execution-tool
// injection + approver wiring happen after unlock.
func (s *AgentServer) makeExitCallback(
	ctx context.Context, sid openacp.SessionId, ss *agentSession,
	rt *kernel.Runtime, sender openacp.SessionEventSender,
) func() error {
	return func() error {
		target := ss.PreviousMode()
		if target == "" || target == "plan" {
			target = "auto"
		}

		// Flip mode and clear the plan panel atomically with any
		// concurrent plan_create/plan_update notification.
		ss.modeMu.Lock()
		if ss.mode != "plan" {
			// Already exited (concurrent exit_plan_mode won the race).
			ss.modeMu.Unlock()
			return nil
		}
		ss.transitionModeLocked(target)
		sender.SendPlanUpdate(nil) // clear panel before mode-change notif
		ss.modeMu.Unlock()

		// Persist + notify (current_mode_update + config_option_update).
		s.persistAndNotifyMode(ctx, sid, target)

		// Swap the permanent tool set + approver for subsequent model calls
		// THIS turn. Safe: rt.AppendTools/RemoveTools are lock-guarded, and
		// the next executeTools batch reads the updated tool set.
		s.applyModeTools(sid, ss, ss.getRuntime())
		return nil
	}
}

// buildDynamicContext assembles per-turn dynamic context from session
// runtime state — plan entries with status, mode instruction, etc.
// Called every turn in OnPrompt; injected into the system prompt via
// Session.DynamicContext → PromptInput → BuildPrompt.
func (s *AgentServer) buildDynamicContext(ss *agentSession) string {
	var b strings.Builder

	// Take one consistent snapshot of the plan entries for the whole
	// render; a torn read between the count and loop has no safety impact
	// (worst case: a one-turn-stale plan block in the system prompt).
	entries := ss.PlanEntries()
	mode := ss.Mode()

	// ── Plan entries with current status ──
	if len(entries) > 0 {
		b.WriteString("## Current Plan\n")
		for _, e := range entries {
			fmt.Fprintf(&b, "- [%s] [%s] %s\n", e.Priority, e.Status, e.Content)
		}
		b.WriteString("\nUpdate plan status with plan_update when starting or completing each step.\n\n")
	}

	// ── Mode instruction ──
	if mode == "plan" {
		b.WriteString("## Plan Mode\n")
		b.WriteString("You are in **plan mode**. You have NO execution tools — you cannot modify files, run shell commands, or create terminals.\n\n")
		b.WriteString("Your available tools:\n")
		b.WriteString("- read, ls, grep — read and search workspace files\n")
		b.WriteString("- webfetch, websearch — fetch web content and search the internet\n")
		b.WriteString("- recall — search conversation history for details not covered by the summary\n")
		b.WriteString("- load_skill, reload_skills — load and manage skill definitions\n")
		if s.clientCanReadFile() {
			b.WriteString("- read_client_file — read files from your machine\n")
		}
		b.WriteString("- plan_create, plan_update, exit_plan_mode — create, update, and exit an execution plan\n")
		b.WriteString("\n**Workflow:**\n")
		b.WriteString("1. Explore — read relevant files, search the web, and recall context to understand the task\n")
		b.WriteString("2. Call plan_create with concrete, actionable steps\n")
		b.WriteString("3. Call exit_plan_mode to leave plan mode and begin execution\n\n")
		b.WriteString("Create a complete plan before calling exit_plan_mode.\n")
	} else if len(entries) == 0 {
		// Auto/manual mode without a plan: hint about enter_plan_mode.
		b.WriteString("## Task Planning\n")
		b.WriteString("If this task is complex (involves multiple steps, multiple files, or requires careful sequencing), consider calling **enter_plan_mode** first. This will give you access to plan_create for structured planning. After creating a plan, call exit_plan_mode to regain your execution tools and work through the plan.\n\n")
	}

	return b.String()
}

// buildSlashContext constructs the slash.Context for command dispatch.
func (s *AgentServer) buildSlashContext(ctx context.Context, sid openacp.SessionId, ss *agentSession) slash.Context {
	return slash.Context{
		SessionID:   string(sid),
		Cwd:         ss.cwd,
		Mode:        ss.Mode(),
		TotalTokens: ss.TotalTokens(),
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
			ss.setTotalTokens(0)
			s.saveTotalTokens(ctx, string(sid), 0)
			ss.ClearPlanEntries()
			s.savePlan(ctx, string(sid), nil)
			return nil
		},
		ListSess: func() ([]slash.SessionInfo, error) {
			list, err := s.Runtime.List(ctx)
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
			m, ok := s.LookupModel(modelID)
			if !ok {
				return fmt.Errorf("unknown model: %s", modelID)
			}
			ss.SetConfigValue("model", modelID)
			s.switchSessionModel(ss, m)
			s.saveConfig(ctx, string(sid))
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
			return s.ModelIDs()
		},
		// Both callbacks touch the session runtime, which is built lazily
		// on the first prompt (slash dispatch runs BEFORE that build). Build
		// on demand so /compact and /context work on a fresh session too.
		Compact: func() (*slash.CompactStats, error) {
			rt := ss.getRuntime()
			if rt == nil {
				rt = s.buildRuntimeForSession(sid, ss)
				ss.setRuntime(rt)
			}
			// Notify the client that compaction is starting.
			if s.updateSender != nil {
				s.updateSender.SendSessionUpdate(sid, openacp.SessionUpdate{
					SessionUpdate: "context_compacting",
					Meta:          map[string]any{},
				})
			}
			st, err := rt.CompressAll(ctx, string(sid))
			if err != nil {
				// Notify the client that compaction failed.
				if s.updateSender != nil {
					s.updateSender.SendSessionUpdate(sid, openacp.SessionUpdate{
						SessionUpdate: "context_compacted",
						Meta:          map[string]any{"error": err.Error()},
					})
				}
				return nil, err
			}
			if st == nil {
				return &slash.CompactStats{}, nil // no compressor configured
			}
			// Notify the client that compaction finished.
			if s.updateSender != nil {
				s.updateSender.SendSessionUpdate(sid, openacp.SessionUpdate{
					SessionUpdate: "context_compacted",
					Meta: map[string]any{
						"compressed_messages": st.Compressed,
						"freed_tokens":        st.FreedTokens,
					},
				})
			}
			return &slash.CompactStats{
				Compressed:    st.Compressed,
				FreedTokens:   st.FreedTokens,
				SummaryTokens: st.SummaryTokens,
			}, nil
		},
		ContextStats: func() (*slash.ContextStats, error) {
			rt := ss.getRuntime()
			if rt == nil {
				rt = s.buildRuntimeForSession(sid, ss)
				ss.setRuntime(rt)
			}
			summary, working, window, err := rt.ContextUsage(ctx, string(sid))
			if err != nil {
				return nil, err
			}
			return &slash.ContextStats{
				SummaryTokens: summary,
				WorkingTokens: working,
				Window:        window,
			}, nil
		},
	}
}

// renameSession persists a new title and sends a session_info_update.
func (s *AgentServer) renameSession(ctx context.Context, sid openacp.SessionId, title string) error {
	info, err := s.Runtime.Get(ctx, string(sid))
	if err != nil || info == nil {
		return fmt.Errorf("session not found")
	}
	info.Title = title
	if err := s.Runtime.Save(ctx, *info); err != nil {
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
	memory    governance.ApprovalMemory // session-scoped "allow always" persistence
}

// Ask implements governance.HumanApprover. The ACP permission response
// carries allow/deny (and optionally modified input); the policy engine
// treats Ask as the human layer.
func (a *acpApprover) Ask(ctx context.Context, call openagent.ToolCall, def openagent.FunctionDefinition, session openagent.Session) (governance.Decision, error) {
	if a.client == nil {
		// Fail closed: no approval channel must mean denied, not silently
		// allowed — an unapprovable call should never auto-execute.
		return governance.Decision{Action: governance.Deny, Reason: "no approval client configured"}, nil
	}
	resp, err := a.client.RequestPermission(ctx, openacp.RequestPermissionRequest{
		SessionID: a.sessionID,
		ToolCall: openacp.ToolCallUpdate{
			ToolCallID: call.ID,
			Title:      opentool.ToolTitle(def.Name, call.Function.Arguments),
			Kind:       "execute",
			Status:     "pending",
			RawInput:   json.RawMessage(call.Function.Arguments),
		},
		// ACP semantics: allow_once = this call only (never remembered),
		// allow_always = remembered for the session. For shell, the grant
		// covers the command's atoms and file accesses (all of them must
		// be remembered to skip approval — see governance.MemoryKeys), so
		// a changed command or a new file target re-asks while reused
		// ones don't. Cross-session rules are a separate configuration
		// layer, not a button grant.
		Options: []openacp.PermissionOption{
			{OptionID: "allow_once", Name: "Allow Once", Kind: openacp.PermissionAllowOnce},
			{OptionID: "allow_always", Name: "Allow Always", Kind: openacp.PermissionAllowAlways},
			{OptionID: "reject_once", Name: "Reject", Kind: openacp.PermissionRejectOnce},
		},
	})
	if err != nil {
		return governance.Decision{Action: governance.Deny, Reason: "permission request failed: " + err.Error()}, nil
	}
	// Outcome is the ACP union discriminant. The spec requires it, but
	// legacy clients predate it and send only {"optionId":"..."}; those
	// are accepted leniently as "selected" so they keep working. An
	// explicit "cancelled" (or an empty response with no OptionID) is
	// denied — fail closed on ambiguous input.
	switch resp.Outcome.Outcome {
	case openacp.PermissionOutcomeCancelled:
		return governance.Decision{Action: governance.Deny, Reason: "cancelled"}, nil
	case openacp.PermissionOutcomeSelected, "":
		// "selected" is the normal path; "" is the legacy lenient path.
	default:
		return governance.Decision{Action: governance.Deny, Reason: "unknown outcome: " + string(resp.Outcome.Outcome)}, nil
	}
	if resp.Outcome.OptionID == nil {
		return governance.Decision{Action: governance.Deny, Reason: "no option selected"}, nil
	}
	switch *resp.Outcome.OptionID {
	case "allow_once":
		// Deliberately NOT remembered (ACP allow_once semantics): the
		// same tool + args asks again next time — a session-level grant
		// is what "Allow Always" is for.
		return governance.Decision{Action: governance.Allow, Reason: "allow once"}, nil
	case "allow_always":
		// Session-scoped (ACP allow_always semantics): the same tool +
		// args no longer asks within this session. NOT persisted across
		// sessions — a cross-session rules layer (settings → governance
		// Rule) is future work, not a button grant.
		d := governance.Decision{Action: governance.Allow, Reason: "allow always"}
		if a.memory != nil {
			// Multi-key tools (shell command atoms + file accesses,
			// write target) remember every key — the policy chain later
			// requires ALL of them to skip approval.
			keys := governance.MemoryKeys(call.Function.Name, json.RawMessage(call.Function.Arguments))
			if len(keys) == 0 {
				keys = []string{governance.ApprovalKey(call.Function.Name, json.RawMessage(call.Function.Arguments))}
			}
			for _, key := range keys {
				if err := a.memory.Remember(ctx, session.ID, key, d); err != nil {
					slog.Warn("approval always persistence failed", "session", session.ID, "error", err)
				}
			}
		}
		return d, nil
	case "reject_once":
		reason := "rejected by user"
		if fb, ok := resp.Outcome.Meta["feedback"].(string); ok && fb != "" {
			reason = fb
		}
		return governance.Decision{Action: governance.Deny, Reason: reason}, nil
	default:
		return governance.Decision{Action: governance.Deny, Reason: "unknown option: " + *resp.Outcome.OptionID}, nil
	}
}

// usageCost computes the USD cost of a run's usage from the model's
// configured per-token pricing. Unconfigured rates are 0 (free): a model
// without pricing reports cost 0 — an explicit value, not an absent one.
func (s *AgentServer) usageCost(ss *agentSession, usage openagent.Usage) *openacp.Cost {
	s.modelsMu.Lock()
	key := s.defaultModelID
	if val, ok := ss.ConfigString("model"); ok {
		key = val
	}
	mc := s.modelConfigs[key]
	s.modelsMu.Unlock()
	cached := usage.CacheReadTokens
	if cached > usage.PromptTokens {
		cached = usage.PromptTokens
	}
	uncached := usage.PromptTokens - cached
	// Costs are USD per 1M tokens; divide by 1e6 to get the per-token rate.
	const perMillion = 1_000_000
	amount := (float64(uncached)*mc.InputCostPerMillion +
		float64(cached)*mc.InputCacheCostPerMillion +
		float64(usage.CompletionTokens)*mc.OutputCostPerMillion) / perMillion
	return &openacp.Cost{Amount: amount, Currency: "USD"}
}

// firstLine truncates s to the first line, up to maxLen characters.
func firstLine(s string, maxLen int) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[:idx]
	}
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
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

// planModeStub replaces an execution tool while the session is in plan
// mode. The tool keeps its name (the model may still reference it) but
// every call returns an actionable error: the model is in plan mode and
// must call exit_plan_mode to regain execution tools.
// planModeStub replaces an execution tool while the session is in plan
// mode. The DEFINITION is the original tool's — the model sees shell/
// write/... exactly as in normal mode, with the same name, description,
// and parameter schema. Only the execution result changes: every call
// returns an actionable error telling the model to exit plan mode.
type planModeStub struct {
	def openagent.FunctionDefinition
}

func (p planModeStub) Definition() openagent.FunctionDefinition {
	return p.def
}

func (p planModeStub) Execute(_ context.Context, _ json.RawMessage) *openagent.ToolResult {
	return openagent.ErrorResult(fmt.Errorf(
		"%s is unavailable in plan mode — you have no execution tools. "+
			"Call exit_plan_mode to leave plan mode and regain execution tools", p.def.Name), false, "")
}

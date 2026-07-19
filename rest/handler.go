package rest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/eventbus"
	"github.com/yusheng-g/openagent-go/session"
)

// ── Handler ──

// Handler serves a REST API for an openagent-go Agent.
//
// Create with [NewHandler], then register on an [http.ServeMux]:
//
//	handler := rest.NewHandler(agent)
//	mux := http.NewServeMux()
//	handler.Register(mux)
//	http.ListenAndServe(":8080", mux)
// ModelInfo describes a registered model for the frontend.
type ModelInfo struct {
	ID       string `json:"id"`
	Provider string `json:"provider,omitempty"`
}

type Handler struct {
	defaultModel openagent.Model
	models       map[string]openagent.Model // modelID → model instance
	modelList    []ModelInfo                // ordered list for /models endpoint
	modelsMu     sync.RWMutex

	tools        []openagent.Tool
	systemPrompts []string
	name         string
	maxTurns     int

	sm *sessionManager[*sessionState] // session CRUD, store, bus
}

// NewHandler creates a Handler from a configured Agent.
// The agent's Model, Memory, Tools, Instructions, Name, and MaxTurns
// are captured as the template for per-session Agent instances.
func NewHandler(agent *openagent.Agent) *Handler {
	h := &Handler{
		defaultModel: agent.Model,
		models:       make(map[string]openagent.Model),
		modelList:    nil,
		tools:        agent.Tools,
		systemPrompts: agent.SystemPrompts,
		name:         agent.Name,
		maxTurns:     agent.MaxTurns,
	}

	bus := eventbus.New[SSEEvent](500)
	h.sm = newSessionManager[*sessionState](nil, agent.Memory, bus, sessionHooks[*sessionState]{
		kind:       "single",
		newEntry:   h.newEntry,
		fillDetail: h.fillDetail,
	})

	return h
}

// fillDetail enriches the SessionDetail with per-handler runtime fields
// (ContextWindow from the agent's model).
func (h *Handler) fillDetail(e *sessionState, detail *SessionDetail) {
	if e.agent != nil && e.agent.Model != nil {
		detail.ContextWindow = e.agent.Model.ContextWindow()
	}
}

// RegisterModel adds a model to the handler's registry.
// id is the string the frontend sends as modelID (e.g. "deepseek-v3").
// provider identifies which API serves this model (e.g. "deepseek", "openai").
// The internal key is "provider:id" so different providers can serve the same model name.
func (h *Handler) RegisterModel(id string, model openagent.Model, provider string) {
	h.modelsMu.Lock()
	defer h.modelsMu.Unlock()
	h.models[provider+":"+id] = model
	h.modelList = append(h.modelList, ModelInfo{ID: id, Provider: provider})
}

// lookupModel finds a registered model. When provider is non-empty, it uses
// the exact composite key "provider:modelId". Otherwise it scans all registered
// models for matching modelId — this handles the common case where the frontend
// sends only modelId without provider.
func (h *Handler) lookupModel(provider, modelID string) openagent.Model {
	h.modelsMu.RLock()
	defer h.modelsMu.RUnlock()
	if provider != "" {
		return h.models[provider+":"+modelID]
	}
	for key, m := range h.models {
		if key == "default" {
			continue
		}
		if strings.HasSuffix(key, ":"+modelID) {
			return m
		}
	}
	return nil
}

// Register adds the handler's routes to mux using Go 1.22+ patterns.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /sessions", h.handleCreateSession)
	mux.HandleFunc("GET /sessions", h.handleListSessions)
	mux.HandleFunc("GET /sessions/{id}", h.handleGetSession)
	mux.HandleFunc("PATCH /sessions/{id}", h.handleUpdateSession)
	mux.HandleFunc("GET /sessions/{id}/messages", h.handleListMessages)
	mux.HandleFunc("DELETE /sessions/{id}", h.handleDeleteSession)
	mux.HandleFunc("POST /sessions/{id}/chat", h.handleChat)
	mux.HandleFunc("POST /sessions/{id}/approve", h.handleApprove)
	mux.HandleFunc("GET /models", h.handleListModels)
}

// WithSessionStore attaches a persistent session metadata store.
// nil (the default) preserves the current in-memory-only behavior.
func (h *Handler) WithSessionStore(s session.Store) *Handler {
	h.sm.SetStore(s)
	return h
}

// StartJanitor starts a background goroutine that evicts idle session entries.
// See sessionManager.StartJanitor for semantics.
func (h *Handler) StartJanitor(ctx context.Context, interval, maxIdle time.Duration) {
	h.sm.StartJanitor(ctx, interval, maxIdle)
}

// WithCleanupDir registers a callback that is invoked when a session is
// deleted (either via DELETE /sessions/{id} or the idle janitor). Use it
// to clean up per-session temp/artifact directories.
func (h *Handler) WithCleanupDir(fn func(sessionID string)) *Handler {
	h.sm.SetCleanupDir(fn)
	return h
}

// ── sessionState ──

// sessionState holds the per-session runtime state.
// Events are published to the Handler-level bus via sm so that multiple
// SSE connections (e.g. browser tabs) all receive the full stream.
type sessionState struct {
	info       session.SessionInfo // ModelID is the session's model preference; empty → handler default
	agent      *openagent.Agent

	mu              sync.Mutex
	running         bool             // true while agent goroutine is active
	pendingApproval *pendingApproval
}

func (s *sessionState) sessionInfo() *session.SessionInfo { return &s.info }

// isActive reports whether the session has an ongoing agent run
// or is awaiting tool approval. Eviction skips active sessions.
func (s *sessionState) isActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running || s.pendingApproval != nil
}

type pendingApproval struct {
	respond chan approveResponse
}

type approveResponse struct {
	allowed bool
	reason  string
}

// ── Session CRUD handlers ──

func (h *Handler) handleCreateSession(w http.ResponseWriter, r *http.Request) { h.sm.create(w, r) }
func (h *Handler) handleListSessions(w http.ResponseWriter, r *http.Request)  { h.sm.list(w, r) }
func (h *Handler) handleGetSession(w http.ResponseWriter, r *http.Request)    { h.sm.get(w, r) }
func (h *Handler) handleUpdateSession(w http.ResponseWriter, r *http.Request) { h.sm.update(w, r) }
func (h *Handler) handleDeleteSession(w http.ResponseWriter, r *http.Request) { h.sm.del(w, r) }

func (h *Handler) handleListMessages(w http.ResponseWriter, r *http.Request) { h.sm.messages(w, r) }

// parseIntParam parses an integer query parameter with bounds.
func parseIntParam(r *http.Request, name string, min, max int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return 0, fmt.Errorf("missing")
	}
	var n int
	if _, err := fmt.Sscanf(raw, "%d", &n); err != nil {
		return 0, fmt.Errorf("invalid integer")
	}
	if n < min {
		n = min
	}
	if n > max {
		n = max
	}
	return n, nil
}

// ── Chat handler ──

func (h *Handler) handleChat(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Message == "" {
		http.Error(w, `{"error":"message is required"}`, http.StatusBadRequest)
		return
	}

	s := h.sm.getOrCreate(id)

	// Reset pending approval for the new chat message.
	s.mu.Lock()
	s.running = true
	s.pendingApproval = nil
	s.mu.Unlock()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
		return
	}
	setSSEHeaders(w)
	flusher.Flush() // flush headers immediately so the client sees streaming start

	// Subscribe to the session's event bus. Live-only — history is NOT
	// replayed because this is a new chat, not a reconnection. Replaying
	// old "done" events would cause the handler to return before the
	// current chat's events arrive.
	sub := h.sm.Bus().SubscribeLive(id)
	defer h.sm.Bus().Unsubscribe(id, sub)

	// Resolve model: chat-level override > session default > handler default.
	provider := body.Provider
	modelID := body.ModelID
	if provider == "" && modelID == "" {
		h.sm.withMeta(id, func(inf *session.SessionInfo) {
			p, _ := session.GetMeta[string](*inf, "provider")
			m, _ := session.GetMeta[string](*inf, "modelId")
			provider = p
			modelID = m
		})
	}
	// Composite key "provider:modelId" for exact match.
	// When provider is empty, find the first registered model for the given ID.
	model := h.lookupModel(provider, modelID)
	if model == nil {
		model = h.defaultModel
	}

	// Persist the resolved model so GET /sessions reflects the actual model.
	if inf, ok := h.sm.withMeta(id, func(inf *session.SessionInfo) {
		inf.SetMeta("modelId", modelID)
		inf.SetMeta("provider", provider)
	}); ok {
		h.sm.syncMeta(inf)
	}

	// Start the agent run in a background goroutine.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		defer func() {
			s.mu.Lock()
			s.running = false
			s.pendingApproval = nil
			s.mu.Unlock()
		}()

		oaSession := openagent.Session{
			ID:        id,
			ModelID:   modelID,
			Model:     model,
			CreatedAt: s.info.CreatedAt,
		}

		ch := s.agent.RunStream(ctx, oaSession, openagent.UserMessage(body.Message))
		for evt := range ch {
			se := streamToSSE(evt)
			select {
			case <-r.Context().Done():
				// Client disconnected — stop publishing.
				// Agent continues with its own ctx; timeout cleans up.
				return
			default:
			}
			h.sm.Bus().Publish(id, se)
		}
	}()

	// Stream events to the SSE response until done/error/disconnect.
	for se := range sub.C {
		if err := writeSSE(w, flusher, se); err != nil {
			return
		}
		if se.Type == "done" || se.Type == "error" {
			return
		}
	}
}

// ── Approve handler ──

func (h *Handler) handleApprove(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body ApproveRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"allowed is required"}`, http.StatusBadRequest)
		return
	}

	s := h.sm.getOrCreate(id)

	s.mu.Lock()
	p := s.pendingApproval
	s.pendingApproval = nil
	s.mu.Unlock()

	if p == nil {
		http.Error(w, `{"error":"no pending approval"}`, http.StatusBadRequest)
		return
	}

	reason := "denied"
	if body.Feedback != "" {
		reason = "denied: " + body.Feedback
	}
	if body.Allowed {
		reason = "approved"
	}
	p.respond <- approveResponse{allowed: body.Allowed, reason: reason}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": reason})
}

// ── Models ──

func (h *Handler) handleListModels(w http.ResponseWriter, r *http.Request) {
	h.modelsMu.RLock()
	models := make([]ModelInfo, len(h.modelList))
	copy(models, h.modelList)
	h.modelsMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"models": models})
}

// ── Factory ──

// newEntry creates a fresh sessionState from session.SessionInfo.
// Used by sessionManager when creating or restoring sessions.
func (h *Handler) newEntry(info session.SessionInfo) *sessionState {
	s := &sessionState{info: info}

	s.agent = openagent.NewAgent(h.name,
		openagent.WithModel(h.defaultModel),
		openagent.WithMemory(h.sm.Memory()),
		openagent.WithTools(h.tools...),
		openagent.WithSystemPrompts(h.systemPrompts...),
		openagent.WithMaxTurns(h.maxTurns),
		openagent.WithRunObserver(&stageObserver{bus: h.sm.Bus(), sid: info.ID}),
		openagent.WithApprover(&restApprover{
			submit: func(call openagent.ToolCall, resp chan approveResponse) {
				h.submitApproval(s, call, resp)
			},
		}),
	)

	return s
}

// ── Approval bridge ──

type restApprover struct {
	submit func(call openagent.ToolCall, resp chan approveResponse)
}

func (a *restApprover) Approve(ctx context.Context, call openagent.ToolCall, def openagent.FunctionDefinition, session openagent.Session) (bool, string) {
	resp := make(chan approveResponse, 1)
	a.submit(call, resp)

	select {
	case <-ctx.Done():
		return false, "cancelled"
	case r := <-resp:
		return r.allowed, r.reason
	}
}

func (h *Handler) submitApproval(s *sessionState, call openagent.ToolCall, resp chan approveResponse) {
	tcj := &SSEToolCall{
		ID: call.ID,
		Function: SSEToolCallFunction{
			Name:      call.Function.Name,
			Arguments: call.Function.Arguments,
		},
	}

	evt := SSEEvent{
		Type:     "tool_approval",
		ToolCall: tcj,
	}

	s.mu.Lock()
	s.pendingApproval = &pendingApproval{respond: resp}
	s.mu.Unlock()

	h.sm.Bus().Publish(s.info.ID, evt)
}

// ── SSE conversion ──

func streamToSSE(evt openagent.StreamEvent) SSEEvent {
	switch evt.Type {
	case openagent.StreamThought:
		return SSEEvent{Type: "thought", Text: evt.Text}

	case openagent.StreamTextDelta:
		return SSEEvent{Type: "text_delta", Text: evt.Text}

	case openagent.StreamToolCall:
		tc := evt.Message.ToolCalls[0]
		return SSEEvent{
			Type: "tool_call",
			ToolCall: &SSEToolCall{
				ID: tc.ID,
				Function: SSEToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			},
		}

	case openagent.StreamToolResult:
		return SSEEvent{
			Type:       "tool_result",
			ToolCallID: evt.Message.ToolCallID,
			Text:       evt.Message.Content,
		}

	case openagent.StreamRetrying:
		msg := "retrying"
		if evt.Error != nil {
			msg = evt.Error.Error()
		}
		return SSEEvent{Type: "retrying", Text: msg}

	case openagent.StreamToolProgress:
		return SSEEvent{Type: "tool_progress", Text: evt.Text, ToolCallID: evt.ToolCallID}

	case openagent.StreamAborted:
		se := SSEEvent{Type: "aborted"}
		if evt.Error != nil {
			se.Text = evt.Error.Error()
		}
		return se

	case openagent.StreamDone:
		se := SSEEvent{Type: "done"}
		if evt.Result != nil {
			se.FinalOutput = evt.Result.FinalOutput
			se.PromptTokens = evt.Result.Usage.PromptTokens
			se.ContextWindow = evt.Result.ContextWindow
		}
		return se

	case openagent.StreamError:
		msg := "unknown error"
		if evt.Error != nil {
			msg = evt.Error.Error()
		}
		return SSEEvent{Type: "error", Text: msg}

	default:
		log.Printf("rest: unknown stream event type %q", evt.Type)
		return SSEEvent{Type: "unknown"}
	}
}

// ── Helpers ──

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// stageObserver publishes pipeline stage events to the SSE bus so
// frontends can render a live pipeline visualization.
type stageObserver struct {
	bus *eventbus.Bus[SSEEvent]
	sid string
}

func (o *stageObserver) ObserveStage(ctx context.Context, evt openagent.StageEvent) {
	sd := struct {
		Name       string         `json:"name"`
		Phase      string         `json:"phase"`
		Detail     map[string]any `json:"detail,omitempty"`
		DurationMs int64          `json:"duration_ms,omitempty"`
		Err        string         `json:"error,omitempty"`
	}{
		Name:   evt.Name,
		Phase:  evt.Phase,
		Detail: evt.Detail,
	}
	if evt.Phase == "leave" {
		sd.DurationMs = evt.Duration.Milliseconds()
	}
	if evt.Err != nil {
		sd.Err = evt.Err.Error()
	}
	b, err := json.Marshal(sd)
	if err != nil {
		o.bus.Publish(o.sid, SSEEvent{Type: "error", Text: "stage marshal failed: " + err.Error()})
		return
	}
	o.bus.Publish(o.sid, SSEEvent{Type: "stage", Stage: b})
}

var _ openagent.RunObserver = (*stageObserver)(nil)

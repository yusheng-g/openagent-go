package rest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/eventbus"
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
type Handler struct {
	model        openagent.Model
	memory       openagent.Memory
	tools        []openagent.Tool
	instructions string
	name         string
	maxTurns     int

	bus *eventbus.Bus[SSEEvent]

	mu       sync.RWMutex
	sessions map[string]*sessionState
}

// NewHandler creates a Handler from a configured Agent.
// The agent's Model, Memory, Tools, Instructions, Name, and MaxTurns
// are captured as the template for per-session Agent instances.
func NewHandler(agent *openagent.Agent) *Handler {
	return &Handler{
		model:        agent.Model,
		memory:       agent.Memory,
		tools:        agent.Tools,
		instructions: agent.Instructions,
		name:         agent.Name,
		maxTurns:     agent.MaxTurns,
		bus:          eventbus.New[SSEEvent](500),
		sessions:     make(map[string]*sessionState),
	}
}

// Register adds the handler's routes to mux using Go 1.22+ patterns.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /sessions", h.handleCreateSession)
	mux.HandleFunc("GET /sessions", h.handleListSessions)
	mux.HandleFunc("GET /sessions/{id}", h.handleGetSession)
	mux.HandleFunc("DELETE /sessions/{id}", h.handleDeleteSession)
	mux.HandleFunc("POST /sessions/{id}/chat", h.handleChat)
	mux.HandleFunc("POST /sessions/{id}/approve", h.handleApprove)
}

// ── sessionState ──

// sessionState holds the per-session runtime state.
// Events are published to the Handler-level bus so that multiple
// SSE connections (e.g. browser tabs) all receive the full stream.
type sessionState struct {
	info    SessionInfo
	agent   *openagent.Agent
	modelID string // session default model (empty = use agent's model)

	mu              sync.Mutex
	pendingApproval *pendingApproval
}

type pendingApproval struct {
	respond chan approveResponse
}

type approveResponse struct {
	allowed bool
	reason  string
}

// ── Session CRUD handlers ──

func (h *Handler) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}

	id := generateID()
	now := time.Now()
	agentName := req.AgentName
	if agentName == "" {
		agentName = h.name
	}

	info := SessionInfo{
		ID:        id,
		Title:     req.Title,
		AgentName: agentName,
		ModelID:   req.ModelID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	h.mu.Lock()
	h.sessions[id] = h.newSession(info, req.ModelID)
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(info)
}

func (h *Handler) handleListSessions(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	list := make([]SessionInfo, 0, len(h.sessions))
	for _, s := range h.sessions {
		list = append(list, s.info)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (h *Handler) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	h.mu.RLock()
	s, ok := h.sessions[id]
	h.mu.RUnlock()

	if !ok {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.info)
}

func (h *Handler) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	h.mu.Lock()
	delete(h.sessions, id)
	h.mu.Unlock()

	if h.memory != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		_ = h.memory.DeleteSession(ctx, id)
	}

	w.WriteHeader(http.StatusNoContent)
}

// ── Chat handler ──

func (h *Handler) handleChat(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Message == "" {
		http.Error(w, `{"error":"message is required"}`, http.StatusBadRequest)
		return
	}

	s := h.getOrCreateSession(id)

	// Reset pending approval for the new chat message.
	s.mu.Lock()
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
	sub := h.bus.SubscribeLive(id)
	defer h.bus.Unsubscribe(id, sub)

	// Resolve model: chat-level override > session default.
	modelID := body.ModelID
	if modelID == "" {
		modelID = s.modelID
	}

	// Start the agent run in a background goroutine.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		oaSession := openagent.Session{
			ID:        id,
			AgentName: s.info.AgentName,
			ModelID:   modelID,
			CreatedAt: s.info.CreatedAt,
		}

		ch := s.agent.RunStream(ctx, oaSession, openagent.UserMessage(body.Message))
		for evt := range ch {
			se := streamToSSE(evt)
			h.bus.Publish(id, se)
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

	h.mu.RLock()
	s, ok := h.sessions[id]
	h.mu.RUnlock()

	if !ok {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	s.mu.Lock()
	p := s.pendingApproval
	s.pendingApproval = nil
	s.mu.Unlock()

	if p == nil {
		http.Error(w, `{"error":"no pending approval"}`, http.StatusBadRequest)
		return
	}

	reason := "denied"
	if body.Allowed {
		reason = "approved"
	}
	p.respond <- approveResponse{allowed: body.Allowed, reason: reason}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": reason})
}

// ── Session management ──

func (h *Handler) getOrCreateSession(id string) *sessionState {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s, ok := h.sessions[id]; ok {
		return s
	}
	info := SessionInfo{
		ID:        id,
		AgentName: h.name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s := h.newSession(info, "")
	h.sessions[id] = s
	return s
}

func (h *Handler) newSession(info SessionInfo, modelID string) *sessionState {
	s := &sessionState{
		info:    info,
		modelID: modelID,
	}

	s.agent = openagent.NewAgent(info.AgentName,
		openagent.WithModel(h.model),
		openagent.WithMemory(h.memory),
		openagent.WithTools(h.tools...),
		openagent.WithInstructions(h.instructions),
		openagent.WithMaxTurns(h.maxTurns),
			openagent.WithRunObserver(&stageObserver{bus: h.bus, sid: info.ID}),
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

	h.bus.Publish(s.info.ID, evt)
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
		return SSEEvent{}
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
	b, _ := json.Marshal(sd)
	o.bus.Publish(o.sid, SSEEvent{Type: "stage", Stage: b})
}

var _ openagent.RunObserver = (*stageObserver)(nil)

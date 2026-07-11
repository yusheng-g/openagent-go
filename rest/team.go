package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/eventbus"
)

// ── TeamHandler ──

// TeamHandler serves a REST API for an openagent-go Team.
//
// Create with [NewTeamHandler], then register on an [http.ServeMux]:
//
//	templates := []rest.TeamAgentTemplate{
//	    {Name: "researcher", Description: "Finds facts", Agent: researcher},
//	    {Name: "writer", Description: "Writes reports", Agent: writer},
//	}
//	th := rest.NewTeamHandler(mem, templates...)
//	th.Register(mux)
type TeamHandler struct {
	agents []TeamAgentTemplate
	memory openagent.Memory
	model  openagent.Model // from first template, used by dynamically added agents

	bus *eventbus.Bus[SSEEvent]

	mu       sync.RWMutex
	sessions map[string]*teamSessionState
}

// TeamAgentTemplate describes an agent to include in every new team session.
type TeamAgentTemplate struct {
	Name        string
	Description string
	Agent       *openagent.Agent // Model, Tools, Instructions, MaxTurns are captured
}

// NewTeamHandler creates a TeamHandler.
// At least one agent template is required (its Model is used for dynamically added agents).
func NewTeamHandler(mem openagent.Memory, agents ...TeamAgentTemplate) *TeamHandler {
	var model openagent.Model
	if len(agents) > 0 {
		model = agents[0].Agent.Model
	}
	return &TeamHandler{
		agents:   agents,
		memory:   mem,
		model:    model,
		bus:      eventbus.New[SSEEvent](500),
		sessions: make(map[string]*teamSessionState),
	}
}

// Register adds the team handler's routes to mux.
func (h *TeamHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /team/sessions", h.handleCreateSession)
	mux.HandleFunc("GET /team/sessions", h.handleListSessions)
	mux.HandleFunc("GET /team/sessions/{id}", h.handleGetSession)
	mux.HandleFunc("DELETE /team/sessions/{id}", h.handleDeleteSession)
	mux.HandleFunc("POST /team/sessions/{id}/chat", h.handleChat)
	mux.HandleFunc("POST /team/sessions/{id}/approve", h.handleApprove)
	mux.HandleFunc("GET /team/sessions/{id}/agents", h.handleListAgents)
	mux.HandleFunc("POST /team/sessions/{id}/agents", h.handleAddAgent)
	mux.HandleFunc("DELETE /team/sessions/{id}/agents", h.handleRemoveAgent)
}

// ── teamSessionState ──

type teamSessionState struct {
	info      SessionInfo
	team      *openagent.Team
	agentList []agentInfo
	agentMems []*teamAgentMemory // per-agent memory wrappers for cleanup

	mu              sync.Mutex
	pendingApproval *pendingApproval
}

type agentInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"` // "internal"
}

// ── Session CRUD ──

func (h *TeamHandler) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}

	id := generateID()
	now := time.Now()

	info := SessionInfo{
		ID:        id,
		Title:     req.Title,
		AgentName: "team",
		CreatedAt: now,
		UpdatedAt: now,
	}

	h.mu.Lock()
	h.sessions[id] = h.newSession(info)
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(info)
}

func (h *TeamHandler) handleListSessions(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	list := make([]SessionInfo, 0, len(h.sessions))
	for _, s := range h.sessions {
		list = append(list, s.info)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (h *TeamHandler) handleGetSession(w http.ResponseWriter, r *http.Request) {
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

func (h *TeamHandler) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	h.mu.Lock()
	s := h.sessions[id]
	delete(h.sessions, id)
	h.mu.Unlock()

	// Session may not exist (double-delete is harmless).
	if s == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Clean up agent-private memory partitions.
	for _, tam := range s.agentMems {
		_ = tam.DeleteSession(ctx, id)
	}

	// Clean up team-shared memory.
	if h.memory != nil {
		_ = h.memory.DeleteSession(ctx, id)
	}

	w.WriteHeader(http.StatusNoContent)
}

// ── Chat ──

func (h *TeamHandler) handleChat(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Message == "" {
		http.Error(w, `{"error":"message is required"}`, http.StatusBadRequest)
		return
	}

	s := h.getOrCreateSession(id)

	s.mu.Lock()
	s.pendingApproval = nil
	s.mu.Unlock()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
		return
	}
	setSSEHeaders(w)

	sub := h.bus.SubscribeLive(id)
	defer h.bus.Unsubscribe(id, sub)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		oaSession := openagent.Session{
			ID:        id,
			AgentName: "team",
			CreatedAt: s.info.CreatedAt,
		}

		ch := s.team.RunStream(ctx, oaSession, openagent.UserMessage(body.Message))
		for evt := range ch {
			se := teamEventToSSE(evt)
			if se.Type == "" {
				continue
			}
			h.bus.Publish(id, se)
		}
	}()

	for se := range sub.C {
		if err := writeSSE(w, flusher, se); err != nil {
			return
		}
		if se.Type == "done" || se.Type == "error" {
			return
		}
	}
}

// ── Approve ──

func (h *TeamHandler) handleApprove(w http.ResponseWriter, r *http.Request) {
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

// ── Agents ──

func (h *TeamHandler) handleListAgents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	h.mu.RLock()
	s, ok := h.sessions[id]
	h.mu.RUnlock()

	if !ok {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	s.mu.Lock()
	list := make([]agentInfo, len(s.agentList))
	copy(list, s.agentList)
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"agents": list})
}

func (h *TeamHandler) handleAddAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		Instructions string `json:"instructions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}

	h.mu.RLock()
	s, ok := h.sessions[id]
	h.mu.RUnlock()

	if !ok {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	if h.model == nil {
		http.Error(w, `{"error":"no model available for new agent"}`, http.StatusInternalServerError)
		return
	}

	// Wrap memory for agent-private persistence, same as newSession.
	var agentMem openagent.Memory
	var tam *teamAgentMemory
	if h.memory != nil {
		tam = newTeamAgentMemory(body.Name, h.memory)
		agentMem = tam
	}
	agent := openagent.NewAgent(body.Name,
		openagent.WithModel(h.model),
		openagent.WithMemory(agentMem),
		openagent.WithInstructions(body.Instructions),
		openagent.WithMaxTurns(3),
		openagent.WithApprover(&restApprover{
			submit: func(call openagent.ToolCall, resp chan approveResponse) {
				h.submitApproval(s, call, resp)
			},
		}),
	)

	if err := s.team.AddAgent(body.Name, body.Description, agent); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.agentList = append(s.agentList, agentInfo{
		Name: body.Name, Description: body.Description, Type: "internal",
	})
	if tam != nil {
		s.agentMems = append(s.agentMems, tam)
	}
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "added"})
}

func (h *TeamHandler) handleRemoveAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, `{"error":"name query param is required"}`, http.StatusBadRequest)
		return
	}

	h.mu.RLock()
	s, ok := h.sessions[id]
	h.mu.RUnlock()

	if !ok {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	s.team.RemoveAgent(name)

	s.mu.Lock()
	// Clean up the agent's private memory partition.
	filteredMems := s.agentMems[:0]
	for _, tam := range s.agentMems {
		if tam.agentName == name {
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			_ = tam.DeleteSession(ctx, id)
			cancel()
		} else {
			filteredMems = append(filteredMems, tam)
		}
	}
	s.agentMems = filteredMems

	// Remove from the agent list.
	filtered := s.agentList[:0]
	for _, a := range s.agentList {
		if a.Name != name {
			filtered = append(filtered, a)
		}
	}
	s.agentList = filtered
	s.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

// ── Session management ──

func (h *TeamHandler) getOrCreateSession(id string) *teamSessionState {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s, ok := h.sessions[id]; ok {
		return s
	}
	info := SessionInfo{
		ID:        id,
		AgentName: "team",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s := h.newSession(info)
	h.sessions[id] = s
	return s
}

func (h *TeamHandler) newSession(info SessionInfo) *teamSessionState {
	s := &teamSessionState{
		info: info,
	}

	teamOpts := make([]openagent.TeamOption, 0, len(h.agents)+1)
	teamOpts = append(teamOpts, openagent.WithTeamMaxHandoffs(10))

	for _, t := range h.agents {
		// Wrap memory so each agent gets agent-private persistence
		// (tool calls/results) separate from team-shared messages
		// (user input, handoffs, text output).
		var agentMem openagent.Memory
		var tam *teamAgentMemory
		if h.memory != nil {
			tam = newTeamAgentMemory(t.Name, h.memory)
			agentMem = tam
		} else if t.Agent.Memory != nil {
			agentMem = t.Agent.Memory
		}
		agent := cloneAgentForSession(t.Agent, agentMem, s, h.submitApproval)
		teamOpts = append(teamOpts,
			openagent.WithTeamAgent(t.Name, t.Description, agent),
		)
		s.agentList = append(s.agentList, agentInfo{
			Name: t.Name, Description: t.Description, Type: "internal",
		})
		if tam != nil {
			s.agentMems = append(s.agentMems, tam)
		}
	}

	s.team = openagent.NewTeam(teamOpts...)
	return s
}

func cloneAgentForSession(tmpl *openagent.Agent, mem openagent.Memory, s *teamSessionState, submitFn func(*teamSessionState, openagent.ToolCall, chan approveResponse)) *openagent.Agent {
	// mem is already resolved by the caller — either a teamAgentMemory
	// wrapper for agent-private persistence or the template's own memory.
	return openagent.NewAgent(tmpl.Name,
		openagent.WithModel(tmpl.Model),
		openagent.WithMemory(mem),
		openagent.WithTools(tmpl.Tools...),
		openagent.WithInstructions(tmpl.Instructions),
		openagent.WithMaxTurns(tmpl.MaxTurns),
		openagent.WithApprover(&restApprover{
			submit: func(call openagent.ToolCall, resp chan approveResponse) {
				submitFn(s, call, resp)
			},
		}),
	)
}

// ── Approval bridge ──

func (h *TeamHandler) submitApproval(s *teamSessionState, call openagent.ToolCall, resp chan approveResponse) {
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

// ── TeamEvent → SSE ──

func teamEventToSSE(evt openagent.TeamEvent) SSEEvent {
	switch evt.Type {
	case openagent.TeamAgentStart:
		return SSEEvent{Type: "agent_start", Agent: evt.Agent}

	case openagent.TeamAgentEnd:
		se := SSEEvent{Type: "agent_end", Agent: evt.Agent}
		if evt.Error != nil {
			se.Error = evt.Error.Error()
		}
		return se
		case openagent.TeamThought:
			return SSEEvent{Type: "thought", Agent: evt.Agent, Text: evt.Text}

	case openagent.TeamTextDelta:
		return SSEEvent{Type: "text_delta", Agent: evt.Agent, Text: evt.Text}

	case openagent.TeamToolCall:
		var tcj *SSEToolCall
		if evt.ToolCall != nil {
			tcj = &SSEToolCall{
				ID: evt.ToolCall.ID,
				Function: SSEToolCallFunction{
					Name:      evt.ToolCall.Function.Name,
					Arguments: evt.ToolCall.Function.Arguments,
				},
			}
		}
		return SSEEvent{Type: "tool_call", Agent: evt.Agent, ToolCall: tcj}

	case openagent.TeamToolProgress:
		return SSEEvent{Type: "tool_progress", Agent: evt.Agent, ToolCallID: evt.ToolCallID, Text: evt.Text}

	case openagent.TeamToolResult:
		return SSEEvent{Type: "tool_result", Agent: evt.Agent, ToolCallID: evt.ToolCallID, Text: evt.Text}

	case openagent.TeamRetrying:
		msg := "retrying"
		if evt.Error != nil {
			msg = evt.Error.Error()
		}
		return SSEEvent{Type: "retrying", Agent: evt.Agent, Text: msg}

	case openagent.TeamHandoff:
		return SSEEvent{Type: "handoff", Agent: evt.Agent, HandoffTo: evt.Target}

	case openagent.TeamDone:
		se := SSEEvent{Type: "done"}
		if evt.Result != nil {
			se.FinalOutput = evt.Result.FinalOutput
		}
		return se

	case openagent.TeamError:
		se := SSEEvent{Type: "error"}
		if evt.Error != nil {
			se.Text = evt.Error.Error()
		}
		return se

	default:
		return SSEEvent{}
	}
}

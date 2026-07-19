package rest

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/session"
	"github.com/yusheng-g/openagent-go/eventbus"
)

// ── TeamHandler ──

// TeamHandler serves a REST API for an openagent-go Team.
type TeamHandler struct {
	agents []TeamAgentTemplate
	model  openagent.Model // from first template, for dynamically added agents

	sm *sessionManager[*teamSessionState] // session CRUD, store, bus
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
	for _, t := range agents {
		if t.Agent.Model == nil {
			log.Printf("team: agent %q has nil Model — chat will fail until a model is set", t.Name)
		}
	}
	if len(agents) > 0 && model == nil {
		log.Printf("team: primary agent has nil Model — dynamically added agents will have no model")
	}

	h := &TeamHandler{agents: agents, model: model}

	bus := eventbus.New[SSEEvent](500)
	h.sm = newSessionManager[*teamSessionState](nil, mem, bus, sessionHooks[*teamSessionState]{
		kind:     "team",
		newEntry: h.newEntry,
		onDelete: func(s *teamSessionState) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			for _, tam := range s.agentMems {
				_ = tam.DeleteSession(ctx, s.info.ID)
			}
		},
	})

	return h
}

// Register adds the team handler's routes to mux.
func (h *TeamHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /team/sessions", h.handleCreateSession)
	mux.HandleFunc("GET /team/sessions", h.handleListSessions)
	mux.HandleFunc("GET /team/sessions/{id}", h.handleGetSession)
	mux.HandleFunc("GET /team/sessions/{id}/messages", h.handleListMessages)
	mux.HandleFunc("PATCH /team/sessions/{id}", h.handleUpdateSession)
	mux.HandleFunc("DELETE /team/sessions/{id}", h.handleDeleteSession)
	mux.HandleFunc("POST /team/sessions/{id}/chat", h.handleChat)
	mux.HandleFunc("POST /team/sessions/{id}/approve", h.handleApprove)
	mux.HandleFunc("GET /team/sessions/{id}/agents", h.handleListAgents)
	mux.HandleFunc("POST /team/sessions/{id}/agents", h.handleAddAgent)
	mux.HandleFunc("DELETE /team/sessions/{id}/agents", h.handleRemoveAgent)
}

// WithSessionStore attaches a persistent session metadata store.
func (h *TeamHandler) WithSessionStore(s session.Store) *TeamHandler {
	h.sm.SetStore(s)
	return h
}

// StartJanitor starts a background goroutine that evicts idle team session entries.
func (h *TeamHandler) StartJanitor(ctx context.Context, interval, maxIdle time.Duration) {
	h.sm.StartJanitor(ctx, interval, maxIdle)
}

// WithCleanupDir registers a callback invoked when a team session is deleted.
func (h *TeamHandler) WithCleanupDir(fn func(sessionID string)) *TeamHandler {
	h.sm.SetCleanupDir(fn)
	return h
}

// ── teamSessionState ──

type teamSessionState struct {
	info       session.SessionInfo
	team       *openagent.Team
	agentList  []agentInfo
	agentMems  []*teamAgentMemory // per-agent memory wrappers for cleanup

	mu              sync.Mutex
	running         bool             // true while agent goroutine is active
	pendingApproval *pendingApproval
}

func (s *teamSessionState) sessionInfo() *session.SessionInfo { return &s.info }

// isActive reports whether the team session has an ongoing agent run
// or is awaiting tool approval. Eviction skips active sessions.
func (s *teamSessionState) isActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running || s.pendingApproval != nil
}

type agentInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"` // "internal"
}

// ── Session CRUD ──

func (h *TeamHandler) handleCreateSession(w http.ResponseWriter, r *http.Request) { h.sm.create(w, r) }
func (h *TeamHandler) handleListSessions(w http.ResponseWriter, r *http.Request)  { h.sm.list(w, r) }
func (h *TeamHandler) handleGetSession(w http.ResponseWriter, r *http.Request)    { h.sm.get(w, r) }
func (h *TeamHandler) handleUpdateSession(w http.ResponseWriter, r *http.Request) { h.sm.update(w, r) }
func (h *TeamHandler) handleDeleteSession(w http.ResponseWriter, r *http.Request) { h.sm.del(w, r) }

func (h *TeamHandler) handleListMessages(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	mem := h.sm.Memory()
	if mem == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]openagent.Message{})
		return
	}

	limit := 50
	if l, err := parseIntParam(r, "limit", 1, 200); err == nil {
		limit = l
	}
	before := 0
	if b, err := parseIntParam(r, "before", 0, 100000); err == nil {
		before = b
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Shared partition: user messages, handoffs, agent text responses.
	msgs, err := mem.Recent(ctx, id, limit+before, 0)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch messages"}`, http.StatusInternalServerError)
		return
	}

	// Each agent's private partition: tool calls and tool results.
	s := h.sm.getOrCreate(id)
	for _, tam := range s.agentMems {
		priv, _ := tam.PrivateRecent(ctx, id, limit+before, 0)
		msgs = append(msgs, priv...)
	}

	// Sort by global insertion index to restore chronological order
	// across shared + private partitions.
	sort.Slice(msgs, func(i, j int) bool { return msgs[i].Index < msgs[j].Index })
	if before > 0 && len(msgs) > before {
		msgs = msgs[:len(msgs)-before]
	} else if before > 0 {
		msgs = nil
	}
	if len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	if msgs == nil {
		msgs = []openagent.Message{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msgs)
}

// ── Chat ──

func (h *TeamHandler) handleChat(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Message == "" {
		http.Error(w, `{"error":"message is required"}`, http.StatusBadRequest)
		return
	}

	s := h.sm.getOrCreate(id)

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
	flusher.Flush() // flush headers immediately

	sub := h.sm.Bus().SubscribeLive(id)
	defer h.sm.Bus().Unsubscribe(id, sub)

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
			CreatedAt: s.info.CreatedAt,
		}

		ch := s.team.RunStream(ctx, oaSession, openagent.UserMessage(body.Message))
		for evt := range ch {
			se := teamEventToSSE(evt)
			if se.Type == "" {
				continue
			}
			h.sm.Bus().Publish(id, se)
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

// ── Agents ──

func (h *TeamHandler) handleListAgents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	s := h.sm.getOrCreate(id)

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

	s := h.sm.getOrCreate(id)

	if h.model == nil {
		http.Error(w, `{"error":"no model available for new agent"}`, http.StatusInternalServerError)
		return
	}

	// Wrap memory for agent-private persistence, same as newEntry.
	var agentMem openagent.Memory
	var tam *teamAgentMemory
	if mem := h.sm.Memory(); mem != nil {
		tam = newTeamAgentMemory(body.Name, mem)
		agentMem = tam
	}
	agent := openagent.NewAgent(body.Name,
		openagent.WithModel(h.model),
		openagent.WithMemory(agentMem),
		openagent.WithSystemPrompts(body.Instructions),
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

	s := h.sm.getOrCreate(id)

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

// ── Factory ──

func (h *TeamHandler) newEntry(info session.SessionInfo) *teamSessionState {
	s := &teamSessionState{
		info: info,
	}

	teamOpts := make([]openagent.TeamOption, 0, len(h.agents)+1)
	teamOpts = append(teamOpts, openagent.WithTeamMaxHandoffs(10))

	mem := h.sm.Memory()
	for _, t := range h.agents {
		// Wrap memory so each agent gets agent-private persistence
		// (tool calls/results) separate from team-shared messages
		// (user input, handoffs, text output).
		var agentMem openagent.Memory
		var tam *teamAgentMemory
		if mem != nil {
			tam = newTeamAgentMemory(t.Name, mem)
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
		openagent.WithSystemPrompts(tmpl.SystemPrompts...),
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

	h.sm.Bus().Publish(s.info.ID, evt)
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
			se.PromptTokens = evt.Result.Usage.PromptTokens
		}
		return se

	case openagent.TeamError:
		se := SSEEvent{Type: "error"}
		if evt.Error != nil {
			se.Text = evt.Error.Error()
		}
		return se

	default:
		log.Printf("rest: unknown team event type %q", evt.Type)
		return SSEEvent{Type: "unknown"}
	}
}

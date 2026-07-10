package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/eventbus"
	"github.com/yusheng-g/openagent-go/plan"
)

// PlanAgentTemplate describes an agent available for plan steps.
type PlanAgentTemplate struct {
	Name        string
	Description string
	Runner      openagent.AgentRunner // *Agent, Team, or external ACP runner
}

// ── PlanHandler ──

// PlanHandler serves a REST API for goal → DAG → execution workflows.
//
// Create with [NewPlanHandler], then register on an [http.ServeMux]:
//
//	templates := []rest.PlanAgentTemplate{
//	    {Name: "coder", Description: "Writes code", Runner: coderAgent},
//	    {Name: "reviewer", Description: "Reviews code", Runner: reviewerAgent},
//	}
//	ph := rest.NewPlanHandler(mem, model, templates...)
//	ph.Register(mux)
type PlanHandler struct {
	agents []PlanAgentTemplate
	model  openagent.Model
	memory openagent.Memory

	bus *eventbus.Bus[SSEEvent]

	mu       sync.RWMutex
	sessions map[string]*planSessionState
}

// planSessionState holds per-session data for a plan workflow.
type planSessionState struct {
	info       SessionInfo
	plan       *plan.Plan
	currentDef *plan.PlanDef // nil until generated

	mu              sync.Mutex
	pendingApproval *pendingApproval
	execCancel      context.CancelFunc // set during execution, nil otherwise
	running         bool               // true while plan is executing

	// Pause/resume support (AutoReplan=false).
	execState *plan.PlanState  // current execution state (set when paused)
	retryCh   chan plan.RetryAction // closed/signaled to resume from pause
}

// NewPlanHandler creates a PlanHandler.
// model is used for both the Planner and step output summarisation.
// At least one agent template is required.
func NewPlanHandler(mem openagent.Memory, model openagent.Model, agents ...PlanAgentTemplate) *PlanHandler {
	return &PlanHandler{
		agents:   agents,
		model:    model,
		memory:   mem,
		bus:      eventbus.New[SSEEvent](1000),
		sessions: make(map[string]*planSessionState),
	}
}

// Register adds the plan handler's routes to mux using Go 1.22+ patterns.
func (h *PlanHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /plan/sessions", h.handleCreateSession)
	mux.HandleFunc("GET /plan/sessions", h.handleListSessions)
	mux.HandleFunc("GET /plan/sessions/{id}", h.handleGetSession)
	mux.HandleFunc("PATCH /plan/sessions/{id}", h.handleUpdateSession)
	mux.HandleFunc("DELETE /plan/sessions/{id}", h.handleDeleteSession)
	mux.HandleFunc("POST /plan/sessions/{id}/generate", h.handleGenerate)
	mux.HandleFunc("GET /plan/sessions/{id}/plan", h.handleGetPlan)
	mux.HandleFunc("PUT /plan/sessions/{id}/plan", h.handleUpdatePlan)
	mux.HandleFunc("POST /plan/sessions/{id}/execute", h.handleExecute)
	mux.HandleFunc("GET /plan/sessions/{id}/events", h.handleEvents)
	mux.HandleFunc("POST /plan/sessions/{id}/cancel", h.handleCancel)
	mux.HandleFunc("POST /plan/sessions/{id}/steps/{stepID}/retry", h.handleStepRetry)
	mux.HandleFunc("POST /plan/sessions/{id}/replan", h.handleReplan)
	mux.HandleFunc("POST /plan/sessions/{id}/approve", h.handleApprove)
}

// ── Session CRUD ──

func (h *PlanHandler) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}

	id := generateID()
	now := time.Now()

	info := SessionInfo{
		ID:        id,
		Title:     req.Title,
		AgentName: "plan",
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

func (h *PlanHandler) handleListSessions(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	list := make([]SessionInfo, 0, len(h.sessions))
	for _, s := range h.sessions {
		list = append(list, s.info)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (h *PlanHandler) handleGetSession(w http.ResponseWriter, r *http.Request) {
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

func (h *PlanHandler) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
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

func (h *PlanHandler) handleUpdateSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
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
	s.info.Title = body.Title
	s.info.UpdatedAt = time.Now()
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// ── Plan generation ──

func (h *PlanHandler) handleGenerate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body struct {
		Goal string `json:"goal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Goal == "" {
		http.Error(w, `{"error":"goal is required"}`, http.StatusBadRequest)
		return
	}

	s := h.getOrCreateSession(id)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
		return
	}
	setSSEHeaders(w)

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	// Emit plan_thinking events as the LLM generates the plan,
	// then plan_generated with the final PlanDef.
	def, err := s.plan.PlanStream(ctx, body.Goal, nil, func(chunk string) {
		_ = writeSSE(w, flusher, SSEEvent{Type: "plan_thinking", Text: chunk})
	})
	if err != nil {
		_ = writeSSE(w, flusher, SSEEvent{Type: "plan_error", Error: err.Error()})
		return
	}

	s.mu.Lock()
	s.currentDef = def
	if def.Goal != "" {
		s.info.Title = def.Goal
		s.info.UpdatedAt = time.Now()
	}
	s.mu.Unlock()

	b, _ := json.Marshal(def)
	_ = writeSSE(w, flusher, SSEEvent{Type: "plan_generated", Text: string(b)})
}

// ── Get current plan ──

func (h *PlanHandler) handleGetPlan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	h.mu.RLock()
	s, ok := h.sessions[id]
	h.mu.RUnlock()

	if !ok {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	s.mu.Lock()
	def := s.currentDef
	s.mu.Unlock()

	if def == nil {
		http.Error(w, `{"error":"no plan generated yet"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(def)
}

// ── Update plan (user edits) ──

func (h *PlanHandler) handleUpdatePlan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	h.mu.RLock()
	s, ok := h.sessions[id]
	h.mu.RUnlock()

	if !ok {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	var def plan.PlanDef
	if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
		http.Error(w, `{"error":"invalid plan JSON"}`, http.StatusBadRequest)
		return
	}

	// Validate the user-edited plan.
	agentNames := make(map[string]bool)
	for _, a := range h.agents {
		agentNames[a.Name] = true
	}
	if err := plan.Validate(&def, agentNames); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	// Enforce max steps (same as generate-time check).
	maxSteps := 20
	if len(def.Steps) > maxSteps {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("plan has %d steps, max is %d", len(def.Steps), maxSteps)})
		return
	}

	s.mu.Lock()
	s.currentDef = &def
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(def)
}

// ── Execute plan (trigger only, returns 202) ──

func (h *PlanHandler) handleExecute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	h.mu.RLock()
	s, ok := h.sessions[id]
	h.mu.RUnlock()

	if !ok {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		http.Error(w, `{"error":"plan is already executing"}`, http.StatusConflict)
		return
	}
	def := s.currentDef
	s.mu.Unlock()

	if def == nil {
		http.Error(w, `{"error":"no plan to execute — call /generate first"}`, http.StatusBadRequest)
		return
	}

	// Create a cancellable context for the execution.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)

	s.mu.Lock()
	s.execCancel = cancel
	s.running = true
	s.mu.Unlock()

	go func() {
		defer cancel()
		defer func() {
			s.mu.Lock()
			s.execCancel = nil
			s.running = false
			s.execState = nil
			s.retryCh = nil
			s.mu.Unlock()
		}()

		oaSession := openagent.Session{
			ID:        id,
			AgentName: "plan",
			CreatedAt: s.info.CreatedAt,
		}

		// Create the plan state once and reuse across pause/resume cycles.
		state := &plan.PlanState{
			ID:        id + "/plan",
			Goal:      def.Goal,
			Status:    plan.PlanStatusApproved,
			Steps:     def.Steps,
			Results:   make(map[string]*plan.StepResult),
			CreatedAt: s.info.CreatedAt,
			UpdatedAt: s.info.CreatedAt,
		}

		for {
			// Execute (or resume after pause).
			ch := s.plan.ExecuteWithState(ctx, oaSession, def, state)

			paused := false
			for evt := range ch {
				se := planEventToSSE(evt)
				if se.Type == "" {
					continue
				}
				h.bus.Publish(id, se)

				if se.Type == "plan_waiting_retry" {
					// Store state and create resume channel.
					s.mu.Lock()
					s.execState = state
					s.retryCh = make(chan plan.RetryAction, 1)
					s.mu.Unlock()
					paused = true
					break // exit inner loop, wait for resume
				}
			}

			if !paused {
				return // plan_done, plan_error, or plan_cancelled — done
			}

			// Wait for retry or replan command.
			select {
			case action := <-s.retryCh:
				switch action.Action {
				case "retry":
					// Reset the failed step so executeBatches picks it up.
					if sr := state.Results[action.StepID]; sr != nil {
						sr.Status = plan.StepStatusPending
						sr.Error = ""
						sr.Retries = 0
					}
				case "replan":
					// Use the replanned definition (already merged by the replan endpoint).
					def = action.NewDef
					state.Steps = def.Steps
					state.ReplanCount++
					state.UpdatedAt = time.Now()
				}
			case <-ctx.Done():
				h.bus.Publish(id, SSEEvent{Type: "plan_cancelled"})
				return
			}
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

// ── Events (SSE stream, EventSource-compatible GET) ──

func (h *PlanHandler) handleEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	h.mu.RLock()
	_, ok := h.sessions[id]
	h.mu.RUnlock()
	if !ok {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
		return
	}
	setSSEHeaders(w)

	sub := h.bus.SubscribeLive(id)
	defer h.bus.Unsubscribe(id, sub)

	for {
		select {
		case <-r.Context().Done():
			return
		case se, ok := <-sub.C:
			if !ok {
				return
			}
			if err := writeSSE(w, flusher, se); err != nil {
				return
			}
			if se.Type == "plan_done" || se.Type == "plan_error" || se.Type == "plan_cancelled" {
				return
			}
		}
	}
}

// ── Cancel execution ──

func (h *PlanHandler) handleCancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	h.mu.RLock()
	s, ok := h.sessions[id]
	h.mu.RUnlock()

	if !ok {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	s.mu.Lock()
	cancel := s.execCancel
	s.mu.Unlock()

	if cancel == nil {
		http.Error(w, `{"error":"no execution in progress"}`, http.StatusBadRequest)
		return
	}

	cancel()
	h.bus.Publish(id, SSEEvent{Type: "plan_cancelled"})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

// ── Manual retry / replan (when AutoReplan is false) ──

// handleStepRetry resumes a paused plan by retrying a failed step.
// The step is reset to pending and execution continues from that point.
func (h *PlanHandler) handleStepRetry(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	stepID := r.PathValue("stepID")

	h.mu.RLock()
	s, ok := h.sessions[id]
	h.mu.RUnlock()

	if !ok {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	s.mu.Lock()
	state := s.execState
	retryCh := s.retryCh
	s.mu.Unlock()

	if state == nil || retryCh == nil {
		http.Error(w, `{"error":"plan is not waiting for retry"}`, http.StatusBadRequest)
		return
	}

	// Verify the step exists and is failed.
	sr := state.Results[stepID]
	if sr == nil || sr.Status != plan.StepStatusFailed {
		http.Error(w, `{"error":"step not found or not in failed state"}`, http.StatusBadRequest)
		return
	}

	// Signal the execution goroutine to retry.
	retryCh <- plan.RetryAction{Action: "retry", StepID: stepID}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "retrying"})
}

// handleReplan regenerates the affected subtree of a failed plan incorporating
// user feedback (natural language suggestions). The new plan is sent to the
// execution goroutine which resumes from where it left off.
func (h *PlanHandler) handleReplan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body struct {
		Feedback string `json:"feedback"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Feedback == "" {
		http.Error(w, `{"error":"feedback is required"}`, http.StatusBadRequest)
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
	state := s.execState
	def := s.currentDef
	retryCh := s.retryCh
	s.mu.Unlock()

	if state == nil || retryCh == nil {
		http.Error(w, `{"error":"plan is not waiting for retry"}`, http.StatusBadRequest)
		return
	}

	// Find the failed step.
	failedID := ""
	for _, step := range def.Steps {
		if sr := state.Results[step.ID]; sr != nil && sr.Status == plan.StepStatusFailed {
			failedID = step.ID
			break
		}
	}
	if failedID == "" {
		http.Error(w, `{"error":"no failed step found"}`, http.StatusBadRequest)
		return
	}

	// Emit replanning event so the UI knows.
	h.bus.Publish(id, SSEEvent{Type: "replanning", StepID: failedID})

	// Call planner with user feedback.
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	newDef, err := s.plan.ReplanWithFeedback(ctx, def, state, failedID, body.Feedback)
	if err != nil {
		h.bus.Publish(id, SSEEvent{Type: "plan_error", Error: err.Error()})
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Update the session's current definition.
	s.mu.Lock()
	s.currentDef = newDef
	s.mu.Unlock()

	// Emit the new plan so the UI can re-render the DAG.
	b, _ := json.Marshal(newDef)
	h.bus.Publish(id, SSEEvent{Type: "plan_generated", Text: string(b)})

	// Signal the execution goroutine to resume with the new plan.
	retryCh <- plan.RetryAction{Action: "replan", NewDef: newDef}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "replanned"})
}

// ── Tool approval during plan execution ──

func (h *PlanHandler) handleApprove(w http.ResponseWriter, r *http.Request) {
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

func (h *PlanHandler) getOrCreateSession(id string) *planSessionState {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s, ok := h.sessions[id]; ok {
		return s
	}
	info := SessionInfo{
		ID:        id,
		AgentName: "plan",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s := h.newSession(info)
	h.sessions[id] = s
	return s
}

func (h *PlanHandler) newSession(info SessionInfo) *planSessionState {
	s := &planSessionState{
		info: info,
	}

	// Build plan with agents.
	opts := make([]plan.PlanOption, 0, len(h.agents)+3)
	opts = append(opts, plan.WithPlanner(plan.NewLLMPlanner(h.model)))
	if h.model != nil {
		opts = append(opts, plan.WithModel(h.model))
	}
	opts = append(opts, plan.WithMaxConcurrency(8))
	opts = append(opts, plan.WithAutoReplan(false)) // pause on failure, let user retry/replan

	for _, t := range h.agents {
		// Clone in-process agents so each session has isolated state.
		runner := t.Runner
		if ag, ok := t.Runner.(*openagent.Agent); ok {
			runner = cloneAgentForPlan(ag, s, h.submitApproval)
		}
		opts = append(opts, plan.WithAgent(t.Name, t.Description, runner))
	}

	s.plan = plan.NewPlan(opts...)
	return s
}

func (h *PlanHandler) submitApproval(s *planSessionState, call openagent.ToolCall, resp chan approveResponse) {
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

// cloneAgentForPlan clones an Agent for use in a plan session, injecting the
// REST approver bridge so tool approvals during plan steps are surfaced to the UI.
func cloneAgentForPlan(tmpl *openagent.Agent, s *planSessionState, submitFn func(*planSessionState, openagent.ToolCall, chan approveResponse)) *openagent.Agent {
	return openagent.NewAgent(tmpl.Name,
		openagent.WithModel(tmpl.Model),
		openagent.WithMemory(tmpl.Memory),
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

// ── PlanEvent → SSE ──

func planEventToSSE(evt plan.PlanEvent) SSEEvent {
	switch evt.Type {
	case plan.PlanEventGenerated:
		se := SSEEvent{Type: "plan_generated"}
		if evt.Def != nil {
			b, _ := json.Marshal(evt.Def)
			se.Text = string(b)
		}
		return se

	case plan.PlanEventApproved:
		return SSEEvent{Type: "plan_approved"}

	case plan.PlanEventStepStart:
		return SSEEvent{Type: "step_start", StepID: evt.StepID, Agent: evt.Agent}

	case plan.PlanEventTextDelta:
		return SSEEvent{Type: "step_text_delta", StepID: evt.StepID, Text: evt.Text}

	case plan.PlanEventToolCall:
		return SSEEvent{
			Type: "step_tool_call", StepID: evt.StepID,
			ToolCall: &SSEToolCall{
				ID: evt.ToolID,
				Function: SSEToolCallFunction{
					Name:      evt.ToolName,
					Arguments: evt.ToolArgs,
				},
			},
		}

	case plan.PlanEventToolProgress:
		return SSEEvent{Type: "step_tool_progress", StepID: evt.StepID, Text: evt.Text}

	case plan.PlanEventToolResult:
		return SSEEvent{Type: "step_tool_result", StepID: evt.StepID, Text: evt.Text}

	case plan.PlanEventStepDone:
		se := SSEEvent{Type: "step_done", StepID: evt.StepID, Agent: evt.Agent}
		if evt.Result != nil {
			se.Text = evt.Result.Summary
		}
		return se

	case plan.PlanEventStepFailed:
		return SSEEvent{Type: "step_failed", StepID: evt.StepID, Agent: evt.Agent, Error: evt.ErrText}

	case plan.PlanEventReplanning:
		return SSEEvent{Type: "replanning", StepID: evt.StepID}

	case plan.PlanEventWaitingRetry:
		return SSEEvent{Type: "plan_waiting_retry", StepID: evt.StepID, Error: evt.ErrText}

	case plan.PlanEventDone:
		return SSEEvent{Type: "plan_done"}

	case plan.PlanEventError:
		return SSEEvent{Type: "plan_error", Error: evt.ErrText}

	default:
		return SSEEvent{}
	}
}

package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/session"
	"github.com/yusheng-g/openagent-go/eventbus"
	"github.com/yusheng-g/openagent-go/orchestrate"
)

// OrchestrateAgentTemplate describes an agent available for plan steps.
type OrchestrateAgentTemplate struct {
	Name        string
	Description string
	Runner      openagent.AgentRunner // *Agent, Team, or external ACP runner
}

// ── OrchestrateHandler ──

// OrchestrateHandler serves a REST API for goal → DAG → execution workflows.
type OrchestrateHandler struct {
	agents []OrchestrateAgentTemplate
	model  openagent.Model

	sm *sessionManager[*planSessionState] // session CRUD, store, bus
}

// planSessionState holds per-session data for a plan workflow.
type planSessionState struct {
	info       session.SessionInfo
	plan       *orchestrate.Plan
	currentDef *orchestrate.PlanDef // nil until generated

	mu              sync.Mutex
	pendingApproval *pendingApproval
	execCancel      context.CancelFunc // set during execution, nil otherwise
	running         bool               // true while plan is executing

	// Pause/resume support (AutoReplan=false).
	execState *orchestrate.PlanState       // current execution state (set when paused)
	retryCh   chan orchestrate.RetryAction // closed/signaled to resume from pause
}

func (s *planSessionState) sessionInfo() *session.SessionInfo { return &s.info }

// isActive reports whether the plan session has a running execution
// or is awaiting tool approval.
func (s *planSessionState) isActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running || s.pendingApproval != nil
}

// NewPlanHandler creates a OrchestrateHandler.
// model is used for both the Planner and step output summarisation.
// At least one agent template is required.
func NewOrchestrateHandler(mem openagent.Memory, model openagent.Model, agents ...OrchestrateAgentTemplate) *OrchestrateHandler {
	h := &OrchestrateHandler{agents: agents, model: model}

	bus := eventbus.New[SSEEvent](1000)
	h.sm = newSessionManager[*planSessionState](nil, mem, bus, sessionHooks[*planSessionState]{
		kind:     "plan",
		newEntry: h.newEntry,
	})

	return h
}

// Register adds the plan handler's routes to mux using Go 1.22+ patterns.
func (h *OrchestrateHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /plan/sessions", h.handleCreateSession)
	mux.HandleFunc("GET /plan/sessions", h.handleListSessions)
	mux.HandleFunc("GET /plan/sessions/{id}", h.handleGetSession)
	mux.HandleFunc("GET /plan/sessions/{id}/messages", h.handlePlanMessages)
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

// WithSessionStore attaches a persistent session metadata store.
func (h *OrchestrateHandler) WithSessionStore(s session.Store) *OrchestrateHandler {
	h.sm.SetStore(s)
	return h
}

// StartJanitor starts a background goroutine that evicts idle plan session entries.
func (h *OrchestrateHandler) StartJanitor(ctx context.Context, interval, maxIdle time.Duration) {
	h.sm.StartJanitor(ctx, interval, maxIdle)
}

// WithCleanupDir registers a callback invoked when a plan session is deleted.
func (h *OrchestrateHandler) WithCleanupDir(fn func(sessionID string)) *OrchestrateHandler {
	h.sm.SetCleanupDir(fn)
	return h
}

// ── Session CRUD ──

func (h *OrchestrateHandler) handleCreateSession(w http.ResponseWriter, r *http.Request) { h.sm.create(w, r) }
func (h *OrchestrateHandler) handleListSessions(w http.ResponseWriter, r *http.Request)  { h.sm.list(w, r) }
func (h *OrchestrateHandler) handleGetSession(w http.ResponseWriter, r *http.Request)    { h.sm.get(w, r) }
func (h *OrchestrateHandler) handleUpdateSession(w http.ResponseWriter, r *http.Request) { h.sm.update(w, r) }
func (h *OrchestrateHandler) handleDeleteSession(w http.ResponseWriter, r *http.Request) { h.sm.del(w, r) }

func (h *OrchestrateHandler) handlePlanMessages(w http.ResponseWriter, r *http.Request) { h.sm.messages(w, r) }

// ── Plan generation ──

func (h *OrchestrateHandler) handleGenerate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body struct {
		Goal string `json:"goal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Goal == "" {
		http.Error(w, `{"error":"goal is required"}`, http.StatusBadRequest)
		return
	}

	s := h.sm.getOrCreate(id)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
		return
	}
	setSSEHeaders(w)
	flusher.Flush() // flush headers immediately

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
	s.mu.Unlock()

	// Set the session title from the goal and persist.
	if def.Goal != "" {
		if inf, ok := h.sm.withMeta(id, func(inf *session.SessionInfo) {
			inf.Title = def.Goal
			inf.UpdatedAt = time.Now()
		}); ok {
			h.sm.syncMeta(inf)
		}
	}

	b, _ := json.Marshal(def)
	_ = writeSSE(w, flusher, SSEEvent{Type: "plan_generated", Text: string(b)})
}

// ── Get current plan ──

func (h *OrchestrateHandler) handleGetPlan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	s := h.sm.getOrCreate(id)

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

func (h *OrchestrateHandler) handleUpdatePlan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	s := h.sm.getOrCreate(id)

	var def orchestrate.PlanDef
	if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
		http.Error(w, `{"error":"invalid plan JSON"}`, http.StatusBadRequest)
		return
	}

	// Validate the user-edited orchestrate.
	agentNames := make(map[string]bool)
	for _, a := range h.agents {
		agentNames[a.Name] = true
	}
	if err := orchestrate.Validate(&def, agentNames); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	// Enforce max steps (same as orchestrate.DefaultPlanConfig().MaxSteps).
	maxSteps := orchestrate.DefaultPlanConfig().MaxSteps
	if len(def.Steps) > maxSteps {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("plan has %d steps, max is %d", len(def.Steps), maxSteps)})
		return
	}

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		http.Error(w, `{"error":"cannot modify plan while it is executing"}`, http.StatusConflict)
		return
	}
	s.currentDef = &def
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(def)
}

// ── Execute plan (trigger only, returns 202) ──

func (h *OrchestrateHandler) handleExecute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	s := h.sm.getOrCreate(id)

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
			CreatedAt: s.info.CreatedAt,
		}

		// Create the plan state once and reuse across pause/resume cycles.
		state := &orchestrate.PlanState{
			ID:        id + "/plan",
			Goal:      def.Goal,
			Status:    orchestrate.PlanStatusApproved,
			Steps:     def.Steps,
			Results:   make(map[string]*orchestrate.StepResult),
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
				h.sm.Bus().Publish(id, se)

				if se.Type == "plan_waiting_retry" {
					// Store state and create resume channel.
					s.mu.Lock()
					s.execState = state
					s.retryCh = make(chan orchestrate.RetryAction, 1)
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
					// Gate step: already done, just continue to next batch.
					// Failed step: reset to pending so executeBatches re-runs it.
					if sr := state.Results[action.StepID]; sr != nil {
						if sr.Status != orchestrate.StepStatusDone {
							sr.Status = orchestrate.StepStatusPending
							sr.Error = ""
							sr.Retries = 0
						}
					}
				case "replan":
					// Use the replanned definition (already merged by the replan endpoint).
					def = action.NewDef
					state.Steps = def.Steps
					state.ReplanCount++
					state.UpdatedAt = time.Now()
				}
			case <-ctx.Done():
				h.sm.Bus().Publish(id, SSEEvent{Type: "plan_cancelled"})
				return
			}
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

// ── Events (SSE stream, EventSource-compatible GET) ──

func (h *OrchestrateHandler) handleEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if !h.sm.Exists(id) {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
		return
	}
	setSSEHeaders(w)
	flusher.Flush() // flush headers immediately

	sub := h.sm.Bus().SubscribeLive(id)
	defer h.sm.Bus().Unsubscribe(id, sub)

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

func (h *OrchestrateHandler) handleCancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	s := h.sm.getOrCreate(id)

	s.mu.Lock()
	cancel := s.execCancel
	s.mu.Unlock()

	if cancel == nil {
		http.Error(w, `{"error":"no execution in progress"}`, http.StatusBadRequest)
		return
	}

	cancel()
	h.sm.Bus().Publish(id, SSEEvent{Type: "plan_cancelled"})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

// ── Manual retry / replan (when AutoReplan is false) ──

// handleStepRetry resumes a paused plan by retrying a failed step.
func (h *OrchestrateHandler) handleStepRetry(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	stepID := r.PathValue("stepID")

	s := h.sm.getOrCreate(id)

	s.mu.Lock()
	state := s.execState
	retryCh := s.retryCh
	curDef := s.currentDef
	s.mu.Unlock()

	if state == nil || retryCh == nil {
		http.Error(w, `{"error":"plan is not waiting for retry"}`, http.StatusBadRequest)
		return
	}

	// Verify the step exists and is in a valid paused state.
	sr := state.Results[stepID]
	if sr == nil {
		http.Error(w, `{"error":"step not found"}`, http.StatusBadRequest)
		return
	}
	isGate := false
	if curDef != nil {
		for _, st := range curDef.Steps {
			if st.ID == stepID && st.Gate {
				isGate = true
				break
			}
		}
	}
	if sr.Status != orchestrate.StepStatusFailed && !(isGate && sr.Status == orchestrate.StepStatusDone) {
		http.Error(w, `{"error":"step is not in a paused state"}`, http.StatusBadRequest)
		return
	}

	// Signal the execution goroutine to continue.
	retryCh <- orchestrate.RetryAction{Action: "retry", StepID: stepID}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "retrying"})
}

// handleReplan regenerates the affected subtree of a failed plan incorporating user feedback.
func (h *OrchestrateHandler) handleReplan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body struct {
		Feedback string `json:"feedback"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Feedback == "" {
		http.Error(w, `{"error":"feedback is required"}`, http.StatusBadRequest)
		return
	}

	s := h.sm.getOrCreate(id)

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
		if sr := state.Results[step.ID]; sr != nil && sr.Status == orchestrate.StepStatusFailed {
			failedID = step.ID
			break
		}
	}
	if failedID == "" {
		http.Error(w, `{"error":"no failed step found"}`, http.StatusBadRequest)
		return
	}

	// Emit replanning event so the UI knows.
	h.sm.Bus().Publish(id, SSEEvent{Type: "replanning", StepID: failedID})

	// Call planner with user feedback, streaming the LLM's thinking.
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	newDef, err := s.plan.ReplanWithFeedbackStream(ctx, def, state, failedID, body.Feedback, func(chunk string) {
		h.sm.Bus().Publish(id, SSEEvent{Type: "plan_thinking", Text: chunk})
	})
	if err != nil {
		h.sm.Bus().Publish(id, SSEEvent{Type: "plan_error", Error: err.Error()})
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Update the session's current definition.
	s.mu.Lock()
	s.currentDef = newDef
	s.mu.Unlock()

	// Emit the new plan so the UI can re-render the DAG.
	b, _ := json.Marshal(newDef)
	h.sm.Bus().Publish(id, SSEEvent{Type: "plan_generated", Text: string(b)})

	// Signal the execution goroutine to resume with the new orchestrate.
	retryCh <- orchestrate.RetryAction{Action: "replan", NewDef: newDef}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "replanned"})
}

// ── Tool approval during plan execution ──

func (h *OrchestrateHandler) handleApprove(w http.ResponseWriter, r *http.Request) {
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

// ── Factory ──

func (h *OrchestrateHandler) newEntry(info session.SessionInfo) *planSessionState {
	s := &planSessionState{
		info: info,
	}

	// Build plan with agents.
	opts := make([]orchestrate.PlanOption, 0, len(h.agents)+3)
	opts = append(opts, orchestrate.WithPlanner(orchestrate.NewLLMPlanner(h.model)))
	if h.model != nil {
		opts = append(opts, orchestrate.WithModel(h.model))
	}
	opts = append(opts, orchestrate.WithMaxConcurrency(8))
	opts = append(opts, orchestrate.WithAutoReplan(false)) // pause on failure, let user retry/replan

	mem := h.sm.Memory()
	for _, t := range h.agents {
		// Clone in-process agents so each session has isolated state.
		runner := t.Runner
		if ag, ok := t.Runner.(*openagent.Agent); ok {
			runner = cloneAgentForPlan(ag, mem, s, h.submitApproval)
		}
		opts = append(opts, orchestrate.WithAgent(t.Name, t.Description, runner))
	}

	s.plan = orchestrate.NewPlan(opts...)
	return s
}

// ── Approval bridge ──

func (h *OrchestrateHandler) submitApproval(s *planSessionState, call openagent.ToolCall, resp chan approveResponse) {
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

// cloneAgentForPlan clones an Agent for use in a plan session, injecting the REST approver bridge.
// If mem is non-nil, it overrides the template's memory so plan steps persist their messages.
func cloneAgentForPlan(tmpl *openagent.Agent, mem openagent.Memory, s *planSessionState, submitFn func(*planSessionState, openagent.ToolCall, chan approveResponse)) *openagent.Agent {
	if mem == nil {
		mem = tmpl.Memory
	}
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

// ── PlanEvent → SSE ──

func planEventToSSE(evt orchestrate.PlanEvent) SSEEvent {
	switch evt.Type {
	case orchestrate.PlanEventGenerated:
		se := SSEEvent{Type: "plan_generated"}
		if evt.Def != nil {
			b, _ := json.Marshal(evt.Def)
			se.Text = string(b)
		}
		return se

	case orchestrate.PlanEventApproved:
		return SSEEvent{Type: "plan_approved"}

	case orchestrate.PlanEventStepStart:
		return SSEEvent{Type: "step_start", StepID: evt.StepID, Agent: evt.Agent}

	case orchestrate.PlanEventTextDelta:
		return SSEEvent{Type: "step_text_delta", StepID: evt.StepID, Text: evt.Text}

	case orchestrate.PlanEventToolCall:
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

	case orchestrate.PlanEventToolProgress:
		return SSEEvent{Type: "step_tool_progress", StepID: evt.StepID, Text: evt.Text}

	case orchestrate.PlanEventToolResult:
		return SSEEvent{Type: "step_tool_result", StepID: evt.StepID, Text: evt.Text}

	case orchestrate.PlanEventStepDone:
		se := SSEEvent{Type: "step_done", StepID: evt.StepID, Agent: evt.Agent}
		if evt.Result != nil {
			se.Text = evt.Result.Summary
		}
		return se

	case orchestrate.PlanEventStepFailed:
		return SSEEvent{Type: "step_failed", StepID: evt.StepID, Agent: evt.Agent, Error: evt.ErrText}

	case orchestrate.PlanEventReplanning:
		return SSEEvent{Type: "replanning", StepID: evt.StepID}

	case orchestrate.PlanEventWaitingRetry:
		return SSEEvent{Type: "plan_waiting_retry", StepID: evt.StepID, Error: evt.ErrText}

	case orchestrate.PlanEventDone:
		return SSEEvent{Type: "plan_done", FinalOutput: evt.Text}

	case orchestrate.PlanEventError:
		return SSEEvent{Type: "plan_error", Error: evt.ErrText}

	default:
		log.Printf("rest: unknown plan event type %q", evt.Type)
		return SSEEvent{Type: "unknown"}
	}
}

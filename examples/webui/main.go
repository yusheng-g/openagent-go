// WebUI chat with SSE streaming + tool approval.
//
//	go run ./examples/webui/
//	open http://localhost:8080        — single agent chat
//	open http://localhost:8080/team   — agent team chat
package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/eventbus"
	"github.com/yusheng-g/openagent-go/memory/sqlite"
	"github.com/yusheng-g/openagent-go/model/openai"
	"github.com/yusheng-g/openagent-go/plan"
	"github.com/yusheng-g/openagent-go/plugin/wasm"
	openacp "github.com/yusheng-g/openagent-go/runner/acp"
	"github.com/yusheng-g/openagent-go/sandbox/native"
	opentool "github.com/yusheng-g/openagent-go/tool"
)

// Frontend display history limit — not model context (that's Memory+Summarizer).
// Large enough to cover typical conversations; capped to avoid bloating SSE connect.
const historyReplayLimit = 500

//go:embed index.html
var indexHTML embed.FS

//go:embed team.html
var teamHTML embed.FS


func main() {
	apiKey := os.Getenv("OPENAGENT_API_KEY")
	modelID := os.Getenv("OPENAGENT_MODEL")
	baseURL := os.Getenv("OPENAGENT_BASE_URL")
	if apiKey == "" || modelID == "" {
		log.Fatal("set OPENAGENT_API_KEY and OPENAGENT_MODEL")
	}

	llm := openai.New(apiKey, modelID, baseURL).WithContextWindow(128_000)

	// ── Native sandbox (OS-level confinement for shell commands) ──
	workDir, _ := filepath.Abs(".")
	log.Printf("Workspace: %s", workDir)
	sandbox, err := native.New(workDir)
	if err != nil {
		log.Printf("WARNING: native sandbox unavailable: %v (shell commands will be rejected)", err)
	}

	var sandboxTools []openagent.Tool
	if sandbox != nil {
		sandboxTools = []openagent.Tool{
			opentool.NewShell(sandbox, workDir),
			opentool.NewReadFile(workDir),
			opentool.NewWriteFile(workDir),
			opentool.NewListDir(workDir),
			opentool.NewGrep(workDir),
		}
	}

	// Built-in tools.
	builtin := []openagent.Tool{
		&calculatorTool{},
		&echoTool{},
	}
	builtin = append(builtin, sandboxTools...)

	// Load WASM plugins if available (non-fatal).
	mgr := wasm.NewManager("./plugins")
	if err := mgr.Discover(context.Background()); err == nil {
		builtin = mergeTools(builtin, mgr.Tools())
	}
	defer mgr.Close()

	// ── Shared SQLite memory ──
	mem, err := sqlite.New("./webui-memory.db")
	if err != nil {
		log.Fatal("open memory: ", err)
	}
	defer mem.Close()

	// ── Session stores (separate for single/team) ──
	singleStore, err := newSessionStore("./webui-sessions.json")
	if err != nil {
		log.Fatal("open session store: ", err)
	}
	teamStore, err := newSessionStore("./webui-team-sessions.json")
	if err != nil {
		log.Fatal("open team session store: ", err)
	}
	planStore, err := newPlanStore("./webui-plans.json")
	if err != nil {
		log.Fatal("open plan store: ", err)
	}

	// ── Single-agent endpoints ──
	hub := newHub(llm, modelID, builtin, mem, singleStore, planStore, workDir)

	idx, _ := fs.ReadFile(indexHTML, "index.html")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(idx)
	})
	http.HandleFunc("/chat", hub.handleChat)
	http.HandleFunc("/events", hub.handleEvents)
	http.HandleFunc("/approve", hub.handleApprove)
	http.HandleFunc("/plan-execute", hub.handlePlanExecute)
	http.HandleFunc("/plan-cancel", hub.handlePlanCancel)
	http.HandleFunc("/plan-retry", hub.handlePlanRetry)
	http.HandleFunc("/plan-replan", hub.handlePlanReplan)
	http.HandleFunc("GET /sessions", hub.handleListSessions)
	http.HandleFunc("POST /sessions", hub.handleCreateSession)
	http.HandleFunc("PATCH /sessions", hub.handleUpdateSession)
	http.HandleFunc("DELETE /sessions", hub.handleDeleteSession)

	// ── Team endpoints ──
	th := newTeamHub(llm, modelID, mem, teamStore)

	tidx, _ := fs.ReadFile(teamHTML, "team.html")
	http.HandleFunc("/team", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(tidx)
	})
	http.HandleFunc("/team/chat", th.handleChat)
	http.HandleFunc("/team/events", th.handleEvents)
	http.HandleFunc("/team/approve", th.handleApprove)
	http.HandleFunc("/team/agents", th.handleAgents)
	http.HandleFunc("GET /team/sessions", th.handleListSessions)
	http.HandleFunc("POST /team/sessions", th.handleCreateSession)
	http.HandleFunc("PATCH /team/sessions", th.handleUpdateSession)
	http.HandleFunc("DELETE /team/sessions", th.handleDeleteSession)

	log.Println("http://localhost:8080  (single + /plan)   http://localhost:8080/team  (team)")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// ── Shared types ──

type sseEvent struct {
	Type         string          `json:"type"`
	Text         string          `json:"text,omitempty"`
	Agent        string          `json:"agent,omitempty"`
	ToolCall     *toolCallJSON   `json:"tool_call,omitempty"`
	ToolCallID   string          `json:"tool_call_id,omitempty"`
	HandoffTo    string          `json:"handoff_to,omitempty"`
	StepID       string          `json:"step_id,omitempty"`
	PlanDef      json.RawMessage `json:"plan_def,omitempty"` // plan_generated
	Error        string          `json:"error,omitempty"`
	Stage        json.RawMessage `json:"stage,omitempty"`    // stage event detail (pipeline panel)
	PromptTokens int             `json:"prompt_tokens,omitempty"`
	ContextWindow int            `json:"context_window,omitempty"`
}

// stageData is the JSON payload for "stage" SSE events.
type stageData struct {
	Name      string         `json:"name"`
	Phase     string         `json:"phase"`
	Detail    map[string]any `json:"detail,omitempty"`
	DurationMs int64         `json:"duration_ms,omitempty"`
	Err       string         `json:"error,omitempty"`
}

type toolCallJSON struct {
	ID       string `json:"id"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// webuiObserver implements openagent.RunObserver, forwarding stage events
// to the SSE bus so the frontend pipeline panel can render live progress.
type webuiObserver struct {
	bus *eventbus.Bus[sseEvent]
	sid string
}

func (o *webuiObserver) ObserveStage(ctx context.Context, evt openagent.StageEvent) {
	sd := stageData{
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
	o.bus.Publish(o.sid, sseEvent{Type: "stage", Stage: b})
}

var _ openagent.RunObserver = (*webuiObserver)(nil)

type pendingApproval struct {
	respond  chan approveResponse
	toolCall openagent.ToolCall // cached for path extraction when granting dir permission
}

type approveResponse struct {
	allowed bool
	reason  string
	scope   string // "" = once, "dir" = remember for directory
}


func writeSSE(w http.ResponseWriter, e sseEvent) {
	b, _ := json.Marshal(e)
	fmt.Fprintf(w, "data: %s\n\n", b)
}

// ── Single-agent Hub ──

type hub struct {
	mu       sync.Mutex
	sessions map[string]*session
	llm      openagent.Model
	modelID  string
	tools    []openagent.Tool
	mem      openagent.Memory
	bus      *eventbus.Bus[sseEvent]
	store     *sessionStore
	planStore *planStore // persists plan state across restarts

	// Plan support — /plan command in chat.
	planAgents    map[string]openagent.AgentRunner
	planAgentInfo []openagent.AgentInfo
	planner       *plan.LLMPlanner

	planMu      sync.Mutex
	planCancels map[string]context.CancelFunc // sessionID → cancel

	// Plan pause/resume support (AutoReplan=false).
	planStates   map[string]*plan.PlanState     // sessionID → execution state (when paused)
	planRetryChs map[string]chan plan.RetryAction // sessionID → resume channel
	workDir      string                         // sandbox workspace root

	// Permission store: session → (tool:path → "dir"|"once").
	// "read:/home/user/src/" = allow reads in that directory tree.
	permStore map[string]map[string]string
}

type session struct {
	id        string
	agent     *openagent.Agent
	oaSession openagent.Session
	title     string
	msgCount  int

	mu              sync.Mutex
	pendingApproval *pendingApproval
	approvalDone    chan struct{} // closed when current approval is resolved
}

func newHub(llm openagent.Model, modelID string, tools []openagent.Tool, mem openagent.Memory, store *sessionStore, pstore *planStore, workDir string) *hub {
	h := &hub{
		sessions:    make(map[string]*session),
		llm:         llm, modelID: modelID, tools: tools, mem: mem,
		bus:         eventbus.New[sseEvent](1000),
		store:       store,
		planStore:   pstore,
		planAgents:  make(map[string]openagent.AgentRunner),
		planner:     plan.NewLLMPlanner(llm),
		planCancels:  make(map[string]context.CancelFunc),
		planStates:   make(map[string]*plan.PlanState),
		planRetryChs: make(map[string]chan plan.RetryAction),
		workDir:      workDir,
		permStore:    make(map[string]map[string]string),
	}

	// Plan agents for /plan command.
	planRunner := func(name, instructions string) *openagent.Agent {
		return openagent.NewAgent(name,
			openagent.WithModel(llm),
			openagent.WithInstructions(instructions),
			openagent.WithMaxTurns(10),
			openagent.WithTools(tools...),
		)
	}

	h.addPlanAgent("researcher", "Researches technical topics — provides comprehensive analysis with pros/cons and recommendations", planRunner("researcher", "Research the given topic thoroughly. Provide comprehensive analysis with pros/cons, alternatives, and data-driven recommendations. Be objective."))
	h.addPlanAgent("architect", "Designs software architecture — produces structured design documents with components and data flow", planRunner("architect", "Design software architecture. Produce clear design with components, interfaces, and data flow. Be specific and structured."))
	h.addPlanAgent("coder", "Writes production-quality Go code with error handling and comments", planRunner("coder", "Write production-quality Go code. Include comments, handle errors, follow idiomatic Go."))
	h.addPlanAgent("reviewer", "Reviews code for correctness, style, and potential bugs", planRunner("reviewer", "Review code for correctness, style, and bugs. List specific issues and suggestions. Be constructive."))
	h.addPlanAgent("writer", "Writes documentation: README, API docs, reports. Uses markdown", planRunner("writer", "Write clear, professional documentation. Use markdown formatting."))

	return h
}

func (h *hub) addPlanAgent(name, desc string, runner openagent.AgentRunner) {
	h.planAgents[name] = runner
	at := openagent.AgentExternal
	if _, ok := runner.(*openagent.Agent); ok {
		at = openagent.AgentInternal
	}
	h.planAgentInfo = append(h.planAgentInfo, openagent.AgentInfo{Name: name, Description: desc, Type: at})
}

func (h *hub) getOrCreate(id string) *session {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s, ok := h.sessions[id]; ok {
		return s
	}
	s := &session{
		id: id,
		oaSession: openagent.Session{
			ID: id, UserID: "user", AgentName: "assistant",
			ModelID: h.modelID, CreatedAt: time.Now(),
			ProjectContext: fmt.Sprintf("Workspace: %s. The agent operates within this directory.", h.workDir),
		},
	}
	// Load title from persisted store.
	if meta := h.store.Get(id); meta != nil {
		s.title = meta.Title
	} else {
		h.store.Create(id, "")
	}

	s.agent = openagent.NewAgent("assistant",
		openagent.WithModel(h.llm),
		openagent.WithInstructions("You are a capable assistant with access to shell, read, write, ls, and grep tools. The shell starts in the workspace root — use relative paths for all operations. Explore files, run commands, and edit code to help the user. Be concise and action-oriented."),
		openagent.WithMemory(h.mem),
		openagent.WithRunObserver(&webuiObserver{bus: h.bus, sid: id}),
		openagent.WithApprover(&webApprover{
			workDir:   h.workDir,
			permStore: h.permStore,
			sid:       id,
			submit: func(call openagent.ToolCall, resp chan approveResponse) {
				h.submitApproval(s, call, resp)
			}}),
		openagent.WithTools(h.tools...),
	)
	h.sessions[id] = s
	return s
}

func (h *hub) handleChat(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Session     string `json:"session"`
		Message     string `json:"message"`
		Feedback    string `json:"feedback,omitempty"`
		CurrentPlan string `json:"current_plan,omitempty"` // JSON of current PlanDef for replan
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Session == "" || body.Message == "" {
		http.Error(w, "session and message required", 400)
		return
	}
	s := h.getOrCreate(body.Session)

	// Auto-title: use first user message if none set.
	if s.title == "" {
		title := body.Message
		if len([]rune(title)) > 50 {
			title = string([]rune(title)[:50])
		}
		s.title = title
		h.store.Update(s.id, title)
	}
	s.msgCount++

	s.mu.Lock()
	s.pendingApproval = nil
	s.mu.Unlock()

	// Publish user message so other tabs see it.
	h.bus.Publish(body.Session, sseEvent{Type: "user_message", Text: body.Message})

	// ── /plan command: generate a DAG from the goal ──
	if strings.HasPrefix(body.Message, "/plan ") {
		goal := strings.TrimPrefix(body.Message, "/plan ")
		go h.handlePlanCommand(s, goal, body.Feedback, body.CurrentPlan)
		return
	}

	// ── /goal command: autonomous goal-driven agent loop ──
	if strings.HasPrefix(body.Message, "/goal ") {
		goal := strings.TrimPrefix(body.Message, "/goal ")
		go h.handleGoalCommand(s, goal)
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			ch := s.agent.RunStreamWithPrefix(ctx, s.oaSession, nil,
			openagent.UserMessage(body.Message))
		for evt := range ch {
			se := streamToSSE(evt)
			h.bus.Publish(body.Session, se)
		}
	}()
}

func (h *hub) handleEvents(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("PANIC in SSE handler for session %s: %v", r.URL.Query().Get("session"), rec)
		}
	}()

	sid := r.URL.Query().Get("session")
	if sid == "" {
		http.Error(w, "session required", 400)
		return
	}
	_ = h.getOrCreate(sid) // ensure session exists
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(200)
	flusher.Flush()

	// ── Replay full history from Memory (authoritative source) ──
	// Memory is the source of truth; EventBus only handles live events.
	// We replay BEFORE subscribing so there are no duplicates.
	if h.mem != nil {
		msgs, _ := h.mem.Recent(r.Context(), sid, historyReplayLimit)
		// Skip leading orphaned tool results — they need a preceding
		// assistant message with tool_calls, which may have been truncated.
		start := 0
		for start < len(msgs) && msgs[start].Role == openagent.RoleTool {
			start++
		}
		for _, msg := range msgs[start:] {
			switch msg.Role {
			case openagent.RoleUser:
				writeSSE(w, sseEvent{Type: "user_message", Text: msg.Content})
			case openagent.RoleAssistant:
				writeSSE(w, sseEvent{Type: "message", Text: msg.Content})
			case openagent.RoleTool:
				writeSSE(w, sseEvent{Type: "tool_result", Text: msg.Content, ToolCallID: msg.ToolCallID})
			}
			flusher.Flush()
		}
	}

	// ── Restore persisted plan ──
	if pp := h.planStore.Load(sid); pp != nil && pp.Def != nil {
		b, _ := json.Marshal(pp.Def)
		se := sseEvent{Type: "plan_restore", PlanDef: b}
		if pp.State != nil {
			sj, _ := json.Marshal(pp.State.Results)
			se.Text = string(sj)
		}
		writeSSE(w, se)
		flusher.Flush()
	}

	// ── Subscribe for live events only (no history replay) ──
	// Memory already handled history; EventBus only forwards new events.
	sub := h.bus.SubscribeLive(sid)
	defer h.bus.Unsubscribe(sid, sub)

	for {
		select {
		case <-r.Context().Done():
			return
		case e, ok := <-sub.C:
			if !ok {
				return
			}
			writeSSE(w, e)
			flusher.Flush()
		}
	}
}

func (h *hub) handleApprove(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Session string `json:"session"`
		Allowed bool   `json:"allowed"`
		Scope   string `json:"scope,omitempty"` // "" = once, "dir" = remember for directory
	}
	json.NewDecoder(r.Body).Decode(&body)
	s := h.getOrCreate(body.Session)
	if resolveApproval(s, body.Allowed, body.Scope, w) {
		h.bus.Publish(body.Session, sseEvent{Type: "tool_approval_cancelled"})
	}
}

func (h *hub) submitApproval(s *session, call openagent.ToolCall, resp chan approveResponse) {
	// Queue: wait for any in-flight approval to resolve before starting a new one.
	s.mu.Lock()
	for s.pendingApproval != nil {
		done := s.approvalDone
		s.mu.Unlock()
		<-done
		s.mu.Lock()
	}
	s.approvalDone = make(chan struct{})
	s.pendingApproval = &pendingApproval{respond: resp, toolCall: call}
	s.mu.Unlock()

	b, _ := json.Marshal(call)
	tcj := &toolCallJSON{}
	json.Unmarshal(b, &tcj)
	se := sseEvent{Type: "tool_approval", ToolCall: tcj}
	h.bus.Publish(s.id, se)
}

func resolveApproval(s *session, allowed bool, scope string, w http.ResponseWriter) bool {
	s.mu.Lock()
	p := s.pendingApproval
	done := s.approvalDone
	s.pendingApproval = nil
	s.approvalDone = nil
	s.mu.Unlock()

	if p == nil {
		http.Error(w, "no pending approval", 400)
		return false
	}
	reason := "user denied"
	if allowed {
		reason = "user approved"
	}
	p.respond <- approveResponse{allowed: allowed, reason: reason, scope: scope}
	close(done) // unblock next queued approval
	return true
}

// ── Session CRUD (single agent) ──

type sessionInfoJSON struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	MsgCount  int       `json:"msgCount"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (h *hub) handleListSessions(w http.ResponseWriter, r *http.Request) {
	list := h.store.List()
	infos := make([]sessionInfoJSON, 0, len(list))
	for _, m := range list {
		msgCount := 0
		h.mu.Lock()
		if s, ok := h.sessions[m.ID]; ok {
			msgCount = s.msgCount
		}
		h.mu.Unlock()
		infos = append(infos, sessionInfoJSON{
			ID: m.ID, Title: m.Title, MsgCount: msgCount,
			CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(infos)
}

func (h *hub) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	id := generateSessionID()
	meta := h.store.Create(id, body.Title)

	// Pre-populate the session in hub so it's ready.
	h.getOrCreate(id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sessionInfoJSON{
		ID: meta.ID, Title: meta.Title,
		CreatedAt: meta.CreatedAt, UpdatedAt: meta.UpdatedAt,
	})
}

func (h *hub) handleUpdateSession(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id required", 400)
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	if !h.store.Update(id, body.Title) {
		http.Error(w, "session not found", 404)
		return
	}

	// Update cached title.
	h.mu.Lock()
	if s, ok := h.sessions[id]; ok {
		s.title = body.Title
	}
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *hub) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id required", 400)
		return
	}

	// Notify tabs viewing this session before removing it.
	h.bus.Publish(id, sseEvent{Type: "session_deleted"})

	h.store.Delete(id)

	h.mu.Lock()
	delete(h.sessions, id)
	delete(h.permStore, id)
	h.mu.Unlock()

	if h.mem != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		_ = h.mem.DeleteSession(ctx, id)
	}

	w.WriteHeader(http.StatusNoContent)
}

func generateSessionID() string {
	return "s" + time.Now().Format("20060102") + "-" + randomHex(8)
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

// ── Plan command handling (/plan <goal> in chat) ──

func (h *hub) handlePlanCommand(s *session, goal string, feedback string, currentPlanJSON string) {
	// Cancel any running plan for this session — generating a new plan
	// replaces the old one. Otherwise both the old executor and the new
	// plan events would stream to the frontend simultaneously.
	h.planMu.Lock()
	if cancel, ok := h.planCancels[s.id]; ok {
		cancel()
		delete(h.planCancels, s.id)
	}
	delete(h.planStates, s.id)
	delete(h.planRetryChs, s.id)
	h.planMu.Unlock()

	h.bus.Publish(s.id, sseEvent{Type: "plan_cancelled"})

	// Tell the frontend planning has started so it can show a loading indicator.
	label := goal
	if feedback != "" {
		label = goal + " (replan)"
	}
	h.bus.Publish(s.id, sseEvent{Type: "plan_planning", Text: label})

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Build the planner prompt. When replanning (feedback + current plan),
	// include the existing plan so the LLM can improve upon it.
	promptGoal := goal
	if feedback != "" && currentPlanJSON != "" {
		// Parse and re-serialise for consistent formatting.
		var cur plan.PlanDef
		if err := json.Unmarshal([]byte(currentPlanJSON), &cur); err == nil {
			pretty, _ := json.MarshalIndent(cur, "", "  ")
			promptGoal = fmt.Sprintf(
				"Original goal: %s\n\n"+
					"Current plan that needs improvement:\n%s\n\n"+
					"User feedback on this plan: %s\n\n"+
					"Please regenerate the plan. Keep what works, fix what the user pointed out. "+
					"Use the feedback to guide changes — re-assign agents, rephrase tasks, or restructure the DAG.",
				goal, string(pretty), feedback,
			)
		}
	} else if feedback != "" {
		promptGoal = goal + "\n\nUser feedback: " + feedback
	}

	def, err := h.planner.Plan(ctx, promptGoal, h.planAgentInfo, nil)
	if err != nil {
		h.bus.Publish(s.id, sseEvent{Type: "plan_error", Error: err.Error()})
		return
	}

	b, _ := json.Marshal(def)
	h.bus.Publish(s.id, sseEvent{Type: "plan_generated", PlanDef: b})

	// Persist the plan def (state=nil means "generated, not yet executed").
	h.planStore.Save(s.id, def, nil)
}

// ── /goal command: autonomous goal-driven agent loop ──

func (h *hub) handleGoalCommand(s *session, goal string) {
	// Cancel any running plan — goal mode replaces the current activity.
	h.planMu.Lock()
	if cancel, ok := h.planCancels[s.id]; ok {
		cancel()
		delete(h.planCancels, s.id)
	}
	delete(h.planStates, s.id)
	delete(h.planRetryChs, s.id)
	h.planMu.Unlock()

	h.bus.Publish(s.id, sseEvent{Type: "plan_cancelled"})

	// Goal mode: clone the session's agent and boost the turn limit so the
	// agent can iterate autonomously. The goal is injected into the system
	// prompt (via RunGoalStream), persisting across all turns.
	goalAgent := s.agent.Clone()
	goalAgent.MaxTurns = 50

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	ch := goalAgent.RunGoalStream(ctx, s.oaSession, goal)
	for evt := range ch {
		se := streamToSSE(evt)
		h.bus.Publish(s.id, se)
	}
}

func (h *hub) handlePlanExecute(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Session string           `json:"session"`
		Def     json.RawMessage  `json:"def"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Session == "" || len(body.Def) == 0 {
		http.Error(w, "session and def required", 400)
		return
	}

	var def plan.PlanDef
	if err := json.Unmarshal(body.Def, &def); err != nil {
		http.Error(w, "invalid plan def", 400)
		return
	}

	s := h.getOrCreate(body.Session)

	// Cancel any running plan for this session.
	h.planMu.Lock()
	if cancel, ok := h.planCancels[body.Session]; ok {
		cancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	h.planCancels[body.Session] = cancel
	h.planMu.Unlock()

	// Build plan with session-specific agents (with approver).
	planOpts := []plan.PlanOption{
		plan.WithPlanner(h.planner),
		plan.WithModel(h.llm),
		plan.WithMaxConcurrency(8),
		plan.WithAutoReplan(false), // pause on failure for manual retry/replan
	}
	for name, runner := range h.planAgents {
		desc := ""
		for _, info := range h.planAgentInfo {
			if info.Name == name {
				desc = info.Description
				break
			}
		}
		// Clone with session-specific approver.
		if ag, ok := runner.(*openagent.Agent); ok {
			runner = openagent.NewAgent(ag.Name,
				openagent.WithModel(ag.Model),
				openagent.WithInstructions(ag.Instructions),
				openagent.WithMaxTurns(ag.MaxTurns),
				openagent.WithTools(ag.Tools...),
				openagent.WithRunObserver(&webuiObserver{bus: h.bus, sid: body.Session}),
				openagent.WithApprover(&webApprover{
					workDir:   h.workDir,
					permStore: h.permStore,
					sid:       s.id,
					submit: func(call openagent.ToolCall, resp chan approveResponse) {
						h.submitApproval(s, call, resp)
					}}),
			)
		}
		planOpts = append(planOpts, plan.WithAgent(name, desc, runner))
	}
	p := plan.NewPlan(planOpts...)

	sid := body.Session
	oaSession := openagent.Session{
		ID:        sid,
		AgentName: "plan",
		CreatedAt: s.oaSession.CreatedAt,
	}

	go func() {
		defer func() {
			h.planMu.Lock()
			delete(h.planCancels, sid)
			delete(h.planStates, sid)
			delete(h.planRetryChs, sid)
			h.planMu.Unlock()
		}()

		// Create plan state once and reuse across pause/resume cycles.
		state := &plan.PlanState{
			ID:        sid + "/plan",
			Goal:      def.Goal,
			Status:    plan.PlanStatusApproved,
			Steps:     def.Steps,
			Results:   make(map[string]*plan.StepResult),
			CreatedAt: s.oaSession.CreatedAt,
			UpdatedAt: s.oaSession.CreatedAt,
		}

		for {
			// Execute (or resume after pause).
			ch := p.ExecuteWithState(ctx, oaSession, &def, state)

			paused := false
			for evt := range ch {
				se := planEventToWebuiSSE(evt)
				if se.Type == "" {
					continue
				}
				h.bus.Publish(sid, se)

				// Persist after state-changing events.
				switch se.Type {
				case "plan_step_done", "plan_step_failed":
					h.planStore.Save(sid, &def, state)
				case "plan_waiting_retry":
					h.planStore.Save(sid, &def, state)
					h.planMu.Lock()
					h.planStates[sid] = state
					h.planRetryChs[sid] = make(chan plan.RetryAction, 1)
					h.planMu.Unlock()
					paused = true
					break
				case "plan_done":
					h.planStore.Delete(sid)
				case "plan_error":
					h.planStore.Save(sid, &def, state)
				}
			}

			if !paused {
				return // plan_done, plan_error, or plan_cancelled
			}

			// Wait for retry or replan command.
			h.planMu.Lock()
			retryCh := h.planRetryChs[sid]
			h.planMu.Unlock()

			select {
			case action := <-retryCh:
				switch action.Action {
				case "retry":
					if sr := state.Results[action.StepID]; sr != nil {
						sr.Status = plan.StepStatusPending
						sr.Error = ""
						sr.Retries = 0
					}
				case "replan":
					def = *action.NewDef
					state.Steps = def.Steps
					state.ReplanCount++
					state.UpdatedAt = time.Now()
					h.planStore.Save(sid, &def, state)
				}
			case <-ctx.Done():
				h.bus.Publish(sid, sseEvent{Type: "plan_cancelled"})
				return
			}
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func (h *hub) handlePlanCancel(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Session string `json:"session"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Session == "" {
		http.Error(w, "session required", 400)
		return
	}

	h.planMu.Lock()
	cancel, ok := h.planCancels[body.Session]
	if ok {
		cancel()
		delete(h.planCancels, body.Session)
	}
	h.planMu.Unlock()

	h.bus.Publish(body.Session, sseEvent{Type: "plan_cancelled"})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

// handlePlanRetry resumes a paused plan by retrying a failed step.
func (h *hub) handlePlanRetry(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Session string `json:"session"`
		StepID  string `json:"step_id"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Session == "" || body.StepID == "" {
		http.Error(w, "session and step_id required", 400)
		return
	}

	h.planMu.Lock()
	state := h.planStates[body.Session]
	retryCh := h.planRetryChs[body.Session]
	h.planMu.Unlock()

	if state == nil || retryCh == nil {
		http.Error(w, "plan not waiting for retry", 400)
		return
	}

	sr := state.Results[body.StepID]
	if sr == nil || sr.Status != plan.StepStatusFailed {
		http.Error(w, "step not found or not failed", 400)
		return
	}

	retryCh <- plan.RetryAction{Action: "retry", StepID: body.StepID}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "retrying"})
}

// handlePlanReplan regenerates the affected subtree with user feedback.
func (h *hub) handlePlanReplan(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Session  string `json:"session"`
		StepID   string `json:"step_id"`
		Feedback string `json:"feedback"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Session == "" || body.StepID == "" || body.Feedback == "" {
		http.Error(w, "session, step_id, and feedback required", 400)
		return
	}

	h.planMu.Lock()
	state := h.planStates[body.Session]
	retryCh := h.planRetryChs[body.Session]
	h.planMu.Unlock()

	if state == nil || retryCh == nil {
		http.Error(w, "plan not waiting for retry", 400)
		return
	}

	// Find current def from the stored state.
	def := &plan.PlanDef{Goal: state.Goal, Steps: state.Steps}

	h.bus.Publish(body.Session, sseEvent{Type: "plan_replanning", StepID: body.StepID})

	// Build a temporary plan instance to use ReplanWithFeedback.
	planOpts := []plan.PlanOption{
		plan.WithPlanner(h.planner),
		plan.WithModel(h.llm),
	}
	for name, runner := range h.planAgents {
		desc := ""
		for _, info := range h.planAgentInfo {
			if info.Name == name {
				desc = info.Description
				break
			}
		}
		planOpts = append(planOpts, plan.WithAgent(name, desc, runner))
	}
	p := plan.NewPlan(planOpts...)

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	newDef, err := p.ReplanWithFeedback(ctx, def, state, body.StepID, body.Feedback)
	if err != nil {
		h.bus.Publish(body.Session, sseEvent{Type: "plan_error", Error: err.Error()})
		http.Error(w, `{"error":"`+err.Error()+`"}`, 500)
		return
	}

	// Emit new plan so UI re-renders DAG.
	b, _ := json.Marshal(newDef)
	h.bus.Publish(body.Session, sseEvent{Type: "plan_generated", PlanDef: b})

	retryCh <- plan.RetryAction{Action: "replan", NewDef: newDef}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "replanned"})
}

func planEventToWebuiSSE(evt plan.PlanEvent) sseEvent {
	switch evt.Type {
	case plan.PlanEventStepStart:
		return sseEvent{Type: "plan_step_start", StepID: evt.StepID, Agent: evt.Agent}
	case plan.PlanEventTextDelta:
		return sseEvent{Type: "plan_step_text", StepID: evt.StepID, Text: evt.Text}
	case plan.PlanEventToolCall:
		return sseEvent{
			Type: "plan_step_tool", StepID: evt.StepID,
			ToolCall: &toolCallJSON{
				ID: evt.ToolID,
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{Name: evt.ToolName, Arguments: evt.ToolArgs},
			},
		}
	case plan.PlanEventToolProgress:
		return sseEvent{Type: "plan_step_tool_progress", StepID: evt.StepID, Text: evt.Text}
	case plan.PlanEventToolResult:
		return sseEvent{Type: "plan_step_tool_result", StepID: evt.StepID, Text: evt.Text}
	case plan.PlanEventStepDone:
		se := sseEvent{Type: "plan_step_done", StepID: evt.StepID, Agent: evt.Agent}
		if evt.Result != nil {
			se.Text = evt.Result.Summary
		}
		return se
	case plan.PlanEventStepFailed:
		return sseEvent{Type: "plan_step_failed", StepID: evt.StepID, Agent: evt.Agent, Error: evt.ErrText}
	case plan.PlanEventReplanning:
		return sseEvent{Type: "plan_replanning", StepID: evt.StepID}
	case plan.PlanEventWaitingRetry:
		return sseEvent{Type: "plan_waiting_retry", StepID: evt.StepID, Error: evt.ErrText}
	case plan.PlanEventDone:
		return sseEvent{Type: "plan_done", Text: evt.Text}
	case plan.PlanEventError:
		return sseEvent{Type: "plan_error", Error: evt.ErrText}
	}
	return sseEvent{}
}

// ── Approver ──

type webApprover struct {
	submit    func(call openagent.ToolCall, resp chan approveResponse)
	workDir   string // workspace root; file ops within this path are auto-approved
	permStore map[string]map[string]string // session → (tool:path → scope)
	sid       string // current session ID
}

func (a *webApprover) Approve(ctx context.Context, call openagent.ToolCall, _ openagent.FunctionDefinition, _ openagent.Session) (bool, string) {
	// Auto-approve file tools when the path is within the workspace.
	if a.workDir != "" {
		switch call.Function.Name {
		case "read", "write", "ls", "grep":
			if a.pathWithinWorkspace(call.Function.Arguments) {
				return true, "workspace"
			}
		}
	}

	// Check permission store — user may have granted dir-level access earlier.
	if reason := a.checkPerm(call.Function.Name, call.Function.Arguments); reason != "" {
		return true, reason
	}

	resp := make(chan approveResponse, 1)
	a.submit(call, resp)
	select {
	case <-ctx.Done():
		return false, "cancelled"
	case r := <-resp:
		if r.allowed && r.scope == "dir" {
			a.grantPerm(call.Function.Name, call.Function.Arguments)
		}
		return r.allowed, r.reason
	}
}

// checkPerm looks up the permission store. Returns non-empty reason if allowed.
func (a *webApprover) checkPerm(tool, rawArgs string) string {
	path := a.extractPath(rawArgs)
	if path == "" {
		return ""
	}
	store := a.permStore[a.sid]
	if store == nil {
		return ""
	}
	// Check exact match then ancestor directories.
	for p := path; p != "" && p != "/"; p = filepath.Dir(p) {
		if store[tool+":"+p] == "dir" {
			return "remembered: " + p
		}
		if p == filepath.Dir(p) {
			break // root, don't loop
		}
	}
	return ""
}

// grantPerm stores a directory-level permission for this session.
func (a *webApprover) grantPerm(tool, rawArgs string) {
	path := a.extractPath(rawArgs)
	if path == "" {
		return
	}
	dir := filepath.Dir(path)
	if a.permStore[a.sid] == nil {
		a.permStore[a.sid] = make(map[string]string)
	}
	a.permStore[a.sid][tool+":"+dir] = "dir"
}

// extractPath returns the file path from tool arguments, or "" if not a file tool.
func (a *webApprover) extractPath(rawArgs string) string {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil || args.Path == "" {
		return ""
	}
	return filepath.Clean(args.Path)
}

func (a *webApprover) pathWithinWorkspace(rawArgs string) bool {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil || args.Path == "" {
		return false
	}
	p := filepath.Clean(args.Path)

	// Resolve to absolute: join with workDir for relative paths,
	// or use directly for absolute paths.
	var abs string
	var err error
	if filepath.IsAbs(p) {
		abs, err = filepath.Abs(p)
	} else {
		abs, err = filepath.Abs(filepath.Join(a.workDir, p))
	}
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(a.workDir, abs)
	return err == nil && !strings.HasPrefix(rel, "..")
}

// ── Single-agent SSE ──

func streamToSSE(evt openagent.StreamEvent) sseEvent {
	switch evt.Type {
	case openagent.StreamThought:
		return sseEvent{Type: "thought", Text: evt.Text}
	case openagent.StreamTextDelta:
		return sseEvent{Type: "delta", Text: evt.Text}
	case openagent.StreamToolCall:
		if len(evt.Message.ToolCalls) > 0 {
			b, _ := json.Marshal(evt.Message.ToolCalls[0])
			tcj := &toolCallJSON{}
			json.Unmarshal(b, &tcj)
			return sseEvent{Type: "tool_call", ToolCall: tcj}
		}
	case openagent.StreamToolResult:
		return sseEvent{Type: "tool_result", Text: evt.Message.Content, ToolCallID: evt.Message.ToolCallID}
	case openagent.StreamRetrying:
		return sseEvent{Type: "retrying", Text: evt.Error.Error()}
	case openagent.StreamDone:
		se := sseEvent{Type: "done", Text: evt.Result.FinalOutput}
		if evt.Result != nil {
			se.PromptTokens = evt.Result.Usage.PromptTokens
			se.ContextWindow = evt.Result.ContextWindow
			log.Printf("usage: prompt=%d ctx_window=%d", evt.Result.Usage.PromptTokens, evt.Result.ContextWindow)
		}
		return se
	case openagent.StreamError:
		return sseEvent{Type: "error", Text: evt.Error.Error()}
	}
	return sseEvent{}
}

// ── Team Hub ──

type teamHub struct {
	mu       sync.Mutex
	sessions map[string]*teamSession
	llm      openagent.Model
	modelID  string
	mem      openagent.Memory
	bus      *eventbus.Bus[sseEvent]
	store    *sessionStore
}

type teamSession struct {
	id      string
	team    *openagent.Team
	title   string

	mu              sync.Mutex
	pendingApproval *pendingApproval
	agentList       []agentInfoJSON // tracked for GET /team/agents
}

func newTeamHub(llm openagent.Model, modelID string, mem openagent.Memory, store *sessionStore) *teamHub {
	return &teamHub{
		sessions: make(map[string]*teamSession),
		llm:      llm, modelID: modelID, mem: mem,
		bus:   eventbus.New[sseEvent](500),
		store: store,
	}
}

func (h *teamHub) getOrCreate(id string) *teamSession {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s, ok := h.sessions[id]; ok {
		return s
	}
	s := &teamSession{
		id: id,
	}
	// Load title from persisted store.
	if meta := h.store.Get(id); meta != nil {
		s.title = meta.Title
	} else {
		h.store.Create(id, "")
	}
	// Team is built once and persisted on the session. This is required
	// for ACP agents (external processes) which cannot be recreated per message.
	s.team = h.buildTeam(s)
	s.agentList = []agentInfoJSON{
		{Name: "analyst", Description: "Understands requirements and produces specifications", Type: "internal"},
		{Name: "designer", Description: "Designs architecture, components, and data flow", Type: "internal"},
		{Name: "coder", Description: "Writes clean, idiomatic Go code with error handling", Type: "internal"},
		{Name: "tester", Description: "Writes tests, identifies edge cases, reports results", Type: "internal"},
		{Name: "reviewer", Description: "Reviews code for correctness, style, and security", Type: "internal"},
	}
	h.sessions[id] = s
	return s
}

func (h *teamHub) handleChat(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Session string `json:"session"`
		Message string `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Session == "" || body.Message == "" {
		http.Error(w, "session and message required", 400)
		return
	}
	ts := h.getOrCreate(body.Session)

	// Auto-title.
	if ts.title == "" {
		title := body.Message
		if len([]rune(title)) > 50 {
			title = string([]rune(title)[:50])
		}
		ts.title = title
		h.store.Update(ts.id, title)
	}

	ts.mu.Lock()
	ts.pendingApproval = nil
	ts.mu.Unlock()

	h.bus.Publish(body.Session, sseEvent{Type: "user_message", Text: body.Message})

	go func() {
		oaSession := openagent.Session{
			ID: body.Session, UserID: "user", AgentName: "team",
			ModelID: h.modelID, CreatedAt: time.Now(),
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			ch := ts.team.RunStream(ctx, oaSession, openagent.UserMessage(body.Message))
		for evt := range ch {
			se := teamEventToSSE(ts, evt)
			if se.Type == "" {
				continue
			}
			h.bus.Publish(body.Session, se)
		}
	}()
}

func (h *teamHub) buildTeam(ts *teamSession) *openagent.Team {
	// Per-session approver closure.
	makeApprove := func() openagent.Approver {
		return &webApprover{sid: ts.id,
			submit: func(call openagent.ToolCall, resp chan approveResponse) {
				h.submitApproval(ts, call, resp)
			}}
	}

	analyst := openagent.NewAgent("analyst",
		openagent.WithModel(h.llm),
		openagent.WithMemory(h.mem),
		openagent.WithInstructions(`You are a requirements analyst. Your job:
1. Understand the user's request
2. Break it down into clear, testable requirements
3. Hand off to the designer with a structured specification
Be specific — include constraints, edge cases, and acceptance criteria.`),
		openagent.WithMaxTurns(2),
		openagent.WithApprover(makeApprove()),
	)

	designer := openagent.NewAgent("designer",
		openagent.WithModel(h.llm),
		openagent.WithMemory(h.mem),
		openagent.WithInstructions(`You are a software designer. Your job:
1. Take the analyst's specification and design the architecture
2. Define components, interfaces, and data flow
3. Hand off to the coder with a clear design document
Be specific about types, function signatures, and module boundaries.`),
		openagent.WithMaxTurns(2),
		openagent.WithApprover(makeApprove()),
	)

	coder := openagent.NewAgent("coder",
		openagent.WithModel(h.llm),
		openagent.WithMemory(h.mem),
		openagent.WithInstructions(`You are a software developer. Your job:
1. Take the designer's spec and write clean, idiomatic Go code
2. Include error handling, comments, and tests
3. Hand off the complete implementation to the tester
Output ONLY code with brief inline comments.`),
		openagent.WithMaxTurns(5),
		openagent.WithApprover(makeApprove()),
	)

	tester := openagent.NewAgent("tester",
		openagent.WithModel(h.llm),
		openagent.WithMemory(h.mem),
		openagent.WithInstructions(`You are a QA engineer. Your job:
1. Review the coder's implementation
2. Identify edge cases and write tests for them
3. If all tests pass, hand off to the reviewer with your test report
4. If tests fail, report the failures clearly — do NOT fix the code
Be thorough. List what you tested and why.`),
		openagent.WithMaxTurns(2),
		openagent.WithApprover(makeApprove()),
	)

	reviewer := openagent.NewAgent("reviewer",
		openagent.WithModel(h.llm),
		openagent.WithMemory(h.mem),
		openagent.WithInstructions(`You are a code reviewer. Your job:
1. Review the complete implementation and test results
2. Check for correctness, style, performance, and security
3. Produce a final review summary: approved, changes requested, or rejected
4. If approved, present the complete deliverable to the user
Do NOT hand off — you are the final gate.`),
		openagent.WithMaxTurns(1),
		openagent.WithApprover(makeApprove()),
	)

	return openagent.NewTeam(
		openagent.WithTeamAgent("analyst", "Understands requirements and produces specifications", analyst),
		openagent.WithTeamAgent("designer", "Designs architecture, components, and data flow", designer),
		openagent.WithTeamAgent("coder", "Writes clean, idiomatic Go code with error handling", coder),
		openagent.WithTeamAgent("tester", "Writes tests, identifies edge cases, reports results", tester),
		openagent.WithTeamAgent("reviewer", "Reviews code for correctness, style, and security", reviewer),
		openagent.WithTeamMaxHandoffs(5),
	)
}

func (h *teamHub) handleEvents(w http.ResponseWriter, r *http.Request) {
	sid := r.URL.Query().Get("session")
	if sid == "" {
		http.Error(w, "session required", 400)
		return
	}
	_ = h.getOrCreate(sid) // ensure session exists
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sub := h.bus.Subscribe(sid)
	defer h.bus.Unsubscribe(sid, sub)

	for {
		select {
		case <-r.Context().Done():
			return
		case e, ok := <-sub.C:
			if !ok {
				return
			}
			writeSSE(w, e)
			flusher.Flush()
		}
	}
}

func (h *teamHub) handleApprove(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Session string `json:"session"`
		Allowed bool   `json:"allowed"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	ts := h.getOrCreate(body.Session)
	if resolveApprovalTeam(ts, body.Allowed, w) {
		h.bus.Publish(body.Session, sseEvent{Type: "tool_approval_cancelled"})
	}
}

func (h *teamHub) submitApproval(ts *teamSession, call openagent.ToolCall, resp chan approveResponse) {
	b, _ := json.Marshal(call)
	tcj := &toolCallJSON{}
	json.Unmarshal(b, &tcj)
	se := sseEvent{Type: "tool_approval", ToolCall: tcj}

	ts.mu.Lock()
	ts.pendingApproval = &pendingApproval{respond: resp}
	ts.mu.Unlock()

	h.bus.Publish(ts.id, se)
}

func resolveApprovalTeam(ts *teamSession, allowed bool, w http.ResponseWriter) bool {
	ts.mu.Lock()
	p := ts.pendingApproval
	ts.pendingApproval = nil
	ts.mu.Unlock()

	if p == nil {
		http.Error(w, "no pending approval", 400)
		return false
	}
	reason := "user denied"
	if allowed {
		reason = "user approved"
	}
	p.respond <- approveResponse{allowed: allowed, reason: reason}
	return true
}

// ── Team Session CRUD ──

func (h *teamHub) handleListSessions(w http.ResponseWriter, r *http.Request) {
	list := h.store.List()
	infos := make([]sessionInfoJSON, 0, len(list))
	for _, m := range list {
		infos = append(infos, sessionInfoJSON{
			ID: m.ID, Title: m.Title,
			CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(infos)
}

func (h *teamHub) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	id := generateSessionID()
	meta := h.store.Create(id, body.Title)
	h.getOrCreate(id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sessionInfoJSON{
		ID: meta.ID, Title: meta.Title,
		CreatedAt: meta.CreatedAt, UpdatedAt: meta.UpdatedAt,
	})
}

func (h *teamHub) handleUpdateSession(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id required", 400)
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	if !h.store.Update(id, body.Title) {
		http.Error(w, "session not found", 404)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *teamHub) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id required", 400)
		return
	}
	h.store.Delete(id)

	h.bus.Publish(id, sseEvent{Type: "session_deleted"})

	h.mu.Lock()
	delete(h.sessions, id)
	h.mu.Unlock()

	if h.mem != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		_ = h.mem.DeleteSession(ctx, id)
	}
	w.WriteHeader(http.StatusNoContent)
}




// ── Team Agents API ──

type agentInfoJSON struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"` // "internal" or "external"
}

type addAgentRequest struct {
	Session      string `json:"session"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Instructions string `json:"instructions,omitempty"`
	Command      string `json:"command,omitempty"` // for ACP agents
}

func (h *teamHub) handleAgents(w http.ResponseWriter, r *http.Request) {
	sid := r.URL.Query().Get("session")
	if sid == "" {
		http.Error(w, "session required", 400)
		return
	}
	ts := h.getOrCreate(sid)

	switch r.Method {
	case http.MethodGet:
		h.listAgents(ts, w)
	case http.MethodPost:
		h.addAgent(ts, w, r)
	case http.MethodDelete:
		h.removeAgent(ts, w, r)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (h *teamHub) listAgents(ts *teamSession, w http.ResponseWriter) {
	// Access team internals via reflection or expose the agent list.
	// For now, use the AgentInfo from the team's perspective.
	// Team doesn't expose a direct list method, so we maintain our own.
	// We'll use a workaround: parse from the team struct.
	// Actually, let's just return what we have in our session.

	// The team's agents are unexported. We need an exported method.
	// For the example, maintain a separate tracking list on teamSession.
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// We'll store agent metadata on teamSession when adding agents.
	// For now, return what we've tracked.
	agents := ts.agentList
	if agents == nil {
		agents = []agentInfoJSON{}
	}

	json.NewEncoder(w).Encode(map[string]any{"agents": agents})
}

func (h *teamHub) addAgent(ts *teamSession, w http.ResponseWriter, r *http.Request) {
	var req addAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body: "+err.Error(), 400)
		return
	}
	if req.Name == "" || req.Description == "" {
		http.Error(w, "name and description required", 400)
		return
	}

	var runner openagent.AgentRunner

	if req.Command != "" {
		// ACP external agent.
		acpRunner, err := openacp.New(req.Name, req.Command)
		if err != nil {
			http.Error(w, "failed to create ACP runner: "+err.Error(), 500)
			return
		}
		runner = acpRunner
	} else {
		// In-process agent.
		agent := openagent.NewAgent(req.Name,
			openagent.WithModel(h.llm),
			openagent.WithMemory(h.mem),
			openagent.WithDescription(req.Description),
			openagent.WithInstructions(req.Instructions),
			openagent.WithMaxTurns(5),
		)
		runner = agent
	}

	if err := ts.team.AddAgent(req.Name, req.Description, runner); err != nil {
		http.Error(w, "add agent: "+err.Error(), 409)
		return
	}

	// Track agent metadata for list.
	ts.mu.Lock()
	at := "internal"
	if req.Command != "" {
		at = "external"
	}
	ts.agentList = append(ts.agentList, agentInfoJSON{
		Name: req.Name, Description: req.Description, Type: at,
	})
	ts.mu.Unlock()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "name": req.Name})
}

func (h *teamHub) removeAgent(ts *teamSession, w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "name required", 400)
		return
	}

	ts.team.RemoveAgent(name)

	ts.mu.Lock()
	filtered := ts.agentList[:0]
	for _, a := range ts.agentList {
		if a.Name != name {
			filtered = append(filtered, a)
		}
	}
	ts.agentList = filtered
	ts.mu.Unlock()

	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "removed": name})
}

// ── Team SSE ──

func teamEventToSSE(ts *teamSession, evt openagent.TeamEvent) sseEvent {
	switch evt.Type {
	case openagent.TeamAgentStart:
		return sseEvent{Type: "agent_start", Agent: evt.Agent}
	case openagent.TeamAgentEnd:
		se := sseEvent{Type: "agent_end", Agent: evt.Agent}
		if evt.Error != nil {
			se.Error = evt.Error.Error()
		}
		return se
	case openagent.TeamTextDelta:
		return sseEvent{Type: "text_delta", Agent: evt.Agent, Text: evt.Text}
	case openagent.TeamToolCall:
		b, _ := json.Marshal(evt.ToolCall)
		tcj := &toolCallJSON{}
		json.Unmarshal(b, &tcj)
		return sseEvent{Type: "tool_call", Agent: evt.Agent, ToolCall: tcj}
	case openagent.TeamToolResult:
		return sseEvent{Type: "tool_result", Agent: evt.Agent, Text: evt.Text}
	case openagent.TeamRetrying:
		return sseEvent{Type: "retrying", Agent: evt.Agent, Error: evt.Error.Error()}
	case openagent.TeamHandoff:
		return sseEvent{Type: "handoff", Agent: evt.Agent, HandoffTo: evt.Message}
	case openagent.TeamDone:
		se := sseEvent{Type: "done"}
		if evt.Result != nil {
			se.Text = evt.Result.FinalOutput
		}
		return se
	case openagent.TeamError:
		return sseEvent{Type: "error", Text: evt.Error.Error()}
	}
	return sseEvent{}
}

// ── Tool dedup ──

func mergeTools(builtin, plugins []openagent.Tool) []openagent.Tool {
	seen := make(map[string]bool, len(builtin))
	for _, t := range builtin {
		seen[t.Definition().Name] = true
	}
	added := 0
	for _, t := range plugins {
		name := t.Definition().Name
		if seen[name] {
			log.Printf("⚠ skipping duplicate tool %q (already registered)", name)
			continue
		}
		seen[name] = true
		builtin = append(builtin, t)
		added++
	}
	if added > 0 {
		log.Printf("loaded %d WASM plugin tool(s)", added)
	}
	return builtin
}

// ── Built-in Tools ──

type calculatorTool struct{}

func (t *calculatorTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "calculator",
		Description: "Evaluate a math expression like '15+27' or '100/3'.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"expression":{"type":"string","description":"The math expression to evaluate"}},"required":["expression"]}`),
	}
}

func (t *calculatorTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct{ Expression string }
	json.Unmarshal(args, &p)
	expr := strings.ReplaceAll(p.Expression, " ", "")
	var a, b int
	var op rune
	fmt.Sscanf(expr, "%d%c%d", &a, &op, &b)
	switch op {
	case '+': return fmt.Sprintf("%d", a+b), nil
	case '-': return fmt.Sprintf("%d", a-b), nil
	case '*': return fmt.Sprintf("%d", a*b), nil
	case '/':
		if b == 0 { return "", fmt.Errorf("division by zero") }
		return fmt.Sprintf("%d", a/b), nil
	}
	return "", fmt.Errorf("unsupported operator: %c", op)
}

type echoTool struct{}

func (t *echoTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "echo",
		Description: "Echoes the input message back.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"message":{"type":"string","description":"The message to echo"}},"required":["message"]}`),
	}
}

func (t *echoTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct{ Message string }
	json.Unmarshal(args, &p)
	return fmt.Sprintf("you said: %s", p.Message), nil
}


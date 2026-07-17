package server

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	_ "modernc.org/sqlite"

	openagent "github.com/yusheng-g/openagent-go"
	openacp "github.com/yusheng-g/openagent-go/acp"
	"github.com/yusheng-g/openagent-go/memory/sqlite"
	"github.com/yusheng-g/openagent-go/model/openai"
	"github.com/yusheng-g/openagent-go/rest"
	sessionsqlite "github.com/yusheng-g/openagent-go/rest/sessionstore/sqlite"
	"github.com/yusheng-g/openagent-go/sandbox/native"
	opentool "github.com/yusheng-g/openagent-go/tool"

	"github.com/yusheng-g/openagent-go/cmd/cli/config"
	"github.com/yusheng-g/openagent-go/cmd/cli/prompt"
	skillfs "github.com/yusheng-g/openagent-go/skill/fs"
)

type Options struct {
	Config *config.Config
	ACP    bool
}

type agentContext struct {
	Agent        *openagent.Agent
	ModelInfos   []modelReg
	Mem          openagent.Memory
	SessionStore rest.SessionStore
	Cleanup      func()
}

func buildAgentContext(cfg *config.Config) (*agentContext, error) {
	cfgDir, _ := config.Path()
	mem, sessionStore, memCleanup, err := buildMemory(filepath.Join(filepath.Dir(cfgDir), "data", "memory.db"))
	if err != nil {
		return nil, err
	}
	if memCleanup != nil {
		defer memCleanup()
	}

	models, modelInfos := buildModels(cfg.Provider)

	primaryModel := firstModel(models)
	if primaryModel == nil {
		log.Println("WARNING: no models configured — server will start but chat will fail")
	}

	var modelID string
	var cw int
	if primaryModel != nil {
		modelID = firstModelID(modelInfos)
		cw = primaryModel.ContextWindow()
	}
	if modelID == "" {
		modelID = "gpt-4o"
	}
	if cw == 0 {
		cw = 128_000
	}

	workDir, _ := os.Getwd()

	homeDir, _ := os.UserHomeDir()
	skillRoot := filepath.Join(homeDir, ".agents", "skills")

	sandbox, err := native.New(workDir)
	var agentTools []openagent.Tool
	if err == nil {
		sandbox.AddMount(skillRoot, "/skills")
		agentTools = buildTools(sandbox, workDir, []string{"shell", "read", "write", "ls", "grep"})
	} else {
		log.Printf("WARNING: sandbox unavailable: %v", err)
	}

	promptBuilder := prompt.DefaultServer(modelID, cw).Build()

	skillLoader := skillfs.New(skillRoot)

	agent := openagent.NewAgent("assistant",
		openagent.WithModel(primaryModel),
		openagent.WithMemory(mem),
		openagent.WithPrompt(promptBuilder),
		openagent.WithSkillLoader(skillLoader),
		openagent.WithTools(agentTools...),
		openagent.WithMaxTurns(100),
	)

	return &agentContext{
		Agent:        agent,
		ModelInfos:   modelInfos,
		Mem:          mem,
		SessionStore: sessionStore,
		Cleanup:      memCleanup,
	}, nil
}

func Run(ctx context.Context, opts Options) error {
	ac, err := buildAgentContext(opts.Config)
	if err != nil {
		return err
	}
	if ac.Cleanup != nil {
		defer ac.Cleanup()
	}

	if opts.ACP {
		return runACP(ac)
	}
	return runREST(ctx, opts.Config, ac.Agent, ac.ModelInfos, ac.Mem, ac.SessionStore)
}

func buildModels(providers map[string]config.ProviderConfig) ([]openagent.Model, []modelReg) {
	var models []openagent.Model
	var infos []modelReg
	for pid, p := range providers {
		for _, mid := range p.Models {
			apiKey := p.APIKey
			if apiKey == "" {
				apiKey = os.Getenv(strings.ToUpper(pid) + "_API_KEY")
			}
			m := openai.New(apiKey, mid, p.BaseURL)
			models = append(models, m)
			infos = append(infos, modelReg{ID: mid, Provider: pid, Model: m})
		}
	}
	return models, infos
}

type modelReg struct {
	ID, Provider string
	Model        openagent.Model
}

func firstModel(models []openagent.Model) openagent.Model {
	for _, m := range models {
		if m != nil {
			return m
		}
	}
	return nil
}

func firstModelID(infos []modelReg) string {
	for _, mi := range infos {
		if mi.ID != "" {
			if mi.Provider != "" {
				return mi.Provider + "/" + mi.ID
			}
			return mi.ID
		}
	}
	return "gpt-4o"
}

func buildMemory(path string) (openagent.Memory, rest.SessionStore, func(), error) {
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	mem, err := sqlite.New(path)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("memory: %w", err)
	}
	store, err := sessionsqlite.New(mem.DB())
	if err != nil {
		mem.Close()
		return nil, nil, nil, fmt.Errorf("session store: %w", err)
	}
	return mem, store, func() { store.Close(); mem.Close() }, nil
}

func buildTools(sandbox *native.Sandbox, workDir string, toolList []string) []openagent.Tool {
	enabled := make(map[string]bool)
	for _, name := range toolList {
		enabled[name] = true
	}
	var tools []openagent.Tool
	if enabled["shell"] {
		tools = append(tools, opentool.NewShell(sandbox, workDir))
	}
	if enabled["read"] {
		tools = append(tools, opentool.NewReadFile(workDir))
	}
	if enabled["write"] {
		tools = append(tools, opentool.NewWriteFile(workDir))
	}
	if enabled["ls"] {
		tools = append(tools, opentool.NewListDir(workDir))
	}
	if enabled["grep"] {
		tools = append(tools, opentool.NewGrep(workDir))
	}
	return tools
}

// ── REST ──

func runREST(ctx context.Context, cfg *config.Config, agent *openagent.Agent, modelInfos []modelReg, mem openagent.Memory, sessionStore rest.SessionStore) error {
	handler := rest.NewHandler(agent).
		WithSessionStore(sessionStore).
		WithCleanupDir(func(sessionID string) {
			dir := filepath.Join(opentool.ArtifactRoot(), sessionID)
			_ = os.RemoveAll(dir)
		})
	// Evict idle single-agent sessions after 24h of inactivity.
	handler.StartJanitor(ctx, 1*time.Hour, 24*time.Hour)
	for _, mi := range modelInfos {
		handler.RegisterModel(mi.ID, mi.Model, mi.Provider)
	}

	mux := http.NewServeMux()
	handler.Register(mux)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{Addr: addr, Handler: withMiddleware(mux)}

	go func() { <-ctx.Done(); log.Println("shutting down..."); srv.Shutdown(context.Background()) }()

	log.Printf("REST server listening on http://localhost%s", addr)
	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func withMiddleware(next http.Handler) http.Handler {
	return recoveryMiddleware(corsMiddleware(next))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("PANIC %s %s: %v", r.Method, r.URL.Path, rec)
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// ── ACP ──

// ACPSessionStore abstracts session persistence. Implementations can
// be in-memory, SQLite-backed, or any other backend.
type ACPSessionStore interface {
	CreateSession(ctx context.Context, cwd string) (string, error)
	ListSessions(ctx context.Context) ([]acpSessionInfo, error)
	GetSession(ctx context.Context, id string) (*acpSessionInfo, error)
	DeleteSession(ctx context.Context, id string) error
	AppendMessage(ctx context.Context, sessionID, role, content string) error
	LoadMessages(ctx context.Context, sessionID string) ([]acpMessage, error)
	SetTitle(ctx context.Context, sessionID, title string) error
	SetModel(ctx context.Context, sessionID, modelID string) error
	GetModel(ctx context.Context, sessionID string) (string, bool)
	EnsureSession(ctx context.Context, id, cwd string) error
	Close() error
}

type acpSessionInfo struct {
	ID        string
	Cwd       string
	Title     string
	UpdatedAt string
}

type acpMessage struct {
	Role    string
	Content string
}

// ── memory store ──

type memoryACPStore struct {
	mu       sync.Mutex
	sessions map[string]*acpSessionInfo
	messages map[string][]acpMessage // sessionID -> messages
	models   map[string]string       // sessionID -> modelID
	nextID   int
}

func newMemoryACPStore() *memoryACPStore {
	return &memoryACPStore{
		sessions: make(map[string]*acpSessionInfo),
		messages: make(map[string][]acpMessage),
		models:   make(map[string]string),
		nextID:   1,
	}
}

func (s *memoryACPStore) CreateSession(ctx context.Context, cwd string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("acp_%d", s.nextID)
	s.nextID++
	now := time.Now().UTC().Format(time.RFC3339)
	s.sessions[id] = &acpSessionInfo{ID: id, Cwd: cwd, UpdatedAt: now}
	return id, nil
}

func (s *memoryACPStore) GetSession(ctx context.Context, id string) (*acpSessionInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	si := s.sessions[id]
	if si == nil {
		return nil, fmt.Errorf("session %s not found", id)
	}
	return si, nil
}

func (s *memoryACPStore) ListSessions(ctx context.Context) ([]acpSessionInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []acpSessionInfo
	for _, si := range s.sessions {
		result = append(result, *si)
	}
	return result, nil
}

func (s *memoryACPStore) DeleteSession(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	delete(s.messages, id)
	return nil
}

func (s *memoryACPStore) AppendMessage(ctx context.Context, sessionID, role, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[sessionID] = append(s.messages[sessionID], acpMessage{Role: role, Content: content})
	return nil
}

func (s *memoryACPStore) LoadMessages(ctx context.Context, sessionID string) ([]acpMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.messages[sessionID], nil
}

func (s *memoryACPStore) SetTitle(ctx context.Context, sessionID, title string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if si := s.sessions[sessionID]; si != nil && si.Title == "" {
		si.Title = title
	}
	return nil
}

func (s *memoryACPStore) SetModel(ctx context.Context, sessionID, modelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.models[sessionID] = modelID
	return nil
}

func (s *memoryACPStore) GetModel(ctx context.Context, sessionID string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.models[sessionID]
	return m, ok
}

func (s *memoryACPStore) EnsureSession(ctx context.Context, id, cwd string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[id]; !ok {
		now := time.Now().UTC().Format(time.RFC3339)
		s.sessions[id] = &acpSessionInfo{ID: id, Cwd: cwd, UpdatedAt: now}
	}
	return nil
}

func (s *memoryACPStore) Close() error { return nil }

// ── sqlite store ──

type sqliteACPStore struct {
	db *sql.DB
}

func newSQLiteACPStore(db *sql.DB) (*sqliteACPStore, error) {
	if err := migrateACPStore(db); err != nil {
		return nil, err
	}
	return &sqliteACPStore{db: db}, nil
}

func migrateACPStore(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS acp_sessions (
			id         TEXT PRIMARY KEY,
			cwd        TEXT NOT NULL DEFAULT '',
			title      TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS acp_seq (
			name  TEXT PRIMARY KEY,
			value INTEGER NOT NULL DEFAULT 0
		);
		INSERT OR IGNORE INTO acp_seq (name, value) VALUES ('session_id', 0);
	`); err != nil {
		return err
	}
	// Best-effort migration for existing databases.
	db.Exec(`ALTER TABLE acp_sessions ADD COLUMN model_id TEXT NOT NULL DEFAULT ''`)
	return nil
}

func (s *sqliteACPStore) nextID() string {
	var id int64
	s.db.QueryRow(`UPDATE acp_seq SET value = value + 1 WHERE name = 'session_id' RETURNING value`).Scan(&id)
	return fmt.Sprintf("acp_%d", id)
}

func (s *sqliteACPStore) CreateSession(ctx context.Context, cwd string) (string, error) {
	id := s.nextID()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO acp_sessions (id, cwd, title, created_at, updated_at) VALUES (?, ?, '', ?, ?)`,
		id, cwd, now, now)
	return id, err
}

func (s *sqliteACPStore) GetSession(ctx context.Context, id string) (*acpSessionInfo, error) {
	si := &acpSessionInfo{ID: id}
	err := s.db.QueryRowContext(ctx,
		`SELECT cwd, title, updated_at FROM acp_sessions WHERE id = ?`, id).
		Scan(&si.Cwd, &si.Title, &si.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session %s not found", id)
	}
	return si, err
}

func (s *sqliteACPStore) ListSessions(ctx context.Context) ([]acpSessionInfo, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, cwd, title, updated_at FROM acp_sessions ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []acpSessionInfo
	for rows.Next() {
		var si acpSessionInfo
		if rows.Scan(&si.ID, &si.Cwd, &si.Title, &si.UpdatedAt) == nil {
			result = append(result, si)
		}
	}
	return result, nil
}

func (s *sqliteACPStore) DeleteSession(ctx context.Context, id string) error {
	s.db.ExecContext(ctx, `DELETE FROM messages WHERE session_id = ?`, id)
	_, err := s.db.ExecContext(ctx, `DELETE FROM acp_sessions WHERE id = ?`, id)
	return err
}

func (s *sqliteACPStore) AppendMessage(ctx context.Context, sessionID, role, content string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO messages (session_id, role, content) VALUES (?, ?, ?)`,
		sessionID, role, content)
	return err
}

func (s *sqliteACPStore) LoadMessages(ctx context.Context, sessionID string) ([]acpMessage, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT role, content FROM messages WHERE session_id = ? ORDER BY id ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []acpMessage
	for rows.Next() {
		var m acpMessage
		if rows.Scan(&m.Role, &m.Content) == nil {
			result = append(result, m)
		}
	}
	return result, nil
}

func (s *sqliteACPStore) SetTitle(ctx context.Context, sessionID, title string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE acp_sessions SET title = CASE WHEN title = '' THEN ?1 ELSE title END, updated_at = ?2 WHERE id = ?3`,
		title, time.Now().UTC().Format(time.RFC3339), sessionID)
	return err
}

func (s *sqliteACPStore) SetModel(ctx context.Context, sessionID, modelID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE acp_sessions SET model_id = ?, updated_at = ? WHERE id = ?`,
		modelID, time.Now().UTC().Format(time.RFC3339), sessionID)
	return err
}

func (s *sqliteACPStore) GetModel(ctx context.Context, sessionID string) (string, bool) {
	var m string
	err := s.db.QueryRowContext(ctx,
		`SELECT model_id FROM acp_sessions WHERE id = ?`, sessionID).Scan(&m)
	if err != nil || m == "" {
		return "", false
	}
	return m, true
}

func (s *sqliteACPStore) EnsureSession(ctx context.Context, id, cwd string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO acp_sessions (id, cwd, title, created_at, updated_at) VALUES (?, ?, '', ?, ?)`,
		id, cwd, now, now)
	return err
}

func (s *sqliteACPStore) Close() error { return nil }

// ── handler ──

func runACP(ac *agentContext) error {
	store := newACPStore(ac.Mem)
	srv := &acpHandler{
		agent:      ac.Agent,
		store:      store,
		modelInfos: ac.ModelInfos,
		defaultModelID: firstModelID(ac.ModelInfos),
	}
	server := openacp.NewServer("openagent-acp", "1.0.0", srv)
	log.Println("ACP server starting on stdio")
	return server.Run(context.Background())
}

func newACPStore(mem openagent.Memory) ACPSessionStore {
	if sm, ok := mem.(*sqlite.Memory); ok {
		s, err := newSQLiteACPStore(sm.DB())
		if err == nil {
			return s
		}
	}
	return newMemoryACPStore()
}

type acpHandler struct {
	agent          *openagent.Agent
	store          ACPSessionStore
	modelInfos     []modelReg
	defaultModelID string
}

func (h *acpHandler) findModel(modelID string) openagent.Model {
	for _, mi := range h.modelInfos {
		id := mi.ID
		if mi.Provider != "" {
			id = mi.Provider + "/" + mi.ID
		}
		if id == modelID {
			return mi.Model
		}
	}
	return nil
}

func (h *acpHandler) OnInitialize(ctx context.Context, req openacp.InitializeRequest) (*openacp.InitializeResponse, error) {
	return &openacp.InitializeResponse{
		ProtocolVersion: 1,
		AgentName:       "openagent-acp",
		AgentVersion:    "1.0.0",
		Capabilities: openacp.AgentCapabilities{
			LoadSession: true,
			SessionCapabilities: openacp.SessionCapabilities{
				List:   true,
				Delete: true,
				Resume: true,
				Close:  true,
			},
		},
	}, nil
}

func (h *acpHandler) OnNewSession(ctx context.Context, req openacp.NewSessionRequest) (*openacp.NewSessionResponse, error) {
	id, err := h.store.CreateSession(ctx, req.Cwd)
	if err != nil {
		return nil, fmt.Errorf("acp create session: %w", err)
	}
	h.store.SetModel(ctx, id, h.defaultModelID)
	return &openacp.NewSessionResponse{SessionID: id}, nil
}

func (h *acpHandler) OnLoadSession(ctx context.Context, req openacp.LoadSessionRequest, sender openacp.SessionEventSender) (*openacp.LoadSessionResponse, error) {
	si, err := h.store.GetSession(ctx, req.SessionID)
	if err != nil {
		h.store.EnsureSession(ctx, req.SessionID, req.Cwd)
		return &openacp.LoadSessionResponse{}, nil
	}
	msgs, err := h.store.LoadMessages(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("acp load messages: %w", err)
	}
	for _, m := range msgs {
		switch m.Role {
		case "user":
			sender.SendUserMessage(m.Content)
		case "assistant":
			sender.SendAgentMessage(m.Content)
		}
	}
	if si.Title != "" {
		sender.SendSessionInfo(si.Title, nil)
	}
	return &openacp.LoadSessionResponse{}, nil
}

func (h *acpHandler) OnListSessions(ctx context.Context, req openacp.ListSessionsRequest) (*openacp.ListSessionsResponse, error) {
	list, err := h.store.ListSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("acp list sessions: %w", err)
	}
	sessions := make([]openacp.SessionInfo, len(list))
	for i, s := range list {
		sessions[i] = openacp.SessionInfo{
			SessionID: s.ID,
			Cwd:       s.Cwd,
			Title:     s.Title,
			UpdatedAt: s.UpdatedAt,
		}
	}
	return &openacp.ListSessionsResponse{Sessions: sessions}, nil
}

func (h *acpHandler) OnPrompt(ctx context.Context, req openacp.PromptRequest, sender openacp.SessionEventSender) (*openacp.PromptResponse, error) {
	var input string
	for _, b := range req.Blocks {
		if b.Text != "" {
			input = b.Text
			break
		}
	}
	if input == "" {
		return nil, fmt.Errorf("no text in prompt")
	}

	h.store.AppendMessage(ctx, req.SessionID, "user", input)

	var responseText strings.Builder
	session := openagent.Session{ID: req.SessionID}
	if mID, ok := h.store.GetModel(ctx, req.SessionID); ok {
		if m := h.findModel(mID); m != nil {
			session.Model = m
		}
	}
	ch := h.agent.RunStream(ctx, session, openagent.UserMessage(input))
	for evt := range ch {
		switch evt.Type {
		case openagent.StreamTextDelta:
			responseText.WriteString(evt.Text)
			sender.SendAgentMessage(evt.Text)
		case openagent.StreamToolCall:
			if len(evt.Message.ToolCalls) > 0 {
				tc := evt.Message.ToolCalls[0]
				sender.SendToolCall(openacp.ToolCallEvent{ID: tc.ID, Title: tc.Function.Name, Status: "in_progress", RawInput: map[string]any{"args": tc.Function.Arguments}})
			}
		case openagent.StreamToolResult:
			sender.SendToolCall(openacp.ToolCallEvent{ID: evt.Message.ToolCallID, Status: "completed", RawOutput: map[string]any{"result": evt.Message.Content}})
		case openagent.StreamError:
			return nil, evt.Error
		case openagent.StreamAborted:
			return &openacp.PromptResponse{StopReason: "cancelled"}, nil
		}
	}

	finalText := responseText.String()
	h.store.AppendMessage(ctx, req.SessionID, "assistant", finalText)

	title := input
	if len(title) > 50 {
		title = title[:50]
	}
	h.store.SetTitle(ctx, req.SessionID, title)

	return &openacp.PromptResponse{StopReason: "end_turn"}, nil
}

func (h *acpHandler) OnCancel(ctx context.Context, sid string) error { return nil }

func (h *acpHandler) OnDeleteSession(ctx context.Context, sid string) error {
	return h.store.DeleteSession(ctx, sid)
}

func (h *acpHandler) OnCloseSession(ctx context.Context, sid string) error { return nil }

func (h *acpHandler) OnSetSessionConfigOption(ctx context.Context, sessionID string, opt openacp.SetConfigOption) error {
	modelID, ok := opt.Value.(string)
	if !ok {
		return nil
	}
	if m := h.findModel(modelID); m == nil {
		return nil
	}
	return h.store.SetModel(ctx, sessionID, modelID)
}

func (h *acpHandler) OnSetSessionMode(ctx context.Context, sessionID string, modeID string) error {
	return nil
}

// Compile-time check: acpHandler implements ConfigHandler.
var _ openacp.ConfigHandler = (*acpHandler)(nil)

// ── ACP over WebSocket ──

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func RunACPWS(ctx context.Context, cfg *config.Config, port int) error {
	ac, err := buildAgentContext(cfg)
	if err != nil {
		return err
	}
	if ac.Cleanup != nil {
		defer ac.Cleanup()
	}

	store := newACPStore(ac.Mem)

	mux := http.NewServeMux()
	mux.HandleFunc("/acp", func(w http.ResponseWriter, r *http.Request) {
		ws, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("acp-ws: upgrade: %v", err)
			return
		}
		defer ws.Close()

		handler := &acpHandler{agent: ac.Agent, store: store, modelInfos: ac.ModelInfos, defaultModelID: firstModelID(ac.ModelInfos)}
		br := &wsBridge{conn: ws}
		server := openacp.NewServer("openagent-acp", "1.0.0", handler)
		log.Printf("acp-ws: client connected")
		if err := server.RunTransport(ctx, br, br); err != nil {
			log.Printf("acp-ws: transport: %v", err)
		}
		log.Printf("acp-ws: client disconnected")
	})

	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() { <-ctx.Done(); srv.Shutdown(context.Background()) }()

	log.Printf("ACP WebSocket server listening on ws://localhost%s/acp", addr)
	err = srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

type wsBridge struct {
	conn *websocket.Conn
	mu   sync.Mutex
	buf  bytes.Buffer
}

func (w *wsBridge) Read(p []byte) (int, error) {
	if w.buf.Len() > 0 {
		return w.buf.Read(p)
	}
	_, msg, err := w.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	w.buf.Write(msg)
	w.buf.WriteByte('\n')
	return w.buf.Read(p)
}

func (w *wsBridge) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	data := bytes.TrimRight(p, "\n")
	if len(data) == 0 {
		return len(p), nil
	}
	if err := w.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return 0, err
	}
	return len(p), nil
}

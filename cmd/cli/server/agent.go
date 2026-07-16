package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	openacp "github.com/yusheng-g/openagent-go/acp"
	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/memory/sqlite"
	"github.com/yusheng-g/openagent-go/model/openai"
	"github.com/yusheng-g/openagent-go/rest"
	sessionsqlite "github.com/yusheng-g/openagent-go/rest/sessionstore/sqlite"
	"github.com/yusheng-g/openagent-go/sandbox/native"
	opentool "github.com/yusheng-g/openagent-go/tool"

	"github.com/yusheng-g/openagent-go/cmd/cli/config"
)

type Options struct {
	Config *config.Config
	ACP    bool
}

func Run(ctx context.Context, opts Options) error {
	cfgDir, _ := config.Path()
	mem, sessionStore, memCleanup, err := buildMemory(filepath.Join(filepath.Dir(cfgDir), "data", "memory.db"))
	if err != nil {
		return err
	}
	if memCleanup != nil {
		defer memCleanup()
	}

	// Build models from provider config.
	models, modelInfos := buildModels(opts.Config.Provider)

	primaryModel := firstModel(models)
	if primaryModel == nil {
		log.Println("WARNING: no models configured — server will start but chat will fail")
	}

	// Sandbox + tools. Workspace is process CWD — per-session isolation
	// is handled by the sandbox (bwrap) and the agent's own session ID.
	workDir, _ := os.Getwd()
	sandbox, err := native.New(workDir)
	var agentTools []openagent.Tool
	if err == nil {
		agentTools = buildTools(sandbox, workDir, []string{"shell", "read", "write", "ls", "grep"})
	} else {
		log.Printf("WARNING: sandbox unavailable: %v", err)
	}

	agent := openagent.NewAgent("assistant",
		openagent.WithModel(primaryModel),
		openagent.WithMemory(mem),
		openagent.WithInstructions("You are a helpful assistant."),
		openagent.WithTools(agentTools...),
		openagent.WithMaxTurns(10),
	)

	if opts.ACP {
		return runACP(agent)
	}
	return runREST(ctx, opts.Config, agent, modelInfos, mem, sessionStore)
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

type modelReg struct{ ID, Provider string; Model openagent.Model }

func firstModel(models []openagent.Model) openagent.Model {
	for _, m := range models {
		if m != nil { return m }
	}
	return nil
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
	for _, name := range toolList { enabled[name] = true }
	var tools []openagent.Tool
	if enabled["shell"] { tools = append(tools, opentool.NewShell(sandbox, workDir)) }
	if enabled["read"]  { tools = append(tools, opentool.NewReadFile(workDir)) }
	if enabled["write"] { tools = append(tools, opentool.NewWriteFile(workDir)) }
	if enabled["ls"]    { tools = append(tools, opentool.NewListDir(workDir)) }
	if enabled["grep"]  { tools = append(tools, opentool.NewGrep(workDir)) }
	return tools
}

// ── REST ──

func runREST(ctx context.Context, cfg *config.Config, agent *openagent.Agent, modelInfos []modelReg, mem openagent.Memory, sessionStore rest.SessionStore) error {
	handler := rest.NewHandler(agent).WithSessionStore(sessionStore)
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
	if err == http.ErrServerClosed { return nil }
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
		if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
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

func runACP(agent *openagent.Agent) error {
	srv := &acpHandler{agent: agent}
	server := openacp.NewServer("openagent-acp", "1.0.0", srv)
	log.Println("ACP server starting on stdio")
	return server.Run(context.Background())
}

type acpHandler struct{ agent *openagent.Agent }

func (h *acpHandler) OnInitialize(ctx context.Context, req openacp.InitializeRequest) (*openacp.InitializeResponse, error) {
	return &openacp.InitializeResponse{ProtocolVersion: 1, AgentName: "openagent-acp", AgentVersion: "1.0.0"}, nil
}
func (h *acpHandler) OnNewSession(ctx context.Context, req openacp.NewSessionRequest) (*openacp.NewSessionResponse, error) {
	return &openacp.NewSessionResponse{SessionID: "acp_1"}, nil
}
func (h *acpHandler) OnLoadSession(ctx context.Context, req openacp.LoadSessionRequest) (*openacp.LoadSessionResponse, error) {
	return &openacp.LoadSessionResponse{}, nil
}
func (h *acpHandler) OnListSessions(ctx context.Context, req openacp.ListSessionsRequest) (*openacp.ListSessionsResponse, error) {
	return &openacp.ListSessionsResponse{}, nil
}
func (h *acpHandler) OnPrompt(ctx context.Context, req openacp.PromptRequest, sender openacp.SessionEventSender) (*openacp.PromptResponse, error) {
	var input string
	for _, b := range req.Blocks { if b.Text != "" { input = b.Text; break } }
	if input == "" { return nil, fmt.Errorf("no text in prompt") }
	session := openagent.Session{ID: req.SessionID}
	ch := h.agent.RunStream(ctx, session, openagent.UserMessage(input))
	for evt := range ch {
		switch evt.Type {
		case openagent.StreamTextDelta:
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
	return &openacp.PromptResponse{StopReason: "end_turn"}, nil
}
func (h *acpHandler) OnCancel(ctx context.Context, sid string) error       { return nil }
func (h *acpHandler) OnDeleteSession(ctx context.Context, sid string) error { return nil }
func (h *acpHandler) OnCloseSession(ctx context.Context, sid string) error  { return nil }

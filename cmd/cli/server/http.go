package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/rest"
	"github.com/yusheng-g/openagent-go/sandbox/native"
	"github.com/yusheng-g/openagent-go/summarizer"
	opentool "github.com/yusheng-g/openagent-go/tool"

	wasm "github.com/yusheng-g/openagent-go/plugin/agent/wasm"
	"github.com/yusheng-g/openagent-go/plugin/wasmhost"

	"github.com/yusheng-g/openagent-go/cmd/cli/config"
)

// ── REST server ──

// RunREST starts the REST API server (HTTP + SSE).
func RunREST(ctx context.Context, cfg *config.Config, caps Capabilities) error {
	models, modelInfos := buildModels(cfg.Provider)
	m := firstModel(models)

	workDir, _ := os.Getwd()
	sb, err := native.NewWithPolicy(workDir, sandboxPolicy(cfg.Sandbox))
	var tools []openagent.Tool
	if err == nil {
		tools = buildTools(sb, workDir, []string{"shell", "read", "write", "edit", "ls", "grep"})
	} else {
		log.Printf("WARNING: sandbox unavailable, tools disabled: %v", err)
	}

	// MCP tools from config. Gated by --mcp (default on, --mcp=off disables).
	if caps.OnMCP() {
		mcpTools, mcpCleanup := connectMcpFromConfig(ctx, cfg.McpServers)
		tools = append(tools, mcpTools...)
		defer mcpCleanup()
	}

	profilesDir := resolveProfilesDir(cfg.Profiles)
	mem, store, cleanup, err := buildMemory(profilesDir)
	if err != nil {
		return err
	}
	defer cleanup()

	opts := []openagent.AgentOption{
		openagent.WithModel(m),
		openagent.WithSystemPrompts(resolveProfiles(cfg.Profiles)...),
		openagent.WithMaxTurns(100),
	}
	if caps.OnMemory() {
		opts = append(opts, openagent.WithMemory(mem))
	}
	if caps.OnTools() {
		opts = append(opts, openagent.WithTools(tools...))
	}
	opts = buildOpts(opts, caps, m)
	agent := openagent.NewAgent("openagent", opts...)

	if caps.OnSummarizer() && m != nil && caps.OnMemory() {
		mem.WithSummarizer(summarizer.New(m).WithMaxTokens(agent.MaxCompressedTokens))
	}

	handler := rest.NewHandler(agent).
		WithSessionStore(store).
		WithCleanupDir(func(sessionID string) {
			dir := filepath.Join(opentool.ArtifactRoot(), sessionID)
			_ = os.RemoveAll(dir)
		}).
		WithApproverEnabled(caps.OnApprover()).
		WithProcessDir(filepath.Join(os.TempDir(), "openagent"))
	handler.StartJanitor(ctx, 1*time.Hour, 24*time.Hour)
	for _, mi := range modelInfos {
		handler.RegisterModel(mi.ID, mi.Model, mi.Provider, mi.APIKey, mi.BaseURL)
	}

	// Plugin manager — loads agent:tools, agent:observers, agent:sessions.
	pluginDir := filepath.Join(profilesDir, "plugins")
	mgr := wasm.NewManager(pluginDir).WithHostAPI(&wasmhost.HostAPI{Logger: &logAdapter{}})
	if err := mgr.Discover(ctx); err != nil {
		log.Printf("WARNING: plugin discover: %v", err)
	} else {
		handler.WithPluginManager(mgr)
	}

	// Start IM channels in the background.
	if err := RunChannels(ctx, agent, cfg.Channels); err != nil {
		log.Printf("channel: %v", err)
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
	err = srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// ── Middleware ──

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

// logAdapter implements wasmhost.Logger by forwarding to the standard log package.
type logAdapter struct{}

func (l *logAdapter) Info(msg string)  { log.Printf("[plugin:info] %s", msg) }
func (l *logAdapter) Warn(msg string)  { log.Printf("[plugin:warn] %s", msg) }
func (l *logAdapter) Error(msg string) { log.Printf("[plugin:error] %s", msg) }

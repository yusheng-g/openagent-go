package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/acp"
	"github.com/yusheng-g/openagent-go/agent"
	"github.com/yusheng-g/openagent-go/guard/llm"
	"github.com/yusheng-g/openagent-go/keyring"
	"github.com/yusheng-g/openagent-go/rest"
	"github.com/yusheng-g/openagent-go/sandbox/native"
	"github.com/yusheng-g/openagent-go/summarizer"
	opentool "github.com/yusheng-g/openagent-go/tool"
	"github.com/yusheng-g/openagent-go/version"

	wasm "github.com/yusheng-g/openagent-go/plugin/agent/wasm"
	cliwasm "github.com/yusheng-g/openagent-go/plugin/cli/wasm"
	"github.com/yusheng-g/openagent-go/plugin/wasmhost"
	"github.com/yusheng-g/openagent-go/scheduler"

	"github.com/yusheng-g/openagent-go/cmd/cli/config"
	clirest "github.com/yusheng-g/openagent-go/cmd/cli/rest"
	ctxpkg "github.com/yusheng-g/openagent-go/context"
)

// ── REST server ──

// RunREST starts the REST API server (HTTP + SSE).
func RunREST(ctx context.Context, cfg *config.Config) error {
	caps := cfg.Capabilities
	_, modelInfos := buildModels(cfg.Provider)
	m := resolveModel(cfg.Model, modelInfos)

	workDir, _ := os.Getwd()
	sb, err := native.NewWithPolicy(workDir, sandboxPolicy(cfg.Sandbox))
	restToolList := []string{"shell", "read", "write", "edit", "ls", "grep", "websearch", "webfetch", "settings"}
	if caps.OnBrowser() {
		restToolList = append(restToolList, "browser")
	}
	if caps.OnOffice() {
		restToolList = append(restToolList, "office")
	}
	var tools []openagent.Tool
	if err == nil {
		tools = buildTools(sb, workDir, restToolList)
	} else {
		slog.Warn("sandbox unavailable, tools disabled", "error", err)
	}

	// MCP tools from config. Gated by --mcp (default on, --mcp=off disables).
	if caps.OnMCP() {
		mcpTools, mcpCleanup := connectMcpFromConfig(ctx, cfg.McpServers)
		tools = append(tools, mcpTools...)
		defer mcpCleanup()
	}

	ms, knowledge, store, cleanup, err := buildMemory(cfg.Embedding, caps.OnEmbedder())
	if err != nil {
		return err
	}
	defer cleanup()

	opts := []agent.Option{
		agent.WithModel(m),
		agent.WithSystemPrompts(resolveProfiles("")...),
		agent.WithMaxTurns(500),
	}
	// REST is single-process with no model hot-reload, so a static guard
	// is correct — no dynamic lookup needed.
	if caps.OnGuard() && m != nil {
		g := llm.New(m)
		opts = append(opts, agent.WithInputGuard(g))
		opts = append(opts, agent.WithOutputGuard(g.Output()))
	}
	opts, skillProvider := buildOpts(opts, caps)
	agentCfg := agent.New(version.Name, opts...)

	holder, _, telemetryShutdown, err := setupTelemetry(ctx, *cfg)
	if err != nil {
		return fmt.Errorf("telemetry init: %w", err)
	}
	defer telemetryShutdown()

	deps := buildRuntimeDeps(caps, cfg.Sensitive, holder)
	deps.Tools = tools
	deps.SkillProvider = skillProvider
	if caps.OnMemory() {
		deps.SessionStore = ms
		deps.Compressor = ms
		deps.MemoryProvider = knowledge
	}

	if caps.OnSummarizer() && m != nil && caps.OnMemory() {
		sumz := summarizer.New(m).WithMaxTokens(agentCfg.MaxCompressedTokens)
		ms.WithSummarizer(sumz)
		deps.Summarizer = sumz
	}

	if err := applyContextProviders(cfg, &deps); err != nil {
		return err
	}
	// The extractor captures the MemoryProvider it writes to — build it
	// AFTER applyContextProviders so the effective provider is used.
	// Building it earlier would fork writes to the local sqlite store
	// while Recall reads the OpenViking index (silent knowledge loss).
	if caps.OnMemory() && m != nil && deps.MemoryProvider != nil {
		deps.Extractor = ctxpkg.NewAsyncExtractor(ctxpkg.NewLLMExtractor(func() openagent.Model { return m }, deps.MemoryProvider))
	}
	handler := rest.NewHandler(agentCfg, deps).
		WithSessionStore(store).
		WithCleanupDir(func(sessionID string) {
			// Matches the artifact hook's layout
			// (<ArtifactRoot()>/sess-<sessionID>/) and the REST
			// handler's process dir layout (sess-<id>), so a single
			// call cleans both artifacts and process output.
			// Sanitized like the artifact writer (result.go): a hostile
			// session id must not escape the artifact root tree.
			dir := filepath.Join(opentool.ArtifactRoot(), "sess-"+openagent.SanitizeName(sessionID))
			_ = os.RemoveAll(dir)
		}).
		WithApproverEnabled(caps.OnApprover()).
		WithProcessDir(opentool.ArtifactRoot())
	handler.StartJanitor(ctx, 1*time.Hour, 24*time.Hour)
	for _, mi := range modelInfos {
		handler.RegisterModel(mi.ID, mi.Model, mi.Provider, mi.APIKey, mi.BaseURL)
	}

	// Plugin manager — loads agent:tools and agent:observers plugins.
	// Scheduled jobs declared by plugins fire on a process-local scheduler
	// that lives for the server's lifetime.
	pluginDir := resolvePluginsDir()
	sch := scheduler.New()
	go sch.Run(ctx)
	mgr := wasm.NewManager(pluginDir).
		WithHostAPI(wasmhost.NewHostAPI(keyring.NewKeyring())).
		WithScheduler(sch)
	if err := mgr.Discover(ctx); err != nil {
		slog.Warn("plugin discover failed", "error", err)
	} else {
		handler.WithPluginManager(mgr)
	}

	// Start IM channels (immediate-connect; fail-fast for an explicitly
	// flagged channel). Both managers are ALWAYS created (even when no
	// channel is configured) so the CLI-level REST API can query status
	// and trigger connect/registration on demand.
	feishuMgr, wechatMgr, wecomMgr, err := RunChannels(ChannelEnv{
		Ctx:         ctx,
		Cfg:         agentCfg,
		Deps:        deps,
		DefaultMode: cfg.DefaultMode,
		WorkDir:     workDir,
		MetaStore:   store,
	}, cfg.Channels)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	handler.Register(mux)
	// CLI-level API (channels, settings) — deployment/configuration
	// endpoints owned by cmd/cli, not the agent-level rest package.
	clirest.Register(mux, feishuMgr, wechatMgr, wecomMgr)
	// cli:http plugins — declared routes served under /api/plugins/<name>/.
	// Routes are registered at plugin load time (process-level table); the
	// dispatcher is a no-op when no cli:http plugin is loaded.
	mux.Handle("/api/plugins/", cliwasm.HTTPHandler())
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	// ReadHeaderTimeout guards the slow-header DoS (a client that never
	// finishes sending headers holds a connection); body reads are
	// bounded per-handler (see the cli:http dispatcher). SSE endpoints
	// are unaffected — this only covers the request-header phase.
	srv := &http.Server{
		Addr:              addr,
		Handler:           withMiddleware(mux),
		ReadHeaderTimeout: 60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		slog.Info("shutting down")
		// Bounded shutdown: SSE long-pollers can block Close() forever.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	// Register the settings watcher for the reload action. REST mode has
	// no AgentServer (srv=nil) so model registry updates are skipped.
	// fsnotify auto-reload is disabled — see acp.go for rationale.
	activeWatcher.Store(&settingsWatcher{
		cfgPath:  config.Path(),
		prev:     cfg,
		holder:   holder,
		shutdown: telemetryShutdown,
		srv:      nil,
	})
	// Set the shared reload function for the settings tool's reload action.
	// REST has no AgentServer so slash commands are unavailable, but the
	// settings tool (if wired via capabilities) still uses this.
	reloadFn := func(ctx context.Context) acp.ReloadResult {
		sw := activeWatcher.Load()
		if sw == nil {
			return acp.ReloadResult{ParseError: "no settings watcher configured"}
		}
		r := sw.reload(ctx)
		return acp.ReloadResult{
			Applied:    r.Applied,
			Violations: r.Violations,
			ParseError: r.ParseError,
		}
	}
	settingsReloadFn.Store(&reloadFn)

	slog.Info("REST server listening", "addr", addr)
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
				slog.Error("panic recovered", "method", r.Method, "path", r.URL.Path, "error", rec)
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

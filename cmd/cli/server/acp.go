package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/acp"
	openacpsdk "github.com/yusheng-g/openagent-go/acp/sdk"
	"github.com/yusheng-g/openagent-go/agent"
	"github.com/yusheng-g/openagent-go/keyring"
	"github.com/yusheng-g/openagent-go/sandbox/native"
	"github.com/yusheng-g/openagent-go/summarizer"
	"github.com/yusheng-g/openagent-go/version"

	wasm "github.com/yusheng-g/openagent-go/plugin/agent/wasm"
	"github.com/yusheng-g/openagent-go/plugin/wasmhost"
	"github.com/yusheng-g/openagent-go/scheduler"

	"github.com/yusheng-g/openagent-go/cmd/cli/config"
	ctxpkg "github.com/yusheng-g/openagent-go/context"
	"github.com/yusheng-g/openagent-go/guard/llm"
)

// RunACP starts the agent in ACP mode over stdio.
//
// Lifecycle:
//  1. Open memory + session store (SQLite).
//  2. Build models from config.
//  3. Create sandbox + standard tools.
//  4. Wire summarizer for long-conversation compression.
//  5. Construct the agent.
//  6. Wrap in AgentServer, launch ACP protocol mux on stdin/stdout.
func RunACP(ctx context.Context, cfg *config.Config) error {
	server, cleanup, err := BuildACPServer(ctx, cfg)
	if err != nil {
		return err
	}
	defer cleanup()
	slog.Info("ACP server starting on stdio")
	return server.Run(ctx)
}

// BuildACPServer constructs the ACP server (memory, models, tools, agent,
// channels) and returns it with a cleanup func. Used by both RunACP (stdio)
// and RunACPTransport (in-process pipe for the TUI).
func BuildACPServer(ctx context.Context, cfg *config.Config) (*openacpsdk.Server, func(), error) {
	caps := cfg.Capabilities
	ms, knowledge, sessionStore, cleanup, err := buildMemory(cfg.Embedding, caps.OnEmbedder())
	if err != nil {
		return nil, nil, err
	}

	_, modelInfos := buildModels(cfg.Provider)
	if len(modelInfos) == 0 {
		slog.Warn("no models configured, ACP server will start but prompt turns will fail")
	}

	modelMap := make(map[string]openagent.Model, len(modelInfos))
	for _, mi := range modelInfos {
		modelMap[mi.Key()] = mi.Model
	}

	// Summarizer and Memory are enabled by default; allow --summarizer=off
	// and --memory=off to disable them.

	// srv is declared here so dynamicModel (below) can close over the
	// variable — the closure reads srv at CALL time, not construction time,
	// so it sees the assigned value once NewAgentServer returns. This lets
	// the summarizer/extractor/guard be constructed with the real resolver
	// in one pass, with no placeholder + later SetModelFn override.
	var srv *acp.AgentServer
	dynamicModel := func() openagent.Model {
		if srv == nil {
			return nil
		}
		if id := srv.GetDefaultModelID(); id != "" {
			if m, ok := srv.LookupModel(id); ok {
				return m
			}
		}
		return nil
	}

	// Tools and sandbox are created once per session (buildRuntimeForSession)
	// scoped to the session's cwd. Agent configuration is pure (model,
	// prompts, limits, guards, skills); runtime capabilities live in
	// kernel.Deps.
	opts := []agent.Option{
		agent.WithSystemPrompts(resolveProfiles("")...),
		agent.WithMaxTurns(500),
	}
	// The guard resolves the judge model via dynamicModel (server default),
	// so it stays in sync with runtime model switches / api_key changes.
	// Guard is a template-level shared component (not per-session); it reads
	// the server default, not each session's model. buildOpts only handles
	// skills + sub-agents.
	if caps.OnGuard() {
		g := llm.NewWithLookup(dynamicModel)
		opts = append(opts, agent.WithInputGuard(g))
		opts = append(opts, agent.WithOutputGuard(g.Output()))
	}
	opts, skillProvider := buildOpts(opts, caps)
	agentCfg := agent.New(version.Name, opts...)

	holder, _, telemetryShutdown, err := setupTelemetry(ctx, *cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("telemetry init: %w", err)
	}
	// NOTE: telemetryShutdown is NOT deferred here — BuildACPServer returns
	// immediately and its defers would fire before server.Run starts, shutting
	// down the TracerProvider prematurely. It is wired into the returned
	// cleanup func so the caller defers it at the right scope.

	deps := buildRuntimeDeps(caps, cfg.Sensitive, holder)
	deps.SkillProvider = skillProvider
	// Pass nil Mem when --memory=off so the AgentServer skips history
	// replay and memory cleanup (all s.Mem uses are nil-guarded). The
	// sessionStore (session metadata) is separate and unaffected.
	if caps.OnMemory() {
		deps.SessionStore = ms
		deps.Compressor = ms
		deps.MemoryProvider = knowledge
	}

	// Summarizer resolves the model via dynamicModel at call time. Summarize
	// is nil-safe (returns an error → prepare.go degrades to tail-trim), so
	// construction does not depend on a model being configured yet.
	var sumz *summarizer.Compressor
	if caps.OnMemory() && caps.OnSummarizer() {
		sumz = summarizer.NewWithLookup(dynamicModel).WithMaxTokens(agentCfg.MaxCompressedTokens)
		ms.WithSummarizer(sumz)
		deps.Summarizer = sumz
	}

	// Plugin manager — loads agent:tools and agent:observers plugins.
	// Discover before constructing the server so a plugin observer can be
	// merged into deps.Observer before the AgentServer snapshots it.
	var pluginMgr *wasm.Manager
	pluginDir := resolvePluginsDir()
	sch := scheduler.New()
	go sch.Run(ctx)
	mgr := wasm.NewManager(pluginDir).
		WithHostAPI(wasmhost.NewHostAPI(keyring.NewKeyring())).
		WithScheduler(sch)
	if err := mgr.Discover(ctx); err != nil {
		slog.Warn("plugin discover failed", "error", err)
	} else {
		pluginMgr = mgr
		if obs := mgr.Observer(); obs != nil {
			if deps.Observer != nil {
				deps.Observer = openagent.MultiObserver(deps.Observer, obs)
			} else {
				deps.Observer = obs
			}
			slog.Info("plugin observer wired", "source", "wasm")
		}
	}

	if err := applyContextProviders(cfg, &deps); err != nil {
		return nil, nil, err
	}
	// The extractor captures the MemoryProvider it writes to — build it
	// AFTER applyContextProviders so the effective provider is used.
	// Building it earlier would fork writes to the local sqlite store
	// while Recall reads the OpenViking index (silent knowledge loss).
	// NewLLMExtractor is nil-safe (nil model → no-op), so construction does
	// not depend on a model being configured yet.
	var extractor *ctxpkg.AsyncExtractor
	if caps.OnMemory() && deps.MemoryProvider != nil {
		extractor = ctxpkg.NewAsyncExtractor(ctxpkg.NewLLMExtractor(dynamicModel, deps.MemoryProvider))
		deps.Extractor = extractor
	}
	srv = acp.NewAgentServer(agentCfg, deps, sessionStore, modelMap)
	srv.AgentName = version.Name
	srv.AgentVersion = version.Version
	srv.MCPEnabled = caps.OnMCP()
	srv.DefaultMode = cfg.DefaultMode
	srv.PluginMgr = pluginMgr
	srv.Summarizer = sumz
	srv.Extractor = extractor
	srv.ProfileResolver = func(cwd string) []string {
		return resolveProfiles(cwd)
	}
	// Wire settings-declared MCP servers so the global config is honored in
	// ACP mode (not just client-advertised ones). mergeMcpServers combines
	// these with the client's per-session list at connect time.
	srv.SetSettingsMcpServers(convertMcpServers(cfg.McpServers))

	// Register model configs for runtime_set_model_config.
	for _, mi := range modelInfos {
		srv.RegisterModel(mi.Key(), mi.Provider, mi.ID, mi.APIKey, mi.BaseURL, acp.ModelPricing{
			MaxInputTokens:           mi.MaxInputTokens,
			MaxOutputTokens:          mi.MaxOutputTokens,
			InputCostPerMillion:      mi.InputCostPerMillion,
			InputCacheCostPerMillion: mi.InputCacheCostPerMillion,
			OutputCostPerMillion:     mi.OutputCostPerMillion,
		})
	}

	// settings "model" ("<provider>/<modelID>") wins as the default;
	// fall back to the first registered model.
	if cfg.Model != "" {
		if !srv.SetDefaultModelID(cfg.Model) {
			slog.Warn("settings model not in provider list, using first registered", "model", cfg.Model)
		}
	}

	policy := sandboxPolicy(cfg.Sandbox)
	baseToolList := []string{"shell", "read", "write", "ls", "grep", "websearch", "webfetch", "settings"}
	if caps.OnBrowser() {
		baseToolList = append(baseToolList, "browser")
	}
	if caps.OnOffice() {
		baseToolList = append(baseToolList, "office")
	}
	srv.ToolFactory = func(cwd string) []openagent.Tool {
		sb, err := native.NewWithPolicy(cwd, policy)
		if err != nil {
			slog.Warn("tool factory: sandbox creation failed; execution tools disabled", "cwd", cwd, "error", err)
			return nil
		}
		return buildTools(sb, cwd, baseToolList)
	}
	server := openacpsdk.NewServer(version.Name, version.Version, srv)
	server.SetLogger(slog.Default())

	// Register the settings watcher so the settings tool's reload action
	// can apply changes on demand. fsnotify auto-reload is disabled —
	// settings changes are applied only via explicit reload (agent calls
	// set → reload, or operator runs `openagent settings reload`).
	// TODO: re-enable fsnotify with a non-broadcast notification mechanism
	// (e.g. DynamicContext injection on next user turn) instead of
	// broadcasting idle turns to all sessions.
	activeWatcher.Store(&settingsWatcher{
		cfgPath:  config.Path(),
		prev:     cfg,
		holder:   holder,
		shutdown: telemetryShutdown,
		srv:      srv,
	})
	// Inject settings callbacks so /settings slash commands and the settings
	// tool can operate on settings.json without acp importing cmd/cli/config.
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
	srv.SetSettingsCallbacks(acp.SettingsCallbacks{
		List: config.ListSettings,
		Get:  config.GetSetting,
		Set:  config.SetSetting,
		Validate: func() (warnings, violations []string, err error) {
			report, err := config.ValidateSettings()
			if err != nil {
				return nil, nil, err
			}
			return report.Warnings, report.EnumViolations, nil
		},
		Reload: reloadFn,
	})

	// Channel agent: clone the template and inject a default Model + Tools
	// so the IM bot can run standalone (the ACP path resolves the model per
	// session, but channels call kernel.New(...).RunStream directly).
	channelCfg := agentCfg.Clone()
	for _, mi := range modelInfos {
		if mi.Model != nil {
			channelCfg.Model = mi.Model
			break
		}
	}
	channelDeps := deps
	cwd, _ := os.Getwd()
	if sb, err := native.NewWithPolicy(cwd, policy); err == nil {
		channelDeps.Tools = buildTools(sb, cwd, baseToolList)
	}

	if _, _, _, err := RunChannels(ChannelEnv{
		Ctx:         ctx,
		Cfg:         channelCfg,
		Deps:        channelDeps,
		DefaultMode: cfg.DefaultMode,
		WorkDir:     cwd,
		MetaStore:   sessionStore,
	}, cfg.Channels); err != nil {
		slog.Warn("channel error", "error", err)
	}

	// Wrap cleanup to also shutdown telemetry (TracerProvider flush).
	// This runs when the caller defers cleanup() — after server.Run exits.
	teardown := func() {
		cleanup()
		telemetryShutdown()
	}
	return server, teardown, nil
}

// RunACPTransport builds the ACP server (same as RunACP) but serves on
// custom I/O streams instead of os.Stdin/os.Stdout. Used by the TUI to
// run the ACP server in-process via io.Pipe — no subprocess needed.
func RunACPTransport(ctx context.Context, cfg *config.Config, w io.Writer, r io.Reader) error {
	server, cleanup, err := BuildACPServer(ctx, cfg)
	if err != nil {
		return err
	}
	defer cleanup()
	return server.RunTransport(ctx, w, r)
}

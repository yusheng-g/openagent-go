package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/acp"
	"github.com/yusheng-g/openagent-go/agent"
	ctxpkg "github.com/yusheng-g/openagent-go/context"
	openaiembed "github.com/yusheng-g/openagent-go/embedder/openai"
	otelhooks "github.com/yusheng-g/openagent-go/hooks/otel"
	redacthook "github.com/yusheng-g/openagent-go/hooks/redact"
	sloghooks "github.com/yusheng-g/openagent-go/hooks/slog"
	"github.com/yusheng-g/openagent-go/kernel"
	"github.com/yusheng-g/openagent-go/model/openai"
	memorysqlite "github.com/yusheng-g/openagent-go/provider/memory/sqlite"
	openviking "github.com/yusheng-g/openagent-go/provider/openviking"
	"github.com/yusheng-g/openagent-go/provider/skill"
	"github.com/yusheng-g/openagent-go/sandbox/native"
	"github.com/yusheng-g/openagent-go/session"
	sessionsqlite "github.com/yusheng-g/openagent-go/session/sqlite"
	"github.com/yusheng-g/openagent-go/skill/fs"
	builtinskills "github.com/yusheng-g/openagent-go/skills"
	opentool "github.com/yusheng-g/openagent-go/tool"
	"github.com/yusheng-g/openagent-go/version"

	"github.com/yusheng-g/openagent-go/cmd/cli/config"
)

// ── Shared agent setup ──

// buildMemory opens the SQLite session store and knowledge provider under
// config.Dir()/memory/. The conversation (SessionStore/Compressor) and the
// knowledge provider (MemoryProvider) share one database file via separate
// connections (WAL); the metadata Store shares the conversation connection.
func buildMemory(emb config.EmbeddingConfig, embedder bool) (*sessionsqlite.MessageStore, ctxpkg.MemoryProvider, session.Store, func(), error) {
	memDir := filepath.Join(configDir(), "memory")
	_ = os.MkdirAll(memDir, 0755)
	path := filepath.Join(memDir, "memory.db")
	ms, err := sessionsqlite.NewMessageStore(path)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("message store: %w", err)
	}
	var knowledge ctxpkg.MemoryProvider
	if embedder {
		kmem, err := memorysqlite.New(path)
		if err != nil {
			ms.Close()
			return nil, nil, nil, nil, fmt.Errorf("knowledge store: %w", err)
		}
		if emb.Provider != "" {
			// External embedding backend (OpenAI-compatible /embeddings:
			// OpenAI, Ollama, Jina, Cohere, local proxies).
			kmem.WithEmbedder(openaiembed.New(emb.BaseURL, emb.APIKey, emb.Model))
		} else {
			// No embedding backend configured: semantic vector recall stays
			// disabled (Memory falls back to keyword search). The built-in
			// BGE embedder was removed to keep the default build cgo-free;
			// configure embedding.provider in settings to enable semantic
			// recall. WithEmbedder is intentionally NOT called — nil embedder.
		}
		knowledge = kmem
	}
	store, err := sessionsqlite.New(ms.DB())
	if err != nil {
		ms.Close()
		return nil, nil, nil, nil, fmt.Errorf("session store: %w", err)
	}
	return ms, knowledge, store, func() {
		store.Close()
		if c, ok := knowledge.(interface{ Close() error }); ok {
			c.Close()
		}
		ms.Close()
	}, nil
}

// buildModels creates OpenAI model instances from config providers. Provider
// keys are sorted so the model order is deterministic (callers that fall back
// to "the first model" — resolveModel, NewAgentServer's default — pick the
// same one every run, not a random map iteration order).
func buildModels(providers map[string]config.ProviderConfig) ([]openagent.Model, []modelReg) {
	pids := make([]string, 0, len(providers))
	for pid := range providers {
		pids = append(pids, pid)
	}
	sort.Strings(pids)
	var models []openagent.Model
	var infos []modelReg
	for _, pid := range pids {
		p := providers[pid]
		for _, mc := range p.Models {
			apiKey := p.APIKey
			if apiKey == "" {
				apiKey = os.Getenv(strings.ToUpper(pid) + "_API_KEY")
			}
			m := openai.New(apiKey, mc.ID, p.BaseURL)
			// Explicit max_input_tokens overrides the built-in vendor
			// lookup (quantized/shrunk custom models may declare 1M but
			// serve 128K).
			if mc.MaxInputTokens > 0 {
				m = m.WithContextWindow(mc.MaxInputTokens)
			}
			models = append(models, m)
			infos = append(infos, modelReg{
				ID:                       mc.ID,
				Provider:                 pid,
				Model:                    m,
				APIKey:                   apiKey,
				BaseURL:                  p.BaseURL,
				MaxInputTokens:           mc.MaxInputTokens,
				MaxOutputTokens:          mc.MaxOutputTokens,
				InputCostPerMillion:      mc.InputCostPerMillion,
				InputCacheCostPerMillion: mc.InputCacheCostPerMillion,
				OutputCostPerMillion:     mc.OutputCostPerMillion,
			})
		}
	}
	return models, infos
}

type modelReg struct {
	ID                       string
	Provider                 string
	Model                    openagent.Model
	APIKey                   string
	BaseURL                  string
	MaxInputTokens           int
	MaxOutputTokens          int
	InputCostPerMillion      float64
	InputCacheCostPerMillion float64
	OutputCostPerMillion     float64
}

// Key returns the registry key for this model: "<provider>/<id>" when a
// provider is set, else just "<id>". This is the canonical key format used
// by modelMap, RegisterModel, resolveModel, and reconfigureModels — keep
// them consistent by routing through here.
func (mi modelReg) Key() string {
	if mi.Provider != "" {
		return mi.Provider + "/" + mi.ID
	}
	return mi.ID
}

// resolveModel picks the model instance for REST/CLI (single-model modes).
// settings "model" ("<provider>/<modelID>") wins when it matches a registered
// model; otherwise the first configured model is used as a fallback. This
// mirrors ACP's srv.SetDefaultModelID(cfg.Model) so all three entrypoints
// honor the same settings.model preference instead of REST/CLI silently
// picking a random map-iteration order.
func resolveModel(cfgModel string, infos []modelReg) openagent.Model {
	if cfgModel != "" {
		for _, mi := range infos {
			if mi.Key() == cfgModel {
				return mi.Model
			}
		}
	}
	for _, mi := range infos {
		if mi.Model != nil {
			return mi.Model
		}
	}
	return nil
}

// applyContextProviders selects the provider backend per capability.
// OpenViking is a whole-context service (one endpoint, server-managed
// memory/resource/skill indexes): a configured endpoint switches ALL
// three domains to it by default — one address is enough. context_providers
// remains as an opt-out escape hatch: an explicit "builtin" for a domain
// keeps the local backend. No endpoint = fully local, no server required.
func applyContextProviders(cfg *config.Config, deps *kernel.Deps) error {
	cp := cfg.ContextProviders
	if cfg.OpenViking.Endpoint == "" {
		return nil
	}
	client, err := openviking.NewClient(cfg.OpenViking.Endpoint, cfg.OpenViking.APIKey)
	if err != nil {
		return fmt.Errorf("openviking: %w", err)
	}
	if cp.Memory != "builtin" {
		deps.MemoryProvider = openviking.NewMemoryWithRecall(client, openviking.RecallConfig{
			Quotas:   cfg.OpenViking.Recall.Quotas,
			MaxChars: cfg.OpenViking.Recall.MaxChars,
			MinScore: cfg.OpenViking.Recall.MinScore,
		})
	}
	if cp.Skill != "builtin" {
		deps.SkillProvider = openviking.NewSkill(client, nil)
	}
	if cp.Resource != "builtin" {
		deps.ResourceProvider = openviking.NewResource(client)
	}
	return nil
}

// sandboxPolicy translates the config-layer SandboxConfig into a
// native.Policy. Empty Network is treated as "host" (matches the
// sandbox package's zero-value default), so missing config yields
// network access for the agent — required for shell tools that
// reach LLM providers, package managers, cloud CLIs, etc.
func sandboxPolicy(cfg config.SandboxConfig) native.Policy {
	return native.Policy{
		Enabled:       cfg.Enabled,
		Network:       cfg.Network,
		WritablePaths: cfg.WritablePaths,
		ReadablePaths: cfg.ReadablePaths,
	}
}

// buildTools creates the standard file/shell tool set using the sandbox.
// workDir is the workspace root; the tool list selects which tools to create.
func buildTools(sandbox *native.Sandbox, workDir string, toolList []string) []openagent.Tool {
	enabled := make(map[string]bool)
	for _, name := range toolList {
		enabled[name] = true
	}
	var tools []openagent.Tool
	if enabled["shell"] {
		tools = append(tools, opentool.NewShell(sandbox))
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
	if enabled["edit"] {
		tools = append(tools, opentool.NewEditFile(workDir))
	}
	if enabled["websearch"] {
		tools = append(tools, opentool.NewWebSearch())
	}
	if enabled["webfetch"] {
		tools = append(tools, opentool.NewWebFetch())
	}
	if enabled["browser"] {
		// One-shot headless tools (browser_navigate/screenshot/evaluate/click)
		// + persistent-session tools (browser_use_open/snapshot/click/type/
		// press/play_media/tabs/switch_tab/close_tab/close).
		tools = append(tools, opentool.NewBrowserTools()...)
		tools = append(tools, opentool.NewBrowserUseTools()...)
	}
	if enabled["office"] {
		// PPT tools: pptx_read (pure Go) + pptx_template_analyze/fill (pure
		// Go) + pptx_write (Node.js PptxGenJS, embedded worker bundle).
		tools = append(tools, opentool.NewOfficeTools(workDir)...)
	}
	if enabled["settings"] {
		// Server-level config tool: read/modify settings.json atomically.
		// Not cwd-scoped (settings.json is global), but built here so it
		// follows the same capability-gated assembly as other tools.
		// Classified Dangerous (not read-only) — settings.json carries
		// apikeys/secrets, so even get/list routes through the approver.
		tools = append(tools, newSettingsTool())
	}
	return tools
}

// ── Static context (AGENTS.md / SOUL.md) ──

// builtinSystemPrompt is a non-overridable system prompt containing
// built-in rules the model must follow regardless of user customization.
// Injected BEFORE the user-overridable prompts (SOUL/SYSTEM/AGENTS) and
// cannot be replaced by a .md file.
const builtinSystemPrompt = `# Built-in Rules
<system-reminder> tags wrap system events, not user messages. They are delivered asynchronously when something happens in the background (e.g. a sub-agent completed, settings were modified). A system event is not a request from the user to continue working — take the indicated action if needed, otherwise stop and wait for the user.`

// methodologyAndRulesPrompt is the built-in default for AGENTS.md.
// It defines working methodology and behavioral rules.
const methodologyAndRulesPrompt = `# Methodology & Rules
CRITICAL: Do not present uncertain conclusions as facts.
CRITICAL: Do not include secrets or credential values in user-facing output.
CRITICAL: Any factual result that depends on the current environment, files, commands, external systems, or runtime state must be obtained through tools or explicitly confirmed by the user.
IMPORTANT: Automate as much as possible to reduce user involvement, but do not perform risky or state-changing actions without appropriate permission.
IMPORTANT: Explain important actions briefly before taking them.
IMPORTANT: If the current dynamic context conflicts with earlier conversation history, prefer the current dynamic context.
- When receiving a large or complex task, decompose it into structured steps before starting work.
- Read existing context before making changes — understand, then act.
- After each tool execution, verify the result before proceeding to the next step.
- Use recall to search conversation history for exact details — commands, file names, dates — not covered by the summary.
- When uncertain about requirements, ask clarifying questions rather than guessing.
`

// personaAndLimitsPrompt is the built-in default for SOUL.md.
// It defines personality, tone, and behavioral boundaries.
const personaAndLimitsPrompt = `# Persona & Limits
IMPORTANT: Always use the same language as the user. If the user asks in Chinese, reasoning and response in Chinese.
IMPORTANT: Help the user complete tasks by using available tools when appropriate. Do not ask the user to perform operations that you can safely perform yourself with available tools.
- Be concise and direct. Do not flatter, apologize excessively, or hedge.
- Never delete, move, or overwrite files without explicit user confirmation.
- When asked to do something impossible or unsafe, explain why and suggest alternatives.
- Respect user time — surface the most relevant information first. Avoid verbose preambles.
- Use clear, imperative language for actions; use structured formatting for complex output.
`

// systemContextPrompt is the built-in default for SYSTEM.md.
// It is a system-level prompt slot for environment-wide instructions that
// sit between persona (SOUL.md) and methodology (AGENTS.md). Override by
// placing SYSTEM.md in the profile directory.
const systemContextPrompt = `# System Instructions
CRITICAL: Do not claim completion unless the relevant work has actually been performed or verified.
IMPORTANT: Be concise, practical, and action-oriented.
IMPORTANT: Keep user-facing text focused on progress, decisions, results, and next actions.

- Prefer direct answers and concrete next actions.
- Avoid long hidden-style reasoning in user-facing text.
- Do not narrate every internal consideration.
- Summarize tool results only as much as needed to continue the task.
- If something fails, explain the failure briefly and choose the next best action.
- Avoid repeating the same status update unless new information was learned.
`

// profileDir returns the prompt content directory: config.Dir()/
// profile — a fixed subdirectory of the configuration directory, so
// OPENAGENT_CLI_CONFIG is the single root for every persistent path.
func profileDir() string {
	return filepath.Join(configDir(), "profile")
}

// resolvePluginsDir returns the agent plugin directory: config.Dir()/
// plugins — the same root as the CLI plugin default (config.DefaultPluginsDir),
// so all plugins live in one place regardless of loader.
func resolvePluginsDir() string {
	return filepath.Join(configDir(), "plugins")
}

// configDir returns the configuration directory (config.Dir), with a
// home fallback when it cannot be resolved.
func configDir() string {
	return config.Dir()
}

// resolveProfiles reads SOUL.md, SYSTEM.md, and AGENTS.md: project-level
// directory. Falls back to built-in defaults when the files are missing.
//
// cwd is the working directory to search for project-level prompts; if empty,
// os.Getwd() is used.
//
// Resolution order (per file):
//  1. $(cwd)/FILE.md
//  2. <config-dir>/profile/FILE.md
//  3. built-in default
//
// The prompts are returned in injection order: SOUL → SYSTEM → AGENTS.
func resolveProfiles(cwd string) []string {
	return []string{
		builtinSystemPrompt, // non-overridable built-in rules
		resolveProfileFile(cwd, "SOUL.md", personaAndLimitsPrompt),
		resolveProfileFile(cwd, "SYSTEM.md", systemContextPrompt),
		resolveProfileFile(cwd, "AGENTS.md", methodologyAndRulesPrompt),
	}
}

func resolveProfileFile(cwd, filename, defaultText string) string {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	// 1.  Project-level: $(cwd)/FILE.md (AGENTS.md convention — the
	// project's own prompt lives with the project).
	if cwd != "" {
		p := filepath.Join(cwd, filename)
		if data, err := os.ReadFile(p); err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	// 2.  User-level: <config-dir>/profile/FILE.md — a custom
	// directory derives from the configuration directory, so a custom
	// OPENAGENT_CLI_CONFIG relocates the prompts with it.
	p := filepath.Join(profileDir(), filename)
	if data, err := os.ReadFile(p); err == nil {
		return strings.TrimSpace(string(data))
	}

	// 3.  Built-in default
	return defaultText
}

// ── Optional capability builders ──

// openSkillProvider creates a file-system skill provider spanning the skill
// directories that exist plus the embedded built-in skills (bundled via
// //go:embed, no build tag required). Roots are passed to fs.New in override
// order (user-level first, project-level last), so skills in a later root
// override same-name skills from an earlier root:
//
//  1. ~/.agents/skills            (user-level)
//  2. <workspace>/.agents/skills  (project-level, overrides user-level)
//  3. embedded built-in skills    (lowest priority, always present)
//
// Directories that do not exist are skipped. Returns nil only when there are
// no disk roots AND no embedded skills.
func openSkillProvider() skill.Provider {
	var roots []fs.RootEntry
	for _, re := range skillDirs() {
		if info, err := os.Stat(re.Path); err == nil && info.IsDir() {
			roots = append(roots, re)
		}
	}
	embedFS := builtinskills.BuiltinFS()
	if len(roots) == 0 && embedFS == nil {
		return nil
	}
	loader := fs.NewWithSources(roots...)
	if embedFS != nil {
		loader = loader.WithEmbedFS(embedFS)
	}
	return skill.NewFSBridge(loader)
}

// skillDirs returns the skill directory candidates in override order:
// user-level first (~/.agents/skills, type="global"), project-level last
// (<cwd>/.agents/skills, type="project", overrides user-level). When home
// equals cwd the two resolve to the same path and only one entry is returned.
func skillDirs() []fs.RootEntry {
	var dirs []fs.RootEntry
	seen := make(map[string]struct{})

	home, err := os.UserHomeDir()
	if err == nil {
		d := filepath.Join(home, ".agents", "skills")
		if _, ok := seen[d]; !ok {
			seen[d] = struct{}{}
			dirs = append(dirs, fs.RootEntry{Path: d, Type: "global"})
		}
	}

	cwd, err := os.Getwd()
	if err == nil {
		d := filepath.Join(cwd, ".agents", "skills")
		if _, ok := seen[d]; !ok {
			seen[d] = struct{}{}
			dirs = append(dirs, fs.RootEntry{Path: d, Type: "project"})
		}
	}
	return dirs
}

// buildSlogHooks creates slog-based RunHooks (the lifecycle axis).
func buildSlogHooks() openagent.RunHooks {
	return sloghooks.New(slog.Default())
}

// buildSlogObserver creates the observation-axis slog Observer (stage
// boundaries + decision events) backed by slog.Default(). The lifecycle-axis
// Hooks are wired separately by buildSlogHooks; both share slog.Default() so
// a single handler config (rotated file / discard / never stderr — stderr is
// the ACP control pipe) governs both axes. Returning *sloghooks.Observer
// (which implements DecisionObserver) lets the kernel's decision dispatch
// type-assert succeed, so DecisionEvents reach the slog sink in HTTP/run mode
// — the old private slogObserver implemented only ObserveStage and silently
// dropped every DecisionEvent.
func buildSlogObserver() openagent.RunObserver {
	return sloghooks.NewObserver(slog.Default())
}

// buildOpts appends capability-gated agent options (skills, sub-agents) to
// opts and returns the skill provider for the runtime deps. The guard is NOT
// wired here — it is constructed by each entrypoint (RunACP/RunREST/RunCLI)
// so it can resolve the judge model at the right scope: ACP uses the server's
// dynamic lookup (after srv exists), REST/CLI use a static model closure.
func buildOpts(opts []agent.Option, caps config.Capabilities) ([]agent.Option, skill.Provider) {
	var sp skill.Provider
	if caps.OnSkills() {
		sp = openSkillProvider()
	}
	// explore is a read-only code-exploration sub-agent. Its tool allowlist
	// (read/ls/grep/shell) keeps it from mutating files; filterChildTools
	// additionally strips sub-agent recursion. MaxTurns 100 lets it work
	// through a read→grep→read investigation in one delegation.
	opts = append(opts, agent.WithSubAgents(exploreSubAgent()))
	return opts, sp
}

// exploreSubAgent is the built-in read-only exploration sub-agent. The model
// delegates a focused investigation; the child runs in an isolated context
// (no parent history) with only read/ls/grep/shell and reports back findings.
func exploreSubAgent() agent.SubAgent {
	return agent.SubAgent{
		Name:        "explore",
		Description: "Read-only code exploration. Use when you need to understand, locate, or analyze code — especially across multiple files or directories. Launch multiple explore sub-agents to cover different areas in parallel, then wait for their completion notifications. Does NOT modify files. Returns an agent_id for follow-up questions via sub_agent_send.",
		SystemPrompt: "You are a read-only exploration sub-agent. Your job is to locate and understand " +
			"code, then report findings — not to modify anything. Use read, ls, grep, and read-only " +
			"shell commands (git, find). Do NOT write or edit files. Deliverable: a concise report " +
			"answering the task directly — the relevant file paths with line numbers, how the pieces " +
			"connect, and the minimal code quoted to make the point. Don't dump whole files.",
		Tools: []string{
			"read", "ls", "grep", "shell",
			"websearch", "webfetch",
			// One-shot headless browser tools — webfetch can't render JS SPAs
			// (GitHub, docs sites); browser_navigate/screenshot/evaluate/click
			// are the read-side fallback for those. browser_use_* (persistent
			// multi-step automation) is excluded — too heavy for exploration.
			"browser_navigate", "browser_screenshot", "browser_evaluate", "browser_click",
		},
		MaxTurns: 100,
	}
}

// setupTelemetry initializes the OpenTelemetry TracerProvider from config.
// Returns the tracer (nil if telemetry is disabled) and a shutdown function
// the caller must defer. When cfg.Telemetry.Endpoint is empty, a no-op
// provider is returned — spans are created but never exported, so the otel
// hook can be wired unconditionally.
func setupTelemetry(ctx context.Context, cfg config.Config) (*otelhooks.TracerHolder, *otelhooks.SetupResult, func(), error) {
	insecure := true
	if cfg.Telemetry.Insecure != nil {
		insecure = *cfg.Telemetry.Insecure
	}
	serviceName := strings.TrimSpace(cfg.Telemetry.ServiceName)
	if serviceName == "" {
		serviceName = version.Name
	}
	result, err := otelhooks.SetupTracer(ctx, otelhooks.Config{
		Endpoint:    cfg.Telemetry.Endpoint,
		Protocol:    cfg.Telemetry.Protocol,
		ServiceName: serviceName,
		Insecure:    insecure,
	})
	if err != nil {
		return nil, nil, func() {}, err
	}
	holder := otelhooks.NewTracerHolder(result.Tracer)
	shutdown := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := result.Shutdown(shutdownCtx); err != nil {
			slog.Warn("telemetry shutdown failed", "error", err)
		}
	}
	return holder, result, shutdown, nil
}

// ── Settings hot-reload ──

// ReloadResult describes what happened during a reload — returned by
// reload() and surfaced to the agent/user via the `reload` action so they
// can see what was applied (vs. the opaque "hot-reloaded if a server is
// running" message).
type ReloadResult struct {
	Applied    []string // changes applied (e.g. "log level: info→trace")
	Skipped    []string // changes skipped (e.g. "sandbox.enabled: restart required")
	Violations []string // enum violations found (reload blocked if non-empty)
	ParseError string   // non-empty if the config failed to parse
}

// settingsWatcher holds the state needed to hot-reload settings.json at
// runtime: the previous config (for diffing), the TracerHolder (for
// telemetry reconfiguration), and the ACP AgentServer (for model registry
// updates, nil in REST/run mode).
type settingsWatcher struct {
	cfgPath    string
	mu         sync.Mutex
	prev       *config.Config
	holder     *otelhooks.TracerHolder
	prevResult *otelhooks.SetupResult
	shutdown   func()
	srv        *acp.AgentServer // nil in REST/run mode
}

// activeWatcher is the process-wide settings watcher, set when the server
// starts. The settings tool's `reload` action uses it to trigger an
// immediate reload (vs. waiting for the 500ms fsnotify debounce). nil when
// no server is running (CLI mode, or before watchSettings starts).
//
// atomic.Pointer provides happens-before synchronization between the writer
// (watchSettings goroutine) and readers (settings tool reload action in ACP
// session goroutines) — a plain global var would be a data race.
var activeWatcher atomic.Pointer[settingsWatcher]

// onExternalChange is called by the fsnotify watcher when settings.json
// changes. In ACP mode, it broadcasts a <system-reminder> to all sessions
// so the model can decide whether to reload. In REST mode (no srv), it
// falls back to auto-reload (no model to notify).
//
// The notification is sent for ALL file changes, including writes by the
// settings tool itself. The model is smart enough to ignore the notification
// when it just called set+reload (it knows the change was its own). This
// avoids the need for a suppression flag (which would be a workaround with
// a race window: if an external edit happens within the 500ms debounce
// after a tool write, the flag would swallow it).
func (sw *settingsWatcher) onExternalChange(ctx context.Context) {
	if sw.srv != nil {
		// ACP: notify all active sessions via idle turn. triggerIdleTurn
		// sends an "idle_turn_end" session/update after the turn ends so
		// the frontend re-enables user input (same pattern as
		// context_compacting / available_skills_update — a custom
		// session/update subtype).
		sw.srv.BroadcastSystemReminder(kernel.FormatSettingsChangeNote())
	} else {
		// REST: no sessions/model to notify — auto-reload.
		sw.reload(ctx)
	}
}

// watchSettings starts an fsnotify watcher on the settings file. When the
// file changes, it debounces 500ms then either notifies the model (ACP) or
// auto-reloads (REST). Returns immediately; runs until ctx is cancelled.
func watchSettings(ctx context.Context, sw *settingsWatcher) {
	// Register as the process-wide watcher so the settings tool's `reload`
	// action can trigger an immediate reload.
	activeWatcher.Store(sw)

	w, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Warn("settings watcher: fsnotify init failed", "error", err)
		return
	}
	defer w.Close()

	// Watch the directory (not the file) — atomic rename (temp → rename)
	// replaces the inode, so a file-level watch would miss the change.
	// Directory-level watch catches the rename event.
	dir := filepath.Dir(sw.cfgPath)
	if err := w.Add(dir); err != nil {
		slog.Warn("settings watcher: cannot watch directory", "dir", dir, "error", err)
		return
	}

	var debounce *time.Timer
	for {
		select {
		case <-ctx.Done():
			if debounce != nil {
				debounce.Stop()
			}
			return
		case event := <-w.Events:
			// Only react to writes/creates on the settings file itself.
			if event.Name != sw.cfgPath {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			// Debounce: editors may save multiple times in quick succession.
			if debounce != nil {
				debounce.Stop()
			}
			debounce = time.AfterFunc(500*time.Millisecond, func() {
				sw.onExternalChange(ctx)
			})
		case err := <-w.Errors:
			slog.Warn("settings watcher error", "error", err)
		}
	}
}

// reload reads the new settings.json, parses it, diffs against the previous
// config, and applies changes to telemetry/log-level/models. Returns a
// ReloadResult describing what was applied/skipped. Called by the fsnotify
// watcher (auto) and by the settings tool's `reload` action (explicit).
func (sw *settingsWatcher) reload(ctx context.Context) ReloadResult {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	var result ReloadResult

	raw, err := os.ReadFile(sw.cfgPath)
	if err != nil {
		result.ParseError = fmt.Sprintf("cannot read file: %v", err)
		slog.Warn("settings reload: cannot read file", "error", err)
		return result
	}
	// Resolve env-var references before unmarshaling, identical to startup
	// (main.go). Warnings (vars referenced without a default that are unset)
	// are logged: a reload is debounced (500ms), so each intentional edit
	// produces at most one warning per unset var — not spam. This matches
	// startup behavior and surfaces misconfigured env refs introduced via
	// hot-reload (e.g. a newly added ${NEW_VAR} that the operator forgot to
	// export). Both sw.prev and newCfg end up expanded, so the
	// reflect.DeepEqual diff below compares resolved values.
	raw, reloadWarns := config.ExpandBytes(raw)
	for _, w := range reloadWarns {
		slog.Warn("settings reload: env var referenced but not set", "var", w)
	}
	var newCfg config.Config
	if err := json.Unmarshal(raw, &newCfg); err != nil {
		result.ParseError = fmt.Sprintf("parse failed: %v", err)
		slog.Warn("settings reload: parse failed, keeping previous config", "error", err)
		return result
	}
	config.ApplyDefaults(&newCfg, sw.cfgPath)

	// Validate-gated reload: if the new config introduces NEW enum violations
	// (violations not present in the previously-accepted config), skip the
	// reload and keep the previous config. The running server stays on the
	// last-known-good config rather than silently degrading.
	//
	// Only NEW violations block the reload: if the server started with an
	// existing violation (startup warns but does not fatal), a subsequent
	// reload that changes an unrelated field (e.g. telemetry) should NOT be
	// blocked by the pre-existing violation — the operator's edit is
	// orthogonal. Differencing against sw.prev avoids this trap. Pre-existing
	// violations are re-logged (info) so they stay visible.
	newViolations := config.CheckEnums(&newCfg)
	prevViolations := config.CheckEnums(sw.prev)
	prevSet := make(map[string]struct{}, len(prevViolations))
	for _, pv := range prevViolations {
		prevSet[pv] = struct{}{}
	}
	var fresh []string
	for _, v := range newViolations {
		if _, ok := prevSet[v]; !ok {
			fresh = append(fresh, v)
		}
	}
	if len(fresh) > 0 {
		result.Violations = fresh
		for _, v := range fresh {
			slog.Warn("settings reload: new validation violation, keeping previous config", "violation", v)
		}
		return result
	}
	// No new violations. Re-log any pre-existing ones so the operator is
	// reminded they're still present (the server is running with them).
	for _, v := range newViolations {
		slog.Warn("settings reload: pre-existing violation still present (not blocking)", "violation", v)
	}

	// Telemetry.
	if !reflect.DeepEqual(sw.prev.Telemetry, newCfg.Telemetry) {
		sw.reconfigureTelemetry(ctx, newCfg)
		result.Applied = append(result.Applied, "telemetry reconfigured")
	}

	// Log level.
	if sw.prev.Log.Level != newCfg.Log.Level {
		reconfigureLogLevel(newCfg.Log.Level)
		result.Applied = append(result.Applied, fmt.Sprintf("log level: %s→%s", sw.prev.Log.Level, newCfg.Log.Level))
		slog.Info("settings reloaded: log level", "level", newCfg.Log.Level)
	}

	// Providers/models (ACP only — REST/run have no AgentServer).
	if sw.srv != nil && !reflect.DeepEqual(sw.prev.Provider, newCfg.Provider) {
		sw.reconfigureModels(newCfg)
		result.Applied = append(result.Applied, fmt.Sprintf("providers (disk): %d→%d — note: model registry is additive-only (deleted providers stay in memory until restart; new sessions can still use them)", len(sw.prev.Provider), len(newCfg.Provider)))
	}

	// MCP servers (ACP only). Settings servers are merged with client-
	// advertised ones at session connect; hot-swapping affects new sessions
	// only (existing sessions keep their connected tools).
	if sw.srv != nil && !reflect.DeepEqual(sw.prev.McpServers, newCfg.McpServers) {
		sw.srv.SetSettingsMcpServers(convertMcpServers(newCfg.McpServers))
		result.Applied = append(result.Applied, fmt.Sprintf("mcp servers: %d→%d (new sessions)", len(sw.prev.McpServers), len(newCfg.McpServers)))
		slog.Info("settings reloaded: mcp servers", "count", len(newCfg.McpServers))
	}

	sw.prev = &newCfg
	if len(result.Applied) == 0 {
		result.Applied = append(result.Applied, "no hot-reloadable changes (restart may be required for non-reloadable fields)")
	}
	slog.Info("settings reloaded")
	return result
}

// reconfigureTelemetry shuts down the old TracerProvider and creates a new
// one with the updated endpoint/protocol. The TracerHolder is updated so
// all hooks and observers pick up the new tracer.
func (sw *settingsWatcher) reconfigureTelemetry(ctx context.Context, cfg config.Config) {
	// Shutdown old tracer with a timeout so a stuck exporter doesn't block.
	if sw.prevResult != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := sw.prevResult.Shutdown(shutdownCtx); err != nil {
			slog.Warn("settings reload: old telemetry shutdown failed", "error", err)
		}
		cancel()
	}

	insecure := true
	if cfg.Telemetry.Insecure != nil {
		insecure = *cfg.Telemetry.Insecure
	}
	serviceName := strings.TrimSpace(cfg.Telemetry.ServiceName)
	if serviceName == "" {
		serviceName = version.Name
	}
	result, err := otelhooks.SetupTracer(ctx, otelhooks.Config{
		Endpoint:    cfg.Telemetry.Endpoint,
		Protocol:    cfg.Telemetry.Protocol,
		ServiceName: serviceName,
		Insecure:    insecure,
	})
	if err != nil {
		slog.Warn("settings reload: telemetry setup failed", "error", err)
		return
	}
	sw.prevResult = result
	sw.shutdown = func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := result.Shutdown(shutdownCtx); err != nil {
			slog.Warn("telemetry shutdown failed", "error", err)
		}
	}
	// Swap the tracer in the holder — all hooks/observers pick it up.
	if sw.holder != nil {
		sw.holder.Set(result.Tracer)
	}
	slog.Info("settings reloaded: telemetry", "endpoint", cfg.Telemetry.Endpoint, "protocol", cfg.Telemetry.Protocol)
}

// reconfigureModels applies provider/model additions and updates to the ACP
// model registry. New providers/models are inserted via SetModel; changed
// providers (edited API key/baseURL/context window) are replaced in place.
//
// Providers no longer in settings are NOT removed. Two reasons:
//  1. Plugin-injected providers (from cli:settings plugins) are absent from
//     the file-only config that reload parses, so a delete-on-missing loop
//     would scrub them on every reload. SetModel is replace-or-insert, so
//     add/update still works; the missing-from-file case is simply left
//     untouched.
//  2. Existing sessions may still reference a model the user just removed
//     from settings.json; deletion would break them. Removal happens on
//     restart.
//
// This matches the original design intent this function's comment stated
// ("removed ones are logged but not deleted") before the delete loop was
// added in contradiction to it.
func (sw *settingsWatcher) reconfigureModels(cfg config.Config) {
	_, newInfos := buildModels(cfg.Provider)
	for _, mi := range newInfos {
		sw.srv.SetModel(mi.Provider, mi.ID, mi.APIKey, mi.BaseURL,
			mi.MaxInputTokens, mi.MaxOutputTokens)
	}
	// The extractor uses a dynamic model lookup (SetModelFn at startup),
	// so it automatically picks up the new model instances — no manual
	// update needed here.
	// Notify all active sessions so the frontend model dropdown refreshes.
	sw.srv.BroadcastConfigOptions()
	slog.Info("settings reloaded: models", "providers", len(cfg.Provider), "models", len(newInfos))
}

// modes: the RunHooks pipeline and the stage observer. Mode-specific
// capabilities (Tools, Memory, Approver) are added by the caller.
//
// sensitive carries the user-configured sensitive env-var names; it is
// honored only when caps.OnHooks() is true (redact rides the hooks pipeline).
// tracer, when non-nil, adds an OTel hook that emits agent.run and tool.<name>
// spans. Hook order is redact → otel → slog: redact first (secrets masked
// before any other hook sees the data), otel before slog (spans carry the
// redacted args).
func buildRuntimeDeps(caps config.Capabilities, sensitive config.SensitiveConfig, holder *otelhooks.TracerHolder) kernel.Deps {
	hooks := []openagent.RunHooks{
		redacthook.NewHook(sensitive.Env),
	}
	observers := []openagent.RunObserver{buildSlogObserver()}
	// Always mount OTel hooks/observer (even when tracer is nil at startup)
	// so runtime telemetry activation via settings hot-reload works without
	// rebuilding kernel.Deps. When tracer is nil, Start() returns a no-op
	// span — zero overhead. When holder.Set(newTracer) is called later by
	// the settings watcher, spans start being created and exported.
	if holder != nil {
		hooks = append(hooks, otelhooks.NewWithHolder(holder))
		observers = append(observers, otelhooks.NewObserverWithHolder(holder))
	}
	hooks = append(hooks, buildSlogHooks())
	return kernel.Deps{
		Hooks:    openagent.MultiHooks(hooks...),
		Observer: openagent.MultiObserver(observers...),
	}
}

// channelDir returns the per-channel state directory: config.Dir()/
// channel/<name> — channel locks, credentials, QR caches, and media
// live beside memory and profile, not inside either.
func channelDir(name string) string {
	return filepath.Join(configDir(), "channel", name)
}

// sessionObserver is the process-wide session lifecycle observer, wired at
// startup by main.go via SetSessionObservers.  nil = no observers (events
// silently dropped).  Used by RunCLI and the ACP/REST servers to emit
// session create/close/delete events.
var sessionObserver openagent.SessionObserver

// SetSessionObservers wires the process-wide session lifecycle observer.
// Call once at startup (e.g. in main.go after track.Init()).  Pass nil to
// disable.
func SetSessionObservers(obs openagent.SessionObserver) {
	sessionObserver = obs
}

package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/guard/llm"
	artifacthook "github.com/yusheng-g/openagent-go/hooks/artifact"
	redacthook "github.com/yusheng-g/openagent-go/hooks/redact"
	sloghooks "github.com/yusheng-g/openagent-go/hooks/slog"
	"github.com/yusheng-g/openagent-go/memory/sqlite"
	"github.com/yusheng-g/openagent-go/model/openai"
	"github.com/yusheng-g/openagent-go/sandbox/native"
	"github.com/yusheng-g/openagent-go/session"
	sessionsqlite "github.com/yusheng-g/openagent-go/session/sqlite"
	"github.com/yusheng-g/openagent-go/skill/fs"
	opentool "github.com/yusheng-g/openagent-go/tool"

	"github.com/yusheng-g/openagent-go/cmd/cli/config"
)

// ── Shared agent setup ──

// buildMemory opens the SQLite memory and session store under profilesDir/memory/.
func buildMemory(profilesDir string) (*sqlite.Memory, session.Store, func(), error) {
	memDir := filepath.Join(profilesDir, "memory")
	_ = os.MkdirAll(memDir, 0755)
	mem, err := sqlite.New(filepath.Join(memDir, "memory.db"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("memory: %w", err)
	}
	mem.WithSemanticMD(filepath.Join(memDir, "semantic.md"))
	store, err := sessionsqlite.New(mem.DB())
	if err != nil {
		mem.Close()
		return nil, nil, nil, fmt.Errorf("session store: %w", err)
	}
	return mem, store, func() { store.Close(); mem.Close() }, nil
}

// buildModels creates OpenAI model instances from config providers.
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
			infos = append(infos, modelReg{ID: mid, Provider: pid, Model: m, APIKey: apiKey, BaseURL: p.BaseURL})
		}
	}
	return models, infos
}

type modelReg struct {
	ID       string
	Provider string
	Model    openagent.Model
	APIKey   string
	BaseURL  string
}

func firstModel(models []openagent.Model) openagent.Model {
	for _, m := range models {
		if m != nil {
			return m
		}
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
	return tools
}

// ── Static context (AGENTS.md / SOUL.md) ──

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
const personaAndLimitsPrompt = `You are openagent, a fully pluggable AI agent.
# Persona & Limits
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
// placing SYSTEM.md in the profiles directory.
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

// resolveProfilesDir resolves the profiles directory to an absolute path.
// $(pwd)/$(profiles) takes priority, ~/$(profiles) is fallback.
func resolveProfilesDir(profiles string) string {
	if profiles == "" {
		profiles = ".openagent"
	}
	if cwd, err := os.Getwd(); err == nil {
		p := filepath.Join(cwd, profiles)
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, profiles)
	}
	return filepath.Join(os.TempDir(), profiles)
}

// resolveProfiles reads SOUL.md, SYSTEM.md, and AGENTS.md from the profiles
// directory. Falls back to built-in defaults when the files are missing.
//
// cwd is the working directory to search for project-level profiles; if empty,
// os.Getwd() is used.
//
// Resolution order (per file):
//  1. $(cwd)/FILE.md, $(cwd)/$(profiles)/FILE.md
//  2. ~/$(profiles)/FILE.md
//  3. built-in default
//
// The prompts are returned in injection order: SOUL → SYSTEM → AGENTS.
func resolveProfiles(profiles, cwd string) []string {
	return []string{
		resolveProfileFile(profiles, cwd, "SOUL.md", personaAndLimitsPrompt),
		resolveProfileFile(profiles, cwd, "SYSTEM.md", systemContextPrompt),
		resolveProfileFile(profiles, cwd, "AGENTS.md", methodologyAndRulesPrompt),
	}
}

func resolveProfileFile(profiles, cwd, filename, defaultText string) string {
	if profiles == "" {
		return defaultText
	}
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	// 1.  Project-level: $(cwd)/FILE.md, $(cwd)/$(profiles)/FILE.md
	if cwd != "" {
		p := filepath.Join(cwd, filename)
		if data, err := os.ReadFile(p); err == nil {
			return strings.TrimSpace(string(data))
		}

		q := filepath.Join(cwd, profiles, filename)
		if data, err := os.ReadFile(q); err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	// 2.  User-level: ~/$(profiles)/FILE.md
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, profiles, filename)
		if data, err := os.ReadFile(p); err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	// 3.  Built-in default
	return defaultText
}

// ── Optional capability builders ──

// openSkillLoader creates a file-system skill loader spanning the skill
// directories that exist. Roots are passed to fs.New in priority order
// (least authoritative first), so skills in a later root override
// same-name skills from an earlier root:
//
//  1. ~/.agents/skills            (user-level)
//  2. <workspace>/.agents/skills  (project-level, overrides user-level)
//
// Directories that do not exist are skipped. Returns nil if none exist.
func openSkillLoader() openagent.SkillLoader {
	var roots []string
	for _, dir := range skillDirs() {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			roots = append(roots, dir)
		}
	}
	if len(roots) == 0 {
		return nil
	}
	return fs.New(roots...)
}

// skillDirs returns the skill directory candidates in override order:
// least authoritative first (~/.agents/skills), then most authoritative
// last (<cwd>/.agents/skills). When home equals cwd the two resolve to
// the same path and only one entry is returned.
func skillDirs() []string {
	var dirs []string
	seen := make(map[string]struct{})

	home, err := os.UserHomeDir()
	if err == nil {
		d := filepath.Join(home, ".agents", "skills")
		if _, ok := seen[d]; !ok {
			seen[d] = struct{}{}
			dirs = append(dirs, d)
		}
	}

	cwd, err := os.Getwd()
	if err == nil {
		d := filepath.Join(cwd, ".agents", "skills")
		if _, ok := seen[d]; !ok {
			seen[d] = struct{}{}
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// buildGuard creates an LLM guard using the given model as judge.
func buildGuard(model openagent.Model) *llm.Guard {
	return llm.New(model)
}

// buildSlogHooks creates slog-based RunHooks.
func buildSlogHooks() openagent.RunHooks {
	return sloghooks.New(slog.Default())
}

// slogObserver logs stage events to a dedicated stderr logger.
type slogObserver struct {
	logger *slog.Logger
}

func (o slogObserver) ObserveStage(_ context.Context, event openagent.StageEvent) {
	o.logger.Info("stage", "name", event.Name, "phase", event.Phase, "duration", event.Duration)
}

// buildSlogObserver creates a minimal stderr stage observer.
func buildSlogObserver() openagent.RunObserver {
	return slogObserver{
		logger: slog.Default(),
	}
}

// buildOpts appends capability-gated agent options (skills, guard, hooks,
// observer) to opts. model is used by the guard; it may be nil if no models
// are configured, in which case the guard is skipped regardless of caps.
//
// sensitive carries the user-configured sensitive env-var names; it is
// honored only when caps.OnHooks() is true (redact rides the hooks pipeline).
// Hook order is redact → slog → artifact: redact must run first so logs and
// artifact files never see raw secrets.
func buildOpts(opts []openagent.AgentOption, caps Capabilities, model openagent.Model, sensitive config.SensitiveConfig) []openagent.AgentOption {
	if caps.OnSkills() {
		if sl := openSkillLoader(); sl != nil {
			opts = append(opts, openagent.WithSkillLoader(sl))
		}
	}
	if caps.OnGuard() && model != nil {
		g := buildGuard(model)
		opts = append(opts, openagent.WithInputGuard(g))
		opts = append(opts, openagent.WithOutputGuard(g.Output()))
	}

	// Order: redact → slog → artifact. redact must run FIRST so every
	// downstream observer (slog error logging, artifact disk save)
	// sees already-redacted data. Putting slog before redact would
	// write the raw secret into the log on tool errors.
	opts = append(opts, openagent.WithRunHooks(
		redacthook.NewHook(sensitive.Env),
		buildSlogHooks(),
		artifacthook.NewHook(),
	))

	opts = append(opts, openagent.WithRunObserver(buildSlogObserver()))
	return opts
}

package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/memory/sqlite"
	"github.com/yusheng-g/openagent-go/model/openai"
	"github.com/yusheng-g/openagent-go/sandbox/native"
	"github.com/yusheng-g/openagent-go/session"
	sessionsqlite "github.com/yusheng-g/openagent-go/session/sqlite"
	opentool "github.com/yusheng-g/openagent-go/tool"

	"github.com/yusheng-g/openagent-go/cmd/cli/config"
)

// ── Shared agent setup ──

// buildMemory opens the SQLite memory and session store at path.
func buildMemory(path string) (*sqlite.Memory, session.Store, func(), error) {
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
			infos = append(infos, modelReg{ID: mid, Provider: pid, Model: m})
		}
	}
	return models, infos
}

type modelReg struct {
	ID       string
	Provider string
	Model    openagent.Model
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
- Use recall to search conversation history for relevant context or past decisions.
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

// resolveProfiles reads AGENTS.md and SOUL.md from the profiles directory.
// Falls back to built-in defaults when the files are missing.
//
// Resolution order (per file):
//  1. $(pwd)/$(profiles)/FILE.md
//  2. ~/$(profiles)/FILE.md
//  3. built-in default
func resolveProfiles(profiles string) []string {
	return []string{
		resolveProfileFile(profiles, "AGENTS.md", methodologyAndRulesPrompt),
		resolveProfileFile(profiles, "SOUL.md", personaAndLimitsPrompt),
	}
}

func resolveProfileFile(profiles, filename, defaultText string) string {
	if profiles == "" {
		return defaultText
	}

	// 1.  Project-level: $(pwd)/$(profiles)/FILE.md
	if cwd, err := os.Getwd(); err == nil {
		p := filepath.Join(cwd, profiles, filename)
		if data, err := os.ReadFile(p); err == nil {
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

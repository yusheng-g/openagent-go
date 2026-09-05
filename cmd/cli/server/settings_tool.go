package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/cmd/cli/config"
)

// ── settings ──
//
// The settings tool lets the agent read and modify settings.json — the
// same operations as the `openagent settings` CLI subcommand (set/get/
// list/append/delete), against the same on-disk file. Writes are atomic
// (temp file + rename); a running server's fsnotify watcher picks up the
// change and hot-reloads applicable config (telemetry, log level, models,
// mcp_servers) within ~500ms.
//
// The tool is NOT read-only — it is classified Dangerous so every action
// (including get/list) routes through the approver in manual mode.
// settings.json carries apikeys and other secrets; even reading it should
// be visible to the user, and sub-agents are excluded from it entirely.

type settingsTool struct{}

type settingsParams struct {
	Action string `json:"action" jsonschema:"description=Operation: set|get|list|append|delete|validate|reload|schema"`
	Key    string `json:"key,omitempty" jsonschema:"description=Dotted-path key (e.g. telemetry.endpoint, provider.openai.api_key, mcp_servers.weather.command). Required for set/get/append/delete; ignored for list/validate/reload/schema. Numeric segments index into arrays (e.g. provider.openai.models.0.id)."`
	Value  string `json:"value,omitempty" jsonschema:"description=Value for set/append. Parsed as JSON when valid (number/bool/object/array); otherwise treated as a plain string. For secrets (api_key, token, secret), PREFER ${ENV_VAR} over a literal value so the key is resolved from env at runtime and the on-disk file stays literal. Ignored for get/list/delete/validate/reload/schema."`
}

func (t *settingsTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name: "settings",
		Description: "Read and modify the agent's settings.json configuration. " +
			"DO NOT proactively modify settings unless the user explicitly asks. " +
			"Actions: set <key> <value>, get <key>, list, append <key> <value>, delete <key>, validate, reload, schema. " +
			"BEFORE any write, call list to see current state. Call schema for the full config structure, valid values, and type rules. " +
			"Workflow: set (write) → validate (check) → reload (apply). Writes do NOT auto-apply — always call reload to activate changes. " +
			"delete is IRREVERSIBLE — the key is removed from disk immediately, no undo. Before calling delete, tell the user what will be removed and ask for confirmation. If a provider or mcp_servers entry is deleted, running sessions may lose their model or tools. " +
			"For secrets (api_key/token/secret), prefer ${ENV_VAR} over literal values. NEVER display secret values in replies.",
		Parameters: openagent.SchemaOf[settingsParams](),
	}
}

// settingsSchemaDoc is the detailed schema reference returned by the
// `schema` action. Kept out of the tool Description so the description stays
// short for the model's context window — the agent fetches this on demand.
const settingsSchemaDoc = `CONFIG SCHEMA (hot-reloadable groups — object keys are NAMES, never array indices):

{
  "provider": {                        // OBJECT keyed by provider name
    "openai": {
      "api_key": "sk-...",             // or ${ENV_VAR} (preferred for secrets)
      "base_url": "https://api.openai.com/v1",
      "models": ["gpt-4o", {"id": "qwen-128k", "max_input_tokens": 128000, "max_output_tokens": 8192, "input_cost_per_million": 1, "input_cache_cost_per_million": 0.1, "output_cost_per_million": 2}]
    }
  },
  "mcp_servers": {                     // OBJECT keyed by server name
    "weather": {"command": "uvx", "args": ["mcp-server-weather"], "env": {"API_KEY": "..."}},
    "remote1": {"url": "https://...", "type": "http"}
  },
  "telemetry": {
    "endpoint": "localhost:4318",      // bare host:port OR full URL "http(s)://..."
    "protocol": "http",                // "http" (default) | "grpc"
    "service_name": "openagent",
    "insecure": true                   // bool; ignored when endpoint is a full URL
  },
  "log": {"file": "...", "level": "info", "max_size": 10, "max_backups": 5, "max_age": 30}
}

TYPE RULES (type-mismatched writes are REJECTED before save):
- provider/mcp_servers MUST be objects keyed by name; arrays are rejected.
- provider.<name>.models: plain string OR object, never bare number.
- mcp_servers.<name>.type: "stdio" (default) | "http" | "sse" (case-sensitive).
- log.level: "trace" | "debug" | "info" | "warn" | "error" (case-insensitive).
- telemetry.protocol: "http" | "grpc" (case-insensitive).
- default_mode/tui.mode: "auto" | "manual" | "plan" (case-sensitive).
- sandbox.network: "host" | "isolated" (case-sensitive).

HOT-RELOAD (no restart): telemetry.*, log.level, provider.* (models), mcp_servers.*.
RESTART-REQUIRED: sandbox.*, capabilities.*, embedding.*, openviking.*, context_providers.*, sensitive.*, channels.*, server.*, env, plugins, default_mode, tui.*.

SECRETS: fields tagged sensitive (provider.*.api_key, channels.*.token/secret/app_secret, embedding.api_key, openviking.api_key). Prefer ${ENV_VAR} — disk stays literal, server resolves from env.

MECHANISM: atomic writes (temp+rename), validated against Config schema before save.`

func (t *settingsTool) Execute(ctx context.Context, args json.RawMessage) *openagent.ToolResult {
	p, err := openagent.ParseArgs[settingsParams](args)
	if err != nil {
		return openagent.ErrorResult(fmt.Errorf("settings: %w", err), false, "")
	}
	action := strings.TrimSpace(p.Action)
	switch action {
	case "set":
		if strings.TrimSpace(p.Key) == "" {
			return openagent.ErrorResult(fmt.Errorf("settings set: key is required"), false, "")
		}
		if err := config.SetSetting(p.Key, p.Value); err != nil {
			return openagent.ErrorResult(fmt.Errorf("settings set: %w", err), true, "")
		}
		msg := fmt.Sprintf("set %s = %s. Call reload to apply.", p.Key, p.Value)
		// If the key is a secret field and the value is a literal (not a
		// ${ENV_VAR} reference), prompt the agent to use env-var indirection
		// so the key is not stored in plaintext on disk.
		if isSecretKey(p.Key) && !looksLikeEnvRef(p.Value) {
			msg += "\nTIP: this looks like a secret written as a literal value. Consider using ${ENV_VAR} instead (e.g. set " + p.Key + " ${MY_API_KEY}) — the on-disk file stays literal and the key is resolved from env at runtime. Export the env var in your shell or .env file."
		}
		return &openagent.ToolResult{Content: msg}
	case "get":
		if strings.TrimSpace(p.Key) == "" {
			return openagent.ErrorResult(fmt.Errorf("settings get: key is required"), false, "")
		}
		val, err := config.GetSetting(p.Key)
		if err != nil {
			return openagent.ErrorResult(fmt.Errorf("settings get: %w", err), false, "")
		}
		return &openagent.ToolResult{Content: val}
	case "list":
		val, err := config.ListSettings()
		if err != nil {
			return openagent.ErrorResult(fmt.Errorf("settings list: %w", err), false, "")
		}
		return &openagent.ToolResult{Content: val}
	case "append":
		if strings.TrimSpace(p.Key) == "" {
			return openagent.ErrorResult(fmt.Errorf("settings append: key is required"), false, "")
		}
		if err := config.AppendSetting(p.Key, p.Value); err != nil {
			return openagent.ErrorResult(fmt.Errorf("settings append: %w", err), true, "")
		}
		return &openagent.ToolResult{Content: fmt.Sprintf("appended to %s. Call reload to apply.", p.Key)}
	case "delete":
		if strings.TrimSpace(p.Key) == "" {
			return openagent.ErrorResult(fmt.Errorf("settings delete: key is required"), false, "")
		}
		if err := config.DeleteSetting(p.Key); err != nil {
			return openagent.ErrorResult(fmt.Errorf("settings delete: %w", err), true, "")
		}
		return &openagent.ToolResult{Content: fmt.Sprintf("deleted %s. Call reload to apply.", p.Key)}
	case "validate":
		report, err := config.ValidateSettings()
		if err != nil {
			return openagent.ErrorResult(fmt.Errorf("settings validate: %w", err), false, "")
		}
		var msg string
		if len(report.Warnings) == 0 && len(report.EnumViolations) == 0 {
			msg = "settings.json is valid (no warnings, no violations)"
		} else {
			parts := []string{"settings.json validation report:"}
			if len(report.Warnings) > 0 {
				parts = append(parts, fmt.Sprintf("%d unset env var(s): %s", len(report.Warnings), strings.Join(report.Warnings, ", ")))
			}
			if len(report.EnumViolations) > 0 {
				parts = append(parts, fmt.Sprintf("%d enum violation(s): %s", len(report.EnumViolations), strings.Join(report.EnumViolations, "; ")))
			}
			msg = strings.Join(parts, " ")
		}
		return &openagent.ToolResult{Content: msg}
	case "reload":
		fn := loadSettingsReloadFn()
		if fn == nil {
			return openagent.ErrorResult(fmt.Errorf("settings reload: no server running (reload is only available when a server is live)"), false, "")
		}
		result := fn(ctx)
		var parts []string
		if result.ParseError != "" {
			parts = append(parts, "reload FAILED: "+result.ParseError)
			parts = append(parts, "previous config kept")
		} else if len(result.Violations) > 0 {
			parts = append(parts, "reload BLOCKED by validation violations:")
			parts = append(parts, result.Violations...)
			parts = append(parts, "previous config kept")
		} else {
			parts = append(parts, "reload OK. Applied:")
			parts = append(parts, result.Applied...)
		}
		return &openagent.ToolResult{Content: strings.Join(parts, "\n")}
	case "schema":
		return &openagent.ToolResult{Content: settingsSchemaDoc}
	default:
		return openagent.ErrorResult(fmt.Errorf("settings: unknown action %q (want set|get|list|append|delete|validate|reload|schema)", action), false, "")
	}
}

// newSettingsTool returns the settings tool. It is server-level (not
// scoped to a session cwd) because settings.json is a global file.
// The reload action uses the shared settingsReloadFn (set at startup).
func newSettingsTool() openagent.Tool {
	return &settingsTool{}
}

// isSecretKey reports whether the dotted key path targets a field tagged
// `sensitive:"true"` in the Config struct tree. Used to prompt the agent to
// use ${ENV_VAR} instead of a literal value, keeping secrets off disk.
// Tag-driven: adding `sensitive:"true"` to a new field automatically makes
// it recognized here — no edits to this file needed.
func isSecretKey(key string) bool {
	return config.IsSecretKey(key)
}

// looksLikeEnvRef reports whether the value is an env-var reference
// (${VAR}, ${VAR:-default}, $VAR) rather than a literal. Used to skip the
// "use ${ENV_VAR}" tip when the agent is already using one.
func looksLikeEnvRef(value string) bool {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "${") || strings.HasPrefix(trimmed, "$") {
		// Distinguish $VAR from a literal $ — check the char after $.
		if trimmed == "$" {
			return false
		}
		next := trimmed[1]
		return next == '{' || (next >= 'A' && next <= 'Z') || (next >= 'a' && next <= 'z') || next == '_'
	}
	return false
}

// logLocationClause returns the log-file sentence for the tool description,
// using the path captured at SetupLog time. Empty path means logging is
// discarded (no log file configured).
func logLocationClause() string {
	p := logFilePath.Load()
	if p == nil || *p == "" {
		return "Logging is currently discarded (no log.file configured)."
	}
	return fmt.Sprintf("The log file is %s — read it with the read tool to inspect server output.", *p)
}

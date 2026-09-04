package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yusheng-g/openagent-go/version"
)

type Config struct {
	// Model is the default model ("<provider>/<modelID>") used when a
	// session has not selected one — overrides the first-registered
	// fallback.
	Model        string                     `json:"model,omitempty"`
	FastModel    string                     `json:"fast_model,omitempty"`
	Provider     map[string]ProviderConfig  `json:"provider,omitempty"`
	Server       ServerConfig               `json:"server,omitempty"`
	Channels     ChannelsConfig             `json:"channels,omitempty"`
	Sandbox      SandboxConfig              `json:"sandbox,omitempty"`
	Log          LogConfig                  `json:"log,omitempty"`
	McpServers   map[string]McpServerConfig `json:"mcp_servers,omitempty"`
	Plugins      []string                   `json:"plugins,omitempty"`
	Env          map[string]string          `json:"env,omitempty"`
	Sensitive    SensitiveConfig            `json:"sensitive,omitempty"`
	Capabilities Capabilities               `json:"capabilities,omitempty"`
	Embedding    EmbeddingConfig            `json:"embedding,omitempty"`
	// DefaultMode is the session mode new sessions start in ("auto",
	// "manual", "plan"). Empty = "manual" (approval-based safe default).
	DefaultMode string `json:"default_mode,omitempty" valid:"enum=auto|manual|plan;case=cs"`
	// ContextProviders overrides the backend per capability. A non-empty
	// OpenViking.Endpoint already switches ALL domains to OpenViking (it
	// is a whole-context service); set a domain to "builtin" here to keep
	// the local backend for it. Empty = no override.
	ContextProviders ContextProviderConfig `json:"context_providers,omitempty"`
	// OpenViking holds the connection settings for the OpenViking
	// backend. A configured endpoint enables OpenViking for memory,
	// skills, and resources (per-domain opt-out via ContextProviders).
	OpenViking OpenVikingConfig `json:"openviking,omitempty"`
	// Telemetry configures OpenTelemetry trace export. When Endpoint is
	// non-empty, agent runs and tool calls are sent as OTel spans to the
	// OTLP collector (Jaeger, Tempo, Datadog, Langfuse, Phoenix, ...).
	Telemetry TelemetryConfig `json:"telemetry,omitempty"`
	// TUI configures the interactive TUI client (openagent tui). When
	// unset, the TUI uses built-in defaults.
	TUI TUIConfig `json:"tui,omitempty"`
}

// TUIConfig configures the interactive TUI client. All fields optional;
// empty fields keep the built-in defaults.
type TUIConfig struct {
	// Mode is the initial session mode ("auto"|"manual"|"plan"). Empty
	// falls back to DefaultMode, then "manual" (ApplyDefaults resolves).
	Mode string `json:"mode,omitempty" valid:"enum=auto|manual|plan;case=cs;skipif=default_mode"`
	// Suggestions overrides the welcome-page placeholder suggestion list.
	// Empty = built-in defaults.
	Suggestions []string `json:"suggestions,omitempty"`
	// Colors overrides the theme palette. Fields left empty keep the
	// built-in default. Hex strings, e.g. "#0a0a0a".
	Colors TUIColors `json:"colors,omitempty"`
	// Logo overrides the welcome-page logo. A multi-line string (newline-
	// separated); empty = built-in default block-art logo.
	Logo string `json:"logo,omitempty"`
	// LogoGradient, when non-empty, renders the logo with a vertical color
	// gradient (top→bottom) interpolated across the listed hex stops, e.g.
	// ["#ffffff","#d0d0d0"]. Empty = single LogoColor (or the default white
	// gradient).
	LogoGradient []string `json:"logo_gradient,omitempty"`
}

// TUIColors overrides individual theme palette entries. Every field is
// optional; an empty string keeps the built-in default for that color.
type TUIColors struct {
	BgNormal    string `json:"bg_normal,omitempty"`
	BgSecondary string `json:"bg_secondary,omitempty"`
	BgSurface   string `json:"bg_surface,omitempty"`
	Primary     string `json:"primary,omitempty"` // plan mode badge / 蓝
	Success     string `json:"success,omitempty"` // manual mode badge / 绿
	Warning     string `json:"warning,omitempty"` // auto mode badge / 黄
	Danger      string `json:"danger,omitempty"`
	TextNormal  string `json:"text_normal,omitempty"`
	TextAsh     string `json:"text_ash,omitempty"`
	BorderGray  string `json:"border_gray,omitempty"`
	LogoColor   string `json:"logo_color,omitempty"` // welcome-page logo
}

// TelemetryConfig configures OpenTelemetry trace export.
type TelemetryConfig struct {
	// Endpoint is the OTLP collector target. Accepts either a bare
	// "host:port" (recommended, e.g. "localhost:4318" for HTTP,
	// "localhost:4317" for gRPC — the scheme is derived from Protocol +
	// Insecure) or a full URL with scheme (e.g.
	// "http://localhost:4318" or "https://collector.example:4318/otlp").
	// When empty, telemetry is disabled.
	//
	// Do NOT pass a full URL when also setting Protocol/Insecure for a
	// bare endpoint — pick one form. The two are mutually exclusive in
	// intent: a bare host:port pairs with Protocol/Insecure; a full URL
	// carries its own scheme/path/TLS.
	Endpoint string `json:"endpoint,omitempty"`
	// Protocol selects the OTLP transport: "http" (default) or "grpc".
	// Ignored when Endpoint is a full URL whose scheme implies the
	// transport (http:// → HTTP, the gRPC exporter still uses host:port
	// form in practice — prefer bare host:port for gRPC).
	Protocol string `json:"protocol,omitempty" valid:"enum=http|grpc;case=ci"`
	// ServiceName is the OTel resource service.name attribute.
	// Default: "openagent".
	ServiceName string `json:"service_name,omitempty"`
	// Insecure disables TLS when the endpoint is plain HTTP. Default:
	// true (most local collectors use plain HTTP). Set false for a
	// TLS-secured collector. Has no effect when Endpoint is a full URL
	// — the URL's own scheme (http:// vs https://) governs TLS.
	Insecure *bool `json:"insecure,omitempty"`
}

// ContextProviderConfig overrides the backend for each context capability.
// Empty value = follow the endpoint default ("openviking" when
// OpenViking.Endpoint is set, "builtin" otherwise).
type ContextProviderConfig struct {
	Memory   string `json:"memory,omitempty"` // "" | "builtin" | "openviking"
	Skill    string `json:"skill,omitempty"`
	Resource string `json:"resource,omitempty"`
}

// OpenVikingConfig connects to an OpenViking server (direct HTTP API —
// search/remember/read, no SDK).
type OpenVikingConfig struct {
	Endpoint string       `json:"endpoint,omitempty"`                 // e.g. "http://127.0.0.1:1933"
	APIKey   string       `json:"api_key,omitempty" sensitive:"true"` // Bearer token; empty = no auth
	Recall   RecallConfig `json:"recall,omitempty"`
}

// RecallConfig controls OpenViking's type-quota memory recall endpoint
// (POST /api/v1/search/recall). The endpoint searches memory subtrees
// independently by type then renders a bounded context block — without
// quotas the recall can return empty or irrelevant results.
//
// Quotas caps how many items each memory type contributes. A zero value
// disables that type. Keys: "events", "entities", "preferences",
// "experiences". Empty = server defaults (events=10, entities=10,
// preferences=3, experiences=0).
//
// MaxChars bounds the rendered output size. MinScore filters low-relevance
// hits. Both empty = server defaults (6500 / 0.1).
type RecallConfig struct {
	Quotas   map[string]int `json:"quotas,omitempty"`
	MaxChars int            `json:"max_chars,omitempty"`
	MinScore float64        `json:"min_score,omitempty"`
}

// EmbeddingConfig selects the semantic-embedding backend for knowledge
// recall. When empty (or Provider == ""), NO embedding backend is wired:
// the knowledge store stays open (memory CRUD + keyword search work) but
// semantic vector recall is disabled. Set provider/base_url/model/api_key
// to call an OpenAI-compatible /embeddings API (OpenAI, Ollama, Jina,
// Cohere, or a local proxy) and enable vector recall. There is no embedded
// model — the default build is pure Go (CGO_ENABLED=0).
type EmbeddingConfig struct {
	Provider string `json:"provider,omitempty"` // "openai" (OpenAI-compatible /embeddings)
	Model    string `json:"model,omitempty"`    // e.g. "text-embedding-3-small"
	BaseURL  string `json:"base_url,omitempty"` // e.g. "https://api.openai.com/v1"
	APIKey   string `json:"api_key,omitempty" sensitive:"true"`
}

// SensitiveConfig controls redaction of sensitive values in tool results.
//
// Env is a list of environment-variable *names* (not values) whose values,
// when present in a tool *result* (output) or error, are replaced with
// "[REDACTED]" and flagged with a hint telling the model not to reconstruct
// them. Values are resolved lazily via os.Getenv at redaction time, so
// secrets never live in this struct and envvars set after agent construction
// are still honored.
//
// Scope limits: this redacts ONLY tool results/errors, NOT tool call
// arguments (args). Only the exact value of a named env var is matched —
// no regex, so unregistered secrets, PII, keys, etc. are not detected.
// Values shorter than 8 characters are skipped (they are rarely real
// secrets and would mis-mask normal output).
// See the hooks/redact package doc for the full coverage scope and known
// limitations (e.g. JSON-escaped secret values are not matched).
type SensitiveConfig struct {
	Env []string `json:"env,omitempty"`
}

type ProviderConfig struct {
	APIKey  string        `json:"api_key" sensitive:"true"`
	BaseURL string        `json:"base_url"`
	Models  []ModelConfig `json:"models,omitempty"`
}

// ModelConfig is one entry of a provider's models array. Accepts either a
// plain string ("gpt-4o") or an object with per-model capabilities and
// pricing (litellm-shaped fields):
//
//	"models": ["gpt-4o", {
//	  "id": "qwen-128k",
//	  "max_input_tokens": 128000,
//	  "max_output_tokens": 8192,
//	  "input_cost_per_million": 1,
//	  "input_cache_cost_per_million": 0.1,
//	  "output_cost_per_million": 2
//	}]
//
// max_input_tokens overrides the built-in vendor lookup — required for
// custom providers serving quantized/shrunk models whose real window is
// smaller than the table (a declared 1M with an actual 128K would fail
// requests past 128K with no diagnostics). The cost fields feed usage
// reporting (ACP usage_update cost). 0/absent = built-in lookup / no cost.
type ModelConfig struct {
	ID                       string  `json:"id"`
	MaxInputTokens           int     `json:"max_input_tokens,omitempty"`
	MaxOutputTokens          int     `json:"max_output_tokens,omitempty"`
	InputCostPerMillion      float64 `json:"input_cost_per_million,omitempty"`
	InputCacheCostPerMillion float64 `json:"input_cache_cost_per_million,omitempty"`
	OutputCostPerMillion     float64 `json:"output_cost_per_million,omitempty"`
}

// UnmarshalJSON accepts both the legacy string form ("gpt-4o") and the
// object form ({"id": ..., "context_window": ...}).
func (m *ModelConfig) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		return json.Unmarshal(b, &m.ID)
	}
	type alias ModelConfig
	var a alias
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	*m = ModelConfig(a)
	return nil
}

type ServerConfig struct {
	Port int `json:"port,omitempty"`
}

// McpServerConfig describes an MCP server using the standard MCP config format
// (same as Claude Code / Claude Desktop / Cursor).
// The map key is the server name.
type McpServerConfig struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`                                      // HTTP/SSE endpoint
	Type    string            `json:"type,omitempty" valid:"enum=stdio|http|sse;case=cs"` // "stdio" (default), "http", "sse"
}

// ChannelsConfig holds per-platform IM channel configuration.
type ChannelsConfig struct {
	Feishu *FeishuConfig `json:"feishu,omitempty"`
	Wechat *WechatConfig `json:"wechat,omitempty"`
	Wecom  *WecomConfig  `json:"wecom,omitempty"`
}

// WecomConfig holds credentials for the WeCom smart-robot channel
// (official long-connection API). BotID + Secret come from the admin
// console (manual) or the official QR authorization flow (scan → robot
// created automatically).
type WecomConfig struct {
	BotID  string `json:"bot_id"`
	Secret string `json:"secret" sensitive:"true"`

	// Explicit marks a channel requested via --channel on the command
	// line (never persisted, never read from settings). Only explicit
	// channels auto-connect at startup (fail-fast when they cannot
	// start); settings credentials alone never trigger a connection —
	// the frontend connects via POST /connect.
	Explicit bool `json:"-"`
}

// WechatConfig holds credentials for the personal WeChat channel
// (ilinkai official channel). The token is the bot session credential
// issued by QR login; base_url may be redirected per-account by the
// login flow.
type WechatConfig struct {
	Token     string `json:"token" sensitive:"true"`
	BaseURL   string `json:"base_url,omitempty"`
	AccountID string `json:"account_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`

	// Explicit marks a channel requested via --channel on the command
	// line (never persisted, never read from settings). Only explicit
	// channels auto-connect at startup (fail-fast when they cannot
	// start); settings credentials alone never trigger a connection —
	// the frontend connects via POST /connect.
	Explicit bool `json:"-"`
}

// FeishuConfig holds credentials for a Feishu (Lark) App Bot.
// https://open.feishu.cn/document/home/develop-a-bot-in-5-minutes
type FeishuConfig struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret" sensitive:"true"`

	// Explicit marks a channel requested via --channel on the command
	// line (never persisted, never read from settings). Only explicit
	// channels auto-connect at startup (fail-fast when they cannot
	// start); settings credentials alone never trigger a connection —
	// the frontend connects via POST /connect.
	Explicit bool `json:"-"`
}

// SandboxConfig controls the native sandbox used by the CLI server modes.
//
// Enabled governs whether the sandbox is active:
//   - false (default) → commands run unconfined (no bwrap/seatbelt)
//   - true            → commands run inside the OS-native sandbox
//
// Network governs outbound network access from inside the sandbox
// (only effective when Enabled is true):
//   - "" or "host"     → share the host's network namespace (network allowed)
//   - "isolated"       → unshare the network namespace (no outbound network)
//
// WritablePaths / ReadablePaths are additional host paths bind-mounted
// into the sandbox (writable / read-only respectively), on top of the
// workspace directory and the system paths already mounted.
type SandboxConfig struct {
	Enabled       bool     `json:"enabled,omitempty"`
	Network       string   `json:"network,omitempty" valid:"enum=host|isolated;case=cs"`
	WritablePaths []string `json:"writable_paths,omitempty"`
	ReadablePaths []string `json:"readable_paths,omitempty"`
}

// LogConfig controls logging output.
type LogConfig struct {
	// File is the log file path. Logs are written to stderr when empty.
	File string `json:"file,omitempty"`
	// MaxSize is the maximum size in megabytes before rotation. Default: 10.
	MaxSize int `json:"max_size,omitempty"`
	// MaxBackups is the number of rotated files to keep. Default: 5.
	MaxBackups int `json:"max_backups,omitempty"`
	// MaxAge is the maximum age in days to keep rotated files. Default: 30.
	MaxAge int `json:"max_age,omitempty"`
	// Level filters log messages below this threshold. Default: "info".
	// Valid: "trace", "debug", "info", "warn", "error". "trace" enables
	// prompt dumps (every message sent to the model — content included,
	// which may contain user data and secrets).
	Level string `json:"level,omitempty" valid:"enum=trace|debug|info|warn|warning|error;case=ci"`
}

// Path returns the config file path. The default config dir is
// ~/.<version.Name> (e.g. ~/.openagent), set at build time via ldflags;
// OPENAGENT_CLI_CONFIG overrides the settings.json location entirely.
func Path() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "home dir: %v\n", err)
	}
	defaultPath := filepath.Join(home, version.ConfigDirName(), "settings.json")
	if p := os.Getenv("OPENAGENT_CLI_CONFIG"); p != "" {
		info, err := os.Stat(p)
		if info != nil && info.IsDir() {
			fmt.Fprintf(os.Stderr, "OPENAGENT_CLI_CONFIG is a dir(%v), use: %s\n", p, defaultPath)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "OPENAGENT_CLI_CONFIG err: %v\n", err)
			fmt.Fprintf(os.Stderr, "Use: %s\n", defaultPath)
		}
		return p
	}
	return defaultPath
}

// Dir returns the configuration directory — the directory that contains
// settings.json (OPENAGENT_CLI_CONFIG or the default). Every persistent
// path (profile, plugins, channel state) derives from this single root,
// so pointing OPENAGENT_CLI_CONFIG at a custom settings file relocates
// the whole working set with it.
func Dir() string {
	return filepath.Dir(Path())
}

// DefaultPluginsDir returns the default plugins directory under the
// configuration directory.
func DefaultPluginsDir() string {
	return filepath.Join(Dir(), "plugins")
}

// ApplyDefaults fills zero-value fields with the built-in defaults. It is
// the single source of defaults, shared by Load and cmd/cli's
// plugin-merged parse (which cannot call Load directly). settingsPath is
// the settings file location (used to derive the default log dir); empty
// resolves via Path().
func ApplyDefaults(cfg *Config, settingsPath string) {
	if settingsPath == "" {
		settingsPath = Path()
	}
	if cfg.Provider == nil {
		cfg.Provider = make(map[string]ProviderConfig)
	}
	if len(cfg.Plugins) == 0 {
		cfg.Plugins = []string{DefaultPluginsDir()}
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Log.File == "" {
		cfg.Log.File = filepath.Join(filepath.Dir(settingsPath), "logs", fmt.Sprintf("%s.log", version.SafeName()))
	}
	if cfg.Log.MaxSize == 0 {
		cfg.Log.MaxSize = 10
	}
	if cfg.Log.MaxBackups == 0 {
		cfg.Log.MaxBackups = 5
	}
	if cfg.Log.MaxAge == 0 {
		cfg.Log.MaxAge = 30
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	// TUI mode resolution: tui.mode → default_mode → "manual". Mirrors
	// acp/server.go defaultMode() so the TUI and server agree on the
	// safe default when neither is set.
	if cfg.TUI.Mode == "" {
		cfg.TUI.Mode = cfg.DefaultMode
	}
	if cfg.TUI.Mode == "" {
		cfg.TUI.Mode = "manual"
	}
}

func Load(path string) (*Config, error) {
	p := path
	if p == "" {
		p = Path()
	}
	cfg := &Config{}
	ApplyDefaults(cfg, p)

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read settings: %w", err)
	}
	// Resolve environment-variable references (${VAR}, ${VAR:-default},
	// $VAR) in the raw JSON bytes before unmarshaling. One byte-level pass
	// covers every string/int/bool field and keeps the on-disk file literal.
	// Warnings (unset vars referenced without a default) are suppressed here
	// — Load is a library/test entry point; callers that want to surface them
	// (startup, reload) call ExpandBytes themselves.
	data, _ = ExpandBytes(data)
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse settings: %w", err)
	}
	return cfg, nil
}

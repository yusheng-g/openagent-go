package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Model      string                     `json:"model,omitempty"`
	FastModel  string                     `json:"fast_model,omitempty"`
	Provider   map[string]ProviderConfig  `json:"provider,omitempty"`
	Server     ServerConfig               `json:"server,omitempty"`
	Channels   ChannelsConfig             `json:"channels,omitempty"`
	Sandbox    SandboxConfig              `json:"sandbox,omitempty"`
	Log        LogConfig                  `json:"log,omitempty"`
	McpServers map[string]McpServerConfig `json:"mcp_servers,omitempty"`
	Plugins    []string                   `json:"plugins,omitempty"`
	Profiles   string                     `json:"profiles,omitempty"`
	Env        map[string]string          `json:"env,omitempty"`
}

type ProviderConfig struct {
	APIKey  string   `json:"api_key"`
	BaseURL string   `json:"base_url"`
	Models  []string `json:"models,omitempty"`
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
	URL     string            `json:"url,omitempty"`  // HTTP/SSE endpoint
	Type    string            `json:"type,omitempty"` // "stdio" (default), "http", "sse"
}

// ChannelsConfig holds per-platform IM channel configuration.
type ChannelsConfig struct {
	Feishu *FeishuConfig `json:"feishu,omitempty"`
}

// FeishuConfig holds credentials for a Feishu (Lark) App Bot.
// https://open.feishu.cn/document/home/develop-a-bot-in-5-minutes
type FeishuConfig struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
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
	Network       string   `json:"network,omitempty"`
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
	// Valid: "debug", "info", "warn", "error".
	Level string `json:"level,omitempty"`
}

// Path returns the config file path. Respects OPENAGENT_CLI_CONFIG env var.
func Path() (string, error) {
	if p := os.Getenv("OPENAGENT_CLI_CONFIG"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".openagent", "settings.json"), nil
}

// DefaultPluginsDir returns the default plugins directory.
func DefaultPluginsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openagent", "plugins")
}

func Load(path string) (*Config, error) {
	p, _ := Path()
	if p == "" {
		home, _ := os.UserHomeDir()
		p = filepath.Join(home, ".openagent", "settings.json")
	}
	cfg := &Config{
		Provider: make(map[string]ProviderConfig),
		Plugins:  []string{DefaultPluginsDir()},
		Profiles: ".openagent/profile",
		Server:   ServerConfig{Port: 8080},
		Log: LogConfig{
			File:       filepath.Join(filepath.Dir(p), "logs", "openagent.log"),
			MaxSize:    10,
			MaxBackups: 5,
			MaxAge:     30,
			Level:      "info",
		},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read settings: %w", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse settings: %w", err)
	}
	return cfg, nil
}

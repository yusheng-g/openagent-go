package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Model     string                    `json:"model,omitempty"`
	FastModel string                    `json:"fast_model,omitempty"`
	Provider  map[string]ProviderConfig `json:"provider,omitempty"`
	Server    ServerConfig              `json:"server,omitempty"`
	Channels  ChannelsConfig            `json:"channels,omitempty"`
	Sandbox   SandboxConfig             `json:"sandbox,omitempty"`
	Plugins   []string                  `json:"plugins,omitempty"`
	Profiles  string                    `json:"profiles,omitempty"`
	Env       map[string]string         `json:"env,omitempty"`
}

type ProviderConfig struct {
	APIKey  string   `json:"api_key"`
	BaseURL string   `json:"base_url"`
	Models  []string `json:"models,omitempty"`
}

type ServerConfig struct {
	Port int `json:"port,omitempty"`
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
// Network governs outbound network access from inside the sandbox:
//   - "" or "host"     → share the host's network namespace (network allowed)
//   - "isolated"       → unshare the network namespace (no outbound network)
//
// WritablePaths / ReadablePaths are additional host paths bind-mounted
// into the sandbox (writable / read-only respectively), on top of the
// workspace directory and the system paths already mounted.
type SandboxConfig struct {
	Network       string   `json:"network,omitempty"`
	WritablePaths []string `json:"writable_paths,omitempty"`
	ReadablePaths []string `json:"readable_paths,omitempty"`
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
	cfg := &Config{}
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
	applyDefaults(cfg)
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Provider == nil {
		cfg.Provider = make(map[string]ProviderConfig)
	}
	if len(cfg.Plugins) == 0 {
		cfg.Plugins = []string{DefaultPluginsDir()}
	}
	if cfg.Profiles == "" {
		cfg.Profiles = ".openagent/profile"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
}

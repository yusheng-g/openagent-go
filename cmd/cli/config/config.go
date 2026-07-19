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
	Plugins   []string                  `json:"plugins,omitempty"`
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
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
}

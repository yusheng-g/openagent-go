package wasm

import "strings"

// ── CLI Plugin ABI ──
//
// Every CLI plugin exports:
//   alloc(size) → ptr
//   metadata() → uint64(ptr, len)   — CLIPluginMeta JSON
//
// Settings-injection plugins also export:
//   init(settings_ptr, settings_len) → uint64(ptr, len) — merged settings
//
// Command-injection plugins also export:
//   commands() → uint64(ptr, len) — []CommandDef JSON
//   run_<name>(args_ptr, args_len) → uint64(ptr, len) — output text
//
// Plugin may import from host: keyring_get, keyring_set, http_request.

const PluginCLIPrefix = "cli:"

type CLIPluginMeta struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (m CLIPluginMeta) Is(kind string) bool {
	target := "cli:" + kind
	for _, part := range strings.Split(m.Type, ",") {
		if part == target {
			return true
		}
	}
	return false
}

type CommandDef struct {
	Name  string `json:"name"`
	Use   string `json:"use"`
	Short string `json:"short"`
	Long  string `json:"long,omitempty"`
}

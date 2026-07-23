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
//   commands() → uint64(ptr, len) — []CommandDef JSON (tree with children)
//   run_<name>(input_ptr, input_len) → uint64(ptr, len) — output text
//     <name> is the leaf command's name with dashes replaced by underscores.
//     input is JSON: {"args": [...], "flags": {...}}
//
// Plugin may import from host: keyring_get/set/delete, http_request,
// fs_read, fs_write, fs_readdir, log_info/warn/error, utc_now.

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

// CommandDef describes a single CLI command or command group.
// Commands with Children are groups (no RunE); leaves have RunE.
type CommandDef struct {
	Name     string        `json:"name"`
	Use      string        `json:"use"`
	Short    string        `json:"short"`
	Long     string        `json:"long,omitempty"`
	Args     string        `json:"args,omitempty"`     // arg rule: "exact=2", "min=1", "max=3", "range=1,5", "" = any
	Flags    []FlagDef     `json:"flags,omitempty"`
	Children []CommandDef  `json:"children,omitempty"`  // non-empty = group node, no RunE
	Aliases  []string      `json:"aliases,omitempty"`
	Example  string        `json:"example,omitempty"`
}

// FlagDef describes a command-line flag for a leaf command.
type FlagDef struct {
	Name         string `json:"name"`
	Short        string `json:"short,omitempty"`
	Kind         string `json:"kind"`          // "string" | "bool" | "int"
	DefaultValue string `json:"default_value"`
	Description  string `json:"description"`
}

// CommandInput is the JSON payload passed to run_<name>().
// It carries positional args and parsed flag values.
type CommandInput struct {
	Args  []string       `json:"args"`
	Flags map[string]any `json:"flags"`
}

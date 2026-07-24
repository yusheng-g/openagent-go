// Package wasm provides a runtime plugin system via WASM modules.
// Plugins are loaded from a directory and exposed as standard openagent
// interfaces: Tool plugins → []Tool, Stage plugins → RunObserver.
//
// The plugin system is itself pluggable: if the user never creates a Manager
// or specifies no plugin directory, the system is inert.
//
//	// Without plugins: nothing changes.
//	agent := openagent.NewAgent("bot", openagent.WithModel(model))
//
//	// With plugins:
//	mgr := wasm.NewManager("./plugins")
//	mgr.Discover(ctx)
//	agent := openagent.NewAgent("bot",
//	    openagent.WithModel(model),
//	    openagent.WithTools(mgr.Tools()...),
//	    openagent.WithRunObserver(mgr.Observer()),
//	)
package wasm

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/plugin/wasmhost"
)

// ── Plugin metadata (returned by guest's metadata() export) ──

// PluginMeta is the JSON metadata blob every .wasm module exports via metadata().
type PluginMeta struct {
	Type        string          `json:"type"`                 // "agent:tools", "agent:observers", "agent:sessions"
	Name        string          `json:"name"`                 // unique name
	Description string          `json:"description"`          // human-readable
	Parameters  json.RawMessage `json:"parameters,omitempty"` // tools: JSON Schema
	Stage       string          `json:"stage,omitempty"`      // observers: which stage
	Phase       string          `json:"phase,omitempty"`      // observers: "enter" | "leave" | "*"
}

// ── Stage input/output ──

// StageInput is passed to observers plugins' run().
type StageInput struct {
	Name   string         `json:"name"`             // stage constant e.g. "model.call"
	Phase  string         `json:"phase"`            // "enter" or "leave"
	Detail map[string]any `json:"detail,omitempty"` // optional metadata
	Error  string         `json:"error,omitempty"`  // non-empty if stage failed
}

// StageOutput is returned from stage plugins' run().
type StageOutput struct {
	Action string `json:"action"` // "continue" or "abort"
	Reason string `json:"reason,omitempty"`
}

// ── Tool input/output ──

// ToolInput is passed to tool plugins' execute().
type ToolInput struct {
	Args json.RawMessage `json:"args"`
}

// ToolOutput is returned from tool plugins' execute().
type ToolOutput struct {
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// ── Plugin type constants ──

// PluginAgentPrefix is the type prefix for agent-level plugins ("agent:tools", "agent:observers").
// CLI plugins use "cli:" prefix. See [PluginTypeTools] and [PluginTypeObservers].
const PluginAgentPrefix = "agent:"

const (
	PluginTypeSessions  = PluginAgentPrefix + "sessions"
	PluginTypeTools     = PluginAgentPrefix + "tools"
	PluginTypeObservers = PluginAgentPrefix + "observers"
)

// Stage name constants for agent:sessions plugins.
const (
	StageSessionInit    = "init"
	StageSessionDestroy = "destroy"
)

// ── Stage name constants (match openagent.StageXxx) ──

const (
	StageMemoryFetch  = "memory.fetch"
	StageGuardIn      = "guard.in"
	StagePromptBuild  = "prompt.build"
	StageModelCall    = "model.call"
	StageGuardOut     = "guard.out"
	StageToolExecute  = "tool.execute"
	StageMemoryAppend = "memory.append"
)

// ── Stage phase constants ──

const (
	PhaseEnter = "enter"
	PhaseLeave = "leave"
	PhaseAll   = "*"
)

// ── Stage action constants ──

const (
	ActionContinue = "continue"
	ActionAbort    = "abort"
)

// ── Session lifecycle types ──

// SessionCtx is the JSON input to agent:sessions plugin's session_init and session_destroy exports.
type SessionCtx struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id,omitempty"`
}

// SessionConfig is the JSON output from agent:sessions plugin's session_init, specifying how the Agent
// should be configured for this session. Only non-zero / non-empty fields
// override the handler's default opts.
type SessionConfig struct {
	SystemPrompts    []string `json:"system_prompts,omitempty"`
	Description      string   `json:"description,omitempty"`
	Tools            []string `json:"tools,omitempty"`
	MaxTurns         int      `json:"max_turns,omitempty"`
	MaxWorkingTokens int      `json:"max_working_tokens,omitempty"`
	SkillDir         string   `json:"skill_dir,omitempty"`
	MemoryPath       string   `json:"memory_path,omitempty"`
}

// ── Agent runtime ──

// BuildAgentRuntime constructs a wasmhost.AgentRuntime backed by the given
// openagent Agent and Session. The Get/Set closures directly read/write
// Agent and Session fields. setModel is called by runtime_set_model_config
// to replace a model in the global registry; it may be nil.
func BuildAgentRuntime(agent *openagent.Agent, session *openagent.Session, setModel func(provider, modelID, apiKey, baseURL string)) *wasmhost.AgentRuntime {
	return &wasmhost.AgentRuntime{
		SetModel: setModel,
		Get: func(key string) (string, bool) {
			switch key {
			case wasmhost.RuntimeKeySessionID:
				return session.ID, true
			case wasmhost.RuntimeKeyUserID:
				return session.UserID, true
			case wasmhost.RuntimeKeyTurnCount:
				return fmt.Sprint(session.Turn), true
			case wasmhost.RuntimeKeyModelID:
				return session.ModelID, true
			case wasmhost.RuntimeKeyProvider:
				return session.Provider, true
			default:
				if strings.HasPrefix(key, wasmhost.RuntimeKeyMetadataPrefix) {
					k := strings.TrimPrefix(key, wasmhost.RuntimeKeyMetadataPrefix)
					v, ok := session.Metadata[k]
					if !ok {
						return "", false
					}
					s, _ := v.(string)
					return s, true
				}
				return "", false
			}
		},
		Set: func(key string, value string) error {
			switch key {
			case wasmhost.RuntimeKeyModelID:
				session.ModelID = value
			case "system_prompts":
				return json.Unmarshal([]byte(value), &agent.SystemPrompts)
			case "max_turns":
				n, err := strconv.Atoi(value)
				if err != nil {
					return err
				}
				agent.MaxTurns = n
			default:
				if strings.HasPrefix(key, wasmhost.RuntimeKeyMetadataPrefix) {
					k := strings.TrimPrefix(key, wasmhost.RuntimeKeyMetadataPrefix)
					if session.Metadata == nil {
						session.Metadata = make(map[string]any)
					}
					session.Metadata[k] = value
					return nil
				}
				return fmt.Errorf("unknown runtime key: %s", key)
			}
			return nil
		},
	}
}

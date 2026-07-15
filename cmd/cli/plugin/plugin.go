// Package plugin provides a WASM plugin system for the CLI layer.
// It is independent of the core openagent plugin/wasm package — the two
// do not share types or state. This keeps CLI extensions (tool plugins,
// model provider plugins, etc.) decoupled from the agent runtime.
//
// Plugin ABI (guest exports):
//
//	metadata() → packed(ptr, len)  — JSON PluginMeta
//	alloc(size) → ptr
//	execute(ptr, len) → packed(ptr, len)  — JSON output
//
// The metadata type field determines how execute's output is interpreted:
//   - "tool":           PluginToolOutput
//   - "model_provider": PluginModelProviderOutput
package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	openagent "github.com/yusheng-g/openagent-go"
)

// ── Plugin metadata ──

// PluginMeta is the JSON metadata blob every .wasm module exports via metadata().
type PluginMeta struct {
	Type        string          `json:"type"`                 // "tool" or "model_provider"
	Name        string          `json:"name"`                 // unique name
	Description string          `json:"description"`          // human-readable
	Parameters  json.RawMessage `json:"parameters,omitempty"` // tool: JSON Schema
}

const (
	PluginTypeTool          = "tool"
	PluginTypeModelProvider = "model_provider"
)

// ── Tool plugin I/O ──

// PluginToolInput is passed to tool plugins' execute().
type PluginToolInput struct {
	Args json.RawMessage `json:"args"`
}

// PluginToolOutput is returned from tool plugins' execute().
type PluginToolOutput struct {
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// ── Model provider plugin I/O ──

// PluginModelProviderOutput is returned from model_provider plugins' execute().
type PluginModelProviderOutput struct {
	Models []PluginModelEntry `json:"models"`
	Error  string             `json:"error,omitempty"`
}

// PluginModelEntry represents a single model resource provided by a plugin.
type PluginModelEntry struct {
	ApiKey  string `json:"apiKey"`
	BaseUrl string `json:"baseUrl"`
	ModelId string `json:"modelId"`
	Label   string `json:"label"`
}

// ── WASM module wrapper ──

type module struct {
	mod api.Module
}

func (m *module) call(ctx context.Context, fnName string, ptr, length uint32) (uint32, uint32, error) {
	fn := m.mod.ExportedFunction(fnName)
	if fn == nil {
		return 0, 0, fmt.Errorf("plugin missing export %q", fnName)
	}
	results, err := fn.Call(ctx, uint64(ptr), uint64(length))
	if err != nil {
		return 0, 0, fmt.Errorf("%s: %w", fnName, err)
	}
	if len(results) < 1 {
		return 0, 0, fmt.Errorf("%s: no result", fnName)
	}
	packed := results[0]
	return uint32(packed >> 32), uint32(packed & 0xFFFFFFFF), nil
}

func (m *module) alloc(ctx context.Context, size uint32) (uint32, error) {
	fn := m.mod.ExportedFunction("alloc")
	if fn == nil {
		return 0, fmt.Errorf("plugin missing export alloc")
	}
	results, err := fn.Call(ctx, uint64(size))
	if err != nil {
		return 0, fmt.Errorf("alloc: %w", err)
	}
	return uint32(results[0]), nil
}

func (m *module) invoke(ctx context.Context, fnName string, input []byte) ([]byte, error) {
	buf, err := m.alloc(ctx, uint32(len(input)))
	if err != nil {
		return nil, err
	}
	if !m.mod.Memory().Write(buf, input) {
		return nil, fmt.Errorf("%s: write out of bounds", fnName)
	}
	ptr, length, err := m.call(ctx, fnName, buf, uint32(len(input)))
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return nil, nil
	}
	data, ok := m.mod.Memory().Read(ptr, length)
	if !ok {
		return nil, fmt.Errorf("%s: read result out of bounds (%d, %d)", fnName, ptr, length)
	}
	out := make([]byte, length)
	copy(out, data)
	return out, nil
}

func (m *module) parseMeta(ctx context.Context) (PluginMeta, error) {
	fn := m.mod.ExportedFunction("metadata")
	if fn == nil {
		return PluginMeta{}, fmt.Errorf("plugin missing export metadata")
	}
	results, err := fn.Call(ctx)
	if err != nil {
		return PluginMeta{}, fmt.Errorf("metadata: %w", err)
	}
	if len(results) < 1 {
		return PluginMeta{}, fmt.Errorf("metadata: no result")
	}
	packed := results[0]
	ptr := uint32(packed >> 32)
	length := uint32(packed & 0xFFFFFFFF)

	data, ok := m.mod.Memory().Read(ptr, length)
	if !ok {
		return PluginMeta{}, fmt.Errorf("metadata: read out of bounds (%d, %d)", ptr, length)
	}
	out := make([]byte, length)
	copy(out, data)

	var meta PluginMeta
	if err := json.Unmarshal(out, &meta); err != nil {
		return PluginMeta{}, fmt.Errorf("parse metadata: %w", err)
	}
	return meta, nil
}

// ── Plugin adapter types ──

// cliTool adapts a WASM tool plugin to openagent.Tool.
type cliTool struct {
	mod  *module
	meta PluginMeta
}

var _ openagent.Tool = (*cliTool)(nil)

func (t *cliTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        t.meta.Name,
		Description: t.meta.Description,
		Parameters:  t.meta.Parameters,
	}
}

func (t *cliTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	input, err := json.Marshal(PluginToolInput{Args: args})
	if err != nil {
		return "", fmt.Errorf("plugin tool %q: marshal input: %w", t.meta.Name, err)
	}
	output, err := t.mod.invoke(ctx, "execute", input)
	if err != nil {
		return "", fmt.Errorf("plugin tool %q: %w", t.meta.Name, err)
	}
	var out PluginToolOutput
	if err := json.Unmarshal(output, &out); err != nil {
		return "", fmt.Errorf("plugin tool %q: parse output: %w", t.meta.Name, err)
	}
	if out.Error != "" {
		return "", fmt.Errorf("plugin tool %q: %s", t.meta.Name, out.Error)
	}
	return out.Result, nil
}

// cliModelProvider adapts a WASM model_provider plugin.
type cliModelProvider struct {
	mod  *module
	meta PluginMeta
}

func (p *cliModelProvider) Name() string { return p.meta.Name }

func (p *cliModelProvider) Load(ctx context.Context) ([]PluginModelEntry, error) {
	output, err := p.mod.invoke(ctx, "execute", []byte("{}"))
	if err != nil {
		return nil, fmt.Errorf("plugin model_provider %q: %w", p.meta.Name, err)
	}
	var out PluginModelProviderOutput
	if err := json.Unmarshal(output, &out); err != nil {
		return nil, fmt.Errorf("plugin model_provider %q: parse output: %w", p.meta.Name, err)
	}
	if out.Error != "" {
		return nil, fmt.Errorf("plugin model_provider %q: %s", p.meta.Name, out.Error)
	}
	return out.Models, nil
}

// ── Manager ──

// Manager discovers and manages WASM plugins from a directory.
type Manager struct {
	dir            string
	runtime        wazero.Runtime
	tools          []*cliTool
	modelProviders []*cliModelProvider
}

// NewManager creates a Manager for the given plugin directory.
// Pass an empty string to create an inert Manager (no plugins loaded).
func NewManager(dir string) *Manager {
	return &Manager{dir: dir}
}

// Discover scans the plugin directory for .wasm files, instantiates each one,
// reads its metadata, and registers it by type.
// Plugins that fail to instantiate or produce invalid metadata are logged as
// warnings and skipped so that a single bad plugin does not prevent others
// from loading.
// If the directory is empty or doesn't exist, Discover is a no-op.
func (m *Manager) Discover(ctx context.Context) error {
	if m.dir == "" {
		return nil
	}

	entries, err := os.ReadDir(m.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("plugin dir: %w", err)
	}

	if m.runtime == nil {
		m.runtime = wazero.NewRuntime(ctx)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".wasm" {
			continue
		}
		path := filepath.Join(m.dir, entry.Name())
		if err := m.loadOne(ctx, path); err != nil {
			log.Printf("WARNING: skipping wasm plugin %s: %v", entry.Name(), err)
		}
	}

	return nil
}

func (m *Manager) loadOne(ctx context.Context, path string) error {
	wasmBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	cfg := wazero.NewModuleConfig().WithName(filepath.Base(path))
	mod, err := m.runtime.InstantiateWithConfig(ctx, wasmBytes, cfg)
	if err != nil {
		return fmt.Errorf("instantiate: %w", err)
	}

	wmod := &module{mod: mod}
	meta, err := wmod.parseMeta(ctx)
	if err != nil {
		return err
	}

	switch meta.Type {
	case PluginTypeTool:
		m.tools = append(m.tools, &cliTool{mod: wmod, meta: meta})
	case PluginTypeModelProvider:
		m.modelProviders = append(m.modelProviders, &cliModelProvider{mod: wmod, meta: meta})
	default:
		return fmt.Errorf("unknown plugin type %q", meta.Type)
	}

	return nil
}

// Tools returns loaded tool plugins as openagent.Tool values.
func (m *Manager) Tools() []openagent.Tool {
	tools := make([]openagent.Tool, len(m.tools))
	for i, t := range m.tools {
		tools[i] = t
	}
	return tools
}

// ModelProviders returns loaded model provider plugins.
func (m *Manager) ModelProviders() []*cliModelProvider {
	return m.modelProviders
}

// Close releases the wazero runtime.
func (m *Manager) Close() error {
	if m.runtime == nil {
		return nil
	}
	return m.runtime.Close(context.Background())
}

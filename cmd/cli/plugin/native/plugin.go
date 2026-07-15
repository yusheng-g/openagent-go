// Package native provides a Go .so plugin system where plugins are compiled with
// go build -buildmode=plugin and loaded at runtime via the standard library
// plugin package.
//
// Plugin convention:
//
// A .so plugin must export a top-level function:
//
//	func NewPlugin() native.Plugin
//
// The returned Plugin's Meta().Type determines how the host treats it:
//   - "tool":           must implement ToolPlugin (adds Execute)
//   - "model_provider": must implement ModelProviderPlugin (adds Models)
//
// Example (tool plugin):
//
//	package main
//
//	import (
//	    "context"
//	    "encoding/json"
//	    "github.com/yusheng-g/openagent-go/cmd/cli/plugin/native"
//	)
//
//	type echoTool struct{}
//
//	func (t *echoTool) Meta() native.PluginMeta {
//	    return native.PluginMeta{
//	        Type:        native.PluginTypeTool,
//	        Name:        "echo",
//	        Description: "Echoes input",
//	        Parameters:  json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}}}`),
//	    }
//	}
//
//	func (t *echoTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
//	    return string(args), nil
//	}
//
//	func NewPlugin() native.Plugin { return &echoTool{} }
//
// Build with:
//
//	go build -buildmode=plugin -o echo.so .
package native

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plugin"

	api "github.com/runzhi214/openagent-go-plugin-api"
	openagent "github.com/yusheng-g/openagent-go"
)

// ── Plugin interfaces ──

// Re-exported from the shared API module so that .so plugins compiled
// as separate modules share the same type identities.

type PluginMeta = api.PluginMeta

const (
	PluginTypeTool          = api.PluginTypeTool
	PluginTypeModelProvider = api.PluginTypeModelProvider
)

type Plugin = api.Plugin

type ToolPlugin = api.ToolPlugin

type ModelProviderPlugin = api.ModelProviderPlugin

type ModelEntry = api.ModelEntry

// ── Plugin adapters ──

// nativeTool adapts a ToolPlugin to openagent.Tool.
type nativeTool struct {
	plugin ToolPlugin
	meta   PluginMeta
}

var _ openagent.Tool = (*nativeTool)(nil)

func (t *nativeTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        t.meta.Name,
		Description: t.meta.Description,
		Parameters:  t.meta.Parameters,
	}
}

func (t *nativeTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	result, err := t.plugin.Execute(ctx, args)
	if err != nil {
		return "", fmt.Errorf("native tool %q: %w", t.meta.Name, err)
	}
	return result, nil
}

// nativeModelProvider adapts a ModelProviderPlugin.
type nativeModelProvider struct {
	plugin ModelProviderPlugin
	meta   PluginMeta
}

func (p *nativeModelProvider) Name() string { return p.meta.Name }

func (p *nativeModelProvider) Load(ctx context.Context) ([]ModelEntry, error) {
	entries, err := p.plugin.Models(ctx)
	if err != nil {
		return nil, fmt.Errorf("native model_provider %q: %w", p.meta.Name, err)
	}
	return entries, nil
}

// ── Manager ──

// Manager discovers and manages native .so plugins from a directory.
type Manager struct {
	dir            string
	tools          []openagent.Tool
	modelProviders []*nativeModelProvider
}

// NewManager creates a Manager for the given plugin directory.
// Pass an empty string to create an inert Manager.
func NewManager(dir string) *Manager {
	return &Manager{dir: dir}
}

// Discover scans the plugin directory for .so files, loads each one via
// plugin.Open, looks up the NewPlugin symbol, and registers it by type.
// Plugins that fail to load or produce invalid metadata are logged as
// warnings and skipped so that a single bad plugin does not prevent others
// from loading.
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

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".so" {
			continue
		}
		path := filepath.Join(m.dir, entry.Name())

		if err := m.loadOne(path); err != nil {
			log.Printf("WARNING: skipping native plugin %s: %v", entry.Name(), err)
		}
	}

	return nil
}

func (m *Manager) loadOne(path string) error {
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}

	sym, err := p.Lookup("NewPlugin")
	if err != nil {
		return fmt.Errorf("lookup NewPlugin: %w", err)
	}

	newPlugin, ok := sym.(func() Plugin)
	if !ok {
		return fmt.Errorf("NewPlugin has wrong type %T, expected func() native.Plugin", sym)
	}

	plg := newPlugin()
	meta := plg.Meta()

	switch meta.Type {
	case PluginTypeTool:
		tp, ok := plg.(ToolPlugin)
		if !ok {
			return fmt.Errorf("plugin type %q does not implement ToolPlugin", meta.Type)
		}
		m.tools = append(m.tools, &nativeTool{plugin: tp, meta: meta})
	case PluginTypeModelProvider:
		mp, ok := plg.(ModelProviderPlugin)
		if !ok {
			return fmt.Errorf("plugin type %q does not implement ModelProviderPlugin", meta.Type)
		}
		m.modelProviders = append(m.modelProviders, &nativeModelProvider{plugin: mp, meta: meta})
	default:
		return fmt.Errorf("unknown plugin type %q", meta.Type)
	}

	return nil
}

// Tools returns loaded tool plugins as openagent.Tool values.
func (m *Manager) Tools() []openagent.Tool {
	return m.tools
}

// ModelProviders returns loaded model provider plugins.
func (m *Manager) ModelProviders() []*nativeModelProvider {
	return m.modelProviders
}

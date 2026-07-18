package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"github.com/yusheng-g/openagent-go/plugin/wasmhost"
)

var commandModules sync.Map

type Module struct {
	Mod  api.Module
	Meta CLIPluginMeta
}

func (r *Runtime) Instantiate(ctx context.Context, wasmBytes []byte, name string) (*Module, CLIPluginMeta, error) {
	mod, err := r.rt.InstantiateWithConfig(ctx, wasmBytes,
		wazero.NewModuleConfig().WithName(name).WithSysNanosleep().WithSysNanotime())
	if err != nil {
		return nil, CLIPluginMeta{}, fmt.Errorf("instantiate: %w", err)
	}
	meta, err := readCLIMeta(ctx, mod)
	if err != nil {
		return nil, CLIPluginMeta{}, fmt.Errorf("metadata: %w", err)
	}
	if !strings.HasPrefix(meta.Type, PluginCLIPrefix) {
		return nil, CLIPluginMeta{}, fmt.Errorf("plugin type %q does not start with %q", meta.Type, PluginCLIPrefix)
	}
	return &Module{Mod: mod, Meta: meta}, meta, nil
}

func (m *Module) CallInit(ctx context.Context, settingsJSON []byte) ([]byte, error) {
	if m.Mod.ExportedFunction("init") == nil {
		return settingsJSON, nil
	}
	if len(settingsJSON) == 0 {
		return callExport(ctx, m.Mod, "init")
	}
	allocFn := m.Mod.ExportedFunction("alloc")
	if allocFn == nil {
		return nil, fmt.Errorf("alloc not exported")
	}
	allocRes, err := allocFn.Call(ctx, uint64(len(settingsJSON)))
	if err != nil || len(allocRes) == 0 {
		return nil, fmt.Errorf("alloc settings: %w", err)
	}
	ptr := uint32(allocRes[0])
	m.Mod.Memory().Write(ptr, settingsJSON)
	fn := m.Mod.ExportedFunction("init")
	if fn == nil {
		return settingsJSON, nil
	}
	results, err := fn.Call(ctx, uint64(ptr), uint64(len(settingsJSON)))
	if err != nil {
		return nil, fmt.Errorf("init: %w", err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("init: no result")
	}
	return wasmhost.ReadPacked(m.Mod, results[0]), nil
}

func (m *Module) ReadCommands(ctx context.Context) ([]CommandDef, error) {
	if m.Mod.ExportedFunction("commands") == nil {
		return nil, nil
	}
	raw, err := callExport(ctx, m.Mod, "commands")
	if err != nil {
		return nil, fmt.Errorf("commands: %w", err)
	}
	var cmds []CommandDef
	if err := json.Unmarshal(raw, &cmds); err != nil {
		return nil, fmt.Errorf("parse commands: %w", err)
	}
	for _, cd := range cmds {
		if _, loaded := commandModules.LoadOrStore(cd.Name, m.Mod); loaded {
			log.Printf("plugin %q: command %q already registered, overwritten", m.Meta.Name, cd.Name)
		}
	}
	return cmds, nil
}

func RunCommand(ctx context.Context, cmdName string, argsJSON string) (string, error) {
	v, ok := commandModules.Load(cmdName)
	if !ok {
		return "", fmt.Errorf("command %q not found", cmdName)
	}
	mod := v.(api.Module)
	fn := mod.ExportedFunction("run_" + cmdName)
	if fn == nil {
		return "", fmt.Errorf("command %q has no export run_%s", cmdName, cmdName)
	}
	allocFn := mod.ExportedFunction("alloc")
	if allocFn == nil {
		return "", fmt.Errorf("alloc not exported")
	}
	argsBytes := []byte(argsJSON)
	allocRes, err := allocFn.Call(ctx, uint64(len(argsBytes)))
	if err != nil || len(allocRes) == 0 {
		return "", fmt.Errorf("alloc: %w", err)
	}
	ptr := uint32(allocRes[0])
	mod.Memory().Write(ptr, argsBytes)
	results, err := fn.Call(ctx, uint64(ptr), uint64(len(argsBytes)))
	if err != nil {
		return "", fmt.Errorf("run_%s: %w", cmdName, err)
	}
	if len(results) == 0 {
		return "", fmt.Errorf("run_%s: no result", cmdName)
	}
	data := wasmhost.ReadPacked(mod, results[0])
	if data == nil {
		return "", fmt.Errorf("run_%s: read result failed", cmdName)
	}
	return string(data), nil
}

func readCLIMeta(ctx context.Context, mod api.Module) (CLIPluginMeta, error) {
	raw, err := callExport(ctx, mod, "metadata")
	if err != nil {
		return CLIPluginMeta{}, err
	}
	var meta CLIPluginMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return CLIPluginMeta{}, fmt.Errorf("parse metadata: %w", err)
	}
	return meta, nil
}

func callExport(ctx context.Context, mod api.Module, name string) ([]byte, error) {
	fn := mod.ExportedFunction(name)
	if fn == nil {
		return nil, fmt.Errorf("export %q not found", name)
	}
	results, err := fn.Call(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("%s: no result", name)
	}
	return wasmhost.ReadPacked(mod, results[0]), nil
}

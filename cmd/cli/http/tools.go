package http

import (
	"context"
	"log"
	"os"
	"path/filepath"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/cmd/cli/plugin"
	nativeplugin "github.com/yusheng-g/openagent-go/cmd/cli/plugin/native"
	"github.com/yusheng-g/openagent-go/sandbox/native"
	"github.com/yusheng-g/openagent-go/tool"
)

type ToolResult struct {
	Tools     []openagent.Tool
	Mgr       *plugin.Manager
	NativeMgr *nativeplugin.Manager
}

func loadTools() ToolResult {
	// ── Sandbox + tools ──
	workDir, _ := filepath.Abs(".")
	sandbox, err := native.New(workDir)
	var tools []openagent.Tool
	if err == nil {
		tools = []openagent.Tool{
			tool.NewShell(sandbox, workDir),
			tool.NewReadFile(workDir),
			tool.NewWriteFile(workDir),
			tool.NewListDir(workDir),
			tool.NewGrep(workDir),
		}
	} else {
		log.Printf("WARNING: sandbox unavailable: %v", err)
	}

	// ── WASM plugin tools ──
	pluginDir := os.Getenv("OPENAGENT_PLUGIN_DIR")
	mgr := plugin.NewManager(pluginDir)
	if pluginDir != "" {
		if err := mgr.Discover(context.Background()); err != nil {
			log.Printf("WARNING: plugin discovery failed: %v", err)
		} else {
			tools = append(tools, mgr.Tools()...)
		}
	}

	// ── Native binary plugin tools ──
	nativePluginDir := os.Getenv("OPENAGENT_NATIVE_PLUGIN_DIR")
	nativeMgr := nativeplugin.NewManager(nativePluginDir)
	if nativePluginDir != "" {
		if err := nativeMgr.Discover(context.Background()); err != nil {
			log.Printf("WARNING: native plugin discovery failed: %v", err)
		} else {
			tools = append(tools, nativeMgr.Tools()...)
		}
	}

	return ToolResult{Tools: tools, Mgr: mgr, NativeMgr: nativeMgr}
}

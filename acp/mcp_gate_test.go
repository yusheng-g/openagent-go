package acp

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	openacp "github.com/yusheng-g/openagent-go/acp/sdk"
)

// TestNewAgentServer_MCPEnabledDefault verifies the MCP gate defaults to
// enabled so existing callers (who don't set it) keep MCP behavior.
func TestNewAgentServer_MCPEnabledDefault(t *testing.T) {
	srv := NewAgentServer(nil, nil, nil, nil)
	if !srv.MCPEnabled {
		t.Error("NewAgentServer should default MCPEnabled to true")
	}
}

// TestConnectMCP_Gate verifies that MCPEnabled=false prevents any MCP
// connection attempt, while MCPEnabled=true allows it. The attempt is
// detected via a stdio "touch" side effect: a spawned process creates a
// file iff connectMCP actually invokes connectOneMCP.
func TestConnectMCP_Gate(t *testing.T) {
	touch, err := exec.LookPath("touch")
	if err != nil {
		t.Skip("touch not available; skipping MCP gate side-effect test")
	}

	srv := NewAgentServer(nil, nil, nil, nil)
	// A stdio MCP server whose "server" is just `touch <file>`. connectMCP
	// will spawn it (creating the file) then fail the MCP handshake — the
	// failure is logged and non-fatal, but the file proves the spawn.
	mkServers := func(file string) []openacp.McpServer {
		return []openacp.McpServer{{Name: "x", Command: touch, Args: []string{file}}}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ── Disabled: no spawn, no file ──
	fileOff := filepath.Join(t.TempDir(), "spawned-off")
	srv.MCPEnabled = false
	if sess, tools := srv.connectMCP(ctx, mkServers(fileOff)); sess != nil || tools != nil {
		t.Errorf("disabled connectMCP = %v, %v; want nil, nil", sess, tools)
	}
	if _, err := os.Stat(fileOff); err == nil {
		t.Error("connectMCP spawned a process despite MCPEnabled=false")
	}

	// ── Enabled: spawn happens, file created ──
	fileOn := filepath.Join(t.TempDir(), "spawned-on")
	srv.MCPEnabled = true
	srv.connectMCP(ctx, mkServers(fileOn))
	// The spawn is synchronous up to handshake, but allow a brief window
	// for the filesystem to reflect the touch.
	deadline := time.Now().Add(2 * time.Second)
	var found bool
	for time.Now().Before(deadline) {
		if _, err := os.Stat(fileOn); err == nil {
			found = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !found {
		t.Error("connectMCP did not spawn process with MCPEnabled=true (file not created)")
	}
}

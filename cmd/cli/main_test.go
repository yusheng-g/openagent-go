package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/yusheng-g/openagent-go/cmd/cli/server"
)

// newCapCmd creates a cobra.Command with capability flags and calls
// addCapabilityFlags + parseCapabilities, returning the result.
func newCapCmd(args []string) server.Capabilities {
	cmd := &cobra.Command{}
	addCapabilityFlags(cmd)
	// Cobra parses os.Args by default; override with explicit args.
	cmd.SetArgs(args)
	_ = cmd.Execute()

	var caps server.Capabilities
	parseCapabilities(cmd, &caps)
	return caps
}

func TestParseCapabilities_Defaults(t *testing.T) {
	caps := newCapCmd([]string{})
	if !caps.OnMemory() {
		t.Error("memory default: want on")
	}
	if !caps.OnSummarizer() {
		t.Error("summarizer default: want on")
	}
	if !caps.OnTools() {
		t.Error("tools default: want on")
	}
	if !caps.OnSkills() {
		t.Error("skills default: want on")
	}
	if !caps.OnMCP() {
		t.Error("mcp default: want on")
	}
	if caps.OnGuard() {
		t.Error("guard default: want off")
	}
	if caps.OnApprover() {
		t.Error("approver default: want off")
	}
	if caps.OnHooks() {
		t.Error("hooks default: want off")
	}
	if caps.OnObserver() {
		t.Error("observer default: want off")
	}
}

func TestParseCapabilities_ExplicitOnOff(t *testing.T) {
	caps := newCapCmd([]string{
		"--memory=off", "--summarizer=off", "--tools=off",
		"--skills=off", "--mcp=off",
		"--guard=on", "--approver=on", "--hooks=on", "--observer=on",
	})
	if caps.OnMemory() {
		t.Error("memory=off: want off")
	}
	if caps.OnSummarizer() {
		t.Error("summarizer=off: want off")
	}
	if caps.OnTools() {
		t.Error("tools=off: want off")
	}
	if caps.OnSkills() {
		t.Error("skills=off: want off")
	}
	if caps.OnMCP() {
		t.Error("mcp=off: want off")
	}
	if !caps.OnGuard() {
		t.Error("guard=on: want on")
	}
	if !caps.OnApprover() {
		t.Error("approver=on: want on")
	}
	if !caps.OnHooks() {
		t.Error("hooks=on: want on")
	}
	if !caps.OnObserver() {
		t.Error("observer=on: want on")
	}
}

func TestParseCapabilities_InvalidValue(t *testing.T) {
	// Invalid values should be ignored → Capabilities uses defaults.
	caps := newCapCmd([]string{
		"--memory=maybe",  // invalid → use default (on)
		"--guard=invalid", // invalid → use default (off)
	})
	if !caps.OnMemory() {
		t.Error("memory=maybe (invalid) → should fall back to default on")
	}
	if caps.OnGuard() {
		t.Error("guard=invalid → should fall back to default off")
	}
}

func TestParseCapabilities_PartialOverride(t *testing.T) {
	// Only override one flag; others stay default.
	caps := newCapCmd([]string{"--memory=off"})
	if caps.OnMemory() {
		t.Error("memory=off: want off")
	}
	// All others should use defaults.
	if !caps.OnSummarizer() {
		t.Error("summarizer default: want on")
	}
	if caps.OnGuard() {
		t.Error("guard default: want off")
	}
}

func TestParseCapabilities_CaseInsensitive(t *testing.T) {
	// "ON"/"OFF", "On"/"Off" should all be accepted (case-insensitive).
	caps := newCapCmd([]string{
		"--memory=ON", "--summarizer=On", "--tools=off",
		"--skills=OFF", "--mcp=on",
		"--guard=ON", "--approver=On", "--hooks=OFF", "--observer=Off",
	})
	if !caps.OnMemory() {
		t.Error("memory=ON → should be on")
	}
	if !caps.OnSummarizer() {
		t.Error("summarizer=On → should be on")
	}
	if caps.OnTools() {
		t.Error("tools=off → should be off")
	}
	if caps.OnSkills() {
		t.Error("skills=OFF → should be off")
	}
	if !caps.OnMCP() {
		t.Error("mcp=on → should be on")
	}
	if !caps.OnGuard() {
		t.Error("guard=ON → should be on")
	}
	if !caps.OnApprover() {
		t.Error("approver=On → should be on")
	}
	if caps.OnHooks() {
		t.Error("hooks=OFF → should be off")
	}
	if caps.OnObserver() {
		t.Error("observer=Off → should be off")
	}
}

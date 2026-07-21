package server

import (
	"testing"
)

func TestCapabilitiesDefaults(t *testing.T) {
	caps := Capabilities{}
	// Default on
	if !caps.OnMemory() {
		t.Error("Memory should default on")
	}
	if !caps.OnSummarizer() {
		t.Error("Summarizer should default on")
	}
	if !caps.OnTools() {
		t.Error("Tools should default on")
	}
	if !caps.OnSkills() {
		t.Error("Skills should default on")
	}
	if !caps.OnMCP() {
		t.Error("MCP should default on")
	}
	// Default off
	if caps.OnGuard() {
		t.Error("Guard should default off")
	}
	if caps.OnApprover() {
		t.Error("Approver should default off")
	}
	if caps.OnHooks() {
		t.Error("Hooks should default off")
	}
	if caps.OnObserver() {
		t.Error("Observer should default off")
	}
}

func TestCapabilitiesExplicit(t *testing.T) {
	on := true
	off := false
	caps := Capabilities{
		Memory:     &off,
		Summarizer: &off,
		Tools:      &off,
		Skills:     &off,
		MCP:        &off,
		Guard:      &on,
		Approver:   &on,
		Hooks:      &on,
		Observer:   &on,
	}
	if caps.OnMemory() {
		t.Error("Memory explicitly off")
	}
	if caps.OnSummarizer() {
		t.Error("Summarizer explicitly off")
	}
	if caps.OnTools() {
		t.Error("Tools explicitly off")
	}
	if caps.OnSkills() {
		t.Error("Skills explicitly off")
	}
	if caps.OnMCP() {
		t.Error("MCP explicitly off")
	}
	if !caps.OnGuard() {
		t.Error("Guard explicitly on")
	}
	if !caps.OnApprover() {
		t.Error("Approver explicitly on")
	}
	if !caps.OnHooks() {
		t.Error("Hooks explicitly on")
	}
	if !caps.OnObserver() {
		t.Error("Observer explicitly on")
	}
}

func TestCapabilitiesMixed(t *testing.T) {
	off := false
	caps := Capabilities{
		Memory: &off,
		// Guard, Skills, Hooks nil — use defaults
	}
	if caps.OnMemory() {
		t.Error("Memory overridden to off")
	}
	if !caps.OnSkills() {
		t.Error("Skills nil → should default on")
	}
	if caps.OnGuard() {
		t.Error("Guard nil → should default off")
	}
	if caps.OnHooks() {
		t.Error("Hooks nil → should default off")
	}
	if !caps.OnMCP() {
		t.Error("MCP nil → should default on")
	}
}

func TestCapabilitiesAllOff(t *testing.T) {
	off := false
	caps := Capabilities{
		Memory: &off, Summarizer: &off, Tools: &off,
		Skills: &off, MCP: &off, Guard: &off,
		Approver: &off, Hooks: &off, Observer: &off,
	}
	if caps.OnMemory() || caps.OnSummarizer() || caps.OnTools() ||
		caps.OnSkills() || caps.OnMCP() || caps.OnGuard() ||
		caps.OnApprover() || caps.OnHooks() || caps.OnObserver() {
		t.Error("All capabilities should be off")
	}
}

func TestCapabilitiesAllOn(t *testing.T) {
	on := true
	caps := Capabilities{
		Memory: &on, Summarizer: &on, Tools: &on,
		Skills: &on, MCP: &on, Guard: &on,
		Approver: &on, Hooks: &on, Observer: &on,
	}
	if !caps.OnMemory() || !caps.OnSummarizer() || !caps.OnTools() ||
		!caps.OnSkills() || !caps.OnMCP() || !caps.OnGuard() ||
		!caps.OnApprover() || !caps.OnHooks() || !caps.OnObserver() {
		t.Error("All capabilities should be on")
	}
}

func TestCapabilities_OnMethod_Table(t *testing.T) {
	tests := []struct {
		name     string
		on       func(Capabilities) bool
		field    **bool // pointer to the Capabilities field
		defaultV bool
	}{
		{"Memory", Capabilities.OnMemory, nil, true},
		{"Summarizer", Capabilities.OnSummarizer, nil, true},
		{"Tools", Capabilities.OnTools, nil, true},
		{"Skills", Capabilities.OnSkills, nil, true},
		{"MCP", Capabilities.OnMCP, nil, true},
		{"Guard", Capabilities.OnGuard, nil, false},
		{"Approver", Capabilities.OnApprover, nil, false},
		{"Hooks", Capabilities.OnHooks, nil, false},
		{"Observer", Capabilities.OnObserver, nil, false},
	}

	for _, tt := range tests {
		// nil field → default
		caps := Capabilities{}
		if got := tt.on(caps); got != tt.defaultV {
			t.Errorf("%s nil → %v, want default %v", tt.name, got, tt.defaultV)
		}

		// explicit true
		tru := true
		caps = capsWith(tt.name, &tru)
		if got := tt.on(caps); !got {
			t.Errorf("%s = true → %v, want true", tt.name, got)
		}

		// explicit false
		fals := false
		caps = capsWith(tt.name, &fals)
		if got := tt.on(caps); got {
			t.Errorf("%s = false → %v, want false", tt.name, got)
		}
	}
}

// capsWith returns a Capabilities with only the named field set to val.
func capsWith(name string, val *bool) Capabilities {
	c := Capabilities{}
	switch name {
	case "Memory":
		c.Memory = val
	case "Summarizer":
		c.Summarizer = val
	case "Tools":
		c.Tools = val
	case "Skills":
		c.Skills = val
	case "MCP":
		c.MCP = val
	case "Guard":
		c.Guard = val
	case "Approver":
		c.Approver = val
	case "Hooks":
		c.Hooks = val
	case "Observer":
		c.Observer = val
	}
	return c
}

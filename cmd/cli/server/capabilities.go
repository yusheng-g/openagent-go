package server

// Capabilities controls which pluggable modules are enabled at startup.
// Zero value means "use defaults" (some on, some off).
type Capabilities struct {
	Memory     *bool // default on
	Summarizer *bool // default on
	Tools      *bool // default on
	Skills     *bool // default on
	MCP        *bool // default on
	Guard      *bool // default off
	Approver   *bool // default off
	Hooks      *bool // default off
	Observer   *bool // default off
}

func (c Capabilities) on(field *bool, defaultOn bool) bool {
	if field != nil {
		return *field
	}
	return defaultOn
}

// OnMemory reports whether Memory is enabled.
func (c Capabilities) OnMemory() bool { return c.on(c.Memory, true) }

// OnSummarizer reports whether Summarizer is enabled.
func (c Capabilities) OnSummarizer() bool { return c.on(c.Summarizer, true) }

// OnTools reports whether built-in Tools (shell, read, write, ls, grep) are enabled.
func (c Capabilities) OnTools() bool { return c.on(c.Tools, true) }

// OnSkills reports whether SkillLoader is enabled.
func (c Capabilities) OnSkills() bool { return c.on(c.Skills, true) }

// OnMCP reports whether MCP tools are enabled.
func (c Capabilities) OnMCP() bool { return c.on(c.MCP, true) }

// OnGuard reports whether LLM Guard is enabled.
func (c Capabilities) OnGuard() bool { return c.on(c.Guard, false) }

// OnApprover reports whether Approver is enabled.
func (c Capabilities) OnApprover() bool { return c.on(c.Approver, false) }

// OnHooks reports whether RunHooks are enabled.
func (c Capabilities) OnHooks() bool { return c.on(c.Hooks, false) }

// OnObserver reports whether RunObserver is enabled.
func (c Capabilities) OnObserver() bool { return c.on(c.Observer, false) }

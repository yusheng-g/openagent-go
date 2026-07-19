package openagent

// AgentOption is a functional option for configuring an Agent.
type AgentOption func(*Agent)

// WithModel sets the LLM provider.
func WithModel(m Model) AgentOption {
	return func(a *Agent) { a.Model = m }
}

// WithDescription sets a human-readable description of the agent.
func WithDescription(desc string) AgentOption {
	return func(a *Agent) { a.Description = desc }
}

// WithSystemPrompts replaces all system prompts.
func WithSystemPrompts(prompts ...string) AgentOption {
	return func(a *Agent) { a.SystemPrompts = prompts }
}

// WithTools adds tools to the agent.
func WithTools(tools ...Tool) AgentOption {
	return func(a *Agent) { a.Tools = append(a.Tools, tools...) }
}

// WithMemory sets the memory backend for conversation persistence.
func WithMemory(m Memory) AgentOption {
	return func(a *Agent) { a.Memory = m }
}

// WithPrompt replaces the default prompt builder.
func WithPrompt(pb PromptBuilder) AgentOption {
	return func(a *Agent) { a.Prompt = pb }
}

// WithInputGuard sets the input guard.
func WithInputGuard(g InputGuard) AgentOption {
	return func(a *Agent) { a.InGuard = g }
}

// WithOutputGuard sets the output guard.
func WithOutputGuard(g OutputGuard) AgentOption {
	return func(a *Agent) { a.OutGuard = g }
}

// WithApprover sets the tool-call approver. If nil (default), all tool calls
// are executed without asking.
func WithApprover(ap Approver) AgentOption {
	return func(a *Agent) { a.Approver = ap }
}

// WithRunHooks sets lifecycle hooks for the agent.
func WithRunHooks(h RunHooks) AgentOption {
	return func(a *Agent) { a.Hooks = h }
}

// WithRunObserver sets the stage-level observer for the agent.
// Use WithRunObservers to combine multiple observers.
func WithRunObserver(o RunObserver) AgentOption {
	return func(a *Agent) { a.Observer = o }
}

// WithRunObservers combines multiple RunObservers into one.
// Equivalent to: WithRunObserver(MultiObserver(observers...))
func WithRunObservers(observers ...RunObserver) AgentOption {
	return WithRunObserver(MultiObserver(observers...))
}

// WithSkillLoader sets the skill loader for on-demand skill loading.
func WithSkillLoader(sl SkillLoader) AgentOption {
	return func(a *Agent) { a.SkillLoader = sl }
}

// WithMaxTurns sets the maximum number of loop iterations per run.
func WithMaxTurns(n int) AgentOption {
	return func(a *Agent) { a.MaxTurns = n }
}

// WithMaxWorkingTokens sets the max token budget for the working message set.
// When exceeded, the runner triggers incremental compression via Memory.Compact().
// 0 (default) means auto: 70% of the model's context window, or 20000 as fallback.
func WithMaxWorkingTokens(n int) AgentOption {
	return func(a *Agent) { a.MaxWorkingTokens = n }
}

// WithMaxCompressedTokens sets the maximum token budget for the compressed
// summary. The summarizer will merge and de-duplicate facts when the budget
// is exceeded. 0 means no explicit limit (default 2048).
func WithMaxCompressedTokens(n int) AgentOption {
	return func(a *Agent) { a.MaxCompressedTokens = n }
}

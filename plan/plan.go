package plan

import (
	"context"
	"fmt"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
)

// Plan orchestrates multi-agent execution through a Planner-generated DAG.
//
// Create with NewPlan + PlanOption functions:
//
//	plan := plan.NewPlan(
//	    plan.WithPlanner(plan.NewLLMPlanner(model)),
//	    plan.WithAgent("coder", "writes code", coderAgent),
//	    plan.WithAgent("reviewer", "reviews code", reviewerAgent),
//	    plan.WithMaxConcurrency(4),
//	)
//
//	// One-shot: plan + execute.
//	result, err := plan.Run(ctx, session, "Write a REST API")
//
//	// Two-step: review plan before executing.
//	def, err := plan.Plan(ctx, "Write a REST API", nil)
//	// show def to user, let them edit...
//	result, err := plan.Execute(ctx, session, def)
type Plan struct {
	planner    Planner
	agents     map[string]openagent.AgentRunner
	agentInfos []openagent.AgentInfo
	model      openagent.Model // for step output summarisation
	config     PlanConfig
	approver   PlanApprover // nil = auto-approve
}

// PlanOption configures a Plan.
type PlanOption func(*Plan)

// WithPlanner sets the Planner used to decompose goals into DAGs.
func WithPlanner(p Planner) PlanOption {
	return func(pl *Plan) { pl.planner = p }
}

// WithAgent adds an AgentRunner to the plan's agent pool.
// name must be unique. description is shown to the Planner so it can
// assign appropriate tasks.
func WithAgent(name, description string, runner openagent.AgentRunner) PlanOption {
	return func(pl *Plan) {
		pl.agents[name] = runner
		at := openagent.AgentExternal
		if _, ok := runner.(*openagent.Agent); ok {
			at = openagent.AgentInternal
		}
		pl.agentInfos = append(pl.agentInfos, openagent.AgentInfo{
			Name: name, Description: description, Type: at,
		})
	}
}

// WithModel sets the model used for step output summarisation.
// If nil, summaries fall back to truncated raw output.
func WithModel(model openagent.Model) PlanOption {
	return func(pl *Plan) { pl.model = model }
}

// WithConfig sets the full execution configuration.
func WithConfig(cfg PlanConfig) PlanOption {
	return func(pl *Plan) { pl.config = cfg }
}

// WithMaxConcurrency sets the maximum number of steps that can run concurrently.
func WithMaxConcurrency(n int) PlanOption {
	return func(pl *Plan) { pl.config.MaxConcurrency = n }
}

// WithStepTimeout sets the maximum duration for a single step.
func WithStepTimeout(d time.Duration) PlanOption {
	return func(pl *Plan) { pl.config.StepTimeout = d }
}

// WithAutoReplan controls whether the executor auto-replans on step failure.
// When false (default for interactive UIs), execution pauses on failure so the
// user can manually retry or replan with feedback.
func WithAutoReplan(enabled bool) PlanOption {
	return func(pl *Plan) { pl.config.AutoReplan = enabled }
}

// WithPlanApprover sets a callback for user confirmation before execution.
// nil (default) means auto-approve — [Plan.Run] executes immediately.
func WithPlanApprover(a PlanApprover) PlanOption {
	return func(pl *Plan) { pl.approver = a }
}

// ── Constructor ──

// NewPlan creates a Plan with the given options.
// At least one agent and a Planner must be configured.
func NewPlan(opts ...PlanOption) *Plan {
	p := &Plan{
		agents: make(map[string]openagent.AgentRunner),
		config: DefaultPlanConfig(),
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

// ── Plan (generate DAG only) ──

// Plan generates a PlanDef from a goal. This is the "dry run" step —
// the caller can display the plan, let the user edit it, then call [Plan.Execute].
func (p *Plan) Plan(ctx context.Context, goal string, history []openagent.Message) (*PlanDef, error) {
	if p.planner == nil {
		return nil, fmt.Errorf("plan: no Planner configured")
	}
	if len(p.agents) == 0 {
		return nil, fmt.Errorf("plan: no agents configured")
	}

	def, err := p.planner.Plan(ctx, goal, p.agentInfos, history)
	if err != nil {
		return nil, err
	}

	// Validate agent references.
	agentNames := make(map[string]bool)
	for _, a := range p.agentInfos {
		agentNames[a.Name] = true
	}
	if err := Validate(def, agentNames); err != nil {
		return nil, fmt.Errorf("planner produced invalid plan: %w", err)
	}

	// Enforce max steps.
	if len(def.Steps) > p.config.MaxSteps {
		return nil, fmt.Errorf("plan has %d steps, max is %d", len(def.Steps), p.config.MaxSteps)
	}

	return def, nil
}

// PlanStream generates a PlanDef with streaming text chunks emitted via onChunk.
// If the underlying Planner supports streaming (i.e. is an *LLMPlanner), LLM
// output tokens are passed to onChunk as they arrive. Otherwise falls back to
// a synchronous Plan() call with no streaming.
func (p *Plan) PlanStream(ctx context.Context, goal string, history []openagent.Message, onChunk func(string)) (*PlanDef, error) {
	if p.planner == nil {
		return nil, fmt.Errorf("plan: no Planner configured")
	}
	if len(p.agents) == 0 {
		return nil, fmt.Errorf("plan: no agents configured")
	}

	// Use streaming if planner supports it.
	type streamPlanner interface {
		PlanStream(ctx context.Context, goal string, agents []openagent.AgentInfo, history []openagent.Message, onChunk func(string)) (*PlanDef, error)
	}
	if sp, ok := p.planner.(streamPlanner); ok {
		def, err := sp.PlanStream(ctx, goal, p.agentInfos, history, onChunk)
		if err != nil {
			return nil, err
		}
		// Re-validate.
		agentNames := make(map[string]bool)
		for _, a := range p.agentInfos {
			agentNames[a.Name] = true
		}
		if err := Validate(def, agentNames); err != nil {
			return nil, fmt.Errorf("planner produced invalid plan: %w", err)
		}
		if len(def.Steps) > p.config.MaxSteps {
			return nil, fmt.Errorf("plan has %d steps, max is %d", len(def.Steps), p.config.MaxSteps)
		}
		return def, nil
	}

	// Fallback: no streaming support in planner.
	return p.Plan(ctx, goal, history)
}

// ── Execute (run an existing plan) ──

// Execute runs a previously generated PlanDef to completion.
func (p *Plan) Execute(ctx context.Context, session openagent.Session, def *PlanDef) (*PlanResult, error) {
	if p.approver != nil {
		approved, reason := p.approver.ApprovePlan(ctx, def)
		if !approved {
			return nil, fmt.Errorf("plan rejected: %s", reason)
		}
	}

	return p.execute(ctx, session, def, nil)
}

// ExecuteStream runs a PlanDef with real-time progress events.
func (p *Plan) ExecuteStream(ctx context.Context, session openagent.Session, def *PlanDef) <-chan PlanEvent {
	ch := make(chan PlanEvent, 32)
	go func() {
		defer close(ch)
		if p.approver != nil {
			approved, reason := p.approver.ApprovePlan(ctx, def)
			if !approved {
				ch <- PlanEvent{Type: PlanEventError, ErrText: fmt.Sprintf("plan rejected: %s", reason)}
				return
			}
			ch <- PlanEvent{Type: PlanEventApproved}
		}
		if _, err := p.execute(ctx, session, def, ch); err != nil {
			ch <- PlanEvent{Type: PlanEventError, ErrText: err.Error()}
		}
	}()
	return ch
}

// ExecuteWithState resumes a paused plan execution from an existing [PlanState].
// Use after [PlanEventWaitingRetry] to retry a failed step or continue with a
// replanned definition.
//
// The caller is responsible for resetting the failed step to pending before
// calling this (for retry), or replacing state.Steps with the new definition
// (for replan).
func (p *Plan) ExecuteWithState(ctx context.Context, session openagent.Session, def *PlanDef, state *PlanState) <-chan PlanEvent {
	ch := make(chan PlanEvent, 32)
	go func() {
		defer close(ch)
		ex := &executor{
			config:     p.config,
			agents:     p.agents,
			agentInfos: p.agentInfos,
			model:      p.model,
			sessionID:  session.ID,
		}
		if _, err := ex.execute(ctx, def, state, ch); err != nil {
			ch <- PlanEvent{Type: PlanEventError, ErrText: err.Error()}
		}
	}()
	return ch
}

// ReplanWithFeedback calls the Planner to regenerate the affected subtree of a
// failed plan, incorporating user feedback (natural language suggestions for
// how to approach the replan). It returns a new PlanDef with surviving steps
// merged with the replacement subtree.
//
// The caller should update the session's currentDef with the returned PlanDef
// and call [Plan.ExecuteWithState] to resume execution.
func (p *Plan) ReplanWithFeedback(ctx context.Context, def *PlanDef, state *PlanState, failedID string, feedback string) (*PlanDef, error) {
	ex := &executor{
		config:     p.config,
		agents:     p.agents,
		agentInfos: p.agentInfos,
		model:      p.model,
	}
	return ex.replanWithFeedback(ctx, def, state, failedID, feedback)
}

// ── Run (plan + execute in one call) ──

// Run generates a plan from the goal and executes it.
// Use when you don't need user review of the plan.
func (p *Plan) Run(ctx context.Context, session openagent.Session, goal string, history []openagent.Message) (*PlanResult, error) {
	def, err := p.Plan(ctx, goal, history)
	if err != nil {
		return nil, fmt.Errorf("plan: %w", err)
	}
	return p.Execute(ctx, session, def)
}

// RunStream generates a plan and executes it with streaming events.
func (p *Plan) RunStream(ctx context.Context, session openagent.Session, goal string, history []openagent.Message) <-chan PlanEvent {
	ch := make(chan PlanEvent, 32)
	go func() {
		defer close(ch)
		def, err := p.Plan(ctx, goal, history)
		if err != nil {
			ch <- PlanEvent{Type: PlanEventError, ErrText: fmt.Sprintf("plan: %s", err.Error())}
			return
		}
		ch <- PlanEvent{Type: PlanEventGenerated, Goal: def.Goal, Def: def}
		// Continue with streaming execution.
		// We can't re-use ExecuteStream easily because of the double channel,
		// so run the executor inline.
		if p.approver != nil {
			approved, reason := p.approver.ApprovePlan(ctx, def)
			if !approved {
				ch <- PlanEvent{Type: PlanEventError, ErrText: fmt.Sprintf("plan rejected: %s", reason)}
				return
			}
			ch <- PlanEvent{Type: PlanEventApproved}
		}
		if _, err := p.execute(ctx, session, def, ch); err != nil {
			ch <- PlanEvent{Type: PlanEventError, ErrText: err.Error()}
		}
	}()
	return ch
}

// ── Internal execution ──

func (p *Plan) execute(ctx context.Context, session openagent.Session, def *PlanDef, eventCh chan<- PlanEvent) (*PlanResult, error) {
	now := time.Now()
	state := &PlanState{
		ID:        session.ID + "/plan",
		Goal:      def.Goal,
		Status:    PlanStatusApproved,
		Steps:     def.Steps,
		Results:   make(map[string]*StepResult),
		CreatedAt: now,
		UpdatedAt: now,
	}

	ex := &executor{
		config:     p.config,
		agents:     p.agents,
		agentInfos: p.agentInfos,
		model:      p.model,
		sessionID:  session.ID,
	}

	return ex.execute(ctx, def, state, eventCh)
}

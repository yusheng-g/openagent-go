package openagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/yusheng-g/openagent-go/tokenizer"
)

// runner is the internal mainline loop executor. Users call Agent.Run(),
// which creates a runner and starts the loop. runner is not exported.
type runner struct {
	agent *Agent

	// Cached state
	skills       []SkillInfo          // Discover result, refreshed by reload_skills
	loadedSkills map[string]string    // name → body, populated by use_skill
	builtinTools []FunctionDefinition // auto-injected tools (use_skill, reload_skills)
	compressed   *CompressedContext   // Memory.Compressed result, set once per Run()

	// Per-run state
	runModel Model // resolved model for this run (session override > agent default)
}

// compactionInfo carries compaction outcome from prepareMemory back to run().
type compactionInfo struct {
	err   error
	count int // new messages compacted this run (0 = none)
	from  int // global index of first new compacted message
	to    int // global index after compaction (ThroughIndex)
}

// observe emits a stage event to the agent's RunObserver if configured.
func (r *runner) observe(ctx context.Context, name string, phase string, detail map[string]any, start time.Time, err error) {
	if r.agent.Observer == nil {
		return
	}
	event := StageEvent{Name: name, Phase: phase, Detail: detail, Err: err}
	if phase == "leave" {
		event.Duration = time.Since(start)
	}
	r.agent.Observer.ObserveStage(ctx, event)
}

// run executes the 8-node mainline loop.
//
//	① Memory fetch → ② Prompt build → ③ Guard.in → ④ Model call
//	⑤ Guard.out → ⑥ Approval → ⑦ Tool execution → ⑧ Memory store
//	Has tool_calls → loop back to ②, else return.
//
// prefix messages are injected after Memory.Recent() and before input.
// They participate in this run only — not persisted to Memory.
func (r *runner) run(ctx context.Context, session Session, prefix []Message, input Message, ch chan<- StreamEvent) (_ *RunResult, runErr error) {
	maxTurns := r.agent.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 20
	}

	// Resolve model for this run.
	r.runModel = r.agent.Model
	if session.Model != nil {
		r.runModel = session.Model
	}

	// Init: cache skills if loader present
	if r.agent.SkillLoader != nil {
		skills, _ := r.agent.SkillLoader.Discover(ctx)
		r.skills = skills
		r.loadedSkills = make(map[string]string)
		r.builtinTools = builtinSkillToolDefs()
	}
	if r.agent.Memory != nil {
		r.builtinTools = append(r.builtinTools, builtinRecallDef)
	}
	if !r.agent.noSpawn {
		r.builtinTools = append(r.builtinTools, builtinSubAgentDef)
	}

	result := &RunResult{}

	// Append initial user input to memory
	r.appendMemory(ctx, session, input)

	// Track last request/response for RunHooks.OnAgentEnd
	var lastReq ChatCompletionRequest
	var lastResp *ChatCompletionResponse
	var agentHookState any

	// ── RunHooks.OnAgentStart ──
	// Build the tool list once so hooks see the full set (including built-in
	// skill tools) that will be sent to the model.
	allToolDefs := toolDefinitions(r.agent.Tools)
	if len(r.builtinTools) > 0 {
		allToolDefs = append(allToolDefs, r.builtinTools...)
	}
	if r.agent.Hooks != nil {
		agentHookState, _ = r.agent.Hooks.OnAgentStart(ctx, ChatCompletionRequest{
			Model:    session.ModelID,
			Messages: []Message{input},
			Tools:    allToolDefs,
		})
	}
	defer func() {
		if r.agent.Hooks != nil {
			resp := lastResp
			if resp == nil {
				resp = &ChatCompletionResponse{}
			}
			r.agent.Hooks.OnAgentEnd(ctx, lastReq, resp, runErr, agentHookState)
		}
	}()

	var workingMessages []Message

	turn := 0
	for turn = 1; turn <= maxTurns; turn++ {
		select {
		case <-ctx.Done():
			runErr = ctx.Err()
			if ch != nil {
				ch <- StreamEvent{Type: StreamAborted, Error: runErr}
			}
			return result, runErr
		default:
		}
		session.Turn = turn

		// ── ① Build working message set on first turn ──
		// Order: memory history → prefix (transient, not persisted) → input
		if turn == 1 {
			// prepareMemory handles compaction + fetch in one call

			mfStart := time.Now()
			r.observe(ctx, StageMemoryFetch, "enter", nil, time.Time{}, nil)

			var history []Message
			var compressedErr error // fetching Compressed() context (Layer 2)
			var ci compactionInfo
			if r.agent.Memory != nil {
				history, ci = r.prepareMemory(ctx, session)
				// The input was just appended to memory — strip it
				// from history since we add it back after prefix.
				if len(history) > 0 && history[len(history)-1].Role == RoleUser {
					history = history[:len(history)-1]
				}

				// Fetch compressed context (Layer 2 of the memory model).
				// Errors are collected and reported in the leave event below —
				// compressed context is an optimization, not a requirement.
				cc, err := r.agent.Memory.Compressed(ctx, session.ID)
				if err == nil {
					r.compressed = cc
				} else {
					compressedErr = err
				}
			}

			workingMessages = make([]Message, 0, len(history)+len(prefix)+1)
			workingMessages = append(workingMessages, history...)
			workingMessages = append(workingMessages, prefix...)
			workingMessages = append(workingMessages, input)

			mfDetail := map[string]any{}
			if compressedErr != nil {
				mfDetail["compressed_error"] = compressedErr.Error()
			}
			if ci.err != nil {
				mfDetail["compaction_error"] = ci.err.Error()
			} else if ci.count > 0 {
				mfDetail["compacted_count"] = ci.count
				mfDetail["compacted_from"] = ci.from
				mfDetail["compacted_to"] = ci.to
				if r.compressed != nil && r.compressed.Summary != "" {
					mfDetail["compacted_summary"] = r.compressed.Summary
				}
			}
			r.observe(ctx, StageMemoryFetch, "leave", mfDetail, mfStart, nil)

			// ── ③ Guard.in: input check with full history (memory + prefix + input) ──
			giStart := time.Now()
			r.observe(ctx, StageGuardIn, "enter", nil, time.Time{}, nil)
			if r.agent.InGuard != nil {
				gr := r.agent.InGuard.Check(ctx, GuardInput{
					Session: session,
					Input:   input,
					History: workingMessages,
				})
				if !gr.Allowed {
					runErr = fmt.Errorf("input guard blocked: %s", gr.Reason)
					r.observe(ctx, StageGuardIn, "leave", nil, giStart, runErr)
					if ch != nil {
						ch <- StreamEvent{Type: StreamError, Error: runErr}
					}
					return result, runErr
				}
				if gr.Tripwire {
					runErr = fmt.Errorf("input guard tripwire: %s", gr.Reason)
					r.observe(ctx, StageGuardIn, "leave", nil, giStart, runErr)
					if ch != nil {
						ch <- StreamEvent{Type: StreamError, Error: runErr}
					}
					return result, runErr
				}
			}
			r.observe(ctx, StageGuardIn, "leave", nil, giStart, nil)
		}

		// ── ② Prompt: build message list ──
		pbStart := time.Now()
		r.observe(ctx, StagePromptBuild, "enter", nil, time.Time{}, nil)
		messages := r.buildPrompt(ctx, session, workingMessages)
		r.observe(ctx, StagePromptBuild, "leave", nil, pbStart, nil)

		// ── ④ Model: call LLM ──
		lastReq = r.buildModelRequest(session, messages)

		// Last-resort truncation: if the full message set exceeds the model's
		// context window, drop oldest non-system messages to fit. The compaction
		// pipeline normally keeps the working set within budget; this triggers
		// only when system prompts, compressed context, or large tool results
		// push it past the hard limit.
		if cw := r.runModel.ContextWindow(); cw > 0 {
			est := countMessages(tokenizerModelID(r.runModel), messages)
			if est > cw {
				before := len(messages)
				messages = trimToContextWindow(tokenizerModelID(r.runModel), messages, cw)
				trimmed := before - len(messages)
				// When using the default prompt builder, every non-system
				// message in the model input originates from workingMessages,
				// so trimToContextWindow drops exactly workingMessages[:trimmed].
				// With a custom PromptBuilder this mapping doesn't hold (the
				// builder can insert non-system messages anywhere), so we skip
				// the sync — it's just an optimization, and the compaction
				// pipeline normally prevents this path anyway.
				if r.agent.Prompt == nil && trimmed > 0 && trimmed <= len(workingMessages) {
					workingMessages = workingMessages[trimmed:]
				}
				lastReq = r.buildModelRequest(session, messages)
				r.observe(ctx, StageModelCall, "enter",
					map[string]any{
						"warning":          "context window exceeded — messages trimmed",
						"estimated_tokens": est,
						"window":           cw,
						"trimmed":          trimmed,
					},
					time.Time{}, nil)
			}
		}

		mcStart := time.Now()
		r.observe(ctx, StageModelCall, "enter", map[string]any{
			"turn":     turn,
			"maxTurns": maxTurns,
		}, time.Time{}, nil)
		resp, err := r.callModel(ctx, lastReq, ch)

		if err != nil {
			r.observe(ctx, StageModelCall, "leave", nil, mcStart, err)
			if ch != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					ch <- StreamEvent{Type: StreamAborted, Error: err}
				} else {
					ch <- StreamEvent{Type: StreamError, Error: err}
				}
			}
			runErr = fmt.Errorf("model call: %w", err)
			return result, runErr
		}
		r.observe(ctx, StageModelCall, "leave", map[string]any{
			"tokens_prompt":     resp.Usage.PromptTokens,
			"tokens_completion": resp.Usage.CompletionTokens,
		}, mcStart, nil)
		lastResp = resp

		if len(resp.Choices) == 0 {
			runErr = fmt.Errorf("model returned no choices")
			if ch != nil {
				ch <- StreamEvent{Type: StreamError, Error: runErr}
			}
			return result, runErr
		}
		choice := resp.Choices[0]
		result.Usage.PromptTokens += resp.Usage.PromptTokens
		result.Usage.CompletionTokens += resp.Usage.CompletionTokens
		result.Usage.TotalTokens += resp.Usage.TotalTokens
		result.FinalOutput = choice.Message.Content
		result.Messages = append(result.Messages, choice.Message)

		// Emit tool call events
		for _, tc := range choice.Message.ToolCalls {
			if ch != nil {
				ch <- StreamEvent{Type: StreamToolCall, Message: Message{ToolCalls: []ToolCall{tc}}}
			}
		}

		// Track response in working set and memory.
		// Stamp the agent name so frontend history can label who said what.
		if choice.Message.Name == "" {
			choice.Message.Name = r.agent.Name
		}
		workingMessages = append(workingMessages, choice.Message)
		r.appendMemory(ctx, session, choice.Message)

		// ── ⑤ Guard.out: output check (model output + tool results) ──
		var guardOutStart time.Time
		if r.agent.OutGuard != nil {
			guardOutStart = time.Now()
			r.observe(ctx, StageGuardOut, "enter", nil, time.Time{}, nil)
			gr := r.agent.OutGuard.Check(ctx, GuardOutput{
				Session: session,
				Output:  choice.Message,
				History: workingMessages,
			})
			if !gr.Allowed {
				runErr = fmt.Errorf("output guard blocked: %s", gr.Reason)
				r.observe(ctx, StageGuardOut, "leave", nil, guardOutStart, runErr)
				if ch != nil {
					ch <- StreamEvent{Type: StreamError, Error: runErr}
				}
				return result, runErr
			}
			if gr.Tripwire {
				runErr = fmt.Errorf("output guard tripwire: %s", gr.Reason)
				r.observe(ctx, StageGuardOut, "leave", nil, guardOutStart, runErr)
				if ch != nil {
					ch <- StreamEvent{Type: StreamError, Error: runErr}
				}
				return result, runErr
			}
			// Leave deferred until after tool result guard checks.
		}

		// ── Abnormal finish ──
		if len(choice.Message.ToolCalls) == 0 {
			if r.agent.OutGuard != nil {
				r.observe(ctx, StageGuardOut, "leave", nil, guardOutStart, nil)
			}
			if choice.FinishReason != "" && choice.FinishReason != "stop" {
				result.StopReason = choice.FinishReason
				result.Messages = append(result.Messages, Message{
					Role:    RoleSystem,
					Content: fmt.Sprintf("[run ended: finish_reason=%s]", choice.FinishReason),
				})
			}
			break
		}

		// ── ⑥ + ⑦ Tool execution ──
		toolResults := r.executeTools(ctx, session, choice.Message.ToolCalls, ch)

		// Guard tool results before feeding back to context
		for _, tr := range toolResults {
			if r.agent.OutGuard != nil {
				gr := r.agent.OutGuard.Check(ctx, GuardOutput{
					Session: session,
					Output:  tr,
					History: workingMessages,
				})
				if gr.Tripwire {
					runErr = fmt.Errorf("output guard tripwire on tool result: %s", gr.Reason)
					r.observe(ctx, StageGuardOut, "leave", nil, guardOutStart, runErr)
					if ch != nil {
						ch <- StreamEvent{Type: StreamError, Error: runErr}
					}
					return result, runErr
				}
				if !gr.Allowed {
					tr.Content = fmt.Sprintf("[blocked: %s]", gr.Reason)
				}
			}
			if ch != nil {
				ch <- StreamEvent{Type: StreamToolResult, Message: tr}
			}
			result.Messages = append(result.Messages, tr)
			r.appendMemory(ctx, session, tr)
			workingMessages = append(workingMessages, tr)
		}
		if r.agent.OutGuard != nil {
			r.observe(ctx, StageGuardOut, "leave", nil, guardOutStart, nil)
		}

		// If any tool executed this turn is EndTurn (e.g. transfer_to_*),
		// break immediately — the agent committed to a handoff.
		// Aligns with OpenAI Agents SDK's NextStepHandoff semantics.
		for _, call := range choice.Message.ToolCalls {
			def := r.toolDef(call.Function.Name)
			if def != nil && def.EndTurn {
				result.StopReason = "handoff"
				break
			}
		}
		if result.StopReason == "handoff" {
			break
		}

		// ── Loop back to ② with tool results included ──
	}

	result.TurnCount = turn
	result.ContextWindow = r.runModel.ContextWindow()
	if ch != nil {
		ch <- StreamEvent{Type: StreamDone, Result: result}
	}
	return result, nil
}

// ── Internal helpers ──

// workingTokenBudget returns the token budget for the working message set.
// If MaxWorkingTokens is set explicitly, use it. Otherwise, use 70% of the
// model's context window. Falls back to 20000 if the model doesn't report
// its context window.
func (r *runner) workingTokenBudget() int {
	if r.agent.MaxWorkingTokens > 0 {
		return r.agent.MaxWorkingTokens
	}
	if cw := r.runModel.ContextWindow(); cw > 0 {
		return cw * 7 / 10 // 70%
	}
	return 20000
}

// prepareMemory fetches the working message set, triggers token-based
// compaction if needed, and trims to the token budget. It replaces the
// previous compactIfNeeded + fetchMemory pair, eliminating a redundant
// Recent() call. Messages are NEVER deleted — compaction only updates
// the summary.
//
// The returned error carries a compaction failure if one occurred
// (observability only; the working set is still usable).
func (r *runner) prepareMemory(ctx context.Context, session Session) ([]Message, compactionInfo) {
	if r.agent.Memory == nil {
		return nil, compactionInfo{}
	}

	budget := r.workingTokenBudget()

	// ── Subtract fixed overhead that buildPrompt adds ──
	// System instructions, compressed summary, project context, and
	// skills all consume tokens outside the working message set.
	// If we don't account for them, the model sees more tokens than
	// the budget expects and trimToContextWindow becomes the only
	// defence — which it was designed as a last-resort, not the
	// primary mechanism.
	modelID := tokenizerModelID(r.runModel)
	overhead := r.estimatePromptOverhead(ctx, session, modelID)
	budget -= overhead
	if budget < 500 {
		budget = 500 // keep a minimal working window
	}

	var ci compactionInfo

	// Fetch total count and recent messages — one Recent() call for both
	// compaction and working-set trimming.
	totalCount, err := r.agent.Memory.Count(ctx, session.ID)
	if err != nil {
		r.observe(ctx, StageMemoryFetch, "leave",
			map[string]any{"error": err.Error()}, time.Now(), err)
		return nil, ci
	}
	if totalCount == 0 {
		return nil, ci
	}
	fetchN := totalCount
	if fetchN > 5000 {
		fetchN = 5000
	}
	msgs, err := r.agent.Memory.Recent(ctx, session.ID, fetchN, 0)
	if err != nil || len(msgs) == 0 {
		return nil, ci
	}
	globalOffset := totalCount - len(msgs)

	// ── Compaction pass: compress overflow messages ──
	// Count tokens backwards from the latest message. Messages before the
	// overflow point dont fit in the budget and are candidates for compaction.
	overflow := len(msgs)
	tokens := 0
	for i := len(msgs) - 1; i >= 0; i-- {
		tokens += countMessageTokens(tokenizerModelID(r.runModel), msgs[i])
		if tokens > budget {
			overflow = i + 1 // messages[0:i] overflow, messages[i+1:] fit
			break
		}
	}
	if overflow < len(msgs) {
		overflow = SafeCompressionBoundary(msgs, overflow)
		// Record pre-compaction ThroughIndex so we can detect whether
		// Compact() actually covered new messages.
		oldTI := 0
		if cc, _ := r.agent.Memory.Compressed(ctx, session.ID); cc != nil {
			oldTI = cc.ThroughIndex
		}
		globalCutoff := globalOffset + overflow
		ci.err = r.agent.Memory.Compact(ctx, session.ID, globalCutoff, msgs)
		if ci.err == nil {
			// Only report compaction if ThroughIndex advanced.
			if cc, _ := r.agent.Memory.Compressed(ctx, session.ID); cc != nil && cc.ThroughIndex > oldTI {
				ci.count = cc.ThroughIndex - oldTI
				ci.from = globalOffset + oldTI
				ci.to = globalOffset + cc.ThroughIndex
			}
		}
	}

	// ── Working set: trim to token budget ──
	keep := overflow
	if keep >= len(msgs) {
		return msgs, ci
	}
	return msgs[keep:], ci
}

// estimatePromptOverhead returns the estimated token count of everything
// defaultBuildPrompt adds BEFORE the working messages: system instructions,
// compressed summary, skills catalog, etc.  This is subtracted from the
// working token budget so that the total prompt (overhead + working) fits
// within the model's context window.
//
// When a custom PromptBuilder is configured we cannot predict what it will
// produce, so we return 0 — the caller's budget becomes best-effort.
func (r *runner) estimatePromptOverhead(ctx context.Context, session Session, modelID string) int {
	if r.agent.Prompt != nil {
		return 0 // custom PromptBuilder — can't know
	}

	var n int

	// System prompts (Agent.SystemPrompts).
	sys := strings.Join(r.agent.SystemPrompts, "\n\n")
	if r.agent.Description != "" {
		sys = r.agent.Description + "\n\n" + sys
	}
	if sys != "" {
		n += tokenizer.Count(modelID, sys) + 4
	}

	// Project context.
	if session.ProjectContext != "" {
		n += tokenizer.Count(modelID, session.ProjectContext) + 4
	}

	// Compressed summary + hints.
	if cc, err := r.agent.Memory.Compressed(ctx, session.ID); err == nil && cc != nil && cc.Summary != "" {
		content := "## Conversation Summary\n" + cc.Summary
		if len(cc.Hints) > 0 {
			content += "\n\n### Retrieval Hints\n"
			for i, h := range cc.Hints {
				content += fmt.Sprintf("%d. %s (query: %s)\n", i+1, h.Description, h.Query)
			}
		}
		n += tokenizer.Count(modelID, content) + 4
	}

	// Skills catalog.
	if len(r.skills) > 0 {
		var catalog string
		for _, s := range r.skills {
			catalog += "\n### " + s.Name + "\n"
			for k, v := range s.Frontmatter {
				catalog += fmt.Sprintf("%s: %v\n", k, v)
			}
		}
		n += tokenizer.Count(modelID, "## Available Skills\n"+catalog) + 4
	}

	// Loaded skill bodies.
	for name, body := range r.loadedSkills {
		n += tokenizer.Count(modelID, "## Loaded Skill: "+name+"\n\n"+body) + 4
	}

	return n
}

func (r *runner) buildPrompt(ctx context.Context, session Session, working []Message) []Message {
	input := PromptInput{
		AgentDescription: r.agent.Description,
		Instructions:     strings.Join(r.agent.SystemPrompts, "\n\n"),
		WorkingMessages:  working,
		Tools:            toolDefinitions(r.agent.Tools),
		UserProfile:      session.UserProfile,
		ProjectContext:   session.ProjectContext,
	}

	if r.compressed != nil {
		input.Compressed = r.compressed
	}

	if len(r.skills) > 0 {
		input.AvailableSkills = r.skills
	}
	if len(r.loadedSkills) > 0 {
		input.LoadedSkills = r.loadedSkills
	}

	if r.agent.Prompt != nil {
		if msgs, err := r.agent.Prompt(ctx, input); err == nil {
			return msgs
		}
	}
	return defaultBuildPrompt(input)
}

// defaultBuildPrompt assembles system instructions + skill catalog +
// loaded skill bodies + working messages.
func defaultBuildPrompt(input PromptInput) []Message {
	var msgs []Message

	system := input.Instructions
	if input.AgentDescription != "" {
		system = input.AgentDescription + "\n\n" + system
	}
	msgs = append(msgs, Message{Role: RoleSystem, Content: system})

	if input.ProjectContext != "" {
		msgs = append(msgs, Message{Role: RoleSystem, Content: input.ProjectContext})
	}

	if input.Compressed != nil && input.Compressed.Summary != "" {
		content := "## Conversation Summary\n" + input.Compressed.Summary
		if len(input.Compressed.Hints) > 0 {
			content += "\n\n### Retrieval Hints\n"
			for i, h := range input.Compressed.Hints {
				content += fmt.Sprintf("%d. %s (query: %s)\n", i+1, h.Description, h.Query)
			}
		}
		msgs = append(msgs, Message{Role: RoleSystem, Content: content})
	}

	if len(input.AvailableSkills) > 0 {
		var catalog string
		for _, s := range input.AvailableSkills {
			catalog += "\n### " + s.Name + "\n"
			for k, v := range s.Frontmatter {
				catalog += fmt.Sprintf("%s: %v\n", k, v)
			}
		}
		msgs = append(msgs, Message{
			Role:    RoleSystem,
			Content: "## Available Skills\n" + catalog,
		})
	}

	for name, body := range input.LoadedSkills {
		msgs = append(msgs, Message{
			Role:    RoleSystem,
			Content: "## Loaded Skill: " + name + "\n\n" + body,
		})
	}

	msgs = append(msgs, input.WorkingMessages...)
	return msgs
}

func (r *runner) buildModelRequest(session Session, messages []Message) ChatCompletionRequest {
	tools := toolDefinitions(r.agent.Tools)
	if len(r.builtinTools) > 0 {
		tools = append(tools, r.builtinTools...)
	}
	return ChatCompletionRequest{
		Model:           session.ModelID,
		Messages:        messages,
		Tools:           tools,
		Temperature:     session.Temperature,
		MaxTokens:       session.MaxTokens,
		ReasoningEffort: r.agent.ReasoningEffort,
	}
}

func (r *runner) executeTools(ctx context.Context, session Session, calls []ToolCall, ch chan<- StreamEvent) []Message {
	if len(calls) == 0 {
		return nil
	}

	results := make([]Message, len(calls))

	// When an Approver is configured, fire all approvals first (the user
	// clicks through dialogs quickly), then execute approved tools in
	// parallel. Before this change, each tool's approval + execution was
	// serialised — tool_1's approval dialog wouldn't appear until tool_0
	// finished executing (subagent runs can take 10+ seconds).
	if r.agent.Approver != nil {
		// Phase 1: approve all tools sequentially.
		approved := make([]bool, len(calls))
		for i, call := range calls {
			def := r.toolDef(call.Function.Name)
			if def == nil {
				results[i] = Message{
					Role:       RoleTool,
					ToolCallID: call.ID,
					Content:    fmt.Sprintf("tool %q not found", call.Function.Name),
				}
				continue
			}
			tool := r.findTool(call.Function.Name)
			needsApproval := !strings.HasPrefix(call.Function.Name, "transfer_to_")
			if needsApproval && tool != nil {
				if sa, ok := tool.(SelfApproving); ok && sa.CanSelfApprove(json.RawMessage(call.Function.Arguments)) {
					needsApproval = false
				}
			}
			if needsApproval {
				allowed, reason := r.agent.Approver.Approve(ctx, call, *def, session)
				if !allowed {
					results[i] = Message{
						Role:       RoleTool,
						ToolCallID: call.ID,
						Content:    fmt.Sprintf("tool %q rejected: %s", call.Function.Name, reason),
					}
					continue
				}
			}
			approved[i] = true
		}

		// Phase 2: execute approved tools concurrently.
		var wg sync.WaitGroup
		for i, call := range calls {
			if !approved[i] {
				continue
			}
			wg.Add(1)
			go func(idx int, tc ToolCall) {
				defer wg.Done()
				defer func() {
					if rec := recover(); rec != nil {
						results[idx] = Message{
							Role:       RoleTool,
							ToolCallID: tc.ID,
							Content:    fmt.Sprintf("tool panic: %v", rec),
						}
					}
				}()
				results[idx] = r.executeOneToolInternal(ctx, session, tc, ch)
			}(i, call)
		}
		wg.Wait()
		return results
	}

	// No approver — run tools concurrently.
	var wg sync.WaitGroup
	for i, call := range calls {
		wg.Add(1)
		go func(idx int, tc ToolCall) {
			defer wg.Done()
			defer func() {
				if rec := recover(); rec != nil {
					results[idx] = Message{
						Role:       RoleTool,
						ToolCallID: tc.ID,
						Content:    fmt.Sprintf("tool panic: %v", rec),
					}
				}
			}()
			results[idx] = r.executeOneToolInternal(ctx, session, tc, ch)
		}(i, call)
	}

	wg.Wait()
	return results
}

// executeOneToolInternal executes a single tool call — resolving definitions,
// firing hooks/observer events, and dispatching built-in vs registered tools.
// Approval is handled upstream by [executeTools] (Phase 1); this function
// assumes the tool has already been approved.
func (r *runner) executeOneToolInternal(ctx context.Context, session Session, call ToolCall, ch chan<- StreamEvent) Message {
	def := r.toolDef(call.Function.Name)
	if def == nil {
		return Message{
			Role:       RoleTool,
			ToolCallID: call.ID,
			Content:    fmt.Sprintf("tool %q not found", call.Function.Name),
		}
	}

	args := json.RawMessage(call.Function.Arguments)

	// Inject session into ctx so tools and hooks can retrieve it via
	// SessionFromContext — used by artifact hooks, audit logging, etc.
	toolCtx := WithSession(ctx, session)

	// Built-in tools: execute directly, share hooks/observer pipeline.
	switch call.Function.Name {
	case "use_skill":
		toolStart := r.fireToolHooks(toolCtx, *def, args)
		msg := r.executeUseSkill(toolCtx, call)
		r.fireToolHooksEnd(toolCtx, *def, args, msg.Content, toolStart, nil)
		return msg
	case "reload_skills":
		toolStart := r.fireToolHooks(toolCtx, *def, args)
		msg := r.executeReloadSkills(toolCtx, call)
		r.fireToolHooksEnd(toolCtx, *def, args, msg.Content, toolStart, nil)
		return msg
	case "recall":
		toolStart := r.fireToolHooks(toolCtx, *def, args)
		msg := r.executeRecall(toolCtx, session, call)
		r.fireToolHooksEnd(toolCtx, *def, args, msg.Content, toolStart, nil)
		return msg
	case "subagent":
		toolStart := r.fireToolHooks(toolCtx, *def, args)
		msg := r.executeSubAgent(toolCtx, session, call, ch)
		r.fireToolHooksEnd(toolCtx, *def, args, msg.Content, toolStart, nil)
		return msg
	}

	tool := r.findTool(call.Function.Name)

	var toolHookState any
	if r.agent.Hooks != nil {
		toolHookState, _ = r.agent.Hooks.OnToolStart(toolCtx, *def, args)
	}

	teStart := time.Now()
	r.observe(toolCtx, StageToolExecute, "enter", map[string]any{"tool": call.Function.Name}, time.Time{}, nil)
	var output string
	var execErr error

	// ── Streaming path (optional interface) ──
	if se, ok := tool.(StreamExecutor); ok {
		toolCh := se.ExecuteStream(toolCtx, args)
		var buf strings.Builder
		for chunk := range toolCh {
			if chunk.Error != nil {
				execErr = chunk.Error
				break
			}
			buf.WriteString(chunk.Content)
			// Emit progress for real-time display. ToolCallID disambiguates
			// concurrent streaming tools in the same turn.
			if ch != nil {
				ch <- StreamEvent{
					Type:       StreamToolProgress,
					Text:       chunk.Content,
					ToolCallID: call.ID,
				}
			}
		}
		output = buf.String()
	} else {
		// ── Blocking path (default) ──
		output, execErr = tool.Execute(toolCtx, args)
	}

	r.observe(ctx, StageToolExecute, "leave", map[string]any{"tool": call.Function.Name}, teStart, execErr)

	if r.agent.Hooks != nil {
		r.agent.Hooks.OnToolEnd(toolCtx, *def, args, &output, &execErr, toolHookState)
	}

	content := output
	if execErr != nil {
		content = fmt.Sprintf("error: %v", execErr)
	}

	return Message{
		Role:       RoleTool,
		ToolCallID: call.ID,
		Content:    content,
	}
}

// toolHookCtx bundles state passed from fireToolHooks to fireToolHooksEnd.
type toolHookCtx struct {
	start     time.Time
	hookState any // opaque value from RunHooks.OnToolStart
}

// fireToolHooks emits observer enter + OnToolStart for built-in tools.
func (r *runner) fireToolHooks(ctx context.Context, def FunctionDefinition, args json.RawMessage) toolHookCtx {
	tc := toolHookCtx{start: time.Now()}
	if r.agent.Hooks != nil {
		tc.hookState, _ = r.agent.Hooks.OnToolStart(ctx, def, args)
	}
	r.observe(ctx, StageToolExecute, "enter", map[string]any{"tool": def.Name}, time.Time{}, nil)
	return tc
}

// fireToolHooksEnd emits observer leave + OnToolEnd for built-in tools.
func (r *runner) fireToolHooksEnd(ctx context.Context, def FunctionDefinition, args json.RawMessage, output string, tc toolHookCtx, err error) {
	r.observe(ctx, StageToolExecute, "leave", map[string]any{"tool": def.Name}, tc.start, err)
	if r.agent.Hooks != nil {
		r.agent.Hooks.OnToolEnd(ctx, def, args, &output, &err, tc.hookState)
	}
}

// built-in skill tool definitions — single source of truth used by both
// toolDef (name→definition lookup) and builtinSkillToolDefs (model tool list).
var (
	builtinUseSkillDef = FunctionDefinition{
		Name:        "use_skill",
		Description: "Load a skill's full instructions from its SKILL.md. Use when you need detailed guidance on a specific topic.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"name":{"type":"string","description":"Name of the skill to load"}},"required":["name"]}`),
	}
	builtinReloadSkillsDef = FunctionDefinition{
		Name:        "reload_skills",
		Description: "Rescan the skills directory for newly installed or removed skills. Use after installing or uninstalling a skill.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
	}
	builtinRecallDef = FunctionDefinition{
		Name:        "recall",
		Description: "Search past conversation history for relevant information. Use this when you need to remember previously discussed facts, user preferences, decisions, or context that may not be in the current conversation window. Returns ranked results with relevance scores.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"Search query to find relevant memories (e.g. 'user favourite colour', 'database version', 'project deadline')"}},"required":["query"]}`),
	}
)

func (r *runner) findTool(name string) Tool {
	for _, t := range r.agent.Tools {
		if t.Definition().Name == name {
			return t
		}
	}
	return nil
}

// toolDef resolves a tool name to its FunctionDefinition.
// Returns nil if the tool is not found (neither built-in nor registered).
func (r *runner) toolDef(name string) *FunctionDefinition {
	switch name {
	case "use_skill":
		return &builtinUseSkillDef
	case "reload_skills":
		return &builtinReloadSkillsDef
	case "recall":
		return &builtinRecallDef
	case "subagent":
		return &builtinSubAgentDef
	}
	if t := r.findTool(name); t != nil {
		d := t.Definition()
		return &d
	}
	return nil
}

func (r *runner) appendMemory(ctx context.Context, session Session, msg Message) {
	if msg.Transient {
		return
	}
	if r.agent.Memory == nil {
		return
	}
	maStart := time.Now()
	r.observe(ctx, StageMemoryAppend, "enter", nil, time.Time{}, nil)
	err := r.agent.Memory.Append(ctx, session.ID, msg)
	r.observe(ctx, StageMemoryAppend, "leave", nil, maStart, err)
}

func toolDefinitions(tools []Tool) []FunctionDefinition {
	if len(tools) == 0 {
		return nil
	}
	defs := make([]FunctionDefinition, len(tools))
	for i, t := range tools {
		defs[i] = t.Definition()
	}
	return defs
}

// tokenizerModelID returns the canonical encoding name for token counting.
// Uses the optional TokenizerModeler interface, falling back to "gpt-4"
// (cl100k_base, which covers most modern LLMs).
func tokenizerModelID(model Model) string {
	if tm, ok := model.(TokenizerModeler); ok {
		if name := tm.TokenizerModel(); name != "" {
			return name
		}
	}
	return "gpt-4"
}

// countMessageTokens returns the token count for a message using the
// model-specific tokenizer (tiktoken). Falls back to a heuristic if the
// tokenizer is unavailable.
func countMessageTokens(modelID string, m Message) int {
	n := tokenizer.Count(modelID, m.Content)
	n += tokenizer.Count(modelID, m.ReasoningContent)
	for _, tc := range m.ToolCalls {
		n += tokenizer.Count(modelID, tc.Function.Name)
		n += tokenizer.Count(modelID, tc.Function.Arguments)
	}
	// Message formatting overhead: role prefix, JSON structure (~4 tokens).
	return n + 4
}

// countMessages returns the total token count for a set of messages.
func countMessages(modelID string, msgs []Message) int {
	n := 0
	for _, m := range msgs {
		n += countMessageTokens(modelID, m)
	}
	return n
}

// trimToContextWindow drops oldest non-system messages until the total
// estimated token count fits within the model's context window.
// System messages are always preserved. A 5% safety margin accounts for
// tokenizer estimation inaccuracy. Leading orphaned tool results are
// cleaned up after each removal.
//
// This is a last-resort protection — the compaction pipeline normally
// keeps the working set within the configured token budget. It triggers
// only when system prompts, compressed context, or large tool results
// push the total past the model's hard limit.
func trimToContextWindow(modelID string, messages []Message, window int) []Message {
	if window <= 0 || len(messages) == 0 {
		return messages
	}

	// Separate system messages (always preserved).
	var sys, rest []Message
	for _, m := range messages {
		if m.Role == RoleSystem {
			sys = append(sys, m)
		} else {
			rest = append(rest, m)
		}
	}

	// 5% safety margin — tiktoken is accurate but not exact.
	budget := window * 95 / 100
	sysTokens := countMessages(modelID, sys)
	budget -= sysTokens
	if budget <= 0 {
		budget = window / 4 // keep at least 25% for non-system
	}

	// Drop oldest non-system messages one at a time.
	for len(rest) > 2 {
		if countMessages(modelID, rest) <= budget {
			break
		}
		rest = rest[1:]
		// Clean up orphaned tool results (RoleTool without preceding
		// assistant tool_calls provides no useful context).
		for len(rest) > 0 && rest[0].Role == RoleTool {
			rest = rest[1:]
		}
	}

	// Ensure the first non-system message is a user message. Starting
	// with assistant (with or without tool_calls) violates the API's
	// conversation format — the model expects user/assistant alternation
	// beginning with user. If the first message is an assistant with
	// tool_calls, remove it and all consecutive tool results as a unit.
	for len(rest) > 0 && rest[0].Role != RoleUser {
		if rest[0].Role == RoleAssistant && len(rest[0].ToolCalls) > 0 {
			rest = rest[1:] // drop assistant
			for len(rest) > 0 && rest[0].Role == RoleTool {
				rest = rest[1:] // drop its tool results
			}
		} else {
			rest = rest[1:]
		}
	}

	result := make([]Message, 0, len(sys)+len(rest))
	result = append(result, sys...)
	result = append(result, rest...)
	return result
}

// ── Streaming + retry ──

// callModel calls the model with streaming preferred, retrying on transient errors.
func (r *runner) callModel(ctx context.Context, req ChatCompletionRequest, ch chan<- StreamEvent) (*ChatCompletionResponse, error) {
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			var re *RetryableError
			if errors.As(lastErr, &re) && re.RetryAfter > 0 {
				backoff = re.RetryAfter
			}
			if ch != nil {
				ch <- StreamEvent{Type: StreamRetrying, Error: lastErr}
			}
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, err := r.callModelOnce(ctx, req, ch)
		if err == nil {
			return resp, nil
		}
		var re *RetryableError
		if !errors.As(err, &re) {
			return nil, err
		}
		lastErr = err
	}
	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// callModelOnce tries streaming first, falls back to non-streaming.
func (r *runner) callModelOnce(ctx context.Context, req ChatCompletionRequest, ch chan<- StreamEvent) (*ChatCompletionResponse, error) {
	reader, err := r.runModel.ChatCompletionStream(ctx, req)
	if err != nil {
		return nil, err
	}
	if reader != nil {
		defer reader.Close()
		return accumulateStream(reader, ch)
	}
	// Non-streaming fallback: the model doesn't support streaming.
	// Emit the full response as a single text_delta so consumers (WebUI, TUI)
	// see output immediately rather than waiting for StreamDone.
	resp, err := r.runModel.ChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}
	if ch != nil && len(resp.Choices) > 0 {
		if rc := resp.Choices[0].Message.ReasoningContent; rc != "" {
			ch <- StreamEvent{Type: StreamThought, Text: rc}
		}
		if resp.Choices[0].Message.Content != "" {
			ch <- StreamEvent{Type: StreamTextDelta, Text: resp.Choices[0].Message.Content}
		}
	}
	return resp, nil
}

// accumulateStream drains a StreamReader, assembling the full ChatCompletionResponse.
// Sends text_delta events to ch (if non-nil) as content arrives.
func accumulateStream(reader StreamReader, ch chan<- StreamEvent) (*ChatCompletionResponse, error) {
	var (
		content      string
		reasoning    string
		finishReason string
		usage        Usage
	)
	toolAcc := make(map[int]*ToolCall)

	for reader.Next() {
		chunk := reader.Current()
		if chunk.Usage != nil {
			usage = *chunk.Usage
		}
		for _, delta := range chunk.Choices {
			content += delta.Content
			reasoning += delta.ReasoningContent
			if delta.FinishReason != "" {
				finishReason = delta.FinishReason
			}
			if ch != nil {
				if delta.ReasoningContent != "" {
					ch <- StreamEvent{Type: StreamThought, Text: delta.ReasoningContent}
				}
				if delta.Content != "" {
					ch <- StreamEvent{Type: StreamTextDelta, Text: delta.Content}
				}
			}
			for _, tcd := range delta.ToolCalls {
				tc := toolAcc[tcd.Index]
				if tc == nil {
					tc = &ToolCall{}
					toolAcc[tcd.Index] = tc
				}
				if tcd.ID != "" {
					tc.ID = tcd.ID
				}
				if tcd.Type != "" {
					tc.Type = tcd.Type
				}
				if tcd.Function.Name != "" {
					tc.Function.Name = tcd.Function.Name
				}
				tc.Function.Arguments += tcd.Function.Arguments
			}
		}
	}
	if err := reader.Err(); err != nil {
		return nil, err
	}

	var toolCalls []ToolCall
	for i := 0; i < len(toolAcc); i++ {
		if tc, ok := toolAcc[i]; ok {
			toolCalls = append(toolCalls, *tc)
		}
	}

	return &ChatCompletionResponse{
		Choices: []Choice{{
			Index:        0,
			Message:      Message{Role: RoleAssistant, Content: content, ReasoningContent: reasoning, ToolCalls: toolCalls},
			FinishReason: finishReason,
		}},
		Usage: usage,
	}, nil
}

// ── Built-in skill tools ──

func builtinSkillToolDefs() []FunctionDefinition {
	return []FunctionDefinition{builtinUseSkillDef, builtinReloadSkillsDef}
}

func (r *runner) executeUseSkill(ctx context.Context, call ToolCall) Message {
	var args struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return Message{
			Role:       RoleTool,
			ToolCallID: call.ID,
			Content:    fmt.Sprintf("use_skill: invalid arguments: %v", err),
		}
	}
	if args.Name == "" {
		return Message{
			Role:       RoleTool,
			ToolCallID: call.ID,
			Content:    "use_skill: name is required",
		}
	}

	// Idempotent: return cached
	if body, ok := r.loadedSkills[args.Name]; ok {
		return Message{Role: RoleTool, ToolCallID: call.ID, Content: body}
	}

	// Find skill in catalog
	var info SkillInfo
	found := false
	for _, s := range r.skills {
		if s.Name == args.Name {
			info = s
			found = true
			break
		}
	}
	if !found {
		return Message{
			Role:       RoleTool,
			ToolCallID: call.ID,
			Content:    fmt.Sprintf("skill %q not found in catalog. Use reload_skills to refresh.", args.Name),
		}
	}

	body, err := r.agent.SkillLoader.Load(ctx, info)
	if err != nil {
		return Message{
			Role:       RoleTool,
			ToolCallID: call.ID,
			Content:    fmt.Sprintf("failed to load skill %q: %v", args.Name, err),
		}
	}

	full := "**Directory:** " + info.Path + "\n\n" + body
	r.loadedSkills[args.Name] = full

	return Message{Role: RoleTool, ToolCallID: call.ID, Content: full}
}

func (r *runner) executeReloadSkills(ctx context.Context, call ToolCall) Message {
	skills, err := r.agent.SkillLoader.Discover(ctx)
	if err != nil {
		return Message{
			Role:       RoleTool,
			ToolCallID: call.ID,
			Content:    fmt.Sprintf("reload_skills failed: %v", err),
		}
	}

	r.skills = skills

	// Prune loaded skills that have been removed from disk
	seen := make(map[string]bool)
	for _, s := range skills {
		seen[s.Name] = true
	}
	for name := range r.loadedSkills {
		if !seen[name] {
			delete(r.loadedSkills, name)
		}
	}

	summary := fmt.Sprintf("%d skills available", len(r.skills))
	if len(r.loadedSkills) > 0 {
		names := make([]string, 0, len(r.loadedSkills))
		for name := range r.loadedSkills {
			names = append(names, name)
		}
		summary += fmt.Sprintf(", %d loaded: %v", len(r.loadedSkills), names)
	}

	return Message{Role: RoleTool, ToolCallID: call.ID, Content: summary}
}

// executeRecall handles the recall_memory builtin tool. It searches the agent's
// memory backend and returns ranked results with relevance scores.
func (r *runner) executeRecall(ctx context.Context, session Session, call ToolCall) Message {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil || args.Query == "" {
		return Message{
			Role:       RoleTool,
			ToolCallID: call.ID,
			Content:    "recall: a non-empty 'query' is required",
		}
	}

	if r.agent.Memory == nil {
		return Message{
			Role:       RoleTool,
			ToolCallID: call.ID,
			Content:    "recall: memory is not configured",
		}
	}

	results, err := r.agent.Memory.Search(ctx, session.ID, args.Query, 5)
	if err != nil {
		return Message{
			Role:       RoleTool,
			ToolCallID: call.ID,
			Content:    fmt.Sprintf("recall: search failed: %v", err),
		}
	}

	if len(results) == 0 {
		return Message{
			Role:       RoleTool,
			ToolCallID: call.ID,
			Content:    fmt.Sprintf("recall: no results found for %q", args.Query),
		}
	}

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("Found %d relevant memories for %q:\n\n", len(results), args.Query))
	for i, r := range results {
		if r.Score > 0 {
			buf.WriteString(fmt.Sprintf("%d. [score: %.2f] %s\n", i+1, r.Score, r.Message.Content))
		} else {
			buf.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Message.Content))
		}
	}

	return Message{Role: RoleTool, ToolCallID: call.ID, Content: buf.String()}
}

// executeSubAgent handles the built-in subagent tool. It creates a temporary
// sub-agent on the fly from the model-provided name/description/prompt, runs it
// with the caller's Model and Tools (minus subagent/AsTool tools), and returns
// the result. The sub-agent has no Approver and no Memory.
func (r *runner) executeSubAgent(ctx context.Context, session Session, call ToolCall, ch chan<- StreamEvent) Message {
	var args struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Prompt      string `json:"prompt"`
		Task        string `json:"task"`
	}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil || args.Name == "" || args.Prompt == "" || args.Task == "" {
		return Message{
			Role:       RoleTool,
			ToolCallID: call.ID,
			Content:    "subagent: name, prompt, and task are required",
		}
	}

	if args.Description == "" {
		args.Description = args.Task
		if len(args.Description) > 80 {
			args.Description = args.Description[:77] + "..."
		}
	}

	// Build a sub-agent from the caller's capabilities.
	sub := Agent{
		Name:         args.Name,
		Description:  args.Description,
		SystemPrompts: []string{args.Prompt},
		Model:        r.runModel,
		Tools:        stripAgentTools(r.agent.Tools),
		MaxTurns:     3,
		noSpawn:      true,
		// no Approver, no Memory — safe sub-agent
	}

	subSession := Session{
		ID:        fmt.Sprintf("%s-%d-%d", args.Name, time.Now().UnixNano(), globalAgentSeq.Add(1)),
		CreatedAt: time.Now(),
	}

	// Streaming: forward sub-agent output as tool progress events.
	if ch != nil {
		streamCh := sub.RunStream(ctx, subSession, UserMessage(args.Task))
		var buf strings.Builder
		var finalOutput string
		for evt := range streamCh {
			switch evt.Type {
			case StreamThought:
				ch <- StreamEvent{
					Type:       StreamToolProgress,
					Text:       evt.Text,
					ToolCallID: call.ID,
				}
			case StreamTextDelta:
				buf.WriteString(evt.Text)
				ch <- StreamEvent{
					Type:       StreamToolProgress,
					Text:       evt.Text,
					ToolCallID: call.ID,
				}
			case StreamError:
				errText := "unknown error"
				if evt.Error != nil {
					errText = evt.Error.Error()
				}
				return Message{
					Role:       RoleTool,
					ToolCallID: call.ID,
					Content:    fmt.Sprintf("subagent: %s", errText),
				}
			case StreamAborted:
				return Message{
					Role:       RoleTool,
					ToolCallID: call.ID,
					Content:    buf.String(),
				}
			case StreamDone:
				if evt.Result != nil && evt.Result.FinalOutput != "" {
					finalOutput = evt.Result.FinalOutput
				}
			}
		}
		if buf.Len() > 0 {
			return Message{Role: RoleTool, ToolCallID: call.ID, Content: buf.String()}
		}
		if finalOutput == "" {
			finalOutput = "(completed, no output)"
		}
		return Message{Role: RoleTool, ToolCallID: call.ID, Content: finalOutput}
	}

	// Blocking path: no event channel.
	result, err := sub.Run(ctx, subSession, UserMessage(args.Task))
	if err != nil {
		return Message{
			Role:       RoleTool,
			ToolCallID: call.ID,
			Content:    fmt.Sprintf("subagent error: %v", err),
		}
	}
	return Message{Role: RoleTool, ToolCallID: call.ID, Content: result.FinalOutput}
}

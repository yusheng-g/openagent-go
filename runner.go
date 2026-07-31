package openagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
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
	loadedSkills map[string]string    // name → body, populated by load_skill
	builtinTools []FunctionDefinition // auto-injected tools (load_skill, reload_skills)
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
		maxTurns = 100
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
	if !r.agent.NoSpawn {
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
	// Snapshot the tool set under the lock (toolsMu) before iterating; a
	// tool callback (exit_plan_mode) may AppendTools concurrently within
	// an executeTools batch. Build the definitions from the snapshot.
	allToolDefs := toolDefinitions(r.agent.SnapshotTools())
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
			// Persist tool results for every unresolved tool_call
			// so the conversation stays valid for the next turn.
			//
			// Two paths reach this handler:
			//
			// A) Direct cancel during tool execution — the
			//    assistant with tool_calls is the last message.
			//    Inject "cancelled by user" results.
			//
			// B) Approval rejection — executeTools already
			//    created rejection messages in workingMessages
			//    but appendMemory silently failed because ctx is
			//    cancelled.  Re-persist those results and do NOT
			//    inject duplicates.
			bg := context.Background()
			covered := make(map[string]bool, 4)
			for i := len(workingMessages) - 1; i >= 0; i-- {
				wm := workingMessages[i]
				if wm.Role == RoleTool {
					covered[wm.ToolCallID] = true
					r.appendMemory(bg, session, wm)
					continue
				}
				if wm.Role == RoleAssistant && len(wm.ToolCalls) > 0 {
					for _, tc := range wm.ToolCalls {
						if covered[tc.ID] {
							continue
						}
						cancelled := Message{
							Role:       RoleTool,
							ToolCallID: tc.ID,
							Content:    "cancelled by user",
						}
						r.appendMemory(bg, session, cancelled)
						if ch != nil {
							select {
							case ch <- StreamEvent{Type: StreamToolResult, Message: cancelled}:
							default:
							}
						}
						result.Messages = append(result.Messages, cancelled)
					}
					break
				}
				// user or system message — gone past this turn.
				break
			}
			runErr = ctx.Err()
			if ch != nil {
				select {
				case ch <- StreamEvent{Type: StreamAborted, Error: runErr}:
				default:
				}
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

				// Strip orphaned assistant tool_calls left by
				// cancelled turns. When OnCancel fires during tool
				// execution, the assistant message with tool_calls
				// is already in Memory but the tool results never
				// arrive. Without cleanup the model API rejects the
				// assistant(tool_calls)→user sequence as invalid.
				for len(history) > 0 && history[len(history)-1].Role == RoleAssistant && len(history[len(history)-1].ToolCalls) > 0 {
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
						select {
						case ch <- StreamEvent{Type: StreamError, Error: runErr}:
						default:
						}
					}
					return result, runErr
				}
				if gr.Tripwire {
					runErr = fmt.Errorf("input guard tripwire: %s", gr.Reason)
					r.observe(ctx, StageGuardIn, "leave", nil, giStart, runErr)
					if ch != nil {
						select {
						case ch <- StreamEvent{Type: StreamError, Error: runErr}:
						default:
						}
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

		reqBody, _ := json.Marshal(lastReq)
		slog.Debug("model request",
			"session", session.ID, "user", session.UserID,
			"model", lastReq.Model, "turn", turn, "maxTurns", maxTurns,
			"messages", len(messages), "tools", len(lastReq.Tools),
			"maxTokens", lastReq.MaxTokens, "bodyKB", len(reqBody)/1024,
			"metadata", session.Metadata)
		for i, m := range messages {
			slog.Debug("  msg", "i", i, "role", m.Role,
				"content", m.Content, "tool_calls", len(m.ToolCalls), "name", m.Name)
		}

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
				if trimmed > 0 && trimmed <= len(workingMessages) {
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
				evtType := StreamError
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					evtType = StreamAborted
				}
				select {
				case ch <- StreamEvent{Type: evtType, Error: err}:
				default:
				}
			}
			runErr = fmt.Errorf("model call: %w", err)
			return result, runErr
		}
		r.observe(ctx, StageModelCall, "leave", map[string]any{
			"tokens_prompt":     resp.Usage.PromptTokens,
			"tokens_completion": resp.Usage.CompletionTokens,
			"finish_reason":     string(resp.Choices[0].FinishReason),
			"content_preview":   truncateStr(resp.Choices[0].Message.Content, 200),
			"tool_calls":        toolCallNames(resp.Choices[0].Message.ToolCalls),
		}, mcStart, nil)
		lastResp = resp

		if len(resp.Choices) == 0 {
			runErr = fmt.Errorf("model returned no choices")
			if ch != nil {
				select {
				case ch <- StreamEvent{Type: StreamError, Error: runErr}:
				default:
				}
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
				select {
				case ch <- StreamEvent{Type: StreamToolCall, Message: Message{ToolCalls: []ToolCall{tc}}}:
				default:
				}
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
					select {
					case ch <- StreamEvent{Type: StreamError, Error: runErr}:
					default:
					}
				}
				return result, runErr
			}
			if gr.Tripwire {
				runErr = fmt.Errorf("output guard tripwire: %s", gr.Reason)
				r.observe(ctx, StageGuardOut, "leave", nil, guardOutStart, runErr)
				if ch != nil {
					select {
					case ch <- StreamEvent{Type: StreamError, Error: runErr}:
					default:
					}
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
						select {
						case ch <- StreamEvent{Type: StreamError, Error: runErr}:
						default:
						}
					}
					return result, runErr
				}
				if !gr.Allowed {
					tr.Content = fmt.Sprintf("[blocked: %s]", gr.Reason)
				}
			}
			if ch != nil {
				select {
				case ch <- StreamEvent{Type: StreamToolResult, Message: tr}:
				case <-ctx.Done():
				}
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
		select {
		case ch <- StreamEvent{Type: StreamDone, Result: result}:
		case <-ctx.Done():
		}
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
// BuildPrompt adds BEFORE the working messages. This is subtracted from the
// working token budget so that the total prompt (overhead + working) fits
// within the model's context window.
func (r *runner) estimatePromptOverhead(ctx context.Context, session Session, modelID string) int {
	var n int

	// Static context.
	static := strings.Join(r.agent.SystemPrompts, "\n\n")
	if session.ProjectContext != "" {
		static += "\n\n## Project Context\n\n" + session.ProjectContext
	}
	if static != "" {
		n += tokenizer.Count(modelID, static) + 4
	}

	// Dynamic context — same assembly order as buildPrompt.
	if len(r.skills) > 0 {
		n += tokenizer.Count(modelID, buildSkillsSection(r.skills)) + 4
	}
	for name, body := range r.loadedSkills {
		n += tokenizer.Count(modelID, "## Loaded Skill: "+name+"\n\n"+body) + 4
	}
	if session.DynamicContext != "" {
		n += tokenizer.Count(modelID, session.DynamicContext) + 4
	}

	// Semantic memory.
	if path := r.semanticMDPath(); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			if content := strings.TrimSpace(string(data)); content != "" {
				n += tokenizer.Count(modelID, "## Semantic Memory\n\n"+content) + 4
			}
		}
	}

	// Compressed summary + hints.
	if cc, err := r.agent.Memory.Compressed(ctx, session.ID); err == nil && cc != nil && cc.Summary != "" {
		n += tokenizer.Count(modelID, buildCompressedSection(cc)) + 4
	}

	return n
}

func (r *runner) buildPrompt(ctx context.Context, session Session, working []Message) []Message {
	// ── Static context (assembled once per run, never changes) ──
	static := strings.Join(r.agent.SystemPrompts, "\n\n")
	if session.ProjectContext != "" {
		static += "\n\n## Project Context\n\n" + session.ProjectContext
	}

	// ── Dynamic context (re-assembled every turn) ──
	var dynamicParts []string
	dynamicParts = append(dynamicParts, fmt.Sprintf(`

IMPORTANT: The context below is generated fresh for this turn. If it conflicts with static instructions or earlier conversation, the latest context here is authoritative. Earlier summaries, skill lists, semantic memory, or plan state may be outdated.

OS: %s
Arch: %s
Date: %s

`, runtime.GOOS, runtime.GOARCH, time.Now().Format("2006-01-02")))

	// Skills catalog + loaded skill bodies.
	if len(r.skills) > 0 {
		dynamicParts = append(dynamicParts, buildSkillsSection(r.skills))
	} else {
		dynamicParts = append(dynamicParts,
			"\nIMPORTANT: No available skills."+
				"",
		)
	}

	for name, body := range r.loadedSkills {
		dynamicParts = append(dynamicParts, "## Loaded Skill: "+name+"\n\n"+body)
	}

	// ACP / plan-mode context (injected by Session.DynamicContext).
	if session.DynamicContext != "" {
		dynamicParts = append(dynamicParts, session.DynamicContext)
	}

	// Semantic memory — persistent facts/preferences/rules, re-read every turn.
	if semanticPath := r.semanticMDPath(); semanticPath != "" {
		dynamicParts = append(dynamicParts, buildSemanticSection(semanticPath))
	}

	// Compressed conversation summary — Layer 2 of the memory model.
	if r.compressed != nil && r.compressed.Summary != "" {
		dynamicParts = append(dynamicParts, buildCompressedSection(r.compressed))
	} else {
		dynamicParts = append(dynamicParts, "## Conversation Summary\n\n(no prior conversation history)")
	}

	input := PromptInput{
		StaticContext:   static,
		DynamicContext:  strings.Join(dynamicParts, "\n\n"),
		WorkingMessages: working,
	}

	msgs, _ := r.agent.Prompt(ctx, input)
	return msgs
}

// ── Section builders ──

func (r *runner) semanticMDPath() string {
	if r.agent.Memory == nil {
		return ""
	}
	s, ok := r.agent.Memory.(interface{ SemanticMDPath() string })
	if !ok {
		return ""
	}
	return s.SemanticMDPath()
}

func buildSemanticSection(path string) string {
	var b strings.Builder
	b.WriteString("## Semantic Memory\n\n")
	b.WriteString(fmt.Sprintf("File: %s\n", path))

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if dir := filepath.Dir(path); dir != "" {
			err = os.MkdirAll(dir, 0755)
		}
		if err == nil {
			err = os.WriteFile(path, nil, 0644)
		}
	}
	if err != nil {
		// Read, create, or write failed — surface it so the section
		// isn't silently empty and the agent can react.
		b.WriteString(fmt.Sprintf("\n(error accessing %s: %v)\n", path, err))
	}

	b.WriteString("Persistent facts, preferences, and rules. One list item per line.\n")
	b.WriteString("To remember: append a line. To forget: delete the line.\n")

	if len(data) == 0 {
		b.WriteString("(empty)\n")
	} else {
		b.WriteString(strings.TrimSpace(string(data)))
	}
	return b.String()
}

func buildSkillsSection(skills []SkillInfo) string {
	var b strings.Builder
	b.WriteString("## Available Skills\n")
	b.WriteString("To load a skill, call the `load_skill` tool with the skill name.\n")
	for _, s := range skills {
		b.WriteString("\n### " + s.Name + "\n")
		for k, v := range s.Frontmatter {
			fmt.Fprintf(&b, "%s: %v\n", k, v)
		}
	}
	return b.String()
}

func buildCompressedSection(cc *CompressedContext) string {
	var b strings.Builder
	b.WriteString("## Conversation Summary\n")
	b.WriteString(cc.Summary)
	if len(cc.Hints) > 0 {
		b.WriteString("\n\n### Retrieval Hints\n")
		for i, h := range cc.Hints {
			fmt.Fprintf(&b, "%d. %s (query: %s)\n", i+1, h.Description, h.Query)
		}
	}
	return b.String()
}

func (r *runner) buildModelRequest(session Session, messages []Message) ChatCompletionRequest {
	tools := toolDefinitions(r.agent.SnapshotTools())
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
						Content:    fmt.Sprintf("this call rejected by user, reason: %s", reason),
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

	// noErr is a stable nil error value whose address is passed to
	// fireToolHooksEnd for built-in tools that have no error to report.
	// Passing a valid *error (pointing to nil) instead of a nil pointer
	// prevents downstream hooks from nil-dereferencing err.
	var noErr error

	args := json.RawMessage(call.Function.Arguments)

	// Inject session into ctx so tools and hooks can retrieve it via
	// SessionFromContext — used by artifact hooks, audit logging, etc.
	toolCtx := WithSession(ctx, session)

	// Built-in tools: execute directly, share hooks/observer pipeline.
	switch call.Function.Name {
	case "load_skill":
		toolStart := r.fireToolHooks(toolCtx, *def, args)
		msg := r.executeLoadSkill(toolCtx, call)
		r.fireToolHooksEnd(toolCtx, *def, args, &msg.Content, toolStart, &noErr)
		return msg
	case "reload_skills":
		toolStart := r.fireToolHooks(toolCtx, *def, args)
		msg := r.executeReloadSkills(toolCtx, call)
		r.fireToolHooksEnd(toolCtx, *def, args, &msg.Content, toolStart, &noErr)
		return msg
	case "recall":
		toolStart := r.fireToolHooks(toolCtx, *def, args)
		msg := r.executeRecall(toolCtx, session, call)
		r.fireToolHooksEnd(toolCtx, *def, args, &msg.Content, toolStart, &noErr)
		return msg
	case "subagent":
		toolStart := r.fireToolHooks(toolCtx, *def, args)
		msg := r.executeSubAgent(toolCtx, session, call, ch)
		r.fireToolHooksEnd(toolCtx, *def, args, &msg.Content, toolStart, &noErr)
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
		// Streaming chunks are forwarded to the client as
		// StreamToolProgress for live UX. They do NOT enter the model
		// context — only the final, post-OnToolEnd result is appended to
		// the working message history (see the append below). A redaction
		// hook therefore never needs to buffer: the model sees only the
		// redacted final result, while the user sees live (possibly
		// raw) progress, which is the point of streaming.
		toolCh := se.ExecuteStream(toolCtx, args)
		var buf strings.Builder
		// Rate-limit: some tools (shell) produce hundreds of chunks/sec.
		// Emitting every chunk as a StreamToolProgress event floods the
		// runner's event channel and deadlocks when downstream (ACP
		// stdout pipe) reads slower than the tool produces.
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		var pending string
		flush := func() {
			if pending != "" && ch != nil {
				select {
				case ch <- StreamEvent{
					Type:       StreamToolProgress,
					Text:       pending,
					ToolCallID: call.ID,
				}:
				case <-toolCtx.Done():
				}
				pending = ""
			}
		}
		done := false
		for !done {
			select {
			case chunk, ok := <-toolCh:
				if !ok {
					done = true
				} else if chunk.Error != nil {
					execErr = chunk.Error
					done = true
				} else {
					buf.WriteString(chunk.Content)
					pending += chunk.Content
				}
			case <-ticker.C:
				flush()
			case <-toolCtx.Done():
				flush()
				done = true
			}
		}
		flush()
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
// It mutates *output and *err in place: OnToolEnd receives them as pointers
// and may redact or truncate them, so we pass the caller's pointers straight
// through rather than copying by value.
//
// The observer "leave" event is emitted BEFORE OnToolEnd with the pre-hook
// err. This means an observer that records err.Error() would see the raw
// (unredacted) error. Redact does not protect the observer channel; observer
// implementations must avoid logging err/result content. See hooks/redact
// package doc.
func (r *runner) fireToolHooksEnd(ctx context.Context, def FunctionDefinition, args json.RawMessage, output *string, tc toolHookCtx, err *error) {
	var errVal error
	if err != nil {
		errVal = *err
	}
	r.observe(ctx, StageToolExecute, "leave", map[string]any{"tool": def.Name}, tc.start, errVal)
	if r.agent.Hooks != nil {
		r.agent.Hooks.OnToolEnd(ctx, def, args, output, err, tc.hookState)
	}
}

// built-in skill tool definitions — single source of truth used by both
// toolDef (name→definition lookup) and builtinSkillToolDefs (model tool list).
var (
	builtinLoadSkillDef = FunctionDefinition{
		Name:        "load_skill",
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
		Description: "Search the full message archive for exact details — commands, file names, dates, numbers, or verbatim text — that the conversation summary may have omitted. Do NOT use for general context or preferences; the conversation summary and semantic memory already cover those. Returns ranked results with relevance scores.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"Specific keywords to find (e.g. 'kubectl rollout restart', 'benchmark_2024.csv', 'port 5432')"}},"required":["query"]}`),
	}
)

func (r *runner) findTool(name string) Tool {
	for _, t := range r.agent.SnapshotTools() {
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
	case "load_skill":
		return &builtinLoadSkillDef
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

func toolCallNames(calls []ToolCall) []string {
	if len(calls) == 0 {
		return nil
	}
	names := make([]string, len(calls))
	for i, tc := range calls {
		names[i] = tc.Function.Name
	}
	return names
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
				select {
				case ch <- StreamEvent{Type: StreamRetrying, Error: lastErr}:
				default:
				}
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
		return accumulateStream(ctx, reader, ch)
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
			select {
			case ch <- StreamEvent{Type: StreamThought, Text: rc}:
			case <-ctx.Done():
			}
		}
		if resp.Choices[0].Message.Content != "" {
			select {
			case ch <- StreamEvent{Type: StreamTextDelta, Text: resp.Choices[0].Message.Content}:
			case <-ctx.Done():
			}
		}
	}
	return resp, nil
}

// accumulateStream drains a StreamReader, assembling the full ChatCompletionResponse.
// Sends text_delta events to ch (if non-nil) as content arrives. All sends are
// cancellable via ctx to prevent deadlocks when the downstream consumer is gone.
func accumulateStream(ctx context.Context, reader StreamReader, ch chan<- StreamEvent) (*ChatCompletionResponse, error) {
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
					select {
					case ch <- StreamEvent{Type: StreamThought, Text: delta.ReasoningContent}:
					case <-ctx.Done():
					}
				}
				if delta.Content != "" {
					select {
					case ch <- StreamEvent{Type: StreamTextDelta, Text: delta.Content}:
					case <-ctx.Done():
					}
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
	return []FunctionDefinition{builtinLoadSkillDef, builtinReloadSkillsDef}
}

func (r *runner) executeLoadSkill(ctx context.Context, call ToolCall) Message {
	var args struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return Message{
			Role:       RoleTool,
			ToolCallID: call.ID,
			Content:    fmt.Sprintf("load_skill: invalid arguments: %v", err),
		}
	}
	if args.Name == "" {
		return Message{
			Role:       RoleTool,
			ToolCallID: call.ID,
			Content:    "load_skill: name is required",
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
		Name:          args.Name,
		Description:   args.Description,
		SystemPrompts: []string{args.Prompt},
		Prompt:        BuildPrompt,
		Model:         r.runModel,
		Tools:         stripAgentTools(r.agent.SnapshotTools()),
		MaxTurns:      3,
		NoSpawn:       true,
		// no Approver, no Memory — safe sub-agent
	}

	subSession := Session{
		ID:        fmt.Sprintf("%s-%d-%d", args.Name, time.Now().UnixNano(), globalAgentSeq.Add(1)),
		CreatedAt: time.Now(),
	}

	// Streaming: forward sub-agent output as tool progress events.
	// Rate-limited to 1/sec to prevent pipe deadlocks when the ACP
	// transport or downstream SSE consumer reads slower than the
	// sub-agent produces text deltas. Same pattern as the StreamExecutor
	// path in executeOneToolInternal.
	if ch != nil {
		streamCh := sub.RunStream(ctx, subSession, UserMessage(args.Task))
		var buf strings.Builder
		var finalOutput string
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		var pending string
		flush := func() {
			if pending != "" {
				select {
				case ch <- StreamEvent{
					Type:       StreamToolProgress,
					Text:       pending,
					ToolCallID: call.ID,
				}:
				case <-ctx.Done():
				}
				pending = ""
			}
		}
		done := false
		for !done {
			select {
			case evt, ok := <-streamCh:
				if !ok {
					done = true
				} else {
					switch evt.Type {
					case StreamThought, StreamTextDelta:
						buf.WriteString(evt.Text)
						pending += evt.Text
					case StreamError:
						flush()
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
						flush()
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
			case <-ticker.C:
				flush()
			case <-ctx.Done():
				flush()
				if buf.Len() > 0 {
					return Message{Role: RoleTool, ToolCallID: call.ID, Content: buf.String()}
				}
				return Message{Role: RoleTool, ToolCallID: call.ID, Content: "(cancelled)"}
			}
		}
		flush()
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

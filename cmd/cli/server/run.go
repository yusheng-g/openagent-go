package server

import (
	"context"
	"fmt"
	"os"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/agent"
	"github.com/yusheng-g/openagent-go/cmd/cli/config"
	ctxpkg "github.com/yusheng-g/openagent-go/context"
	"github.com/yusheng-g/openagent-go/guard/llm"
	"github.com/yusheng-g/openagent-go/kernel"
	"github.com/yusheng-g/openagent-go/sandbox/native"
	"github.com/yusheng-g/openagent-go/summarizer"
	"github.com/yusheng-g/openagent-go/version"
)

// RunCLI runs a one-shot chat turn with streaming output to stdout.
// Same memory wiring as the servers: the conversation is persisted under a
// fresh session id (each run is its own session), and durable knowledge is
// extracted after the run and recalled across runs (user-level scope), so
// "run" participates in the knowledge closed loop.
func RunCLI(ctx context.Context, cfg *config.Config, message string) error {
	// 1. Build model from config (unexported: buildModels, resolveModel).
	// settings "model" wins, else the first configured model.
	_, modelInfos := buildModels(cfg.Provider)
	m := resolveModel(cfg.Model, modelInfos)
	if m == nil {
		return fmt.Errorf("no models configured. Please add a provider in %s", config.Path())
	}

	// 2. Memory + knowledge (same wiring as RunACP/RunREST).
	// Capabilities come from settings.json like every other mode — run is
	// not special-cased to defaults-on.
	caps := cfg.Capabilities
	ms, knowledge, _, cleanup, err := buildMemory(cfg.Embedding, caps.OnEmbedder())
	if err != nil {
		return err
	}
	defer cleanup()

	// 3. System prompts (unexported: resolveProfiles)
	prompts := resolveProfiles("")

	// 4. Sandbox + standard tools (unexported: sandboxPolicy, buildTools)
	workDir, _ := os.Getwd()
	policy := sandboxPolicy(cfg.Sandbox)
	sb, err := native.NewWithPolicy(workDir, policy)
	runToolList := []string{"shell", "read", "write", "ls", "grep", "websearch", "webfetch", "settings"}
	if caps.OnBrowser() {
		runToolList = append(runToolList, "browser")
	}
	if caps.OnOffice() {
		runToolList = append(runToolList, "office")
	}
	var tools []openagent.Tool
	if err == nil {
		tools = buildTools(sb, workDir, runToolList)
	} else {
		fmt.Fprintf(os.Stderr, "sandbox unavailable, tools disabled: %v\n", err)
	}

	// 5. Construct agent config (pure) + runtime deps.
	opts := []agent.Option{
		agent.WithModel(m),
		agent.WithSystemPrompts(prompts...),
		agent.WithMaxTurns(500),
	}
	// CLI is single-process with no model hot-reload, so a static guard
	// is correct — no dynamic lookup needed.
	if caps.OnGuard() && m != nil {
		g := llm.New(m)
		opts = append(opts, agent.WithInputGuard(g))
		opts = append(opts, agent.WithOutputGuard(g.Output()))
	}
	opts, skillProvider := buildOpts(opts, caps)
	agentCfg := agent.New(version.Name, opts...)

	holder, _, telemetryShutdown, err := setupTelemetry(ctx, *cfg)
	if err != nil {
		return fmt.Errorf("telemetry init: %w", err)
	}
	defer telemetryShutdown()

	deps := buildRuntimeDeps(caps, cfg.Sensitive, holder)
	deps.Tools = tools
	deps.SkillProvider = skillProvider
	if caps.OnMemory() {
		deps.SessionStore = ms
		deps.Compressor = ms
		deps.MemoryProvider = knowledge
		// One shared background extractor: knowledge from this run is
		// stored and recalled by later runs (and by the servers sharing
		// this db).
		deps.Extractor = ctxpkg.NewAsyncExtractor(ctxpkg.NewLLMExtractor(func() openagent.Model { return m }, knowledge))
	}
	if caps.OnMemory() && caps.OnSummarizer() && m != nil {
		sumz := summarizer.New(m).WithMaxTokens(agentCfg.MaxCompressedTokens)
		ms.WithSummarizer(sumz)
		// Share the summarizer with sub-agent children so their in-memory
		// stores get compaction parity with the parent.
		deps.Summarizer = sumz
	}

	// 6. Fresh session per run (no cross-run conversation history, but
	// durable knowledge is user-level and carries across runs).
	session := openagent.Session{
		ID:        fmt.Sprintf("cli-%d", time.Now().UnixNano()),
		CreatedAt: time.Now(),
	}

	// Emit session lifecycle events for observers (tracking, telemetry, etc.).
	if sessionObserver != nil {
		sessionObserver.OnSessionCreate(ctx, openagent.SessionLifecycleEvent{
			SessionID:   session.ID,
			EntryPoint:  "cli",
			SessionMode: "",
			CreatedAt:   session.CreatedAt,
		})
	}

	// 7. Run and stream events to terminal
	ch := kernel.New(agentCfg, deps).RunStream(ctx, session, openagent.UserMessage(message))
	var runErr error
	for evt := range ch {
		switch evt.Type {
		case openagent.StreamTextDelta:
			fmt.Print(evt.Text)

		case openagent.StreamDone:
			fmt.Println()
			if evt.Result != nil {
				u := evt.Result.Usage
				fmt.Fprintf(os.Stderr, "─── %d prompt + %d completion = %d tokens, %d turns\n",
					u.PromptTokens, u.CompletionTokens, u.TotalTokens,
					evt.Result.TurnCount)
			}

		case openagent.StreamError:
			runErr = evt.Error

		case openagent.StreamAborted:
			if evt.Error != nil {
				runErr = evt.Error
			} else {
				runErr = ctx.Err()
			}
		}
	}

	if sessionObserver != nil {
		closeEvt := openagent.SessionLifecycleEvent{
			SessionID:  session.ID,
			EntryPoint: "cli",
			DurationMs: time.Since(session.CreatedAt).Milliseconds(),
		}
		if runErr != nil {
			closeEvt.Err = runErr
		}
		// Use a fresh context — ctx may be cancelled (Ctrl+C) but we
		// still want the close event to reach the tracking endpoint.
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		sessionObserver.OnSessionClose(closeCtx, closeEvt)
	}

	return runErr
}

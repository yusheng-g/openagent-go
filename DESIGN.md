# openagent-go Architecture

> [中文](DESIGN.zh.md) | [README](README.md) | [README (中文)](README.zh.md)

## Overview

openagent-go is a **fully pluggable**, open-source AI agent framework in Go. The core is a minimal mainline loop—all capabilities are added through pluggable modules.

**Design principles:**

- Follow industry standards (OpenAI API shape); no custom protocols
- The Runner is the sole mediator — modules never call each other
- No module configured = capability absent; nil means skip that node
- Avoid code bloat: think first, build second, no speculative abstractions
- Library code never reads environment variables (that's the application layer)

**Two extension paths:**

| Path | For | Mechanism |
|------|-----|-----------|
| Compile-time | Platform developers | Implement Go interface → inject via `WithXxx()` |
| Runtime | Community / end users | Drop `.wasm` files into plugin dir → auto-loaded |

Both coexist. Compile-time interfaces are the "backbone"; runtime plugins are one implementation source for those interfaces.

---

## 8-Node Mainline Loop

```
Agent.Run(ctx, session, input)
  │
  ├─ turn 1 only:
  │   ① Memory.Compact()    ← token-based compaction (Runner-driven)
 │      Memory.Recent()    ← pure query, no side effects
  │   ③ Guard.in.Check()    ← input safety check
  │
  └─ for turn in 1..maxTurns:
      ② PromptBuilder() or defaultBuildPrompt()
         ├─ system instructions
         ├─ compressed summary + hints (auto-injected)
         ├─ skill catalog + loaded skills
         └─ working messages
      ④ Model.ChatCompletionStream()  → fallback ChatCompletion()
         ├─ 429/503 → RetryableError → exponential backoff (max 3)
         └─ StreamTextDelta real-time push
      ⑤ Guard.out.Check()
      ⑥ Approver.Approve() → Tool.Execute() (concurrent goroutines)
         └─ tool result → Guard.out re-check
      ⑧ Memory.Append()
      has tool_calls → loop back to ②
      no tool_calls → StreamDone → return
```

Each node: `if module != nil { module.Call(...) }`.

---

## Core Types

### Agent

```go
type Agent struct {
    Name, Description, Instructions string
    Model       Model
    Tools       []Tool
    Memory      Memory
    Prompt      PromptBuilder    // nil = default
    InGuard     InputGuard
    OutGuard    OutputGuard
    Approver    Approver
    Hooks       RunHooks
    Observer    RunObserver      // nil = no stage events
    SkillLoader SkillLoader
    MaxTurns    int             // default 20
    MaxWorkingTokens    int    // default 0 = 70% of context window
    MaxCompressedTokens int    // default 2048
}

agent.Run(ctx, session, input) → (*RunResult, error)
agent.RunStream(ctx, session, input) → <-chan StreamEvent
agent.RunGoal(ctx, session, goal) → (*RunResult, error)
agent.RunGoalStream(ctx, session, goal) → <-chan StreamEvent
agent.Clone() → *Agent
```

The Runner is private — `Agent.Run()` creates it internally.

### StreamEvent

```go
const (
    StreamThought      = "thought"        // reasoning content (o1, deepseek-r1)
    StreamTextDelta    = "text_delta"     // per-character output
    StreamToolCall     = "tool_call"      // tool invocation start
    StreamToolProgress = "tool_progress"  // streaming tool output chunk
    StreamToolResult   = "tool_result"    // tool result (final)
    StreamRetrying     = "retrying"       // 429 backoff in progress
    StreamDone         = "done"           // normal completion
    StreamError        = "error"          // execution failure
    StreamAborted      = "aborted"        // external interrupt (cancel/timeout)
)
```

### Session

```go
type Session struct {
    ID, UserID, ModelID   string
    Temperature, MaxTokens float64 / int
    UserProfile, ProjectContext string
    Turn                         int
    CreatedAt                    time.Time
}
```

Pure data carrier. The application layer owns CRUD. The Runner does not create Sessions.

---

## Module Interfaces

### ① Memory (Three-Layer Model)

```
Layer 1: Working    — Recent() pure query; Runner manages token budget via MaxWorkingTokens
Layer 2: Compressed — Compressed() auto-injected; Compact() incremental/rolling via Summarizer
Layer 3: Archive    — Search() + recall_memory tool; original messages NEVER deleted
```

```go
type Memory interface {
    io.Closer
    Count(ctx, sessionID) (int, error)
    Recent(ctx, sessionID, n int, offset int) ([]Message, error)         // pure query
    Compact(ctx, sessionID, throughIndex int, messages []Message) error  // Runner-driven
    Compressed(ctx, sessionID) (*CompressedContext, error)
    Search(ctx, sessionID, query string, limit int) ([]SearchResult, error)
    Append(ctx, sessionID, msg Message) error
    DeleteSession(ctx, sessionID) error
}
```

The Runner drives compaction via token budget. It counts tokens backward from the
most recent message against MaxWorkingTokens (default: 70% of model context window),
adjusts to a safe boundary (not cutting tool_call/tool_result pairs via
SafeCompressionBoundary), and calls Compact(). The backend compresses only newly
overflowed messages with the previous summary for incremental/rolling compression.
Original messages are NEVER deleted.

Implementations: `memory/file` (JSONL, zero-dependency), `memory/sqlite` (SQLite + FTS5, optional vector search via `WithEmbedder`).

### Summarizer (Memory dependency)

```go
type Summarizer interface {
    Summarize(ctx context.Context, messages []Message, previous *CompressedContext) (*CompressedContext, error)
}
```

nil = no compaction. Configured on Memory via `WithSummarizer()`.
When previous is non-nil, this is incremental/rolling compression — the implementation
should preserve existing facts and incorporate new messages.

### Embedder (Memory dependency)

```go
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float64, error)
    Dimensions() int
}
```

nil = fallback to keyword/FTS5 search. Configured on Memory via `WithEmbedder()`.

### ② PromptBuilder

```go
type PromptBuilder func(ctx context.Context, input PromptInput) ([]Message, error)

type PromptInput struct {
    AgentDescription, Instructions string
    WorkingMessages   []Message
    Compressed        *CompressedContext
    Tools             []FunctionDefinition
    AvailableSkills   []SkillInfo
    LoadedSkills      map[string]string
    UserProfile, ProjectContext string
}
```

Function type — single method, no state needed. nil = `defaultBuildPrompt()`.

### ③ / ⑤ Guard

```go
type InputGuard interface {
    Check(ctx, input GuardInput) GuardResult
}
type OutputGuard interface {
    Check(ctx, output GuardOutput) GuardResult
}

type GuardResult struct {
    Allowed  bool
    Reason   string
    Tripwire bool  // true → terminate the run
}
```

InGuard checks once before the loop. OutGuard checks every model output + every tool result. Implementation: `guard/llm` — LLM-as-judge for content safety.

### ④ Model

```go
type Model interface {
    ChatCompletion(ctx, req) (*ChatCompletionResponse, error)
    ChatCompletionStream(ctx, req) (StreamReader, error)  // nil,nil = not supported
    ContextWindow() int
}
```

Implementation: `model/openai` (openai-go v3 SDK). Streaming preferred, non-streaming fallback.

### Tool

```go
type Tool interface {
    Definition() FunctionDefinition
    Execute(ctx context.Context, args json.RawMessage) (string, error)
}

type StreamExecutor interface {
    ExecuteStream(ctx context.Context, args json.RawMessage) <-chan ToolStreamChunk
}
```

Built-in tools: `shell`, `read`, `write`, `ls`, `grep` (in `tool/` package). Auto-injected tools: `use_skill`, `reload_skills`, `recall_memory`, `subagent`. WASM tool plugins via `plugin/wasm`.

**Subagent tool:** The `subagent` built-in tool lets the model dynamically spawn temporary sub-agents at runtime — `subagent(name, description, prompt, task)`. The sub-agent runs with the caller's tools (minus subagent itself), no Approver, no Memory, limited turns. Results stream back as `StreamToolProgress` events. Safe by construction: noSpawn prevents recursion.

**AsTool():** For pre-configured agents with specific tool subsets: `coder.AsTool()` wraps the agent as a Tool. The sub-agent gets MaxTurns=3, no Approver, no Memory, noSpawn, and stripped agent-spawning tools.

**Two-phase execution:** When an Approver is configured, tools are first approved sequentially (user clicks through dialogs quickly), then approved tools execute concurrently in goroutines. Before this, tool execution was serial under approval.

### ⑥ Approver

```go
type Approver interface {
    Approve(ctx, call ToolCall, def FunctionDefinition, session Session) (allowed bool, reason string)
}
```

nil = allow all. Implementations: `cmd/tui` (bubbletea v2 Y/N), `examples/backend` (SSE dialog with Allow Once / Allow Directory).

### ⑦ RunHooks

```go
type RunHooks interface {
    OnAgentStart(ctx context.Context, req ChatCompletionRequest) (any, error)
    OnAgentEnd(ctx context.Context, req ChatCompletionRequest, resp *ChatCompletionResponse, runErr error, startState any)
    OnToolStart(ctx context.Context, tool FunctionDefinition, args json.RawMessage) (any, error)
    OnToolEnd(ctx context.Context, tool FunctionDefinition, args json.RawMessage, result *string, err *error, startState any)
}
```

Start methods return an opaque `any` value that the Runner passes to the corresponding End method. This lets implementations carry state from start to finish — OTEL creates a span in Start, defers End in the callback. slog captures `time.Now()` in Start and logs `time.Since()` in End. `result` and `err` are pointers so hooks can redact/truncate/inject metadata before memory storage.

Aligned with OpenAI Agents SDK naming. Implementations: `hooks/slog`, `hooks/otel`.

### RunObserver

```go
type RunObserver interface {
    ObserveStage(ctx context.Context, event StageEvent)
}

type StageEvent struct {
    Name     string         // "memory.fetch", "model.call", ...
    Phase    string         // "enter" or "leave"
    Detail   map[string]any // turn, tokens, tool name, ...
    Duration time.Duration  // wall-clock on "leave"
    Err      error
}
```

Per-stage enter/leave events with durations. Use for pipeline panels, tracing, monitoring. Multiple observers via `MultiObserver()`.

### Skill

```go
type SkillLoader interface {
    Discover(ctx) ([]SkillInfo, error)
    Load(ctx, skill SkillInfo) (string, error)
}
```

Workflow: Discover → inject catalog into prompt → model calls `use_skill(name)` → Load returns full body. `reload_skills` rescans and prunes removed skills. Implementation: `skill/fs`.

---

## Module Non-Interference

The Runner is the sole mediator. Modules never call each other:

```
Runner.compactIfNeeded:
  → count tokens backward → adjust boundary → Memory.Compact()
Runner.buildPrompt:
  msgs = Memory.Recent()      ← pure query, no side effects
  cc   = Memory.Compressed()
  input = PromptInput{...}
  result = PromptBuilder(input)

Runner ferries data:
  - Memory → PromptInput
  - Tool.Execute result → Memory.Append
  - Model response → Guard → Approver → Tool
```

**Key:** `Embedder` and `Summarizer` are Memory dependencies, not Agent dependencies. They are configured on Memory at construction time.

---

## Directory Structure

```
openagent-go/
├── agent.go              Agent + Run/RunStream/RunGoal/Clone + StreamEvent
├── runner.go             private runner + 8-node loop + defaultBuildPrompt
├── model.go              Model, Embedder, Summarizer interfaces + request/response types
├── message.go            Message + ContentPart (multimodal)
├── tool.go               Tool interface + FunctionDefinition + StreamExecutor
├── sandbox.go            Sandbox interface + Command/Result types
├── memory.go             Memory interface + CompressedContext
├── tokenizer/            Model-aware token counting (tiktoken)
├── prompt.go             PromptInput + PromptBuilder + RetrievalHint
├── guard.go              InputGuard / OutputGuard
├── approver.go           Approver
├── hooks.go              RunHooks
├── observer.go           RunObserver + StageEvent
├── skill.go              SkillLoader + SkillInfo
├── router.go             Router + FirstAgentRouter + LLMRouter
├── team.go               Team + TeamResult + HandoffEntry + handoffTool
├── options.go            WithXxx() AgentOption + TeamOption
├── session.go            Session
├── doc.go                Package documentation
│
├── tool/                 Built-in Tool implementations
│   ├── shell.go          Shell: OS sandbox command execution with streaming
│   ├── file.go           ReadFile / WriteFile / ListDir (path traversal protection)
│   └── grep.go           Grep: recursive file search
│
├── sandbox/native/       OS-native sandbox
│   ├── native.go         New() factory + public API
│   ├── native_darwin.go  macOS: sandbox-exec + Seatbelt profile
│   ├── native_linux.go   Linux: bwrap namespace isolation
│   └── native_windows.go Windows: stub
│
├── model/openai/         OpenAI model implementation
├── memory/file/          JSONL file memory
├── memory/sqlite/        SQLite + FTS5 + vector search
├── guard/llm/            LLM-as-judge guard
├── hooks/slog/           slog logger hooks
├── hooks/otel/           OpenTelemetry tracing hooks
├── skill/fs/             Filesystem skill loader
├── plugin/wasm/          WASM plugin runtime (wazero)
├── acp/                  ACP protocol (Agent ↔ IDE)
├── mcp/                  MCP protocol (tool interoperability)
├── plan/                 Goal → DAG → parallel execution + replan
├── eventbus/             Generic pub/sub with history replay
├── rest/                 REST API (HTTP access layer)
├── runner/acp/           ACP Runner: external agent as Team member
│
├── cmd/
│   ├── tui/              Terminal chat (bubbletea v2)
│   └── cli/              Full CLI: WASM plugin runtime, cobra commands, Rust SDK
│
├── examples/
│   ├── basic/            Non-streaming example
│   ├── stream/           Streaming example
│   ├── skill/            Skill loading example
│   ├── memory/           Memory persistence example
│   ├── hooks/            Lifecycle hooks example
│   ├── observer/         Stage observer example
│   ├── guard/            Safety guard example
│   ├── team/             Multi-agent team example
│   ├── delegate/         Agent-as-tool parallel delegation
│   ├── plugin/           WASM plugin example
│   ├── sandbox/          Sandbox demo
│   ├── backend/          Full REST + SSE API server
│   └── frontend/
│       └── vue-app/   Vue 3 SPA reference UI
│
├── DESIGN.md             Architecture (English)
├── DESIGN.zh.md          Architecture (Chinese)
└── README.md
```

All interfaces in root package. Implementations in sub-packages. No circular dependencies.

---

## Key Design Decisions

**1. Why is Runner private?** Users call `Agent.Run()`, never construct a Runner. Runner is an internal implementation detail.

**2. Why does the Runner trigger compaction?** Token budget depends on the model's context window, which only the Runner knows. The Runner counts tokens and decides when to compact; Memory just executes the compaction. Clean separation: Runner owns the decision, Memory owns the storage.

**3. Why no auto-search (RelevantFacts)?** Archive retrieval is model-driven via the `recall_memory` tool. The model decides when and what to search. Compressed context already provides passive reminders. Dual Archive search channels (auto + tool) are redundant.

**4. Why aren't Embedder/Summarizer on Agent?** They are Memory dependencies, not Agent capabilities. Memory decides whether it needs embeddings or summaries. Preserves "modules don't call each other".

**5. Why streaming by default?** `callModelOnce` prefers `ChatCompletionStream`, falls back to non-streaming. Lowest time-to-first-token.

**6. Why is PromptBuilder a function type?** One method, no state. Function types are simpler.

**7. Why is Handoff a Tool rather than Router choosing each step?** The model has full context and makes better handoff decisions than a router. Router only does two things: initial message routing and policy vetoes.

**8. Why inject hints instead of erroring on loops?** Two-layer loop detection: first give the model a hint ("you're in a loop, answer directly"), then remove transfer_to_* tools if it persists. Graceful degradation.

**9. Why independent Memory per Agent instead of shared Team Memory?** Keep it simple. Agent already supports independent Memory. Add shared memory later if needed, without breaking existing interfaces.

---

## Extension Paths

### Compile-time Extensions (Go interfaces)

| Node | Interface | Status | Notes |
|------|-----------|--------|-------|
| ①② | Memory | ✅ | file / sqlite + Compact + recall_memory |
| ① | Embedder | ✅ | nil = FTS5 fallback |
| ① | Summarizer | ✅ | nil = no compaction |
| ② | PromptBuilder | ✅ | function type, nil = default |
| ④ | Model | ✅ | OpenAI implementation |
| ⑥ | Tool | ✅ | compile-time + builtin + WASM |
| — | SkillLoader | ✅ | filesystem implementation |
| ③ | InputGuard | ✅ | guard/llm |
| ⑤ | OutputGuard | ✅ | guard/llm |
| ⑥ | Approver | ✅ | TUI + Frontend, human-in-the-loop |
| ⑦ | RunHooks | ✅ | slog + OpenTelemetry |
| — | RunObserver | ✅ | per-stage enter/leave, Runner wired |
| — | Router | ✅ | first-agent + LLM-based |
| — | Team | ✅ | multi-agent with handoff + loop detection |
| — | Plan | ✅ | goal → DAG → parallel execution + replan |
| — | EventBus | ✅ | generic pub/sub, per-session topics |

### Runtime Extensions (WASM Plugins)

```go
// No plugins: zero overhead
agent := openagent.NewAgent("bot", openagent.WithModel(model))

// With plugins:
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

mgr := wasm.NewManager("./plugins")
mgr.Discover(ctx)
mgr.OnAbort(func(reason string) { cancel() })

agent := openagent.NewAgent("bot",
    openagent.WithModel(model),
    openagent.WithTools(mgr.Tools()...),
    openagent.WithRunObserver(mgr.Observer()),
)
```

**Plugin types:**

| Type | Purpose | Injected as | ABI exports |
|------|---------|------------|-------------|
| Tool | New tools | `openagent.Tool` | `alloc`, `metadata`, `execute` |
| Stage | Stage event observation + abort | `RunObserver` | `alloc`, `metadata`, `run` |

WASM runtime: [wazero](https://github.com/tetratelabs/wazero) — pure Go, zero CGO. One `.wasm` file per plugin.

---

## Team (Multi-Agent Orchestration)

Team is an orchestration layer above the single-agent loop. Handoff = Tool with `EndTurn: true`. Each agent has independent Memory, Tools, and Guard.

```go
team := openagent.NewTeam(
    openagent.WithTeamAgent("researcher", "analyzes questions", researcher),
    openagent.WithTeamAgent("calculator", "performs math", calculator),
)
result, _ := team.Run(ctx, session, input)
stream := team.RunStream(ctx, session, input)
```

**Loop detection (layered):**

| Layer | Detection | Action |
|-------|-----------|--------|
| L1 Ping-pong | A→B→A→B pattern | Inject hint |
| L2 Frequency | Same agent ≥3 times | Inject hint |
| L3 Hard limit | Hint followed by another handoff | Remove transfer_to_* tools |

No hard error — graceful degradation.

### Router

```go
type AgentInfo struct {
    Name        string
    Description string
    Type        AgentType // AgentInternal or AgentExternal
}

type Router interface {
    Route(ctx, input, agents) (string, error)
    CanHandoff(ctx, entry, chain, session) error
}
```

`AgentInfo.Type` auto-populated: `WithTeamAgent` → `AgentInternal`, `AddAgent` with ACP runner → `AgentExternal`. Flows to Router, Team prompt, Plan planner, and frontend.

### Agent as Tool (Parallel Delegation)

```go
coordinator := openagent.NewAgent("coordinator",
    openagent.WithTools(researcher.AsTool(), writer.AsTool()),
)
```

Three-layer isolation: new session per call, no coordinator history leaked, only the task string as input.

---

## Plan (DAG Goal Decomposition)

```go
p := plan.NewPlan(
    plan.WithPlanner(plan.NewLLMPlanner(model)),
    plan.WithAgent("coder", "writes code", coderAgent),
    plan.WithAgent("reviewer", "reviews code", reviewerAgent),
)
result, _ := p.Run(ctx, session, "Build a REST API for todos")
```

| | Team | Plan |
|---|------|------|
| Decision | Runtime, agent-initiated handoff | Pre-execution, Planner generates DAG |
| Parallelism | None (serial handoff chain) | Topological batches auto-parallel |
| Failure | Agent handles itself | Subtree replan with LLM |
| Use case | Conversational collaboration | Structured execution pipelines |

`Planner` generates a DAG from the goal. `Executor` topo-sorts into batches, runs each batch with goroutines, auto-replans on failure (up to `MaxReplans` times). `AutoReplan=false` pauses on failure for manual retry/replan.

**Frontend UI:** The Plan page provides a full workflow — input a goal, watch the LLM stream its thinking via `plan_thinking` events, then review the rendered DAG. Pre-execution actions: `[Execute]` `[Replan]` `[Clear]`. Replan accepts natural language feedback to regenerate the DAG before execution begins. During execution, step cards update in real-time with status colors and expandable output. Failed steps offer `[Retry]` and `[Replan]` with feedback.

## Goal (Autonomous Mode)

```go
agent.RunGoal(ctx, session, "Fix all failing tests")
```

Unlike `Run()` where the input is a user message that may scroll out of context, `RunGoal()` injects the goal into the system prompt — it persists across all turns. The agent iterates autonomously: plan → execute → evaluate → continue until done or impossible.

| | Run | RunGoal | Plan.Run |
|---|---|---|---|
| Goal placement | User message (scrolls out) | System prompt (persistent) | Planner-generated DAG |
| Behavior | One-shot Q&A | Autonomous iteration | DAG parallel execution |
| Stops when | No more tool calls | Goal achieved or impossible | All steps complete |

## EventBus

```go
bus := eventbus.New[MyEvent](500) // max 500 history events per session
sub := bus.Subscribe(sessionID)   // subscribe (auto-replays history)
bus.Publish(sessionID, evt)       // fanout to all subscribers
```

Generic pub/sub with per-session topics and history replay. Used by the frontend for multi-tab sync, plan progress, and pipeline panel events.

## Sandbox

OS-native security: macOS Seatbelt (`sandbox-exec`), Linux Bubblewrap (`bwrap`), Windows stub.

```go
sb, _ := native.New("./workspace")
// File tools: direct host I/O with path traversal protection
// Shell tool: fork → OS sandbox → exec
```

Three-layer security: file tools validate paths → shell tool runs inside OS sandbox → workspace boundary enforced at both levels.

---

## Pipeline Panel (Observability)

The Runner emits `StageEvent` at each of the 8 nodes (enter/leave). A `RunObserver` implementation bridges these to the frontend, where an sidebar Monitor panel renders live:

- 7 nodes: Fetch → Guard-In → Prompt → Model → Guard-Out → Tool → Store
- Status: gray (pending) → blue pulse (active) → green (done + duration) → red (error)

- Info bar: round counter + token usage

The observer adds `turn`/`maxTurns` and `tokens_prompt`/`tokens_completion` to `model.call` detail so the frontend shows per-round progress.

---

## Comparison

| | openai-agents | Claude Code | openagent-go |
|---|---|---|---|
| Sandbox | Docker SDK + macOS sandbox-exec | seccomp + namespaces | macOS Seatbelt / Linux bwrap |
| File tools | read/write/ls/rm/mkdir | Read/Write/Glob | ReadFile/WriteFile/ListDir/Grep |
| Streaming | PTY-based | Bash tool | Shell tool (line streaming, no PTY) |
| Multi-agent | Handoff chain | — | Team (handoff) + Plan (DAG parallel) |
| Goal mode | — | `/goal` | RunGoal + Plan.Run |
| Observability | — | — | RunObserver + SVG pipeline panel |
| Plugins | — | — | WASM (wazero, zero CGO) |

---

## UI Examples

- `cmd/cli/` — CLI tool: `openagent run "msg"` and `openagent goal "task"`, streaming output
- `cmd/tui/` — bubbletea v2 terminal chat, streaming + Y/N approval
- `examples/backend/` — Full REST + SSE API server (single, team, plan)
- `examples/frontend/vue-app/` — Vue 3 SPA with Chat, Team, Plan modes, streaming, DAG, pipeline monitor

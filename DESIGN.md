# openagent-go Architecture

> [中文](DESIGN.zh.md) | [README](README.md) | [README (中文)](README.zh.md)

## Overview

openagent-go is a **fully pluggable**, open-source AI agent framework in Go. The core is a minimal mainline loop—all capabilities are added through pluggable modules.

**Design principles:**

- Follow industry standards (OpenAI API shape, ACP v1 protocol); no custom protocols
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
         ├─ system prompts (static: SystemPrompts)
         ├─ dynamic context (per-turn: plan entries, mode)
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

**Two-layer prompt model:**

| Layer | Source | Content |
|-------|--------|---------|
| Static | `Agent.SystemPrompts` + `Description` | Fixed instructions, set at construction |
| Dynamic | `Session.DynamicContext` | Plan entries, mode instructions — rebuilt each turn |

The Runner passes `Session.DynamicContext` through to `PromptInput` → `defaultBuildPrompt`. The ACP layer builds it from session runtime state.

---

## Core Types

### Agent

```go
type Agent struct {
    Name, Description string
    SystemPrompts   []string   // static system prompts (replaces single Instructions)
    Model           Model
    Tools           []Tool
    Memory          Memory
    Prompt          PromptBuilder    // nil = default
    InGuard         InputGuard
    OutGuard        OutputGuard
    Approver        Approver
    Hooks           RunHooks
    Observer        RunObserver      // nil = no stage events
    SkillLoader     SkillLoader
    MaxTurns        int             // default 20
    MaxWorkingTokens    int         // default 0 = 70% of context window
    MaxCompressedTokens int         // default 2048
    ReasoningEffort    string       // "none","minimal","low","medium","high","xhigh"
}

agent.Run(ctx, session, input) → (*RunResult, error)
agent.RunStream(ctx, session, input) → <-chan StreamEvent
agent.RunGoal(ctx, session, goal) → (*RunResult, error)
agent.RunGoalStream(ctx, session, goal) → <-chan StreamEvent
agent.Clone() → *Agent
```

The Runner is private — `Agent.Run()` creates it internally. `Clone()` returns a shallow copy with an independent Tools backing array; used by `AgentServer.agentForTurn()` for per-session isolation.

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
    ID, UserID, ModelID     string
    Temperature, MaxTokens  float64 / int
    UserProfile, ProjectContext string
    DynamicContext              string  // per-turn plan + mode context (ACP layer builds)
    Turn                        int
    CreatedAt                   time.Time
    Metadata                    map[string]any
}
```

Pure data carrier. The application layer owns CRUD. The Runner does not create Sessions.

---

## Module Interfaces

### ① Memory (Three-Layer Model)

```
Layer 1: Working    — Recent() pure query; Runner manages token budget via MaxWorkingTokens
Layer 2: Compressed — Compressed() auto-injected; Compact() incremental/rolling via Summarizer
Layer 3: Archive    — Search() + recall tool; original messages NEVER deleted
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
most recent message against MaxWorkingTokens (default: 70% of model context window,
minus fixed prompt overhead from `estimatePromptOverhead`), adjusts to a safe
boundary (not cutting tool_call/tool_result pairs via SafeCompressionBoundary), and
calls Compact(). The backend compresses only newly overflowed messages with the
previous summary for incremental/rolling compression. Original messages are NEVER deleted.

Implementations: `memory/file` (JSONL, zero-dependency), `memory/sqlite` (SQLite + FTS5,
CJK tokenizer, optional vector search via `WithEmbedder`).

### Summarizer (Memory dependency)

```go
type Summarizer interface {
    Summarize(ctx context.Context, messages []Message, previous *CompressedContext) (*CompressedContext, error)
}
```

nil = no compaction. Configured on Memory via `WithSummarizer()`. Implementation:
`summarizer/llm.go` — LLM-based incremental compression using the agent's Model.

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
    DynamicContext    string   // per-turn plan + mode info
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
`ChatCompletionRequest` carries `ReasoningEffort` — passed through from `Agent.ReasoningEffort`
to the model's `reasoning_effort` parameter (OpenAI o-series, Anthropic extended thinking).

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

Built-in tools: `shell`, `read`, `write`, `ls`, `grep` (in `tool/` package). Auto-injected tools:
`load_skill`, `reload_skills`, `recall`, `subagent`. ACP RPC tools: `read_client_file`,
`write_client_file`, `terminal_create`/`output`/`wait`/`kill`/`release` (Agent→Client).
Plan tools: `plan_create`, `plan_update` (LLM outputs structured plan entries via function-calling).

**Subagent tool:** The `subagent` built-in tool lets the model dynamically spawn temporary sub-agents at runtime — `subagent(name, description, prompt, task)`. The sub-agent runs with the caller's tools (minus subagent itself), no Approver, no Memory, limited turns.

**Two-phase execution:** When an Approver is configured, tools are first approved sequentially, then approved tools execute concurrently.

### Sandbox

```go
type Sandbox interface {
    Run(ctx context.Context, cmd Command) (Result, error)
    CWD() string    // working dir from the tool's perspective (may differ from host path)
}
```

`CWD()` returns the path as seen from inside the sandbox — `/workspace` under bwrap, host path otherwise. Shell tool reads its working directory from the sandbox rather than carrying its own. Implementation: `sandbox/native` (Linux bwrap, macOS Seatbelt).

### ⑥ Approver

```go
type Approver interface {
    Approve(ctx, call ToolCall, def FunctionDefinition, session Session) (allowed bool, reason string)
}
```

nil = allow all. ACP mode bridges to the client via `session/request_permission` RPC.
Implementations: `cmd/tui` (bubbletea v2 Y/N), `examples/backend` (SSE dialog).

### ⑦ RunHooks

```go
type RunHooks interface {
    OnAgentStart(ctx context.Context, req ChatCompletionRequest) (any, error)
    OnAgentEnd(ctx context.Context, req ChatCompletionRequest, resp *ChatCompletionResponse, runErr error, startState any)
    OnToolStart(ctx context.Context, tool FunctionDefinition, args json.RawMessage) (any, error)
    OnToolEnd(ctx context.Context, tool FunctionDefinition, args json.RawMessage, result *string, err *error, startState any)
}
```

Start methods return an opaque `any` value passed to the corresponding End method. Implementations: `hooks/slog`, `hooks/otel`.

### RunObserver

```go
type RunObserver interface {
    ObserveStage(ctx context.Context, event StageEvent)
}
```

Per-stage enter/leave events with durations. Multiple observers via `MultiObserver()`.

### Skill

```go
type SkillLoader interface {
    Discover(ctx) ([]SkillInfo, error)
    Load(ctx, skill SkillInfo) (string, error)
}
```

Implementation: `skill/fs`.

---

## ACP v1 Protocol

The agent speaks the Agent Client Protocol natively. An `AgentServer` wraps
an `openagent.Agent` as an ACP-compliant handler:

```go
agent := openagent.NewAgent("bot", ...)
srv := acp.NewAgentServer(agent, mem, store, models)
server := openacpsdk.NewServer("openagent-acp", "1.0.0", srv)
server.Run(ctx)  // blocks on stdin/stdout
```

**Protocol layers:**

| Layer | Package | Role |
|-------|---------|------|
| Types | `acp/sdk/` | ACP v1 schema — 958 lines, zero dependencies |
| Transport | `acp/sdk/` | JSON-RPC 2.0 over stdio — mux, client session, Agent→Client RPC |
| Integration | `acp/server.go` | AgentServer — session CRUD, prompt turns, plan mode, MCP, slash commands |

**ACP modes:** `chat` (conversational with tools) and `plan` (structured planning via `plan_create`/`plan_update` tools). Model selection via config options — multiple models from the registry surfaced as select options.

---

## Plan Mode

Plan mode uses `plan_create` and `plan_update` tools — the LLM outputs structured
plan entries directly via function-calling arguments. No separate code path is needed.

```
User goal → agent.RunStream
  → agent calls plan_create(goal, steps[{id, content, priority}])
  → plan text enters conversation context
  → agent calls plan_update(updates[{id, status}]) as it progresses
  → plan entries persisted in SessionStore._meta["plan"]
  → each turn: DynamicContext injects current plan state into system prompt
```

**Orchestrate** (`orchestrate/`) is separate — multi-agent DAG decomposition + parallel
execution. Not ACP plan mode; used by the REST API for goal→DAG→execute pipelines.

---

## Slash Commands

Server-side slash commands intercepted before they reach the agent:

```
/help      — list available commands
/mode      — switch session mode (chat/plan)
/model     — list or switch models
/context   — show token usage
/cwd       — show working directory
/clear     — reset session messages
/rename    — rename session title
/sessions  — list all sessions
```

Commands are registered via `slash/` Registry and dispatched from `OnPrompt`. Unknown
`/` commands fall through to the agent for natural language handling.

---

## Directory Structure

```
openagent-go/
├── agent.go              Agent + Run/RunStream/RunGoal/Clone + StreamEvent
├── runner.go             private runner + 8-node loop + defaultBuildPrompt
├── model.go              Model, Embedder, Summarizer interfaces + request/response types
├── message.go            Message + ContentPart (multimodal)
├── tool.go               Tool interface + FunctionDefinition + StreamExecutor
├── sandbox.go            Sandbox interface + Command/Result types + CWD()
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
├── session.go            Session (+ DynamicContext)
├── doc.go                Package documentation
│
├── tool/                 Built-in Tool implementations
│   ├── shell.go          Shell: OS sandbox command execution with streaming (CWD from sandbox)
│   ├── file.go           ReadFile / WriteFile / ListDir (path traversal protection)
│   ├── grep.go           Grep: recursive file search
│   ├── acp_fs.go         ACPReadFile / ACPWriteFile (Agent→Client RPC)
│   └── acp_terminal.go   ACPTerminal (create/output/wait/kill/release)
│
├── plan/                 Plan mode tools
│   ├── entry.go          Entry + Priority + Status types
│   └── tool.go           CreateTool (plan_create) + UpdateTool (plan_update)
│
├── slash/                Slash command registry
│   └── slash.go          Registry, Context, Command, Handler
│
├── summarizer/           LLM-based compression
│   └── llm.go            Compressor (implements Summarizer)
│
├── sandbox/native/       OS-native sandbox
│   ├── native.go         New() factory + CWD() + public API
│   ├── native_darwin.go  macOS: sandbox-exec + Seatbelt profile
│   ├── native_linux.go   Linux: bwrap namespace isolation
│   └── native_windows.go Windows: stub
│
├── acp/                  ACP protocol integration
│   ├── sdk/              ACP v1 SDK (types, JSON-RPC 2.0 mux, client)
│   │   ├── types.go      958-line ACP v1 schema
│   │   ├── server.go     Mux + AgentHandler interface + SessionEventSender
│   │   ├── client.go     Client + Session + EventHandler
│   │   └── doc.go        Package reference
│   ├── server.go         AgentServer (Agent → ACP handler)
│   └── commands.go       Built-in slash command registry
│
├── orchestrate/          Multi-agent DAG decomposition + execution
├── model/openai/         OpenAI model implementation
├── memory/file/          JSONL file memory
├── memory/sqlite/        SQLite + FTS5 + CJK tokenizer + vector search
├── guard/llm/            LLM-as-judge guard
├── hooks/slog/           slog logger hooks
├── hooks/otel/           OpenTelemetry tracing hooks
├── skill/fs/             Filesystem skill loader
├── plugin/wasmhost/      Shared WASM host layer (keyring, HTTP, logging, utc_now)
├── plugin/agent/wasm/    Agent WASM plugin runtime
├── plugin/cli/           CLI plugin host + WASM runtime
├── plugin/cli/wasm/      CLI WASM loader, observer hub, command runner
├── plugin/pdk/rust/      Plugin SDK (Rust crate for plugin authors)
├── mcp/                  Model Context Protocol client
├── eventbus/             Generic pub/sub with history replay
├── session/              Session metadata types + persistent Store
│   ├── sqlite/           SQLite session store
│   └── file/             File-based session store
├── rest/                 REST API (HTTP access layer)
│
├── cmd/
│   ├── tui/              Terminal chat (bubbletea v2)
│   └── cli/              Full CLI: cobra commands, WASM plugin runtime
│       ├── main.go       Entry point + plugin load + command dispatch
│       ├── config/       Settings + provider configuration
│       ├── keyring/      System keyring integration
│       ├── server/       Server runners (ACP, REST) + shared agent setup
│       └── examples/plugin/  CLI plugin examples (telemetry, settings, commands)
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
│   ├── acp/              ACP protocol examples (server + client)
│   ├── iac/              Multi-agent IaC pipeline
│   ├── backend/          Full REST + SSE API server
│   └── frontend/
│       └── vue-app/      Vue 3 SPA reference UI
│
├── DESIGN.md             Architecture (English)
├── DESIGN.zh.md          Architecture (Chinese)
└── README.md
```

All interfaces in root package. Implementations in sub-packages. No circular dependencies.

---

## Key Design Decisions

**1. Why is Runner private?** Users call `Agent.Run()`, never construct a Runner. Runner is an internal implementation detail.

**2. Why does the Runner trigger compaction?** Token budget depends on the model's context window, which only the Runner knows. The Runner counts tokens and decides when to compact; Memory just executes the compaction.

**3. Why no auto-search?** Archive retrieval is model-driven via the `recall` tool. The model decides when and what to search.

**4. Why aren't Embedder/Summarizer on Agent?** They are Memory dependencies. Preserves "modules don't call each other".

**5. Why streaming by default?** `callModelOnce` prefers `ChatCompletionStream`, falls back to non-streaming.

**6. Why a function type for PromptBuilder?** One method, no state.

**7. Why is Handoff a Tool?** The model has full context and makes better handoff decisions than a router.

**8. Why clone for agentForTurn?** `s.Agent` is shared across all sessions. Clone creates an isolated copy so per-turn overrides (Approver sessionID binding, tools, ReasoningEffort) don't race.

**9. Why ToolFactory for per-turn tool creation?** Tools need the session's cwd, which differs from the process cwd (Docker containers, bwrap). Creating them at startup would bind the wrong path.

**10. Why DynamicContext on Session?** Plan entries and mode change every turn. The Runner shouldn't know about ACP or plans — Session is the neutral transport channel.

---

## Extension Paths

### Compile-time Extensions (Go interfaces)

| Node | Interface | Status | Notes |
|------|-----------|--------|-------|
| ①② | Memory | ✅ | file / sqlite + Compact + recall |
| ① | Embedder | ✅ | nil = FTS5 fallback |
| ① | Summarizer | ✅ | summarizer/llm.go |
| ② | PromptBuilder | ✅ | function type, nil = default |
| ④ | Model | ✅ | OpenAI + ReasoningEffort |
| ⑥ | Tool | ✅ | compile-time + builtin + WASM + ACP RPC |
| — | SkillLoader | ✅ | filesystem implementation |
| ③ | InputGuard | ✅ | guard/llm |
| ⑤ | OutputGuard | ✅ | guard/llm |
| ⑥ | Approver | ✅ | TUI + ACP permission bridge |
| ⑦ | RunHooks | ✅ | slog + OpenTelemetry |
| — | RunObserver | ✅ | per-stage enter/leave |
| — | Router | ✅ | first-agent + LLM-based |
| — | Team | ✅ | multi-agent with handoff |
| — | Orchestrate | ✅ | LLM-driven DAG + parallel + replan |
| — | EventBus | ✅ | generic pub/sub |
| — | Slash | ✅ | command registry + dispatch |

### Runtime Extensions (WASM Plugins)

**Plugin types:**

| Type | Purpose | Injected as | ABI exports |
|------|---------|------------|-------------|
| `agent:tools` | Add new tools to the agent's tool set | `openagent.Tool` | `alloc`, `metadata`, `execute` |
| `agent:observers` | Observe pipeline stages, abort runs | `RunObserver` | `alloc`, `metadata`, `run` |
| `cli:settings` | Inject provider credentials, modify settings JSON | `init()` transformation | `alloc`, `metadata`, `init` |
| `cli:commands` | Add custom cobra sub-commands | `run_<name>()` handler | `alloc`, `metadata`, `commands`, `run_<name>` |
| `cli:observers` | Lifecycle event logging | `on_startup`, `on_shutdown`, `on_command_start/end` | `alloc`, `metadata`, event handlers |

WASM runtime: [wazero](https://github.com/tetratelabs/wazero) — pure Go, zero CGO.

---

## Team (Multi-Agent Orchestration)

```go
team := openagent.NewTeam(
    openagent.WithTeamAgent("researcher", "analyzes questions", researcher),
    openagent.WithTeamAgent("calculator", "performs math", calculator),
)
result, _ := team.Run(ctx, session, input)
```

Handoff = Tool with `EndTurn: true`. Each agent has independent Memory, Tools, and Guard.
Loop detection: ping-pong pattern detection, frequency counter, hard limit with tool removal.

### Orchestrate (LLM-Driven DAG Execution)

```go
p := orchestrate.NewPlan(
    orchestrate.WithPlanner(orchestrate.NewLLMPlanner(model)),
    orchestrate.WithAgent("coder", "writes code", coderAgent),
    orchestrate.WithAgent("reviewer", "reviews code", reviewerAgent),
)
result, _ := p.Run(ctx, session, "Build a REST API for todos")
```

| | Team | Orchestrate |
|---|------|------------|
| Decision | Runtime, agent-initiated | Pre-execution, LLM generates DAG |
| Parallelism | None | Topological batches auto-parallel |
| Failure | Agent handles itself | Subtree replan with LLM |

---

## Sandbox

OS-native security: macOS Seatbelt (`sandbox-exec`), Linux Bubblewrap (`bwrap`).

```go
sb, _ := native.New("./workspace")
cwd := sb.CWD()  // "/workspace" under bwrap, host path otherwise
```

Three-layer security: file tools validate paths → shell tool runs inside OS sandbox → workspace boundary enforced at both levels.

---

## Comparison

| | openai-agents | Claude Code | openagent-go |
|---|---|---|---|
| Protocol | — | — | ACP v1 (JSON-RPC 2.0) |
| Sandbox | Docker SDK + macOS sandbox-exec | seccomp + namespaces | macOS Seatbelt / Linux bwrap |
| File tools | read/write/ls | Read/Write/Glob | ReadFile/WriteFile/ListDir/Grep |
| Streaming | PTY-based | Bash tool | Shell tool (line streaming) |
| Multi-agent | Handoff chain | — | Team + Orchestrate |
| Plan mode | — | Tool-based | plan_create/plan_update tools |
| Observability | — | — | RunObserver + StageEvent |
| Plugins | — | — | WASM (wazero, zero CGO) |
| Slash commands | — | — | Registry + built-ins + extensible |
| Memory compression | — | — | LLM incremental summarizer |

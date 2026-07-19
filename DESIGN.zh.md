# openagent-go Architecture

> [English](DESIGN.md) | [README](README.md) | [README (中文)](README.zh.md)

## 概述

openagent-go 是一个**全系统可插拔**的开源 AI Agent。核心是一条极简的主线 loop，所有能力通过插拔模块叠加。

**设计原则：**

- 遵循业界标准（OpenAI API），不自定义协议
- Runner 是唯一中介者，模块之间不互相调用
- 没插模块 = 没有那个能力，nil 则跳过对应节点
- 避免代码膨胀：先想清楚再写，不做预判式抽象
- 库代码不使用环境变量，环境变量属于应用层

**两种扩展方式：**

| 方式 | 适用场景 | 机制 |
|------|---------|------|
| **编译时扩展** | 平台开发者 | 实现 Go 接口 → `WithXxx()` 注入 |
| **运行时扩展** | 社区/终端用户 | 插件文件放入目录 → Agent 动态加载 |

两者并存，不互斥。编译时接口是"主干"，运行时插件是接口的一种实现来源。

---

## 主线 8 节点

```
Agent.Run(ctx, session, input)
  │
  ├─ turn 1 only:
  │   ① Memory.Compact()    ← Token 驱动的增量压缩（Runner 决策）
 │      Memory.Recent()    ← 纯查询，无副作用
  │   ③ Guard.in.Check()    ← 输入安全检查
  │
  └─ for turn in 1..maxTurns:
      ② PromptBuilder() 或 defaultBuildPrompt()
         ├─ system instructions
         ├─ compressed summary + hints（自动注入）
         ├─ skill catalog + loaded skills
         └─ working messages
      ④ Model.ChatCompletionStream()  → fallback ChatCompletion()
         ├─ 429/503 → RetryableError → 指数退避重试（max 3）
         └─ StreamTextDelta 实时推送
      ⑤ Guard.out.Check()
      ⑥ Approver.Approve() → Tool.Execute() (并发 goroutine)
         └─ tool result → Guard.out 再次检查
      ⑧ Memory.Append()
      has tool_calls → 回到 ②
      无 tool_calls → StreamDone → 返回
```

每个节点：`if module != nil { module.Call(...) }`。

**运行时 Stage 插件可以挂在任意节点前后**（见"扩展路径"章节）。

---

## 核心类型

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
    MaxWorkingTokens    int    // default 0 = 模型上下文窗口的 70%
    MaxCompressedTokens int    // default 2048
}

// 非流式：阻塞返回完整结果
agent.Run(ctx, session, input) → (*RunResult, error)

// 流式：返回 channel，实时推送 delta
agent.RunStream(ctx, session, input) → <-chan StreamEvent

// Goal mode：goal 持久注入 system prompt，Agent 自主迭代直到完成
agent.RunGoal(ctx, session, goal) → (*RunResult, error)
agent.RunGoalStream(ctx, session, goal) → <-chan StreamEvent

// Clone：安全可修改的浅拷贝，Tools 底层数组独立
agent.Clone() → *Agent
```

Runner 是私有类型，`Agent.Run()` 内部创建。用户不直接接触 Runner。

### StreamEvent

```go
type StreamEventType string
const (
    StreamThought      StreamEventType = "thought"       // 推理内容 (o1, deepseek-r1)
    StreamTextDelta    StreamEventType = "text_delta"     // 逐字符输出
    StreamToolCall     StreamEventType = "tool_call"      // 工具调用开始
    StreamToolProgress StreamEventType = "tool_progress"  // 流式工具输出 chunk
    StreamToolResult   StreamEventType = "tool_result"    // 工具结果（最终）
    StreamRetrying     StreamEventType = "retrying"       // 429 重试中
    StreamDone         StreamEventType = "done"           // 正常完成
    StreamError        StreamEventType = "error"          // 执行失败
    StreamAborted      StreamEventType = "aborted"        // 外部中断（cancel/timeout）
)
```

### Session

```go
type Session struct {
    ID, UserID, ModelID   string
    Temperature, MaxTokens         float64 / int
    UserProfile, ProjectContext    string
    Turn                           int
    CreatedAt                      time.Time
}
```

纯数据载体，应用层管理 CRUD。Runner 不创建 Session。

---

## 模块接口

### ① Memory（三层模型）

```
Layer 1: Working    — Recent() 纯查询；Runner 按 MaxWorkingTokens 管理 token 预算
Layer 2: Compressed — Compressed() 自动注入 + Compact() Token 驱动增量压缩
Layer 3: Archive    — Search() + recall_memory 工具；消息永不删除
Layer 2: Compressed — Compressed() + Compact()；Runner 按 token 驱动增量压缩
Layer 3: Archive    — Search() 全文/向量检索 + recall_memory 工具；消息永不删除
```

```go
type Memory interface {
    io.Closer
    Count(ctx, sessionID) (int, error)
    Recent(ctx, sessionID, n int, offset int) ([]Message, error)                                      // 纯查询
    Compact(ctx, sessionID string, throughIndex int, messages []Message) error              // Runner 驱动的压缩
    Compressed(ctx, sessionID) (*CompressedContext, error)
    Search(ctx, sessionID, query string, limit int) ([]SearchResult, error)
    Append(ctx, sessionID, msg Message) error
    DeleteSession(ctx, sessionID) error
}
```

**Runner 驱动压缩**。Runner 从末尾向前数 token，对 MaxWorkingTokens
（默认：模型上下文窗口的 70%）进行截断，调整到安全边界（不切断 tool_call/tool_result 对），
然后调用 Compact()。后端仅压缩新溢出的消息，与之前的摘要进行增量/滚动压缩。
原始消息永不删除。

实现：
- ✅ `memory/file` — JSONL 文件，零依赖，子串搜索
- ✅ `memory/sqlite` — SQLite + FTS5，可选向量搜索（`WithEmbedder`）

### Summarizer（Memory 的依赖）

```go
type Summarizer interface {
    Summarize(ctx context.Context, messages []Message, previous *CompressedContext) (*CompressedContext, error)
}
```

nil = 不压缩。Memory 通过 `WithSummarizer()` 配置。✅ `model/openai/summarizer.go`
当 previous 非 nil 时，为增量（滚动）压缩——实现应保留已有事实并融入新消息。

### Embedder（Memory 的依赖）

```go
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float64, error)
    Dimensions() int
}
```

nil = 降级为关键词/FTS5 搜索。Memory 通过 `WithEmbedder()` 配置。✅ `model/openai/embedder.go`

### ② PromptBuilder

```go
type PromptBuilder func(ctx context.Context, input PromptInput) ([]Message, error)

type PromptInput struct {
    AgentDescription, Instructions string
    WorkingMessages   []Message
    Compressed        *CompressedContext
    RelevantFacts     []string
    Tools             []FunctionDefinition
    AvailableSkills   []SkillInfo
    LoadedSkills      map[string]string
    UserProfile, ProjectContext string
}
```

nil = `defaultBuildPrompt()`：system → compressed → facts → skills → working messages。函数类型而非 struct——只有一个方法，不需要状态。

### ③ / ⑤ Guard

```go
type InputGuard interface {
    Check(ctx context.Context, input GuardInput) GuardResult
}
type OutputGuard interface {
    Check(ctx context.Context, output GuardOutput) GuardResult
}

type GuardResult struct {
    Allowed  bool
    Reason   string
    Tripwire bool  // true → 终止 run
}
```

InGuard 在循环前只检查一次。OutGuard 检查每次 model 输出 + 每个 tool result。

实现：
- ✅ `guard/llm` — LLM-as-judge，对齐 OpenAI Moderations / Llama Guard 模式。调用独立的 judge 模型判断内容安全。一个 Guard 同时实现 InputGuard 和 OutputGuard（通过 `Output()` 返回适配器）。默认 prompt 覆盖 prompt injection、越狱、PII 泄漏、有害内容。

### ④ Model

```go
type Model interface {
    ChatCompletion(ctx, req) (*ChatCompletionResponse, error)
    ChatCompletionStream(ctx, req) (StreamReader, error)
    ContextWindow() int
}
```

✅ `model/openai`（openai-go v3 SDK）。流式优先，fallback 非流式。

### Tool

```go
type Tool interface {
    Definition() FunctionDefinition
    Execute(ctx context.Context, args json.RawMessage) (string, error)
}
```

对标 OpenAI Function Calling。内置 skill 工具（`use_skill`、`reload_skills`）由 Runner 自动注入。支持编译时注入（`WithTools`）和运行时 WASM 插件加载（`plugin/agent/wasm`）。

### ⑥ Approver

```go
type Approver interface {
    Approve(ctx context.Context, call ToolCall, def FunctionDefinition, session Session) (allowed bool, reason string)
}
```

nil = 全部放行。
	
	实现：
	- `cmd/tui/approver.go` — bubbletea v2 TUI，Y/N 键盘交互
	- `examples/backend/main.go` — Web UI，SSE + 浏览器弹窗确认
	- 两种实现共享同一模式：同步 `Approve()` → channel → 异步 UI → channel → 返回

### ⑦ RunHooks（生命周期观测）

```go
type RunHooks interface {
    OnAgentStart(ctx context.Context, req ChatCompletionRequest) (any, error)
    OnAgentEnd(ctx context.Context, req ChatCompletionRequest, resp *ChatCompletionResponse, runErr error, startState any)
    OnToolStart(ctx context.Context, tool FunctionDefinition, args json.RawMessage) (any, error)
    OnToolEnd(ctx context.Context, tool FunctionDefinition, args json.RawMessage, result *string, err *error, startState any)
}
```

命名对齐 OpenAI Agents SDK RunHooks。纯观测，不影响执行流程。用于日志、追踪、计费、指标采集。

实现：
- ✅ `hooks/slog` — `log/slog` 标准库，零外部依赖
- ✅ `hooks/otel` — OpenTelemetry tracing

### RunObserver（阶段级观测）

```go
type RunObserver interface {
    ObserveStage(ctx context.Context, event StageEvent)
}

type StageEvent struct {
    Name     string        // "memory.fetch", "prompt.build", "model.call" ...
    Phase    string        // "enter" or "leave"
    Detail   map[string]any // optional metadata
    Duration time.Duration // wall-clock on "leave"
    Err      error
}
```

Runner 在 8 节点每个阶段前后各调一次，传递 enter/leave 事件。用户可实现接口对接任意后端（文件、HTTP 上报告警、指标平台）。

与 RunHooks 分工：
- RunHooks：agent/tool 生命周期级（OpenAI 对齐）
- RunObserver：8 节点阶段级（运维排障）

多个 observer 可通过 `MultiObserver(observers...)` 合并，或直接用 `WithRunObservers(observers...)`：
```go
agent := openagent.NewAgent("bot",
    openagent.WithRunObservers(
        wasmPluginObserver,  // WASM 阶段插件
        otelObserver,        // OpenTelemetry tracing
    ),
)
```

✅ 接口已实现，Runner 已埋点。

### Skill

```go
type SkillLoader interface {
    Discover(ctx) ([]SkillInfo, error)
    Load(ctx, skill SkillInfo) (string, error)
}

type SkillInfo struct {
    Name        string
    Description string
    Frontmatter map[string]any
    Path        string         // 绝对路径，模型可引用目录内脚本
}
```

工作流：
1. Discover → 注入 AvailableSkills 到 prompt（全部 frontmatter 字段）
2. 模型调 `use_skill(name)` → Load 返回 `**Directory:** path\n\n{body}`
3. 可调 `reload_skills` → 重新扫描 → 自动清理已删除 skill

✅ `skill/fs`（`root/*/SKILL.md`，YAML frontmatter）。

---

## 模块不互相调用

Runner 是唯一中介：

```
Runner.compactIfNeeded:
  → 从末尾向前数 token → 调整安全边界 → Memory.Compact()
Runner.buildPrompt:
  msgs = Memory.Recent()      ← 纯查询，无副作用
  cc   = Memory.Compressed()  ← 自动注入摘要
  input = PromptInput{...}
  result = PromptBuilder(input)

Runner 搬运数据：
  - Memory → PromptInput
  - Tool.Execute result → Memory.Append
  - Model response → Guard → Approver → Tool
```

**关键：** `Embedder` 和 `Summarizer` 是 Memory 的依赖，不是 Agent 的依赖。Agent 不持有它们。Memory 在构造时配置。

---

## 目录结构

```
openagent-go/
├── agent.go              Agent + Run/RunStream + StreamEvent
├── runner.go             private runner + 8-node loop + defaultBuildPrompt
├── model.go              Model, Embedder, Summarizer 接口 + 请求/响应类型
├── message.go            Message + ContentPart（多模态）
├── tool.go               Tool 接口 + FunctionDefinition
├── tool_shell.go         ShellTool（已移除 → tool/ 包）
│
├── tool/                 内置 Tool 实现
│   ├── shell.go          Shell：OS 沙箱执行命令 + 流式输出
│   └── file.go           ReadFile / WriteFile / ListDir（路径穿越防护）
│
├── sandbox.go            Sandbox 接口 + Command/Result 类型
├── sandbox/
│   └── native/
│       ├── native.go         公共逻辑 + New() 工厂
│       ├── native_darwin.go  macOS：sandbox-exec + Seatbelt profile
│       ├── native_linux.go   Linux：bwrap (Bubblewrap) namespace 隔离
│       └── native_windows.go Windows：Stub
│
├── memory.go             Memory 接口 + CompressedContext
├── prompt.go             PromptInput + PromptBuilder + RetrievalHint
├── guard.go              InputGuard / OutputGuard

├── guard/
│   └── llm/
│       └── guard.go          LLM-as-judge 实现
├── approver.go           Approver
├── hooks.go              RunHooks
├── skill.go              SkillLoader + SkillInfo
├── router.go             Router 接口 + FirstAgentRouter + LLMRouter
├── team.go               Team + TeamResult + HandoffEntry + handoffTool
├── options.go            WithXxx() AgentOption + TeamOption
├── session.go            Session

├── model/
│   └── openai/
│       ├── openai.go         Model 实现（openai-go v3 + streaming）
│       ├── embedder.go       Embedder 实现
│       └── summarizer.go     Summarizer 实现

├── hooks/
│   ├── slog/
│   │   └── hooks.go         slog 实现（零依赖）
│   └── otel/
│       └── hooks.go         OpenTelemetry 实现
├── observer.go           RunObserver 接口 + StageEvent

├── memory/
│   ├── file/
│   │   └── memory.go         JSONL 文件存储（零依赖）
│   └── sqlite/
│       └── memory.go         SQLite + FTS5 + 向量搜索

├── skill/
│   └── fs/
│       └── loader.go         文件系统 Skill Loader
│
├── plugin/
│   └── wasm/
│       ├── abi.go            ABI 类型定义 + 常量
│       ├── loader.go         wazero runtime + 模块加载
│       ├── manager.go        Manager: Discover + Tools() + Observer()
│       ├── tool.go           wasmTool → Tool 适配器
│       └── stage.go          wasmStage → 阶段事件 dispatch
│
├── acp/                      ACP 协议（Agent ↔ IDE/编辑器）
│   ├── server.go             Server + AgentHandler 接口 + agentBridge
│   ├── client.go             连接外部 ACP agent 子进程
│   └── types.go              请求/响应类型 + EventHandler
│
├── mcp/                      MCP 协议（工具互通）
│   ├── server.go             暴露 openagent.Tool 为 MCP tools
│   ├── client.go             导入外部 MCP server tools
│   └── convert.go            FunctionDefinition ↔ MCP Tool schema 转换
│
├── orchestrate/              Orchestrate：LLM 驱动 DAG 目标分解 → 并发执行
│   ├── types.go              PlanDef, StepDef, PlanState, StepResult, PlanConfig, PlanEvent
│   ├── planner.go            Planner 接口 + LLMPlanner（LLM 生成 DAG）
│   ├── executor.go           拓扑排序 + 并行批次 + Step 执行 + 摘要 + Replan
│   ├── validate.go           DAG 校验（Kahn 环路检测 + 引用完整性）
│   └── orchestrate.go        Plan 类型 + Run/RunStream + PlanOption
│
├── eventbus/                 泛型 pub/sub 事件总线（多 tab 同步）
│   └── bus.go                Bus[T]：per-session topic + history replay
│
├── session/                  会话元数据类型与持久化 Store
│   ├── types.go              SessionInfo（标准字段 + time.Time + GetMeta/SetMeta）
│   ├── store.go              Store 接口（Save/Get/List/Delete/Close）
│   ├── sqlite/               SQLite session store
│   └── file/                 文件 session store
│
├── plugin/wasmhost/           共享 WASM host 层（ABI + host 模块）
├── plugin/agent/wasm/         Agent 插件 WASM runtime
├── plugin/cli/                CLI 插件 host + WASM runtime
├── plugin/sdk/rust/           插件 SDK（Rust crate，供插件作者使用）
│
├── acp/                       ACP 协议集成（Agent ↔ IDE）
│   ├── sdk/                   ACP v1 SDK（纯标准库，零依赖）
│   └── server.go              AgentServer（openagent Agent → ACP AgentHandler）
│
├── rest/                     REST API（HTTP 访问层）
│   ├── handler.go            Handler：单 Agent 会话 CRUD + SSE 对话 + 审批
│   ├── team.go               TeamHandler：多 Agent Team 编排 + 动态 agent 管理
│   ├── orchestrate_handler.go OrchestrateHandler：Orchestrate 会话 CRUD + 生成/编辑/执行 + SSE
│   ├── types.go              请求/响应类型 + SSEEvent + SessionDetail
│   ├── session.go             Web session 状态管理 + idle 驱逐
│   └── sse.go                SSE 工具（writeSSE, setSSEHeaders）

├── cmd/
│   ├── tui/
│   │   ├── main.go           入口 + Agent 创建（bubbletea v2）
│   │   ├── model.go           viewport + textarea + 状态机
│   │   └── approver.go       TUIApprover: channel 桥接
│   └── cli/
│       └── main.go            CLI（run + goal 命令，流式输出）

└── examples/
    ├── basic/                非流式示例
    ├── stream/               流式示例
    ├── skill/                技能加载示例
    ├── memory/               记忆持久化示例
    ├── hooks/                生命周期观测示例
    ├── observer/             阶段级观测示例
    ├── guard/                安全守卫示例
    ├── team/                 多 Agent Team 编排示例
    ├── plugin/               WASM 插件示例（+ tool/stage 子目录）
    ├── sandbox/              沙箱 Demo：OS 隔离 + 流式 + 路径防护
    ├── backend/              完整 REST + SSE API 服务
│   └── frontend/
│       └── vue-app/       Vue 3 SPA 参考前端
```

所有接口在根包，实现在子包。无环依赖。

---

## 关键设计决策

**1. Runner 为什么是私有的？**

用户调用 `Agent.Run()`，不直接构造 Runner。Runner 是内部实现细节。

**2. 为什么 Runner 触发压缩？**

压缩触发基于 token 预算，而 token 预算取决于模型上下文窗口——只有 Runner 有这个信息。Runner 用 tiktoken 精确计数，Memory 只负责执行。关注点分离：决策在 Runner，存储在 Memory。

**3. 为什么砍掉自动搜索（RelevantFacts）？**

Archive 的检索由模型通过 `recall_memory` 工具驱动——模型自己决定何时搜、搜什么。Compressed 已经是被动提醒。两个 Archive 搜索通道是冗余的。

**4. 为什么 Embedder/Summarizer 不在 Agent 上？**

它们是 Memory 的依赖，不是 Agent 的能力。Agent 只需要知道有没有 Memory。Memory 决定是否需要嵌入或摘要能力。保持"模块不互相调用"。

**4. 流式为什么是默认？**

`callModelOnce` 优先 `ChatCompletionStream`，失败 fallback 到非流式。`RunStream()` 返回 channel 给 TUI/Web，`Run()` 内部流式收集后返回完整结果。时间到首 token 最短。

**5. 为什么 PromptBuilder 是函数类型而不是 struct？**

只有一个方法，不需要状态。函数类型更简洁，default = `defaultBuildPrompt`。

**6. 为什么 Handoff = Tool 而不是 Router 做每次选择？**

模型有完整上下文，能做出比 Router 更好的交接决策。Router 只做两件事：首条消息路由（用户没说找谁）和策略否决（阻止不安全的交接）。默认 Router 不干预。

**7. 为什么不报错而是注入 hint + 强制最终输出？**

环路检测分两层：先给模型 hint（"你陷入循环了，请直接回答"），如果模型仍然 handoff，摘掉所有 transfer_to_* tools 让模型被迫给出答案。优雅降级，用户体验最好。

**8. 为什么每个 Agent 有独立 Memory 而不是 Team 共享？**

保持简单，不做预判。Agent 本身已经支持独立 Memory。如果以后需要跨 Agent 共享上下文，再加 Team.SharedMemory，不影响现有接口。

---

## 扩展路径

### 状态标记

| 标记 | 含义 |
|------|------|
| ✅ | 已实现 |
| 🔧 | 接口已定义，无实现 |
| 📋 | 规划中 |

### 编译时扩展（Go 接口）

所有模块通过实现 Go 接口、`WithXxx()` 注入即可替换。

| 节点 | 接口 | 状态 | 说明 |
|------|------|------|------|
| ①② | Memory | ✅ | file / sqlite 两种实现 |
| ①内 | Embedder | ✅ | nil = 关键词降级 |
| ①内 | Summarizer | ✅ | nil = 不压缩 |
| ② | PromptBuilder | ✅ | 函数类型，nil = default |
| ④ | Model | ✅ | openai 实现 |
| ⑥ | Tool | ✅ | 编译时注入 + builtin skill 工具 |
| Skill | SkillLoader | ✅ | fs 实现 |
| ③ | InputGuard | ✅ | `guard/llm` LLM-as-judge |
| ⑤ | OutputGuard | ✅ | `guard/llm` LLM-as-judge |
| ⑥ | Approver | ✅ | TUI (bubbletea v2) + Frontend (Vue SPA)，human-in-the-loop |
| ⑦ | RunHooks | ✅ | slog + OpenTelemetry 两种实现 |
| — | RunObserver | ✅ | 阶段级 enter/leave 事件，Runner 已埋点 |
| — | Router | ✅ | 首条消息路由 + handoff 策略门 |
| — | Team | ✅ | 多 Agent 编排，handoff + 环路检测 |
| — | Orchestrate | ✅ | LLM 驱动 DAG → 并发执行 + Replan |
| — | EventBus | ✅ | 泛型 pub/sub，per-session topic + history replay |
| — | 运行时插件 | ✅ | WASM Tool + Stage 插件，wazero runtime |

### Hooks 实现说明

- **Hooks 实现**：对接 OpenTelemetry（trace/metrics）、结构化日志（slog/zap）、计费系统。Hooks 只能观测不能修改数据；如需修改数据，用 Stage 运行时插件。

### 运行时扩展（插件）

✅ 已实现。

编译时扩展要求用户写 Go 代码并重新编译。运行时扩展面向社区和终端用户——放 `.wasm` 文件到插件目录，启动时动态加载。

**核心原则：插件系统本身也可插拔。** Agent 上没有 PluginManager 字段。用户显式创建 Manager + 声明目录 → 插件加载。不创建 = 零开销。

```go
// 没插件：Agent 不知道插件的存在
agent := openagent.NewAgent("bot", openagent.WithModel(model))

// 有插件：
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

mgr := wasm.NewManager("./plugins")
mgr.Discover(ctx)
mgr.OnAbort(func(reason string) { cancel() }) // stage 插件 abort → 取消 run

agent := openagent.NewAgent("bot",
    openagent.WithModel(model),
    openagent.WithTools(mgr.Tools()...),       // Tool 插件 → []Tool
    openagent.WithRunObserver(mgr.Observer()),  // Stage 插件 → RunObserver
)
```

**两种插件类型：**

| 类型 | 做什么 | 插入方式 | ABI |
|------|--------|---------|-----|
| Tool 插件 | 提供新工具 | 转换为 `openagent.Tool`，注入 Agent.Tools | 导出 `alloc`, `metadata`, `execute` |
| Stage 插件 | 阶段事件观测 + abort | 转换为 `RunObserver`，匹配 stage+phase 后触发 | 导出 `alloc`, `metadata`, `run` |

**WASM ABI（每个 .wasm 模块）：**

| 导出函数 | 签名 | 说明 |
|---------|------|------|
| `alloc` | `(size: i32) -> i32` | 分配 guest 内存，返回偏移量 |
| `metadata` | `() -> i64` | 返回 packed (ptr<<32 \| len) → JSON |
| `execute` | `(ptr: i32, len: i32) -> i64` | Tool：输入 JSON args → packed output |
| `run` | `(ptr: i32, len: i32) -> i64` | Stage：输入 JSON event → packed output |

**插件编写：** 任何可编译为纯 WASM（零导入）的语言均可。推荐 Rust `#![no_std]` + `wasm32-unknown-unknown` target — 天然零运行时，无需 WASI。示例见 `examples/plugin/tool/echo.rs` 和 `examples/plugin/stage/logger.rs`。

**加载机制：** `wazero`（纯 Go WASM runtime，零 CGO）。一个文件一个插件，扫描 `plugins/` 目录下所有 `.wasm`。目录不存在或无文件 → Manager 空转，Tools/Observer 返回 nil。

**Stage 插件 vs RunHooks 的区别：**

| | RunHooks (Go 接口) | Stage 插件 |
|---|---|---|
| 能力 | agent/tool 生命周期观测 | 8 节点阶段观测 + abort |
| 加载 | 编译时 | 运行时（WASM）|
| 定位 | 平台功能（日志/追踪/计费） | 用户/社区扩展 |
| 语言 | Go | 任意（Rust/C/Zig/WAT...）|

### Team（多 Agent 编排）

✅ 已实现。

Team 是高于单个 Agent loop 的编排层。每个子 Agent 独立配置 Memory、Tools、Guard——Team 只搬运数据，不介入 Agent 内部 loop。

**Handoff = Tool（`EndTurn: true`）：** Team 自动为每个 Agent 注入 `transfer_to_<name>` 工具，标记为 `EndTurn`。模型调用后 agent loop **立即终止**，Team 取**第一个** handoff（对齐 OpenAI Agents SDK `NextStepHandoff` 语义）。

**Team 感知：** 每个 Agent 的 system prompt 自动拼接 Team Context 块——队里有谁、各自能力、交接历史。

**上下文传递：** 每个 Agent 有独立 Memory。交接时新 Agent 拿到上家的 handoff message（模型自己写的）作为输入。

```go
type Team struct {
    agents      map[string]*agentEntry
    router      Router
    maxHandoffs int  // default 10
}

// TeamOption:
//   WithTeamAgent(name, description, agent)
//   WithTeamRouter(r)
//   WithTeamMaxHandoffs(n)
//   WithTeamObserver(o)

func NewTeam(opts ...TeamOption) *Team
func (t *Team) Run(ctx context.Context, session Session, input Message) (*TeamResult, error)
func (t *Team) RunStream(ctx context.Context, session Session, input Message) <-chan TeamEvent

type TeamResult struct {
    FinalOutput  string
    HandoffChain []HandoffEntry
    TotalTurns   int
    Usage        Usage
}

type HandoffEntry struct {
    From    string
    To      string
    Message string
}

// TeamEvent 类型（RunStream 实时推送）：
const (
    TeamAgentStart // agent 开始执行
    TeamAgentEnd   // agent 结束（含 error）
    TeamHandoff    // handoff 发生
    TeamTextDelta  // 逐 token 文本
    TeamToolCall   // 非 handoff 工具调用（审批用）
    TeamToolResult // 工具结果
    TeamRetrying   // 重试中
    TeamDone       // team 完成
    TeamError      // 错误
)
// transfer_to_* 工具调用不会推送给前端（内部机制）。
```

**环路检测（分层）：**

| 层级 | 检测 | 动作 |
|------|------|------|
| L1 乒乓 | A→B→A→B 连续模式 | 注入 hint：告诉模型它在循环，请直接输出结果 |
| L2 频率 | 同一 Agent 出现 ≥3 次 | 注入 hint：告知模型已多次接手，做最终决定 |
| L3 硬上限 | hint 后仍然交接 | 移除 transfer_to_* tools，强制当前 agent 最终输出 |

不报错终止——优雅降级，摘掉 handoff 工具让模型被迫给出答案。

### Router

✅ 已实现。

Router 有两个职责：

```go
type AgentInfo struct {
    Name        string
    Description string
    Type        AgentType // AgentInternal (in-process) or AgentExternal (ACP/MCP)
}

type Router interface {
    // 首条消息路由：用户输入来了，给哪个 Agent？
    Route(ctx context.Context, input Message, agents []AgentInfo) (string, error)

    // Handoff 策略门：模型想交接，Router 可以否决。
    // nil 返回 = 放行。返回 error = 否决（error 消息注入为 hint）。
    CanHandoff(ctx context.Context, entry HandoffEntry, chain []HandoffEntry, session Session) error
}
```

`AgentInfo.Type` 在 Team 构造 `AgentInfo` 时自动填充（`WithTeamAgent` 设 `AgentInternal`，`AddAgent` 根据 runner 类型自动判断）。类型信息已流通到 Router、Team prompt（显示 `[internal]` / `[external]`）、Orchestrate Planner 和前端。

**设计原则：** 模型自主驱动 handoff（transfer_to_* tools），Router 默认不干预。Router 只做 veto，不做选择。

**默认实现：**
- `FirstAgentRouter`：所有消息路由到第一个 Agent，从不否决 handoff
- `LLMRouter`：用 judge model 判断意图，选择最匹配的 Agent。CanHandoff 始终放行

nil Router = FirstAgentRouter。

### Agent as Tool（并行派发）

✅ 已实现。

Handoff（`transfer_to_*`）是**串行链**：A → B → C。`Agent.AsTool()` 支持**并行派发**：coordinator 在同一轮调多个 sub-agent，它们并发执行，结果汇总后 coordinator 继续。

```go
coordinator := openagent.NewAgent("coordinator",
    openagent.WithTools(
        researcher.AsTool(),
        writer.AsTool(),
    ),
)
```

**与 handoff 的区别：**

| | handoff (`transfer_to_X`) | delegate (`agent.AsTool()`) |
|---|---|---|
| EndTurn | `true` → loop 立即终止 | `false` → 普通 tool，coordinator 继续 |
| 语义 | "我搞不定了，你来" | "帮我把这个子任务做了" |
| 并行 | 不支持（串行链） | 天然支持（tool 并发执行） |
| 结果 | 交给新 agent，原 agent 退出 | 结果还给 coordinator，coordinator 综合 |

**三层隔离：**

| | coordinator | sub-agent |
|---|---|---|
| Memory | 自己的 session | 新 session，互不可见 |
| Context | 完整对话链 | 只有系统指令 + 分配的任务 |
| Tools | coordinator 的工具集 | sub-agent 的工具集 |

`AsTool()` 不改 Tool 接口、不改 Runner、不改 Team。40 行 wrapper：`agentTool` 实现 `Tool`，`Execute()` 中创建新 session + `Agent.Run()` → 返回 `FinalOutput`。

### REST API（HTTP 访问层）

✅ 已实现。`rest/` 包提供可复用的 HTTP REST API，对标 `acp/` 的协议层定位。

**单 Agent（[rest.Handler](rest/handler.go)）**：

```go
handler := rest.NewHandler(agent)
handler.Register(mux) // POST/GET/DELETE /sessions, POST /sessions/{id}/chat, /approve
```

**Team 编排（[rest.TeamHandler](rest/team.go)）**：

```go
th := rest.NewTeamHandler(mem,
    rest.TeamAgentTemplate{Name: "researcher", Description: "...", Agent: researcher},
    rest.TeamAgentTemplate{Name: "writer", Description: "...", Agent: writer},
)
th.Register(mux) // 同上 + GET/POST/DELETE /team/sessions/{id}/agents
```

**设计要点**：
- 会话元数据在内存（lost on restart），消息持久化由 Memory 负责
- SSE 流式对话（`text_delta`, `tool_call`, `tool_result`, `tool_approval`, `done`, `error`）
- Team 模式额外支持：`agent_start`, `agent_end`, `handoff`
- 审批通过 channel 桥接：Agent（同步）↔ SSE（异步）↔ HTTP POST
- 所有 handler 不绑定端口/TLS/认证——上层决定

**三种协议对比**：

| | ACP | MCP | REST |
|---|---|---|---|
| 传输 | stdio | HTTP/stdio | HTTP |
| 用途 | IDE/编辑器集成 | 工具互通 | Web/移动端访问 |
| 方向 | 双向 Agent↔Client | Client→Server | Client→Server |
| 包 | `acp/` + `runner/acp/` | `mcp/` | `rest/` |

### Orchestrate（LLM 驱动 DAG 编排执行）

✅ 已实现。`orchestrate/` 包提供 LLM 驱动目标分解 → DAG 依赖图 → 拓扑排序并行执行 → 失败自动 Replan。

Orchestrate 是继 Team 之后的第二种多 Agent 编排模式：

| | Team | Orchestrate |
|------|------|------|
| 决策时机 | 运行时，Agent 自主 Handoff | 执行前，LLM 生成 DAG |
| 并行 | 无（串行 handoff 链） | 拓扑批次自动并行 |
| 失败处理 | Agent 自己兜底 | 局部子树 Replan |
| 确定性 | 低（Agent 可能不 Handoff） | 高（LLM 生成 DAG 结构） |
| 适用 | 对话式协作（客服分流、讨论） | 结构化执行（代码生成、数据处理流水线） |

**核心类型：**

```go
type StepDef struct {
    ID         string
    Agent      string
    Task       string
    DependsOn  []string
    Final      bool
    MaxRetries int
}

type PlanDef struct {
    Goal  string
    Steps []StepDef
}
```

**Planner：**

```go
type Planner interface {
    Plan(ctx, goal, agents, history) (*PlanDef, error)
}

// LLMPlanner：调 LLM 生成 DAG JSON → Validate() → 最多 3 轮修正
planner := orchestrate.NewLLMPlanner(model)
```

**Executor：**

```
PlanDef → 拓扑排序（Kahn）→ 分批 →
  批次内 goroutine 并行执行 Step
  同批任一步骤失败 → batchCancel() 取消其余
  Step 执行：StepContext（前置结果摘要）→ Agent.RunWithPrefix()
  LLM Summarizer：压缩 Step 输出为摘要传给下游
  失败 → 根据 AutoReplan 配置决定行为（见下）
  全部完成 → PlanResult
```

**PlanConfig：**

```go
type PlanConfig struct {
    StepTimeout    time.Duration // 单步超时（default 5m）
    MaxReplans     int           // 自动 replan 上限（default 3）
    MaxSteps       int           // Plan 最大步骤数（default 20）
    MaxConcurrency int           // 同批最大并发（default 8）
    AutoReplan     bool          // true=失败自动 replan（default）; false=暂停等待人工介入
}
```

**终止信号区分：**

Runner 的 9 个退出路径都发送终端事件，调用方可精确区分：

| 事件 | 含义 |
|------|------|
| `StreamDone` | 正常完成（携带 `*RunResult`） |
| `StreamError` | 执行失败（guard 拦截、模型无返回等） |
| `StreamAborted` | 外部中断（context.Canceled / DeadlineExceeded） |

**数据传递（Step 间隐式传递）：**

每个 Step 执行完 → LLM 生成 2-3 句摘要 → 下游 Step 的 prompt 包含所有前置 Step 的摘要 + 原始输出。不做显式 I/O 端口绑定——Planner 只需声明依赖关系，Executor 自动处理数据流。

**Replanner（两种模式）：**

| 模式 | 触发条件 | 行为 |
|------|---------|------|
| **Auto** (`AutoReplan=true`) | Step 失败 | 自动定位受影响子树 → 局部重规划 → 合并 → 继续（最多 MaxReplans 次） |
| **Manual** (`AutoReplan=false`) | Step 失败 | 发送 `plan_waiting_retry` 事件 → 暂停 → 等待用户 Retry 或 Replan with feedback |

```
AutoReplan=true:
  Step X 失败
    → 找到所有传递依赖 X 的 Step（受影响子树）
    → 成功 Step 不受影响
    → Planner 只重新规划受影响子树
    → 合并回原 DAG，继续执行

AutoReplan=false (REST/Frontend 默认):
  Step X 失败
    → plan_waiting_retry 事件 → 前端显示 [🔄 Retry] [💬 Replan]
    → Retry: 重置 Step → 重新执行 → 成功则继续下游
    → Replan: 用户输入自然语言 feedback → Planner 基于当前 plan + feedback 重新规划
    → SSE 连接在 pause 期间保持打开，resume 后继续推送事件
```

**Step 隔离：**

每个 Step 使用独立 Session（`sessionID + "/steps/" + stepID`），Step 之间不通过 Memory 泄露内部历史。

**Step 内 Tool 调用可见：**

`PlanEvent` 携带 `ToolName`、`ToolArgs`、`ToolID` 字段。执行期间 tool 调用实时推送到前端 popup，展示工具名、格式化的 arguments JSON、以及结果文本（超 600 字符截断）。

**DAG 校验（[orchestrate/validate.go](orchestrate/validate.go)）：**

- Kahn 算法环路检测
- Agent 引用存在性
- 自依赖检测
- 重复 ID / 重复依赖
- 可达性检查

**Orchestrate 用法：**

```go
p := orchestrate.NewPlan(
    orchestrate.WithPlanner(plan.NewLLMPlanner(model)),
    orchestrate.WithAgent("coder", "Writes code", coderAgent),
    orchestrate.WithAgent("reviewer", "Reviews code", reviewerAgent),
    orchestrate.WithMaxConcurrency(8),
    orchestrate.WithAutoReplan(false), // 可选：失败时暂停等人工介入
)

// 一步：规划 + 执行
result, err := p.Run(ctx, session, "Write a REST API")

// 两步：先审阅计划再执行
def, err := p.Plan(ctx, "Write a REST API", nil)
// 展示 def 给用户，用户可编辑...
result, err := p.Execute(ctx, session, def)

// 流式
for evt := range p.RunStream(ctx, session, goal, nil) { ... }

// 暂停后恢复（AutoReplan=false）
state := &PlanState{...}
for evt := range p.ExecuteWithState(ctx, session, def, state) { ... }

// 带 feedback 的 Replan
newDef, err := p.ReplanWithFeedback(ctx, def, state, failedID, feedback)
```

**PlanEvent 类型：**

| 事件 | 说明 |
|------|------|
| `plan_generated` | Planner 生成 PlanDef |
| `plan_approved` | 用户批准执行 |
| `step_start` | Step 开始执行 |
| `step_text_delta` | Step 流式输出 token |
| `step_tool_call` | Step 调用工具（携带 tool name + args） |
| `step_tool_result` | 工具返回结果 |
| `step_done` | Step 完成（携带 summary） |
| `step_failed` | Step 失败（携带 error） |
| `replanning` | Replan 进行中 |
| `plan_waiting_retry` | 暂停，等待用户 Retry/Replan（AutoReplan=false 时） |
| `plan_done` | Plan 全部完成 |
| `plan_error` | 致命错误 |

**Frontend Orchestrate 页面：**

Plan 页面输入 goal → 按钮显示 "生成执行计划..." → `plan_thinking` 事件流式展示 LLM 思考过程 → Planner 生成 DAG → 渲染为步骤卡片网格。

Pre-execution 按钮：`[Execute] [Replan] [Clear]`

- **Execute**: 触发执行，DAG 卡片实时更新状态（pending→running→done/failed）
- **Replan**: 展开反馈输入框，填入自然语言意见 → 带 feedback 重新调用 Planner 生成新 DAG
- **Clear**: 清除 DAG，回到空白状态

执行期间：

- 卡片边框颜色反映状态，展开可查看 Agent 输出和 tool 调用记录
- 失败步骤显示 `[Retry] [Replan]` 操作按钮
- **Retry**: 重置失败步骤，重新执行
- **Replan**: 输入自然语言 feedback → Planner 基于当前 plan + feedback 重新规划受影响子树 → 合并回 DAG 继续执行
- `plan_waiting_retry` 事件暂停执行，用户在对话框中输入反馈后点击 Replan 恢复
```go
ph := rest.NewOrchestrateHandler(mem, model,
    rest.OrchestrateAgentTemplate{Name: "coder", Description: "...", Runner: coder},
)
ph.Register(mux)
// POST   /orchestrate/sessions                         — 创建 session
// POST   /orchestrate/sessions/{id}/generate           — {goal} → PlanDef
// GET    /plan/sessions/{id}/plan               — 获取当前 PlanDef
// PUT    /plan/sessions/{id}/plan               — 用户编辑 PlanDef
// POST   /orchestrate/sessions/{id}/execute            — 触发执行（202）
// GET    /plan/sessions/{id}/events             — SSE 流式进度（pause 期间保持连接）
// POST   /orchestrate/sessions/{id}/cancel             — 取消执行
// POST   /orchestrate/sessions/{id}/approve            — 工具审批
// POST   /orchestrate/sessions/{id}/steps/{stepID}/retry — 手动重试失败步骤
// POST   /orchestrate/sessions/{id}/replan             — {feedback} → 带用户反馈的 Replan
```

### Goal（自主目标模式）

✅ 已实现。`RunGoal()` / `RunGoalStream()` 提供持久化目标的自主 Agent 循环。

**与 `Run()` 的关键区别：**

| | Run(userMessage) | RunGoal(goal) |
|---|---|---|
| Goal 位置 | user message，随对话滚动可能被挤出 | system prompt 头部，跨所有轮次持久 |
| Agent 行为 | 一问一答，无 tool call 即停止 | 自主迭代：执行 → 观察 → 调整 → 继续 |
| 终止条件 | 模型不调工具 | 目标达成 / 确认无法达成 / 达到 MaxTurns |
| 典型场景 | "写个测试" | "通过所有测试"（写 → 跑 → 失败 → 修复 → 重试） |

**实现：**

```go
func (a *Agent) RunGoal(ctx, session, goal) (*RunResult, error)
func (a *Agent) RunGoalStream(ctx, session, goal) <-chan StreamEvent
```

内部 `Clone()` Agent → 将 goal 嵌入 `Instructions` 头部，追加自主迭代规则（"计划→执行→评估→继续，直到目标达成"）。Goal 在 system message 中持久存在，不被 `workingMessages` 的增长挤出。

**与 Plan.Run() 的区别：**

| | RunGoal | Orchestrate.Run |
|---|---|---|
| 执行模式 | 单 Agent 串行自主循环 | 多 Agent DAG 并行执行 |
| 规划方式 | Agent 自己边做边想 | Planner 提前生成完整 DAG |
| 适用 | 开放式探索任务 | 结构化多步骤工作流 |

**Frontend `/goal` 命令：** Chat 页面输入 `/goal <目标>` → 取消正在运行的 plan → 用 Clone + MaxTurns=50 启动 Goal mode → SSE 实时推送进度。

### EventBus

✅ 已实现。`eventbus/` 包提供泛型 session-scoped pub/sub 事件总线。

```go
bus := eventbus.New[MyEvent](500) // 每 session 最多缓存 500 条历史

sub := bus.Subscribe(sessionID)   // 订阅（自动 replay 历史）
bus.Publish(sessionID, evt)       // 发布到所有订阅者
bus.Unsubscribe(sessionID, sub)   // 取消订阅
```

**用途：**
- Frontend 多 tab 同步：用户消息、Agent 响应、会话删除、审批状态跨 tab 同步
- Plan 执行进度推送：多个 EventSource 连接收到相同事件流
- 历史 replay：新订阅者立即收到缓冲区中的历史事件

### Sandbox

✅ 已实现。`sandbox/native/` 包提供基于操作系统原生安全机制的沙箱实现，零外部依赖。

**三层架构**（对齐 Claude Code / Codex 的设计）：

```
Agent（无 sandbox）
  → Tool execution fork 子进程
    → [OS 原生沙箱] → exec 命令
```

| 层 | macOS | Linux | Windows |
|----|-------|-------|---------|
| OS 安全 | `sandbox-exec` + Seatbelt profile | `bwrap`（Bubblewrap namespace 隔离） | Restricted Token（NYI，stub） |
| 文件系统 | 仅 workspace r/w + 系统 bin r/o | bind mount：仅 workspace r/w | — |
| 网络 | `(deny network*)` | `--unshare-all` 含 network namespace | — |
| 降级 | 必须（sandbox-exec 是 macOS 内置） | bwrap 不可用 → `NoNewPrivs` + warning | "not yet implemented" |

**接口**（`sandbox.go`）：

```go
type Sandbox interface {
    Run(ctx context.Context, cmd Command) (Result, error)
}

type Command struct {
    Program string
    Args    []string
    Env     []string
    WorkDir string
    Stdin   string
}

type Result struct {
    Stdout   string
    Stderr   string
    ExitCode int
}
```

**工厂**：

```go
sb, _ := native.New(".")  // 当前目录即工作空间
```

**内置工具**（`tool/` 包）：

| 工具 | 实现 | 模型名 | 功能 |
|------|------|--------|------|
| `Shell` | `Tool` + `StreamExecutor` | `shell` | fork + OS 沙箱执行命令，stdout/stderr 流式输出 |
| `ReadFile` | `Tool` | `read` | 读文件（100KB 上限），路径穿越 + symlink 防护 |
| `WriteFile` | `Tool` | `write` | 写文件（10MB 上限），路径穿越 + symlink 防护 |
| `ListDir` | `Tool` | `ls` | 列目录，dirs first 排序 |

**安全分层**：

```
File tools (ReadFile/WriteFile/ListDir):
  → 宿主机直接 IO（快，不 fork）
  → validatePath: filepath.Abs → EvalSymlinks → Rel 边界检查
  → 工作空间外路径 → 拒绝

Shell tool:
  → fork 子进程 → OS 沙箱 → exec
  → 即使 validatePath 被绕过，OS 沙箱兜底
```

**流式执行**：

Shell 命令的 stdout/stderr 通过 `RunStream` 逐行实时推送。`StreamExecutor` 接口被 Runner 自动检测——实现了就流式，没实现就阻塞。参考 openai-agents 的 PTY 流式方案。

**Demo**：`go run ./examples/sandbox/` —— 6 个测试展示沙箱隔离、流式输出、路径防护。

**对比业界**：

| | openai-agents | Claude Code | openagent-go |
|---|---|---|---|
| 隔离机制 | Docker SDK + macOS `sandbox-exec` | seccomp + user/mount/pid namespaces | macOS Seatbelt / Linux bwrap |
| 依赖 | `docker-py`（重） | 原生 `apply-seccomp` 辅助二进制 | 零编译依赖（bwrap/sandbox-exec 是系统自带） |
| 文件工具 | read/write/ls/rm/mkdir | Read/Write/Glob | ReadFile/WriteFile/ListDir |
| Shell 工具 | `exec_command` + `write_stdin`（PTY） | Bash tool | `shell`（流式，无 PTY） |

---

## 编译时扩展 vs 运行时扩展：总结

```
用户想要...
  "换个 LLM 提供商"         → 编译时：实现 Model 接口
  "换个存储后端"            → 编译时：实现 Memory 接口
  "加个安全审核"            → 编译时：实现 InputGuard/OutputGuard 接口
  "接入自己的监控系统"       → 编译时：实现 Hooks 接口
  "社区贡献一个天气工具"     → 运行时：Tool 插件（WASM）✅
  "在每次调模型前打印 debug" → 运行时：Stage 插件（WASM）✅
  "多 Agent 协作"           → 编译时：Team（handoff + 环路检测）✅
  "做个聊天界面"            → TUI / Frontend 示例 ✅
```

两种方式共存，不互斥。编译时接口是"主干"，运行时插件实现同样的接口（wasmTool → Tool，stageObserver → RunObserver），由 Manager 加载后注入 Agent。

**UI 示例：**
- `cmd/cli/` — 命令行工具，`openagent run "msg"` 单轮 + `openagent goal "task"` 自主模式，流式输出
- `cmd/tui/` — bubbletea v2 终端聊天，流式输出 + Y/N 审批，viewport 滚动
- `examples/backend/` — 完整 REST + SSE API 服务（单 agent / team / plan）
- `examples/frontend/vue-app/` — Vue 3 SPA，Chat / Team / Plan 三种模式

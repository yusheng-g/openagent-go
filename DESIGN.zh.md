# openagent-go Architecture

> [English](DESIGN.md) | [README](README.md) | [README (中文)](README.zh.md)

## 概述

openagent-go 是一个**全系统可插拔**的开源 AI Agent。核心是一条极简的主线 loop，所有能力通过插拔模块叠加。

**设计原则：**

- 遵循业界标准（OpenAI API、ACP v1 协议），不自定义协议
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
         ├─ 静态 system prompts
         ├─ 动态上下文（plan entries、mode 指令）
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

**两层 prompt 模型：**

| 层 | 来源 | 内容 |
|----|------|------|
| Static | `Agent.SystemPrompts` + `Description` | 构造时设定，不变 |
| Dynamic | `Session.DynamicContext` | 每 turn 构建：plan entries + mode 指令 |

Runner 将 `Session.DynamicContext` 传递给 `PromptInput` → `defaultBuildPrompt`。ACP 层从 session 运行时状态构建。

---

## 核心类型

### Agent

```go
type Agent struct {
    Name, Description string
    SystemPrompts   []string   // 静态系统提示词（替代单字段 Instructions）
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
    MaxWorkingTokens    int         // default 0 = 上下文窗口的 70%
    MaxCompressedTokens int         // default 2048
    ReasoningEffort    string       // "none","minimal","low","medium","high","xhigh"
}

agent.Run(ctx, session, input) → (*RunResult, error)
agent.RunStream(ctx, session, input) → <-chan StreamEvent
agent.RunGoal(ctx, session, goal) → (*RunResult, error)
agent.RunGoalStream(ctx, session, goal) → <-chan StreamEvent
agent.Clone() → *Agent
```

Runner 是私有类型，`Agent.Run()` 内部创建。`Clone()` 返回浅拷贝，Tools 底层数组独立；
被 `AgentServer.agentForTurn()` 用于 per-session 隔离。

### StreamEvent

```go
const (
    StreamThought      = "thought"        // 推理内容 (o1, deepseek-r1)
    StreamTextDelta    = "text_delta"     // 逐字符输出
    StreamToolCall     = "tool_call"      // 工具调用开始
    StreamToolProgress = "tool_progress"  // 流式工具输出 chunk
    StreamToolResult   = "tool_result"    // 工具结果（最终）
    StreamRetrying     = "retrying"       // 429 重试中
    StreamDone         = "done"           // 正常完成
    StreamError        = "error"          // 执行失败
    StreamAborted      = "aborted"        // 外部中断（cancel/timeout）
)
```

### Session

```go
type Session struct {
    ID, UserID, ModelID     string
    Temperature, MaxTokens  float64 / int
    UserProfile, ProjectContext string
    DynamicContext              string     // 每轮 plan + mode 上下文
    Turn                        int
    CreatedAt                   time.Time
    Metadata                    map[string]any
}
```

纯数据载体，应用层管理 CRUD。Runner 不创建 Session。

---

## 模块接口

### ① Memory（三层模型）

```
Layer 1: Working    — Recent() 纯查询；Runner 按 MaxWorkingTokens 管理 token 预算
Layer 2: Compressed — Compressed() 自动注入；Compact() 增量压缩
Layer 3: Archive    — Search() + recall 工具；消息永不删除
```

```go
type Memory interface {
    io.Closer
    Count(ctx, sessionID) (int, error)
    Recent(ctx, sessionID, n int, offset int) ([]Message, error)         // 纯查询
    Compact(ctx, sessionID, throughIndex int, messages []Message) error  // Runner 驱动
    Compressed(ctx, sessionID) (*CompressedContext, error)
    Search(ctx, sessionID, query string, limit int) ([]SearchResult, error)
    Append(ctx, sessionID, msg Message) error
    DeleteSession(ctx, sessionID) error
}
```

Runner 驱动压缩：从末尾向前数 token，对 MaxWorkingTokens 进行截断，调整到安全边界，
调用 Compact()。后端仅压缩新溢出的消息，增量/滚动压缩。CJK 内容自动分词（`ftsTokenizeCJK`）。

实现：`memory/file`（JSONL，零依赖）、`memory/sqlite`（SQLite + FTS5 + CJK 分词 + 可选向量搜索）。

### Summarizer（Memory 的依赖）

```go
type Summarizer interface {
    Summarize(ctx context.Context, messages []Message, previous *CompressedContext) (*CompressedContext, error)
}
```

nil = 不压缩。通过 `WithSummarizer()` 配置。实现：`summarizer/llm.go` — LLM 增量压缩。

### Embedder（Memory 的依赖）

```go
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float64, error)
    Dimensions() int
}
```

nil = 降级为 FTS5 搜索。通过 `WithEmbedder()` 配置。

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
    DynamicContext    string   // 每轮 plan + mode 信息
}
```

函数类型 — 单一方法，无需状态。nil = `defaultBuildPrompt()`。

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
    Tripwire bool  // true → 终止 run
}
```

InGuard 在循环前检查一次。OutGuard 检查每次 model 输出 + 每个 tool result。实现：`guard/llm`。

### ④ Model

```go
type Model interface {
    ChatCompletion(ctx, req) (*ChatCompletionResponse, error)
    ChatCompletionStream(ctx, req) (StreamReader, error)  // nil,nil = 不支持
    ContextWindow() int
}
```

实现：`model/openai`（openai-go v3 SDK）。`ChatCompletionRequest.ReasoningEffort` 从
`Agent.ReasoningEffort` 传递到模型的 `reasoning_effort` 参数。

### Tool

```go
type Tool interface {
    Definition() FunctionDefinition
    Execute(ctx context.Context, args json.RawMessage) (string, error)
}
```

内置工具：`shell`、`read`、`write`、`ls`、`grep`。自动注入工具：`load_skill`、`reload_skills`、
`recall`、`subagent`。ACP RPC 工具：`read_client_file`、`write_client_file`、
`terminal_create/output/wait/kill/release`。Plan 工具：`plan_create`、`plan_update`。

### Sandbox

```go
type Sandbox interface {
    Run(ctx context.Context, cmd Command) (Result, error)
    CWD() string    // 工具视角的工作目录（可能与宿主路径不同）
}
```

`CWD()` 返回沙箱内部视角的路径 — bwrap 下为 `/workspace`，否则为宿主路径。Shell 工具从 sandbox 获取工作目录而非自己携带。实现：`sandbox/native`（Linux bwrap、macOS Seatbelt）。

### ⑥ Approver

```go
type Approver interface {
    Approve(ctx, call ToolCall, def FunctionDefinition, session Session) (allowed bool, reason string)
}
```

nil = 全部放行。ACP 模式通过 `session/request_permission` RPC 桥接到客户端。

### ⑦ RunHooks

Start 方法返回不透明 `any` 值，Runner 传递给对应的 End 方法。实现：`hooks/slog`、`hooks/otel`。

### RunObserver

每阶段 enter/leave 事件，含耗时。多个 observer 通过 `MultiObserver()` 组合。

---

## ACP v1 协议

Agent 原生支持 ACP 协议。`AgentServer` 将 `openagent.Agent` 包装为 ACP handler：

```go
agent := openagent.NewAgent("bot", ...)
srv := acp.NewAgentServer(agent, mem, store, models)
server := openacpsdk.NewServer("openagent-acp", "1.0.0", srv)
server.Run(ctx)  // 阻塞在 stdin/stdout
```

**协议层次：**

| 层 | 包 | 角色 |
|----|----|------|
| 类型 | `acp/sdk/` | ACP v1 schema — 958 行，零依赖 |
| 传输 | `acp/sdk/` | JSON-RPC 2.0 over stdio — mux、client session、Agent→Client RPC |
| 集成 | `acp/server.go` | AgentServer — session CRUD、prompt turns、plan mode、MCP、slash 命令 |

**ACP 模式：** `chat`（对话+工具）和 `plan`（通过 `plan_create`/`plan_update` 工具结构化规划）。
Model 注册表支持多模型选择。

---

## Plan 模式

使用 `plan_create` 和 `plan_update` 工具 — LLM 通过 function-calling 参数直接输出结构化 plan 条目：

```
用户目标 → agent.RunStream
  → agent 调用 plan_create(goal, steps[{id, content, priority}])
  → plan 文本进入对话上下文
  → agent 调用 plan_update(updates[{id, status}]) 更新进度
  → plan entries 持久化到 SessionStore._meta["plan"]
  → 每轮：DynamicContext 注入当前 plan 状态到 system prompt
```

**Orchestrate** (`orchestrate/`) 独立 — 多 agent DAG 分解 + 并行执行。非 ACP plan 模式。

---

## Slash 命令

服务端 slash 命令在传递给 agent 之前拦截：

```
/help      — 列出可用命令
/mode      — 切换会话模式 (chat/plan)
/model     — 列出或切换模型
/context   — 显示 token 使用
/cwd       — 显示工作目录
/clear     — 重置会话消息
/rename    — 重命名会话标题
/sessions  — 列出所有会话
```

命令通过 `slash/` Registry 注册，从 `OnPrompt` 分发。未知 `/` 命令传递给 agent 处理。

---

## 目录结构

```
openagent-go/
├── agent.go              Agent + Run/RunStream/RunGoal/Clone + StreamEvent
├── runner.go             private runner + 8-node loop + defaultBuildPrompt
├── model.go              Model、Embedder、Summarizer 接口 + 请求/响应类型
├── message.go            Message + ContentPart（多模态）
├── tool.go               Tool 接口 + FunctionDefinition + StreamExecutor
├── sandbox.go            Sandbox 接口 + Command/Result 类型 + CWD()
├── memory.go             Memory 接口 + CompressedContext
├── tokenizer/            模型感知 token 计数（tiktoken）
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
├── doc.go                包文档
│
├── tool/                 内置工具实现
│   ├── shell.go          Shell（CWD 从 sandbox 获取）
│   ├── file.go           ReadFile / WriteFile / ListDir
│   ├── grep.go           Grep
│   ├── acp_fs.go         ACPReadFile / ACPWriteFile（Agent→Client RPC）
│   └── acp_terminal.go   ACPTerminal（create/output/wait/kill/release）
│
├── plan/                 Plan 模式工具
│   ├── entry.go          Entry + Priority + Status 类型
│   └── tool.go           CreateTool (plan_create) + UpdateTool (plan_update)
│
├── slash/                Slash 命令注册表
│   └── slash.go          Registry、Context、Command、Handler
│
├── summarizer/           LLM 压缩
│   └── llm.go            Compressor（实现 Summarizer 接口）
│
├── sandbox/native/       OS 原生沙箱
├── acp/                  ACP 协议集成
│   ├── sdk/              ACP v1 SDK（类型、JSON-RPC 2.0 mux、客户端）
│   ├── server.go         AgentServer（Agent → ACP handler）
│   └── commands.go       内置 slash 命令注册
├── orchestrate/          多 agent DAG 分解 + 执行
├── model/openai/         OpenAI 模型实现
├── memory/file/          JSONL 文件记忆
├── memory/sqlite/        SQLite + FTS5 + CJK 分词 + 向量搜索
├── guard/llm/            LLM 守卫
├── hooks/slog/           slog 日志钩子
├── hooks/otel/           OpenTelemetry 追踪钩子
├── skill/fs/             文件系统技能加载
├── plugin/wasmhost/      共享 WASM host 层
├── plugin/agent/wasm/    Agent WASM 插件运行时
├── plugin/cli/           CLI 插件宿主
├── plugin/cli/wasm/      CLI WASM 加载器、observer hub
├── plugin/pdk/rust/      插件 SDK（Rust crate）
├── mcp/                  MCP 协议客户端
├── eventbus/             发布订阅事件总线
├── session/              会话元数据类型 + 存储接口
├── rest/                 REST API
├── cmd/tui/              终端聊天
├── cmd/cli/              完整 CLI
├── examples/             示例
│
├── DESIGN.md             架构（英文）
├── DESIGN.zh.md          架构（中文）
└── README.md
```

所有接口在根包。实现在子包。无循环依赖。

---

## 关键设计决策

**1. 为什么 Runner 是私有的？** 用户调用 `Agent.Run()`，从不直接构造 Runner。

**2. 为什么 Runner 触发压缩？** Token 预算由模型的上下文窗口决定，只有 Runner 知道。Runner 计数 token 并决定何时压缩。

**3. 为什么无自动搜索？** Archive 检索由模型通过 `recall` 工具驱动。模型决定何时搜、搜什么。

**4. 为什么 Embedder/Summarizer 不在 Agent 上？** 它们是 Memory 的依赖，不是 Agent 的能力。

**5. 为什么默认流式？** `callModelOnce` 优先流式，fallback 非流式。最低首 token 延迟。

**6. 为什么 PromptBuilder 是函数类型？** 单一方法，无需状态。

**7. 为什么 Handoff 是 Tool？** 模型有完整上下文，比路由器更擅长交接决策。

**8. 为什么 agentForTurn 需要 Clone？** `s.Agent` 被所有 session 共享。Clone 创建隔离副本。

**9. 为什么 ToolFactory 每 turn 创建工具？** 工具需要 session 的 cwd（Docker 容器/bwrap 下与进程 cwd 不同）。

**10. 为什么 Session 上有 DynamicContext？** Plan entries 和 mode 每 turn 变化。Runner 不应知道 ACP 或 plan — Session 是中性传输通道。

---

## 运行时扩展（WASM 插件）

**插件类型：**

| 类型 | 用途 | ABI 导出 |
|------|------|---------|
| `agent:tools` | 向 agent 添加新工具 | `alloc`、`metadata`、`execute` |
| `agent:observers` | 观测 pipeline 阶段、中止运行 | `alloc`、`metadata`、`run` |
| `cli:settings` | 注入凭证、修改 settings JSON | `alloc`、`metadata`、`init` |
| `cli:commands` | 添加自定义 cobra 子命令 | `alloc`、`metadata`、`commands`、`run_<name>` |
| `cli:observers` | 生命周期事件记录 | `alloc`、`metadata`、事件处理函数 |

WASM 运行时：[wazero](https://github.com/tetratelabs/wazero) — 纯 Go，零 CGO。

---

## Team（多 Agent 编排）

```go
team := openagent.NewTeam(
    openagent.WithTeamAgent("researcher", "分析问题", researcher),
    openagent.WithTeamAgent("calculator", "执行计算", calculator),
)
```

Handoff = Tool with `EndTurn: true`。每个 agent 有独立的 Memory、Tools、Guard。

### Orchestrate（LLM 驱动 DAG 执行）

```go
p := orchestrate.NewPlan(
    orchestrate.WithPlanner(orchestrate.NewLLMPlanner(model)),
    orchestrate.WithAgent("coder", "写代码", coderAgent),
    orchestrate.WithAgent("reviewer", "审查代码", reviewerAgent),
)
```

| | Team | Orchestrate |
|---|------|------------|
| 决策 | 运行时，agent 自主发起 | 执行前，LLM 生成 DAG |
| 并行 | 无（串行交接链） | 拓扑批量自动并行 |
| 失败 | agent 自行处理 | LLM 子树 replan |

---

## 沙箱

OS 原生安全：macOS Seatbelt、Linux Bubblewrap。

```go
sb, _ := native.New("./workspace")
cwd := sb.CWD()  // bwrap 下为 "/workspace"，否则为宿主路径
```

---

## 对比

| | openai-agents | Claude Code | openagent-go |
|---|---|---|---|
| 协议 | — | — | ACP v1 (JSON-RPC 2.0) |
| 沙箱 | Docker SDK + macOS sandbox-exec | seccomp + namespaces | macOS Seatbelt / Linux bwrap |
| 文件工具 | read/write/ls | Read/Write/Glob | ReadFile/WriteFile/ListDir/Grep |
| 流式 | PTY-based | Bash tool | Shell tool (line streaming) |
| 多 agent | Handoff chain | — | Team + Orchestrate |
| Plan 模式 | — | 工具驱动 | plan_create/plan_update 工具 |
| 可观测性 | — | — | RunObserver + StageEvent |
| 插件 | — | — | WASM (wazero, 零 CGO) |
| Slash 命令 | — | — | 注册表 + 内置 + 可扩展 |
| Memory 压缩 | — | — | LLM 增量 summarizer |

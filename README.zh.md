# openagent-go

> [English](README.md) | [Architecture](DESIGN.md) | [架构 (中文)](DESIGN.zh.md)

一个完全可插拔的多智能体 AI Agent 框架，Go 语言实现。

## 特性

- **全插件架构** — 每个组件都是接口：Model、Memory、Tools、Guards、Approver、Hooks、Observer
- **ACP v1 协议** — 完整的 Agent Client Protocol 实现，基于 stdio（JSON-RPC 2.0）。可用任何 ACP 客户端（VSCode 插件、Zed 等）
- **Plan 模式** — `plan_create`/`plan_update` 工具让 agent 将复杂任务分解为结构化步骤，实时追踪进度
- **多智能体团队** — agent 之间通过 `transfer_to_*` 工具交接任务；每个 agent 有独立的记忆、工具和守卫
- **多智能体编排** — LLM 驱动的 DAG 分解、并行执行和自动重规划（`orchestrate/`）
- **SSE 流式输出** — 实时逐 token 渲染，支持 reasoning 展示、工具调用卡片
- **三层记忆系统** — Working（token 驱动）、Compressed（LLM 增量摘要，`summarizer/`）、Archive（FTS5/向量检索，永不删除）
- **沙箱环境** — 原生 OS 级别隔离（Linux bwrap、macOS Seatbelt），安全执行 shell、文件、网络操作
- **WASM 插件** — Agent 级：`agent:tools` 和 `agent:observers` 接入工具/观测器管线。CLI 级：`cli:settings`、`cli:commands`、`cli:observers`，用于设置注入、命令扩展和生命周期监控
- **静态上下文配置** — `AGENTS.md`（工作规则）和 `SOUL.md`（性格与底线），支持用户级和项目级覆盖
- **Slash 命令** — 内置 `/help`、`/mode`、`/model`、`/context`、`/cwd`、`/clear`、`/rename`、`/sessions`，通过 `slash/` 注册表扩展
- **完整 CLI** — `openagent-cli`，cobra 命令、配置驱动模型、keyring 密钥管理、WASM 插件运行时
- **IM 频道** — 接入飞书/Lark WebSocket，基于卡片的流式输出（Markdown 渲染、工具调用卡片），一键扫码创建应用
- **RunHooks 状态传递** — Start/End 回调共享不透明状态，OTEL 正确嵌套 span，slog 精确计时
- **动态上下文** — 会话级 plan 状态和 mode 指令每轮自动注入 prompt

## 快速开始

```bash
# 编译 CLI
go build -o openagent-cli ./cmd/cli/

# ACP 模式（stdio — 配合 VSCode/Zed ACP 插件使用）
./openagent-cli serve --acp

# REST 模式（HTTP + SSE）
./openagent-cli serve --port 8080

# 前端
cd examples/frontend/vue-app && npm install && npm run dev
```

### 配置

创建 `~/.openagent/settings.json`：

```json
{
  "provider": {
    "openai": {
      "api_key": "sk-...",
      "models": ["gpt-4o"]
    }
  },
  "profiles": ".openagent/profile"
}
```

将 `AGENTS.md` 和 `SOUL.md` 放在 `~/.openagent/profile/` 或 `$(pwd)/.openagent/profile/` 来自定义 agent 的行为。

打开 `http://localhost:5173` 或连接 ACP 客户端 — 服务端同时支持两种协议。

### 飞书集成

将 agent 接入飞书（Lark），支持群聊、私聊、Markdown 卡片渲染、流式输出。

**首次使用（无需凭据）：**

```bash
./openagent-cli serve --channel feishu
```

终端会出现二维码。打开飞书 App 扫码，确认创建应用即可。SDK 会自动创建机器人应用并配置好权限（`im:message`、`im:message:send_as_bot`、`im.message.receive_v1` 事件），凭据保存在本地。

**如果已有应用，在 `settings.json` 中配置：**

```json
{
  "provider": {
    "openai": { "api_key": "sk-...", "models": ["gpt-4o"] }
  },
  "channels": {
    "feishu": {
      "app_id": "cli_xxxxxxxxxxxxxxxx",
      "app_secret": "xxxxxxxxxxxxxxxxxxxxxxxxxx"
    }
  }
}
```

然后带 flag 启动：

```bash
./openagent-cli serve --channel feishu
```

`--channel` flag 是必须的 — 仅配置 settings.json 不会自动启动 bot。如果凭据已在 settings.json 中，启动时会跳过扫码步骤。

**凭据解析优先级：**

| 优先级 | 来源 | 场景 |
|--------|------|------|
| 1 | `settings.json` → `channels.feishu` | 已有应用凭据 |
| 2 | `~/.openagent/data/feishu_app.json` | 上次扫码自动保存 |
| 3 | 扫码注册 | 首次使用，无任何凭据 |

**组合其他模式：**

```bash
# REST API + 飞书机器人
./openagent-cli serve --channel feishu

# ACP 模式（stdio，配合 VSCode/Zed）+ 飞书机器人
./openagent-cli serve --acp --channel feishu
```

**配置 MCP 工具（可选）：**

```json
{
  "mcp_servers": {
    "browser": {
      "command": "npx",
      "args": ["-y", "@anthropic/mcp-server-browser-tools"]
    }
  }
}
```

MCP 工具在启动时即可用，每次工具调用以卡片形式展示在飞书中。

**日志：**

```json
{
  "log": {
    "file": "/var/log/openagent/openagent.log",
    "max_size": 10,
    "max_backups": 5,
    "level": "debug"
  }
}
```

所有字段都是可选的。默认值：`~/.openagent/data/openagent.log`，10 MB 轮转，保留 5 个备份，info 级别。
单位是 MB。日志同时输出到 stderr 和文件。设为 `"debug"` 可查看每次 API 请求详情。

## 架构

```
┌──────────────────────────────────────────────┐
│  Agent                                       │
│  ├── Model        (LLM 提供商)               │
│  ├── Memory       (对话持久化)               │
│  ├── Tools        (shell, 读写文件, ...)     │
│  ├── InGuard      (输入校验)                 │
│  ├── OutGuard     (输出校验)                 │
│  ├── Approver     (工具调用的用户确认)       │
│  ├── Hooks        (生命周期回调)             │
│  └── Observer     (pipeline 监控)            │
└──────────────────────────────────────────────┘
```

## 插件

插件是 **WASM 模块**（.wasm 文件）。任何能编译到 WASM 的语言都可以 — Rust、Go、TypeScript、Zig 等。宿主运行时（wazero）在沙箱环境中加载和执行它们。

每个插件通过元数据声明自己的类型。目前我们提供了 Rust SDK（`plugin/pdk/rust/`）封装了 FFI 契约，但 ABI 足够简单，任何语言都能直接实现。

| 插件类型 | 功能 |
|----------|------|
| `agent:tools` | 为 agent 添加自定义工具 — agent 可以像调用 shell/read/write 一样调用它们 |
| `agent:observers` | 挂载到 agent 的运行管线（每个阶段的 enter/leave） |
| `cli:settings` | 启动时转换 settings.json（合并环境变量、添加 provider 等） |
| `cli:commands` | 注册额外的 cobra 子命令到 CLI |
| `cli:observers` | 监控 CLI 命令生命周期（启动/关闭/命令 enter/exit） |

### 工作原理

每种插件类型暴露一或两个导出函数：

| 类型 | 导出函数 | 签名 |
|------|---------|------|
| `agent:tools` | `openagent_agent_tools()` → JSON | 返回 `[{name, description, parameters}]` |
| | `openagent_execute(name, args)` → string | agent 调用工具时执行 |
| `agent:observers` | `openagent_on_stage(event_json)` | 每个阶段 enter/leave 时调用 |
| `cli:settings` | `openagent_cli_init(settings_json)` → JSON | 返回合并后的 settings |
| `cli:commands` | `openagent_cli_commands()` → JSON | 返回 `[{use, short, long}]` |
| | `openagent_cli_run(name, args_json)` → string | 命令执行时调用 |
| `cli:observers` | `openagent_cli_on_startup()` / `...on_shutdown()` 等 | 生命周期回调 |

宿主运行时（wazero + `plugin/wasmhost/`）提供了一组可导入的 host 函数（`log_info`、`keyring_get`、`http_request`、`utc_now` 等），插件可以调用。

### 启用插件

将 `.wasm` 文件放入目录并在 `settings.json` 中配置：

```json
{
  "plugins": ["~/.openagent/plugins"]
}
```

CLI 启动时会扫描所有配置目录下的 `.wasm` 文件，读取元数据、实例化，并接入 agent 或 CLI 命令树。

### 编译插件（Rust 示例）

```bash
# 前置条件：Rust + wasm32-unknown-unknown target
rustup target add wasm32-unknown-unknown

# 编译
cd examples/plugin/tool
cargo build --release --target wasm32-unknown-unknown

# 复制到插件目录
cp target/wasm32-unknown-unknown/release/example_agent_tool.wasm ~/.openagent/plugins/echo.wasm
```

或使用 Makefile 一步完成：

```bash
make -C examples/plugin
```

### 编写工具插件 (agent:tools)

```rust
use openagent_sdk::tool::{register_tools, ToolDef};

#[no_mangle]
pub extern "C" fn openagent_agent_tools() -> *const u8 {
    register_tools(&[ToolDef {
        name: "echo",
        description: "回显输入消息。",
        parameters: r#"{"type":"object","properties":{"message":{"type":"string"}},"required":["message"]}"#,
    }])
}

#[no_mangle]
pub extern "C" fn openagent_execute(name: &str, args: &str) -> String {
    format!("echo: {}", extract_field(args, "message"))
}
```

### 编写观测器插件 (agent:observers)

```rust
use openagent_sdk::observer::StageEvent;

#[no_mangle]
pub extern "C" fn openagent_on_stage(event_json: &str) {
    let e: StageEvent = serde_json::from_str(event_json).unwrap();
    if e.phase == "enter" {
        // log_info 是可导入的 host 函数
    }
}
```

### Host API（任何语言都可调用）

| 函数 | 用途 |
|------|------|
| `log_info(msg)` / `log_warn(msg)` / `log_error(msg)` | 通过宿主记录日志 |
| `utc_now() -> i64` | 当前纳秒时间戳 |
| `keyring_get(service, key) -> string` | 读取系统密钥环 |
| `keyring_set(service, key, value)` | 写入系统密钥环 |
| `keyring_delete(service, key)` | 删除系统密钥环 |
| `http_request(method, url, headers_json, body) -> {status, body}` | 发送 HTTP 请求 |

完整示例见 `examples/plugin/`。Rust SDK：`plugin/pdk/rust/`。

## 示例

| 示例 | 说明 |
|------|------|
| `examples/basic/` | 最小化 agent + model |
| `examples/stream/` | 流式文本输出 |
| `examples/memory/` | 记忆 + 摘要压缩 |
| `examples/team/` | 多 agent 交接 |
| `examples/guard/` | 输入/输出守卫 |
| `examples/hooks/` | 生命周期钩子 |
| `examples/observer/` | Pipeline 观测器 |
| `examples/delegate/` | Agent 作为工具委托 |
| `examples/sandbox/` | 原生沙箱工具 |
| `examples/plugin/` | WASM 工具 + 观测器插件 |
| `examples/skill/` | 按需加载技能 |
| `examples/acp/` | ACP agent 协议（server + client） |
| `examples/iac/` | 多 agent IaC 流水线 |
| `examples/backend/` | 完整 REST + SSE API 服务 |
| `examples/frontend/vue-app/` | Vue 3 SPA 参考前端 |
| `cmd/cli/` | 完整 CLI，含 WASM 插件运行时 |

## 包

| 包 | 用途 |
|----|------|
| `openagent` | 核心类型、Agent、Team、Runner、Memory、Sandbox |
| `acp/sdk/` | ACP v1 协议 SDK — 类型定义、JSON-RPC 2.0 mux、客户端 |
| `acp/` | AgentServer — 将 Agent 包装为 ACP handler |
| `rest/` | REST + SSE 处理器（单 agent / team / orchestrate） |
| `orchestrate/` | 多 agent DAG 分解 + 流式执行 |
| `plan/` | `plan_create`/`plan_update` 工具（ACP plan 模式） |
| `slash/` | Slash 命令注册表和分发 |
| `summarizer/` | 基于 LLM 的增量对话压缩 |
| `memory/sqlite/` | SQLite + FTS5 + 向量记忆后端 |
| `memory/file/` | JSONL 文件记忆后端 |
| `model/openai/` | OpenAI ChatCompletion + 流式 |
| `tokenizer/` | tiktoken 模型感知 token 计数 |
| `sandbox/native/` | OS 级进程隔离（bwrap/Seatbelt） |
| `session/` | 会话元数据类型和存储接口 |
| `session/sqlite/` | SQLite 会话存储 |
| `session/file/` | 文件会话存储 |
| `eventbus/` | 会话级发布订阅（SSE） |
| `plugin/wasmhost/` | 共享 WASM host 模块（keyring、HTTP、日志、utc_now） |
| `plugin/agent/wasm/` | Agent 级 WASM 插件宿主 |
| `plugin/cli/` | CLI 插件管理和类型 |
| `plugin/cli/wasm/` | CLI 级 WASM 运行时、加载器、observer hub |
| `plugin/pdk/rust/` | Rust SDK crate，用于构建 WASM 插件 |
| `skill/fs/` | 文件系统技能加载器 |
| `mcp/` | Model Context Protocol 客户端 |
| `guard/llm/` | 基于 LLM 的输入/输出守卫 |
| `hooks/otel/` | OpenTelemetry 钩子 |
| `hooks/slog/` | 结构化日志钩子 |
| `tool/` | 内置工具 (shell, read, write, ls, grep, ACP fs, ACP terminal) |
| `channel/` | IM 平台适配器 — 飞书 WebSocket、卡片渲染 |
| `cmd/cli/` | CLI 运行时、WASM 宿主、Rust SDK 示例 |

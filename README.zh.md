# openagent-go

> [English](README.md) | [Architecture](DESIGN.md) | [架构 (中文)](DESIGN.zh.md)

一个完全可插拔的多智能体 AI Agent 框架，Go 语言实现。

## 特性

- **全插件架构** — 每个组件都是接口：Model、Memory、Tools、Guards、Approver、Hooks、Observer
- **多智能体团队** — agent 之间通过 `transfer_to_*` 交接任务，全程携带上下文
- **Plan 模式** — LLM 将目标分解为 DAG，流式执行每一步
- **SSE 流式输出** — 实时逐 token 渲染，支持 reasoning 展示、工具调用卡片
- **三层记忆系统** — Working（token 驱动）、Compressed（增量摘要）、Archive（可检索，永不删除）
- **沙箱环境** — 原生 OS 级别隔离，安全执行 shell、文件、网络操作
- **WASM 插件系统** — 设置注入、命令扩展、生命周期观察，通过 `.wasm` 模块动态加载
- **内置子代理工具** — 模型运行时动态 spawn 并行子 agent
- **完整 CLI** — `openagent-cli` 配置驱动模型、keyring 密钥管理、cobra 命令扩展
- **RunHooks 状态传递** — Start/End 回调共享不透明状态，OTEL 正确嵌套 span，slog 精确计时

## 快速开始

```bash
# CLI
go build -o openagent-cli ./cmd/cli/
./openagent-cli serve --port 8080

# 后端
OPENAGENT_API_KEY=sk-... OPENAGENT_MODEL=gpt-4o go run ./examples/backend/

# 前端
cd examples/frontend/vue-app && npm install && npm run dev
```

打开 `http://localhost:5173`，三种模式：
- **Chat** — 单 agent 对话
- **Team** — 多 agent 流水线协作
- **Plan** — 目标 → DAG → 流式执行

## 架构

```
┌──────────────────────────────────────────┐
│  Agent                                   │
│  ├── Model      (LLM 提供商)             │
│  ├── Memory     (对话持久化)             │
│  ├── Tools      (shell, 读写文件, ...)   │
│  ├── InGuard    (输入校验)               │
│  ├── OutGuard   (输出校验)               │
│  ├── Approver   (工具调用的用户确认)     │
│  ├── Hooks      (生命周期回调)           │
│  └── Observer   (pipeline 监控)          │
└──────────────────────────────────────────┘
```

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
| `examples/plugin/` | WASM 工具 + 阶段插件 |
| `examples/skill/` | 按需加载技能 |
| `examples/acp/` | ACP agent 协议 |
| `examples/backend/` | 完整 REST + SSE API 服务 |
| `examples/frontend/vue-app/` | Vue 3 SPA 参考前端 |
| `cmd/cli/` | 完整 CLI，含 WASM 插件运行时 |

## 包

| 包 | 用途 |
|----|------|
| `openagent` | 核心类型、Agent、Team、Runner、Memory |
| `rest/` | REST + SSE 处理器（单 agent / team / plan） |
| `plan/` | 目标分解、DAG 执行 |
| `memory/sqlite/` | SQLite + FTS5 + 向量记忆后端 |
| `memory/file/` | JSONL 文件记忆后端 |
| `model/openai/` | OpenAI ChatCompletion + 摘要器 |
| `tokenizer/` | tiktoken 模型感知 token 计数 |
| `sandbox/native/` | OS 级进程隔离 |
| `eventbus/` | 会话级发布订阅（SSE） |
| `plugin/agent/wasm/` | WASM 插件宿主 |
| `skill/fs/` | 文件系统技能加载器 |
| `mcp/` | Model Context Protocol |
| `acp/` | Agent Communication Protocol |
| `guard/llm/` | 基于 LLM 的输入/输出守卫 |
| `hooks/otel/` | OpenTelemetry 钩子 |
| `hooks/slog/` | 结构化日志钩子 |
| `tool/` | 内置工具 (shell, read, write, ls, grep) |
| `cmd/cli/` | CLI 运行时、WASM 宿主、Rust SDK |

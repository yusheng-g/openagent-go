# openagent-go

A fully pluggable, multi-agent AI agent framework in Go.

## Features

- **Pluggable architecture** — every component is an interface: Model, Memory, Tools, Guards, Approver, Hooks, Observer
- **Multi-agent team** — agents hand off tasks via `transfer_to_*` with full context passing
- **Plan mode** — LLM decomposes goals into DAGs, executes steps with streaming progress
- **Streaming SSE** — real-time token-by-token output, reasoning display, tool call cards
- **Memory system** — three-layer: Working (token-driven), Compressed (incremental summary), Archive (searchable)
- **Sandbox** — native OS-level confinement for shell, file, and network operations
- **ACP support** — Agent Communication Protocol for external agent processes
- **Observability** — pipeline stage monitor with timing and tool details

## Quick Start

```bash
# Backend
OPENAGENT_API_KEY=sk-... OPENAGENT_MODEL=gpt-4o go run ./examples/backend/

# Frontend
cd examples/frontend/vue-app && npm install && npm run dev
```

Open `http://localhost:5173` — three modes available:
- **Chat** — single-agent conversation
- **Team** — multi-agent pipeline with handoff
- **Plan** — goal → DAG → streaming execution

## Architecture

```
┌──────────────────────────────────────────┐
│  Agent                                   │
│  ├── Model      (LLM provider)           │
│  ├── Memory     (conversation storage)   │
│  ├── Tools      (shell, file, grep, ...) │
│  ├── InGuard    (input validation)       │
│  ├── OutGuard   (output validation)      │
│  ├── Approver   (tool call confirmation) │
│  ├── Hooks      (lifecycle callbacks)    │
│  └── Observer   (pipeline monitoring)    │
└──────────────────────────────────────────┘
```

## Examples

| Example | Description |
|---------|-------------|
| `examples/basic/` | Minimal agent + model |
| `examples/stream/` | Streaming text deltas |
| `examples/memory/` | Memory + summarizer |
| `examples/team/` | Multi-agent handoff |
| `examples/guard/` | Input/output guards |
| `examples/hooks/` | Lifecycle hooks |
| `examples/observer/` | Pipeline observer |
| `examples/delegate/` | Agent as tool delegation |
| `examples/sandbox/` | Native sandbox tools |
| `examples/plugin/` | WASM plugin system |
| `examples/skill/` | On-demand skill loading |
| `examples/acp/` | ACP agent protocol |
| `examples/backend/` | Full REST + SSE API server |
| `examples/frontend/vue-app/` | Vue 3 SPA reference UI |

## Packages

| Package | Purpose |
|---------|---------|
| `openagent` | Core types, Agent, Team, Runner, Memory |
| `rest/` | REST + SSE handlers for single/team/plan |
| `plan/` | Goal decomposition, DAG execution |
| `memory/sqlite/` | SQLite + FTS5 + vector memory backend |
| `memory/file/` | JSONL file memory backend |
| `model/openai/` | OpenAI ChatCompletion + Summarizer |
| `tokenizer/` | tiktoken model-aware token counting |
| `sandbox/native/` | OS-level process confinement |
| `eventbus/` | Session-scoped pub/sub for SSE |
| `plugin/wasm/` | WASM plugin host |
| `skill/fs/` | Filesystem skill loader |
| `mcp/` | Model Context Protocol |
| `acp/` | Agent Communication Protocol |
| `guard/llm/` | LLM-based input/output guard |
| `hooks/otel/` | OpenTelemetry hooks |
| `hooks/slog/` | Structured logging hooks |
| `tool/` | Built-in tools (shell, read, write, ls, grep) |

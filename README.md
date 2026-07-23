# openagent-go

> [中文](README.zh.md) | [Architecture](DESIGN.md) | [架构 (中文)](DESIGN.zh.md)

A fully pluggable, multi-agent AI agent framework in Go.

## Features

- **Pluggable architecture** — every component is an interface: Model, Memory, Tools, Guards, Approver, Hooks, Observer
- **ACP v1 protocol** — full Agent Client Protocol implementation over stdio (JSON-RPC 2.0). Use any ACP-compatible client (VSCode extension, Zed, etc.)
- **Plan mode** — `plan_create`/`plan_update` tools let the agent decompose complex tasks into structured steps with live progress tracking
- **Multi-agent team** — agents hand off tasks via `transfer_to_*` tools; each agent has independent memory, tools, and guard
- **Multi-agent orchestration** — LLM-driven DAG decomposition, parallel execution, and auto-replan via `orchestrate/`
- **Streaming SSE** — real-time token-by-token output, reasoning display, tool call cards
- **Three-layer memory** — Working (token-driven), Compressed (LLM incremental summary via `summarizer/`), Archive (FTS5/vector searchable, never deleted)
- **Sandbox** — native OS-level confinement (Linux bwrap, macOS Seatbelt) for shell, file, and network operations
- **WASM plugins** — agent-level: `agent:tools` and `agent:observers` plug into the tool/observer pipeline. CLI-level: `cli:settings`, `cli:commands`, `cli:observers` for settings injection, command extension, and lifecycle monitoring
- **Static context profiles** — `AGENTS.md` (working rules) and `SOUL.md` (persona & limits) with user-level and project-level resolution
- **Slash commands** — built-in `/help`, `/mode`, `/model`, `/context`, `/cwd`, `/clear`, `/rename`, `/sessions`, extensible via `slash/` registry
- **Full CLI** — `openagent-cli` with cobra commands, config-driven models, keyring secrets, WASM plugin runtime
- **IM channels** — connect to Feishu/Lark via WebSocket; card-based streaming output with markdown, tool call cards, and one-click QR code setup
- **RunHooks with state** — start/end callbacks share opaque state; OTEL spans nest, slog logs duration
- **Dynamic context** — session-level plan status and mode injected into every prompt turn

## Quick Start

```bash
# Build CLI
go build -o openagent-cli ./cmd/cli/

# ACP mode (stdio — for VSCode/Zed ACP plugins)
./openagent-cli serve --acp

# REST mode (HTTP + SSE)
./openagent-cli serve --port 8080

# Frontend
cd examples/frontend/vue-app && npm install && npm run dev
```

### Configuration

Create `~/.openagent/settings.json`:

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

Put `AGENTS.md` and `SOUL.md` in `~/.openagent/profile/` or `$(pwd)/.openagent/profile/` to customise the agent's behaviour.

Open `http://localhost:5173` or connect an ACP client — the server supports both protocols.

### Feishu / Lark Integration

Connect your agent to Feishu (Lark) so users can chat with it in IM — group chats, private chats, cards with markdown rendering, and real-time streaming output.

**First-time setup (no credentials needed):**

```bash
./openagent-cli serve --channel feishu
```

A QR code will appear in your terminal. Open Feishu on your phone, scan it, and confirm the app creation. The SDK automatically provisions a bot app with the correct permissions (`im:message`, `im:message:send_as_bot`, `im.message.receive_v1` event) and saves the credentials locally.

**If you already have an app, configure it in `settings.json`:**

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

Then run with the flag to enable the channel:

```bash
./openagent-cli serve --channel feishu
```

The `--channel` flag is always required to start the bot — settings.json alone won't auto-start it. If your credentials are in settings.json, the setup step is skipped automatically.

**Where credentials are stored:**

| Priority | Source | When to use |
|----------|--------|-------------|
| 1 | `settings.json` → `channels.feishu` | You have the app ID and secret |
| 2 | `~/.openagent/data/feishu_app.json` | Auto-saved after QR registration |
| 3 | QR code registration | First time, no credentials at all |

**Combine with other modes:**

```bash
# REST API + Feishu bot
./openagent-cli serve --channel feishu

# ACP mode (stdio for VSCode/Zed) + Feishu bot
./openagent-cli serve --acp --channel feishu
```

**Adding MCP tools (optional):**

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

MCP tools are available to the Feishu bot at startup. Each tool call renders as a card in the chat.

**Logging:**

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

All fields are optional. Defaults: `~/.openagent/logs/openagent.log`, 10 MB rotation, 5 backups, info level.
Each `max_size` unit is megabytes. Logs go to both stderr *and* the file. Set `level` to `"debug"` to see every API request.

## Architecture

```
┌──────────────────────────────────────────────┐
│  Agent                                       │
│  ├── Model        (LLM provider)             │
│  ├── Memory       (conversation storage)     │
│  ├── Tools        (shell, file, grep, ...)   │
│  ├── InGuard      (input validation)         │
│  ├── OutGuard     (output validation)        │
│  ├── Approver     (tool call confirmation)   │
│  ├── Hooks        (lifecycle callbacks)      │
│  └── Observer     (pipeline monitoring)      │
└──────────────────────────────────────────────┘
```

## Plugins

Plugins are **WASM modules** (.wasm files). Any language that compiles to WASM works — Rust, Go, TypeScript, Zig, etc. The host runtime (wazero) loads and executes them in a sandboxed environment.

A plugin declares its type via metadata. Currently we provide a Rust SDK (`plugin/pdk/rust/`) that wraps the FFI contract, but the ABI is simple enough to implement from any language.

| Plugin type | What it does |
|-------------|--------------|
| `agent:tools` | Adds custom tools to the agent — the agent can call them like shell/read/write |
| `agent:observers` | Hooks into the agent's run pipeline (enter/leave for each stage) |
| `cli:settings` | Transforms `settings.json` at startup (merge env vars, add providers, etc.) |
| `cli:commands` | Registers extra cobra subcommands into the CLI |
| `cli:observers` | Monitors CLI command lifecycle (startup/shutdown/command enter/exit) |

### How it works

Each plugin type exposes one or two exported functions:

| Type | Exports | Signature |
|------|---------|-----------|
| `agent:tools` | `openagent_agent_tools()` → JSON | Returns `[{name, description, parameters}]` |
| | `openagent_execute(name, args)` → string | Called when the agent invokes the tool |
| `agent:observers` | `openagent_on_stage(event_json)` | Called on each stage enter/leave |
| `cli:settings` | `openagent_cli_init(settings_json)` → JSON | Returns merged settings |
| `cli:commands` | `openagent_cli_commands()` → JSON | Returns `[{use, short, long}]` |
| | `openagent_cli_run(name, args_json)` → string | Called when the command runs |
| `cli:observers` | `openagent_cli_on_startup()` / `...on_shutdown()` / etc. | Lifecycle callbacks |

The host runtime (wazero + `plugin/wasmhost/`) provides a set of importable host functions (`log_info`, `keyring_get`, `http_request`, `utc_now`, etc.) that plugins can call.

### Enabling plugins

Place `.wasm` files in a directory and configure it in `settings.json`:

```json
{
  "plugins": ["~/.openagent/plugins"]
}
```

At startup the CLI scans all configured directories for `.wasm` files, reads their metadata, instantiates them, and wires them into the agent or CLI command tree.

### Compiling a plugin (Rust example)

```bash
# Prerequisites: Rust + wasm32-unknown-unknown target
rustup target add wasm32-unknown-unknown

# Build
cd examples/plugin/tool
cargo build --release --target wasm32-unknown-unknown

# Copy to plugins directory
cp target/wasm32-unknown-unknown/release/example_agent_tool.wasm ~/.openagent/plugins/echo.wasm
```

Or use the Makefile:

```bash
make -C examples/plugin
```

### Writing a tool plugin (agent:tools)

```rust
use openagent_sdk::tool::{register_tools, ToolDef};

#[no_mangle]
pub extern "C" fn openagent_agent_tools() -> *const u8 {
    register_tools(&[ToolDef {
        name: "echo",
        description: "Echo back the input message.",
        parameters: r#"{"type":"object","properties":{"message":{"type":"string"}},"required":["message"]}"#,
    }])
}

#[no_mangle]
pub extern "C" fn openagent_execute(name: &str, args: &str) -> String {
    format!("echo: {}", extract_field(args, "message"))
}
```

### Writing an observer plugin (agent:observers)

```rust
use openagent_sdk::observer::StageEvent;

#[no_mangle]
pub extern "C" fn openagent_on_stage(event_json: &str) {
    let e: StageEvent = serde_json::from_str(event_json).unwrap();
    if e.phase == "enter" {
        // log_info is an imported host function
    }
}
```

### Host API (importable from any language)

| Function | Purpose |
|----------|---------|
| `log_info(msg)` / `log_warn(msg)` / `log_error(msg)` | Logging through the host |
| `utc_now() -> i64` | Current time in nanoseconds |
| `keyring_get(service, key) -> string` | Read from system keyring |
| `keyring_set(service, key, value)` | Write to system keyring |
| `keyring_delete(service, key)` | Delete from system keyring |
| `http_request(method, url, headers_json, body) -> {status, body}` | Outbound HTTP |

Full example: `examples/plugin/`. Rust SDK: `plugin/pdk/rust/`.

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
| `examples/plugin/` | WASM tool + observer plugins |
| `examples/skill/` | On-demand skill loading |
| `examples/acp/` | ACP agent protocol (server + client) |
| `examples/iac/` | Multi-agent IaC pipeline |
| `examples/backend/` | Full REST + SSE API server |
| `examples/frontend/vue-app/` | Vue 3 SPA reference UI |
| `cmd/cli/` | Full-featured CLI with WASM plugin runtime |

## Packages

| Package | Purpose |
|---------|---------|
| `openagent` | Core types, Agent, Team, Runner, Memory, Sandbox |
| `acp/sdk/` | ACP v1 protocol SDK — types, JSON-RPC 2.0 mux, client |
| `acp/` | AgentServer — wraps an Agent as an ACP handler |
| `rest/` | REST + SSE handlers (single, team, orchestrate) |
| `orchestrate/` | Multi-agent DAG decomposition + streaming execution |
| `plan/` | `plan_create`/`plan_update` tools (ACP plan mode) |
| `slash/` | Slash command registry and dispatch |
| `summarizer/` | LLM-based incremental conversation compression |
| `memory/sqlite/` | SQLite + FTS5 + vector memory backend |
| `memory/file/` | JSONL file memory backend |
| `model/openai/` | OpenAI ChatCompletion + streaming |
| `tokenizer/` | tiktoken model-aware token counting |
| `sandbox/native/` | OS-level process confinement (bwrap/Seatbelt) |
| `session/` | Session metadata types and store interface |
| `session/sqlite/` | SQLite session store |
| `session/file/` | File-backed session store |
| `eventbus/` | Session-scoped pub/sub for SSE |
| `plugin/wasmhost/` | Shared WASM host module (keyring, HTTP, logging, utc_now) |
| `plugin/agent/wasm/` | Agent-scoped WASM plugin host |
| `plugin/cli/` | CLI plugin manager and types |
| `plugin/cli/wasm/` | CLI-scoped WASM runtime, loader, observer hub |
| `plugin/pdk/rust/` | Rust SDK crate for building WASM plugins |
| `skill/fs/` | Filesystem skill loader |
| `mcp/` | Model Context Protocol client |
| `guard/llm/` | LLM-based input/output guard |
| `hooks/otel/` | OpenTelemetry hooks |
| `hooks/slog/` | Structured logging hooks |
| `tool/` | Built-in tools (shell, read, write, ls, grep, ACP fs, ACP terminal) |
| `channel/` | IM platform adapters — Feishu WebSocket, card rendering |
| `cmd/cli/` | CLI runtime, WASM host, Rust SDK examples |

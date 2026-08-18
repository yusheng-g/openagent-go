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
- **Structured tool results** — `ToolResult` carries content/JSON/error/truncation; oversized output spills to disk automatically (line-wrapped, read/grep-friendly) instead of flooding the model context
- **Approval policy engine** — layered chain (rules → safety → approval memory → human) with argument editing and persistent "always allow" decisions
- **Self-evolution** — LLM extractor turns finished runs into durable knowledge, recalled into later sessions
- **Three-layer memory** — Working (token-driven), Compressed (LLM incremental summary via `summarizer/`), Archive (vector/keyword searchable, never deleted); all three are provider-pluggable, including a remote OpenViking context database
- **Sandbox** — native OS-level confinement (Linux bwrap, macOS Seatbelt) for shell, file, and network operations
- **WASM plugins** — agent-level: `agent:tools` and `agent:observers` plug into the tool/observer pipeline. CLI-level: `cli:settings`, `cli:commands`, `cli:observers`, `cli:http` for settings injection, command extension, lifecycle monitoring, and custom HTTP routes. Any plugin can declare cron-scheduled jobs.
- **Static context profiles** — `AGENTS.md` (working rules) and `SOUL.md` (persona & limits) with user-level and project-level resolution
- **Slash commands** — built-in `/help`, `/mode`, `/model`, `/compact`, `/context`, `/cwd`, `/clear`, `/rename`, `/sessions`, extensible via `slash/` registry
- **Full CLI** — `hwcloud` with cobra commands, config-driven models, keyring secrets, WASM plugin runtime
- **IM channels** — Feishu/Lark (WebSocket, card-based streaming output with markdown and tool call cards, one-click QR setup, inline approval buttons, /clear and /mode commands), personal WeChat (Tencent ilinkai channel, QR login with pairing code, /clear command), and WeCom 企业微信 (official long connection, native streaming replies, QR robot auto-creation, /clear command)
- **RunHooks with state** — start/end callbacks share opaque state; OTEL spans nest, slog logs duration
- **Dynamic context** — session-level plan status and mode injected into every prompt turn

## Quick Start

```bash
# Build CLI
go build -o hwcloud ./cmd/cli/

# Show version
./hwcloud -v

# ACP mode (stdio — for VSCode/Zed ACP plugins)
./hwcloud serve --acp

# REST mode (HTTP + SSE)
./hwcloud serve --port 8080

# One-shot chat with streaming output
./hwcloud run "Hello, introduce yourself briefly"

# Enable OS-native sandbox for shell commands
./hwcloud serve --sandbox --port 8080

# Toggle capabilities on/off (defaults: memory/summarizer/skills/mcp/embedder on, guard/approver off)
./hwcloud serve --guard on --approver on

# Suppress all log output
./hwcloud serve -q --port 8080

# Manage secrets in the system keyring
./hwcloud keyring set mykey keyvalue
./hwcloud keyring get mykey
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
}
```

Put `AGENTS.md` and `SOUL.md` in the profile directory (default `~/.openagent/profile/`; when `OPENAGENT_CLI_CONFIG` points settings elsewhere, `<config-dir>/profile/` next to it) to customise the agent's behaviour. Project-level `$(pwd)/AGENTS.md` overrides it.

Connect an ACP client (VSCode/Zed plugin).

#### Web Search backend

The `websearch` tool supports two backends, selected by `OPENAGENT_WEB_SEARCH_ENGINE`:

| Engine | Default | Reachable in mainland China | API key | Env vars |
|--------|---------|----------------------------|---------|----------|
| `tavily` | yes | sometimes (AWS us-east) | optional (keyless works) | `TAVILY_API_KEY` (for higher rate limits) |
| `bocha` | no | yes | required | `BOCHA_API_KEY` |

Default is `tavily` (keyless, no account needed). If Tavily is unreachable from your network, the error message includes a hint to switch. To use Bocha (recommended for mainland-China users):

```bash
export OPENAGENT_WEB_SEARCH_ENGINE=bocha
export BOCHA_API_KEY=<your-key>   # get one at https://open.bochaai.com
```

### Feishu / Lark Integration

Connect your agent to Feishu (Lark) so users can chat with it in IM — group chats, private chats, cards with markdown rendering, and real-time streaming output.

<img src=".github/images/feishu-bot-effect.jpg" alt="Feishu bot in action" width="750" />

**First-time setup (no credentials needed):**

```bash
./hwcloud serve --channel feishu
```

A QR code will appear in your terminal. Open Feishu on your phone, scan it, and confirm the app creation. The SDK automatically provisions a bot app with the correct permissions (`im:message`, `im:message:send_as_bot`, `im.message.receive_v1` event, `card.action.trigger` for approval/mode button callbacks) and saves the credentials locally.

![First login - scan QR code](.github/images/feishu-first-login.jpg)

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
./hwcloud serve --channel feishu
```

The `--channel` flag is always required to start the bot — settings.json alone won't auto-start it. If your credentials are in settings.json, the setup step is skipped automatically.

![Subsequent login - start with credentials](.github/images/feishu-subsequent-login.jpg)

**Where credentials are stored:**

| Priority | Source | When to use |
|----------|--------|-------------|
| 1 | `settings.json` → `channels.feishu` | You have the app ID and secret |
| 2 | settings.json `channels.feishu` | Auto-saved after QR registration (settings is the single credential source) |
| 3 | QR code registration | First time, no credentials at all |

**Combine with other modes:**

```bash
# REST API + Feishu bot
./hwcloud serve --channel feishu

# ACP mode (stdio for VSCode/Zed) + Feishu bot
./hwcloud serve --acp --channel feishu
```

**Frontend control panel:**

The Feishu connection is a **process-level daemon** — the frontend only triggers and observes; closing or refreshing the page never affects it. Serve exposes two endpoints:

| Endpoint | Purpose |
|----------|---------|
| `GET /api/channels/feishu/status` | Connection state (`connected`, `app_id`, `connected_at`) — call on page load, then poll every few seconds |
| `POST /api/channels/feishu/connect` | Start the connection; with no persisted credentials it starts QR registration and returns `202 {status:"registration", qr_url}` for the frontend to render |
| `POST /api/channels/feishu/disconnect` | Tear down the connection (releases the machine lock); a later `POST /connect` re-establishes it |
| `GET /api/channels/feishu/qr` | The registration QR (URL + base64 PNG + remaining lifetime) — re-fetch after a page refresh; `POST /connect` is idempotent while registering and does not re-issue it |
| `GET /api/settings/channels/feishu` | The feishu configuration (`app_id`, masked `app_secret`) — the secret never leaves the server unmasked |
| `PUT /api/settings/channels/feishu` | Store the feishu configuration (`{app_id, app_secret}`, empty secret keeps the current one) — written to settings.json, applied on the next connect |
| `DELETE /api/settings/channels/feishu` | Clear the feishu credentials (a running connection keeps working until the next connect) — the "re-register" flow is DELETE + `POST /connect`, which then has no credentials and runs QR registration |

Configuration is separate from the connection: `PUT /api/settings/channels/feishu` only saves (persisted to settings.json `channels.feishu`, applied to the running server's in-memory config); applying the new values is the frontend's explicit two-step — `POST /disconnect` + `POST /connect` (a connect after disconnect cannot short-circuit on a stale "connected"). settings.json is the single credential source: manual config, frontend submissions, and QR-registration artifacts all land there. Frontend flow: load → `GET status` → show state → "Connect" button (QR registration), the config form (`GET/PUT/DELETE /api/settings/channels/feishu`), or "re-register" (DELETE + connect) when credentials failed → poll `status` until `connected`.

**Single instance per config dir:** one Feishu app = one active WebSocket. The server holds a machine-level lock (`<config-dir>/channel/feishu/feishu.lock`) for the whole connection lifetime — a second `--channel feishu` instance fails fast instead of silently stealing events. The lock is released automatically by the kernel if the process dies. For production, run under systemd/Docker so the process (and its connection) is supervised.

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

**Slash commands in IM:**

All three IM channels support `/clear` — deletes the session's conversation history and replies with a confirmation. No agent round-trip; the command is intercepted before reaching the model.

Feishu additionally supports `/mode` — switches between Manual and Auto execution modes:

| Mode | Behaviour |
|------|-----------|
| **Manual** (default) | Each non-readonly tool call shows an approval card with 同意/拒绝 buttons before executing |
| **Auto** | Tools execute immediately without human approval (higher risk) |

`/mode` with no argument shows a mode-switch card with clickable buttons. `/mode auto` or `/mode manual` switches directly. The mode is per-chat (each group/private chat remembers its own setting).

The initial mode for new chats defaults to Manual; set `"default_mode": "auto"` in settings.json to change the default.

**Run card layout (Feishu):**

Each agent run renders as a single card that updates in place (debounced patches). The body interleaves segments in arrival order: thinking (collapsed panel) → text → tool call (collapsed panel, titled with tool name + status ✓/✗) → text → … When a run completes, the card switches to an expanded state. Long runs that exceed the 28KB card limit auto-rotate: the old card folds to a collapsed "done" state and a fresh card starts with the last few blocks.

Approval requests in Manual mode embed their buttons directly in the run card (no separate approval card). When the user clicks 同意/拒绝, the card updates in-place and the agent continues or stops.

### WeChat (personal) Integration

Connect your agent to your **personal WeChat** via Tencent's official ilinkai channel (`ilinkai.weixin.qq.com`) — no SDK, plain HTTP long-poll. The agent replies once per message (WeChat has no streaming/message-edit API; a "对方正在输入" typing indicator shows while the agent works). Media markers (`[file: /path]` in reply text) are uploaded and sent as file/image messages.

**First-time setup (scan to create the bot):**

```bash
./hwcloud serve --channel wechat
```

A QR code appears in the terminal — scan it with WeChat and confirm. The bot is created automatically; if the server asks for a **pairing code** (digits shown on your phone), type it in the terminal. Credentials are saved to settings.json.

**Frontend flow:** the channel has a pairing-code step and a "scanned" state — the frontend polls the QR endpoint for the interactive flags:

| Endpoint | Purpose |
|----------|---------|
| `GET /api/channels/wechat/status` | Connection state (`connected`, `account_id`, `last_error`) |
| `POST /api/channels/wechat/connect` | Start the connection; with no credentials it starts QR login and returns `202 {status:"registration", qr_url, qr_img_base64, expires_in}` |
| `POST /api/channels/wechat/disconnect` | Tear down the connection |
| `GET /api/channels/wechat/qr` | The registration QR + `scanned` / `verify_code_required` / `verify_code_retry` flags (the frontend shows the pairing-code input box when required) |
| `POST /api/channels/wechat/verifycode` | Submit the pairing code (`{code}`) — the only channel for the `need_verifycode` step |
| `GET/PUT/DELETE /api/settings/channels/wechat` | Credentials (`token`, `base_url`, `account_id`, `user_id`; masked token on GET) |

A session that expires server-side (`errcode -14`) clears the credentials automatically — the next connect re-runs the QR login.

### WeCom (企业微信) Integration

Connect your agent to a **WeCom smart robot** via the official long-connection API (`wss://openws.work.weixin.qq.com`) — the richest of the three channels: **native streaming replies** (one message that grows in place), group chats with @-mentions, and voice already transcribed to text.

**First-time setup (scan to auto-create the robot):**

```bash
./hwcloud serve --channel wecom
```

A QR code appears — scan it with the WeCom app; the robot is created automatically and the BotID/Secret are saved to settings.json. Alternatively, create the robot manually in the WeCom admin console (安全与管理 → 管理工具 → 智能机器人 → API 模式 → 长连接) and configure it via the settings endpoint:

```json
{
  "channels": { "wecom": { "bot_id": "aibs...", "secret": "..." } }
}
```

**Frontend control panel** — same shape as Feishu (only the credential fields differ):

| Endpoint | Purpose |
|----------|---------|
| `GET /api/channels/wecom/status` | Connection state (`connected`, `bot_id`, `connected_at`, `last_error`) |
| `POST /api/channels/wecom/connect` | Start the connection; with no credentials it starts QR authorization and returns `202 {status:"registration", qr_url, qr_img_base64, expires_in}` |
| `POST /api/channels/wecom/disconnect` | Tear down the connection |
| `GET /api/channels/wecom/qr` | The authorization QR (re-fetch after a refresh) |
| `GET/PUT/DELETE /api/settings/channels/wecom` | Credentials (`bot_id`, masked `secret`) |

**Streaming replies:** the agent's answer is sent as a stream message — `finish=false` refreshes grow the same message until `finish=true` ends it. The session rate limit is 30 messages/minute.

**Connection semantics (all three channels):** settings credentials never auto-connect — the only auto-connect entry points are `--channel <name>` (fail-fast) and the frontend's `POST /connect`. Scanned/configured credentials are reused across restarts; the machine lock (`<config-dir>/channel/<name>/<name>.lock`) keeps one live connection per config dir.

### OpenViking Integration

OpenViking is a context database that provides server-side memory, skill, and resource management. Configuring an endpoint switches all three domains from local storage to the OpenViking server — one address is enough.

```json
{
  "openviking": {
    "endpoint": "http://127.0.0.1:1933",
    "api_key": "ov-xxxxxxxxxxxx"
  }
}
```

- `endpoint` — OpenViking server address. Required to enable.
- `api_key` — Bearer token sent as `Authorization: Bearer <key>`. Optional; empty = no auth (dev mode only).

To keep a specific domain on local storage while using OpenViking for the rest:

```json
{
  "openviking": { "endpoint": "http://127.0.0.1:1933", "api_key": "ov-xxx" },
  "context_providers": { "memory": "builtin" }
}
```

`context_providers` accepts `"builtin"` or `"openviking"` for each of `memory`, `skill`, `resource`. Empty = follow the endpoint default.

## Architecture

```
┌──────────────────────────────────────────────┐
│  Application (rest / acp / cmd/cli)          │
│    assembles agent config + runtime deps     │
└──────────────────┬───────────────────────────┘
                   ▼
┌──────────────────────────────────────────────┐
│  kernel.Runtime  (8-node execution engine)   │
│  ├─ context     (AgentContext assembly +     │
│  │               knowledge recall)           │
│  ├─ execution   (tool jobs, retry, stream)   │
│  ├─ governance  (approval policy chain:      │
│  │               rules→safety→memory→human)  │
│  ├─ session     (store + token-budget        │
│  │               compression)                │
│  ├─ provider/   (memory | skill | resource)  │
│  └─ eventbus    (audit events)               │
└──────────────────────────────────────────────┘
```

The `agent.Agent` is pure configuration (model, prompts, guards, sub-agents);
everything executable lives in the runtime and its dependencies — tools,
stores, policies, hooks, and observers are interfaces injected at assembly.

## Plugins

Plugins are **WASM modules** (.wasm files). Any language that compiles to WASM works — Rust, Go, TypeScript, Zig, etc. The host runtime (wazero) loads and executes them in a sandboxed environment.

A plugin declares its type(s) via metadata. A single module can implement multiple types (comma-separated, e.g. `"cli:settings,cli:http"`). We provide a Rust SDK (`plugin/pdk/rust/`, crate `openagent-pdk`) that wraps the FFI contract via the `Plugin` trait + `export!` macro — but the ABI is simple enough to implement from any language.

| Plugin type | What it does |
|-------------|--------------|
| `agent:tools` | Adds custom tools to the agent — the agent can call them like shell/read/write |
| `agent:observers` | Hooks into the agent's run pipeline (enter/leave for each stage) |
| `cli:settings` | Transforms `settings.json` at startup (merge env vars, add providers, etc.) |
| `cli:commands` | Registers extra cobra subcommands into the CLI |
| `cli:observers` | Monitors CLI command lifecycle (startup/shutdown/command enter/exit) |
| `cli:http` | Registers custom HTTP routes served at `/api/plugins/<name>/<path>` |

Any plugin type can also declare **cron-scheduled jobs** in its metadata — the host registers them with the scheduler and calls back when they fire.

### How it works

Every plugin must export `metadata()` (returns JSON with type, name, description, schedules, routes), `alloc(size)` (host-to-guest memory allocation), and optionally `dealloc(ptr)` (memory reclamation). Each plugin type then adds its own entry-point exports:

| Type | Exports | Signature |
|------|---------|-----------|
| `agent:tools` | `execute(ptr, len)` → packed JSON | Input: `ToolInput{args}`. Returns `ToolOutput{result, error}`. Tool name/description/parameters come from `metadata()`. |
| `agent:observers` | `run(ptr, len)` → packed JSON | Input: `StageInput{name, phase, detail, error}`. Returns `StageOutput{action, reason}` — `"continue"` or `"abort"`. |
| `cli:settings` | `init(ptr, len)` → packed JSON | Input: current settings JSON. Returns merged settings JSON. |
| `cli:commands` | `commands()` → JSON | Returns `CommandDef[]` (name, use, short, long, args, flags, children, aliases, example). |
| | `run_<name>(ptr, len)` → packed | One export per leaf command (dashes → underscores). Input: `CommandInput{args, flags}`. |
| `cli:observers` | `on_startup()` / `on_shutdown()` | No-arg lifecycle callbacks. |
| | `on_command_start(ptr, len)` / `on_command_end(ptr, len)` | Input: command path string (end includes error suffix). |
| `cli:http` | `handle_request(ptr, len)` → packed JSON | Input: `HttpRequest{method, path, params, query, headers, body}`. Returns `HttpResp{status, headers, body}`. |
| Scheduled jobs | `run_scheduled(ptr, len)` → packed JSON | Input: `ScheduledJobInput{id, scheduled_at}`. Returns `ScheduledJobResult{result, error}`. |

The host runtime (wazero + `plugin/wasmhost/`) provides 28 importable host functions that plugins can call.

### Enabling plugins

Place `.wasm` files in a directory (or reference individual files) and configure in `settings.json`:

```json
{
  "plugins": ["~/.openagent/plugins"]
}
```

Each entry may be a directory (scanned for `*.wasm`) or a single `.wasm` file path. At startup the CLI reads each module's metadata, instantiates it, and wires it into the agent or CLI.

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

Or use the Makefile (builds tool, observer, and envsync examples):

```bash
make -C examples/plugin
```

### Writing a tool plugin (agent:tools)

```rust
#![no_std]
#![no_main]
extern crate alloc;
use openagent_pdk::prelude::*;
use openagent_pdk::export::Plugin;

struct EchoPlugin;
impl Plugin for EchoPlugin {
    fn plugin_type() -> &'static str { "agent:tools" }
    fn name() -> &'static str { "echo" }
    fn description() -> &'static str { "Echo back the input message." }
    fn tool_parameters() -> Option<&'static str> {
        Some(r#"{"type":"object","properties":{"message":{"type":"string"}},"required":["message"]}"#)
    }
    fn execute(args: &serde_json::Value) -> Result<String, String> {
        let msg = args.get("message").and_then(|v| v.as_str()).unwrap_or("(empty)");
        Ok(format!("echo: {}", msg))
    }
}

openagent_pdk::export!(EchoPlugin);
```

### Writing an observer plugin (agent:observers)

```rust
#![no_std]
#![no_main]
extern crate alloc;
use openagent_pdk::prelude::*;
use openagent_pdk::export::Plugin;

struct LoggerPlugin;
impl Plugin for LoggerPlugin {
    fn plugin_type() -> &'static str { "agent:observers" }
    fn name() -> &'static str { "observer_logger" }
    fn stage_filter() -> (&'static str, &'static str) { ("*", "*") }

    fn observe_stage(event: &StageInput) -> StageOutput {
        host::log_info(&format!("stage={} phase={}", event.name, event.phase));
        StageOutput { action: String::from("continue"), reason: String::new() }
    }
}

openagent_pdk::export!(LoggerPlugin);
```

### Writing a scheduled-job plugin

```rust
#![no_std]
#![no_main]
extern crate alloc;
use openagent_pdk::prelude::*;
use openagent_pdk::export::Plugin;

struct EnvSyncPlugin;
impl Plugin for EnvSyncPlugin {
    fn name() -> &'static str { "envsync" }
    fn description() -> &'static str { "Sync keyring secret into env every 5 min" }

    fn scheduled_jobs() -> Vec<ScheduledJob> {
        vec![ScheduledJob {
            id: "sync-keyring-env".into(),
            cron: "*/5 * * * *".into(),
            description: "sync keyring secret into env".into(),
        }]
    }

    fn run_scheduled_job(job: &ScheduledJobInput) -> Result<String, String> {
        let secret = host::keyring_get("openagent", "PLUGIN_SECRET")?;
        host::env_set("OPENAGENT_PLUGIN_SECRET", &secret)?;
        Ok(format!("job {} synced {} bytes", job.id, secret.len()))
    }
}

openagent_pdk::export!(EnvSyncPlugin);
```

### Host API (importable from any language)

| Category | Function | Purpose |
|----------|----------|---------|
| Logging | `log_info(msg)` / `log_warn(msg)` / `log_error(msg)` | Logging through the host |
| Time | `utc_now() -> u64` | Current time in nanoseconds |
| Keyring | `keyring_get(service, key) -> string` | Read from system keyring |
| | `keyring_set(service, key, value)` | Write to system keyring |
| | `keyring_delete(service, key)` | Delete from system keyring |
| HTTP | `http_request(method, url, headers_json, body) -> {status, body}` | Outbound HTTP |
| Process | `exec_command(cmd, args, cwd, env, env_replace, timeout_ms) -> {stdout, stderr, exit_code}` | Run a child process |
| Env | `env_get(key) -> string` | Read host process env var |
| | `env_set(key, value)` | Set host process env var |
| | `env_unset(key)` | Unset host process env var |
| | `env_list() -> [{key, value}]` | List full host environment |
| Filesystem | `fs_read(path) -> base64` | Read file as base64 |
| | `fs_write(path, data)` | Write file |
| | `fs_readdir(path) -> [{name, is_dir}]` | List directory |
| | `file_md5(path) -> string` | MD5 hash of a file |
| | `directory_md5(path) -> string` | Aggregate MD5 of a directory |
| Runtime | `runtime_session_id() -> string` | Current session ID |
| | `runtime_user_id() -> string` | Current user ID |
| | `runtime_turn_count() -> string` | Current turn count |
| | `runtime_model_id() -> string` | Current model ID |
| | `runtime_provider() -> string` | Current provider name |
| | `runtime_get_metadata(key) -> string` | Read session metadata |
| | `runtime_set_metadata(key, value)` | Write session metadata |
| | `runtime_set_model_config(json)` | Replace model (provider/model_id/api_key/base_url) |
| | `runtime_set_system_prompts(json)` | Override system prompts |
| | `runtime_set_max_turns(n)` | Override max turns |

Full examples: `examples/plugin/`. Rust SDK: `plugin/pdk/rust/`.

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
| `examples/plugin/` | WASM tool, observer, and scheduled-job plugins |
| `examples/skill/` | On-demand skill loading |
| `examples/acp/` | ACP agent protocol (server + client) |
| `examples/artifact/` | Result policy — large tool results spill to disk |
| `examples/browser-agent/` | Browser agent via Playwright MCP |
| `examples/mcp-client/` | MCP client demo (IaC pipeline) |
| `examples/frontend/` | Vue.js frontend control panel (channel status, settings, QR rendering) |
| `cmd/cli/` | Full-featured CLI with WASM plugin runtime |
| `cmd/tui/` | TUI chat client (bubbletea v2, streaming, human-in-the-loop approval) |

## Packages

| Package | Purpose |
|---------|---------|
| `openagent` | Core types — Agent (pure config), Team, ToolResult, token helpers |
| `agent/` | Agent config builders — options, goal instructions, sub-agent router |
| `kernel/` | Runtime — the 8-node execution engine (memory → prompt → guard → model → guard → policy → tools → store) |
| `execution/` | Tool execution — parallel jobs, retry, streaming, result policy |
| `governance/` | Approval policy engine — rules → safety → memory → human, persistent decisions |
| `context/` | AgentContext assembly — knowledge recall, skill match, LLM extractor (self-evolution) |
| `acp/sdk/` | ACP v1 protocol SDK — types, JSON-RPC 2.0 mux, client |
| `acp/` | AgentServer — wraps an Agent as an ACP handler |
| `rest/` | REST + SSE handlers (single, team, orchestrate) |
| `orchestrate/` | Multi-agent DAG decomposition + streaming execution |
| `plan/` | `plan_create`/`plan_update` tools (ACP plan mode) |
| `slash/` | Slash command registry and dispatch |
| `summarizer/` | LLM-based incremental conversation compression |
| `session/` | Session store interface + token-budget compression |
| `session/sqlite/` | SQLite session store (FTS5 full-text index) |
| `session/file/` | File-backed session store |
| `provider/memory/` | Durable knowledge backends (sqlite with vector recall, file) |
| `provider/skill/` | On-demand skill matching/loading |
| `provider/resource/` | External reference resources |
| `provider/openviking/` | OpenViking context database client (memory/skill/resource over HTTP) |
| `model/openai/` | OpenAI ChatCompletion + streaming + embeddings |
| `embedder/` | Embedding backends — BGE (offline ONNX Runtime) and OpenAI-compatible API |
| `tokenizer/` | tiktoken model-aware token counting (sampled estimate for huge texts) |
| `sandbox/native/` | OS-level process confinement (bwrap/Seatbelt) |
| `eventbus/` | Session-scoped pub/sub for SSE |
| `plugin/wasmhost/` | Shared WASM host module (keyring, HTTP, filesystem, env, logging, runtime) |
| `plugin/agent/wasm/` | Agent-scoped WASM plugin host |
| `plugin/cli/` | CLI plugin manager and types |
| `plugin/cli/wasm/` | CLI-scoped WASM runtime, loader, observer hub, HTTP route dispatcher |
| `plugin/pdk/rust/` | Rust SDK crate (`openagent-pdk`) for building WASM plugins |
| `skill/fs/` | Filesystem skill loader |
| `mcp/` | Model Context Protocol client |
| `guard/llm/` | LLM-based input/output guard |
| `hooks/otel/` | OpenTelemetry hooks |
| `hooks/slog/` | Structured logging hooks |
| `hooks/redact/` | Masks sensitive env-var values in tool results |
| `tool/` | Built-in tools (shell, read, write, ls, grep, edit, websearch, webfetch, ACP fs, ACP terminal) |
| `channel/` | IM platform adapters — Feishu (WebSocket, card rendering), WeChat (ilinkai HTTP), WeCom (长连接 streaming) |
| `keyring/` | System keychain wrapper (Linux Secret Service/kernel keyring, macOS Keychain, Windows Credential Manager) |
| `process/` | Background shell-process lifecycle management (track, persist output, kill across turns) |
| `scheduler/` | Cron-based job scheduling for WASM plugin scheduled tasks |
| `utils/` | Shared helpers — SSRF-hardened HTTP client, filepath validation, flock, JSON |
| `iac/` | Terraform wrapper — binary install/mirror management, init/plan/apply/destroy |
| `version/` | Build-time binary identity (name + version via ldflags) |
| `cmd/cli/` | CLI runtime, WASM host, REST/ACP server, settings, channel managers |
| `cmd/tui/` | TUI chat client (bubbletea v2) |
| `cmd/mcp/` | IaC MCP server — cloud deployment tools (HuaweiCloud, Aliyun) over MCP stdio |

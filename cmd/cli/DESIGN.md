# openagent-cli Design

## What is it?

openagent-cli is the CLI entry point for openagent-go. It is not a simple launch script — it is an **extensible agent runtime platform**:

1. Configuration from `~/.openagent/settings.json`
2. Capabilities injected by WASM plugins at startup
3. Exposes REST or ACP service on launch

## Core Principle

> CLI is the Runtime. Plugin is the Capability Provider.
> User config + Plugin injection → merged RuntimeContext → build Agent → start server.

---

## Architecture

```
settings.json              plugins/*.wasm            Final Config
══════════════             ════════════             ════════════
{                          extended-settings        {
  "model": "...",            ↓                        "model": "...",
  "provider": {...},        read keyring              "provider": {
  "server": {...}           inject provider             deepseek: {...},
}                           inject env vars             huawei: {...}
                                                        },
                                                        "env": {...}
                                                      }
```

```
main()
  ├─ 1. Read settings.json
  ├─ 2. Open system keyring
  ├─ 3. Create one WASM runtime (wazero)
  ├─ 4. For each .wasm in plugins/:
  │      ├─ Instantiate module
  │      ├─ Read metadata → route by type
  │      ├─ "cli:settings"  → pipe settings through init(settings)
  │      ├─ "cli:commands"  → register cobra commands
  │      └─ "cli:observers" → add to ObserverHub
  ├─ 5. Parse final merged Config
  ├─ 6. Set env vars from Config.Env
  ├─ 7. Build cobra command tree
  ├─ 8. Wire observer hooks (on_startup, on_command_start/end, on_shutdown)
  └─ 9. Execute cobra
```

---

## Plugin Types

| Type | metadata.type | Exports | Purpose |
|------|--------------|---------|---------|
| Settings injection | `cli:settings` | `init(settings) → settings` | Pipeline-transform settings JSON. Plugin reads keyring/HTTP, returns merged result. |
| Command injection | `cli:commands` | `commands()`, `run_<name>(args)` | Register new cobra commands at startup. |
| Lifecycle observer | `cli:observers` | `on_startup`, `on_shutdown`, `on_command_start`, `on_command_end` | Hook into CLI lifecycle for telemetry, monitoring, audit. |

A single plugin can have multiple types: `"type": "cli:settings,cli:commands"`.

---

## Plugin ABI (Language-Neutral)

### Guest exports (plugin → host)

```
alloc(size: u32) → ptr: u32
metadata() → (ptr: u32, len: u32) packed as u64   // CLIPluginMeta JSON
init(settings_ptr: u32, settings_len: u32) → u64  // merged settings JSON (optional)
commands() → u64                                    // [CommandDef] JSON (optional)
run_<name>(args_ptr: u32, args_len: u32) → u64     // output text (optional)
on_startup()                                        // (optional)
on_shutdown()                                       // (optional)
on_command_start(cmd_ptr: u32, cmd_len: u32)        // (optional)
on_command_end(cmd_ptr: u32, cmd_len: u32)          // (optional)
```

### Host imports (host → plugin, via wazero "host" module)

```
keyring_get(service_ptr, service_len, key_ptr, key_len) → u64
keyring_set(service_ptr, service_len, key_ptr, key_len, val_ptr, val_len)
http_request(method_ptr,method_len, url_ptr,url_len, headers_ptr,headers_len, body_ptr,body_len) → u64
log_info(msg_ptr, msg_len)
log_warn(msg_ptr, msg_len)
log_error(msg_ptr, msg_len)
```

All types: `(ptr: u32, len: u32)` paired as `u64` (high 32 = ptr, low 32 = len).

---

## settings.json

```json
{
  "model": "deepseek/deepseek-v3",
  "fast_model": "deepseek/deepseek-v4-flash",
  "provider": {
    "deepseek": {
      "api_key": "sk-xxx",
      "base_url": "https://api.deepseek.com/v1"
    }
  },
  "server": { "port": 8080 },
  "plugins": ["~/.openagent/plugins"],
  "env": {}
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `model` | No | Default model, format `provider_id/model_id` |
| `fast_model` | No | Lightweight model for quick tasks |
| `provider.<id>.api_key` | No | API key; falls back to env `<ID>_API_KEY` or keyring |
| `provider.<id>.base_url` | Yes | API endpoint |
| `provider.<id>.models` | No | Explicit model list; omit to auto-discover via `GET /models` |
| `server.port` | No | Default 8080 |
| `plugins` | No | List of dirs/files; default `~/.openagent/plugins` |
| `env` | No | Static env vars; plugins can inject additional ones |

---

## Config merge (Pipeline)

Settings-injection plugins receive the **current** settings JSON as input and return the **merged** result. Plugins are applied in order:

```
settings.json → plugin_0.init(settings) → plugin_1.init(merged) → ... → final Config
```

The host never distinguishes between "user config" and "plugin config". Everything goes through the same Config struct.

---

## Command Tree (cobra)

```
openagent-cli
├─ serve [--acp] [--port N]        Built-in: REST or ACP server
├─ keyring                          Built-in: credential management
│  ├─ set <key> <value>
│  ├─ get <key>
│  └─ delete <key>
└─ <plugin commands>                 Injected by cli:commands plugins
```

---

## Observer Lifecycle

```
on_startup()
  → on_command_start("openagent-cli serve")
    → server running...
    → Ctrl+C
  → on_command_end("openagent-cli serve", nil)
on_shutdown()
```

All commands (built-in and plugin) trigger `on_command_start`/`on_command_end` via a recursive `wrapCmd()` that wraps every `RunE` in the cobra tree.

---

## Rust SDK

Plugins are built with the `openagent-cli-sdk` crate. The SDK provides:

| Module | What |
|--------|------|
| `prelude::*` | `String`, host API, ABI helpers |
| `host::keyring_get(service, key)` | Read credential |
| `host::keyring_set(service, key, val)` | Store credential |
| `host::http_request(method, url, headers, body)` | HTTP request via host |
| `host::log_info(msg)` | Log via host |
| `sdk_alloc(size)` | Bump allocator (128 KB heap) |
| `sdk_meta(json)` | Metadata export helper |
| `sdk_return(data)` | Allocate + pack return value |

Example (settings plugin):

```rust
#![no_std]
extern crate openagent_cli_sdk as sdk;
use sdk::prelude::*;

static META: &str = r#"{"type":"cli:settings","name":"my-plugin","description":"..."}"#;

#[no_mangle] pub extern "C" fn alloc(size: u32) -> u32 { sdk_alloc(size) }
#[no_mangle] pub extern "C" fn metadata() -> u64 { sdk_meta(META) }

#[no_mangle]
pub extern "C" fn init(p: u32, l: u32) -> u64 {
    let settings = unsafe { wasm_str(p, l) };
    let ak = host::keyring_get("openagent", "my_key").unwrap_or("");
    // ... build merged JSON ...
    sdk_return(merged.as_bytes())
}
```

Build: `cargo build --release --target wasm32-unknown-unknown -p openagent-cli-sdk && rustc ... --extern openagent_cli_sdk`

---

## Directory Layout

```
cmd/cli/
  main.go                     Entry point, startup sequence, cobra tree
  config/config.go             Settings loader, defaults, OPENAGENT_CLI_CONFIG env var
  keyring/keyring.go           System keychain via go-keyring
  plugin/
    host_api.go                Go interfaces: Keyring, HTTPClient, Logger
    manager.go                 Scans plugin dirs, resolves .wasm files
    wasm/
      runtime.go               Single wazero runtime, registers "host" module
      host_module.go           Host exports: keyring_*, http_request, log_*
      loader.go                Instantiate, CallInit, ReadCommands, RunCommand
      observer.go              ObserverHub: OnStartup/OnShutdown/OnCommandStart/OnCommandEnd
      abi.go                   PluginMeta, Is() matcher, CommandDef
  server/agent.go              Build openagent.Agent from Config, REST + ACP server
  plugin/sdk/rust/                    Rust SDK crate (openagent-cli-sdk)
  examples/plugin/
    extended-settings.rs       cli:settings  — reads keyring, merges provider+env
    stats-cmd.rs               cli:commands  — adds "stats" command
    telemetry.rs               cli:observers — logs lifecycle events
  build/
    openagent-cli              Compiled binary
    plugins/                   Compiled .wasm files
```

---

## Key Design Decisions

1. **Plugins receive settings, return settings.** No separate registration API. The plugin is a pure function `settings → settings`. Simple, testable, composable.

2. **One WASM runtime, one instantiation per .wasm, multiple capability types.** A single .wasm can export `init()`, `commands()`, and `on_startup()` — the host routes by metadata type.

3. **Keyring, not settings.json, for secrets.** `api_key` is optional in settings. Plugins read credentials from the system keychain at startup.

4. **cobra `RunE` wrapping for observer error propagation.** `PersistentPreRunE` doesn't pass errors. We recursively wrap every `RunE` to capture the original error for `on_command_end`.

5. **`filepath.Join` everywhere.** No string concatenation for paths. Portable.

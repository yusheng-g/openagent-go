# openagent-cli

The command-line entry point for [openagent-go](https://github.com/yusheng-g/openagent-go). Start an LLM agent server (REST or ACP) with config-driven models and WASM-powered plugins.

## Quick Start

Build and run:

```bash
go build -o openagent-cli ./cmd/cli/
mkdir -p ~/.openagent/plugins
```

Create `~/.openagent/settings.json`:

```json
{
  "model": "deepseek/deepseek-v3",
  "fast_model": "deepseek/deepseek-v4-flash",
  "provider": {
    "deepseek": {
      "api_key": "sk-your-key",
      "base_url": "https://api.deepseek.com/v1"
    }
  },
  "server": { "port": 8080 },
  "plugins": ["~/.openagent/plugins"]
}
```

Start the server:

```bash
./openagent-cli serve
```

The REST API is now on `http://localhost:8080`. Connect the [frontend](https://github.com/yusheng-g/openagent-go/tree/master/examples/frontend) or any OpenAI-compatible client.

For ACP (Agent Communication Protocol) over stdio:

```bash
./openagent-cli serve --acp
```

## Configuration

### Settings File

All configuration lives in `~/.openagent/settings.json`. The path can be overridden via `OPENAGENT_CLI_CONFIG` env var.

| Field | Required | Description |
|-------|----------|-------------|
| `model` | No | Default model: `provider_id/model_id` (e.g. `deepseek/deepseek-v3`) |
| `fast_model` | No | Lightweight model for quick tasks |
| `provider.<id>.api_key` | No | API key. Falls back to env `<ID>_API_KEY` or system keyring |
| `provider.<id>.base_url` | Yes | API endpoint |
| `provider.<id>.models` | No | Model list. Omit to auto-discover via `GET /models` |
| `server.port` | No | Server port (default: 8080) |
| `plugins` | No | Plugin directories or `.wasm` files (default: `~/.openagent/plugins`) |
| `env` | No | Static environment variables for the process |

### Keyring

Credentials that shouldn't live in settings.json (API keys, JWTs, anything sensitive) go into your system keychain:

```bash
./openagent-cli keyring set my_provider_api_key sk-xxx
./openagent-cli keyring set my_provider_base_url https://api.example.com/v1
./openagent-cli keyring set my_provider_models model-a,model-b
```

Plugins read these at startup. Supported backends: Linux Secret Service (gnome-keyring/kwallet), macOS Keychain, Windows Credential Manager.

## Plugins

Plugins are `.wasm` files loaded at startup. There are three types:

| Type | `metadata.type` | What it does |
|------|----------------|--------------|
| Settings injection | `cli:settings` | Transforms `settings.json` at startup — reads keyring, adds providers, injects env vars |
| Command injection | `cli:commands` | Adds new `openagent-cli` commands (e.g. `openagent-cli stats`) |
| Lifecycle observer | `cli:observers` | Hooks into startup, shutdown, and command execution for telemetry |

A single plugin can mix types: `"type": "cli:settings,cli:commands,cli:observers"`.

### Building Plugins

Plugins are built in Rust with the `openagent-cli-sdk` crate:

```bash
# Build the SDK
cd cmd/cli
cargo build --release --target wasm32-unknown-unknown -p openagent-cli-sdk

# Build a plugin
rustc --target wasm32-unknown-unknown -C opt-level=z --edition 2021 \
  --crate-type cdylib -C link-arg=--no-entry \
  -L target/wasm32-unknown-unknown/release \
  --extern openagent_cli_sdk \
  -o build/plugins/my-plugin.wasm examples/plugin/my-plugin.rs
```

### Example: Settings Plugin

Reads credentials from keyring and injects a provider + env vars:

```rust
#![no_std]
extern crate openagent_cli_sdk as sdk;
use sdk::prelude::*;

static META: &str = r#"{"type":"cli:settings","name":"my-plugin","description":"Adds my provider"}"#;

#[no_mangle] pub extern "C" fn alloc(size: u32) -> u32 { sdk_alloc(size) }
#[no_mangle] pub extern "C" fn metadata() -> u64 { sdk_meta(META) }

#[no_mangle]
pub extern "C" fn init(p: u32, l: u32) -> u64 {
    let settings = unsafe { wasm_str(p, l) };
    let ak = host::keyring_get("openagent", "my_provider_api_key").unwrap_or("");
    // ... build merged settings ...
    sdk_return(merged.as_bytes())
}
```

See `examples/plugin/` for more examples.

### Example: Command Plugin

```rust
#![no_std]
extern crate openagent_cli_sdk as sdk;
use sdk::prelude::*;

static META: &str = r#"{"type":"cli:commands","name":"my-cmd","description":"My command"}"#;
static CMDS: &str = r#"[{"name":"my-cmd","use":"my-cmd","short":"Runs my command"}]"#;

#[no_mangle] pub extern "C" fn alloc(size: u32) -> u32 { sdk_alloc(size) }
#[no_mangle] pub extern "C" fn metadata() -> u64 { sdk_meta(META) }
#[no_mangle] pub extern "C" fn commands() -> u64 { sdk_return(CMDS.as_bytes()) }

#[no_mangle]
pub extern "C" fn run_my_cmd(p: u32, l: u32) -> u64 {
    let args = unsafe { wasm_str(p, l) };
    let result = format!("Hello from my plugin! args: {}", args);
    sdk_return(result.as_bytes())
}
```

## Commands

```
openagent-cli
├─ serve [--acp] [--port N]        REST or ACP server
├─ keyring                          Credential management
│  ├─ set <key> <value>
│  ├─ get <key>
│  └─ delete <key>
└─ <plugin commands>                 Injected at startup
```

## Architecture

See [DESIGN.md](DESIGN.md) for the full architecture, plugin ABI specification, and design rationale.

## License

MIT — see the root [LICENSE](../../LICENSE) file.

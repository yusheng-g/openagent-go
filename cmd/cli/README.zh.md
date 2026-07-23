# openagent-cli

[openagent-go](https://github.com/yusheng-g/openagent-go) 的命令行入口。通过配置驱动模型和 WASM 插件，启动一个 LLM Agent 服务器（REST 或 ACP）。

## 快速开始

编译并运行：

```bash
go build -o openagent-cli ./cmd/cli/
mkdir -p ~/.openagent/plugins
```

创建 `~/.openagent/settings.json`：

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

启动服务器：

```bash
./openagent-cli serve
```

REST API 运行在 `http://localhost:8080`。连接[前端](https://github.com/yusheng-g/openagent-go/tree/master/examples/frontend)或任何 OpenAI 兼容客户端。

ACP（Agent 通信协议）模式：

```bash
./openagent-cli serve --acp
```

## 配置

### 设置文件

所有配置存放在 `~/.openagent/settings.json` 中。可通过 `OPENAGENT_CLI_CONFIG` 环境变量覆盖文件路径。

| 字段 | 必填 | 说明 |
|------|------|------|
| `model` | 否 | 默认模型，格式：`provider_id/model_id`（如 `deepseek/deepseek-v3`） |
| `fast_model` | 否 | 轻量快速模型 |
| `provider.<id>.api_key` | 否 | API 密钥；可从 `env` 或系统 keyring 读取 |
| `provider.<id>.base_url` | 是 | API 端点 |
| `provider.<id>.models` | 否 | 模型列表；不填则通过 `GET /models` 自动发现 |
| `server.port` | 否 | 端口（默认 8080） |
| `plugins` | 否 | 插件目录或 `.wasm` 文件列表（默认 `~/.openagent/plugins`） |
| `env` | 否 | 进程静态环境变量 |

### 系统 Keyring

不应写在 settings.json 中的凭证（API 密钥、JWT 等）存入系统密钥环：

```bash
./openagent-cli keyring set my_provider_api_key sk-xxx
./openagent-cli keyring set my_provider_base_url https://api.example.com/v1
./openagent-cli keyring set my_provider_models model-a,model-b
```

插件在启动时读取这些密钥。支持：Linux Secret Service（gnome-keyring/kwallet）、macOS Keychain、Windows Credential Manager。

## 插件

插件是启动时加载的 `.wasm` 文件，分三类：

| 类型 | `metadata.type` | 功能 |
|------|----------------|------|
| 设置注入 | `cli:settings` | 启动时修改 settings.json：读 keyring、添加 provider、注入环境变量 |
| 命令注入 | `cli:commands` | 添加新的 `openagent-cli` 命令（如 `openagent-cli stats`） |
| 生命周期观察 | `cli:observers` | 钩入启动、关闭、命令执行过程，用于遥测监控 |

一个插件可混搭多种类型：`"type": "cli:settings,cli:commands,cli:observers"`。

### 编译插件

插件用 Rust + `openagent-pdk` crate 编译：

```bash
# 构建 SDK
cd cmd/cli
cargo build --release --target wasm32-unknown-unknown -p openagent-pdk

# 构建插件
rustc --target wasm32-unknown-unknown -C opt-level=z --edition 2021 \
  --crate-type cdylib -C link-arg=--no-entry \
  -L target/wasm32-unknown-unknown/release \
  --extern openagent_pdk \
  -o build/plugins/my-plugin.wasm examples/plugin/my-plugin.rs
```

### 示例：设置注入插件

从 keyring 读取凭证，注入 provider 和环境变量：

```rust
#![no_std]
extern crate openagent_pdk as sdk;
use sdk::prelude::*;

static META: &str = r#"{"type":"cli:settings","name":"my-plugin","description":"添加我的 provider"}"#;

#[no_mangle] pub extern "C" fn alloc(size: u32) -> u32 { sdk_alloc(size) }
#[no_mangle] pub extern "C" fn metadata() -> u64 { sdk_meta(META) }

#[no_mangle]
pub extern "C" fn init(p: u32, l: u32) -> u64 {
    let settings = unsafe { wasm_str(p, l) };
    let ak = host::keyring_get("openagent", "my_provider_api_key").unwrap_or("");
    // ... 构建合并后的 JSON ...
    sdk_return(merged.as_bytes())
}
```

更多示例见 `examples/plugin/`。

### 示例：命令注入插件

```rust
#![no_std]
extern crate openagent_pdk as sdk;
use sdk::prelude::*;

static META: &str = r#"{"type":"cli:commands","name":"my-cmd","description":"我的命令"}"#;
static CMDS: &str = r#"[{"name":"my-cmd","use":"my-cmd","short":"执行我的命令"}]"#;

#[no_mangle] pub extern "C" fn alloc(size: u32) -> u32 { sdk_alloc(size) }
#[no_mangle] pub extern "C" fn metadata() -> u64 { sdk_meta(META) }
#[no_mangle] pub extern "C" fn commands() -> u64 { sdk_return(CMDS.as_bytes()) }

#[no_mangle]
pub extern "C" fn run_my_cmd(p: u32, l: u32) -> u64 {
    let args = unsafe { wasm_str(p, l) };
    let result = format!("来自我的插件！参数: {}", args);
    sdk_return(result.as_bytes())
}
```

## 命令

```
openagent-cli
├─ serve [--acp] [--port N] [--sandbox]   REST 或 ACP 服务
├─ keyring                                 凭证管理
│  ├─ set <key> <value>
│  ├─ get <key>
│  └─ delete <key>
└─ <插件命令>                               启动时注入
```

`--sandbox` 启用 OS 原生沙箱（Linux bwrap / macOS Seatbelt），默认关闭。关闭时命令直接在宿主机执行。启用后命令在隔离的命名空间中运行，仅工作区可写。

## 架构

完整架构、插件 ABI 规范及设计理由见 [DESIGN.zh.md](DESIGN.zh.md)。

## 协议

MIT — 参见根目录 [LICENSE](../../LICENSE) 文件。

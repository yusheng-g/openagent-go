# openagent-cli 设计文档

## 这是什么？

openagent-cli 是 openagent-go 的命令行入口。不是简单的启动脚本，而是一个**可扩展的 Agent 运行时平台**：

1. 配置来自 `~/.openagent/settings.json`
2. 能力由 WASM 插件在启动时动态注入
3. 启动后输出 REST 或 ACP 服务

## 核心原则

> CLI 是 Runtime，Plugin 是能力提供者 (Capability Provider)。
> 用户配置 + 插件注入 → 合并 RuntimeContext → 构建 Agent → 启动服务。

---

## 总体架构

### 启动流程

```
main()
  ├─ 1. 读取 settings.json
  ├─ 2. 打开系统 keyring
  ├─ 3. 创建一个 WASM 运行时 (wazero)
  ├─ 4. 遍历 plugins/ 下每个 .wasm:
  │      ├─ 实例化模块
  │      ├─ 读 metadata → 按类型路由
  │      ├─ "cli:settings"  → 管道式 transform: init(settings) → 新 settings
  │      ├─ "cli:commands"  → 注册 cobra 命令
  │      └─ "cli:observers" → 加入 ObserverHub
  ├─ 5. 解析最终合并的 Config
  ├─ 6. 从 Config.Env 设置环境变量
  ├─ 7. 构建 cobra 命令树
  ├─ 8. 安装观察者钩子 (on_startup, on_command_start/end, on_shutdown)
  └─ 9. 执行 cobra
```

### 配置合并 (Pipeline)

```
settings.json → plugin_0.init(settings) → plugin_1.init(merged) → ... → 最终 Config
```

Host 不区分"用户配置"和"插件配置"，全部经过同一个 Config struct。

---

## 三类插件

| 类型 | metadata.type | 导出函数 | 作用 |
|------|--------------|---------|------|
| 配置注入 | `cli:settings` | `init(settings) → settings` | 管道式 transform。插件读 keyring/HTTP，返回合并结果。 |
| 命令注入 | `cli:commands` | `commands()` + `run_<name>(args)` | 启动时注册 cobra 命令。 |
| 生命周期观察 | `cli:observers` | `on_startup`, `on_shutdown`, `on_command_start`, `on_command_end` | 钩入 CLI 生命周期，做遥测、监控、审计。 |

一个插件可以有多种类型：`"type": "cli:settings,cli:commands"`。

---

## Plugin ABI (语言无关的二进制接口)

### Guest 导出 (plugin → host)

```
alloc(size: u32) → ptr: u32
metadata() → (ptr, len) 打包为 u64      // CLIPluginMeta JSON
init(settings_ptr, settings_len) → u64   // 合并后的 settings JSON
commands() → u64                          // [CommandDef] JSON
run_<name>(args_ptr, args_len) → u64     // 输出文本
on_startup()
on_shutdown()
on_command_start(cmd_ptr, cmd_len)
on_command_end(cmd_ptr, cmd_len)
```

### Host 导入 (host → plugin, wazero "host" 模块)

```
keyring_get(service_ptr,service_len, key_ptr,key_len) → u64
keyring_set(service_ptr,service_len, key_ptr,key_len, val_ptr,val_len)
http_request(method_ptr,method_len, url_ptr,url_len, headers_ptr,headers_len, body_ptr,body_len) → u64
log_info(msg_ptr, msg_len)
log_warn(msg_ptr, msg_len)
log_error(msg_ptr, msg_len)
```

所有类型：`(ptr: u32, len: u32)` 对，打包为一个 `u64`（高32位=ptr，低32位=len）。

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

| 字段 | 必填 | 说明 |
|------|------|------|
| `model` | 否 | 默认模型，格式 `provider_id/model_id` |
| `fast_model` | 否 | 轻量快速模型 |
| `provider.<id>.api_key` | 否 | API 密钥；可从 `env` 或 keyring 读取 |
| `provider.<id>.base_url` | 是 | API 端点 |
| `provider.<id>.models` | 否 | 显式模型列表；不填则自动发现 `GET /models` |
| `server.port` | 否 | 默认 8080 |
| `plugins` | 否 | 插件目录或文件列表；默认 `~/.openagent/plugins` |
| `env` | 否 | 静态环境变量；插件可注入更多 |

---

## 命令树 (cobra)

```
openagent-cli
├─ serve [--acp] [--port N]        内置: REST 或 ACP 服务
├─ keyring                          内置: 凭证管理
│  ├─ set <key> <value>
│  ├─ get <key>
│  └─ delete <key>
└─ <插件命令>                        由 cli:commands 插件注入
```

---

## 观察者生命周期

```
on_startup()
  → on_command_start("openagent-cli serve")
    → server 运行中...
    → Ctrl+C
  → on_command_end("openagent-cli serve", nil)
on_shutdown()
```

所有命令（内置和插件）都通过 `wrapCmd()` 递归包装每个 `RunE`，从而向 `on_command_start`/`on_command_end` 传递正确的错误信息。

---

## Rust SDK

插件使用 `openagent-pdk` crate 构建：

| 模块 | 功能 |
|------|------|
| `prelude::*` | `String`、host API、ABI 助手 |
| `host::keyring_get(service, key)` | 读凭证 |
| `host::keyring_set(service, key, val)` | 存凭证 |
| `host::http_request(method, url, headers, body)` | 通过 host 发 HTTP |
| `host::log_info(msg)` | 通过 host 记日志 |
| `sdk_alloc(size)` | 128KB bump 分配器 |
| `sdk_meta(json)` | metadata 导出助手 |
| `sdk_return(data)` | 分配 + 打包返回值 |

设置注入示例：

```rust
#![no_std]
extern crate openagent_pdk as sdk;
use sdk::prelude::*;

static META: &str = r#"{"type":"cli:settings","name":"my-plugin","description":"..."}"#;

#[no_mangle] pub extern "C" fn alloc(size: u32) -> u32 { sdk_alloc(size) }
#[no_mangle] pub extern "C" fn metadata() -> u64 { sdk_meta(META) }

#[no_mangle]
pub extern "C" fn init(p: u32, l: u32) -> u64 {
    let settings = unsafe { wasm_str(p, l) };
    let ak = host::keyring_get("openagent", "my_key").unwrap_or("");
    // ... 构建合并后的 JSON ...
    sdk_return(merged.as_bytes())
}
```

编译：`cargo build --release --target wasm32-unknown-unknown -p openagent-pdk && rustc ... --extern openagent_pdk`

---

## 目录结构

```
cmd/cli/
  main.go                     入口、启动序列、cobra 命令树
  config/config.go             settings 加载器、默认值、OPENAGENT_CLI_CONFIG 环境变量
  keyring/keyring.go           系统 keyring (go-keyring)
  plugin/
    host_api.go               Go 接口: Keyring, HTTPClient, Logger
    manager.go                扫描插件目录、resolve .wasm 文件
    wasm/
      runtime.go              单一 wazero 运行时、注册 "host" 模块
      host_module.go          host 导出: keyring_*, http_request, log_*
      loader.go               实例化、CallInit、ReadCommands、RunCommand
      observer.go             ObserverHub: OnStartup/OnShutdown/OnCommandStart/OnCommandEnd
      abi.go                  PluginMeta、Is() 匹配器、CommandDef
  server/agent.go             从 Config 构建 openagent.Agent、REST + ACP 服务
  plugin/pdk/rust/                   Rust SDK crate (openagent-pdk)
  examples/plugin/
    extended-settings.rs      cli:settings  — 读 keyring、合并 provider+env
    stats-cmd.rs              cli:commands  — 添加 "stats" 命令
    telemetry.rs              cli:observers — 记录生命周期事件
  build/
    openagent-cli             编译后的二进制
    plugins/                  编译后的 .wasm 文件
```

---

## 关键设计决策

1. **插件接收 settings，返回 settings。** 没有单独的注册 API。插件是一个纯函数 `settings → settings`。简单、可测试、可组合。

2. **一个 WASM 运行时，每个 .wasm 实例化一次，多种能力类型。** 一个 .wasm 可以导出 `init()`、`commands()`、`on_startup()` — host 根据 metadata type 路由。

3. **密钥在 keyring 中，不在 settings.json 中。** `api_key` 在 settings 中是可选的。插件在启动时从系统 keyring 读取凭证。

4. **cobra `RunE` 包装以传递观察者错误。** `PersistentPreRunE` 不传递错误。我们递归包装每个 `RunE`，以捕获原始错误传给 `on_command_end`。

5. **全程使用 `filepath.Join` 拼接路径。** 可移植，不依赖字符串拼接。

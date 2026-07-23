use alloc::string::String;
// openagent-pdk types — host API return values and plugin metadata.

use serde::{Deserialize, Serialize};

// ── Host API return types ──

/// Returned by host::keyring_get.
#[derive(Deserialize, Serialize, Default)]
pub struct KeyringResult {
    #[serde(default)]
    pub value: String,
    #[serde(default)]
    pub error: String,
}

/// Returned by host::keyring_set, keyring_delete, fs_write.
#[derive(Deserialize, Serialize, Default)]
pub struct HostResult {
    #[serde(default)]
    pub error: String,
}

/// Returned by host::http_request.
#[derive(Deserialize, Serialize, Default)]
pub struct HttpResponse {
    #[serde(default)]
    pub status: u32,
    #[serde(default)]
    pub body: String,
    #[serde(default)]
    pub error: String,
}

/// Returned by host::fs_read.
#[derive(Deserialize, Serialize, Default)]
pub struct FsReadResult {
    #[serde(default)]
    pub data: String, // base64
    #[serde(default)]
    pub error: String,
}

/// A single entry returned by host::fs_readdir.
#[derive(Deserialize, Serialize)]
pub struct DirEntry {
    #[serde(default)]
    pub name: String,
    #[serde(default, alias = "is_dir")]
    pub is_dir: bool,
}

/// Returned by host::fs_readdir.
#[derive(Deserialize, Serialize, Default)]
pub struct FsReaddirResult {
    #[serde(default)]
    pub entries: alloc::vec::Vec<DirEntry>,
    #[serde(default)]
    pub error: String,
}

// ── Plugin metadata ──

/// Returned by a plugin's `metadata` export as JSON.
/// Matches agent/wasm/abi.go PluginMeta.
#[derive(Serialize, Default)]
pub struct PluginMeta {
    /// Plugin type: "cli:settings", "agent:tools", "agent:observers", etc.
    /// Multiple types separated by comma: "cli:settings,cli:agent".
    #[serde(rename = "type", default)]
    pub plugin_type: String,
    #[serde(default)]
    pub name: String,
    #[serde(default)]
    pub description: String,
}

// ── Stage events (agent:observers plugins) ──

/// Input passed to observer plugin's run() export.
/// Matches agent/wasm/abi.go StageInput.
#[derive(Deserialize, Default)]
pub struct StageInput {
    #[serde(default)]
    pub name: String,
    #[serde(default)]
    pub phase: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub detail: Option<serde_json::Value>,
    #[serde(default)]
    pub error: String,
}

/// Output returned by observer plugin's run() export.
/// Matches agent/wasm/abi.go StageOutput.
#[derive(Serialize)]
pub struct StageOutput {
    pub action: String, // "continue" or "abort"
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub reason: String,
}

// ── Command definitions (cli:commands plugins) ──

/// A single command registered by a cli:commands plugin.
/// Matches cli/wasm/abi.go CommandDef.
#[derive(Serialize, Default)]
pub struct CommandDef {
    #[serde(default)]
    pub name: &'static str,
    #[serde(default)]
    pub r#use: &'static str,
    #[serde(default)]
    pub short: &'static str,
    #[serde(default, skip_serializing_if = "str::is_empty")]
    pub long: &'static str,
}

// ── Tool input/output (agent:tools plugins) ──

/// Input passed to tool plugin's execute() export.
#[derive(Deserialize, Default)]
pub struct ToolInput {
    #[serde(default)]
    pub args: serde_json::Value,
}

/// Output returned by tool plugin's execute() export.
#[derive(Serialize)]
pub struct ToolOutput {
    #[serde(default)]
    pub result: String,
    #[serde(default)]
    pub error: String,
}

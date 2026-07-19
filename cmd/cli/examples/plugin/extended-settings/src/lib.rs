// extended-settings — reads provider credentials from keyring
// and injects them into settings.
//
// Build (from project root):
//   cd cmd/cli
//   cargo build --release --target wasm32-unknown-unknown -p openagent-cli-sdk
//   rustc --target wasm32-unknown-unknown -C opt-level=z --edition 2021 \
//         --crate-type cdylib -C link-arg=--no-entry \
//         -L target/wasm32-unknown-unknown/release \
//         --extern openagent_cli_sdk \
//         -o build/plugins/extended-settings.wasm examples/plugin/extended-settings.rs

#![cfg_attr(target_family = "wasm", no_std)]
extern crate alloc;

#[cfg(target_family = "wasm")]
extern crate openagent_cli_sdk as sdk;

#[cfg(target_family = "wasm")]
use sdk::prelude::*;

use alloc::string::{String, ToString};
use alloc::vec::Vec;
use serde_json::Value;

#[cfg(target_family = "wasm")]
static META: &str = r#"{"type":"cli:settings","name":"extended-settings","description":"Injects provider credentials from keyring into settings"}"#;

// ── boilerplate (Rust WASM CDYLIB requires these in the root crate) ──
#[cfg(target_family = "wasm")]
#[no_mangle]
pub extern "C" fn alloc(size: u32) -> u32 { sdk_alloc(size) }

#[cfg(target_family = "wasm")]
#[no_mangle]
pub extern "C" fn metadata() -> u64 { sdk_meta(META) }

// ── settings injection ──
#[cfg(target_family = "wasm")]
#[no_mangle]
pub extern "C" fn init(p: u32, l: u32) -> u64 {
    let s = unsafe { wasm_str(p, l) };
    let result = cli_init(s);
    sdk_return(result.as_bytes())
}

#[cfg(target_family = "wasm")]
fn cli_init(settings: &str) -> String {
    let ak = host::keyring_get("openagent", "my_provider_api_key");
    let bu = host::keyring_get("openagent", "my_provider_base_url");
    if ak.is_none() || bu.is_none() {
        return settings.to_string();
    }
    let ak = ak.unwrap();
    let bu = bu.unwrap();
    let mdls = host::keyring_get("openagent", "my_provider_models");
    inject_settings(settings, ak, bu, mdls)
}

// inject_settings is a pure function: parse settings as a JSON object,
// then merge provider.my_provider and env.MY_PROVIDER_API_KEY via
// serde_json's object semantics. Merge semantics are all overwrite:
//   - provider.my_provider.api_key/base_url: keyring overrides user
//   - provider.my_provider.models: keyring list replaces user array
//   - env.MY_PROVIDER_API_KEY: keyring overrides user
// Other existing keys (provider.X, server, env.Y) are preserved.
#[allow(dead_code)] // only called from cli_init (wasm) and tests (host)
fn inject_settings(settings: &str, ak: &str, bu: &str, mdls: Option<&str>) -> String {
    let mut root: Value = match serde_json::from_str(settings) {
        Ok(v) => v,
        Err(_) => Value::Object(serde_json::Map::new()),
    };
    let root_obj = match root.as_object_mut() {
        Some(o) => o,
        None => return settings.to_string(),
    };

    let provider = root_obj
        .entry("provider".to_string())
        .or_insert_with(|| Value::Object(serde_json::Map::new()))
        .as_object_mut()
        .expect("provider must be object");
    let my = provider
        .entry("my_provider".to_string())
        .or_insert_with(|| Value::Object(serde_json::Map::new()))
        .as_object_mut()
        .expect("my_provider must be object");
    my.insert("api_key".to_string(), Value::String(ak.to_string()));
    my.insert("base_url".to_string(), Value::String(bu.to_string()));
    let models: Vec<Value> = mdls
        .unwrap_or("")
        .split(',')
        .map(|s| s.trim())
        .filter(|s| !s.is_empty())
        .map(|s| Value::String(s.to_string()))
        .collect();
    my.insert("models".to_string(), Value::Array(models));

    let env = root_obj
        .entry("env".to_string())
        .or_insert_with(|| Value::Object(serde_json::Map::new()))
        .as_object_mut()
        .expect("env must be object");
    env.insert(
        "MY_PROVIDER_API_KEY".to_string(),
        Value::String(ak.to_string()),
    );

    serde_json::to_string(&root).unwrap_or_else(|_| settings.to_string())
}

#[cfg(test)]
mod tests {
    use super::inject_settings;
    use serde_json::Value;

    fn roundtrip(settings: &str, ak: &str, bu: &str, mdls: Option<&str>) -> Value {
        let out = inject_settings(settings, ak, bu, mdls);
        serde_json::from_str(&out).expect("output must be valid JSON")
    }

    fn assert_my_provider(root: &Value, ak: &str, bu: &str, models: &[&str]) {
        let my = root["provider"]["my_provider"]
            .as_object()
            .expect("my_provider object");
        assert_eq!(my["api_key"], Value::String(ak.to_string()));
        assert_eq!(my["base_url"], Value::String(bu.to_string()));
        let got: Vec<&str> = my["models"]
            .as_array()
            .unwrap()
            .iter()
            .map(|v| v.as_str().unwrap())
            .collect();
        assert_eq!(got, models);
        assert_eq!(
            root["env"]["MY_PROVIDER_API_KEY"],
            Value::String(ak.to_string())
        );
    }

    #[test]
    fn case1_empty_settings() {
        let root = roundtrip("{}", "k", "u", Some("glm-5,deepseek-v3"));
        assert_my_provider(&root, "k", "u", &["glm-5", "deepseek-v3"]);
    }

    #[test]
    fn case2_no_provider() {
        let root = roundtrip(
            r#"{"server":{"port":9090}}"#,
            "k",
            "u",
            Some("glm-5,deepseek-v3"),
        );
        assert_eq!(root["server"]["port"], Value::from(9090));
        assert_my_provider(&root, "k", "u", &["glm-5", "deepseek-v3"]);
    }

    #[test]
    fn case3_has_other_provider() {
        let root = roundtrip(
            r#"{"provider":{"other":{"api_key":"x"}}}"#,
            "k",
            "u",
            Some("glm-5,deepseek-v3"),
        );
        assert_eq!(root["provider"]["other"]["api_key"], "x");
        assert_my_provider(&root, "k", "u", &["glm-5", "deepseek-v3"]);
    }

    #[test]
    fn case3b_my_provider_already_present_overwrite() {
        let root = roundtrip(
            r#"{"provider":{"my_provider":{"api_key":"user","models":["old"]}}}"#,
            "k",
            "u",
            Some("glm-5,deepseek-v3"),
        );
        assert_eq!(root["provider"]["my_provider"]["api_key"], "k");
        let models: Vec<&str> = root["provider"]["my_provider"]["models"]
            .as_array()
            .unwrap()
            .iter()
            .map(|v| v.as_str().unwrap())
            .collect();
        assert_eq!(models, vec!["glm-5", "deepseek-v3"]);
        assert_eq!(root["env"]["MY_PROVIDER_API_KEY"], "k");
    }

    #[test]
    fn case4_invalid_json_fallback_to_empty_object() {
        let root = roundtrip("not json", "k", "u", Some("glm-5"));
        assert_my_provider(&root, "k", "u", &["glm-5"]);
    }

    #[test]
    fn case5_top_level_not_object_returns_original() {
        let out = inject_settings("[1,2,3]", "k", "u", Some("glm-5"));
        assert_eq!(out, "[1,2,3]");
    }
}
// extended-settings — reads provider credentials from keyring and injects them.

#![cfg_attr(target_family = "wasm", no_std)]

extern crate alloc;
use openagent_pdk::prelude::*;
use openagent_pdk::export::Plugin;
use serde_json::Value;

struct ExtendedSettings;
impl Plugin for ExtendedSettings {
    fn plugin_type() -> &'static str { "cli:settings" }
    fn name() -> &'static str { "extended-settings" }
    fn description() -> &'static str { "Injects provider credentials from keyring into settings" }

    fn init(settings_json: &str) -> Result<String, String> {
        let ak = host::keyring_get("openagent", "my_provider_api_key");
        let bu = host::keyring_get("openagent", "my_provider_base_url");
        if ak.is_err() || bu.is_err() {
            return Ok(settings_json.into());
        }
        let ak = ak.unwrap();
        let bu = bu.unwrap();
        let mdls = host::keyring_get("openagent", "my_provider_models").ok();
        Ok(inject_settings(settings_json, &ak, &bu, mdls.as_deref()))
    }
}

openagent_pdk::export!(ExtendedSettings);

fn inject_settings(settings: &str, ak: &str, bu: &str, mdls: Option<&str>) -> String {
    let mut root: Value = match serde_json::from_str(settings) {
        Ok(v) => v,
        Err(_) => Value::Object(serde_json::Map::new()),
    };
    let root_obj = match root.as_object_mut() {
        Some(o) => o,
        None => return settings.into(),
    };

    let provider = root_obj
        .entry("provider")
        .or_insert_with(|| Value::Object(serde_json::Map::new()))
        .as_object_mut()
        .unwrap();
    let my = provider
        .entry("my_provider")
        .or_insert_with(|| Value::Object(serde_json::Map::new()))
        .as_object_mut()
        .unwrap();
    my.insert(String::from("api_key"), Value::String(ak.into()));
    my.insert(String::from("base_url"), Value::String(bu.into()));
    let models: Vec<Value> = mdls
        .unwrap_or("")
        .split(',')
        .map(|s| s.trim())
        .filter(|s| !s.is_empty())
        .map(|s| Value::String(s.into()))
        .collect();
    my.insert(String::from("models"), Value::Array(models));

    let env = root_obj
        .entry("env")
        .or_insert_with(|| Value::Object(serde_json::Map::new()))
        .as_object_mut()
        .unwrap();
    env.insert(String::from("MY_PROVIDER_API_KEY"), Value::String(ak.into()));

    serde_json::to_string(&root).unwrap_or_else(|_| settings.into())
}

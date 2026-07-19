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

#![no_std]
extern crate openagent_cli_sdk as sdk;
use sdk::prelude::*;

static META: &str = r#"{"type":"cli:settings","name":"extended-settings","description":"Injects provider credentials from keyring into settings"}"#;

// ── boilerplate (Rust WASM CDYLIB requires these in the root crate) ──
#[no_mangle] pub extern "C" fn alloc(size: u32) -> u32 { sdk_alloc(size) }
#[no_mangle] pub extern "C" fn metadata() -> u64 { sdk_meta(META) }

// ── settings injection ──
#[no_mangle]
pub extern "C" fn init(p: u32, l: u32) -> u64 {
    let s = unsafe { wasm_str(p, l) };
    let result = cli_init(s);
    sdk_return(result.as_bytes())
}

fn cli_init(settings: &str) -> String {
    let ak = host::keyring_get("openagent", "my_provider_api_key");
    let bu = host::keyring_get("openagent", "my_provider_base_url");
    if ak.is_none() || bu.is_none() {
        return String::from(settings);
    }
    let ak = ak.unwrap();
    let bu = bu.unwrap();
    let mdls = host::keyring_get("openagent", "my_provider_models");

    let trimmed = settings.trim_end();
    let end = if trimmed.ends_with('}') { trimmed.len() - 1 } else { trimmed.len() };

    let mut out = String::from(&settings[..end]);
    out.push_str(",\"provider\":{\"my_provider\":{\"api_key\":\"");
    out.push_str(ak);
    out.push_str("\",\"base_url\":\"");
    out.push_str(bu);
    out.push_str("\",\"models\":[");

    if let Some(mdls) = mdls {
        let mut first = true;
        for m in mdls.split(',') {
            let m = m.trim();
            if m.is_empty() { continue }
            if !first { out.push(',') } else { first = false }
            out.push('\"');
            out.push_str(m);
            out.push('\"');
        }
    }
    out.push_str("]}}");

    out.push_str(",\"env\":{\"MY_PROVIDER_API_KEY\":\"");
    out.push_str(ak);
    out.push_str("\"}}");

    out
}

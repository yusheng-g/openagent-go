// stats-cmd — adds `openagent-cli stats` command.
//
// Build:
//   rustc --target wasm32-unknown-unknown -C opt-level=z --edition 2021 \
//         --crate-type cdylib -C link-arg=--no-entry \
//         -L ../../../target/wasm32-unknown-unknown/release \
//         --extern openagent_cli_sdk \
//         -o stats-cmd.wasm stats-cmd.rs

#![no_std]
extern crate openagent_cli_sdk as sdk;
use sdk::prelude::*;

static META: &str = r#"{"type":"cli:commands","name":"stats-cmd","description":"Provides the stats command"}"#;
static COMMANDS: &str = r#"[{"name":"stats","use":"stats","short":"Show plugin stats and keyring status"}]"#;

#[no_mangle] pub extern "C" fn alloc(size: u32) -> u32 { sdk_alloc(size) }
#[no_mangle] pub extern "C" fn metadata() -> u64 { sdk_meta(META) }
#[no_mangle] pub extern "C" fn commands() -> u64 { sdk_return(COMMANDS.as_bytes()) }

#[no_mangle]
pub extern "C" fn run_stats(p: u32, l: u32) -> u64 {
    let _args = unsafe { wasm_str(p, l) };
    let has_ak = host::keyring_get("openagent", "my_provider_api_key").is_some();
    let has_bu = host::keyring_get("openagent", "my_provider_base_url").is_some();

    let a = if has_ak { "found" } else { "not found" };
    let b = if has_bu { "found" } else { "not found" };

    let mut s = String::from("plugin stats:\n  keyring my_provider_api_key: ");
    s.push_str(a);
    s.push_str("\n  keyring my_provider_base_url: ");
    s.push_str(b);
    s.push('\n');

    sdk_return(s.as_bytes())
}

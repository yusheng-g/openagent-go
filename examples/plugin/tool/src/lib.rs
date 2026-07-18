// Echo tool plugin — agent:tools example using the plugin SDK.
//
// Build:
//   cargo +stable build --release --target wasm32-unknown-unknown
//   cp target/wasm32-unknown-unknown/release/*.wasm ../plugins/echo.wasm

#![no_std]
#![no_main]

extern crate alloc;
extern crate openagent_cli_sdk as sdk;

use sdk::prelude::*;

// SDK provides #[global_allocator] and #[panic_handler].

#[no_mangle]
pub extern "C" fn alloc(size: u32) -> u32 {
    sdk_alloc(size + 8)
}

#[no_mangle]
pub extern "C" fn metadata() -> u64 {
    host::log_info("echo plugin: metadata() called");
    sdk_meta(r#"{
        "type":"agent:tools",
        "name":"echo",
        "description":"Echoes back the input message. Uses host API for logging.",
        "parameters":{
            "type":"object",
            "properties":{
                "message":{"type":"string","description":"The message to echo"}
            },
            "required":["message"]
        }
    }"#)
}

#[no_mangle]
pub extern "C" fn execute(ptr: u32, len: u32) -> u64 {
    host::log_info("echo plugin: executing...");

    let input = unsafe { wasm_str(ptr, len) };

    if let Some(_secret) = host::keyring_get("openagent", "echo-test") {
        host::log_info("echo plugin: keyring entry found");
    }

    let msg = find_message(input).unwrap_or("(empty)");
    let mut out = String::from("{\"result\":\"you said: ");
    out.push_str(msg);
    out.push_str("\"}");
    sdk_return(out.as_bytes())
}

fn find_message(input: &str) -> Option<&str> {
    let start = input.find("\"message\":\"")? + 11;
    let end = input[start..].find('"')?;
    Some(&input[start..start + end])
}

// telemetry — openagent-cli observer plugin. Logs lifecycle events via host log functions.
//
// Build (from cmd/cli):
//   cargo build --release --target wasm32-unknown-unknown -p openagent-cli-sdk
//   rustc --target wasm32-unknown-unknown -C opt-level=z --edition 2021 \
//         --crate-type cdylib -C link-arg=--no-entry \
//         -L target/wasm32-unknown-unknown/release \
//         --extern openagent_cli_sdk \
//         -o build/plugins/telemetry.wasm examples/plugin/telemetry.rs

#![no_std]
extern crate openagent_cli_sdk as sdk;
use sdk::prelude::*;

static META: &str = r#"{"type":"cli:observers","name":"telemetry","description":"Logs lifecycle events"}"#;

#[no_mangle] pub extern "C" fn alloc(size: u32) -> u32 { sdk_alloc(size) }
#[no_mangle] pub extern "C" fn metadata() -> u64 { sdk_meta(META) }

#[no_mangle]
pub extern "C" fn on_startup() {
    host::log_info("telemetry: startup");
}

#[no_mangle]
pub extern "C" fn on_shutdown() {
    host::log_info("telemetry: shutdown");
}

#[no_mangle]
pub extern "C" fn on_command_start(p: u32, l: u32) {
    let cmd = unsafe { wasm_str(p, l) };

    // Use stack buffer to build the log message (avoids bump alloc String issues).
    let mut buf = [0u8; 256];
    let pre = b"telemetry: command start -- ";
    let mut pos = 0usize;
    let cp = |buf: &mut [u8], pos: &mut usize, src: &[u8]| {
        let end = *pos + src.len();
        buf[*pos..end].copy_from_slice(src);
        *pos = end;
    };
    cp(&mut buf, &mut pos, pre);
    cp(&mut buf, &mut pos, cmd.as_bytes());

    let msg = unsafe { core::str::from_utf8_unchecked(&buf[..pos]) };
    host::log_info(msg);
}

#[no_mangle]
pub extern "C" fn on_command_end(p: u32, l: u32) {
    let payload = unsafe { wasm_str(p, l) };

    let mut buf = [0u8; 256];
    let pre = b"telemetry: command end -- ";
    let mut pos = 0usize;
    buf[..pre.len()].copy_from_slice(pre);
    pos += pre.len();
    buf[pos..pos + payload.len()].copy_from_slice(payload.as_bytes());
    pos += payload.len();

    let msg = unsafe { core::str::from_utf8_unchecked(&buf[..pos]) };
    host::log_info(msg);
}

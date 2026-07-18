#![no_std]
#![no_main]

extern crate alloc;
extern crate openagent_cli_sdk as sdk;

use sdk::prelude::*;

#[no_mangle]
pub extern "C" fn alloc(size: u32) -> u32 {
    sdk_alloc(size + 8)
}

#[no_mangle]
pub extern "C" fn metadata() -> u64 {
    sdk_meta(r#"{
        "type":"agent:observers",
        "name":"observer_logger",
        "stage":"*",
        "phase":"*"
    }"#)
}

#[no_mangle]
pub extern "C" fn run(_ptr: u32, _len: u32) -> u64 {
    host::log_info("observer run called");
    sdk_return(b"{\"action\":\"continue\"}")
}

// openagent-cli-sdk — crate for building WASM plugins for openagent-cli.
//
// Add to Cargo.toml:
//   [dependencies]
//   openagent-cli-sdk = { path = "../../cmd/cli/sdk/rust" }
//
// Get started:
//   #![no_std]
//   extern crate openagent_cli_sdk as sdk;
//   use sdk::prelude::*;

#![no_std]

extern crate alloc;

pub mod host;

// ── allocator ──

use core::alloc::{GlobalAlloc, Layout};

struct BumpAlloc;
unsafe impl GlobalAlloc for BumpAlloc {
    unsafe fn alloc(&self, layout: core::alloc::Layout) -> *mut u8 {
        let size = layout.size();
        let align = layout.align();
        let ptr = HEAP.as_ptr() as *mut u8;
        let offset = OFF;
        let aligned = (offset + align - 1) & !(align - 1);
        if aligned + size <= HEAP.len() {
            OFF = aligned + size;
            ptr.add(aligned)
        } else {
            core::ptr::null_mut()
        }
    }
    unsafe fn dealloc(&self, _ptr: *mut u8, _layout: core::alloc::Layout) {}
}

#[global_allocator]
static ALLOC: BumpAlloc = BumpAlloc;

const HEAP: [u8; 131072] = [0; 131072]; // 128 KB
static mut OFF: usize = 0;

// ── panic ──

use core::panic::PanicInfo;
#[panic_handler]
fn _panic(_: &PanicInfo) -> ! { loop {} }

// ── helpers ──

pub fn pk(p: u32, l: u32) -> u64 { ((p as u64) << 32) | (l as u64) }
pub fn up(u: u64) -> (u32, u32) { ((u >> 32) as u32, (u & 0xFFFF_FFFF) as u32) }

pub unsafe fn wasm_str(p: u32, l: u32) -> &'static str {
    if p == 0 && l == 0 { return "" }
    core::str::from_utf8_unchecked(core::slice::from_raw_parts(p as *const u8, l as usize))
}

pub fn sdk_meta(json: &str) -> u64 { pk(json.as_ptr() as u32, json.len() as u32) }

pub fn sdk_alloc(size: u32) -> u32 {
    let layout = Layout::array::<u8>(size as usize).unwrap();
    unsafe { GlobalAlloc::alloc(&ALLOC, layout) as u32 }
}

pub fn sdk_return(data: &[u8]) -> u64 {
    if data.is_empty() { return 0 }
    let layout = Layout::array::<u8>(data.len()).unwrap();
    let p = unsafe { GlobalAlloc::alloc(&ALLOC, layout) as u32 };
    unsafe { core::slice::from_raw_parts_mut(p as *mut u8, data.len()).copy_from_slice(data) }
    pk(p, data.len() as u32)
}

pub mod prelude {
    pub use alloc::string::String;
    pub use crate::host;
    pub use crate::{pk, up, sdk_alloc, sdk_meta, sdk_return, wasm_str};
}

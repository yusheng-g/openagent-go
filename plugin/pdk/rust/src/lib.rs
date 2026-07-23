// openagent-pdk — Rust PDK for building WASM plugins for openagent-go.
//
// Add to Cargo.toml:
//   [dependencies]
//   openagent-pdk = { git = "https://github.com/yusheng-g/openagent-go", path = "plugin/pdk/rust" }
//
// Quick start:
//   use openagent_pdk::prelude::*;

#![no_std]

extern crate alloc;

pub mod host;
pub mod types;
pub mod export;

// ── allocator ──

use core::alloc::{GlobalAlloc, Layout};
use core::ptr::addr_of_mut;

pub struct BumpAlloc;

static mut HEAP: [u8; 131072] = [0; 131072]; // 128 KB
static mut OFF: usize = 0;

unsafe impl GlobalAlloc for BumpAlloc {
    unsafe fn alloc(&self, layout: Layout) -> *mut u8 {
        let size = layout.size();
        let align = layout.align();
        let heap_ptr = addr_of_mut!(HEAP) as *mut u8;
        let aligned = (OFF + align - 1) & !(align - 1);
        if aligned + size <= 131072 {
            OFF = aligned + size;
            heap_ptr.add(aligned)
        } else {
            core::ptr::null_mut()
        }
    }
    unsafe fn dealloc(&self, _ptr: *mut u8, _layout: Layout) {}
}

#[global_allocator]
pub static ALLOC: BumpAlloc = BumpAlloc;

// ── panic handler ──

use core::panic::PanicInfo;
#[panic_handler]
fn panic_handler(_: &PanicInfo) -> ! { loop {} }

// ── ABI helpers ──

/// Pack (ptr, len) into a single u64 — high 32 = ptr, low 32 = len.
pub fn pk(p: u32, l: u32) -> u64 { ((p as u64) << 32) | (l as u64) }

/// Unpack a u64 into (ptr, len).
pub fn up(u: u64) -> (u32, u32) { ((u >> 32) as u32, (u & 0xFFFF_FFFF) as u32) }

/// Read a string from WASM linear memory at (ptr, len).
pub unsafe fn wasm_str(p: u32, l: u32) -> &'static str {
    if p == 0 && l == 0 { return "" }
    core::str::from_utf8_unchecked(core::slice::from_raw_parts(p as *const u8, l as usize))
}

/// Allocate memory in the WASM linear heap. Export this from your .wasm.
pub fn sdk_alloc(size: u32) -> u32 {
    let layout = Layout::array::<u8>(size as usize).unwrap();
    unsafe { GlobalAlloc::alloc(&ALLOC, layout) as u32 }
}

/// Pack static JSON for no-arg exports (e.g. metadata).
pub fn sdk_meta(json: &str) -> u64 { pk(json.as_ptr() as u32, json.len() as u32) }

/// Allocate + copy data into guest memory, return packed (ptr, len).
pub fn sdk_return(data: &[u8]) -> u64 {
    if data.is_empty() { return 0 }
    let p = sdk_alloc(data.len() as u32);
    unsafe { core::slice::from_raw_parts_mut(p as *mut u8, data.len()).copy_from_slice(data) }
    pk(p, data.len() as u32)
}

/// sdk_return on a serializable value.
pub fn sdk_return_json(v: &impl serde::Serialize) -> u64 {
    match serde_json::to_string(v) {
        Ok(s) => sdk_return(s.as_bytes()),
        Err(_) => 0,
    }
}

/// Read bytes from guest memory at packed (ptr, len).
pub fn read_input(packed: u64) -> &'static [u8] {
    let (p, l) = up(packed);
    if p == 0 && l == 0 { return &[] }
    unsafe { core::slice::from_raw_parts(p as *const u8, l as usize) }
}

/// Deserialize guest memory at packed (ptr, len) into T.
/// Returns default value on parse failure.
pub fn read_input_json<T: serde::de::DeserializeOwned + Default + 'static>(packed: u64) -> T {
    serde_json::from_slice(read_input(packed)).unwrap_or_default()
}

/// Read a string from guest memory at (ptr, len). Safe wrapper around wasm_str.
pub fn read_input_str(ptr: u32, len: u32) -> &'static str {
    unsafe { wasm_str(ptr, len) }
}

/// Dispatch a CLI command by name. Used by cli:commands plugins in their
/// hand-written `run_<name>` exports (one line per command).
pub fn dispatch_command<T: export::Plugin>(ptr: u32, len: u32, name: &str) -> u64 {
    let args = read_input_str(ptr, len);
    match T::run_command(name, args) {
        Ok(s) => sdk_return(s.as_bytes()),
        Err(e) => sdk_return(e.as_bytes()),
    }
}

// ── prelude ──

pub mod prelude {
    pub use alloc::string::String;
    pub use alloc::vec::Vec;
    pub use alloc::format;
    pub use serde_json;
    pub use crate::host;
    pub use crate::types::*;
    pub use crate::{sdk_alloc, sdk_meta, sdk_return, sdk_return_json, pk, up, wasm_str};
}

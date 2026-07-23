use alloc::string::String;
// Host API — WASM imports from the "host" module.
//
// All host functions that can fail return packed JSON with an "error" field.
// Each wrapper here calls the raw FFI, reads the returned bytes via up+wasm_str,
// deserializes via serde_json, and checks the error field.

use crate::up;
use crate::types::*;

mod ffi {
    #[link(wasm_import_module = "host")]
    extern "C" {
        pub fn keyring_get(svc_p: u32, svc_l: u32, key_p: u32, key_l: u32) -> u64;
        pub fn keyring_set(svc_p: u32, svc_l: u32, key_p: u32, key_l: u32, val_p: u32, val_l: u32) -> u64;
        pub fn keyring_delete(svc_p: u32, svc_l: u32, key_p: u32, key_l: u32) -> u64;
        pub fn http_request(
            method_p: u32, method_l: u32, url_p: u32, url_l: u32,
            headers_p: u32, headers_l: u32, body_p: u32, body_l: u32,
        ) -> u64;
        pub fn fs_read(path_p: u32, path_l: u32) -> u64;
        pub fn fs_write(path_p: u32, path_l: u32, data_p: u32, data_l: u32) -> u64;
        pub fn fs_readdir(path_p: u32, path_l: u32) -> u64;
        pub fn log_info(msg_p: u32, msg_l: u32);
        pub fn log_warn(msg_p: u32, msg_l: u32);
        pub fn log_error(msg_p: u32, msg_l: u32);
        pub fn utc_now() -> u64;
    }
}

// ── keyring ──

pub fn keyring_get(service: &str, key: &str) -> Result<String, String> {
    let packed = unsafe { ffi::keyring_get(
        service.as_ptr() as u32, service.len() as u32,
        key.as_ptr() as u32, key.len() as u32,
    )};
    let r: KeyringResult = serde_json::from_slice(wasm_str_packed(packed)).unwrap_or_default();
    if !r.error.is_empty() { Err(r.error) } else { Ok(r.value) }
}

pub fn keyring_set(service: &str, key: &str, val: &str) -> Result<(), String> {
    let packed = unsafe { ffi::keyring_set(
        service.as_ptr() as u32, service.len() as u32,
        key.as_ptr() as u32, key.len() as u32,
        val.as_ptr() as u32, val.len() as u32,
    )};
    let r: HostResult = serde_json::from_slice(wasm_str_packed(packed)).unwrap_or_default();
    if !r.error.is_empty() { Err(r.error) } else { Ok(()) }
}

pub fn keyring_delete(service: &str, key: &str) -> Result<(), String> {
    let packed = unsafe { ffi::keyring_delete(
        service.as_ptr() as u32, service.len() as u32,
        key.as_ptr() as u32, key.len() as u32,
    )};
    let r: HostResult = serde_json::from_slice(wasm_str_packed(packed)).unwrap_or_default();
    if !r.error.is_empty() { Err(r.error) } else { Ok(()) }
}

// ── HTTP ──

pub fn http_request(method: &str, url: &str, headers: &str, body: &[u8]) -> Result<(u32, String), String> {
    let packed = unsafe { ffi::http_request(
        method.as_ptr() as u32, method.len() as u32,
        url.as_ptr() as u32, url.len() as u32,
        headers.as_ptr() as u32, headers.len() as u32,
        if body.is_empty() { 0 } else { body.as_ptr() as u32 }, body.len() as u32,
    )};
    let r: HttpResponse = serde_json::from_slice(wasm_str_packed(packed)).unwrap_or_default();
    if !r.error.is_empty() { Err(r.error) } else { Ok((r.status, r.body)) }
}

// ── Filesystem ──

pub fn fs_read(path: &str) -> Result<alloc::vec::Vec<u8>, String> {
    let packed = unsafe { ffi::fs_read(path.as_ptr() as u32, path.len() as u32) };
    let r: FsReadResult = serde_json::from_slice(wasm_str_packed(packed)).unwrap_or_default();
    if !r.error.is_empty() { return Err(r.error) }
    base64::Engine::decode(&base64::engine::general_purpose::STANDARD, &r.data)
        .map_err(|e| alloc::format!("base64 decode: {}", e))
}

pub fn fs_read_str(path: &str) -> Result<String, String> {
    let bytes = fs_read(path)?;
    String::from_utf8(bytes).map_err(|e| alloc::format!("invalid UTF-8: {}", e))
}

pub fn fs_write(path: &str, data: &[u8]) -> Result<(), String> {
    let packed = unsafe { ffi::fs_write(
        path.as_ptr() as u32, path.len() as u32,
        if data.is_empty() { 0 } else { data.as_ptr() as u32 }, data.len() as u32,
    )};
    let r: HostResult = serde_json::from_slice(wasm_str_packed(packed)).unwrap_or_default();
    if !r.error.is_empty() { Err(r.error) } else { Ok(()) }
}

pub fn fs_write_str(path: &str, content: &str) -> Result<(), String> {
    fs_write(path, content.as_bytes())
}

pub fn fs_readdir(path: &str) -> Result<alloc::vec::Vec<DirEntry>, String> {
    let packed = unsafe { ffi::fs_readdir(path.as_ptr() as u32, path.len() as u32) };
    let r: FsReaddirResult = serde_json::from_slice(wasm_str_packed(packed)).unwrap_or_default();
    if !r.error.is_empty() { Err(r.error) } else { Ok(r.entries) }
}

// ── Logging ──

pub fn log_info(msg: &str) {
    unsafe { ffi::log_info(msg.as_ptr() as u32, msg.len() as u32) }
}
pub fn log_warn(msg: &str) {
    unsafe { ffi::log_warn(msg.as_ptr() as u32, msg.len() as u32) }
}
pub fn log_error(msg: &str) {
    unsafe { ffi::log_error(msg.as_ptr() as u32, msg.len() as u32) }
}

// ── Time ──

pub fn utc_now() -> u64 {
    unsafe { ffi::utc_now() }
}

// ── Internal ──

/// Convert a packed (ptr, len) u64 into bytes, using the PK convention.
fn wasm_str_packed(packed: u64) -> &'static [u8] {
    let (p, l) = up(packed);
    if p == 0 && l == 0 { return &[] }
    unsafe { core::slice::from_raw_parts(p as *const u8, l as usize) }
}

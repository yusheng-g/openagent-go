// Host API — keyring + HTTP for WASM plugins.

use crate::{up, wasm_str};

mod ffi {
    #[link(wasm_import_module = "host")]
    extern "C" {
        pub fn keyring_get(sp: u32, sl: u32, kp: u32, kl: u32) -> u64;
        pub fn keyring_set(sp: u32, sl: u32, kp: u32, kl: u32, vp: u32, vl: u32);
        pub fn http_request(
            mp: u32, ml: u32, up: u32, ul: u32, hp: u32, hl: u32, bp: u32, bl: u32,
        ) -> u64;
        pub fn log_info(mp: u32, ml: u32);
        pub fn log_warn(mp: u32, ml: u32);
        pub fn log_error(mp: u32, ml: u32);
        pub fn get_env(kp: u32, kl: u32) -> u64;
        pub fn get_time_utc() -> u64;
    }
}

pub fn keyring_get(service: &str, key: &str) -> Option<&'static str> {
    let (p, l) = up(unsafe {
        ffi::keyring_get(
            service.as_ptr() as u32, service.len() as u32,
            key.as_ptr() as u32, key.len() as u32,
        )
    });
    if p == 0 && l == 0 { None } else { Some(unsafe { wasm_str(p, l) }) }
}

pub fn keyring_set(service: &str, key: &str, val: &str) {
    unsafe {
        ffi::keyring_set(
            service.as_ptr() as u32, service.len() as u32,
            key.as_ptr() as u32, key.len() as u32,
            val.as_ptr() as u32, val.len() as u32,
        )
    }
}

pub fn http_request(method: &str, url: &str, headers: &str, body: &[u8]) -> (u32, &'static [u8]) {
    let (rp, rl) = up(unsafe {
        ffi::http_request(
            method.as_ptr() as u32, method.len() as u32,
            url.as_ptr() as u32, url.len() as u32,
            headers.as_ptr() as u32, headers.len() as u32,
            if body.is_empty() { 0 } else { body.as_ptr() as u32 }, body.len() as u32,
        )
    });
    if rp == 0 && rl == 0 { return (0, &[]) }
    let raw = unsafe { core::slice::from_raw_parts(rp as *const u8, rl as usize) };
    let st = find_u32(raw, b"\"status\":");
    let bs = find_body(raw);
    (st, &raw[bs..])
}

fn find_u32(d: &[u8], k: &[u8]) -> u32 {
    let mut i = 0;
    while i + k.len() <= d.len() {
        if &d[i..i + k.len()] == k {
            i += k.len();
            let mut n: u32 = 0;
            while i < d.len() && d[i] >= b'0' && d[i] <= b'9' { n = n * 10 + (d[i] - b'0') as u32; i += 1 }
            return n;
        }
        i += 1;
    }
    0
}

pub fn log_info(msg: &str) { unsafe { ffi::log_info(msg.as_ptr() as u32, msg.len() as u32) } }
pub fn log_warn(msg: &str) { unsafe { ffi::log_warn(msg.as_ptr() as u32, msg.len() as u32) } }
pub fn log_error(msg: &str) { unsafe { ffi::log_error(msg.as_ptr() as u32, msg.len() as u32) } }

pub fn get_env(key: &str) -> Option<&'static str> {
    let (p, l) = up(unsafe { ffi::get_env(key.as_ptr() as u32, key.len() as u32) });
    if p == 0 && l == 0 { None } else { Some(unsafe { wasm_str(p, l) }) }
}

pub fn get_time_utc() -> Option<&'static str> {
    let (p, l) = up(unsafe { ffi::get_time_utc() });
    if p == 0 && l == 0 { None } else { Some(unsafe { wasm_str(p, l) }) }
}

fn find_body(d: &[u8]) -> usize {
    let pat = b"\"body\":\"";
    let mut i = 0;
    while i + pat.len() <= d.len() {
        if &d[i..i + pat.len()] == pat { return i + pat.len() }
        i += 1;
    }
    0
}

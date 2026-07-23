// telemetry — openagent-cli observer plugin using the high-level Plugin trait.

#![no_std]

extern crate alloc;
use openagent_pdk::prelude::*;
use openagent_pdk::export::Plugin;

struct Telemetry;
impl Plugin for Telemetry {
    fn plugin_type() -> &'static str { "cli:observers" }
    fn name() -> &'static str { "telemetry" }
    fn description() -> &'static str { "Logs lifecycle events" }

    fn on_startup() { host::log_info("telemetry: startup"); }
    fn on_shutdown() { host::log_info("telemetry: shutdown"); }
    fn on_command_start(cmd: &str) {
        let mut s = String::from("telemetry: command start -- ");
        s.push_str(cmd);
        host::log_info(&s);
    }
    fn on_command_end(payload: &str) {
        let mut s = String::from("telemetry: command end -- ");
        s.push_str(payload);
        host::log_info(&s);
    }
}

openagent_pdk::export!(Telemetry);

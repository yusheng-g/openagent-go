// Observer logger plugin — agent:observers using the high-level Plugin trait.

#![no_std]
#![no_main]

extern crate alloc;
use openagent_pdk::prelude::*;
use openagent_pdk::export::Plugin;

struct LoggerPlugin;
impl Plugin for LoggerPlugin {
    fn plugin_type() -> &'static str { "agent:observers" }
    fn name() -> &'static str { "observer_logger" }

    fn stage_filter() -> (&'static str, &'static str) { ("*", "*") }

    fn observe_stage(event: &StageInput) -> StageOutput {
        host::log_info(&alloc::format!(
            "observer: stage={} phase={} error={}",
            event.name, event.phase, event.error
        ));
        StageOutput { action: String::from("continue"), reason: String::new() }
    }
}

openagent_pdk::export!(LoggerPlugin);

// Echo tool plugin — agent:tools example using the high-level Plugin trait.

#![no_std]
#![no_main]

extern crate alloc;
use openagent_pdk::prelude::*;
use openagent_pdk::export::Plugin;

struct EchoPlugin;
impl Plugin for EchoPlugin {
    fn plugin_type() -> &'static str { "agent:tools" }
    fn name() -> &'static str { "echo" }
    fn description() -> &'static str {
        "Echoes back the input message. Uses host API for logging."
    }
    fn tool_parameters() -> Option<&'static str> {
        Some(r#"{"type":"object","properties":{"message":{"type":"string","description":"The message to echo"}},"required":["message"]}"#)
    }

    fn execute(args: &serde_json::Value) -> Result<String, String> {
        host::log_info("echo plugin: executing...");
        let msg = args.get("message")
            .and_then(|v| v.as_str())
            .unwrap_or("(empty)");
        Ok(alloc::format!("you said: {}", msg))
    }
}

openagent_pdk::export!(EchoPlugin);

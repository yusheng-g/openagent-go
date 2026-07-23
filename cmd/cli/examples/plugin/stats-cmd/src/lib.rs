// stats-cmd — adds `openagent-cli stats` command.

#![no_std]

extern crate alloc;
use alloc::vec;
use openagent_pdk::prelude::*;
use openagent_pdk::export::Plugin;

struct StatsCmd;
impl Plugin for StatsCmd {
    fn plugin_type() -> &'static str { "cli:commands" }
    fn name() -> &'static str { "stats-cmd" }
    fn description() -> &'static str { "Provides the stats command" }

    fn commands() -> Vec<CommandDef> {
        vec![CommandDef {
            name: "stats",
            r#use: "stats [--verbose]",
            short: "Show plugin stats and keyring status",
            flags: &[
                FlagDef {
                    name: "verbose", short: "v", kind: "bool",
                    default_value: "false", description: "Verbose output",
                },
            ],
            ..Default::default()
        }]
    }

    fn run_command(_name: &str, input: &CommandInput) -> Result<String, String> {
        let verbose = input.flags.get("verbose")
            .and_then(|v| v.as_bool()).unwrap_or(false);
        let has_ak = host::keyring_get("openagent", "my_provider_api_key").is_ok();
        let has_bu = host::keyring_get("openagent", "my_provider_base_url").is_ok();
        let mut s = String::from("plugin stats:\n  keyring my_provider_api_key: ");
        s.push_str(if has_ak { "found" } else { "not found" });
        s.push_str("\n  keyring my_provider_base_url: ");
        s.push_str(if has_bu { "found" } else { "not found" });
        if verbose {
            s.push_str("\n  plugin type: cli:commands\n  PDK version: 0.0.1");
        }
        s.push('\n');
        Ok(s)
    }
}

openagent_pdk::export!(StatsCmd);

#[no_mangle]
pub extern "C" fn run_stats(ptr: u32, len: u32) -> u64 {
    openagent_pdk::dispatch_command::<StatsCmd>(ptr, len, "stats")
}

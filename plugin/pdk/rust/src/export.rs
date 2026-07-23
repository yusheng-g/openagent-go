// Export trait + macro — the high-level API for plugin authors.
//
// Instead of hand-writing extern "C" wrappers, implement Plugin and call
// export!(YourType). The macro generates alloc, metadata, and every
// entry-point the plugin type declares.

use alloc::string::String;

use crate::types::*;

// ── Plugin trait ──

/// Implement this trait to create a plugin. Every method has a no-op default.
/// Override only the methods your plugin type needs.
pub trait Plugin: Sized {
    // ── Metadata (required for all plugins) ──

    /// Plugin type(s), comma-separated. E.g. "cli:settings", "agent:tools".
    fn plugin_type() -> &'static str { "agent:tools" }
    /// Human-readable plugin name.
    fn name() -> &'static str { "" }
    /// Short description of what this plugin does.
    fn description() -> &'static str { "" }

    // ── cli:settings ──

    /// Called at startup with the current settings JSON. Return the
    /// (possibly modified) settings. Default: pass-through.
    fn init(_settings_json: &str) -> Result<String, String> {
        Err("init not implemented".into())
    }

    // ── cli:commands ──

    /// Return the commands this plugin registers.
    fn commands() -> alloc::vec::Vec<CommandDef> { alloc::vec::Vec::new() }

    /// Handle execution of a registered command. args is a JSON array of
    /// positional arguments from the CLI.
    fn run_command(_name: &str, _args_json: &str) -> Result<String, String> {
        Err("unknown command".into())
    }

    // ── cli:observers ──

    fn on_startup() {}
    fn on_shutdown() {}

    /// cmd_path is the full cobra command path, e.g. "openagent-cli serve".
    fn on_command_start(_cmd_path: &str) {}

    /// cmd_path + optional " error=..." suffix.
    fn on_command_end(_payload: &str) {}

    // ── agent:tools ──

    /// JSON Schema for tool arguments. Only needed for agent:tools plugins.
    fn tool_parameters() -> Option<&'static str> { None }

    /// Execute the tool. args is the parsed JSON arguments from the model.
    fn execute(_args: &serde_json::Value) -> Result<String, String> {
        Err("execute not implemented".into())
    }

    // ── agent:observers ──

    /// Filter: (stage, phase). "*" means match all. Only needed for
    /// agent:observers plugins.  Default: match nothing (observer disabled).
    fn stage_filter() -> (&'static str, &'static str) { ("", "") }

    /// Called when a stage event fires. Return `StageOutput { action:
    /// "continue" }` to proceed, or `{ action: "abort", reason: "..." }`
    /// to abort the pipeline.
    fn observe_stage(_event: &StageInput) -> StageOutput {
        StageOutput { action: alloc::string::String::from("continue"), reason: alloc::string::String::new() }
    }

    // ── Internal: assemble metadata JSON ──

    fn build_metadata() -> PluginMeta {
        let mut meta = PluginMeta::default();
        meta.plugin_type = Self::plugin_type().into();
        meta.name = Self::name().into();
        meta.description = Self::description().into();
        meta
    }
}

// ── export! macro ──

/// Generate all `extern "C"` entry-points for a plugin type.
///
/// Place this at crate root:
/// ```
/// struct MyPlugin;
/// impl export::Plugin for MyPlugin { ... }
/// openagent_pdk::export!(MyPlugin);
/// ```
#[macro_export]
macro_rules! export {
    ($t:ty) => {
        // ── alloc ──
        #[no_mangle]
        pub extern "C" fn alloc(size: u32) -> u32 {
            $crate::sdk_alloc(size)
        }

        // ── metadata ──
        #[no_mangle]
        pub extern "C" fn metadata() -> u64 {
            let meta = <$t as $crate::export::Plugin>::build_metadata();
            // Build JSON by hand to avoid pulling serde_json into every build.
            // PluginMeta has a custom Serialize that outputs the right fields.
            $crate::sdk_return_json(&meta)
        }

        // ── cli:settings: init ──
        #[no_mangle]
        pub extern "C" fn init(ptr: u32, len: u32) -> u64 {
            let input = unsafe { $crate::wasm_str(ptr, len) };
            match <$t as $crate::export::Plugin>::init(input) {
                Ok(s) => $crate::sdk_return(s.as_bytes()),
                Err(e) => {
                    $crate::host::log_error(&e);
                    // Return original settings unchanged on error.
                    $crate::sdk_return(input.as_bytes())
                }
            }
        }

        // ── cli:commands: commands ──
        #[no_mangle]
        pub extern "C" fn commands() -> u64 {
            let cmds = <$t as $crate::export::Plugin>::commands();
            $crate::sdk_return_json(&cmds)
        }

        // ── cli:commands: run_<name> — macro can't generate dynamic exports
        // so author must manually write #[no_mangle] pub extern "C" fn run_<name>.
        // See run_command! helper below.

        // ── cli:observers ──
        #[no_mangle]
        pub extern "C" fn on_startup() {
            <$t as $crate::export::Plugin>::on_startup();
        }

        #[no_mangle]
        pub extern "C" fn on_shutdown() {
            <$t as $crate::export::Plugin>::on_shutdown();
        }

        #[no_mangle]
        pub extern "C" fn on_command_start(ptr: u32, len: u32) {
            let cmd = unsafe { $crate::wasm_str(ptr, len) };
            <$t as $crate::export::Plugin>::on_command_start(cmd);
        }

        #[no_mangle]
        pub extern "C" fn on_command_end(ptr: u32, len: u32) {
            let payload = unsafe { $crate::wasm_str(ptr, len) };
            <$t as $crate::export::Plugin>::on_command_end(payload);
        }

        // ── agent:tools: execute ──
        #[no_mangle]
        pub extern "C" fn execute(ptr: u32, len: u32) -> u64 {
            let input: $crate::types::ToolInput = $crate::read_input_json(
                $crate::pk(ptr, len)
            );
            let result = <$t as $crate::export::Plugin>::execute(&input.args);
            let out = match result {
                Ok(r) => $crate::types::ToolOutput { result: r, error: alloc::string::String::new() },
                Err(e) => $crate::types::ToolOutput { result: alloc::string::String::new(), error: e },
            };
            $crate::sdk_return_json(&out)
        }

        // ── agent:observers: run ──
        #[no_mangle]
        pub extern "C" fn run(ptr: u32, len: u32) -> u64 {
            let input: $crate::types::StageInput = $crate::read_input_json(
                $crate::pk(ptr, len)
            );
            let out = <$t as $crate::export::Plugin>::observe_stage(&input);
            $crate::sdk_return_json(&out)
        }
    };
}

/// Helper for cli:commands plugins: generates a single run_<name> export.
///
/// Usage:
/// ```
/// run_command!(stats, MyPlugin);
/// ```
/// Expands to:
/// ```
/// #[no_mangle] pub extern "C" fn run_stats(ptr: u32, len: u32) -> u64 { ... }
/// ```
#[macro_export]
macro_rules! run_command {
    ($name:ident, $t:ty) => {
        #[no_mangle]
        pub extern "C" fn $name(ptr: u32, len: u32) -> u64 {
            // concat_idents is unstable, so we use a fixed pattern:
            // the command name is the function name with "run_" prefix stripped.
            let cmd_name = stringify!($name);
            let args = unsafe { $crate::wasm_str(ptr, len) };
            match <$t as $crate::export::Plugin>::run_command(cmd_name, args) {
                Ok(s) => $crate::sdk_return(s.as_bytes()),
                Err(e) => $crate::sdk_return(e.as_bytes()),
            }
        }
    };
}

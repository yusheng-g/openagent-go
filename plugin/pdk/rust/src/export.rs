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

    /// Handle execution of a registered command. input carries parsed args
    /// and flags from the CLI (see CommandInput).
    fn run_command(_name: &str, _input: &CommandInput) -> Result<String, String> {
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

    // ── agent:sessions ──

    /// Called on session creation. Return Some(SessionConfig) to override
    /// Agent opts for this session, or None to leave defaults.
    fn on_session_init(_ctx: &SessionCtx) -> Result<Option<SessionConfig>, String> {
        Ok(None)
    }

    /// Called on session deletion. Fire-and-forget — return value is ignored.
    fn on_session_destroy(_ctx: &SessionCtx) {}

    // ── Internal: assemble metadata JSON ──

    fn build_metadata() -> PluginMeta {
        let mut meta = PluginMeta::default();
        meta.plugin_type = Self::plugin_type().into();
        meta.name = Self::name().into();
        meta.description = Self::description().into();
        let (stage, phase) = Self::stage_filter();
        meta.stage = stage;
        meta.phase = phase;
        meta
    }
}

// ── export! macro ──

#[macro_export]
macro_rules! export {
    ($t:ty) => {
        #[no_mangle]
        pub extern "C" fn alloc(size: u32) -> u32 { $crate::sdk_alloc(size) }

        #[no_mangle]
        pub extern "C" fn metadata() -> u64 {
            let meta = <$t as $crate::export::Plugin>::build_metadata();
            $crate::sdk_return_json(&meta)
        }

        // ── cli:settings ──
        #[no_mangle]
        pub extern "C" fn init(ptr: u32, len: u32) -> u64 {
            let input = unsafe { $crate::wasm_str(ptr, len) };
            match <$t as $crate::export::Plugin>::init(input) {
                Ok(s) => $crate::sdk_return(s.as_bytes()),
                Err(e) => { $crate::host::log_error(&e); $crate::sdk_return(input.as_bytes()) }
            }
        }

        // ── cli:commands ──
        #[no_mangle]
        pub extern "C" fn commands() -> u64 {
            let cmds = <$t as $crate::export::Plugin>::commands();
            $crate::sdk_return_json(&cmds)
        }

        // ── cli:observers ──
        #[no_mangle] pub extern "C" fn on_startup() { <$t as $crate::export::Plugin>::on_startup(); }
        #[no_mangle] pub extern "C" fn on_shutdown() { <$t as $crate::export::Plugin>::on_shutdown(); }
        #[no_mangle] pub extern "C" fn on_command_start(ptr: u32, len: u32) {
            let cmd = unsafe { $crate::wasm_str(ptr, len) };
            <$t as $crate::export::Plugin>::on_command_start(cmd);
        }
        #[no_mangle] pub extern "C" fn on_command_end(ptr: u32, len: u32) {
            let payload = unsafe { $crate::wasm_str(ptr, len) };
            <$t as $crate::export::Plugin>::on_command_end(payload);
        }

        // ── agent:tools ──
        #[no_mangle]
        pub extern "C" fn execute(ptr: u32, len: u32) -> u64 {
            let input: $crate::types::ToolInput = $crate::read_input_json($crate::pk(ptr, len));
            let result = <$t as $crate::export::Plugin>::execute(&input.args);
            let out = match result {
                Ok(r) => $crate::types::ToolOutput { result: r, error: alloc::string::String::new() },
                Err(e) => $crate::types::ToolOutput { result: alloc::string::String::new(), error: e },
            };
            $crate::sdk_return_json(&out)
        }

        // ── agent:observers ──
        #[no_mangle]
        pub extern "C" fn run(ptr: u32, len: u32) -> u64 {
            let input: $crate::types::StageInput = $crate::read_input_json($crate::pk(ptr, len));
            let out = <$t as $crate::export::Plugin>::observe_stage(&input);
            $crate::sdk_return_json(&out)
        }

        // ── agent:sessions ──
        #[no_mangle]
        pub extern "C" fn session_init(ptr: u32, len: u32) -> u64 {
            let ctx: $crate::types::SessionCtx = $crate::read_input_json($crate::pk(ptr, len));
            match <$t as $crate::export::Plugin>::on_session_init(&ctx) {
                Ok(Some(cfg)) => $crate::sdk_return_json(&cfg),
                Ok(None) => $crate::sdk_return(b"null"),
                Err(e) => $crate::sdk_return_json(&$crate::types::HostResult { error: e }),
            }
        }

        #[no_mangle]
        pub extern "C" fn session_destroy(ptr: u32, len: u32) {
            let ctx: $crate::types::SessionCtx = $crate::read_input_json($crate::pk(ptr, len));
            <$t as $crate::export::Plugin>::on_session_destroy(&ctx);
        }
    };
}

/// Helper for cli:commands plugins: generates a single run_<name> export.
#[macro_export]
macro_rules! run_command {
    ($name:ident, $t:ty) => {
        #[no_mangle]
        pub extern "C" fn $name(ptr: u32, len: u32) -> u64 {
            let cmd_name = stringify!($name);
            let args = unsafe { $crate::wasm_str(ptr, len) };
            match <$t as $crate::export::Plugin>::run_command(cmd_name, args) {
                Ok(s) => $crate::sdk_return(s.as_bytes()),
                Err(e) => $crate::sdk_return(e.as_bytes()),
            }
        }
    };
}

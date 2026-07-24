# BUGS.md — Known Issues & Technical Debt

> Last updated 2026-07-22 (rev 11).
> Format: `[P0]` = critical, `[P1]` = high, `[P2]` = medium, `[P3]` = low.
> `[DEBT]` = technical debt (no immediate breakage, will compound).

---

## 🐛 Bugs

### [P1] ACP approval: "Allow Always" does not persist — asks again on next tool call

[acp/server.go:1383-1386](acp/server.go), [acp/server.go:1417](acp/server.go): The `acpApprover` struct is stateless — selecting "Allow Always" in the approval dialog behaves identically to "Allow" (one-shot). The `case "allow", "always"` branch at line 1417 returns `true` but does not record the decision. Additionally, `agentForTurn()` (line 1196) creates a new `acpApprover` for each turn, so any state added to the struct would not survive across turns.

**Root cause**: Two compounding issues:
1. `acpApprover.Approve()` handles `"allow"` and `"always"` identically — no decision caching
2. `agentForTurn()` creates a fresh `acpApprover` per turn, so per-struct cache would still not persist

**Fix plan**: Store "allow_always" decisions in a session-scoped cache on `agentSession` (via `sync.Map`), shared with each turn's `acpApprover` through a pointer. When the cache has an entry for the current tool name, skip the `RequestPermission` round-trip entirely. Only the "always" option updates the cache; "allow" (one-shot) remains uncached.

Repro:
1. Set mode to "manual" in VS Code ACP plugin
2. Trigger a tool (e.g., `bash_execute "echo hello"`)
3. Select "Allow Always" in the approval dialog
4. Trigger the same tool again — dialog reappears (should be skipped)

---

### [P1] ~~`cli serve` fails to start with "unexpected end of JSON input" when settings.json missing/empty and no plugins~~ ✅ FIXED

[cmd/cli/main.go:62,65-67](cmd/cli/main.go): **Fixed in this commit** — `settings` is normalized to `[]byte("{}")` when empty before the plugin loop, so both `CallInit` and the final `Unmarshal` always receive valid JSON.

Original issue: `settings = raw` (line 62) ended up as a nil or empty byte slice in three scenarios — (1) `settings.json` not created, (2) file exists but empty, (3) no settings plugin loaded → `json.Unmarshal(settings, &cfg)` (line 122) failed with `unexpected end of JSON input` and aborted startup. Repro: `./openagent-cli serve --acp` with no plugins + missing/empty settings.json.

Note: line 43 `json.Unmarshal(raw, &preCfg)` silently swallows the same empty-input error — non-fatal there (defaults to empty `Config{}`, falls back to `DefaultPluginsDir()`). Left untouched as this is the expected graceful-degradation behavior.

---

### [P1] ~~extended-settings plugin emits invalid JSON / overwrites existing `provider` map~~ ✅ FIXED

[cmd/cli/examples/plugin/extended-settings/src/lib.rs](cmd/cli/examples/plugin/extended-settings/src/lib.rs): **Fixed in this commit** — `cli_init` now delegates to a pure `inject_settings` helper that parses settings via `serde_json` and deep-merges `provider.my_provider` and `env.MY_PROVIDER_API_KEY` into the existing object tree (creating maps if absent). All merge semantics are overwrite: keyring values replace user values for `api_key`/`base_url`/`models`/`env.MY_PROVIDER_API_KEY`; other existing keys preserved. 6 unit tests cover all three cases below plus overwrite, invalid-JSON fallback, and non-object root.

Original issue: `cli_init` unconditionally prefixed the injected block with `,` and unconditionally created a fresh `"provider"` object, producing three failure modes: (1) empty `{}` → `{,"provider":...` invalid JSON; (2) non-empty without `provider` — same fragile code path, only syntactically valid by accident; (3) non-empty with existing `provider` → overwrote the whole map → silent data loss of user-configured providers. Symptom: `parse merged settings: invalid character ',' looking for beginning of object key string` after `plugin: loaded extended-settings (...)`.

---

### [P1] ~~`memory/sqlite` FTS — CJK search returns nothing~~ ✅ FIXED

[memory/sqlite/memory.go](memory/sqlite/memory.go): **Fixed in this commit** — `migrate` now creates `messages_fts` with the `trigram` tokenizer (rebuilds legacy `unicode61` tables in place + backfills from `messages`). `ftsSearch` trims leading/trailing punctuation from each token, OR-joins them as phrases with BM25 ranking, and falls back to a `LIKE` substring scan for tokens too short (<3 chars) for trigram.

Original issue: `migrate()` created `messages_fts` with the default `unicode61` tokenizer, which treats a run of CJK characters as one token, so CJK queries matched nothing. A punctuation (`! ? . , ; / #`) crash was partially mitigated by stripping those characters before `MATCH` and returning `nil` on empty queries, but FTS5 still only worked for whitespace-separated languages (English, European).

---

### [P2] ~~VSCode ACP plugin mode indicator not updated after /mode or exit_plan_mode~~ ✅ FIXED

[acp/server.go:671-691](acp/server.go), [acp/server.go:851-879](acp/server.go): **Fixed in this commit** — `setSessionMode` now sends both `current_mode_update` and `config_option_update`; `exit_plan_mode` now calls `setSessionMode` instead of manually setting mode + sending only `current_mode_update`; `OnSetSessionConfigOption` skips duplicate `config_option_update` when mode was changed.

`setSessionMode` only sends `current_mode_update` (line 684-688), not `config_option_update`. The VSCode ACP plugin renders the mode selector as a config option (ID: `"mode"`) and relies on `config_option_update` to refresh its value. When mode is changed via `/mode` slash command, `session/set_mode` RPC, or `exit_plan_mode` tool, the plugin's mode indicator is not updated — even though the agent's internal mode (and actual tool gating behavior) has correctly changed. `OnSetSessionConfigOption` is the only path that sends both notifications, so only mode changes via the plugin's own config UI work correctly.

Additionally, `exit_plan_mode` (line 851-879) manually sets `ss.mode` and sends only `current_mode_update`, bypassing `setSessionMode` entirely. This has the same symptom: after the agent calls `exit_plan_mode`, the actual mode reverts to auto/manual, but the plugin indicator still shows "plan".

Repro:
1. `/mode plan` → echo shows success, agent enters plan mode, but plugin indicator still shows previous mode
2. In plan mode, agent calls `exit_plan_mode` → actual mode reverts, but plugin indicator still shows "plan"

Fix:
1. Add `config_option_update` notification to `setSessionMode` (alongside the existing `current_mode_update`).
2. Replace the manual mode-setting + single notification in `exit_plan_mode`'s callback with a call to `setSessionMode`.
3. In `OnSetSessionConfigOption`, skip the explicit `config_option_update` when mode was changed (to avoid double-sending since `setSessionMode` now sends it).

---

### [P2] ~~Team: no model selection, no ContextWindow, stale ModelID~~ ✅ FIXED

[rest/team.go](rest/team.go): **Fixed in this commit** — `TeamHandler` now carries a model registry (`models`/`modelList`/`modelsMu`) mirroring `Handler`, with `RegisterModel`/`lookupModel`. `handleChat` resolves the model from `ChatRequest.ModelID`/`Provider` (falling back to stored session meta, then the first-template model), persists `modelId`/`provider` to session meta via `withMeta`+`syncMeta`, and sets `Model`/`ModelID` on the `openagent.Session` handed to `team.RunStream`. Because `runner.go:68-70` does `r.runModel = session.Model if non-nil`, the selected model overrides every team agent's model for that run. `NewTeamHandler` wires a `fillDetail` hook that resolves the session's effective model (stored meta > handler default) and sets `detail.ContextWindow`, so `GET /team/sessions/{id}` returns a non-zero `contextWindow` and the current (non-stale) `modelId`.

Original issue:
1. `handleChat` ignored `ChatRequest.ModelID` — `oaSession` constructed without `Model` or `ModelID`. Frontend model selector had no effect in team mode.
2. `s.info.ModelID` never synced to team handler. `GET /team/sessions/{id}` returned creation-time `ModelID` forever.
3. `handleGetSession` missing `ContextWindow` — Frontend showed `contextWindow: 0` for team sessions.

Tests: `rest/team_model_test.go` — `TestTeamHandleChatModelOverride` (model-b is invoked when selected) and `TestTeamGetSessionContextWindow` (`contextWindow == 16000` and `_meta.modelId == "model-b"` after a model-b chat).

---

### [P2] `fetchMessages` can overwrite live SSE stream

[examples/frontend/vue-app/src/stores/chat.ts](examples/frontend/vue-app/src/stores/chat.ts): `watchEffect` → `clearChat()` → `fetchMessages()` (async). Between `clearChat()` and `fetchMessages` completing, a live SSE stream can push messages. When `fetchMessages` resolves, `messages.value = converted` unconditionally overwrites live messages.

---

### [P2] ~~`guard/llm/guard.go` — Parse failure defaults to block (fail-closed)~~ ✅ FIXED

[guard/llm/guard.go](guard/llm/guard.go): **Fixed in this commit** — `parseResult` accepts a `failOpen bool` parameter (signature `func parseResult(content string, failOpen bool) openagent.GuardResult`) and honors it on the parse-failure path: when `json.Unmarshal` fails and both `"allowed": true/false` substring matches miss, it returns `Allowed: true` if `failOpen` is set, `Allowed: false` otherwise. The `judge` method forwards `g.failOpen` into `parseResult`, so `failOpen` now covers network/API errors, empty choices, **and** parse errors — all three failure modes. The default (`WithFailOpen` unset → `failOpen=false`) remains fail-closed, the documented safety posture.

Original issue (stale — described a version before `995bbb8`): `parseResult` did substring match on `"allowed": true/false`. If substring match also failed, defaulted to `Allowed: false`. The `failOpen` option only covered network/API errors and empty choices, not parse errors (`parseResult` ignored `failOpen` when it couldn't extract a boolean). This was already addressed in commit `995bbb8` (refactor that threaded `failOpen` into `parseResult`), predating this entry.

---

### [P2] Dynamic team agents not persisted across restarts

[rest/team.go](rest/team.go): `handleAddAgent` creates agents at runtime. `SessionStore` only persists `SessionInfo` — agent list not stored. After restart, `getOrCreate` → `newEntry` rebuilds from templates only.

---

### [P3] ~~API credit leak on client SSE disconnect~~ ✅ FIXED

[rest/handler.go:301](rest/handler.go), [rest/team.go:227](rest/team.go), [rest/orchestrate_handler.go:268](rest/orchestrate_handler.go): **Fixed in this commit** — all three chat goroutines now derive the run context from the request via `context.WithTimeout(r.Context(), ...)` instead of `context.WithTimeout(context.Background(), ...)`. On client SSE disconnect, `r.Context()` is cancelled → the derived ctx is cancelled → `runner.go:126` sees `ctx.Done()` and returns from `run` → `RunStream`'s internal goroutine runs `defer close(ch)` (`agent.go:120`) → the publish `for range ch` loop exits. The LLM call stops immediately instead of running to the timeout with no consumer. The timeout still caps the run on a healthy connection. `handler.go`'s manual `select <-r.Context().Done()` check in the publish loop was removed (now redundant — ctx propagation stops the agent directly). `rest/sse_disconnect_test.go` adds two end-to-end httptest cases: `TestSSEDisconnectCancelsAgent` (disconnect → model ctx cancelled within 3s; confirmed to hang on the old `context.Background` code) and `TestSSENormalCompletionNotPrematurelyCancelled` (normal completion → model returns, not prematurely cancelled).

Original issue: all three used `context.Background()` (with long timeout) for agent goroutines. SSE client disconnects → goroutine continues running with no consumer.

---

### [P2] ~~`cmd/cli/keyring` silently falls back to `MemStore` in D-Bus-less environments, losing persisted secrets~~ ✅ FIXED

[cmd/cli/keyring/keyring.go:46-58](cmd/cli/keyring/keyring.go),
[cmd/cli/keyring/keyring_linux.go:29-138](cmd/cli/keyring/keyring_linux.go),
[cmd/cli/main.go:317-338](cmd/cli/main.go),
[go.mod:9,23](go.mod): **Fixed in this commit** — `Open()` delegates to a platform-dispatched `openBackend()`. On Linux, `isSecretServiceAvailable()` explicitly checks `dbus.SessionBus()` + `NameHasOwner("org.freedesktop.secrets")` (no more `gkr.Get("__probe__")` autolaunch), then falls back to a kernel-keyring backend (`keyctlBackend`) implemented on `golang.org/x/sys/unix` (`KeyctlLink`/`KeyctlSearch`/`KeyctlRead`/`AddKey`/`KeyctlInt(KEYCTL_UNLINK)`). `ErrKeyringUnavailable` sentinel and `HasSupport()` added. `cli keyring set`/`delete` use a new `keyringOrFail()` helper that `log.Fatalf`s with an actionable message ("install `dbus-x11` or run with `--cap-add=keyutils`") instead of silently writing to `MemStore` — eliminating the silent-data-loss derived issue. `github.com/godbus/dbus/v5` and `golang.org/x/sys` promoted to direct deps. macOS/Windows paths unchanged (still use `zalando` `__probe__`, correct for non-D-Bus backends).

Original issue: `Open()` probed the system keychain with `gkr.Get("openagent", "__probe__")`. On Linux, `zalando/go-keyring v0.2.8` routes through the Secret Service API (`github.com/godbus/dbus/v5`). When `DBUS_SESSION_BUS_ADDRESS` was unset and no `dbus-launch` existed on `PATH`, `godbus` attempted to exec `dbus-launch` and returned `exec: "dbus-launch": executable file not found in $PATH`. This error propagated up to `openKeyring()`, which substituted `MemStore` with a WARNING. Repro: Huawei Cloud EulerOS 2 container (aarch64, no `dbus-x11` installed).

**Remaining follow-up (not fixed, explicitly out of scope per scope notes):**
- Double WARNING: `main()` calls `openKeyring()` once at startup, then `keyring get` calls it again, printing the same warning twice. `set`/`delete` no longer double-warn (they use `keyringOrFail()`). Scope notes (below) record this as not being fixed in this change.

SCOPE NOTES (per user direction, honored):
- keyring library was NOT swapped to `99designs/keyring`; `zalando/go-keyring` stays.
- File-based fallback (Plan B) was rejected; kernel keyring is the last non-volatile tier, and a missing kernel-keyring capability surfaces as an explicit error.

---

### [P1] ~~Sandbox environment has no outbound network connectivity~~ ✅ FIXED

[cmd/cli/main.go:128-130](cmd/cli/main.go),
[cmd/cli/settings/settings.json:18-21](cmd/cli/settings/settings.json),
[cmd/cli/server/http.go:25-54](cmd/cli/server/http.go),
[sandbox/native/native.go](sandbox/native/native.go),
[sandbox/native/native_linux.go](sandbox/native/native_linux.go),
[cmd/cli/config/config.go](cmd/cli/config/config.go):

**Fixed in this commit** — root cause was
`sandbox/native/native_linux.go:bwrapArgs()` unconditionally passing
`--unshare-all` to `bwrap`. That flag implies `--unshare-net`, putting
the sandboxed process into a fresh network namespace with no routes,
no DNS, and no outbound connectivity. Every shell command the agent
ran (`curl`, `pip install`, `hcloud`, etc.) failed as a result. (The
agent's own LLM HTTP calls go through the main Go process, not the
sandbox, so those were unaffected — only shell-tool network was dead.)

Fix is five-layered:

1. **Policy API in `sandbox/native`** — added `Policy{Network,
   WritablePaths, ReadablePaths}` and `NewWithPolicy(workDir, Policy)`.
   `New(workDir)` now delegates to `NewWithPolicy(workDir, Policy{})`
   whose zero-value `Network == ""` means **host** (network allowed).
2. **`bwrapArgs()` reworked** — replaced `--unshare-all` (which
  hard-fails on user-namespace creation in restricted containers) with
   explicit namespace flags: `--unshare-user-try` (graceful),
   `--unshare-ipc`, `--unshare-pid`, `--unshare-uts`,
   `--unshare-cgroup-try`. Network is governed directly: `isolated`
   policy adds `--unshare-net`; `host`/default omits it entirely (no
   more `--unshare-all` + `--share-net` roundtrip). `WritablePaths` /
   `ReadablePaths` produce additional `--bind` / `--ro-bind` entries.
3. **`/etc` network config mounted read-only** — `bwrapArgs()` now
   bind-mounts `/etc/resolv.conf`, `/etc/hosts`, `/etc/nsswitch.conf`,
   and `/etc/ssl` via `--ro-bind-try`. Without these, even with host
   network namespace sharing, glibc inside the sandbox cannot resolve
   DNS (no resolv.conf) and TLS verification fails (no CA certs).
   This was the second root cause: the first-round fix added
   `--share-net` but the agent still saw `curl exit 6 (Couldn't
   resolve host)` because resolv.conf wasn't mounted.
4. **bwrap-startup-failure fallback** — `confineAndRun` and
   `confineAndRunStream` now detect bwrap setup failures (empty
   stdout + stderr starting with `bwrap:`) and fall back to
   unconfined execution silently, instead of returning the bwrap
   error to the agent. Previously the fallback only triggered
   when bwrap was not found (`exec.LookPath` failed); if bwrap was
   installed but couldn't start (e.g. `setting up uid map: Permission
   denied` in containers), the shell command never ran and the agent
   only saw the bwrap error. The fallback is silent for the
   high-frequency bwrap-startup-failure path (every shell command in
   containers) to avoid log spam; a WARNING is still logged for the
   low-frequency cases (bwrap not installed, `c.Start()` fails).
5. **Configurable from `settings.json`** — `cmd/cli/config/config.go`
   gained a `SandboxConfig{Network, WritablePaths, ReadablePaths}`
   field. `cmd/cli/server/{http,acp}.go` switched from `native.New()`
   to `native.NewWithPolicy(cwd, sandboxPolicy(cfg.Sandbox))`. Missing
   config defaults to host networking — the agent can finally reach
   the internet through shell tools. Users who want the old isolated
   behavior can set `{"sandbox": {"network": "isolated"}}` in
   `settings.json`.

The policy tests in `native_policy_linux_test.go` (previously failing
to compile because the API didn't exist) now pass, including
`TestBwrapArgsEtcMounts` (verifies /etc network files are mounted)
and `TestBwrapArgsNetworkPolicy` (verifies `--unshare-net` is
absent for host/default, present for isolated). The renamed
`TestSandboxIsolatedPolicyBlocksNetwork` verifies the isolated policy
still blocks network end-to-end — it auto-skips when the sandbox
itself can't start (via `sandboxFunctional` helper), so it no longer
false-fails in containers that block bwrap.

Original issue: the containerized sandbox (observed in the local
runtime log) had no outbound network at all. This was an
environment-level limitation that broke core `openagent-cli serve`
runtime paths assuming network access.

Evidence (log lines 205-227, 405-432):
- `curl https://www.baidu.com` and `curl https://www.google.com` →
  HTTP `000` (immediate failure)
- `host google.com` / `nslookup` → DNS resolution failed
- `ip route` → empty routing table; no visible network interfaces
- `HTTP_PROXY`/`HTTPS_PROXY` configured in `settings.json:18-21` point
  to `proxynj.huawei.com:8080` (Huawei internal proxy), which is
  equally unreachable from inside the sandbox

Impact on `openagent-cli serve` (before the fix):
- `server/http.go:38-53` `buildModels` constructs OpenAI clients for
  every provider in `settings.json` (`api.deepseek.com`,
  `open.bigmodel.cn`). Every agent turn that calls the model endpoint
  fails — the agent cannot produce any response.
- `main.go:128-130` injects `cfg.Env` (including the proxy vars) via
  `os.Setenv`, but since the proxy host is itself unreachable from the
  sandbox, the proxy configuration provides no relief.
- Runtime package installation (`pip install`, `hcloud` download) is
  impossible, so SDKs/CLIs cannot be added on the fly.

Repro (log lines 84-99, 194-203): user asked to query cn-north-4 ECS
list — with no network and no preinstalled cloud CLI/SDK, the agent
could not reach any Huawei Cloud endpoint and the user had to supply
data manually.

**Remaining follow-ups (not fixed in this commit):**

1. Add `openagent-cli doctor` subcommand that probes network
   reachability (proxy host, DNS, routing table) at startup and
   reports the degraded mode explicitly. Today failures only surface
   when an agent turn actually tries to call out, producing opaque
   timeouts.
2. Detect empty routing table / unreachable proxy during `serve`
   startup and fail fast with an actionable message instead of letting
   every LLM call hit a timeout.
3. Document that in network-isolated sandboxes (when the user opts
   into `sandbox.network: "isolated"`), the agent must delegate
   network-bound operations to the host machine via
   `terminal_create` / `read_client_file` rather than executing them
   in-sandbox.
4. ✅ Resolved — `TestSandboxWorkspaceAccess` / `TestSandboxStreaming`
   now pass via the bwrap-startup-failure fallback (they fall back to
   unconfined execution and the commands succeed). The isolation tests
   `TestSandboxBlocksExternalAccess` /
   `TestSandboxIsolatedPolicyBlocksNetwork` auto-skip via the
   `sandboxFunctional` helper when bwrap can't start, instead of
   false-failing.

---

### [P1] ~~`cmd/cli` server modes hardcode `MaxTurns=10` — complex tasks silently truncated~~ ✅ FIXED (partial)

[cmd/cli/server/acp.go:63](cmd/cli/server/acp.go#L63),
[cmd/cli/server/http.go:53](cmd/cli/server/http.go#L53): **Fixed in this commit** — both call sites bumped from `WithMaxTurns(10)` to `WithMaxTurns(100)`. 100 turns covers any realistic single-task workflow (read → search → edit → test → fix → rerun) without exhausting the budget mid-investigation, while still providing a safety cap against runaway loops.

[runner.go:59-62](runner.go#L59),
[runner.go:121](runner.go#L121),
[runner.go:456-463](runner.go#L456),
[agent.go:77](agent.go#L77),
[cmd/tui/main.go:44](cmd/tui/main.go#L44):

Original issue: Both CLI server entry points hardcoded `openagent.WithMaxTurns(10)` when
constructing the agent, and the value was not exposed in `settings.json`
or any CLI flag. One "turn" in this framework equals one LLM call plus
one round of tool execution (`runner.go:121`), so on any non-trivial
task — read a few files, grep, edit, run tests, fix, rerun — the budget
was exhausted before the agent finishes. The CLI server modes were in fact
*more restrictive than the framework's own default* of 20
(`runner.go:60-62`, `agent.go:77`, used by `cmd/tui/main.go:44`).

**Silent truncation is the worst part.** When `turn > maxTurns`, the
`for` loop at `runner.go:121` simply exits, falls through to
`runner.go:456-463`:

```go
}  // end of for-loop
result.TurnCount = turn
result.ContextWindow = r.runModel.ContextWindow()
if ch != nil {
    ch <- StreamEvent{Type: StreamDone, Result: result}
}
return result, nil
```

`result.StopReason` is left empty, no error is returned, and a normal
`StreamDone` event is emitted — indistinguishable from a graceful
"model returned no tool_calls" stop (`runner.go:393-405`). The user sees
a half-finished answer that looks complete; tool_calls left pending by
the last assistant turn are never executed and never reported.

**Impact:**
- ACP mode (`cli serve --acp`): any task needing >10 turns returns a
  truncated, "successful-looking" response. Observed during the
  Huawei Cloud ECS diagnosis session — the agent ran out of turns
  mid-investigation and returned partial findings as if finished.
- REST mode (`cli serve`): same silent truncation over SSE; frontend
  shows a normal `done` event with an incomplete answer.
- No workaround available to end users without recompiling — the value
  is neither in `settings.json` nor a CLI flag.

**Remaining follow-ups (not fixed in this commit):**

1. ✅ Bump the CLI server default — done (10 → 100).
2. Surface hitting the cap explicitly. In `runner.go:456`, detect
   `turn > maxTurns` (i.e. the loop exited without `break` and the last
   `choice.Message.ToolCalls` was non-empty) and set
   `result.StopReason = "max_turns"` plus emit a `StreamEvent{Type:
   StreamWarning, ...}` (or at minimum log a WARNING). Frontends and
   ACP clients can then prompt the user to continue.
3. Make the value configurable: add `maxTurns` (or `agent.maxTurns`)
   to `settings.json` schema in `cmd/cli/settings/`, and a
   `--max-turns` flag on `cli serve`, defaulting to the bumped value.
   Mirror the pattern already used for `cfg.Provider` / `cfg.Profiles`.

---

### [P1] ~~ACP Agent→Client RPC tools registered without checking client capabilities~~ ✅ FIXED

[acp/server.go](acp/server.go): **Fixed in this commit** — `OnInitialize` now persists `req.ClientCapabilities` (guarded by `s.mu`). Three capability helpers (`clientCanReadFile`, `clientCanWriteFile`, `clientCanTerminal`) gate tool registration in `agentForTurn` (all three modes: plan, auto, manual) and `injectExecutionTools`. The plan-mode system prompt in `buildDynamicContext` conditionally advertises `read_client_file` only when the client supports it.

Original issue: `OnInitialize` receives `req.ClientCapabilities` (including `fs.readTextFile`, `fs.writeTextFile`, `terminal`) from the client during the `initialize` handshake, but discards it entirely — only hardcoded `AgentCapabilities` are returned. `agentForTurn` and `injectExecutionTools` then register `read_client_file`, `write_client_file`, and all `terminal/*` tools based solely on `s.clientRPC != nil`, without checking whether the client actually advertised support for these RPCs.

When a client that does not implement `fs/read_text_file` (e.g., a browser-based or mobile ACP client) connects, the LLM is offered `read_client_file` and calls it, but the client rejects the `fs/read_text_file` RPC with JSON-RPC `-32601 Method not found`. The agent wraps this as `read_client_file: acp: fs/read_text_file call failed: ... not available on this client`. The same applies to `write_client_file` and all `terminal/*` tools.

Per the ACP spec, capabilities are negotiated during `initialize` — presence signals support, absence signals the feature is unavailable. The agent must not offer tools whose Agent→Client RPCs the client cannot handle.

---

### [P1] ~~Skills not recognized or usable in `cli serve` (ACP + REST modes)~~ ✅ FIXED

[cmd/cli/server/acp.go:65](cmd/cli/server/acp.go), [cmd/cli/server/http.go:62](cmd/cli/server/http.go), [cmd/cli/server/shared.go:270-288](cmd/cli/server/shared.go): **Fixed in this commit** — both CLI server entry points call `buildOpts(opts, caps, model)`, which wires `openagent.WithSkillLoader(openSkillLoader())` when `caps.OnSkills()` is true (default on) and a skill directory exists. `openSkillLoader()` (`shared.go:216-223`) auto-discovers skill directories via `skillDirs()` (`shared.go:225-238`), probing four locations: `~/.openagent/skills`, `~/.agents/skills`, `$cwd/.agents/skills`, `$cwd/.openagent/skills`. The runner's nil-gate at `runner.go:74-79` now gets a non-nil loader when any directory exists, enabling `use_skill`/`reload_skills` tools and the skill catalog in the dynamic prompt.

Original issue: Both CLI server entry points constructed the agent **without** `openagent.WithSkillLoader(...)`, so `agent.SkillLoader` stayed nil and the runner short-circuited the entire skill subsystem (no `use_skill`/`reload_skills` tools, no "## Available Skills" catalog, no "## Loaded Skill:" body). The `Config` struct had no skills directory field. Cross-confirmation: `WithSkillLoader` was called only in `cmd/iac-mcp/main.go`, `examples/skill/main.go`, and `examples/iac/agents.go` — never under `cmd/cli/server/`.

Implementation note: the fix used auto-discovery of four well-known directories rather than the `Config.Skills string` field proposed in the original fix plan, and named the helper `openSkillLoader` rather than `resolveSkillLoader`. Same effect; no `settings.json` field needed.

---

### [P2] ~~`plan_create` available in all modes, bypassing plan-mode workflow~~ ✅ FIXED

[plan/tool.go:205-260](plan/tool.go), [acp/server.go:740-750](acp/server.go), [acp/server.go:867-919](acp/server.go), [acp/server.go:1263-1285](acp/server.go): **Fixed in this commit** — Added `EnterTool` (`enter_plan_mode`) to `plan/tool.go` as a symmetrical counterpart to `ExitTool` (`exit_plan_mode`). `plan_create` registration moved inside the `if ss.mode == "plan"` block in `OnPrompt`; `enter_plan_mode` registered in the `else` branch (auto/manual modes). `enterPlanMode` helper added to `AgentServer` to persist the mode transition via `setSessionMode`. `buildDynamicContext` updated to hint auto/manual agents about `enter_plan_mode` when no plan exists. The `enter_plan_mode` tool result card is suppressed in the CLI channel UI (same as `plan_update`). Cross-turn approach avoids removing execution tools from the agent clone mid-turn. `enter_plan_mode` inherits the current mode's approver automatically — auto runs without approval, manual triggers user confirmation via `acpApprover`.

Original issue: `plan_create` was registered unconditionally in `OnPrompt` — available in auto, manual, and plan modes. The agent in auto/manual mode could call `plan_create` and immediately begin executing, bypassing the "enter plan mode → create plan → user review → exit plan mode → execute" workflow. The symmetry was also broken: `exit_plan_mode` had no `enter_plan_mode` counterpart.

Final tool availability:

| Tool | auto | manual | plan |
|------|:----:|:------:|:----:|
| `enter_plan_mode` | ✓ (no approval) | ✓ (needs approval) | ✗ |
| `plan_create` | ✗ | ✗ | ✓ |
| `plan_update` | ✓ | ✓ | ✓ |
| `exit_plan_mode` | ✗ | ✗ | ✓ |

---

### [P2] ~~`memory/file` `countLinesLocked` hits `bufio.Scanner` default 64KB cap — long messages cause session amnesia / append deadlock~~ ✅ FIXED

[memory/file/memory.go](memory/file/memory.go), [memory/file/memory_test.go](memory/file/memory_test.go): **Fixed in this commit** — `countLinesLocked` now counts by newline via `bufio.Reader.ReadString('\n')` instead of `bufio.Scanner`. A Scanner inherits the 64KB default token cap and returns `bufio.ErrTooLong` on any single line exceeding it; `ReadString` chunks oversized lines and only counts a record on the terminating `'\n'` (which `Append` always writes), so `Count` is no longer capped at any fixed buffer size. 6 unit tests in `memory/file/memory_test.go` cover all four failure modes below plus the 2MB `(>1MB readAllLocked cap)` and partial-trailing-line edge cases — verified to fail on the old code (`bufio.Scanner: token too long`) and pass after the fix.

Original issue: `countLinesLocked` used `bufio.NewScanner(f)` without `scanner.Buffer(...)`, so it inherited the stdlib default `bufio.MaxScanTokenSize = 64*1024` (64KB). The sibling write path (`Append`, no limit) and read path `readAllLocked` (explicit 1MB cap) were unaffected — a single JSON message > 64KB could be written and read back, **only "count lines" returned `bufio.ErrTooLong`**. Trigger threshold was low: one assistant message embedding a large artifact (a `read` of a big single-line file, a `grep` full-repo hit, a base64 screenshot, an SQL dump) sufficed. `Compact` never deleted original messages, so the oversized line **persisted permanently** — one trigger became a chronic condition.

Note: `cli serve` (REST + ACP) uses `memory/sqlite` (whose `Count` is `SELECT COUNT(*)` — no scanner, no cap); `file` memory was only reached via `examples/iac`, `examples/memory`, and downstream embedded users, so the main product surface was unaffected. Side-by-side repro (`file` vs `sqlite` over the same >64KB message set) confirmed sqlite never failed; file now matches.

Impact (all four reproduced against the real `*file.Memory` and now fixed):

1. **Silent amnesia mid-run** — `prepareMemory` (`runner.go:521`) got `ErrTooLong` from `Count`, returned `nil, ci` without fataling; the main loop continued with **zero history** for the turn. No error surfaced to the user. Compaction/summarizer also stopped firing.
2. **Restart append deadlock** — `Append` (`memory.go:91-97`) seeded `nextIdx` via `countLinesLocked` on first use. Once the file held a >64KB line, the first `Append` after restart errored, leaving `nextIdx` at 0; **every subsequent `Append` re-entered the `==0` branch and failed again**. `appendMemory` (`runner.go:1013-1024`) was void and only observed, so the in-memory conversation kept running while all new messages were silently dropped.
3. **Count/Recent split-brain** — for 64KB < line ≤ 1MB, `readAllLocked` succeeded but `Count` errored. `globalOffset := totalCount - len(msgs)` (`runner.go:538`) went negative, skewing compaction and indexing.
4. **Error-swallowing callers made sessions "vanish"** — `acp/server.go:467`, `rest/session.go:188/208`, and `rest/team_memory.go:77-80` discarded `Count` errors, treating them as `n=0`. A session with messages was judged empty and disappeared from REST lists / ACP replay (file still present, `Recent` still readable). With `Count` no longer erroring on oversized lines, these callers now see the true count.

Historical repro (now passes after the fix):
1. Start an agent with file memory (`examples/iac` or an embedded path).
2. Have the agent `read`/`grep` a single-line file > 64KB.
3. Next turn: agent forgets the conversation (empty history).
4. Restart: new inputs to that session fail to persist (`file memory append: bufio.Scanner: token too long`) until the oversized line is manually split.

---

## 🔧 Workarounds

### `runner.go` — Emergency context window trimming

[runner.go:222-253](runner.go): "last-resort truncation" triggers when system prompts + compressed context + large tool results push past the model's hard limit. The new `estimatePromptOverhead` in `prepareMemory` accounts for fixed overhead tokens, so this path should now only fire when tool results are unexpectedly large. Still a valid safety net.

---

### `team.go` — Handoff hint retry for forgetful models

[team.go](team.go): Agent has handoff tools but doesn't use them → retry with hardcoded prompt. Root cause in the model, not the framework.

---

### `runner.go` — Fragile history dedup

[runner.go:148-149](runner.go): `appendMemory(input)` → `Recent()` → strip last `RoleUser` message. If concurrent access inserts another user message between Append and Recent, wrong message is removed.

---

### `guard/llm/guard.go` — Substring matching for safety verdict

[guard/llm/guard.go](guard/llm/guard.go): Looking for `"allowed": false` and `"allowed": true` as substrings in a lowercased string. Can produce false positives on edge cases.

---

## 💣 Technical Debt

### [DEBT] `runner.go:58-403` — Monolithic `run()` loop

[runner.go](runner.go): The entire 8-node mainline loop is one function (~400 lines). Unit testing individual stages impossible without mocking the entire loop. File ~1383 lines total (grew with subagent + two-phase executeTools + estimatePromptOverhead).

---

### [DEBT] `prompt.go:34` — `PromptBuilder` is a function type, not an interface

[prompt.go](prompt.go): `type PromptBuilder func(context.Context, PromptInput) ([]Message, error)`. Cannot add methods. Zero value panics.

---

### [DEBT] `memory.go:62-66` — `ThroughIndex` zero value semantically overloaded

[memory.go](memory.go): `ThroughIndex = 0` means either "never compressed" or "first compaction covered 0 messages." With the summarizer now implemented (summarizer/llm.go), the distinction matters more.

---

### [DEBT] `runner.go` — `prepareMemory` `overflow` variable semantic confusion

[runner.go](runner.go): `overflow` starts as `len(msgs)`, becomes a keep-from index, expanded by `SafeCompressionBoundary`, then used as both compaction cutoff AND trim keep-point.

---

### [DEBT] `agent.go` — `RunGoal`: goal text duplicated in prompt

[agent.go](agent.go): Goal injected into system instructions AND passed as first `UserMessage(goal)`. Same text appears twice.

---

### [DEBT] `team.go` — Lock-release-external-relock TOCTOU pattern

[team.go](team.go): Each window has explicit nil/orphan checks, but the pattern is fragile throughout.

---

### [DEBT] `router.go` — `containsWord` is `strings.Contains`, no word-boundary matching

[router.go](router.go): `"I don't think billing is appropriate"` matches agent `"billing"`.

---

### [DEBT] `session.go` — `Session` passed by value, mutations invisible to caller

[session.go](session.go): Runner sets `session.Turn = turn` on a struct copy. Caller never sees updated turn count.

---

### [DEBT] Pervasive silent error suppression

Errors discarded without logging:

| File | What's lost |
|------|-------------|
| [rest/team_memory.go](rest/team_memory.go) | `Recent()` errors from shared/private memory |
| [rest/team_memory.go](rest/team_memory.go) | `Count()` errors |
| [memory/file/memory.go](memory/file/memory.go) | Corrupt JSON lines silently skipped |
| [memory/sqlite/memory.go](memory/sqlite/memory.go) | Vector scan row errors silently skipped |
| [runner.go](runner.go) | `Memory.Compact()` errors → silent budget overflow |

---

### [DEBT] Hardcoded `/bin/bash` — not portable

[tool/shell.go](tool/shell.go) and [sandbox/native/native_linux.go](sandbox/native/native_linux.go): Breaks on NixOS, Alpine, macOS.

---

### [DEBT] `memory/file/memory.go` — Scanner buffer initialized with length 0

[memory/file/memory.go:292](memory/file/memory.go#L292): `scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)` — `bufio.Scanner` ignores 0-length buffer and allocates its own.

---

### [DEBT] Sandbox disabled by default — credential mounting unresolved when enabled

[sandbox/native/native_linux.go](sandbox/native/native_linux.go):
`bwrapArgs()` does not mount `$HOME` or credential directories (`~/.hcloud`, `~/.aws`, `~/.kube`, `~/.config/gcloud`, `~/.docker`). When the sandbox is enabled via `--sandbox`, cloud CLIs inside bwrap cannot read auth configs. Workaround: use `readable_paths` in `settings.json`, or fix bwrap to mount credential dirs automatically. Additionally, bwrap requires setuid or `newuidmap` to function on this host — without them it silently falls back to unconfined execution.

---

## Legend

| Tag | Meaning |
|-----|---------|
| `P0` | Critical — data loss, API contract violation, resource leak |
| `P1` | High — incorrect behavior in common scenarios |
| `P2` | Medium — incorrect behavior in edge cases |
| `P3` | Low — cosmetic or harmless |
| `DEBT` | Technical debt — will compound as codebase grows |

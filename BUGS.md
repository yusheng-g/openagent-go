# BUGS.md — Known Issues & Technical Debt

> Last updated 2026-07-21 (rev 7).
> Format: `[P0]` = critical, `[P1]` = high, `[P2]` = medium, `[P3]` = low.
> `[DEBT]` = technical debt (no immediate breakage, will compound).

---

## 🐛 Bugs

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

### [P2] Team: no model selection, no ContextWindow, stale ModelID

[rest/team.go](rest/team.go):

1. `handleChat` ignores `ChatRequest.ModelID` — `oaSession` constructed without `Model` or `ModelID`. Frontend model selector has no effect in team mode.
2. `s.info.ModelID` never synced to team handler. `GET /team/sessions/{id}` returns creation-time `ModelID` forever.
3. `handleGetSession` missing `ContextWindow` — Frontend shows `contextWindow: 0` for team sessions.

---

### [P2] `fetchMessages` can overwrite live SSE stream

[examples/frontend/vue-app/src/stores/chat.ts](examples/frontend/vue-app/src/stores/chat.ts): `watchEffect` → `clearChat()` → `fetchMessages()` (async). Between `clearChat()` and `fetchMessages` completing, a live SSE stream can push messages. When `fetchMessages` resolves, `messages.value = converted` unconditionally overwrites live messages.

---

### [P2] `guard/llm/guard.go` — Parse failure defaults to block (fail-closed)

[guard/llm/guard.go](guard/llm/guard.go): `parseResult` does substring match on `"allowed": true/false`. If substring match also fails, defaults to `Allowed: false`. The `failOpen` option only covers network/API errors and empty choices, not parse errors (`parseResult` ignores `failOpen` when it can't extract a boolean).

---

### [P2] Dynamic team agents not persisted across restarts

[rest/team.go](rest/team.go): `handleAddAgent` creates agents at runtime. `SessionStore` only persists `SessionInfo` — agent list not stored. After restart, `getOrCreate` → `newEntry` rebuilds from templates only.

---

### [P3] API credit leak on client SSE disconnect

[rest/handler.go:272](rest/handler.go#L272), [rest/team.go:222](rest/team.go#L222), [rest/orchestrate_handler.go:264](rest/orchestrate_handler.go#L264): All three use `context.Background()` (with long timeout) for agent goroutines. SSE client disconnects → goroutine continues running with no consumer.

---

### [P2] `cmd/cli/keyring` silently falls back to `MemStore` in D-Bus-less environments, losing persisted secrets

[cmd/cli/keyring/keyring.go:19-25](cmd/cli/keyring/keyring.go#L19),
[cmd/cli/main.go:215-222](cmd/cli/main.go#L215),
[cmd/cli/main.go:225-230](cmd/cli/main.go#L225),
[go.mod:16](go.mod#L16):

`Open()` probes the system keychain with `gkr.Get("openagent", "__probe__")`.
On Linux, `zalando/go-keyring v0.2.8` routes through the Secret Service API
(`github.com/godbus/dbus/v5`). When `DBUS_SESSION_BUS_ADDRESS` is unset and no
`dbus-launch` exists on `PATH`, `godbus` attempts to exec `dbus-launch` and
returns `exec: "dbus-launch": executable file not found in $PATH`. This error
propagates up to `openKeyring()`, which substitutes `MemStore` with a WARNING.
Repro: Huawei Cloud EulerOS 2 container (aarch64, no `dbus-x11` installed).

Derived issues:

1. Silent data loss: `cli keyring set key value` succeeds with exit code 0 in
   the fallback path, but secrets are written to a per-process `MemStore` and
   evaporate on process exit — no ERROR is surfaced.
2. Double WARNING: `main()` calls `openKeyring()` once at startup, then each
   `keyring{set,get,delete}` subcommand calls it again, printing the same
   warning twice per invocation.
3. Brittle probe strategy: `Open()` distinguishes "backend unavailable" from
   "key does not exist" solely by the error of a single `Get`. Backend
   initialization failures are indistinguishable from missing-key states.

Implementation plan (Plan A, keeping `zalando/go-keyring`, modeled on
`~/projects/hdspace-models/credential`):

1. Promote existing indirect deps to direct: `github.com/godbus/dbus/v5`,
   `golang.org/x/sys` (both already in `go.mod` as `// indirect`).
2. In `cmd/cli/keyring/keyring.go`, replace the `gkr.Get("__probe__")` probe
   with an explicit availability check:
   - Linux: `dbus.SessionBus()` (catching the autolaunch error instead of
     triggering `dbus-launch`), then `NameHasOwner("org.freedesktop.secrets")`
     via `org.freedesktop.DBus` → `/org/freedesktop/DBus`. Mirrors
     `isSecretServiceAvailable()` in `credential_linux.go`.
   - macOS / Windows: probe via `zalando` as today (keychain backends there do
     not depend on D-Bus).
3. Add a Linux kernel-keyring fallback (`KeyCtlBackend` equivalent) implemented
   directly on `golang.org/x/sys/unix`:
   - `KeyctlLink(KEYCTL_LINK, KEY_SPEC_USER_KEYRING, KEY_SPEC_SESSION_KEYRING, 0, 0)`
     to attach the user keyring to the session keyring (matches
     `ensureKeyringLinked()`).
   - Store secrets under `user:openagent:<service>:<key>` key descriptors;
     values are base64-encoded (parity with `hdspace-models` secret blob
     encoding) to survive binary payloads safely.
   - `Get` uses `KeyctlSearch(KEY_SPEC_USER_KEYRING, "user", keyDesc, 0)` then
     `KeyctlRead`; `Set` uses `KeyctlSet`; `Delete` uses `KeyctlUnlink`.
4. Introduce `func HasSupport() bool` on `Store` (or package-level) so callers
   can explicitly detect loss of persistence instead of inheriting `MemStore`
   silently. Modeled on `HasCredentialSupport()` in `credential.go:46-48`.
5. `cmd/cli/main.go`:
   - `openKeyring()` returns a sentinel `ErrKeyringUnavailable` when neither
     Secret Service nor KeyCtl is usable; `main()` initializes a single
     global `Store` (or `MemStore`) once and shares it with all subcommands.
   - `cli keyring set` / `cli keyring delete` in the `MemStore` fallback path
     `log.Fatalf` with a clear message ("no keyring backend available: install
     `dbus-x11` or run with `--cap-add=keyutils`") rather than silently
     succeeding. `cli keyring get` may still return "" for read-only callers
     like `serve`.
6. macOS / Windows code paths unchanged: `zalando`'s `keychain`/`wincred`
   backends do not invoke D-Bus.

SCOPE NOTES (per user direction):
- keyring library is NOT swapped to `99designs/keyring`; `zalando/go-keyring`
  stays.
- The double-WARNING issue (item 2 above) is recorded for context only and is
  NOT being fixed in this change.
- File-based fallback (Plan B) is rejected; kernel keyring is the last
  non-volatile tier, and a missing kernel-keyring capability surfaces as an
  explicit error.

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

### [P1] Skills not recognized or usable in `cli serve` (ACP + REST modes)

[cmd/cli/server/acp.go:62-66](cmd/cli/server/acp.go),
[cmd/cli/server/http.go:53-59](cmd/cli/server/http.go),
[runner.go:72-77](runner.go),
[runner.go:706-725](runner.go),
[runner.go:731-735](runner.go),
[prompt.go:18-19](prompt.go),
[cmd/cli/config/config.go:10-22](cmd/cli/config/config.go):

Both CLI server entry points construct the agent **without**
`openagent.WithSkillLoader(...)`, so `agent.SkillLoader` stays nil.
The runner then short-circuits the entire skill subsystem:

```go
// runner.go:72-77
if r.agent.SkillLoader != nil {           // ← false in cli serve
    skills, _ := r.agent.SkillLoader.Discover(ctx)
    r.skills = skills
    r.loadedSkills = make(map[string]string)
    r.builtinTools = builtinSkillToolDefs() // use_skill / reload_skills
}
```

With the loader nil, three things vanish simultaneously:

1. **No `use_skill` / `reload_skills` tools** — `r.builtinTools` never
   receives them, so `buildModelRequest` (`runner.go:731-735`) never
   offers them to the model. The agent has no mechanism to load a skill.
2. **No "## Available Skills" catalog in the system prompt** —
   `defaultBuildPrompt` (`runner.go:706-718`) emits the catalog only
   when `len(input.AvailableSkills) > 0`; `r.skills` is nil so
   `input.AvailableSkills` stays empty.
3. **No "## Loaded Skill:" body** — same gate at `runner.go:720-725`,
   `input.LoadedSkills` is empty.

**This is NOT a prompt bug.** The prompt layer (`defaultBuildPrompt`)
correctly reflects the (empty) skill state handed to it. The root cause
is one level up: the CLI server never wires a `SkillLoader` into the
agent, so the runner never discovers skills in the first place.

Cross-confirmation: `WithSkillLoader` is called only in
`cmd/iac-mcp/main.go:198`, `examples/skill/main.go:39`, and
`examples/iac/agents.go:123` — never under `cmd/cli/server/`. The
`Config` struct (`cmd/cli/config/config.go:10-22`) has no skills
directory field at all, so users cannot configure one via
`settings.json` even if they wanted to.

Repro:
1. `./openagent-cli serve --acp`
2. Place a `SKILL.md` under `~/.openagent/skills/my-skill/`
3. Prompt the agent with a task that the skill covers
4. Observe: no `use_skill` tool call, no skill catalog in the system
   prompt (visible via guard/observer logs), agent behaves as if the
   skill does not exist.

Notes:
- `Agent.Clone()` (`agent.go:64-71`) is a shallow `*a` copy, so the
  `SkillLoader` interface field propagates to the per-turn clone in
  `acp/server.go:1120` (`agentForTurn`) and to the channel agent clone
  in `acp.go:82`. Wiring the loader once on the template agent is
  sufficient — no changes needed in `agentForTurn`.
- The REST server (`http.go`) has the same gap; this is not ACP-specific.

Proposed fix (not yet applied):
1. `cmd/cli/config/config.go` — add `Skills string` field
   (`json:"skills,omitempty"`), default `".openagent/skills"`.
2. `cmd/cli/server/shared.go` — add `resolveSkillLoader(skills string)
   openagent.SkillLoader` helper mirroring `resolveProfileFile`: probe
   `$(pwd)/$(skills)/` then `~/$(skills)/`, return `fs.New(dir)` when
   the directory exists, else nil.
3. `cmd/cli/server/acp.go` and `cmd/cli/server/http.go` — add
   `openagent.WithSkillLoader(resolveSkillLoader(cfg.Skills))` to both
   `NewAgent` calls.

---

### [P2] `plan_create` available in all modes, bypassing plan-mode workflow

[acp/server.go:855-864](acp/server.go#L855), [acp/server.go:866-885](acp/server.go#L866), [plan/tool.go:16-116](plan/tool.go):

`plan_create` is registered unconditionally in `OnPrompt` — available in auto, manual, and plan modes. The agent in auto/manual mode can call `plan_create` to produce a structured plan AND immediately begin executing it within the same turn, bypassing the "enter plan mode → create plan → user review → exit plan mode → execute" workflow that plan mode was designed for.

This conflates planning and execution: the plan is created while execution tools remain active, so the agent can skip user review entirely. The symmetry is also broken — `exit_plan_mode` has no `enter_plan_mode` counterpart, so agents in auto/manual mode cannot proactively switch to plan mode when they detect a complex task.

Current vs intended tool availability:

| Tool | auto (current→target) | manual (current→target) | plan |
|------|:---:|:---:|:----:|
| `enter_plan_mode` | ✗→✓ | ✗→✓ | ✗ |
| `plan_create` | ✓→✗ | ✓→✗ | ✓ |
| `plan_update` | ✓→✓ | ✓→✓ | ✓ |
| `exit_plan_mode` | ✗→✗ | ✗→✗ | ✓ |

**Proposed fix:**
1. Add `enter_plan_mode` tool to `plan/tool.go` (symmetrical to existing `exit_plan_mode`).
2. Move `plan_create` registration inside the `if ss.mode == "plan"` block in `OnPrompt`.
3. Register `enter_plan_mode` in the else branch (auto/manual modes).
4. Update `buildDynamicContext` to hint auto/manual agents about `enter_plan_mode` for complex tasks.
5. Cross-turn approach: `enter_plan_mode` calls `setSessionMode("plan")` to persist the mode change; the next `OnPrompt` turn picks up plan mode and registers `plan_create` + `exit_plan_mode`. This avoids the complexity of removing execution tools from the agent clone mid-turn.

**Approval behavior:** `enter_plan_mode` inherits the current mode's approver automatically — in auto mode it runs without approval; in manual mode it triggers `acpApprover.requestPermission` for user confirmation. No extra wiring needed.

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

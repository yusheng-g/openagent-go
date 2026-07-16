# BUGS.md — Known Issues & Technical Debt

> Last updated 2026-07-15.
> Format: `[P0]` = critical, `[P1]` = high, `[P2]` = medium, `[P3]` = low.
> `[DEBT]` = technical debt (no immediate breakage, will compound).

---

## 🐛 Bugs

### [P1] SQLite SessionStore missing `provider` column — provider lost across restarts

[rest/sessionstore/sqlite/sqlite.go](rest/sessionstore/sqlite/sqlite.go): Schema had no `provider` column. INSERT only had 6 columns (id, kind, title, model_id, created_at, updated_at). SELECT similarly omitted provider. After restart, `SessionInfo.Provider` always empty string. Frontend model selector shows wrong provider label for restored sessions.

**Fix:** Add `provider TEXT NOT NULL DEFAULT ''` to schema, include in all INSERT/SELECT/scan paths.

**Status:** ✅ Fixed 2026-07-15.

---

### [P2] File memory never sets `Message.Index` — team cross-partition ordering broken

[memory/file/memory.go](memory/file/memory.go): `readAllLocked` deserializes messages without assigning `msg.Index`. All file-backed messages have `Index: 0`. Team mode's `sort.Slice(msgs, func(i, j int) bool { return msgs[i].Index < msgs[j].Index })` degenerates to unstable order. SQLite sets `msg.Index = id` (auto-increment PK); file memory has no equivalent.

**Fix:** Track a per-session monotonic counter and set `msg.Index` in `readAllLocked`.

**Status:** ✅ Fixed 2026-07-15.

---

### [P2] `handler.go` nil model fallback clears provider/modelID

[rest/handler.go#L229-L234](rest/handler.go#L229-L234): When `lookupModel` returns nil, the fallback clears `provider = ""` and `modelID = ""`. These empty values get persisted via `withMeta` → `syncMeta`, permanently losing the session's model preference.

**Fix:** Don't clear provider/modelID on fallback — just use the default model for this run.

**Status:** ✅ Fixed 2026-07-15.

---

### [P2] Team/Plan: no model selection, no ContextWindow, stale ModelID

Three related gaps — team and plan handlers never got the model-registry treatment that the single handler has:

1. **Team `handleChat` ignores `ChatRequest.ModelID`** — [rest/team.go#L297-L314](rest/team.go#L297-L314): The `oaSession` is constructed without `Model` or `ModelID` fields. The frontend sends `modelId` in the chat body, but the team runner always uses each agent's built-in model. The model selector in the frontend has zero effect in team mode.

2. **`s.info.ModelID` never synced** — [rest/handler.go#L352-L354](rest/handler.go#L352-L354): Single handler syncs both `s.modelID` and `s.info.ModelID`. Team and Plan handlers have no equivalent. `GET /team/sessions/{id}` and `GET /plan/sessions/{id}` return the creation-time `ModelID` forever.

3. **Team/Plan `handleGetSession` missing `ContextWindow`** — [rest/handler.go#L190-L192](rest/handler.go#L190-L192): Single handler sets `ContextWindow` from `s.agent.Model.ContextWindow()`. Team [rest/team.go#L164](rest/team.go#L164) and Plan [rest/plan_handler.go#L165](rest/plan_handler.go#L165) don't set it — frontend shows `contextWindow: 0` for team/plan sessions.

**Fix:** Wire `ChatRequest.ModelID` into team `oaSession`; sync `s.info.ModelID` after resolution; set `ContextWindow` in all three `handleGetSession` variants.

---

### [P2] `fetchMessages` can overwrite live SSE stream

[examples/frontend/vue-app/src/stores/chat.ts#L388](examples/frontend/vue-app/src/stores/chat.ts#L388) and [TeamAgentView.vue#L32-L38](examples/frontend/vue-app/src/views/TeamAgentView.vue#L32-L38): `watchEffect` calls `clearChat()` → `fetchMessages()` (async). Between `clearChat()` and `fetchMessages` completing, a live SSE stream can push messages into the array. When `fetchMessages` resolves, `messages.value = converted` unconditionally overwrites the live messages, causing them to vanish from the UI.

**Fix:** Skip overwriting `messages.value` when `streaming.value` is true; or merge fetched messages with any that arrived in-flight.

---

### [P2] `sandbox/native/native_linux.go` — Silent fallback to unconfined when bwrap missing

[sandbox/native/native_linux.go#L18-L20](sandbox/native/native_linux.go#L18-L20): If `bwrap` is not installed, `confineAndRun` falls through to `unconfinedRun` **with no warning logged**. The comment at line 16 says "falls back to unsandboxed execution with a warning" — but no log or warning is emitted anywhere in the code path. Commands run without any sandbox, silently.

---

### [P2] `guard/llm/guard.go` — Parse failure defaults to block (fail-closed)

[guard/llm/guard.go#L126-L134](guard/llm/guard.go#L126-L134): JSON parse fails → substring match on `"allowed": true/false`. If the substring match also fails (malformed enough), defaults to `Allowed: false`. The `failOpen` option only covers network/API errors (line 100) and empty choices (line 110), not parse errors. A slightly malformed-but-safe response gets blocked.

---

### [P2] Dynamic team agents not persisted across restarts

[rest/team.go#L271-L327](rest/team.go#L271-L327): `handleAddAgent` creates agents at runtime and adds them to the in-memory `s.team`. `SessionStore` only persists `SessionInfo` — the agent list is not stored. After restart, `getOrCreate` → `newEntry` rebuilds the team from `h.agents` templates only. Dynamically added agents (name, description, instructions) are lost.

**Fix:** Requires schema change — persist agent definitions alongside session metadata (separate `session_agents` table or metadata column). Needs design discussion.

---

### [P2] Plan mode agents not wired to handler memory

[rest/plan_handler.go](rest/plan_handler.go): `cloneAgentForPlan` used `tmpl.Memory` (from the template agent), ignoring the handler's configured memory. Plan steps executed without persistence. `handlePlanMessages` returns empty because messages are stored under step sub-sessions (`sessionID/steps/stepID`), not the parent session ID.

**Status:** Agent memory wiring fixed 2026-07-15. `handlePlanMessages` still returns empty — plan UI uses SSE events, not message history.

---

### [P3] API credit leak on client SSE disconnect

[rest/handler.go#L245-L268](rest/handler.go#L245-L268), [rest/team.go#L190-L207](rest/team.go#L190-L207), [rest/plan_handler.go#L238](rest/plan_handler.go#L238): All three handlers use `context.Background()` (with timeout) for agent/team/plan goroutines. When the SSE client disconnects mid-stream, the goroutine continues executing tool calls and model requests for up to 5 minutes (30 for plan), consuming API credits with no consumer.

**Fix:** Derive the agent context from the request context with `context.WithCancel(r.Context())`, or wire cancellation from the SSE write loop to the agent goroutine.

---

## 🔧 Workarounds

### `runner.go:221-237` — Emergency context window trimming

[runner.go#L221-L237](runner.go#L221-L237): Comment explicitly says "last-resort truncation" that triggers "only when system prompts, compressed context, or large tool results push it past the hard limit." **If compaction were reliable, this emergency path would never exist.** It's a safety net admitting the compaction pipeline is not fully trusted.

---

### `team.go:484-498` — Handoff hint retry for forgetful models

[team.go#L484-L498](team.go#L484-L498): When an agent has handoff tools but doesn't use them, retry once with a hardcoded English prompt urging handoff. Workaround for models that "forget" to use transfer tools. Root cause is in the model (not respecting function calling instructions), not the framework. Hint is provider-specific — different model families may need different strategies. Currently works well with DeepSeek.

---

### `runner.go:146-148` — Fragile history dedup

[runner.go#L146-L148](runner.go#L146-L148): `appendMemory(input)` is called, then `Recent()` fetches history, then the last `RoleUser` message is stripped from history (assumed to be the input just appended). If any concurrent session access inserts another `RoleUser` message between the Append and the Recent fetch, the wrong message is removed.

---

### `hooks/otel/hooks.go:32-41,75` — OTEL spans ended immediately (no parent-child nesting)

[hooks/otel/hooks.go#L32-L41](hooks/otel/hooks.go#L32-L41): `OnAgentStart` creates a span and calls `span.End()` immediately. Same for `OnToolStart`. The `RunHooks` interface provides no mechanism to pass state between paired `OnXxxStart`/`OnXxxEnd` calls, so they create independent spans with no parent-child relationship. See [DEBT] below.

---

### `acp/client.go:461-486` + `acp/server.go:460-485` — JSON round-trip for type conversion

[acp/client.go#L461-L486](acp/client.go#L461-L486): SDK types ↔ project types are converted via `json.Marshal` → `json.Unmarshal`. Mismatched JSON tags = silent data loss. Workaround while waiting for SDK stabilization.

---

### `guard/llm/guard.go:126-134` — Substring matching for safety verdict

[guard/llm/guard.go#L126-L134](guard/llm/guard.go#L126-L134): Looking for `"allowed": false` and `"allowed": true` as substrings in a lowercased string. Can produce false positives (e.g. content containing the literal string `"allowed": false` in an explanation).

---

## 💣 Technical Debt

### [DEBT] `runner.go:58-403` — 345-line monolithic `run()` loop

[runner.go#L58-L403](runner.go#L58-L403): The entire 8-node mainline loop (memory fetch, prompt build, guard checks, model calls, tool execution, stream events, retry, EndTurn) is one function. **Unit testing individual stages is impossible without mocking the entire loop.** File is now 1230 lines total.

---

### [DEBT] `hooks.go` — `RunHooks` interface: paired Start/End methods can't share state

[hooks.go](hooks.go): `OnAgentStart`/`OnAgentEnd` and `OnToolStart`/`OnToolEnd` are paired but have no mechanism to pass state between them. OTEL hooks create independent spans; slog hooks discard duration timers.

**Fix:** Return an opaque token from `Start` methods that is passed to corresponding `End` methods.

---

### [DEBT] `prompt.go:35` — `PromptBuilder` is a function type, not an interface

[prompt.go#L35](prompt.go#L35): `type PromptBuilder func(context.Context, PromptInput) ([]Message, error)`. Cannot add methods later. Zero value panics on call.

---

### [DEBT] `memory.go:66-68` — `ThroughIndex` zero value is semantically overloaded

[memory.go#L66-L68](memory.go#L66-L68): `ThroughIndex = 0` means either "never compressed" or "first compaction covered 0 messages." Logic cannot distinguish.

---

### [DEBT] `model.go:112` — `Summarizer.ThroughIndex` contract unenforceable

[model.go#L112](model.go#L112): Comment says ThroughIndex "should be set by the caller (memory backend), not the implementation." Nothing prevents a buggy `Summarizer` from setting it arbitrarily.

---

### [DEBT] `runner.go:381-431` — `prepareMemory` `overflow` variable semantic confusion

[runner.go#L381-L431](runner.go#L381-L431): `overflow` starts as `len(msgs)`, becomes a keep-from index, is expanded by `SafeCompressionBoundary`, then used as both compaction cutoff AND trim keep-point.

---

### [DEBT] `agent.go:164-170` — `RunGoal`: goal text duplicated in prompt

[agent.go#L164-L170](agent.go#L164-L170): Goal is injected into system instructions AND passed as the first `UserMessage(goal)`. Same text appears twice in the message list.

---

### [DEBT] `team.go` — Lock-release-external-relock TOCTOU pattern throughout

[team.go](team.go) lines 211-216, 246-250, 385-390, 521-523, 747-749: Each window has explicit nil/orphan checks, but the pattern is fragile.

---

### [DEBT] `router.go:116-118` — `containsWord` is `strings.Contains`, no word-boundary matching

[router.go#L116-L118](router.go#L116-L118): Plain substring search. `"I don't think billing is appropriate"` matches an agent named `"billing"`.

---

### [DEBT] `session.go:112` — `Session` passed by value, mutations invisible to caller

[session.go#L112](session.go#L112): Runner sets `session.Turn = turn` on a **struct copy**. Caller never sees updated turn count.

---

### [DEBT] Pervasive silent error suppression (partial — handler.go, runner.go, plan/executor.go done; memory layer deferred)

Errors discarded without logging across the codebase:

| File | Line | What's lost |
|------|------|-------------|
| [rest/team_memory.go](rest/team_memory.go#L55-L56) | 55-56 | `Recent()` errors from shared/private memory |
| [rest/team_memory.go](rest/team_memory.go#L72-L74) | 72-74 | `Count()` errors |
| [memory/file/memory.go](memory/file/memory.go#L282) | 282 | Corrupt JSON lines silently skipped |
| [memory/sqlite/memory.go](memory/sqlite/memory.go#L398) | 398 | Vector scan row errors silently skipped |
| [runner.go](runner.go#L422) | 422 | `Memory.Compact()` errors → silent budget overflow |

---

### [DEBT] Hardcoded `/bin/bash` — not portable

[tool/shell.go#L105](tool/shell.go#L105) and [sandbox/native/native_linux.go](sandbox/native/native_linux.go): `/bin/bash` is hardcoded. Breaks on NixOS, Alpine containers, macOS.

---

### [DEBT] `plan/planner.go:134-143` — Markdown fence stripping is fragile

[plan/planner.go#L134-L143](plan/planner.go#L134-L143): Fence stripping uses `strings.TrimPrefix` which only matches exact prefixes. Prompt already asks for "No markdown fences" — the parsing is defensive but fragile.

---

### [DEBT] `plan/plan.go:247` — `ReplanWithFeedback` passes `nil` onChunk ambiguously

[plan/plan.go#L247](plan/plan.go#L247): `ReplanInput` struct's `onChunk` field is left nil, relying on the planner to check the **function parameter** for nil. Works correctly but is fragile.

---

### [DEBT] `memory/file/memory.go:272` — Scanner buffer initialized with length 0

[memory/file/memory.go#L272](memory/file/memory.go#L272): `scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)` — buffer has length 0, so `bufio.Scanner` ignores it and allocates its own. Capacity is wasted.

---

## ✅ Recently Fixed

| Issue | Commit |
|-------|--------|
| [P1] Team `handleListMessages` only reads shared memory | Fixed in `refactor(rest): sessionManager generics` — aggregates private partitions + sorts by Index |
| [P1] SQLite SessionStore missing `provider` column | `fix(rest): sqlite SessionStore missing provider column — lost across restarts` |
| [P2] File memory never sets `Message.Index` | Per-session monotonic counter, `msg.Index = idx` in `readAllLocked` |
| [P2] `handler.go` nil model fallback clears provider/modelID | No longer clears provider/modelID on fallback |
| [P2] Plan mode agents not wired to handler memory | `cloneAgentForPlan` now accepts `mem` param, prefers handler memory over template |
| [P3] Vite proxy missing `/team` and `/plan` paths | Now proxied at [vite.config.ts:17-18](examples/frontend/vue-app/vite.config.ts#L17-L18) |
| [P0] `rowToMessage` missing `ReasoningContent` assignment | Included in `feat(rest): model registry` batch |
| [P2] Compaction invisible (empty mfDetail) | `compactionInfo` struct carries count/from/to/summary |
| [P2] Summarizer losing facts across incremental passes | APPENDER prompt: copy existing verbatim, only append |
| [P2] `handleUpdateSession` race condition | Wrapped in `s.mu.Lock()`/`Unlock()` |
| [P2] Session mode cross-contamination | Three separate refs: singleSessions, teamSessions, planSessions |
| [P2] `r.runModel = r.runModel` self-assignment | Fixed to `r.agent.Model` |
| [P3] `modelID = "default"` sent to API | Changed to `modelID = ""` so Model instance decides |
| [P2] Team/Plan returning bare `SessionInfo` instead of `SessionDetail` | Both handlers return `SessionDetail` now |

## Legend

| Tag | Meaning |
|-----|---------|
| `P0` | Critical — data loss, API contract violation, resource leak |
| `P1` | High — incorrect behavior in common scenarios |
| `P2` | Medium — incorrect behavior in edge cases |
| `P3` | Low — cosmetic or harmless |
| `DEBT` | Technical debt — will compound as codebase grows |

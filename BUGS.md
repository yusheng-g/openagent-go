# BUGS.md — Known Issues & Technical Debt

> Last updated 2026-07-19.
> Format: `[P0]` = critical, `[P1]` = high, `[P2]` = medium, `[P3]` = low.
> `[DEBT]` = technical debt (no immediate breakage, will compound).

---

## 🐛 Bugs

### [P1] ~~`cli serve` fails to start with "unexpected end of JSON input" when settings.json missing/empty and no plugins~~ ✅ FIXED

[cmd/cli/main.go:62,65-67](cmd/cli/main.go): **Fixed in this commit** — `settings` is normalized to `[]byte("{}")` when empty before the plugin loop, so both `CallInit` and the final `Unmarshal` always receive valid JSON.

Original issue: `settings = raw` (line 62) ended up as a nil or empty byte slice in three scenarios — (1) `settings.json` not created, (2) file exists but empty, (3) no settings plugin loaded → `json.Unmarshal(settings, &cfg)` (line 122) failed with `unexpected end of JSON input` and aborted startup. Repro: `./openagent-cli serve --acp` with no plugins + missing/empty settings.json.

Note: line 43 `json.Unmarshal(raw, &preCfg)` silently swallows the same empty-input error — non-fatal there (defaults to empty `Config{}`, falls back to `DefaultPluginsDir()`). Left untouched as this is the expected graceful-degradation behavior.

---

### [P1] `memory/sqlite` FTS — CJK search returns nothing

[memory/sqlite/memory.go](memory/sqlite/memory.go) — `migrate()` creates `messages_fts` with default `unicode61` tokenizer, which treats a run of CJK characters as one token. CJK queries match nothing.

Punctuation (`! ? . , ; / #`) crash was partially mitigated in `ftsSearch`: characters are now stripped before calling `MATCH`, and empty queries return `nil` instead of crashing. Still, FTS5 only works for whitespace-separated languages (English, European).

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

## Legend

| Tag | Meaning |
|-----|---------|
| `P0` | Critical — data loss, API contract violation, resource leak |
| `P1` | High — incorrect behavior in common scenarios |
| `P2` | Medium — incorrect behavior in edge cases |
| `P3` | Low — cosmetic or harmless |
| `DEBT` | Technical debt — will compound as codebase grows |

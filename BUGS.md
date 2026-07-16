# BUGS.md — Known Issues & Technical Debt

> Last updated 2026-07-16.
> Format: `[P0]` = critical, `[P1]` = high, `[P2]` = medium, `[P3]` = low.
> `[DEBT]` = technical debt (no immediate breakage, will compound).

---

## 🐛 Bugs

### [P2] Team/Plan: no model selection, no ContextWindow, stale ModelID

Three related gaps — team and plan handlers never got the model-registry treatment:

1. **Team `handleChat` ignores `ChatRequest.ModelID`** — [rest/team.go](rest/team.go): `oaSession` constructed without `Model` or `ModelID`. Frontend model selector has no effect in team mode.

2. **`s.info.ModelID` never synced** — Team and Plan handlers don't sync model preference. `GET /team/sessions/{id}` and `GET /plan/sessions/{id}` return creation-time `ModelID` forever.

3. **Team/Plan `handleGetSession` missing `ContextWindow`** — Frontend shows `contextWindow: 0` for team/plan sessions.

**Fix:** Wire `ChatRequest.ModelID` into team session; sync `s.info.ModelID`; set `ContextWindow` in all three `handleGetSession` variants.

---

### [P2] `fetchMessages` can overwrite live SSE stream

[examples/frontend/vue-app/src/stores/chat.ts](examples/frontend/vue-app/src/stores/chat.ts): `watchEffect` → `clearChat()` → `fetchMessages()` (async). Between `clearChat()` and `fetchMessages` completing, a live SSE stream can push messages. When `fetchMessages` resolves, `messages.value = converted` unconditionally overwrites live messages.

**Fix:** Skip overwriting `messages.value` when `streaming.value` is true.

---

### [P2] `sandbox/native/native_linux.go` — Silent fallback to unconfined when bwrap missing

[sandbox/native/native_linux.go](sandbox/native/native_linux.go): If `bwrap` is not installed, `confineAndRun` falls through to `unconfinedRun` with **no warning**. Comment says "falls back with a warning" but no log is emitted.

---

### [P2] `guard/llm/guard.go` — Parse failure defaults to block (fail-closed)

[guard/llm/guard.go](guard/llm/guard.go): JSON parse fails → substring match on `"allowed": true/false`. If substring match also fails, defaults to `Allowed: false`. The `failOpen` option only covers network/API errors and empty choices, not parse errors.

---

### [P2] Dynamic team agents not persisted across restarts

[rest/team.go](rest/team.go): `handleAddAgent` creates agents at runtime. `SessionStore` only persists `SessionInfo` — agent list not stored. After restart, `getOrCreate` → `newEntry` rebuilds from templates only.

**Fix:** Needs schema design.

---

### [P2] Plan mode `handlePlanMessages` always empty

[rest/plan_handler.go](rest/plan_handler.go): Plan steps run in isolated sub-sessions (`sessionID/steps/stepID`). Messages stored under step sub-sessions, never under parent session. `Recent(ctx, parentSessionID, ...)` always returns empty.

---

### [P3] API credit leak on client SSE disconnect

[rest/handler.go](rest/handler.go), [rest/team.go](rest/team.go), [rest/plan_handler.go](rest/plan_handler.go): All three use `context.Background()` (with timeout) for agent goroutines. SSE client disconnects → goroutine continues running for up to 5 minutes (30 for plan), consuming credits with no consumer.

**Fix:** Derive agent context from `r.Context()` with `context.WithCancel()`.

---

## 🔧 Workarounds

### `runner.go` — Emergency context window trimming

[runner.go](runner.go): "last-resort truncation" triggers when system prompts + compressed context + large tool results push past the hard limit. If compaction were reliable, this emergency path would never exist.

---

### `team.go` — Handoff hint retry for forgetful models

[team.go](team.go): Agent has handoff tools but doesn't use them → retry with hardcoded prompt. Root cause in the model, not the framework. Works well with DeepSeek.

---

### `runner.go` — Fragile history dedup

[runner.go](runner.go): `appendMemory(input)` → `Recent()` → strip last `RoleUser` message. If concurrent access inserts another user message between Append and Recent, wrong message is removed.

---

### `acp/client.go` + `acp/server.go` — JSON round-trip for type conversion

[acp/client.go](acp/client.go): SDK types ↔ project types via `json.Marshal` → `json.Unmarshal`. Mismatched JSON tags = silent data loss.

---

### `guard/llm/guard.go` — Substring matching for safety verdict

[guard/llm/guard.go](guard/llm/guard.go): Looking for `"allowed": false` and `"allowed": true` as substrings in a lowercased string. Can produce false positives.

---

## 💣 Technical Debt

### [DEBT] `runner.go:58-403` — Monolithic `run()` loop

[runner.go](runner.go): The entire 8-node mainline loop is one function (~400 lines). Unit testing individual stages impossible without mocking the entire loop. File ~1380 lines total (grew from 1230 after subagent + two-phase executeTools).

---

### [DEBT] `prompt.go:35` — `PromptBuilder` is a function type, not an interface

[prompt.go](prompt.go): `type PromptBuilder func(context.Context, PromptInput) ([]Message, error)`. Cannot add methods. Zero value panics.

---

### [DEBT] `memory.go:66-68` — `ThroughIndex` zero value semantically overloaded

[memory.go](memory.go): `ThroughIndex = 0` means either "never compressed" or "first compaction covered 0 messages."

---

### [DEBT] `model.go:112` — `Summarizer.ThroughIndex` contract unenforceable

[model.go](model.go): Comment says ThroughIndex "should be set by the caller (memory backend), not the implementation." Nothing prevents a buggy Summarizer from setting it arbitrarily.

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

### [DEBT] `plan/planner.go` — Markdown fence stripping is fragile

[plan/planner.go](plan/planner.go): `strings.TrimPrefix` only matches exact prefixes.

---

### [DEBT] `plan/plan.go` — `ReplanWithFeedback` passes `nil` onChunk ambiguously

[plan/plan.go](plan/plan.go): Relies on planner to check function parameter for nil.

---

### [DEBT] `memory/file/memory.go` — Scanner buffer initialized with length 0

[memory/file/memory.go](memory/file/memory.go): `scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)` — `bufio.Scanner` ignores 0-length buffer and allocates its own.

## Legend

| Tag | Meaning |
|-----|---------|
| `P0` | Critical — data loss, API contract violation, resource leak |
| `P1` | High — incorrect behavior in common scenarios |
| `P2` | Medium — incorrect behavior in edge cases |
| `P3` | Low — cosmetic or harmless |
| `DEBT` | Technical debt — will compound as codebase grows |

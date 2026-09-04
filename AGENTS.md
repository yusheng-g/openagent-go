# AGENTS.md

Agent-facing notes for working in this repository.

## Repository map

- Binary entrypoint: `cmd/cli/main.go` — cobra root with subcommands `tui`
  (bubbletea), `serve` (REST+SSE, or `--acp` stdio JSON-RPC), `run`,
  `keyring`, `settings`, `cd`. `cmd/mcp/iac-server` is a separate binary
  (provider plugin architecture, incl. `huaweicloud`).
- Architecture (read before editing core logic): root package = core types
  (ToolResult, Message, RunHooks…); `kernel/` = the 8-node runtime loop;
  `agent/` = pure config. `DESIGN.md` is the authoritative doc; each
  `cmd/*/` has its own DESIGN.md.
- Built-in skills are shipped via `//go:embed` from `skills/builtin/` — no
  build tag, always included. Add a skill by dropping a directory there.
- `site/` is an independent Next.js static site (own package.json). CI
  deploys it to gh-pages and only triggers on `site/**` changes — Go-only
  commits never run it.

## Environment quirks

- `./openagent` at the repo root is a gitignored build artifact; `tmp/` is
  unversioned local scratch (TUI feature/pitfall notes — not a source of
  truth). `third_party/` holds vendored runtime deps (embedder models,
  onnxruntime).
- Config dir is `~/.<version.Name>/settings.json` (default
  `~/.openagent`) — a branded build (`OPENAGENT_BINARY_NAME=myagent`)
  reads `~/.myagent/` instead. `OPENAGENT_CLI_CONFIG` overrides the
  settings path entirely. Secrets go in the system keyring via
  `openagent keyring`.
- TUI is pure Go (bubbletea v2); Rust is only used for the optional WASM
  plugin PDK in `examples/plugin/`.

## Building

```sh
./build.sh                # → ./openagent (serve/run/keyring + tui subcommand)
```

`go build ./...`, `go vet ./...` and `go test ./...` must pass before
handing work back. TUI changes are verified end-to-end with `tu` (below),
not by piping stdin/stdout — bubbletea needs a real PTY.

## Terminal E2E verification with `tu` (terminal-use)

Some verification targets are interactive TUIs (`openagent tui`), which
cannot be driven by piping stdin — they need a real terminal. Use `tu`
(<https://github.com/flipbit03/terminal-use>): a headless PTY + terminal
emulator that can spawn the TUI, screenshot the rendered screen (text or
PNG), and drive keyboard/mouse.

Install (no Rust toolchain needed):

```sh
curl -fsSL https://raw.githubusercontent.com/flipbit03/terminal-use/main/install.sh | sh
# → ~/.local/bin/tu
```

Standard session conventions (keep screenshots comparable across runs):

- size `120x36`, TERM `xterm-256color`, COLORTERM `truecolor` (defaults)
- session name `oag`; always `tu kill --name oag` when done

Core commands:

```sh
tu run --name oag --size 120x36 --cwd /path ./openagent tui   # spawn
tu wait --name oag --stable 800 --timeout 8000                # screen settled
tu screenshot --name oag                                      # text (grep-able JSON)
tu screenshot --name oag --png --no-cursor --output shot.png  # PNG image
tu type --name oag "/models"                                  # literal text
tu press --name oag Enter Ctrl+P Escape                       # named keys
tu status --name oag                                          # pid, alive/exited
tu kill --name oag                                            # teardown
```

Verification recipe: spawn → `wait --stable` → assert on the **text**
screenshot's `content` field (plain `grep`; it is one JSON line) → drive
keys → `wait --stable` → assert again → Ctrl+C → confirm `status` reports
`exited`.

Caveats:

- The bundled PNG font has no CJK glyphs: Chinese text renders as tofu
  boxes in `--png` output. It renders fine in real terminals; for content
  assertions use the text screenshot, not the PNG.
- A prompt round-trip needs a configured provider (settings.json +
  keyring). UI smoke tests that must run anywhere should stay on screens
  that do not require a live model: welcome, slash sheet, `/models`,
  `/sessions`, Ctrl+P palette.

`scripts/tui-smoke.sh` wraps the standard no-model smoke flow — run it
after any TUI change:

```sh
scripts/tui-smoke.sh
```

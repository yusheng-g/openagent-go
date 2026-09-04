#!/usr/bin/env bash
# scripts/tui-smoke.sh — headless E2E smoke test for the TUI, driven by tu
# (terminal-use). Verifies the screens that render without a configured
# model backend: welcome header badge, the /models popup (rows, uniform
# highlight, no marker dots), the slash sheet, and a clean exit.
#
# Usage:
#   scripts/tui-smoke.sh                 # build + run the flow
#   SKIP_BUILD=1 scripts/tui-smoke.sh    # reuse an existing binary
#   TUI_SMOKE_BIN=./openagent scripts/tui-smoke.sh
#
# Requires: go (unless SKIP_BUILD=1), tu on PATH or ~/.local/bin.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

NAME="tui-smoke"
SIZE="${TUI_SMOKE_SIZE:-120x36}"
WORKDIR="$(mktemp -d)"
BIN="${TUI_SMOKE_BIN:-$WORKDIR/openagent-smoke}"
FAILED=""
trap 'tu kill --name "$NAME" >/dev/null 2>&1 || true; rm -rf "$WORKDIR"' EXIT

if ! command -v tu >/dev/null 2>&1 && [ ! -x "$HOME/.local/bin/tu" ]; then
    echo "tu not found. Install it:"
    echo "  curl -fsSL https://raw.githubusercontent.com/flipbit03/terminal-use/main/install.sh | sh"
    exit 1
fi
PATH="$HOME/.local/bin:$PATH"

if [ "${SKIP_BUILD:-0}" != "1" ]; then
    echo "==> Building"
    go build -o "$BIN" ./cmd/cli/
fi
BIN="${TUI_SMOKE_BIN:-$(ls "$WORKDIR"/openagent-smoke 2>/dev/null || echo "${TUI_SMOKE_BIN:?SKIP_BUILD=1 requires TUI_SMOKE_BIN}")}"

say() { echo "==> $*"; }
fail() { FAILED="$1"; echo "FAIL: $1"; tu screenshot --name "$NAME" || true; exit 1; }

shot() { # shot <desc> — capture the screen text; echoes it for grepping
    tu wait --name "$NAME" --stable "${2:-800}" --timeout 8000 >/dev/null 2>&1 || true
    local out
    out="$(tu screenshot --name "$NAME")"
    echo "$out" >"$WORKDIR/last.json"
    echo "$out"
}

expect() { # expect <desc> <needle>
    grep -q "$2" "$WORKDIR/last.json" || fail "$1: expected /$2/ on screen"
    say "ok: $1 (/$2/)"
}

refuse() { # refuse <desc> <needle>
    if grep -q "$2" "$WORKDIR/last.json"; then fail "$1: /$2/ must not appear"; fi
    say "ok: $1 (no /$2/)"
}

say "Spawning TUI in tu ($NAME, $SIZE, cwd=$WORKDIR)"
tu kill --name "$NAME" >/dev/null 2>&1 || true
tu run --name "$NAME" --cwd "$WORKDIR" --size "$SIZE" "$BIN" tui >/dev/null

say "Welcome screen"
shot welcome
expect "welcome" "Run /sessions to list, pin, and continue sessions"
expect "welcome" "0.0.1"                                       # status bar version
expect "welcome" "ctrl+p commands"                             # input footer hints

say "Slash command sheet"
tu type --name "$NAME" "/"
shot slash
expect "slash sheet" "Switch session"
refuse "slash sheet" "Unknown command"

say "Models popup"
tu press --name "$NAME" Escape
sleep 0.3 # Esc + "/" in one PTY read decodes as Alt+/, so let Esc settle
tu type --name "$NAME" "/models"
tu press --name "$NAME" Enter
shot models
expect "models popup" "Models"
expect "models popup" "esc close"
if ! grep -q "No model options" "$WORKDIR/last.json"; then
    # Rows are listed: the highlight must be the fill alone — no state dots
    # on list rows (toggle ○/● live in the command sheet, not here).
    refuse "models popup" "●"
fi

say "Exit"
tu press --name "$NAME" Escape
sleep 0.3
tu type --name "$NAME" "/exit"
tu press --name "$NAME" Enter
for _ in $(seq 1 20); do
    if tu status --name "$NAME" 2>/dev/null | grep -q '"exited"'; then break; fi
    if ! tu status --name "$NAME" >/dev/null 2>&1; then break; fi
    sleep 0.2
done
if tu status --name "$NAME" 2>/dev/null | grep -q '"alive": *true'; then
    fail "process did not exit after /exit"
fi
say "ok: process exited cleanly"

echo ""
if [ -n "$FAILED" ]; then
    echo "SMOKE FAILED: $FAILED"
    exit 1
fi
echo "TUI SMOKE PASSED"

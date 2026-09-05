#!/usr/bin/env bash
# build.sh — convenience build for the openagent-go binary.
#
# Produces <name> (Go CLI/server with the interactive TUI built in as the
# `tui` subcommand). Built-in skills are bundled via //go:embed (no build
# tag needed) and the binary is stripped (-s -w, smaller). The binary's
# identity is set via ldflags (version.Name / version.Version).
#
# The TUI is pure Go (bubbletea), no longer a separate Rust binary — run
# `./<name> tui` to launch it.
#
# Usage:
#   ./build.sh                         # name=openagent (default)
#   OPENAGENT_BINARY_NAME=myagent ./build.sh
#
# Requires: Go (any recent). No Rust toolchain needed for the TUI; Rust is
# only used for the optional WASM plugin PDK (see examples/plugin/).
set -euo pipefail

NAME="${OPENAGENT_BINARY_NAME:-openagent}"
VERSION="${OPENAGENT_VERSION:-$(git describe --tags --always 2>/dev/null || echo dev)}"

# Event tracking (ldflags-injected; empty = disabled).
EVENT_POST_URL="${OPENAGENT_EVENT_POST_URL:-}"
EVENT_APP_ID="${OPENAGENT_EVENT_APP_ID:-}"
SKIP_VERIFY="${OPENAGENT_SKIP_VERIFY:-false}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$REPO_ROOT"

echo "==> Building Go binary: $NAME (version $VERSION, embedded skills)"
# Built-in skills (skills/builtin/*) are bundled via //go:embed — no tag needed.
# -s -w: strip symbol table and DWARF debug info for a smaller binary.
go build \
         -ldflags "-s -w \
                   -X github.com/yusheng-g/openagent-go/version.Name=$NAME \
                   -X github.com/yusheng-g/openagent-go/version.Version=$VERSION \
                   -X github.com/yusheng-g/openagent-go/track.EventPostUrl=$EVENT_POST_URL \
                   -X github.com/yusheng-g/openagent-go/track.AppID=$EVENT_APP_ID \
                   -X github.com/yusheng-g/openagent-go/track.EventSkipVerify=$SKIP_VERIFY" \
         -o "$NAME" ./cmd/cli/

echo ""
echo "Built:"
echo "  $NAME    (serve/run/keyring + tui subcommand)"
echo ""
echo "Run:  $NAME tui"

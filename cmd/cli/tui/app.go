package tui

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	openacp "github.com/yusheng-g/openagent-go/acp/sdk"
	"github.com/yusheng-g/openagent-go/cmd/cli/config"
	"github.com/yusheng-g/openagent-go/cmd/cli/server"
	"github.com/yusheng-g/openagent-go/cmd/cli/tui/components"
	"github.com/yusheng-g/openagent-go/cmd/cli/tui/theme"
	"github.com/yusheng-g/openagent-go/cmd/cli/tui/views/chat"
	"github.com/yusheng-g/openagent-go/version"
)

// Mouse tracking is driven by hand instead of bubbletea's built-in modes.
// tea.MouseModeCellMotion emits \x1b[?1002h (button-event tracking), which
// forwards every button drag to the app and disables the terminal emulator's
// native text selection. Enabling \x1b[?1000h (normal tracking) instead
// still delivers wheel and click events — ultraviolet parses SGR mouse codes
// regardless of the rendered mode — but leaves drags free so the user can
// box-select text (e.g. to copy from the transcript). The tracking is
// enabled before the program starts rendering and reset right after it
// exits, so the mode never lingers in the shell.
const (
	mouseTrackingEnable  = "\x1b[?1000h\x1b[?1006h"
	mouseTrackingDisable = "\x1b[?1000l\x1b[?1006l"
)

// StartInteractiveTUI launches the fullscreen interactive TUI. It runs the
// ACP server in-process via os.Pipe (no subprocess), connects as an ACP
// client, and streams agent responses into the chat transcript.
//
// ctx is the parent context (from main.go's signal handler); the TUI derives
// a cancelable child so ctrl+c kills both the TUI and the ACP server.
// cfg provides everything: models, memory, capabilities, and the TUI section.
func StartInteractiveTUI(ctx context.Context, cfg config.Config) error {
	// Some environments (SSH sessions, tmux-derived TERMs, sanitized
	// sandboxes) lack a usable TERM; fall back to xterm-256color so
	// lipgloss renders colors and box drawing correctly. An existing
	// valid value is kept untouched.
	if os.Getenv("TERM") == "" {
		_ = os.Setenv("TERM", "xterm-256color")
	}

	tuiCfg := cfg.TUI
	ver := version.Version

	theme.ApplyOverrides(tuiColorMap(tuiCfg.Colors))
	components.SetSuggestions(tuiCfg.Suggestions)
	components.SetLogo(tuiCfg.Logo)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	workDir, _ := os.Getwd()

	model := chat.NewModel(ctx, cancel, workDir, ver, tuiCfg.Mode, tuiCfg.Colors.LogoColor, tuiCfg.LogoGradient)
	// Force truecolor: the TUI theme is 24-bit hex, and bubbletea's default
	// colorprofile.Detect can resolve to NoTTY/ASCII on some PTYs (e.g. a
	// headless/terminal-use emulator), which makes the renderer strip every
	// foreground and background color from the frame. Pinning TrueColor keeps
	// the theme (transcript cards, panel surfaces, selected-row highlight)
	// rendering as authored. See cmd/cli/tui/views/chat for the styling.
	p := tea.NewProgram(model, tea.WithColorProfile(colorprofile.TrueColor))
	model.SetProgram(p)

	// See the mouseTracking* consts above: 1000h tracking (not tea's 1002h
	// CellMotion) keeps wheel/click handling while freeing drags for native
	// text selection. Written before the renderer starts and reset after the
	// program exits, so there is no write interleaving with rendered frames.
	fmt.Fprint(os.Stdout, mouseTrackingEnable)
	defer fmt.Fprint(os.Stdout, mouseTrackingDisable)

	go startACPInProcess(ctx, model, p, cfg, ver, workDir)

	_, err := p.Run()
	return err
}

// startACPInProcess creates two os.Pipe pairs for client↔server communication.
// The ACP server runs in a goroutine via RunACPTransport; the client connects
// via ConnectIO, performs the initialize/newSession handshake, registers the
// event handler, and injects the session into the model.
func startACPInProcess(ctx context.Context, model *chat.Model, p *tea.Program, cfg config.Config, ver, workDir string) {
	// os.Pipe (buffered, 64KB) lets the client write requests before the
	// server finishes building; they buffer until RunTransport reads.
	serverR, clientW, err := os.Pipe()
	if err != nil {
		p.Send(chat.AcpErrorMsg(err))
		return
	}
	clientR, serverW, err := os.Pipe()
	if err != nil {
		p.Send(chat.AcpErrorMsg(err))
		return
	}

	// ACP server: build + run in a goroutine.
	go func() {
		if err := server.RunACPTransport(ctx, &cfg, serverW, serverR); err != nil {
			p.Send(chat.AcpErrorMsg(err))
		}
		_ = serverW.Close()
		_ = serverR.Close()
	}()

	// ACP client: connect and handshake.
	client := openacp.NewClient("openagent-tui", ver)
	sess := client.ConnectIO(ctx, clientW, clientR)

	if _, err := sess.Initialize(ctx, openacp.InitializeRequest{
		ProtocolVersion: 1,
		ClientInfo:      &openacp.Implementation{Name: "openagent-tui", Version: ver},
	}); err != nil {
		p.Send(chat.AcpErrorMsg(err))
		return
	}

	handler := chat.NewAcpEventHandler(p)
	sess.SetEventHandler(handler)
	sess.SetClientRequestHandler(handler)
	model.SetACPSession(sess)
	// The ACP session is created lazily on the user's first prompt — merely
	// opening the program (and the welcome page) must not persist a session.
	// ActiveSessionID stays empty until the first NewSession lands.
	p.Send(chat.AcpSessionReadyMsg("", nil))
}

// tuiColorMap translates config.TUIColors into the flat map shape
// theme.ApplyOverrides expects (snake_case keys → hex strings). Empty
// fields are dropped so ApplyOverrides keeps the built-in default.
func tuiColorMap(c config.TUIColors) map[string]string {
	m := map[string]string{}
	if c.BgNormal != "" {
		m["bg_normal"] = c.BgNormal
	}
	if c.BgSecondary != "" {
		m["bg_secondary"] = c.BgSecondary
	}
	if c.BgSurface != "" {
		m["bg_surface"] = c.BgSurface
	}
	if c.Primary != "" {
		m["primary"] = c.Primary
	}
	if c.Success != "" {
		m["success"] = c.Success
	}
	if c.Warning != "" {
		m["warning"] = c.Warning
	}
	if c.Danger != "" {
		m["danger"] = c.Danger
	}
	if c.TextNormal != "" {
		m["text_normal"] = c.TextNormal
	}
	if c.TextAsh != "" {
		m["text_ash"] = c.TextAsh
	}
	if c.BorderGray != "" {
		m["border_gray"] = c.BorderGray
	}
	if c.LogoColor != "" {
		m["logo_color"] = c.LogoColor
	}
	return m
}

package chat

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/yusheng-g/openagent-go/cmd/cli/tui/components"
	"github.com/yusheng-g/openagent-go/cmd/cli/tui/theme"
)

// welcomeInputWidth is the width of the whole input area (frame, padding,
// badges row and footer) on the welcome page: three quarters of the welcome
// column grown by 50%, floored for narrow terminals and capped at the
// column so the box never overruns the welcome layout.
func welcomeInputWidth(colWidth int) int {
	w := max(36, min(colWidth*3/4, 64))
	return min(w*3/2, colWidth)
}

// welcomeDrop is how many rows the welcome block hangs below exact vertical
// center: strict centering parks the input box a touch too high, so the page
// lets it settle a little lower instead.
const welcomeDrop = 2

// welcomeTopPad is the row the welcome block starts on: half the working
// area's slack (vertical centering) plus welcomeDrop, capped to the slack so
// the block never overflows the height-2 area on short terminals.
func (m *Model) welcomeTopPad(contentH int) int {
	room := max(0, m.height-2-contentH)
	return min(room, room/2+welcomeDrop)
}

// renderWelcome renders the pre-chat page: logo, input, status bar, footer.
// It writes no model state — the geometry a caller needs (input box position,
// used to dock the slash-command sheet) is written into geom.
func (m *Model) renderWelcome(geom *viewGeom) string {
	w := m.getContentWidth()
	content := m.welcomeInner()
	ch := lipgloss.Height(content)
	// Record where the input box starts so the slash-command card (floated
	// above it) does not land on the screen center.
	geom.inputTopY = m.welcomeTopPad(ch) + lipgloss.Height(m.welcomeLogo(w))

	// Height-2: leaves the last two rows for the footer and one blank row
	// below it, so WorkDir/Version float one row above the bottom edge.
	// Top placement with our own top pad (welcomeTopPad) instead of
	// PlaceVertical's Center: the block hangs welcomeDrop rows below strict
	// center.
	if pad := m.welcomeTopPad(ch); pad > 0 {
		content = strings.Repeat("\n", pad) + content
	}
	full := lipgloss.Place(m.width, m.height-2, lipgloss.Center, lipgloss.Top, content, lipgloss.WithWhitespaceStyle(theme.BaseStyle()))

	workDirValue := theme.BaseStyle().PaddingLeft(1).Foreground(theme.TextAsh).Render(m.workDir)
	versionValue := theme.BaseStyle().Width(m.width - lipgloss.Width(workDirValue)).PaddingRight(1).Align(lipgloss.Right).Foreground(theme.TextAsh).Render(m.version)
	footer := lipgloss.JoinHorizontal(lipgloss.Left, workDirValue, versionValue)

	return theme.BaseStyle().Width(m.width).Height(m.height).Render(
		lipgloss.JoinVertical(lipgloss.Top,
			full,
			footer,
		),
	)
}

// welcomeInner builds the centered block of the welcome page (logo, input,
// status, tips). renderWelcome measures its height to position the block and
// the input box (welcomeTopPad + geom), so the measured geometry can never
// drift from what is actually rendered.
func (m *Model) welcomeInner() string {
	w := m.getContentWidth()
	logo := m.welcomeLogo(w)
	// The welcome input is narrower than the chat box: three quarters of the
	// welcome column (capped), so the page stays airy on wide terminals.
	inputRaw := m.renderInputAt(welcomeInputWidth(w))
	// Center the box within the welcome column. The whitespace style is the
	// plain page background, so the box keeps its own width (the BgSurface
	// rows must not be stretched to the full column).
	input := lipgloss.PlaceHorizontal(w, lipgloss.Center, inputRaw, lipgloss.WithWhitespaceStyle(theme.BaseStyle()))
	status := m.renderStatus()
	return lipgloss.JoinVertical(lipgloss.Top,
		logo, input, status,
		theme.BaseStyle().Width(w).Render(""),
		theme.BaseStyle().Width(w).Align(lipgloss.Center).Render(m.tips),
	)
}

// welcomeLogo renders the welcome-page logo art at the given column width.
// Pure: reads logoColor/logoGradient and the theme, writes nothing.
func (m *Model) welcomeLogo(w int) string {
	logoArt := components.GetLogo(w)
	// Apply color: gradient (if set) → single logoColor → default TextAsh.
	colored := components.RenderLogoColored(logoArt, m.logoColor, m.logoGradient)
	if colored == logoArt {
		// no override applied — use theme.LogoColor (defaults to TextAsh)
		colored = theme.BaseStyle().Foreground(theme.LogoColor).Render(logoArt)
	}
	return theme.BaseStyle().Width(w).Align(lipgloss.Center).PaddingBottom(1).Render(colored)
}

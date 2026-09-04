package chat

import (
	"fmt"
	"strconv"
	"time"
	"image/color"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/yusheng-g/openagent-go/cmd/cli/tui/components"
	"github.com/yusheng-g/openagent-go/cmd/cli/tui/layout"
	"github.com/yusheng-g/openagent-go/cmd/cli/tui/theme"
	"github.com/yusheng-g/openagent-go/cmd/cli/tui/utils"
)

// windowTitle is the terminal window title set via the OSC 2 protocol
// (the renderer emits \x1b]2;<title>\x07 once, when it first changes).
const windowTitle = "OpenAgent"

func createView(text string) tea.View {
	v := tea.NewView(text)
	v.AltScreen = true
	v.ReportFocus = true
	// MouseMode stays None: CellMotion would emit \x1b[?1002h, which hands
	// every button drag to the app and disables the terminal's native text
	// selection. app.go enables plain 1000h tracking instead — wheel and
	// click events still arrive, while drags stay free for box selection.
	v.MouseMode = tea.MouseModeNone
	v.WindowTitle = windowTitle
	return v
}

func (m *Model) View() tea.View {
	if m.width <= 0 || m.height <= 0 {
		return createView("Initializing...")
	}

	if m.width < layout.MinWidth || m.height < layout.MinHeight {
		return createView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			fmt.Sprintf("The terminal size must be greater than %d x %d. \r\nCurrent is %d x %d", layout.MinWidth, layout.MinHeight, m.width, m.height)))
	}

	var background string
	var geom viewGeom
	if !m.inChat {
		background = m.renderWelcome(&geom)
	} else {
		background = m.renderMainView(&geom)
	}

	// Panel floats above the current view with a dimmed backdrop. Which
	// list is shown depends on the panel mode (commands/sessions/models).
	if m.panelOpen {
		var panel string
		switch m.panelMode {
		case panelModeSessions:
			panel = m.renderSessionPanel()
		case panelModeModels:
			panel = m.renderModelPanel()
		case panelModeConfig:
			panel = m.renderConfigPanel()
		case panelModeHelp:
			panel = m.renderHelpPanel()
		case panelModeExport:
			panel = m.renderExportPanel()
		case panelModeSearch:
			panel = m.renderSearchPanel()
		case panelModeEdit:
			panel = m.renderEditPanel()
		case panelModePlugins:
			panel = m.renderPluginsPanel()
		default:
			// The "/"-triggered picker is its own card; Ctrl+P keeps the
			// orange command-palette styling.
			if m.panelFromSlash {
				panel = m.renderSlashPanel()
			} else {
				panel = m.renderCommandPanel()
			}
		}
		// The slash sheet docks flush against the input box (no gap),
		// aligned to its left edge; every other panel stays centered.
		yPos, yOff := layout.Center, 0
		xPos, xOff := layout.Center, 0
		if m.panelFromSlash {
			yPos = layout.Top
			yOff = geom.inputTopY - lipgloss.Height(panel)
			xPos = layout.Left
			xOff = m.inputBoxX()
		}
		background = layout.CompositeMasked(panel, background,
			xPos, yPos, xOff, yOff)
	}

	// Final pass: force the page background onto every cell that has none,
	// so lipgloss alignment padding (plain spaces) never shows the
	// terminal's default color through the UI.
	background = theme.PaintBackground(background, theme.BgNormal)

	return createView(background)
}

func (m *Model) renderMainView(geom *viewGeom) string {
	left := m.renderLeft(geom)
	right := m.renderRight()
	content := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	// BaseStyle carries the page background so every padded cell is painted
	// black; a bare NewStyle here would leak the terminal default color.
	return theme.BaseStyle().Width(m.width).Height(m.height).Render(content)
}

func (m *Model) renderLeft(geom *viewGeom) string {
	leftW := layout.GetLeftWidth(m.width)
	vpH := layout.GetViewHeight(m.height)

	// When permission dialog is open, shrink the viewport to make room for
	// the permission panel (bottom-aligned, above status bar). The viewport
	// height and option-row Y positions are already synced by syncViewport
	// in Update; here we only render against them.
	var inputArea string
	if m.permissionReq != nil {
		inputArea = m.renderPermissionPanel(m.getContentWidth()-1, 0)
		vpHeight := m.chatViewport.Height()
		sb := m.renderScrollbar(vpHeight)
		scrollContainer := lipgloss.JoinHorizontal(lipgloss.Top, m.chatViewport.View(), sb)
		status := m.renderStatus()
		return theme.BaseStyle().Width(leftW).Padding(0, 1).Render(
			lipgloss.JoinVertical(lipgloss.Left, scrollContainer, inputArea, status),
		)
	}

	// Split view (/split): the transcript gets the top half and a compact
	// context pane the bottom half; otherwise the viewport spans the full
	// height.
	if m.splitView && vpH >= 10 {
		splitH := m.chatViewport.Height()
		ctxH := vpH - splitH
		sb := m.renderScrollbar(splitH)
		scrollContainer := lipgloss.JoinHorizontal(lipgloss.Top, m.chatViewport.View(), sb)
		ctxPane := m.renderSplitPane(ctxH)
		inputArea = m.renderInput()
		status := m.renderStatus()
		geom.inputTopY = vpH + 1 // viewport + blank row, split or not
		return theme.BaseStyle().Width(leftW).Padding(0, 1).Render(
			lipgloss.JoinVertical(lipgloss.Left, scrollContainer, ctxPane, inputArea, status),
		)
	}

	// Normal: full-height viewport + input + status.
	vpHeight := m.chatViewport.Height()
	sb := m.renderScrollbar(vpHeight)
	scrollContainer := lipgloss.JoinHorizontal(lipgloss.Top, m.chatViewport.View(), sb)
	inputArea = m.renderInput()
	status := m.renderStatus()
	geom.inputTopY = vpH + 1 // viewport + blank row, split or not
	return theme.BaseStyle().Width(leftW).Padding(0, 1).Render(
		lipgloss.JoinVertical(lipgloss.Left, scrollContainer, "", inputArea, status),
	)
}

// renderSplitPane draws the lower pane of split view: a compact context
// summary (session, model, plan steps).
func (m *Model) renderSplitPane(h int) string {
	w := layout.GetViewWidth(m.width)
	base := theme.BaseStyle().Width(w).Height(max(1, h)).Background(theme.BgSurface).Padding(0, 1)
	var b strings.Builder
	b.WriteString(theme.BaseStyle().Foreground(theme.TextNormal).Bold(true).Render("Context"))
	b.WriteString("\nSession: " + m.activeSessionID)
	b.WriteString("\nModel: " + m.currentModel())
	for _, e := range m.planEntries {
		mark := "-"
		switch e.Status {
		case "completed":
			mark = "[✓]"
		case "in_progress":
			mark = "[▶]"
		}
		b.WriteString("\n" + mark + " " + utils.TruncateByWidth(e.Content, max(4, w-4)))
	}
	return base.Render(b.String())
}

func createHalf(width int, borderColor color.Color) string {
	halfBlock := lipgloss.NewStyle().
		Background(theme.BgNormal).
		Foreground(theme.BgSurface).
		Render("▀")

	content := lipgloss.NewStyle().
		Background(theme.BgNormal).
		Foreground(borderColor).
		Render("╹")
	for i := 0; i < width-1; i++ {
		content += halfBlock
	}

	return content

}

func (m *Model) renderInput() string {
	return m.renderInputAt(m.getContentWidth() - 1)
}

// renderInputAt renders the input box at a specific content width (the
// welcome page uses a narrower box than the chat page).
func (m *Model) renderInputAt(contentWidth int) string {
	borderColor := theme.BorderGray
	isFocused := m.chatTextarea.Focused()
	if isFocused {
		borderColor = theme.Primary
	}
	headerStyle := theme.BaseStyle().Width(contentWidth).Height(1).Background(theme.BgSurface).PaddingLeft(1)
	contentStyle := theme.BaseStyle().Width(contentWidth).Height(layout.InputHeight).Background(theme.BgSurface).PaddingLeft(1)
	content := ""
	if m.chatTextarea.Value() == "" {
		placeholder := m.chatTextarea.Placeholder
		if len(placeholder) == 0 {
			content = contentStyle.Foreground(theme.TextAsh).Render("")
		} else if !m.blink || !isFocused {
			content = contentStyle.Foreground(theme.TextAsh).Render(placeholder)
		} else {
			firstRune, firstSize := utf8.DecodeRuneInString(placeholder)
			firstChar := theme.BaseStyle().Background(theme.TextNormal).Foreground(theme.TextInk).Render(string(firstRune))
			remainingText := ""
			if len(placeholder) > firstSize {
				remainingText = theme.BaseStyle().Background(theme.BgSurface).Foreground(theme.TextAsh).Render(placeholder[firstSize:])
			}
			content = contentStyle.Render(lipgloss.JoinHorizontal(lipgloss.Left, firstChar, remainingText))
		}

	} else {
		// The textarea is sized to the current box's inner width by
		// updateInputWidth (called from Update on resize/page switch), so its
		// View always fits inside the frame it is rendered in.
		content = contentStyle.Render(m.chatTextarea.View())
	}
	header := headerStyle.Render("")

	info := headerStyle.Render(joinBadges(
		m.renderModeBadge(),
		m.renderQueuedBadge(),
		m.renderModelBadge(),
		m.renderThinkingBadge(),
	))
	footer := createHalf(contentWidth, borderColor)
	input := theme.BaseStyle().
		Width(contentWidth).
		Background(theme.BgSurface).
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderBackground(theme.BgNormal).
		BorderForeground(borderColor).Render(lipgloss.JoinVertical(lipgloss.Left, header, content, info))
	return lipgloss.JoinVertical(lipgloss.Left, input, footer)
}

// joinBadges concatenates the input-header badges with a spaced dot (" · "),
// skipping empty badges so the leading badge never carries a dangling
// separator (the mode badge renders "" until the config options arrive).
// Every part carries its own BgSurface fill; the separator shares it, so no
// surface-colored seam shows between badges.
func joinBadges(parts ...string) string {
	sep := theme.BaseStyle().Background(theme.BgSurface).Foreground(theme.TextNormal).Render(" · ")
	out, first := "", true
	for _, p := range parts {
		if p == "" {
			continue
		}
		if !first {
			out += sep
		}
		out += p
		first = false
	}
	return out
}

// renderModeBadge shows the agent's session mode as the input header's
// first badge, adapting to the modes the agent actually defines: Auto
// (fully automated) in the theme primary, Manual (approval required) in
// green, Plan (read-only, plan-first) in cyan. Empty (config options not
// yet fetched) renders no badge at all.
func (m *Model) renderModeBadge() string {
	var label string
	var col color.Color
	switch m.mode {
	case "auto":
		label, col = "Auto", theme.Primary
	case "manual":
		label, col = "Manual", theme.Success
	case "plan":
		label, col = "Plan", theme.Notify
	default:
		return ""
	}
	return theme.BaseStyle().Background(theme.BgSurface).Foreground(col).Render(label)
}

// renderModelBadge shows the active model id in the input header, joined
// to the other badges with a spaced dot (" · "). The id is usually
// "<provider>/<model>"; truncated so a long id cannot wrap the single-line
// header.
func (m *Model) renderModelBadge() string {
	model := m.currentModel()
	if model == "" {
		return ""
	}
	model = utils.TruncateByWidth(model, 32)
	return theme.BaseStyle().Background(theme.BgSurface).Foreground(theme.TextNormal).
		Render(model)
}

// renderThinkingBadge shows the active thinking strength (the session's
// thought_level option) in the input header, joined with a spaced dot and
// without a leading icon; empty when the agent exposes no such option.
// Always drawn in the warning orange so it stands out from the neutral
// model badge.
func (m *Model) renderThinkingBadge() string {
	lvl := m.currentThoughtLevel()
	if lvl == "" {
		return ""
	}
	return theme.BaseStyle().Background(theme.BgSurface).Foreground(theme.Warning).Render(lvl)
}

// renderQueuedBadge shows how many user inputs are waiting behind the
// in-flight prompt (max maxInputQueue). Empty string when nothing is queued.
func (m *Model) renderQueuedBadge() string {
	if len(m.inputQueue) == 0 {
		return ""
	}
	return theme.BaseStyle().Background(theme.BgSurface).Foreground(theme.Warning).
		Render(fmt.Sprintf("[Queued:%d]", len(m.inputQueue)))
}

func (m *Model) renderStatus() string {
	contentWidth := m.getContentWidth()
	help := components.RenderCommandTip("enter", "send")
	help = help + components.RenderCommandTip("ctrl+c", "quit")
	help = help + components.RenderCommandTip("ctrl+p", "commands")
	m.statusBar.Width = contentWidth
	if m.notifyMsg != "" {
		// Transient toast: cyan, replaces the persistent status line until
		// it auto-clears.
		m.statusBar.Status = theme.BaseStyle().Foreground(theme.Notify).Render(m.notifyMsg)
	} else if m.loading {
		m.statusBar.Status = m.spinner.View() + " " + m.statusText
	} else {
		m.statusBar.Status = m.statusText
	}
	m.statusBar.Help = help
	return m.statusBar.View()
}

// renderPlanList draws the agent's plan as a TODO list ("Plans n/m" + status
// checkboxes) in the right sidebar. Empty when no plan has arrived.
func (m *Model) renderPlanList(background lipgloss.Style, width int) string {
	if len(m.planEntries) == 0 {
		return background.Width(width).Render("")
	}
	done := 0
	for _, e := range m.planEntries {
		if e.Status == "completed" {
			done++
		}
	}
	title := background.Width(width).Foreground(theme.TextNormal).Bold(true).
		Render(fmt.Sprintf("Plans %d/%d", done, len(m.planEntries)))
	rows := []string{title}
	for _, e := range m.planEntries {
		mark := "[ ]"
		fg := theme.TextAsh
		switch e.Status {
		case "completed":
			mark = "[✓]"
		case "in_progress":
			mark = "[▶]"
			fg = theme.Primary
		}
		body := utils.TruncateByWidth(e.Content, max(0, width-4))
		rows = append(rows, background.Width(width).Foreground(fg).Render(mark+" "+body))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// formatTokens renders a token count compactly for the sidebar: 834,
// 12.3k, 1.2M (a trailing .0 is trimmed).
func formatTokens(n int) string {
	compact := func(v float64, suffix string) string {
		s := strconv.FormatFloat(v, 'f', 1, 64)
		return strings.TrimSuffix(s, ".0") + suffix
	}
	switch {
	case n >= 1_000_000:
		return compact(float64(n)/1_000_000, "M")
	case n >= 1_000:
		return compact(float64(n)/1_000, "k")
	default:
		return strconv.Itoa(n)
	}
}

// contextValue renders the sidebar's context line: "used / window tokens"
// once usage has been reported, the window alone before the first update.
func (m *Model) contextValue() string {
	switch {
	case m.usedTokens > 0 && m.contextSize > 0:
		return fmt.Sprintf("%s / %s tokens", formatTokens(m.usedTokens), formatTokens(m.contextSize))
	case m.contextSize > 0:
		return formatTokens(m.contextSize) + " tokens"
	case m.usedTokens > 0:
		return formatTokens(m.usedTokens) + " tokens"
	default:
		return "0 tokens"
	}
}

func (m *Model) renderRight() string {
	width := layout.GetRightWidth(m.width)
	if width <= 0 {
		return ""
	}
	background := theme.BaseStyle().Background(theme.BgSecondary)
	rightStyle := background.Width(width).Height(m.height).PaddingLeft(1)

	sessionTitle := background.Width(width - 1).Foreground(theme.TextNormal).Bold(true).Render("Session")
	sessionValue := background.Width(width - 1).Foreground(theme.TextAsh).Render(m.activeSessionID)

	contextTitle := background.Width(width - 1).Foreground(theme.TextNormal).Bold(true).Render("Context")
	contextText := background.Width(width - 1).Foreground(theme.TextAsh).Render(m.contextValue())
	turnsTitle := background.Width(width - 1).Foreground(theme.TextNormal).Bold(true).Render("Turns")
	turnsValue := background.Width(width - 1).Foreground(theme.TextAsh).Render(strconv.Itoa(m.promptCount))

	todoContent := m.renderPlanList(background, width-1)
	header := lipgloss.JoinVertical(lipgloss.Left,
		sessionTitle, sessionValue, "",
		contextTitle, contextText, "",
		turnsTitle, turnsValue, "",
		todoContent,
	)

	workDirTitle := background.Width(width - 1).Foreground(theme.TextNormal).Bold(true).Render("WorkDir")
	workDirValue := background.Width(width - 1).Foreground(theme.TextAsh).Render(m.workDir)

	versionTitle := background.Width(width - 1).Foreground(theme.TextNormal).Bold(true).Render("Version")
	versionValue := background.Width(width - 1).Foreground(theme.TextAsh).Render(m.version)
	footer := lipgloss.JoinVertical(lipgloss.Left, workDirTitle, workDirValue, "", versionTitle, versionValue)

	headerH := lipgloss.Height(header)
	footerH := lipgloss.Height(footer)

	spacesH := max(0, m.height-headerH-footerH-1)
	space := strings.Repeat("\n", spacesH)
	return rightStyle.
		Render(lipgloss.JoinVertical(lipgloss.Left, header, space, footer))
}

func (m *Model) renderScrollbar(height int) string {
	if height <= 0 {
		return ""
	}
	thumbStyle := theme.BaseStyle().Background(theme.ThumbBackGround)
	trackStyle := theme.BaseStyle().Background(theme.TrackBackGround)

	emptyLine := trackStyle.Render(" ")
	trackLine := trackStyle.Render(" ")
	thumbLine := thumbStyle.Render(" ")

	totalLines := m.chatViewport.TotalLineCount()
	yOffset := m.chatViewport.YOffset

	var doc strings.Builder
	if totalLines <= height {
		for i := 0; i < height; i++ {
			doc.WriteString(emptyLine)
			if i < height-1 {
				doc.WriteString("\n")
			}
		}
		return doc.String()
	}
	thumbH := max(1, height*height/totalLines)
	maxOffset := totalLines - height
	thumbY := yOffset() * (height - thumbH) / maxOffset
	for i := 0; i < height; i++ {
		if i >= thumbY && i < thumbY+thumbH {
			doc.WriteString(thumbLine)
		} else {
			doc.WriteString(trackLine)
		}
		if i < height-1 {
			doc.WriteString("\n")
		}
	}
	return doc.String()
}

// renderPermissionPanel renders an inline panel (replacing the input area)
// showing the tool call that needs approval. Bottom-aligned above status.
func (m *Model) renderPermissionPanel(width, _ int) string {
	req := m.permissionReq
	tc := req.ToolCall
	title := tc.Title
	if title == "" {
		title = "Tool Call"
	}

	contentW := width
	contentStyle := theme.BaseStyle().Width(contentW).Background(theme.BgPanel)
	yellow := lipgloss.Color("#ffd60a")
	warnStyle := theme.BaseStyle().Background(theme.BgPanel).Foreground(yellow)

	icon := warnStyle.Render("⚠")
	allow := warnStyle.Bold(true).Render("allow")
	space := warnStyle.Render(" ")
	toolName := warnStyle.Render(title)
	question := warnStyle.Render("?")
	headerTitle := contentStyle.Render(lipgloss.JoinHorizontal(lipgloss.Left, icon, space, allow, space, toolName, question))

	// Options: one per line, full width, separator line between them.
	// Selected option uses Primary foreground + "▶" marker.
	sep := contentStyle.Foreground(theme.BorderGray).Render(strings.Repeat("─", contentW-1))
	optionParts := make([]string, 0, len(req.Options)*2)
	for i, opt := range req.Options {
		name := opt.Name
		if name == "" {
			name = string(opt.OptionID)
		}
		marker := "  "
		optStyle := contentStyle.Foreground(theme.TextAsh)
		if i == m.permissionSelectedIdx {
			marker = "▶ "
			optStyle = contentStyle.Foreground(theme.Primary)
		}
		optionParts = append(optionParts, optStyle.Render(marker+name))
		if i < len(req.Options)-1 {
			optionParts = append(optionParts, sep)
		}
	}
	optionList := lipgloss.JoinVertical(lipgloss.Left, optionParts...)

	footer := contentStyle.PaddingRight(1).Align(lipgloss.Right).Render(
		lipgloss.JoinHorizontal(lipgloss.Right,
			components.RenderCommandTipSurface("↑ ↓", "switch"),
			components.RenderCommandTipSurface("esc", "cancel"),
			components.RenderCommandTipSurface("enter", "select"),
		),
	)

	content := lipgloss.JoinVertical(lipgloss.Left, headerTitle, "", optionList, "", footer)

	borderColor := theme.Warning
	return theme.BaseStyle().
		Width(width).
		Background(theme.BgPanel).
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderBackground(theme.BgPanel).
		BorderForeground(borderColor).
		Render(content)
}

// panelBox returns the outer (frame) and inner content width for the
// floating panels. Content is roughly half the terminal, capped so the
// palette stays readable on wide screens. The "/" sheet is NOT sized by
// this: it docks to the input box instead.
func (m *Model) panelBox() (panelW, contentW int) {
	contentW = m.width / 2
	if contentW > 56 {
		contentW = 56
	}
	if contentW < 36 {
		contentW = 36
	}
	return contentW + 2, contentW
}

// inputBoxWidth returns the column width of the current input box. The
// "/" sheet docks to this width (and to inputBoxX) so the picker lines up
// exactly with the input it extends.
func (m *Model) inputBoxWidth() int {
	if !m.inChat {
		return welcomeInputWidth(layout.GetWelcomeWidth(m.width))
	}
	return m.getContentWidth() - 1
}

// inputBoxX returns the screen column where the current input box starts:
// the chat input sits inside the left column's padding, the welcome input
// is centered under the logo.
func (m *Model) inputBoxX() int {
	if !m.inChat {
		return max(0, (m.width-m.inputBoxWidth())/2)
	}
	return 1
}

// panelFooter renders the shared key-hint line, clamped to contentW so it
// never widens the frame (footer is joined into the same block as rows).
// bg carries the popup surface so the hints sit on the same fill.
func panelFooter(base lipgloss.Style, contentW int, bg color.Color) string {
	// RenderCommandTipOn renders a leading space before every pair; the
	// first pair must not carry one, so the footer's text starts flush with
	// the popup's title and rows (opencode-style shared left edge). The
	// whole line is nudged 1 column right to match the rows' inner padding.
	first := base.Foreground(theme.TextNormal).Render("↑ ↓") + base.Foreground(theme.TextAsh).Render(" switch")
	footerText := lipgloss.JoinHorizontal(lipgloss.Left,
		base.Render(" "),
		first,
		components.RenderCommandTipOn("⏎", "run", bg),
		components.RenderCommandTipOn("esc", "close", bg),
	)
	// Width (not MaxWidth) fills the line out to contentW with the popup
	// surface: MaxWidth only clips, so the tail past the text was plain
	// spaces showing the terminal's own background through the popup.
	return base.Width(contentW).Render(footerText)
}

// ── borderless popups ──

// The floating panels (except the docked "/" sheet) are borderless popups: a
// clean centered block on the panel surface. They share this skeleton — a
// "title … esc" header, the body rows, and the key-hint footer — sized to
// contentW so the block measures a uniform width and Composite centers it.

// popupBase returns the popup surface style and its inner content width.
func (m *Model) popupBase() (lipgloss.Style, int) {
	_, contentW := m.panelBox()
	return theme.BaseStyle().Background(theme.BgPanel), contentW
}

// popupHeader renders the popup title row: "title" left, an optional live
// filter (already styled) beside it, and "esc" right, spanned to contentW.
// A leading pad aligns the title with the option rows' inner text (every
// row carries 1-column horizontal padding inside its fill).
func popupHeader(base lipgloss.Style, contentW int, title, filter string) string {
	titleS := base.Foreground(theme.TextNormal).Bold(true).Render(title)
	escS := base.Foreground(theme.TextMute).Render("esc")
	// A soft space keeps a live filter from gluing to the title
	// ("Searcha" would read like a typo).
	filterS := base.Render(" ") + filter
	gap := max(0, contentW-1-lipgloss.Width(titleS)-lipgloss.Width(filterS)-lipgloss.Width(escS))
	return base.Width(contentW).Render(lipgloss.JoinHorizontal(lipgloss.Left, base.Render(" "), titleS, filterS, base.Width(gap).Render(""), escS))
}

// popupFrame wraps a popup body in the borderless panel surface with a
// 1-column inner padding. Popups draw no frame at all — no side borders, no
// edges — only the docked "/" sheet keeps gray side borders (drawPanelSides).
func popupFrame(content string) string {
	return theme.BaseStyle().Background(theme.BgPanel).Padding(0, 1).Render(content)
}

// popupPanel assembles a borderless popup around pre-rendered body rows: the
// "title … esc" header, the rows, and the key-hint footer. Rows must be
// styled by the caller at Width(contentW) so every cell carries the panel
// background (lipgloss stretches even blank lines to the block width).
func (m *Model) popupPanel(title, filter string, rows []string) string {
	base, contentW := m.popupBase()
	return m.popupPanelStyled(base, contentW, title, filter, rows, panelFooter(base, contentW, theme.BgPanel))
}

// popupPanelStyled is popupPanel with a caller-chosen footer line, for
// panels whose key hints differ from the standard switch/run/close set.
func (m *Model) popupPanelStyled(base lipgloss.Style, contentW int, title, filter string, rows []string, footer string) string {
	header := popupHeader(base, contentW, title, filter)
	list := lipgloss.JoinVertical(lipgloss.Left, rows...)
	content := lipgloss.JoinVertical(lipgloss.Left, header, "", list, "", footer)
	return popupFrame(content)
}

// windowPanelRows windows n list rows around the selection so the popup never
// outgrows the terminal (the sessions popup's budget: terminal height minus
// chrome). It returns the full range when the list fits.
func (m *Model) windowPanelRows(n int) (first, last int) {
	first, last = 0, n
	if maxRows := max(1, m.height-6); n > maxRows {
		first = min(max(0, m.panelIdx-maxRows/2), n-maxRows)
		last = first + maxRows
	}
	return first, last
}

// renderCommandPanel draws the Ctrl+P command palette as a centered popup: a
// "Commands" title (with the live "/filter" once the user types) over the
// command rows, borderless like every other popup. The selected row is
// filled with the CommandActive peach edge-to-edge.
func (m *Model) renderCommandPanel() string {
	cmds := m.buildPanelCommands()
	base, contentW := m.popupBase()
	filter := ""
	if m.panelFilter != "" {
		filter = base.Foreground(theme.CommandActive).Render("/" + m.panelFilter)
	}
	rows := m.paletteRows(base, contentW, cmds, m.panelIdx)
	return m.popupPanel("Commands", filter, rows)
}

// maxSheetRows caps how many commands the slash sheet shows at once; the
// window slides so the selected row always stays visible (a real picker
// scrolls, instead of growing past the terminal).
const maxSheetRows = 8

// renderSlashPanel draws the "/"-triggered command picker as a pull-up
// sheet docked to the input box: gray side borders, a slim header with a
// hairline (only while filtering), full-width rows with the selected one
// filled in warning orange edge-to-edge, and no frame padding so the
// highlight reaches the borders. The inner content spans the input box width
// minus the two border columns, so the framed sheet is exactly as wide as the
// input box it docks against.
func (m *Model) renderSlashPanel() string {
	cmds := m.buildPanelCommands()
	// Inner content width = input box width minus the two border columns
	// (no frame padding, so a selected row fills edge-to-edge).
	contentW := max(0, m.inputBoxWidth()-2)
	base := theme.BaseStyle().Background(theme.BgSurface)

	// No title row and no query echo: while filtering, the typed text lives
	// in the input box below the sheet; the sheet only mirrors the matching
	// commands.

	// Fixed-column rows: icon + slash column + description column. The
	// slash column is padded to a constant width so every description
	// starts at the same offset.
	nameCol := 22
	descCol := contentW - 3 - nameCol // icon, spacer, separator
	if descCol < 10 {
		nameCol = max(0, contentW-3-10)
		descCol = 10
	}
	panelIdx := m.panelIdx
	first := 0
	last := len(cmds)
	if len(cmds) > maxSheetRows {
		first = min(max(0, panelIdx-maxSheetRows/2), len(cmds)-maxSheetRows)
		last = first + maxSheetRows
	}
	var rows []string
	for i := first; i < last; i++ {
		pc := cmds[i]
		selected := i == panelIdx
		// The selected row is a full-width peach block (CommandActive, the
		// same accent opencode uses) matching the command-palette look; every
		// part of it (icon, spacers, name, description, right tail) shares
		// that fill, so no surface-colored seam shows.
		rowStyle := base
		iconFg, nameFg, descFg := theme.TextAsh, theme.TextAsh, theme.TextMute
		if selected {
			rowStyle = base.Background(theme.CommandActive)
			iconFg, nameFg, descFg = theme.TextInk, theme.TextInk, theme.TextInk
		} else if !pc.enabled {
			nameFg = theme.TextMute
		}
		icon := rowStyle.Foreground(iconFg).Render(" ")
		if pc.space {
			icon = rowStyle.Foreground(iconFg).Render(m.toggleIcon(pc.slash))
		}
		descText := utils.TruncateByWidth(pc.title, descCol)
		namePad := rowStyle.Render(strings.Repeat(" ", max(1, nameCol-lipgloss.Width(pc.slash))))
		// A leading pad inside the row gives the highlight (and every row) a
		// uniform inner padding; the Width(contentW) fill pads the tail.
		row := lipgloss.JoinHorizontal(lipgloss.Left,
			rowStyle.Render(" "),
			icon, rowStyle.Render(" "),
			rowStyle.Foreground(nameFg).Render(pc.slash), namePad,
			rowStyle.Foreground(descFg).Render(descText))
		rows = append(rows, rowStyle.Width(contentW).Render(row))
	}
	list := lipgloss.JoinVertical(lipgloss.Left, rows...)

	// Slim key-hint footer, kept inside contentW. With no matches the
	// footer says so — without echoing the query (it lives in the input
	// box, the sheet stays quiet).
	footer := base.Foreground(theme.TextMute).Width(contentW).
		Render("↑↓ move · enter run · esc close")
	if len(cmds) == 0 {
		footer = base.Foreground(theme.CommandActive).Width(contentW).
			Render("No matching commands")
	}

	// Frame the pull-up sheet with gray side borders and no frame padding, so
	// the selected row's orange fill spans edge-to-edge inside the borders.
	// The surface background is kept so the sheet reads as the input growing
	// upward; the highlight's own inner pad comes from the row layout.
	content := lipgloss.JoinVertical(lipgloss.Left, list, footer)
	return drawPanelSides(strings.Split(content, "\n"), theme.BorderGray, theme.BgSurface)
}

// paletteRows renders the command-list body for the Ctrl+P palette:
// fixed-column rows with the selected one filled in the CommandActive peach
// edge-to-edge. base carries the popup background so rows paint uniformly;
// each row is filled to contentW so the popup block measures uniformly.
// Rows carry no leading icon/marker column: like opencode's palette, command
// names start on the popup's left edge, flush with the title and footer.
func (m *Model) paletteRows(base lipgloss.Style, contentW int, cmds []panelCommand, panelIdx int) []string {
	// Fixed-column rows: slash column + description column. The slash
	// column is padded to a constant width (always keeping one separator
	// space) so every description starts at the same offset. Both budgets
	// live inside the row's 1-column horizontal padding.
	nameCol := 24
	descCol := contentW - 2 - nameCol
	if descCol < 10 {
		nameCol = max(0, contentW-12)
		descCol = 10
	}
	var rows []string
	for i, pc := range cmds {
		selected := i == panelIdx
		// The selected row is a full-width peach block (CommandActive), the
		// same treatment as the slash sheet; every part shares the fill so no
		// surface-colored seam shows. The row's horizontal padding keeps the
		// text off the fill's edges.
		rowStyle := base
		nameFg, descFg := theme.TextAsh, theme.TextMute
		if selected {
			rowStyle = base.Background(theme.CommandActive)
			nameFg, descFg = theme.TextInk, theme.TextInk
		} else if !pc.enabled {
			nameFg = theme.TextMute
		}
		descText := utils.TruncateByWidth(pc.title, descCol)
		namePad := rowStyle.Render(strings.Repeat(" ", max(1, nameCol-lipgloss.Width(pc.slash))))
		row := lipgloss.JoinHorizontal(lipgloss.Left,
			rowStyle.Foreground(nameFg).Render(pc.slash), namePad,
			rowStyle.Foreground(descFg).Render(descText))
		rows = append(rows, rowStyle.Padding(0, 1).Width(contentW).Render(row))
	}
	return rows
}

// renderSessionPanel lists the resumable sessions. The selected row gets
// the full CommandActive fill — selection is the fill alone, no marker
// glyphs, so every popup's highlighted row reads the same. While
// ListSessions is in flight the panel displays "Loading...".
//
// It renders as a borderless, centered contentW-wide popup (every line drawn
// at exactly contentW so the block measures a uniform width and Composite
// can center it): a "Sessions" + "esc" header, rows windowed to the terminal
// height, and a filled footer so the panel's background stays uniform to the
// bottom edge.
func (m *Model) renderSessionPanel() string {
	base, contentW := m.popupBase()

	var rows []string
	switch {
	case m.sessionsLoading:
		rows = append(rows, base.Padding(0, 1).Width(contentW).Render(base.Foreground(theme.TextMute).Render("Loading...")))
	case len(m.sessionItems) == 0:
		rows = append(rows, base.Padding(0, 1).Width(contentW).Render(base.Foreground(theme.TextMute).Render("No sessions")))
	default:
		first, last := m.windowPanelRows(len(m.sessionItems))
		for i, it := range m.sessionItems[first:last] {
			// No marker column: like every popup row, the name starts on the
			// shared left edge (1 column of inner row padding); selection is
			// the full-row CommandActive fill. The last-update time rides the
			// right edge in muted text (ink on the selected row).
			rowStyle := base
			nameFg, tsFg := theme.TextAsh, theme.TextMute
			if first+i == m.panelIdx {
				rowStyle = base.Background(theme.CommandActive)
				nameFg, tsFg = theme.TextInk, theme.TextInk
			}
			ts := relativeTime(time.Now(), it.updated)
			nameBudget := contentW - 6 // inner width minus the 2 row-pad columns
			if ts != "" {
				nameBudget -= lipgloss.Width(ts) + 2
			}
			name := utils.TruncateByWidth(it.title, max(8, nameBudget))
			gap := max(0, contentW-2-lipgloss.Width(name)-lipgloss.Width(ts))
			cells := []string{rowStyle.Foreground(nameFg).Render(name)}
			if ts != "" {
				cells = append(cells,
					rowStyle.Render(strings.Repeat(" ", gap)),
					rowStyle.Foreground(tsFg).Render(ts),
				)
			} else {
				cells = append(cells, rowStyle.Render(strings.Repeat(" ", gap)))
			}
			rows = append(rows, rowStyle.Padding(0, 1).Width(contentW).Render(lipgloss.JoinHorizontal(lipgloss.Left, cells...)))
		}
	}
	return m.popupPanel("Sessions", "", rows)
}

// relativeTime renders an RFC3339 timestamp as a compact relative label for
// the sessions panel: "just now", "5m ago", "2h ago", "3d ago", falling back
// to the plain date for older sessions. Empty input and unparseable values
// render as "" (no time column for that row).
func relativeTime(now time.Time, rfc3339 string) string {
	if rfc3339 == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return ""
	}
	d := now.Sub(t)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	return t.Format("2006-01-02")
}

// renderModelPanel lists the selectable models as a borderless popup. The
// selected row gets the full CommandActive fill — selection is the fill
// alone, no marker glyphs, matching the sessions popup. Long model lists
// are windowed to the terminal height like the sessions popup.
func (m *Model) renderModelPanel() string {
	base, contentW := m.popupBase()

	opts := m.modelOptions()
	var rows []string
	if len(opts) == 0 {
		rows = append(rows, base.Padding(0, 1).Width(contentW).Render(base.Foreground(theme.TextMute).Render("No model options")))
	} else {
		first, last := m.windowPanelRows(len(opts))
		for i, id := range opts[first:last] {
			rowStyle := base
			nameFg := theme.TextAsh
			if first+i == m.panelIdx {
				rowStyle = base.Background(theme.CommandActive)
				nameFg = theme.TextInk
			}
			// The name lives inside the row's 1-column horizontal padding.
			name := utils.TruncateByWidth(id, contentW-6)
			pad := max(0, contentW-2-lipgloss.Width(name))
			rows = append(rows, rowStyle.Padding(0, 1).Width(contentW).Render(lipgloss.JoinHorizontal(lipgloss.Left,
				rowStyle.Foreground(nameFg).Render(name),
				rowStyle.Render(strings.Repeat(" ", pad)),
			)))
		}
	}
	return m.popupPanel("Models", "", rows)
}

// renderConfigPanel is a select config-option picker (session mode, thought
// level, ...): a centered popup like the models picker, its rows mirrored
// from the agent's config option, opened preselected on the current value.
// The highlighted row is the CommandActive fill alone — no marker glyphs,
// matching every other popup.
func (m *Model) renderConfigPanel() string {
	base, contentW := m.popupBase()

	var rows []string
	opts := m.configPickerOptions(m.configPickerID)
	if len(opts) == 0 {
		rows = append(rows, base.Padding(0, 1).Width(contentW).Render(base.Foreground(theme.TextMute).Render("No options")))
	}
	for i, opt := range opts {
		rowStyle := base
		nameFg, descFg := theme.TextAsh, theme.TextMute
		if i == m.panelIdx {
			rowStyle = base.Background(theme.CommandActive)
			nameFg, descFg = theme.TextInk, theme.TextInk
		}
		name := utils.TruncateByWidth(opt.name, 12)
		// The description budget accounts for the name column (fixed at 14
		// cells) plus inner padding and the row's 2 padding columns, so a
		// long description can never wrap the row into two lines.
		desc := utils.TruncateByWidth(opt.desc, max(8, contentW-14-8))
		pad := max(0, 14-lipgloss.Width(name))
		gap := max(0, contentW-2-lipgloss.Width(name)-pad-lipgloss.Width(desc))
		rows = append(rows, rowStyle.Padding(0, 1).Width(contentW).Render(lipgloss.JoinHorizontal(lipgloss.Left,
			rowStyle.Foreground(nameFg).Render(name),
			rowStyle.Render(strings.Repeat(" ", pad)),
			rowStyle.Foreground(descFg).Render(desc),
			rowStyle.Render(strings.Repeat(" ", gap)),
		)))
	}
	title := "Options"
	if o := sessionConfigOption(m.configOptions, m.configPickerID); o != nil && o.Name != "" {
		title = o.Name // data-driven: "Session Mode", "Reasoning Level", ...
	}
	return m.popupPanel(title, "", rows)
}

// renderHelpPanel draws the /help overlay: a short explanation plus key
// hints. It is dismiss-only (any key closes it), matching the spec.
// renderExportPanel surfaces the /export outcome in a dismiss-only dialog:
// the written file path (hard-wrapped, never truncated — the user needs the
// whole path) or the reason nothing was written. The footer says so — the
// standard switch/run/close hints do not apply here.
func (m *Model) renderExportPanel() string {
	base, contentW := m.popupBase()
	var rows []string
	for _, ln := range strings.Split(m.exportNotice, "\n") {
		for {
			r := []rune(ln)
			if len(r) <= contentW-2 {
				rows = append(rows, base.Padding(0, 1).Width(contentW).Render(base.Foreground(theme.TextAsh).Render(ln)))
				break
			}
			rows = append(rows, base.Padding(0, 1).Width(contentW).Render(base.Foreground(theme.TextAsh).Render(string(r[:contentW-2]))))
			ln = string(r[contentW-2:])
		}
	}
	footer := base.Padding(0, 1).Width(contentW).Render(base.Foreground(theme.TextMute).Render("any key closes"))
	return m.popupPanelStyled(base, contentW, "Export", "", rows, footer)
}

func (m *Model) renderHelpPanel() string {
	base, contentW := m.popupBase()

	lines := []string{
		"Press ctrl+p to see all available",
		"actions and commands in any context.",
		"",
		"Tips:",
		"  /toggle_*         flip transcript visibility",
		"  /sessions         resume a past session",
		"  /models           switch the model",
		"  /toggle_mode      auto / manual / plan",
		"  /exit             quit",
	}
	var rows []string
	for _, ln := range lines {
		rows = append(rows, base.Padding(0, 1).Width(contentW).Render(base.Foreground(theme.TextAsh).Render(ln)))
	}
	return m.popupPanel("Help", "", rows)
}

// renderSearchPanel draws the /search overlay as a borderless popup: a live
// query beside the title, matched messages below (the selection is filled),
// enter jumps there.
func (m *Model) renderSearchPanel() string {
	base, contentW := m.popupBase()

	filter := ""
	if m.panelFilter != "" {
		filter = base.Foreground(theme.CommandActive).Render(m.panelFilter)
	}

	var rows []string
	switch {
	case m.panelFilter == "":
		rows = append(rows, base.Padding(0, 1).Width(contentW).Render(base.Foreground(theme.TextMute).Render("Type to search the transcript...")))
	case len(m.searchResults) == 0:
		rows = append(rows, base.Padding(0, 1).Width(contentW).Render(base.Foreground(theme.TextMute).Render("No matches for \""+m.panelFilter+"\"")))
	default:
		first, last := m.windowPanelRows(len(m.searchResults))
		for i, mi := range m.searchResults[first:last] {
			src := m.messages[mi]
			snippet := src.Role + ": " + utils.TruncateByWidth(
				strings.Join(strings.Split(utils.UnifiedEndOfLine(src.Content), "\n")[:1], ""), contentW-4)
			rowStyle := base
			fg := theme.TextAsh
			if first+i == m.panelIdx {
				rowStyle = base.Background(theme.CommandActive)
				fg = theme.TextInk
			}
			rows = append(rows, rowStyle.Padding(0, 1).Width(contentW).Render(rowStyle.Foreground(fg).Render(snippet)))
		}
	}
	return m.popupPanel("Search", filter, rows)
}

// renderEditPanel draws the /edit picker as a borderless popup: past user
// messages, newest first; enter copies the selection back into the input for
// editing. At most 20 entries are shown, windowed on short terminals.
func (m *Model) renderEditPanel() string {
	base, contentW := m.popupBase()

	idxs := m.editableMessages()
	var rows []string
	if len(idxs) == 0 {
		rows = append(rows, base.Padding(0, 1).Width(contentW).Render(base.Foreground(theme.TextMute).Render("No user messages to edit")))
	} else {
		shown := idxs
		if len(shown) > 20 {
			shown = shown[:20]
		}
		first, last := m.windowPanelRows(len(shown))
		for i, mi := range shown[first:last] {
			firstLine := strings.Split(utils.UnifiedEndOfLine(m.messages[mi].Content), "\n")[0]
			snippet := utils.TruncateByWidth("user: "+firstLine, contentW-4)
			rowStyle := base
			fg := theme.TextAsh
			if first+i == m.panelIdx {
				rowStyle = base.Background(theme.CommandActive)
				fg = theme.TextInk
			}
			rows = append(rows, rowStyle.Padding(0, 1).Width(contentW).Render(rowStyle.Foreground(fg).Render(snippet)))
		}
		if len(idxs) > 20 {
			rows = append(rows, base.Padding(0, 1).Width(contentW).Render(base.Foreground(theme.TextMute).Render(fmt.Sprintf("… %d older", len(idxs)-20))))
		}
	}
	return m.popupPanel("Edit message", "", rows)
}

// renderPluginsPanel draws the installed-plugin list (/plugins) as a
// borderless popup.
func (m *Model) renderPluginsPanel() string {
	base, contentW := m.popupBase()

	var rows []string
	if len(m.pluginItems) == 0 {
		rows = append(rows, base.Padding(0, 1).Width(contentW).Render(base.Foreground(theme.TextMute).Render("No plugins installed")))
	} else {
		first, last := m.windowPanelRows(len(m.pluginItems))
		for i, p := range m.pluginItems[first:last] {
			rowStyle := base
			fg := theme.TextAsh
			if first+i == m.panelIdx {
				rowStyle = base.Background(theme.CommandActive)
				fg = theme.TextInk
			}
			rows = append(rows, rowStyle.Padding(0, 1).Width(contentW).Render(rowStyle.Foreground(fg).Render(utils.TruncateByWidth(p, contentW-4))))
		}
	}
	return m.popupPanel("Plugins", "", rows)
}

// drawPanelSides frames panel lines with left and right borders in the given
// color (no top/bottom edges) and no inner padding, so the content is flush
// against the borders and a full-width highlight can reach them. It is the
// docked-sheet counterpart of the borderless popups. Trailing plain padding
// from the panel's lipgloss joins is trimmed and re-painted so no cell shows
// the terminal default background through the panel.
func drawPanelSides(lines []string, borderColor color.Color, bg color.Color) string {
	w := 0
	trimmed := make([]string, len(lines))
	for i, l := range lines {
		l = strings.TrimRight(l, " ")
		trimmed[i] = l
		if lw := lipgloss.Width(l); lw > w {
			w = lw
		}
	}
	edge := theme.BaseStyle().Foreground(borderColor).Background(bg)
	surface := theme.BaseStyle().Background(bg)

	var sb strings.Builder
	for i, l := range trimmed {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(edge.Render("┃"))
		sb.WriteString(l)
		if pad := w - lipgloss.Width(l); pad > 0 {
			sb.WriteString(surface.Render(strings.Repeat(" ", pad)))
		}
		sb.WriteString(edge.Render("┃"))
	}
	return sb.String()
}

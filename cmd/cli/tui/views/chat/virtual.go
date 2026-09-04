package chat

import (
	"strings"

	"github.com/yusheng-g/openagent-go/cmd/cli/tui/layout"
	"github.com/yusheng-g/openagent-go/cmd/cli/tui/theme"
	"github.com/yusheng-g/openagent-go/cmd/cli/tui/utils"
)

// ── line-level virtual scrolling (17.2) ──
//
// The viewport is fed a *windowed* document: messages whose row range
// intersects the visible window are styled (through the per-message render
// cache) and their rows are emitted verbatim; every other row is a cheap
// placeholder. Styling cost is proportional to the visible window plus a
// one-time pass that measures every message's exact rendered height (the
// cache makes the measurement free after the first feed). Positioning uses
// those exact heights — the rows actually drawn come from the styled
// blocks, so window math and drawn rows can never drift apart.

// placeholderRow is the muted gutter row shown for off-window lines. The
// padding is measured with the same width function fitRow uses, so the row
// is exactly vpW wide whatever go-runewidth decides for the rail glyph
// (box-drawing width varies with the EastAsianWidth flag some dependencies
// toggle globally) — uniform rows keep the soft-wrap mapping 1:1 with the
// virtual window.
func placeholderRow(vpW int) string {
	rail := "┆"
	return theme.BaseStyle().Background(theme.BgSurface).
		Foreground(theme.TextMute).Render(rail + strings.Repeat(" ", max(0, vpW-utils.DisplayWidth(rail))))
}

// fitRow normalizes an (ANSI-styled) transcript row to exactly vpW columns.
// bubbles viewport soft-wraps any row wider than its content width, which
// would shift its line↔row mapping away from our virtual window offsets;
// uniform rows keep that mapping 1:1. Long rows are cut to width with their
// styling preserved (a card's box-drawing rail measures double width under
// runewidth, so style-stripping here would drain every bordered row of its
// colors); short rows are padded with trailing spaces.
func fitRow(row string, vpW int) string {
	c := utils.DisplayWidth(row)
	switch {
	case c == vpW:
		return row
	case c > vpW:
		return utils.TruncateStyled(row, vpW)
	default:
		return row + strings.Repeat(" ", vpW-c)
	}
}

// virtualLineHeights returns the exact rendered height of every message, in
// viewport rows. Heights come from the styled blocks themselves (through the
// per-message render cache, so each block is styled once) — any estimate
// drifts from reality as soon as the card chrome changes, and a short
// estimate makes the virtual window cut the block's bottom padding and
// margin rows off. Hidden (visibility-gated) messages occupy no rows.
func (m *Model) virtualLineHeights(vpW int) []int {
	h := make([]int, len(m.messages))
	for i := range m.messages {
		block, skip := m.renderMessageBlock(i, m.messages[i], vpW)
		if skip || block == "" {
			continue
		}
		h[i] = strings.Count(block, "\n") + 1
	}
	return h
}

// virtualPrefixLines returns the sum of rendered rows before message idx,
// i.e. the content line at which the message starts.
func (m *Model) virtualPrefixLines(idx int) int {
	vpW := layout.GetViewWidth(m.width)
	n := 0
	for i, h := range m.virtualLineHeights(vpW) {
		if i >= idx {
			break
		}
		n += h
	}
	return n
}

// messageAtLine returns the message index whose estimated row range
// contains the given content line, plus the row offset within it. Lines
// past the end clamp into the last message.
func messageAtLine(heights []int, line int) (idx, within int) {
	if len(heights) == 0 {
		return -1, 0
	}
	line = max(0, line)
	for i, h := range heights {
		if line < h {
			return i, line
		}
		line -= h
	}
	last := len(heights) - 1
	return last, min(line, max(0, heights[last]-1))
}

// feedViewport refreshes the viewport content with the windowed document.
// It refeeds when the scroll offset or viewport size moved the window, so
// scrolling supplements the newly revealed messages (styling them on
// demand) while stable ones reuse the render cache. Called by syncViewport
// from Update, never by the render pass.
func (m *Model) feedViewport(h int) {
	m.chatViewport.SetHeight(h)
	offset := m.chatViewport.YOffset()
	if m.viewportDirty || offset != m.fedOffset || h != m.fedHeight {
		m.chatViewport.SetContent(m.renderVirtualDocAt(h, offset))
		if m.needAutoScroll {
			m.chatViewport.GotoBottom()
		}
		m.fedOffset = m.chatViewport.YOffset()
		m.fedHeight = h
		// Auto-scroll moved the offset: rebase the styled window to it.
		if m.fedOffset != offset {
			m.chatViewport.SetContent(m.renderVirtualDocAt(h, m.fedOffset))
		}
		m.viewportDirty = false
	}
}

// renderVirtualDoc builds the viewport document for the current scroll
// position: real styled rows inside the visible window [offset,
// offset+height), placeholder rows elsewhere. The row count always equals
// the estimated total, so bubbles' scroll clamping and the scrollbar stay
// consistent.
func (m *Model) renderVirtualDoc(height int) string {
	return m.renderVirtualDocAt(height, m.chatViewport.YOffset())
}

// renderVirtualDocAt is renderVirtualDoc for an explicit window offset —
// search jumps must feed a document whose styled window is the destination
// before SetYOffset clamps there.
func (m *Model) renderVirtualDocAt(height, offset int) string {
	vpW := layout.GetViewWidth(m.width)
	if len(m.messages) == 0 {
		return ""
	}
	heights := m.virtualLineHeights(vpW)
	offset = max(0, offset)
	windowH := max(1, height)
	windowEnd := offset + windowH
	first, startWithin := messageAtLine(heights, offset)
	last, _ := messageAtLine(heights, max(0, windowEnd-1))

	var b strings.Builder
	emitPlaceholder := func(rows int) {
		for r := 0; r < rows; r++ {
			b.WriteString(placeholderRow(vpW))
			b.WriteByte('\n')
		}
	}

	// Rows above the window.
	above := 0
	for i := 0; i < first; i++ {
		above += heights[i]
	}
	emitPlaceholder(above)

	// Window messages: real block rows where visible, placeholders where
	// the window clips the message's estimated range or styling is gated.
	cursor := offset - startWithin // absolute row of the first message's row 0
	for i := first; i <= last; i++ {
		block, skip := m.renderMessageBlock(i, m.messages[i], vpW)
		var lines []string
		if !skip && block != "" {
			lines = strings.Split(block, "\n")
		}
		for r := 0; r < heights[i]; r++ {
			abs := cursor + r
			if abs < offset || abs >= windowEnd || skip || r >= len(lines) {
				b.WriteString(placeholderRow(vpW))
			} else {
				b.WriteString(fitRow(lines[r], vpW))
			}
			b.WriteByte('\n')
		}
		cursor += heights[i]
	}

	// Rows below the window.
	below := 0
	for i := last + 1; i < len(heights); i++ {
		below += heights[i]
	}
	emitPlaceholder(below)

	return strings.TrimRight(b.String(), "\n")
}

package chat

import (
	"charm.land/lipgloss/v2"

	"github.com/yusheng-g/openagent-go/cmd/cli/tui/layout"
)

// layoutSnap is the piece of render geometry that must be computed off the
// render pass so View stays a pure renderer. syncViewport fills it during
// Update; the click handler and the render pass only read it.
type layoutSnap struct {
	// permOptionY are the terminal Y coordinates of each permission-option
	// row while the approval dialog is open (used for mouse hit-testing).
	permOptionY []int
}

// viewGeom is the geometry a single render pass carries so the render
// functions never write to the model. inputTopY is the screen row where the
// input box starts, used to dock the slash-command sheet above it.
type viewGeom struct {
	inputTopY int
}

// viewportHeight returns the transcript viewport height for the current page
// layout. Pure: reads state, never mutates it.
func (m *Model) viewportHeight() int {
	if m.permissionReq != nil {
		// The permission panel replaces the input area; shrink the viewport
		// so transcript + panel + status fit exactly in the terminal.
		panel := m.renderPermissionPanel(m.getContentWidth()-1, 0)
		ph := lipgloss.Height(panel)
		return max(3, m.height-1-ph-layout.StatusHeight)
	}
	vpH := layout.GetViewHeight(m.height)
	if m.splitView {
		return max(1, vpH/2)
	}
	return vpH
}

// syncViewport refeeds the transcript viewport when its content, size or
// scroll offset changed, and remembers the permission-dialog option rows. It
// is called at the end of every Update (bubbletea runs cmd goroutines and
// renders between Updates, so driving the viewport here instead of in View
// keeps the render pass free of state mutation and prevents a render from
// swallowing a pending flush). No-op on the welcome page, which has no
// transcript viewport.
func (m *Model) syncViewport() {
	if !m.inChat {
		return
	}

	h := m.viewportHeight()

	// Permission option rows sit just below the viewport (panelY = h).
	m.geom.permOptionY = m.geom.permOptionY[:0]
	if m.permissionReq != nil {
		for i := range m.permissionReq.Options {
			m.geom.permOptionY = append(m.geom.permOptionY, h+2+i*2)
		}
	}

	m.feedViewport(h)
}

// updateInputWidth sizes the textarea to the current page's input-box inner
// width, so its View never spills past the box frame. Called from Update when
// the window resizes or the page (welcome ⇄ chat) changes.
func (m *Model) updateInputWidth() {
	m.chatTextarea.SetWidth(max(1, m.inputTextWidth()-2))
}

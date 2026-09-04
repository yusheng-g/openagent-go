package chat

import "github.com/yusheng-g/openagent-go/cmd/cli/tui/layout"

// getContentWidth returns the column width for the page currently shown:
// the airy welcome column on the welcome page, the chat column otherwise.
// Keyed on inChat (the page switch), not on session existence — a boot
// session exists before the user ever enters chat, so activeSessionID was
// a wrong proxy that stretched the welcome layout to full chat width.
func (m *Model) getContentWidth() int {
	if !m.inChat {
		return layout.GetWelcomeWidth(m.width)
	}
	return layout.GetContentWidth(m.width)
}

// inputTextWidth returns the inner text width of the input box for the
// page currently shown: the box frame width minus its left border and the
// single-column left padding. The chat textarea is sized to this so its
// View always fits inside the box it is rendered in.
func (m *Model) inputTextWidth() int {
	if !m.inChat {
		return welcomeInputWidth(layout.GetWelcomeWidth(m.width)) - 2
	}
	return m.getContentWidth() - 1 - 2
}

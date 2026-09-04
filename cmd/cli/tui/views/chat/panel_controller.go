package chat

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// handlePanelKey routes a keypress while the command panel is open. It returns
// the command to emit (nil for none) and whether the key was consumed. All
// panel state lives on the model; this method exists so the panel's key
// handling is kept out of the main Update switch, which is already large.
func (m *Model) handlePanelKey(k tea.KeyPressMsg) (cmd tea.Cmd, handled bool) {
	// Help and export panels are dismiss-only: any key closes them.
	if m.panelMode == panelModeHelp || m.panelMode == panelModeExport {
		m.panelOpen = false
		return nil, true
	}

	// Plugins list is read-only: ↑/↓ navigate, enter/esc close.
	if m.panelMode == panelModePlugins {
		switch k.String() {
		case "ctrl+c", "esc", "enter":
			m.panelOpen = false
			m.panelMode = panelModeCommand
		case "up":
			if m.panelIdx > 0 {
				m.panelIdx--
			}
		case "down":
			if m.panelIdx < len(m.pluginItems)-1 {
				m.panelIdx++
			}
		}
		return nil, true
	}

	// Edit picker: ↑/↓ choose a past user message, enter copies it back into
	// the input for editing.
	if m.panelMode == panelModeEdit {
		switch k.String() {
		case "ctrl+c", "esc":
			m.panelOpen = false
			m.panelMode = panelModeCommand
		case "up":
			if m.panelIdx > 0 {
				m.panelIdx--
			}
		case "down":
			if m.panelIdx < len(m.editableMessages())-1 {
				m.panelIdx++
			}
		case "enter":
			_, cmd = m.execEditSelection()
		}
		return cmd, true
	}

	// Search overlay: typing extends the query, ↑/↓ pick a match, enter jumps
	// the viewport to it.
	if m.panelMode == panelModeSearch {
		switch k.String() {
		case "ctrl+c", "esc":
			m.panelOpen = false
			m.panelFilter = ""
		case "up":
			if len(m.searchResults) > 0 && m.panelIdx > 0 {
				m.panelIdx--
			}
		case "down":
			if m.panelIdx < len(m.searchResults)-1 {
				m.panelIdx++
			}
		case "enter":
			_, cmd = m.execSearchSelection()
		case "backspace":
			if m.panelFilter != "" {
				r := []rune(m.panelFilter)
				m.panelFilter = string(r[:len(r)-1])
				m.panelIdx = 0
				m.refreshSearch()
			}
		case "space":
			m.panelFilter += " "
			m.panelIdx = 0
			m.refreshSearch()
		default:
			if s := k.String(); len(s) == 1 && s[0] > ' ' && s[0] < 0x7f {
				m.panelFilter += s
				m.panelIdx = 0
				m.refreshSearch()
			}
		}
		return cmd, true
	}

	// Slash sheet: the typed text lives in the input box (the box shows
	// "/..." while filtering), not in the panel — printable keys, space and
	// backspace fall through to the textarea, and the filter re-derives from
	// the box after every keystroke. The sheet never echoes the query; it
	// only mirrors the matching commands. Enter runs the selected command
	// and clears the box (panelExecute); esc closes and resets the box to
	// the empty state the sheet was opened from; backspacing the slash away
	// closes the sheet (syncSlashSheet).
	if m.panelMode == panelModeCommand && m.panelFromSlash {
		switch k.String() {
		case "ctrl+c", "esc":
			// Closing the sheet resets the box too: the sheet only ever
			// opens on an empty input, so the "/"-filter draft it leaves
			// behind is transient — keeping it would prefix the next
			// prompt with "/" or double the slash on a retype.
			m.chatTextarea.SetValue("")
			m.panelOpen = false
			m.panelMode = panelModeCommand
			m.panelFilter = ""
		case "up":
			if n := m.panelItemCount(); n > 0 && m.panelIdx > 0 {
				m.panelIdx--
			}
		case "down":
			if n := m.panelItemCount(); n > 0 && m.panelIdx < n-1 {
				m.panelIdx++
			}
		case "enter":
			// Nothing selected (no matches): keep the sheet open so the
			// draft stays editable instead of bouncing the user.
			if len(m.buildPanelCommands()) > 0 {
				_, cmd = m.panelExecute()
			}
		default:
			var tcmd tea.Cmd
			m.chatTextarea, tcmd = m.chatTextarea.Update(k)
			cmd = tcmd
			m.syncSlashSheet()
		}
		return cmd, true
	}

	// Command palette / sessions / models: shared navigation, plus filter
	// input for the command palette.
	switch k.String() {
	case "ctrl+c", "esc":
		m.panelOpen = false
		m.panelMode = panelModeCommand
		m.panelFilter = ""
	case "up":
		if n := m.panelItemCount(); n > 0 && m.panelIdx > 0 {
			m.panelIdx--
		}
	case "down":
		if n := m.panelItemCount(); n > 0 && m.panelIdx < n-1 {
			m.panelIdx++
		}
	case "enter":
		_, cmd = m.panelExecute()
	case "backspace":
		if m.panelMode == panelModeCommand && m.panelFilter != "" {
			m.panelFilter = m.panelFilter[:len(m.panelFilter)-1]
			m.panelIdx = 0
		}
	case "space":
		if m.panelMode != panelModeCommand {
			return nil, true
		}
		// Empty filter: space toggles the selected command.
		if m.panelFilter == "" {
			_, cmd = m.panelToggleSelected()
			return cmd, true
		}
		m.panelFilter += " "
		m.panelIdx = 0
	default:
		if m.panelMode != panelModeCommand {
			return nil, true
		}
		// Printable rune → extend the filter.
		if s := k.String(); len(s) == 1 && s[0] > ' ' && s[0] < 0x7f {
			m.panelFilter += s
			m.panelIdx = 0
		}
	}
	return cmd, true
}

// syncSlashSheet re-derives the slash sheet's filter from the input box
// after the textarea took a keystroke, and closes the sheet once the box
// stops looking like a command (the "/" backspaced away, a paste without
// one). The selection resets whenever the filter changes, matching the
// palette's filter-editing behavior.
func (m *Model) syncSlashSheet() {
	v := m.chatTextarea.Value()
	if !strings.HasPrefix(v, "/") {
		m.panelOpen = false
		m.panelMode = panelModeCommand
		m.panelFilter = ""
		return
	}
	filter := strings.TrimPrefix(v, "/")
	if filter != m.panelFilter {
		m.panelFilter = filter
		m.panelIdx = 0
	}
}

// handlePermissionKey routes a keypress while a tool-call permission dialog is
// open. The user's selection is written back through the reply channel; a
// non-nil command (tea.Quit) is returned for Ctrl+C.
func (m *Model) handlePermissionKey(k tea.KeyPressMsg) tea.Cmd {
	switch k.String() {
	case "ctrl+c":
		return tea.Quit
	case "esc":
		m.respondPermission(-1)
	case "up", "left":
		if m.permissionSelectedIdx > 0 {
			m.permissionSelectedIdx--
		}
	case "down", "right":
		if m.permissionReq != nil && m.permissionSelectedIdx < len(m.permissionReq.Options)-1 {
			m.permissionSelectedIdx++
		}
	case "enter":
		m.respondPermission(m.permissionSelectedIdx)
	}
	return nil
}

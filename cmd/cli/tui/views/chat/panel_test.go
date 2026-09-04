package chat

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// TestCommandPanelFrameConsistent guards the Ctrl+P palette layout: every
// rendered line of the borderless popup must share one width — any overflow
// or mis-measured line makes the block ragged. Regression for the
// lipgloss-vs-runewidth width-accounting bug (go-runewidth counts ↑↓○●▶ as
// 2 columns where lipgloss counts 1).
func TestCommandPanelFrameConsistent(t *testing.T) {
	for _, width := range []int{60, 80, 100, 140} {
		m := newTestModel()
		m.width = width
		m.height = 30
		panel := m.renderCommandPanel()
		lines := strings.Split(panel, "\n")
		if len(lines) == 0 {
			t.Fatalf("width=%d: empty panel", width)
		}
		want := lipgloss.Width(lines[0])
		for i, ln := range lines {
			if got := lipgloss.Width(ln); got != want {
				t.Errorf("width=%d line %d: width %d, want %d (line=%q)",
					width, i, got, want, ln)
			}
		}
	}
}

// TestCommandPanelFitsTerminal ensures the palette never exceeds the
// terminal width (it must stay centered and readable).
func TestCommandPanelFitsTerminal(t *testing.T) {
	for _, width := range []int{60, 80, 140} {
		m := newTestModel()
		m.width = width
		m.height = 30
		panel := m.renderCommandPanel()
		if got := lipgloss.Width(panel); got > width {
			t.Errorf("width=%d: panel width %d exceeds terminal", width, got)
		}
	}
}

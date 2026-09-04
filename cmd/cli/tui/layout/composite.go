package layout

import (
	"fmt"
	"image/color"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type Position int

const (
	Top Position = iota + 1
	Right
	Bottom
	Left
	Center
)

// Composite merges and flattens the background and foreground views into a single view.
// This implementation is based off of the one used by Superfile -
// https://github.com/yorukot/superfile/blob/main/src/pkg/string_function/overplace.go
func Composite(fg, bg string, xPos, yPos Position, xOff, yOff int) string {
	if fg == "" {
		return bg
	}
	if bg == "" {
		return fg
	}
	if strings.Count(fg, "\n") == 0 && strings.Count(bg, "\n") == 0 {
		// Simple case: both are single line
		return fg
	}

	fgWidth, fgHeight := lipgloss.Size(fg)
	bgWidth, bgHeight := lipgloss.Size(bg)

	if fgWidth >= bgWidth && fgHeight >= bgHeight {
		return fg
	}

	x, y := offsets(fg, bg, xPos, yPos, xOff, yOff)
	x = clamp(x, 0, bgWidth-fgWidth)
	y = clamp(y, 0, bgHeight-fgHeight)

	fgLines := lines(fg)
	bgLines := lines(bg)
	var sb strings.Builder

	for i, bgLine := range bgLines {
		if i > 0 {
			sb.WriteByte('\n')
		}
		if i < y || i >= y+fgHeight {
			sb.WriteString(bgLine)
			continue
		}

		pos := 0
		if x > 0 {
			left := ansi.Truncate(bgLine, x, "")
			pos = ansi.StringWidth(left)
			sb.WriteString(left)
			if pos < x {
				sb.WriteString(whitespace(x - pos))
				pos = x
			}
		}

		fgLine := fgLines[i-y]
		sb.WriteString(fgLine)
		pos += ansi.StringWidth(fgLine)

		right := ansi.TruncateLeft(bgLine, pos, "")
		bgWidth := ansi.StringWidth(bgLine)
		rightWidth := ansi.StringWidth(right)
		if rightWidth <= bgWidth-pos {
			sb.WriteString(whitespace(bgWidth - rightWidth - pos))
		}
		sb.WriteString(right)
	}
	return sb.String()
}

// offsets calculates the actual vertical and horizontal offsets used to position the foreground
// tea.Model relative to the background tea.Model.
func offsets(fg, bg string, xPos, yPos Position, xOff, yOff int) (int, int) {
	var x, y int

	// Handle X axis positioning
	switch xPos {
	case Left:
		x = 0
	case Center:
		halfBackgroundWidth := lipgloss.Width(bg) / 2
		halfForegroundWidth := lipgloss.Width(fg) / 2
		x = halfBackgroundWidth - halfForegroundWidth
	case Right:
		x = lipgloss.Width(bg) - lipgloss.Width(fg)
	}

	// Handle Y axis positioning
	switch yPos {
	case Top:
		y = 0
	case Center:
		halfBackgroundHeight := lipgloss.Height(bg) / 2
		halfForegroundHeight := lipgloss.Height(fg) / 2
		y = halfBackgroundHeight - halfForegroundHeight
	case Bottom:
		y = lipgloss.Height(bg) - lipgloss.Height(fg)
	}

	return x + xOff, y + yOff
}

// clamp clamps a value between lower and upper bounds.
func clamp(v, lower, upper int) int {
	// Handle case where bounds are inverted (e.g. min > max)
	if lower > upper {
		lower, upper = upper, lower
	}
	if v < lower {
		return lower
	}
	if v > upper {
		return upper
	}
	return v
}

// lines normalises any non standard new lines within a string, and then splits and returns a slice
// of strings split on the new lines.
func lines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
}

// whitescpace returns a string of whitespace characters of the requested length.
func whitespace(length int) string {
	return strings.Repeat(" ", length)
}

var (
	ScrimColor = "rgba(0,0,0,0.8)"
	// DefaultBg is the page black the scrim darkens toward, so the popup
	// backdrop stays the same black as the page instead of a warm gray.
	DefaultBg = "#000000"
)

var sgrResetRe = regexp.MustCompile(`\x1b\[(0(;\d+)*)?m`)

func CompositeMasked(fg, bg string, xPos, yPos Position, xOff, yOff int, mask ...bool) string {
	enabled := true
	if len(mask) > 0 {
		enabled = mask[0]
	}
	if !enabled {
		return Composite(fg, bg, xPos, yPos, xOff, yOff)
	}

	return Composite(fg, applyScrim(bg), xPos, yPos, xOff, yOff)
}

func applyScrim(s string) string {
	cc := color.RGBAModel.Convert(lipgloss.Color(ScrimColor))
	c, ok := cc.(color.RGBA)
	if !ok {
		// applyScrim: unable to convert color to RGBA; fall through with zero alpha
	}
	alpha := float64(c.A) / 255
	if alpha <= 0 {
		return s
	}

	dark := lipgloss.Darken(lipgloss.Color(DefaultBg), alpha)
	bgCode := colorBgCode(dark)

	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = applyScrimLine(line, bgCode)
	}
	return strings.Join(lines, "\n")
}

func colorBgCode(c color.Color) string {
	rgbC := color.RGBAModel.Convert(c)
	rgba, ok := rgbC.(color.RGBA)
	if !ok {
		// colorBgCode: unable to convert color to RGBA; return empty
		return ""
	}
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", rgba.R, rgba.G, rgba.B)
}

func applyScrimLine(line, bgCode string) string {
	s := "\x1b[2m" + bgCode + line
	s = sgrResetRe.ReplaceAllStringFunc(s, func(match string) string {
		return match + "\x1b[2m" + bgCode
	})
	s += "\x1b[22m\x1b[49m"
	return s
}

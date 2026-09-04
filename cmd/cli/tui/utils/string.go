// Package utils holds string helpers shared by the TUI views and
// components. All width-based operations use display width (CJK runes
// count as 2) via go-runewidth so that Chinese text never breaks the
// layout.
package utils

import (
	"regexp"
	"strings"

	"github.com/mattn/go-runewidth"
)

// ansiRe matches CSI escape sequences (colors, cursor moves, etc.).
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes ANSI escape sequences from s. Used before measuring
// or truncating text that contains styled tool output.
func StripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// DisplayWidth returns the visible width of s: ANSI sequences are
// stripped first, then CJK/wide runes count as 2 columns.
func DisplayWidth(s string) int {
	return runewidth.StringWidth(StripANSI(s))
}

// TruncateByWidth cuts s to at most maxWidth display columns, appending
// the ellipsis "…". Layout-affecting ANSI styling is stripped so the
// truncated string stays well-formed.
func TruncateByWidth(s string, maxWidth int) string {
	return runewidth.Truncate(StripANSI(s), maxWidth, "…")
}

// TruncateStyled cuts s to at most maxWidth display columns while keeping
// the ANSI styling of the surviving cells intact. Unlike TruncateByWidth,
// this is used on rows that are only slightly over-width (e.g. a
// box-drawing glyph measured as double width by runewidth), where throwing
// away the row's colors would visibly degrade the transcript. The result
// always ends in a reset so no open style leaks onto the following line.
func TruncateStyled(s string, maxWidth int) string {
	if DisplayWidth(s) <= maxWidth {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	width := 0
	rest := s
	for rest != "" {
		loc := ansiRe.FindStringIndex(rest)
		plain := rest
		if loc != nil {
			plain = rest[:loc[0]]
		}
		if plain != "" {
			space := maxWidth - width
			if space <= 0 {
				b.WriteString("\x1b[m")
				return b.String()
			}
			keep := runewidth.Truncate(plain, space, "")
			b.WriteString(keep)
			width += runewidth.StringWidth(keep)
			if runewidth.StringWidth(plain) > space {
				b.WriteString("\x1b[m")
				return b.String()
			}
		}
		if loc == nil {
			return b.String()
		}
		b.WriteString(rest[loc[0]:loc[1]])
		rest = rest[loc[1]:]
	}
	return b.String()
}

// UnifiedEndOfLine normalizes \r\n and legacy \r line endings to \n.
// Markdown rendering and tool output both assume Unix line endings.
func UnifiedEndOfLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

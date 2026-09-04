package components

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Package blockfont_mini renders text in the classic 3-row × 4-col
// half-block banner style: every letterform is three terminal rows drawn
// with █ ▀ ▄ (and spaces), one glyph per cell with a single space between
// neighbouring glyphs. This is the letterface of the welcome-page logo —
// "openagent" reads as (the sixth letterform is the lowercase 'g' — a bowl
// over a closed double-storey base; all other letters keep their uppercase
// faces)
//
//	█▀▀█ █▀▀█ █▀▀█ █▀▀▄ ▄▀▀▄ █▀▀█ █▀▀█ █▀▀▄ ▀█▀▀
//	█  █ █  █ █▀▀▀ █  █ █▀▀█ ▀▀▀█ █▀▀▀ █  █  █
//	▀▀▀▀ █▀▀▀ ▀▀▀▀ ▀  ▀ ▀  ▀ ▀▀▀▀ ▀▀▀▀ ▀  ▀  ▀
//
// The letterforms for O, P, E, N follow the reference banner verbatim; the
// rest are drawn in the same 4×3 half-block idiom. A-Z and 0-9 are
// supported; lowercase runes fall back to their uppercase face unless a
// dedicated lowercase letterform (currently only 'g') is defined. Every
// other rune renders as a blank glyph.
const (
	miniGlyphWidth  = 4
	miniGlyphHeight = 3
)

// miniGlyphs maps each rune to its 3-row letterform. Rows are stored
// trimmed on the right; leading spaces carry the shape's left offset.
var miniGlyphs = map[rune][miniGlyphHeight]string{
	'A': {"▄▀▀▄", "█▀▀█", "▀  ▀"},
	'B': {"█▀▀█", "█▀▀▄", "▀▀▀▀"},
	'C': {"█▀▀▀", "█", "▀▀▀▀"},
	'D': {"█▀▀▀", "█  █", "▀▀▀▄"},
	'E': {"█▀▀█", "█▀▀▀", "▀▀▀▀"},
	'F': {"█▀▀█", "█▀▀▀", "█"},
	'G': {"█▀▀▀", "█ ▀█", "▀▀▀▀"},
	'g': {"█▀▀█", "▀▀▀█", "▀▀▀▀"},
	'H': {"█  █", "█▀▀█", "▀  ▀"},
	'I': {"▀▀█▀", "  █", "  █"},
	'J': {"  █", "  █", "▀▀▀▄"},
	'K': {"█ ▄▀", "█▀▄", "▀  ▀"},
	'L': {"█", "█", "▀▀▀▀"},
	'M': {"█▄▄█", "█▀▀█", "▀  ▀"},
	'N': {"█▀▀▄", "█  █", "▀  ▀"},
	'O': {"█▀▀█", "█  █", "▀▀▀▀"},
	'P': {"█▀▀█", "█  █", "█▀▀▀"},
	'Q': {"█▀▀█", "█  █", "▀▀▀█"},
	'R': {"█▀▀█", "█▀█", "▀  ▀"},
	'S': {"█▀▀▀", "▀▀▀▄", "▄▀▀▀"},
	'T': {"▀█▀▀", " █  ", " ▀  "},
	'U': {"█  █", "█  █", "▀▀▀▀"},
	'V': {"█  █", "▀▄▄▀", "  ▀"},
	'W': {"█  █", "█▄▄█", " ▀▀"},
	'X': {"▀▄▄▀", " ██", "▀  ▀"},
	'Y': {"█  █", " ▀█", "  ▀"},
	'Z': {"▀▀▀█", " █▀", "▀▀▀▀"},
	'0': {"█▀▀█", "█  █", "▀▀▀▀"},
	'1': {"  █", "  █", "  █"},
	'2': {"▀▀▀█", "▀▀▀█", "▄▄▄█"},
	'3': {"█▀▀█", "  ▀█", "▀▀▀█"},
	'4': {"█  █", "█▀▀█", "   █"},
	'5': {"█▀▀█", "█▀▀▄", "▀▀▀▄"},
	'6': {"█▀▀█", "█▀▀▄", "▀▀▀▄"},
	'7': {"▀▀▀█", "  █", "  █"},
	'8': {"█▀▀█", "█▀▀█", "▀▀▀▀"},
	'9': {"█▀▀█", "█▀▀▄", "▀▀▀█"},
	' ': {"", "", ""},
}

// miniGlyph returns the letterform for ch, falling back to uppercase (so
// lowercase input renders with the uppercase face) and finally to a blank
// glyph.
func miniGlyph(ch rune) [miniGlyphHeight]string {
	if g, ok := miniGlyphs[ch]; ok {
		return g
	}
	if u := unicode.ToUpper(ch); u != ch {
		if g, ok := miniGlyphs[u]; ok {
			return g
		}
	}
	return miniGlyphs[' ']
}

// RenderBlockMini renders text in the 3-row half-block banner style, one
// space between glyphs, right-trimmed per row so the banner's right edge is
// flush. Non-alphanumeric input is sanitized like the other block faces.
func RenderBlockMini(text string) string {
	filtered, _ := sanitizeLogo(text)
	if filtered == "" {
		return ""
	}
	rows := make([]string, miniGlyphHeight)
	for i, ch := range filtered {
		g := miniGlyph(ch)
		for r := 0; r < miniGlyphHeight; r++ {
			if i > 0 {
				rows[r] += " "
			}
			rows[r] += strings.TrimRight(g[r], " ")
		}
	}
	for r := range rows {
		rows[r] = strings.TrimRight(rows[r], " ")
	}
	return strings.Join(rows, "\n")
}

// BlockWidthMini returns the rendered width in terminal columns of text in
// the 3-row banner face: each glyph contributes its widest row, plus one
// column per inter-glyph gap.
func BlockWidthMini(text string) int {
	filtered, _ := sanitizeLogo(text)
	if filtered == "" {
		return 0
	}
	total := 0
	for _, ch := range filtered {
		g := miniGlyph(ch)
		w := 0
		for r := 0; r < miniGlyphHeight; r++ {
			if l := utf8.RuneCountInString(strings.TrimRight(g[r], " ")); l > w {
				w = l
			}
		}
		total += w
	}
	return total + len([]rune(filtered)) - 1
}

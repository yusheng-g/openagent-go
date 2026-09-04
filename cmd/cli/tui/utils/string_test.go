package utils

import (
	"strings"
	"testing"
)

func TestDisplayWidth(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"Hello", 5},
		{"你好", 4},
		{"Hello你好", 9},
		{"a\x1b[31mb\x1b[0m", 2}, // ANSI stripped
		{"✓✗○●▶", 8},             // ✓✗ wide 1, ○●▶ wide 2
	}
	for _, c := range cases {
		if got := DisplayWidth(c.in); got != c.want {
			t.Errorf("DisplayWidth(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestTruncateByWidth(t *testing.T) {
	cases := []struct {
		in       string
		maxWidth int
		want     string
	}{
		{"short", 10, "short"},
		// NB: the ellipsis "…" is 2 columns wide, so maxWidth 5 leaves 3.
		{"你好世界", 5, "你…"},                // 2 + 2(ellipsis) = 4 ≤ 5
		{"abcdefgh", 5, "abc…"},          // 3 + 2(ellipsis) = 5
		{"\x1b[31mabc\x1b[0m", 5, "abc"}, // styling stripped, no ellipsis needed
	}
	for _, c := range cases {
		if got := TruncateByWidth(c.in, c.maxWidth); got != c.want {
			t.Errorf("TruncateByWidth(%q, %d) = %q, want %q", c.in, c.maxWidth, got, c.want)
		}
	}
}

func TestTruncateStyled(t *testing.T) {
	styled := "\x1b[38;2;0;122;255;48;2;45;45;48m┃\x1b[48;2;45;45;48m\x1b[48;2;45;45;48m 内容\x1b[m"
	for _, mw := range []int{1, 3, 7} {
		out := TruncateStyled(styled, mw)
		if dw := DisplayWidth(out); dw > mw {
			t.Errorf("TruncateStyled(…, %d) width = %d, want ≤ %d", mw, dw, mw)
		}
		// A reset must close the row so no style leaks onto the next line.
		if !strings.HasSuffix(out, "\x1b[m") {
			t.Errorf("TruncateStyled(…, %d) must end in a reset, got %q", mw, out)
		}
	}
	// The rail color of a surviving cell must be preserved, not stripped.
	if out := TruncateStyled(styled, 7); !strings.Contains(out, "\x1b[38;2;0;122;255") {
		t.Errorf("TruncateStyled dropped the rail color: %q", out)
	}
	// A row that already fits comes back untouched.
	if got := TruncateStyled(styled, 99); got != styled {
		t.Errorf("TruncateStyled should pass through fitting rows")
	}
}

func TestStripANSI(t *testing.T) {
	in := "\x1b[38;2;255;255;255m█\x1b[0m"
	if got := StripANSI(in); got != "█" {
		t.Errorf("StripANSI(%q) = %q, want %q", in, got, "█")
	}
}

func TestUnifiedEndOfLine(t *testing.T) {
	in := "line1\r\nline2\rline3"
	if got := UnifiedEndOfLine(in); got != "line1\nline2\nline3" {
		t.Errorf("UnifiedEndOfLine(%q) = %q", in, got)
	}
}

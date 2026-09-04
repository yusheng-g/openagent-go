package components

import (
	"strings"
	"testing"
)

func TestRenderBlockMiniOpenAgent(t *testing.T) {
	got := RenderBlockMini("openagent")
	want := strings.Join([]string{
		"█▀▀█ █▀▀█ █▀▀█ █▀▀▄ ▄▀▀▄ █▀▀█ █▀▀█ █▀▀▄ ▀█▀▀",
		"█  █ █  █ █▀▀▀ █  █ █▀▀█ ▀▀▀█ █▀▀▀ █  █  █",
		"▀▀▀▀ █▀▀▀ ▀▀▀▀ ▀  ▀ ▀  ▀ ▀▀▀▀ ▀▀▀▀ ▀  ▀  ▀",
	}, "\n")
	if got != want {
		t.Fatalf("RenderBlockMini(openagent)\n got: %q\nwant: %q", got, want)
	}
	if n := len(strings.Split(got, "\n")); n != 3 {
		t.Fatalf("RenderBlockMini(openagent) rows = %d, want 3", n)
	}
	if BlockWidthMini("openagent") != 44 {
		t.Fatalf("BlockWidthMini(openagent) = %d, want 44", BlockWidthMini("openagent"))
	}
}

func TestRenderBlockMiniLowercase(t *testing.T) {
	// Lowercase runes use their uppercase face except where a dedicated
	// lowercase letterform exists ('g'), so the logo's 'g' differs from 'G'.
	low := RenderBlockMini("openagent")
	up := RenderBlockMini("OPENAGENT")
	if !strings.Contains(low, "▀▀▀█") {
		t.Errorf("lowercase 'g' should carry its double-storey bar (▀▀▀█):\n%s", low)
	}
	if !strings.Contains(up, "█ ▀█") {
		t.Errorf("uppercase 'G' should keep the spur form (█ ▀█):\n%s", up)
	}
}

func TestRenderBlockMiniSanitize(t *testing.T) {
	got := RenderBlockMini("OPEN-AGENT!")
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("rows = %d, want 3", len(lines))
	}
	if strings.Contains(got, "-") || strings.Contains(got, "!") {
		t.Fatalf("sanitized art still contains symbols: %q", got)
	}
	// "-" collapses to a single space glyph between N and A (two columns of
	// gap), which the sanitizer keeps as one blank glyph.
	if !strings.Contains(lines[0], "█▀▀▄  ▄▀▀▄") {
		t.Fatalf("expected collapsed space glyph, got row: %q", lines[0])
	}
	if w := BlockWidthMini("OPEN AGENT"); w != 45 {
		t.Fatalf("BlockWidthMini(OPEN AGENT) = %d, want 45", w)
	}
}

func TestRenderBlockMiniNoTrailingSpace(t *testing.T) {
	for _, line := range strings.Split(RenderBlockMini("OPENAGENT XYZ 012"), "\n") {
		if strings.HasSuffix(line, " ") {
			t.Fatalf("row has trailing padding: %q", line)
		}
	}
}

func TestGetLogoMiniArt(t *testing.T) {
	name := "openagent" // version default; keep in sync with version.Name
	art := RenderBlockMini(name)
	if got := GetLogo(72); got != art {
		t.Fatalf("GetLogo(72) = %q, want %q", got, art)
	}
	if got := GetLogo(10); got != name {
		t.Fatalf("GetLogo(10) = %q, want bare name %q", got, name)
	}
}

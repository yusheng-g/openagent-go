package components

import (
	"strings"

	"github.com/yusheng-g/openagent-go/version"
)

// Package-level logo handling. The built-in logo renders version.Name in
// the classic 3-row × 4-col half-block banner face (█ ▀ ▄). A user override
// (SetLogo) replaces it verbatim. Coloring (single color or gradient) is
// applied by the caller in view_welcome.go, so GetLogo returns unstyled art.

// logoOverride holds a user-supplied logo (raw multi-line string). When
// non-empty, GetLogo returns it verbatim instead of the built-in art.
var logoOverride string

// SetLogo overrides the built-in welcome-page logo. A multi-line string
// (newline-separated); an empty string keeps the default (version.Name as
// block-font art). Call once at TUI startup, before the first render.
func SetLogo(logo string) {
	logoOverride = strings.TrimSpace(logo)
}

// GetLogo returns the logo as unstyled multi-line art. When a user override
// is set it is returned as-is; otherwise version.Name is rendered with the
// 3-row half-block banner face (kept small so the welcome page stays airy).
// The name keeps its original case — glyphs are uppercase by default, with
// lowercase letterforms (e.g. 'g') where defined. width is the available
// content width — if the block art would overflow it, the bare name string
// is returned instead so it never wraps. width <= 0 skips the check.
func GetLogo(width int) string {
	if logoOverride != "" {
		return logoOverride
	}

	name := version.Name
	art := RenderBlockMini(name)
	if width > 0 && BlockWidthMini(name) > width {
		// Too wide for the terminal — fall back to the plain name.
		return name
	}
	return art
}

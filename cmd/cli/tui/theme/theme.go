package theme

import (
	"fmt"
	"image/color"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
)

// Package theme holds the TUI color palette and base styles.

var (
	// Color Design
	BgNormal    = lipgloss.Color("#000000") // page background (pure black)
	BgSecondary = lipgloss.Color("#141414") // sidebar
	BgSurface   = lipgloss.Color("#1c1c1c") // input + message blocks (opencode-matched)
	BgPanel     = lipgloss.Color("#141414") // floating panels (popup black)
	BgGray      = lipgloss.Color("#888B7E")

	BorderGray = lipgloss.Color("#484848")

	TextNormal   = lipgloss.Color("#fdfcfc") // user text, titles
	TextAsh      = lipgloss.Color("#9a9898") // agent output
	TextStone    = lipgloss.Color("#6e6e73") // thought text
	TextMute     = lipgloss.Color("#646262")
	TextBody     = lipgloss.Color("#424245")
	TextCharcoal = lipgloss.Color("#302c2c")
	TextInk      = lipgloss.Color("#201d1d")

	// LogoColor is the welcome-page logo foreground. Defaults to TextAsh
	// (matches the built-in logo); overridable via settings.json
	// tui.colors.logo_color.
	LogoColor = TextAsh

	Primary = lipgloss.Color("#007aff")
	Danger  = lipgloss.Color("#ff3b30")
	Success = lipgloss.Color("#30d158")
	Warning = lipgloss.Color("#ff9f0a")

	// Notify is the transient-toast color (cyan per the feature spec).
	Notify = lipgloss.Color("#00d7d0")

	WarningHover  = lipgloss.Color("#cc7f08")
	WarningActive = lipgloss.Color("#995f06")

	// scrollbar colors
	ThumbBackGround = lipgloss.Color("#484848")
	TrackBackGround = BgSecondary

	ThumbBackGroundActive = lipgloss.Color("#545454")
	TrackBackGroundActive = BgSurface
	ThumbBackGroundDrag   = lipgloss.Color("#666666")

	// command palette colors
	CommandActive   = lipgloss.Color("#fab283")
	CommandInactive = lipgloss.Color("#995f06")
)

func BaseStyle() lipgloss.Style {
	return lipgloss.NewStyle().Background(BgNormal).Foreground(TextNormal)
}

func ButtonStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(BgGray).
		Foreground(TextNormal).
		Padding(0, 1).
		MarginBackground(BgNormal) // set margin background so margins match the page
}

func ActiveButtonStyle() lipgloss.Style {
	return ButtonStyle().
		Background(Primary).
		Foreground(TextNormal).
		Underline(true)
}

func HelpLabel() lipgloss.Style {
	return lipgloss.NewStyle().Background(BgNormal).Foreground(TextStone).Padding(0, 1)
}

// Reset restores the built-in palette, undoing any ApplyOverrides calls
// (used by the live /theme presets to return to the default scheme).
func Reset() {
	BgNormal = lipgloss.Color("#000000")
	BgSecondary = lipgloss.Color("#141414")
	BgSurface = lipgloss.Color("#1c1c1c")
	BgPanel = lipgloss.Color("#141414")
	BgGray = lipgloss.Color("#888B7E")
	BorderGray = lipgloss.Color("#484848")
	TextNormal = lipgloss.Color("#fdfcfc")
	TextAsh = lipgloss.Color("#9a9898")
	TextStone = lipgloss.Color("#6e6e73")
	TextMute = lipgloss.Color("#646262")
	TextBody = lipgloss.Color("#424245")
	TextCharcoal = lipgloss.Color("#302c2c")
	TextInk = lipgloss.Color("#201d1d")
	LogoColor = lipgloss.Color("#9a9898")
	Primary = lipgloss.Color("#007aff")
	Danger = lipgloss.Color("#ff3b30")
	Success = lipgloss.Color("#30d158")
	Warning = lipgloss.Color("#ff9f0a")
	Notify = lipgloss.Color("#00d7d0")
	WarningHover = lipgloss.Color("#cc7f08")
	WarningActive = lipgloss.Color("#995f06")
	ThumbBackGround = lipgloss.Color("#484848")
	TrackBackGround = lipgloss.Color("#141414")
	ThumbBackGroundActive = lipgloss.Color("#545454")
	TrackBackGroundActive = lipgloss.Color("#1c1c1c")
	ThumbBackGroundDrag = lipgloss.Color("#666666")
	CommandActive = lipgloss.Color("#fab283")
	CommandInactive = lipgloss.Color("#995f06")
}

// ApplyOverrides merges hex color overrides onto the package-level palette
// vars. Keys map to the config.TUIColors JSON field names; an empty value
// (or absent key) keeps the built-in default. Call once at TUI startup,
// before any BaseStyle() call, so the overridden vars are picked up.
//
// Accepted keys: bg_normal, bg_secondary, bg_surface, primary, success,
// warning, danger, text_normal, text_ash, border_gray, logo_color.
// bg_panel is intentionally not overridable: popup panels always use the
// built-in popup black so they stay a distinct layer on any page palette.
//
// logo_color defaults to TextAsh; when text_ash is overridden but
// logo_color is not, the logo follows the new TextAsh. When logo_color is
// set explicitly, it wins.
func ApplyOverrides(overrides map[string]string) {
	logoSet := false
	for key, val := range overrides {
		if val == "" {
			continue
		}
		c := lipgloss.Color(val)
		switch key {
		case "bg_normal":
			BgNormal = c
		case "bg_secondary":
			BgSecondary = c
		case "bg_surface":
			BgSurface = c
		case "primary":
			Primary = c
		case "success":
			Success = c
		case "warning":
			Warning = c
		case "danger":
			Danger = c
		case "text_normal":
			TextNormal = c
		case "text_ash":
			TextAsh = c
		case "border_gray":
			BorderGray = c
		case "logo_color":
			LogoColor = c
			logoSet = true
		}
	}
	// If logo_color wasn't explicitly set, re-bind it to the (possibly
	// overridden) TextAsh so the logo tracks text_ash changes.
	if !logoSet {
		LogoColor = TextAsh
	}
}

// colorVar is the element type of the palette vars (color.Color), kept as
// a compile-time assertion that overrides target the right type.
var _ color.Color = BgNormal

var sgrResetRe = regexp.MustCompile(`\x1b\[(0(;\d+)*)?m`)

func colorBgCode(c color.Color) string {
	rgbC := color.RGBAModel.Convert(c)
	rgba, ok := rgbC.(color.RGBA)
	if !ok {
		return ""
	}
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", rgba.R, rgba.G, rgba.B)
}

// PaintBackground forces bg onto every cell of s that has no explicit
// background. lipgloss pads short rows in JoinVertical/JoinHorizontal with
// plain spaces, and those cells would otherwise show the terminal's default
// color through the UI. Each line starts with the bg and the fill is
// re-applied after every SGR reset, so later explicit backgrounds (message
// blocks, input box, popup panels) still win.
func PaintBackground(s string, bg color.Color) string {
	bgCode := colorBgCode(bg)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		filled := bgCode + line
		filled = sgrResetRe.ReplaceAllStringFunc(filled, func(match string) string {
			return match + bgCode
		})
		lines[i] = filled
	}
	return strings.Join(lines, "\n")
}

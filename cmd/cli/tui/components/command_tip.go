package components

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/yusheng-g/openagent-go/cmd/cli/tui/theme"
)

func RenderCommandTip(cmd string, desc string) string {
	return theme.BaseStyle().Foreground(theme.TextNormal).Render(" "+cmd) + theme.BaseStyle().Foreground(theme.TextAsh).Render(" "+desc)
}

func RenderCommandTipSecondary(cmd string, desc string) string {
	return theme.BaseStyle().Background(theme.BgSecondary).Foreground(theme.TextNormal).Render(" "+cmd) + theme.BaseStyle().Background(theme.BgSecondary).Foreground(theme.TextAsh).Render(" "+desc)
}

// RenderCommandTipSurface styles key hints inside floating panels, which
// sit on the popup background.
func RenderCommandTipSurface(cmd string, desc string) string {
	return RenderCommandTipOn(cmd, desc, theme.BgPanel)
}

// RenderCommandTipOn styles key hints on an explicit popup background, so
// panels with a non-standard surface (e.g. the slash-command card) can keep
// their hint row on the same fill.
func RenderCommandTipOn(cmd string, desc string, bg color.Color) string {
	return theme.BaseStyle().Background(bg).Foreground(theme.TextNormal).Render(" "+cmd) + theme.BaseStyle().Background(bg).Foreground(theme.TextAsh).Render(" "+desc)
}

func NewTips() []Command {
	return []Command{
		{Title: "Switch session", Key: "", Slash: "/sessions", Alias: "", Icon: "", Enable: true, Desc: " to list, pin, and continue sessions"},
		{Title: "Switch model", Key: "", Slash: "/models", Alias: "", Icon: "", Enable: true, Desc: " to see and switch between available AI models"},
	}
}

var tipList = NewTips()
var tipIndex = 0

func NextHelpTip() string {
	nextIndex := tipIndex + 1
	if nextIndex >= len(tipList) {
		nextIndex = 0
	}
	result := RenderHelpTip(tipList[tipIndex])
	tipIndex = nextIndex
	return result
}

func RenderHelpTip(command Command) string {
	return lipgloss.JoinHorizontal(lipgloss.Left,
		theme.BaseStyle().Render("Run "),
		theme.BaseStyle().Foreground(theme.CommandActive).Render(command.Slash),
		theme.BaseStyle().Render(command.Desc),
	)
}

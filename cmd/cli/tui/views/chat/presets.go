package chat

import (
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/yusheng-g/openagent-go/cmd/cli/config"
	"github.com/yusheng-g/openagent-go/cmd/cli/tui/theme"
)

// ── theme presets (/theme) ──

// themePreset is one selectable color scheme. A nil colors map restores the
// built-in palette via theme.Reset(); otherwise theme.ApplyOverrides applies
// the hex overrides live (237 colors are switched by the next render).
type themePreset struct {
	name   string
	colors map[string]string
}

var themePresets = []themePreset{
	{"default", nil},
	{"light", map[string]string{
		"bg_normal":    "#f2f2f7",
		"bg_secondary": "#e5e5ea",
		"bg_surface":   "#ffffff",
		"primary":      "#1a56db",
		"success":      "#0d7a3d",
		"warning":      "#b3560a",
		"danger":       "#c62828",
		"text_normal":  "#1c1c1e",
		"text_ash":     "#55555b",
		"border_gray":  "#c8c8cc",
	}},
	{"high-contrast", map[string]string{
		"bg_normal":    "#000000",
		"bg_secondary": "#0d0d0d",
		"bg_surface":   "#1a1a1a",
		"primary":      "#00e5ff",
		"success":      "#00ff7f",
		"warning":      "#ffd60a",
		"danger":       "#ff4d4d",
		"text_normal":  "#ffffff",
		"text_ash":     "#d0d0d0",
		"border_gray":  "#666666",
	}},
}

// cycleTheme advances to the next color preset, applies it live and raises
// a toast. Styles are resolved at render time, so the new palette applies
// on the next frame.
func (m *Model) cycleTheme() (tea.Model, tea.Cmd) {
	m.themeIdx = (m.themeIdx + 1) % len(themePresets)
	p := themePresets[m.themeIdx]
	if p.colors != nil {
		theme.ApplyOverrides(p.colors)
	} else {
		theme.Reset()
	}
	m.viewportDirty = true
	m.textareaDirty = true
	return m, m.notify("Theme: " + p.name)
}

// ── plugins panel (/plugins) ──

// pluginsDir returns the directory holding installed plugins. Overridable in
// tests so the panel can point at a fixture tree.
var pluginsDir = func() string { return config.DefaultPluginsDir() }

// setPluginsDir overrides the plugin directory (tests).
func setPluginsDir(p string) { pluginsDir = func() string { return p } }

// openPluginsPanel lists the installed plugins (subdirectories of the
// plugins dir) in a read-only panel.
func (m *Model) openPluginsPanel() (tea.Model, tea.Cmd) {
	m.pluginItems = nil
	if entries, err := os.ReadDir(pluginsDir()); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				m.pluginItems = append(m.pluginItems, e.Name())
			} else if filepath.Ext(e.Name()) == ".json" || filepath.Ext(e.Name()) == ".yaml" {
				m.pluginItems = append(m.pluginItems, e.Name())
			}
		}
	}
	m.panelOpen = true
	m.panelMode = panelModePlugins
	m.panelFromSlash = false // centered
	m.panelIdx = 0
	return m, nil
}

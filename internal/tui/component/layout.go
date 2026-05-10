// Package component contains shared TUI styling and rendering helpers.
// Authored by: OpenCode
package component

import (
	"strings"

	lipgloss "charm.land/lipgloss/v2"
)

// MenuItem describes a renderable primary action in a vertical menu.
//
// Example:
//
//	items := []component.MenuItem{{Label: "Sync Data", Enabled: true}}
//	_ = items
//
// Authored by: OpenCode
type MenuItem struct {
	Label       string
	Enabled     bool
	Description string
}

// RenderMenu renders the current screen's primary vertical menu.
//
// Example:
//
//	menu := component.RenderMenu(component.DefaultTheme(), []component.MenuItem{{Label: "Sync Data", Enabled: true}}, 0)
//	_ = menu
//
// Authored by: OpenCode
func RenderMenu(theme Theme, items []MenuItem, selected int) string {
	var lines []string
	for index, item := range items {
		var prefix = "  "
		var style = theme.BodyText
		if index == selected {
			prefix = "> "
			style = theme.SelectedItem
		}
		if !item.Enabled {
			prefix = "x "
			style = theme.DisabledItem
		}

		var line = prefix + item.Label
		if item.Description != "" {
			line += " - " + item.Description
		}
		lines = append(lines, style.Render(line))
	}
	return strings.Join(lines, "\n")
}

// RenderScreen composes a full-screen layout with header, body, status, and
// footer regions.
//
// Example:
//
//	view := component.RenderScreen(component.DefaultTheme(), 100, 32, "Title", "Subtitle", "Body", "Status", "Help")
//	_ = view
//
// Authored by: OpenCode
func RenderScreen(theme Theme, width int, height int, title string, subtitle string, body string, status string, footer string) string {
	if width <= 0 {
		width = 100
	}
	if height <= 0 {
		height = 32
	}

	var header = theme.Panel.Width(width - 4).Render(strings.TrimSpace(theme.Title.Render(title) + "\n" + theme.Subtitle.Render(subtitle)))
	var bodyPanel = theme.SummaryPanel.Width(width - 4).Render(body)
	var statusPanel = theme.Panel.Width(width - 4).Render(status)
	var footerPanel = theme.Panel.Width(width - 4).Render(footer)

	var content = lipgloss.JoinVertical(lipgloss.Left, header, bodyPanel, statusPanel, footerPanel)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Top, content)
}

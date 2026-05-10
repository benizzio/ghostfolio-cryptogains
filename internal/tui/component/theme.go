// Package component contains shared TUI styling and rendering helpers.
// Authored by: OpenCode
package component

import lipgloss "charm.land/lipgloss/v2"

// Theme defines the shared visual language used across the full-screen TUI.
//
// Example:
//
//	theme := component.DefaultTheme()
//	_ = theme.Title
//
// Authored by: OpenCode
type Theme struct {
	Title         lipgloss.Style
	Subtitle      lipgloss.Style
	Panel         lipgloss.Style
	SelectedItem  lipgloss.Style
	DisabledItem  lipgloss.Style
	BodyText      lipgloss.Style
	MutedText     lipgloss.Style
	InputLabel    lipgloss.Style
	SuccessStatus lipgloss.Style
	FailureStatus lipgloss.Style
	NeutralStatus lipgloss.Style
	HelpText      lipgloss.Style
	SummaryPanel  lipgloss.Style
}

// DefaultTheme returns the Ghostfolio-inspired TUI theme used by this slice.
//
// Example:
//
//	theme := component.DefaultTheme()
//	_ = theme.SelectedItem
//
// Authored by: OpenCode
func DefaultTheme() Theme {
	var panel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3686cf")).
		Padding(1, 2)

	return Theme{
		Title:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#36cfcc")),
		Subtitle:     lipgloss.NewStyle().Foreground(lipgloss.Color("#3686cf")),
		Panel:        panel,
		SelectedItem: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#36cfcc")),
		DisabledItem: lipgloss.NewStyle().Faint(true),
		BodyText:     lipgloss.NewStyle(),
		MutedText:    lipgloss.NewStyle().Faint(true),
		InputLabel:   lipgloss.NewStyle().Bold(true),
		SuccessStatus: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#36cfcc")),
		FailureStatus: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#dc3545")),
		NeutralStatus: lipgloss.NewStyle().Foreground(lipgloss.Color("#3686cf")),
		HelpText:      lipgloss.NewStyle().Faint(true),
		SummaryPanel:  panel.BorderForeground(lipgloss.Color("#36cfcc")),
	}
}

// Package screen renders full-screen workflow states for the terminal
// application.
// Authored by: OpenCode
package screen

import (
	"fmt"

	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

// MainMenuScreenParams contains the render state for the main menu.
//
// Example:
//
//	view := screen.MainMenuScreenView(screen.MainMenuScreenParams{Theme: component.DefaultTheme(), Width: 100, Height: 32})
//	_ = view
//
// Authored by: OpenCode
type MainMenuScreenParams struct {
	Theme         component.Theme
	Width         int
	Height        int
	MenuItems     []component.MenuItem
	SelectedIndex int
	ServerOrigin  string
	HelpText      string
}

// MainMenuScreenView renders the main menu for the current slice.
//
// Example:
//
//	view := screen.MainMenuScreenView(params)
//	_ = view
//
// Authored by: OpenCode
func MainMenuScreenView(params MainMenuScreenParams) string {
	var body = fmt.Sprintf(
		"Selected Server: %s\nSetup Status: complete\n\n%s",
		params.ServerOrigin,
		component.RenderMenu(params.Theme, params.MenuItems, params.SelectedIndex),
	)

	return component.RenderScreen(
		params.Theme,
		params.Width,
		params.Height,
		"Main Menu",
		"Sync Data is the only business workflow available in this release.",
		body,
		"Choose Sync Data to validate Ghostfolio communication. Persistence and report generation are not available in this slice.",
		params.HelpText,
	)
}

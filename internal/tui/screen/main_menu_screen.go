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
// Supply the shared `Theme`, terminal dimensions, current menu selection, the
// active server summary, and footer help text that should be visible while the
// main menu is active. The screen renderer reads these values only and does not
// own any workflow state transitions.
//
// Authored by: OpenCode
type MainMenuScreenParams struct {
	Theme               component.Theme
	Width               int
	Height              int
	MenuItems           []component.MenuItem
	SelectedIndex       int
	ServerOrigin        string
	ProtectedDataExists bool
	HelpText            string
}

// MainMenuScreenView renders the main menu for the current slice.
//
// Example:
//
//	view := screen.MainMenuScreenView(params)
//	_ = view
//
// `MainMenuScreenView` formats the startup-complete state, the selected server
// summary, the primary menu, and the footer help into the shared full-screen
// layout. Use it after setup has completed and the application should expose
// `Sync Data` as the only available business workflow.
//
// Authored by: OpenCode
func MainMenuScreenView(params MainMenuScreenParams) string {
	var body = fmt.Sprintf(
		"Selected Server: %s\nSetup Status: complete\nProtected Data: %s\n\n%s",
		params.ServerOrigin,
		protectedDataStatusLabel(params.ProtectedDataExists),
		component.RenderMenu(params.Theme, params.MenuItems, params.SelectedIndex),
	)

	return component.RenderScreen(
		params.Theme,
		params.Width,
		params.Height,
		"Main Menu",
		"Sync Data is the only business workflow available in this release.",
		body,
		"Choose Sync Data to authenticate, retrieve, validate, and store protected activity history. Protected-data presence may be shown, but no cached activity details are exposed in this slice.",
		params.HelpText,
	)
}

// protectedDataStatusLabel formats the active readable-protected-data summary without exposing activity details.
// Authored by: OpenCode
func protectedDataStatusLabel(present bool) string {
	if present {
		return "readable snapshot loaded for this run"
	}
	return "none loaded for this run"
}

// Package screen renders full-screen workflow states for the terminal
// application.
// Authored by: OpenCode
package screen

import (
	"fmt"

	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

// ServerReplacementScreenParams contains the render state for the server-
// mismatch confirmation screen.
//
// Supply the currently loaded protected-data server, the newly selected setup
// server, the confirmation menu state, and footer help text. The renderer uses
// these values to explain the replacement boundary without owning the decision
// logic or starting sync work itself.
//
// Authored by: OpenCode
type ServerReplacementScreenParams struct {
	Theme         component.Theme
	Width         int
	Height        int
	MenuItems     []component.MenuItem
	SelectedIndex int
	CurrentServer string
	NewServer     string
	HelpText      string
}

// ServerReplacementScreenView renders the explicit server-replacement
// confirmation workflow.
//
// Example:
//
//	view := screen.ServerReplacementScreenView(params)
//	_ = view
//
// Use this renderer after runtime detects that the selected setup server does
// not match the readable protected snapshot already loaded for the current run.
// It explains that existing protected data remains unchanged unless the
// replacement sync later completes successfully.
//
// Authored by: OpenCode
func ServerReplacementScreenView(params ServerReplacementScreenParams) string {
	var body = fmt.Sprintf(
		"Current Protected Server: %s\nSelected Server: %s\n\nContinuing will replace the current protected data tied to that token and server only after the replacement sync completes successfully.\n\n%s",
		params.CurrentServer,
		params.NewServer,
		component.RenderMenu(params.Theme, params.MenuItems, params.SelectedIndex),
	)

	return component.RenderScreen(
		params.Theme,
		params.Width,
		params.Height,
		"Server Replacement",
		"Review the server mismatch before starting replacement sync.",
		body,
		"Cancel leaves the active readable protected data unchanged.",
		params.HelpText,
	)
}

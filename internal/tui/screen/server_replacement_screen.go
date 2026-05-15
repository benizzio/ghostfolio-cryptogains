// Package screen renders full-screen workflow states for the terminal
// application.
// Authored by: OpenCode
package screen

import (
	"fmt"

	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

// ServerReplacementScreenParams contains the render state for the server-mismatch confirmation screen.
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

// ServerReplacementScreenView renders the explicit server-replacement confirmation workflow.
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

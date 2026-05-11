// Package screen renders full-screen workflow states for the terminal
// application.
// Authored by: OpenCode
package screen

import (
	"strings"

	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

// SetupScreenParams contains the render state for the setup screen.
//
// Populate this value with the current setup menu, optional custom-origin input
// state, any startup invalidation message, and the footer help text for the
// setup workflow. The renderer treats the struct as immutable view data and
// expects the caller to decide whether saving is currently allowed.
//
// Authored by: OpenCode
type SetupScreenParams struct {
	Theme               component.Theme
	Width               int
	Height              int
	MenuItems           []component.MenuItem
	SelectedIndex       int
	ShowOriginInput     bool
	OriginInput         string
	InvalidSetupMessage string
	ValidationMessage   string
	HelpText            string
	CanSave             bool
}

// SetupScreenView renders the initial setup screen.
//
// Example:
//
//	view := screen.SetupScreenView(params)
//	_ = view
//
// `SetupScreenView` formats the server-selection workflow into the shared
// full-screen layout, including the primary menu, optional custom-origin input,
// validation messaging, and footer help. Use it whenever setup must be
// completed or repaired before `Sync Data` can run.
//
// Authored by: OpenCode
func SetupScreenView(params SetupScreenParams) string {
	var bodyParts = []string{
		"Choose which Ghostfolio server this machine-local setup should remember.",
		component.RenderMenu(params.Theme, params.MenuItems, params.SelectedIndex),
		params.Theme.MutedText.Render("Production-like custom origins require https. http is allowed only in explicit development mode."),
	}
	if params.ShowOriginInput {
		bodyParts = append(
			bodyParts,
			params.Theme.InputLabel.Render("Custom Server Origin"),
			params.OriginInput,
		)
	}
	if params.InvalidSetupMessage != "" {
		bodyParts = append([]string{params.Theme.FailureStatus.Render(params.InvalidSetupMessage)}, bodyParts...)
	}

	var status = "Setup must be completed before Sync Data can run."
	if params.ValidationMessage != "" {
		status = params.ValidationMessage
	}
	if !params.CanSave {
		status += " Save And Continue stays disabled until the selected origin is valid."
	}

	return component.RenderScreen(
		params.Theme,
		params.Width,
		params.Height,
		"Setup",
		"Select the Ghostfolio server for this slice.",
		strings.Join(bodyParts, "\n\n"),
		status,
		params.HelpText,
	)
}

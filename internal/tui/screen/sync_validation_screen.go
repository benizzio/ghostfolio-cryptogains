// Package screen renders full-screen workflow states for the terminal
// application.
// Authored by: OpenCode
package screen

import (
	"fmt"
	"strings"

	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

// SyncValidationScreenParams contains the render state for the sync-validation
// entry screen.
//
// Supply the current token input rendering, the primary menu state, any
// validation message, and busy-state details when a communication check is in
// flight. The renderer uses these fields to switch between the idle entry view
// and the busy progress view without mutating workflow state.
//
// Authored by: OpenCode
type SyncValidationScreenParams struct {
	Theme               component.Theme
	Width               int
	Height              int
	MenuItems           []component.MenuItem
	SelectedIndex       int
	TokenInput          string
	ValidationMessage   string
	HelpText            string
	Busy                bool
	BusyText            string
	SpinnerFrame        string
	ProtectedDataExists bool
}

// SyncValidationScreenView renders the sync-data entry screen.
//
// Example:
//
//	view := screen.SyncValidationScreenView(params)
//	_ = view
//
// `SyncValidationScreenView` renders the token-entry workflow for `Sync Data`.
// It shows the runtime-only security-token field, the primary action menu when
// idle, or the progress indicator when sync and protected storage are running.
//
// Authored by: OpenCode
func SyncValidationScreenView(params SyncValidationScreenParams) string {
	var bodyParts = []string{
		"The application will authenticate, retrieve activity history, validate it, and store it securely for future use only.",
		fmt.Sprintf("Protected Data Loaded For This Run: %s", syncProtectedDataStatusLabel(params.ProtectedDataExists)),
		params.Theme.InputLabel.Render("Ghostfolio Security Token"),
		params.TokenInput,
	}

	if params.Busy {
		bodyParts = append(bodyParts, fmt.Sprintf("%s %s", params.SpinnerFrame, params.BusyText))
	} else {
		bodyParts = append(bodyParts, component.RenderMenu(params.Theme, params.MenuItems, params.SelectedIndex))
	}

	var status = "Enter the Ghostfolio security token only when starting Sync Data."
	if params.ValidationMessage != "" {
		status = params.ValidationMessage
	}

	return component.RenderScreen(
		params.Theme,
		params.Width,
		params.Height,
		"Sync Data",
		"Retrieve, validate, and securely store supported activity history.",
		strings.Join(bodyParts, "\n\n"),
		status,
		params.HelpText,
	)
}

// syncProtectedDataStatusLabel formats the sync-entry protected-data status without exposing cached activity details.
// Authored by: OpenCode
func syncProtectedDataStatusLabel(present bool) string {
	if present {
		return "yes"
	}
	return "no"
}

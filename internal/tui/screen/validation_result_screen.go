// Package screen renders full-screen workflow states for the terminal
// application.
// Authored by: OpenCode
package screen

import (
	"fmt"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

// ValidationResultScreenParams contains the render state for the validation
// result screen.
//
// Example:
//
//	view := screen.ValidationResultScreenView(screen.ValidationResultScreenParams{Theme: component.DefaultTheme(), Width: 100, Height: 32, Outcome: runtime.ValidationOutcome{Success: true}})
//	_ = view
//
// Authored by: OpenCode
type ValidationResultScreenParams struct {
	Theme         component.Theme
	Width         int
	Height        int
	MenuItems     []component.MenuItem
	SelectedIndex int
	Outcome       runtime.ValidationOutcome
	HelpText      string
}

// ValidationResultScreenView renders the result of a completed validation
// attempt.
//
// Example:
//
//	view := screen.ValidationResultScreenView(params)
//	_ = view
//
// Authored by: OpenCode
func ValidationResultScreenView(params ValidationResultScreenParams) string {
	var resultLine = params.Theme.SuccessStatus.Render("Success")
	if !params.Outcome.Success {
		resultLine = params.Theme.FailureStatus.Render(fmt.Sprintf("Failure Category: %s", params.Outcome.FailureCategory))
	}

	var body = fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s",
		resultLine,
		params.Outcome.SummaryMessage,
		params.Outcome.FollowUpNote,
		component.RenderMenu(params.Theme, params.MenuItems, params.SelectedIndex),
	)

	return component.RenderScreen(
		params.Theme,
		params.Width,
		params.Height,
		"Validation Result",
		"Review the communication outcome and choose the next step.",
		body,
		"Results are transient and are not shown again after restart.",
		params.HelpText,
	)
}

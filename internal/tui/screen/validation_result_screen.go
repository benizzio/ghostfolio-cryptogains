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
// Populate this value with the final validation outcome, the result-menu state,
// and footer help text for the result workflow. The renderer converts the
// structured outcome into user-facing summary and follow-up text without owning
// any retry or navigation behavior.
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
// `ValidationResultScreenView` formats the completed communication outcome,
// follow-up guidance, and available next actions into the shared full-screen
// layout. Use it after a validation attempt finishes so the user can retry or
// return to the main menu without persisting any remote data.
//
// Authored by: OpenCode
func ValidationResultScreenView(params ValidationResultScreenParams) string {
	var resultLine = params.Theme.SuccessStatus.Render("Success")
	if !params.Outcome.Success {
		resultLine = params.Theme.FailureStatus.Render(
			fmt.Sprintf(
				"Failure Category: %s",
				params.Outcome.FailureReason,
			),
		)
	}

	var body = fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s",
		resultLine,
		validationSummaryText(params.Outcome),
		validationFollowUpText(params.Outcome),
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

// validationSummaryText converts the structured validation outcome into the
// primary result text shown on the TUI result screen.
// Authored by: OpenCode
func validationSummaryText(outcome runtime.ValidationOutcome) string {
	if outcome.Success {
		return "Communication with the selected Ghostfolio server is working."
	}
	return "Communication validation did not succeed."
}

// validationFollowUpText converts the structured validation outcome into the
// secondary guidance shown on the TUI result screen.
// Authored by: OpenCode
func validationFollowUpText(outcome runtime.ValidationOutcome) string {
	if outcome.Success {
		return "No Ghostfolio data was stored locally, and reporting is not available in this slice."
	}

	switch outcome.FailureReason {
	case runtime.ValidationFailureIncompatibleServerContract:
		return "The selected server responded, but it did not satisfy the supported contract for this slice."
	default:
		return "Validate again or return to the main menu. No Ghostfolio data was stored locally."
	}
}

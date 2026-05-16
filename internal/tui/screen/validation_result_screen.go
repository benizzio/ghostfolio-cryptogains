// Package screen renders full-screen workflow states for the terminal
// application.
// Authored by: OpenCode
package screen

import (
	"fmt"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

// SyncResultScreenParams contains the render state for the sync result screen.
//
// Populate this value with the final validation outcome, the result-menu state,
// and footer help text for the result workflow. The renderer converts the
// structured outcome into user-facing summary and follow-up text without owning
// any retry or navigation behavior.
//
// Authored by: OpenCode
type SyncResultScreenParams struct {
	Theme         component.Theme
	Width         int
	Height        int
	MenuItems     []component.MenuItem
	SelectedIndex int
	Outcome       runtime.SyncOutcome
	HelpText      string
}

// SyncResultScreenView renders the result of a completed sync attempt.
//
// Example:
//
//	view := screen.SyncResultScreenView(params)
//	_ = view
//
// `SyncResultScreenView` formats the completed sync outcome, follow-up
// guidance, and available next actions into the shared full-screen layout.
//
// Authored by: OpenCode
func SyncResultScreenView(params SyncResultScreenParams) string {
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
		"Sync Result",
		"Review the protected-storage outcome and choose the next step.",
		body,
		"Results are transient and are not shown again after restart.",
		params.HelpText,
	)
}

// validationSummaryText converts the structured sync outcome into the
// primary result text shown on the TUI result screen.
// Authored by: OpenCode
func validationSummaryText(outcome runtime.SyncOutcome) string {
	if outcome.Success {
		return "Activity data was stored securely for future use."
	}
	return "Sync and secure storage did not succeed."
}

// validationFollowUpText converts the structured sync outcome into the
// secondary guidance shown on the TUI result screen.
// Authored by: OpenCode
func validationFollowUpText(outcome runtime.SyncOutcome) string {
	if outcome.Success {
		return "No report-generation, report-preview, or cached-data browsing workflow is available in this slice."
	}

	if outcome.Diagnostic.Path != "" {
		return fmt.Sprintf("A synced-data diagnostic report was generated at %s.", outcome.Diagnostic.Path)
	}
	if outcome.Diagnostic.Eligible {
		return "You can generate a synced-data diagnostic report for this failure from this screen."
	}

	switch outcome.FailureReason {
	case runtime.SyncFailureServerReplacementCancelled:
		return "The existing protected data was left unchanged because server replacement was cancelled before retrieval started."
	case runtime.SyncFailureRejectedToken:
		return "The supplied token was rejected. Try again with a valid Ghostfolio security token. Local protected data was left unchanged."
	case runtime.SyncFailureIncompatibleServerContract:
		return "The selected server responded, but it did not satisfy the supported contract for this slice."
	case runtime.SyncFailureUnsupportedStoredDataVersion:
		return "Existing protected data uses an unsupported stored-data version, so it was not loaded or overwritten."
	case runtime.SyncFailureIncompatibleNewSyncData:
		return "The newly retrieved data could not be stored safely, so it was discarded and any previously readable protected data was left unchanged."
	case runtime.SyncFailureUnsupportedActivityHistory:
		return "The retrieved activity history is not supported safely by this slice, so no protected data was stored."
	default:
		return "Sync again or return to the main menu. No protected activity data was stored."
	}
}

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
// Populate this value with the final sync outcome, the result-menu state,
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
	Busy          bool
	StatusMessage string
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
		syncSummaryText(params.Outcome),
		syncFollowUpText(params.Outcome, params.Busy),
		component.RenderMenu(params.Theme, params.MenuItems, params.SelectedIndex),
	)

	var status = resultStatusText(params.Outcome, params.StatusMessage)

	return component.RenderScreen(
		params.Theme,
		params.Width,
		params.Height,
		"Sync Result",
		"Review the protected-storage outcome and choose the next step.",
		body,
		status,
		params.HelpText,
	)
}

// syncSummaryText converts the structured sync outcome into the
// primary result text shown on the TUI result screen.
// Authored by: OpenCode
func syncSummaryText(outcome runtime.SyncOutcome) string {
	if outcome.Success {
		return component.ProtectedDataStoredMessage
	}
	return "Sync and secure storage did not succeed."
}

// syncFollowUpText converts the structured sync outcome into the
// secondary guidance shown on the TUI result screen.
// Authored by: OpenCode
func syncFollowUpText(outcome runtime.SyncOutcome, busy bool) string {
	if outcome.Success {
		return "Return to Sync and Reports to generate a capital gains report from the newly stored protected data."
	}
	if busy {
		return component.SyncDiagnosticGeneratingMessage
	}

	if outcome.Diagnostic.Path != "" {
		return component.SyncDiagnosticReportGeneratedMessage(outcome.Diagnostic.Path)
	}
	if outcome.Diagnostic.Eligible {
		return component.SyncDiagnosticAvailableFromScreenMessage
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

// resultStatusText renders transient result-screen status feedback.
// Authored by: OpenCode
func resultStatusText(outcome runtime.SyncOutcome, statusMessage string) string {
	if statusMessage != "" {
		return statusMessage
	}
	if outcome.Diagnostic.Path != "" {
		return component.ResultsTransientStatusText
	}
	return component.ResultsTransientStatusText
}

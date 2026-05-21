// Package screen renders full-screen workflow states for the terminal
// application.
// Authored by: OpenCode
package screen

import (
	"fmt"
	"strings"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

const syncReportsLastSuccessfulSyncLayout = "2006-01-02 15:04:05 MST"

// SyncReportsScreenParams contains the render state for the Sync and Reports
// workflow screen.
//
// Supply the current selected server, menu state, and protected-data summary
// values that should be visible for the active Sync and Reports context.
//
// Authored by: OpenCode
type SyncReportsScreenParams struct {
	Theme              component.Theme
	Width              int
	Height             int
	MenuItems          []component.MenuItem
	SelectedIndex      int
	ServerOrigin       string
	ProtectedDataState runtime.ProtectedDataState
	SyncOutcome        runtime.SyncOutcome
	Busy               bool
	StatusMessage      string
	UnavailableMessage string
	HelpText           string
}

// SyncReportsScreenView renders the Sync and Reports workflow screen.
//
// Example:
//
//	view := screen.SyncReportsScreenView(params)
//	_ = view
//
// Authored by: OpenCode
func SyncReportsScreenView(params SyncReportsScreenParams) string {
	var readiness = syncReportsReadinessSummary(params.ProtectedDataState, params.UnavailableMessage)
	var syncFeedback = syncReportsSyncFeedback(params.SyncOutcome, params.Busy)
	var body = fmt.Sprintf(
		"Selected Server: %s\n\nProtected Data Readiness\n%s%s\n\n%s",
		params.ServerOrigin,
		readiness,
		syncFeedback,
		component.RenderMenu(params.Theme, params.MenuItems, params.SelectedIndex),
	)

	return component.RenderScreen(
		params.Theme,
		params.Width,
		params.Height,
		"Sync and Reports",
		"Use Sync Data or generate a report from protected synced activity history.",
		body,
		syncReportsStatusText(params.ProtectedDataState, params.SyncOutcome, params.Busy, params.StatusMessage, params.UnavailableMessage),
		params.HelpText,
	)
}

// syncReportsReadinessSummary formats the user-visible protected-data readiness
// summary for the unlocked Sync and Reports context.
// Authored by: OpenCode
func syncReportsReadinessSummary(state runtime.ProtectedDataState, unavailableMessage string) string {
	var lines []string
	if !state.HasReadableSnapshot {
		lines = append(lines, "Sync Data: no synced data available")
		lines = append(lines, fmt.Sprintf("Generate Capital Gains Report: unavailable - %s", syncReportsUnavailableReason(unavailableMessage)))
		return strings.Join(lines, "\n")
	}

	lines = append(lines, fmt.Sprintf("Sync Data: last successful sync %s", formatSyncReportsLastSuccessfulSync(state.LastSuccessfulSyncAt)))
	lines = append(lines, fmt.Sprintf("Protected Activity Count: %d", state.ActivityCount))
	if len(state.AvailableReportYears) == 0 {
		lines = append(lines, fmt.Sprintf("Generate Capital Gains Report: unavailable - %s", syncReportsUnavailableReason(unavailableMessage)))
		return strings.Join(lines, "\n")
	}

	lines = append(lines, fmt.Sprintf("Available Report Years: %s", formatSyncReportsAvailableYears(state.AvailableReportYears)))
	lines = append(lines, "Generate Capital Gains Report: available")
	return strings.Join(lines, "\n")
}

// syncReportsStatusText formats the footer status guidance for the unlocked Sync
// and Reports context.
// Authored by: OpenCode
func syncReportsStatusText(state runtime.ProtectedDataState, outcome runtime.SyncOutcome, busy bool, statusMessage string, unavailableMessage string) string {
	if strings.TrimSpace(statusMessage) != "" {
		return statusMessage
	}
	if busy {
		return "Generating a local synced-data diagnostic report for this failure."
	}
	if outcome.Diagnostic.Path != "" {
		return fmt.Sprintf("A synced-data diagnostic report was generated at %s.", outcome.Diagnostic.Path)
	}
	if outcome.Diagnostic.Eligible {
		return "A synced-data diagnostic report is available for this failure from this context."
	}
	if !outcome.Success && outcome.FailureReason != "" {
		return fmt.Sprintf("The last sync attempt failed with category %s. Sync Data stays available from this context.", outcome.FailureReason)
	}
	if !state.HasReadableSnapshot {
		return fmt.Sprintf("Sync Data is available now. Generate Capital Gains Report stays unavailable until %s.", syncReportsUnavailableReason(unavailableMessage))
	}
	if len(state.AvailableReportYears) == 0 {
		return fmt.Sprintf("Protected synced data is readable, but report generation stays unavailable until %s.", syncReportsUnavailableReason(unavailableMessage))
	}
	return "Protected synced data is ready. Sync Data stays available, and report generation is available from this context."
}

// syncReportsUnavailableReason normalizes the current report-unavailable reason.
// Authored by: OpenCode
func syncReportsUnavailableReason(value string) string {
	if strings.TrimSpace(value) == "" {
		return "no synced data available"
	}
	return value
}

// formatSyncReportsLastSuccessfulSync formats one local timestamp for the
// readiness summary.
// Authored by: OpenCode
func formatSyncReportsLastSuccessfulSync(value time.Time) string {
	if value.IsZero() {
		return "unknown"
	}
	return value.Local().Format(syncReportsLastSuccessfulSyncLayout)
}

// formatSyncReportsAvailableYears formats the available report-year list for the
// readiness summary.
// Authored by: OpenCode
func formatSyncReportsAvailableYears(years []int) string {
	var labels = make([]string, 0, len(years))
	for _, year := range years {
		labels = append(labels, fmt.Sprintf("%d", year))
	}
	return strings.Join(labels, ", ")
}

// syncReportsSyncFeedback formats the latest sync outcome details rendered in
// the active `Sync and Reports` context.
// Authored by: OpenCode
func syncReportsSyncFeedback(outcome runtime.SyncOutcome, busy bool) string {
	if outcome.Attempt.AttemptID == "" && outcome.FailureReason == "" && outcome.DetailReason == "" && !outcome.Success && !outcome.Diagnostic.Eligible && outcome.Diagnostic.Path == "" {
		return ""
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, "Latest Sync Result")
	if outcome.Success {
		lines = append(lines, "Activity data was stored securely for future use.")
		return "\n" + strings.Join(lines, "\n")
	}
	lines = append(lines, fmt.Sprintf("Failure Category: %s", outcome.FailureReason))
	if busy {
		lines = append(lines, "Generating diagnostic report...")
	} else if outcome.Diagnostic.Path != "" {
		lines = append(lines, fmt.Sprintf("A synced-data diagnostic report was generated at %s.", outcome.Diagnostic.Path))
	} else if outcome.Diagnostic.Eligible {
		lines = append(lines, "Generate Diagnostic Report is available for this failure from this context.")
	}

	return "\n" + strings.Join(lines, "\n")
}

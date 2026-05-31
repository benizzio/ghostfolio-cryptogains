// Package component contains shared TUI styling and rendering helpers.
// Authored by: OpenCode
package component

import "fmt"

const (
	// SyncAndReportsActionLabel labels the main workflow entry.
	// Authored by: OpenCode
	SyncAndReportsActionLabel = "Sync and Reports"

	// UnlockActionLabel labels the unlock action.
	// Authored by: OpenCode
	UnlockActionLabel = "Unlock"

	// BackActionLabel labels the generic back action.
	// Authored by: OpenCode
	BackActionLabel = "Back"

	// StartSyncActionLabel labels the standalone sync action.
	// Authored by: OpenCode
	StartSyncActionLabel = "Start Sync"

	// GenerateDiagnosticReportActionLabel labels diagnostic generation.
	// Authored by: OpenCode
	GenerateDiagnosticReportActionLabel = "Generate Diagnostic Report"

	// SyncAgainActionLabel labels the sync retry action.
	// Authored by: OpenCode
	SyncAgainActionLabel = "Sync Again"

	// BackToMainMenuActionLabel labels returning to the main menu.
	// Authored by: OpenCode
	BackToMainMenuActionLabel = "Back To Main Menu"

	// SyncDataActionLabel labels the in-context sync action.
	// Authored by: OpenCode
	SyncDataActionLabel = "Sync Data"

	// GenerateCapitalGainsReportActionLabel labels report generation.
	// Authored by: OpenCode
	GenerateCapitalGainsReportActionLabel = "Generate Capital Gains Report"

	// GenerateReportActionLabel labels starting one report run.
	// Authored by: OpenCode
	GenerateReportActionLabel = "Generate Report"

	// BackToSyncReportsActionLabel labels returning to the unlocked context.
	// Authored by: OpenCode
	BackToSyncReportsActionLabel = "Back To Sync and Reports"

	// GenerateAnotherReportActionLabel labels starting another report selection.
	// Authored by: OpenCode
	GenerateAnotherReportActionLabel = "Generate Another Report"

	// ContinueAndReplaceActionLabel labels the replacement confirmation action.
	// Authored by: OpenCode
	ContinueAndReplaceActionLabel = "Continue And Replace"

	// CancelActionLabel labels the cancellation action.
	// Authored by: OpenCode
	CancelActionLabel = "Cancel"

	// SyncReportsUnlockIntroText explains the one-time unlock entry.
	// Authored by: OpenCode
	SyncReportsUnlockIntroText = "Enter the Ghostfolio security token once to unlock Sync Data and future reporting actions for this run."

	// SyncReportsUnlockIdleStatusText explains the unlock idle state.
	// Authored by: OpenCode
	SyncReportsUnlockIdleStatusText = "Enter the Ghostfolio security token to unlock Sync and Reports for this run."

	// MainMenuSubtitleText explains the main business entry.
	// Authored by: OpenCode
	MainMenuSubtitleText = "Sync and Reports is the business workflow entry for this slice."

	// MainMenuStatusText explains what stays hidden on the main menu.
	// Authored by: OpenCode
	MainMenuStatusText = "Choose Sync and Reports to enter the token-unlocked workflow context. Protected sync metadata and reporting readiness stay hidden on the main menu."

	// ProtectedDataStoredMessage confirms successful protected persistence.
	// Authored by: OpenCode
	ProtectedDataStoredMessage = "Activity data was stored securely for future use."

	// SyncDiagnosticGeneratingMessage explains an in-flight synced-data diagnostic write.
	// Authored by: OpenCode
	SyncDiagnosticGeneratingMessage = "Generating a local synced-data diagnostic report for this failure."

	// SyncDiagnosticAvailableFromContextMessage explains that diagnostics can be generated from the unlocked context.
	// Authored by: OpenCode
	SyncDiagnosticAvailableFromContextMessage = "Generate Diagnostic Report is available for this failure from this context."

	// SyncDiagnosticAvailableFromScreenMessage explains that diagnostics can be generated from the result screen.
	// Authored by: OpenCode
	SyncDiagnosticAvailableFromScreenMessage = "You can generate a synced-data diagnostic report for this failure from this screen."

	// ResultsTransientStatusText explains that sync results are not retained across restarts.
	// Authored by: OpenCode
	ResultsTransientStatusText = "Results are transient and are not shown again after restart."

	// ReportSavedPathsTransientStatusText explains that saved report paths are not retained after dismissal.
	// Authored by: OpenCode
	ReportSavedPathsTransientStatusText = "Saved paths are transient and are cleared when this result is dismissed."

	// ReportDiagnosticAvailableFromScreenMessage explains that report diagnostics can be generated from the report-result screen.
	// Authored by: OpenCode
	ReportDiagnosticAvailableFromScreenMessage = "Generate Diagnostic Report is available for this failure from this screen."
)

// SyncDiagnosticReportGeneratedMessage formats one synced-data diagnostic-report success message.
// Authored by: OpenCode
func SyncDiagnosticReportGeneratedMessage(path string) string {
	return fmt.Sprintf("A synced-data diagnostic report was generated at %s.", path)
}

// Package flow owns the Bubble Tea root model and workflow routing for this
// sync-and-storage slice.
// Authored by: OpenCode
package flow

import "github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"

// menuActionID identifies one selectable workflow action independently from its
// current rendered index.
// Authored by: OpenCode
type menuActionID string

const (
	mainMenuActionSyncReports menuActionID = "main.sync_reports"

	syncMenuActionUnlock    menuActionID = "sync.unlock"
	syncMenuActionStartSync menuActionID = "sync.start"
	syncMenuActionBack      menuActionID = "sync.back"

	resultMenuActionGenerateDiagnostic menuActionID = "result.generate_diagnostic"
	resultMenuActionSyncAgain          menuActionID = "result.sync_again"
	resultMenuActionBackToMainMenu     menuActionID = "result.back_to_main_menu"

	syncReportsMenuActionSyncData           menuActionID = "sync_reports.sync_data"
	syncReportsMenuActionGenerateReport     menuActionID = "sync_reports.generate_report"
	syncReportsMenuActionGenerateDiagnostic menuActionID = "sync_reports.generate_diagnostic"
	syncReportsMenuActionBackToMainMenu     menuActionID = "sync_reports.back_to_main_menu"

	reportSelectionActionGenerateReport menuActionID = "report_selection.generate_report"
	reportSelectionActionBack           menuActionID = "report_selection.back"

	reportResultActionGenerateDiagnostic menuActionID = "report_result.generate_diagnostic"
	reportResultActionBackToSyncReports  menuActionID = "report_result.back_to_sync_reports"
	reportResultActionGenerateAnother    menuActionID = "report_result.generate_another"

	serverReplacementActionContinue menuActionID = "server_replacement.continue"
	serverReplacementActionCancel   menuActionID = "server_replacement.cancel"
)

// menuActionAt returns the action identifier for one rendered menu index.
// Authored by: OpenCode
func menuActionAt(actions []menuActionID, index int) menuActionID {
	if index < 0 || index >= len(actions) {
		return ""
	}

	return actions[index]
}

// menuIndexForAction returns the rendered index for one action identifier.
// Authored by: OpenCode
func menuIndexForAction(actions []menuActionID, target menuActionID) int {
	for index, action := range actions {
		if action == target {
			return index
		}
	}

	return 0
}

// mainMenuActions returns the stable main-menu action ordering.
// Authored by: OpenCode
func (m *Model) mainMenuActions() []menuActionID {
	return []menuActionID{mainMenuActionSyncReports}
}

// syncMenuActions returns the stable action ordering for standalone sync or
// Sync and Reports unlock entry.
// Authored by: OpenCode
func (m *Model) syncMenuActions() []menuActionID {
	if m.active == syncReportsUnlockScreenKey {
		return []menuActionID{syncMenuActionUnlock, syncMenuActionBack}
	}

	return []menuActionID{syncMenuActionStartSync, syncMenuActionBack}
}

// resultMenuActions returns the stable sync-result action ordering.
// Authored by: OpenCode
func (m *Model) resultMenuActions() []menuActionID {
	if m.result.Busy {
		return []menuActionID{
			resultMenuActionGenerateDiagnostic,
			resultMenuActionSyncAgain,
			resultMenuActionBackToMainMenu,
		}
	}
	if m.result.Outcome.Diagnostic.Eligible && m.result.Outcome.Diagnostic.Path == "" {
		return []menuActionID{
			resultMenuActionGenerateDiagnostic,
			resultMenuActionSyncAgain,
			resultMenuActionBackToMainMenu,
		}
	}

	return []menuActionID{resultMenuActionSyncAgain, resultMenuActionBackToMainMenu}
}

// syncReportsMenuActions returns the stable unlocked-context action ordering.
// Authored by: OpenCode
func (m *Model) syncReportsMenuActions() []menuActionID {
	var actions = []menuActionID{
		syncReportsMenuActionSyncData,
		syncReportsMenuActionGenerateReport,
	}
	if m.syncReportsHasPendingDiagnostic() {
		actions = append(actions, syncReportsMenuActionGenerateDiagnostic)
	}
	actions = append(actions, syncReportsMenuActionBackToMainMenu)
	return actions
}

// reportSelectionActions returns the stable report-selection action ordering.
// Authored by: OpenCode
func (m *Model) reportSelectionActions() []menuActionID {
	return []menuActionID{reportSelectionActionGenerateReport, reportSelectionActionBack}
}

// reportResultActions returns the stable report-result action ordering.
// Authored by: OpenCode
func (m *Model) reportResultActions() []menuActionID {
	var actions []menuActionID
	if m.report.Busy {
		actions = append(actions, reportResultActionGenerateDiagnostic)
	}
	if m.reportOutcomeHasPendingDiagnostic() {
		actions = append(actions, reportResultActionGenerateDiagnostic)
	}
	actions = append(actions, reportResultActionBackToSyncReports)
	if m.syncReports.ProtectedData.HasReadableSnapshot && len(m.syncReports.ProtectedData.AvailableReportYears) > 0 {
		actions = append(actions, reportResultActionGenerateAnother)
	}
	return actions
}

// serverReplacementActions returns the stable server-replacement action ordering.
// Authored by: OpenCode
func (m *Model) serverReplacementActions() []menuActionID {
	return []menuActionID{serverReplacementActionContinue, serverReplacementActionCancel}
}

// selectedMainMenuAction returns the selected main-menu action.
// Authored by: OpenCode
func (m *Model) selectedMainMenuAction() menuActionID {
	return menuActionAt(m.mainMenuActions(), 0)
}

// selectedSyncMenuAction returns the selected sync or unlock action.
// Authored by: OpenCode
func (m *Model) selectedSyncMenuAction() menuActionID {
	return menuActionAt(m.syncMenuActions(), m.sync.MenuIndex)
}

// selectedResultMenuAction returns the selected sync-result action.
// Authored by: OpenCode
func (m *Model) selectedResultMenuAction() menuActionID {
	return menuActionAt(m.resultMenuActions(), m.result.MenuIndex)
}

// selectedSyncReportsMenuAction returns the selected unlocked-context action.
// Authored by: OpenCode
func (m *Model) selectedSyncReportsMenuAction() menuActionID {
	return menuActionAt(m.syncReportsMenuActions(), m.sync.MenuIndex)
}

// selectedReportSelectionAction returns the selected report-selection action.
// Authored by: OpenCode
func (m *Model) selectedReportSelectionAction() menuActionID {
	return menuActionAt(m.reportSelectionActions(), m.report.ActionIndex)
}

// selectedReportResultAction returns the selected report-result action.
// Authored by: OpenCode
func (m *Model) selectedReportResultAction() menuActionID {
	return menuActionAt(m.reportResultActions(), m.report.ActionIndex)
}

// selectedServerReplacementAction returns the selected replacement action.
// Authored by: OpenCode
func (m *Model) selectedServerReplacementAction() menuActionID {
	return menuActionAt(m.serverReplacementActions(), m.replacement.MenuIndex)
}

// syncReportsReportActionIndex returns the unlocked-context index for the report action.
// Authored by: OpenCode
func (m *Model) syncReportsReportActionIndex() int {
	return menuIndexForAction(m.syncReportsMenuActions(), syncReportsMenuActionGenerateReport)
}

// syncReportsDefaultMenuAction returns the preferred unlocked-context action
// after one sync attempt or diagnostic generation completes.
// Authored by: OpenCode
func (m *Model) syncReportsDefaultMenuAction() menuActionID {
	if m.syncReportsHasPendingDiagnostic() {
		return syncReportsMenuActionGenerateDiagnostic
	}

	return syncReportsMenuActionSyncData
}

// reportUnavailable returns whether report generation is currently available in
// the unlocked context.
// Authored by: OpenCode
func (m *Model) reportUnavailable() bool {
	return !m.syncReports.ProtectedData.HasReadableSnapshot || len(m.syncReports.ProtectedData.AvailableReportYears) == 0
}

// syncReportsReportUnavailableReason returns the current user-visible report
// unavailable reason.
// Authored by: OpenCode
func (m *Model) syncReportsReportUnavailableReason() runtime.ReportFailureReason {
	if m.reportUnavailable() {
		return m.syncReports.ReportUnavailable
	}

	return runtime.ReportFailureNone
}

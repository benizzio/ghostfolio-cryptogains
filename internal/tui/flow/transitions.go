// Package flow owns the Bubble Tea root model and workflow routing for this
// sync-and-storage slice.
// Authored by: OpenCode
package flow

import (
	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
)

// clearSyncReportsRuntimeState scrubs token and transient report state for the
// active `Sync and Reports` context.
// Authored by: OpenCode
func (m *Model) clearSyncReportsRuntimeState() {
	m.syncReports = newSyncReportsContextState(m.currentServerOrigin(), m.deps.SyncService.ProtectedDataState())
	m.report = newReportState(m.syncReports.ProtectedData.AvailableReportYears)
}

// enterSetup routes the application to the setup workflow without a startup requirement.
// Authored by: OpenCode
func (m *Model) enterSetup(message string) {
	m.active = setupScreenKey
	m.setup = newSetupState(m.currentConfig, bootstrap.SetupRequirementNone)
	m.setup.ValidationMessage = message
}

// enterMainMenu routes the application back to the main menu.
// Authored by: OpenCode
func (m *Model) enterMainMenu() {
	m.active = mainMenuScreenKey
	m.result = resultState{}
	m.sync = newSyncState()
	m.clearSyncReportsRuntimeState()
	m.sync.InputFocused = false
	m.setup.ValidationMessage = ""
	m.setup.StartupReason = bootstrap.SetupRequirementNone
}

// enterSyncReportsUnlock routes the application to the token-unlock entry
// screen for the active `Sync and Reports` context.
// Authored by: OpenCode
func (m *Model) enterSyncReportsUnlock() tea.Cmd {
	m.active = syncReportsUnlockScreenKey
	m.sync = newSyncState()
	m.clearSyncReportsRuntimeState()
	return m.sync.TokenInput.Focus()
}

// enterSyncReportsMenu routes the application to the unlocked Sync and Reports context menu.
// Authored by: OpenCode
func (m *Model) enterSyncReportsMenu(unlocked runtime.SyncReportsContextResult, token string) {
	m.active = syncReportsMenuScreenKey
	m.sync = newSyncState()
	m.sync.InputFocused = false
	m.sync.TokenInput.Blur()
	m.sync.MenuIndex = menuIndexForAction(m.syncReportsMenuActions(), syncReportsMenuActionSyncData)
	m.syncReports.Active = true
	m.syncReports.RuntimeToken = token
	m.syncReports.SelectedServerOrigin = m.currentServerOrigin()
	m.syncReports.ProtectedData = unlocked.ProtectedData
	m.syncReports.ReportUnavailable = unlocked.ReportUnavailableReason
	m.syncReports.UnlockFailure = runtime.SyncFailureNone
	m.report = newReportState(unlocked.ProtectedData.AvailableReportYears)
}

// enterReportSelection routes the application to the report-selection screen.
// Authored by: OpenCode
func (m *Model) enterReportSelection() {
	m.active = reportSelectionScreenKey
	m.report = newReportState(m.syncReports.ProtectedData.AvailableReportYears)
	m.report.ActionIndex = menuIndexForAction(m.reportSelectionActions(), reportSelectionActionGenerateReport)
}

// enterReportResult routes the application to the report result screen.
// Authored by: OpenCode
func (m *Model) enterReportResult(outcome runtime.ReportOutcome) {
	m.active = reportResultScreenKey
	m.report.Busy = false
	m.report.BusyText = ""
	m.report.AttemptID = ""
	m.report.ActionIndex = 0
	m.syncReports.ReportResult = outcome
	m.refreshReportResultViewport(true)
}

// enterSync routes the application to the sync entry screen.
// Authored by: OpenCode
func (m *Model) enterSync() tea.Cmd {
	m.active = syncScreenKey
	m.sync = newSyncState()
	return m.sync.TokenInput.Focus()
}

// enterSyncWithContextToken routes the application to the sync entry screen in
// token-free context mode, reusing the active `Sync and Reports` token without
// exposing it in the renderer or input state.
// Authored by: OpenCode
func (m *Model) enterSyncWithContextToken() {
	m.active = syncScreenKey
	m.sync = newSyncState()
	m.sync.UseContextToken = true
	m.sync.InputFocused = false
	m.sync.TokenInput.Blur()
	m.sync.MenuIndex = menuIndexForAction(m.syncMenuActions(), syncMenuActionStartSync)
}

// enterServerReplacement routes the application to the server-mismatch confirmation screen.
// Authored by: OpenCode
func (m *Model) enterServerReplacement(check runtime.ServerReplacementCheck, pendingToken string) {
	m.active = serverReplacementScreenKey
	m.replacement = serverReplacementState{
		PendingToken:  pendingToken,
		CurrentServer: check.ActiveServerOrigin,
		NewServer:     check.SelectedServerOrigin,
	}
}

// enterSyncResult routes the application to the sync result screen.
// Authored by: OpenCode
func (m *Model) enterSyncResult(outcome runtime.SyncOutcome) {
	m.active = syncResultScreenKey
	m.result = resultState{Outcome: outcome}
	if outcome.Success {
		m.result.MenuIndex = menuIndexForAction(m.resultMenuActions(), resultMenuActionBackToMainMenu)
	}
}

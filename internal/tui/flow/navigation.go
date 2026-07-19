// Package flow owns the Bubble Tea root model and workflow routing for this
// sync-and-storage slice.
// Authored by: OpenCode
package flow

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

var (
	cachedUpBinding        = key.NewBinding(key.WithKeys("up"), key.WithHelp("up", "move up"))
	cachedDownBinding      = key.NewBinding(key.WithKeys("down"), key.WithHelp("down", "move down"))
	cachedEnterBinding     = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select"))
	cachedFocusBinding     = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "toggle focus"))
	cachedCancelBinding    = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))
	cachedEditSetupBinding = key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "edit setup"))
	cachedQuitBinding      = key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit"))
	cachedPageUpBinding    = key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "scroll up"))
	cachedPageDownBinding  = key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "scroll down"))
	cachedReportActionHelp = key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("up/down", "choose action"))
	cachedReportScrollHelp = key.NewBinding(key.WithKeys("pgup", "pgdown"), key.WithHelp("pgup/pgdn", "scroll"))
)

// upBinding returns the shared upward menu navigation binding.
// Authored by: OpenCode
func upBinding() key.Binding {
	return cachedUpBinding
}

// downBinding returns the shared downward menu navigation binding.
// Authored by: OpenCode
func downBinding() key.Binding {
	return cachedDownBinding
}

// enterBinding returns the shared primary-action binding.
// Authored by: OpenCode
func enterBinding() key.Binding {
	return cachedEnterBinding
}

// focusBinding returns the shared input-focus toggle binding.
// Authored by: OpenCode
func focusBinding() key.Binding {
	return cachedFocusBinding
}

// cancelBinding returns the setup cancel binding for remembered setup edits.
// Authored by: OpenCode
func cancelBinding() key.Binding {
	return cachedCancelBinding
}

// editSetupBinding returns the main-menu edit-setup binding.
// Authored by: OpenCode
func editSetupBinding() key.Binding {
	return cachedEditSetupBinding
}

// quitBinding returns the shared quit binding.
// Authored by: OpenCode
func quitBinding() key.Binding {
	return cachedQuitBinding
}

// pageUpBinding returns the report-result upward paging binding.
// Authored by: OpenCode
func pageUpBinding() key.Binding {
	return cachedPageUpBinding
}

// pageDownBinding returns the report-result downward paging binding.
// Authored by: OpenCode
func pageDownBinding() key.Binding {
	return cachedPageDownBinding
}

// reportActionHelpBinding returns compact report-result action-navigation help.
// Authored by: OpenCode
func reportActionHelpBinding() key.Binding {
	return cachedReportActionHelp
}

// reportScrollHelpBinding returns compact report-result paging help.
// Authored by: OpenCode
func reportScrollHelpBinding() key.Binding {
	return cachedReportScrollHelp
}

// updateMainMenu handles main-menu navigation.
// Authored by: OpenCode
func (m *Model) updateMainMenu(message tea.Msg) (tea.Model, tea.Cmd) {
	var keyMessage, ok = message.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch {
	case key.Matches(keyMessage, editSetupBinding()):
		return m, m.enterSetup("", bootstrap.SetupRequirementNone)
	case key.Matches(keyMessage, enterBinding()):
		return m, m.enterSyncReportsUnlock()
	default:
		return m, nil
	}
}

// updateSyncResult handles sync-result navigation.
// Authored by: OpenCode
func (m *Model) updateSyncResult(message tea.Msg) (tea.Model, tea.Cmd) {
	if diagnosticMessage, ok := message.(diagnosticReportFinishedMsg); ok {
		return m.handleDiagnosticReportFinished(diagnosticMessage)
	}

	var keyMessage, ok = message.(tea.KeyPressMsg)
	if !ok || m.result.Busy {
		return m, nil
	}

	switch {
	case key.Matches(keyMessage, upBinding()):
		m.moveSyncResultSelection(-1)
	case key.Matches(keyMessage, downBinding()):
		m.moveSyncResultSelection(1)
	case key.Matches(keyMessage, enterBinding()):
		return m.activateSyncResultSelection()
	}

	return m, nil
}

// moveSyncResultSelection advances the sync-result selection by one item.
// Authored by: OpenCode
func (m *Model) moveSyncResultSelection(step int) {
	if step == 0 {
		return
	}

	var nextIndex = m.result.MenuIndex + step
	if nextIndex < 0 || nextIndex >= len(m.resultMenuItems()) {
		return
	}

	m.result.MenuIndex = nextIndex
}

// activateSyncResultSelection runs the selected sync-result action.
// Authored by: OpenCode
func (m *Model) activateSyncResultSelection() (tea.Model, tea.Cmd) {
	switch m.selectedResultMenuAction() {
	case resultMenuActionGenerateDiagnostic:
		return m.generateDiagnosticReport()
	case resultMenuActionSyncAgain:
		if m.syncReports.Active {
			return m, m.enterSyncWithContextToken()
		}
		return m, m.enterSync()
	case resultMenuActionBackToMainMenu:
		m.enterMainMenu()
		return m, nil
	default:
		return m, nil
	}
}

// updateSyncReportsMenu handles unlocked Sync and Reports context navigation.
// Authored by: OpenCode
func (m *Model) updateSyncReportsMenu(message tea.Msg) (tea.Model, tea.Cmd) {
	var handledModel, handledCmd, handled = m.handleSyncReportsMenuMessage(message)
	if handled {
		return handledModel, handledCmd
	}

	var keyMessage, ok = message.(tea.KeyPressMsg)
	if !ok || m.syncReports.SyncResult.Busy {
		return m, nil
	}

	var menuItems = m.syncReportsMenuItems()

	switch {
	case key.Matches(keyMessage, upBinding()):
		m.moveSyncReportsMenuSelection(-1, menuItems)
	case key.Matches(keyMessage, downBinding()):
		m.moveSyncReportsMenuSelection(1, menuItems)
	case key.Matches(keyMessage, enterBinding()):
		return m.activateSyncReportsMenuSelection()
	}

	return m, nil
}

// handleSyncReportsMenuMessage handles non-key messages routed to the unlocked Sync and Reports menu.
// Authored by: OpenCode
func (m *Model) handleSyncReportsMenuMessage(message tea.Msg) (tea.Model, tea.Cmd, bool) {
	if typedMessage, ok := message.(diagnosticReportFinishedMsg); ok {
		updatedModel, cmd := m.handleDiagnosticReportFinished(typedMessage)
		return updatedModel, cmd, true
	}

	return m, nil, false
}

// activateSyncReportsMenuSelection runs the currently selected unlocked-context action.
// Authored by: OpenCode
func (m *Model) activateSyncReportsMenuSelection() (tea.Model, tea.Cmd) {
	var action = m.selectedSyncReportsMenuAction()
	if m.shouldFallbackToSyncReportsBackAction(action) {
		m.enterMainMenu()
		return m, nil
	}

	switch action {
	case syncReportsMenuActionSyncData:
		return m, m.enterSyncWithContextToken()
	case syncReportsMenuActionGenerateReport:
		return m.activateSyncReportsGenerateReport()
	case syncReportsMenuActionGenerateDiagnostic:
		return m.activateSyncReportsGenerateDiagnostic()
	case syncReportsMenuActionBackToMainMenu:
		m.enterMainMenu()
		return m, nil
	default:
		return m, nil
	}
}

// activateSyncReportsGenerateReport routes into report selection when report generation is available.
// Authored by: OpenCode
func (m *Model) activateSyncReportsGenerateReport() (tea.Model, tea.Cmd) {
	if m.reportUnavailable() {
		return m, nil
	}

	m.enterReportSelection()
	return m, nil
}

// activateSyncReportsGenerateDiagnostic runs diagnostic generation or falls back to the main menu.
// Authored by: OpenCode
func (m *Model) activateSyncReportsGenerateDiagnostic() (tea.Model, tea.Cmd) {
	if m.syncReportsHasPendingDiagnostic() {
		return m.generateDiagnosticReport()
	}

	m.enterMainMenu()
	return m, nil
}

// shouldFallbackToSyncReportsBackAction preserves the legacy raw-index fallback for the final Back action row.
// Authored by: OpenCode
func (m *Model) shouldFallbackToSyncReportsBackAction(action menuActionID) bool {
	return action == "" && m.sync.MenuIndex == 3
}

// moveSyncReportsMenuSelection advances the unlocked-context menu selection to
// the next enabled item in one direction, skipping disabled rows.
// Authored by: OpenCode
func (m *Model) moveSyncReportsMenuSelection(step int, items []component.MenuItem) {
	if step == 0 || len(items) == 0 {
		return
	}

	var index = m.sync.MenuIndex
	for {
		index += step
		if index < 0 || index >= len(items) {
			return
		}
		if items[index].Enabled {
			m.sync.MenuIndex = index
			return
		}
	}
}

// generateDiagnosticReport writes one local synced-data or report-failure
// diagnostic report from the current result screen.
// Authored by: OpenCode
func (m *Model) generateDiagnosticReport() (tea.Model, tea.Cmd) {
	var request = m.result.Outcome.Diagnostic.Request
	var useSyncReportsContext = m.active == syncReportsMenuScreenKey
	if useSyncReportsContext {
		request = m.syncReports.SyncResult.Outcome.Diagnostic.Request
	} else if m.active == reportResultScreenKey {
		request = m.syncReports.ReportResult.Diagnostic.Request
	}
	if request.ServerOrigin == "" && m.currentConfig != nil {
		request.ServerOrigin = m.currentConfig.ServerOrigin
	}
	if request.Attempt.AttemptID == "" {
		if useSyncReportsContext {
			request.Attempt = m.syncReports.SyncResult.Outcome.Attempt
		} else if m.active == reportResultScreenKey {
			request.Attempt = m.syncReports.ReportResult.Attempt
		} else {
			request.Attempt = m.result.Outcome.Attempt
		}
	}
	if useSyncReportsContext {
		m.syncReports.SyncResult.Outcome.Diagnostic.Request = request
		m.syncReports.SyncResult.Busy = true
		m.syncReports.SyncResult.StatusMessage = "Generating diagnostic report..."
		m.sync.MenuIndex = m.syncReportsDefaultMenuIndex()
	} else {
		if m.active == reportResultScreenKey {
			m.syncReports.ReportResult.Diagnostic.Request = request
			m.report.Busy = true
			m.syncReports.ReportResult.Diagnostic.GenerationMessage = "Generating diagnostic report..."
			m.refreshReportResultViewport(false)
		} else {
			m.result.Outcome.Diagnostic.Request = request
			m.result.Busy = true
			m.result.StatusMessage = "Generating diagnostic report..."
		}
	}
	return m, m.generateDiagnosticReportCmd(request)
}

// handleDiagnosticReportFinished applies the result of one async
// diagnostic-report write request.
// Authored by: OpenCode
func (m *Model) handleDiagnosticReportFinished(message diagnosticReportFinishedMsg) (tea.Model, tea.Cmd) {
	if m.active == syncReportsMenuScreenKey {
		m.syncReports.SyncResult.Busy = false
		if message.Err != nil {
			m.syncReports.SyncResult.StatusMessage = "Diagnostic report generation failed. Try again."
			m.syncReports.SyncResult.Outcome.Diagnostic.GenerationMessage = m.syncReports.SyncResult.StatusMessage
			m.sync.MenuIndex = m.syncReportsDefaultMenuIndex()
			return m, nil
		}

		m.syncReports.SyncResult.Outcome.Diagnostic.Path = message.Path
		m.syncReports.SyncResult.StatusMessage = "Diagnostic report generated successfully."
		m.syncReports.SyncResult.Outcome.Diagnostic.GenerationMessage = m.syncReports.SyncResult.StatusMessage
		m.sync.MenuIndex = m.syncReportsDefaultMenuIndex()
		return m, nil
	}
	if m.active == reportResultScreenKey {
		m.report.Busy = false
		if message.Err != nil {
			m.syncReports.ReportResult.Diagnostic.GenerationMessage = "Diagnostic report generation failed. Try again."
			m.refreshReportResultViewport(false)
			return m, nil
		}

		m.syncReports.ReportResult.Diagnostic.Path = message.Path
		m.syncReports.ReportResult.Diagnostic.GenerationMessage = "Diagnostic report generated successfully."
		m.refreshReportResultViewport(false)
		return m, nil
	}

	m.result.Busy = false
	if message.Err != nil {
		m.result.StatusMessage = "Diagnostic report generation failed. Try again."
		m.result.Outcome.Diagnostic.GenerationMessage = m.result.StatusMessage
		return m, nil
	}

	m.result.Outcome.Diagnostic.Path = message.Path
	m.result.StatusMessage = "Diagnostic report generated successfully."
	m.result.Outcome.Diagnostic.GenerationMessage = m.result.StatusMessage
	return m, nil
}

// updateServerReplacement handles server-mismatch confirmation navigation.
// Authored by: OpenCode
func (m *Model) updateServerReplacement(message tea.Msg) (tea.Model, tea.Cmd) {
	var keyMessage, ok = message.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch {
	case key.Matches(keyMessage, upBinding()):
		if m.replacement.MenuIndex > 0 {
			m.replacement.MenuIndex--
		}
	case key.Matches(keyMessage, downBinding()):
		if m.replacement.MenuIndex < len(m.serverReplacementMenuItems())-1 {
			m.replacement.MenuIndex++
		}
	case key.Matches(keyMessage, enterBinding()):
		if m.selectedServerReplacementAction() == serverReplacementActionContinue {
			return m.startConfirmedServerReplacement()
		}
		m.replacement.PendingToken = ""
		m.sync.TokenInput.Reset()
		if m.syncReports.Active {
			m.active = syncReportsMenuScreenKey
			m.sync.MenuIndex = menuIndexForAction(m.syncReportsMenuActions(), syncReportsMenuActionSyncData)
			return m, nil
		}
		m.enterSyncResult(runtime.SyncOutcome{Success: false, FailureReason: runtime.SyncFailureServerReplacementCancelled, DetailReason: string(runtime.SyncFailureServerReplacementCancelled)})
	}

	return m, nil
}

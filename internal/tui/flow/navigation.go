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
	switch typedMessage := message.(type) {
	case diagnosticReportFinishedMsg:
		return m.handleDiagnosticReportFinished(typedMessage)
	}

	var keyMessage, ok = message.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	if m.result.Busy {
		return m, nil
	}

	var menuItems = m.resultMenuItems()

	switch {
	case key.Matches(keyMessage, upBinding()):
		if m.result.MenuIndex > 0 {
			m.result.MenuIndex--
		}
	case key.Matches(keyMessage, downBinding()):
		if m.result.MenuIndex < len(menuItems)-1 {
			m.result.MenuIndex++
		}
	case key.Matches(keyMessage, enterBinding()):
		if m.result.Outcome.Diagnostic.Eligible && m.result.Outcome.Diagnostic.Path == "" {
			switch m.result.MenuIndex {
			case 0:
				return m.generateDiagnosticReport()
			case 1:
				if m.syncReports.Active {
					return m, m.enterSyncWithContextToken()
				}
				return m, m.enterSync()
			default:
				m.enterMainMenu()
				return m, nil
			}
		}
		if m.result.MenuIndex == 0 {
			if m.syncReports.Active {
				return m, m.enterSyncWithContextToken()
			}
			return m, m.enterSync()
		}
		m.enterMainMenu()
	}

	return m, nil
}

// updateSyncReportsMenu handles unlocked Sync and Reports context navigation.
// Authored by: OpenCode
func (m *Model) updateSyncReportsMenu(message tea.Msg) (tea.Model, tea.Cmd) {
	switch typedMessage := message.(type) {
	case diagnosticReportFinishedMsg:
		return m.handleDiagnosticReportFinished(typedMessage)
	}

	var keyMessage, ok = message.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	if m.syncReports.SyncResult.Busy {
		return m, nil
	}

	var menuItems = m.syncReportsMenuItems()

	switch {
	case key.Matches(keyMessage, upBinding()):
		m.moveSyncReportsMenuSelection(-1, menuItems)
	case key.Matches(keyMessage, downBinding()):
		m.moveSyncReportsMenuSelection(1, menuItems)
	case key.Matches(keyMessage, enterBinding()):
		switch m.sync.MenuIndex {
		case 0:
			return m, m.enterSyncWithContextToken()
		case 1:
			if m.syncReports.ProtectedData.HasReadableSnapshot && len(m.syncReports.ProtectedData.AvailableReportYears) > 0 {
				m.enterReportSelection()
				return m, nil
			}
			return m, nil
		case 2:
			if m.syncReportsHasPendingDiagnostic() {
				return m.generateDiagnosticReport()
			}
			m.enterMainMenu()
			return m, nil
		case 3:
			m.enterMainMenu()
			return m, nil
		default:
			return m, nil
		}
	}

	return m, nil
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
			return m, nil
		}

		m.syncReports.ReportResult.Diagnostic.Path = message.Path
		m.syncReports.ReportResult.Diagnostic.GenerationMessage = "Diagnostic report generated successfully."
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
		if m.replacement.MenuIndex == 0 {
			return m.startConfirmedServerReplacement()
		}
		m.replacement.PendingToken = ""
		m.sync.TokenInput.Reset()
		if m.syncReports.Active {
			m.active = syncReportsMenuScreenKey
			m.sync.MenuIndex = 0
			return m, nil
		}
		m.enterSyncResult(runtime.SyncOutcome{Success: false, FailureReason: runtime.SyncFailureServerReplacementCancelled, DetailReason: string(runtime.SyncFailureServerReplacementCancelled)})
	}

	return m, nil
}

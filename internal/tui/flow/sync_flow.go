// Package flow owns the Bubble Tea root model and workflow routing for this
// sync-and-storage slice.
// Authored by: OpenCode
package flow

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

const busyStatusText = "Syncing and storing activity history..."

// updateSync handles sync-entry navigation, token input, busy-state spinner
// updates, and sync completion routing.
// Authored by: OpenCode
func (m *Model) updateSync(message tea.Msg) (tea.Model, tea.Cmd) {
	switch typedMessage := message.(type) {
	case syncFinishedMsg:
		return m.handleSyncFinished(typedMessage)
	case tea.PasteMsg, tea.PasteStartMsg, tea.PasteEndMsg:
		return m.handleSyncPaste(message)
	case spinner.TickMsg:
		return m.handleSyncSpinnerTick(typedMessage)
	case tea.KeyPressMsg:
		return m.handleSyncKeyPress(typedMessage)
	default:
		return m, nil
	}
}

// handleSyncFinished applies a completed sync attempt.
// Authored by: OpenCode
func (m *Model) handleSyncFinished(message syncFinishedMsg) (tea.Model, tea.Cmd) {
	if message.Attempt != m.sync.AttemptID {
		return m, nil
	}

	var returnToSyncReports = m.syncReports.Active && m.syncReports.RuntimeToken != ""
	m.sync.Busy = false
	m.sync.BusyText = ""
	m.sync.AttemptID = ""
	m.sync.Cancel = nil
	if returnToSyncReports {
		m.sync.TokenInput.Reset()
		m.syncReports.SyncResult = syncContextResultState{Outcome: message.Outcome}
		m.syncReports.ProtectedData = m.deps.SyncService.ProtectedDataState()
		if m.syncReports.ProtectedData.HasReadableSnapshot && len(m.syncReports.ProtectedData.AvailableReportYears) > 0 {
			m.syncReports.ReportUnavailable = runtime.ReportFailureNone
		} else if len(m.syncReports.ProtectedData.AvailableReportYears) == 0 {
			m.syncReports.ReportUnavailable = runtime.ReportFailureNoReportableYearsAvailable
			if !m.syncReports.ProtectedData.HasReadableSnapshot {
				m.syncReports.ReportUnavailable = runtime.ReportFailureNoSyncedDataAvailable
			}
		}
		m.active = syncReportsMenuScreenKey
		m.sync.MenuIndex = m.syncReportsDefaultMenuIndex()
		return m, nil
	}

	m.sync.TokenInput.Reset()
	m.enterSyncResult(message.Outcome)
	return m, nil
}

// handleSyncPaste routes paste events to the focused token input when the sync
// workflow is idle.
// Authored by: OpenCode
func (m *Model) handleSyncPaste(message tea.Msg) (tea.Model, tea.Cmd) {
	if m.active == syncReportsUnlockScreenKey && m.syncReports.UnlockFailure == runtime.SyncFailureRejectedToken {
		return m, nil
	}
	if m.sync.Busy || m.sync.UseContextToken || !m.sync.InputFocused {
		return m, nil
	}

	return m.updateSyncTokenInput(message)
}

// handleSyncSpinnerTick updates the busy-state spinner while sync work is in
// flight.
//
// Authored by: OpenCode
func (m *Model) handleSyncSpinnerTick(message spinner.TickMsg) (tea.Model, tea.Cmd) {
	if !m.sync.Busy {
		return m, nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(message)
	return m, cmd
}

// handleSyncKeyPress routes sync-entry key presses to the focused token input
// or the primary sync menu.
// Authored by: OpenCode
func (m *Model) handleSyncKeyPress(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.sync.Busy {
		return m, nil
	}

	if model, cmd, handled := m.handleFocusedSyncKey(message); handled {
		return model, cmd
	}

	return m.handleSyncMenuKey(message)
}

// handleFocusedSyncKey handles key presses while the token input owns focus.
// Authored by: OpenCode
func (m *Model) handleFocusedSyncKey(message tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if m.active == syncReportsUnlockScreenKey && m.syncReports.UnlockFailure == runtime.SyncFailureRejectedToken {
		return m, nil, false
	}
	if !m.sync.InputFocused || m.sync.UseContextToken {
		return m, nil, false
	}

	switch {
	case key.Matches(message, enterBinding()):
		return m.releaseSyncInputToSyncMenu()
	case key.Matches(message, focusBinding()), key.Matches(message, cancelBinding()):
		return m.blurSyncInput()
	default:
		var model, cmd = m.updateSyncTokenInput(message)
		return model, cmd, true
	}
}

// handleSyncMenuKey handles key presses while the sync menu owns focus.
// Authored by: OpenCode
func (m *Model) handleSyncMenuKey(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(message, upBinding()):
		if m.active == syncReportsUnlockScreenKey {
			m.moveSyncMenuSelection(-1, m.syncMenuItems())
			return m, nil
		}
		if m.sync.MenuIndex > 0 {
			m.sync.MenuIndex--
		}
		return m, nil
	case key.Matches(message, downBinding()):
		if m.active == syncReportsUnlockScreenKey {
			m.moveSyncMenuSelection(1, m.syncMenuItems())
			return m, nil
		}
		if m.sync.MenuIndex < len(m.syncMenuItems())-1 {
			m.sync.MenuIndex++
		}
		return m, nil
	case key.Matches(message, focusBinding()):
		if m.active == syncReportsUnlockScreenKey && m.syncReports.UnlockFailure == runtime.SyncFailureRejectedToken {
			return m, nil
		}
		if m.sync.UseContextToken {
			return m, nil
		}
		return m.focusSyncTokenInput()
	case key.Matches(message, enterBinding()):
		return m.activateSyncSelection()
	default:
		return m, nil
	}
}

// updateSyncTokenInput updates the focused token input and clears stale sync-entry state.
// Authored by: OpenCode
func (m *Model) updateSyncTokenInput(message tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.sync.TokenInput, cmd = m.sync.TokenInput.Update(message)
	m.sync.ValidationMessage = ""
	return m, cmd
}

// releaseSyncInputToSyncMenu returns focus from the token input to the
// primary sync action.
// Authored by: OpenCode
func (m *Model) releaseSyncInputToSyncMenu() (tea.Model, tea.Cmd, bool) {
	m.blurSyncTokenInput()
	m.sync.MenuIndex = 0
	return m, nil, true
}

// blurSyncInput removes focus from the token input without changing menu
// selection.
// Authored by: OpenCode
func (m *Model) blurSyncInput() (tea.Model, tea.Cmd, bool) {
	m.blurSyncTokenInput()
	return m, nil, true
}

// blurSyncTokenInput clears token-input focus state.
// Authored by: OpenCode
func (m *Model) blurSyncTokenInput() {
	m.sync.InputFocused = false
	m.sync.TokenInput.Blur()
}

// focusSyncTokenInput focuses the Ghostfolio security-token input.
// Authored by: OpenCode
func (m *Model) focusSyncTokenInput() (tea.Model, tea.Cmd) {
	if m.sync.UseContextToken {
		return m, nil
	}

	m.sync.InputFocused = true
	return m, m.sync.TokenInput.Focus()
}

// activateSyncSelection runs the currently selected sync menu action.
// Authored by: OpenCode
func (m *Model) activateSyncSelection() (tea.Model, tea.Cmd) {
	if m.active == syncReportsUnlockScreenKey {
		return m.activateSyncReportsUnlockSelection()
	}

	switch m.sync.MenuIndex {
	case 0:
		return m.startSync()
	case 1:
		return m.leaveSync()
	default:
		return m, nil
	}
}

// activateSyncReportsUnlockSelection runs the currently selected unlock-menu
// action for the active `Sync and Reports` context shell.
// Authored by: OpenCode
func (m *Model) activateSyncReportsUnlockSelection() (tea.Model, tea.Cmd) {
	var menuItems = m.syncMenuItems()
	if m.sync.MenuIndex < 0 || m.sync.MenuIndex >= len(menuItems) || !menuItems[m.sync.MenuIndex].Enabled {
		return m, nil
	}
	if m.sync.MenuIndex == 0 {
		return m.unlockSyncReportsContext()
	}
	return m.leaveSyncReportsUnlock()
}

// startSync validates token input and starts one asynchronous sync run.
// Authored by: OpenCode
func (m *Model) startSync() (tea.Model, tea.Cmd) {
	if m.currentConfig == nil {
		return m, m.enterSetup("Complete setup before Sync Data can run.", bootstrap.SetupRequirementNone)
	}

	var token string
	if m.sync.UseContextToken {
		token = strings.TrimSpace(m.syncReports.RuntimeToken)
	} else {
		token = strings.TrimSpace(m.sync.TokenInput.Value())
	}
	if token == "" {
		if m.sync.UseContextToken {
			m.sync.ValidationMessage = "The active Sync and Reports token is unavailable. Return to the context menu and unlock again."
			return m, nil
		}
		m.sync.ValidationMessage = "Enter the Ghostfolio security token before starting sync."
		return m, nil
	}

	var config = *m.currentConfig
	var replacementCheck = m.deps.SyncService.CheckServerReplacement(config)
	if replacementCheck.Required {
		m.enterServerReplacement(replacementCheck, token)
		return m, nil
	}

	return m.startSyncAttempt(token, false)
}

// unlockSyncReportsContext validates token input and opens the initial active
// context shell by reusing the current sync workflow with the captured token.
// Authored by: OpenCode
func (m *Model) unlockSyncReportsContext() (tea.Model, tea.Cmd) {
	if m.currentConfig == nil {
		return m, m.enterSetup("Complete setup before Sync and Reports can run.", bootstrap.SetupRequirementNone)
	}

	var token = strings.TrimSpace(m.sync.TokenInput.Value())
	if token == "" {
		m.sync.ValidationMessage = "Enter the Ghostfolio security token before unlocking Sync and Reports."
		return m, nil
	}

	var unlocked = m.deps.SyncService.UnlockSelectedServerSnapshot(context.Background(), *m.currentConfig, token)
	if unlocked.ReportUnavailableReason == runtime.ReportFailureUnsupportedStoredDataVersion {
		m.sync.ValidationMessage = "unsupported stored-data version"
		m.syncReports.UnlockFailure = runtime.SyncFailureUnsupportedStoredDataVersion
		return m, nil
	}
	if unlocked.UnlockState == runtime.SyncReportsUnlockStateRejectedToken || unlocked.FailureReason == runtime.SyncFailureRejectedToken {
		m.sync.ValidationMessage = ""
		m.syncReports.UnlockFailure = runtime.SyncFailureRejectedToken
		m.sync.MenuIndex = 1
		return m, nil
	}
	return m, m.enterSyncReportsMenu(unlocked, token)
}

// startSyncAttempt starts one async sync request.
// Authored by: OpenCode
func (m *Model) startSyncAttempt(token string, confirmServerReplacement bool) (tea.Model, tea.Cmd) {
	var syncContext, cancel = context.WithCancel(context.Background())
	m.sync.Cancel = cancel
	m.sync.Busy = true
	m.sync.BusyText = busyStatusText
	m.sync.AttemptID = nextAttemptID()
	m.spinner = spinner.New(spinner.WithSpinner(spinner.Line))

	var config = *m.currentConfig
	if m.syncReports.Active {
		m.syncReports.RuntimeToken = token
		m.syncReports.SelectedServerOrigin = config.ServerOrigin
		m.syncReports.ProtectedData = m.deps.SyncService.ProtectedDataState()
	}
	m.active = syncScreenKey
	return m, tea.Batch(
		m.spinner.Tick,
		m.syncCmd(
			syncContext,
			m.sync.AttemptID,
			runtime.SyncRequest{
				Config:                   config,
				SecurityToken:            token,
				ConfirmServerReplacement: confirmServerReplacement,
			},
		),
	)
}

// leaveSyncReportsUnlock clears the initial unlock prompt and returns to the
// main menu before any context is activated.
// Authored by: OpenCode
func (m *Model) leaveSyncReportsUnlock() (tea.Model, tea.Cmd) {
	m.syncReports.UnlockFailure = runtime.SyncFailureNone
	m.sync.TokenInput.Reset()
	m.enterMainMenu()
	return m, nil
}

// moveSyncMenuSelection advances the sync or unlock menu selection to the next
// enabled item in one direction, skipping disabled rows.
// Authored by: OpenCode
func (m *Model) moveSyncMenuSelection(step int, items []component.MenuItem) {
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

// startConfirmedServerReplacement resumes sync after explicit server-replacement confirmation.
// Authored by: OpenCode
func (m *Model) startConfirmedServerReplacement() (tea.Model, tea.Cmd) {
	var token = m.replacement.PendingToken
	m.replacement.PendingToken = ""
	return m.startSyncAttempt(token, true)
}

// leaveSync clears transient token state and returns to the main
// menu.
// Authored by: OpenCode
func (m *Model) leaveSync() (tea.Model, tea.Cmd) {
	if m.syncReports.Active {
		m.sync.TokenInput.Reset()
		m.active = syncReportsMenuScreenKey
		m.sync.MenuIndex = 0
		return m, nil
	}

	m.sync.TokenInput.Reset()
	m.enterMainMenu()
	return m, nil
}

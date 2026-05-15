// Package flow owns the Bubble Tea root model and workflow routing for this
// validation-only slice.
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
)

const busyStatusText = "Syncing and storing activity history..."

// updateSyncValidation handles sync-entry navigation, token input, busy-state
// spinner updates, and validation completion routing.
// Authored by: OpenCode
func (m *Model) updateSyncValidation(message tea.Msg) (tea.Model, tea.Cmd) {
	switch typedMessage := message.(type) {
	case validationFinishedMsg:
		return m.handleValidationFinished(typedMessage)
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

// handleValidationFinished applies a completed sync-validation attempt.
// Authored by: OpenCode
func (m *Model) handleValidationFinished(message validationFinishedMsg) (tea.Model, tea.Cmd) {
	if message.Attempt != m.sync.AttemptID {
		return m, nil
	}

	m.sync.Busy = false
	m.sync.BusyText = ""
	m.sync.AttemptID = ""
	m.sync.Cancel = nil
	m.sync.TokenInput.Reset()
	m.enterValidationResult(message.Outcome)
	return m, nil
}

// handleSyncPaste routes paste events to the focused token input when the sync
// workflow is idle.
// Authored by: OpenCode
func (m *Model) handleSyncPaste(message tea.Msg) (tea.Model, tea.Cmd) {
	if m.sync.Busy || !m.sync.InputFocused {
		return m, nil
	}

	return m.updateSyncTokenInput(message)
}

// handleSyncSpinnerTick updates the busy-state spinner while validation is in
// flight.
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
	if !m.sync.InputFocused {
		return m, nil, false
	}

	switch {
	case key.Matches(message, enterBinding()):
		return m.releaseSyncInputToValidationMenu()
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
		if m.sync.MenuIndex > 0 {
			m.sync.MenuIndex--
		}
		return m, nil
	case key.Matches(message, downBinding()):
		if m.sync.MenuIndex < len(m.syncMenuItems())-1 {
			m.sync.MenuIndex++
		}
		return m, nil
	case key.Matches(message, focusBinding()):
		return m.focusSyncTokenInput()
	case key.Matches(message, enterBinding()):
		return m.activateSyncSelection()
	default:
		return m, nil
	}
}

// updateSyncTokenInput updates the focused token input and clears stale
// validation state.
// Authored by: OpenCode
func (m *Model) updateSyncTokenInput(message tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.sync.TokenInput, cmd = m.sync.TokenInput.Update(message)
	m.sync.ValidationMessage = ""
	return m, cmd
}

// releaseSyncInputToValidationMenu returns focus from the token input to the
// primary validation action.
// Authored by: OpenCode
func (m *Model) releaseSyncInputToValidationMenu() (tea.Model, tea.Cmd, bool) {
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
	m.sync.InputFocused = true
	return m, m.sync.TokenInput.Focus()
}

// activateSyncSelection runs the currently selected sync menu action.
// Authored by: OpenCode
func (m *Model) activateSyncSelection() (tea.Model, tea.Cmd) {
	switch m.sync.MenuIndex {
	case 0:
		return m.startSyncValidation()
	case 1:
		return m.leaveSyncValidation()
	default:
		return m, nil
	}
}

// startSyncValidation validates token input and starts the async communication
// probe.
// Authored by: OpenCode
func (m *Model) startSyncValidation() (tea.Model, tea.Cmd) {
	if m.currentConfig == nil {
		return m, m.enterSetup("Complete setup before Sync Data can run.", bootstrap.SetupRequirementNone)
	}

	var token = strings.TrimSpace(m.sync.TokenInput.Value())
	if token == "" {
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

// startSyncAttempt starts one async sync request.
// Authored by: OpenCode
func (m *Model) startSyncAttempt(token string, confirmServerReplacement bool) (tea.Model, tea.Cmd) {
	var validationContext, cancel = context.WithCancel(context.Background())
	m.sync.Cancel = cancel
	m.sync.Busy = true
	m.sync.BusyText = busyStatusText
	m.sync.AttemptID = nextAttemptID()
	m.spinner = spinner.New(spinner.WithSpinner(spinner.Line))

	var config = *m.currentConfig
	m.active = syncValidationScreenKey
	return m, tea.Batch(m.spinner.Tick, m.validationCmd(validationContext, m.sync.AttemptID, runtime.ValidateRequest{Config: config, SecurityToken: token, ConfirmServerReplacement: confirmServerReplacement}))
}

// startConfirmedServerReplacement resumes sync after explicit server-replacement confirmation.
// Authored by: OpenCode
func (m *Model) startConfirmedServerReplacement() (tea.Model, tea.Cmd) {
	return m.startSyncAttempt(m.replacement.PendingToken, true)
}

// leaveSyncValidation clears transient token state and returns to the main
// menu.
// Authored by: OpenCode
func (m *Model) leaveSyncValidation() (tea.Model, tea.Cmd) {
	m.sync.TokenInput.Reset()
	m.enterMainMenu()
	return m, nil
}

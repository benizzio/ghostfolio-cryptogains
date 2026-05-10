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
)

const busyStatusText = "Validating Ghostfolio communication..."

// updateSyncValidation handles sync-entry navigation, token input, busy-state
// spinner updates, and validation completion routing.
// Authored by: OpenCode
func (m *Model) updateSyncValidation(message tea.Msg) (tea.Model, tea.Cmd) {
	switch typedMessage := message.(type) {
	case validationFinishedMsg:
		if typedMessage.Attempt != m.sync.AttemptID {
			return m, nil
		}
		m.sync.Busy = false
		m.sync.BusyText = ""
		m.sync.AttemptID = ""
		m.sync.Cancel = nil
		m.sync.TokenInput.Reset()
		m.enterValidationResult(typedMessage.Outcome)
		return m, nil
	case spinner.TickMsg:
		if !m.sync.Busy {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(typedMessage)
		return m, cmd
	case tea.KeyPressMsg:
		if m.sync.Busy {
			return m, nil
		}

		if m.sync.InputFocused {
			switch {
			case key.Matches(typedMessage, focusBinding()), key.Matches(typedMessage, cancelBinding()):
				m.sync.InputFocused = false
				m.sync.TokenInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.sync.TokenInput, cmd = m.sync.TokenInput.Update(message)
				m.sync.ValidationMessage = ""
				return m, cmd
			}
		}

		switch {
		case key.Matches(typedMessage, upBinding()):
			if m.sync.MenuIndex > 0 {
				m.sync.MenuIndex--
			}
		case key.Matches(typedMessage, downBinding()):
			if m.sync.MenuIndex < len(m.syncMenuItems())-1 {
				m.sync.MenuIndex++
			}
		case key.Matches(typedMessage, focusBinding()):
			m.sync.InputFocused = true
			return m, m.sync.TokenInput.Focus()
		case key.Matches(typedMessage, enterBinding()):
			switch m.sync.MenuIndex {
			case 0:
				if m.currentConfig == nil {
					return m, m.enterSetup("Complete setup before Sync Data can run.")
				}

				var token = strings.TrimSpace(m.sync.TokenInput.Value())
				if token == "" {
					m.sync.ValidationMessage = "Enter the Ghostfolio security token before validating communication."
					return m, nil
				}

				var validationContext, cancel = context.WithCancel(context.Background())
				m.sync.Cancel = cancel
				m.sync.Busy = true
				m.sync.BusyText = busyStatusText
				m.sync.AttemptID = nextAttemptID()
				m.spinner = spinner.New(spinner.WithSpinner(spinner.Line))

				var config = *m.currentConfig
				return m, tea.Batch(m.spinner.Tick, m.validationCmd(validationContext, m.sync.AttemptID, config, token))
			case 1:
				m.sync.TokenInput.Reset()
				m.enterMainMenu()
			}
		}
	}

	return m, nil
}

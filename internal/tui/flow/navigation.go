// Package flow owns the Bubble Tea root model and workflow routing for this
// validation-only slice.
// Authored by: OpenCode
package flow

import (
	"context"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
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
		return m, m.enterSyncValidation()
	default:
		return m, nil
	}
}

// updateValidationResult handles validation-result navigation.
// Authored by: OpenCode
func (m *Model) updateValidationResult(message tea.Msg) (tea.Model, tea.Cmd) {
	var keyMessage, ok = message.(tea.KeyPressMsg)
	if !ok {
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
				return m, m.enterSyncValidation()
			default:
				m.enterMainMenu()
				return m, nil
			}
		}
		if m.result.MenuIndex == 0 {
			return m, m.enterSyncValidation()
		}
		m.enterMainMenu()
	}

	return m, nil
}

// generateDiagnosticReport writes one local synced-data diagnostic report from the current result screen.
// Authored by: OpenCode
func (m *Model) generateDiagnosticReport() (tea.Model, tea.Cmd) {
	var request = m.result.Outcome.Diagnostic.Request
	if request.ServerOrigin == "" && m.currentConfig != nil {
		request.ServerOrigin = m.currentConfig.ServerOrigin
	}
	if request.Attempt.AttemptID == "" {
		request.Attempt = m.result.Outcome.Attempt
	}
	path, err := m.deps.SyncService.GenerateDiagnosticReport(context.Background(), request)
	if err != nil {
		return m, nil
	}

	m.result.Outcome.Diagnostic.Path = path
	m.result.Outcome.Diagnostic.Request = request
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
		m.sync.TokenInput.Reset()
		m.enterValidationResult(runtime.ValidationOutcome{Success: false, FailureReason: runtime.SyncFailureServerReplacementCancelled, DetailReason: string(runtime.SyncFailureServerReplacementCancelled)})
	}

	return m, nil
}

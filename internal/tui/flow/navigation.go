// Package flow owns the Bubble Tea root model and workflow routing for this
// validation-only slice.
// Authored by: OpenCode
package flow

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
)

// upBinding returns the shared upward menu navigation binding.
// Authored by: OpenCode
func upBinding() key.Binding {
	return key.NewBinding(key.WithKeys("up"), key.WithHelp("up", "move up"))
}

// downBinding returns the shared downward menu navigation binding.
// Authored by: OpenCode
func downBinding() key.Binding {
	return key.NewBinding(key.WithKeys("down"), key.WithHelp("down", "move down"))
}

// enterBinding returns the shared primary-action binding.
// Authored by: OpenCode
func enterBinding() key.Binding {
	return key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select"))
}

// focusBinding returns the shared input-focus toggle binding.
// Authored by: OpenCode
func focusBinding() key.Binding {
	return key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "toggle focus"))
}

// cancelBinding returns the setup cancel binding for remembered setup edits.
// Authored by: OpenCode
func cancelBinding() key.Binding {
	return key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))
}

// editSetupBinding returns the main-menu edit-setup binding.
// Authored by: OpenCode
func editSetupBinding() key.Binding {
	return key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "edit setup"))
}

// quitBinding returns the shared quit binding.
// Authored by: OpenCode
func quitBinding() key.Binding {
	return key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit"))
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

	switch {
	case key.Matches(keyMessage, upBinding()):
		if m.result.MenuIndex > 0 {
			m.result.MenuIndex--
		}
	case key.Matches(keyMessage, downBinding()):
		if m.result.MenuIndex < len(m.resultMenuItems())-1 {
			m.result.MenuIndex++
		}
	case key.Matches(keyMessage, enterBinding()):
		if m.result.MenuIndex == 0 {
			return m, m.enterSyncValidation()
		}
		m.enterMainMenu()
	}

	return m, nil
}

// Package flow owns the Bubble Tea root model and workflow routing for this
// validation-only slice.
// Authored by: OpenCode
package flow

import (
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
)

// updateSetup handles setup workflow input, validation, and persistence.
// Authored by: OpenCode
func (m *Model) updateSetup(message tea.Msg) (tea.Model, tea.Cmd) {
	switch typedMessage := message.(type) {
	case setupSavedMsg:
		return m.handleSetupSaved(typedMessage)
	case tea.PasteMsg, tea.PasteStartMsg, tea.PasteEndMsg:
		return m.handleSetupPaste(message)
	case tea.KeyPressMsg:
		return m.handleSetupKeyPress(typedMessage)
	default:
		return m, nil
	}
}

// handleSetupSaved applies the result of a setup persistence attempt.
// Authored by: OpenCode
func (m *Model) handleSetupSaved(message setupSavedMsg) (tea.Model, tea.Cmd) {
	if message.Err != nil {
		m.setup.ValidationMessage = "Setup could not be saved. Try again."
		return m, nil
	}

	var config = message.Result.Config
	m.currentConfig = &config
	m.enterMainMenu()
	return m, nil
}

// handleSetupPaste routes paste events to the focused origin input.
// Authored by: OpenCode
func (m *Model) handleSetupPaste(message tea.Msg) (tea.Model, tea.Cmd) {
	if !m.setup.InputFocused {
		return m, nil
	}

	return m.updateSetupOriginInput(message)
}

// handleSetupKeyPress routes setup key presses to either the focused origin
// input or the primary setup menu.
// Authored by: OpenCode
func (m *Model) handleSetupKeyPress(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := m.handleFocusedSetupKey(message); handled {
		return model, cmd
	}

	return m.handleSetupMenuKey(message)
}

// handleFocusedSetupKey handles key presses while the origin input owns focus.
// Authored by: OpenCode
func (m *Model) handleFocusedSetupKey(message tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if !m.setup.InputFocused {
		return m, nil, false
	}

	switch {
	case key.Matches(message, enterBinding()):
		return m.releaseSetupInputToSavePath()
	case key.Matches(message, focusBinding()), key.Matches(message, cancelBinding()):
		return m.blurSetupInput()
	default:
		var model, cmd = m.updateSetupOriginInput(message)
		return model, cmd, true
	}
}

// handleSetupMenuKey handles key presses while the setup menu owns focus.
// Authored by: OpenCode
func (m *Model) handleSetupMenuKey(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(message, upBinding()):
		if m.setup.MenuIndex > 0 {
			m.setup.MenuIndex--
		}
		return m, nil
	case key.Matches(message, downBinding()):
		if m.setup.MenuIndex < len(m.setupMenuItems())-1 {
			m.setup.MenuIndex++
		}
		return m, nil
	case key.Matches(message, focusBinding()):
		return m.focusSetupOriginInputFromMenu()
	case key.Matches(message, cancelBinding()):
		if m.currentConfig != nil {
			m.enterMainMenu()
		}
		return m, nil
	case key.Matches(message, enterBinding()):
		return m.activateSetupSelection()
	default:
		return m, nil
	}
}

// updateSetupOriginInput updates the focused origin input and clears stale
// validation state.
// Authored by: OpenCode
func (m *Model) updateSetupOriginInput(message tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.setup.OriginInput, cmd = m.setup.OriginInput.Update(message)
	m.setup.ValidationMessage = ""
	return m, cmd
}

// releaseSetupInputToSavePath returns focus from the origin input to the save
// action path.
// Authored by: OpenCode
func (m *Model) releaseSetupInputToSavePath() (tea.Model, tea.Cmd, bool) {
	m.blurSetupOriginInput()
	m.setup.MenuIndex = 2
	return m, nil, true
}

// blurSetupInput removes focus from the origin input without changing menu
// selection.
// Authored by: OpenCode
func (m *Model) blurSetupInput() (tea.Model, tea.Cmd, bool) {
	m.blurSetupOriginInput()
	return m, nil, true
}

// blurSetupOriginInput clears origin-input focus state.
// Authored by: OpenCode
func (m *Model) blurSetupOriginInput() {
	m.setup.InputFocused = false
	m.setup.OriginInput.Blur()
}

// focusSetupOriginInputFromMenu focuses the custom-origin input when the setup
// menu allows it.
// Authored by: OpenCode
func (m *Model) focusSetupOriginInputFromMenu() (tea.Model, tea.Cmd) {
	if m.setup.SelectedMode != configmodel.ServerModeCustomOrigin && m.setup.MenuIndex != 1 {
		return m, nil
	}

	m.setup.SelectedMode = configmodel.ServerModeCustomOrigin
	m.setup.MenuIndex = 1
	m.setup.InputFocused = true
	return m, m.setup.OriginInput.Focus()
}

// activateSetupSelection runs the currently selected setup menu action.
// Authored by: OpenCode
func (m *Model) activateSetupSelection() (tea.Model, tea.Cmd) {
	switch m.setup.MenuIndex {
	case 0:
		return m.selectGhostfolioCloud()
	case 1:
		m.setup.ValidationMessage = ""
		return m.focusSetupOriginInputFromMenu()
	case 2:
		return m.saveSetupSelection()
	default:
		return m, nil
	}
}

// selectGhostfolioCloud applies the default hosted origin to setup state.
// Authored by: OpenCode
func (m *Model) selectGhostfolioCloud() (tea.Model, tea.Cmd) {
	m.setup.SelectedMode = configmodel.ServerModeGhostfolioCloud
	m.setup.OriginInput.SetValue(configmodel.GhostfolioCloudOrigin)
	m.setup.ValidationMessage = ""
	m.blurSetupOriginInput()
	return m, nil
}

// saveSetupSelection validates and persists the currently selected setup.
// Authored by: OpenCode
func (m *Model) saveSetupSelection() (tea.Model, tea.Cmd) {
	if !m.setupCanSave() {
		m.setup.ValidationMessage = "Provide a valid Ghostfolio origin before saving setup."
		return m, nil
	}

	m.setup.ValidationMessage = "Saving setup..."
	return m, m.saveSetupCmd(runtime.SaveSetupRequest{
		ServerMode:   m.setup.SelectedMode,
		ServerOrigin: m.selectedSetupOrigin(),
		SavedAt:      time.Now(),
	})
}

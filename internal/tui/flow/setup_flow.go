// Package flow owns the Bubble Tea root model and workflow routing for this
// validation-only slice.
// Authored by: OpenCode
package flow

import (
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
)

// updateSetup handles setup workflow input, validation, and persistence.
// Authored by: OpenCode
func (m *Model) updateSetup(message tea.Msg) (tea.Model, tea.Cmd) {
	switch typedMessage := message.(type) {
	case setupSavedMsg:
		if typedMessage.Err != nil {
			m.setup.ValidationMessage = "Setup could not be saved. Try again."
			return m, nil
		}
		var config = typedMessage.Config
		m.currentConfig = &config
		m.enterMainMenu()
		return m, nil
	case tea.KeyPressMsg:
		if m.setup.InputFocused {
			switch {
			case key.Matches(typedMessage, focusBinding()), key.Matches(typedMessage, cancelBinding()):
				m.setup.InputFocused = false
				m.setup.OriginInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.setup.OriginInput, cmd = m.setup.OriginInput.Update(message)
				m.setup.ValidationMessage = ""
				return m, cmd
			}
		}

		switch {
		case key.Matches(typedMessage, upBinding()):
			if m.setup.MenuIndex > 0 {
				m.setup.MenuIndex--
			}
		case key.Matches(typedMessage, downBinding()):
			if m.setup.MenuIndex < len(m.setupMenuItems())-1 {
				m.setup.MenuIndex++
			}
		case key.Matches(typedMessage, focusBinding()):
			if m.setup.SelectedMode == configmodel.ServerModeCustomOrigin || m.setup.MenuIndex == 1 {
				m.setup.SelectedMode = configmodel.ServerModeCustomOrigin
				m.setup.MenuIndex = 1
				m.setup.InputFocused = true
				return m, m.setup.OriginInput.Focus()
			}
		case key.Matches(typedMessage, cancelBinding()):
			if m.currentConfig != nil {
				m.enterMainMenu()
			}
		case key.Matches(typedMessage, enterBinding()):
			switch m.setup.MenuIndex {
			case 0:
				m.setup.SelectedMode = configmodel.ServerModeGhostfolioCloud
				m.setup.OriginInput.SetValue(configmodel.GhostfolioCloudOrigin)
				m.setup.ValidationMessage = ""
				m.setup.InputFocused = false
				m.setup.OriginInput.Blur()
			case 1:
				m.setup.SelectedMode = configmodel.ServerModeCustomOrigin
				m.setup.ValidationMessage = ""
				m.setup.InputFocused = true
				return m, m.setup.OriginInput.Focus()
			case 2:
				if !m.setupCanSave() {
					m.setup.ValidationMessage = "Provide a valid Ghostfolio origin before saving setup."
					return m, nil
				}

				var config, err = configmodel.NewSetupConfig(
					m.setup.SelectedMode,
					m.selectedSetupOrigin(),
					m.deps.Options.AllowDevHTTP,
					time.Now(),
				)
				if err != nil {
					m.setup.ValidationMessage = err.Error()
					return m, nil
				}

				m.setup.ValidationMessage = "Saving setup..."
				return m, m.saveSetupCmd(config)
			}
		}
	}

	return m, nil
}

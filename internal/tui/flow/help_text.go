// Package flow owns the Bubble Tea root model and workflow routing for this
// sync-and-storage slice.
// Authored by: OpenCode
package flow

import (
	"charm.land/bubbles/v2/key"

	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

// setupHelpText renders the visible hotkeys for the setup screen.
// Authored by: OpenCode
func (m *Model) setupHelpText() string {
	var bindings = []key.Binding{upBinding(), downBinding(), enterBinding(), focusBinding(), quitBinding()}
	if m.currentConfig != nil {
		bindings = append(bindings, cancelBinding())
	}
	return component.RenderHelp(component.ContentWidthForScreen(m.width), component.Bindings{Short: bindings})
}

// mainMenuHelpText renders the visible hotkeys for the main menu.
//
// Authored by: OpenCode
func (m *Model) mainMenuHelpText() string {
	return component.RenderHelp(
		component.ContentWidthForScreen(m.width),
		component.Bindings{Short: []key.Binding{enterBinding(), editSetupBinding(), quitBinding()}},
	)
}

// syncHelpText renders the visible hotkeys for the sync screen.
//
// Authored by: OpenCode
func (m *Model) syncHelpText() string {
	var bindings = []key.Binding{
		upBinding(),
		downBinding(),
		enterBinding(),
		quitBinding(),
	}
	if !m.sync.UseContextToken {
		bindings = append(bindings, focusBinding())
	}

	return component.RenderHelp(
		component.ContentWidthForScreen(m.width),
		component.Bindings{
			Short: bindings,
		},
	)
}

// syncReportsHelpText renders the visible hotkeys for the unlocked Sync and Reports menu.
// Authored by: OpenCode
func (m *Model) syncReportsHelpText() string {
	return component.RenderHelp(
		component.ContentWidthForScreen(m.width),
		component.Bindings{Short: []key.Binding{upBinding(), downBinding(), enterBinding(), quitBinding()}},
	)
}

// reportSelectionHelpText renders the visible hotkeys for report selection.
// Authored by: OpenCode
func (m *Model) reportSelectionHelpText() string {
	return component.RenderHelp(
		component.ContentWidthForScreen(m.width),
		component.Bindings{Short: []key.Binding{upBinding(), downBinding(), enterBinding(), focusBinding(), quitBinding()}},
	)
}

// reportBusyHelpText renders the visible hotkeys for report busy state.
// Authored by: OpenCode
func (m *Model) reportBusyHelpText() string {
	return component.RenderHelp(
		component.ContentWidthForScreen(m.width),
		component.Bindings{Short: []key.Binding{quitBinding()}},
	)
}

// reportResultHelpText renders the visible hotkeys for report result navigation.
// Authored by: OpenCode
func (m *Model) reportResultHelpText() string {
	return component.RenderHelp(
		component.ContentWidthForScreen(m.width),
		component.Bindings{Short: []key.Binding{upBinding(), downBinding(), enterBinding(), quitBinding()}},
	)
}

// serverReplacementHelpText renders the visible hotkeys for the server-replacement screen.
// Authored by: OpenCode
func (m *Model) serverReplacementHelpText() string {
	return component.RenderHelp(
		component.ContentWidthForScreen(m.width),
		component.Bindings{Short: []key.Binding{upBinding(), downBinding(), enterBinding(), quitBinding()}},
	)
}

// resultHelpText renders the visible hotkeys for the sync-result screen.
//
// Authored by: OpenCode
func (m *Model) resultHelpText() string {
	return component.RenderHelp(
		component.ContentWidthForScreen(m.width),
		component.Bindings{Short: []key.Binding{upBinding(), downBinding(), enterBinding(), quitBinding()}},
	)
}

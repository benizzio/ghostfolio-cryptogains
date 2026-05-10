// Package component contains shared TUI styling and rendering helpers.
// Authored by: OpenCode
package component

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
)

// Bindings groups visible hotkeys for a screen and satisfies the Bubbles help
// contract.
//
// Example:
//
//	bindings := component.Bindings{Short: []key.Binding{}}
//	_ = bindings.ShortHelp()
//
// Authored by: OpenCode
type Bindings struct {
	Short []key.Binding
	Full  [][]key.Binding
}

// ShortHelp returns the compact hotkey list.
// Authored by: OpenCode
func (b Bindings) ShortHelp() []key.Binding {
	return b.Short
}

// FullHelp returns the expanded hotkey list.
// Authored by: OpenCode
func (b Bindings) FullHelp() [][]key.Binding {
	return b.Full
}

// RenderHelp renders screen-local help text using the shared Bubbles help
// view.
//
// Example:
//
//	text := component.RenderHelp(80, bindings)
//	_ = text
//
// Authored by: OpenCode
func RenderHelp(width int, bindings Bindings) string {
	if width <= 0 {
		width = 40
	}
	var helpModel = help.New()
	helpModel.SetWidth(width)
	return strings.TrimSpace(helpModel.View(bindings))
}

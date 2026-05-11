// Package component contains shared TUI styling and rendering helpers.
// Authored by: OpenCode
package component

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
)

// Bindings groups the hotkeys that one screen exposes through the shared help
// footer and optional expanded help view.
//
// Populate `Short` with the bindings that should remain visible in the compact
// footer. Populate `Full` when a screen also needs grouped extended help. The
// type satisfies the Bubbles help contract directly, so callers can pass a
// `Bindings` value anywhere the help component expects a provider.
//
// Authored by: OpenCode
type Bindings struct {
	Short []key.Binding
	Full  [][]key.Binding
}

// ShortHelp returns the compact hotkey list.
//
// Example:
//
//	bindings := component.Bindings{Short: []key.Binding{}}
//	_ = bindings.ShortHelp()
//
// ShortHelp satisfies the Bubbles help contract for the collapsed help view and
// returns the exact bindings that should be shown in the single-line footer.
// Authored by: OpenCode
func (b Bindings) ShortHelp() []key.Binding {
	return b.Short
}

// FullHelp returns the expanded hotkey list.
//
// Example:
//
//	bindings := component.Bindings{Full: [][]key.Binding{{}}}
//	_ = bindings.FullHelp()
//
// FullHelp satisfies the Bubbles help contract for the expanded help view and
// returns grouped bindings in the order that should be rendered.
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
// `RenderHelp` applies the shared help presenter used across the full-screen
// workflows, trims surrounding whitespace from the Bubbles output, and falls
// back to a readable minimum width when the caller provides a non-positive
// width. Use it when a screen needs footer help text that stays visually
// aligned with the rest of the TUI.
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

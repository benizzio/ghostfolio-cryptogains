// Package testutil provides shared helpers for the repository's black-box test
// suites so unit and integration tests can reuse the same Bubble Tea command
// execution behavior.
// Authored by: OpenCode
package testutil

import (
	tea "charm.land/bubbletea/v2"
)

// RunCmd executes one Bubble Tea command and returns its resulting message.
//
// It keeps tests focused on workflow state changes without repeating the nil
// command guard at each call site.
//
// Example usage:
//
//	updated, cmd := model.Update(msg)
//	result := testutil.RunCmd(cmd)
//	updated, _ = model.Update(result)
//
// Authored by: OpenCode
func RunCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

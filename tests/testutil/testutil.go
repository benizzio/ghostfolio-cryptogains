// Package testutil provides shared helpers for the repository's black-box test
// suites so unit and integration tests can reuse the same command execution and
// string-matching behavior.
// Authored by: OpenCode
package testutil

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// RunCmd executes one Bubble Tea command and returns its resulting message.
// Authored by: OpenCode
func RunCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

// Contains reports whether expected is present within content.
// Authored by: OpenCode
func Contains(content string, expected string) bool {
	return strings.Contains(content, expected)
}

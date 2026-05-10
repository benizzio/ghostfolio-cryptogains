package integration

import "strings"

import tea "charm.land/bubbletea/v2"

func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func contains(content string, expected string) bool {
	return strings.Contains(content, expected)
}

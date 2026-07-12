package integration

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil/runtimeflow"
)

// openSyncEntry enters the sync workflow from the main menu through the Sync
// and Reports unlock step.
// Authored by: OpenCode
func openSyncEntry(t *testing.T, model *flow.Model) *flow.Model {
	t.Helper()
	var updated tea.Model
	var cmd tea.Cmd
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	model = runtimeflow.AssertFlowModel(t, updated)
	if model.ActiveScreen() != "sync_reports_unlock" {
		t.Fatalf("expected sync and reports unlock screen, got %s", model.ActiveScreen())
	}
	return model
}

// typeToken types a token into the focused Ghostfolio security-token field.
// Authored by: OpenCode
func typeToken(t *testing.T, model *flow.Model, token string) *flow.Model {
	t.Helper()
	for _, runeValue := range token {
		var updated tea.Model
		var cmd tea.Cmd
		updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Text: string(runeValue), Code: runeValue}))
		_ = testutil.RunCmd(cmd)
		model = runtimeflow.AssertFlowModel(t, updated)
	}
	return model
}

// blurTokenInputFromSyncEntry returns focus from the token input to the unlock
// or sync-entry menu.
// Authored by: OpenCode
func blurTokenInputFromSyncEntry(t *testing.T, model *flow.Model) *flow.Model {
	t.Helper()
	var updated tea.Model
	var cmd tea.Cmd
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	_ = testutil.RunCmd(cmd)
	return runtimeflow.AssertFlowModel(t, updated)
}

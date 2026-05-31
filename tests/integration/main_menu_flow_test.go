package integration

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
)

func TestMainMenuOnlyExposesSyncAndReportsWorkflow(t *testing.T) {
	t.Parallel()

	var config = mustCloudSetupConfig(t)

	var model = flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &config}, false, integrationSyncService{}))

	var content = model.View().Content
	if !strings.Contains(content, "Sync and Reports") {
		t.Fatalf("expected Sync and Reports action")
	}
	if strings.Contains(content, "Protected Data:") || strings.Contains(content, "Last Successful Sync") {
		t.Fatalf("unexpected protected metadata exposure on main menu: %q", content)
	}
}

func TestMainMenuEnterNavigatesToSyncReportsUnlock(t *testing.T) {
	t.Parallel()

	var config = mustCloudSetupConfig(t)

	var model = flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &config}, false, integrationSyncService{}))

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	model = updated.(*flow.Model)

	if model.ActiveScreen() != "sync_reports_unlock" {
		t.Fatalf("expected sync and reports unlock screen, got %s", model.ActiveScreen())
	}
}

func TestFocusedTokenInputEnterReturnsToSyncMenuPath(t *testing.T) {
	t.Parallel()

	var config = mustCloudSetupConfig(t)

	var model = flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &config}, false, integrationSyncService{}))

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	model = updated.(*flow.Model)

	updated, _ = model.Update(tea.PasteMsg{Content: "token-123"})
	model = updated.(*flow.Model)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*flow.Model)

	if got := model.View().Content; !strings.Contains(got, "> Unlock") {
		t.Fatalf("expected unlock menu focus to return to Unlock, got %q", got)
	}

	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*flow.Model)
	if model.ActiveScreen() != "sync_reports_menu" {
		t.Fatalf("expected unlock to route into sync and reports menu, got %s", model.ActiveScreen())
	}
	if got := model.View().Content; !strings.Contains(got, "Sync Data") || !strings.Contains(got, "Generate Capital Gains Report") {
		t.Fatalf("expected sync and reports context actions after unlock, got %q", got)
	}
	if got := model.View().Content; !strings.Contains(got, "no synced data available") {
		t.Fatalf("expected no-data readiness after unlock, got %q", got)
	}
	if cmd != nil {
		_ = testutil.RunCmd(cmd)
	}
}

func TestFocusedTokenInputPasteDoesNotTriggerWorkflowNavigation(t *testing.T) {
	t.Parallel()

	var config = mustCloudSetupConfig(t)

	var model = flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &config}, false, integrationSyncService{}))

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	model = updated.(*flow.Model)

	updated, _ = model.Update(tea.PasteStartMsg{})
	model = updated.(*flow.Model)
	updated, _ = model.Update(tea.PasteMsg{Content: "token-123"})
	model = updated.(*flow.Model)
	updated, _ = model.Update(tea.PasteEndMsg{})
	model = updated.(*flow.Model)

	if model.ActiveScreen() != "sync_reports_unlock" {
		t.Fatalf("expected unlock screen to remain active during paste, got %s", model.ActiveScreen())
	}
	if got := model.View().Content; !strings.Contains(got, "*********") {
		t.Fatalf("expected pasted token to remain masked, got %q", got)
	}
	if got := model.View().Content; !strings.Contains(got, "Unlock") {
		t.Fatalf("expected unlock workflow to remain active after paste, got %q", got)
	}
}

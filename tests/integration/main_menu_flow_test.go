package integration

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
)

func TestMainMenuOnlyExposesSyncDataWorkflow(t *testing.T) {
	t.Parallel()

	var config = mustCloudSetupConfig(t)

	var model = flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &config}, false, integrationSyncService{}))

	var content = model.View().Content
	if !strings.Contains(content, "Sync Data") {
		t.Fatalf("expected Sync Data action")
	}
	if strings.Contains(content, "Report") {
		t.Fatalf("unexpected reporting workflow exposure: %q", content)
	}
}

func TestMainMenuEnterNavigatesToSync(t *testing.T) {
	t.Parallel()

	var config = mustCloudSetupConfig(t)

	var model = flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &config}, false, integrationSyncService{}))

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	model = updated.(*flow.Model)

	if model.ActiveScreen() != "sync" {
		t.Fatalf("expected sync screen, got %s", model.ActiveScreen())
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

	if got := model.View().Content; !strings.Contains(got, "> Start Sync") {
		t.Fatalf("expected sync menu focus to return to Start Sync, got %q", got)
	}

	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*flow.Model)
	if got := model.View().Content; !strings.Contains(got, "Syncing and storing activity history") {
		t.Fatalf("expected sync path to remain reachable, got %q", got)
	}
	_ = testutil.RunCmd(cmd)
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

	if model.ActiveScreen() != "sync" {
		t.Fatalf("expected sync screen to remain active during paste, got %s", model.ActiveScreen())
	}
	if got := model.View().Content; !strings.Contains(got, "*********") {
		t.Fatalf("expected pasted token to remain masked, got %q", got)
	}
	if got := model.View().Content; !strings.Contains(got, "Start Sync") {
		t.Fatalf("expected sync workflow to remain active after paste, got %q", got)
	}
}

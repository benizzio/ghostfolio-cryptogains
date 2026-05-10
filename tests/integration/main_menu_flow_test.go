package integration

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
)

func TestMainMenuOnlyExposesSyncDataWorkflow(t *testing.T) {
	t.Parallel()

	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var model = flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &config}, false, integrationSyncService{}))

	var content = model.View().Content
	if !contains(content, "Sync Data") {
		t.Fatalf("expected Sync Data action")
	}
	if contains(content, "Report") {
		t.Fatalf("unexpected reporting workflow exposure: %q", content)
	}
}

func TestMainMenuEnterNavigatesToSyncValidation(t *testing.T) {
	t.Parallel()

	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var model = flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &config}, false, integrationSyncService{}))

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = runCmd(cmd)
	model = updated.(*flow.Model)

	if model.ActiveScreen() != "sync_validation" {
		t.Fatalf("expected sync validation, got %s", model.ActiveScreen())
	}
}

func TestFocusedTokenInputEnterReturnsToValidationMenuPath(t *testing.T) {
	t.Parallel()

	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var model = flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &config}, false, integrationSyncService{}))

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = runCmd(cmd)
	model = updated.(*flow.Model)

	updated, _ = model.Update(tea.PasteMsg{Content: "token-123"})
	model = updated.(*flow.Model)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*flow.Model)

	if got := model.View().Content; !contains(got, "> Validate Communication") {
		t.Fatalf("expected sync menu focus to return to Validate Communication, got %q", got)
	}

	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*flow.Model)
	if got := model.View().Content; !contains(got, "Validating Ghostfolio communication") {
		t.Fatalf("expected validation path to remain reachable, got %q", got)
	}
	_ = runCmd(cmd)
}

func TestFocusedTokenInputPasteDoesNotTriggerWorkflowNavigation(t *testing.T) {
	t.Parallel()

	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var model = flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &config}, false, integrationSyncService{}))

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = runCmd(cmd)
	model = updated.(*flow.Model)

	updated, _ = model.Update(tea.PasteStartMsg{})
	model = updated.(*flow.Model)
	updated, _ = model.Update(tea.PasteMsg{Content: "token-123"})
	model = updated.(*flow.Model)
	updated, _ = model.Update(tea.PasteEndMsg{})
	model = updated.(*flow.Model)

	if model.ActiveScreen() != "sync_validation" {
		t.Fatalf("expected sync validation screen to remain active during paste, got %s", model.ActiveScreen())
	}
	if got := model.View().Content; !contains(got, "*********") {
		t.Fatalf("expected pasted token to remain masked, got %q", got)
	}
	if got := model.View().Content; !contains(got, "Validate Communication") {
		t.Fatalf("expected sync workflow to remain active after paste, got %q", got)
	}
}

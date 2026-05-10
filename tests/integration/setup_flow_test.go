package integration

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
)

type integrationSyncService struct{}

func (integrationSyncService) Validate(context.Context, configmodel.AppSetupConfig, string) runtime.ValidationOutcome {
	return runtime.ValidationOutcome{Success: true, SummaryMessage: "Communication with the selected Ghostfolio server is working.", DetailReason: "communication_ok", FollowUpNote: "No Ghostfolio data was stored locally, and reporting is not available in this slice."}
}

func TestFreshRunCompletesSetupAndReachesMainMenu(t *testing.T) {
	t.Parallel()

	var store = configstore.NewJSONStore(t.TempDir())
	var model = flow.NewModel(flow.Dependencies{
		Options:     bootstrap.DefaultOptions(),
		Startup:     bootstrap.StartupState{NeedsSetup: true},
		ConfigStore: store,
		SyncService: integrationSyncService{},
	})

	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*flow.Model)
	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = runCmd(cmd)
	model = updated.(*flow.Model)

	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	_ = runCmd(cmd)
	model = updated.(*flow.Model)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*flow.Model)
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	result := runCmd(cmd)
	updated, _ = model.Update(result)
	model = updated.(*flow.Model)

	if model.ActiveScreen() != "main_menu" {
		t.Fatalf("expected main menu, got %s", model.ActiveScreen())
	}

	var loaded, err = store.Load(context.Background())
	if err != nil {
		t.Fatalf("load remembered setup: %v", err)
	}
	if loaded.ServerOrigin != configmodel.GhostfolioCloudOrigin {
		t.Fatalf("remembered origin mismatch: %q", loaded.ServerOrigin)
	}
}

func TestStartupSkipsSetupWhenRememberedConfigExists(t *testing.T) {
	t.Parallel()

	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var model = flow.NewModel(flow.Dependencies{
		Options:     bootstrap.DefaultOptions(),
		Startup:     bootstrap.StartupState{ActiveConfig: &config},
		ConfigStore: configstore.NewJSONStore(t.TempDir()),
		SyncService: integrationSyncService{},
	})

	if model.ActiveScreen() != "main_menu" {
		t.Fatalf("expected main menu startup, got %s", model.ActiveScreen())
	}
}

func TestInvalidRememberedSetupFallsBackToSetup(t *testing.T) {
	t.Parallel()

	var model = flow.NewModel(flow.Dependencies{
		Options:     bootstrap.DefaultOptions(),
		Startup:     bootstrap.StartupState{NeedsSetup: true, InvalidSetupMessage: "The saved server selection is no longer valid. Complete setup again before sync validation can run."},
		ConfigStore: configstore.NewJSONStore(t.TempDir()),
		SyncService: integrationSyncService{},
	})

	if model.ActiveScreen() != "setup" {
		t.Fatalf("expected setup screen, got %s", model.ActiveScreen())
	}
	if got := model.View().Content; !contains(got, "saved server selection is no longer valid") {
		t.Fatalf("expected invalid remembered setup message, got %q", got)
	}
}

func TestSetupFileRemovalAfterStartupDoesNotBreakCurrentRun(t *testing.T) {
	t.Parallel()

	var store = configstore.NewJSONStore(t.TempDir())
	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}
	if err := store.Save(context.Background(), config); err != nil {
		t.Fatalf("save config: %v", err)
	}

	var model = flow.NewModel(flow.Dependencies{
		Options:     bootstrap.DefaultOptions(),
		Startup:     bootstrap.StartupState{ActiveConfig: &config},
		ConfigStore: store,
		SyncService: integrationSyncService{},
	})

	if err := store.Delete(context.Background()); err != nil {
		t.Fatalf("delete setup file: %v", err)
	}

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = runCmd(cmd)
	model = updated.(*flow.Model)

	if model.ActiveScreen() != "sync_validation" {
		t.Fatalf("expected current run to keep working after setup file removal")
	}
}

func TestFocusedCustomOriginInputEnterReturnsToSavePath(t *testing.T) {
	t.Parallel()

	var model = flow.NewModel(flow.Dependencies{
		Options:     bootstrap.DefaultOptions(),
		Startup:     bootstrap.StartupState{NeedsSetup: true},
		ConfigStore: configstore.NewJSONStore(t.TempDir()),
		SyncService: integrationSyncService{},
	})

	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*flow.Model)
	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = runCmd(cmd)
	model = updated.(*flow.Model)

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*flow.Model)

	if got := model.View().Content; !contains(got, "> Save And Continue") {
		t.Fatalf("expected setup menu focus to return to Save And Continue, got %q", got)
	}

	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	result := runCmd(cmd)
	updated, _ = model.Update(result)
	model = updated.(*flow.Model)

	if model.ActiveScreen() != "main_menu" {
		t.Fatalf("expected save path to remain reachable, got %s", model.ActiveScreen())
	}
	if got := model.View().Content; !contains(got, "ghostfolio-cryptogains") {
		t.Fatalf("expected persistent header on main menu, got %q", got)
	}
}

func TestFocusedCustomOriginInputPasteDoesNotTriggerWorkflowNavigation(t *testing.T) {
	t.Parallel()

	var model = flow.NewModel(flow.Dependencies{
		Options:     bootstrap.DefaultOptions(),
		Startup:     bootstrap.StartupState{NeedsSetup: true},
		ConfigStore: configstore.NewJSONStore(t.TempDir()),
		SyncService: integrationSyncService{},
	})

	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*flow.Model)
	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = runCmd(cmd)
	model = updated.(*flow.Model)

	updated, _ = model.Update(tea.PasteStartMsg{})
	model = updated.(*flow.Model)
	updated, _ = model.Update(tea.PasteMsg{Content: "https://localhost:8080"})
	model = updated.(*flow.Model)
	updated, _ = model.Update(tea.PasteEndMsg{})
	model = updated.(*flow.Model)

	if model.ActiveScreen() != "setup" {
		t.Fatalf("expected setup screen to remain active during paste, got %s", model.ActiveScreen())
	}
	if got := model.View().Content; !contains(got, "https://localhost:8080") {
		t.Fatalf("expected pasted origin in setup input, got %q", got)
	}
	if got := model.View().Content; !contains(got, "Use Custom Server") {
		t.Fatalf("expected setup workflow to remain active after paste, got %q", got)
	}
}

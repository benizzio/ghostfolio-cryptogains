package integration

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
)

func TestMainMenuOnlyExposesSyncDataWorkflow(t *testing.T) {
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

	var model = flow.NewModel(flow.Dependencies{
		Options:     bootstrap.DefaultOptions(),
		Startup:     bootstrap.StartupState{ActiveConfig: &config},
		ConfigStore: configstore.NewJSONStore(t.TempDir()),
		SyncService: integrationSyncService{},
	})

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = runCmd(cmd)
	model = updated.(*flow.Model)

	if model.ActiveScreen() != "sync_validation" {
		t.Fatalf("expected sync validation, got %s", model.ActiveScreen())
	}
}

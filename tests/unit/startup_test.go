package unit

import (
	"context"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
)

func TestLoadStartupStateWithoutRememberedSetup(t *testing.T) {
	t.Parallel()

	var state, err = bootstrap.LoadStartupState(context.Background(), configstore.NewJSONStore(t.TempDir()), false)
	if err != nil {
		t.Fatalf("load startup state: %v", err)
	}
	if !state.NeedsSetup || state.SetupRequirementReason != bootstrap.SetupRequirementMissing {
		t.Fatalf("expected setup to be required")
	}
}

func TestLoadStartupStateReturnsActiveConfigWhenValid(t *testing.T) {
	t.Parallel()

	var store = configstore.NewJSONStore(t.TempDir())
	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}
	if err := store.Save(context.Background(), config); err != nil {
		t.Fatalf("save setup config: %v", err)
	}

	var state bootstrap.StartupState
	state, err = bootstrap.LoadStartupState(context.Background(), store, false)
	if err != nil {
		t.Fatalf("load startup state: %v", err)
	}
	if state.ActiveConfig == nil || state.ActiveConfig.ServerOrigin != config.ServerOrigin {
		t.Fatalf("expected active config to be loaded")
	}
}

package integration

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
)

// assertFlowModel converts the updated Bubble Tea model into the integration
// test's concrete flow model type.
// Authored by: OpenCode
func assertFlowModel(t *testing.T, updated tea.Model) *flow.Model {
	t.Helper()

	var model, ok = updated.(*flow.Model)
	if !ok {
		t.Fatalf("expected updated model to be *flow.Model, got %T", updated)
	}

	return model
}

// newFlowDependencies constructs test workflow dependencies using a temporary
// JSON-backed setup store for the current test.
// Authored by: OpenCode
func newFlowDependencies(t *testing.T, startup bootstrap.StartupState, allowDevHTTP bool, syncService runtime.SyncService) flow.Dependencies {
	t.Helper()
	return newFlowDependenciesWithStore(t, startup, allowDevHTTP, syncService, configstore.NewJSONStore(t.TempDir()))
}

// newFlowDependenciesWithStore constructs test workflow dependencies using the
// provided store and the repository's default bootstrap options.
// Authored by: OpenCode
func newFlowDependenciesWithStore(t *testing.T, startup bootstrap.StartupState, allowDevHTTP bool, syncService runtime.SyncService, store configstore.Store) flow.Dependencies {
	t.Helper()

	var options = bootstrap.DefaultOptions()
	options.AllowDevHTTP = allowDevHTTP

	return flow.Dependencies{
		Options:      options,
		Startup:      startup,
		SetupService: runtime.NewSetupService(store, allowDevHTTP),
		SyncService:  syncService,
	}
}

// mustCloudSetupConfig returns a valid remembered Ghostfolio Cloud setup for
// integration tests that start from the main menu.
// Authored by: OpenCode
func mustCloudSetupConfig(t *testing.T) configmodel.AppSetupConfig {
	t.Helper()

	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	return config
}

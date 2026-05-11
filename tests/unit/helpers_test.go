package unit

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
)

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

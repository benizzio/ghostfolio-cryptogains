package integration

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
)

// newFlowDependencies constructs lightweight flow dependencies for sync-focused
// integration tests using a temporary JSON-backed setup store.
// Authored by: OpenCode
func newFlowDependencies(t *testing.T, startup bootstrap.StartupState, allowDevHTTP bool, syncService runtime.SyncService) flow.Dependencies {
	t.Helper()
	return newFlowDependenciesWithStore(t, startup, allowDevHTTP, syncService, configstore.NewJSONStore(t.TempDir()))
}

// newFlowDependenciesWithStore constructs lightweight flow dependencies with
// the supplied setup store for sync-focused integration tests.
// Authored by: OpenCode
func newFlowDependenciesWithStore(t *testing.T, startup bootstrap.StartupState, allowDevHTTP bool, syncService runtime.SyncService, store configstore.Store) flow.Dependencies {
	t.Helper()

	var options = bootstrap.DefaultOptions()
	options.AllowDevHTTP = allowDevHTTP

	return flow.Dependencies{Options: options, Startup: startup, SetupService: runtime.NewSetupService(store, allowDevHTTP), SyncService: syncService}
}

package integration

import "strings"

import tea "charm.land/bubbletea/v2"

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
)

func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func contains(content string, expected string) bool {
	return strings.Contains(content, expected)
}

func newFlowDependencies(t *testing.T, startup bootstrap.StartupState, allowDevHTTP bool, syncService runtime.SyncService) flow.Dependencies {
	t.Helper()
	return newFlowDependenciesWithStore(t, startup, allowDevHTTP, syncService, configstore.NewJSONStore(t.TempDir()))
}

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

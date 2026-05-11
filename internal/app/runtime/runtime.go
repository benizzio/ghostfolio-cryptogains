// Package runtime assembles application dependencies for the TUI runtime.
// Authored by: OpenCode
package runtime

import (
	"fmt"
	"net/http"
	"os"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
)

// App contains the assembled dependencies required by the Bubble Tea model.
//
// Example:
//
//	app, err := runtime.New(bootstrap.DefaultOptions())
//	if err != nil {
//		panic(err)
//	}
//	_ = app.ConfigStore
//
// App keeps the bootstrap store available for startup loading and exposes the
// application services that the TUI uses for setup persistence and sync
// validation.
// Authored by: OpenCode
type App struct {
	Options      bootstrap.Options
	ConfigStore  configstore.Store
	SetupService SetupService
	SyncService  SyncService
}

// New assembles the runtime dependencies required by the application.
//
// Example:
//
//	app, err := runtime.New(bootstrap.DefaultOptions())
//	if err != nil {
//		panic(err)
//	}
//	_ = app.SyncService
//
// New resolves the config directory, constructs the bootstrap store, and wires
// the setup and sync application services used by the terminal workflow.
// Authored by: OpenCode
func New(options bootstrap.Options) (*App, error) {
	var baseConfigDir = options.ConfigDir
	if baseConfigDir == "" {
		var userConfigDir, err = os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("resolve user config directory: %w", err)
		}
		baseConfigDir = userConfigDir
	}

	var bootstrapStore = configstore.NewJSONStore(baseConfigDir)
	var setupService = NewSetupService(bootstrapStore, options.AllowDevHTTP)
	var syncService = NewSyncService(ghostfolioclient.New(&http.Client{Timeout: options.RequestTimeout}), options.RequestTimeout)

	return &App{
		Options:      options,
		ConfigStore:  bootstrapStore,
		SetupService: setupService,
		SyncService:  syncService,
	}, nil
}

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
// Authored by: OpenCode
type App struct {
	Options     bootstrap.Options
	ConfigStore configstore.Store
	SyncService SyncService
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
	var syncService = NewSyncService(ghostfolioclient.New(&http.Client{}), options.RequestTimeout)

	return &App{
		Options:     options,
		ConfigStore: bootstrapStore,
		SyncService: syncService,
	}, nil
}

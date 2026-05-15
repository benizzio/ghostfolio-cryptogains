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
	snapshotenvelope "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/envelope"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	syncvalidate "github.com/benizzio/ghostfolio-cryptogains/internal/sync/validate"
)

// App contains the assembled dependencies required by the Bubble Tea model.
//
// App keeps the bootstrap store available for startup loading and exposes the
// application services that the TUI uses for setup persistence and sync
// validation.
//
// Authored by: OpenCode
type App struct {
	Options        bootstrap.Options
	ConfigStore    configstore.Store
	DecimalService decimalsupport.Service
	SyncNormalizer syncnormalize.Normalizer
	SyncValidator  syncvalidate.Validator
	SnapshotCodec  snapshotenvelope.Codec
	SnapshotStore  snapshotstore.Store
	SetupService   SetupService
	SyncService    SyncService
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
	var decimalService = decimalsupport.NewService()
	var syncNormalizer = syncnormalize.NewNormalizer()
	var syncValidator = syncvalidate.NewValidator()
	var snapshotCodec = snapshotenvelope.NewJSONCodec()
	var protectedSnapshotStore = snapshotstore.NewEncryptedStore(baseConfigDir, snapshotCodec)
	var setupService = NewSetupService(bootstrapStore, options.AllowDevHTTP)
	var syncService = NewSyncService(
		ghostfolioclient.New(&http.Client{Timeout: options.RequestTimeout}),
		options.RequestTimeout,
		baseConfigDir,
		options.AllowDevHTTP,
		decimalService,
		syncNormalizer,
		syncValidator,
		protectedSnapshotStore,
	)

	return &App{
		Options:        options,
		ConfigStore:    bootstrapStore,
		DecimalService: decimalService,
		SyncNormalizer: syncNormalizer,
		SyncValidator:  syncValidator,
		SnapshotCodec:  snapshotCodec,
		SnapshotStore:  protectedSnapshotStore,
		SetupService:   setupService,
		SyncService:    syncService,
	}, nil
}

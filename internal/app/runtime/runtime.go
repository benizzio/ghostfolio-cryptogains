// Package runtime assembles application dependencies for the TUI runtime.
// Authored by: OpenCode
package runtime

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
	"github.com/benizzio/ghostfolio-cryptogains/internal/integration/currency"
	reportcalculate "github.com/benizzio/ghostfolio-cryptogains/internal/report/calculate"
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
// storage.
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
	ReportService  ReportService
}

// SetReportBundleRendererForTesting replaces the assembled report renderer for
// deterministic runtime integration tests. Production assembly does not call
// this method and does not derive renderer behavior from process environment.
//
// Example:
//
//	app.SetReportBundleRendererForTesting(func(reportmodel.ReportOutputFormat, reportmodel.CapitalGainsReport) ([]reportmodel.ReportDocument, error) {
//		return nil, errors.New("forced render failure")
//	})
//
// Authored by: OpenCode
func (app *App) SetReportBundleRendererForTesting(renderer reportBundleRenderer) error {
	if app == nil {
		return errors.New("runtime app is unavailable")
	}
	var service, ok = app.ReportService.(*reportService)
	if !ok || service == nil {
		return errors.New("runtime report service does not support renderer replacement")
	}
	service.renderBundle = renderer
	return nil
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
	var reportCurrencyRates = currency.NewCurrencyRateService(currency.NewCurrencyRateSessionCache())
	return NewWithReportCurrencyRateService(options, reportCurrencyRates)
}

// NewWithReportCurrencyRateService assembles runtime dependencies with a
// caller-supplied report currency-rate service.
//
// Production entrypoints should call New so the runtime uses the fixed official
// provider service created by currency.NewCurrencyRateService. This constructor
// exists for callers that already own a report currency-rate service and need
// runtime to coordinate snapshot, report, rendering, and output lifecycle around
// that dependency. The supplied service is passed only to report calculation and
// is not persisted.
//
// Example:
//
//	app, err := runtime.NewWithReportCurrencyRateService(options, currencyRates)
//	if err != nil {
//		panic(err)
//	}
//	_ = app.ReportService
//
// Authored by: OpenCode
func NewWithReportCurrencyRateService(options bootstrap.Options, reportCurrencyRates reportcalculate.CurrencyRateService) (*App, error) {
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
	var sharedSnapshots = newSnapshotLifecycle(protectedSnapshotStore, newActiveSnapshotState(), protectedPayloadBuilder{})
	var setupService = NewSetupService(bootstrapStore, options.AllowDevHTTP)
	var syncService = newSyncService(
		ghostfolioclient.New(&http.Client{Timeout: options.RequestTimeout}),
		options.RequestTimeout,
		baseConfigDir,
		options.AllowDevHTTP,
		decimalService,
		syncNormalizer,
		syncValidator,
		sharedSnapshots,
	)
	var reportService = newReportService(sharedSnapshots, baseConfigDir, options.AllowDevHTTP, reportCurrencyRates)

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
		ReportService:  reportService,
	}, nil
}

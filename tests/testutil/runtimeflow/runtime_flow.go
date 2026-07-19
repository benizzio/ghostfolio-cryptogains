// Package runtimeflow provides reusable runtime-backed black-box fixtures for
// repository test suites.
//
// Authored by: OpenCode
package runtimeflow

import (
	"context"
	"testing"
	"time"

	"github.com/cockroachdb/apd/v3"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/integration/currency"
	reportcalculate "github.com/benizzio/ghostfolio-cryptogains/internal/report/calculate"
	reportpdf "github.com/benizzio/ghostfolio-cryptogains/internal/report/pdf"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil/runtimeapp"
)

// RuntimeBackedFlowHarness groups the real application services and TUI model
// used by black-box report workflows. For example, construct it with
// NewRuntimeBackedFlowHarness, seed a snapshot, then drive Model through a report flow.
// Authored by: OpenCode
type RuntimeBackedFlowHarness struct {
	BaseDir string
	App     *runtime.App
	Config  configmodel.AppSetupConfig
	Model   *flow.Model
}

// NewRuntimeBackedFlowHarness creates an application-backed TUI harness with
// deterministic currency-rate evidence. For example, pass a temporary base
// directory and MustCloudSetupConfig(t) before seeding a protected snapshot.
// Authored by: OpenCode
func NewRuntimeBackedFlowHarness(t *testing.T, baseDir string, config configmodel.AppSetupConfig, allowDevHTTP bool) RuntimeBackedFlowHarness {
	t.Helper()
	return NewRuntimeBackedFlowHarnessWithCurrencyRateService(t, baseDir, config, allowDevHTTP, DeterministicCurrencyRates{})
}

// NewRuntimeBackedFlowHarnessWithCurrencyRateService creates an
// application-backed TUI harness with the supplied report currency-rate service.
// For example, pass a custom reportcalculate.CurrencyRateService when a test
// needs to control a rate lookup, then seed a snapshot and drive the returned
// Model through a report flow.
// Authored by: OpenCode
func NewRuntimeBackedFlowHarnessWithCurrencyRateService(
	t *testing.T,
	baseDir string,
	config configmodel.AppSetupConfig,
	allowDevHTTP bool,
	currencyRates reportcalculate.CurrencyRateService,
) RuntimeBackedFlowHarness {
	t.Helper()
	return NewRuntimeBackedFlowHarnessWithCurrencyRateServiceAndPDFByteFinalizer(t, baseDir, config, allowDevHTTP, currencyRates, nil)
}

// NewRuntimeBackedFlowHarnessWithCurrencyRateServiceAndPDFByteFinalizer creates
// an application-backed TUI harness with one renderer-scoped PDF finalizer.
// Authored by: OpenCode
func NewRuntimeBackedFlowHarnessWithCurrencyRateServiceAndPDFByteFinalizer(
	t *testing.T,
	baseDir string,
	config configmodel.AppSetupConfig,
	allowDevHTTP bool,
	currencyRates reportcalculate.CurrencyRateService,
	finalizer reportpdf.ByteFinalizer,
) RuntimeBackedFlowHarness {
	t.Helper()
	var options = bootstrap.DefaultOptions()
	options.ConfigDir = baseDir
	options.AllowDevHTTP = allowDevHTTP
	var app = runtimeapp.NewWithReportCurrencyRateServiceAndPDFByteFinalizer(t, options, currencyRates, finalizer)
	var err = app.ConfigStore.Save(context.Background(), config)
	if err != nil {
		t.Fatalf("save setup config: %v", err)
	}
	var model = flow.NewModel(flow.Dependencies{Options: options, Startup: bootstrap.StartupState{ActiveConfig: &config}, SetupService: app.SetupService, SyncService: app.SyncService, ReportService: app.ReportService})
	return RuntimeBackedFlowHarness{BaseDir: baseDir, App: app, Config: config, Model: model}
}

// NewRuntimeBackedFlowHarnessWithCurrencyRateServiceAndReportPipelineOptions creates a
// runtime-backed harness with immutable renderer-scoped report
// options while retaining the production calculation, snapshot, output, and
// TUI seams.
// Authored by: OpenCode
func NewRuntimeBackedFlowHarnessWithCurrencyRateServiceAndReportPipelineOptions(
	t *testing.T,
	baseDir string,
	config configmodel.AppSetupConfig,
	allowDevHTTP bool,
	currencyRates reportcalculate.CurrencyRateService,
	pipelineOptions runtime.ReportPipelineOptions,
) RuntimeBackedFlowHarness {
	t.Helper()
	var options = bootstrap.DefaultOptions()
	options.ConfigDir = baseDir
	options.AllowDevHTTP = allowDevHTTP
	var app, err = runtime.NewWithReportCurrencyRateServiceAndReportPipelineOptions(options, currencyRates, pipelineOptions)
	if err != nil {
		t.Fatalf("runtime new: %v", err)
	}
	err = app.ConfigStore.Save(context.Background(), config)
	if err != nil {
		t.Fatalf("save setup config: %v", err)
	}
	var model = flow.NewModel(flow.Dependencies{Options: options, Startup: bootstrap.StartupState{ActiveConfig: &config}, SetupService: app.SetupService, SyncService: app.SyncService, ReportService: app.ReportService})
	return RuntimeBackedFlowHarness{BaseDir: baseDir, App: app, Config: config, Model: model}
}

// DeterministicCurrencyRates supplies fixed currency-rate evidence to
// runtime-backed test fixtures. Use it with
// NewRuntimeBackedFlowHarnessWithCurrencyRateService when a test needs the
// standard deterministic conversion evidence.
// Authored by: OpenCode
type DeterministicCurrencyRates struct{}

// LookupRate returns fixed currency-rate evidence for a requested activity.
// For example, report calculation can call it through
// reportcalculate.CurrencyRateService to obtain the standard deterministic
// evidence without contacting a provider.
// Authored by: OpenCode
func (DeterministicCurrencyRates) LookupRate(_ context.Context, request currency.RateLookupRequest) (currency.ExchangeRateEvidence, error) {
	var rateDate = time.Date(request.ActivityDate.Year(), request.ActivityDate.Month(), request.ActivityDate.Day(), 0, 0, 0, 0, time.UTC)
	// The H.10 fixture marks the 2024-01-06 observation unavailable, so use its prior available date.
	if rateDate.Format(time.DateOnly) == "2024-01-06" {
		rateDate = rateDate.AddDate(0, 0, -1)
	}
	var authority = currency.RateAuthorityFederalReserve
	var providerID = currency.ProviderIDFederalReserveH10
	var rateKind = currency.RateKindFederalReserveH10NoonBuying
	var quoteDirection = currency.QuoteDirectionSourcePerBase
	var reference = "integration deterministic Federal Reserve H.10 fixture"
	if request.BaseCurrency == currency.BaseCurrencyEUR {
		authority = currency.RateAuthorityEuropeanCentralBank
		providerID = currency.ProviderIDECBEXR
		rateKind = currency.RateKindECBEXRDailyReference
		reference = "EXR/D." + request.SourceCurrency + ".EUR.SP00.A integration deterministic fixture"
	}
	if request.BaseCurrency == currency.BaseCurrencyUSD && request.SourceCurrency == currency.BaseCurrencyEUR {
		quoteDirection = currency.QuoteDirectionBasePerSource
	}
	return currency.NewExchangeRateEvidence(request, rateDate, authority, providerID, rateKind, quoteDirection, *apd.New(11, -1), reference)
}

// ProviderCategoryForBaseCurrency returns fixed provider metadata for a base
// currency. For example, a test can call it with currency.BaseCurrencyEUR to
// assert that the deterministic ECB provider category is selected.
// Authored by: OpenCode
func (DeterministicCurrencyRates) ProviderCategoryForBaseCurrency(baseCurrency string) string {
	if baseCurrency == currency.BaseCurrencyEUR {
		return string(currency.ProviderIDECBEXR)
	}
	if baseCurrency == currency.BaseCurrencyUSD {
		return string(currency.ProviderIDFederalReserveH10)
	}
	return ""
}

// SeedProtectedSnapshot writes a protected cache that the harness can unlock.
// For example, call SeedProtectedSnapshot(t, harness, token, cache) before
// driving the Sync and Reports workflow.
// Authored by: OpenCode
func SeedProtectedSnapshot(t *testing.T, harness RuntimeBackedFlowHarness, token string, cache syncmodel.ProtectedActivityCache) snapshotstore.Candidate {
	t.Helper()
	var syncedAt = cache.SyncedAt
	if syncedAt.IsZero() {
		syncedAt = time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC)
	}
	var candidate, err = harness.App.SnapshotStore.Write(context.Background(), snapshotstore.WriteRequest{SecurityToken: token, ServerOrigin: harness.Config.ServerOrigin, Payload: snapshotmodel.Payload{StoredDataVersion: snapshotmodel.DefaultStoredDataVersion(""), RegisteredLocalUser: snapshotmodel.RegisteredLocalUser{LocalUserID: "integration-user", CreatedAt: syncedAt, UpdatedAt: syncedAt, LastSuccessfulSyncAt: syncedAt}, SetupProfile: snapshotmodel.SetupProfile{ServerOrigin: harness.Config.ServerOrigin, ServerMode: harness.Config.ServerMode, AllowDevHTTP: harness.Config.AllowDevHTTP, LastValidatedAt: harness.Config.UpdatedAt, SourceAPIBasePath: "api/v1"}, ProtectedActivityCache: cache}})
	if err != nil {
		t.Fatalf("seed protected snapshot: %v", err)
	}
	return candidate
}

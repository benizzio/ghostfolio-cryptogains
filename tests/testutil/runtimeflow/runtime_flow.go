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
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
	"github.com/benizzio/ghostfolio-cryptogains/internal/integration/currency"
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
	Store   configstore.Store
	Model   *flow.Model
}

// NewRuntimeBackedFlowHarness creates an application-backed TUI harness with
// deterministic currency-rate evidence. For example, pass a temporary base
// directory and MustCloudSetupConfig(t) before seeding a protected snapshot.
// Authored by: OpenCode
func NewRuntimeBackedFlowHarness(t *testing.T, baseDir string, config configmodel.AppSetupConfig, allowDevHTTP bool) RuntimeBackedFlowHarness {
	t.Helper()

	var options = bootstrap.DefaultOptions()
	options.ConfigDir = baseDir
	options.AllowDevHTTP = allowDevHTTP
	var app = runtimeapp.NewWithReportCurrencyRateService(t, options, deterministicCurrencyRates{})
	var store = configstore.NewJSONStore(baseDir)
	var err = store.Save(context.Background(), config)
	if err != nil {
		t.Fatalf("save setup config: %v", err)
	}
	var model = flow.NewModel(flow.Dependencies{Options: options, Startup: bootstrap.StartupState{ActiveConfig: &config}, SetupService: app.SetupService, SyncService: app.SyncService, ReportService: app.ReportService})
	return RuntimeBackedFlowHarness{BaseDir: baseDir, App: app, Config: config, Store: store, Model: model}
}

// deterministicCurrencyRates supplies fixed currency-rate evidence to
// runtime-backed test fixtures.
// Authored by: OpenCode
type deterministicCurrencyRates struct{}

// LookupRate returns fixed currency-rate evidence for a requested activity.
// Authored by: OpenCode
func (deterministicCurrencyRates) LookupRate(_ context.Context, request currency.RateLookupRequest) (currency.ExchangeRateEvidence, error) {
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

// ProviderCategoryForBaseCurrency returns fixed provider metadata for a base currency.
// Authored by: OpenCode
func (deterministicCurrencyRates) ProviderCategoryForBaseCurrency(baseCurrency string) string {
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

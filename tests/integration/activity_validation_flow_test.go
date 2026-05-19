package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	syncvalidate "github.com/benizzio/ghostfolio-cryptogains/internal/sync/validate"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
)

func TestActivityValidationFlowRejectsUnsupportedHistoryAndKeepsExistingSnapshot(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	server := newTokenAwareStorageServer(t)
	server.SetTokenPages("token-one", []storagePageFixture{{
		Count:          1,
		ActivitiesJSON: `[{"id":"buy-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"baseCurrency":"USD","SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"USD"}}]`,
	}})
	service := newActivityValidationSyncService(baseDir, server)
	config := mustActivityValidationConfig(t, server.URL())
	inspector := snapshotstore.NewEncryptedStore(baseDir, nil)

	if outcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "token-one"}); !outcome.Success {
		t.Fatalf("expected baseline sync success, got %#v", outcome)
	}
	candidates, err := snapshotstore.DiscoverServerCandidates(context.Background(), inspector, server.URL())
	if err != nil {
		t.Fatalf("discover candidates: %v", err)
	}
	if len(candidates) == 0 {
		t.Fatalf("expected discovered snapshot candidates")
	}
	beforePayload, err := inspector.Read(context.Background(), snapshotstore.ReadRequest{Candidate: candidates[0], SecurityToken: "token-one"})
	if err != nil {
		t.Fatalf("read baseline snapshot: %v", err)
	}

	failureCases := []struct {
		name           string
		activitiesJSON string
	}{
		{
			name:           "unsupported activity type",
			activitiesJSON: `[{"id":"unsupported-1","date":"2024-01-02T10:00:00Z","type":"TRANSFER","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"baseCurrency":"USD","SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"USD"}}]`,
		},
	}

	for _, testCase := range failureCases {
		t.Run(testCase.name, func(t *testing.T) {
			server.SetTokenPages("token-one", []storagePageFixture{{
				Count:          1,
				ActivitiesJSON: testCase.activitiesJSON,
			}})
			outcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "token-one"})
			if outcome.FailureReason != runtime.SyncFailureUnsupportedActivityHistory {
				t.Fatalf("expected unsupported activity history outcome, got %#v", outcome)
			}
			afterPayload, err := inspector.Read(context.Background(), snapshotstore.ReadRequest{Candidate: candidates[0], SecurityToken: "token-one"})
			if err != nil {
				t.Fatalf("read retained snapshot: %v", err)
			}
			if afterPayload.ProtectedActivityCache.Activities[0].SourceID != beforePayload.ProtectedActivityCache.Activities[0].SourceID {
				t.Fatalf("expected previous readable snapshot to stay unchanged")
			}
		})
	}
}

func TestActivityValidationFlowAllowsProdLikeOrderTierPrecisionDifferences(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	server := newTokenAwareStorageServer(t)
	server.SetTokenPages("token-one", []storagePageFixture{{
		Count: 1,
		ActivitiesJSON: `[
			{"id":"prod-like-buy","date":"2025-06-22T09:41:33.202Z","type":"BUY","quantity":238.70829827,"currency":"USD","unitPrice":1.254775813,"value":299.5253990315857,"fee":0,"feeInAssetProfileCurrency":0,"valueInBaseCurrency":260.52719207767325,"feeInBaseCurrency":0,"unitPriceInAssetProfileCurrency":1.254775813,"baseCurrency":"EUR","comment":"Blockchain migration swapped from MATIC","SymbolProfile":{"symbol":"POL28321-USD.CC","name":"POL (ex-MATIC)","currency":"USD"},"account":{"id":"24f2c5ed-c7c8-4802-aa25-f18395640308","name":"Cryptofolio"}}
		]`,
	}})
	service := newActivityValidationSyncService(baseDir, server)
	config := mustActivityValidationConfig(t, server.URL())

	outcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "token-one"})
	if !outcome.Success {
		t.Fatalf("expected prod-like precision mismatch success, got %#v", outcome)
	}
}

// TestActivityValidationFlowNormalizesDuplicatesAndSameAssetSameDayOrdering
// verifies duplicate removal and persisted same-day ordering through the full sync flow.
// Authored by: OpenCode
func TestActivityValidationFlowNormalizesDuplicatesAndSameAssetSameDayOrdering(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	server := newTokenAwareStorageServer(t)
	server.SetTokenPages("token-one", []storagePageFixture{{
		Count: 4,
		ActivitiesJSON: `[
			{"id":"sell-z","date":"2024-01-01T00:00:00Z","type":"SELL","quantity":1,"valueInBaseCurrency":120,"unitPriceInAssetProfileCurrency":120,"baseCurrency":"USD","SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"USD"}},
			{"id":"buy-a","date":"2024-01-01T23:59:59Z","type":"BUY","quantity":2,"valueInBaseCurrency":200,"unitPriceInAssetProfileCurrency":100,"baseCurrency":"USD","SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"USD"}},
			{"id":"buy-a","date":"2024-01-01T23:59:59Z","type":"BUY","quantity":2,"valueInBaseCurrency":200,"unitPriceInAssetProfileCurrency":100,"baseCurrency":"USD","SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"USD"}},
			{"id":"buy-b","date":"2024-01-01T12:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":110,"unitPriceInAssetProfileCurrency":110,"baseCurrency":"USD","SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"USD"}}
		]`,
	}})
	service := newActivityValidationSyncService(baseDir, server)
	config := mustActivityValidationConfig(t, server.URL())
	inspector := snapshotstore.NewEncryptedStore(baseDir, nil)

	outcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "token-one"})
	if !outcome.Success {
		t.Fatalf("expected normalized sync success, got %#v", outcome)
	}
	candidates, err := snapshotstore.DiscoverServerCandidates(context.Background(), inspector, server.URL())
	if err != nil {
		t.Fatalf("discover candidates: %v", err)
	}
	if len(candidates) == 0 {
		t.Fatalf("expected discovered snapshot candidates")
	}
	payload, err := inspector.Read(context.Background(), snapshotstore.ReadRequest{Candidate: candidates[0], SecurityToken: "token-one"})
	if err != nil {
		t.Fatalf("read payload: %v", err)
	}
	if payload.ProtectedActivityCache.ActivityCount != 3 {
		t.Fatalf("expected duplicate removal, got %d activities", payload.ProtectedActivityCache.ActivityCount)
	}
	if payload.ProtectedActivityCache.Activities[0].SourceID != "buy-a" || payload.ProtectedActivityCache.Activities[1].SourceID != "buy-b" || payload.ProtectedActivityCache.Activities[2].SourceID != "sell-z" {
		t.Fatalf("expected deterministic same-day ordering, got %#v", payload.ProtectedActivityCache.Activities)
	}
	if payload.ProtectedActivityCache.Activities[0].RawHash == "" {
		t.Fatalf("expected normalized activities to persist duplicate hashes")
	}
	if payload.ProtectedActivityCache.Activities[0].OccurredAt != "2024-01-01T23:59:59Z" || payload.ProtectedActivityCache.Activities[2].OccurredAt != "2024-01-01T00:00:00Z" {
		t.Fatalf("expected preserved original timestamps, got %#v", payload.ProtectedActivityCache.Activities)
	}
}

func TestActivityValidationFlowRejectsBelowZeroHoldings(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	server := newTokenAwareStorageServer(t)
	server.SetTokenPages("token-one", []storagePageFixture{{
		Count: 2,
		ActivitiesJSON: `[
			{"id":"buy-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"baseCurrency":"USD","SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"USD"}},
			{"id":"sell-1","date":"2024-01-02T10:00:00Z","type":"SELL","quantity":2,"valueInBaseCurrency":200,"unitPriceInAssetProfileCurrency":100,"baseCurrency":"USD","SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"USD"}}
		]`,
	}})
	service := newActivityValidationSyncService(baseDir, server)
	config := mustActivityValidationConfig(t, server.URL())

	outcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "token-one"})
	if outcome.FailureReason != runtime.SyncFailureUnsupportedActivityHistory {
		t.Fatalf("expected below-zero holdings rejection, got %#v", outcome)
	}
}

// TestActivityValidationFlowUsesSameDayReplayOrderingForArbitraryGhostfolioTimes
// verifies that same-day holdings replay ignores arbitrary Ghostfolio clock values.
// Authored by: OpenCode
func TestActivityValidationFlowUsesSameDayReplayOrderingForArbitraryGhostfolioTimes(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	server := newTokenAwareStorageServer(t)
	server.SetTokenPages("token-one", []storagePageFixture{{
		Count: 2,
		ActivitiesJSON: `[
			{"id":"sell-early-clock","date":"2024-01-01T00:00:00Z","type":"SELL","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"baseCurrency":"USD","SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"USD"}},
			{"id":"buy-late-clock","date":"2024-01-01T23:59:59Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"baseCurrency":"USD","SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"USD"}}
		]`,
	}})
	service := newActivityValidationSyncService(baseDir, server)
	config := mustActivityValidationConfig(t, server.URL())
	inspector := snapshotstore.NewEncryptedStore(baseDir, nil)

	outcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "token-one"})
	if !outcome.Success {
		t.Fatalf("expected same-day replay ordering success, got %#v", outcome)
	}
	candidates, err := snapshotstore.DiscoverServerCandidates(context.Background(), inspector, server.URL())
	if err != nil {
		t.Fatalf("discover candidates: %v", err)
	}
	if len(candidates) == 0 {
		t.Fatalf("expected discovered snapshot candidates")
	}
	payload, err := inspector.Read(context.Background(), snapshotstore.ReadRequest{Candidate: candidates[0], SecurityToken: "token-one"})
	if err != nil {
		t.Fatalf("read payload: %v", err)
	}
	if payload.ProtectedActivityCache.Activities[0].SourceID != "buy-late-clock" || payload.ProtectedActivityCache.Activities[1].SourceID != "sell-early-clock" {
		t.Fatalf("expected same-day replay ordering to ignore arbitrary Ghostfolio times, got %#v", payload.ProtectedActivityCache.Activities)
	}
	if payload.ProtectedActivityCache.Activities[0].OccurredAt != "2024-01-01T23:59:59Z" || payload.ProtectedActivityCache.Activities[1].OccurredAt != "2024-01-01T00:00:00Z" {
		t.Fatalf("expected stored timestamps to remain unchanged, got %#v", payload.ProtectedActivityCache.Activities)
	}
}

func TestActivityValidationFlowAppliesZeroPriceRules(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		activities   string
		wantSuccess  bool
		wantCategory runtime.SyncFailureReason
	}{
		{
			name:         "reject buy with zero price",
			activities:   `[{"id":"buy-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":0,"unitPriceInAssetProfileCurrency":0,"baseCurrency":"USD","SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"USD"}}]`,
			wantSuccess:  false,
			wantCategory: runtime.SyncFailureUnsupportedActivityHistory,
		},
		{
			name: "reject sell with zero price and no comment",
			activities: `[
				{"id":"buy-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"baseCurrency":"USD","SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"USD"}},
				{"id":"sell-1","date":"2024-01-02T10:00:00Z","type":"SELL","quantity":1,"valueInBaseCurrency":0,"unitPriceInAssetProfileCurrency":0,"baseCurrency":"USD","SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"USD"}}
			]`,
			wantSuccess:  false,
			wantCategory: runtime.SyncFailureUnsupportedActivityHistory,
		},
		{
			name: "accept sell with zero price and comment",
			activities: `[
				{"id":"buy-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"baseCurrency":"USD","SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"USD"}},
				{"id":"sell-1","date":"2024-01-02T10:00:00Z","type":"SELL","quantity":1,"valueInBaseCurrency":0,"unitPriceInAssetProfileCurrency":0,"baseCurrency":"USD","comment":"manual reduction","SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"USD"}}
			]`,
			wantSuccess: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			baseDir := t.TempDir()
			server := newTokenAwareStorageServer(t)
			server.SetTokenPages("token-one", []storagePageFixture{{Count: strings.Count(testCase.activities, "\"id\""), ActivitiesJSON: testCase.activities}})
			service := newActivityValidationSyncService(baseDir, server)
			config := mustActivityValidationConfig(t, server.URL())

			outcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "token-one"})
			if testCase.wantSuccess {
				if !outcome.Success {
					t.Fatalf("expected success, got %#v", outcome)
				}
				return
			}
			if outcome.FailureReason != testCase.wantCategory {
				t.Fatalf("unexpected failure category: got %q want %q", outcome.FailureReason, testCase.wantCategory)
			}
		})
	}
}

func TestActivityValidationFlowPreservesMixedCurrencyContext(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	server := newTokenAwareStorageServer(t)
	server.SetTokenPages("token-one", []storagePageFixture{{
		Count: 1,
		ActivitiesJSON: `[
			{"id":"buy-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"currency":"CHF","unitPrice":90,"value":90,"fee":2,"feeInAssetProfileCurrency":1.8,"valueInBaseCurrency":100,"feeInBaseCurrency":2.2,"unitPriceInAssetProfileCurrency":95,"baseCurrency":"USD","SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"EUR"}}
		]`,
	}})
	service := newActivityValidationSyncService(baseDir, server)
	config := mustActivityValidationConfig(t, server.URL())
	inspector := snapshotstore.NewEncryptedStore(baseDir, nil)

	outcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "token-one"})
	if !outcome.Success {
		t.Fatalf("expected mixed-currency sync success, got %#v", outcome)
	}
	candidates, err := snapshotstore.DiscoverServerCandidates(context.Background(), inspector, server.URL())
	if err != nil {
		t.Fatalf("discover candidates: %v", err)
	}
	if len(candidates) == 0 {
		t.Fatalf("expected discovered snapshot candidates")
	}
	payload, err := inspector.Read(context.Background(), snapshotstore.ReadRequest{Candidate: candidates[0], SecurityToken: "token-one"})
	if err != nil {
		t.Fatalf("read payload: %v", err)
	}
	record := payload.ProtectedActivityCache.Activities[0]
	if record.OrderCurrency != "CHF" || record.AssetProfileCurrency != "EUR" || record.BaseCurrency != "USD" {
		t.Fatalf("expected preserved currency context, got %#v", record)
	}
	if record.OrderUnitPrice == nil || record.OrderGrossValue == nil || record.OrderFeeAmount == nil || record.AssetProfileUnitPrice == nil || record.AssetProfileFeeAmount == nil || record.BaseGrossValue == nil || record.BaseFeeAmount == nil {
		t.Fatalf("expected preserved source monetary groups, got %#v", record)
	}
}

func TestActivityValidationFlowAllowsSingleUninformedCurrencyTierWhenOtherTiersRemainInformed(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	server := newTokenAwareStorageServer(t)
	setTokenAwareCurrencyContextFixtures(
		server,
		"token-order-null",
		testutil.GhostfolioUserBody("USD"),
		testutil.GhostfolioNullableOrderCurrencyActivityJSON(),
	)
	setTokenAwareCurrencyContextFixtures(
		server,
		"token-user-base-missing",
		testutil.GhostfolioUserBody(""),
		testutil.GhostfolioMissingSymbolProfileCurrencyActivityJSON(),
	)
	service := newActivityValidationSyncService(baseDir, server)
	config := mustActivityValidationConfig(t, server.URL())

	for _, securityToken := range []string{"token-order-null", "token-user-base-missing"} {
		t.Run(securityToken, func(t *testing.T) {
			outcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: securityToken})
			if !outcome.Success {
				t.Fatalf("expected valid mixed-tier sync success, got %#v", outcome)
			}
		})
	}
}

func TestActivityValidationFlowRejectsIncompleteCurrencyContext(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	server := newTokenAwareStorageServer(t)
	setTokenAwareCurrencyContextFixtures(
		server,
		"token-one",
		testutil.GhostfolioUserBody(""),
		testutil.GhostfolioAllTierUninformedCurrencyActivityJSON(),
	)
	service := newActivityValidationSyncService(baseDir, server)
	config := mustActivityValidationConfig(t, server.URL())

	outcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "token-one"})
	if outcome.FailureReason != runtime.SyncFailureUnsupportedActivityHistory {
		t.Fatalf("expected incomplete currency context rejection, got %#v", outcome)
	}
	if !outcome.Diagnostic.Eligible {
		t.Fatalf("expected diagnostic eligibility for currency-context rejection, got %#v", outcome)
	}
	if !strings.Contains(outcome.Diagnostic.Request.Context.FailureDetail, "uninformed across order, asset-profile, and base tiers") {
		t.Fatalf("expected all-tier-uninformed diagnostic detail, got %#v", outcome.Diagnostic.Request.Context)
	}
}

func newActivityValidationSyncService(baseDir string, server *tokenAwareStorageServer) runtime.SyncService {
	return runtime.NewSyncService(
		ghostfolioclient.New(server.Client()),
		time.Second,
		baseDir,
		true,
		decimalsupport.NewService(),
		syncnormalize.NewNormalizer(),
		syncvalidate.NewValidator(),
		snapshotstore.NewEncryptedStore(baseDir, nil),
	)
}

func mustActivityValidationConfig(t *testing.T, origin string) configmodel.AppSetupConfig {
	t.Helper()

	config, err := configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, origin, true, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}
	return config
}

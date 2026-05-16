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
)

func TestActivityValidationFlowRejectsUnsupportedHistoryAndKeepsExistingSnapshot(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	server := newTokenAwareStorageServer(t)
	server.SetTokenPages("token-one", []storagePageFixture{{
		Count:          1,
		ActivitiesJSON: `[{"id":"buy-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}]`,
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
	beforePayload, err := inspector.Read(context.Background(), snapshotstore.ReadRequest{Candidate: candidates[0], SecurityToken: "token-one"})
	if err != nil {
		t.Fatalf("read baseline snapshot: %v", err)
	}

	server.SetTokenPages("token-one", []storagePageFixture{{
		Count:          1,
		ActivitiesJSON: `[{"id":"unsupported-1","date":"2024-01-02T10:00:00Z","type":"TRANSFER","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}]`,
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
			{"id":"sell-z","date":"2024-01-01T00:00:00Z","type":"SELL","quantity":1,"valueInBaseCurrency":120,"unitPriceInAssetProfileCurrency":120,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}},
			{"id":"buy-a","date":"2024-01-01T23:59:59Z","type":"BUY","quantity":2,"valueInBaseCurrency":200,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}},
			{"id":"buy-a","date":"2024-01-01T23:59:59Z","type":"BUY","quantity":2,"valueInBaseCurrency":200,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}},
			{"id":"buy-b","date":"2024-01-01T12:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":110,"unitPriceInAssetProfileCurrency":110,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}
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
			{"id":"buy-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}},
			{"id":"sell-1","date":"2024-01-02T10:00:00Z","type":"SELL","quantity":2,"valueInBaseCurrency":200,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}
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
			{"id":"sell-early-clock","date":"2024-01-01T00:00:00Z","type":"SELL","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}},
			{"id":"buy-late-clock","date":"2024-01-01T23:59:59Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}
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
			activities:   `[{"id":"buy-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":0,"unitPriceInAssetProfileCurrency":0,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}]`,
			wantSuccess:  false,
			wantCategory: runtime.SyncFailureUnsupportedActivityHistory,
		},
		{
			name: "reject sell with zero price and no comment",
			activities: `[
				{"id":"buy-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}},
				{"id":"sell-1","date":"2024-01-02T10:00:00Z","type":"SELL","quantity":1,"valueInBaseCurrency":0,"unitPriceInAssetProfileCurrency":0,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}
			]`,
			wantSuccess:  false,
			wantCategory: runtime.SyncFailureUnsupportedActivityHistory,
		},
		{
			name: "accept sell with zero price and comment",
			activities: `[
				{"id":"buy-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}},
				{"id":"sell-1","date":"2024-01-02T10:00:00Z","type":"SELL","quantity":1,"valueInBaseCurrency":0,"unitPriceInAssetProfileCurrency":0,"comment":"manual reduction","SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}
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

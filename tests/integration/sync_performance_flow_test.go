// Package integration verifies black-box workflow behavior for the current
// slice, including the documented large-history performance verification path.
// Authored by: OpenCode
package integration

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	syncvalidate "github.com/benizzio/ghostfolio-cryptogains/internal/sync/validate"
)

const performanceVerificationEnvironmentVariable = "GHOSTFOLIO_CRYPTOGAINS_RUN_PERFORMANCE"

// TestSyncPerformanceFlowLargeHistoryFixture verifies the documented SC-006
// path with a deterministic 10,000-activity snapshot refresh when explicitly
// enabled.
// Authored by: OpenCode
func TestSyncPerformanceFlowLargeHistoryFixture(t *testing.T) {
	if os.Getenv(performanceVerificationEnvironmentVariable) != "1" {
		t.Skipf("set %s=1 to run the SC-006 performance verification path", performanceVerificationEnvironmentVariable)
	}

	const activityCount = 10000
	const minimumYearSpan = 5
	const threshold = 2 * time.Minute

	var baseDir = t.TempDir()
	var server = newTokenAwareStorageServer(t)
	var token = "performance_token"
	var service = newPerformanceSyncService(baseDir, server)
	var config = mustSnapshotReuseConfig(t, server.URL())

	server.SetTokenPages(token, []storagePageFixture{{
		Count:          1,
		ActivitiesJSON: `[{"id":"baseline-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}]`,
	}})
	var baselineOutcome = service.Validate(context.Background(), runtime.ValidateRequest{Config: config, SecurityToken: token})
	if !baselineOutcome.Success {
		t.Fatalf("expected baseline sync success before refresh timing, got %#v", baselineOutcome)
	}

	var inspector = snapshotstore.NewEncryptedStore(baseDir, nil)
	var beforeCandidates, err = snapshotstore.DiscoverServerCandidates(context.Background(), inspector, server.URL())
	if err != nil {
		t.Fatalf("discover baseline candidates: %v", err)
	}
	if len(beforeCandidates) != 1 {
		t.Fatalf("expected one baseline snapshot candidate, got %d", len(beforeCandidates))
	}

	server.SetTokenPages(token, buildLargeHistoryPages(activityCount, 250))
	var startedAt = time.Now()
	var outcome = service.Validate(context.Background(), runtime.ValidateRequest{Config: config, SecurityToken: token})
	var elapsed = time.Since(startedAt)
	if !outcome.Success {
		t.Fatalf("expected large-history refresh success, got %#v", outcome)
	}
	if elapsed >= threshold {
		t.Fatalf("expected SC-006 verification under %s, got %s", threshold, elapsed)
	}

	var afterCandidates, discoverErr = snapshotstore.DiscoverServerCandidates(context.Background(), inspector, server.URL())
	if discoverErr != nil {
		t.Fatalf("discover refreshed snapshot candidates: %v", discoverErr)
	}
	if len(afterCandidates) != 1 {
		t.Fatalf("expected one refreshed snapshot candidate, got %d", len(afterCandidates))
	}
	if afterCandidates[0].SnapshotID != beforeCandidates[0].SnapshotID {
		t.Fatalf("expected protected refresh to replace the existing snapshot, got %q then %q", beforeCandidates[0].SnapshotID, afterCandidates[0].SnapshotID)
	}

	var payload, readErr = inspector.Read(context.Background(), snapshotstore.ReadRequest{
		Candidate:     afterCandidates[0],
		SecurityToken: token,
	})
	if readErr != nil {
		t.Fatalf("read refreshed snapshot payload: %v", readErr)
	}
	if payload.ProtectedActivityCache.ActivityCount != activityCount {
		t.Fatalf("expected %d stored activities, got %d", activityCount, payload.ProtectedActivityCache.ActivityCount)
	}
	if len(payload.ProtectedActivityCache.AvailableReportYears) < minimumYearSpan {
		t.Fatalf("expected at least %d available report years, got %d", minimumYearSpan, len(payload.ProtectedActivityCache.AvailableReportYears))
	}

	t.Logf(
		"SC-006 verification completed in %s for %d activities across %d available report years",
		elapsed,
		activityCount,
		len(payload.ProtectedActivityCache.AvailableReportYears),
	)
}

// newPerformanceSyncService creates one sync service with a timeout large
// enough for the explicit SC-006 verification path.
// Authored by: OpenCode
func newPerformanceSyncService(baseDir string, server *tokenAwareStorageServer) runtime.SyncService {
	return runtime.NewSyncService(
		ghostfolioclient.New(server.Client()),
		3*time.Minute,
		baseDir,
		true,
		decimalsupport.NewService(),
		syncnormalize.NewNormalizer(),
		syncvalidate.NewValidator(),
		snapshotstore.NewEncryptedStore(baseDir, nil),
	)
}

// buildLargeHistoryPages constructs the deterministic 10,000-activity fixture
// used by the explicit SC-006 performance verification path.
// Authored by: OpenCode
func buildLargeHistoryPages(activityCount int, pageSize int) []storagePageFixture {
	var pages = make([]storagePageFixture, 0, (activityCount+pageSize-1)/pageSize)
	var startedAt = time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)

	for offset := 0; offset < activityCount; offset += pageSize {
		var end = offset + pageSize
		if end > activityCount {
			end = activityCount
		}

		var builder strings.Builder
		builder.WriteByte('[')
		for index := offset; index < end; index++ {
			if index > offset {
				builder.WriteByte(',')
			}

			var occurredAt = startedAt.Add(time.Duration(index) * 5 * time.Hour).Format(time.RFC3339)
			builder.WriteString(fmt.Sprintf(
				`{"id":"activity-%05d","date":"%s","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}`,
				index+1,
				occurredAt,
			))
		}
		builder.WriteByte(']')

		pages = append(pages, storagePageFixture{
			Count:          activityCount,
			ActivitiesJSON: builder.String(),
		})
	}

	return pages
}

//go:build performance

// Authored by: OpenCode
package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
	"github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/dto"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	syncvalidate "github.com/benizzio/ghostfolio-cryptogains/internal/sync/validate"
)

// TestSyncPerformanceFlowLargeHistoryFixture verifies SC-006 with a
// deterministic 10,000-activity protected snapshot refresh.
// Authored by: OpenCode
func TestSyncPerformanceFlowLargeHistoryFixture(t *testing.T) {
	const activityCount = 10000
	const minimumYearSpan = 5
	const threshold = 2 * time.Minute
	var baseDir = t.TempDir()
	var pages = largeHistoryPages(activityCount, 250)
	var server = newPerformanceSyncServer(t, pages, activityCount)
	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, server.URL, true, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}
	var service = runtime.NewSyncService(ghostfolioclient.New(server.Client()), 3*time.Minute, baseDir, true, decimalsupport.NewService(), syncnormalize.NewNormalizer(), syncvalidate.NewValidator(), snapshotstore.NewEncryptedStore(baseDir, nil))
	var startedAt = time.Now()
	var outcome = service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "performance-token"})
	var elapsed = time.Since(startedAt)
	if !outcome.Success {
		t.Fatalf("expected large-history sync success, got %#v", outcome)
	}
	if elapsed >= threshold {
		t.Fatalf("expected SC-006 under %s, got %s", threshold, elapsed)
	}
	var inspector = snapshotstore.NewEncryptedStore(baseDir, nil)
	var candidates, discoverErr = snapshotstore.DiscoverServerCandidates(context.Background(), inspector, server.URL)
	if discoverErr != nil {
		t.Fatalf("discover snapshot: %v", discoverErr)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected one snapshot candidate, got %d", len(candidates))
	}
	var payload, readErr = inspector.Read(context.Background(), snapshotstore.ReadRequest{Candidate: candidates[0], SecurityToken: "performance-token"})
	if readErr != nil {
		t.Fatalf("read snapshot payload: %v", readErr)
	}
	if payload.ProtectedActivityCache.ActivityCount != activityCount {
		t.Fatalf("expected %d activities, got %d", activityCount, payload.ProtectedActivityCache.ActivityCount)
	}
	if len(payload.ProtectedActivityCache.AvailableReportYears) < minimumYearSpan {
		t.Fatalf("expected at least %d report years, got %d", minimumYearSpan, len(payload.ProtectedActivityCache.AvailableReportYears))
	}
	t.Logf("SC-006 verification completed in %s for %d activities across %d available report years", elapsed, activityCount, len(payload.ProtectedActivityCache.AvailableReportYears))
}

type performanceSyncServer struct{ *httptest.Server }

func newPerformanceSyncServer(t *testing.T, pages [][]dto.ActivityPageEntry, totalCount int) *performanceSyncServer {
	t.Helper()
	var pageIndex int
	var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/v1/auth/anonymous":
			var input struct {
				AccessToken string `json:"accessToken"`
			}
			if err := json.NewDecoder(request.Body).Decode(&input); err != nil || input.AccessToken != "performance-token" {
				writer.WriteHeader(http.StatusForbidden)
				return
			}
			_, _ = writer.Write([]byte(`{"authToken":"jwt-performance-token"}`))
		case "/api/v1/user":
			_, _ = writer.Write([]byte(`{"settings":{"baseCurrency":"USD"}}`))
		case "/api/v1/activities":
			if pageIndex >= len(pages) {
				pageIndex = len(pages) - 1
			}
			var response = dto.ActivityPageResponse{Activities: pages[pageIndex], Count: totalCount}
			var responseBody, err = json.Marshal(response)
			if err != nil {
				t.Errorf("marshal activities response: %v", err)
				http.Error(writer, "marshal activities response", http.StatusInternalServerError)
				return
			}
			_, _ = writer.Write(responseBody)
			pageIndex++
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return &performanceSyncServer{Server: server}
}

func largeHistoryPages(activityCount int, pageSize int) [][]dto.ActivityPageEntry {
	var pages = make([][]dto.ActivityPageEntry, 0, (activityCount+pageSize-1)/pageSize)
	var startedAt = time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	for offset := 0; offset < activityCount; offset += pageSize {
		var end = offset + pageSize
		if end > activityCount {
			end = activityCount
		}
		var page = make([]dto.ActivityPageEntry, 0, end-offset)
		for index := offset; index < end; index++ {
			page = append(page, dto.ActivityPageEntry{
				ID:                              fmt.Sprintf("activity-%05d", index+1),
				Date:                            startedAt.Add(time.Duration(index) * 5 * time.Hour).Format(time.RFC3339),
				Type:                            "BUY",
				Quantity:                        json.Number("1"),
				ValueInBaseCurrency:             json.Number("100"),
				UnitPriceInAssetProfileCurrency: json.Number("100"),
				SymbolProfile: dto.ActivitySymbolProfile{
					ID:       "asset-btc-performance-001",
					Symbol:   "BTC",
					Name:     "Bitcoin",
					Currency: "USD",
				},
			})
		}
		pages = append(pages, page)
	}
	return pages
}

package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	syncvalidate "github.com/benizzio/ghostfolio-cryptogains/internal/sync/validate"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
)

func TestSyncStorageFlowCreatesProtectedSnapshotAfterSuccessfulMultiPageSync(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	store := configstore.NewJSONStore(tempDir)
	server := newGhostfolioStorageServer(t, []storagePageFixture{
		{Count: 3, ActivitiesJSON: `[{"id":"activity-1","date":"2024-12-31T23:30:00-02:00","type":"BUY","quantity":1.25,"valueInBaseCurrency":62500,"feeInBaseCurrency":25,"unitPriceInAssetProfileCurrency":50000,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"},"account":{"id":"account-1","name":"Main"}}]`},
		{Count: 3, ActivitiesJSON: `[{"id":"activity-2","date":"2025-01-01T00:15:00+02:00","type":"BUY","quantity":0.50,"valueInBaseCurrency":25000,"unitPriceInAssetProfileCurrency":50000,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}]`},
		{Count: 3, ActivitiesJSON: `[{"id":"activity-3","date":"2026-05-01T09:00:00Z","type":"SELL","quantity":0.25,"valueInBaseCurrency":15000,"unitPriceInAssetProfileCurrency":60000,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}]`},
	})
	fixture := newSyncStorageFixture(t, tempDir, server.Client(), server.URL, time.Second)
	if err := store.Save(context.Background(), fixture.config); err != nil {
		t.Fatalf("save config: %v", err)
	}

	model := flow.NewModel(newFlowDependenciesWithStore(t, bootstrap.StartupState{ActiveConfig: &fixture.config}, true, fixture.service, store))
	model = openSyncValidation(t, model)
	model = typeToken(t, model, "abc123")
	model = blurTokenInput(t, model)
	model, cmd := startSyncValidationAttempt(t, model)
	model = applyValidationBatch(t, model, cmd)

	if model.ActiveScreen() != "validation_result" {
		t.Fatalf("expected validation result screen, got %s", model.ActiveScreen())
	}
	content := model.View().Content
	if !strings.Contains(content, "Activity data was stored securely for future use.") {
		t.Fatalf("expected storage success summary, got %q", content)
	}

	snapshots, err := os.ReadDir(filepath.Join(tempDir, "ghostfolio-cryptogains", snapshotstore.SnapshotDirectoryName))
	if err != nil {
		t.Fatalf("read snapshot directory: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("expected one protected snapshot, got %d", len(snapshots))
	}
}

func TestSyncStorageFlowHandlesEmptyHistoryAsSuccessfulStoredState(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	server := newGhostfolioStorageServer(t, []storagePageFixture{{Count: 0, ActivitiesJSON: `[]`}})
	fixture := newSyncStorageFixture(t, tempDir, server.Client(), server.URL, time.Second)
	model := flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &fixture.config}, true, fixture.service))

	model = openSyncValidation(t, model)
	model = typeToken(t, model, "abc123")
	model = blurTokenInput(t, model)
	model, cmd := startSyncValidationAttempt(t, model)
	model = applyValidationBatch(t, model, cmd)

	content := model.View().Content
	if !strings.Contains(content, "Activity data was stored securely for future use.") {
		t.Fatalf("expected empty-history sync to remain successful, got %q", content)
	}
}

type storagePageFixture struct {
	Count          int
	ActivitiesJSON string
}

type syncStorageFixture struct {
	config  configmodel.AppSetupConfig
	service runtime.SyncService
}

func newGhostfolioStorageServer(t *testing.T, pages []storagePageFixture) *httptest.Server {
	t.Helper()

	var requestCount int
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/v1/auth/anonymous":
			_, _ = writer.Write([]byte(`{"authToken":"jwt"}`))
		case "/api/v1/activities":
			if request.URL.Query().Get("sortColumn") != "date" || request.URL.Query().Get("sortDirection") != "asc" {
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			if requestCount >= len(pages) {
				requestCount = len(pages) - 1
			}
			page := pages[requestCount]
			requestCount++
			_, _ = writer.Write([]byte(fmt.Sprintf(`{"activities":%s,"count":%d}`, page.ActivitiesJSON, page.Count)))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func newSyncStorageFixture(t *testing.T, tempDir string, client *http.Client, origin string, requestTimeout time.Duration) syncStorageFixture {
	t.Helper()

	config, err := configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, origin, true, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	service := runtime.NewSyncService(
		ghostfolioclient.New(client),
		requestTimeout,
		decimalsupport.NewService(),
		syncnormalize.NewNormalizer(),
		syncvalidate.NewValidator(),
		snapshotstore.NewEncryptedStore(tempDir, nil),
	)

	return syncStorageFixture{config: config, service: service}
}

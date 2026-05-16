package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	syncvalidate "github.com/benizzio/ghostfolio-cryptogains/internal/sync/validate"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
)

func TestServerReplacementFlowCancelKeepsReadableSnapshotUnchanged(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	firstServer := newGhostfolioStorageServer(t, []storagePageFixture{{Count: 1, ActivitiesJSON: `[{"id":"activity-old","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}]`}})
	service := runtime.NewSyncService(ghostfolioclient.New(firstServer.Client()), time.Second, baseDir, true, decimalsupport.NewService(), syncnormalize.NewNormalizer(), syncvalidate.NewValidator(), snapshotstore.NewEncryptedStore(baseDir, nil))
	firstConfig := mustCustomSetupConfig(t, firstServer.URL)
	if outcome := service.Run(context.Background(), runtime.SyncRequest{Config: firstConfig, SecurityToken: "token-one"}); !outcome.Success {
		t.Fatalf("expected initial sync success, got %#v", outcome)
	}

	secondServer := newGhostfolioStorageServer(t, []storagePageFixture{{Count: 1, ActivitiesJSON: `[{"id":"activity-new","date":"2025-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"ETH","name":"Ether"}}]`}})
	secondConfig := mustCustomSetupConfig(t, secondServer.URL)
	model := flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &secondConfig}, true, service))
	model = openSyncValidation(t, model)
	model = typeToken(t, model, "token-one")
	model = blurTokenInput(t, model)

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "server_replacement" {
		t.Fatalf("expected server replacement confirmation, got %s", model.ActiveScreen())
	}

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "sync_result" {
		t.Fatalf("expected cancel to route to sync result, got %s", model.ActiveScreen())
	}
	content := model.View().Content
	if !strings.Contains(content, "server replacement cancelled") {
		t.Fatalf("expected cancellation outcome, got %q", content)
	}
}

func TestServerReplacementFlowConfirmSuccessReplacesSnapshot(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	firstServer := newGhostfolioStorageServer(t, []storagePageFixture{{Count: 1, ActivitiesJSON: `[{"id":"activity-old","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}]`}})
	sharedStore := snapshotstore.NewEncryptedStore(baseDir, nil)
	service := runtime.NewSyncService(ghostfolioclient.New(firstServer.Client()), time.Second, baseDir, true, decimalsupport.NewService(), syncnormalize.NewNormalizer(), syncvalidate.NewValidator(), sharedStore)
	if outcome := service.Run(context.Background(), runtime.SyncRequest{Config: mustCustomSetupConfig(t, firstServer.URL), SecurityToken: "token-one"}); !outcome.Success {
		t.Fatalf("expected initial sync success, got %#v", outcome)
	}

	secondServer := newGhostfolioStorageServer(t, []storagePageFixture{{Count: 1, ActivitiesJSON: `[{"id":"activity-new","date":"2025-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"ETH","name":"Ether"}}]`}})
	secondConfig := mustCustomSetupConfig(t, secondServer.URL)
	replacementService := runtime.NewSyncService(ghostfolioclient.New(secondServer.Client()), time.Second, baseDir, true, decimalsupport.NewService(), syncnormalize.NewNormalizer(), syncvalidate.NewValidator(), sharedStore)
	if outcome := replacementService.Run(context.Background(), runtime.SyncRequest{Config: mustCustomSetupConfig(t, firstServer.URL), SecurityToken: "token-one"}); !outcome.Success {
		t.Fatalf("expected preload success, got %#v", outcome)
	}
	model := flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &secondConfig}, true, replacementService))

	model = openSyncValidation(t, model)
	model = typeToken(t, model, "token-one")
	model = blurTokenInput(t, model)
	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "server_replacement" {
		t.Fatalf("expected server replacement confirmation, got %s", model.ActiveScreen())
	}
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	model = applyValidationBatch(t, model, cmd)
	if model.ActiveScreen() != "sync_result" {
		t.Fatalf("expected result screen after confirmed replacement, got %s", model.ActiveScreen())
	}

	inspector := snapshotstore.NewEncryptedStore(baseDir, nil)
	candidates, err := snapshotstore.DiscoverServerCandidates(context.Background(), inspector, secondServer.URL)
	if err != nil {
		t.Fatalf("discover replacement candidates: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected one replacement snapshot for new server, got %d", len(candidates))
	}
	payload, err := inspector.Read(context.Background(), snapshotstore.ReadRequest{Candidate: candidates[0], SecurityToken: "token-one"})
	if err != nil {
		t.Fatalf("read replacement payload: %v", err)
	}
	if payload.SetupProfile.ServerOrigin != secondServer.URL {
		t.Fatalf("expected replacement payload to track new server, got %q", payload.SetupProfile.ServerOrigin)
	}
	if payload.ProtectedActivityCache.Activities[0].SourceID != "activity-new" {
		t.Fatalf("expected replacement payload content, got %#v", payload.ProtectedActivityCache.Activities)
	}
}

func TestServerReplacementFlowFailedReplacementRetainsPreviousSnapshot(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	firstServer := newGhostfolioStorageServer(t, []storagePageFixture{{Count: 1, ActivitiesJSON: `[{"id":"activity-old","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}]`}})
	baseStore := snapshotstore.NewEncryptedStore(baseDir, nil)
	service := runtime.NewSyncService(ghostfolioclient.New(firstServer.Client()), time.Second, baseDir, true, decimalsupport.NewService(), syncnormalize.NewNormalizer(), syncvalidate.NewValidator(), baseStore)
	firstConfig := mustCustomSetupConfig(t, firstServer.URL)
	if outcome := service.Run(context.Background(), runtime.SyncRequest{Config: firstConfig, SecurityToken: "token-one"}); !outcome.Success {
		t.Fatalf("expected initial sync success, got %#v", outcome)
	}
	inspector := snapshotstore.NewEncryptedStore(baseDir, nil)
	firstCandidates, err := snapshotstore.DiscoverServerCandidates(context.Background(), inspector, firstServer.URL)
	if err != nil {
		t.Fatalf("discover initial candidates: %v", err)
	}
	beforeBytes, err := os.ReadFile(firstCandidates[0].Path)
	if err != nil {
		t.Fatalf("read original snapshot: %v", err)
	}

	secondServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/v1/auth/anonymous":
			_, _ = writer.Write([]byte(`{"authToken":"jwt"}`))
		case "/api/v1/activities":
			_, _ = writer.Write([]byte(`{"activities":[{"id":"bad-sell","date":"2025-01-01T10:00:00Z","type":"SELL","quantity":10,"valueInBaseCurrency":1000,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}],"count":1}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer secondServer.Close()
	replacementService := runtime.NewSyncService(ghostfolioclient.New(secondServer.Client()), time.Second, baseDir, true, decimalsupport.NewService(), syncnormalize.NewNormalizer(), syncvalidate.NewValidator(), baseStore)
	if preload := replacementService.Run(context.Background(), runtime.SyncRequest{Config: firstConfig, SecurityToken: "token-one"}); !preload.Success {
		t.Fatalf("expected preload success to set active snapshot, got %#v", preload)
	}
	beforeBytes, err = os.ReadFile(firstCandidates[0].Path)
	if err != nil {
		t.Fatalf("read snapshot after preload: %v", err)
	}

	secondConfig, err := configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, secondServer.URL, true, time.Now())
	if err != nil {
		t.Fatalf("new second config: %v", err)
	}
	outcome := replacementService.Run(context.Background(), runtime.SyncRequest{Config: secondConfig, SecurityToken: "token-one", ConfirmServerReplacement: true})
	if outcome.FailureReason != runtime.SyncFailureUnsupportedActivityHistory {
		t.Fatalf("expected failed replacement to reject invalid new history, got %#v", outcome)
	}
	afterBytes, err := os.ReadFile(firstCandidates[0].Path)
	if err != nil {
		t.Fatalf("read retained snapshot: %v", err)
	}
	if string(beforeBytes) != string(afterBytes) {
		t.Fatalf("expected failed replacement to keep previous snapshot bytes unchanged")
	}
}

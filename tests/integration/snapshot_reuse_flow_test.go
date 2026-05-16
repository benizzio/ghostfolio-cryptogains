package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
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

func TestSnapshotReuseFlowRefreshesExistingSnapshotWithSameToken(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	server := newTokenAwareStorageServer(t)
	server.SetTokenPages("token-one", []storagePageFixture{{
		Count:          1,
		ActivitiesJSON: `[{"id":"activity-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}]`,
	}})
	service := newTokenAwareSyncService(baseDir, server)
	config := mustSnapshotReuseConfig(t, server.URL())
	inspector := snapshotstore.NewEncryptedStore(baseDir, nil)

	firstOutcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "token-one"})
	if !firstOutcome.Success {
		t.Fatalf("expected first sync success, got %#v", firstOutcome)
	}
	firstCandidates, err := snapshotstore.DiscoverServerCandidates(context.Background(), inspector, server.URL())
	if err != nil {
		t.Fatalf("discover first candidates: %v", err)
	}
	if len(firstCandidates) != 1 {
		t.Fatalf("expected one snapshot after first sync, got %d", len(firstCandidates))
	}
	firstPayload, err := inspector.Read(context.Background(), snapshotstore.ReadRequest{Candidate: firstCandidates[0], SecurityToken: "token-one"})
	if err != nil {
		t.Fatalf("read first payload: %v", err)
	}

	server.SetTokenPages("token-one", []storagePageFixture{{
		Count:          1,
		ActivitiesJSON: `[{"id":"activity-2","date":"2025-01-01T10:00:00Z","type":"BUY","quantity":2,"valueInBaseCurrency":200,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}]`,
	}})
	secondOutcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "token-one"})
	if !secondOutcome.Success {
		t.Fatalf("expected refresh success, got %#v", secondOutcome)
	}
	secondCandidates, err := snapshotstore.DiscoverServerCandidates(context.Background(), inspector, server.URL())
	if err != nil {
		t.Fatalf("discover second candidates: %v", err)
	}
	if len(secondCandidates) != 1 {
		t.Fatalf("expected one snapshot after refresh, got %d", len(secondCandidates))
	}
	if secondCandidates[0].SnapshotID != firstCandidates[0].SnapshotID {
		t.Fatalf("expected same snapshot to be refreshed, got %q then %q", firstCandidates[0].SnapshotID, secondCandidates[0].SnapshotID)
	}
	secondPayload, err := inspector.Read(context.Background(), snapshotstore.ReadRequest{Candidate: secondCandidates[0], SecurityToken: "token-one"})
	if err != nil {
		t.Fatalf("read refreshed payload: %v", err)
	}
	if secondPayload.RegisteredLocalUser.LocalUserID != firstPayload.RegisteredLocalUser.LocalUserID {
		t.Fatalf("expected local user to be reused across refresh")
	}
	if secondPayload.ProtectedActivityCache.Activities[0].SourceID != "activity-2" {
		t.Fatalf("expected refreshed activity payload, got %#v", secondPayload.ProtectedActivityCache.Activities)
	}
}

func TestSnapshotReuseFlowCreatesIsolatedSnapshotForDifferentValidToken(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	server := newTokenAwareStorageServer(t)
	server.SetTokenPages("token-one", []storagePageFixture{{
		Count:          1,
		ActivitiesJSON: `[{"id":"activity-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}]`,
	}})
	server.SetTokenPages("token-two", []storagePageFixture{{
		Count:          1,
		ActivitiesJSON: `[{"id":"activity-2","date":"2025-01-01T10:00:00Z","type":"BUY","quantity":2,"valueInBaseCurrency":200,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"ETH","name":"Ether"}}]`,
	}})
	service := newTokenAwareSyncService(baseDir, server)
	config := mustSnapshotReuseConfig(t, server.URL())
	inspector := snapshotstore.NewEncryptedStore(baseDir, nil)

	if outcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "token-one"}); !outcome.Success {
		t.Fatalf("expected first sync success, got %#v", outcome)
	}
	if outcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "token-two"}); !outcome.Success {
		t.Fatalf("expected second-token sync success, got %#v", outcome)
	}

	candidates, err := snapshotstore.DiscoverServerCandidates(context.Background(), inspector, server.URL())
	if err != nil {
		t.Fatalf("discover candidates: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected two isolated snapshots, got %d", len(candidates))
	}

	var tokenOneUnlocks int
	var tokenTwoUnlocks int
	for _, candidate := range candidates {
		payload, readErr := inspector.Read(context.Background(), snapshotstore.ReadRequest{Candidate: candidate, SecurityToken: "token-one"})
		if readErr == nil {
			tokenOneUnlocks++
			if payload.ProtectedActivityCache.Activities[0].SourceID != "activity-1" {
				t.Fatalf("expected token-one payload to remain isolated, got %#v", payload.ProtectedActivityCache.Activities)
			}
		}
		payload, readErr = inspector.Read(context.Background(), snapshotstore.ReadRequest{Candidate: candidate, SecurityToken: "token-two"})
		if readErr == nil {
			tokenTwoUnlocks++
			if payload.ProtectedActivityCache.Activities[0].SourceID != "activity-2" {
				t.Fatalf("expected token-two payload to remain isolated, got %#v", payload.ProtectedActivityCache.Activities)
			}
		}
	}
	if tokenOneUnlocks != 1 || tokenTwoUnlocks != 1 {
		t.Fatalf("expected one unlockable snapshot per token, got token-one=%d token-two=%d", tokenOneUnlocks, tokenTwoUnlocks)
	}
}

func TestSnapshotReuseFlowDeniesWrongTokenWithoutChangingExistingSnapshot(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	server := newTokenAwareStorageServer(t)
	server.SetTokenPages("token-one", []storagePageFixture{{
		Count:          1,
		ActivitiesJSON: `[{"id":"activity-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}]`,
	}})
	server.RejectToken("wrong-token")
	service := newTokenAwareSyncService(baseDir, server)
	config := mustSnapshotReuseConfig(t, server.URL())
	inspector := snapshotstore.NewEncryptedStore(baseDir, nil)

	if outcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "token-one"}); !outcome.Success {
		t.Fatalf("expected first sync success, got %#v", outcome)
	}
	candidates, err := snapshotstore.DiscoverServerCandidates(context.Background(), inspector, server.URL())
	if err != nil {
		t.Fatalf("discover candidates: %v", err)
	}
	beforeBytes, err := os.ReadFile(candidates[0].Path)
	if err != nil {
		t.Fatalf("read existing snapshot: %v", err)
	}
	if _, err := inspector.Read(context.Background(), snapshotstore.ReadRequest{Candidate: candidates[0], SecurityToken: "wrong-token"}); err == nil {
		t.Fatalf("expected wrong token to fail snapshot unlock")
	}

	outcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "wrong-token"})
	if outcome.FailureReason != runtime.SyncFailureRejectedToken {
		t.Fatalf("expected rejected token outcome, got %#v", outcome)
	}
	afterBytes, err := os.ReadFile(candidates[0].Path)
	if err != nil {
		t.Fatalf("read existing snapshot after failure: %v", err)
	}
	if string(beforeBytes) != string(afterBytes) {
		t.Fatalf("expected wrong-token denial to keep snapshot unchanged")
	}
}

func TestSnapshotReuseFlowLeavesLocalDataUnchangedForInvalidToken(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	server := newTokenAwareStorageServer(t)
	server.SetTokenPages("token-one", []storagePageFixture{{
		Count:          1,
		ActivitiesJSON: `[{"id":"activity-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}]`,
	}})
	server.RejectToken("invalid-token")
	service := newTokenAwareSyncService(baseDir, server)
	config := mustSnapshotReuseConfig(t, server.URL())
	inspector := snapshotstore.NewEncryptedStore(baseDir, nil)

	if outcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "token-one"}); !outcome.Success {
		t.Fatalf("expected first sync success, got %#v", outcome)
	}
	beforeCandidates, err := snapshotstore.DiscoverServerCandidates(context.Background(), inspector, server.URL())
	if err != nil {
		t.Fatalf("discover candidates before invalid attempt: %v", err)
	}
	beforeBytes, err := os.ReadFile(beforeCandidates[0].Path)
	if err != nil {
		t.Fatalf("read snapshot before invalid attempt: %v", err)
	}

	outcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "invalid-token"})
	if outcome.FailureReason != runtime.SyncFailureRejectedToken {
		t.Fatalf("expected rejected token outcome, got %#v", outcome)
	}
	afterCandidates, err := snapshotstore.DiscoverServerCandidates(context.Background(), inspector, server.URL())
	if err != nil {
		t.Fatalf("discover candidates after invalid attempt: %v", err)
	}
	if len(afterCandidates) != len(beforeCandidates) {
		t.Fatalf("expected invalid token to leave candidate count unchanged")
	}
	afterBytes, err := os.ReadFile(afterCandidates[0].Path)
	if err != nil {
		t.Fatalf("read snapshot after invalid attempt: %v", err)
	}
	if string(beforeBytes) != string(afterBytes) {
		t.Fatalf("expected invalid token to leave snapshot unchanged")
	}
}

type tokenAwareStorageServer struct {
	testServer *httptest.Server
	mutex      sync.Mutex
	tokenPages map[string][]storagePageFixture
	rejected   map[string]bool
	pageIndex  map[string]int
	url        string
}

func newTokenAwareStorageServer(t *testing.T) *tokenAwareStorageServer {
	t.Helper()

	server := &tokenAwareStorageServer{
		tokenPages: map[string][]storagePageFixture{},
		rejected:   map[string]bool{},
		pageIndex:  map[string]int{},
	}
	server.testServer = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/v1/auth/anonymous":
			var payload struct {
				AccessToken string `json:"accessToken"`
			}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			server.mutex.Lock()
			rejected := server.rejected[payload.AccessToken]
			_, known := server.tokenPages[payload.AccessToken]
			server.mutex.Unlock()
			if rejected || !known {
				writer.WriteHeader(http.StatusForbidden)
				return
			}
			_, _ = writer.Write([]byte(`{"authToken":"jwt-` + payload.AccessToken + `"}`))
		case "/api/v1/activities":
			token := request.Header.Get("Authorization")
			token = token[len("Bearer jwt-"):]
			server.mutex.Lock()
			pages := server.tokenPages[token]
			index := server.pageIndex[token]
			if index >= len(pages) {
				index = len(pages) - 1
			}
			page := pages[index]
			server.pageIndex[token] = index + 1
			server.mutex.Unlock()
			_, _ = writer.Write([]byte(fmt.Sprintf(`{"activities":%s,"count":%d}`, page.ActivitiesJSON, page.Count)))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	server.url = server.testServer.URL
	t.Cleanup(server.testServer.Close)
	return server
}

func (s *tokenAwareStorageServer) Client() *http.Client {
	return s.testServer.Client()
}

func (s *tokenAwareStorageServer) URL() string {
	return s.url
}

func (s *tokenAwareStorageServer) SetTokenPages(token string, pages []storagePageFixture) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.tokenPages[token] = pages
	s.pageIndex[token] = 0
	delete(s.rejected, token)
}

func (s *tokenAwareStorageServer) RejectToken(token string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.rejected[token] = true
	delete(s.tokenPages, token)
	s.pageIndex[token] = 0
}

func newTokenAwareSyncService(baseDir string, server *tokenAwareStorageServer) runtime.SyncService {
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

func mustSnapshotReuseConfig(t *testing.T, origin string) configmodel.AppSetupConfig {
	t.Helper()

	config, err := configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, origin, true, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}
	return config
}

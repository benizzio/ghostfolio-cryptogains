package integration

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
	snapshotenvelope "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/envelope"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	syncvalidate "github.com/benizzio/ghostfolio-cryptogains/internal/sync/validate"
)

func TestSnapshotCompatibilityFlowRejectsUnsupportedEnvelopeVersion(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	server := newTokenAwareStorageServer(t)
	server.SetTokenPages("token-one", []storagePageFixture{{Count: 1, ActivitiesJSON: `[]`}})
	service := newTokenAwareSyncService(baseDir, server)
	config := mustSnapshotReuseConfig(t, server.URL())
	path := writeIntegrationSnapshotFixture(t, baseDir, rawSnapshotFixture{
		SnapshotID:    "unsupported-envelope",
		ServerOrigin:  server.URL(),
		SecurityToken: "token-one",
		FormatVersion: snapshotmodel.EnvelopeFormatVersion + 1,
		Payload:       defaultRawSnapshotPayload(server.URL()),
	})
	beforeBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot before validate: %v", err)
	}

	outcome := service.Validate(context.Background(), runtime.ValidateRequest{Config: config, SecurityToken: "token-one"})
	if outcome.FailureReason != runtime.SyncFailureUnsupportedStoredDataVersion {
		t.Fatalf("expected unsupported stored-data version outcome, got %#v", outcome)
	}
	afterBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot after validate: %v", err)
	}
	if string(beforeBytes) != string(afterBytes) {
		t.Fatalf("expected unsupported envelope version to leave snapshot unchanged")
	}
}

func TestSnapshotCompatibilityFlowRejectsUnsupportedPayloadVersion(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	server := newTokenAwareStorageServer(t)
	server.SetTokenPages("token-one", []storagePageFixture{{Count: 1, ActivitiesJSON: `[]`}})
	service := newTokenAwareSyncService(baseDir, server)
	config := mustSnapshotReuseConfig(t, server.URL())
	payload := defaultRawSnapshotPayload(server.URL())
	payload.StoredDataVersion.PayloadSchemaVersion++
	path := writeIntegrationSnapshotFixture(t, baseDir, rawSnapshotFixture{
		SnapshotID:    "unsupported-payload",
		ServerOrigin:  server.URL(),
		SecurityToken: "token-one",
		FormatVersion: snapshotmodel.EnvelopeFormatVersion,
		Payload:       payload,
	})
	beforeBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot before validate: %v", err)
	}

	outcome := service.Validate(context.Background(), runtime.ValidateRequest{Config: config, SecurityToken: "token-one"})
	if outcome.FailureReason != runtime.SyncFailureUnsupportedStoredDataVersion {
		t.Fatalf("expected unsupported stored-data version outcome, got %#v", outcome)
	}
	afterBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot after validate: %v", err)
	}
	if string(beforeBytes) != string(afterBytes) {
		t.Fatalf("expected unsupported payload version to leave snapshot unchanged")
	}
}

func TestSnapshotCompatibilityFlowRetainsReadableSnapshotWhenNewWriteIsIncompatible(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	server := newTokenAwareStorageServer(t)
	server.SetTokenPages("token-one", []storagePageFixture{{
		Count:          1,
		ActivitiesJSON: `[{"id":"activity-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}]`,
	}})
	baseStore := snapshotstore.NewEncryptedStore(baseDir, nil)
	wrappingStore := &failingCompatibilityWriteStore{Store: baseStore}
	service := runtime.NewSyncService(
		ghostfolioclient.New(server.Client()),
		time.Second,
		decimalsupport.NewService(),
		syncnormalize.NewNormalizer(),
		syncvalidate.NewValidator(),
		wrappingStore,
	)
	config := mustSnapshotReuseConfig(t, server.URL())

	if outcome := service.Validate(context.Background(), runtime.ValidateRequest{Config: config, SecurityToken: "token-one"}); !outcome.Success {
		t.Fatalf("expected first sync success, got %#v", outcome)
	}
	inspector := snapshotstore.NewEncryptedStore(baseDir, nil)
	candidates, err := snapshotstore.DiscoverServerCandidates(context.Background(), inspector, server.URL())
	if err != nil {
		t.Fatalf("discover candidates after first sync: %v", err)
	}
	beforeBytes, err := os.ReadFile(candidates[0].Path)
	if err != nil {
		t.Fatalf("read snapshot before incompatible write: %v", err)
	}
	beforePayload, err := inspector.Read(context.Background(), snapshotstore.ReadRequest{Candidate: candidates[0], SecurityToken: "token-one"})
	if err != nil {
		t.Fatalf("read payload before incompatible write: %v", err)
	}

	wrappingStore.failWrites = true
	server.SetTokenPages("token-one", []storagePageFixture{{
		Count:          1,
		ActivitiesJSON: `[{"id":"activity-2","date":"2025-01-01T10:00:00Z","type":"BUY","quantity":2,"valueInBaseCurrency":200,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}]`,
	}})
	outcome := service.Validate(context.Background(), runtime.ValidateRequest{Config: config, SecurityToken: "token-one"})
	if outcome.FailureReason != runtime.SyncFailureIncompatibleNewSyncData {
		t.Fatalf("expected incompatible new sync data outcome, got %#v", outcome)
	}
	afterBytes, err := os.ReadFile(candidates[0].Path)
	if err != nil {
		t.Fatalf("read snapshot after incompatible write: %v", err)
	}
	if string(beforeBytes) != string(afterBytes) {
		t.Fatalf("expected incompatible new sync data to leave snapshot bytes unchanged")
	}
	afterPayload, err := inspector.Read(context.Background(), snapshotstore.ReadRequest{Candidate: candidates[0], SecurityToken: "token-one"})
	if err != nil {
		t.Fatalf("read payload after incompatible write: %v", err)
	}
	if afterPayload.ProtectedActivityCache.Activities[0].SourceID != beforePayload.ProtectedActivityCache.Activities[0].SourceID {
		t.Fatalf("expected previous readable payload to remain active and unchanged")
	}
}

type failingCompatibilityWriteStore struct {
	snapshotstore.Store
	failWrites bool
}

func (s *failingCompatibilityWriteStore) Write(ctx context.Context, request snapshotstore.WriteRequest) (snapshotstore.Candidate, error) {
	if s.failWrites {
		return snapshotstore.Candidate{}, snapshotstore.ErrIncompatibleStoredData
	}
	return s.Store.Write(ctx, request)
}

func writeIntegrationSnapshotFixture(t *testing.T, baseDir string, fixture rawSnapshotFixture) string {
	t.Helper()

	codec := snapshotenvelope.NewJSONCodec()
	filesystem := snapshotstore.NewFilesystemStore(baseDir, codec)
	payloadBytes, err := json.Marshal(fixture.Payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	header := snapshotmodel.EnvelopeHeader{
		Magic:              snapshotmodel.EnvelopeMagic,
		FormatVersion:      fixture.FormatVersion,
		ServerDiscoveryKey: snapshotenvelope.DeriveServerDiscoveryKey(fixture.ServerOrigin),
		KDFParameters:      snapshotmodel.DefaultKDFParameters(),
		Salt:               make([]byte, snapshotmodel.DefaultSaltLength),
		Nonce:              make([]byte, snapshotmodel.DefaultNonceLength),
	}
	ciphertext, err := snapshotenvelope.SealCiphertext(header, fixture.SecurityToken, payloadBytes)
	if err != nil {
		t.Fatalf("seal ciphertext: %v", err)
	}
	rawEnvelope, err := codec.Encode(snapshotmodel.Envelope{Header: header, Ciphertext: ciphertext})
	if err != nil {
		t.Fatalf("encode envelope: %v", err)
	}
	path := filesystem.SnapshotPath(fixture.SnapshotID)
	if err := snapshotstore.ReplaceFileAtomically(path, rawEnvelope); err != nil {
		t.Fatalf("write snapshot fixture: %v", err)
	}
	return path
}

type rawSnapshotFixture struct {
	SnapshotID    string
	ServerOrigin  string
	SecurityToken string
	FormatVersion int
	Payload       snapshotmodel.Payload
}

func defaultRawSnapshotPayload(serverOrigin string) snapshotmodel.Payload {
	return snapshotmodel.Payload{
		StoredDataVersion: snapshotmodel.DefaultStoredDataVersion(""),
		RegisteredLocalUser: snapshotmodel.RegisteredLocalUser{
			LocalUserID:          "user",
			CreatedAt:            time.Unix(1, 0).UTC(),
			UpdatedAt:            time.Unix(1, 0).UTC(),
			LastSuccessfulSyncAt: time.Unix(1, 0).UTC(),
		},
		SetupProfile: snapshotmodel.SetupProfile{
			ServerOrigin:      serverOrigin,
			ServerMode:        "custom_origin",
			LastValidatedAt:   time.Unix(1, 0).UTC(),
			SourceAPIBasePath: "api/v1",
		},
		ProtectedActivityCache: syncmodel.ProtectedActivityCache{
			SyncedAt:             time.Unix(1, 0).UTC(),
			AvailableReportYears: []int{},
			Activities:           []syncmodel.ActivityRecord{},
		},
	}
}

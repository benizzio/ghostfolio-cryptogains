package contract

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
	snapshotenvelope "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/envelope"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	syncvalidate "github.com/benizzio/ghostfolio-cryptogains/internal/sync/validate"
)

func TestProtectedSnapshotContractLimitsDiscoveryToSelectedServer(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	store := snapshotstore.NewEncryptedStore(baseDir, snapshotenvelope.NewJSONCodec())
	_, err := store.Write(context.Background(), snapshotstore.WriteRequest{
		SnapshotID:    "selected-server",
		SecurityToken: "token-a",
		ServerOrigin:  "https://selected.example",
		Payload:       protectedSnapshotContractPayload("https://selected.example"),
	})
	if err != nil {
		t.Fatalf("write selected-server snapshot: %v", err)
	}
	_, err = store.Write(context.Background(), snapshotstore.WriteRequest{
		SnapshotID:    "other-server",
		SecurityToken: "token-b",
		ServerOrigin:  "https://other.example",
		Payload:       protectedSnapshotContractPayload("https://other.example"),
	})
	if err != nil {
		t.Fatalf("write other-server snapshot: %v", err)
	}

	candidates, err := snapshotstore.DiscoverServerCandidates(context.Background(), store, "https://selected.example")
	if err != nil {
		t.Fatalf("discover server candidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0].SnapshotID != "selected-server" {
		t.Fatalf("unexpected server-scoped candidates: %#v", candidates)
	}
}

func TestProtectedSnapshotContractFailsSafelyForUnsupportedStoredDataVersions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		mutate  func(snapshotmodel.Payload) snapshotmodel.Payload
		version int
	}{
		{
			name:    "unsupported envelope version",
			mutate:  func(payload snapshotmodel.Payload) snapshotmodel.Payload { return payload },
			version: snapshotmodel.EnvelopeFormatVersion + 1,
		},
		{
			name: "unsupported payload version",
			mutate: func(payload snapshotmodel.Payload) snapshotmodel.Payload {
				payload.StoredDataVersion.PayloadSchemaVersion++
				return payload
			},
			version: snapshotmodel.EnvelopeFormatVersion,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			baseDir := t.TempDir()
			authRequests := 0
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				authRequests++
				writer.Header().Set("Content-Type", "application/json")
				_, _ = writer.Write([]byte(`{"authToken":"jwt"}`))
			}))
			defer server.Close()

			payload := testCase.mutate(protectedSnapshotContractPayload(server.URL))
			path := writeProtectedSnapshotContractFixture(t, baseDir, snapshotFixtureContract{
				SnapshotID:    "fixture",
				ServerOrigin:  server.URL,
				SecurityToken: "token",
				FormatVersion: testCase.version,
				Payload:       payload,
			})
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read snapshot before validate: %v", err)
			}

			config, err := configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, server.URL, true, time.Now())
			if err != nil {
				t.Fatalf("new setup config: %v", err)
			}
			service := runtime.NewSyncService(
				ghostfolioclient.New(server.Client()),
				time.Second,
				baseDir,
				true,
				decimalsupport.NewService(),
				syncnormalize.NewNormalizer(),
				syncvalidate.NewValidator(),
				snapshotstore.NewEncryptedStore(baseDir, snapshotenvelope.NewJSONCodec()),
			)

			outcome := service.Validate(context.Background(), runtime.ValidateRequest{Config: config, SecurityToken: "token"})
			if outcome.FailureReason != runtime.SyncFailureUnsupportedStoredDataVersion {
				t.Fatalf("expected unsupported stored-data version outcome, got %#v", outcome)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read snapshot after validate: %v", err)
			}
			if string(before) != string(after) {
				t.Fatalf("expected unsupported stored-data version to leave snapshot unchanged")
			}
			if authRequests != 0 {
				t.Fatalf("expected local compatibility failure before auth, got %d auth requests", authRequests)
			}
		})
	}
}

type snapshotFixtureContract struct {
	SnapshotID    string
	ServerOrigin  string
	SecurityToken string
	FormatVersion int
	Payload       snapshotmodel.Payload
}

func writeProtectedSnapshotContractFixture(t *testing.T, baseDir string, fixture snapshotFixtureContract) string {
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

func protectedSnapshotContractPayload(serverOrigin string) snapshotmodel.Payload {
	return snapshotmodel.Payload{
		StoredDataVersion: snapshotmodel.DefaultStoredDataVersion(""),
		RegisteredLocalUser: snapshotmodel.RegisteredLocalUser{
			LocalUserID:          "user-1",
			CreatedAt:            time.Unix(1, 0).UTC(),
			UpdatedAt:            time.Unix(1, 0).UTC(),
			LastSuccessfulSyncAt: time.Unix(1, 0).UTC(),
		},
		SetupProfile: snapshotmodel.SetupProfile{
			ServerOrigin:      serverOrigin,
			ServerMode:        string(configmodel.ServerModeCustomOrigin),
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

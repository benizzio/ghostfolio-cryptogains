package unit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	snapshotenvelope "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/envelope"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

func TestDiscoverServerCandidatesFiltersBySelectedServer(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	store := snapshotstore.NewEncryptedStore(baseDir, snapshotenvelope.NewJSONCodec())
	writeDiscoverySnapshot(t, store, "snapshot-a", "https://server-a.example", "token-a")
	writeDiscoverySnapshot(t, store, "snapshot-b", "https://server-b.example", "token-b")

	candidates, err := snapshotstore.DiscoverServerCandidates(context.Background(), store, "https://server-a.example")
	if err != nil {
		t.Fatalf("discover server candidates: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected one matching candidate, got %d", len(candidates))
	}
	if candidates[0].SnapshotID != "snapshot-a" {
		t.Fatalf("unexpected candidate: %#v", candidates[0])
	}
}

// writeDiscoverySnapshot persists one minimal protected snapshot for discovery tests.
// Authored by: OpenCode
func writeDiscoverySnapshot(t *testing.T, store snapshotstore.Store, snapshotID string, serverOrigin string, token string) {
	t.Helper()

	_, err := store.Write(context.Background(), snapshotstore.WriteRequest{
		SnapshotID:    snapshotID,
		SecurityToken: token,
		ServerOrigin:  serverOrigin,
		Payload: snapshotmodel.Payload{
			StoredDataVersion: snapshotmodel.DefaultStoredDataVersion(""),
			RegisteredLocalUser: snapshotmodel.RegisteredLocalUser{
				LocalUserID:          snapshotID,
				CreatedAt:            time.Now().UTC(),
				UpdatedAt:            time.Now().UTC(),
				LastSuccessfulSyncAt: time.Now().UTC(),
			},
			SetupProfile: snapshotmodel.SetupProfile{
				ServerOrigin:      serverOrigin,
				ServerMode:        "custom_origin",
				LastValidatedAt:   time.Now().UTC(),
				SourceAPIBasePath: "api/v1",
			},
			ProtectedActivityCache: syncmodel.ProtectedActivityCache{
				SyncedAt:             time.Now().UTC(),
				AvailableReportYears: []int{},
				Activities:           []syncmodel.ActivityRecord{},
			},
		},
	})
	if err != nil {
		t.Fatalf("write discovery snapshot: %v", err)
	}
}

func TestDiscoverServerCandidatesPreservesSupportedAndUnsupportedHeadersForMatchingServer(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	writeRawSnapshotFixture(t, baseDir, rawSnapshotFixture{
		SnapshotID:    "supported",
		ServerOrigin:  "https://server-a.example",
		SecurityToken: "token-a",
		FormatVersion: snapshotmodel.EnvelopeFormatVersion,
		Payload:       defaultRawSnapshotPayload("https://server-a.example"),
	})
	writeRawSnapshotFixture(t, baseDir, rawSnapshotFixture{
		SnapshotID:    "unsupported",
		ServerOrigin:  "https://server-a.example",
		SecurityToken: "token-b",
		FormatVersion: snapshotmodel.EnvelopeFormatVersion + 1,
		Payload:       defaultRawSnapshotPayload("https://server-a.example"),
	})
	writeRawSnapshotFixture(t, baseDir, rawSnapshotFixture{
		SnapshotID:    "other-server",
		ServerOrigin:  "https://server-b.example",
		SecurityToken: "token-c",
		FormatVersion: snapshotmodel.EnvelopeFormatVersion + 1,
		Payload:       defaultRawSnapshotPayload("https://server-b.example"),
	})

	store := snapshotstore.NewEncryptedStore(baseDir, snapshotenvelope.NewJSONCodec())
	candidates, err := snapshotstore.DiscoverServerCandidates(context.Background(), store, "https://server-a.example")
	if err != nil {
		t.Fatalf("discover server candidates: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected two matching candidates, got %d", len(candidates))
	}
	if candidates[0].SnapshotID != "supported" || candidates[1].SnapshotID != "unsupported" {
		t.Fatalf("unexpected candidate order: %#v", candidates)
	}
}

type rawSnapshotFixture struct {
	SnapshotID    string
	ServerOrigin  string
	SecurityToken string
	FormatVersion int
	Payload       snapshotmodel.Payload
}

func writeRawSnapshotFixture(t *testing.T, baseDir string, fixture rawSnapshotFixture) {
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
	if err := snapshotstore.ReplaceFileAtomically(filesystem.SnapshotPath(fixture.SnapshotID), rawEnvelope); err != nil {
		t.Fatalf("write raw snapshot fixture: %v", err)
	}
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

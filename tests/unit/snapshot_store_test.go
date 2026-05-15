package unit

import (
	"context"
	"testing"
	"time"

	snapshotenvelope "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/envelope"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

func TestSnapshotStoreWritesAndReadsProtectedPayloadAtomically(t *testing.T) {
	t.Parallel()

	store := snapshotstore.NewEncryptedStore(t.TempDir(), snapshotenvelope.NewJSONCodec())
	payload := snapshotmodel.Payload{
		StoredDataVersion: snapshotmodel.DefaultStoredDataVersion(""),
		RegisteredLocalUser: snapshotmodel.RegisteredLocalUser{
			LocalUserID:          "user-1",
			CreatedAt:            time.Now().UTC(),
			UpdatedAt:            time.Now().UTC(),
			LastSuccessfulSyncAt: time.Now().UTC(),
		},
		SetupProfile: snapshotmodel.SetupProfile{ServerOrigin: "https://ghostfol.io", ServerMode: "ghostfolio_cloud", LastValidatedAt: time.Now().UTC(), SourceAPIBasePath: "api/v1"},
		ProtectedActivityCache: syncmodel.ProtectedActivityCache{
			SyncedAt:             time.Now().UTC(),
			RetrievedCount:       0,
			ActivityCount:        0,
			AvailableReportYears: []int{},
			ScopeReliability:     syncmodel.ScopeReliabilityUnavailable,
			Activities:           []syncmodel.ActivityRecord{},
		},
	}

	candidate, err := store.Write(context.Background(), snapshotstore.WriteRequest{SecurityToken: "token", ServerOrigin: "https://ghostfol.io", Payload: payload})
	if err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	loaded, err := store.Read(context.Background(), snapshotstore.ReadRequest{Candidate: candidate, SecurityToken: "token"})
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if loaded.SetupProfile.ServerOrigin != "https://ghostfol.io" {
		t.Fatalf("unexpected loaded payload: %#v", loaded)
	}
}

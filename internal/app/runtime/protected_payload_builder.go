// Package runtime assembles application dependencies for the TUI runtime.
// Authored by: OpenCode
package runtime

import (
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// protectedPayloadBuildRequest contains the inputs required to build one
// protected snapshot payload for a successful sync.
// Authored by: OpenCode
type protectedPayloadBuildRequest struct {
	Config          configmodel.AppSetupConfig
	Cache           syncmodel.ProtectedActivityCache
	ExistingPayload snapshotmodel.Payload
	HasExisting     bool
}

// protectedPayloadBuilder builds the persisted protected snapshot payload.
// Authored by: OpenCode
type protectedPayloadBuilder struct{}

// Build constructs the protected payload stored after a successful sync.
// Authored by: OpenCode
func (protectedPayloadBuilder) Build(request protectedPayloadBuildRequest) snapshotmodel.Payload {
	var now = request.Cache.SyncedAt.UTC()
	var registeredLocalUser snapshotmodel.RegisteredLocalUser
	if request.HasExisting {
		registeredLocalUser = request.ExistingPayload.RegisteredLocalUser
		registeredLocalUser.UpdatedAt = now
		registeredLocalUser.LastSuccessfulSyncAt = now
	} else {
		var localUserID string
		if generatedID, err := randomIdentifier(16); err == nil {
			localUserID = generatedID
		}
		registeredLocalUser = snapshotmodel.RegisteredLocalUser{
			LocalUserID:          localUserID,
			CreatedAt:            now,
			UpdatedAt:            now,
			LastSuccessfulSyncAt: now,
		}
	}

	return snapshotmodel.Payload{
		StoredDataVersion:   snapshotmodel.DefaultStoredDataVersion(""),
		RegisteredLocalUser: registeredLocalUser,
		SetupProfile: snapshotmodel.SetupProfile{
			ServerOrigin:      request.Config.ServerOrigin,
			ServerMode:        request.Config.ServerMode,
			AllowDevHTTP:      request.Config.AllowDevHTTP,
			LastValidatedAt:   now,
			SourceAPIBasePath: "api/v1",
		},
		ProtectedActivityCache: request.Cache,
	}
}

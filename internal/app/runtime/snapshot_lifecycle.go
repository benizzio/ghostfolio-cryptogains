// Package runtime assembles application dependencies for the TUI runtime.
// Authored by: OpenCode
package runtime

import (
	"context"
	"errors"

	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// Test seams wrap envelope compatibility checks so runtime tests can inject
// stored-data validation failures safely.
// Authored by: OpenCode
var validateSnapshotEnvelopeCompatibility = snapshotstore.ValidateEnvelopeCompatibility

var errSnapshotStoreUnavailable = errors.New("protected snapshot store is unavailable")

// snapshotUnlockResult stores the readable protected snapshot discovered before
// a new sync starts.
// Authored by: OpenCode
type snapshotUnlockResult struct {
	Candidate snapshotstore.Candidate
	Payload   snapshotmodel.Payload
	Unlocked  bool
}

// snapshotPersistRequest contains the inputs required to build and store the
// next protected snapshot payload.
// Authored by: OpenCode
type snapshotPersistRequest struct {
	Config        configmodel.AppSetupConfig
	SecurityToken string
	Cache         syncmodel.ProtectedActivityCache
	Existing      snapshotUnlockResult
}

// snapshotLifecycle coordinates snapshot discovery, unlock attempts, payload
// construction, protected writes, and active-state updates.
// Authored by: OpenCode
type snapshotLifecycle struct {
	store    snapshotstore.Store
	state    *activeSnapshotState
	payloads protectedPayloadBuilder
}

// newSnapshotLifecycle creates the runtime snapshot lifecycle collaborator.
// Authored by: OpenCode
func newSnapshotLifecycle(
	store snapshotstore.Store,
	state *activeSnapshotState,
	payloads protectedPayloadBuilder,
) *snapshotLifecycle {
	if state == nil {
		state = newActiveSnapshotState()
	}

	return &snapshotLifecycle{store: store, state: state, payloads: payloads}
}

// ProtectedDataState reports whether a readable protected snapshot is active for this run.
// Authored by: OpenCode
func (s *snapshotLifecycle) ProtectedDataState() ProtectedDataState {
	if s == nil {
		return ProtectedDataState{}
	}

	return s.state.ProtectedDataState()
}

// CheckServerReplacement compares the selected server against the active readable snapshot.
// Authored by: OpenCode
func (s *snapshotLifecycle) CheckServerReplacement(config configmodel.AppSetupConfig) ServerReplacementCheck {
	if s == nil {
		return ServerReplacementCheck{}
	}

	return s.state.CheckServerReplacement(config)
}

// SetActiveSnapshot records the readable protected snapshot for the current run.
// Authored by: OpenCode
func (s *snapshotLifecycle) SetActiveSnapshot(candidate snapshotstore.Candidate, payload snapshotmodel.Payload) {
	if s == nil {
		return
	}

	s.state.Set(candidate, payload)
}

// DiscoverAndUnlock enumerates selected-server snapshot candidates and tries to
// unlock them with the supplied token.
// Authored by: OpenCode
func (s *snapshotLifecycle) DiscoverAndUnlock(
	ctx context.Context,
	serverOrigin string,
	securityToken string,
) (snapshotUnlockResult, error) {
	if s == nil || s.store == nil {
		return snapshotUnlockResult{}, errSnapshotStoreUnavailable
	}

	candidates, err := snapshotstore.DiscoverServerCandidates(ctx, s.store, serverOrigin)
	if err != nil {
		return snapshotUnlockResult{}, err
	}
	for _, candidate := range candidates {
		if err := validateSnapshotEnvelopeCompatibility(candidate.Header); err != nil {
			return snapshotUnlockResult{}, err
		}
	}
	for _, candidate := range candidates {
		payload, err := s.store.Read(ctx, snapshotstore.ReadRequest{
			Candidate:     candidate,
			SecurityToken: securityToken,
		})
		if err == nil {
			return snapshotUnlockResult{Candidate: candidate, Payload: payload, Unlocked: true}, nil
		}
		if errors.Is(err, snapshotstore.ErrUnsupportedStoredDataVersion) {
			return snapshotUnlockResult{}, err
		}
	}

	return snapshotUnlockResult{}, nil
}

// Persist builds the next protected payload, writes it atomically, and updates
// the readable in-memory snapshot state on success.
// Authored by: OpenCode
func (s *snapshotLifecycle) Persist(ctx context.Context, request snapshotPersistRequest) error {
	if s == nil || s.store == nil {
		return errSnapshotStoreUnavailable
	}

	var payload = s.payloads.Build(protectedPayloadBuildRequest{
		Config:          request.Config,
		Cache:           request.Cache,
		ExistingPayload: request.Existing.Payload,
		HasExisting:     request.Existing.Unlocked,
	})
	persistedCandidate, err := s.store.Write(ctx, snapshotstore.WriteRequest{
		SnapshotID:    request.Existing.Candidate.SnapshotID,
		SecurityToken: request.SecurityToken,
		ServerOrigin:  request.Config.ServerOrigin,
		Payload:       payload,
	})
	if err != nil {
		return err
	}

	s.state.Set(persistedCandidate, payload)
	return nil
}

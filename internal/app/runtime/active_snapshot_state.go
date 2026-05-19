// Package runtime assembles application dependencies for the TUI runtime.
// Authored by: OpenCode
package runtime

import (
	"sync"

	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
)

// activeReadableSnapshot stores the current run's successfully unlocked local
// protected snapshot.
// Authored by: OpenCode
type activeReadableSnapshot struct {
	Candidate snapshotstore.Candidate
	Payload   snapshotmodel.Payload
	Present   bool
}

// activeSnapshotState manages the readable protected snapshot kept in memory
// for the current process.
// Authored by: OpenCode
type activeSnapshotState struct {
	mutex    sync.Mutex
	snapshot activeReadableSnapshot
}

// newActiveSnapshotState creates the in-memory readable-snapshot state manager.
// Authored by: OpenCode
func newActiveSnapshotState() *activeSnapshotState {
	return &activeSnapshotState{}
}

// Set records the readable protected snapshot for the current run.
// Authored by: OpenCode
func (s *activeSnapshotState) Set(candidate snapshotstore.Candidate, payload snapshotmodel.Payload) {
	if s == nil {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.snapshot = activeReadableSnapshot{Candidate: candidate, Payload: payload, Present: true}
}

// ProtectedDataState reports whether a readable protected snapshot is active for this run.
// Authored by: OpenCode
func (s *activeSnapshotState) ProtectedDataState() ProtectedDataState {
	if s == nil {
		return ProtectedDataState{}
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.snapshot.Present {
		return ProtectedDataState{}
	}

	return ProtectedDataState{
		HasReadableSnapshot: true,
		ServerOrigin:        s.snapshot.Payload.SetupProfile.ServerOrigin,
	}
}

// CheckServerReplacement compares the selected server against the active readable snapshot.
// Authored by: OpenCode
func (s *activeSnapshotState) CheckServerReplacement(config configmodel.AppSetupConfig) ServerReplacementCheck {
	var state = s.ProtectedDataState()
	if !state.HasReadableSnapshot || state.ServerOrigin == "" || state.ServerOrigin == config.ServerOrigin {
		return ServerReplacementCheck{}
	}

	return ServerReplacementCheck{
		Required:             true,
		ActiveServerOrigin:   state.ServerOrigin,
		SelectedServerOrigin: config.ServerOrigin,
	}
}

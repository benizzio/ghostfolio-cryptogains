// Package runtime assembles application dependencies for the TUI runtime.
// Authored by: OpenCode
package runtime

import (
	"sync"

	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
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

	var cache = s.snapshot.Payload.ProtectedActivityCache
	var lastSuccessfulSyncAt = cache.SyncedAt
	if lastSuccessfulSyncAt.IsZero() {
		lastSuccessfulSyncAt = s.snapshot.Payload.RegisteredLocalUser.LastSuccessfulSyncAt
	}

	return ProtectedDataState{
		HasReadableSnapshot:  true,
		ServerOrigin:         s.snapshot.Payload.SetupProfile.ServerOrigin,
		ActivityCount:        cache.ActivityCount,
		LastSuccessfulSyncAt: lastSuccessfulSyncAt,
		AvailableReportYears: append([]int(nil), cache.AvailableReportYears...),
	}
}

// ReadableProtectedActivityCache returns the currently unlocked protected
// activity cache for this run.
// Authored by: OpenCode
func (s *activeSnapshotState) ReadableProtectedActivityCache() (syncmodel.ProtectedActivityCache, bool) {
	if s == nil {
		return syncmodel.ProtectedActivityCache{}, false
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.snapshot.Present {
		return syncmodel.ProtectedActivityCache{}, false
	}

	return cloneProtectedActivityCache(s.snapshot.Payload.ProtectedActivityCache), true
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

// cloneProtectedActivityCache copies slice-backed cache fields for read access.
// Authored by: OpenCode
func cloneProtectedActivityCache(cache syncmodel.ProtectedActivityCache) syncmodel.ProtectedActivityCache {
	cache.AvailableReportYears = append([]int(nil), cache.AvailableReportYears...)
	cache.Activities = append([]syncmodel.ActivityRecord(nil), cache.Activities...)
	for index := range cache.Activities {
		cache.Activities[index] = cloneActivityRecord(cache.Activities[index])
	}
	return cache
}

// cloneActivityRecord copies pointer-backed activity fields for read access.
// Authored by: OpenCode
func cloneActivityRecord(record syncmodel.ActivityRecord) syncmodel.ActivityRecord {
	record.OrderUnitPrice = cloneDecimal(record.OrderUnitPrice)
	record.OrderGrossValue = cloneDecimal(record.OrderGrossValue)
	record.OrderFeeAmount = cloneDecimal(record.OrderFeeAmount)
	record.AssetProfileUnitPrice = cloneDecimal(record.AssetProfileUnitPrice)
	record.AssetProfileFeeAmount = cloneDecimal(record.AssetProfileFeeAmount)
	record.BaseGrossValue = cloneDecimal(record.BaseGrossValue)
	record.BaseFeeAmount = cloneDecimal(record.BaseFeeAmount)
	if record.SourceScope != nil {
		var scope = *record.SourceScope
		record.SourceScope = &scope
	}
	return record
}

// cloneDecimal copies an optional decimal value.
// Authored by: OpenCode
func cloneDecimal(value *apd.Decimal) *apd.Decimal {
	if value == nil {
		return nil
	}
	var clone apd.Decimal
	clone.Set(value)
	return &clone
}

// Package model defines normalized sync data structures shared across sync,
// snapshot, and runtime packages.
// Authored by: OpenCode
package model

import "time"

// ScopeReliability identifies how much source scope data can be trusted for one
// normalized activity timeline.
// Authored by: OpenCode
type ScopeReliability string

const (
	// ScopeReliabilityReliable indicates a stable non-empty source scope.
	ScopeReliabilityReliable ScopeReliability = "reliable"

	// ScopeReliabilityPartial indicates incomplete or contradictory scope data.
	ScopeReliabilityPartial ScopeReliability = "partial"

	// ScopeReliabilityUnavailable indicates that no usable scope data was present.
	ScopeReliabilityUnavailable ScopeReliability = "unavailable"
)

// ProtectedActivityCache stores the normalized activity history prepared for
// protected snapshot persistence.
// Authored by: OpenCode
type ProtectedActivityCache struct {
	SyncedAt             time.Time
	RetrievedCount       int
	ActivityCount        int
	AvailableReportYears []int
	ScopeReliability     ScopeReliability
	Activities           []ActivityRecord
}

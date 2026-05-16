// Package model defines normalized sync data structures shared across sync,
// snapshot, and runtime packages.
// Authored by: OpenCode
package model

import "strings"

// ActivityOrderingKey stores the deterministic replay ordering inputs derived
// from one normalized activity.
// Authored by: OpenCode
type ActivityOrderingKey struct {
	SourceDate   string
	AssetSymbol  string
	ActivityType ActivityType
	SourceID     string
	OccurredAt   string
	RawHash      string
}

// NewActivityOrderingKey builds the shared deterministic ordering tuple used by
// normalization and validation.
// Authored by: OpenCode
func NewActivityOrderingKey(record ActivityRecord, sourceDate string) ActivityOrderingKey {
	return ActivityOrderingKey{
		SourceDate:   sourceDate,
		AssetSymbol:  record.AssetSymbol,
		ActivityType: record.ActivityType,
		SourceID:     record.SourceID,
		OccurredAt:   record.OccurredAt,
		RawHash:      record.RawHash,
	}
}

// CompareActivityOrdering compares two deterministic activity ordering tuples.
// Authored by: OpenCode
func CompareActivityOrdering(left ActivityOrderingKey, right ActivityOrderingKey) int {
	if comparison := strings.Compare(left.SourceDate, right.SourceDate); comparison != 0 {
		return comparison
	}
	if comparison := strings.Compare(left.AssetSymbol, right.AssetSymbol); comparison != 0 {
		return comparison
	}
	if leftActivityOrder, rightActivityOrder := activityTypeOrder(left.ActivityType), activityTypeOrder(right.ActivityType); leftActivityOrder != rightActivityOrder {
		if leftActivityOrder < rightActivityOrder {
			return -1
		}
		return 1
	}
	if comparison := strings.Compare(left.SourceID, right.SourceID); comparison != 0 {
		return comparison
	}
	if comparison := strings.Compare(left.OccurredAt, right.OccurredAt); comparison != 0 {
		return comparison
	}

	return strings.Compare(left.RawHash, right.RawHash)
}

// HasAmbiguousActivityOrdering reports whether two activities still collide
// after the supported same-day tie-break rules have been applied.
// Authored by: OpenCode
func HasAmbiguousActivityOrdering(left ActivityOrderingKey, right ActivityOrderingKey) bool {
	return left.SourceDate == right.SourceDate &&
		left.AssetSymbol == right.AssetSymbol &&
		left.ActivityType == right.ActivityType &&
		left.SourceID == right.SourceID &&
		left.RawHash != right.RawHash
}

// activityTypeOrder ranks supported activity types for same-asset same-day ordering.
// Authored by: OpenCode
func activityTypeOrder(activityType ActivityType) int {
	switch activityType {
	case ActivityTypeBuy:
		return 0
	case ActivityTypeSell:
		return 1
	default:
		return 2
	}
}

// Package model defines normalized sync data structures shared across sync,
// snapshot, and runtime packages.
// Authored by: OpenCode
package model

import (
	"strings"
	"time"
)

const activityOrderingSourceDateLayout = "2006-01-02"

// ActivityOrderingKey stores the deterministic same-asset replay ordering
// inputs derived from one normalized activity record.
//
// Normalization and validation use this value to keep the same source-calendar-
// date, activity-type, and source-identifier ordering rule aligned.
// Authored by: OpenCode
type ActivityOrderingKey struct {
	SourceDate   string
	AssetSymbol  string
	ActivityType ActivityType
	SourceID     string
	OccurredAt   string
	RawHash      string
}

// NewActivityOrderingKey builds the shared deterministic ordering tuple from a
// normalized activity and a caller-supplied source calendar date.
//
// Example:
//
//	key := model.NewActivityOrderingKey(record, "2024-01-31")
//	_ = key.SourceID
//
// Use this helper when the caller already parsed `record.OccurredAt` and only
// needs the repository-standard ordering tuple.
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

// NewActivityOrderingKeyFromRecord parses one normalized activity timestamp and
// builds the repository-standard deterministic ordering tuple for that record.
//
// Example:
//
//	key, occurredAt, err := model.NewActivityOrderingKeyFromRecord(record)
//	if err != nil {
//		panic(err)
//	}
//	_, _ = key, occurredAt
//
// Use this helper when the caller needs both the parsed `OccurredAt` value and
// the shared same-asset same-day ordering fields without rebuilding them inline.
// Authored by: OpenCode
func NewActivityOrderingKeyFromRecord(record ActivityRecord) (ActivityOrderingKey, time.Time, error) {
	var occurredAt, err = time.Parse(time.RFC3339Nano, record.OccurredAt)
	if err != nil {
		return ActivityOrderingKey{}, time.Time{}, err
	}

	return NewActivityOrderingKey(record, occurredAt.Format(activityOrderingSourceDateLayout)), occurredAt, nil
}

// CompareActivityOrdering compares two deterministic activity ordering tuples
// using the supported source-date, asset, type, source-id, timestamp, and hash
// precedence.
//
// Example:
//
//	comparison := model.CompareActivityOrdering(left, right)
//	_ = comparison < 0
//
// A negative result means `left` sorts before `right`, zero means the tuples are
// identical, and a positive result means `right` sorts first.
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
//
// Example:
//
//	if model.HasAmbiguousActivityOrdering(left, right) {
//		panic("ambiguous ordering")
//	}
//
// This helper is used after exact-duplicate removal to reject same-asset same-
// day records that still cannot be ordered uniquely.
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

// Package normalize defines the normalized activity-history transformation
// boundary.
// Authored by: OpenCode
package normalize

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// NormalizationError stores offending-record context for synced-data
// normalization failures.
// Authored by: OpenCode
type NormalizationError struct {
	message string
	context syncmodel.DiagnosticContext
}

// Error returns the non-secret normalization failure detail.
// Authored by: OpenCode
func (e *NormalizationError) Error() string {
	if e == nil {
		return ""
	}
	return e.message
}

// DiagnosticContext returns the structured troubleshooting context for one normalization failure.
// Authored by: OpenCode
func (e *NormalizationError) DiagnosticContext() syncmodel.DiagnosticContext {
	if e == nil {
		return syncmodel.DiagnosticContext{}
	}
	return e.context
}

// Normalizer defines the full-history normalization contract used by the sync
// workflow.
//
// Example:
//
//	var normalizer Normalizer
//	_, _ = normalizer.Normalize(nil)
//
// Implementations are expected to canonicalize supported activity inputs,
// establish deterministic ordering, remove exact duplicates, and derive the
// protected activity cache persisted after a successful sync.
// Authored by: OpenCode
type Normalizer interface {
	Normalize([]syncmodel.ActivityRecord) (syncmodel.ProtectedActivityCache, error)
}

// defaultNormalizer implements the chronological normalization seam used by the
// sync workflow.
// Authored by: OpenCode
type defaultNormalizer struct{}

// NewNormalizer creates the foundational normalization service used by runtime
// wiring.
//
// Example:
//
//	normalizer := normalize.NewNormalizer()
//	_, _ = normalizer.Normalize(nil)
//
// Authored by: OpenCode
func NewNormalizer() Normalizer {
	return defaultNormalizer{}
}

// Normalize sorts normalized activities chronologically, removes exact
// duplicates, derives available report years, and records duplicate hashes.
// Authored by: OpenCode
func (defaultNormalizer) Normalize(records []syncmodel.ActivityRecord) (syncmodel.ProtectedActivityCache, error) {
	typedRecords := make([]normalizedRecord, 0, len(records))
	for _, record := range records {
		occurredAt, err := time.Parse(time.RFC3339Nano, record.OccurredAt)
		if err != nil {
			return syncmodel.ProtectedActivityCache{}, newNormalizationError(fmt.Sprintf("normalize occurred_at: %v", err), record)
		}

		rawHash, err := recordHash(record)
		if err != nil {
			return syncmodel.ProtectedActivityCache{}, newNormalizationError(err.Error(), record)
		}

		typedRecords = append(typedRecords, normalizedRecord{
			OccurredAt: occurredAt,
			RawHash:    rawHash,
			Record:     record,
		})
	}

	sort.Slice(typedRecords, func(left int, right int) bool {
		if typedRecords[left].OccurredAt.Equal(typedRecords[right].OccurredAt) {
			return typedRecords[left].Record.SourceID < typedRecords[right].Record.SourceID
		}
		return typedRecords[left].OccurredAt.Before(typedRecords[right].OccurredAt)
	})

	if err := ensureStableOrdering(typedRecords); err != nil {
		return syncmodel.ProtectedActivityCache{}, err
	}

	activities, years := deduplicateAndDeriveYears(typedRecords)

	return syncmodel.ProtectedActivityCache{
		RetrievedCount:       len(records),
		ActivityCount:        len(activities),
		AvailableReportYears: years,
		ScopeReliability:     deriveScopeReliability(activities),
		Activities:           activities,
	}, nil
}

// normalizedRecord stores the parsed ordering key and duplicate hash for one activity.
// Authored by: OpenCode
type normalizedRecord struct {
	OccurredAt time.Time
	RawHash    string
	Record     syncmodel.ActivityRecord
}

// ensureStableOrdering rejects same-instant same-source collisions that are not exact duplicates.
// Authored by: OpenCode
func ensureStableOrdering(records []normalizedRecord) error {
	for index := 1; index < len(records); index++ {
		var previous = records[index-1]
		var current = records[index]
		if previous.OccurredAt.Equal(current.OccurredAt) && previous.Record.SourceID == current.Record.SourceID && previous.RawHash != current.RawHash {
			return newNormalizationError(
				fmt.Sprintf("supported activity ordering is ambiguous for source %q", current.Record.SourceID),
				previous.Record,
				current.Record,
			)
		}
	}

	return nil
}

// newNormalizationError captures the offending normalized records for one normalization failure.
// Authored by: OpenCode
func newNormalizationError(message string, records ...syncmodel.ActivityRecord) error {
	var diagnosticRecords = make([]syncmodel.DiagnosticRecord, 0, len(records))
	for _, record := range records {
		diagnosticRecords = append(diagnosticRecords, syncmodel.DiagnosticRecordFromActivityRecord(record))
	}

	return &NormalizationError{
		message: message,
		context: syncmodel.DiagnosticContext{
			FailureStage:  syncmodel.DiagnosticFailureStageNormalization,
			FailureDetail: message,
			Records:       diagnosticRecords,
		},
	}
}

// deduplicateAndDeriveYears keeps the first activity for each exact duplicate hash and derives distinct years.
// Authored by: OpenCode
func deduplicateAndDeriveYears(records []normalizedRecord) ([]syncmodel.ActivityRecord, []int) {
	var activities = make([]syncmodel.ActivityRecord, 0, len(records))
	var yearsByValue = make(map[int]struct{}, len(records))

	for _, item := range records {
		item.Record.RawHash = item.RawHash
		if len(activities) > 0 && activities[len(activities)-1].RawHash == item.RawHash {
			continue
		}

		yearsByValue[item.OccurredAt.Year()] = struct{}{}
		activities = append(activities, item.Record)
	}

	var years = make([]int, 0, len(yearsByValue))
	for year := range yearsByValue {
		years = append(years, year)
	}
	sort.Ints(years)

	return activities, years
}

// deriveScopeReliability records a coarse phase-3 scope-reliability summary.
// Authored by: OpenCode
func deriveScopeReliability(records []syncmodel.ActivityRecord) syncmodel.ScopeReliability {
	if len(records) == 0 {
		return syncmodel.ScopeReliabilityUnavailable
	}

	var timelinesByAsset = make(map[string][]syncmodel.ActivityRecord)
	for _, record := range records {
		timelinesByAsset[record.AssetSymbol] = append(timelinesByAsset[record.AssetSymbol], record)
	}

	var sawReliable = false
	for _, timeline := range timelinesByAsset {
		var reliability = deriveTimelineScopeReliability(timeline)
		if reliability == syncmodel.ScopeReliabilityPartial {
			return syncmodel.ScopeReliabilityPartial
		}
		if reliability == syncmodel.ScopeReliabilityReliable {
			sawReliable = true
		}
	}

	if sawReliable {
		return syncmodel.ScopeReliabilityReliable
	}

	return syncmodel.ScopeReliabilityUnavailable
}

// deriveTimelineScopeReliability evaluates one asset timeline for stable source-scope reliability.
// Authored by: OpenCode
func deriveTimelineScopeReliability(records []syncmodel.ActivityRecord) syncmodel.ScopeReliability {
	var sawUsableScope = false
	var expectedScopeID string
	var expectedScopeKind syncmodel.SourceScopeKind

	for _, record := range records {
		if record.SourceScope == nil || strings.TrimSpace(record.SourceScope.ID) == "" || record.SourceScope.Kind == "" {
			if sawUsableScope {
				return syncmodel.ScopeReliabilityPartial
			}
			continue
		}

		if !sawUsableScope {
			sawUsableScope = true
			expectedScopeID = strings.TrimSpace(record.SourceScope.ID)
			expectedScopeKind = record.SourceScope.Kind
			continue
		}
		if strings.TrimSpace(record.SourceScope.ID) != expectedScopeID || record.SourceScope.Kind != expectedScopeKind {
			return syncmodel.ScopeReliabilityPartial
		}
	}

	if !sawUsableScope {
		return syncmodel.ScopeReliabilityUnavailable
	}
	for _, record := range records {
		if record.SourceScope == nil || strings.TrimSpace(record.SourceScope.ID) == "" || record.SourceScope.Kind == "" {
			return syncmodel.ScopeReliabilityPartial
		}
	}

	return syncmodel.ScopeReliabilityReliable
}

// recordHash computes the exact duplicate-removal hash from normalized source fields.
// Authored by: OpenCode
func recordHash(record syncmodel.ActivityRecord) (string, error) {
	quantity, err := decimalsupport.CanonicalString(record.Quantity)
	if err != nil {
		return "", fmt.Errorf("hash activity quantity: %w", err)
	}
	unitPrice, err := decimalsupport.CanonicalString(record.UnitPrice)
	if err != nil {
		return "", fmt.Errorf("hash activity unit price: %w", err)
	}
	grossValue, err := decimalsupport.CanonicalString(record.GrossValue)
	if err != nil {
		return "", fmt.Errorf("hash activity gross value: %w", err)
	}
	feeAmount, err := decimalsupport.CanonicalStringPointer(record.FeeAmount)
	if err != nil {
		return "", fmt.Errorf("hash activity fee: %w", err)
	}

	var sourceScopeID = ""
	var sourceScopeName = ""
	var sourceScopeKind = ""
	var sourceScopeReliability = ""
	if record.SourceScope != nil {
		sourceScopeID = record.SourceScope.ID
		sourceScopeName = record.SourceScope.Name
		sourceScopeKind = string(record.SourceScope.Kind)
		sourceScopeReliability = string(record.SourceScope.Reliability)
	}

	var parts = []string{
		record.SourceID,
		record.OccurredAt,
		string(record.ActivityType),
		record.AssetSymbol,
		record.AssetName,
		record.BaseCurrency,
		quantity,
		unitPrice,
		grossValue,
		feeAmount,
		record.Comment,
		record.DataSource,
		sourceScopeID,
		sourceScopeName,
		sourceScopeKind,
		sourceScopeReliability,
	}

	var sum = sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:]), nil
}

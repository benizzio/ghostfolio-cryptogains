package normalize

import (
	"errors"
	"testing"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

func TestNormalizationErrorHelpersAndUnreadableTimestamp(t *testing.T) {
	t.Parallel()

	var nilError *NormalizationError
	if nilError.Error() != "" {
		t.Fatalf("expected nil normalization error string to be empty")
	}
	var context = nilError.DiagnosticContext()
	if context.FailureStage != "" || context.FailureDetail != "" || len(context.Records) != 0 {
		t.Fatalf("expected nil normalization error context to be empty, got %#v", context)
	}

	_, err := NewNormalizer().Normalize([]syncmodel.ActivityRecord{{OccurredAt: "bad-time"}})
	var normalizationError *NormalizationError
	if !errors.As(err, &normalizationError) {
		t.Fatalf("expected normalization error, got %v", err)
	}
	if normalizationError.DiagnosticContext().FailureStage != syncmodel.DiagnosticFailureStageNormalization {
		t.Fatalf("expected normalization failure stage, got %#v", normalizationError.DiagnosticContext())
	}
	if normalizationError.Error() == "" {
		t.Fatalf("expected non-empty normalization error string")
	}
}

func TestCompareNormalizedRecordsAndHashBranches(t *testing.T) {
	t.Parallel()

	left := normalizedRecord{OrderKey: syncmodel.ActivityOrderingKey{SourceDate: "2024-01-01", AssetSymbol: "BTC", ActivityType: syncmodel.ActivityTypeBuy, SourceID: "a", OccurredAt: "2024-01-01T12:00:00Z", RawHash: "aaa"}, Record: syncmodel.ActivityRecord{AssetSymbol: "BTC", SourceID: "a", OccurredAt: "2024-01-01T12:00:00Z"}, RawHash: "aaa"}
	right := normalizedRecord{OrderKey: syncmodel.ActivityOrderingKey{SourceDate: "2024-01-02", AssetSymbol: "BTC", ActivityType: syncmodel.ActivityTypeBuy, SourceID: "a", OccurredAt: "2024-01-01T12:00:00Z", RawHash: "aaa"}, Record: syncmodel.ActivityRecord{AssetSymbol: "BTC", SourceID: "a", OccurredAt: "2024-01-01T12:00:00Z"}, RawHash: "aaa"}
	if syncmodel.CompareActivityOrdering(left.OrderKey, right.OrderKey) >= 0 {
		t.Fatalf("expected earlier source date to sort first")
	}

	right.OrderKey.SourceDate = left.OrderKey.SourceDate
	right.Record.AssetSymbol = "ETH"
	right.OrderKey.AssetSymbol = "ETH"
	if syncmodel.CompareActivityOrdering(left.OrderKey, right.OrderKey) >= 0 {
		t.Fatalf("expected asset symbol tie-breaker")
	}

	right.Record.AssetSymbol = left.Record.AssetSymbol
	right.OrderKey.AssetSymbol = left.OrderKey.AssetSymbol
	right.OrderKey.ActivityType = syncmodel.ActivityTypeSell
	if syncmodel.CompareActivityOrdering(left.OrderKey, right.OrderKey) >= 0 {
		t.Fatalf("expected activity order tie-breaker")
	}
	if syncmodel.CompareActivityOrdering(right.OrderKey, left.OrderKey) <= 0 {
		t.Fatalf("expected reverse activity order tie-breaker")
	}

	right.OrderKey.ActivityType = left.OrderKey.ActivityType
	right.Record.SourceID = "z"
	right.OrderKey.SourceID = "z"
	if syncmodel.CompareActivityOrdering(left.OrderKey, right.OrderKey) >= 0 {
		t.Fatalf("expected source id tie-breaker")
	}

	right.Record.SourceID = left.Record.SourceID
	right.OrderKey.SourceID = left.OrderKey.SourceID
	right.Record.OccurredAt = "2024-01-01T13:00:00Z"
	right.OrderKey.OccurredAt = "2024-01-01T13:00:00Z"
	if syncmodel.CompareActivityOrdering(left.OrderKey, right.OrderKey) >= 0 {
		t.Fatalf("expected occurred_at tie-breaker")
	}

	right.Record.OccurredAt = left.Record.OccurredAt
	right.OrderKey.OccurredAt = left.OrderKey.OccurredAt
	right.RawHash = "zzz"
	right.OrderKey.RawHash = "zzz"
	if syncmodel.CompareActivityOrdering(left.OrderKey, right.OrderKey) >= 0 {
		t.Fatalf("expected raw-hash tie-breaker")
	}

	record := normalizationTestRecord(t, "activity-1", syncmodel.ActivityTypeBuy)
	record.SourceScope = &syncmodel.SourceScope{ID: "scope-1", Name: "Primary", Kind: syncmodel.SourceScopeKindWallet, Reliability: syncmodel.ScopeReliabilityReliable}
	hash, err := recordHash(record)
	if err != nil {
		t.Fatalf("record hash: %v", err)
	}
	if hash == "" {
		t.Fatalf("expected non-empty record hash")
	}

	var invalid apd.Decimal
	invalid.Form = apd.Infinite
	record.Quantity = invalid
	if _, err := recordHash(record); err == nil {
		t.Fatalf("expected invalid quantity canonicalization to fail")
	}

	record = normalizationTestRecord(t, "activity-2", syncmodel.ActivityTypeBuy)
	record.UnitPrice = invalid
	if _, err := recordHash(record); err == nil {
		t.Fatalf("expected invalid unit-price canonicalization to fail")
	}

	record = normalizationTestRecord(t, "activity-3", syncmodel.ActivityTypeBuy)
	record.GrossValue = invalid
	if _, err := recordHash(record); err == nil {
		t.Fatalf("expected invalid gross-value canonicalization to fail")
	}

	record = normalizationTestRecord(t, "activity-4", syncmodel.ActivityTypeBuy)
	record.FeeAmount = &invalid
	if _, err := recordHash(record); err == nil {
		t.Fatalf("expected invalid fee canonicalization to fail")
	}
}

func TestDeriveTimelineScopeReliabilityAndNormalizeSummaryBranches(t *testing.T) {
	t.Parallel()

	if got := deriveTimelineScopeReliability(nil); got != syncmodel.ScopeReliabilityUnavailable {
		t.Fatalf("expected unavailable reliability for empty timeline, got %q", got)
	}

	reliableRecord := normalizationTestRecord(t, "activity-1", syncmodel.ActivityTypeBuy)
	reliableRecord.SourceScope = &syncmodel.SourceScope{ID: "scope-1", Kind: syncmodel.SourceScopeKindAccount}
	if got := deriveTimelineScopeReliability([]syncmodel.ActivityRecord{reliableRecord}); got != syncmodel.ScopeReliabilityReliable {
		t.Fatalf("expected reliable timeline, got %q", got)
	}

	partialRecord := normalizationTestRecord(t, "activity-2", syncmodel.ActivityTypeBuy)
	if got := deriveTimelineScopeReliability([]syncmodel.ActivityRecord{reliableRecord, partialRecord}); got != syncmodel.ScopeReliabilityPartial {
		t.Fatalf("expected partial reliability when scope disappears, got %q", got)
	}

	conflictingRecord := normalizationTestRecord(t, "activity-3", syncmodel.ActivityTypeBuy)
	conflictingRecord.SourceScope = &syncmodel.SourceScope{ID: "scope-2", Kind: syncmodel.SourceScopeKindWallet}
	if got := deriveTimelineScopeReliability([]syncmodel.ActivityRecord{reliableRecord, conflictingRecord}); got != syncmodel.ScopeReliabilityPartial {
		t.Fatalf("expected partial reliability when scope changes, got %q", got)
	}

	if got := deriveTimelineScopeReliability([]syncmodel.ActivityRecord{partialRecord, reliableRecord}); got != syncmodel.ScopeReliabilityPartial {
		t.Fatalf("expected partial reliability when usable scope appears after a missing scope, got %q", got)
	}

	cache, err := NewNormalizer().Normalize([]syncmodel.ActivityRecord{normalizationTestRecord(t, "activity-4", syncmodel.ActivityTypeSell)})
	if err != nil {
		t.Fatalf("normalize single record: %v", err)
	}
	if cache.ScopeReliability != syncmodel.ScopeReliabilityUnavailable || len(cache.AvailableReportYears) != 1 || cache.AvailableReportYears[0] != 2024 {
		t.Fatalf("unexpected normalized cache summary: %#v", cache)
	}
}

// TestNormalizeCoversHashAndAmbiguousOrderingFailures verifies the remaining
// normalization-failure paths for hash generation and same-source collisions.
// Authored by: OpenCode
func TestNormalizeCoversHashAndAmbiguousOrderingFailures(t *testing.T) {
	t.Parallel()

	ambiguousLeft := normalizationTestRecord(t, "activity-1", syncmodel.ActivityTypeBuy)
	ambiguousRight := normalizationTestRecord(t, "activity-1", syncmodel.ActivityTypeBuy)
	ambiguousQuantity, _, err := decimalsupport.ParseString("2")
	if err != nil {
		t.Fatalf("parse ambiguous quantity: %v", err)
	}
	ambiguousRight.Quantity = ambiguousQuantity

	_, err = NewNormalizer().Normalize([]syncmodel.ActivityRecord{ambiguousLeft, ambiguousRight})
	var normalizationError *NormalizationError
	if !errors.As(err, &normalizationError) {
		t.Fatalf("expected ambiguous normalization error, got %v", err)
	}
	if len(normalizationError.DiagnosticContext().Records) != 2 {
		t.Fatalf("expected both ambiguous records in diagnostic context, got %#v", normalizationError.DiagnosticContext())
	}

	invalidRecord := normalizationTestRecord(t, "activity-2", syncmodel.ActivityTypeBuy)
	var invalid apd.Decimal
	invalid.Form = apd.Infinite
	invalidRecord.GrossValue = invalid
	_, err = NewNormalizer().Normalize([]syncmodel.ActivityRecord{invalidRecord})
	if !errors.As(err, &normalizationError) {
		t.Fatalf("expected hash-generation normalization error, got %v", err)
	}
	if normalizationError.Error() == "" {
		t.Fatalf("expected non-empty normalization error string")
	}
}

func normalizationTestRecord(t *testing.T, sourceID string, activityType syncmodel.ActivityType) syncmodel.ActivityRecord {
	t.Helper()

	quantity, _, err := decimalsupport.ParseString("1")
	if err != nil {
		t.Fatalf("parse quantity: %v", err)
	}
	unitPrice, _, err := decimalsupport.ParseString("100")
	if err != nil {
		t.Fatalf("parse unit price: %v", err)
	}
	grossValue, _, err := decimalsupport.ParseString("100")
	if err != nil {
		t.Fatalf("parse gross value: %v", err)
	}

	return syncmodel.ActivityRecord{
		SourceID:     sourceID,
		OccurredAt:   "2024-01-01T10:00:00Z",
		ActivityType: activityType,
		AssetSymbol:  "BTC",
		Quantity:     quantity,
		UnitPrice:    unitPrice,
		GrossValue:   grossValue,
	}
}

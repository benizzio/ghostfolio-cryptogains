// Package calculate verifies package-local activity-input helper branches that
// are easier to exercise directly than through black-box report runs.
// Authored by: OpenCode
package calculate

import (
	"strings"
	"testing"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

// TestSelectActivityCalculationInputAdditionalFailures verifies the remaining
// direct activity-input failure branches.
// Authored by: OpenCode
func TestSelectActivityCalculationInputAdditionalFailures(t *testing.T) {
	t.Parallel()

	t.Run("rejects invalid occurred-at timestamp through exported selector", func(t *testing.T) {
		var record = validActivityInputRecord(t)
		record.OccurredAt = "not-a-timestamp"

		_, err := SelectActivityCalculationInput(record)
		if err == nil || !strings.Contains(err.Error(), `activity "activity-1" occurred_at is invalid`) {
			t.Fatalf("expected invalid timestamp failure, got %v", err)
		}
	})

	t.Run("rejects non-finite priced quantity", func(t *testing.T) {
		var invalid apd.Decimal
		invalid.Form = apd.NaNSignaling

		var record = validActivityInputRecord(t)
		record.Quantity = invalid
		record.BaseCurrency = "USD"
		record.BaseGrossValue = activityInputDecimalPointer(t, "100")
		record.BaseFeeAmount = activityInputDecimalPointer(t, "1")

		_, err := SelectActivityCalculationInput(record)
		if err == nil || !strings.Contains(err.Error(), `activity "activity-1" quantity is invalid`) {
			t.Fatalf("expected invalid quantity failure, got %v", err)
		}
	})

	t.Run("rejects priced activity with no complete tier", func(t *testing.T) {
		_, err := SelectActivityCalculationInput(validActivityInputRecord(t))
		if err == nil || !strings.Contains(err.Error(), `priced activity requires one complete order, asset_profile, or base currency context`) {
			t.Fatalf("expected missing tier failure, got %v", err)
		}
	})
}

// TestActivityInputHelperBranches verifies direct helper behavior that the
// exported selector guards or normalizes earlier.
// Authored by: OpenCode
func TestActivityInputHelperBranches(t *testing.T) {
	t.Parallel()

	var blankRecord = syncmodel.ActivityRecord{SourceID: "activity-blank", OccurredAt: "   "}
	if _, err := parseActivityOccurredAt(blankRecord); err == nil || !strings.Contains(err.Error(), `activity "activity-blank" occurred_at is required`) {
		t.Fatalf("expected blank timestamp to fail, got %v", err)
	}

	var invalid apd.Decimal
	invalid.Form = apd.Infinite
	if value := informedGrossValue(&invalid, "USD", mustActivityInputDecimal(t, "1")); value != nil {
		t.Fatalf("expected invalid gross-value derivation to return nil, got %v", value)
	}
}

// validActivityInputRecord returns one minimal normalized activity fixture for
// package-local activity-input tests.
// Authored by: OpenCode
func validActivityInputRecord(t *testing.T) syncmodel.ActivityRecord {
	t.Helper()

	return syncmodel.ActivityRecord{
		SourceID:         "activity-1",
		OccurredAt:       "2024-02-01T10:11:12+02:00",
		ActivityType:     syncmodel.ActivityTypeBuy,
		AssetIdentityKey: "asset-btc",
		AssetSymbol:      "BTC",
		AssetName:        "Bitcoin",
		Quantity:         mustActivityInputDecimal(t, "10"),
	}
}

// activityInputDecimalPointer returns one exact decimal pointer for direct
// activity-input tests.
// Authored by: OpenCode
func activityInputDecimalPointer(t *testing.T, raw string) *apd.Decimal {
	t.Helper()

	var value = mustActivityInputDecimal(t, raw)
	return &value
}

// mustActivityInputDecimal parses one exact decimal for package-local
// activity-input tests.
// Authored by: OpenCode
func mustActivityInputDecimal(t *testing.T, raw string) apd.Decimal {
	t.Helper()

	var value, _, err = decimalsupport.ParseString(raw)
	if err != nil {
		t.Fatalf("parse activity-input decimal %q: %v", raw, err)
	}

	return value
}

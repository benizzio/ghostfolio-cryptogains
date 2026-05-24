// Package calculate verifies package-local activity-input helper branches that
// are easier to exercise directly than through black-box report runs.
// Authored by: OpenCode
package calculate

import (
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
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

// TestActivityInputTierHelpersCoverRemainingBranches verifies direct helper
// branches for explicit-currency tier derivation and guard behavior.
// Authored by: OpenCode
func TestActivityInputTierHelpersCoverRemainingBranches(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when an explicit tier has neither gross value nor unit price", func(t *testing.T) {
		var record = validActivityInputRecord(t)
		var tier = activityMoneyTier{
			name:         reportmodel.SelectedCurrencyContextBase,
			currencyCode: "USD",
			feeAmount:    activityInputDecimalPointer(t, "1"),
			hasAnyValue:  true,
		}

		var grossValue, err = selectTierGrossValue(record, tier)
		if err != nil {
			t.Fatalf("select tier gross value: %v", err)
		}
		if grossValue != nil {
			t.Fatalf("expected missing explicit tier gross value to stay nil, got %v", grossValue)
		}

		_, err = selectExplicitCurrencyTier(record, tier)
		if err == nil || !strings.Contains(err.Error(), `activity "activity-1" base currency context is incomplete`) {
			t.Fatalf("expected incomplete explicit tier failure, got %v", err)
		}
	})

	t.Run("surfaces gross-value derivation failure from an explicit tier", func(t *testing.T) {
		var invalid apd.Decimal
		invalid.Form = apd.Infinite

		var record = validActivityInputRecord(t)
		_, err := selectExplicitCurrencyTier(record, activityMoneyTier{
			name:         reportmodel.SelectedCurrencyContextOrder,
			currencyCode: "USD",
			unitPrice:    &invalid,
			feeAmount:    activityInputDecimalPointer(t, "0"),
			hasAnyValue:  true,
		})
		if err == nil || !strings.Contains(err.Error(), `activity "activity-1" order gross value cannot be derived from quantity and unit price`) {
			t.Fatalf("expected gross-value derivation failure, got %v", err)
		}
	})

	t.Run("returns nil unit price when gross value is absent", func(t *testing.T) {
		var unitPrice, err = selectTierUnitPrice(validActivityInputRecord(t), activityMoneyTier{
			name:         reportmodel.SelectedCurrencyContextAssetProfile,
			currencyCode: "EUR",
		}, nil)
		if err != nil {
			t.Fatalf("select tier unit price: %v", err)
		}
		if unitPrice != nil {
			t.Fatalf("expected absent gross value to keep unit price nil, got %v", unitPrice)
		}
	})

	t.Run("surfaces unit-price derivation failure from an explicit tier", func(t *testing.T) {
		var invalid apd.Decimal
		invalid.Form = apd.Infinite

		var record = validActivityInputRecord(t)
		_, err := selectExplicitCurrencyTier(record, activityMoneyTier{
			name:         reportmodel.SelectedCurrencyContextBase,
			currencyCode: "USD",
			grossValue:   &invalid,
			feeAmount:    activityInputDecimalPointer(t, "0"),
			hasAnyValue:  true,
		})
		if err == nil || !strings.Contains(err.Error(), `activity "activity-1" base unit price cannot be derived from gross value and quantity`) {
			t.Fatalf("expected explicit-tier unit-price derivation failure, got %v", err)
		}
	})

	t.Run("surfaces direct unit-price derivation failure", func(t *testing.T) {
		var invalid apd.Decimal
		invalid.Form = apd.Infinite

		_, err := selectTierUnitPrice(validActivityInputRecord(t), activityMoneyTier{
			name:         reportmodel.SelectedCurrencyContextOrder,
			currencyCode: "USD",
		}, &invalid)
		if err == nil || !strings.Contains(err.Error(), `activity "activity-1" order unit price cannot be derived from gross value and quantity`) {
			t.Fatalf("expected direct unit-price derivation failure, got %v", err)
		}
	})

	t.Run("covers informed gross-value guard and success branches", func(t *testing.T) {
		if value := informedGrossValue(activityInputDecimalPointer(t, "2"), "", mustActivityInputDecimal(t, "3")); value != nil {
			t.Fatalf("expected uninformed tier gross value to stay nil, got %v", value)
		}

		var value = informedGrossValue(activityInputDecimalPointer(t, "2"), "USD", mustActivityInputDecimal(t, "3"))
		if value == nil {
			t.Fatalf("expected informed gross-value derivation to succeed")
		}
		var canonical, err = decimalsupport.CanonicalString(*value)
		if err != nil {
			t.Fatalf("canonicalize derived gross value: %v", err)
		}
		if canonical != "6" {
			t.Fatalf("unexpected informed gross-value derivation: got %q want %q", canonical, "6")
		}
	})
}

// TestZeroPricedHoldingReductionHelpers verifies the direct helper branches for
// explained zero-priced holding-reduction detection and preserved zero values.
// Authored by: OpenCode
func TestZeroPricedHoldingReductionHelpers(t *testing.T) {
	t.Parallel()

	t.Run("ignores non-sell and missing-comment rows", func(t *testing.T) {
		var buyRecord = validActivityInputRecord(t)
		if values, ok, err := selectZeroPricedHoldingReductionValues(buyRecord); err != nil || ok || values != (zeroPricedHoldingReductionValues{}) {
			t.Fatalf("expected BUY row to skip zero-priced handling, got values=%#v ok=%v err=%v", values, ok, err)
		}

		var sellWithoutComment = validActivityInputRecord(t)
		sellWithoutComment.ActivityType = syncmodel.ActivityTypeSell
		if values, ok, err := selectZeroPricedHoldingReductionValues(sellWithoutComment); err != nil || ok || values != (zeroPricedHoldingReductionValues{}) {
			t.Fatalf("expected SELL without comment to skip zero-priced handling, got values=%#v ok=%v err=%v", values, ok, err)
		}
	})

	t.Run("rejects non-zero preserved monetary values", func(t *testing.T) {
		var record = validActivityInputRecord(t)
		record.ActivityType = syncmodel.ActivityTypeSell
		record.Comment = "manual transfer"
		record.OrderUnitPrice = activityInputDecimalPointer(t, "1")

		values, ok, err := selectZeroPricedHoldingReductionValues(record)
		if err != nil {
			t.Fatalf("select zero-priced holding reduction values: %v", err)
		}
		if ok || values != (zeroPricedHoldingReductionValues{}) {
			t.Fatalf("expected non-zero preserved value to prevent zero-priced classification, got values=%#v ok=%v", values, ok)
		}
	})

	t.Run("rejects invalid preserved zero-priced values through helper and selector", func(t *testing.T) {
		var invalid apd.Decimal
		invalid.Form = apd.NaNSignaling

		var record = validActivityInputRecord(t)
		record.ActivityType = syncmodel.ActivityTypeSell
		record.Comment = "manual transfer"
		record.OrderUnitPrice = &invalid

		if _, ok, err := selectZeroPricedHoldingReductionValues(record); err == nil || ok || !strings.Contains(err.Error(), `activity "activity-1" zero-priced holding reduction values are invalid`) {
			t.Fatalf("expected invalid preserved zero-priced helper error, got ok=%v err=%v", ok, err)
		}
		if _, err := SelectActivityCalculationInput(record); err == nil || !strings.Contains(err.Error(), `activity "activity-1" zero-priced holding reduction values are invalid`) {
			t.Fatalf("expected selector to propagate zero-priced helper failure, got %v", err)
		}
	})

	t.Run("preserves first available explicit zero values", func(t *testing.T) {
		var record = validActivityInputRecord(t)
		record.ActivityType = syncmodel.ActivityTypeSell
		record.Comment = "manual transfer"
		record.OrderUnitPrice = activityInputDecimalPointer(t, "0")
		record.OrderGrossValue = activityInputDecimalPointer(t, "0")
		record.OrderFeeAmount = activityInputDecimalPointer(t, "0")
		record.AssetProfileUnitPrice = activityInputDecimalPointer(t, "0")
		record.BaseGrossValue = activityInputDecimalPointer(t, "0")
		record.BaseFeeAmount = activityInputDecimalPointer(t, "0")

		values, ok, err := selectZeroPricedHoldingReductionValues(record)
		if err != nil {
			t.Fatalf("select zero-priced holding reduction values: %v", err)
		}
		if !ok {
			t.Fatalf("expected explained zero-valued SELL row to classify as zero-priced holding reduction")
		}
		if values.unitPrice != record.OrderUnitPrice || values.grossValue != record.OrderGrossValue || values.feeAmount != record.OrderFeeAmount {
			t.Fatalf("expected first explicit zero pointers to be preserved, got %#v", values)
		}
	})

	t.Run("covers direct zero helpers", func(t *testing.T) {
		var zero = activityInputDecimalPointer(t, "0")
		var nonZero = activityInputDecimalPointer(t, "1")
		var invalid apd.Decimal
		invalid.Form = apd.Infinite

		allZero, err := allPresentDecimalsAreZero([]*apd.Decimal{nil, zero})
		if err != nil || !allZero {
			t.Fatalf("expected nil and zero values to count as all zero, got allZero=%v err=%v", allZero, err)
		}
		allZero, err = allPresentDecimalsAreZero([]*apd.Decimal{zero, nonZero})
		if err != nil || allZero {
			t.Fatalf("expected non-zero value to break all-zero detection, got allZero=%v err=%v", allZero, err)
		}
		if _, err = allPresentDecimalsAreZero([]*apd.Decimal{&invalid}); err == nil {
			t.Fatalf("expected invalid decimal to fail zero detection")
		}

		if got := firstExplicitZeroValue(nil, zero, nonZero); got != zero {
			t.Fatalf("expected first explicit zero pointer, got %v want %v", got, zero)
		}
		if got := firstExplicitZeroValue(nil, nil); got != nil {
			t.Fatalf("expected nil when no explicit zero pointers exist, got %v", got)
		}
	})
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

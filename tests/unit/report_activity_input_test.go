// Package unit verifies focused report-calculation seams without the full
// yearly report runtime.
// Authored by: OpenCode
package unit

import (
	"strings"
	"testing"

	reportcalculate "github.com/benizzio/ghostfolio-cryptogains/internal/report/calculate"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

// TestSelectActivityCalculationInputUsesOrderTierBeforeLowerTiers verifies the
// strict tier priority and selected currency preservation for priced rows.
// Authored by: OpenCode
func TestSelectActivityCalculationInputUsesOrderTierBeforeLowerTiers(t *testing.T) {
	t.Parallel()

	var record = pricedActivityRecord()
	record.OrderCurrency = "USD"
	record.OrderGrossValue = decimalPointer(t, "100")
	record.OrderFeeAmount = decimalPointer(t, "1")
	record.OrderUnitPrice = decimalPointer(t, "10")
	record.AssetProfileCurrency = "EUR"
	record.AssetProfileUnitPrice = decimalPointer(t, "11")
	record.AssetProfileFeeAmount = decimalPointer(t, "2")
	record.BaseCurrency = "CHF"
	record.BaseGrossValue = decimalPointer(t, "120")
	record.BaseFeeAmount = decimalPointer(t, "3")

	input, err := reportcalculate.SelectActivityCalculationInput(record)
	if err != nil {
		t.Fatalf("select activity input: %v", err)
	}

	assertSelectedContext(t, input, reportmodel.SelectedCurrencyContextOrder, "USD", "100", "1", "10")
	if input.DisplayLabel != "BTC" {
		t.Fatalf("unexpected display label: %q", input.DisplayLabel)
	}
}

// TestSelectActivityCalculationInputUsesAssetProfileTierWithoutMixing verifies
// complete asset-profile selection when the higher tier is absent.
// Authored by: OpenCode
func TestSelectActivityCalculationInputUsesAssetProfileTierWithoutMixing(t *testing.T) {
	t.Parallel()

	var record = pricedActivityRecord()
	record.OrderCurrency = ""
	record.AssetProfileCurrency = "EUR"
	record.AssetProfileUnitPrice = decimalPointer(t, "12.5")
	record.AssetProfileFeeAmount = decimalPointer(t, "0")
	record.BaseCurrency = "USD"
	record.BaseGrossValue = decimalPointer(t, "999")
	record.BaseFeeAmount = decimalPointer(t, "9")

	input, err := reportcalculate.SelectActivityCalculationInput(record)
	if err != nil {
		t.Fatalf("select activity input: %v", err)
	}

	assertSelectedContext(t, input, reportmodel.SelectedCurrencyContextAssetProfile, "EUR", "125", "0", "12.5")
}

// TestSelectActivityCalculationInputSkipsCurrencylessHigherTier verifies that a
// higher-priority tier with financial values but no explicit currency is
// skipped before completeness validation.
// Authored by: OpenCode
func TestSelectActivityCalculationInputSkipsCurrencylessHigherTier(t *testing.T) {
	t.Parallel()

	var record = pricedActivityRecord()
	record.OrderCurrency = ""
	record.OrderGrossValue = decimalPointer(t, "100")
	record.OrderFeeAmount = decimalPointer(t, "1")
	record.OrderUnitPrice = decimalPointer(t, "10")
	record.AssetProfileCurrency = "EUR"
	record.AssetProfileUnitPrice = decimalPointer(t, "12.5")
	record.AssetProfileFeeAmount = decimalPointer(t, "0")

	input, err := reportcalculate.SelectActivityCalculationInput(record)
	if err != nil {
		t.Fatalf("select activity input: %v", err)
	}

	assertSelectedContext(t, input, reportmodel.SelectedCurrencyContextAssetProfile, "EUR", "125", "0", "12.5")
}

// TestSelectActivityCalculationInputUsesBaseTierAndDerivesExactUnitPrice
// verifies exact unit-price derivation from a complete base tier.
// Authored by: OpenCode
func TestSelectActivityCalculationInputUsesBaseTierAndDerivesExactUnitPrice(t *testing.T) {
	t.Parallel()

	var record = pricedActivityRecord()
	record.OrderCurrency = ""
	record.AssetProfileCurrency = ""
	record.BaseCurrency = "GBP"
	record.BaseGrossValue = decimalPointer(t, "150")
	record.BaseFeeAmount = decimalPointer(t, "5")

	input, err := reportcalculate.SelectActivityCalculationInput(record)
	if err != nil {
		t.Fatalf("select activity input: %v", err)
	}

	assertSelectedContext(t, input, reportmodel.SelectedCurrencyContextBase, "GBP", "150", "5", "15")
}

// TestSelectActivityCalculationInputSkipsIncompleteHigherTier verifies that an
// unusable explicit-currency tier does not block a later complete tier.
// Authored by: OpenCode
func TestSelectActivityCalculationInputSkipsIncompleteHigherTier(t *testing.T) {
	t.Parallel()

	var record = pricedActivityRecord()
	record.OrderCurrency = "USD"
	record.OrderGrossValue = decimalPointer(t, "100")
	record.AssetProfileCurrency = "EUR"
	record.AssetProfileUnitPrice = decimalPointer(t, "10")
	record.AssetProfileFeeAmount = decimalPointer(t, "1")

	input, err := reportcalculate.SelectActivityCalculationInput(record)
	if err != nil {
		t.Fatalf("select activity input: %v", err)
	}

	assertSelectedContext(t, input, reportmodel.SelectedCurrencyContextAssetProfile, "EUR", "100", "1", "10")
}

// TestSelectActivityCalculationInputRejectsMissingFeeInsteadOfTreatingZero
// verifies that explicit zero is valid but missing fee is not.
// Authored by: OpenCode
func TestSelectActivityCalculationInputRejectsMissingFeeInsteadOfTreatingZero(t *testing.T) {
	t.Parallel()

	var record = pricedActivityRecord()
	record.BaseCurrency = "USD"
	record.BaseGrossValue = decimalPointer(t, "100")

	_, err := reportcalculate.SelectActivityCalculationInput(record)
	if err == nil {
		t.Fatalf("expected missing fee to fail")
	}
	if !strings.Contains(err.Error(), "base currency context is incomplete") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSelectActivityCalculationInputSkipsNonDerivableHigherTier verifies that a
// later explicit-currency tier may still satisfy the input when an earlier
// explicit-currency tier cannot derive one exact unit price safely.
// Authored by: OpenCode
func TestSelectActivityCalculationInputSkipsNonDerivableHigherTier(t *testing.T) {
	t.Parallel()

	var record = pricedActivityRecord()
	record.Quantity = mustDecimal(t, "3")
	record.OrderCurrency = "USD"
	record.OrderGrossValue = decimalPointer(t, "1")
	record.OrderFeeAmount = decimalPointer(t, "0")
	record.BaseCurrency = "GBP"
	record.BaseGrossValue = decimalPointer(t, "300")
	record.BaseFeeAmount = decimalPointer(t, "0")

	input, err := reportcalculate.SelectActivityCalculationInput(record)
	if err != nil {
		t.Fatalf("select activity input: %v", err)
	}

	assertSelectedContext(t, input, reportmodel.SelectedCurrencyContextBase, "GBP", "300", "0", "100")
}

// TestSelectActivityCalculationInputRejectsNonPositivePricedQuantity verifies
// priced-row quantity validation.
// Authored by: OpenCode
func TestSelectActivityCalculationInputRejectsNonPositivePricedQuantity(t *testing.T) {
	t.Parallel()

	var zeroQuantityRecord = pricedActivityRecord()
	zeroQuantityRecord.Quantity = mustDecimal(t, "0")
	zeroQuantityRecord.BaseCurrency = "USD"
	zeroQuantityRecord.BaseGrossValue = decimalPointer(t, "100")
	zeroQuantityRecord.BaseFeeAmount = decimalPointer(t, "1")

	if _, err := reportcalculate.SelectActivityCalculationInput(zeroQuantityRecord); err == nil || !strings.Contains(err.Error(), "must be greater than zero") {
		t.Fatalf("expected zero quantity failure, got %v", err)
	}

	var negativeQuantityRecord = pricedActivityRecord()
	negativeQuantityRecord.Quantity = mustDecimal(t, "-1")
	negativeQuantityRecord.BaseCurrency = "USD"
	negativeQuantityRecord.BaseGrossValue = decimalPointer(t, "100")
	negativeQuantityRecord.BaseFeeAmount = decimalPointer(t, "1")

	if _, err := reportcalculate.SelectActivityCalculationInput(negativeQuantityRecord); err == nil || !strings.Contains(err.Error(), "must be greater than zero") {
		t.Fatalf("expected negative quantity failure, got %v", err)
	}
}

// TestSelectActivityCalculationInputRejectsNonTerminatingUnitPriceDerivation
// verifies failure when exact division does not terminate.
// Authored by: OpenCode
func TestSelectActivityCalculationInputRejectsNonTerminatingUnitPriceDerivation(t *testing.T) {
	t.Parallel()

	var record = pricedActivityRecord()
	record.Quantity = mustDecimal(t, "3")
	record.BaseCurrency = "USD"
	record.BaseGrossValue = decimalPointer(t, "1")
	record.BaseFeeAmount = decimalPointer(t, "0")

	_, err := reportcalculate.SelectActivityCalculationInput(record)
	if err == nil {
		t.Fatalf("expected non-terminating unit-price derivation to fail")
	}
	if !strings.Contains(err.Error(), "cannot be derived exactly") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSelectActivityCalculationInputPrefersGrossValueDerivation verifies that a
// selected tier may derive gross value by multiplication before considering any
// division-based fallback or lower-priority tier.
// Authored by: OpenCode
func TestSelectActivityCalculationInputPrefersGrossValueDerivation(t *testing.T) {
	t.Parallel()

	var record = pricedActivityRecord()
	record.OrderCurrency = "USD"
	record.OrderUnitPrice = decimalPointer(t, "12.5")
	record.OrderFeeAmount = decimalPointer(t, "0")
	record.BaseCurrency = "GBP"
	record.BaseGrossValue = decimalPointer(t, "999")
	record.BaseFeeAmount = decimalPointer(t, "9")

	input, err := reportcalculate.SelectActivityCalculationInput(record)
	if err != nil {
		t.Fatalf("select activity input: %v", err)
	}

	assertSelectedContext(t, input, reportmodel.SelectedCurrencyContextOrder, "USD", "125", "0", "12.5")
}

// TestSelectActivityCalculationInputCreatesZeroPricedHoldingReduction verifies
// that explained zero-priced SELL rows carry no selected monetary context.
// Authored by: OpenCode
func TestSelectActivityCalculationInputCreatesZeroPricedHoldingReduction(t *testing.T) {
	t.Parallel()

	var record = pricedActivityRecord()
	record.ActivityType = syncmodel.ActivityTypeSell
	record.Comment = "manual custody transfer"

	input, err := reportcalculate.SelectActivityCalculationInput(record)
	if err != nil {
		t.Fatalf("select activity input: %v", err)
	}
	if !input.IsZeroPricedHoldingReduction {
		t.Fatalf("expected zero-priced holding reduction")
	}
	if input.GrossValue != nil || input.FeeAmount != nil || input.UnitPrice != nil {
		t.Fatalf("expected no monetary values for zero-priced holding reduction")
	}
	if input.SelectedCurrencyContext != "" || input.SelectedCurrencyCode != "" {
		t.Fatalf("expected no selected currency context for zero-priced holding reduction")
	}
}

// TestSelectActivityCalculationInputPreservesExplicitZeroValuedHoldingReductionFields
// verifies that production-shaped zero-priced SELL rows keep explicit zero-
// valued source details without becoming priced inputs.
// Authored by: OpenCode
func TestSelectActivityCalculationInputPreservesExplicitZeroValuedHoldingReductionFields(t *testing.T) {
	t.Parallel()

	var record = pricedActivityRecord()
	record.ActivityType = syncmodel.ActivityTypeSell
	record.Comment = "manual custody transfer"
	record.OrderCurrency = "USD"
	record.OrderUnitPrice = decimalPointer(t, "0")
	record.OrderGrossValue = decimalPointer(t, "0")
	record.OrderFeeAmount = decimalPointer(t, "0")

	input, err := reportcalculate.SelectActivityCalculationInput(record)
	if err != nil {
		t.Fatalf("select activity input: %v", err)
	}
	if !input.IsZeroPricedHoldingReduction {
		t.Fatalf("expected zero-priced holding reduction")
	}
	assertDecimalPointerString(t, input.UnitPrice, "0", "unit price")
	assertDecimalPointerString(t, input.GrossValue, "0", "gross value")
	assertDecimalPointerString(t, input.FeeAmount, "0", "fee amount")
	if input.SelectedCurrencyContext != "" || input.SelectedCurrencyCode != "" {
		t.Fatalf("expected no selected currency context for zero-priced holding reduction")
	}
}

// TestSelectActivityCalculationInputPreservesExplicitZeroFee verifies that an
// explicit fee value of zero remains selected.
// Authored by: OpenCode
func TestSelectActivityCalculationInputPreservesExplicitZeroFee(t *testing.T) {
	t.Parallel()

	var record = pricedActivityRecord()
	record.ActivityType = syncmodel.ActivityTypeSell
	record.BaseCurrency = "JPY"
	record.BaseGrossValue = decimalPointer(t, "2400")
	record.BaseFeeAmount = decimalPointer(t, "0")

	input, err := reportcalculate.SelectActivityCalculationInput(record)
	if err != nil {
		t.Fatalf("select activity input: %v", err)
	}

	assertSelectedContext(t, input, reportmodel.SelectedCurrencyContextBase, "JPY", "2400", "0", "240")
}

// pricedActivityRecord creates one minimal normalized activity fixture for the
// report activity-input seam.
// Authored by: OpenCode
func pricedActivityRecord() syncmodel.ActivityRecord {
	return syncmodel.ActivityRecord{
		SourceID:         "activity-1",
		OccurredAt:       "2024-02-01T10:11:12+02:00",
		ActivityType:     syncmodel.ActivityTypeBuy,
		AssetIdentityKey: "asset-btc",
		AssetSymbol:      "BTC",
		AssetName:        "Bitcoin",
		Quantity:         mustDecimal(nil, "10"),
	}
}

// assertSelectedContext verifies the selected priced activity currency context.
// Authored by: OpenCode
func assertSelectedContext(
	t *testing.T,
	input reportmodel.ActivityCalculationInput,
	context reportmodel.SelectedCurrencyContext,
	currency string,
	grossValue string,
	feeAmount string,
	unitPrice string,
) {
	t.Helper()

	if input.SelectedCurrencyContext != context {
		t.Fatalf("unexpected selected currency context: got %q want %q", input.SelectedCurrencyContext, context)
	}
	if input.SelectedCurrencyCode != currency {
		t.Fatalf("unexpected selected currency code: got %q want %q", input.SelectedCurrencyCode, currency)
	}
	assertDecimalPointerString(t, input.GrossValue, grossValue, "gross value")
	assertDecimalPointerString(t, input.FeeAmount, feeAmount, "fee amount")
	assertDecimalPointerString(t, input.UnitPrice, unitPrice, "unit price")
}

// assertDecimalPointerString compares one decimal pointer against a canonical
// decimal string expectation.
// Authored by: OpenCode
func assertDecimalPointerString(t *testing.T, value *apd.Decimal, want string, label string) {
	t.Helper()

	if value == nil {
		t.Fatalf("expected %s to be present", label)
	}

	var canonical, err = decimalsupport.CanonicalString(*value)
	if err != nil {
		t.Fatalf("canonicalize %s: %v", label, err)
	}
	if canonical != want {
		t.Fatalf("unexpected %s: got %q want %q", label, canonical, want)
	}
}

// decimalPointer parses one decimal pointer for activity-input tests.
// Authored by: OpenCode
func decimalPointer(t *testing.T, raw string) *apd.Decimal {
	t.Helper()

	var value = mustDecimal(t, raw)
	return &value
}

// mustDecimal parses one exact decimal test fixture.
// Authored by: OpenCode
func mustDecimal(t *testing.T, raw string) apd.Decimal {
	if t != nil {
		t.Helper()
	}

	value, _, err := decimalsupport.ParseString(raw)
	if err != nil {
		if t == nil {
			panic(err)
		}
		t.Fatalf("parse decimal %q: %v", raw, err)
	}

	return value
}

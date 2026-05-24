package model

import (
	"strings"
	"testing"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// TestResolveActivityAmountsCoversRemainingBranches verifies the remaining
// exact-currency amount selection and derivation branches.
// Authored by: OpenCode
func TestResolveActivityAmountsCoversRemainingBranches(t *testing.T) {
	t.Parallel()

	t.Run("resolve derives unit price from base gross value", func(t *testing.T) {
		record := ActivityRecord{
			SourceID:       "base-success",
			Quantity:       mustActivityAmountDecimal(t, "2"),
			BaseCurrency:   "USD",
			BaseGrossValue: mustActivityAmountDecimalPointer(t, "10"),
		}

		amounts, err := ResolveActivityAmounts(record)
		if err != nil {
			t.Fatalf("resolve activity amounts: %v", err)
		}
		if amounts.UnitPriceCurrency != "USD" || amounts.GrossValueCurrency != "USD" {
			t.Fatalf("expected base currency-derived amounts, got %#v", amounts)
		}
		if got, err := decimalsupport.CanonicalStringPointer(amounts.UnitPrice); err != nil || got != "5" {
			t.Fatalf("expected derived unit price of 5, got %q err=%v", got, err)
		}
		if got, err := decimalsupport.CanonicalStringPointer(amounts.GrossValue); err != nil || got != "10" {
			t.Fatalf("expected preserved gross value of 10, got %q err=%v", got, err)
		}
	})

	t.Run("resolve returns unit price derivation error", func(t *testing.T) {
		amounts, err := ResolveActivityAmounts(ActivityRecord{
			SourceID:       "unit-inexact",
			Quantity:       mustActivityAmountDecimal(t, "3"),
			BaseCurrency:   "USD",
			BaseGrossValue: mustActivityAmountDecimalPointer(t, "10"),
		})
		if err != nil {
			t.Fatalf("expected repeating unit-price derivation success, got %v", err)
		}
		if amounts.UnitPriceCurrency != "USD" {
			t.Fatalf("expected derived unit price currency USD, got %#v", amounts)
		}
		if got, canonicalErr := decimalsupport.CanonicalStringPointer(amounts.UnitPrice); canonicalErr != nil || got != "3.3333333333333333" {
			t.Fatalf("expected rounded repeating unit price, got %q err=%v", got, canonicalErr)
		}
	})

	t.Run("resolve wraps unit price derivation failure", func(t *testing.T) {
		_, err := ResolveActivityAmounts(ActivityRecord{
			SourceID:       "unit-zero-divisor",
			Quantity:       mustActivityAmountDecimal(t, "0"),
			BaseCurrency:   "USD",
			BaseGrossValue: mustActivityAmountDecimalPointer(t, "10"),
		})
		if err == nil || !strings.Contains(err.Error(), `activity "unit-zero-divisor" unit price basis input is invalid`) {
			t.Fatalf("expected wrapped unit-price derivation error, got %v", err)
		}
	})

	t.Run("resolve returns gross value error", func(t *testing.T) {
		_, err := ResolveActivityAmounts(ActivityRecord{
			SourceID: "gross-required",
			Quantity: mustActivityAmountDecimal(t, "1"),
		})
		if err == nil || err.Error() != `activity "gross-required" gross value basis input is required` {
			t.Fatalf("expected required gross-value error, got %v", err)
		}
	})

	t.Run("resolve returns fee currency-context error", func(t *testing.T) {
		_, err := ResolveActivityAmounts(ActivityRecord{
			SourceID:       "fee-uninformed",
			Quantity:       mustActivityAmountDecimal(t, "2"),
			BaseCurrency:   "USD",
			BaseGrossValue: mustActivityAmountDecimalPointer(t, "10"),
			OrderFeeAmount: mustActivityAmountDecimalPointer(t, "1"),
		})
		if err == nil || !strings.Contains(err.Error(), `activity "fee-uninformed" fee amount currency context is uninformed`) {
			t.Fatalf("expected fee currency-context error, got %v", err)
		}
	})

	t.Run("unit price helper covers direct helper-only guards", func(t *testing.T) {
		unitPrice, currency, err := resolveUnitPrice(ActivityRecord{
			SourceID:       "unit-order",
			OrderCurrency:  "CHF",
			OrderUnitPrice: mustActivityAmountDecimalPointer(t, "5"),
		}, nil, "")
		if err != nil || currency != "CHF" {
			t.Fatalf("expected order unit-price success, got unitPrice=%#v currency=%q err=%v", unitPrice, currency, err)
		}
		if got, canonicalErr := decimalsupport.CanonicalStringPointer(unitPrice); canonicalErr != nil || got != "5" {
			t.Fatalf("expected preserved order unit price, got %q err=%v", got, canonicalErr)
		}

		unitPrice, currency, err = resolveUnitPrice(ActivityRecord{
			SourceID:              "unit-asset",
			AssetProfileCurrency:  "EUR",
			AssetProfileUnitPrice: mustActivityAmountDecimalPointer(t, "7"),
		}, nil, "")
		if err != nil || currency != "EUR" {
			t.Fatalf("expected asset-profile unit-price success, got unitPrice=%#v currency=%q err=%v", unitPrice, currency, err)
		}
		if got, canonicalErr := decimalsupport.CanonicalStringPointer(unitPrice); canonicalErr != nil || got != "7" {
			t.Fatalf("expected preserved asset-profile unit price, got %q err=%v", got, canonicalErr)
		}

		_, _, err = resolveUnitPrice(ActivityRecord{SourceID: "unit-uninformed", OrderUnitPrice: mustActivityAmountDecimalPointer(t, "1")}, nil, "")
		if err == nil || !strings.Contains(err.Error(), `activity "unit-uninformed" unit price currency context is uninformed`) {
			t.Fatalf("expected unit-price currency-context error, got %v", err)
		}

		_, _, err = resolveUnitPrice(ActivityRecord{SourceID: "unit-required"}, nil, "")
		if err == nil || err.Error() != `activity "unit-required" unit price basis input is required` {
			t.Fatalf("expected required unit-price error, got %v", err)
		}

		var grossValue = mustActivityAmountDecimal(t, "10")
		_, _, err = resolveUnitPrice(ActivityRecord{SourceID: "unit-missing-currency"}, &grossValue, "")
		if err == nil || err.Error() != `activity "unit-missing-currency" unit price basis input is required` {
			t.Fatalf("expected direct helper fallback error, got %v", err)
		}

		_, _, err = resolveUnitPrice(ActivityRecord{SourceID: "unit-helper-zero", Quantity: mustActivityAmountDecimal(t, "0")}, &grossValue, "USD")
		if err == nil || !strings.Contains(err.Error(), `activity "unit-helper-zero" unit price basis input is invalid`) {
			t.Fatalf("expected helper derivation error, got %v", err)
		}

		var rounded apd.Decimal
		rounded, err = divideActivityAmountRoundHalfUp(mustActivityAmountDecimal(t, "1"), mustActivityAmountDecimal(t, "3"))
		if err != nil {
			t.Fatalf("expected rounded helper success, got %v", err)
		}
		if got, canonicalErr := decimalsupport.CanonicalString(rounded); canonicalErr != nil || got != "0.3333333333333333" {
			t.Fatalf("expected rounded helper quotient, got %q err=%v", got, canonicalErr)
		}

		rounded, err = divideActivityAmountRoundHalfUp(mustActivityAmountDecimal(t, "1"), mustActivityAmountDecimal(t, "6"))
		if err != nil {
			t.Fatalf("expected half-up rounded helper success, got %v", err)
		}
		if got, canonicalErr := decimalsupport.CanonicalString(rounded); canonicalErr != nil || got != "0.1666666666666667" {
			t.Fatalf("expected half-up rounded helper quotient, got %q err=%v", got, canonicalErr)
		}

		rounded, err = divideActivityAmountRoundHalfUp(mustActivityAmountDecimal(t, "-1"), mustActivityAmountDecimal(t, "3"))
		if err != nil {
			t.Fatalf("expected negative rounded helper success, got %v", err)
		}
		if got, canonicalErr := decimalsupport.CanonicalString(rounded); canonicalErr != nil || got != "-0.3333333333333333" {
			t.Fatalf("expected negative rounded helper quotient, got %q err=%v", got, canonicalErr)
		}

		_, err = divideActivityAmountRoundHalfUp(invalidActivityAmountDecimal(), mustActivityAmountDecimal(t, "1"))
		if err == nil || !strings.Contains(err.Error(), "derive activity amount from gross value and quantity") {
			t.Fatalf("expected non-finite dividend to fail rounded helper, got %v", err)
		}

		_, err = divideActivityAmountRoundHalfUp(mustActivityAmountDecimal(t, "1"), invalidActivityAmountDecimal())
		if err == nil || !strings.Contains(err.Error(), "derive activity amount from gross value and quantity") {
			t.Fatalf("expected non-finite divisor to fail rounded helper, got %v", err)
		}

		_, err = divideActivityAmountRoundHalfUp(mustActivityAmountDecimal(t, "1"), mustActivityAmountDecimal(t, "0"))
		if err == nil || !strings.Contains(err.Error(), "non-zero divisor is required") {
			t.Fatalf("expected zero divisor to fail rounded helper, got %v", err)
		}

		var numerator, denominator = finiteActivityDecimalFraction(mustActivityAmountDecimal(t, "-1.25"))
		if numerator.String() != "-125" || denominator.String() != "100" {
			t.Fatalf("expected exact negative finite activity fraction, got %s/%s", numerator.String(), denominator.String())
		}
	})

	t.Run("gross value helper covers derived and error branches", func(t *testing.T) {
		grossValue, currency, err := resolveGrossValue(ActivityRecord{
			SourceID:        "order-preserved",
			Quantity:        mustActivityAmountDecimal(t, "1"),
			OrderCurrency:   "CHF",
			OrderGrossValue: mustActivityAmountDecimalPointer(t, "11"),
		})
		if err != nil {
			t.Fatalf("resolve preserved order gross value: %v", err)
		}
		if currency != "CHF" {
			t.Fatalf("expected CHF preserved gross value currency, got %q", currency)
		}
		if got, canonicalErr := decimalsupport.CanonicalStringPointer(grossValue); canonicalErr != nil || got != "11" {
			t.Fatalf("expected preserved order gross value of 11, got %q err=%v", got, canonicalErr)
		}

		grossValue, currency, err = resolveGrossValue(ActivityRecord{
			SourceID:       "order-derived",
			Quantity:       mustActivityAmountDecimal(t, "2"),
			OrderCurrency:  "CHF",
			OrderUnitPrice: mustActivityAmountDecimalPointer(t, "5"),
		})
		if err != nil {
			t.Fatalf("resolve order-derived gross value: %v", err)
		}
		if currency != "CHF" {
			t.Fatalf("expected CHF gross value currency, got %q", currency)
		}
		if got, err := decimalsupport.CanonicalStringPointer(grossValue); err != nil || got != "10" {
			t.Fatalf("expected derived gross value of 10, got %q err=%v", got, err)
		}

		grossValue, currency, err = resolveGrossValue(ActivityRecord{
			SourceID:              "asset-derived",
			Quantity:              mustActivityAmountDecimal(t, "2"),
			AssetProfileCurrency:  "EUR",
			AssetProfileUnitPrice: mustActivityAmountDecimalPointer(t, "5"),
		})
		if err != nil {
			t.Fatalf("resolve asset-derived gross value: %v", err)
		}
		if currency != "EUR" {
			t.Fatalf("expected EUR derived gross value currency, got %q", currency)
		}
		if got, canonicalErr := decimalsupport.CanonicalStringPointer(grossValue); canonicalErr != nil || got != "10" {
			t.Fatalf("expected derived asset-profile gross value of 10, got %q err=%v", got, canonicalErr)
		}

		_, _, err = resolveGrossValue(ActivityRecord{
			SourceID:       "order-invalid",
			Quantity:       invalidActivityAmountDecimal(),
			OrderCurrency:  "CHF",
			OrderUnitPrice: mustActivityAmountDecimalPointer(t, "5"),
		})
		if err == nil || !strings.Contains(err.Error(), `activity "order-invalid" gross value basis input is invalid`) {
			t.Fatalf("expected invalid order gross-value error, got %v", err)
		}

		_, _, err = resolveGrossValue(ActivityRecord{
			SourceID:              "asset-invalid",
			Quantity:              invalidActivityAmountDecimal(),
			AssetProfileCurrency:  "EUR",
			AssetProfileUnitPrice: mustActivityAmountDecimalPointer(t, "5"),
		})
		if err == nil || !strings.Contains(err.Error(), `activity "asset-invalid" gross value basis input is invalid`) {
			t.Fatalf("expected invalid asset-profile gross-value error, got %v", err)
		}

		grossValue, currency, err = resolveGrossValue(ActivityRecord{
			SourceID:       "base-derived",
			Quantity:       mustActivityAmountDecimal(t, "1"),
			BaseCurrency:   "USD",
			BaseGrossValue: mustActivityAmountDecimalPointer(t, "7"),
		})
		if err != nil {
			t.Fatalf("resolve base gross value: %v", err)
		}
		if currency != "USD" {
			t.Fatalf("expected USD gross value currency, got %q", currency)
		}
		if got, err := decimalsupport.CanonicalStringPointer(grossValue); err != nil || got != "7" {
			t.Fatalf("expected preserved base gross value of 7, got %q err=%v", got, err)
		}

		_, _, err = resolveGrossValue(ActivityRecord{
			SourceID:        "gross-uninformed",
			Quantity:        mustActivityAmountDecimal(t, "1"),
			OrderGrossValue: mustActivityAmountDecimalPointer(t, "7"),
		})
		if err == nil || !strings.Contains(err.Error(), `activity "gross-uninformed" gross value currency context is uninformed`) {
			t.Fatalf("expected gross-value currency-context error, got %v", err)
		}

		_, _, err = resolveGrossValue(ActivityRecord{SourceID: "gross-required-direct", Quantity: mustActivityAmountDecimal(t, "1")})
		if err == nil || err.Error() != `activity "gross-required-direct" gross value basis input is required` {
			t.Fatalf("expected direct helper gross-value required error, got %v", err)
		}
	})

	t.Run("fee helper covers uninformed fee branch", func(t *testing.T) {
		feeAmount, currency, err := resolveFeeAmount(ActivityRecord{SourceID: "fee-order", OrderCurrency: "CHF", OrderFeeAmount: mustActivityAmountDecimalPointer(t, "1")})
		if err != nil || currency != "CHF" {
			t.Fatalf("expected order-fee success, got amount=%#v currency=%q err=%v", feeAmount, currency, err)
		}

		feeAmount, currency, err = resolveFeeAmount(ActivityRecord{SourceID: "fee-asset", AssetProfileCurrency: "EUR", AssetProfileFeeAmount: mustActivityAmountDecimalPointer(t, "2")})
		if err != nil || currency != "EUR" {
			t.Fatalf("expected asset-profile fee success, got amount=%#v currency=%q err=%v", feeAmount, currency, err)
		}

		feeAmount, currency, err = resolveFeeAmount(ActivityRecord{SourceID: "fee-base", BaseCurrency: "USD", BaseFeeAmount: mustActivityAmountDecimalPointer(t, "3")})
		if err != nil || currency != "USD" {
			t.Fatalf("expected base fee success, got amount=%#v currency=%q err=%v", feeAmount, currency, err)
		}

		_, _, err = resolveFeeAmount(ActivityRecord{
			SourceID:       "fee-helper-uninformed",
			OrderFeeAmount: mustActivityAmountDecimalPointer(t, "1"),
		})
		if err == nil || !strings.Contains(err.Error(), `activity "fee-helper-uninformed" fee amount currency context is uninformed`) {
			t.Fatalf("expected helper fee currency-context error, got %v", err)
		}
	})
}

// mustActivityAmountDecimal parses one exact decimal fixture for activity-amount
// resolution tests.
// Authored by: OpenCode
func mustActivityAmountDecimal(t *testing.T, raw string) apd.Decimal {
	t.Helper()

	value, _, err := decimalsupport.ParseString(raw)
	if err != nil {
		t.Fatalf("parse activity amount decimal: %v", err)
	}

	return value
}

// mustActivityAmountDecimalPointer parses one exact decimal fixture pointer for
// activity-amount resolution tests.
// Authored by: OpenCode
func mustActivityAmountDecimalPointer(t *testing.T, raw string) *apd.Decimal {
	t.Helper()

	value := mustActivityAmountDecimal(t, raw)
	return &value
}

// invalidActivityAmountDecimal returns one non-finite decimal that forces apd
// multiplication traps in the direct helper tests.
// Authored by: OpenCode
func invalidActivityAmountDecimal() apd.Decimal {
	var value apd.Decimal
	value.Form = apd.NaNSignaling
	return value
}

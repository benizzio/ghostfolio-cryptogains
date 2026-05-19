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
		_, err := ResolveActivityAmounts(ActivityRecord{
			SourceID:       "unit-inexact",
			Quantity:       mustActivityAmountDecimal(t, "3"),
			BaseCurrency:   "USD",
			BaseGrossValue: mustActivityAmountDecimalPointer(t, "10"),
		})
		if err == nil || !strings.Contains(err.Error(), `activity "unit-inexact" unit price basis input is not exact`) {
			t.Fatalf("expected exact-division unit price failure, got %v", err)
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
		_, _, err := resolveUnitPrice(ActivityRecord{SourceID: "unit-uninformed", OrderUnitPrice: mustActivityAmountDecimalPointer(t, "1")}, nil, "")
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
	})

	t.Run("gross value helper covers derived and error branches", func(t *testing.T) {
		grossValue, currency, err := resolveGrossValue(ActivityRecord{
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

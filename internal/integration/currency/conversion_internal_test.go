// Package currency verifies exact source-to-base conversion behavior.
// Authored by: OpenCode
package currency

import "testing"

// TestConvertAmountUsesExactQuoteDirectionFormula verifies provider quote
// directions are applied without float math.
// Authored by: OpenCode
func TestConvertAmountUsesExactQuoteDirectionFormula(t *testing.T) {
	t.Parallel()

	var testCases = []struct {
		name          string
		amount        string
		rate          string
		direction     QuoteDirection
		wantConverted string
	}{
		{name: "ECB source per base divides", amount: "109.21", rate: "1.0921", direction: QuoteDirectionSourcePerBase, wantConverted: "100"},
		{name: "Federal Reserve unstarred source per base divides", amount: "1691.40", rate: "16.9140", direction: QuoteDirectionSourcePerBase, wantConverted: "100"},
		{name: "Federal Reserve starred base per source multiplies", amount: "100", rate: "1.0946", direction: QuoteDirectionBasePerSource, wantConverted: "109.46"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var converted, err = ConvertAmountToBase(mustCurrencyDecimal(t, testCase.amount), mustCurrencyDecimal(t, testCase.rate), testCase.direction)
			if err != nil {
				t.Fatalf("convert amount: %v", err)
			}
			assertCurrencyDecimalString(t, converted, testCase.wantConverted)
		})
	}
}

// TestConvertAmountBoundsRepeatingDivisionToSixteenDecimals verifies division
// results use the feature's 16-decimal round-half-up internal bound.
// Authored by: OpenCode
func TestConvertAmountBoundsRepeatingDivisionToSixteenDecimals(t *testing.T) {
	t.Parallel()

	var converted, err = ConvertAmountToBase(mustCurrencyDecimal(t, "1"), mustCurrencyDecimal(t, "3"), QuoteDirectionSourcePerBase)
	if err != nil {
		t.Fatalf("convert repeating division: %v", err)
	}
	assertCurrencyDecimalString(t, converted, "0.3333333333333333")
}

// TestConvertAmountRejectsInvalidInputs verifies conversion math fails before
// producing non-defensible monetary values.
// Authored by: OpenCode
func TestConvertAmountRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	var _, zeroRateErr = ConvertAmountToBase(mustCurrencyDecimal(t, "1"), mustCurrencyDecimal(t, "0"), QuoteDirectionSourcePerBase)
	if zeroRateErr == nil {
		t.Fatalf("expected zero-rate rejection")
	}

	var _, quoteErr = ConvertAmountToBase(mustCurrencyDecimal(t, "1"), mustCurrencyDecimal(t, "1"), QuoteDirection("ambiguous"))
	if quoteErr == nil {
		t.Fatalf("expected ambiguous quote direction rejection")
	}
}

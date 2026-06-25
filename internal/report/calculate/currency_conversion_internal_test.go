// Package calculate verifies exact report-domain source-to-base conversion behavior.
// Authored by: OpenCode
package calculate

import (
	"strings"
	"testing"

	currencyintegration "github.com/benizzio/ghostfolio-cryptogains/internal/integration/currency"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// TestConvertAmountUsesExactQuoteDirectionFormula verifies provider quote
// directions are applied by the report calculator without float math.
// Authored by: OpenCode
func TestConvertAmountUsesExactQuoteDirectionFormula(t *testing.T) {
	t.Parallel()

	var testCases = []struct {
		name          string
		amount        string
		rate          string
		direction     currencyintegration.QuoteDirection
		wantConverted string
	}{
		{name: "ECB source per base divides", amount: "109.21", rate: "1.0921", direction: currencyintegration.QuoteDirectionSourcePerBase, wantConverted: "100"},
		{name: "Federal Reserve unstarred source per base divides", amount: "1691.40", rate: "16.9140", direction: currencyintegration.QuoteDirectionSourcePerBase, wantConverted: "100"},
		{name: "Federal Reserve starred base per source multiplies", amount: "100", rate: "1.0946", direction: currencyintegration.QuoteDirectionBasePerSource, wantConverted: "109.46"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var converted, err = convertAmountToBase(mustReportDecimal(t, testCase.amount), mustReportDecimal(t, testCase.rate), testCase.direction)
			if err != nil {
				t.Fatalf("convert amount: %v", err)
			}
			assertCalculatedDecimalString(t, converted, testCase.wantConverted)
		})
	}
}

// TestConvertAmountBoundsRepeatingDivisionToSixteenDecimals verifies division
// results use the feature's 16-decimal round-half-up internal bound.
// Authored by: OpenCode
func TestConvertAmountBoundsRepeatingDivisionToSixteenDecimals(t *testing.T) {
	t.Parallel()

	var converted, err = convertAmountToBase(mustReportDecimal(t, "1"), mustReportDecimal(t, "3"), currencyintegration.QuoteDirectionSourcePerBase)
	if err != nil {
		t.Fatalf("convert repeating division: %v", err)
	}
	assertCalculatedDecimalString(t, converted, "0.3333333333333333")
}

// TestConvertAmountRejectsInvalidInputs verifies conversion math fails before
// producing non-defensible monetary values.
// Authored by: OpenCode
func TestConvertAmountRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	var invalidAmount apd.Decimal
	invalidAmount.Form = apd.Infinite
	var _, amountErr = convertAmountToBase(invalidAmount, mustReportDecimal(t, "1"), currencyintegration.QuoteDirectionSourcePerBase)
	if amountErr == nil || !strings.Contains(amountErr.Error(), "conversion amount is invalid") {
		t.Fatalf("expected invalid amount rejection, got %v", amountErr)
	}

	var _, zeroRateErr = convertAmountToBase(mustReportDecimal(t, "1"), mustReportDecimal(t, "0"), currencyintegration.QuoteDirectionSourcePerBase)
	if zeroRateErr == nil {
		t.Fatalf("expected zero-rate rejection")
	}

	var _, quoteErr = convertAmountToBase(mustReportDecimal(t, "1"), mustReportDecimal(t, "1"), currencyintegration.QuoteDirection("ambiguous"))
	if quoteErr == nil {
		t.Fatalf("expected ambiguous quote direction rejection")
	}
}

// assertCalculatedDecimalString verifies one canonical calculated decimal string.
// Authored by: OpenCode
func assertCalculatedDecimalString(t *testing.T, value apd.Decimal, expected string) {
	t.Helper()

	var actual, err = decimalsupport.CanonicalString(value)
	if err != nil {
		t.Fatalf("format decimal: %v", err)
	}
	if actual != expected {
		t.Fatalf("unexpected decimal: got %s want %s", actual, expected)
	}
}

// Package presentation tests ordered converted-amount presentation entries.
// Authored by: OpenCode
package presentation

import (
	"reflect"
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/cockroachdb/apd/v3"
)

// TestConvertedAmountsCoversCanonicalSubsequences verifies every order-preserving
// subsequence of the supported converted-amount kinds, including the empty
// subsequence and the three-entry subsequence.
// Authored by: OpenCode
func TestConvertedAmountsCoversCanonicalSubsequences(t *testing.T) {
	var cases = []struct {
		name          string
		includedKinds []reportmodel.ConvertedAmountKind
		want          []string
	}{
		{name: "empty", want: nil},
		{name: "unit price", includedKinds: []reportmodel.ConvertedAmountKind{reportmodel.ConvertedAmountKindUnitPrice}, want: []string{
			"unit_price: 10.01 -> 20.01",
		}},
		{name: "gross value", includedKinds: []reportmodel.ConvertedAmountKind{reportmodel.ConvertedAmountKindGrossValue}, want: []string{
			"gross_value: 30.00 -> 40.00",
		}},
		{name: "fee amount", includedKinds: []reportmodel.ConvertedAmountKind{reportmodel.ConvertedAmountKindFeeAmount}, want: []string{
			"fee_amount: 50.01 -> 60.01",
		}},
		{name: "unit price and gross value", includedKinds: []reportmodel.ConvertedAmountKind{reportmodel.ConvertedAmountKindUnitPrice, reportmodel.ConvertedAmountKindGrossValue}, want: []string{
			"unit_price: 10.01 -> 20.01",
			"gross_value: 30.00 -> 40.00",
		}},
		{name: "unit price and fee amount", includedKinds: []reportmodel.ConvertedAmountKind{reportmodel.ConvertedAmountKindUnitPrice, reportmodel.ConvertedAmountKindFeeAmount}, want: []string{
			"unit_price: 10.01 -> 20.01",
			"fee_amount: 50.01 -> 60.01",
		}},
		{name: "gross value and fee amount", includedKinds: []reportmodel.ConvertedAmountKind{reportmodel.ConvertedAmountKindGrossValue, reportmodel.ConvertedAmountKindFeeAmount}, want: []string{
			"gross_value: 30.00 -> 40.00",
			"fee_amount: 50.01 -> 60.01",
		}},
		{name: "unit price gross value and fee amount", includedKinds: []reportmodel.ConvertedAmountKind{reportmodel.ConvertedAmountKindUnitPrice, reportmodel.ConvertedAmountKindGrossValue, reportmodel.ConvertedAmountKindFeeAmount}, want: []string{
			"unit_price: 10.01 -> 20.01",
			"gross_value: 30.00 -> 40.00",
			"fee_amount: 50.01 -> 60.01",
		}},
	}

	for _, testCase := range cases {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			var amounts = canonicalConvertedAmounts(t)
			for index := range amounts {
				if !containsConvertedAmountKind(testCase.includedKinds, amounts[index].AmountKind) {
					amounts[index].OriginalAmount = mustFinancialDecimal(t, "0")
					amounts[index].ConvertedAmount = mustFinancialDecimal(t, "0")
				}
			}

			got, err := ConvertedAmounts(11, amounts)
			if err != nil {
				t.Fatalf("convert canonical subsequence: %v", err)
			}
			if !reflect.DeepEqual(got, testCase.want) {
				t.Fatalf("converted entries = %#v, want %#v", got, testCase.want)
			}
		})
	}
}

// TestConvertedAmountsOmitsOnlyExactZeroToZero verifies that an exact pair of
// zero amounts is omitted while a zero on either side remains an entry.
// Authored by: OpenCode
func TestConvertedAmountsOmitsOnlyExactZeroToZero(t *testing.T) {
	var amounts = []reportmodel.ConvertedActivityAmount{
		convertedAmount(t, reportmodel.ConvertedAmountKindUnitPrice, "0", "0"),
		convertedAmount(t, reportmodel.ConvertedAmountKindGrossValue, "0", "2"),
		convertedAmount(t, reportmodel.ConvertedAmountKindFeeAmount, "3", "0"),
	}

	got, err := ConvertedAmounts(12, amounts)
	if err != nil {
		t.Fatalf("convert zero-pair subsequence: %v", err)
	}
	var want = []string{
		"gross_value: 0.00 -> 2.00",
		"fee_amount: 3.00 -> 0.00",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("converted entries = %#v, want %#v", got, want)
	}
}

// TestConvertedAmountsIncludesNonZeroValuesDisplayedAsZero verifies exact
// non-zero amounts remain included when financial formatting displays 0.00.
// Authored by: OpenCode
func TestConvertedAmountsIncludesNonZeroValuesDisplayedAsZero(t *testing.T) {
	var amounts = []reportmodel.ConvertedActivityAmount{
		convertedAmount(t, reportmodel.ConvertedAmountKindUnitPrice, "0.004", "0.004"),
		convertedAmount(t, reportmodel.ConvertedAmountKindGrossValue, "0.004", "0"),
		convertedAmount(t, reportmodel.ConvertedAmountKindFeeAmount, "0", "0.004"),
	}

	got, err := ConvertedAmounts(13, amounts)
	if err != nil {
		t.Fatalf("convert sub-cent amounts: %v", err)
	}
	var want = []string{
		"unit_price: 0.00 -> 0.00",
		"gross_value: 0.00 -> 0.00",
		"fee_amount: 0.00 -> 0.00",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("converted entries = %#v, want %#v", got, want)
	}
}

// TestConvertedAmountsPreservesReceivedSupportedKindOrder verifies duplicate
// and non-canonical supported kinds are retained exactly as received.
// Authored by: OpenCode
func TestConvertedAmountsPreservesReceivedSupportedKindOrder(t *testing.T) {
	var amounts = []reportmodel.ConvertedActivityAmount{
		convertedAmount(t, reportmodel.ConvertedAmountKindFeeAmount, "1", "2"),
		convertedAmount(t, reportmodel.ConvertedAmountKindUnitPrice, "3", "4"),
		convertedAmount(t, reportmodel.ConvertedAmountKindFeeAmount, "5", "6"),
		convertedAmount(t, reportmodel.ConvertedAmountKindGrossValue, "7", "8"),
		convertedAmount(t, reportmodel.ConvertedAmountKindUnitPrice, "9", "10"),
	}

	got, err := ConvertedAmounts(14, amounts)
	if err != nil {
		t.Fatalf("convert received supported-kind order: %v", err)
	}
	var want = []string{
		"fee_amount: 1.00 -> 2.00",
		"unit_price: 3.00 -> 4.00",
		"fee_amount: 5.00 -> 6.00",
		"gross_value: 7.00 -> 8.00",
		"unit_price: 9.00 -> 10.00",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("converted entries = %#v, want received order %#v", got, want)
	}
}

// TestConvertedAmountsReturnsComponentErrors verifies original and converted
// component failures retain the conversion entry, amount, and component context.
// Authored by: OpenCode
func TestConvertedAmountsReturnsComponentErrors(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		configure func(*reportmodel.ConvertedActivityAmount)
		want      string
	}{
		{name: "original amount", configure: func(amount *reportmodel.ConvertedActivityAmount) {
			amount.OriginalAmount = nonFiniteConvertedAmountDecimal()
		}, want: "render conversion audit entry 15 amount 0 original amount"},
		{name: "converted amount", configure: func(amount *reportmodel.ConvertedActivityAmount) {
			amount.ConvertedAmount = nonFiniteConvertedAmountDecimal()
		}, want: "render conversion audit entry 15 amount 0 converted amount"},
	} {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			var amount = convertedAmount(t, reportmodel.ConvertedAmountKindUnitPrice, "1", "2")
			testCase.configure(&amount)

			got, err := ConvertedAmounts(15, []reportmodel.ConvertedActivityAmount{amount})
			if err == nil {
				t.Fatalf("converted entries = %#v without an error", got)
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %q, want context %q", err, testCase.want)
			}
		})
	}
}

// TestConvertedAmountsDoesNotMutateSourceSequence verifies conversion
// presentation reads the received amount records and decimal values without
// changing their order or contents.
// Authored by: OpenCode
func TestConvertedAmountsDoesNotMutateSourceSequence(t *testing.T) {
	var amounts = []reportmodel.ConvertedActivityAmount{
		convertedAmount(t, reportmodel.ConvertedAmountKindFeeAmount, "5.005", "6.006"),
		convertedAmount(t, reportmodel.ConvertedAmountKindUnitPrice, "0.004", "0.004"),
		convertedAmount(t, reportmodel.ConvertedAmountKindFeeAmount, "0", "0"),
		convertedAmount(t, reportmodel.ConvertedAmountKindGrossValue, "7.007", "8.008"),
	}
	var before = append([]reportmodel.ConvertedActivityAmount(nil), amounts...)

	got, err := ConvertedAmounts(16, amounts)
	if err != nil {
		t.Fatalf("convert immutable source sequence: %v", err)
	}
	if !reflect.DeepEqual(amounts, before) {
		t.Fatalf("source sequence changed: before=%#v after=%#v", before, amounts)
	}
	var want = []string{
		"fee_amount: 5.01 -> 6.01",
		"unit_price: 0.00 -> 0.00",
		"gross_value: 7.01 -> 8.01",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("converted entries = %#v, want %#v", got, want)
	}
}

// canonicalConvertedAmounts returns one synthetic amount for each supported
// kind in the canonical presentation order.
// Authored by: OpenCode
func canonicalConvertedAmounts(t *testing.T) []reportmodel.ConvertedActivityAmount {
	t.Helper()

	return []reportmodel.ConvertedActivityAmount{
		convertedAmount(t, reportmodel.ConvertedAmountKindUnitPrice, "10.005", "20.005"),
		convertedAmount(t, reportmodel.ConvertedAmountKindGrossValue, "30.004", "40.004"),
		convertedAmount(t, reportmodel.ConvertedAmountKindFeeAmount, "50.005", "60.005"),
	}
}

// containsConvertedAmountKind reports whether a canonical subsequence includes
// a supported converted-amount kind.
// Authored by: OpenCode
func containsConvertedAmountKind(kinds []reportmodel.ConvertedAmountKind, want reportmodel.ConvertedAmountKind) bool {
	for _, kind := range kinds {
		if kind == want {
			return true
		}
	}
	return false
}

// convertedAmount builds one synthetic converted amount for presentation tests.
// Authored by: OpenCode
func convertedAmount(t *testing.T, kind reportmodel.ConvertedAmountKind, original string, converted string) reportmodel.ConvertedActivityAmount {
	t.Helper()

	return reportmodel.ConvertedActivityAmount{
		SourceID:           "synthetic-source",
		AmountKind:         kind,
		OriginalCurrency:   "EUR",
		OriginalAmount:     mustFinancialDecimal(t, original),
		ReportBaseCurrency: reportmodel.ReportBaseCurrencyUSD,
		ConvertedAmount:    mustFinancialDecimal(t, converted),
	}
}

// nonFiniteConvertedAmountDecimal builds a non-finite decimal to exercise
// component-level presentation errors without using real financial data.
// Authored by: OpenCode
func nonFiniteConvertedAmountDecimal() apd.Decimal {
	return apd.Decimal{Form: apd.Infinite}
}

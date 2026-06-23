// Package model verifies report-model validation helpers and constructors.
// Authored by: OpenCode
package model

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// reportInvalidDecimal is a non-finite decimal used to exercise finite-value
// guardrails in report-model validation.
// Authored by: OpenCode
var reportInvalidDecimal = apd.Decimal{Form: apd.Infinite}

// TestCalculationErrorHandlesFallbacks verifies structured calculation-error
// fallback messages, references, and nil-safe accessors.
// Authored by: OpenCode
func TestCalculationErrorHandlesFallbacks(t *testing.T) {
	t.Parallel()

	var cause = errors.New("root cause")
	var err = NewCalculationError(CalculationErrorKindBasisAllocation, "  ", " source-1 ", " BTC ", cause)

	if err.Error() != "root cause (asset \"BTC\", source \"source-1\")" {
		t.Fatalf("unexpected calculation error string: %q", err.Error())
	}
	if !errors.Is(err, cause) {
		t.Fatalf("expected calculation error to unwrap cause")
	}
	if err.Kind() != CalculationErrorKindBasisAllocation {
		t.Fatalf("unexpected calculation error kind: %q", err.Kind())
	}
	if err.SourceID() != "source-1" {
		t.Fatalf("unexpected calculation error source ID: %q", err.SourceID())
	}
	if err.DisplayLabel() != "BTC" {
		t.Fatalf("unexpected calculation error display label: %q", err.DisplayLabel())
	}

	if err.DiagnosticFailureDetail() != err.Error() {
		t.Fatalf("expected calculation error to expose report diagnostic detail, got %q", err.DiagnosticFailureDetail())
	}
	var diagnosticCauseChain = err.DiagnosticFailureCauseChain()
	if len(diagnosticCauseChain) != 2 || diagnosticCauseChain[0] != err.Error() || diagnosticCauseChain[1] != "root cause" {
		t.Fatalf("expected wrapped calculation cause chain, got %#v", diagnosticCauseChain)
	}

	err = NewCalculationError(CalculationErrorKindInvalidRequest, "", "", "", nil)
	if err.Error() != "unsupported report calculation" {
		t.Fatalf("expected default calculation error message, got %q", err.Error())
	}
	if got := err.DiagnosticFailureCauseChain(); len(got) != 1 || got[0] != "unsupported report calculation" {
		t.Fatalf("expected default calculation error cause chain, got %#v", got)
	}

	err = NewCalculationError(
		CalculationErrorKindActivityInput,
		"outer failure",
		"source-2",
		"ETH",
		fmt.Errorf("lower token abc123 layer: %w", errors.New("Bearer jwt-secret")),
	)
	diagnosticCauseChain = err.DiagnosticFailureCauseChain()
	if len(diagnosticCauseChain) != 3 {
		t.Fatalf("expected redacted wrapped cause chain, got %#v", diagnosticCauseChain)
	}
	if diagnosticCauseChain[0] != err.Error() {
		t.Fatalf("expected actionable outer failure first, got %#v", diagnosticCauseChain)
	}
	if !strings.Contains(diagnosticCauseChain[1], "token [REDACTED]") || diagnosticCauseChain[2] != "Bearer [REDACTED]" {
		t.Fatalf("expected nested causes to be redacted, got %#v", diagnosticCauseChain)
	}
	if strings.Contains(strings.Join(diagnosticCauseChain, " "), "abc123") || strings.Contains(strings.Join(diagnosticCauseChain, " "), "jwt-secret") {
		t.Fatalf("expected secret-bearing wrapped causes to be redacted, got %#v", diagnosticCauseChain)
	}

	var nilError *CalculationError
	if nilError.Error() != "" {
		t.Fatalf("expected nil calculation error string to be empty")
	}
	if got := nilError.DiagnosticFailureDetail(); got != "" {
		t.Fatalf("expected nil calculation error diagnostic detail to be empty, got %q", got)
	}
	if got := nilError.DiagnosticFailureCauseChain(); len(got) != 0 {
		t.Fatalf("expected nil calculation error diagnostic cause chain to be empty, got %#v", got)
	}
	if nilError.Unwrap() != nil {
		t.Fatalf("expected nil calculation error unwrap to be nil")
	}
	if nilError.Kind() != "" {
		t.Fatalf("expected nil calculation error kind to be empty")
	}
	if nilError.SourceID() != "" {
		t.Fatalf("expected nil calculation error source ID to be empty")
	}
	if nilError.DisplayLabel() != "" {
		t.Fatalf("expected nil calculation error display label to be empty")
	}
}

// TestCalculationErrorHelpersCoverRemainingBranches verifies blank wrapped
// causes, duplicate suppression, blank outer detail fallback, and nil unwrap
// handling for diagnostics helper paths.
// Authored by: OpenCode
func TestCalculationErrorHelpersCoverRemainingBranches(t *testing.T) {
	t.Parallel()

	var chain = calculationErrorCauseChain("   ", errors.New("inner cause"))
	if len(chain) != 1 || chain[0] != "inner cause" {
		t.Fatalf("expected blank outer detail to defer to wrapped cause, got %#v", chain)
	}

	chain = calculationErrorCauseChain("outer detail", errors.New("   "))
	if len(chain) != 1 || chain[0] != "outer detail" {
		t.Fatalf("expected blank wrapped cause detail to be ignored, got %#v", chain)
	}

	chain = calculationErrorCauseChain("same detail", errors.New("same detail"))
	if len(chain) != 1 || chain[0] != "same detail" {
		t.Fatalf("expected duplicate wrapped cause detail to be suppressed, got %#v", chain)
	}

	if unwrapSingle(nil) != nil {
		t.Fatalf("expected nil unwrap helper input to return nil")
	}
}

// TestNewReportRequestValidatesRequiredFields verifies the reusable request
// constructor guardrails.
// Authored by: OpenCode
func TestNewReportRequestValidatesRequiredFields(t *testing.T) {
	t.Parallel()

	_, err := NewReportRequest(0, CostBasisMethodFIFO, ReportBaseCurrencyUSD, time.Now())
	if err == nil || !strings.Contains(err.Error(), "year must be greater than zero") {
		t.Fatalf("expected invalid year error, got %v", err)
	}

	_, err = NewReportRequest(2024, CostBasisMethod("bad"), ReportBaseCurrencyUSD, time.Now())
	if err == nil || !strings.Contains(err.Error(), "unsupported cost basis method") {
		t.Fatalf("expected invalid method error, got %v", err)
	}

	_, err = NewReportRequest(2024, CostBasisMethodFIFO, ReportBaseCurrencyUSD, time.Time{})
	if err == nil || !strings.Contains(err.Error(), "requested-at timestamp is required") {
		t.Fatalf("expected missing timestamp error, got %v", err)
	}

	_, err = NewReportRequest(2024, CostBasisMethodFIFO, ReportBaseCurrency("GBP"), time.Now())
	if err == nil || !strings.Contains(err.Error(), "unsupported report base currency") {
		t.Fatalf("expected invalid report base currency error, got %v", err)
	}

	request, err := NewReportRequest(2024, CostBasisMethodFIFO, ReportBaseCurrencyUSD, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}
	if err = request.Validate(); err != nil {
		t.Fatalf("validate request: %v", err)
	}
}

// TestReportRequestValidatesBaseCurrency verifies missing and unsupported base
// currencies are rejected by both constructor and direct request validation.
// Authored by: OpenCode
func TestReportRequestValidatesBaseCurrency(t *testing.T) {
	t.Parallel()

	var requestedAt = time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC)
	var testCases = []struct {
		name     string
		currency ReportBaseCurrency
		want     string
	}{
		{name: "missing", currency: ReportBaseCurrency(""), want: "report base currency is required"},
		{name: "invalid", currency: ReportBaseCurrency("GBP"), want: "unsupported report base currency"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewReportRequest(2024, CostBasisMethodFIFO, testCase.currency, requestedAt)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("expected constructor error containing %q, got %v", testCase.want, err)
			}

			var request = ReportRequest{
				Year:               2024,
				CostBasisMethod:    CostBasisMethodFIFO,
				ReportBaseCurrency: testCase.currency,
				RequestedAt:        requestedAt,
			}
			err = request.Validate()
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("expected validation error containing %q, got %v", testCase.want, err)
			}
		})
	}
}

// TestNewCapitalGainsReportValidatesNestedContent verifies that the top-level
// report helper rejects invalid nested rows.
// Authored by: OpenCode
func TestNewCapitalGainsReportValidatesNestedContent(t *testing.T) {
	t.Parallel()

	request, err := NewReportRequest(2024, CostBasisMethodHIFO, ReportBaseCurrencyUSD, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}

	_, err = NewCapitalGainsReport(
		request,
		time.Date(2026, time.May, 21, 12, 0, 0, 0, time.UTC),
		"",
		[]AssetSummaryEntry{{
			AssetIdentityKey: "",
			DisplayLabel:     "BTC",
			NetGainOrLoss:    mustReportDecimal(t, "1"),
		}},
		mustReportDecimal(t, "1"),
		nil,
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "summary entry 0") {
		t.Fatalf("expected nested summary validation failure, got %v", err)
	}

	report, err := NewCapitalGainsReport(
		request,
		time.Date(2026, time.May, 21, 12, 0, 0, 0, time.UTC),
		"NOT APPLICABLE",
		[]AssetSummaryEntry{{
			AssetIdentityKey:          "asset-btc",
			DisplayLabel:              "BTC",
			NetGainOrLoss:             mustReportDecimal(t, "1"),
			ReportCalculationCurrency: "NOT APPLICABLE",
		}},
		mustReportDecimal(t, "1"),
		[]ReferenceLiquidationEntry{{
			AssetIdentityKey:                   "asset-btc",
			DisplayLabel:                       "BTC",
			FullLiquidationCountThroughYearEnd: 1,
			MainSectionStatus:                  ReferenceSectionStatusIncludedInMainSections,
		}},
		[]AssetDetailSection{{
			AssetIdentityKey:    "asset-btc",
			DisplayLabel:        "BTC",
			OpeningQuantity:     mustReportDecimal(t, "1"),
			OpeningCostBasis:    mustReportDecimal(t, "10"),
			ClosingQuantity:     mustReportDecimal(t, "0"),
			ClosingCostBasis:    mustReportDecimal(t, "0"),
			CalculationCurrency: "USD",
			ActivityRows: []AssetActivityRow{{
				SourceID:            "sell-1",
				OccurredAt:          time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
				ActivityType:        ActivityTypeSell,
				Quantity:            mustReportDecimal(t, "1"),
				GrossValue:          decimalPointer(t, "12"),
				FeeAmount:           decimalPointer(t, "0"),
				BasisAfterRow:       mustReportDecimal(t, "0"),
				CalculationCurrency: "USD",
				QuantityAfterRow:    mustReportDecimal(t, "0"),
				LiquidationCalculation: &LiquidationCalculation{
					SourceID:               "sell-1",
					OccurredAt:             time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
					DisposedQuantity:       mustReportDecimal(t, "1"),
					AllocatedBasis:         mustReportDecimal(t, "10"),
					NetLiquidationProceeds: mustReportDecimal(t, "12"),
					GainOrLoss:             mustReportDecimal(t, "2"),
					ActivityCurrency:       "USD",
					CalculationCurrency:    "USD",
					Matches: []BasisMatch{{
						AcquisitionSourceID: "buy-1",
						MatchedQuantity:     mustReportDecimal(t, "1"),
						MatchedBasis:        mustReportDecimal(t, "10"),
						MatchedProceeds:     decimalPointer(t, "12"),
						MatchedGainOrLoss:   decimalPointer(t, "2"),
					}},
				},
			}},
			LiquidationSummaries: []LiquidationCalculation{{
				SourceID:               "sell-1",
				OccurredAt:             time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
				DisposedQuantity:       mustReportDecimal(t, "1"),
				AllocatedBasis:         mustReportDecimal(t, "10"),
				NetLiquidationProceeds: mustReportDecimal(t, "12"),
				GainOrLoss:             mustReportDecimal(t, "2"),
				ActivityCurrency:       "USD",
				CalculationCurrency:    "USD",
				Matches: []BasisMatch{{
					AcquisitionSourceID: "buy-1",
					MatchedQuantity:     mustReportDecimal(t, "1"),
					MatchedBasis:        mustReportDecimal(t, "10"),
					MatchedProceeds:     decimalPointer(t, "12"),
					MatchedGainOrLoss:   decimalPointer(t, "2"),
				}},
			}},
		}},
	)
	if err != nil {
		t.Fatalf("new capital gains report: %v", err)
	}
	if err = report.Validate(); err != nil {
		t.Fatalf("validate capital gains report: %v", err)
	}
}

// TestReportConstructorsCloneOptionalDetailDecimals verifies that report detail
// constructors do not alias caller-owned optional decimal pointers.
// Authored by: OpenCode
func TestReportConstructorsCloneOptionalDetailDecimals(t *testing.T) {
	t.Parallel()

	var sectionUnitPrice = decimalPointer(t, "11")
	var sectionGrossValue = decimalPointer(t, "12")
	var sectionFeeAmount = decimalPointer(t, "1")
	var sectionMatchedProceeds = decimalPointer(t, "12")
	var sectionMatchedGainOrLoss = decimalPointer(t, "2")
	var section, sectionErr = NewAssetDetailSection(
		"asset-btc",
		"BTC",
		mustReportDecimal(t, "1"),
		mustReportDecimal(t, "10"),
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "0"),
		"USD",
		[]AssetActivityRow{{
			SourceID:         "sell-1",
			OccurredAt:       time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
			ActivityType:     ActivityTypeSell,
			Quantity:         mustReportDecimal(t, "1"),
			UnitPrice:        sectionUnitPrice,
			GrossValue:       sectionGrossValue,
			FeeAmount:        sectionFeeAmount,
			BasisAfterRow:    mustReportDecimal(t, "0"),
			QuantityAfterRow: mustReportDecimal(t, "0"),
		}},
		[]LiquidationCalculation{{
			SourceID:               "sell-1",
			OccurredAt:             time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
			DisposedQuantity:       mustReportDecimal(t, "1"),
			AllocatedBasis:         mustReportDecimal(t, "10"),
			NetLiquidationProceeds: mustReportDecimal(t, "12"),
			GainOrLoss:             mustReportDecimal(t, "2"),
			ActivityCurrency:       "USD",
			CalculationCurrency:    "USD",
			Matches: []BasisMatch{{
				AcquisitionSourceID: "buy-1",
				MatchedQuantity:     mustReportDecimal(t, "1"),
				MatchedBasis:        mustReportDecimal(t, "10"),
				MatchedProceeds:     sectionMatchedProceeds,
				MatchedGainOrLoss:   sectionMatchedGainOrLoss,
			}},
		}},
	)
	if sectionErr != nil {
		t.Fatalf("new asset detail section: %v", sectionErr)
	}

	*sectionUnitPrice = mustReportDecimal(t, "101")
	*sectionGrossValue = mustReportDecimal(t, "102")
	*sectionFeeAmount = mustReportDecimal(t, "103")
	*sectionMatchedProceeds = mustReportDecimal(t, "104")
	*sectionMatchedGainOrLoss = mustReportDecimal(t, "105")
	assertOptionalDecimalString(t, section.ActivityRows[0].UnitPrice, "11")
	assertOptionalDecimalString(t, section.ActivityRows[0].GrossValue, "12")
	assertOptionalDecimalString(t, section.ActivityRows[0].FeeAmount, "1")
	assertOptionalDecimalString(t, section.LiquidationSummaries[0].Matches[0].MatchedProceeds, "12")
	assertOptionalDecimalString(t, section.LiquidationSummaries[0].Matches[0].MatchedGainOrLoss, "2")

	var reportUnitPrice = decimalPointer(t, "21")
	var reportGrossValue = decimalPointer(t, "22")
	var reportFeeAmount = decimalPointer(t, "3")
	var reportMatchedProceeds = decimalPointer(t, "22")
	var reportMatchedGainOrLoss = decimalPointer(t, "4")
	var reportSections = []AssetDetailSection{{
		AssetIdentityKey:    "asset-eth",
		DisplayLabel:        "ETH",
		OpeningQuantity:     mustReportDecimal(t, "1"),
		OpeningCostBasis:    mustReportDecimal(t, "18"),
		ClosingQuantity:     mustReportDecimal(t, "0"),
		ClosingCostBasis:    mustReportDecimal(t, "0"),
		CalculationCurrency: "USD",
		ActivityRows: []AssetActivityRow{{
			SourceID:         "sell-2",
			OccurredAt:       time.Date(2024, time.January, 3, 0, 0, 0, 0, time.UTC),
			ActivityType:     ActivityTypeSell,
			Quantity:         mustReportDecimal(t, "1"),
			UnitPrice:        reportUnitPrice,
			GrossValue:       reportGrossValue,
			FeeAmount:        reportFeeAmount,
			BasisAfterRow:    mustReportDecimal(t, "0"),
			QuantityAfterRow: mustReportDecimal(t, "0"),
		}},
		LiquidationSummaries: []LiquidationCalculation{{
			SourceID:               "sell-2",
			OccurredAt:             time.Date(2024, time.January, 3, 0, 0, 0, 0, time.UTC),
			DisposedQuantity:       mustReportDecimal(t, "1"),
			AllocatedBasis:         mustReportDecimal(t, "18"),
			NetLiquidationProceeds: mustReportDecimal(t, "22"),
			GainOrLoss:             mustReportDecimal(t, "4"),
			ActivityCurrency:       "USD",
			CalculationCurrency:    "USD",
			Matches: []BasisMatch{{
				AcquisitionSourceID: "buy-2",
				MatchedQuantity:     mustReportDecimal(t, "1"),
				MatchedBasis:        mustReportDecimal(t, "18"),
				MatchedProceeds:     reportMatchedProceeds,
				MatchedGainOrLoss:   reportMatchedGainOrLoss,
			}},
		}},
	}}
	var request, requestErr = NewReportRequest(2024, CostBasisMethodFIFO, ReportBaseCurrencyUSD, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
	if requestErr != nil {
		t.Fatalf("new report request: %v", requestErr)
	}
	var report, reportErr = NewCapitalGainsReport(request, time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC), "USD", nil, mustReportDecimal(t, "0"), nil, reportSections)
	if reportErr != nil {
		t.Fatalf("new capital gains report: %v", reportErr)
	}

	*reportUnitPrice = mustReportDecimal(t, "201")
	*reportGrossValue = mustReportDecimal(t, "202")
	*reportFeeAmount = mustReportDecimal(t, "203")
	*reportMatchedProceeds = mustReportDecimal(t, "204")
	*reportMatchedGainOrLoss = mustReportDecimal(t, "205")
	reportSections[0].ActivityRows[0].SourceID = "mutated"
	reportSections[0].LiquidationSummaries[0].Matches[0].AcquisitionSourceID = "mutated"
	assertOptionalDecimalString(t, report.DetailSections[0].ActivityRows[0].UnitPrice, "21")
	assertOptionalDecimalString(t, report.DetailSections[0].ActivityRows[0].GrossValue, "22")
	assertOptionalDecimalString(t, report.DetailSections[0].ActivityRows[0].FeeAmount, "3")
	assertOptionalDecimalString(t, report.DetailSections[0].LiquidationSummaries[0].Matches[0].MatchedProceeds, "22")
	assertOptionalDecimalString(t, report.DetailSections[0].LiquidationSummaries[0].Matches[0].MatchedGainOrLoss, "4")
	if report.DetailSections[0].ActivityRows[0].SourceID != "sell-2" {
		t.Fatalf("expected report detail activity rows to be independent, got %#v", report.DetailSections[0].ActivityRows[0])
	}
	if report.DetailSections[0].LiquidationSummaries[0].Matches[0].AcquisitionSourceID != "buy-2" {
		t.Fatalf("expected report liquidation summaries to be independent, got %#v", report.DetailSections[0].LiquidationSummaries[0].Matches[0])
	}
}

// TestReferenceAndDetailValidationGuardrails verifies remaining report-model
// validation branches for reference, detail, activity, and liquidation rows.
// Authored by: OpenCode
func TestReferenceAndDetailValidationGuardrails(t *testing.T) {
	t.Parallel()

	_, err := NewReferenceLiquidationEntry("asset-btc", "BTC", -1, ReferenceSectionStatusReferenceOnly)
	if err == nil || !strings.Contains(err.Error(), "full liquidation count must not be negative") {
		t.Fatalf("expected negative liquidation count error, got %v", err)
	}

	_, err = NewReferenceLiquidationEntry("asset-btc", "BTC", 0, ReferenceSectionStatus("bad"))
	if err == nil || !strings.Contains(err.Error(), "unsupported reference section status") {
		t.Fatalf("expected unsupported reference status error, got %v", err)
	}

	_, err = NewAssetDetailSection(
		"asset-btc",
		"BTC",
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "0"),
		"USD",
		[]AssetActivityRow{{
			SourceID:         "sell-1",
			OccurredAt:       time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
			ActivityType:     ActivityType("swap"),
			Quantity:         mustReportDecimal(t, "1"),
			BasisAfterRow:    mustReportDecimal(t, "0"),
			QuantityAfterRow: mustReportDecimal(t, "0"),
		}},
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "activity row 0: asset activity row activity type") {
		t.Fatalf("expected unsupported activity type error, got %v", err)
	}

	_, err = NewAssetDetailSection(
		"asset-btc",
		"BTC",
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "0"),
		"USD",
		nil,
		[]LiquidationCalculation{{
			SourceID:               "sell-1",
			OccurredAt:             time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
			DisposedQuantity:       mustReportDecimal(t, "1"),
			AllocatedBasis:         mustReportDecimal(t, "0"),
			NetLiquidationProceeds: mustReportDecimal(t, "1"),
			GainOrLoss:             reportInvalidDecimal,
			ActivityCurrency:       "USD",
		}},
	)
	if err == nil || !strings.Contains(err.Error(), "liquidation summary 0: liquidation calculation gain or loss") {
		t.Fatalf("expected invalid liquidation gain or loss error, got %v", err)
	}

	var validRow = AssetActivityRow{
		SourceID:         "buy-1",
		OccurredAt:       time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
		ActivityType:     ActivityTypeBuy,
		Quantity:         mustReportDecimal(t, "1"),
		BasisAfterRow:    mustReportDecimal(t, "1"),
		QuantityAfterRow: mustReportDecimal(t, "1"),
	}
	validRow.UnitPrice = &reportInvalidDecimal
	if err = validRow.Validate(); err == nil || !strings.Contains(err.Error(), "asset activity row unit price") {
		t.Fatalf("expected invalid unit-price optional decimal error, got %v", err)
	}
	validRow.UnitPrice = nil
	validRow.GrossValue = &reportInvalidDecimal
	if err = validRow.Validate(); err == nil || !strings.Contains(err.Error(), "asset activity row gross value") {
		t.Fatalf("expected invalid optional decimal error, got %v", err)
	}

	var invalidLiquidation = LiquidationCalculation{
		SourceID:               "sell-2",
		OccurredAt:             time.Date(2024, time.January, 3, 0, 0, 0, 0, time.UTC),
		DisposedQuantity:       mustReportDecimal(t, "1"),
		AllocatedBasis:         mustReportDecimal(t, "0"),
		NetLiquidationProceeds: mustReportDecimal(t, "1"),
		GainOrLoss:             mustReportDecimal(t, "1"),
	}
	if err = invalidLiquidation.Validate(); err == nil || !strings.Contains(err.Error(), "activity currency is required") {
		t.Fatalf("expected missing liquidation currency error, got %v", err)
	}

	if err = (BasisMatch{}).Validate(); err == nil || !strings.Contains(err.Error(), "acquisition source ID is required") {
		t.Fatalf("expected blank basis-match acquisition source ID to fail, got %v", err)
	}
}

// TestReportDocumentAndOutputValidation verifies the rendered-document and
// output outcome helpers used by renderer and writer packages.
// Authored by: OpenCode
func TestReportDocumentAndOutputValidation(t *testing.T) {
	t.Parallel()

	_, err := NewReportDocument(ReportDocumentType("html"), "# Report\n", 2024, CostBasisMethodFIFO, time.Now())
	if err == nil || !strings.Contains(err.Error(), "unsupported report document type") {
		t.Fatalf("expected invalid document type error, got %v", err)
	}

	document, err := NewReportDocument(ReportDocumentTypeMarkdown, "# Report\n", 2024, CostBasisMethodFIFO, time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report document: %v", err)
	}
	if err = document.Validate(); err != nil {
		t.Fatalf("validate report document: %v", err)
	}

	_, err = NewReportOutputFile("/tmp/Documents", "report.md", "/tmp/Documents/report.md", time.Now(), false, "open failed")
	if err == nil || !strings.Contains(err.Error(), "open error requires an open request") {
		t.Fatalf("expected open error validation failure, got %v", err)
	}

	outputFile, err := NewReportOutputFile("/tmp/Documents", "report.md", "/tmp/Documents/report.md", time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC), true, "open failed")
	if err != nil {
		t.Fatalf("new report output file: %v", err)
	}
	if err = outputFile.Validate(); err != nil {
		t.Fatalf("validate report output file: %v", err)
	}
}

// TestReportDocumentValidationGuardrails verifies remaining document and report
// constructor validation branches.
// Authored by: OpenCode
func TestReportDocumentValidationGuardrails(t *testing.T) {
	t.Parallel()

	request, err := NewReportRequest(2024, CostBasisMethodFIFO, ReportBaseCurrencyUSD, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}

	_, err = NewCapitalGainsReport(
		request,
		time.Date(2026, time.May, 21, 12, 0, 0, 0, time.UTC),
		"USD",
		nil,
		reportInvalidDecimal,
		nil,
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "capital gains report yearly net total") {
		t.Fatalf("expected invalid yearly net total error, got %v", err)
	}

	_, err = NewReportDocument(ReportDocumentTypeMarkdown, "   ", 2024, CostBasisMethodFIFO, time.Now())
	if err == nil || !strings.Contains(err.Error(), "report document content is required") {
		t.Fatalf("expected missing content error, got %v", err)
	}

	_, err = NewReportDocument(ReportDocumentTypeMarkdown, "# Report\n", 2024, CostBasisMethod("bad"), time.Now())
	if err == nil || !strings.Contains(err.Error(), "report document cost basis method") {
		t.Fatalf("expected invalid report document method error, got %v", err)
	}
}

// TestReportConstructorsAndValidationHelpersCoverRemainingBranches verifies
// constructor success paths and direct validation helper branches not exercised
// by the broader report-model scenarios.
// Authored by: OpenCode
func TestReportConstructorsAndValidationHelpersCoverRemainingBranches(t *testing.T) {
	t.Parallel()

	var summaryEntry, err = NewAssetSummaryEntry(" asset-btc ", " BTC ", mustReportDecimal(t, "2"), " USD ")
	if err != nil {
		t.Fatalf("new asset summary entry: %v", err)
	}
	if summaryEntry.AssetIdentityKey != "asset-btc" || summaryEntry.DisplayLabel != "BTC" || summaryEntry.ReportCalculationCurrency != "USD" {
		t.Fatalf("expected trimmed summary entry fields, got %#v", summaryEntry)
	}

	if err = (AssetSummaryEntry{AssetIdentityKey: "asset-btc", NetGainOrLoss: reportInvalidDecimal}).Validate(); err == nil || !strings.Contains(err.Error(), "asset summary entry net gain or loss") {
		t.Fatalf("expected invalid net gain or loss to fail, got %v", err)
	}

	var detailSection, detailErr = NewAssetDetailSection(
		" asset-btc ",
		" BTC ",
		mustReportDecimal(t, "1"),
		mustReportDecimal(t, "10"),
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "0"),
		" USD ",
		nil,
		nil,
	)
	if detailErr != nil {
		t.Fatalf("new asset detail section: %v", detailErr)
	}
	if detailSection.AssetIdentityKey != "asset-btc" || detailSection.DisplayLabel != "BTC" || detailSection.CalculationCurrency != "USD" {
		t.Fatalf("expected trimmed detail section fields, got %#v", detailSection)
	}

	if err = (AssetDetailSection{AssetIdentityKey: "asset-btc", OpeningQuantity: mustReportDecimal(t, "-1")}).Validate(); err == nil || !strings.Contains(err.Error(), "opening quantity") {
		t.Fatalf("expected negative opening quantity to fail, got %v", err)
	}
	if err = (AssetDetailSection{AssetIdentityKey: "asset-btc", OpeningQuantity: mustReportDecimal(t, "0"), OpeningCostBasis: mustReportDecimal(t, "0"), ClosingQuantity: mustReportDecimal(t, "0"), ClosingCostBasis: mustReportDecimal(t, "-1")}).Validate(); err == nil || !strings.Contains(err.Error(), "closing cost basis") {
		t.Fatalf("expected negative closing basis to fail, got %v", err)
	}

	if err = (AssetActivityRow{SourceID: "sell-1", OccurredAt: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC), ActivityType: ActivityTypeSell, Quantity: mustReportDecimal(t, "1"), BasisAfterRow: mustReportDecimal(t, "0"), QuantityAfterRow: mustReportDecimal(t, "0"), LiquidationCalculation: &LiquidationCalculation{}}).Validate(); err == nil || !strings.Contains(err.Error(), "asset activity row liquidation calculation") {
		t.Fatalf("expected nested liquidation validation to fail, got %v", err)
	}
	if err = (AssetActivityRow{SourceID: "sell-1", OccurredAt: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC), ActivityType: ActivityTypeSell, Quantity: mustReportDecimal(t, "1"), BasisAfterRow: mustReportDecimal(t, "-1"), QuantityAfterRow: mustReportDecimal(t, "0")}).Validate(); err == nil || !strings.Contains(err.Error(), "basis after row") {
		t.Fatalf("expected negative basis-after-row to fail, got %v", err)
	}
	if err = (AssetActivityRow{SourceID: "sell-1", OccurredAt: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC), ActivityType: ActivityTypeSell, Quantity: mustReportDecimal(t, "1"), BasisAfterRow: mustReportDecimal(t, "0"), QuantityAfterRow: mustReportDecimal(t, "-1")}).Validate(); err == nil || !strings.Contains(err.Error(), "quantity after row") {
		t.Fatalf("expected negative quantity-after-row to fail, got %v", err)
	}

	if err = (LiquidationCalculation{SourceID: "sell-1", OccurredAt: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC), DisposedQuantity: mustReportDecimal(t, "1"), AllocatedBasis: mustReportDecimal(t, "-1"), NetLiquidationProceeds: mustReportDecimal(t, "1"), GainOrLoss: mustReportDecimal(t, "0"), ActivityCurrency: "USD"}).Validate(); err == nil || !strings.Contains(err.Error(), "allocated basis") {
		t.Fatalf("expected negative allocated basis to fail, got %v", err)
	}
	if err = (LiquidationCalculation{SourceID: "sell-1", OccurredAt: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC), DisposedQuantity: mustReportDecimal(t, "1"), AllocatedBasis: mustReportDecimal(t, "0"), NetLiquidationProceeds: reportInvalidDecimal, GainOrLoss: mustReportDecimal(t, "0"), ActivityCurrency: "USD"}).Validate(); err == nil || !strings.Contains(err.Error(), "net liquidation proceeds") {
		t.Fatalf("expected invalid proceeds to fail, got %v", err)
	}
	if err = (LiquidationCalculation{SourceID: "sell-1", OccurredAt: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC), DisposedQuantity: mustReportDecimal(t, "1"), AllocatedBasis: mustReportDecimal(t, "0"), NetLiquidationProceeds: mustReportDecimal(t, "1"), GainOrLoss: mustReportDecimal(t, "1"), ActivityCurrency: "USD", Matches: []BasisMatch{{}}}).Validate(); err == nil || !strings.Contains(err.Error(), "basis match 0") {
		t.Fatalf("expected invalid basis match to fail, got %v", err)
	}

	var request, requestErr = NewReportRequest(2024, CostBasisMethodFIFO, ReportBaseCurrencyUSD, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
	if requestErr != nil {
		t.Fatalf("new report request: %v", requestErr)
	}
	var report, reportErr = NewCapitalGainsReport(request, time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC), "USD", nil, mustReportDecimal(t, "0"), nil, nil)
	if reportErr != nil {
		t.Fatalf("new capital gains report: %v", reportErr)
	}
	if report.ReportCalculationCurrency != "USD" {
		t.Fatalf("expected report calculation currency to be preserved, got %#v", report)
	}

	var section, sectionErr = NewAssetDetailSection(
		"asset-btc",
		"BTC",
		mustReportDecimal(t, "1"),
		mustReportDecimal(t, "10"),
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "0"),
		"USD",
		nil,
		[]LiquidationCalculation{{
			SourceID:               "sell-1",
			OccurredAt:             time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
			DisposedQuantity:       mustReportDecimal(t, "1"),
			AllocatedBasis:         mustReportDecimal(t, "10"),
			NetLiquidationProceeds: mustReportDecimal(t, "12"),
			GainOrLoss:             mustReportDecimal(t, "2"),
			ActivityCurrency:       "USD",
			CalculationCurrency:    "USD",
			Matches: []BasisMatch{{
				AcquisitionSourceID: "buy-1",
				MatchedQuantity:     mustReportDecimal(t, "1"),
				MatchedBasis:        mustReportDecimal(t, "10"),
				MatchedProceeds:     decimalPointer(t, "12"),
				MatchedGainOrLoss:   decimalPointer(t, "2"),
			}},
		}},
	)
	if sectionErr != nil {
		t.Fatalf("new asset detail section with basis matches: %v", sectionErr)
	}
	section.LiquidationSummaries[0].Matches[0].AcquisitionSourceID = "mutated"
	if section.LiquidationSummaries[0].Matches[0].AcquisitionSourceID != "mutated" {
		t.Fatalf("expected local section mutation to succeed for copied value")
	}
	if err = (CapitalGainsReport{Year: 2024, CostBasisMethod: CostBasisMethodFIFO, YearlyNetTotal: mustReportDecimal(t, "0")}).Validate(); err == nil || !strings.Contains(err.Error(), "generated-at timestamp is required") {
		t.Fatalf("expected missing generated-at timestamp to fail, got %v", err)
	}

	if err = (ReferenceLiquidationEntry{}).Validate(); err == nil || !strings.Contains(err.Error(), "asset identity key is required") {
		t.Fatalf("expected blank reference asset identity key to fail, got %v", err)
	}

	if err = (AssetDetailSection{}).Validate(); err == nil || !strings.Contains(err.Error(), "asset identity key is required") {
		t.Fatalf("expected blank detail asset identity key to fail, got %v", err)
	}
	if err = (AssetDetailSection{AssetIdentityKey: "asset-btc", OpeningCostBasis: reportInvalidDecimal}).Validate(); err == nil || !strings.Contains(err.Error(), "opening cost basis") {
		t.Fatalf("expected invalid opening cost basis to fail, got %v", err)
	}
	if err = (AssetDetailSection{AssetIdentityKey: "asset-btc", ClosingQuantity: reportInvalidDecimal}).Validate(); err == nil || !strings.Contains(err.Error(), "closing quantity") {
		t.Fatalf("expected invalid closing quantity to fail, got %v", err)
	}

	if err = (AssetActivityRow{}).Validate(); err == nil || !strings.Contains(err.Error(), "source ID is required") {
		t.Fatalf("expected blank activity row source ID to fail, got %v", err)
	}
	if err = (AssetActivityRow{SourceID: "row-1"}).Validate(); err == nil || !strings.Contains(err.Error(), "occurred-at timestamp is required") {
		t.Fatalf("expected blank activity row timestamp to fail, got %v", err)
	}
	if err = (AssetActivityRow{SourceID: "row-1", OccurredAt: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC), ActivityType: ActivityTypeBuy}).Validate(); err == nil || !strings.Contains(err.Error(), "quantity") {
		t.Fatalf("expected missing positive activity quantity to fail, got %v", err)
	}
	if err = (AssetActivityRow{SourceID: "row-1", OccurredAt: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC), ActivityType: ActivityTypeBuy, Quantity: mustReportDecimal(t, "1"), FeeAmount: &reportInvalidDecimal, BasisAfterRow: mustReportDecimal(t, "0"), QuantityAfterRow: mustReportDecimal(t, "0")}).Validate(); err == nil || !strings.Contains(err.Error(), "fee amount") {
		t.Fatalf("expected invalid activity fee amount to fail, got %v", err)
	}

	if err = (LiquidationCalculation{}).Validate(); err == nil || !strings.Contains(err.Error(), "source ID is required") {
		t.Fatalf("expected blank liquidation source ID to fail, got %v", err)
	}
	if err = (LiquidationCalculation{SourceID: "sell-1"}).Validate(); err == nil || !strings.Contains(err.Error(), "occurred-at timestamp is required") {
		t.Fatalf("expected blank liquidation timestamp to fail, got %v", err)
	}
	if err = (LiquidationCalculation{SourceID: "sell-1", OccurredAt: time.Date(2024, time.January, 3, 0, 0, 0, 0, time.UTC)}).Validate(); err == nil || !strings.Contains(err.Error(), "disposed quantity") {
		t.Fatalf("expected missing positive disposed quantity to fail, got %v", err)
	}

	if _, err = NewCapitalGainsReport(ReportRequest{}, time.Now(), "USD", nil, mustReportDecimal(t, "0"), nil, nil); err == nil || !strings.Contains(err.Error(), "capital gains report request") {
		t.Fatalf("expected invalid report request to fail top-level constructor, got %v", err)
	}
	if err = (CapitalGainsReport{Year: 2024, CostBasisMethod: CostBasisMethod("bad"), GeneratedAt: time.Now(), YearlyNetTotal: mustReportDecimal(t, "0")}).Validate(); err == nil || !strings.Contains(err.Error(), "cost basis method") {
		t.Fatalf("expected invalid report cost basis method to fail, got %v", err)
	}
	if err = (CapitalGainsReport{Year: 2024, CostBasisMethod: CostBasisMethodFIFO, GeneratedAt: time.Now(), YearlyNetTotal: mustReportDecimal(t, "0"), ReferenceEntries: []ReferenceLiquidationEntry{{}}}).Validate(); err == nil || !strings.Contains(err.Error(), "reference entry 0") {
		t.Fatalf("expected invalid nested reference entry to fail, got %v", err)
	}
	if err = (CapitalGainsReport{Year: 2024, CostBasisMethod: CostBasisMethodFIFO, GeneratedAt: time.Now(), YearlyNetTotal: mustReportDecimal(t, "0"), DetailSections: []AssetDetailSection{{}}}).Validate(); err == nil || !strings.Contains(err.Error(), "detail section 0") {
		t.Fatalf("expected invalid nested detail section to fail, got %v", err)
	}

	if err = (ReportDocument{DocumentType: ReportDocumentTypeMarkdown, Content: "# Report\n", Year: 0, CostBasisMethod: CostBasisMethodFIFO, GeneratedAt: time.Now()}).Validate(); err == nil || !strings.Contains(err.Error(), "report document year must be greater than zero") {
		t.Fatalf("expected missing document year to fail, got %v", err)
	}
	if err = (ReportDocument{DocumentType: ReportDocumentTypeMarkdown, Content: "# Report\n", Year: 2024, CostBasisMethod: CostBasisMethodFIFO}).Validate(); err == nil || !strings.Contains(err.Error(), "generated-at timestamp is required") {
		t.Fatalf("expected missing document timestamp to fail, got %v", err)
	}

	if err = (ReportOutputFile{DocumentsDirectory: "/tmp/docs", Filename: "report.md", Path: "/tmp/docs/report.md", OpenRequested: true}).Validate(); err == nil || !strings.Contains(err.Error(), "saved-at timestamp is required") {
		t.Fatalf("expected missing output timestamp to fail, got %v", err)
	}
	if err = (ReportOutputFile{Filename: "report.md", Path: "/tmp/docs/report.md", SavedAt: time.Now()}).Validate(); err == nil || !strings.Contains(err.Error(), "documents directory") {
		t.Fatalf("expected missing output documents directory to fail, got %v", err)
	}
	if err = (ReportOutputFile{DocumentsDirectory: "/tmp/docs", Filename: "report.md", SavedAt: time.Now()}).Validate(); err == nil || !strings.Contains(err.Error(), "output path") {
		t.Fatalf("expected missing output path to fail, got %v", err)
	}

	if err = validateOptionalDecimal(nil, "optional"); err != nil {
		t.Fatalf("expected nil optional decimal to validate, got %v", err)
	}
	if err = validatePositiveDecimal(reportInvalidDecimal, "positive"); err == nil || !strings.Contains(err.Error(), "positive") {
		t.Fatalf("expected non-finite positive decimal to fail, got %v", err)
	}
	if err = validatePositiveDecimal(mustReportDecimal(t, "0"), "positive"); err == nil || !strings.Contains(err.Error(), "must be greater than zero") {
		t.Fatalf("expected zero positive decimal to fail, got %v", err)
	}
	if err = validateNonNegativeDecimal(reportInvalidDecimal, "non-negative"); err == nil || !strings.Contains(err.Error(), "non-negative") {
		t.Fatalf("expected non-finite non-negative decimal to fail, got %v", err)
	}
	if err = validateNonNegativeDecimal(mustReportDecimal(t, "-1"), "non-negative"); err == nil || !strings.Contains(err.Error(), "must not be negative") {
		t.Fatalf("expected negative non-negative decimal to fail, got %v", err)
	}
}

// TestBasisMatchValidationAndCloneOptionalDecimalCoverRemainingBranches
// verifies the remaining basis-match guardrails and optional-decimal clone
// branches.
// Authored by: OpenCode
func TestBasisMatchValidationAndCloneOptionalDecimalCoverRemainingBranches(t *testing.T) {
	t.Parallel()

	if err := (BasisMatch{AcquisitionSourceID: "buy-1"}).Validate(); err == nil || !strings.Contains(err.Error(), "matched quantity") {
		t.Fatalf("expected missing matched quantity to fail, got %v", err)
	}
	if err := (BasisMatch{AcquisitionSourceID: "buy-1", MatchedQuantity: mustReportDecimal(t, "1"), MatchedBasis: reportInvalidDecimal}).Validate(); err == nil || !strings.Contains(err.Error(), "matched basis") {
		t.Fatalf("expected invalid matched basis to fail, got %v", err)
	}
	if err := (BasisMatch{AcquisitionSourceID: "buy-1", MatchedQuantity: mustReportDecimal(t, "1"), MatchedBasis: mustReportDecimal(t, "0"), MatchedProceeds: &reportInvalidDecimal}).Validate(); err == nil || !strings.Contains(err.Error(), "matched proceeds") {
		t.Fatalf("expected invalid matched proceeds to fail, got %v", err)
	}
	if err := (BasisMatch{AcquisitionSourceID: "buy-1", MatchedQuantity: mustReportDecimal(t, "1"), MatchedBasis: mustReportDecimal(t, "0"), MatchedGainOrLoss: &reportInvalidDecimal}).Validate(); err == nil || !strings.Contains(err.Error(), "matched gain or loss") {
		t.Fatalf("expected invalid matched gain or loss to fail, got %v", err)
	}

	if cloned := cloneOptionalDecimal(nil); cloned != nil {
		t.Fatalf("expected nil optional decimal clone to stay nil, got %#v", cloned)
	}

	var original = mustReportDecimal(t, "1.5")
	var cloned = cloneOptionalDecimal(&original)
	if cloned == nil || cloned == &original || cloned.Cmp(&original) != 0 {
		t.Fatalf("expected optional decimal clone to copy the original value, got original=%#v cloned=%#v", original, cloned)
	}
	original = mustReportDecimal(t, "2")
	if got, err := decimalsupport.CanonicalStringPointer(cloned); err != nil || got != "1.5" {
		t.Fatalf("expected cloned optional decimal to remain independent, got %q err=%v", got, err)
	}
}

// TestConversionAuditValidationGuardrails verifies conversion audit model
// success paths and relationship guardrails around evidence and amounts.
// Authored by: OpenCode
func TestConversionAuditValidationGuardrails(t *testing.T) {
	t.Parallel()

	var activityDate = time.Date(2024, time.January, 5, 0, 0, 0, 0, time.UTC)
	var evidence = ExchangeRateEvidence{
		SourceCurrency:   "EUR",
		BaseCurrency:     ReportBaseCurrencyUSD,
		ActivityDate:     activityDate,
		RateDate:         activityDate,
		Authority:        ExchangeRateAuthorityFederalReserve,
		ProviderID:       ExchangeRateProviderIDFederalReserveH10,
		RateKind:         "noon buying rate in New York for cable transfers payable in listed currencies",
		QuoteDirection:   ExchangeRateQuoteDirectionSourcePerBase,
		RateValue:        mustReportDecimal(t, "2"),
		DatasetReference: "H10/EUR/2024-01-05",
	}
	if err := evidence.Validate(); err != nil {
		t.Fatalf("expected valid evidence: %v", err)
	}

	var amount = ConvertedActivityAmount{
		SourceID:             "eur-buy-1",
		AmountKind:           ConvertedAmountKindGrossValue,
		OriginalCurrency:     "EUR",
		OriginalAmount:       mustReportDecimal(t, "100"),
		ReportBaseCurrency:   ReportBaseCurrencyUSD,
		ConvertedAmount:      mustReportDecimal(t, "50"),
		ExchangeRateEvidence: &evidence,
		ConversionStatus:     ConversionStatusConverted,
	}
	if err := amount.Validate(); err != nil {
		t.Fatalf("expected valid converted amount: %v", err)
	}

	var sameCurrencyAmount = ConvertedActivityAmount{
		SourceID:           "usd-buy-1",
		AmountKind:         ConvertedAmountKindGrossValue,
		OriginalCurrency:   "USD",
		OriginalAmount:     mustReportDecimal(t, "100"),
		ReportBaseCurrency: ReportBaseCurrencyUSD,
		ConvertedAmount:    mustReportDecimal(t, "100"),
		ConversionStatus:   ConversionStatusSameCurrency,
	}
	if err := sameCurrencyAmount.Validate(); err != nil {
		t.Fatalf("expected valid same-currency amount: %v", err)
	}

	var entry = ConversionAuditEntry{
		SourceID:           "eur-buy-1",
		AssetLabel:         "BTC",
		ActivityDate:       activityDate,
		SourceCurrency:     "EUR",
		ReportBaseCurrency: ReportBaseCurrencyUSD,
		RateDate:           activityDate,
		RateAuthority:      ExchangeRateAuthorityFederalReserve,
		RateKind:           evidence.RateKind,
		RateValue:          mustReportDecimal(t, "2"),
		QuoteDirection:     ExchangeRateQuoteDirectionSourcePerBase,
		Amounts:            []ConvertedActivityAmount{amount},
	}
	if err := entry.Validate(); err != nil {
		t.Fatalf("expected valid audit entry: %v", err)
	}
	if !entry.matchesExchangeRateEvidence(evidence) {
		t.Fatalf("expected retained provider authority and rate kind to remain part of audit evidence matching")
	}

	var invalidEvidence = evidence
	invalidEvidence.SourceCurrency = "USD"
	if err := invalidEvidence.Validate(); err == nil || !strings.Contains(err.Error(), "must differ") {
		t.Fatalf("expected same source/base evidence rejection, got %v", err)
	}
	invalidEvidence = evidence
	invalidEvidence.SourceCurrency = "USD"
	invalidEvidence.BaseCurrency = ReportBaseCurrencyEUR
	if err := invalidEvidence.Validate(); err == nil || !strings.Contains(err.Error(), "provider does not match") {
		t.Fatalf("expected provider/base mismatch rejection, got %v", err)
	}
	invalidEvidence = evidence
	invalidEvidence.ActivityDate = time.Time{}
	if err := invalidEvidence.Validate(); err == nil || !strings.Contains(err.Error(), "activity date") {
		t.Fatalf("expected missing evidence activity date rejection, got %v", err)
	}
	invalidEvidence = evidence
	invalidEvidence.RateDate = activityDate.AddDate(0, 0, 1)
	if err := invalidEvidence.Validate(); err == nil || !strings.Contains(err.Error(), "must not be after") {
		t.Fatalf("expected future rate date rejection, got %v", err)
	}
	invalidEvidence = evidence
	invalidEvidence.DatasetReference = " "
	if err := invalidEvidence.Validate(); err == nil || !strings.Contains(err.Error(), "dataset reference") {
		t.Fatalf("expected missing dataset reference rejection, got %v", err)
	}
	invalidEvidence = evidence
	invalidEvidence.SourceCurrency = " "
	if err := invalidEvidence.Validate(); err == nil || !strings.Contains(err.Error(), "source currency is required") {
		t.Fatalf("expected missing evidence source currency rejection, got %v", err)
	}
	invalidEvidence = evidence
	invalidEvidence.BaseCurrency = ReportBaseCurrency("GBP")
	if err := invalidEvidence.Validate(); err == nil || !strings.Contains(err.Error(), "base currency") {
		t.Fatalf("expected unsupported evidence base currency rejection, got %v", err)
	}
	invalidEvidence = evidence
	invalidEvidence.RateDate = time.Time{}
	if err := invalidEvidence.Validate(); err == nil || !strings.Contains(err.Error(), "rate date") {
		t.Fatalf("expected missing evidence rate date rejection, got %v", err)
	}
	invalidEvidence = evidence
	invalidEvidence.Authority = ExchangeRateAuthority("market")
	if err := invalidEvidence.Validate(); err == nil || !strings.Contains(err.Error(), "authority") {
		t.Fatalf("expected unsupported evidence authority rejection, got %v", err)
	}
	invalidEvidence = evidence
	invalidEvidence.ProviderID = ExchangeRateProviderID("market")
	if err := invalidEvidence.Validate(); err == nil || !strings.Contains(err.Error(), "provider") {
		t.Fatalf("expected unsupported evidence provider rejection, got %v", err)
	}
	invalidEvidence = evidence
	invalidEvidence.RateKind = " "
	if err := invalidEvidence.Validate(); err == nil || !strings.Contains(err.Error(), "rate kind") {
		t.Fatalf("expected missing evidence rate kind rejection, got %v", err)
	}
	invalidEvidence = evidence
	invalidEvidence.QuoteDirection = ExchangeRateQuoteDirection("ambiguous")
	if err := invalidEvidence.Validate(); err == nil || !strings.Contains(err.Error(), "quote direction") {
		t.Fatalf("expected unsupported evidence quote direction rejection, got %v", err)
	}
	invalidEvidence = evidence
	invalidEvidence.RateValue = mustReportDecimal(t, "0")
	if err := invalidEvidence.Validate(); err == nil || !strings.Contains(err.Error(), "rate value") {
		t.Fatalf("expected non-positive evidence rate rejection, got %v", err)
	}

	var invalidAmount = sameCurrencyAmount
	invalidAmount.SourceID = " "
	if err := invalidAmount.Validate(); err == nil || !strings.Contains(err.Error(), "source ID") {
		t.Fatalf("expected missing amount source ID rejection, got %v", err)
	}
	invalidAmount = sameCurrencyAmount
	invalidAmount.AmountKind = ConvertedAmountKind("unknown")
	if err := invalidAmount.Validate(); err == nil || !strings.Contains(err.Error(), "amount kind") {
		t.Fatalf("expected unsupported amount kind rejection, got %v", err)
	}
	invalidAmount = sameCurrencyAmount
	invalidAmount.OriginalCurrency = " "
	if err := invalidAmount.Validate(); err == nil || !strings.Contains(err.Error(), "original currency") {
		t.Fatalf("expected missing original currency rejection, got %v", err)
	}
	invalidAmount = sameCurrencyAmount
	invalidAmount.ReportBaseCurrency = ReportBaseCurrency("GBP")
	if err := invalidAmount.Validate(); err == nil || !strings.Contains(err.Error(), "report base currency") {
		t.Fatalf("expected unsupported amount base currency rejection, got %v", err)
	}
	invalidAmount = sameCurrencyAmount
	invalidAmount.OriginalAmount = reportInvalidDecimal
	if err := invalidAmount.Validate(); err == nil || !strings.Contains(err.Error(), "original amount") {
		t.Fatalf("expected invalid original amount rejection, got %v", err)
	}
	invalidAmount = sameCurrencyAmount
	invalidAmount.ConvertedAmount = reportInvalidDecimal
	if err := invalidAmount.Validate(); err == nil || !strings.Contains(err.Error(), "converted amount") {
		t.Fatalf("expected invalid converted amount rejection, got %v", err)
	}
	invalidAmount = sameCurrencyAmount
	invalidAmount.ConversionStatus = ConversionStatus("unknown")
	if err := invalidAmount.Validate(); err == nil || !strings.Contains(err.Error(), "conversion status") {
		t.Fatalf("expected unsupported conversion status rejection, got %v", err)
	}
	invalidAmount = sameCurrencyAmount
	invalidAmount.OriginalCurrency = "EUR"
	if err := invalidAmount.Validate(); err == nil || !strings.Contains(err.Error(), "must match") {
		t.Fatalf("expected same-currency source mismatch rejection, got %v", err)
	}
	invalidAmount = sameCurrencyAmount
	invalidAmount.ExchangeRateEvidence = &evidence
	if err := invalidAmount.Validate(); err == nil || !strings.Contains(err.Error(), "must not include") {
		t.Fatalf("expected same-currency evidence rejection, got %v", err)
	}
	invalidAmount = sameCurrencyAmount
	invalidAmount.ConvertedAmount = mustReportDecimal(t, "99")
	if err := invalidAmount.Validate(); err == nil || !strings.Contains(err.Error(), "must equal") {
		t.Fatalf("expected same-currency amount mismatch rejection, got %v", err)
	}
	invalidAmount = amount
	invalidAmount.ExchangeRateEvidence = nil
	if err := invalidAmount.Validate(); err == nil || !strings.Contains(err.Error(), "evidence is required") {
		t.Fatalf("expected missing converted evidence rejection, got %v", err)
	}
	invalidAmount = amount
	invalidAmount.OriginalCurrency = "USD"
	if err := invalidAmount.Validate(); err == nil || !strings.Contains(err.Error(), "must differ") {
		t.Fatalf("expected converted same-currency rejection, got %v", err)
	}
	invalidAmount = amount
	var mismatchedEvidence = evidence
	mismatchedEvidence.SourceCurrency = "GBP"
	invalidAmount.ExchangeRateEvidence = &mismatchedEvidence
	if err := invalidAmount.Validate(); err == nil || !strings.Contains(err.Error(), "source currency mismatch") {
		t.Fatalf("expected evidence source mismatch rejection, got %v", err)
	}
	invalidAmount = amount
	mismatchedEvidence = evidence
	mismatchedEvidence.BaseCurrency = ReportBaseCurrencyEUR
	invalidAmount.ReportBaseCurrency = ReportBaseCurrencyUSD
	invalidAmount.ExchangeRateEvidence = &mismatchedEvidence
	if err := invalidAmount.Validate(); err == nil || !strings.Contains(err.Error(), "exchange-rate evidence") {
		t.Fatalf("expected evidence base mismatch rejection, got %v", err)
	}
	invalidAmount = amount
	invalidAmount.OriginalCurrency = "GBP"
	mismatchedEvidence = evidence
	mismatchedEvidence.SourceCurrency = "GBP"
	mismatchedEvidence.BaseCurrency = ReportBaseCurrencyEUR
	mismatchedEvidence.Authority = ExchangeRateAuthorityEuropeanCentralBank
	mismatchedEvidence.ProviderID = ExchangeRateProviderIDECBEXR
	invalidAmount.ExchangeRateEvidence = &mismatchedEvidence
	if err := invalidAmount.Validate(); err == nil || !strings.Contains(err.Error(), "base currency mismatch") {
		t.Fatalf("expected valid-evidence amount base mismatch rejection, got %v", err)
	}

	var invalidEntry = entry
	invalidEntry.SourceID = " "
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "source ID") {
		t.Fatalf("expected audit missing source ID rejection, got %v", err)
	}
	invalidEntry = entry
	invalidEntry.AssetLabel = " "
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "asset label") {
		t.Fatalf("expected audit missing asset label rejection, got %v", err)
	}
	invalidEntry = entry
	invalidEntry.ActivityDate = time.Time{}
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "activity date") {
		t.Fatalf("expected audit missing activity date rejection, got %v", err)
	}
	invalidEntry = entry
	invalidEntry.SourceCurrency = " "
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "source currency") {
		t.Fatalf("expected audit missing source currency rejection, got %v", err)
	}
	invalidEntry = entry
	invalidEntry.ReportBaseCurrency = ReportBaseCurrency("GBP")
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "base currency") {
		t.Fatalf("expected audit unsupported base currency rejection, got %v", err)
	}
	invalidEntry = entry
	invalidEntry.RateDate = time.Time{}
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "rate date") {
		t.Fatalf("expected audit missing rate-date rejection, got %v", err)
	}
	invalidEntry = entry
	invalidEntry.RateAuthority = ExchangeRateAuthority("market")
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "authority") {
		t.Fatalf("expected audit unsupported authority rejection, got %v", err)
	}
	invalidEntry = entry
	invalidEntry.RateKind = " "
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "rate kind") {
		t.Fatalf("expected audit missing rate kind rejection, got %v", err)
	}
	invalidEntry = entry
	invalidEntry.RateKind = "different retained provider rate kind"
	var mismatchedAmount = amount
	mismatchedAmount.ExchangeRateEvidence = &evidence
	invalidEntry.Amounts = []ConvertedActivityAmount{mismatchedAmount}
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "exchange-rate evidence mismatch") {
		t.Fatalf("expected retained audit rate-kind mismatch rejection, got %v", err)
	}
	invalidEntry = entry
	invalidEntry.RateValue = mustReportDecimal(t, "0")
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "rate value") {
		t.Fatalf("expected audit non-positive rate rejection, got %v", err)
	}
	invalidEntry = entry
	invalidEntry.QuoteDirection = ExchangeRateQuoteDirection("ambiguous")
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "quote direction") {
		t.Fatalf("expected audit unsupported quote direction rejection, got %v", err)
	}
	invalidEntry = entry
	invalidEntry.SourceCurrency = "USD"
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "must differ") {
		t.Fatalf("expected audit source/base mismatch rejection, got %v", err)
	}
	invalidEntry = entry
	invalidEntry.RateDate = activityDate.AddDate(0, 0, 1)
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "must not be after") {
		t.Fatalf("expected audit future rate-date rejection, got %v", err)
	}
	invalidEntry = entry
	invalidEntry.Amounts = nil
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "amounts are required") {
		t.Fatalf("expected audit missing amounts rejection, got %v", err)
	}
	invalidEntry = entry
	invalidEntry.Amounts = []ConvertedActivityAmount{sameCurrencyAmount}
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "amount must be converted") {
		t.Fatalf("expected audit same-currency amount rejection, got %v", err)
	}
	invalidEntry = entry
	mismatchedAmount = amount
	mismatchedAmount.SourceID = "other"
	invalidEntry.Amounts = []ConvertedActivityAmount{mismatchedAmount}
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "source ID mismatch") {
		t.Fatalf("expected audit source ID mismatch rejection, got %v", err)
	}
	invalidEntry = entry
	mismatchedAmount = amount
	mismatchedAmount.OriginalCurrency = "GBP"
	mismatchedEvidence = evidence
	mismatchedEvidence.SourceCurrency = "GBP"
	mismatchedAmount.ExchangeRateEvidence = &mismatchedEvidence
	invalidEntry.Amounts = []ConvertedActivityAmount{mismatchedAmount}
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "source currency mismatch") {
		t.Fatalf("expected audit source currency mismatch rejection, got %v", err)
	}
	invalidEntry = entry
	invalidEntry.SourceCurrency = "GBP"
	mismatchedAmount = amount
	mismatchedAmount.ReportBaseCurrency = ReportBaseCurrencyEUR
	mismatchedAmount.OriginalCurrency = "GBP"
	mismatchedEvidence = evidence
	mismatchedEvidence.SourceCurrency = "GBP"
	mismatchedEvidence.BaseCurrency = ReportBaseCurrencyEUR
	mismatchedEvidence.Authority = ExchangeRateAuthorityEuropeanCentralBank
	mismatchedEvidence.ProviderID = ExchangeRateProviderIDECBEXR
	mismatchedAmount.ExchangeRateEvidence = &mismatchedEvidence
	invalidEntry.Amounts = []ConvertedActivityAmount{mismatchedAmount}
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "report base currency") {
		t.Fatalf("expected audit base currency mismatch rejection, got %v", err)
	}
	invalidEntry = entry
	mismatchedAmount = amount
	mismatchedEvidence = evidence
	mismatchedEvidence.RateValue = mustReportDecimal(t, "3")
	mismatchedAmount.ExchangeRateEvidence = &mismatchedEvidence
	invalidEntry.Amounts = []ConvertedActivityAmount{mismatchedAmount}
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "exchange-rate evidence mismatch") {
		t.Fatalf("expected audit evidence mismatch rejection, got %v", err)
	}

	invalidEntry = entry
	invalidAmount = amount
	invalidAmount.AmountKind = ConvertedAmountKind("bad")
	invalidEntry.Amounts = []ConvertedActivityAmount{invalidAmount}
	if err := invalidEntry.Validate(); err == nil || !strings.Contains(err.Error(), "amount 0") {
		t.Fatalf("expected audit invalid nested amount rejection, got %v", err)
	}

	var report = CapitalGainsReport{Year: 2024, CostBasisMethod: CostBasisMethodFIFO, GeneratedAt: activityDate, ReportCalculationCurrency: "USD", YearlyNetTotal: mustReportDecimal(t, "0")}
	invalidEvidence = evidence
	invalidEvidence.RateValue = mustReportDecimal(t, "0")
	report.RateSources = []ExchangeRateEvidence{invalidEvidence}
	if err := report.Validate(); err == nil || !strings.Contains(err.Error(), "rate source 0") {
		t.Fatalf("expected invalid report rate source rejection, got %v", err)
	}

	report.RateSources = []ExchangeRateEvidence{evidence}
	report.ReportCalculationCurrency = "EUR"
	if err := report.Validate(); err == nil || !strings.Contains(err.Error(), "base currency must match") {
		t.Fatalf("expected report rate-source currency mismatch rejection, got %v", err)
	}

	report.ReportCalculationCurrency = "USD"
	invalidEntry = entry
	invalidEntry.SourceID = " "
	report.ConversionAuditEntries = []ConversionAuditEntry{invalidEntry}
	if err := report.Validate(); err == nil || !strings.Contains(err.Error(), "conversion audit entry 0") {
		t.Fatalf("expected invalid report audit entry rejection, got %v", err)
	}

	report.ConversionAuditEntries = []ConversionAuditEntry{entry}
	report.RateSources = nil
	if err := report.Validate(); err == nil || !strings.Contains(err.Error(), "matching rate source") {
		t.Fatalf("expected missing report rate source rejection, got %v", err)
	}

	report.ReportCalculationCurrency = "NOT APPLICABLE"
	report.RateSources = []ExchangeRateEvidence{evidence}
	report.ConversionAuditEntries = nil
	if err := report.validateRateSourceCurrency(0, evidence); err != nil {
		t.Fatalf("expected NOT APPLICABLE report currency to skip rate-source currency validation: %v", err)
	}
	invalidEntry = entry
	invalidEntry.SourceCurrency = "GBP"
	if report.hasMatchingRateSource(invalidEntry) {
		t.Fatalf("expected report without audit source fields to report no direct match")
	}
}

// mustReportDecimal parses one decimal literal for report-model tests.
// Authored by: OpenCode
func mustReportDecimal(t *testing.T, raw string) apd.Decimal {
	t.Helper()

	var value, _, err = decimalsupport.ParseString(raw)
	if err != nil {
		t.Fatalf("parse decimal %q: %v", raw, err)
	}

	return value
}

// decimalPointer returns one decimal pointer for report-model tests.
// Authored by: OpenCode
func decimalPointer(t *testing.T, raw string) *apd.Decimal {
	t.Helper()

	var value = mustReportDecimal(t, raw)
	return &value
}

// assertOptionalDecimalString verifies one optional decimal's canonical value.
// Authored by: OpenCode
func assertOptionalDecimalString(t *testing.T, value *apd.Decimal, expected string) {
	t.Helper()

	var actual, err = decimalsupport.CanonicalStringPointer(value)
	if err != nil {
		t.Fatalf("canonical optional decimal string: %v", err)
	}
	if actual != expected {
		t.Fatalf("expected optional decimal %q, got %q", expected, actual)
	}
}

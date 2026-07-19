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

// TestRateProviderDisplayLabelsRemainPlainText verifies model display helpers do
// not include presentation-format markup.
// Authored by: OpenCode
func TestRateProviderDisplayLabelsRemainPlainText(t *testing.T) {
	t.Parallel()

	if got := RateProviderDisplayLabel(RateProviderIDECBEXR); got != "ECB Data Portal EXR" {
		t.Fatalf("unexpected ECB provider label: %q", got)
	}
	if got := RateProviderDisplayLabel(RateProviderID("custom-provider")); got != "custom-provider" {
		t.Fatalf("unexpected custom provider label: %q", got)
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

	_, err := NewReportRequest(0, CostBasisMethodFIFO, ReportBaseCurrencyUSD, ReportOutputFormatMarkdown, time.Now())
	if err == nil || !strings.Contains(err.Error(), "year must be greater than zero") {
		t.Fatalf("expected invalid year error, got %v", err)
	}

	_, err = NewReportRequest(2024, CostBasisMethod("bad"), ReportBaseCurrencyUSD, ReportOutputFormatMarkdown, time.Now())
	if err == nil || !strings.Contains(err.Error(), "unsupported cost basis method") {
		t.Fatalf("expected invalid method error, got %v", err)
	}

	_, err = NewReportRequest(2024, CostBasisMethodFIFO, ReportBaseCurrencyUSD, ReportOutputFormatMarkdown, time.Time{})
	if err == nil || !strings.Contains(err.Error(), "requested-at timestamp is required") {
		t.Fatalf("expected missing timestamp error, got %v", err)
	}

	_, err = NewReportRequest(2024, CostBasisMethodFIFO, ReportBaseCurrency("GBP"), ReportOutputFormatMarkdown, time.Now())
	if err == nil || !strings.Contains(err.Error(), "unsupported report base currency") {
		t.Fatalf("expected invalid report base currency error, got %v", err)
	}

	_, err = NewReportRequest(2024, CostBasisMethodFIFO, ReportBaseCurrencyUSD, ReportOutputFormat("html"), time.Now())
	if err == nil || !strings.Contains(err.Error(), "unsupported report output format") {
		t.Fatalf("expected invalid output format error, got %v", err)
	}

	request, err := NewReportRequest(2024, CostBasisMethodFIFO, ReportBaseCurrencyUSD, ReportOutputFormatMarkdown, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
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

			_, err := NewReportRequest(2024, CostBasisMethodFIFO, testCase.currency, ReportOutputFormatMarkdown, requestedAt)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("expected constructor error containing %q, got %v", testCase.want, err)
			}

			var request = ReportRequest{
				Year:               2024,
				CostBasisMethod:    CostBasisMethodFIFO,
				ReportBaseCurrency: testCase.currency,
				OutputFormat:       ReportOutputFormatMarkdown,
				RequestedAt:        requestedAt,
			}
			err = request.Validate()
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("expected validation error containing %q, got %v", testCase.want, err)
			}
		})
	}
}

// TestRenderLabels verifies closed user-facing labels for report presentation.
// Authored by: OpenCode
func TestRenderLabels(t *testing.T) {
	var conversionLabel, err = RenderConversionStatusLabel(ConversionStatusSameCurrency)
	if err != nil || conversionLabel != "Same currency" {
		t.Fatalf("same-currency label = %q, err = %v", conversionLabel, err)
	}
	conversionLabel, err = RenderConversionStatusLabel(ConversionStatusConverted)
	if err != nil || conversionLabel != "Converted" {
		t.Fatalf("converted label = %q, err = %v", conversionLabel, err)
	}
	if _, err = RenderConversionStatusLabel(ConversionStatus("same_currency_new")); err == nil {
		t.Fatalf("expected unsupported conversion status label to fail")
	}

	var quoteLabel string
	quoteLabel, err = RenderQuoteDirectionLabel(QuoteDirectionSourcePerBase)
	if err != nil || quoteLabel != "Source currency per base currency" {
		t.Fatalf("source-per-base quote label = %q, err = %v", quoteLabel, err)
	}
	quoteLabel, err = RenderQuoteDirectionLabel(QuoteDirectionBasePerSource)
	if err != nil || quoteLabel != "Base currency per source currency" {
		t.Fatalf("base-per-source quote label = %q, err = %v", quoteLabel, err)
	}
	if _, err = RenderQuoteDirectionLabel(QuoteDirection("base_per_source_new")); err == nil {
		t.Fatalf("expected unsupported quote direction label to fail")
	}

	var activityLabel string
	activityLabel, err = RenderActivityTypeLabel(AssetActivityRow{
		ActivityType: ActivityTypeSell,
		UnitPrice:    decimalPointer(t, "0"),
		GrossValue:   decimalPointer(t, "0"),
		FeeAmount:    decimalPointer(t, "0"),
	})
	if err != nil || activityLabel != "BLOCKCHAIN OP" {
		t.Fatalf("zero-priced sell label = %q, err = %v", activityLabel, err)
	}
	activityLabel, err = RenderActivityTypeLabel(AssetActivityRow{ActivityType: ActivityTypeBuy})
	if err != nil || activityLabel != "BUY" {
		t.Fatalf("buy label = %q, err = %v", activityLabel, err)
	}
	var invalidDecimal apd.Decimal
	invalidDecimal.Form = apd.NaNSignaling
	if _, err = RenderActivityTypeLabel(AssetActivityRow{ActivityType: ActivityTypeSell, UnitPrice: &invalidDecimal}); err == nil || !strings.Contains(err.Error(), "zero-priced fields") {
		t.Fatalf("expected invalid zero-priced monetary field to fail, got %v", err)
	}
	if _, err = RenderAuditActivityTypeLabel(AuditActivityEntry{ActivityType: ActivityTypeSell, GrossValue: &invalidDecimal}); err == nil || !strings.Contains(err.Error(), "render audit activity type label zero-priced fields") {
		t.Fatalf("expected invalid audit gross value to fail, got %v", err)
	}
	if _, err = RenderAuditActivityTypeLabel(AuditActivityEntry{ActivityType: ActivityTypeSell, FeeAmount: &invalidDecimal}); err == nil || !strings.Contains(err.Error(), "render audit activity type label zero-priced fields") {
		t.Fatalf("expected invalid audit fee amount to fail, got %v", err)
	}
}

// TestReportOutputFormatContract verifies supported output formats, labels, and
// missing-format validation.
// Authored by: OpenCode
func TestReportOutputFormatContract(t *testing.T) {
	t.Parallel()

	var formats = SupportedReportOutputFormats()
	if len(formats) != 2 || formats[0] != ReportOutputFormatMarkdown || formats[1] != ReportOutputFormatPDF {
		t.Fatalf("unexpected supported output formats: %#v", formats)
	}
	if ReportOutputFormatMarkdown.Label() != "Markdown" || ReportOutputFormatPDF.Label() != "PDF" {
		t.Fatalf("unexpected output format labels")
	}
	if got := ReportOutputFormat("html").Label(); got != "" {
		t.Fatalf("expected unsupported format to have no label, got %q", got)
	}
	if err := (ReportRequest{Year: 2024, CostBasisMethod: CostBasisMethodFIFO, ReportBaseCurrency: ReportBaseCurrencyUSD, RequestedAt: time.Now()}).Validate(); err == nil || !strings.Contains(err.Error(), "output format") {
		t.Fatalf("expected missing output format validation error, got %v", err)
	}
}

// TestReportDocumentConstructorsCoverCompatibilityBranches verifies document role/content combinations.
// Authored by: OpenCode
func TestReportDocumentConstructorsCoverCompatibilityBranches(t *testing.T) {
	var generatedAt = time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC)

	var pdfAsMarkdownRole = ReportDocument{
		DocumentType:    ReportDocumentTypePDF,
		Role:            ReportDocumentRoleMain,
		Content:         []byte("%PDF-1.7"),
		Year:            2024,
		CostBasisMethod: CostBasisMethodFIFO,
		GeneratedAt:     generatedAt,
	}
	if err := pdfAsMarkdownRole.Validate(); err == nil || !strings.Contains(err.Error(), "pdf report document role") {
		t.Fatalf("expected PDF role compatibility failure, got %v", err)
	}

	var markdownAsCombined = ReportDocument{
		DocumentType:    ReportDocumentTypeMarkdown,
		Role:            ReportDocumentRoleCombined,
		Content:         []byte("# Report"),
		Year:            2024,
		CostBasisMethod: CostBasisMethodFIFO,
		GeneratedAt:     generatedAt,
	}
	if err := markdownAsCombined.Validate(); err == nil || !strings.Contains(err.Error(), "markdown report document role") {
		t.Fatalf("expected Markdown role compatibility failure, got %v", err)
	}

	var emptyPDF = ReportDocument{DocumentType: ReportDocumentTypePDF, Role: ReportDocumentRoleCombined, Year: 2024, CostBasisMethod: CostBasisMethodFIFO, GeneratedAt: generatedAt}
	if err := emptyPDF.Validate(); err == nil || !strings.Contains(err.Error(), "PDF content") {
		t.Fatalf("expected empty PDF content failure, got %v", err)
	}
}

// TestReportOutputFileConstructorsCoverCompatibilityBranches verifies output metadata guardrails.
// Authored by: OpenCode
func TestReportOutputFileConstructorsCoverCompatibilityBranches(t *testing.T) {
	var savedAt = time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC)
	var dir = t.TempDir()
	var path = dir + "/report.md"

	var invalidRole = ReportOutputFile{DocumentsDirectory: dir, Filename: "report.md", Path: path, Role: ReportDocumentRole("bad"), MediaType: ReportMediaTypeMarkdown, SavedAt: savedAt}
	if err := invalidRole.Validate(); err == nil || !strings.Contains(err.Error(), "report output role") {
		t.Fatalf("expected invalid role failure, got %v", err)
	}
	var invalidMedia = ReportOutputFile{DocumentsDirectory: dir, Filename: "report.md", Path: path, Role: ReportDocumentRoleMain, MediaType: "text/plain", SavedAt: savedAt}
	if err := invalidMedia.Validate(); err == nil || !strings.Contains(err.Error(), "unsupported report output media type") {
		t.Fatalf("expected invalid media failure, got %v", err)
	}
	var relErrorPath = ReportOutputFile{DocumentsDirectory: string([]byte{0}), Filename: "report.md", Path: path, Role: ReportDocumentRoleMain, MediaType: ReportMediaTypeMarkdown, SavedAt: savedAt}
	if err := relErrorPath.Validate(); err == nil || !strings.Contains(err.Error(), "inside documents directory") {
		t.Fatalf("expected path relation failure, got %v", err)
	}
}

// TestNewCapitalGainsReportValidatesNestedContent verifies that the top-level
// report helper rejects invalid nested rows.
// Authored by: OpenCode
func TestNewCapitalGainsReportValidatesNestedContent(t *testing.T) {
	t.Parallel()

	request, err := NewReportRequest(2024, CostBasisMethodHIFO, ReportBaseCurrencyUSD, ReportOutputFormatMarkdown, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}

	_, err = NewCapitalGainsReport(
		request,
		time.Date(2026, time.May, 21, 12, 0, 0, 0, time.UTC),
		"USD",
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
		"USD",
		[]AssetSummaryEntry{{
			AssetIdentityKey:          "asset-btc",
			DisplayLabel:              "BTC",
			NetGainOrLoss:             mustReportDecimal(t, "1"),
			ReportCalculationCurrency: "USD",
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

// TestNewCapitalGainsReportRequiresSupportedCalculationCurrency verifies that
// calculated reports always retain the selected report base currency.
// Authored by: OpenCode
func TestNewCapitalGainsReportRequiresSupportedCalculationCurrency(t *testing.T) {
	t.Parallel()

	var request, requestErr = NewReportRequest(2024, CostBasisMethodFIFO, ReportBaseCurrencyUSD, ReportOutputFormatMarkdown, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
	if requestErr != nil {
		t.Fatalf("new report request: %v", requestErr)
	}

	var testCases = []struct {
		name     string
		currency string
	}{
		{name: "empty", currency: ""},
		{name: "not applicable", currency: "NOT APPLICABLE"},
		{name: "unsupported", currency: "GBP"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var _, err = NewCapitalGainsReport(
				request,
				time.Date(2026, time.May, 21, 12, 0, 0, 0, time.UTC),
				testCase.currency,
				nil,
				mustReportDecimal(t, "0"),
				nil,
				nil,
			)
			if err == nil || !strings.Contains(err.Error(), "capital gains report calculation currency") {
				t.Fatalf("expected calculation-currency rejection, got %v", err)
			}
		})
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
	var request, requestErr = NewReportRequest(2024, CostBasisMethodFIFO, ReportBaseCurrencyUSD, ReportOutputFormatMarkdown, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
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

	_, err := NewReportDocument(ReportDocumentType("html"), ReportDocumentRoleMain, []byte("# Report\n"), 2024, CostBasisMethodFIFO, time.Now())
	if err == nil || !strings.Contains(err.Error(), "unsupported report document type") {
		t.Fatalf("expected invalid document type error, got %v", err)
	}

	document, err := NewReportDocument(ReportDocumentTypeMarkdown, ReportDocumentRoleMain, []byte("# Report\n"), 2024, CostBasisMethodFIFO, time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report document: %v", err)
	}
	if err = document.Validate(); err != nil {
		t.Fatalf("validate report document: %v", err)
	}

	_, err = NewReportOutputFile("/tmp/Documents", "report.md", "/tmp/report.md", ReportDocumentRoleMain, ReportMediaTypeMarkdown, time.Now())
	if err == nil || !strings.Contains(err.Error(), "inside documents directory") {
		t.Fatalf("expected path scope validation failure, got %v", err)
	}

	outputFile, err := NewReportOutputFile("/tmp/Documents", "report.md", "/tmp/Documents/report.md", ReportDocumentRoleMain, ReportMediaTypeMarkdown, time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report output file: %v", err)
	}
	if err = outputFile.Validate(); err != nil {
		t.Fatalf("validate report output file: %v", err)
	}

	pdfDocument, err := NewReportDocument(ReportDocumentTypePDF, ReportDocumentRoleCombined, []byte("%PDF-1.7"), 2024, CostBasisMethodFIFO, time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new PDF report document: %v", err)
	}
	if err = pdfDocument.Validate(); err != nil {
		t.Fatalf("validate PDF report document: %v", err)
	}
	var payload = []byte("%PDF-1.7")
	var copiedDocument, copyErr = NewReportDocument(ReportDocumentTypePDF, ReportDocumentRoleCombined, payload, 2024, CostBasisMethodFIFO, time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC))
	if copyErr != nil {
		t.Fatalf("new copied PDF report document: %v", copyErr)
	}
	payload[0] = '!'
	if string(copiedDocument.Content) != "%PDF-1.7" {
		t.Fatalf("expected report document content to be independent of input mutation, got %q", copiedDocument.Content)
	}

	_, err = NewReportDocument(ReportDocumentTypePDF, ReportDocumentRoleMain, []byte("%PDF-1.7"), 2024, CostBasisMethodFIFO, time.Now())
	if err == nil || !strings.Contains(err.Error(), "pdf report document role must be combined") {
		t.Fatalf("expected PDF role validation failure, got %v", err)
	}
}

// TestValidateRenderedDocumentsValidatesBundleShapeAndMetadata verifies that
// output formats reject incomplete document roles and inconsistent report
// metadata before output files are reserved.
// Authored by: OpenCode
func TestValidateRenderedDocumentsValidatesBundleShapeAndMetadata(t *testing.T) {
	t.Parallel()

	var generatedAt = time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC)
	var main, mainErr = NewReportDocument(ReportDocumentTypeMarkdown, ReportDocumentRoleMain, []byte("# Report\n"), 2024, CostBasisMethodFIFO, generatedAt)
	if mainErr != nil {
		t.Fatalf("new main document: %v", mainErr)
	}
	var annex, annexErr = NewReportDocument(ReportDocumentTypeMarkdown, ReportDocumentRoleAnnex, []byte("# Annex\n"), 2024, CostBasisMethodFIFO, generatedAt)
	if annexErr != nil {
		t.Fatalf("new annex document: %v", annexErr)
	}
	var pdf, pdfErr = NewReportDocument(ReportDocumentTypePDF, ReportDocumentRoleCombined, []byte("%PDF-1.7"), 2024, CostBasisMethodFIFO, generatedAt)
	if pdfErr != nil {
		t.Fatalf("new PDF document: %v", pdfErr)
	}

	var testCases = []struct {
		name         string
		outputFormat ReportOutputFormat
		documents    []ReportDocument
		want         string
	}{
		{name: "invalid format", outputFormat: ReportOutputFormat("html"), documents: []ReportDocument{main}, want: "unsupported report output format"},
		{name: "markdown count", outputFormat: ReportOutputFormatMarkdown, documents: []ReportDocument{main}, want: "exactly two documents"},
		{name: "markdown main", outputFormat: ReportOutputFormatMarkdown, documents: []ReportDocument{annex, annex}, want: "document 0 must be the main"},
		{name: "markdown annex", outputFormat: ReportOutputFormatMarkdown, documents: []ReportDocument{main, main}, want: "document 1 must be the Annex"},
		{name: "pdf count", outputFormat: ReportOutputFormatPDF, documents: []ReportDocument{pdf, pdf}, want: "exactly one document"},
		{name: "pdf combined", outputFormat: ReportOutputFormatPDF, documents: []ReportDocument{main}, want: "must be the combined PDF document"},
		{name: "year mismatch", outputFormat: ReportOutputFormatMarkdown, documents: []ReportDocument{main, func() ReportDocument { var document = annex; document.Year = 2025; return document }()}, want: "year does not match"},
		{name: "method mismatch", outputFormat: ReportOutputFormatMarkdown, documents: []ReportDocument{main, func() ReportDocument {
			var document = annex
			document.CostBasisMethod = CostBasisMethodLIFO
			return document
		}()}, want: "cost basis method does not match"},
		{name: "timestamp mismatch", outputFormat: ReportOutputFormatMarkdown, documents: []ReportDocument{main, func() ReportDocument {
			var document = annex
			document.GeneratedAt = generatedAt.Add(time.Second)
			return document
		}()}, want: "generated-at timestamp does not match"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var err = ValidateRenderedDocuments(testCase.outputFormat, testCase.documents)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("expected error containing %q, got %v", testCase.want, err)
			}
		})
	}

	if err := ValidateRenderedDocuments(ReportOutputFormatMarkdown, []ReportDocument{main, annex}); err != nil {
		t.Fatalf("validate Markdown documents: %v", err)
	}
	if err := ValidateRenderedDocuments(ReportOutputFormatPDF, []ReportDocument{pdf}); err != nil {
		t.Fatalf("validate PDF document: %v", err)
	}
}

// TestReportDocumentRequiresExplicitMarkdownRole verifies bundle documents do
// not infer a compatibility role from their type.
// Authored by: OpenCode
func TestReportDocumentRequiresExplicitMarkdownRole(t *testing.T) {
	t.Parallel()

	var document = ReportDocument{
		DocumentType:    ReportDocumentTypeMarkdown,
		Content:         []byte("# Report\n"),
		Year:            2024,
		CostBasisMethod: CostBasisMethodFIFO,
		GeneratedAt:     time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC),
	}
	if err := document.Validate(); err == nil {
		t.Fatal("expected blank Markdown role to fail validation")
	}
}

// TestAuditAnnexAndOutputBundleValidation verifies the foundational Annex 1
// shell and saved-output bundle contracts.
// Authored by: OpenCode
func TestAuditAnnexAndOutputBundleValidation(t *testing.T) {
	t.Parallel()

	var annex = DefaultAuditAnnex()
	if err := annex.Validate(); err != nil {
		t.Fatalf("validate default audit annex: %v", err)
	}
	if annex.Title != AuditAnnexTitle() {
		t.Fatalf("unexpected audit annex title: %q", annex.Title)
	}
	annex.SectionOrder[0] = AuditAnnexSectionCurrencyConversionAudit
	if err := DefaultAuditAnnex().Validate(); err != nil {
		t.Fatalf("default audit annex must not share section-order backing array: %v", err)
	}

	_, err := NewAuditAnnex("Audit", RequiredAuditAnnexSectionOrder())
	if err == nil || !strings.Contains(err.Error(), "audit annex title") {
		t.Fatalf("expected title validation failure, got %v", err)
	}
	_, err = NewAuditAnnex(AuditAnnexTitle(), []AuditAnnexSection{AuditAnnexSectionCurrencyConversionAudit, AuditAnnexSectionPerAssetReport})
	if err == nil || !strings.Contains(err.Error(), "section order") {
		t.Fatalf("expected section order validation failure, got %v", err)
	}

	var savedAt = time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC)
	var mainFile, mainErr = NewReportOutputFile("/tmp/Documents", "report.md", "/tmp/Documents/report.md", ReportDocumentRoleMain, ReportMediaTypeMarkdown, savedAt)
	if mainErr != nil {
		t.Fatalf("new main output file: %v", mainErr)
	}
	var annexFile, annexErr = NewReportOutputFile("/tmp/Documents", "report-annex-1.md", "/tmp/Documents/report-annex-1.md", ReportDocumentRoleAnnex, ReportMediaTypeMarkdown, savedAt)
	if annexErr != nil {
		t.Fatalf("new annex output file: %v", annexErr)
	}
	var pdfFile, pdfErr = NewReportOutputFile("/tmp/Documents", "report.pdf", "/tmp/Documents/report.pdf", ReportDocumentRoleCombined, ReportMediaTypePDF, savedAt)
	if pdfErr != nil {
		t.Fatalf("new PDF output file: %v", pdfErr)
	}

	if _, err = NewReportOutputBundle(ReportOutputFormatMarkdown, []ReportOutputFile{mainFile, annexFile}, savedAt, false, ""); err != nil {
		t.Fatalf("new Markdown output bundle: %v", err)
	}
	if _, err = NewReportOutputBundle(ReportOutputFormatPDF, []ReportOutputFile{pdfFile}, savedAt, true, "open failed"); err != nil {
		t.Fatalf("new PDF output bundle with open warning: %v", err)
	}
	if _, err = NewReportOutputBundle(ReportOutputFormatMarkdown, []ReportOutputFile{mainFile}, savedAt, false, ""); err == nil || !strings.Contains(err.Error(), "exactly two files") {
		t.Fatalf("expected Markdown bundle file-count failure, got %v", err)
	}
	if _, err = NewReportOutputBundle(ReportOutputFormatPDF, []ReportOutputFile{mainFile}, savedAt, false, ""); err == nil || !strings.Contains(err.Error(), "combined") {
		t.Fatalf("expected PDF bundle role failure, got %v", err)
	}
	if _, err = NewReportOutputBundle(ReportOutputFormatPDF, []ReportOutputFile{pdfFile}, savedAt, false, "open failed"); err == nil || !strings.Contains(err.Error(), "open error requires an open request") {
		t.Fatalf("expected bundle open error validation failure, got %v", err)
	}
}

// TestReportDocumentValidationGuardrails verifies remaining document and report
// constructor validation branches.
// Authored by: OpenCode
func TestReportDocumentValidationGuardrails(t *testing.T) {
	t.Parallel()

	request, err := NewReportRequest(2024, CostBasisMethodFIFO, ReportBaseCurrencyUSD, ReportOutputFormatMarkdown, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
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

	_, err = NewReportDocument(ReportDocumentTypeMarkdown, ReportDocumentRoleMain, []byte("   "), 2024, CostBasisMethodFIFO, time.Now())
	if err == nil || !strings.Contains(err.Error(), "report document content is required") {
		t.Fatalf("expected missing content error, got %v", err)
	}

	_, err = NewReportDocument(ReportDocumentTypeMarkdown, ReportDocumentRoleMain, []byte("# Report\n"), 2024, CostBasisMethod("bad"), time.Now())
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
	if err = (AssetActivityRow{SourceID: "sell-1", OccurredAt: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC), ActivityType: ActivityTypeSell, Quantity: mustReportDecimal(t, "1"), BasisAfterRow: mustReportDecimal(t, "0"), QuantityAfterRow: mustReportDecimal(t, "0"), ConversionStatus: ConversionStatus("unknown")}).Validate(); err == nil || !strings.Contains(err.Error(), "conversion status") {
		t.Fatalf("expected unsupported row conversion status to fail, got %v", err)
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

	var request, requestErr = NewReportRequest(2024, CostBasisMethodFIFO, ReportBaseCurrencyUSD, ReportOutputFormatMarkdown, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
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
	if err = (CapitalGainsReport{Year: 2024, CostBasisMethod: CostBasisMethodFIFO, GeneratedAt: time.Now(), ReportCalculationCurrency: "USD", YearlyNetTotal: mustReportDecimal(t, "0"), ReferenceEntries: []ReferenceLiquidationEntry{{}}}).Validate(); err == nil || !strings.Contains(err.Error(), "reference entry 0") {
		t.Fatalf("expected invalid nested reference entry to fail, got %v", err)
	}
	if err = (CapitalGainsReport{Year: 2024, CostBasisMethod: CostBasisMethodFIFO, GeneratedAt: time.Now(), ReportCalculationCurrency: "USD", YearlyNetTotal: mustReportDecimal(t, "0"), DetailSections: []AssetDetailSection{{}}}).Validate(); err == nil || !strings.Contains(err.Error(), "detail section 0") {
		t.Fatalf("expected invalid nested detail section to fail, got %v", err)
	}

	if err = (ReportDocument{DocumentType: ReportDocumentTypeMarkdown, Role: ReportDocumentRoleMain, Content: []byte("# Report\n"), Year: 0, CostBasisMethod: CostBasisMethodFIFO, GeneratedAt: time.Now()}).Validate(); err == nil || !strings.Contains(err.Error(), "report document year must be greater than zero") {
		t.Fatalf("expected missing document year to fail, got %v", err)
	}
	if err = (ReportDocument{DocumentType: ReportDocumentTypeMarkdown, Role: ReportDocumentRoleMain, Content: []byte("# Report\n"), Year: 2024, CostBasisMethod: CostBasisMethodFIFO}).Validate(); err == nil || !strings.Contains(err.Error(), "generated-at timestamp is required") {
		t.Fatalf("expected missing document timestamp to fail, got %v", err)
	}

	if err = (ReportOutputFile{DocumentsDirectory: "/tmp/docs", Filename: "report.md", Path: "/tmp/docs/report.md", Role: ReportDocumentRoleMain, MediaType: ReportMediaTypeMarkdown}).Validate(); err == nil || !strings.Contains(err.Error(), "saved-at timestamp is required") {
		t.Fatalf("expected missing output timestamp to fail, got %v", err)
	}
	if err = (ReportOutputFile{Filename: "report.md", Path: "/tmp/docs/report.md", Role: ReportDocumentRoleMain, MediaType: ReportMediaTypeMarkdown, SavedAt: time.Now()}).Validate(); err == nil || !strings.Contains(err.Error(), "documents directory") {
		t.Fatalf("expected missing output documents directory to fail, got %v", err)
	}
	if err = (ReportOutputFile{DocumentsDirectory: "/tmp/docs", Filename: "report.md", Role: ReportDocumentRoleMain, MediaType: ReportMediaTypeMarkdown, SavedAt: time.Now()}).Validate(); err == nil || !strings.Contains(err.Error(), "output path") {
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

// TestCloneConvertedActivityAmountsCopiesEvidencePointers verifies conversion
// amount cloning preserves nil evidence and deep-copies non-nil evidence.
// Authored by: OpenCode
func TestCloneConvertedActivityAmountsCopiesEvidencePointers(t *testing.T) {
	var activityDate = time.Date(2024, time.January, 5, 0, 0, 0, 0, time.UTC)
	var evidence = ExchangeRateEvidence{
		SourceCurrency:   "EUR",
		BaseCurrency:     ReportBaseCurrencyUSD,
		ActivityDate:     activityDate,
		RateDate:         activityDate,
		Authority:        RateAuthorityFederalReserve,
		ProviderID:       RateProviderIDFederalReserveH10,
		RateKind:         "noon buying rate",
		QuoteDirection:   QuoteDirectionBasePerSource,
		RateValue:        mustReportDecimal(t, "1.0957"),
		DatasetReference: "H10/RXI$US_N.B.EU",
	}
	var amounts = []ConvertedActivityAmount{
		{SourceID: "same", AmountKind: ConvertedAmountKindGrossValue, OriginalCurrency: "USD", OriginalAmount: mustReportDecimal(t, "1"), ReportBaseCurrency: ReportBaseCurrencyUSD, ConvertedAmount: mustReportDecimal(t, "1"), ConversionStatus: ConversionStatusSameCurrency},
		{SourceID: "converted", AmountKind: ConvertedAmountKindGrossValue, OriginalCurrency: "EUR", OriginalAmount: mustReportDecimal(t, "2"), ReportBaseCurrency: ReportBaseCurrencyUSD, ConvertedAmount: mustReportDecimal(t, "2.1914"), ExchangeRateEvidence: &evidence, ConversionStatus: ConversionStatusConverted},
	}

	var cloned = cloneConvertedActivityAmounts(amounts)
	if cloned[0].ExchangeRateEvidence != nil {
		t.Fatalf("expected nil evidence to remain nil")
	}
	if cloned[1].ExchangeRateEvidence == nil || cloned[1].ExchangeRateEvidence == amounts[1].ExchangeRateEvidence {
		t.Fatalf("expected non-nil evidence to be deep-copied")
	}
}

// TestCloneExchangeRateEvidenceCopiesRateValues verifies retained rate evidence
// cloning does not share decimal coefficient storage between copies.
// Authored by: OpenCode
func TestCloneExchangeRateEvidenceCopiesRateValues(t *testing.T) {
	t.Parallel()

	var activityDate = time.Date(2024, time.January, 5, 0, 0, 0, 0, time.UTC)
	var sources = []ExchangeRateEvidence{{
		SourceCurrency:   "EUR",
		BaseCurrency:     ReportBaseCurrencyUSD,
		ActivityDate:     activityDate,
		RateDate:         activityDate,
		Authority:        RateAuthorityFederalReserve,
		ProviderID:       RateProviderIDFederalReserveH10,
		RateKind:         "noon buying rate",
		QuoteDirection:   QuoteDirectionBasePerSource,
		RateValue:        mustReportDecimal(t, "123456789.987654321"),
		DatasetReference: "H10/RXI$US_N.B.EU",
	}}

	var cloned = cloneExchangeRateEvidence(sources)
	cloned[0].RateValue.SetInt64(2)

	if sources[0].RateValue.Cmp(&cloned[0].RateValue) == 0 {
		t.Fatalf("expected cloned rate value mutation not to affect source")
	}
}

// TestBasisMatchValidationGuardrails verifies the remaining basis-match
// guardrails.
// Authored by: OpenCode
func TestBasisMatchValidationGuardrails(t *testing.T) {
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
		Authority:        RateAuthorityFederalReserve,
		ProviderID:       RateProviderIDFederalReserveH10,
		RateKind:         "noon buying rate in New York for cable transfers payable in listed currencies",
		QuoteDirection:   QuoteDirectionSourcePerBase,
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
		RateAuthority:      RateAuthorityFederalReserve,
		RateKind:           evidence.RateKind,
		RateValue:          mustReportDecimal(t, "2"),
		QuoteDirection:     QuoteDirectionSourcePerBase,
		Amounts:            []ConvertedActivityAmount{amount},
	}
	if err := entry.Validate(); err != nil {
		t.Fatalf("expected valid audit entry: %v", err)
	}
	if !entry.matchesExchangeRateEvidence(evidence) {
		t.Fatalf("expected retained provider authority and rate kind to remain part of audit evidence matching")
	}
	var zeroAmount = amount
	zeroAmount.AmountKind = ConvertedAmountKindFeeAmount
	zeroAmount.OriginalAmount = mustReportDecimal(t, "0")
	zeroAmount.ConvertedAmount = mustReportDecimal(t, "0")
	var groupedEntryWithZeroSlot = entry
	groupedEntryWithZeroSlot.Amounts = []ConvertedActivityAmount{amount, zeroAmount}
	if err := groupedEntryWithZeroSlot.Validate(); err != nil {
		t.Fatalf("expected grouped audit entry with retained zero-to-zero amount slot to stay valid: %v", err)
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
	invalidEvidence.Authority = RateAuthority("market")
	if err := invalidEvidence.Validate(); err == nil || !strings.Contains(err.Error(), "authority") {
		t.Fatalf("expected unsupported evidence authority rejection, got %v", err)
	}
	invalidEvidence = evidence
	invalidEvidence.ProviderID = RateProviderID("market")
	if err := invalidEvidence.Validate(); err == nil || !strings.Contains(err.Error(), "provider") {
		t.Fatalf("expected unsupported evidence provider rejection, got %v", err)
	}
	invalidEvidence = evidence
	invalidEvidence.RateKind = " "
	if err := invalidEvidence.Validate(); err == nil || !strings.Contains(err.Error(), "rate kind") {
		t.Fatalf("expected missing evidence rate kind rejection, got %v", err)
	}
	invalidEvidence = evidence
	invalidEvidence.QuoteDirection = QuoteDirection("ambiguous")
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
	mismatchedEvidence.Authority = RateAuthorityEuropeanCentralBank
	mismatchedEvidence.ProviderID = RateProviderIDECBEXR
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
	invalidEntry.RateAuthority = RateAuthority("market")
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
	invalidEntry.QuoteDirection = QuoteDirection("ambiguous")
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
	var sameCalendarLaterInstantEntry = entry
	sameCalendarLaterInstantEntry.ActivityDate = time.Date(2024, time.January, 5, 9, 0, 0, 0, time.UTC)
	sameCalendarLaterInstantEntry.RateDate = time.Date(2024, time.January, 5, 17, 0, 0, 0, time.UTC)
	if err := sameCalendarLaterInstantEntry.validateRateEvidence(); err != nil {
		t.Fatalf("expected audit same-calendar later instant rate date to validate: %v", err)
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
	mismatchedEvidence.Authority = RateAuthorityEuropeanCentralBank
	mismatchedEvidence.ProviderID = RateProviderIDECBEXR
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

	var report = CapitalGainsReport{Year: 2024, CostBasisMethod: CostBasisMethodFIFO, GeneratedAt: activityDate, ReportCalculationCurrency: "USD", YearlyNetTotal: mustReportDecimal(t, "0"), AuditAnnex: DefaultAuditAnnex()}
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
	report.AuditAnnex.ConversionAuditEntries = []ConversionAuditEntry{invalidEntry}
	if err := report.Validate(); err == nil || !strings.Contains(err.Error(), "conversion audit entry 0") {
		t.Fatalf("expected invalid report audit entry rejection, got %v", err)
	}

	report.AuditAnnex.ConversionAuditEntries = []ConversionAuditEntry{entry}
	report.RateSources = nil
	if err := report.Validate(); err == nil || !strings.Contains(err.Error(), "matching rate source") {
		t.Fatalf("expected missing report rate source rejection, got %v", err)
	}

	report.RateSources = []ExchangeRateEvidence{evidence}
	report.AuditAnnex.ConversionAuditEntries = []ConversionAuditEntry{entry}
	report.DetailSections = []AssetDetailSection{{
		AssetIdentityKey:    "asset-btc",
		DisplayLabel:        "BTC",
		OpeningQuantity:     mustReportDecimal(t, "1"),
		OpeningCostBasis:    mustReportDecimal(t, "100"),
		ClosingQuantity:     mustReportDecimal(t, "0"),
		ClosingCostBasis:    mustReportDecimal(t, "0"),
		CalculationCurrency: "USD",
		ActivityRows: []AssetActivityRow{{
			SourceID:            "eur-buy-1",
			OccurredAt:          activityDate,
			ActivityType:        ActivityTypeSell,
			Quantity:            mustReportDecimal(t, "1"),
			GrossValue:          decimalPointer(t, "50"),
			ActivityCurrency:    "EUR",
			BasisAfterRow:       mustReportDecimal(t, "0"),
			CalculationCurrency: "USD",
			QuantityAfterRow:    mustReportDecimal(t, "0"),
			ConversionStatus:    ConversionStatusSameCurrency,
		}},
	}}
	if err := report.Validate(); err == nil || !strings.Contains(err.Error(), "must not be same-currency") {
		t.Fatalf("expected audited same-currency detail row contradiction rejection, got %v", err)
	}
	report.DetailSections[0].ActivityRows[0].ConversionStatus = ConversionStatusConverted
	if err := report.Validate(); err != nil {
		t.Fatalf("expected audited converted detail row to validate: %v", err)
	}

	report.ReportCalculationCurrency = ""
	report.RateSources = []ExchangeRateEvidence{evidence}
	report.AuditAnnex.ConversionAuditEntries = nil
	report.DetailSections = nil
	if err := report.Validate(); err == nil || !strings.Contains(err.Error(), "calculation currency") {
		t.Fatalf("expected empty report currency rejection, got %v", err)
	}

	report.ReportCalculationCurrency = "USD"
	var sameCalendarSource = evidence
	sameCalendarSource.ActivityDate = time.Date(2024, time.January, 5, 14, 30, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	sameCalendarSource.RateDate = time.Date(2024, time.January, 5, 8, 15, 0, 0, time.FixedZone("UTC-5", -5*60*60))
	var sameCalendarEntry = entry
	sameCalendarEntry.ActivityDate = time.Date(2024, time.January, 5, 9, 0, 0, 0, time.UTC)
	sameCalendarEntry.RateDate = time.Date(2024, time.January, 5, 7, 0, 0, 0, time.UTC)
	report.RateSources = []ExchangeRateEvidence{sameCalendarSource}
	report.AuditAnnex.ConversionAuditEntries = []ConversionAuditEntry{sameCalendarEntry}
	if !report.hasMatchingRateSource(sameCalendarEntry) {
		t.Fatalf("expected same calendar dates in different zones to match rate source")
	}
	invalidEntry = entry
	invalidEntry.SourceCurrency = "GBP"
	if report.hasMatchingRateSource(invalidEntry) {
		t.Fatalf("expected report without audit source fields to report no direct match")
	}
}

// TestReportModelCoverageGapBranches verifies narrow constructor and validation
// branches that are otherwise only reached by malformed renderer/output wiring.
// Authored by: OpenCode
func TestReportModelCoverageGapBranches(t *testing.T) {
	t.Parallel()

	var savedAt = time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC)
	var mainFile, mainErr = NewReportOutputFile("/tmp/Documents", "report.md", "/tmp/Documents/report.md", ReportDocumentRoleMain, ReportMediaTypeMarkdown, savedAt)
	if mainErr != nil {
		t.Fatalf("new main output file: %v", mainErr)
	}
	var annexFile, annexErr = NewReportOutputFile("/tmp/Documents", "report-annex.md", "/tmp/Documents/report-annex.md", ReportDocumentRoleAnnex, ReportMediaTypeMarkdown, savedAt)
	if annexErr != nil {
		t.Fatalf("new annex output file: %v", annexErr)
	}
	var pdfFile, pdfErr = NewReportOutputFile("/tmp/Documents", "report.pdf", "/tmp/Documents/report.pdf", ReportDocumentRoleCombined, ReportMediaTypePDF, savedAt)
	if pdfErr != nil {
		t.Fatalf("new PDF output file: %v", pdfErr)
	}

	if err := (ReportDocument{DocumentType: ReportDocumentTypePDF, Content: []byte("%PDF"), Year: 2024, CostBasisMethod: CostBasisMethodFIFO, GeneratedAt: savedAt}).Validate(); err == nil || !strings.Contains(err.Error(), "report document role") {
		t.Fatalf("expected missing PDF document role failure, got %v", err)
	}

	if err := (ReportOutputBundle{OutputFormat: ReportOutputFormat("html"), SavedAt: savedAt}).Validate(); err == nil || !strings.Contains(err.Error(), "report output bundle format") {
		t.Fatalf("expected invalid bundle format failure, got %v", err)
	}
	if err := (ReportOutputBundle{OutputFormat: ReportOutputFormatMarkdown, Files: []ReportOutputFile{mainFile, annexFile}}).Validate(); err == nil || !strings.Contains(err.Error(), "saved-at timestamp") {
		t.Fatalf("expected missing bundle saved-at failure, got %v", err)
	}
	var invalidFile = mainFile
	invalidFile.Path = ""
	if err := (ReportOutputBundle{OutputFormat: ReportOutputFormatMarkdown, Files: []ReportOutputFile{invalidFile, annexFile}, SavedAt: savedAt}).Validate(); err == nil || !strings.Contains(err.Error(), "file 0") {
		t.Fatalf("expected nested bundle file failure, got %v", err)
	}
	if err := (ReportOutputBundle{OutputFormat: ReportOutputFormat("html")}).validateFileShape(); err == nil || !strings.Contains(err.Error(), "unsupported report output format") {
		t.Fatalf("expected direct bundle shape format failure, got %v", err)
	}
	var wrongAnnexRole = annexFile
	wrongAnnexRole.Role = ReportDocumentRoleMain
	if err := (ReportOutputBundle{OutputFormat: ReportOutputFormatMarkdown, Files: []ReportOutputFile{mainFile, wrongAnnexRole}, SavedAt: savedAt}).Validate(); err == nil || !strings.Contains(err.Error(), "file 1 must be annex") {
		t.Fatalf("expected Markdown annex-role failure, got %v", err)
	}
	var wrongMainMedia = mainFile
	wrongMainMedia.MediaType = ReportMediaTypePDF
	if err := (ReportOutputBundle{OutputFormat: ReportOutputFormatMarkdown, Files: []ReportOutputFile{wrongMainMedia, annexFile}, SavedAt: savedAt}).Validate(); err == nil || !strings.Contains(err.Error(), "media type") {
		t.Fatalf("expected Markdown media type failure, got %v", err)
	}
	if err := (ReportOutputBundle{OutputFormat: ReportOutputFormatPDF, Files: []ReportOutputFile{pdfFile, pdfFile}, SavedAt: savedAt}).Validate(); err == nil || !strings.Contains(err.Error(), "exactly one file") {
		t.Fatalf("expected PDF file-count failure, got %v", err)
	}
	var wrongPDFMedia = pdfFile
	wrongPDFMedia.MediaType = ReportMediaTypeMarkdown
	if err := (ReportOutputBundle{OutputFormat: ReportOutputFormatPDF, Files: []ReportOutputFile{wrongPDFMedia}, SavedAt: savedAt}).Validate(); err == nil || !strings.Contains(err.Error(), "media type") {
		t.Fatalf("expected PDF media type failure, got %v", err)
	}
}

// TestCloneAuditAnnexCopiesNestedSlices verifies the audit-annex clone does not
// share mutable nested slices or optional decimal pointers with the source.
// Authored by: OpenCode
func TestCloneAuditAnnexCopiesNestedSlices(t *testing.T) {
	t.Parallel()

	var activityDate = time.Date(2024, time.January, 5, 0, 0, 0, 0, time.UTC)
	var unitPrice = decimalPointer(t, "10")
	var grossValue = decimalPointer(t, "10")
	var amount = ConvertedActivityAmount{
		SourceID:           "eur-buy-1",
		AmountKind:         ConvertedAmountKindGrossValue,
		OriginalCurrency:   "EUR",
		OriginalAmount:     mustReportDecimal(t, "100"),
		ReportBaseCurrency: ReportBaseCurrencyUSD,
		ConvertedAmount:    mustReportDecimal(t, "110"),
		ConversionStatus:   ConversionStatusConverted,
	}
	var annex = AuditAnnex{
		Title:        AuditAnnexTitle(),
		SectionOrder: RequiredAuditAnnexSectionOrder(),
		PerAssetAuditSections: []PerAssetAuditSection{{
			AssetIdentityKey: "asset-btc",
			DisplayLabel:     "BTC",
			Entries: []AuditActivityEntry{{
				SourceID:              "eur-buy-1",
				OccurredAt:            activityDate,
				ActivityType:          ActivityTypeBuy,
				Quantity:              mustReportDecimal(t, "1"),
				UnitPrice:             unitPrice,
				GrossValue:            grossValue,
				CalculationCurrency:   "USD",
				QuantityAfterActivity: mustReportDecimal(t, "1"),
				BasisAfterActivity:    mustReportDecimal(t, "110"),
			}},
		}},
		ConversionAuditEntries: []ConversionAuditEntry{{
			SourceID:           "eur-buy-1",
			AssetLabel:         "BTC",
			ActivityDate:       activityDate,
			SourceCurrency:     "EUR",
			ReportBaseCurrency: ReportBaseCurrencyUSD,
			RateDate:           activityDate,
			RateAuthority:      RateAuthorityFederalReserve,
			RateKind:           "noon buying rate",
			RateValue:          mustReportDecimal(t, "1.1"),
			QuoteDirection:     QuoteDirectionBasePerSource,
			Amounts:            []ConvertedActivityAmount{amount},
		}},
	}

	var cloned = cloneAuditAnnex(annex)
	annex.SectionOrder[0] = AuditAnnexSectionCurrencyConversionAudit
	annex.PerAssetAuditSections[0].Entries[0].SourceID = "mutated"
	annex.ConversionAuditEntries[0].Amounts[0].SourceID = "mutated"
	*unitPrice = mustReportDecimal(t, "20")
	*grossValue = mustReportDecimal(t, "30")

	if cloned.SectionOrder[0] != AuditAnnexSectionPerAssetReport {
		t.Fatalf("expected cloned section order to be independent, got %#v", cloned.SectionOrder)
	}
	if cloned.PerAssetAuditSections[0].Entries[0].SourceID != "eur-buy-1" {
		t.Fatalf("expected cloned audit activity entries to be independent, got %#v", cloned.PerAssetAuditSections[0].Entries[0])
	}
	if cloned.ConversionAuditEntries[0].Amounts[0].SourceID != "eur-buy-1" {
		t.Fatalf("expected cloned conversion audit amounts to be independent, got %#v", cloned.ConversionAuditEntries[0].Amounts[0])
	}
	assertOptionalDecimalString(t, cloned.PerAssetAuditSections[0].Entries[0].UnitPrice, "10")
	assertOptionalDecimalString(t, cloned.PerAssetAuditSections[0].Entries[0].GrossValue, "10")
}

// TestNewPerAssetAuditSectionClonesInheritedClassificationAndAuditValues verifies
// the Annex 1 model retains the inherited classification and every audit value
// after the caller mutates its source entry.
// Authored by: OpenCode
func TestNewPerAssetAuditSectionClonesInheritedClassificationAndAuditValues(t *testing.T) {
	t.Parallel()

	var activityDate = time.Date(2024, time.January, 5, 0, 0, 0, 0, time.UTC)
	var quantity = mustReportDecimal(t, "340282366920938463463374607431768211457")
	var quantityAfter = mustReportDecimal(t, "340282366920938463463374607431768211458")
	var basisAfter = mustReportDecimal(t, "340282366920938463463374607431768211459")
	var entries = []AuditActivityEntry{{
		SourceID:                     "audit-buy-1",
		OccurredAt:                   activityDate,
		ActivityType:                 ActivityTypeBuy,
		Quantity:                     quantity,
		UnitPrice:                    decimalPointer(t, "10"),
		GrossValue:                   decimalPointer(t, "10"),
		FeeAmount:                    decimalPointer(t, "1"),
		ActivityCurrency:             "EUR",
		CalculationCurrency:          "USD",
		QuantityAfterActivity:        quantityAfter,
		BasisAfterActivity:           basisAfter,
		FullLiquidationEvent:         true,
		IsZeroPricedHoldingReduction: true,
		AllocatedBasis:               decimalPointer(t, "11"),
		NetLiquidationProceeds:       decimalPointer(t, "10"),
		GainOrLoss:                   decimalPointer(t, "-1"),
		ConversionStatus:             ConversionStatusConverted,
		Note:                         "audit note",
	}}

	var section, err = NewPerAssetAuditSection("asset-btc", "BTC", entries)
	if err != nil {
		t.Fatalf("new per-asset audit section: %v", err)
	}

	entries[0].SourceID = "mutated-source"
	entries[0].OccurredAt = activityDate.Add(time.Hour)
	entries[0].ActivityType = ActivityTypeSell
	var increment apd.BigInt
	increment.SetInt64(1)
	entries[0].Quantity.Coeff.Add(&entries[0].Quantity.Coeff, &increment)
	*entries[0].UnitPrice = mustReportDecimal(t, "20")
	*entries[0].GrossValue = mustReportDecimal(t, "20")
	*entries[0].FeeAmount = mustReportDecimal(t, "2")
	entries[0].ActivityCurrency = "GBP"
	entries[0].CalculationCurrency = "EUR"
	entries[0].QuantityAfterActivity.Coeff.Add(&entries[0].QuantityAfterActivity.Coeff, &increment)
	entries[0].BasisAfterActivity.Coeff.Add(&entries[0].BasisAfterActivity.Coeff, &increment)
	if quantity.Text('f') != "340282366920938463463374607431768211458" ||
		quantityAfter.Text('f') != "340282366920938463463374607431768211459" ||
		basisAfter.Text('f') != "340282366920938463463374607431768211460" {
		t.Fatalf("expected coefficient mutation to change shallow decimal copies")
	}
	entries[0].FullLiquidationEvent = false
	entries[0].IsZeroPricedHoldingReduction = false
	*entries[0].AllocatedBasis = mustReportDecimal(t, "20")
	*entries[0].NetLiquidationProceeds = mustReportDecimal(t, "20")
	*entries[0].GainOrLoss = mustReportDecimal(t, "0")
	entries[0].ConversionStatus = ConversionStatusSameCurrency
	entries[0].Note = "mutated note"

	var cloned = section.Entries[0]
	if cloned.SourceID != "audit-buy-1" || !cloned.OccurredAt.Equal(activityDate) || cloned.ActivityType != ActivityTypeBuy {
		t.Fatalf("expected cloned audit identity values to remain unchanged, got %#v", cloned)
	}
	if cloned.Quantity.Text('f') != "340282366920938463463374607431768211457" {
		t.Fatalf("expected cloned quantity to remain unchanged, got %s", cloned.Quantity.Text('f'))
	}
	assertOptionalDecimalString(t, cloned.UnitPrice, "10")
	assertOptionalDecimalString(t, cloned.GrossValue, "10")
	assertOptionalDecimalString(t, cloned.FeeAmount, "1")
	if cloned.ActivityCurrency != "EUR" || cloned.CalculationCurrency != "USD" {
		t.Fatalf("expected cloned currencies to retain pre-format values, got %#v", cloned)
	}
	if cloned.QuantityAfterActivity.Text('f') != "340282366920938463463374607431768211458" ||
		cloned.BasisAfterActivity.Text('f') != "340282366920938463463374607431768211459" {
		t.Fatalf("expected cloned replay values to remain unchanged, got %#v", cloned)
	}
	if !cloned.FullLiquidationEvent || !cloned.IsZeroPricedHoldingReduction {
		t.Fatalf("expected cloned boolean audit values to remain unchanged, got %#v", cloned)
	}
	assertOptionalDecimalString(t, cloned.AllocatedBasis, "11")
	assertOptionalDecimalString(t, cloned.NetLiquidationProceeds, "10")
	assertOptionalDecimalString(t, cloned.GainOrLoss, "-1")
	if cloned.ConversionStatus != ConversionStatusConverted || cloned.Note != "audit note" {
		t.Fatalf("expected cloned classification and note values to remain unchanged, got %#v", cloned)
	}
}

// TestAuditModelRemainingValidationBranches verifies focused guardrails that are
// otherwise only reached by malformed Annex 1 or renderer/output wiring.
// Authored by: OpenCode
func TestAuditModelRemainingValidationBranches(t *testing.T) {
	t.Parallel()

	var activityDate = time.Date(2024, time.January, 5, 0, 0, 0, 0, time.UTC)
	var validEntry = AuditActivityEntry{
		SourceID:               "buy-1",
		OccurredAt:             activityDate,
		ActivityType:           ActivityTypeBuy,
		Quantity:               mustReportDecimal(t, "1"),
		UnitPrice:              decimalPointer(t, "10"),
		GrossValue:             decimalPointer(t, "10"),
		FeeAmount:              decimalPointer(t, "0"),
		CalculationCurrency:    "USD",
		QuantityAfterActivity:  mustReportDecimal(t, "1"),
		BasisAfterActivity:     mustReportDecimal(t, "10"),
		AllocatedBasis:         decimalPointer(t, "0"),
		NetLiquidationProceeds: decimalPointer(t, "0"),
		GainOrLoss:             decimalPointer(t, "0"),
		ConversionStatus:       ConversionStatusConverted,
	}

	var testCases = []struct {
		name   string
		entry  AuditActivityEntry
		wanted string
	}{
		{name: "missing timestamp", entry: auditEntryWith(validEntry, func(entry *AuditActivityEntry) { entry.OccurredAt = time.Time{} }), wanted: "occurred-at timestamp"},
		{name: "activity type", entry: auditEntryWith(validEntry, func(entry *AuditActivityEntry) { entry.ActivityType = ActivityType("swap") }), wanted: "activity type"},
		{name: "unit price", entry: auditEntryWith(validEntry, func(entry *AuditActivityEntry) { entry.UnitPrice = &reportInvalidDecimal }), wanted: "unit price"},
		{name: "gross value", entry: auditEntryWith(validEntry, func(entry *AuditActivityEntry) { entry.GrossValue = &reportInvalidDecimal }), wanted: "gross value"},
		{name: "fee amount", entry: auditEntryWith(validEntry, func(entry *AuditActivityEntry) { entry.FeeAmount = &reportInvalidDecimal }), wanted: "fee amount"},
		{name: "calculation currency", entry: auditEntryWith(validEntry, func(entry *AuditActivityEntry) { entry.CalculationCurrency = " " }), wanted: "calculation currency"},
		{name: "quantity after", entry: auditEntryWith(validEntry, func(entry *AuditActivityEntry) { entry.QuantityAfterActivity = mustReportDecimal(t, "-1") }), wanted: "quantity after activity"},
		{name: "basis after", entry: auditEntryWith(validEntry, func(entry *AuditActivityEntry) { entry.BasisAfterActivity = mustReportDecimal(t, "-1") }), wanted: "basis after activity"},
		{name: "allocated basis", entry: auditEntryWith(validEntry, func(entry *AuditActivityEntry) { entry.AllocatedBasis = &reportInvalidDecimal }), wanted: "allocated basis"},
		{name: "net proceeds", entry: auditEntryWith(validEntry, func(entry *AuditActivityEntry) { entry.NetLiquidationProceeds = &reportInvalidDecimal }), wanted: "net liquidation proceeds"},
		{name: "gain or loss", entry: auditEntryWith(validEntry, func(entry *AuditActivityEntry) { entry.GainOrLoss = &reportInvalidDecimal }), wanted: "gain or loss"},
		{name: "conversion status", entry: auditEntryWith(validEntry, func(entry *AuditActivityEntry) { entry.ConversionStatus = ConversionStatus("unknown") }), wanted: "conversion status"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var err = testCase.entry.Validate()
			if err == nil || !strings.Contains(err.Error(), testCase.wanted) {
				t.Fatalf("expected audit entry error containing %q, got %v", testCase.wanted, err)
			}
		})
	}

	if _, err := NewPerAssetAuditSection("asset-btc", " ", nil); err == nil || !strings.Contains(err.Error(), "display label") {
		t.Fatalf("expected per-asset display label rejection, got %v", err)
	}
	if _, err := NewAuditAnnex(AuditAnnexTitle(), RequiredAuditAnnexSectionOrder()); err != nil {
		t.Fatalf("expected valid audit annex constructor to succeed: %v", err)
	}
	if err := (AuditAnnex{Title: AuditAnnexTitle(), SectionOrder: []AuditAnnexSection{AuditAnnexSectionPerAssetReport}}).Validate(); err == nil || !strings.Contains(err.Error(), "must contain 2 sections") {
		t.Fatalf("expected audit annex section-count rejection, got %v", err)
	}
	if err := (AuditAnnex{Title: AuditAnnexTitle(), SectionOrder: RequiredAuditAnnexSectionOrder(), ConversionAuditEntries: []ConversionAuditEntry{{}}}).Validate(); err == nil || !strings.Contains(err.Error(), "conversion audit entry 0") {
		t.Fatalf("expected audit annex conversion entry rejection, got %v", err)
	}

	var report = CapitalGainsReport{Year: 2024, CostBasisMethod: CostBasisMethodFIFO, GeneratedAt: activityDate, ReportCalculationCurrency: "USD", YearlyNetTotal: mustReportDecimal(t, "0"), AuditAnnex: AuditAnnex{Title: "bad", SectionOrder: RequiredAuditAnnexSectionOrder()}}
	if err := report.Validate(); err == nil || !strings.Contains(err.Error(), "capital gains report audit annex") {
		t.Fatalf("expected report audit annex rejection, got %v", err)
	}
	if _, err := RenderActivityTypeLabel(AssetActivityRow{ActivityType: ActivityType("swap")}); err == nil || !strings.Contains(err.Error(), "unsupported activity type") {
		t.Fatalf("expected unsupported activity label rejection, got %v", err)
	}
}

// TestReportOutputBundleRemainingMarkdownRoleBranch verifies the first Markdown
// bundle file must be the main report file.
// Authored by: OpenCode
func TestReportOutputBundleRemainingMarkdownRoleBranch(t *testing.T) {
	t.Parallel()

	var savedAt = time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC)
	var annexFile, annexErr = NewReportOutputFile("/tmp/Documents", "report-annex.md", "/tmp/Documents/report-annex.md", ReportDocumentRoleAnnex, ReportMediaTypeMarkdown, savedAt)
	if annexErr != nil {
		t.Fatalf("new annex output file: %v", annexErr)
	}

	var err = (ReportOutputBundle{OutputFormat: ReportOutputFormatMarkdown, Files: []ReportOutputFile{annexFile, annexFile}, SavedAt: savedAt}).Validate()
	if err == nil || !strings.Contains(err.Error(), "file 0 must be main") {
		t.Fatalf("expected Markdown main-role failure, got %v", err)
	}
}

// auditEntryWith returns a modified copy of an audit entry for validation
// branch tests.
// Authored by: OpenCode
func auditEntryWith(entry AuditActivityEntry, mutate func(*AuditActivityEntry)) AuditActivityEntry {
	mutate(&entry)
	return entry
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

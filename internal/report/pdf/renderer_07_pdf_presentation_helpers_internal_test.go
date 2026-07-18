package pdf

import (
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/report/presentation"
	"github.com/cockroachdb/apd/v3"
)

// TestPDFFormattingHelperFallbackBranches covers PDF-only label and conversion
// fallback paths that are not naturally exercised by the report fixtures.
// Authored by: OpenCode
func TestPDFFormattingHelperFallbackBranches(t *testing.T) {
	var fallbackLabel = renderDisplayLabel("", "asset-fallback")
	if fallbackLabel != "asset-fallback" {
		t.Fatalf("fallback display label = %q, want asset-fallback", fallbackLabel)
	}
	var unknownLabel = renderDisplayLabel("", "")
	if unknownLabel != "Unknown Asset" {
		t.Fatalf("unknown display label = %q, want Unknown Asset", unknownLabel)
	}

	var unitPrice = apd.New(1, 0)
	var status, err = conversionStatusColumn(reportmodel.AssetActivityRow{ActivityCurrency: "USD", CalculationCurrency: "USD", UnitPrice: unitPrice})
	if err != nil || status != "Same currency" {
		t.Fatalf("same-currency conversion status = %q, %v; want Same currency", status, err)
	}
	status, err = conversionStatusColumn(reportmodel.AssetActivityRow{ActivityCurrency: "EUR", CalculationCurrency: "USD", UnitPrice: unitPrice})
	if err != nil || status != "Converted" {
		t.Fatalf("converted status = %q, %v; want Converted", status, err)
	}
	status, err = conversionStatusColumn(reportmodel.AssetActivityRow{ActivityCurrency: "USD"})
	if err != nil || status != "" {
		t.Fatalf("no-monetary conversion status = %q, %v; want empty", status, err)
	}
	if _, err = conversionStatusColumn(reportmodel.AssetActivityRow{GrossValue: unitPrice, ActivityCurrency: "USD", ConversionStatus: reportmodel.ConversionStatus("unknown")}); err == nil {
		t.Fatalf("expected unsupported conversion status to fail")
	}
	if label := calculationCurrencyLabel(""); label != "NOT APPLICABLE" {
		t.Fatalf("empty calculation currency label = %q, want NOT APPLICABLE", label)
	}
	if label := calculationCurrencyLabelWithFallback("", "USD"); label != "USD" {
		t.Fatalf("fallback calculation currency label = %q, want USD", label)
	}
	if label := rateProviderLabel(reportmodel.RateProviderIDECBEXR); label != "ECB Data Portal EXR" {
		t.Fatalf("ECB provider label = %q, want ECB Data Portal EXR", label)
	}
}

// TestPDFFormattingHelperErrorBranches covers defensive renderer helper failures.
// Authored by: OpenCode
func TestPDFFormattingHelperErrorBranches(t *testing.T) {
	var invalidDecimal = nonFiniteDecimal()
	var validReport = pdfPresentationReportFixture(t)

	var summaryReport = validReport
	summaryReport.SummaryEntries = []reportmodel.AssetSummaryEntry{{AssetIdentityKey: "asset-invalid-summary", DisplayLabel: "BAD", NetGainOrLoss: invalidDecimal, ReportCalculationCurrency: "USD"}}
	assertErrorContains(t, func() error { return renderSummarySection(&layoutRecorder{}, summaryReport, "USD") }, "net gain or loss")

	var totalReport = validReport
	totalReport.SummaryEntries = nil
	totalReport.YearlyNetTotal = invalidDecimal
	assertErrorContains(t, func() error { return renderSummarySection(&layoutRecorder{}, totalReport, "USD") }, "yearly net total")

	assertErrorContains(t, func() error {
		return renderPositionBlock(&layoutRecorder{}, "Bad", invalidDecimal, *apd.New(1, 0), "USD", "USD")
	}, "quantity")
	assertErrorContains(t, func() error {
		return renderPositionBlock(&layoutRecorder{}, "Bad", *apd.New(1, 0), invalidDecimal, "USD", "USD")
	}, "cost basis")
	assertErrorContains(t, func() error {
		_, err := renderActivityRow(reportmodel.AssetActivityRow{SourceID: "row", Quantity: invalidDecimal})
		return err
	}, "quantity")
	assertErrorContains(t, func() error {
		_, err := renderLiquidationRow(reportmodel.LiquidationCalculation{SourceID: "liq", DisposedQuantity: invalidDecimal}, "USD")
		return err
	}, "disposed quantity")
	assertErrorContains(t, func() error {
		_, err := renderAnnexActivityRow(reportmodel.AuditActivityEntry{SourceID: "entry", Quantity: invalidDecimal})
		return err
	}, "quantity")

	var invalidConversion = pdfAnnexConversionEntry()
	invalidConversion.RateValue = invalidDecimal
	assertErrorContains(t, func() error { _, err := renderConversionAuditRow(0, invalidConversion); return err }, "rate value")

	var badQuote = pdfAnnexConversionEntry()
	badQuote.QuoteDirection = reportmodel.QuoteDirection("bad_direction")
	assertErrorContains(t, func() error { _, err := renderConversionAuditRow(0, badQuote); return err }, "quote direction")

	var zeroAmount = pdfAnnexConversionEntry().Amounts[0]
	zeroAmount.OriginalAmount = *apd.New(0, 0)
	zeroAmount.ConvertedAmount = *apd.New(0, 0)
	if rendered, err := presentation.ConvertedAmounts(0, []reportmodel.ConvertedActivityAmount{zeroAmount}); err != nil || len(rendered) != 0 {
		t.Fatalf("zero converted amounts = %#v, %v; want empty nil", rendered, err)
	}
}

// TestPDFFormattingHelperCompleteErrorBranches covers decimal and label failure
// branches in row formatting helpers.
// Authored by: OpenCode
func TestPDFFormattingHelperCompleteErrorBranches(t *testing.T) {
	var invalidDecimal = nonFiniteDecimal()
	var validActivity = reportmodel.AssetActivityRow{SourceID: "row", ActivityType: reportmodel.ActivityTypeSell, Quantity: *apd.New(1, 0), BasisAfterRow: *apd.New(0, 0), QuantityAfterRow: *apd.New(0, 0)}
	var activityCases = []struct {
		name string
		row  reportmodel.AssetActivityRow
		want string
	}{
		{name: "unit price", row: withActivityUnitPrice(validActivity, invalidDecimal), want: "unit price"},
		{name: "gross", row: withActivityGrossValue(validActivity, invalidDecimal), want: "gross value"},
		{name: "fee", row: withActivityFee(validActivity, invalidDecimal), want: "fee"},
		{name: "basis after", row: withActivityBasisAfterRow(validActivity, invalidDecimal), want: "basis after row"},
		{name: "quantity after", row: withActivityQuantityAfterRow(validActivity, invalidDecimal), want: "quantity after row"},
		{name: "type", row: withActivityType(validActivity, reportmodel.ActivityType("bad_type")), want: "type label"},
		{name: "conversion", row: withActivityConversionStatus(validActivity, reportmodel.ConversionStatus("bad_status")), want: "conversion status label"},
	}
	for _, testCase := range activityCases {
		var testCase = testCase
		t.Run("activity "+testCase.name, func(t *testing.T) {
			assertErrorContains(t, func() error { _, err := renderActivityRow(testCase.row); return err }, testCase.want)
		})
	}

	var validLiquidation = reportmodel.LiquidationCalculation{SourceID: "liq", DisposedQuantity: *apd.New(1, 0), AllocatedBasis: *apd.New(1, 0), NetLiquidationProceeds: *apd.New(1, 0), GainOrLoss: *apd.New(0, 0)}
	for _, testCase := range []struct {
		name        string
		liquidation reportmodel.LiquidationCalculation
		want        string
	}{
		{name: "allocated", liquidation: withAllocatedBasis(validLiquidation, invalidDecimal), want: "allocated basis"},
		{name: "proceeds", liquidation: withNetLiquidationProceeds(validLiquidation, invalidDecimal), want: "net proceeds"},
		{name: "gain", liquidation: withGainOrLoss(validLiquidation, invalidDecimal), want: "gain or loss"},
	} {
		var testCase = testCase
		t.Run("liquidation "+testCase.name, func(t *testing.T) {
			assertErrorContains(t, func() error { _, err := renderLiquidationRow(testCase.liquidation, "USD"); return err }, testCase.want)
		})
	}

	var validAnnex = reportmodel.AuditActivityEntry{SourceID: "entry", ActivityType: reportmodel.ActivityTypeSell, Quantity: *apd.New(1, 0), QuantityAfterActivity: *apd.New(0, 0), BasisAfterActivity: *apd.New(0, 0)}
	for _, testCase := range []struct {
		name  string
		entry reportmodel.AuditActivityEntry
		want  string
	}{
		{name: "unit price", entry: withAnnexUnitPrice(validAnnex, invalidDecimal), want: "unit price"},
		{name: "gross", entry: withAnnexGrossValue(validAnnex, invalidDecimal), want: "gross value"},
		{name: "fee", entry: withAnnexFee(validAnnex, invalidDecimal), want: "fee"},
		{name: "quantity after", entry: withAnnexQuantityAfter(validAnnex, invalidDecimal), want: "quantity after activity"},
		{name: "basis after", entry: withAnnexBasisAfter(validAnnex, invalidDecimal), want: "basis after activity"},
		{name: "allocated", entry: withAnnexAllocatedBasis(validAnnex, invalidDecimal), want: "allocated basis"},
		{name: "proceeds", entry: withAnnexProceeds(validAnnex, invalidDecimal), want: "net liquidation proceeds"},
		{name: "gain", entry: withAnnexGain(validAnnex, invalidDecimal), want: "gain or loss"},
		{name: "type", entry: withAnnexActivityType(validAnnex, reportmodel.ActivityType("bad_type")), want: "activity type label"},
		{name: "conversion", entry: withAnnexConversionStatus(validAnnex, reportmodel.ConversionStatus("bad_status")), want: "conversion status label"},
	} {
		var testCase = testCase
		t.Run("annex "+testCase.name, func(t *testing.T) {
			assertErrorContains(t, func() error { _, err := renderAnnexActivityRow(testCase.entry); return err }, testCase.want)
		})
	}

	var badOriginal = pdfAnnexConversionEntry()
	badOriginal.Amounts = append([]reportmodel.ConvertedActivityAmount(nil), badOriginal.Amounts...)
	badOriginal.Amounts[0].OriginalAmount = invalidDecimal
	assertErrorContains(t, func() error { _, err := renderConversionAuditRow(0, badOriginal); return err }, "original amount")
	var badConverted = pdfAnnexConversionEntry()
	badConverted.Amounts = append([]reportmodel.ConvertedActivityAmount(nil), badConverted.Amounts...)
	badConverted.Amounts[0].ConvertedAmount = invalidDecimal
	assertErrorContains(t, func() error { _, err := renderConversionAuditRow(0, badConverted); return err }, "converted amount")
}

// TestStructuredRendererSuccessBranches covers non-empty summary, liquidation,
// duplicate rate-source, and empty Annex 1 paths.
// Authored by: OpenCode
func TestStructuredRendererSuccessBranches(t *testing.T) {
	var recorder = &layoutRecorder{}
	var report = pdfNonZeroLiquidationReportFixture(t)
	var conversion = pdfAnnexConversionEntry()
	report.RateSources = []reportmodel.ExchangeRateEvidence{*conversion.Amounts[0].ExchangeRateEvidence, *conversion.Amounts[0].ExchangeRateEvidence}

	if err := renderMainReport(recorder, report); err != nil {
		t.Fatalf("render non-zero report: %v", err)
	}
	assertTableHeader(t, recorder, "Net Gain Or Loss")
	assertTableHeader(t, recorder, "Disposed Quantity")
	assertTableCell(t, recorder, "5")
	assertTableCell(t, recorder, "7")

	var annexRecorder = &layoutRecorder{}
	if err := renderAnnex(annexRecorder, reportmodel.DefaultAuditAnnex()); err != nil {
		t.Fatalf("render empty annex: %v", err)
	}
	assertContains(t, annexRecorder.paragraphs, "No per-asset audit activity is available for this report.")
	assertContains(t, annexRecorder.paragraphs, "No converted activity was present for this report.")
}

// TestStructuredRendererErrorPropagation covers layout-seam failures at each
// renderer stage without involving concrete gopdf output.
// Authored by: OpenCode
func TestStructuredRendererErrorPropagation(t *testing.T) {
	var report = pdfNonZeroLiquidationReportFixture(t)
	for _, testCase := range []struct {
		name     string
		recorder *errorLayoutRecorder
		want     string
	}{
		{name: "year", recorder: &errorLayoutRecorder{failKey: "Year"}, want: "key failed"},
		{name: "method", recorder: &errorLayoutRecorder{failKey: "Cost Basis Method"}, want: "key failed"},
		{name: "generated", recorder: &errorLayoutRecorder{failKey: "Generated At"}, want: "key failed"},
		{name: "currency", recorder: &errorLayoutRecorder{failKey: "Report Calculation Currency"}, want: "key failed"},
		{name: "summary", recorder: &errorLayoutRecorder{failSection: "Gains-And-Losses Summary"}, want: "section failed"},
		{name: "summary table", recorder: &errorLayoutRecorder{failTable: "Gains-And-Losses Summary Table"}, want: "table failed"},
		{name: "rate key", recorder: &errorLayoutRecorder{failKey: "Report Base Currency"}, want: "key failed"},
		{name: "reference", recorder: &errorLayoutRecorder{failSection: "Reference Section"}, want: "section failed"},
		{name: "asset", recorder: &errorLayoutRecorder{failSection: "Asset Detail: GAIN"}, want: "section failed"},
		{name: "position", recorder: &errorLayoutRecorder{failSubsection: "Opening Position"}, want: "opening position"},
		{name: "activity table", recorder: &errorLayoutRecorder{failTable: "In-Year Activity"}, want: "in-year activity"},
		{name: "liquidation table", recorder: &errorLayoutRecorder{failTable: "Liquidation Calculations"}, want: "liquidation calculations"},
		{name: "position key", recorder: &errorLayoutRecorder{failKey: "Quantity"}, want: "opening position"},
	} {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			assertErrorContains(t, func() error { return renderMainReport(testCase.recorder, report) }, testCase.want)
		})
	}

	var annex = pdfAnnexReportFixture(t).AuditAnnex
	for _, testCase := range []struct {
		name     string
		recorder *errorLayoutRecorder
		want     string
	}{
		{name: "title", recorder: &errorLayoutRecorder{failTitle: AnnexTitle}, want: "title failed"},
		{name: "per asset section", recorder: &errorLayoutRecorder{failSection: "Detailed Per-Asset Audit Report"}, want: "section failed"},
		{name: "asset subsection", recorder: &errorLayoutRecorder{failSubsection: "Asset: BTC"}, want: "subsection failed"},
		{name: "asset table", recorder: &errorLayoutRecorder{failTable: "Per-Asset Audit Activity"}, want: "table failed"},
		{name: "conversion section", recorder: &errorLayoutRecorder{failSection: "Currency Conversion Audit"}, want: "section failed"},
		{name: "conversion table", recorder: &errorLayoutRecorder{failTable: "Currency Conversion Audit Table"}, want: "table failed"},
	} {
		var testCase = testCase
		t.Run("annex "+testCase.name, func(t *testing.T) {
			assertErrorContains(t, func() error { return renderAnnex(testCase.recorder, annex) }, testCase.want)
		})
	}
}

// TestRemainingRendererErrorBranches covers narrow error paths that are not hit
// by the broader propagation table.
// Authored by: OpenCode
func TestRemainingRendererErrorBranches(t *testing.T) {
	var invalidDecimal = nonFiniteDecimal()
	var zeroReport = minimalPDFReportFixture(t)
	assertErrorContains(t, func() error {
		return renderSummarySection(&errorLayoutRecorder{failParagraph: true}, zeroReport, "USD")
	}, "paragraph failed")
	assertErrorContains(t, func() error {
		return renderRateSourceSection(&errorLayoutRecorder{failSection: "Rate Source Summary"}, zeroReport)
	}, "section failed")
	var rateReport = pdfNonZeroLiquidationReportFixture(t)
	var conversion = pdfAnnexConversionEntry()
	rateReport.RateSources = []reportmodel.ExchangeRateEvidence{*conversion.Amounts[0].ExchangeRateEvidence}
	for _, testCase := range []struct {
		name string
		key  string
	}{
		{name: "authority", key: "Authority"},
		{name: "provider", key: "Provider"},
		{name: "rate kind", key: "Rate Kind"},
		{name: "unavailable", key: "Unavailable-Date Rule"},
	} {
		var testCase = testCase
		t.Run("rate source "+testCase.name, func(t *testing.T) {
			assertErrorContains(t, func() error {
				return renderRateSourceSection(&errorLayoutRecorder{failKey: testCase.key}, rateReport)
			}, "key failed")
		})
	}
	assertErrorContains(t, func() error {
		return renderPositionBlock(&errorLayoutRecorder{failKey: "Cost Basis"}, "Position", *apd.New(1, 0), *apd.New(1, 0), "USD", "USD")
	}, "key failed")

	var historicalReport = pdfPresentationReportFixture(t)
	historicalReport.DetailSections = []reportmodel.AssetDetailSection{{AssetIdentityKey: "asset-historical", DisplayLabel: "HIST", ClosingQuantity: invalidDecimal, CalculationCurrency: "USD"}}
	assertErrorContains(t, func() error { return renderDetailSections(&layoutRecorder{}, historicalReport, "USD") }, "historical position")

	var closingReport = pdfNonZeroLiquidationReportFixture(t)
	closingReport.DetailSections[0].ClosingQuantity = invalidDecimal
	assertErrorContains(t, func() error { return renderDetailSections(&layoutRecorder{}, closingReport, "USD") }, "closing position")

	assertErrorContains(t, func() error {
		return renderActivityRows(&layoutRecorder{}, reportmodel.AssetDetailSection{ActivityRows: []reportmodel.AssetActivityRow{{SourceID: "bad", Quantity: invalidDecimal}}})
	}, "quantity")
	assertErrorContains(t, func() error {
		return renderLiquidationRows(&layoutRecorder{}, reportmodel.AssetDetailSection{LiquidationSummaries: []reportmodel.LiquidationCalculation{{SourceID: "bad", DisposedQuantity: *apd.New(1, 0), AllocatedBasis: invalidDecimal}}}, "USD")
	}, "allocated basis")

	if err := renderAnnex(&layoutRecorder{}, reportmodel.AuditAnnex{}); err != nil {
		t.Fatalf("default annex render: %v", err)
	}
	assertErrorContains(t, func() error {
		return renderAnnexPerAssetAudit(&layoutRecorder{}, reportmodel.AuditAnnex{Title: AnnexTitle, SectionOrder: reportmodel.RequiredAuditAnnexSectionOrder(), PerAssetAuditSections: []reportmodel.PerAssetAuditSection{{AssetIdentityKey: "asset", DisplayLabel: "ASSET", Entries: []reportmodel.AuditActivityEntry{{SourceID: "bad", Quantity: invalidDecimal}}}}})
	}, "quantity")
	var badConversion = pdfAnnexConversionEntry()
	badConversion.RateValue = invalidDecimal
	assertErrorContains(t, func() error {
		return renderAnnexConversionAudit(&layoutRecorder{}, reportmodel.AuditAnnex{Title: AnnexTitle, SectionOrder: reportmodel.RequiredAuditAnnexSectionOrder(), ConversionAuditEntries: []reportmodel.ConversionAuditEntry{badConversion}})
	}, "rate value")
}

package pdf

import (
	"bytes"
	"strings"
	"testing"
	"time"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/cockroachdb/apd/v3"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

// TestRenderMainReportRecordsWarningBetweenMetadataAndSummary verifies the PDF
// main-report operation order around the legal-use warning.
// Authored by: OpenCode
func TestRenderMainReportRecordsWarningBetweenMetadataAndSummary(t *testing.T) {
	var recorder = &layoutRecorder{}
	if err := renderMainReport(recorder, minimalPDFReportFixture(t)); err != nil {
		t.Fatalf("render main report: %v", err)
	}

	var currencyIndex = findLayoutOperation(recorder.operations, "key-value", "Report Calculation Currency", "")
	var warningIndex = findLayoutOperation(recorder.operations, "bold-wrapped-paragraph", "", testutil.ReportPresentationLegalWarningText)
	var summaryIndex = findLayoutOperation(recorder.operations, "section-heading", "", "Gains-And-Losses Summary")
	if currencyIndex < 0 || warningIndex < 0 || summaryIndex < 0 {
		t.Fatalf("metadata, warning, or summary operation is missing: %#v", recorder.operations)
	}
	if warningIndex != currencyIndex+1 {
		t.Fatalf("warning operation index = %d, want immediately after currency index %d: %#v", warningIndex, currencyIndex, recorder.operations)
	}
	if summaryIndex != warningIndex+1 {
		t.Fatalf("summary operation index = %d, want immediately after warning index %d: %#v", summaryIndex, warningIndex, recorder.operations)
	}
}

// TestRenderMainReportUsesDedicatedFullyBoldWrappedWarning verifies the warning
// is represented by one exact, fully bold, wrapped paragraph operation.
// Authored by: OpenCode
func TestRenderMainReportUsesDedicatedFullyBoldWrappedWarning(t *testing.T) {
	var recorder = &layoutRecorder{}
	if err := renderMainReport(recorder, minimalPDFReportFixture(t)); err != nil {
		t.Fatalf("render main report: %v", err)
	}

	var warningOperations []pdfLayoutOperation
	for _, operation := range recorder.operations {
		if operation.kind == "bold-wrapped-paragraph" {
			warningOperations = append(warningOperations, operation)
		}
	}
	if len(warningOperations) != 1 {
		t.Fatalf("bold wrapped warning operations = %#v, want one exact operation", warningOperations)
	}
	var warning = warningOperations[0]
	if warning.text != testutil.ReportPresentationLegalWarningText {
		t.Fatalf("warning text = %q, want %q", warning.text, testutil.ReportPresentationLegalWarningText)
	}
	if !warning.fullyBold || !warning.wrapped {
		t.Fatalf("warning operation style = %#v, want fully bold wrapped operation", warning)
	}
}

// TestPDFRecorderRendersDirectFinancialMatrixValues verifies direct summary and
// position values as exact semantic recorder values rather than flattened text.
// Authored by: OpenCode
func TestPDFRecorderRendersDirectFinancialMatrixValues(t *testing.T) {
	var report = minimalPDFReportFixture(t)
	report.SummaryEntries = []reportmodel.AssetSummaryEntry{
		{AssetIdentityKey: "direct-positive", DisplayLabel: "DIRECT POSITIVE", NetGainOrLoss: *apd.New(1005, -3), ReportCalculationCurrency: "USD"},
		{AssetIdentityKey: "direct-negative", DisplayLabel: "DIRECT NEGATIVE", NetGainOrLoss: *apd.New(-1005, -3), ReportCalculationCurrency: "USD"},
	}
	report.YearlyNetTotal = *apd.New(9995, -3)
	report.DetailSections = []reportmodel.AssetDetailSection{
		{
			AssetIdentityKey:    "historical-direct",
			DisplayLabel:        "HISTORICAL",
			ClosingQuantity:     *apd.New(1, -1),
			ClosingCostBasis:    *apd.New(1004, -3),
			CalculationCurrency: "USD",
		},
		{
			AssetIdentityKey:    "active-direct",
			DisplayLabel:        "ACTIVE",
			OpeningQuantity:     *apd.New(1000, -4),
			OpeningCostBasis:    *apd.New(1005, -3),
			ClosingQuantity:     *apd.New(1, -8),
			ClosingCostBasis:    *apd.New(9995, -3),
			CalculationCurrency: "USD",
			ActivityRows: []reportmodel.AssetActivityRow{{
				SourceID:            "direct-activity",
				OccurredAt:          time.Date(2024, time.January, 2, 10, 0, 0, 0, time.UTC),
				ActivityType:        reportmodel.ActivityTypeBuy,
				Quantity:            *apd.New(1, 0),
				BasisAfterRow:       *apd.New(0, 0),
				CalculationCurrency: "USD",
				QuantityAfterRow:    *apd.New(1, 0),
			}},
		},
	}

	var recorder = &layoutRecorder{}
	if err := renderMainReport(recorder, report); err != nil {
		t.Fatalf("render main report: %v", err)
	}

	assertTableCellAt(t, recorder, "Gains-And-Losses Summary Table", 0, "DIRECT POSITIVE", 1, "1.01")
	assertTableCellAt(t, recorder, "Gains-And-Losses Summary Table", 0, "DIRECT NEGATIVE", 1, "-1.01")
	assertTableCellAt(t, recorder, "Gains-And-Losses Summary Table", 0, "Overall Yearly Net Total", 1, "10.00")
	assertKeyValueOperation(t, recorder, "Cost Basis", "1.00")
	assertKeyValueOperation(t, recorder, "Cost Basis", "1.01")
	assertKeyValueOperation(t, recorder, "Cost Basis", "10.00")
	assertKeyValueOperation(t, recorder, "Quantity", "0.1")
	assertKeyValueOperation(t, recorder, "Quantity", "0.00000001")
}

// TestPDFRecorderRendersRowBuiltFinancialMatrixValues verifies exact financial
// cells from activity, liquidation, Annex, and conversion row builders.
// Authored by: OpenCode
func TestPDFRecorderRendersRowBuiltFinancialMatrixValues(t *testing.T) {
	var activity = reportmodel.AssetActivityRow{
		SourceID:            "matrix-activity",
		OccurredAt:          time.Date(2024, time.January, 2, 10, 0, 0, 0, time.UTC),
		ActivityType:        reportmodel.ActivityTypeSell,
		Quantity:            *apd.New(1000, -4),
		UnitPrice:           apd.New(1005, -3),
		GrossValue:          apd.New(1004, -3),
		FeeAmount:           apd.New(9995, -3),
		ActivityCurrency:    "USD",
		BasisAfterRow:       *apd.New(1005, -3),
		CalculationCurrency: "USD",
		QuantityAfterRow:    *apd.New(1, -8),
		ConversionStatus:    reportmodel.ConversionStatusSameCurrency,
	}
	var liquidation = reportmodel.LiquidationCalculation{
		SourceID:               "matrix-liquidation",
		OccurredAt:             activity.OccurredAt,
		DisposedQuantity:       *apd.New(1000, -4),
		AllocatedBasis:         *apd.New(1005, -3),
		NetLiquidationProceeds: *apd.New(-1005, -3),
		GainOrLoss:             *apd.New(-4, -3),
		ActivityCurrency:       "USD",
		CalculationCurrency:    "USD",
	}

	var recorder = &layoutRecorder{}
	var section = reportmodel.AssetDetailSection{ActivityRows: []reportmodel.AssetActivityRow{activity}, LiquidationSummaries: []reportmodel.LiquidationCalculation{liquidation}}
	if err := renderActivityRows(recorder, section); err != nil {
		t.Fatalf("render activity rows: %v", err)
	}
	if err := renderLiquidationRows(recorder, section, "USD"); err != nil {
		t.Fatalf("render liquidation rows: %v", err)
	}

	var annexReport = pdfAnnexReportFixture(t)
	var annexEntry = annexReport.AuditAnnex.PerAssetAuditSections[0].Entries[0]
	annexEntry.UnitPrice = apd.New(1005, -3)
	annexEntry.GrossValue = apd.New(1004, -3)
	annexEntry.FeeAmount = apd.New(9995, -3)
	annexEntry.Quantity = *apd.New(1000, -4)
	annexEntry.QuantityAfterActivity = *apd.New(1, -8)
	annexEntry.BasisAfterActivity = *apd.New(1005, -3)
	annexEntry.AllocatedBasis = apd.New(1005, -3)
	annexEntry.NetLiquidationProceeds = apd.New(-1005, -3)
	annexEntry.GainOrLoss = apd.New(-4, -3)
	annexReport.AuditAnnex.PerAssetAuditSections[0].Entries[0] = annexEntry

	var conversion = pdfAnnexConversionEntry()
	conversion.RateValue = *apd.New(169140, -4)
	conversion.Amounts[0].AmountKind = reportmodel.ConvertedAmountKindUnitPrice
	conversion.Amounts[0].OriginalAmount = *apd.New(1005, -3)
	conversion.Amounts[0].ConvertedAmount = *apd.New(1004, -3)
	conversion.Amounts[0].ExchangeRateEvidence.RateValue = conversion.RateValue
	annexReport.AuditAnnex.ConversionAuditEntries = []reportmodel.ConversionAuditEntry{conversion}

	var annexRecorder = &layoutRecorder{}
	if err := renderAnnex(annexRecorder, annexReport.AuditAnnex); err != nil {
		t.Fatalf("render annex: %v", err)
	}

	assertTableCellAt(t, recorder, "In-Year Activity", 1, "matrix-activity", 3, "0.1")
	assertTableCellAt(t, recorder, "In-Year Activity", 1, "matrix-activity", 4, "1.01")
	assertTableCellAt(t, recorder, "In-Year Activity", 1, "matrix-activity", 5, "1.00")
	assertTableCellAt(t, recorder, "In-Year Activity", 1, "matrix-activity", 6, "10.00")
	assertTableCellAt(t, recorder, "In-Year Activity", 1, "matrix-activity", 7, "0.00000001")
	assertTableCellAt(t, recorder, "In-Year Activity", 1, "matrix-activity", 8, "1.01")
	assertTableCellAt(t, recorder, "Liquidation Calculations", 1, "matrix-liquidation", 2, "0.1")
	assertTableCellAt(t, recorder, "Liquidation Calculations", 1, "matrix-liquidation", 3, "1.01")
	assertTableCellAt(t, recorder, "Liquidation Calculations", 1, "matrix-liquidation", 4, "-1.01")
	assertTableCellAt(t, recorder, "Liquidation Calculations", 1, "matrix-liquidation", 5, "0.00")

	assertTableCellAt(t, annexRecorder, "Per-Asset Audit Activity", 1, "pdf-annex-sell", 3, "0.1")
	assertTableCellAt(t, annexRecorder, "Per-Asset Audit Activity", 1, "pdf-annex-sell", 4, "1.01")
	assertTableCellAt(t, annexRecorder, "Per-Asset Audit Activity", 1, "pdf-annex-sell", 5, "1.00")
	assertTableCellAt(t, annexRecorder, "Per-Asset Audit Activity", 1, "pdf-annex-sell", 6, "10.00")
	assertTableCellAt(t, annexRecorder, "Per-Asset Audit Activity", 1, "pdf-annex-sell", 9, "0.00000001")
	assertTableCellAt(t, annexRecorder, "Per-Asset Audit Activity", 1, "pdf-annex-sell", 10, "1.01")
	assertTableCellAt(t, annexRecorder, "Per-Asset Audit Activity", 1, "pdf-annex-sell", 12, "1.01")
	assertTableCellAt(t, annexRecorder, "Per-Asset Audit Activity", 1, "pdf-annex-sell", 13, "-1.01")
	assertTableCellAt(t, annexRecorder, "Per-Asset Audit Activity", 1, "pdf-annex-sell", 14, "0.00")
	assertTableCellAt(t, annexRecorder, "Currency Conversion Audit Table", 1, "pdf-annex-sell", 6, "unit_price: 1.01 -> 1.00")
	assertTableCellAt(t, annexRecorder, "Currency Conversion Audit Table", 1, "pdf-annex-sell", 8, "16.914")
}

// TestPDFRecorderPreservesCanonicalQuantitiesAndRates verifies presentation
// leaves quantity and normalized rate representations outside the money policy.
// Authored by: OpenCode
func TestPDFRecorderPreservesCanonicalQuantitiesAndRates(t *testing.T) {
	var quantity = *apd.New(1000, -4)
	var quantityBefore = quantity
	var activity = reportmodel.AssetActivityRow{
		SourceID:            "canonical-activity",
		OccurredAt:          time.Date(2024, time.January, 2, 10, 0, 0, 0, time.UTC),
		ActivityType:        reportmodel.ActivityTypeBuy,
		Quantity:            quantity,
		BasisAfterRow:       *apd.New(0, 0),
		CalculationCurrency: "USD",
		QuantityAfterRow:    *apd.New(1, -8),
	}
	var activityRow, err = renderActivityRow(activity)
	if err != nil {
		t.Fatalf("render activity row: %v", err)
	}
	if activityRow[3] != "0.1" || activityRow[7] != "0.00000001" {
		t.Fatalf("activity quantities = %q/%q, want canonical values", activityRow[3], activityRow[7])
	}
	if quantity.Cmp(&quantityBefore) != 0 {
		t.Fatalf("activity quantity changed from %s to %s", quantityBefore.String(), quantity.String())
	}

	var conversion = pdfAnnexConversionEntry()
	conversion.RateValue = *apd.New(169140, -4)
	conversion.Amounts[0].ExchangeRateEvidence.RateValue = conversion.RateValue
	var rateBefore = conversion.RateValue
	var conversionRow []string
	conversionRow, err = renderConversionAuditRow(0, conversion)
	if err != nil {
		t.Fatalf("render conversion row: %v", err)
	}
	if conversionRow[8] != "16.914" {
		t.Fatalf("rate value = %q, want canonical 16.914", conversionRow[8])
	}
	if conversion.RateValue.Cmp(&rateBefore) != 0 || conversion.Amounts[0].ExchangeRateEvidence.RateValue.Cmp(&rateBefore) != 0 {
		t.Fatalf("conversion rate changed from %s to %s", rateBefore.String(), conversion.RateValue.String())
	}
}

// TestRenderMainReportPropagatesBoldWarningLayoutError verifies the dedicated
// warning operation does not swallow a layout-seam failure.
// Authored by: OpenCode
func TestRenderMainReportPropagatesBoldWarningLayoutError(t *testing.T) {
	var recorder = &errorLayoutRecorder{failBoldParagraph: true}
	assertErrorContains(t, func() error { return renderMainReport(recorder, minimalPDFReportFixture(t)) }, "bold warning failed")
}

// TestRendererRenderValidationAndSuccessBranches verifies the exported render
// boundary rejects invalid inputs and returns a PDF payload with extracted text.
// Authored by: OpenCode
func TestRendererRenderValidationAndSuccessBranches(t *testing.T) {
	var _, err = NewRenderer(RenderOptions{})
	if err == nil || !strings.Contains(err.Error(), "font data") {
		t.Fatalf("expected renderer construction to validate fonts, got %v", err)
	}

	var renderer Renderer
	_, err = renderer.Render(minimalPDFReportFixture(t))
	if err == nil || !strings.Contains(err.Error(), "regular font data") {
		t.Fatalf("expected zero-value renderer to reject missing fonts, got %v", err)
	}

	renderer, err = NewRenderer(RenderOptions{Fonts: FontData{Regular: goregular.TTF, Bold: gobold.TTF}})
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	_, err = renderer.Render(reportmodel.CapitalGainsReport{})
	if err == nil || !strings.Contains(err.Error(), "capital gains report year must be greater than zero") {
		t.Fatalf("expected renderer to reject invalid report, got %v", err)
	}

	var payload []byte
	payload, err = renderer.Render(pdfAnnexReportFixture(t))
	if err != nil {
		t.Fatalf("render PDF: %v", err)
	}
	if !bytes.HasPrefix(payload, []byte("%PDF-")) {
		t.Fatalf("expected rendered PDF payload, got %q", payload)
	}
	renderer, err = NewRenderer(RenderOptions{Fonts: FontData{Regular: []byte("not-a-ttf"), Bold: []byte("not-a-ttf")}})
	if err != nil {
		t.Fatalf("new renderer with non-empty invalid font bytes: %v", err)
	}
	_, err = renderer.Render(minimalPDFReportFixture(t))
	if err == nil || !strings.Contains(err.Error(), "load regular font") {
		t.Fatalf("expected render to wrap concrete font-load failure, got %v", err)
	}
}

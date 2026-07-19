package pdf

import (
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

// TestRenderMainReportUsesStructuredLayoutPrimitives verifies BUG-003 behavior:
// headings, styled labels, and tables are emitted as structured layout calls
// instead of a plain sequential line dump.
// Authored by: OpenCode
func TestRenderMainReportUsesStructuredLayoutPrimitives(t *testing.T) {
	var recorder = &layoutRecorder{}
	var err = renderMainReport(recorder, pdfPresentationReportFixture(t))
	if err != nil {
		t.Fatalf("render main report: %v", err)
	}

	assertContains(t, recorder.titles, MainReportTitle)
	assertContains(t, recorder.sections, "Gains-And-Losses Summary")
	assertContains(t, recorder.sections, "Rate Source Summary")
	assertContains(t, recorder.sections, "Reference Section")
	assertContains(t, recorder.subsections, "Historical Position")
	assertKeyValue(t, recorder, "Report Calculation Currency", "USD")
	assertNoSubsection(t, recorder, "Rate Source Summary Table")
	assertNoSubsection(t, recorder, "Reference Table")
	assertTableHeader(t, recorder, "Historical Full Liquidation Count")
	assertTableCell(t, recorder, "BLOCKCHAIN OP")
	assertTableCell(t, recorder, "Converted")
	assertTableCell(t, recorder, "converted-sell")
	if len(recorder.tables) < 3 {
		t.Fatalf("table count = %d, want structured summary/reference/activity tables", len(recorder.tables))
	}
	assertNoMarkdownStructuralSyntax(t, recorder.allText())
}

// TestPDFLayoutSatisfiesRegressionRules verifies the renderer seams for the
// production layout defects patched by BUG-004.
// Authored by: OpenCode
func TestPDFLayoutSatisfiesRegressionRules(t *testing.T) {
	var recorder = &layoutRecorder{}
	var report = pdfNonZeroLiquidationReportFixture(t)
	var conversion = pdfAnnexConversionEntry()
	report.RateSources = []reportmodel.ExchangeRateEvidence{*conversion.Amounts[0].ExchangeRateEvidence}

	var err = renderMainReport(recorder, report)
	if err != nil {
		t.Fatalf("render main report: %v", err)
	}

	assertNoSubsection(t, recorder, "Reference Table")
	assertNoSubsection(t, recorder, "Rate Source Summary Table")
	assertKeyValue(t, recorder, "Authority", reportmodel.RateAuthorityDisplayLabel(conversion.Amounts[0].ExchangeRateEvidence.Authority))
	assertKeyValue(t, recorder, "Provider", rateProviderLabel(conversion.Amounts[0].ExchangeRateEvidence.ProviderID))
	assertSummaryTotalInsideTable(t, recorder)
	assertTablesWithinPrintableWidth(t, recorder)
}

// TestRenderAnnexUsesStructuredLayoutPrimitives verifies Annex 1 uses a page
// break plus table layout for per-asset and conversion evidence.
// Authored by: OpenCode
func TestRenderAnnexUsesStructuredLayoutPrimitives(t *testing.T) {
	var recorder = &layoutRecorder{}
	var report = pdfAnnexReportFixture(t)

	var err = renderAnnex(recorder, report.AuditAnnex)
	if err != nil {
		t.Fatalf("render annex: %v", err)
	}

	assertContains(t, recorder.titles, AnnexTitle)
	assertContains(t, recorder.sections, "Detailed Per-Asset Audit Report")
	assertContains(t, recorder.sections, "Currency Conversion Audit")
	assertContains(t, recorder.subsections, "Asset: BTC")
	assertTableHeader(t, recorder, "Gain/Loss")
	assertTableHeader(t, recorder, "Quote Direction")
	assertTableCell(t, recorder, "pdf-annex-sell")
	assertTableCell(t, recorder, "Base currency per source currency")
	assertNoMarkdownStructuralSyntax(t, recorder.allText())
}

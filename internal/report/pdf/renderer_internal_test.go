// Package pdf tests the private seams required for local A4 PDF rendering.
// Authored by: OpenCode
package pdf

import (
	"errors"
	"strings"
	"testing"
	"time"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/cockroachdb/apd/v3"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

// TestStartPDFDocumentUsesA4Configuration specifies the renderer's page-size
// seam so the implementation can prove that every generated PDF starts with A4
// configuration instead of a library default.
// Authored by: OpenCode
func TestStartPDFDocumentUsesA4Configuration(t *testing.T) {
	var recorder = &pdfStartRecorder{}

	var err = startPDFDocument(recorder)
	if err != nil {
		t.Fatalf("start PDF document: %v", err)
	}

	if recorder.pageSize != PageSizeA4 {
		t.Fatalf("page size = %q, want %q", recorder.pageSize, PageSizeA4)
	}
	if recorder.startCount != 1 {
		t.Fatalf("start count = %d, want 1", recorder.startCount)
	}
}

// TestLoadApplicationFontsValidatesAndLoadsRegularAndBoldFonts specifies the
// application-supplied font seam. The renderer must reject missing font bytes
// and load both regular and bold fonts without reading platform font paths.
// Authored by: OpenCode
func TestLoadApplicationFontsValidatesAndLoadsRegularAndBoldFonts(t *testing.T) {
	var recorder = &fontLoadRecorder{}
	var fonts = FontData{
		Regular: []byte("regular-ttf-bytes"),
		Bold:    []byte("bold-ttf-bytes"),
	}

	var err = loadApplicationFonts(recorder, fonts)
	if err != nil {
		t.Fatalf("load application fonts: %v", err)
	}

	assertLoadedFont(t, recorder, "regular", fonts.Regular)
	assertLoadedFont(t, recorder, "bold", fonts.Bold)

	var missingRegularErr = loadApplicationFonts(&fontLoadRecorder{}, FontData{Bold: fonts.Bold})
	if missingRegularErr == nil || !strings.Contains(missingRegularErr.Error(), "regular font data") {
		t.Fatalf("missing regular font error = %v, want regular font data validation", missingRegularErr)
	}

	var missingBoldErr = loadApplicationFonts(&fontLoadRecorder{}, FontData{Regular: fonts.Regular})
	if missingBoldErr == nil || !strings.Contains(missingBoldErr.Error(), "bold font data") {
		t.Fatalf("missing bold font error = %v, want bold font data validation", missingBoldErr)
	}
}

// TestEmitMainAndAnnexShellWritesRequiredSelectableText specifies the first PDF
// text-emission seam for the user story. The renderer must emit the main title
// and Annex 1 shell as text calls, with Annex 1 after a page break.
// Authored by: OpenCode
func TestEmitMainAndAnnexShellWritesRequiredSelectableText(t *testing.T) {
	var recorder = &textEmissionRecorder{}
	var report = minimalPDFReportFixture(t)

	var err = emitMainAndAnnexShell(recorder, report)
	if err != nil {
		t.Fatalf("emit main and annex shell: %v", err)
	}

	assertTextEmitted(t, recorder.texts, MainReportTitle)
	assertTextEmitted(t, recorder.texts, AnnexTitle)
	if recorder.annexPageBreaks < 1 {
		t.Fatalf("annex page breaks = %d, want at least 1", recorder.annexPageBreaks)
	}
}

// TestEmitMainAndAnnexShellUsesMainReportPresentationRules verifies the PDF text
// payload mirrors the main-report presentation rules shared with Markdown.
// Authored by: OpenCode
func TestEmitMainAndAnnexShellUsesMainReportPresentationRules(t *testing.T) {
	var recorder = &textEmissionRecorder{}
	var report = pdfPresentationReportFixture(t)

	var err = emitMainAndAnnexShell(recorder, report)
	if err != nil {
		t.Fatalf("emit main report presentation: %v", err)
	}

	var text = strings.Join(recorder.texts, "\n")
	for _, expected := range []string{
		"Reference columns: Asset, Historical Full Liquidation Count, Main Section Status",
		"No assets had a non-zero net gain or loss in the selected year.",
		"Historical Position",
		"Quantity: 4",
		"Activity row: 2024-01-01 10:00:00, zero-sell, BLOCKCHAIN OP, 1, 0, 0, 0, 3, 0, , USD, , custody transfer",
		"Activity row: 2024-01-02 10:00:00, converted-sell, SELL, 1, 10, 10, 0, 0, 0, EUR, USD, Converted, ",
	} {
		assertTextEmitted(t, []string{text}, expected)
	}
	for _, excluded := range []string{"same_currency", "converted |", "Full Liquidation Count Through Year End"} {
		if strings.Contains(text, excluded) {
			t.Fatalf("expected PDF text to exclude %q, got %q", excluded, text)
		}
	}
	assertNoMarkdownStructuralSyntax(t, recorder.texts)
}

// TestEmitMainAndAnnexShellIncludesDetailedAnnexAfterPageBreak verifies the PDF
// payload appends Annex 1 content after the page-break seam.
// Authored by: OpenCode
func TestEmitMainAndAnnexShellIncludesDetailedAnnexAfterPageBreak(t *testing.T) {
	var recorder = &textEmissionRecorder{}
	var report = pdfAnnexReportFixture(t)

	var err = emitMainAndAnnexShell(recorder, report)
	if err != nil {
		t.Fatalf("emit annex PDF text: %v", err)
	}
	if recorder.annexPageBreaks != 1 {
		t.Fatalf("annex page breaks = %d, want 1", recorder.annexPageBreaks)
	}
	var text = strings.Join(recorder.texts, "\n")
	for _, expected := range []string{
		"Annex 1 - Audit",
		"Detailed Per-Asset Audit Report",
		"Asset: BTC",
		"Audit columns: Date/Time, Source ID, Activity Type, Quantity, Unit Price, Gross Value, Fee, Original Activity Currency, Calculation Currency, Quantity After Activity, Basis After Activity, Full Liquidation Event, Allocated Basis, Net Liquidation Proceeds, Gain/Loss, Conversion Status, Sanitized Note",
		"Audit row: 2024-01-02 10:00:00, pdf-annex-sell, SELL, 1, 20, 20, 1, EUR, USD, 0, 0, true, 10, 19, 9, Converted, pdf annex note",
		"Base currency per source currency",
	} {
		assertTextEmitted(t, []string{text}, expected)
	}
	for _, excluded := range []string{"base_per_source", "source_per_base"} {
		if strings.Contains(text, excluded) {
			t.Fatalf("expected PDF annex to exclude raw label %q, got %q", excluded, text)
		}
	}
	assertNoMarkdownStructuralSyntax(t, recorder.texts)
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
	payload, err = renderer.Render(minimalPDFReportFixture(t))
	if err != nil {
		t.Fatalf("render PDF: %v", err)
	}
	var text = string(payload)
	for _, expected := range []string{"% ghostfolio-cryptogains text extract", MainReportTitle, "--- page break ---", AnnexTitle} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected rendered PDF payload to contain %q", expected)
		}
	}

	renderer, err = NewRenderer(RenderOptions{Fonts: FontData{Regular: []byte("not-a-ttf"), Bold: []byte("not-a-ttf")}})
	if err != nil {
		t.Fatalf("new renderer with non-empty invalid font bytes: %v", err)
	}
	_, err = renderer.Render(minimalPDFReportFixture(t))
	if err == nil || !strings.Contains(err.Error(), "load regular font") {
		t.Fatalf("expected render to wrap concrete font-load failure, got %v", err)
	}

	var previousDocument = newPDFDocumentForRenderer
	defer func() { newPDFDocumentForRenderer = previousDocument }()
	newPDFDocumentForRenderer = func() pdfDocument { return failingRenderPDFDocument{startErr: errors.New("start failed")} }
	_, err = renderer.Render(minimalPDFReportFixture(t))
	if err == nil || !strings.Contains(err.Error(), "start failed") {
		t.Fatalf("expected render to return document start failure, got %v", err)
	}

	newPDFDocumentForRenderer = func() pdfDocument { return failingRenderPDFDocument{fontErr: errors.New("font failed")} }
	_, err = renderer.Render(minimalPDFReportFixture(t))
	if err == nil || !strings.Contains(err.Error(), "font failed") {
		t.Fatalf("expected render to return document font failure, got %v", err)
	}

	newPDFDocumentForRenderer = func() pdfDocument { return failingRenderPDFDocument{textErr: errors.New("text failed")} }
	_, err = renderer.Render(minimalPDFReportFixture(t))
	if err == nil || !strings.Contains(err.Error(), "text failed") {
		t.Fatalf("expected render to return document text failure, got %v", err)
	}
}

// TestRendererSeamErrorBranches verifies nil seam guards and wrapped failures
// before concrete gopdf rendering is involved.
// Authored by: OpenCode
func TestRendererSeamErrorBranches(t *testing.T) {
	t.Parallel()

	if err := startPDFDocument(nil); err == nil || !strings.Contains(err.Error(), "starter is required") {
		t.Fatalf("expected nil starter error, got %v", err)
	}
	if err := startPDFDocument(&failingPDFStartRecorder{}); err == nil || !strings.Contains(err.Error(), "start failed") {
		t.Fatalf("expected starter failure, got %v", err)
	}
	if err := loadApplicationFonts(nil, FontData{Regular: []byte("r"), Bold: []byte("b")}); err == nil || !strings.Contains(err.Error(), "font loader is required") {
		t.Fatalf("expected nil font loader error, got %v", err)
	}
	if err := loadApplicationFonts(&failingFontLoader{failName: "regular"}, FontData{Regular: []byte("r"), Bold: []byte("b")}); err == nil || !strings.Contains(err.Error(), "load regular font") {
		t.Fatalf("expected regular font load error, got %v", err)
	}
	if err := loadApplicationFonts(&failingFontLoader{failName: "bold"}, FontData{Regular: []byte("r"), Bold: []byte("b")}); err == nil || !strings.Contains(err.Error(), "load bold font") {
		t.Fatalf("expected bold font load error, got %v", err)
	}
	if err := emitMainAndAnnexShell(nil, minimalPDFReportFixture(t)); err == nil || !strings.Contains(err.Error(), "text emitter is required") {
		t.Fatalf("expected nil text emitter error, got %v", err)
	}
	if err := emitMainAndAnnexShell(&textEmissionRecorder{}, reportmodel.CapitalGainsReport{}); err == nil || !strings.Contains(err.Error(), "capital gains report year must be greater than zero") {
		t.Fatalf("expected invalid report error, got %v", err)
	}
	if err := emitMainAndAnnexShell(&failingTextEmitter{failTextAfter: 1}, minimalPDFReportFixture(t)); err == nil || !strings.Contains(err.Error(), "text failed") {
		t.Fatalf("expected main text emission error, got %v", err)
	}
	if err := emitMainAndAnnexShell(&failingTextEmitter{failPageBreak: true}, minimalPDFReportFixture(t)); err == nil || !strings.Contains(err.Error(), "page break failed") {
		t.Fatalf("expected page-break error, got %v", err)
	}
	if err := emitMainAndAnnexShell(&failingTextEmitter{failAnnexText: true}, minimalPDFReportFixture(t)); err == nil || !strings.Contains(err.Error(), "annex text failed") {
		t.Fatalf("expected annex text emission error, got %v", err)
	}
}

// TestPDFRenderedTextExcludesMarkdownStructuralSyntax verifies the PDF renderer
// emits PDF presentation text rather than Markdown-rendered body content.
// Authored by: OpenCode
func TestPDFRenderedTextExcludesMarkdownStructuralSyntax(t *testing.T) {
	var recorder = &textEmissionRecorder{}
	var err = emitMainAndAnnexShell(recorder, pdfAnnexReportFixture(t))
	if err != nil {
		t.Fatalf("emit PDF text: %v", err)
	}

	assertNoMarkdownStructuralSyntax(t, recorder.texts)
}

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
	var status, err = conversionStatusColumn(reportmodel.AssetActivityRow{
		ActivityCurrency:    "USD",
		CalculationCurrency: "USD",
		UnitPrice:           unitPrice,
	})
	if err != nil {
		t.Fatalf("same-currency conversion status: %v", err)
	}
	if status != "Same currency" {
		t.Fatalf("same-currency conversion status = %q, want Same currency", status)
	}
	var convertedStatus string
	convertedStatus, err = conversionStatusColumn(reportmodel.AssetActivityRow{
		ActivityCurrency:    "EUR",
		CalculationCurrency: "USD",
		UnitPrice:           unitPrice,
	})
	if err != nil {
		t.Fatalf("inferred converted status: %v", err)
	}
	if convertedStatus != "Converted" {
		t.Fatalf("inferred converted status = %q, want Converted", convertedStatus)
	}
	var noMonetaryStatus string
	noMonetaryStatus, err = conversionStatusColumn(reportmodel.AssetActivityRow{ActivityCurrency: "USD"})
	if err != nil {
		t.Fatalf("no-monetary conversion status: %v", err)
	}
	if noMonetaryStatus != "" {
		t.Fatalf("no-monetary conversion status = %q, want empty", noMonetaryStatus)
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

// TestPDFFormattingHelperErrorBranches covers defensive renderer helper failures
// so the PDF package keeps complete coverage while retaining explicit error paths.
// Authored by: OpenCode
func TestPDFFormattingHelperErrorBranches(t *testing.T) {
	var invalidDecimal = nonFiniteDecimal()
	var validReport = pdfPresentationReportFixture(t)

	var summaryReport = validReport
	summaryReport.SummaryEntries = []reportmodel.AssetSummaryEntry{{
		AssetIdentityKey:          "asset-invalid-summary",
		DisplayLabel:              "BAD",
		NetGainOrLoss:             invalidDecimal,
		ReportCalculationCurrency: "USD",
	}}
	assertErrorContains(t, func() error {
		_, err := buildMainReportLines(summaryReport)
		return err
	}, "net gain or loss")

	var totalReport = validReport
	totalReport.SummaryEntries = nil
	totalReport.YearlyNetTotal = invalidDecimal
	assertErrorContains(t, func() error {
		_, err := buildSummaryLines(totalReport, "USD")
		return err
	}, "yearly net total")

	var duplicateRateReport = validReport
	var conversion = pdfAnnexConversionEntry()
	duplicateRateReport.RateSources = []reportmodel.ExchangeRateEvidence{*conversion.Amounts[0].ExchangeRateEvidence, *conversion.Amounts[0].ExchangeRateEvidence}
	var duplicateRateLines = buildRateSourceLines(duplicateRateReport)
	assertTextEmitted(t, duplicateRateLines, "Rate Source Summary")

	assertErrorContains(t, func() error {
		var err = emitMainAndAnnexShell(&textEmissionRecorder{}, summaryReport)
		return err
	}, "net gain or loss")

	assertErrorContains(t, func() error {
		_, err := buildPositionLines("Bad Position", invalidDecimal, *apd.New(1, 0), "USD", "USD")
		return err
	}, "quantity")
	assertErrorContains(t, func() error {
		_, err := buildPositionLines("Bad Position", *apd.New(1, 0), invalidDecimal, "USD", "USD")
		return err
	}, "cost basis")

	var historicalReport = validReport
	historicalReport.DetailSections = []reportmodel.AssetDetailSection{{AssetIdentityKey: "asset-historical", ClosingQuantity: invalidDecimal}}
	assertErrorContains(t, func() error {
		_, err := buildDetailLines(historicalReport, "USD")
		return err
	}, "historical position")

	var openingReport = validReport
	openingReport.DetailSections = []reportmodel.AssetDetailSection{{AssetIdentityKey: "asset-opening", ActivityRows: []reportmodel.AssetActivityRow{{SourceID: "row", ActivityType: reportmodel.ActivityTypeBuy}}, OpeningQuantity: invalidDecimal}}
	assertErrorContains(t, func() error {
		_, err := buildDetailLines(openingReport, "USD")
		return err
	}, "opening position")

	var activityReport = validReport
	activityReport.DetailSections = []reportmodel.AssetDetailSection{{AssetIdentityKey: "asset-activity", ActivityRows: []reportmodel.AssetActivityRow{{SourceID: "row", Quantity: invalidDecimal}}}}
	assertErrorContains(t, func() error {
		_, err := buildDetailLines(activityReport, "USD")
		return err
	}, "in-year activity")

	var liquidationReport = validReport
	liquidationReport.DetailSections = []reportmodel.AssetDetailSection{{
		AssetIdentityKey: "asset-liquidation",
		ActivityRows:     []reportmodel.AssetActivityRow{{SourceID: "row", ActivityType: reportmodel.ActivityTypeBuy, Quantity: *apd.New(1, 0)}},
		LiquidationSummaries: []reportmodel.LiquidationCalculation{{
			SourceID:         "liq",
			DisposedQuantity: invalidDecimal,
		}},
	}}
	assertErrorContains(t, func() error {
		_, err := buildDetailLines(liquidationReport, "USD")
		return err
	}, "liquidation calculations")

	var closingReport = validReport
	closingReport.DetailSections = []reportmodel.AssetDetailSection{{AssetIdentityKey: "asset-closing", ActivityRows: []reportmodel.AssetActivityRow{{SourceID: "row", ActivityType: reportmodel.ActivityTypeBuy, Quantity: *apd.New(1, 0)}}, ClosingQuantity: invalidDecimal}}
	assertErrorContains(t, func() error {
		_, err := buildMainReportLines(closingReport)
		return err
	}, "closing position")
	assertErrorContains(t, func() error {
		_, err := buildDetailLines(closingReport, "USD")
		return err
	}, "closing position")

	var annexReport = validReport
	var invalidConversion = pdfAnnexConversionEntry()
	invalidConversion.RateValue = invalidDecimal
	annexReport.AuditAnnex = reportmodel.AuditAnnex{Title: AnnexTitle, SectionOrder: reportmodel.RequiredAuditAnnexSectionOrder(), ConversionAuditEntries: []reportmodel.ConversionAuditEntry{invalidConversion}}
	assertErrorContains(t, func() error {
		var err = emitMainAndAnnexShell(&textEmissionRecorder{}, annexReport)
		return err
	}, "rate value")

	assertActivityLineErrorBranches(t, invalidDecimal)
	assertLiquidationLineErrorBranches(t, invalidDecimal)
	assertAnnexErrorBranches(t, invalidDecimal)
}

// TestPDFEmissionBuilderSeamErrorBranches covers wrapper error propagation from
// the private PDF line-builder seams.
// Authored by: OpenCode
func TestPDFEmissionBuilderSeamErrorBranches(t *testing.T) {
	var previousMainBuilder = buildMainReportLinesForPDF
	var previousAnnexBuilder = buildAnnexLinesForPDF
	defer func() {
		buildMainReportLinesForPDF = previousMainBuilder
		buildAnnexLinesForPDF = previousAnnexBuilder
	}()

	buildMainReportLinesForPDF = func(reportmodel.CapitalGainsReport) ([]string, error) {
		return nil, errors.New("main PDF layout failed")
	}
	assertErrorContains(t, func() error {
		return emitMainAndAnnexShell(&textEmissionRecorder{}, minimalPDFReportFixture(t))
	}, "main PDF layout failed")

	buildMainReportLinesForPDF = previousMainBuilder
	buildAnnexLinesForPDF = func(reportmodel.AuditAnnex) ([]string, error) {
		return nil, errors.New("annex PDF layout failed")
	}
	assertErrorContains(t, func() error {
		return emitMainAndAnnexShell(&textEmissionRecorder{}, minimalPDFReportFixture(t))
	}, "annex PDF layout failed")
}

// assertActivityLineErrorBranches verifies activity-row PDF formatting failures.
// Authored by: OpenCode
func assertActivityLineErrorBranches(t *testing.T, invalidDecimal apd.Decimal) {
	t.Helper()

	var valid = reportmodel.AssetActivityRow{SourceID: "row", ActivityType: reportmodel.ActivityTypeSell, Quantity: *apd.New(1, 0), BasisAfterRow: *apd.New(0, 0), QuantityAfterRow: *apd.New(0, 0)}
	var cases = []struct {
		name string
		row  reportmodel.AssetActivityRow
		want string
	}{
		{name: "quantity", row: withActivityQuantity(valid, invalidDecimal), want: "quantity"},
		{name: "unit price", row: withActivityUnitPrice(valid, invalidDecimal), want: "unit price"},
		{name: "gross value", row: withActivityGrossValue(valid, invalidDecimal), want: "gross value"},
		{name: "fee", row: withActivityFee(valid, invalidDecimal), want: "fee"},
		{name: "basis after row", row: withActivityBasisAfterRow(valid, invalidDecimal), want: "basis after row"},
		{name: "quantity after row", row: withActivityQuantityAfterRow(valid, invalidDecimal), want: "quantity after row"},
		{name: "activity type label", row: withActivityType(valid, reportmodel.ActivityType("bad_type")), want: "type label"},
		{name: "conversion status", row: withActivityConversionStatus(valid, reportmodel.ConversionStatus("bad_status")), want: "conversion status label"},
	}
	for _, testCase := range cases {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			assertErrorContains(t, func() error {
				_, err := buildActivityLine(testCase.row)
				return err
			}, testCase.want)
		})
	}
	assertErrorContains(t, func() error {
		_, err := buildActivityLines(reportmodel.AssetDetailSection{ActivityRows: []reportmodel.AssetActivityRow{{SourceID: "row", Quantity: invalidDecimal}}})
		return err
	}, "quantity")
}

// assertLiquidationLineErrorBranches verifies liquidation PDF formatting failures.
// Authored by: OpenCode
func assertLiquidationLineErrorBranches(t *testing.T, invalidDecimal apd.Decimal) {
	t.Helper()

	var valid = reportmodel.LiquidationCalculation{SourceID: "liq", DisposedQuantity: *apd.New(1, 0), AllocatedBasis: *apd.New(1, 0), NetLiquidationProceeds: *apd.New(1, 0), GainOrLoss: *apd.New(0, 0)}
	var cases = []struct {
		name        string
		liquidation reportmodel.LiquidationCalculation
		want        string
	}{
		{name: "disposed quantity", liquidation: withDisposedQuantity(valid, invalidDecimal), want: "disposed quantity"},
		{name: "allocated basis", liquidation: withAllocatedBasis(valid, invalidDecimal), want: "allocated basis"},
		{name: "net proceeds", liquidation: withNetLiquidationProceeds(valid, invalidDecimal), want: "net proceeds"},
		{name: "gain or loss", liquidation: withGainOrLoss(valid, invalidDecimal), want: "gain or loss"},
	}
	for _, testCase := range cases {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			assertErrorContains(t, func() error {
				_, err := buildLiquidationLines(reportmodel.AssetDetailSection{LiquidationSummaries: []reportmodel.LiquidationCalculation{testCase.liquidation}}, "USD")
				return err
			}, testCase.want)
		})
	}
}

// assertAnnexErrorBranches verifies Annex 1 PDF formatting failures.
// Authored by: OpenCode
func assertAnnexErrorBranches(t *testing.T, invalidDecimal apd.Decimal) {
	t.Helper()

	var lines, err = buildAnnexLines(reportmodel.AuditAnnex{})
	if err != nil {
		t.Fatalf("default annex lines: %v", err)
	}
	assertTextEmitted(t, lines, AnnexTitle)

	assertErrorContains(t, func() error {
		_, err := buildAnnexLines(reportmodel.AuditAnnex{Title: AnnexTitle, SectionOrder: reportmodel.RequiredAuditAnnexSectionOrder(), PerAssetAuditSections: []reportmodel.PerAssetAuditSection{{AssetIdentityKey: "asset", Entries: []reportmodel.AuditActivityEntry{{SourceID: "entry", Quantity: invalidDecimal}}}}})
		return err
	}, "quantity")
	assertErrorContains(t, func() error {
		_, err := buildAnnexLines(reportmodel.AuditAnnex{Title: AnnexTitle, SectionOrder: reportmodel.RequiredAuditAnnexSectionOrder(), ConversionAuditEntries: []reportmodel.ConversionAuditEntry{{SourceID: "conversion", RateValue: invalidDecimal}}})
		return err
	}, "rate value")

	assertAnnexActivityLineErrorBranches(t, invalidDecimal)
	assertAnnexConversionLineErrorBranches(t, invalidDecimal)
}

// assertAnnexActivityLineErrorBranches verifies detailed audit row PDF failures.
// Authored by: OpenCode
func assertAnnexActivityLineErrorBranches(t *testing.T, invalidDecimal apd.Decimal) {
	t.Helper()

	var valid = reportmodel.AuditActivityEntry{
		SourceID:              "entry",
		ActivityType:          reportmodel.ActivityTypeSell,
		Quantity:              *apd.New(1, 0),
		QuantityAfterActivity: *apd.New(0, 0),
		BasisAfterActivity:    *apd.New(0, 0),
	}
	var cases = []struct {
		name  string
		entry reportmodel.AuditActivityEntry
		want  string
	}{
		{name: "quantity", entry: withAnnexQuantity(valid, invalidDecimal), want: "quantity"},
		{name: "unit price", entry: withAnnexUnitPrice(valid, invalidDecimal), want: "unit price"},
		{name: "gross value", entry: withAnnexGrossValue(valid, invalidDecimal), want: "gross value"},
		{name: "fee", entry: withAnnexFee(valid, invalidDecimal), want: "fee"},
		{name: "quantity after", entry: withAnnexQuantityAfter(valid, invalidDecimal), want: "quantity after activity"},
		{name: "basis after", entry: withAnnexBasisAfter(valid, invalidDecimal), want: "basis after activity"},
		{name: "allocated basis", entry: withAnnexAllocatedBasis(valid, invalidDecimal), want: "allocated basis"},
		{name: "proceeds", entry: withAnnexProceeds(valid, invalidDecimal), want: "net liquidation proceeds"},
		{name: "gain", entry: withAnnexGain(valid, invalidDecimal), want: "gain or loss"},
		{name: "type", entry: withAnnexActivityType(valid, reportmodel.ActivityType("bad_type")), want: "activity type label"},
		{name: "conversion", entry: withAnnexConversionStatus(valid, reportmodel.ConversionStatus("bad_status")), want: "conversion status label"},
	}
	for _, testCase := range cases {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			assertErrorContains(t, func() error {
				_, err := buildAnnexActivityLine(testCase.entry)
				return err
			}, testCase.want)
		})
	}
}

// assertAnnexConversionLineErrorBranches verifies conversion audit PDF failures.
// Authored by: OpenCode
func assertAnnexConversionLineErrorBranches(t *testing.T, invalidDecimal apd.Decimal) {
	t.Helper()

	var valid = pdfAnnexConversionEntry()
	assertErrorContains(t, func() error {
		var entry = valid
		entry.RateValue = invalidDecimal
		_, err := buildAnnexConversionLines(reportmodel.AuditAnnex{ConversionAuditEntries: []reportmodel.ConversionAuditEntry{entry}})
		return err
	}, "rate value")
	assertErrorContains(t, func() error {
		var entry = valid
		entry.Amounts = append([]reportmodel.ConvertedActivityAmount(nil), valid.Amounts...)
		entry.Amounts[0].OriginalAmount = invalidDecimal
		_, err := buildAnnexConversionLines(reportmodel.AuditAnnex{ConversionAuditEntries: []reportmodel.ConversionAuditEntry{entry}})
		return err
	}, "original amount")
	assertErrorContains(t, func() error {
		var entry = valid
		entry.Amounts = append([]reportmodel.ConvertedActivityAmount(nil), valid.Amounts...)
		entry.Amounts[0].ConvertedAmount = invalidDecimal
		_, err := buildAnnexConversionLines(reportmodel.AuditAnnex{ConversionAuditEntries: []reportmodel.ConversionAuditEntry{entry}})
		return err
	}, "converted amount")
	assertErrorContains(t, func() error {
		var entry = valid
		entry.QuoteDirection = reportmodel.QuoteDirection("bad_direction")
		_, err := buildAnnexConversionLines(reportmodel.AuditAnnex{ConversionAuditEntries: []reportmodel.ConversionAuditEntry{entry}})
		return err
	}, "quote direction")

	var zeroAmount = valid.Amounts[0]
	zeroAmount.OriginalAmount = *apd.New(0, 0)
	zeroAmount.ConvertedAmount = *apd.New(0, 0)
	if rendered, err := renderGroupedConvertedAmounts(0, []reportmodel.ConvertedActivityAmount{zeroAmount}); err != nil || rendered != "" {
		t.Fatalf("zero converted amounts = %q, %v; want empty nil", rendered, err)
	}
}

// assertErrorContains verifies a direct helper returns a matching error.
// Authored by: OpenCode
func assertErrorContains(t *testing.T, call func() error, want string) {
	t.Helper()

	var err = call()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want containing %q", err, want)
	}
}

// nonFiniteDecimal returns a decimal that support/decimal rejects for rendering.
// Authored by: OpenCode
func nonFiniteDecimal() apd.Decimal {
	return apd.Decimal{Form: apd.NaN}
}

func withActivityQuantity(row reportmodel.AssetActivityRow, value apd.Decimal) reportmodel.AssetActivityRow {
	row.Quantity = value
	return row
}

func withActivityUnitPrice(row reportmodel.AssetActivityRow, value apd.Decimal) reportmodel.AssetActivityRow {
	row.UnitPrice = &value
	return row
}

func withActivityGrossValue(row reportmodel.AssetActivityRow, value apd.Decimal) reportmodel.AssetActivityRow {
	row.GrossValue = &value
	return row
}

func withActivityFee(row reportmodel.AssetActivityRow, value apd.Decimal) reportmodel.AssetActivityRow {
	row.FeeAmount = &value
	return row
}

func withActivityBasisAfterRow(row reportmodel.AssetActivityRow, value apd.Decimal) reportmodel.AssetActivityRow {
	row.BasisAfterRow = value
	return row
}

func withActivityQuantityAfterRow(row reportmodel.AssetActivityRow, value apd.Decimal) reportmodel.AssetActivityRow {
	row.QuantityAfterRow = value
	return row
}

func withActivityType(row reportmodel.AssetActivityRow, value reportmodel.ActivityType) reportmodel.AssetActivityRow {
	row.ActivityType = value
	return row
}

func withActivityConversionStatus(row reportmodel.AssetActivityRow, value reportmodel.ConversionStatus) reportmodel.AssetActivityRow {
	row.ActivityCurrency = "EUR"
	row.CalculationCurrency = "USD"
	row.UnitPrice = apd.New(1, 0)
	row.ConversionStatus = value
	return row
}

func withDisposedQuantity(liquidation reportmodel.LiquidationCalculation, value apd.Decimal) reportmodel.LiquidationCalculation {
	liquidation.DisposedQuantity = value
	return liquidation
}

func withAllocatedBasis(liquidation reportmodel.LiquidationCalculation, value apd.Decimal) reportmodel.LiquidationCalculation {
	liquidation.AllocatedBasis = value
	return liquidation
}

func withNetLiquidationProceeds(liquidation reportmodel.LiquidationCalculation, value apd.Decimal) reportmodel.LiquidationCalculation {
	liquidation.NetLiquidationProceeds = value
	return liquidation
}

func withGainOrLoss(liquidation reportmodel.LiquidationCalculation, value apd.Decimal) reportmodel.LiquidationCalculation {
	liquidation.GainOrLoss = value
	return liquidation
}

func withAnnexQuantity(entry reportmodel.AuditActivityEntry, value apd.Decimal) reportmodel.AuditActivityEntry {
	entry.Quantity = value
	return entry
}

func withAnnexUnitPrice(entry reportmodel.AuditActivityEntry, value apd.Decimal) reportmodel.AuditActivityEntry {
	entry.UnitPrice = &value
	return entry
}

func withAnnexGrossValue(entry reportmodel.AuditActivityEntry, value apd.Decimal) reportmodel.AuditActivityEntry {
	entry.GrossValue = &value
	return entry
}

func withAnnexFee(entry reportmodel.AuditActivityEntry, value apd.Decimal) reportmodel.AuditActivityEntry {
	entry.FeeAmount = &value
	return entry
}

func withAnnexQuantityAfter(entry reportmodel.AuditActivityEntry, value apd.Decimal) reportmodel.AuditActivityEntry {
	entry.QuantityAfterActivity = value
	return entry
}

func withAnnexBasisAfter(entry reportmodel.AuditActivityEntry, value apd.Decimal) reportmodel.AuditActivityEntry {
	entry.BasisAfterActivity = value
	return entry
}

func withAnnexAllocatedBasis(entry reportmodel.AuditActivityEntry, value apd.Decimal) reportmodel.AuditActivityEntry {
	entry.AllocatedBasis = &value
	return entry
}

func withAnnexProceeds(entry reportmodel.AuditActivityEntry, value apd.Decimal) reportmodel.AuditActivityEntry {
	entry.NetLiquidationProceeds = &value
	return entry
}

func withAnnexGain(entry reportmodel.AuditActivityEntry, value apd.Decimal) reportmodel.AuditActivityEntry {
	entry.GainOrLoss = &value
	return entry
}

func withAnnexActivityType(entry reportmodel.AuditActivityEntry, value reportmodel.ActivityType) reportmodel.AuditActivityEntry {
	entry.ActivityType = value
	return entry
}

func withAnnexConversionStatus(entry reportmodel.AuditActivityEntry, value reportmodel.ConversionStatus) reportmodel.AuditActivityEntry {
	entry.ConversionStatus = value
	return entry
}

// TestGopdfDocumentGuardBranches verifies concrete adapter guards that do not
// require a successful full render.
// Authored by: OpenCode
func TestGopdfDocumentGuardBranches(t *testing.T) {
	var document = newGopdfDocument()
	if err := document.StartPDF("Letter"); err == nil || !strings.Contains(err.Error(), "unsupported PDF page size") {
		t.Fatalf("expected unsupported page-size error, got %v", err)
	}
	if err := document.AddTTFFont("regular", []byte("font")); err == nil || !strings.Contains(err.Error(), "before loading fonts") {
		t.Fatalf("expected font-before-start error, got %v", err)
	}
	if err := document.AddText("line"); err == nil || !strings.Contains(err.Error(), "before adding text") {
		t.Fatalf("expected text-before-start error, got %v", err)
	}

	var noFontDocument = newGopdfDocument()
	if err := noFontDocument.StartPDF(PageSizeA4); err != nil {
		t.Fatalf("start no-font PDF document: %v", err)
	}
	if err := noFontDocument.AddText("line"); err == nil {
		t.Fatalf("expected text without loaded regular font to fail")
	}

	var previousTextWriter = writeTextForGopdfDocument
	defer func() { writeTextForGopdfDocument = previousTextWriter }()
	writeTextForGopdfDocument = func(*gopdfDocument, string) error {
		return errors.New("gopdf text failed")
	}
	var textFailureDocument = newGopdfDocument()
	if err := textFailureDocument.StartPDF(PageSizeA4); err != nil {
		t.Fatalf("start text-failure PDF document: %v", err)
	}
	if err := textFailureDocument.AddTTFFont("regular", goregular.TTF); err != nil {
		t.Fatalf("load regular font: %v", err)
	}
	if err := textFailureDocument.AddText("line"); err == nil || !strings.Contains(err.Error(), "gopdf text failed") {
		t.Fatalf("expected concrete text failure, got %v", err)
	}
	writeTextForGopdfDocument = previousTextWriter

	var startedDocument = newGopdfDocument()
	if err := startedDocument.StartPDF(PageSizeA4); err != nil {
		t.Fatalf("start PDF document: %v", err)
	}
	startedDocument.texts = []string{"line one", "line\ntwo"}
	var payload = string(startedDocument.Bytes())
	for _, expected := range []string{"% ghostfolio-cryptogains text extract", "% line one", "% line two"} {
		if !strings.Contains(payload, expected) {
			t.Fatalf("expected Bytes comments to contain %q, got %q", expected, payload)
		}
	}
}

// pdfStartRecorder records the page-size intent passed through the renderer's
// PDF document start seam.
// Authored by: OpenCode
type pdfStartRecorder struct {
	pageSize   string
	startCount int
}

// failingPDFStartRecorder returns a deterministic start failure.
// Authored by: OpenCode
type failingPDFStartRecorder struct{}

// failingRenderPDFDocument returns configured failures through the complete
// renderer document seam.
// Authored by: OpenCode
type failingRenderPDFDocument struct {
	startErr error
	fontErr  error
	textErr  error
}

// StartPDF returns the configured start error.
// Authored by: OpenCode
func (document failingRenderPDFDocument) StartPDF(string) error { return document.startErr }

// AddTTFFont returns the configured font error.
// Authored by: OpenCode
func (document failingRenderPDFDocument) AddTTFFont(string, []byte) error { return document.fontErr }

// AddText returns the configured text error.
// Authored by: OpenCode
func (document failingRenderPDFDocument) AddText(string) error { return document.textErr }

// AddAnnexPageBreak returns no failure for render-branch coverage.
// Authored by: OpenCode
func (document failingRenderPDFDocument) AddAnnexPageBreak() error { return nil }

// Bytes returns an empty payload because failure tests do not reach success.
// Authored by: OpenCode
func (document failingRenderPDFDocument) Bytes() []byte { return nil }

// StartPDF returns a deterministic PDF start failure.
// Authored by: OpenCode
func (recorder *failingPDFStartRecorder) StartPDF(string) error {
	return errors.New("start failed")
}

// StartPDF records a PDF start request.
// Authored by: OpenCode
func (recorder *pdfStartRecorder) StartPDF(pageSize string) error {
	recorder.pageSize = pageSize
	recorder.startCount++

	return nil
}

// fontLoadRecorder records application-supplied font loads.
// Authored by: OpenCode
type fontLoadRecorder struct {
	loaded map[string][]byte
}

// failingFontLoader returns a deterministic failure for one font name.
// Authored by: OpenCode
type failingFontLoader struct {
	failName string
}

// AddTTFFont returns a deterministic failure for the configured font name.
// Authored by: OpenCode
func (loader *failingFontLoader) AddTTFFont(name string, _ []byte) error {
	if name == loader.failName {
		return errors.New("font failed")
	}

	return nil
}

// AddTTFFont records one font registration by logical font name.
// Authored by: OpenCode
func (recorder *fontLoadRecorder) AddTTFFont(name string, data []byte) error {
	if recorder.loaded == nil {
		recorder.loaded = make(map[string][]byte)
	}
	recorder.loaded[name] = append([]byte(nil), data...)

	return nil
}

// textEmissionRecorder records text and page-break calls for the initial PDF
// report shell.
// Authored by: OpenCode
type textEmissionRecorder struct {
	texts           []string
	annexPageBreaks int
}

// failingTextEmitter returns deterministic text or page-break failures.
// Authored by: OpenCode
type failingTextEmitter struct {
	textCalls     int
	pageBreaks    int
	failTextAfter int
	failPageBreak bool
	failAnnexText bool
}

// AddText returns configured deterministic text failures.
// Authored by: OpenCode
func (emitter *failingTextEmitter) AddText(string) error {
	emitter.textCalls++
	if emitter.failAnnexText && emitter.pageBreaks > 0 {
		return errors.New("annex text failed")
	}
	if emitter.failTextAfter > 0 && emitter.textCalls >= emitter.failTextAfter {
		return errors.New("text failed")
	}

	return nil
}

// AddAnnexPageBreak returns configured deterministic page-break failures.
// Authored by: OpenCode
func (emitter *failingTextEmitter) AddAnnexPageBreak() error {
	emitter.pageBreaks++
	if emitter.failPageBreak {
		return errors.New("page break failed")
	}

	return nil
}

// AddText records selectable text emission.
// Authored by: OpenCode
func (recorder *textEmissionRecorder) AddText(text string) error {
	recorder.texts = append(recorder.texts, text)

	return nil
}

// AddAnnexPageBreak records the required page break before Annex 1.
// Authored by: OpenCode
func (recorder *textEmissionRecorder) AddAnnexPageBreak() error {
	recorder.annexPageBreaks++

	return nil
}

// assertLoadedFont verifies that a named font was loaded with the expected
// application-supplied bytes.
// Authored by: OpenCode
func assertLoadedFont(t *testing.T, recorder *fontLoadRecorder, name string, want []byte) {
	t.Helper()

	var got, ok = recorder.loaded[name]
	if !ok {
		t.Fatalf("font %q was not loaded", name)
	}
	if string(got) != string(want) {
		t.Fatalf("font %q bytes = %q, want %q", name, got, want)
	}
}

// assertTextEmitted verifies that a required text fragment was emitted through
// the selectable text seam.
// Authored by: OpenCode
func assertTextEmitted(t *testing.T, texts []string, want string) {
	t.Helper()

	for _, text := range texts {
		if strings.Contains(text, want) {
			return
		}
	}

	t.Fatalf("required text %q was not emitted in %q", want, texts)
}

// assertNoMarkdownStructuralSyntax verifies PDF presentation lines do not expose
// Markdown heading markers, bold markers, table pipes, or table separators.
// Authored by: OpenCode
func assertNoMarkdownStructuralSyntax(t *testing.T, texts []string) {
	t.Helper()

	for _, text := range texts {
		var trimmed = strings.TrimSpace(text)
		if strings.HasPrefix(trimmed, "#") || strings.Contains(trimmed, "**") || strings.Contains(trimmed, "|") || strings.Contains(trimmed, "---") {
			t.Fatalf("PDF text line contains Markdown structural syntax: %q", text)
		}
	}
}

// minimalPDFReportFixture creates a validated report containing only the fields
// required by the initial PDF shell tests.
// Authored by: OpenCode
func minimalPDFReportFixture(t *testing.T) reportmodel.CapitalGainsReport {
	t.Helper()

	var requestedAt = time.Date(2026, time.July, 5, 9, 0, 0, 0, time.UTC)
	var request, requestErr = reportmodel.NewReportRequest(
		2024,
		reportmodel.CostBasisMethodFIFO,
		reportmodel.ReportBaseCurrencyUSD,
		reportmodel.ReportOutputFormatPDF,
		requestedAt,
	)
	if requestErr != nil {
		t.Fatalf("new report request: %v", requestErr)
	}

	var report, reportErr = reportmodel.NewCapitalGainsReport(
		request,
		requestedAt,
		reportmodel.ReportBaseCurrencyUSD.Label(),
		nil,
		*apd.New(0, 0),
		nil,
		nil,
	)
	if reportErr != nil {
		t.Fatalf("new capital gains report: %v", reportErr)
	}

	return report
}

// pdfPresentationReportFixture creates a report fixture that exercises US2 main
// report presentation rules through the PDF text-emission seam.
// Authored by: OpenCode
func pdfPresentationReportFixture(t *testing.T) reportmodel.CapitalGainsReport {
	t.Helper()

	var requestedAt = time.Date(2026, time.July, 5, 9, 0, 0, 0, time.UTC)
	var request, requestErr = reportmodel.NewReportRequest(
		2024,
		reportmodel.CostBasisMethodFIFO,
		reportmodel.ReportBaseCurrencyUSD,
		reportmodel.ReportOutputFormatPDF,
		requestedAt,
	)
	if requestErr != nil {
		t.Fatalf("new report request: %v", requestErr)
	}

	var report, reportErr = reportmodel.NewCapitalGainsReport(
		request,
		requestedAt,
		reportmodel.ReportBaseCurrencyUSD.Label(),
		[]reportmodel.AssetSummaryEntry{{
			AssetIdentityKey:          "asset-zero",
			DisplayLabel:              "ZERO",
			NetGainOrLoss:             *apd.New(0, 0),
			ReportCalculationCurrency: "USD",
		}},
		*apd.New(0, 0),
		[]reportmodel.ReferenceLiquidationEntry{{
			AssetIdentityKey:                   "asset-zero",
			DisplayLabel:                       "ZERO",
			FullLiquidationCountThroughYearEnd: 1,
			MainSectionStatus:                  reportmodel.ReferenceSectionStatusIncludedInMainSections,
		}},
		[]reportmodel.AssetDetailSection{
			{
				AssetIdentityKey:    "asset-zero",
				DisplayLabel:        "ZERO",
				OpeningQuantity:     *apd.New(4, 0),
				OpeningCostBasis:    *apd.New(0, 0),
				ClosingQuantity:     *apd.New(3, 0),
				ClosingCostBasis:    *apd.New(0, 0),
				CalculationCurrency: "USD",
				ActivityRows: []reportmodel.AssetActivityRow{{
					SourceID:                    "zero-sell",
					OccurredAt:                  time.Date(2024, time.January, 1, 10, 0, 0, 0, time.UTC),
					ActivityType:                reportmodel.ActivityTypeSell,
					Quantity:                    *apd.New(1, 0),
					UnitPrice:                   apd.New(0, 0),
					GrossValue:                  apd.New(0, 0),
					FeeAmount:                   apd.New(0, 0),
					BasisAfterRow:               *apd.New(0, 0),
					CalculationCurrency:         "USD",
					QuantityAfterRow:            *apd.New(3, 0),
					HoldingReductionExplanation: "custody transfer",
				}},
			},
			{
				AssetIdentityKey:    "asset-historical",
				DisplayLabel:        "HIST",
				OpeningQuantity:     *apd.New(4, 0),
				OpeningCostBasis:    *apd.New(20, 0),
				ClosingQuantity:     *apd.New(4, 0),
				ClosingCostBasis:    *apd.New(20, 0),
				CalculationCurrency: "USD",
			},
			{
				AssetIdentityKey:    "asset-converted",
				DisplayLabel:        "CONV",
				OpeningQuantity:     *apd.New(1, 0),
				OpeningCostBasis:    *apd.New(10, 0),
				ClosingQuantity:     *apd.New(0, 0),
				ClosingCostBasis:    *apd.New(0, 0),
				CalculationCurrency: "USD",
				ActivityRows: []reportmodel.AssetActivityRow{{
					SourceID:            "converted-sell",
					OccurredAt:          time.Date(2024, time.January, 2, 10, 0, 0, 0, time.UTC),
					ActivityType:        reportmodel.ActivityTypeSell,
					Quantity:            *apd.New(1, 0),
					UnitPrice:           apd.New(10, 0),
					GrossValue:          apd.New(10, 0),
					FeeAmount:           apd.New(0, 0),
					ActivityCurrency:    "EUR",
					BasisAfterRow:       *apd.New(0, 0),
					CalculationCurrency: "USD",
					QuantityAfterRow:    *apd.New(0, 0),
					ConversionStatus:    reportmodel.ConversionStatusConverted,
				}},
			},
		},
	)
	if reportErr != nil {
		t.Fatalf("new capital gains report: %v", reportErr)
	}

	return report
}

// pdfAnnexReportFixture creates one report with detailed Annex 1 evidence for
// PDF text-emission tests.
// Authored by: OpenCode
func pdfAnnexReportFixture(t *testing.T) reportmodel.CapitalGainsReport {
	t.Helper()

	var report = minimalPDFReportFixture(t)
	var conversion = pdfAnnexConversionEntry()
	var err error
	report.AuditAnnex, err = reportmodel.NewDetailedAuditAnnex([]reportmodel.PerAssetAuditSection{{
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Entries: []reportmodel.AuditActivityEntry{{
			SourceID:               "pdf-annex-sell",
			OccurredAt:             time.Date(2024, time.January, 2, 10, 0, 0, 0, time.UTC),
			ActivityType:           reportmodel.ActivityTypeSell,
			Quantity:               *apd.New(1, 0),
			UnitPrice:              apd.New(20, 0),
			GrossValue:             apd.New(20, 0),
			FeeAmount:              apd.New(1, 0),
			ActivityCurrency:       "EUR",
			CalculationCurrency:    "USD",
			QuantityAfterActivity:  *apd.New(0, 0),
			BasisAfterActivity:     *apd.New(0, 0),
			FullLiquidationEvent:   true,
			AllocatedBasis:         apd.New(10, 0),
			NetLiquidationProceeds: apd.New(19, 0),
			GainOrLoss:             apd.New(9, 0),
			ConversionStatus:       reportmodel.ConversionStatusConverted,
			Note:                   "pdf annex note",
		}},
	}}, []reportmodel.ConversionAuditEntry{conversion})
	if err != nil {
		t.Fatalf("new detailed annex: %v", err)
	}
	report.ConversionAuditEntries = []reportmodel.ConversionAuditEntry{conversion}
	report.RateSources = []reportmodel.ExchangeRateEvidence{*conversion.Amounts[0].ExchangeRateEvidence}
	return report
}

// pdfAnnexConversionEntry creates one valid conversion audit entry for PDF tests.
// Authored by: OpenCode
func pdfAnnexConversionEntry() reportmodel.ConversionAuditEntry {
	var activityDate = time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC)
	var evidence = reportmodel.ExchangeRateEvidence{
		SourceCurrency:   "EUR",
		BaseCurrency:     reportmodel.ReportBaseCurrencyUSD,
		ActivityDate:     activityDate,
		RateDate:         activityDate,
		Authority:        reportmodel.RateAuthorityFederalReserve,
		ProviderID:       reportmodel.RateProviderIDFederalReserveH10,
		RateKind:         "daily noon buying rate",
		QuoteDirection:   reportmodel.QuoteDirectionBasePerSource,
		RateValue:        *apd.New(12, -1),
		DatasetReference: "H10 fixture",
	}
	var amount = reportmodel.ConvertedActivityAmount{
		SourceID:             "pdf-annex-sell",
		AmountKind:           reportmodel.ConvertedAmountKindGrossValue,
		OriginalCurrency:     "EUR",
		OriginalAmount:       *apd.New(20, 0),
		ReportBaseCurrency:   reportmodel.ReportBaseCurrencyUSD,
		ConvertedAmount:      *apd.New(24, 0),
		ExchangeRateEvidence: &evidence,
		ConversionStatus:     reportmodel.ConversionStatusConverted,
	}
	return reportmodel.ConversionAuditEntry{
		SourceID:           "pdf-annex-sell",
		AssetLabel:         "BTC",
		ActivityDate:       activityDate,
		SourceCurrency:     "EUR",
		ReportBaseCurrency: reportmodel.ReportBaseCurrencyUSD,
		RateDate:           activityDate,
		RateAuthority:      reportmodel.RateAuthorityFederalReserve,
		RateKind:           "daily noon buying rate",
		RateValue:          *apd.New(12, -1),
		QuoteDirection:     reportmodel.QuoteDirectionBasePerSource,
		Amounts:            []reportmodel.ConvertedActivityAmount{amount},
	}
}

// Package pdf tests the private seams required for local A4 PDF rendering.
// Authored by: OpenCode
package pdf

import (
	"errors"
	"strings"
	"testing"
	"time"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/cockroachdb/apd/v3"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

// startedTestDocument creates one concrete document with valid fonts loaded.
// Authored by: OpenCode
func startedTestDocument(t *testing.T) *gopdfDocument {
	t.Helper()
	var document = newGopdfDocument()
	if err := document.StartPDF(PageSizeA4); err != nil {
		t.Fatalf("start PDF document: %v", err)
	}
	if err := document.AddTTFFont(fontRegular, goregular.TTF); err != nil {
		t.Fatalf("load regular font: %v", err)
	}
	if err := document.AddTTFFont(fontBold, gobold.TTF); err != nil {
		t.Fatalf("load bold font: %v", err)
	}
	return document
}

// layoutRecorder records structured PDF layout operations.
// Authored by: OpenCode
type layoutRecorder struct {
	titles      []string
	sections    []string
	subsections []string
	keyValues   map[string]string
	paragraphs  []string
	tables      []pdfTable
	operations  []pdfLayoutOperation
}

// AddTitle records one title emitted by a renderer helper. Authored by: OpenCode
func (recorder *layoutRecorder) AddTitle(text string) error {
	recorder.operations = append(recorder.operations, pdfLayoutOperation{kind: "title", text: text})
	recorder.titles = append(recorder.titles, text)
	return nil
}

// AddSectionHeading records one section heading. Authored by: OpenCode
func (recorder *layoutRecorder) AddSectionHeading(text string) error {
	recorder.operations = append(recorder.operations, pdfLayoutOperation{kind: "section-heading", text: text})
	recorder.sections = append(recorder.sections, text)
	return nil
}

// AddSubsectionHeading records one subsection heading. Authored by: OpenCode
func (recorder *layoutRecorder) AddSubsectionHeading(text string) error {
	recorder.operations = append(recorder.operations, pdfLayoutOperation{kind: "subsection-heading", text: text})
	recorder.subsections = append(recorder.subsections, text)
	return nil
}

// AddKeyValue records one label/value presentation fact. Authored by: OpenCode
func (recorder *layoutRecorder) AddKeyValue(label string, value string) error {
	recorder.operations = append(recorder.operations, pdfLayoutOperation{kind: "key-value", label: label, text: value})
	if recorder.keyValues == nil {
		recorder.keyValues = make(map[string]string)
	}
	recorder.keyValues[label] = value
	return nil
}

// AddParagraph records one paragraph emitted by a renderer helper. Authored by: OpenCode
func (recorder *layoutRecorder) AddParagraph(text string) error {
	recorder.operations = append(recorder.operations, pdfLayoutOperation{kind: "paragraph", text: text})
	recorder.paragraphs = append(recorder.paragraphs, text)
	return nil
}

// AddBoldParagraph records one fully bold wrapped paragraph operation. Authored by: OpenCode
func (recorder *layoutRecorder) AddBoldParagraph(text string) error {
	recorder.operations = append(recorder.operations, pdfLayoutOperation{kind: "bold-wrapped-paragraph", text: text, fullyBold: true, wrapped: true})
	recorder.paragraphs = append(recorder.paragraphs, text)
	return nil
}

// AddTable records one structured table emitted by a renderer helper. Authored by: OpenCode
func (recorder *layoutRecorder) AddTable(table pdfTable) error {
	recorder.operations = append(recorder.operations, pdfLayoutOperation{kind: "table", text: table.Title})
	recorder.tables = append(recorder.tables, table)
	return nil
}

// pdfLayoutOperation records one ordered renderer operation and its semantic
// text or key/value payload.
// Authored by: OpenCode
type pdfLayoutOperation struct {
	kind      string
	label     string
	text      string
	fullyBold bool
	wrapped   bool
}

// allText flattens recorded content for presentation assertions. Authored by: OpenCode
func (recorder *layoutRecorder) allText() []string {
	var texts []string
	texts = append(texts, recorder.titles...)
	texts = append(texts, recorder.sections...)
	texts = append(texts, recorder.subsections...)
	texts = append(texts, recorder.paragraphs...)
	for key, value := range recorder.keyValues {
		texts = append(texts, key, value)
	}
	for _, table := range recorder.tables {
		for _, column := range table.Columns {
			texts = append(texts, column.Header)
		}
		for _, row := range table.Rows {
			texts = append(texts, row...)
		}
	}
	return texts
}

// failingLayoutDocument returns configured failures through the layout seam.
// Authored by: OpenCode
type failingLayoutDocument struct {
	startErr     error
	fontErr      error
	titleErr     error
	pageBreakErr error
	tableErr     error
	bytesPayload []byte
	bytesErr     error
	bytesCalls   int
}

func (document *failingLayoutDocument) StartPDF(string) error             { return document.startErr }
func (document *failingLayoutDocument) AddTTFFont(string, []byte) error   { return document.fontErr }
func (document *failingLayoutDocument) AddTitle(string) error             { return document.titleErr }
func (document *failingLayoutDocument) AddSectionHeading(string) error    { return nil }
func (document *failingLayoutDocument) AddSubsectionHeading(string) error { return nil }
func (document *failingLayoutDocument) AddKeyValue(string, string) error  { return nil }
func (document *failingLayoutDocument) AddParagraph(string) error         { return nil }

// AddBoldParagraph accepts the current legal-warning operation while this test double injects failures at other layout stages. Authored by: OpenCode
func (document *failingLayoutDocument) AddBoldParagraph(string) error { return nil }

// AddTable returns an injected table failure for renderer finalization tests.
// Authored by: OpenCode
func (document *failingLayoutDocument) AddTable(pdfTable) error  { return document.tableErr }
func (document *failingLayoutDocument) AddAnnexPageBreak() error { return document.pageBreakErr }

// Bytes returns a defensive copy of the configured payload and its injected
// finalization error while counting calls made by the renderer.
// Authored by: OpenCode
func (document *failingLayoutDocument) Bytes() ([]byte, error) {
	document.bytesCalls++
	return append([]byte(nil), document.bytesPayload...), document.bytesErr
}

// secondTitleFailDocument fails only when Render starts the Annex title.
// Authored by: OpenCode
type secondTitleFailDocument struct {
	titleCalls int
}

func (document *secondTitleFailDocument) StartPDF(string) error           { return nil }
func (document *secondTitleFailDocument) AddTTFFont(string, []byte) error { return nil }
func (document *secondTitleFailDocument) AddTitle(string) error {
	document.titleCalls++
	if document.titleCalls == 2 {
		return errors.New("annex title failed")
	}
	return nil
}
func (document *secondTitleFailDocument) AddSectionHeading(string) error    { return nil }
func (document *secondTitleFailDocument) AddSubsectionHeading(string) error { return nil }
func (document *secondTitleFailDocument) AddKeyValue(string, string) error  { return nil }
func (document *secondTitleFailDocument) AddParagraph(string) error         { return nil }

// AddBoldParagraph accepts the current legal-warning operation while this test double injects an Annex-title failure. Authored by: OpenCode
func (document *secondTitleFailDocument) AddBoldParagraph(string) error { return nil }
func (document *secondTitleFailDocument) AddTable(pdfTable) error       { return nil }
func (document *secondTitleFailDocument) AddAnnexPageBreak() error      { return nil }

// Bytes returns an empty successful payload because this double exercises the
// renderer's Annex-title failure path before finalization.
// Authored by: OpenCode
func (document *secondTitleFailDocument) Bytes() ([]byte, error) { return nil, nil }

// errorLayoutRecorder injects layout errors for direct renderer helper tests.
// Authored by: OpenCode
type errorLayoutRecorder struct {
	layoutRecorder
	failTitle         string
	failSection       string
	failSubsection    string
	failKey           string
	failParagraph     bool
	failBoldParagraph bool
	failTable         string
}

func (recorder *errorLayoutRecorder) AddTitle(text string) error {
	if recorder.failTitle == text {
		return errors.New("title failed")
	}
	return recorder.layoutRecorder.AddTitle(text)
}

func (recorder *errorLayoutRecorder) AddSectionHeading(text string) error {
	if recorder.failSection == text {
		return errors.New("section failed")
	}
	return recorder.layoutRecorder.AddSectionHeading(text)
}

func (recorder *errorLayoutRecorder) AddSubsectionHeading(text string) error {
	if recorder.failSubsection == text {
		return errors.New("subsection failed")
	}
	return recorder.layoutRecorder.AddSubsectionHeading(text)
}

func (recorder *errorLayoutRecorder) AddKeyValue(label string, value string) error {
	if recorder.failKey == label {
		return errors.New("key failed")
	}
	return recorder.layoutRecorder.AddKeyValue(label, value)
}

func (recorder *errorLayoutRecorder) AddParagraph(text string) error {
	if recorder.failParagraph {
		return errors.New("paragraph failed")
	}
	return recorder.layoutRecorder.AddParagraph(text)
}

// AddBoldParagraph injects the dedicated warning-operation failure. Authored by: OpenCode
func (recorder *errorLayoutRecorder) AddBoldParagraph(text string) error {
	if recorder.failBoldParagraph && text == testutil.ReportPresentationLegalWarningText {
		return errors.New("bold warning failed")
	}
	return recorder.layoutRecorder.AddBoldParagraph(text)
}

func (recorder *errorLayoutRecorder) AddTable(table pdfTable) error {
	if recorder.failTable != "" && recorder.failTable == table.Title {
		return errors.New("table failed")
	}
	return recorder.layoutRecorder.AddTable(table)
}

// pdfStartRecorder records the page-size intent passed through the start seam.
// Authored by: OpenCode
type pdfStartRecorder struct {
	pageSize   string
	startCount int
}

func (recorder *pdfStartRecorder) StartPDF(pageSize string) error {
	recorder.pageSize = pageSize
	recorder.startCount++
	return nil
}

// failingPDFStartRecorder returns a deterministic start failure.
// Authored by: OpenCode
type failingPDFStartRecorder struct{}

func (recorder *failingPDFStartRecorder) StartPDF(string) error { return errors.New("start failed") }

// fontLoadRecorder records application-supplied font loads.
// Authored by: OpenCode
type fontLoadRecorder struct {
	loaded map[string][]byte
}

func (recorder *fontLoadRecorder) AddTTFFont(name string, data []byte) error {
	if recorder.loaded == nil {
		recorder.loaded = make(map[string][]byte)
	}
	recorder.loaded[name] = append([]byte(nil), data...)
	return nil
}

// failingFontLoader returns a deterministic failure for one font name.
// Authored by: OpenCode
type failingFontLoader struct {
	failName string
}

func (loader *failingFontLoader) AddTTFFont(name string, _ []byte) error {
	if name == loader.failName {
		return errors.New("font failed")
	}
	return nil
}

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

func assertContains(t *testing.T, texts []string, want string) {
	t.Helper()
	for _, text := range texts {
		if strings.Contains(text, want) {
			return
		}
	}
	t.Fatalf("required text %q was not found in %q", want, texts)
}

func assertKeyValue(t *testing.T, recorder *layoutRecorder, key string, want string) {
	t.Helper()
	if recorder.keyValues[key] != want {
		t.Fatalf("key %q = %q, want %q", key, recorder.keyValues[key], want)
	}
}

// findLayoutOperation returns the first operation matching the supplied semantic
// selectors, or -1 when the operation is absent.
// Authored by: OpenCode
func findLayoutOperation(operations []pdfLayoutOperation, kind string, label string, text string) int {
	for index, operation := range operations {
		if operation.kind != kind || operation.label != label {
			continue
		}
		if text != "" && operation.text != text {
			continue
		}
		return index
	}
	return -1
}

// assertKeyValueOperation verifies one exact key/value operation without using
// the recorder's map, which preserves repeated position fields semantically.
// Authored by: OpenCode
func assertKeyValueOperation(t *testing.T, recorder *layoutRecorder, label string, want string) {
	t.Helper()
	for _, operation := range recorder.operations {
		if operation.kind == "key-value" && operation.label == label && operation.text == want {
			return
		}
	}
	t.Errorf("key/value operation %q = %q was not found in %#v", label, want, recorder.operations)
}

// assertTableCellAt verifies one exact semantic cell in a named table row.
// Authored by: OpenCode
func assertTableCellAt(t *testing.T, recorder *layoutRecorder, tableTitle string, sourceCellIndex int, source string, cellIndex int, want string) {
	t.Helper()
	for _, table := range recorder.tables {
		if table.Title != tableTitle {
			continue
		}
		for _, row := range table.Rows {
			if sourceCellIndex >= len(row) || row[sourceCellIndex] != source {
				continue
			}
			if cellIndex >= len(row) {
				t.Errorf("table %q source %q has no cell %d in %#v", tableTitle, source, cellIndex, row)
				return
			}
			if row[cellIndex] != want {
				t.Errorf("table %q source %q cell %d = %q, want %q", tableTitle, source, cellIndex, row[cellIndex], want)
			}
			return
		}
		t.Errorf("table %q source %q was not found in %#v", tableTitle, source, table.Rows)
		return
	}
	t.Errorf("table %q was not found in %#v", tableTitle, recorder.tables)
}

func assertTableHeader(t *testing.T, recorder *layoutRecorder, want string) {
	t.Helper()
	for _, table := range recorder.tables {
		for _, column := range table.Columns {
			if strings.Contains(column.Header, want) {
				return
			}
		}
	}
	t.Fatalf("table header %q was not found in %#v", want, recorder.tables)
}

func assertTableCell(t *testing.T, recorder *layoutRecorder, want string) {
	t.Helper()
	for _, table := range recorder.tables {
		for _, row := range table.Rows {
			for _, cell := range row {
				if strings.Contains(cell, want) {
					return
				}
			}
		}
	}
	t.Fatalf("table cell %q was not found in %#v", want, recorder.tables)
}

func assertNoSubsection(t *testing.T, recorder *layoutRecorder, forbidden string) {
	t.Helper()
	for _, text := range recorder.subsections {
		if strings.Contains(text, forbidden) {
			t.Fatalf("forbidden subsection %q was found in %q", forbidden, recorder.subsections)
		}
	}
}

func assertSummaryTotalInsideTable(t *testing.T, recorder *layoutRecorder) {
	t.Helper()
	for _, table := range recorder.tables {
		if table.Title != "Gains-And-Losses Summary Table" {
			continue
		}
		if !table.StyledLastRow {
			t.Fatalf("summary table must style the total row")
		}
		if len(table.Rows) == 0 {
			t.Fatalf("summary table has no rows")
		}
		var lastRow = table.Rows[len(table.Rows)-1]
		if len(lastRow) == 0 || lastRow[0] != "Overall Yearly Net Total" {
			t.Fatalf("summary final row = %#v, want Overall Yearly Net Total", lastRow)
		}
		return
	}
	t.Fatalf("Gains-And-Losses Summary Table was not rendered")
}

func assertTablesWithinPrintableWidth(t *testing.T, recorder *layoutRecorder) {
	t.Helper()
	for _, table := range recorder.tables {
		var width float64
		for _, column := range table.Columns {
			width += column.Width
		}
		if width > contentWide {
			t.Fatalf("table %q width %.0f exceeds printable width %.0f", table.Title, width, contentWide)
		}
	}
}

func assertNoMarkdownStructuralSyntax(t *testing.T, texts []string) {
	t.Helper()
	for _, text := range texts {
		var trimmed = strings.TrimSpace(text)
		if strings.HasPrefix(trimmed, "#") || strings.Contains(trimmed, "**") || strings.Contains(trimmed, "|------") || strings.Contains(trimmed, "| ") || strings.Contains(trimmed, "---") {
			t.Fatalf("PDF text contains Markdown structural syntax: %q", text)
		}
	}
}

func assertErrorContains(t *testing.T, call func() error, want string) {
	t.Helper()
	var err = call()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want containing %q", err, want)
	}
}

func nonFiniteDecimal() apd.Decimal {
	return apd.Decimal{Form: apd.NaN}
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

// minimalPDFReportFixture creates a validated report containing only required fields.
// Authored by: OpenCode
func minimalPDFReportFixture(t *testing.T) reportmodel.CapitalGainsReport {
	t.Helper()
	var requestedAt = time.Date(2026, time.July, 5, 9, 0, 0, 0, time.UTC)
	var request, requestErr = reportmodel.NewReportRequest(2024, reportmodel.CostBasisMethodFIFO, reportmodel.ReportBaseCurrencyUSD, reportmodel.ReportOutputFormatPDF, requestedAt)
	if requestErr != nil {
		t.Fatalf("new report request: %v", requestErr)
	}
	var report, reportErr = reportmodel.NewCapitalGainsReport(request, requestedAt, reportmodel.ReportBaseCurrencyUSD.Label(), nil, *apd.New(0, 0), nil, nil)
	if reportErr != nil {
		t.Fatalf("new capital gains report: %v", reportErr)
	}
	return report
}

// pdfPresentationReportFixture creates a report fixture for main report rules.
// Authored by: OpenCode
func pdfPresentationReportFixture(t *testing.T) reportmodel.CapitalGainsReport {
	t.Helper()
	var requestedAt = time.Date(2026, time.July, 5, 9, 0, 0, 0, time.UTC)
	var request, requestErr = reportmodel.NewReportRequest(2024, reportmodel.CostBasisMethodFIFO, reportmodel.ReportBaseCurrencyUSD, reportmodel.ReportOutputFormatPDF, requestedAt)
	if requestErr != nil {
		t.Fatalf("new report request: %v", requestErr)
	}
	var report, reportErr = reportmodel.NewCapitalGainsReport(
		request,
		requestedAt,
		reportmodel.ReportBaseCurrencyUSD.Label(),
		[]reportmodel.AssetSummaryEntry{{AssetIdentityKey: "asset-zero", DisplayLabel: "ZERO", NetGainOrLoss: *apd.New(0, 0), ReportCalculationCurrency: "USD"}},
		*apd.New(0, 0),
		[]reportmodel.ReferenceLiquidationEntry{{AssetIdentityKey: "asset-zero", DisplayLabel: "ZERO", FullLiquidationCountThroughYearEnd: 1, MainSectionStatus: reportmodel.ReferenceSectionStatusIncludedInMainSections}},
		[]reportmodel.AssetDetailSection{
			{AssetIdentityKey: "asset-zero", DisplayLabel: "ZERO", OpeningQuantity: *apd.New(4, 0), OpeningCostBasis: *apd.New(0, 0), ClosingQuantity: *apd.New(3, 0), ClosingCostBasis: *apd.New(0, 0), CalculationCurrency: "USD", ActivityRows: []reportmodel.AssetActivityRow{{SourceID: "zero-sell", OccurredAt: time.Date(2024, time.January, 1, 10, 0, 0, 0, time.UTC), ActivityType: reportmodel.ActivityTypeSell, Quantity: *apd.New(1, 0), UnitPrice: apd.New(0, 0), GrossValue: apd.New(0, 0), FeeAmount: apd.New(0, 0), BasisAfterRow: *apd.New(0, 0), CalculationCurrency: "USD", QuantityAfterRow: *apd.New(3, 0), HoldingReductionExplanation: "custody transfer"}}},
			{AssetIdentityKey: "asset-historical", DisplayLabel: "HIST", OpeningQuantity: *apd.New(4, 0), OpeningCostBasis: *apd.New(20, 0), ClosingQuantity: *apd.New(4, 0), ClosingCostBasis: *apd.New(20, 0), CalculationCurrency: "USD"},
			{AssetIdentityKey: "asset-converted", DisplayLabel: "CONV", OpeningQuantity: *apd.New(1, 0), OpeningCostBasis: *apd.New(10, 0), ClosingQuantity: *apd.New(0, 0), ClosingCostBasis: *apd.New(0, 0), CalculationCurrency: "USD", ActivityRows: []reportmodel.AssetActivityRow{{SourceID: "converted-sell", OccurredAt: time.Date(2024, time.January, 2, 10, 0, 0, 0, time.UTC), ActivityType: reportmodel.ActivityTypeSell, Quantity: *apd.New(1, 0), UnitPrice: apd.New(10, 0), GrossValue: apd.New(10, 0), FeeAmount: apd.New(0, 0), ActivityCurrency: "EUR", BasisAfterRow: *apd.New(0, 0), CalculationCurrency: "USD", QuantityAfterRow: *apd.New(0, 0), ConversionStatus: reportmodel.ConversionStatusConverted}}},
		},
	)
	if reportErr != nil {
		t.Fatalf("new capital gains report: %v", reportErr)
	}
	return report
}

// pdfNonZeroLiquidationReportFixture creates a report with summary and
// liquidation rows for table-layout branch tests.
// Authored by: OpenCode
func pdfNonZeroLiquidationReportFixture(t *testing.T) reportmodel.CapitalGainsReport {
	t.Helper()
	var requestedAt = time.Date(2026, time.July, 5, 9, 0, 0, 0, time.UTC)
	var request, requestErr = reportmodel.NewReportRequest(2024, reportmodel.CostBasisMethodFIFO, reportmodel.ReportBaseCurrencyUSD, reportmodel.ReportOutputFormatPDF, requestedAt)
	if requestErr != nil {
		t.Fatalf("new report request: %v", requestErr)
	}
	var report, reportErr = reportmodel.NewCapitalGainsReport(
		request,
		requestedAt,
		reportmodel.ReportBaseCurrencyUSD.Label(),
		[]reportmodel.AssetSummaryEntry{{AssetIdentityKey: "asset-gain", DisplayLabel: "GAIN", NetGainOrLoss: *apd.New(5, 0), ReportCalculationCurrency: "USD"}},
		*apd.New(5, 0),
		[]reportmodel.ReferenceLiquidationEntry{{AssetIdentityKey: "asset-gain", DisplayLabel: "GAIN", FullLiquidationCountThroughYearEnd: 1, MainSectionStatus: reportmodel.ReferenceSectionStatusIncludedInMainSections}},
		[]reportmodel.AssetDetailSection{{
			AssetIdentityKey:    "asset-gain",
			DisplayLabel:        "GAIN",
			OpeningQuantity:     *apd.New(1, 0),
			OpeningCostBasis:    *apd.New(2, 0),
			ClosingQuantity:     *apd.New(0, 0),
			ClosingCostBasis:    *apd.New(0, 0),
			CalculationCurrency: "USD",
			ActivityRows: []reportmodel.AssetActivityRow{{
				SourceID:            "gain-sell",
				OccurredAt:          time.Date(2024, time.January, 2, 10, 0, 0, 0, time.UTC),
				ActivityType:        reportmodel.ActivityTypeSell,
				Quantity:            *apd.New(1, 0),
				UnitPrice:           apd.New(7, 0),
				GrossValue:          apd.New(7, 0),
				FeeAmount:           apd.New(0, 0),
				ActivityCurrency:    "USD",
				BasisAfterRow:       *apd.New(0, 0),
				CalculationCurrency: "USD",
				QuantityAfterRow:    *apd.New(0, 0),
				ConversionStatus:    reportmodel.ConversionStatusSameCurrency,
			}},
			LiquidationSummaries: []reportmodel.LiquidationCalculation{{
				SourceID:               "gain-sell",
				OccurredAt:             time.Date(2024, time.January, 2, 10, 0, 0, 0, time.UTC),
				DisposedQuantity:       *apd.New(1, 0),
				AllocatedBasis:         *apd.New(2, 0),
				NetLiquidationProceeds: *apd.New(7, 0),
				GainOrLoss:             *apd.New(5, 0),
				ActivityCurrency:       "USD",
				CalculationCurrency:    "USD",
			}},
		}},
	)
	if reportErr != nil {
		t.Fatalf("new capital gains report: %v", reportErr)
	}
	return report
}

// pdfAnnexReportFixture creates one report with detailed Annex 1 evidence.
// Authored by: OpenCode
func pdfAnnexReportFixture(t *testing.T) reportmodel.CapitalGainsReport {
	t.Helper()
	var report = minimalPDFReportFixture(t)
	var conversion = pdfAnnexConversionEntry()
	var err error
	report.AuditAnnex, err = reportmodel.NewDetailedAuditAnnex([]reportmodel.PerAssetAuditSection{{AssetIdentityKey: "asset-btc", DisplayLabel: "BTC", Entries: []reportmodel.AuditActivityEntry{{SourceID: "pdf-annex-sell", OccurredAt: time.Date(2024, time.January, 2, 10, 0, 0, 0, time.UTC), ActivityType: reportmodel.ActivityTypeSell, Quantity: *apd.New(1, 0), UnitPrice: apd.New(20, 0), GrossValue: apd.New(20, 0), FeeAmount: apd.New(1, 0), ActivityCurrency: "EUR", CalculationCurrency: "USD", QuantityAfterActivity: *apd.New(0, 0), BasisAfterActivity: *apd.New(0, 0), FullLiquidationEvent: true, AllocatedBasis: apd.New(10, 0), NetLiquidationProceeds: apd.New(19, 0), GainOrLoss: apd.New(9, 0), ConversionStatus: reportmodel.ConversionStatusConverted, Note: "pdf annex note"}}}}, []reportmodel.ConversionAuditEntry{conversion})
	if err != nil {
		t.Fatalf("new detailed annex: %v", err)
	}
	report.AuditAnnex.ConversionAuditEntries = []reportmodel.ConversionAuditEntry{conversion}
	report.RateSources = []reportmodel.ExchangeRateEvidence{*conversion.Amounts[0].ExchangeRateEvidence}
	return report
}

// pdfAnnexConversionEntry creates one valid conversion audit entry for PDF tests.
// Authored by: OpenCode
func pdfAnnexConversionEntry() reportmodel.ConversionAuditEntry {
	var activityDate = time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC)
	var evidence = reportmodel.ExchangeRateEvidence{SourceCurrency: "EUR", BaseCurrency: reportmodel.ReportBaseCurrencyUSD, ActivityDate: activityDate, RateDate: activityDate, Authority: reportmodel.RateAuthorityFederalReserve, ProviderID: reportmodel.RateProviderIDFederalReserveH10, RateKind: "daily noon buying rate", QuoteDirection: reportmodel.QuoteDirectionBasePerSource, RateValue: *apd.New(12, -1), DatasetReference: "H10 fixture"}
	var amount = reportmodel.ConvertedActivityAmount{SourceID: "pdf-annex-sell", AmountKind: reportmodel.ConvertedAmountKindGrossValue, OriginalCurrency: "EUR", OriginalAmount: *apd.New(20, 0), ReportBaseCurrency: reportmodel.ReportBaseCurrencyUSD, ConvertedAmount: *apd.New(24, 0), ExchangeRateEvidence: &evidence, ConversionStatus: reportmodel.ConversionStatusConverted}
	return reportmodel.ConversionAuditEntry{SourceID: "pdf-annex-sell", AssetLabel: "BTC", ActivityDate: activityDate, SourceCurrency: "EUR", ReportBaseCurrency: reportmodel.ReportBaseCurrencyUSD, RateDate: activityDate, RateAuthority: reportmodel.RateAuthorityFederalReserve, RateKind: "daily noon buying rate", RateValue: *apd.New(12, -1), QuoteDirection: reportmodel.QuoteDirectionBasePerSource, Amounts: []reportmodel.ConvertedActivityAmount{amount}}
}

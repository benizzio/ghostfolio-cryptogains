// Package pdf tests the private seams required for local A4 PDF rendering.
// Authored by: OpenCode
package pdf

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	reportcalculate "github.com/benizzio/ghostfolio-cryptogains/internal/report/calculate"
	reportmarkdown "github.com/benizzio/ghostfolio-cryptogains/internal/report/markdown"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/report/presentation"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/cockroachdb/apd/v3"
	"github.com/signintech/gopdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

// TestStartPDFDocumentUsesA4Configuration specifies the renderer's page-size
// seam so every generated PDF starts with A4 configuration.
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

// TestGopdfDocumentUsesLandscapeA4AndPrintableWidth verifies the concrete
// renderer uses landscape A4 dimensions and a printable area with right padding.
// Authored by: OpenCode
func TestGopdfDocumentUsesLandscapeA4AndPrintableWidth(t *testing.T) {
	var document = newGopdfDocument()
	var err = document.StartPDF(PageSizeA4)
	if err != nil {
		t.Fatalf("start PDF document: %v", err)
	}

	if document.pageWidth != gopdf.PageSizeA4Landscape.W || document.pageHeight != gopdf.PageSizeA4Landscape.H {
		t.Fatalf("page size = %.0fx%.0f, want landscape A4 %.0fx%.0f", document.pageWidth, document.pageHeight, gopdf.PageSizeA4Landscape.W, gopdf.PageSizeA4Landscape.H)
	}
	if contentWide != document.pageWidth-2*pageMargin {
		t.Fatalf("content width %.0f, want printable width %.0f", contentWide, document.pageWidth-2*pageMargin)
	}
	if pageBottom > document.pageHeight-pageMargin {
		t.Fatalf("page bottom %.0f exceeds landscape A4 printable height %.0f", pageBottom, document.pageHeight-pageMargin)
	}
}

// TestTableLayoutUsesPrintableWidthSpacingAndRowPreflight verifies that the concrete layout
// adapter uses balanced printable-width tables, 24-point block separation, and
// advances before a header-and-row chunk could cross the bottom margin.
// Authored by: OpenCode
func TestTableLayoutUsesPrintableWidthSpacingAndRowPreflight(t *testing.T) {
	var columns = printableWidthColumns([]pdfColumn{
		{Header: "Wide", Width: 3, Align: "left"},
		{Header: "Narrow", Width: 1, Align: "right"},
	})
	var width float64
	for _, column := range columns {
		width += column.Width
	}
	if width != contentWide {
		t.Fatalf("scaled table width = %.2f, want full printable width %.2f", width, contentWide)
	}
	var equalColumns = printableWidthColumns([]pdfColumn{
		{Header: "First", Align: "left"},
		{Header: "Second", Align: "right"},
	})
	if equalColumns[0].Width != contentWide/2 || equalColumns[1].Width != contentWide/2 {
		t.Fatalf("zero-width columns = %#v, want equal printable-width allocation", equalColumns)
	}
	if sectionSpacing < 24 || tableSpacing < 24 {
		t.Fatalf("section/table spacing = %.0f/%.0f, want at least 24 points", sectionSpacing, tableSpacing)
	}

	var document = startedTestDocument(t)
	if err := document.AddTitle("Title"); err != nil {
		t.Fatalf("add title: %v", err)
	}
	var titleEnd = document.y
	if err := document.AddSectionHeading("Section"); err != nil {
		t.Fatalf("add section: %v", err)
	}
	if document.y-titleEnd-18 < sectionSpacing {
		t.Fatalf("section top gap = %.0f, want at least %.0f", document.y-titleEnd-18, sectionSpacing)
	}
	var sectionEnd = document.y
	if err := document.AddSubsectionHeading("Subsection"); err != nil {
		t.Fatalf("add subsection: %v", err)
	}
	if document.y-sectionEnd-16 < sectionSpacing {
		t.Fatalf("subsection top gap = %.0f, want at least %.0f", document.y-sectionEnd-16, sectionSpacing)
	}

	var preflightDocument = startedTestDocument(t)
	preflightDocument.y = pageBottom - 47
	if capacity := preflightDocument.tableRowCapacity(24); capacity != 0 {
		t.Fatalf("row capacity = %d, want 0 when header and row would cross the bottom margin", capacity)
	}
	if err := preflightDocument.AddTable(pdfTable{
		ContinuationTitle: "Audit table (continued)",
		Columns:           []pdfColumn{{Header: "Entry", Width: 1, Align: "left"}},
		Rows:              [][]string{{"must start on the next page"}},
		RowHeight:         24,
	}); err != nil {
		t.Fatalf("add preflighted table: %v", err)
	}
	var payload, err = preflightDocument.Bytes()
	if err != nil {
		t.Fatalf("finalize preflighted PDF: %v", err)
	}
	var text = string(payload)
	if strings.Contains(text, "Audit table (continued)") || strings.Contains(text, "CONTINUED:") {
		t.Fatalf("table moved before its first row emitted continuation context: %q", text)
	}

	var tallRowDocument = startedTestDocument(t)
	if err := tallRowDocument.AddTable(pdfTable{
		ContinuationTitle: "Tall row (continued)",
		Columns:           []pdfColumn{{Header: "Entry", Width: 1, Align: "left"}},
		Rows:              [][]string{{"one"}, {"two"}},
		RowHeight:         220,
	}); err != nil {
		t.Fatalf("add tall preflighted table: %v", err)
	}
	if tallRowDocument.y > pageBottom {
		t.Fatalf("tall table ended at %.0f, beyond bottom margin %.0f", tallRowDocument.y, pageBottom)
	}

	var unrenderableContinuation = startedTestDocument(t)
	assertErrorContains(t, func() error {
		return unrenderableContinuation.AddTable(pdfTable{
			ContinuationTitle: "Too tall (continued)",
			Columns:           []pdfColumn{{Header: "Entry", Width: 1, Align: "left"}},
			Rows:              [][]string{{"one"}, {"two"}},
			RowHeight:         249,
		})
	}, "does not fit within the printable page area")
}

// TestContinuedTableRepeatsContextAndHeader verifies each continued
// page identifies the table and redraws its header before its next whole row.
// Authored by: OpenCode
func TestContinuedTableRepeatsContextAndHeader(t *testing.T) {
	var document = startedTestDocument(t)
	if err := document.AddTable(pdfTable{
		ContinuationTitle: "Per-Asset Audit Activity (continued)",
		Columns:           []pdfColumn{{Header: "Source ID", Width: 1, Align: "left"}},
		Rows:              [][]string{{"first"}, {"second"}, {"third"}},
		RowHeight:         200,
	}); err != nil {
		t.Fatalf("add continued table: %v", err)
	}

	var payload, err = document.Bytes()
	if err != nil {
		t.Fatalf("finalize continued PDF: %v", err)
	}
	if !bytes.HasPrefix(payload, []byte("%PDF-")) {
		t.Fatalf("expected valid PDF payload, got %q", payload)
	}
}

// TestTableContinuationAndWrappedCellLayout verifies long table cells retain
// their wrapped content inside printable columns and that only actual
// continuation pages repeat the required table context and header.
// Authored by: OpenCode
func TestTableContinuationAndWrappedCellLayout(t *testing.T) {
	var longCell = strings.Repeat("long table cell content ", 24) + "WRAPPED-CELL-END"
	var wrappedDocument = startedTestDocument(t)
	if err := wrappedDocument.AddTable(pdfTable{
		Title: "Wrapped Cell Table",
		Columns: []pdfColumn{
			{Header: "Source ID", Width: 1, Align: "left"},
			{Header: "Note", Width: 1, Align: "left"},
		},
		Rows: [][]string{{"wrapped-source", longCell}},
	}); err != nil {
		t.Fatalf("add wrapped-cell table: %v", err)
	}
	var wrappedPayload, err = wrappedDocument.Bytes()
	if err != nil {
		t.Fatalf("finalize wrapped-cell PDF: %v", err)
	}
	var wrappedInspection testutil.GeneratedPDF
	wrappedInspection, err = testutil.InspectGeneratedPDF(wrappedPayload)
	if err != nil {
		t.Fatalf("inspect wrapped-cell PDF: %v", err)
	}
	if !wrappedInspection.ContainsSearchableText("WRAPPED-CELL-END") {
		t.Fatalf("wrapped cell tail was clipped instead of wrapped within its column: %q", wrappedInspection.SearchableText)
	}
	if wrappedDocument.y <= 136 {
		t.Fatalf("wrapped row advanced to %.0f, want more than the 136-point unwrapped table height", wrappedDocument.y)
	}
	if wrappedInspection.ContainsSearchableText("Wrapped Cell Table (continued)") {
		t.Fatalf("unsplit table emitted continuation context: %q", wrappedInspection.SearchableText)
	}

	var previousTextWriter = writeTextForGopdfDocument
	var continuationContexts []string
	writeTextForGopdfDocument = func(document *gopdfDocument, text string) error {
		continuationContexts = append(continuationContexts, text)
		return previousTextWriter(document, text)
	}
	defer func() { writeTextForGopdfDocument = previousTextWriter }()

	var continuedDocument = startedTestDocument(t)
	if err := continuedDocument.AddTable(pdfTable{
		ContinuationTitle: "Per-Asset Audit Activity (continued)",
		Columns:           []pdfColumn{{Header: "Source ID", Width: 1, Align: "left"}},
		Rows:              [][]string{{"first-complete-row"}, {"second-complete-row"}, {"third-complete-row"}},
		RowHeight:         200,
	}); err != nil {
		t.Fatalf("add continued table: %v", err)
	}
	var continuedInspection testutil.GeneratedPDF
	var continuedPayload []byte
	continuedPayload, err = continuedDocument.Bytes()
	if err != nil {
		t.Fatalf("finalize continued-table PDF: %v", err)
	}
	continuedInspection, err = testutil.InspectGeneratedPDF(continuedPayload)
	if err != nil {
		t.Fatalf("inspect continued-table PDF: %v", err)
	}
	if len(continuedInspection.PageBoxes) != 3 {
		t.Fatalf("continued table pages = %d, want 3 pages for one complete row per page", len(continuedInspection.PageBoxes))
	}
	if len(continuationContexts) != 2 {
		t.Fatalf("continuation context count = %d, want 2", len(continuationContexts))
	}
	for _, context := range continuationContexts {
		if context != "Per-Asset Audit Activity (continued)" {
			t.Fatalf("continuation context = %q, want exact Per-Asset Audit Activity (continued)", context)
		}
	}
	if strings.Contains(continuedInspection.SearchableText, "Continued:") {
		t.Fatalf("forbidden continuation prefix was emitted: %q", continuedInspection.SearchableText)
	}
	var repeatedHeaders = strings.Count(strings.ToUpper(continuedInspection.SearchableText), "SOURCEID")
	if repeatedHeaders < 3 {
		t.Fatalf("repeated table headers = %d, want one header on each page", repeatedHeaders)
	}
	for _, row := range []string{"first-complete-row", "second-complete-row", "third-complete-row"} {
		if !continuedInspection.ContainsSearchableText(row) {
			t.Fatalf("complete continued row %q was not searchable in %q", row, continuedInspection.SearchableText)
		}
	}
	if continuedDocument.y > pageBottom {
		t.Fatalf("continued table ended at %.0f, beyond bottom margin %.0f", continuedDocument.y, pageBottom)
	}
}

// TestT028PDFTableMeasurementMatchesDrawingWrapAndExplicitNewlines verifies
// explicit cell boundaries and indicator-sensitive space wrapping use the same
// measured line count and height as the table drawing path.
// Authored by: OpenCode
func TestT028PDFTableMeasurementMatchesDrawingWrapAndExplicitNewlines(t *testing.T) {
	var document = startedTestDocument(t)
	var columns = printableWidthColumns([]pdfColumn{
		{Header: "Entry", Width: 1, Align: "left"},
		{Header: "Other", Width: 3, Align: "left"},
	})
	var width = columns[0].Width - 4
	var breakOption = &gopdf.BreakOption{Mode: gopdf.BreakModeIndicatorSensitive, BreakIndicator: ' '}
	var cases = []struct {
		name string
		text string
	}{
		{name: "explicit newline", text: "first logical line\nsecond logical line"},
		{name: "long spaces", text: strings.Repeat("long space wrapped content ", 24)},
	}

	if err := document.pdf.SetFont(fontRegular, "", 6.5); err != nil {
		t.Fatalf("set table font: %v", err)
	}
	for _, testCase := range cases {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			var splitLines, err = document.pdf.SplitTextWithOption(testCase.text, width, breakOption)
			if err != nil {
				t.Fatalf("split table text: %v", err)
			}
			if len(splitLines) < 2 {
				t.Fatalf("drawing split lines = %d, want at least 2 for %q", len(splitLines), testCase.text)
			}
			var expectedText = strings.Join(splitLines, "\n")
			var fits bool
			var expectedHeight float64
			fits, expectedHeight, err = document.pdf.IsFitMultiCellWithNewline(&gopdf.Rect{W: width, H: pageBottom - pageMargin}, expectedText)
			if err != nil {
				t.Fatalf("measure drawing lines: %v", err)
			}
			if !fits {
				t.Fatalf("drawing lines do not fit the test rectangle")
			}

			var measuredHeight float64
			var measured bool
			measuredHeight, measured, err = document.tableCellHeight(columns, 0, testCase.text)
			if err != nil {
				t.Fatalf("measure table cell: %v", err)
			}
			if !measured {
				t.Fatal("table cell was not measured")
			}
			if measuredHeight != expectedHeight+4 {
				t.Fatalf("measured row cell height = %.2f, want drawing height %.2f plus padding", measuredHeight, expectedHeight+4)
			}
		})
	}
}

// TestT028PDFTableDrawnLinesMatchMeasuredRowHeightAndBottomMargin verifies
// explicit PDF cell lines appear as distinct drawn runs and remain within the
// printable bottom margin after the row's measured height is applied.
// Authored by: OpenCode
func TestT028PDFTableDrawnLinesMatchMeasuredRowHeightAndBottomMargin(t *testing.T) {
	var document = startedTestDocument(t)
	if err := document.pdf.SetFont(fontRegular, "", 6.5); err != nil {
		t.Fatalf("set table font: %v", err)
	}
	var cell = "ROW-LINE-ONE\nROW-LINE-TWO\nROW-LINE-THREE"
	var table = pdfTable{
		Columns: []pdfColumn{{Header: "Entry", Width: 1, Align: "left"}},
		Rows:    [][]string{{cell}},
	}
	var columns = printableWidthColumns(table.Columns)
	var splitLines, err = document.pdf.SplitTextWithOption(cell, columns[0].Width-4, &gopdf.BreakOption{Mode: gopdf.BreakModeIndicatorSensitive, BreakIndicator: ' '})
	if err != nil {
		t.Fatalf("split table cell: %v", err)
	}
	if len(splitLines) != 3 {
		t.Fatalf("drawn line count = %d, want 3", len(splitLines))
	}
	var measuredRowHeight float64
	measuredRowHeight, err = document.tableRowHeight(columns, table.Rows, 24)
	if err != nil {
		t.Fatalf("measure table row: %v", err)
	}
	var expectedCellHeight float64
	_, expectedCellHeight, err = document.pdf.IsFitMultiCellWithNewline(&gopdf.Rect{W: columns[0].Width - 4, H: pageBottom - pageMargin}, strings.Join(splitLines, "\n"))
	if err != nil {
		t.Fatalf("measure expected table cell: %v", err)
	}
	var expectedRowHeight = expectedCellHeight + 4
	if expectedRowHeight < 24 {
		expectedRowHeight = 24
	}
	if measuredRowHeight != expectedRowHeight {
		t.Fatalf("measured row height = %.2f, want %.2f", measuredRowHeight, expectedRowHeight)
	}
	if err := document.AddTable(table); err != nil {
		t.Fatalf("draw multiline table: %v", err)
	}
	if document.y > pageBottom {
		t.Fatalf("multiline table ended at %.2f, beyond bottom margin %.2f", document.y, pageBottom)
	}
	var payload []byte
	payload, err = document.Bytes()
	if err != nil {
		t.Fatalf("finalize multiline table: %v", err)
	}
	var inspection testutil.GeneratedPDF
	inspection, err = testutil.InspectGeneratedPDF(payload)
	if err != nil {
		t.Fatalf("inspect multiline table: %v", err)
	}
	var drawnLines int
	for _, run := range inspection.TextRuns {
		if strings.Contains(run.Text, "ROW-LINE-") {
			drawnLines++
		}
	}
	if drawnLines != len(splitLines) {
		t.Fatalf("drawn line count = %d, want measured line count %d", drawnLines, len(splitLines))
	}
}

// TestT028PDFFreshPagePreflightMovesWholeRowWithoutContinuationLabel verifies
// a table-start row moves before drawing when only the remaining page is too
// small, without pretending that the table already continued.
// Authored by: OpenCode
func TestT028PDFFreshPagePreflightMovesWholeRowWithoutContinuationLabel(t *testing.T) {
	var document = startedTestDocument(t)
	document.y = pageBottom - 90
	var table = pdfTable{
		ContinuationTitle: "Fresh relocation (continued)",
		Columns:           []pdfColumn{{Header: "Entry", Width: 1, Align: "left"}},
		Rows:              [][]string{{"FRESH-ONE\nFRESH-TWO\nFRESH-THREE\nFRESH-FOUR"}},
	}
	if err := document.AddTable(table); err != nil {
		t.Fatalf("preflight fresh-page row: %v", err)
	}
	var payload, err = document.Bytes()
	if err != nil {
		t.Fatalf("finalize fresh-page table: %v", err)
	}
	var inspection testutil.GeneratedPDF
	inspection, err = testutil.InspectGeneratedPDF(payload)
	if err != nil {
		t.Fatalf("inspect fresh-page table: %v", err)
	}
	if len(inspection.PageBoxes) != 2 {
		t.Fatalf("fresh-page preflight pages = %d, want 2", len(inspection.PageBoxes))
	}
	if inspection.ContainsSearchableText(table.ContinuationTitle) {
		t.Fatalf("table-start relocation emitted continuation label: %q", inspection.SearchableText)
	}
	for _, rowLine := range []string{"FRESH-ONE", "FRESH-TWO", "FRESH-THREE", "FRESH-FOUR"} {
		if !inspection.ContainsSearchableText(rowLine) {
			t.Fatalf("whole row line %q was not drawn", rowLine)
		}
	}
	if document.y > pageBottom {
		t.Fatalf("fresh-page table ended at %.2f, beyond bottom margin %.2f", document.y, pageBottom)
	}
}

// TestT028PDFContinuationRepeatsContextAndHeaderAfterWholeRows verifies actual
// continuation pages repeat the context and header while keeping complete rows.
// Authored by: OpenCode
func TestT028PDFContinuationRepeatsContextAndHeaderAfterWholeRows(t *testing.T) {
	var previousTextWriter = writeTextForGopdfDocument
	var contexts []string
	writeTextForGopdfDocument = func(document *gopdfDocument, text string) error {
		contexts = append(contexts, text)
		return previousTextWriter(document, text)
	}
	defer func() { writeTextForGopdfDocument = previousTextWriter }()

	var document = startedTestDocument(t)
	var table = pdfTable{
		ContinuationTitle: "Whole-row table (continued)",
		Columns:           []pdfColumn{{Header: "Entry", Width: 1, Align: "left"}},
		Rows: [][]string{
			{"WHOLE-ROW-ONE\nONE-CONTINUATION"},
			{"WHOLE-ROW-TWO\nTWO-CONTINUATION"},
			{"WHOLE-ROW-THREE\nTHREE-CONTINUATION"},
		},
		RowHeight: 200,
	}
	if err := document.AddTable(table); err != nil {
		t.Fatalf("draw continued whole-row table: %v", err)
	}
	var payload, err = document.Bytes()
	if err != nil {
		t.Fatalf("finalize continued whole-row table: %v", err)
	}
	var inspection testutil.GeneratedPDF
	inspection, err = testutil.InspectGeneratedPDF(payload)
	if err != nil {
		t.Fatalf("inspect continued whole-row table: %v", err)
	}
	if len(inspection.PageBoxes) != 3 {
		t.Fatalf("continued whole-row pages = %d, want 3", len(inspection.PageBoxes))
	}
	if len(contexts) != 2 {
		t.Fatalf("continuation context count = %d, want 2", len(contexts))
	}
	for _, context := range contexts {
		if context != table.ContinuationTitle {
			t.Fatalf("continuation context = %q, want %q", context, table.ContinuationTitle)
		}
	}
	var headerCount int
	for _, run := range inspection.TextRuns {
		if strings.Contains(strings.ToUpper(run.Text), "ENTRY") {
			headerCount++
		}
	}
	if headerCount < len(inspection.PageBoxes) {
		t.Fatalf("header run count = %d, want at least %d", headerCount, len(inspection.PageBoxes))
	}
	for _, rowLine := range []string{"WHOLE-ROW-ONE", "ONE-CONTINUATION", "WHOLE-ROW-TWO", "TWO-CONTINUATION", "WHOLE-ROW-THREE", "THREE-CONTINUATION"} {
		if !inspection.ContainsSearchableText(rowLine) {
			t.Fatalf("continued row line %q was not searchable", rowLine)
		}
	}
}

// TestT028PDFOverheightNewlineRowFailsBeforeFinalization verifies an explicit
// newline row that cannot fit a fresh page fails before drawing or finalizing.
// Authored by: OpenCode
func TestT028PDFOverheightNewlineRowFailsBeforeFinalization(t *testing.T) {
	var document = startedTestDocument(t)
	var previousDrawer = drawTableForGopdfDocument
	var drawCalls int
	drawTableForGopdfDocument = func(table gopdf.TableLayout) error {
		drawCalls++
		return previousDrawer(table)
	}
	defer func() { drawTableForGopdfDocument = previousDrawer }()
	var overheight = strings.TrimSuffix(strings.Repeat("OVERHEIGHT\n", 100), "\n")
	var err = document.AddTable(pdfTable{
		Columns: []pdfColumn{{Header: "Entry", Width: 1, Align: "left"}},
		Rows:    [][]string{{overheight}},
	})
	if err == nil || !strings.Contains(err.Error(), "does not fit within the printable page area") {
		t.Errorf("overheight error = %v, want printable-area failure", err)
	}
	if drawCalls != 0 {
		t.Errorf("overheight draw calls = %d, want zero before finalization", drawCalls)
	}

	var previousDocument = newPDFDocumentForRenderer
	defer func() { newPDFDocumentForRenderer = previousDocument }()
	var finalizationDocument = &failingLayoutDocument{tableErr: errors.New("table row 1 does not fit within the printable page area")}
	newPDFDocumentForRenderer = func() pdfLayoutDocument { return finalizationDocument }
	var renderer, rendererErr = NewRenderer(RenderOptions{Fonts: FontData{Regular: []byte("regular"), Bold: []byte("bold")}})
	if rendererErr != nil {
		t.Fatalf("new renderer: %v", rendererErr)
	}
	var payload []byte
	payload, rendererErr = renderer.Render(pdfPresentationReportFixture(t))
	if rendererErr == nil || !strings.Contains(rendererErr.Error(), "does not fit within the printable page area") {
		t.Fatalf("renderer overheight error = %v, want printable-area failure", rendererErr)
	}
	if payload != nil {
		t.Fatalf("renderer payload = %q, want nil after overheight failure", payload)
	}
	if finalizationDocument.bytesCalls != 0 {
		t.Fatalf("finalization calls = %d, want zero after overheight failure", finalizationDocument.bytesCalls)
	}
}

// TestT028PDFTableMeasurementAndDrawingErrorsIncludeStageAndRow verifies table
// failures identify whether measurement or drawing failed and which row failed.
// Authored by: OpenCode
func TestT028PDFTableMeasurementAndDrawingErrorsIncludeStageAndRow(t *testing.T) {
	var previousMeasurer = measureTableCellForGopdfDocument
	var previousDrawer = drawTableForGopdfDocument
	defer func() {
		measureTableCellForGopdfDocument = previousMeasurer
		drawTableForGopdfDocument = previousDrawer
	}()

	measureTableCellForGopdfDocument = func(*gopdfDocument, *gopdf.Rect, string) (bool, float64, error) {
		return false, 0, errors.New("synthetic measurement failure")
	}
	var measurementErr = startedTestDocument(t).AddTable(pdfTable{
		Title:   "Measurement stage",
		Columns: []pdfColumn{{Header: "Entry", Width: 1, Align: "left"}},
		Rows:    [][]string{{"MEASURE-ROW"}},
	})
	if measurementErr == nil || !strings.Contains(measurementErr.Error(), "measurement") || !strings.Contains(measurementErr.Error(), "row") {
		t.Errorf("measurement error = %v, want measurement and row context", measurementErr)
	}

	measureTableCellForGopdfDocument = previousMeasurer
	drawTableForGopdfDocument = func(gopdf.TableLayout) error { return errors.New("synthetic drawing failure") }
	var drawingErr = startedTestDocument(t).AddTable(pdfTable{
		Title:   "Drawing stage",
		Columns: []pdfColumn{{Header: "Entry", Width: 1, Align: "left"}},
		Rows:    [][]string{{"DRAW-ROW"}},
	})
	if drawingErr == nil || !strings.Contains(drawingErr.Error(), "drawing") || !strings.Contains(drawingErr.Error(), "row") {
		t.Errorf("drawing error = %v, want drawing and row context", drawingErr)
	}
}

// TestT028PDFControlledNewlinesDoNotWeakenGenericSanitization verifies only
// renderer-controlled cell boundaries survive while arbitrary PDF text remains
// single-line and delimiter-safe.
// Authored by: OpenCode
func TestT028PDFControlledNewlinesDoNotWeakenGenericSanitization(t *testing.T) {
	var dynamic = "dynamic\nlabel|delimiter"
	if got := sanitizeText(dynamic); got != "dynamic label/delimiter" {
		t.Fatalf("generic sanitized text = %q, want single-line delimiter-safe text", got)
	}

	var genericDocument = startedTestDocument(t)
	if err := genericDocument.AddParagraph(dynamic); err != nil {
		t.Fatalf("draw generic text: %v", err)
	}
	var genericPayload, err = genericDocument.Bytes()
	if err != nil {
		t.Fatalf("finalize generic text: %v", err)
	}
	var genericInspection testutil.GeneratedPDF
	genericInspection, err = testutil.InspectGeneratedPDF(genericPayload)
	if err != nil {
		t.Fatalf("inspect generic text: %v", err)
	}
	for _, run := range genericInspection.TextRuns {
		if strings.Contains(run.Text, "dynamic") && strings.Contains(run.Text, "|") {
			t.Fatalf("generic text retained delimiter: %q", run.Text)
		}
	}

	var controlledDocument = startedTestDocument(t)
	var controlledCell = "CONTROLLED-ONE|delimiter\nCONTROLLED-TWO"
	if err := controlledDocument.AddTable(pdfTable{
		Columns: []pdfColumn{{Header: "Entry", Width: 1, Align: "left"}},
		Rows:    [][]string{{controlledCell}},
	}); err != nil {
		t.Fatalf("draw controlled cell: %v", err)
	}
	var controlledPayload []byte
	controlledPayload, err = controlledDocument.Bytes()
	if err != nil {
		t.Fatalf("finalize controlled cell: %v", err)
	}
	var controlledInspection testutil.GeneratedPDF
	controlledInspection, err = testutil.InspectGeneratedPDF(controlledPayload)
	if err != nil {
		t.Fatalf("inspect controlled cell: %v", err)
	}
	var controlledRuns int
	for _, run := range controlledInspection.TextRuns {
		if strings.Contains(run.Text, "CONTROLLED-") {
			controlledRuns++
			if strings.Contains(run.Text, "|") {
				t.Fatalf("controlled cell retained delimiter: %q", run.Text)
			}
		}
	}
	if controlledRuns != 2 {
		t.Fatalf("controlled cell line runs = %d, want 2", controlledRuns)
	}
}

// TestLoadApplicationFontsValidatesAndLoadsRegularAndBoldFonts specifies the
// application-supplied font seam.
// Authored by: OpenCode
func TestLoadApplicationFontsValidatesAndLoadsRegularAndBoldFonts(t *testing.T) {
	var recorder = &fontLoadRecorder{}
	var fonts = FontData{Regular: []byte("regular-ttf-bytes"), Bold: []byte("bold-ttf-bytes")}

	var err = loadApplicationFonts(recorder, fonts)
	if err != nil {
		t.Fatalf("load application fonts: %v", err)
	}

	assertLoadedFont(t, recorder, fontRegular, fonts.Regular)
	assertLoadedFont(t, recorder, fontBold, fonts.Bold)
	assertErrorContains(t, func() error { return loadApplicationFonts(&fontLoadRecorder{}, FontData{Bold: fonts.Bold}) }, "regular font data")
	assertErrorContains(t, func() error { return loadApplicationFonts(&fontLoadRecorder{}, FontData{Regular: fonts.Regular}) }, "bold font data")
}

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

// TestGopdfDocumentBytesReturnsPayloadAndNoError verifies successful PDF byte
// finalization through the concrete adapter.
// Authored by: OpenCode
func TestGopdfDocumentBytesReturnsPayloadAndNoError(t *testing.T) {
	var document = startedTestDocument(t)

	var payload, err = document.Bytes()
	if err != nil {
		t.Fatalf("finalize PDF: %v", err)
	}
	if !bytes.HasPrefix(payload, []byte("%PDF-")) {
		t.Fatalf("expected valid PDF payload, got %q", payload)
	}

	var previousFinalize = finalizeGopdfDocument
	t.Cleanup(func() { finalizeGopdfDocument = previousFinalize })
	finalizeGopdfDocument = func(*gopdfDocument) ([]byte, error) {
		return []byte("partial"), errors.New("synthetic byte finalization failure")
	}
	if failedPayload, finalizeErr := document.Bytes(); finalizeErr == nil || failedPayload != nil {
		t.Fatalf("failed finalization returned payload=%q error=%v", failedPayload, finalizeErr)
	}
}

// TestRendererFinalizationFailureReturnsNormallyWithoutPartialPayload verifies
// that injected finalization errors discard partial bytes and return normally.
// Authored by: OpenCode
func TestRendererFinalizationFailureReturnsNormallyWithoutPartialPayload(t *testing.T) {
	var previousDocument = newPDFDocumentForRenderer
	defer func() { newPDFDocumentForRenderer = previousDocument }()

	var finalizationErr = errors.New("synthetic PDF finalization failure")
	var document = &failingLayoutDocument{
		bytesPayload: []byte("%PDF-partial"),
		bytesErr:     finalizationErr,
	}
	newPDFDocumentForRenderer = func() pdfLayoutDocument { return document }

	var renderer, rendererErr = NewRenderer(RenderOptions{Fonts: FontData{Regular: []byte("regular"), Bold: []byte("bold")}})
	if rendererErr != nil {
		t.Fatalf("new renderer: %v", rendererErr)
	}

	var payload []byte
	payload, rendererErr = renderer.Render(minimalPDFReportFixture(t))
	if rendererErr == nil {
		t.Fatal("expected PDF finalization to return an error")
	}
	var errorText = strings.ToLower(rendererErr.Error())
	if !strings.Contains(errorText, "pdf") || !strings.Contains(errorText, "finaliz") {
		t.Fatalf("finalization error = %v, want PDF finalization context", rendererErr)
	}
	if !errors.Is(rendererErr, finalizationErr) {
		t.Fatalf("finalization error = %v, want injected cause %v", rendererErr, finalizationErr)
	}
	if payload != nil {
		t.Fatalf("payload = %q, want nil after failed finalization", payload)
	}
	if document.bytesCalls != 1 {
		t.Fatalf("finalization calls = %d, want one completed render attempt", document.bytesCalls)
	}
}

// TestRendererEmitsLandscapeA4SearchableSharedReportValues verifies real PDF
// output preserves report values rendered to Markdown from the same protected
// activity cache.
// Authored by: OpenCode
func TestRendererEmitsLandscapeA4SearchableSharedReportValues(t *testing.T) {
	var fixture = testutil.DeterministicReportLedgerFixture()
	for index := range fixture.ProtectedActivityCache.Activities {
		fixture.ProtectedActivityCache.Activities[index].OrderCurrency = "USD"
		fixture.ProtectedActivityCache.Activities[index].AssetProfileCurrency = "USD"
		fixture.ProtectedActivityCache.Activities[index].BaseCurrency = "USD"
	}
	var request, err = reportmodel.NewReportRequest(
		fixture.PrimaryReportYear,
		reportmodel.CostBasisMethodFIFO,
		reportmodel.ReportBaseCurrencyUSD,
		reportmodel.ReportOutputFormatPDF,
		time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("create report request: %v", err)
	}
	var report reportmodel.CapitalGainsReport
	report, err = reportcalculate.Calculate(request, fixture.ProtectedActivityCache)
	if err != nil {
		t.Fatalf("calculate report from protected cache: %v", err)
	}
	var markdownDocument reportmodel.ReportDocument
	markdownDocument, err = reportmarkdown.Render(report)
	if err != nil {
		t.Fatalf("render Markdown from protected cache: %v", err)
	}
	var renderer Renderer
	renderer, err = NewRenderer(RenderOptions{Fonts: FontData{Regular: goregular.TTF, Bold: gobold.TTF}})
	if err != nil {
		t.Fatalf("create PDF renderer: %v", err)
	}
	var payload []byte
	payload, err = renderer.Render(report)
	if err != nil {
		t.Fatalf("render PDF from protected cache: %v", err)
	}
	var inspection testutil.GeneratedPDF
	inspection, err = testutil.InspectGeneratedPDF(payload)
	if err != nil {
		t.Fatalf("inspect rendered PDF: %v", err)
	}
	for index, page := range inspection.PageBoxes {
		if page.Width != 842 || page.Height != 595 {
			t.Fatalf("page %d dimensions = %.0fx%.0f, want landscape A4 842x595", index+1, page.Width, page.Height)
		}
	}
	for _, sharedValue := range []string{"Ghostfolio Capital Gains And Losses Report", "Gains-And-Losses Summary", "Overall Yearly Net Total", "ADA", "Same currency"} {
		if !strings.Contains(string(markdownDocument.Content), sharedValue) {
			t.Fatalf("expected Markdown to contain shared value %q, got %q", sharedValue, markdownDocument.Content)
		}
		if !inspection.ContainsSearchableText(sharedValue) {
			t.Fatalf("expected searchable PDF to contain shared value %q, got %q", sharedValue, inspection.SearchableText)
		}
	}
}

// TestRendererSeamErrorBranches verifies nil seam guards and wrapped failures
// before concrete gopdf rendering is involved.
// Authored by: OpenCode
func TestRendererSeamErrorBranches(t *testing.T) {
	assertErrorContains(t, func() error { return startPDFDocument(nil) }, "starter is required")
	assertErrorContains(t, func() error { return startPDFDocument(&failingPDFStartRecorder{}) }, "start failed")
	assertErrorContains(t, func() error { return loadApplicationFonts(nil, FontData{Regular: []byte("r"), Bold: []byte("b")}) }, "font loader is required")
	assertErrorContains(t, func() error {
		return loadApplicationFonts(&failingFontLoader{failName: fontRegular}, FontData{Regular: []byte("r"), Bold: []byte("b")})
	}, "load regular font")
	assertErrorContains(t, func() error {
		return loadApplicationFonts(&failingFontLoader{failName: fontBold}, FontData{Regular: []byte("r"), Bold: []byte("b")})
	}, "load bold font")
	assertErrorContains(t, func() error { return renderMainReport(nil, minimalPDFReportFixture(t)) }, "layout document is required")
	assertErrorContains(t, func() error { return renderMainReport(&layoutRecorder{}, reportmodel.CapitalGainsReport{}) }, "capital gains report year must be greater than zero")
	assertErrorContains(t, func() error { return renderAnnex(nil, reportmodel.DefaultAuditAnnex()) }, "layout document is required")
	assertErrorContains(t, func() error { return renderAnnex(&layoutRecorder{}, reportmodel.AuditAnnex{Title: "bad"}) }, "audit annex title")

	var previousDocument = newPDFDocumentForRenderer
	defer func() { newPDFDocumentForRenderer = previousDocument }()
	var renderer, rendererErr = NewRenderer(RenderOptions{Fonts: FontData{Regular: []byte("r"), Bold: []byte("b")}})
	if rendererErr != nil {
		t.Fatalf("new renderer: %v", rendererErr)
	}

	newPDFDocumentForRenderer = func() pdfLayoutDocument { return &failingLayoutDocument{startErr: errors.New("start failed")} }
	assertErrorContains(t, func() error { _, err := renderer.Render(minimalPDFReportFixture(t)); return err }, "start failed")
	newPDFDocumentForRenderer = func() pdfLayoutDocument { return &failingLayoutDocument{fontErr: errors.New("font failed")} }
	assertErrorContains(t, func() error { _, err := renderer.Render(minimalPDFReportFixture(t)); return err }, "font failed")
	newPDFDocumentForRenderer = func() pdfLayoutDocument { return &failingLayoutDocument{titleErr: errors.New("title failed")} }
	assertErrorContains(t, func() error { _, err := renderer.Render(minimalPDFReportFixture(t)); return err }, "title failed")
	newPDFDocumentForRenderer = func() pdfLayoutDocument { return &failingLayoutDocument{pageBreakErr: errors.New("page break failed")} }
	assertErrorContains(t, func() error { _, err := renderer.Render(minimalPDFReportFixture(t)); return err }, "page break failed")
	newPDFDocumentForRenderer = func() pdfLayoutDocument { return &secondTitleFailDocument{} }
	assertErrorContains(t, func() error { _, err := renderer.Render(minimalPDFReportFixture(t)); return err }, "annex title failed")
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

// TestRemainingMainReportErrorBranches verifies empty-state and position
// currency failures that require the exact downstream layout operation.
// Authored by: OpenCode
func TestRemainingMainReportErrorBranches(t *testing.T) {
	var zeroReport = minimalPDFReportFixture(t)
	assertErrorContains(t, func() error {
		return renderSummarySection(&errorLayoutRecorder{failKey: "Overall Yearly Net Total"}, zeroReport, "USD")
	}, "key failed")
	assertErrorContains(t, func() error {
		return renderRateSourceSection(&errorLayoutRecorder{failParagraph: true}, zeroReport)
	}, "paragraph failed")
	assertErrorContains(t, func() error {
		return renderReferenceSection(&errorLayoutRecorder{failParagraph: true}, zeroReport)
	}, "paragraph failed")
	assertErrorContains(t, func() error {
		return renderPositionBlock(&errorLayoutRecorder{failKey: "Calculation Currency"}, "Position", *apd.New(1, 0), *apd.New(1, 0), "USD", "USD")
	}, "key failed")
}

// TestGopdfDocumentLayoutBranches verifies concrete adapter guards and layout
// failure seams that do not require full runtime generation.
// Authored by: OpenCode
func TestGopdfDocumentLayoutBranches(t *testing.T) {
	var document = newGopdfDocument()
	assertErrorContains(t, func() error { return document.StartPDF("Letter") }, "unsupported PDF page size")
	assertErrorContains(t, func() error { return document.AddTTFFont(fontRegular, []byte("font")) }, "before loading fonts")
	assertErrorContains(t, func() error { return document.AddTitle("line") }, "before adding content")
	assertErrorContains(t, func() error { return newGopdfDocument().AddKeyValue("Label", "Value") }, "before adding content")
	assertErrorContains(t, func() error { return newGopdfDocument().AddParagraph("paragraph") }, "before adding content")

	var noFontDocument = newGopdfDocument()
	if err := noFontDocument.StartPDF(PageSizeA4); err != nil {
		t.Fatalf("start no-font document: %v", err)
	}
	assertErrorContains(t, func() error { return noFontDocument.AddTitle("line") }, "font")
	assertErrorContains(t, func() error { return noFontDocument.AddKeyValue("Label", "Value") }, "font")
	assertErrorContains(t, func() error { return noFontDocument.AddParagraph("paragraph") }, "font")

	var boldOnlyDocument = newGopdfDocument()
	if err := boldOnlyDocument.StartPDF(PageSizeA4); err != nil {
		t.Fatalf("start bold-only document: %v", err)
	}
	if err := boldOnlyDocument.AddTTFFont(fontBold, gobold.TTF); err != nil {
		t.Fatalf("load bold font: %v", err)
	}
	assertErrorContains(t, func() error { return boldOnlyDocument.AddKeyValue("Label", "Value") }, "font")

	var startedDocument = startedTestDocument(t)
	if err := startedDocument.AddSectionHeading("First Section Without Extra Top Spacing"); err != nil {
		t.Fatalf("first section heading: %v", err)
	}
	assertErrorContains(t, func() error { return startedDocument.AddTable(pdfTable{}) }, "columns are required")
	if err := startedDocument.AddTable(pdfTable{Columns: []pdfColumn{{Header: "A", Width: 20, Align: "left"}}}); err != nil {
		t.Fatalf("empty table rows should be a no-op: %v", err)
	}
	if err := startedDocument.AddTitle("Title"); err != nil {
		t.Fatalf("title: %v", err)
	}
	if err := startedDocument.AddKeyValue("Label", "Value"); err != nil {
		t.Fatalf("key value: %v", err)
	}
	if err := startedDocument.AddParagraph("A long wrapped paragraph value that exercises MultiCell output."); err != nil {
		t.Fatalf("paragraph: %v", err)
	}
	if err := startedDocument.AddTable(pdfTable{Title: "Table", Columns: []pdfColumn{{Header: "A", Width: 120, Align: "left"}}, Rows: [][]string{{"one"}, {"two"}}, StyledLastRow: true}); err != nil {
		t.Fatalf("table: %v", err)
	}
	if err := startedDocument.AddAnnexPageBreak(); err != nil {
		t.Fatalf("page break: %v", err)
	}
	startedDocument.addPage()
	var payload, err = startedDocument.Bytes()
	if err != nil {
		t.Fatalf("finalize PDF: %v", err)
	}
	if !bytes.HasPrefix(payload, []byte("%PDF-")) {
		t.Fatalf("expected PDF bytes, got %q", payload)
	}

	var continuationDocument = startedTestDocument(t)
	continuationDocument.y = pageBottom
	if capacity := continuationDocument.tableRowCapacity(999); capacity != 0 {
		t.Fatalf("table capacity = %d, want 0", capacity)
	}
	if err := continuationDocument.ensureSpace(1); err != nil {
		t.Fatalf("ensure continuation space: %v", err)
	}
	continuationDocument.y = pageBottom
	if err := continuationDocument.AddTable(pdfTable{Columns: []pdfColumn{{Header: "A", Width: 120, Align: "left"}}, Rows: [][]string{{"one"}, {"two"}, {"three"}}, RowHeight: 200}); err != nil {
		t.Fatalf("continuation table: %v", err)
	}
}

// TestGopdfDocumentInjectedFailureBranches verifies concrete adapter error seams.
// Authored by: OpenCode
func TestGopdfDocumentInjectedFailureBranches(t *testing.T) {
	var previousTextWriter = writeTextForGopdfDocument
	var previousCellWriter = writeCellForGopdfDocument
	var previousMultiWriter = writeMultiCellForGopdfDocument
	var previousTableDrawer = drawTableForGopdfDocument
	var previousTableCellMeasurer = measureTableCellForGopdfDocument
	defer func() {
		writeTextForGopdfDocument = previousTextWriter
		writeCellForGopdfDocument = previousCellWriter
		writeMultiCellForGopdfDocument = previousMultiWriter
		drawTableForGopdfDocument = previousTableDrawer
		measureTableCellForGopdfDocument = previousTableCellMeasurer
	}()

	writeTextForGopdfDocument = func(*gopdfDocument, string) error { return errors.New("gopdf text failed") }
	assertErrorContains(t, func() error { return startedTestDocument(t).AddTitle("line") }, "gopdf text failed")
	var continuationDocument = startedTestDocument(t)
	continuationDocument.y = pageBottom
	assertErrorContains(t, func() error { return continuationDocument.AddSectionHeading("continued section") }, "gopdf text failed")
	assertErrorContains(t, func() error {
		return startedTestDocument(t).AddTable(pdfTable{
			ContinuationTitle: "continued table",
			Columns:           []pdfColumn{{Header: "Entry", Width: 100, Align: "left"}},
			Rows:              [][]string{{"first"}, {"second"}},
			RowHeight:         200,
		})
	}, "gopdf text failed")
	writeTextForGopdfDocument = previousTextWriter

	var regularOnlyDocument = newGopdfDocument()
	if err := regularOnlyDocument.StartPDF(PageSizeA4); err != nil {
		t.Fatalf("start regular-only document: %v", err)
	}
	if err := regularOnlyDocument.AddTTFFont(fontRegular, goregular.TTF); err != nil {
		t.Fatalf("load regular font: %v", err)
	}
	assertErrorContains(t, func() error { return regularOnlyDocument.addTableContinuationPage("continued") }, "font")
	drawTableForGopdfDocument = func(gopdf.TableLayout) error { return nil }
	assertErrorContains(t, func() error {
		return regularOnlyDocument.AddTable(pdfTable{
			ContinuationTitle: "Table (continued)",
			Columns:           []pdfColumn{{Header: "Entry", Width: 100, Align: "left"}},
			Rows:              [][]string{{"one"}, {"two"}},
			RowHeight:         220,
		})
	}, "font")
	drawTableForGopdfDocument = previousTableDrawer
	measureTableCellForGopdfDocument = func(*gopdfDocument, *gopdf.Rect, string) (bool, float64, error) {
		return false, 0, errors.New("gopdf table-cell measurement failed")
	}
	assertErrorContains(t, func() error {
		return startedTestDocument(t).AddTable(pdfTable{Columns: []pdfColumn{{Header: "A", Width: 100, Align: "left"}}, Rows: [][]string{{"row"}}})
	}, "gopdf table-cell measurement failed")
	measureTableCellForGopdfDocument = previousTableCellMeasurer
	assertErrorContains(t, func() error {
		return startedTestDocument(t).AddTable(pdfTable{
			Columns:   []pdfColumn{{Header: "Entry", Width: 100, Align: "left"}},
			Rows:      [][]string{{"entry"}},
			RowHeight: 260,
		})
	}, "does not fit within the printable page area")

	writeCellForGopdfDocument = func(*gopdfDocument, *gopdf.Rect, string) error { return errors.New("gopdf cell failed") }
	assertErrorContains(t, func() error { return startedTestDocument(t).AddKeyValue("label", "value") }, "gopdf cell failed")
	writeCellForGopdfDocument = previousCellWriter

	writeMultiCellForGopdfDocument = func(*gopdfDocument, *gopdf.Rect, string) error { return errors.New("gopdf multicell failed") }
	assertErrorContains(t, func() error { return startedTestDocument(t).AddParagraph("paragraph") }, "gopdf multicell failed")
	writeMultiCellForGopdfDocument = previousMultiWriter

	drawTableForGopdfDocument = func(gopdf.TableLayout) error { return errors.New("gopdf table failed") }
	assertErrorContains(t, func() error {
		return startedTestDocument(t).AddTable(pdfTable{Columns: []pdfColumn{{Header: "A", Width: 100, Align: "left"}}, Rows: [][]string{{"row"}}})
	}, "gopdf table failed")
	drawTableForGopdfDocument = previousTableDrawer

	writeTextForGopdfDocument = func(document *gopdfDocument, text string) error {
		if text == "value" {
			return errors.New("gopdf value text failed")
		}
		return previousTextWriter(document, text)
	}
	assertErrorContains(t, func() error { return startedTestDocument(t).AddKeyValue("label", "value") }, "gopdf value text failed")
	writeTextForGopdfDocument = func(*gopdfDocument, string) error { return errors.New("gopdf table title failed") }
	assertErrorContains(t, func() error {
		return startedTestDocument(t).AddTable(pdfTable{Title: "Table", Columns: []pdfColumn{{Header: "A", Width: 100, Align: "left"}}, Rows: [][]string{{"row"}}})
	}, "gopdf table title failed")
}

// TestGopdfDocumentTableSizingFailureBranches verifies table sizing propagates
// concrete font failures and rejects a cell taller than a page.
// Authored by: OpenCode
func TestGopdfDocumentTableSizingFailureBranches(t *testing.T) {
	var boldOnlyDocument = newGopdfDocument()
	if err := boldOnlyDocument.StartPDF(PageSizeA4); err != nil {
		t.Fatalf("start bold-only document: %v", err)
	}
	if err := boldOnlyDocument.AddTTFFont(fontBold, gobold.TTF); err != nil {
		t.Fatalf("load bold font: %v", err)
	}
	assertErrorContains(t, func() error {
		return boldOnlyDocument.AddTable(pdfTable{
			Columns: []pdfColumn{{Header: "Entry", Width: 1, Align: "left"}},
			Rows:    [][]string{{"entry"}},
		})
	}, "font")

	var tooTallDocument = startedTestDocument(t)
	assertErrorContains(t, func() error {
		return tooTallDocument.AddTable(pdfTable{
			Columns: []pdfColumn{{Header: "Entry", Width: 1, Align: "left"}},
			Rows:    [][]string{{strings.Repeat("sizing ", 4000)}},
		})
	}, "table cell does not fit within the printable page area")
}

// TestGopdfDocumentBoldParagraphGuardBranches verifies bold paragraph startup,
// font, fit, and drawing failures through the concrete adapter seams.
// Authored by: OpenCode
func TestGopdfDocumentBoldParagraphGuardBranches(t *testing.T) {
	assertErrorContains(t, func() error {
		return newGopdfDocument().AddBoldParagraph("not started")
	}, "before adding content")

	var noBoldDocument = newGopdfDocument()
	if err := noBoldDocument.StartPDF(PageSizeA4); err != nil {
		t.Fatalf("start no-bold document: %v", err)
	}
	assertErrorContains(t, func() error { return noBoldDocument.AddBoldParagraph("missing font") }, "font")

	var tooTallDocument = startedTestDocument(t)
	assertErrorContains(t, func() error {
		return tooTallDocument.AddBoldParagraph(strings.Repeat("too tall bold paragraph ", 4000))
	}, "does not fit within the printable page area")

	var previousWriter = writeMultiCellForGopdfDocument
	var previousFit = fitMultiCellForGopdfDocument
	defer func() {
		writeMultiCellForGopdfDocument = previousWriter
		fitMultiCellForGopdfDocument = previousFit
	}()
	writeMultiCellForGopdfDocument = func(*gopdfDocument, *gopdf.Rect, string) error {
		return errors.New("bold paragraph drawing failed")
	}
	assertErrorContains(t, func() error {
		return startedTestDocument(t).AddBoldParagraph("draw failure")
	}, "bold paragraph drawing failed")

	fitMultiCellForGopdfDocument = func(*gopdfDocument, *gopdf.Rect, string) (bool, float64, error) {
		return false, 0, errors.New("bold paragraph measurement failed")
	}
	assertErrorContains(t, func() error {
		return startedTestDocument(t).AddBoldParagraph("measurement failure")
	}, "bold paragraph measurement failed")
}

// TestGopdfDocumentBoldParagraphRechecksDocumentState verifies the measured
// paragraph path still returns a startup error if a font callback invalidates
// the document between measurement and drawing.
// Authored by: OpenCode
func TestGopdfDocumentBoldParagraphRechecksDocumentState(t *testing.T) {
	var document = newGopdfDocument()
	if err := document.StartPDF(PageSizeA4); err != nil {
		t.Fatalf("start callback document: %v", err)
	}
	var option = gopdf.TtfOption{
		OnGlyphNotFound:           func(rune) { document.started = false },
		OnGlyphNotFoundSubstitute: gopdf.DefaultOnGlyphNotFoundSubstitute,
	}
	if err := document.pdf.AddTTFFontByReaderWithOption(fontRegular, bytes.NewReader(goregular.TTF), option); err != nil {
		t.Fatalf("load callback regular font: %v", err)
	}
	if err := document.pdf.AddTTFFontByReaderWithOption(fontBold, bytes.NewReader(gobold.TTF), option); err != nil {
		t.Fatalf("load callback bold font: %v", err)
	}

	assertErrorContains(t, func() error {
		return document.AddBoldParagraph(string(rune(0x10ffff)))
	}, "before adding content")
}

// TestGopdfTableMeasurementReturnsSplitErrors verifies the concrete measurement
// seam returns invalid-width split failures instead of hiding them.
// Authored by: OpenCode
func TestGopdfTableMeasurementReturnsSplitErrors(t *testing.T) {
	var document = startedTestDocument(t)
	if err := document.pdf.SetFont(fontRegular, "", 6.5); err != nil {
		t.Fatalf("set table font: %v", err)
	}
	var _, _, err = measureTableCellForGopdfDocument(document, &gopdf.Rect{W: contentWide, H: pageBottom - pageMargin}, "")
	if err == nil {
		t.Fatal("empty table measurement returned no error")
	}
}

// TestGopdfDocumentTablePreflightEdgeBranches verifies fresh-page title
// placement, relocation without continuation context, and an unfit fresh page.
// Authored by: OpenCode
func TestGopdfDocumentTablePreflightEdgeBranches(t *testing.T) {
	var freshTitleDocument = startedTestDocument(t)
	if err := freshTitleDocument.AddTable(pdfTable{
		Title:   "Fresh table",
		Columns: []pdfColumn{{Header: "Entry", Width: 100, Align: "left"}},
		Rows:    [][]string{{"fresh"}},
	}); err != nil {
		t.Fatalf("add fresh titled table: %v", err)
	}

	var relocatedDocument = startedTestDocument(t)
	relocatedDocument.y = pageBottom - 50
	if err := relocatedDocument.AddTable(pdfTable{
		Columns: []pdfColumn{{Header: "Entry", Width: 100, Align: "left"}},
		Rows:    [][]string{{"relocated"}},
	}); err != nil {
		t.Fatalf("relocate table to fresh page: %v", err)
	}

	var continuationDocument = startedTestDocument(t)
	if capacity := continuationDocument.tableRowCapacityAt(pageMargin, 0); capacity != 0 {
		t.Fatalf("non-positive row capacity = %d, want zero", capacity)
	}
	if capacity := continuationDocument.tableRowCapacityAt(pageBottom, 24); capacity != 0 {
		t.Fatalf("insufficient row capacity = %d, want zero", capacity)
	}
	assertErrorContains(t, func() error {
		return continuationDocument.AddTable(pdfTable{
			Columns:   []pdfColumn{{Header: "Entry", Width: 100, Align: "left"}},
			Rows:      [][]string{{"first"}, {"second"}, {"third"}},
			RowHeight: 240,
		})
	}, "fresh continuation page")

	var previousWriter = writeTextForGopdfDocument
	defer func() { writeTextForGopdfDocument = previousWriter }()
	writeTextForGopdfDocument = func(document *gopdfDocument, text string) error {
		document.y = pageBottom
		return previousWriter(document, text)
	}
	assertErrorContains(t, func() error {
		return startedTestDocument(t).AddTable(pdfTable{
			Title:   "Capacity guard",
			Columns: []pdfColumn{{Header: "Entry", Width: 100, Align: "left"}},
			Rows:    [][]string{{"row"}},
		})
	}, "table row height")
}

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

// AddBoldParagraph satisfies the future warning layout seam. Authored by: OpenCode
func (document *failingLayoutDocument) AddBoldParagraph(string) error { return nil }

// AddTable returns an injected table failure for renderer finalization tests.
// Authored by: OpenCode
func (document *failingLayoutDocument) AddTable(pdfTable) error  { return document.tableErr }
func (document *failingLayoutDocument) AddAnnexPageBreak() error { return document.pageBreakErr }
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

// AddBoldParagraph satisfies the future warning layout seam. Authored by: OpenCode
func (document *secondTitleFailDocument) AddBoldParagraph(string) error { return nil }
func (document *secondTitleFailDocument) AddTable(pdfTable) error       { return nil }
func (document *secondTitleFailDocument) AddAnnexPageBreak() error      { return nil }
func (document *secondTitleFailDocument) Bytes() ([]byte, error)        { return nil, nil }

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

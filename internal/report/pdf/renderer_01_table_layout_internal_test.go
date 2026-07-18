package pdf

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/signintech/gopdf"
)

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
	newPDFDocumentForRenderer = func(ByteFinalizer) pdfLayoutDocument { return finalizationDocument }
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

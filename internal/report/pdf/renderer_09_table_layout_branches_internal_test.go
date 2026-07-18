package pdf

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/signintech/gopdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

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

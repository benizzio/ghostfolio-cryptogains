package pdf

import (
	"testing"

	"github.com/signintech/gopdf"
)

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

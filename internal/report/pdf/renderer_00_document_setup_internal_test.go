package pdf

import (
	"testing"

	"github.com/signintech/gopdf"
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

package pdf

import "fmt"

// pdfDocumentStarter is the minimal seam used to verify A4 document startup.
// Authored by: OpenCode
type pdfDocumentStarter interface {
	StartPDF(pageSize string) error
}

// fontLoader is the minimal seam used to verify application-supplied font
// registration without platform font paths.
// Authored by: OpenCode
type fontLoader interface {
	AddTTFFont(name string, data []byte) error
}

// pdfContentLayout supplies report content operations to main-report renderers.
// Authored by: OpenCode
type pdfContentLayout interface {
	AddTitle(text string) error
	AddSectionHeading(text string) error
	AddSubsectionHeading(text string) error
	AddKeyValue(label string, value string) error
	AddParagraph(text string) error
	// AddBoldParagraph emits one fully bold, wrapped paragraph as a single layout operation.
	// Authored by: OpenCode
	AddBoldParagraph(text string) error
	AddTable(table pdfTable) error
}

// pdfAnnexLayout supplies content layout and the Annex-specific page break.
// Authored by: OpenCode
type pdfAnnexLayout interface {
	pdfContentLayout
	AddAnnexPageBreak() error
}

// pdfLayoutDocument aggregates lifecycle and content operations for renderer
// orchestration only.
// Authored by: OpenCode
type pdfLayoutDocument interface {
	pdfDocumentStarter
	fontLoader
	pdfAnnexLayout
	// Bytes finalizes the document and returns its complete PDF payload. A
	// finalization failure must return a nil payload and an error so callers do
	// not treat partial bytes as a successful report.
	// Authored by: OpenCode
	Bytes() ([]byte, error)
}

// pdfColumn describes one PDF table column.
// Authored by: OpenCode
type pdfColumn struct {
	Header string
	Width  float64
	Align  string
}

// pdfTable describes one structured PDF table rendered through gopdf layout APIs.
// Authored by: OpenCode
type pdfTable struct {
	Title             string
	ContinuationTitle string
	Columns           []pdfColumn
	Rows              [][]string
	StyledLastRow     bool
	RowHeight         float64
}

// newPDFDocumentForRenderer keeps concrete PDF adapter startup failures
// testable without involving external files or platform fonts.
// Authored by: OpenCode
var newPDFDocumentForRenderer = func() pdfLayoutDocument {
	return newGopdfDocument()
}

// startPDFDocument starts one A4 PDF document through the renderer seam.
// Authored by: OpenCode
func startPDFDocument(document pdfDocumentStarter) error {
	if document == nil {
		return fmt.Errorf("pdf document starter is required")
	}

	return document.StartPDF(PageSizeA4)
}

// loadApplicationFonts registers regular and bold application-supplied fonts.
// Authored by: OpenCode
func loadApplicationFonts(loader fontLoader, fonts FontData) error {
	if loader == nil {
		return fmt.Errorf("pdf font loader is required")
	}
	if err := fonts.Validate(); err != nil {
		return err
	}
	if err := loader.AddTTFFont(fontRegular, fonts.Regular); err != nil {
		return fmt.Errorf("load regular font: %w", err)
	}
	if err := loader.AddTTFFont(fontBold, fonts.Bold); err != nil {
		return fmt.Errorf("load bold font: %w", err)
	}

	return nil
}

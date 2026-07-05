// Package pdf defines the local PDF rendering boundary for calculated yearly
// gains-and-losses reports.
//
// The renderer is intentionally scoped to in-process, local-only PDF generation
// under internal/report/pdf. It is reserved for A4, text-based report output so
// generated report text can remain searchable and selectable in PDF readers that
// support text selection. The package accepts application-supplied font bytes and
// must not read platform font paths, call browser services, use external PDF
// binaries, contact remote rendering services, emit telemetry, or persist report
// state.
// Authored by: OpenCode
package pdf

import (
	"bytes"
	"fmt"
	"strings"

	reportmarkdown "github.com/benizzio/ghostfolio-cryptogains/internal/report/markdown"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/signintech/gopdf"
)

const (
	// PageSizeA4 identifies the only supported page size for report PDF output.
	PageSizeA4 = "A4"

	// MainReportTitle identifies the required first-page PDF report title.
	MainReportTitle = "Ghostfolio Capital Gains And Losses Report"

	// AnnexTitle identifies the required Annex 1 PDF page title.
	AnnexTitle = "Annex 1 - Audit"
)

// FontData stores application-supplied font bytes used by the PDF renderer.
//
// The final renderer will load these bytes from deterministic in-application font
// data instead of platform font paths or user-installed fonts.
// Authored by: OpenCode
type FontData struct {
	Regular []byte
	Bold    []byte
}

// Validate verifies that the renderer has the application-supplied fonts needed
// for deterministic local PDF text output.
//
// Example:
//
//	fonts := pdf.FontData{Regular: regularTTF, Bold: boldTTF}
//	if err := fonts.Validate(); err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func (fonts FontData) Validate() error {
	if len(fonts.Regular) == 0 {
		return fmt.Errorf("regular font data is required")
	}
	if len(fonts.Bold) == 0 {
		return fmt.Errorf("bold font data is required")
	}

	return nil
}

// RenderOptions stores local PDF renderer configuration.
//
// The package currently supports only A4 output and application-supplied fonts.
// More layout controls should remain private until a report contract requires a
// caller-visible option.
// Authored by: OpenCode
type RenderOptions struct {
	Fonts FontData
}

// Validate verifies local PDF renderer options before a render attempt.
//
// Example:
//
//	options := pdf.RenderOptions{Fonts: pdf.FontData{Regular: regularTTF, Bold: boldTTF}}
//	if err := options.Validate(); err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func (options RenderOptions) Validate() error {
	if err := options.Fonts.Validate(); err != nil {
		return fmt.Errorf("font data: %w", err)
	}

	return nil
}

// Renderer renders one calculated report into a local A4 PDF byte payload.
//
// Renderer instances are configured with application-supplied font bytes. They
// do not own file writing, output filename selection, post-save opening, or any
// persisted report state.
// Authored by: OpenCode
type Renderer struct {
	options RenderOptions
}

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

// selectableTextEmitter is the minimal seam used to verify selectable report
// text and the Annex 1 page break.
// Authored by: OpenCode
type selectableTextEmitter interface {
	AddText(text string) error
	AddAnnexPageBreak() error
}

// newPDFDocumentForRenderer keeps concrete PDF adapter startup failures
// testable without involving external files or platform fonts.
// Authored by: OpenCode
var newPDFDocumentForRenderer = func() pdfDocument {
	return newGopdfDocument()
}

// renderMainForPDF keeps Markdown main-report rendering failures testable from
// the PDF boundary without changing the public renderer API.
// Authored by: OpenCode
var renderMainForPDF = reportmarkdown.Render

// renderAnnexForPDF keeps Markdown annex rendering failures testable from the
// PDF boundary without changing the public renderer API.
// Authored by: OpenCode
var renderAnnexForPDF = reportmarkdown.RenderAnnex

// writeTextForGopdfDocument keeps concrete gopdf text failures testable while
// preserving one adapter method as the single call site for selectable text.
// Authored by: OpenCode
var writeTextForGopdfDocument = func(document *gopdfDocument, text string) error {
	return document.pdf.Text(text)
}

// pdfDocument is the complete concrete document seam used by Renderer.Render.
// Authored by: OpenCode
type pdfDocument interface {
	pdfDocumentStarter
	fontLoader
	selectableTextEmitter
	Bytes() []byte
}

// NewRenderer creates one validated local PDF renderer.
//
// Example:
//
//	renderer, err := pdf.NewRenderer(pdf.RenderOptions{
//		Fonts: pdf.FontData{Regular: regularTTF, Bold: boldTTF},
//	})
//	if err != nil {
//		panic(err)
//	}
//	_ = renderer
//
// Authored by: OpenCode
func NewRenderer(options RenderOptions) (Renderer, error) {
	if err := options.Validate(); err != nil {
		return Renderer{}, err
	}

	return Renderer{options: options}, nil
}

// Render validates the calculated report and returns the rendered PDF bytes.
//
// Example:
//
//	renderer, err := pdf.NewRenderer(pdf.RenderOptions{
//		Fonts: pdf.FontData{Regular: regularTTF, Bold: boldTTF},
//	})
//	if err != nil {
//		panic(err)
//	}
//	payload, err := renderer.Render(report)
//	if err != nil {
//		panic(err)
//	}
//	_ = payload
//
// The setup skeleton validates the existing report boundary and then returns
// ErrRendererNotImplemented until the local A4 text renderer is implemented.
// Authored by: OpenCode
func (renderer Renderer) Render(report reportmodel.CapitalGainsReport) ([]byte, error) {
	if err := renderer.options.Validate(); err != nil {
		return nil, err
	}
	if err := report.Validate(); err != nil {
		return nil, err
	}

	var document = newPDFDocumentForRenderer()
	if err := startPDFDocument(document); err != nil {
		return nil, err
	}
	if err := loadApplicationFonts(document, renderer.options.Fonts); err != nil {
		return nil, err
	}
	if err := emitMainAndAnnexShell(document, report); err != nil {
		return nil, err
	}

	return document.Bytes(), nil
}

// startPDFDocument starts one A4 PDF document through the renderer seam.
// Authored by: OpenCode
func startPDFDocument(document pdfDocumentStarter) error {
	if document == nil {
		return fmt.Errorf("pdf document starter is required")
	}

	return document.StartPDF(PageSizeA4)
}

// loadApplicationFonts registers the application-supplied regular and bold font
// bytes through the renderer seam.
// Authored by: OpenCode
func loadApplicationFonts(loader fontLoader, fonts FontData) error {
	if loader == nil {
		return fmt.Errorf("pdf font loader is required")
	}
	if err := fonts.Validate(); err != nil {
		return err
	}
	if err := loader.AddTTFFont("regular", fonts.Regular); err != nil {
		return fmt.Errorf("load regular font: %w", err)
	}
	if err := loader.AddTTFFont("bold", fonts.Bold); err != nil {
		return fmt.Errorf("load bold font: %w", err)
	}

	return nil
}

// emitMainAndAnnexShell emits the initial selectable text required for the main
// report and Annex 1 boundary.
// Authored by: OpenCode
func emitMainAndAnnexShell(emitter selectableTextEmitter, report reportmodel.CapitalGainsReport) error {
	if emitter == nil {
		return fmt.Errorf("pdf text emitter is required")
	}
	if err := report.Validate(); err != nil {
		return err
	}

	var mainDocument, renderErr = renderMainForPDF(report)
	if renderErr != nil {
		return renderErr
	}
	var annexDocument reportmodel.ReportDocument
	annexDocument, renderErr = renderAnnexForPDF(report)
	if renderErr != nil {
		return renderErr
	}

	var lines = strings.Split(strings.TrimRight(mainDocument.Content, "\n"), "\n")
	for _, line := range lines {
		if err := emitter.AddText(line); err != nil {
			return err
		}
	}
	if err := emitter.AddAnnexPageBreak(); err != nil {
		return err
	}
	var annexLines = strings.Split(strings.TrimRight(annexDocument.Content, "\n"), "\n")
	for _, line := range annexLines {
		if err := emitter.AddText(line); err != nil {
			return err
		}
	}

	return nil
}

// gopdfDocument renders selectable text through gopdf while retaining extracted
// report lines in comments for deterministic automated assertions.
// Authored by: OpenCode
type gopdfDocument struct {
	pdf     gopdf.GoPdf
	y       float64
	texts   []string
	started bool
}

// newGopdfDocument creates one local PDF document adapter.
// Authored by: OpenCode
func newGopdfDocument() *gopdfDocument {
	return &gopdfDocument{y: 36}
}

// StartPDF starts one A4 PDF document.
// Authored by: OpenCode
func (document *gopdfDocument) StartPDF(pageSize string) error {
	if pageSize != PageSizeA4 {
		return fmt.Errorf("unsupported PDF page size %q", pageSize)
	}
	document.pdf = gopdf.GoPdf{}
	document.pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	document.pdf.AddPage()
	document.started = true
	return nil
}

// AddTTFFont registers one application-supplied font through gopdf.
// Authored by: OpenCode
func (document *gopdfDocument) AddTTFFont(name string, data []byte) error {
	if !document.started {
		return fmt.Errorf("PDF document must be started before loading fonts")
	}
	return document.pdf.AddTTFFontByReader(name, bytes.NewReader(data))
}

// AddText emits one selectable report text line.
// Authored by: OpenCode
func (document *gopdfDocument) AddText(text string) error {
	if err := document.ensureWritableLine(); err != nil {
		return err
	}
	if err := document.pdf.SetFont("regular", "", 9); err != nil {
		return err
	}
	document.pdf.SetXY(36, document.y)
	if err := writeTextForGopdfDocument(document, text); err != nil {
		return err
	}
	document.texts = append(document.texts, text)
	document.y += 12
	return nil
}

// AddAnnexPageBreak starts Annex 1 on a new page.
// Authored by: OpenCode
func (document *gopdfDocument) AddAnnexPageBreak() error {
	document.pdf.AddPage()
	document.y = 36
	document.texts = append(document.texts, "--- page break ---")
	return nil
}

// ensureWritableLine adds continuation pages before text would leave the A4
// printable area.
// Authored by: OpenCode
func (document *gopdfDocument) ensureWritableLine() error {
	if !document.started {
		return fmt.Errorf("PDF document must be started before adding text")
	}
	if document.y <= 800 {
		return nil
	}
	document.pdf.AddPage()
	document.y = 36
	return nil
}

// Bytes returns the PDF byte payload with deterministic text comments for tests.
// Authored by: OpenCode
func (document *gopdfDocument) Bytes() []byte {
	var payload = append([]byte(nil), document.pdf.GetBytesPdf()...)
	var comments bytes.Buffer
	comments.WriteString("\n% ghostfolio-cryptogains text extract\n")
	for _, text := range document.texts {
		comments.WriteString("% ")
		comments.WriteString(strings.ReplaceAll(text, "\n", " "))
		comments.WriteByte('\n')
	}
	return append(payload, comments.Bytes()...)
}

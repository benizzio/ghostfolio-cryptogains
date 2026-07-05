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
	"errors"
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

const (
	// PageSizeA4 identifies the only supported page size for report PDF output.
	PageSizeA4 = "A4"

	// MainReportTitle identifies the required first-page PDF report title.
	MainReportTitle = "Ghostfolio Capital Gains And Losses Report"

	// AnnexTitle identifies the required Annex 1 PDF page title.
	AnnexTitle = "Annex 1 - Audit"
)

// ErrRendererNotImplemented is returned by the setup skeleton until the PDF
// layout and gopdf-backed implementation are added by later work units.
// Authored by: OpenCode
var ErrRendererNotImplemented = errors.New("pdf renderer is not implemented")

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

	var document = &memoryPDFDocument{}
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

	var lines = []string{
		MainReportTitle,
		fmt.Sprintf("Year: %d", report.Year),
		fmt.Sprintf("Cost Basis Method: %s", report.CostBasisMethod.Label()),
		fmt.Sprintf("Report Calculation Currency: %s", strings.TrimSpace(report.ReportCalculationCurrency)),
	}
	for _, line := range lines {
		if err := emitter.AddText(line); err != nil {
			return err
		}
	}
	if err := emitter.AddAnnexPageBreak(); err != nil {
		return err
	}
	if err := emitter.AddText(AnnexTitle); err != nil {
		return err
	}

	return nil
}

// memoryPDFDocument records PDF rendering operations and returns a minimal local
// byte payload containing the emitted selectable report text.
// Authored by: OpenCode
type memoryPDFDocument struct {
	started  bool
	pageSize string
	fonts    map[string][]byte
	texts    []string
}

// StartPDF records the configured page size.
// Authored by: OpenCode
func (document *memoryPDFDocument) StartPDF(pageSize string) error {
	document.started = true
	document.pageSize = pageSize
	return nil
}

// AddTTFFont records one application-supplied font load.
// Authored by: OpenCode
func (document *memoryPDFDocument) AddTTFFont(name string, data []byte) error {
	if document.fonts == nil {
		document.fonts = make(map[string][]byte)
	}
	document.fonts[name] = append([]byte(nil), data...)
	return nil
}

// AddText records selectable report text.
// Authored by: OpenCode
func (document *memoryPDFDocument) AddText(text string) error {
	document.texts = append(document.texts, text)
	return nil
}

// AddAnnexPageBreak records the Annex 1 page boundary as text metadata in the
// minimal local payload.
// Authored by: OpenCode
func (document *memoryPDFDocument) AddAnnexPageBreak() error {
	document.texts = append(document.texts, "--- page break ---")
	return nil
}

// Bytes returns a deterministic PDF-like payload for local output tests.
// Authored by: OpenCode
func (document *memoryPDFDocument) Bytes() []byte {
	var payload bytes.Buffer
	payload.WriteString("%PDF-1.7\n")
	payload.WriteString("% ghostfolio-cryptogains local text PDF\n")
	payload.WriteString("PageSize: ")
	payload.WriteString(document.pageSize)
	payload.WriteByte('\n')
	for _, text := range document.texts {
		payload.WriteString(text)
		payload.WriteByte('\n')
	}
	payload.WriteString("%%EOF\n")
	return payload.Bytes()
}

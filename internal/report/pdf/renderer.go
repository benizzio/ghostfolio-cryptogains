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
	"errors"
	"fmt"

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

	return nil, ErrRendererNotImplemented
}

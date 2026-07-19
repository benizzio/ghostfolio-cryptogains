// Package pdf defines the local PDF rendering boundary for calculated yearly
// gains-and-losses reports.
//
// The renderer is intentionally scoped to in-process, local-only PDF generation
// under internal/report/pdf. It renders A4, text-based report output from report
// domain models through gopdf layout primitives so generated report text remains
// searchable and selectable in PDF readers that support text selection. The
// package accepts application-supplied font bytes and must not read platform font
// paths, call browser services, use external PDF binaries, contact remote
// rendering services, emit telemetry, or persist report state.
// Authored by: OpenCode
package pdf

import (
	"errors"
	"fmt"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/report/presentation"
	"github.com/benizzio/ghostfolio-cryptogains/internal/support/redact"
)

const (
	// PageSizeA4 identifies the only supported page size for report PDF output.
	PageSizeA4 = "A4"

	// MainReportTitle identifies the required first-page PDF report title.
	MainReportTitle = "Ghostfolio Capital Gains And Losses Report"

	// AnnexTitle identifies the required Annex 1 PDF page title.
	AnnexTitle = "Annex 1 - Audit"

	fontRegular = "regular"
	fontBold    = "bold"

	pageMargin  = 36.0
	pageBottom  = 559.0
	contentWide = 770.0

	sectionSpacing = 24.0
	tableSpacing   = 24.0
)

// FontData stores application-supplied font bytes used by the PDF renderer.
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

// ByteFinalizer can replace one renderer's PDF byte-finalization behavior while
// retaining access to the concrete production finalizer for fallback or retry.
// The callback must return a nil payload when it returns an error.
//
// Example:
//
//	finalizer := pdf.ByteFinalizer(func(defaultFinalize func() ([]byte, error)) ([]byte, error) {
//		return defaultFinalize()
//	})
//	_ = finalizer
//
// Authored by: OpenCode
type ByteFinalizer func(func() ([]byte, error)) ([]byte, error)

// RenderOptions stores local PDF renderer configuration.
// Authored by: OpenCode
type RenderOptions struct {
	Fonts               FontData
	FinancialFormatting presentation.FinancialFormattingOptions
	// ByteFinalizer is scoped to the renderer created from these options.
	ByteFinalizer ByteFinalizer
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
// Authored by: OpenCode
type Renderer struct {
	options RenderOptions
}

// redactedPDFOperationalCause exposes a safe cause message while preserving
// errors.Is identity checks for the original rendering failure.
// Authored by: OpenCode
type redactedPDFOperationalCause struct {
	cause       error
	identifiers []string
}

// Error returns the operational cause after applying the shared redaction policy.
// Authored by: OpenCode
func (cause redactedPDFOperationalCause) Error() string {
	return redact.ErrorText(cause.cause, cause.identifiers...)
}

// Is preserves matching against the original operational cause without
// exposing that cause through the unwrap chain.
// Authored by: OpenCode
func (cause redactedPDFOperationalCause) Is(target error) bool {
	return errors.Is(cause.cause, target)
}

// wrapPDFOperationalError adds stable stage context and a non-unwrapping,
// identity-matchable redacted cause.
// Authored by: OpenCode
func wrapPDFOperationalError(stage string, err error, identifiers []string) error {
	return fmt.Errorf("%s: %w", stage, redactedPDFOperationalCause{cause: err, identifiers: identifiers})
}

// reportIdentifiers returns report-owned identifiers that are valid in a
// successful export but prohibited from renderer error channels.
// Authored by: OpenCode
func reportIdentifiers(report reportmodel.CapitalGainsReport) []string {
	var identifiers []string
	for _, entry := range report.SummaryEntries {
		identifiers = append(identifiers, entry.AssetIdentityKey, entry.DisplayLabel)
	}
	for _, entry := range report.ReferenceEntries {
		identifiers = append(identifiers, entry.AssetIdentityKey, entry.DisplayLabel)
	}
	identifiers = appendReportDetailIdentifiers(identifiers, report.DetailSections)
	for _, section := range report.AuditAnnex.PerAssetAuditSections {
		identifiers = append(identifiers, section.AssetIdentityKey, section.DisplayLabel)
		for _, entry := range section.Entries {
			identifiers = append(identifiers, entry.SourceID)
		}
	}
	for _, entry := range report.AuditAnnex.ConversionAuditEntries {
		identifiers = append(identifiers, entry.SourceID, entry.AssetLabel)
		for _, amount := range entry.Amounts {
			identifiers = append(identifiers, amount.SourceID)
		}
	}
	return identifiers
}

// appendReportDetailIdentifiers collects identifiers from detailed activity
// and liquidation rows for renderer error redaction.
// Authored by: OpenCode
func appendReportDetailIdentifiers(identifiers []string, sections []reportmodel.AssetDetailSection) []string {
	for _, section := range sections {
		identifiers = append(identifiers, section.AssetIdentityKey, section.DisplayLabel)
		for _, row := range section.ActivityRows {
			identifiers = append(identifiers, row.SourceID)
		}
		for _, liquidation := range section.LiquidationSummaries {
			identifiers = append(identifiers, liquidation.SourceID)
			for _, match := range liquidation.Matches {
				identifiers = append(identifiers, match.AcquisitionSourceID)
			}
		}
	}
	return identifiers
}

// NewRenderer creates one validated local PDF renderer. A nil ByteFinalizer
// preserves the production GetBytesPdfReturnErr finalization path.
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

// Render validates the calculated report and returns rendered PDF bytes.
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
// Authored by: OpenCode
func (renderer Renderer) Render(report reportmodel.CapitalGainsReport) ([]byte, error) {
	if err := renderer.options.Validate(); err != nil {
		return nil, err
	}
	if err := report.Validate(); err != nil {
		return nil, err
	}

	var identifiers = reportIdentifiers(report)
	var document = newPDFDocumentForRenderer(renderer.options.ByteFinalizer)
	if err := startPDFDocument(document); err != nil {
		return nil, wrapPDFOperationalError("start PDF document", err, identifiers)
	}
	if err := loadApplicationFonts(document, renderer.options.Fonts); err != nil {
		return nil, wrapPDFOperationalError("load PDF fonts", err, identifiers)
	}
	if err := renderMainReportWithFinancialFormatting(document, report, renderer.options.FinancialFormatting); err != nil {
		return nil, wrapPDFOperationalError("render PDF main report", err, identifiers)
	}
	if err := document.AddAnnexPageBreak(); err != nil {
		return nil, wrapPDFOperationalError("add PDF Annex page break", err, identifiers)
	}
	if err := renderAnnexWithFinancialFormatting(document, report.AuditAnnex, renderer.options.FinancialFormatting); err != nil {
		return nil, wrapPDFOperationalError("render PDF Annex", err, identifiers)
	}

	var payload, err = document.Bytes()
	if err != nil {
		return nil, wrapPDFOperationalError("PDF byte finalization failed", err, identifiers)
	}

	return payload, nil
}

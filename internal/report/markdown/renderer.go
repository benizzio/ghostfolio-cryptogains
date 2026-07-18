// Package markdown defines Markdown document rendering for calculated yearly
// gains-and-losses reports.
// Authored by: OpenCode
package markdown

import (
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/report/presentation"
)

const notApplicableCalculationCurrency = "NOT APPLICABLE"

// Test seams keep exported Render wrapper branches directly coverable without
// weakening the validated helper behavior.
// Authored by: OpenCode
var (
	renderWriteSummarySection    = writeSummarySection
	renderWriteRateSourceSummary = writeRateSourceSummary
	renderWriteReferenceSection  = writeReferenceSection
	renderWriteDetailSections    = writeDetailSections
	renderAnnexForDocuments      = RenderAnnex
)

// RenderOptions stores immutable Markdown renderer configuration.
// Authored by: OpenCode
type RenderOptions struct {
	FinancialFormatting presentation.FinancialFormattingOptions
}

// Renderer renders Markdown documents with one renderer-scoped formatting
// policy. A zero-valued policy retains the concrete production defaults.
// Authored by: OpenCode
type Renderer struct {
	options RenderOptions
}

// NewRenderer creates one Markdown renderer with immutable local options.
//
// Example:
//
//	renderer := markdown.NewRenderer(markdown.RenderOptions{})
//	_ = renderer
//
// Authored by: OpenCode
func NewRenderer(options RenderOptions) Renderer {
	return Renderer{options: options}
}

// Render converts one calculated yearly capital-gains report into the Markdown
// document contract used by later output-file writers.
//
// Example:
//
//	document, err := markdown.Render(report)
//	if err != nil {
//		panic(err)
//	}
//	_ = document.Content
//
// Authored by: OpenCode
func Render(report reportmodel.CapitalGainsReport) (reportmodel.ReportDocument, error) {
	if err := report.Validate(); err != nil {
		return reportmodel.ReportDocument{}, err
	}

	var builder strings.Builder
	var calculationCurrency = calculationCurrencyLabel(report.ReportCalculationCurrency)

	writeHeader(&builder, report, calculationCurrency)
	builder.WriteString("**")
	builder.WriteString(presentation.LegalWarningText)
	builder.WriteString("**\n\n")
	if err := renderWriteSummarySection(&builder, report, calculationCurrency); err != nil {
		return reportmodel.ReportDocument{}, err
	}
	if err := renderWriteRateSourceSummary(&builder, report); err != nil {
		return reportmodel.ReportDocument{}, err
	}
	if err := renderWriteReferenceSection(&builder, report); err != nil {
		return reportmodel.ReportDocument{}, err
	}
	if err := renderWriteDetailSections(&builder, report, calculationCurrency); err != nil {
		return reportmodel.ReportDocument{}, err
	}
	return reportmodel.NewReportDocument(
		reportmodel.ReportDocumentTypeMarkdown,
		reportmodel.ReportDocumentRoleMain,
		[]byte(builder.String()),
		report.Year,
		report.CostBasisMethod,
		report.GeneratedAt,
	)
}

// RenderDocuments converts one calculated report into its ordered Markdown
// bundle: the main report followed by a separate Annex 1 document. For example,
// pass its result to `output.WriteReportOutputBundle(model.ReportOutputFormatMarkdown, documents)`.
// Authored by: OpenCode
func RenderDocuments(report reportmodel.CapitalGainsReport) ([]reportmodel.ReportDocument, error) {
	var mainDocument, err = Render(report)
	if err != nil {
		return nil, err
	}

	var annexDocument reportmodel.ReportDocument
	annexDocument, err = renderAnnexForDocuments(report)
	if err != nil {
		return nil, err
	}

	return []reportmodel.ReportDocument{mainDocument, annexDocument}, nil
}

// Render renders one report through this renderer's scoped formatting policy.
// Authored by: OpenCode
func (renderer Renderer) Render(report reportmodel.CapitalGainsReport) (reportmodel.ReportDocument, error) {
	if err := report.Validate(); err != nil {
		return reportmodel.ReportDocument{}, err
	}
	var options = renderer.options.FinancialFormatting
	var builder strings.Builder
	var calculationCurrency = calculationCurrencyLabel(report.ReportCalculationCurrency)
	writeHeader(&builder, report, calculationCurrency)
	builder.WriteString("**")
	builder.WriteString(presentation.LegalWarningText)
	builder.WriteString("**\n\n")
	if err := writeSummarySectionWithFinancialFormatting(&builder, report, calculationCurrency, options); err != nil {
		return reportmodel.ReportDocument{}, err
	}
	writeRateSourceSummary(&builder, report)
	writeReferenceSection(&builder, report)
	if err := writeDetailSectionsWithFinancialFormatting(&builder, report, calculationCurrency, options); err != nil {
		return reportmodel.ReportDocument{}, err
	}
	return reportmodel.NewReportDocument(reportmodel.ReportDocumentTypeMarkdown, reportmodel.ReportDocumentRoleMain, []byte(builder.String()), report.Year, report.CostBasisMethod, report.GeneratedAt)
}

// RenderDocuments renders the Markdown main document and Annex with one
// renderer-scoped formatting policy.
// Authored by: OpenCode
func (renderer Renderer) RenderDocuments(report reportmodel.CapitalGainsReport) ([]reportmodel.ReportDocument, error) {
	var mainDocument, err = renderer.Render(report)
	if err != nil {
		return nil, err
	}
	var annexDocument reportmodel.ReportDocument
	annexDocument, err = RenderAnnexWithFinancialFormatting(report, renderer.options.FinancialFormatting)
	if err != nil {
		return nil, err
	}
	return []reportmodel.ReportDocument{mainDocument, annexDocument}, nil
}

// writeHeader renders the required document heading and metadata block.
// Authored by: OpenCode
func writeHeader(builder *strings.Builder, report reportmodel.CapitalGainsReport, calculationCurrency string) {
	builder.WriteString("# Ghostfolio Capital Gains And Losses Report\n\n")
	fmt.Fprintf(builder, "- **Year:** %d\n", report.Year)
	fmt.Fprintf(builder, "- **Cost Basis Method:** %s\n", report.CostBasisMethod.Label())
	fmt.Fprintf(builder, "- **Generated At:** %s\n", report.GeneratedAt.Local().Format("2006-01-02 15:04:05 MST"))
	fmt.Fprintf(builder, "- **Report Calculation Currency:** %s\n\n", calculationCurrency)
}

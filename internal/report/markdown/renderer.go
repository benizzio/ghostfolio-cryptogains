// Package markdown defines Markdown document rendering for calculated yearly
// gains-and-losses reports.
// Authored by: OpenCode
package markdown

import (
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

const notApplicableCalculationCurrency = "NOT APPLICABLE"

// Test seams keep exported Render wrapper branches directly coverable without
// weakening the validated helper behavior.
// Authored by: OpenCode
var (
	renderWriteSummarySection         = writeSummarySection
	renderWriteRateSourceSummary      = writeRateSourceSummary
	renderWriteReferenceSection       = writeReferenceSection
	renderWriteDetailSections         = writeDetailSections
	renderWriteConversionAuditSection = writeConversionAuditSection
)

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
	if err := renderWriteConversionAuditSection(&builder, report); err != nil {
		return reportmodel.ReportDocument{}, err
	}

	return reportmodel.NewReportDocument(
		reportmodel.ReportDocumentTypeMarkdown,
		reportmodel.ReportDocumentRoleMain,
		builder.String(),
		report.Year,
		report.CostBasisMethod,
		report.GeneratedAt,
	)
}

// RenderDocuments converts one calculated report into the selected Markdown
// output documents: the main report and a separate Annex 1 document.
// Authored by: OpenCode
func RenderDocuments(report reportmodel.CapitalGainsReport) ([]reportmodel.ReportDocument, error) {
	var mainDocument, err = Render(report)
	if err != nil {
		return nil, err
	}

	var annexDocument reportmodel.ReportDocument
	annexDocument, err = RenderAnnex(report)
	if err != nil {
		return nil, err
	}

	return []reportmodel.ReportDocument{mainDocument, annexDocument}, nil
}

// writeHeader renders the required document heading and metadata block.
// Authored by: OpenCode
func writeHeader(builder *strings.Builder, report reportmodel.CapitalGainsReport, calculationCurrency string) {
	builder.WriteString("# Ghostfolio Capital Gains And Losses Report\n\n")
	builder.WriteString(fmt.Sprintf("- Year: %d\n", report.Year))
	builder.WriteString(fmt.Sprintf("- Cost Basis Method: %s\n", report.CostBasisMethod.Label()))
	builder.WriteString(fmt.Sprintf("- Generated At: %s\n", report.GeneratedAt.Local().Format("2006-01-02 15:04:05 MST")))
	builder.WriteString(fmt.Sprintf("- Report Calculation Currency: %s\n\n", calculationCurrency))
}

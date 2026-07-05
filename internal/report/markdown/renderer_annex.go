// Package markdown defines Markdown document rendering for calculated yearly
// gains-and-losses reports.
// Authored by: OpenCode
package markdown

import (
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

// RenderAnnex converts one calculated report's audit annex into the separate
// Annex 1 Markdown document required for Markdown output.
// Authored by: OpenCode
func RenderAnnex(report reportmodel.CapitalGainsReport) (reportmodel.ReportDocument, error) {
	if err := report.Validate(); err != nil {
		return reportmodel.ReportDocument{}, err
	}

	var annex = report.AuditAnnex
	if annex.Title == "" && len(annex.SectionOrder) == 0 {
		annex = reportmodel.DefaultAuditAnnex()
	}
	if err := annex.Validate(); err != nil {
		return reportmodel.ReportDocument{}, err
	}

	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(annex.Title)
	builder.WriteString("\n\n")
	builder.WriteString("## Detailed Per-Asset Audit Report\n\n")
	builder.WriteString("No per-asset audit activity is available for this report.\n\n")
	builder.WriteString("## Currency Conversion Audit\n\n")
	if len(report.ConversionAuditEntries) == 0 {
		builder.WriteString("No converted activity was present for this report.\n")
	} else {
		builder.WriteString(fmt.Sprintf("%d converted activity entries are available in the calculated report.\n", len(report.ConversionAuditEntries)))
	}

	return reportmodel.NewReportDocument(
		reportmodel.ReportDocumentTypeMarkdown,
		reportmodel.ReportDocumentRoleAnnex,
		builder.String(),
		report.Year,
		report.CostBasisMethod,
		report.GeneratedAt,
	)
}

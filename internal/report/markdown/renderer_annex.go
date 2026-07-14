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

// Annex section writer seams keep defensive render failures testable after model
// validation has accepted normal report inputs.
// Authored by: OpenCode
var (
	writeAnnexPerAssetAuditForRender   = writeAnnexPerAssetAudit
	writeAnnexConversionAuditForRender = writeAnnexConversionAudit
)

// RenderAnnex converts one calculated report's audit annex into the separate
// Annex 1 Markdown document required for Markdown output. For example, call
// `annexDocument, err := markdown.RenderAnnex(report)` after report calculation.
// Authored by: OpenCode
func RenderAnnex(report reportmodel.CapitalGainsReport) (reportmodel.ReportDocument, error) {
	if err := report.Validate(); err != nil {
		return reportmodel.ReportDocument{}, err
	}

	var annex = report.AuditAnnex

	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(annex.Title)
	builder.WriteString("\n\n")
	if err := writeAnnexPerAssetAuditForRender(&builder, annex); err != nil {
		return reportmodel.ReportDocument{}, err
	}
	if err := writeAnnexConversionAuditForRender(&builder, annex); err != nil {
		return reportmodel.ReportDocument{}, err
	}

	return reportmodel.NewReportDocument(
		reportmodel.ReportDocumentTypeMarkdown,
		reportmodel.ReportDocumentRoleAnnex,
		[]byte(builder.String()),
		report.Year,
		report.CostBasisMethod,
		report.GeneratedAt,
	)
}

// writeAnnexPerAssetAudit renders the Detailed Per-Asset Audit Report section.
// Authored by: OpenCode
func writeAnnexPerAssetAudit(builder *strings.Builder, annex reportmodel.AuditAnnex) error {
	builder.WriteString("## Detailed Per-Asset Audit Report\n\n")
	if len(annex.PerAssetAuditSections) == 0 {
		builder.WriteString("No per-asset audit activity is available for this report.\n\n")
		return nil
	}

	for _, section := range annex.PerAssetAuditSections {
		fmt.Fprintf(builder, "### Asset: %s\n\n", renderDisplayLabel(section.DisplayLabel, section.AssetIdentityKey))
		builder.WriteString("| Date/Time | Source ID | Activity Type | Quantity | Unit Price | Gross Value | Fee | Original Activity Currency | Calculation Currency | Quantity After Activity | Basis After Activity | Full Liquidation Event | Allocated Basis | Net Liquidation Proceeds | Gain/Loss | Conversion Status | Sanitized Note |\n")
		builder.WriteString("|-----------|-----------|---------------|----------|------------|-------------|-----|----------------------------|----------------------|-------------------------|----------------------|------------------------|-----------------|--------------------------|-----------|-------------------|----------------|\n")
		for _, entry := range section.Entries {
			if err := writeAnnexActivityEntry(builder, entry); err != nil {
				return fmt.Errorf("render annex audit entry %q: %w", entry.SourceID, err)
			}
		}
		builder.WriteString("\n")
	}

	return nil
}

// writeAnnexActivityEntry renders one detailed audit activity row.
// Authored by: OpenCode
func writeAnnexActivityEntry(builder *strings.Builder, entry reportmodel.AuditActivityEntry) error {
	var rendered, err = presentation.BuildAnnexActivityRow(entry)
	if err != nil {
		return err
	}

	fmt.Fprintf(builder,
		"| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %t | %s | %s | %s | %s | %s |\n",
		rendered.Date, sanitizeInlineText(rendered.SourceID), sanitizeInlineText(rendered.ActivityType), rendered.Quantity, rendered.UnitPrice, rendered.GrossValue, rendered.Fee, sanitizeInlineText(rendered.ActivityCurrency), sanitizeInlineText(rendered.CalculationCurrency), rendered.QuantityAfter, rendered.BasisAfter, rendered.FullLiquidationEvent == "true", rendered.AllocatedBasis, rendered.NetProceeds, rendered.GainOrLoss, sanitizeInlineText(rendered.ConversionStatus), sanitizeInlineText(rendered.Note),
	)
	return nil
}

// writeAnnexConversionAudit renders the Annex 1 currency conversion audit.
// Authored by: OpenCode
func writeAnnexConversionAudit(builder *strings.Builder, annex reportmodel.AuditAnnex) error {
	builder.WriteString("## Currency Conversion Audit\n\n")
	if len(annex.ConversionAuditEntries) == 0 {
		builder.WriteString("No converted activity was present for this report.\n")
		return nil
	}

	builder.WriteString("| Date | Source ID | Asset | Rate Date | Source Currency | Report Base Currency | Converted Amounts | Quote Direction | Rate Value |\n")
	builder.WriteString("|------|-----------|-------|-----------|-----------------|----------------------|-------------------|-----------------|------------|\n")
	for index, entry := range annex.ConversionAuditEntries {
		if err := writeConversionAuditRow(builder, index, entry); err != nil {
			return err
		}
	}
	builder.WriteString("\n")
	return nil
}

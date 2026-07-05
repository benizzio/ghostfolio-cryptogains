// Package markdown defines Markdown document rendering for calculated yearly
// gains-and-losses reports.
// Authored by: OpenCode
package markdown

import (
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
)

// Annex section writer seams keep defensive render failures testable after model
// validation has accepted normal report inputs.
// Authored by: OpenCode
var (
	writeAnnexPerAssetAuditForRender   = writeAnnexPerAssetAudit
	writeAnnexConversionAuditForRender = writeAnnexConversionAudit
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
		builder.String(),
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
	var quantity, err = decimalsupport.CanonicalString(entry.Quantity)
	if err != nil {
		return fmt.Errorf("quantity: %w", err)
	}
	var unitPrice string
	unitPrice, err = decimalsupport.CanonicalStringPointer(entry.UnitPrice)
	if err != nil {
		return fmt.Errorf("unit price: %w", err)
	}
	var grossValue string
	grossValue, err = decimalsupport.CanonicalStringPointer(entry.GrossValue)
	if err != nil {
		return fmt.Errorf("gross value: %w", err)
	}
	var fee string
	fee, err = decimalsupport.CanonicalStringPointer(entry.FeeAmount)
	if err != nil {
		return fmt.Errorf("fee: %w", err)
	}
	var quantityAfter string
	quantityAfter, err = decimalsupport.CanonicalString(entry.QuantityAfterActivity)
	if err != nil {
		return fmt.Errorf("quantity after activity: %w", err)
	}
	var basisAfter string
	basisAfter, err = decimalsupport.CanonicalString(entry.BasisAfterActivity)
	if err != nil {
		return fmt.Errorf("basis after activity: %w", err)
	}
	var allocatedBasis string
	allocatedBasis, err = decimalsupport.CanonicalStringPointer(entry.AllocatedBasis)
	if err != nil {
		return fmt.Errorf("allocated basis: %w", err)
	}
	var proceeds string
	proceeds, err = decimalsupport.CanonicalStringPointer(entry.NetLiquidationProceeds)
	if err != nil {
		return fmt.Errorf("net liquidation proceeds: %w", err)
	}
	var gainOrLoss string
	gainOrLoss, err = decimalsupport.CanonicalStringPointer(entry.GainOrLoss)
	if err != nil {
		return fmt.Errorf("gain or loss: %w", err)
	}
	var activityTypeLabel string
	activityTypeLabel, err = reportmodel.RenderAuditActivityTypeLabel(entry)
	if err != nil {
		return fmt.Errorf("activity type label: %w", err)
	}
	var conversionStatus string
	if strings.TrimSpace(string(entry.ConversionStatus)) != "" {
		conversionStatus, err = reportmodel.RenderConversionStatusLabel(entry.ConversionStatus)
		if err != nil {
			return fmt.Errorf("conversion status label: %w", err)
		}
	}

	fmt.Fprintf(builder,
		"| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %t | %s | %s | %s | %s | %s |\n",
		entry.OccurredAt.UTC().Format("2006-01-02 15:04:05"),
		sanitizeInlineText(entry.SourceID),
		sanitizeInlineText(activityTypeLabel),
		quantity,
		unitPrice,
		grossValue,
		fee,
		sanitizeInlineText(entry.ActivityCurrency),
		sanitizeInlineText(entry.CalculationCurrency),
		quantityAfter,
		basisAfter,
		entry.FullLiquidationEvent,
		allocatedBasis,
		proceeds,
		gainOrLoss,
		sanitizeInlineText(conversionStatus),
		sanitizeInlineText(entry.Note),
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

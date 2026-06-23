// Package markdown defines Markdown document rendering for calculated yearly
// gains-and-losses reports.
// Authored by: OpenCode
package markdown

import (
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/benizzio/ghostfolio-cryptogains/internal/support/redact"
	"github.com/cockroachdb/apd/v3"
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
		builder.String(),
		report.Year,
		report.CostBasisMethod,
		report.GeneratedAt,
	)
}

// writeRateSourceSummary renders the official rate-provider summary disclosed
// by the report model.
// Authored by: OpenCode
func writeRateSourceSummary(builder *strings.Builder, report reportmodel.CapitalGainsReport) error {
	builder.WriteString("## Rate Source Summary\n\n")
	builder.WriteString(fmt.Sprintf("- Report Base Currency: %s\n", calculationCurrencyLabel(report.ReportCalculationCurrency)))
	if len(report.RateSources) == 0 {
		builder.WriteString("- Exchange Rate Use: No activity required exchange-rate conversion.\n\n")
		return nil
	}

	var rendered = make(map[string]bool)
	for _, source := range report.RateSources {
		var key = strings.Join([]string{
			string(source.Authority),
			string(source.ProviderID),
			source.RateKind,
		}, "|")
		if rendered[key] {
			continue
		}
		rendered[key] = true

		builder.WriteString(fmt.Sprintf("- Authority: %s\n", rateAuthorityLabel(source.Authority)))
		builder.WriteString(fmt.Sprintf("- Provider: %s\n", rateProviderLabel(source.ProviderID)))
		builder.WriteString(fmt.Sprintf("- Rate Kind: %s\n", sanitizeInlineText(source.RateKind)))
		builder.WriteString(fmt.Sprintf("- Unavailable-Date Rule: %s\n", unavailableDateRule(source.ProviderID)))
	}

	builder.WriteString("\n")
	return nil
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

// writeSummarySection renders the summary heading, optional empty state, and
// yearly summary table.
// Authored by: OpenCode
func writeSummarySection(builder *strings.Builder, report reportmodel.CapitalGainsReport, calculationCurrency string) error {
	builder.WriteString("## Gains-And-Losses Summary\n\n")
	if len(report.SummaryEntries) == 0 {
		builder.WriteString("No assets qualified for the main report sections in the selected year.\n\n")
	}

	builder.WriteString("| Asset | Net Gain Or Loss | Report Calculation Currency |\n")
	builder.WriteString("|-------|------------------|-----------------------------|\n")

	for _, entry := range report.SummaryEntries {
		var netGainOrLoss, err = canonicalDecimal(entry.NetGainOrLoss)
		if err != nil {
			return fmt.Errorf("render summary entry %q net gain or loss: %w", entry.AssetIdentityKey, err)
		}

		builder.WriteString(fmt.Sprintf(
			"| %s | %s | %s |\n",
			renderDisplayLabel(entry.DisplayLabel, entry.AssetIdentityKey),
			netGainOrLoss,
			calculationCurrencyLabelWithFallback(entry.ReportCalculationCurrency, calculationCurrency),
		))
	}

	var yearlyNetTotal, err = canonicalDecimal(report.YearlyNetTotal)
	if err != nil {
		return fmt.Errorf("render yearly net total: %w", err)
	}

	builder.WriteString(fmt.Sprintf("| Overall Yearly Net Total | %s | %s |\n\n", yearlyNetTotal, calculationCurrency))
	return nil
}

// writeReferenceSection renders the reference-section heading and either the
// reference table or its empty-state sentence.
// Authored by: OpenCode
func writeReferenceSection(builder *strings.Builder, report reportmodel.CapitalGainsReport) error {
	builder.WriteString("## Reference Section\n\n")
	if len(report.ReferenceEntries) == 0 {
		builder.WriteString("No assets reached full liquidation by year end.\n\n")
		return nil
	}

	builder.WriteString("| Asset | Full Liquidation Count Through Year End | Main Section Status |\n")
	builder.WriteString("|-------|-----------------------------------------|---------------------|\n")

	for _, entry := range report.ReferenceEntries {
		builder.WriteString(fmt.Sprintf(
			"| %s | %d | %s |\n",
			renderDisplayLabel(entry.DisplayLabel, entry.AssetIdentityKey),
			entry.FullLiquidationCountThroughYearEnd,
			sanitizeInlineText(string(entry.MainSectionStatus)),
		))
	}

	builder.WriteString("\n")
	return nil
}

// writeDetailSections renders each per-asset detail section in report order.
// Authored by: OpenCode
func writeDetailSections(builder *strings.Builder, report reportmodel.CapitalGainsReport, calculationCurrency string) error {
	for _, section := range report.DetailSections {
		if err := writeDetailSection(builder, section, calculationCurrency); err != nil {
			return err
		}
	}

	return nil
}

// writeDetailSection renders one per-asset detail section.
// Authored by: OpenCode
func writeDetailSection(builder *strings.Builder, section reportmodel.AssetDetailSection, calculationCurrency string) error {
	builder.WriteString(fmt.Sprintf("## Asset Detail: %s\n\n", renderDisplayLabel(section.DisplayLabel, section.AssetIdentityKey)))

	if err := writePositionBlock(builder, "Opening Position", section.OpeningQuantity, section.OpeningCostBasis, section.CalculationCurrency, calculationCurrency); err != nil {
		return fmt.Errorf("render opening position for %q: %w", section.AssetIdentityKey, err)
	}
	if err := writeActivityBlock(builder, section); err != nil {
		return fmt.Errorf("render in-year activity for %q: %w", section.AssetIdentityKey, err)
	}
	if err := writeLiquidationBlock(builder, section, calculationCurrency); err != nil {
		return fmt.Errorf("render liquidation calculations for %q: %w", section.AssetIdentityKey, err)
	}
	if err := writePositionBlock(builder, "Closing Position", section.ClosingQuantity, section.ClosingCostBasis, section.CalculationCurrency, calculationCurrency); err != nil {
		return fmt.Errorf("render closing position for %q: %w", section.AssetIdentityKey, err)
	}

	return nil
}

// writePositionBlock renders one opening or closing position bullet block.
// Authored by: OpenCode
func writePositionBlock(builder *strings.Builder, heading string, quantity apd.Decimal, basis apd.Decimal, sectionCurrency string, fallbackCurrency string) error {
	var quantityText, err = canonicalDecimal(quantity)
	if err != nil {
		return fmt.Errorf("render quantity: %w", err)
	}
	var basisText string
	basisText, err = canonicalDecimal(basis)
	if err != nil {
		return fmt.Errorf("render cost basis: %w", err)
	}

	builder.WriteString(fmt.Sprintf("### %s\n\n", heading))
	builder.WriteString(fmt.Sprintf("- Quantity: %s\n", quantityText))
	builder.WriteString(fmt.Sprintf("- Cost Basis: %s\n", basisText))
	builder.WriteString(fmt.Sprintf("- Calculation Currency: %s\n\n", calculationCurrencyLabelWithFallback(sectionCurrency, fallbackCurrency)))
	return nil
}

// writeActivityBlock renders the in-year activity table or its no-activity
// sentence.
// Authored by: OpenCode
func writeActivityBlock(builder *strings.Builder, section reportmodel.AssetDetailSection) error {
	builder.WriteString("### In-Year Activity\n\n")
	if len(section.ActivityRows) == 0 {
		builder.WriteString("No in-year activity for the selected year.\n\n")
		return nil
	}

	builder.WriteString("| Date | Source ID | Type | Quantity | Unit Price | Gross Value | Fee | Activity Currency | Basis After Row | Calculation Currency | Quantity After Row | Conversion Status | Note |\n")
	builder.WriteString("|------|-----------|------|----------|------------|-------------|-----|-------------------|-----------------|----------------------|--------------------|-------------------|------|\n")

	for _, row := range section.ActivityRows {
		var quantityText, err = canonicalDecimal(row.Quantity)
		if err != nil {
			return fmt.Errorf("render activity row %q quantity: %w", row.SourceID, err)
		}
		var unitPriceText string
		unitPriceText, err = canonicalDecimalPointer(row.UnitPrice)
		if err != nil {
			return fmt.Errorf("render activity row %q unit price: %w", row.SourceID, err)
		}
		var grossValueText string
		grossValueText, err = canonicalDecimalPointer(row.GrossValue)
		if err != nil {
			return fmt.Errorf("render activity row %q gross value: %w", row.SourceID, err)
		}
		var feeText string
		feeText, err = canonicalDecimalPointer(row.FeeAmount)
		if err != nil {
			return fmt.Errorf("render activity row %q fee: %w", row.SourceID, err)
		}
		var basisAfterRowText string
		basisAfterRowText, err = canonicalDecimal(row.BasisAfterRow)
		if err != nil {
			return fmt.Errorf("render activity row %q basis after row: %w", row.SourceID, err)
		}
		var quantityAfterRowText string
		quantityAfterRowText, err = canonicalDecimal(row.QuantityAfterRow)
		if err != nil {
			return fmt.Errorf("render activity row %q quantity after row: %w", row.SourceID, err)
		}

		builder.WriteString(fmt.Sprintf(
			"| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			row.OccurredAt.Local().Format("2006-01-02 15:04:05"),
			sanitizeInlineText(row.SourceID),
			sanitizeInlineText(string(row.ActivityType)),
			quantityText,
			unitPriceText,
			grossValueText,
			feeText,
			activityCurrencyColumn(row),
			basisAfterRowText,
			calculationCurrencyLabel(row.CalculationCurrency),
			quantityAfterRowText,
			conversionStatusColumn(row),
			sanitizeInlineText(row.HoldingReductionExplanation),
		))
	}

	builder.WriteString("\n")
	return nil
}

// writeConversionAuditSection renders one audit row per converted monetary
// amount disclosed by the report model.
// Authored by: OpenCode
func writeConversionAuditSection(builder *strings.Builder, report reportmodel.CapitalGainsReport) error {
	if len(report.ConversionAuditEntries) == 0 {
		return nil
	}

	builder.WriteString("## Currency Conversion Audit\n\n")
	builder.WriteString("| Date | Source ID | Asset | Amount Kind | Rate Date | Source Currency | Original Amount | Report Base Currency | Converted Amount | Quote Direction | Rate Value |\n")
	builder.WriteString("|------|-----------|-------|-------------|-----------|-----------------|-----------------|----------------------|------------------|-----------------|------------|\n")

	for entryIndex, entry := range report.ConversionAuditEntries {
		var rateValue, err = canonicalDecimal(entry.RateValue)
		if err != nil {
			return fmt.Errorf("render conversion audit entry %d rate value: %w", entryIndex, err)
		}

		for amountIndex, amount := range entry.Amounts {
			var originalAmount string
			originalAmount, err = canonicalDecimal(amount.OriginalAmount)
			if err != nil {
				return fmt.Errorf("render conversion audit entry %d amount %d original amount: %w", entryIndex, amountIndex, err)
			}
			var convertedAmount string
			convertedAmount, err = canonicalDecimal(amount.ConvertedAmount)
			if err != nil {
				return fmt.Errorf("render conversion audit entry %d amount %d converted amount: %w", entryIndex, amountIndex, err)
			}

			builder.WriteString(fmt.Sprintf(
				"| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
				entry.ActivityDate.Local().Format("2006-01-02"),
				sanitizeInlineText(entry.SourceID),
				sanitizeInlineText(entry.AssetLabel),
				sanitizeInlineText(string(amount.AmountKind)),
				entry.RateDate.Local().Format("2006-01-02"),
				sanitizeInlineText(entry.SourceCurrency),
				originalAmount,
				sanitizeInlineText(entry.ReportBaseCurrency.Label()),
				convertedAmount,
				sanitizeInlineText(string(entry.QuoteDirection)),
				rateValue,
			))
		}
	}

	builder.WriteString("\n")
	return nil
}

// conversionStatusColumn classifies rendered priced activity rows without
// exposing provider implementation details.
// Authored by: OpenCode
func conversionStatusColumn(row reportmodel.AssetActivityRow) string {
	if activityCurrencyColumn(row) == "" {
		return ""
	}
	if strings.TrimSpace(row.ActivityCurrency) == strings.TrimSpace(row.CalculationCurrency) {
		return sanitizeInlineText(string(reportmodel.ConversionStatusSameCurrency))
	}

	return sanitizeInlineText(string(reportmodel.ConversionStatusConverted))
}

// rateAuthorityLabel returns report-facing authority labels for canonical rate
// evidence.
// Authored by: OpenCode
func rateAuthorityLabel(authority reportmodel.ExchangeRateAuthority) string {
	switch authority {
	case reportmodel.ExchangeRateAuthorityEuropeanCentralBank:
		return "European Central Bank"
	case reportmodel.ExchangeRateAuthorityFederalReserve:
		return "Federal Reserve"
	default:
		return sanitizeInlineText(string(authority))
	}
}

// rateProviderLabel returns report-facing provider labels for canonical rate
// evidence.
// Authored by: OpenCode
func rateProviderLabel(provider reportmodel.ExchangeRateProviderID) string {
	switch provider {
	case reportmodel.ExchangeRateProviderIDECBEXR:
		return "ECB Data Portal `EXR`"
	case reportmodel.ExchangeRateProviderIDFederalReserveH10:
		return "Federal Reserve Board H.10/Data Download Program"
	default:
		return sanitizeInlineText(string(provider))
	}
}

// unavailableDateRule returns the report-facing prior-observation rule for one
// canonical provider.
// Authored by: OpenCode
func unavailableDateRule(provider reportmodel.ExchangeRateProviderID) string {
	switch provider {
	case reportmodel.ExchangeRateProviderIDECBEXR:
		return "most recent previous available ECB observation"
	case reportmodel.ExchangeRateProviderIDFederalReserveH10:
		return "most recent previous available H.10 observation"
	default:
		return "most recent previous available official observation"
	}
}

// writeLiquidationBlock renders the priced liquidation table when the section
// contains priced in-year liquidations.
// Authored by: OpenCode
func writeLiquidationBlock(builder *strings.Builder, section reportmodel.AssetDetailSection, fallbackCurrency string) error {
	if len(section.LiquidationSummaries) == 0 {
		return nil
	}

	builder.WriteString("### Liquidation Calculations\n\n")
	builder.WriteString("| Date | Source ID | Disposed Quantity | Activity Currency | Allocated Basis | Net Liquidation Proceeds | Gain Or Loss | Calculation Currency |\n")
	builder.WriteString("|------|-----------|-------------------|-------------------|-----------------|--------------------------|--------------|----------------------|\n")

	for _, liquidation := range section.LiquidationSummaries {
		var disposedQuantityText, err = canonicalDecimal(liquidation.DisposedQuantity)
		if err != nil {
			return fmt.Errorf("render liquidation %q disposed quantity: %w", liquidation.SourceID, err)
		}
		var allocatedBasisText string
		allocatedBasisText, err = canonicalDecimal(liquidation.AllocatedBasis)
		if err != nil {
			return fmt.Errorf("render liquidation %q allocated basis: %w", liquidation.SourceID, err)
		}
		var proceedsText string
		proceedsText, err = canonicalDecimal(liquidation.NetLiquidationProceeds)
		if err != nil {
			return fmt.Errorf("render liquidation %q net proceeds: %w", liquidation.SourceID, err)
		}
		var gainOrLossText string
		gainOrLossText, err = canonicalDecimal(liquidation.GainOrLoss)
		if err != nil {
			return fmt.Errorf("render liquidation %q gain or loss: %w", liquidation.SourceID, err)
		}

		builder.WriteString(fmt.Sprintf(
			"| %s | %s | %s | %s | %s | %s | %s | %s |\n",
			liquidation.OccurredAt.Local().Format("2006-01-02 15:04:05"),
			sanitizeInlineText(liquidation.SourceID),
			disposedQuantityText,
			sanitizeInlineText(liquidation.ActivityCurrency),
			allocatedBasisText,
			proceedsText,
			gainOrLossText,
			calculationCurrencyLabelWithFallback(liquidation.CalculationCurrency, fallbackCurrency),
		))
	}

	builder.WriteString("\n")
	return nil
}

// canonicalDecimal renders one exact decimal in canonical fixed-point form.
// Authored by: OpenCode
func canonicalDecimal(value apd.Decimal) (string, error) {
	return decimalsupport.CanonicalString(value)
}

// canonicalDecimalPointer renders one optional exact decimal in canonical
// fixed-point form.
// Authored by: OpenCode
func canonicalDecimalPointer(value *apd.Decimal) (string, error) {
	return decimalsupport.CanonicalStringPointer(value)
}

// calculationCurrencyLabel normalizes the report calculation-currency label.
// Authored by: OpenCode
func calculationCurrencyLabel(raw string) string {
	var normalized = sanitizeInlineText(raw)
	if normalized == "" {
		return notApplicableCalculationCurrency
	}
	return normalized
}

// calculationCurrencyLabelWithFallback returns the normalized explicit label or
// falls back to the report-wide calculation currency.
// Authored by: OpenCode
func calculationCurrencyLabelWithFallback(raw string, fallback string) string {
	var normalized = sanitizeInlineText(raw)
	if normalized == "" {
		return calculationCurrencyLabel(fallback)
	}
	return normalized
}

// renderDisplayLabel returns the safe display label for one asset row or
// section heading.
// Authored by: OpenCode
func renderDisplayLabel(displayLabel string, assetIdentityKey string) string {
	var normalized = sanitizeInlineText(displayLabel)
	if normalized != "" {
		return normalized
	}

	normalized = sanitizeInlineText(assetIdentityKey)
	if normalized != "" {
		return normalized
	}

	return "Unknown Asset"
}

// activityCurrencyColumn renders the activity-currency table cell and leaves it
// blank for rows without one selected activity monetary context, including
// zero-priced holding reductions that still preserve explicit zero-valued source
// details.
// Authored by: OpenCode
func activityCurrencyColumn(row reportmodel.AssetActivityRow) string {
	if strings.TrimSpace(row.ActivityCurrency) == "" {
		return ""
	}
	if row.GrossValue == nil && row.FeeAmount == nil && row.UnitPrice == nil {
		return ""
	}

	return sanitizeInlineText(row.ActivityCurrency)
}

// sanitizeInlineText redacts obvious secret-shaped fragments and normalizes one
// line of text for safe Markdown output.
// Authored by: OpenCode
func sanitizeInlineText(raw string) string {
	var sanitized = redact.Text(raw)
	sanitized = strings.ReplaceAll(sanitized, "\r", " ")
	sanitized = strings.ReplaceAll(sanitized, "\n", " ")
	sanitized = strings.ReplaceAll(sanitized, "\t", " ")
	sanitized = strings.Join(strings.Fields(strings.TrimSpace(sanitized)), " ")
	sanitized = strings.ReplaceAll(sanitized, "|", "\\|")
	return sanitized
}

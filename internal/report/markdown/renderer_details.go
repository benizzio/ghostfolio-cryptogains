// Package markdown defines Markdown document rendering for calculated yearly
// gains-and-losses reports.
// Authored by: OpenCode
package markdown

import (
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/cockroachdb/apd/v3"
)

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
		if err := writeActivityRow(builder, row); err != nil {
			return err
		}
	}

	builder.WriteString("\n")
	return nil
}

// writeActivityRow renders one priced or explanatory activity row.
// Authored by: OpenCode
func writeActivityRow(builder *strings.Builder, row reportmodel.AssetActivityRow) error {
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
	return nil
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
		if err := writeLiquidationRow(builder, liquidation, fallbackCurrency); err != nil {
			return err
		}
	}

	builder.WriteString("\n")
	return nil
}

// writeLiquidationRow renders one liquidation calculation row.
// Authored by: OpenCode
func writeLiquidationRow(builder *strings.Builder, liquidation reportmodel.LiquidationCalculation, fallbackCurrency string) error {
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
	return nil
}

// conversionStatusColumn classifies rendered priced activity rows without
// exposing provider implementation details.
// Authored by: OpenCode
func conversionStatusColumn(row reportmodel.AssetActivityRow) string {
	if activityCurrencyColumn(row) == "" {
		return ""
	}
	if strings.TrimSpace(string(row.ConversionStatus)) != "" {
		return sanitizeInlineText(string(row.ConversionStatus))
	}
	if strings.TrimSpace(row.ActivityCurrency) == strings.TrimSpace(row.CalculationCurrency) {
		return sanitizeInlineText(string(reportmodel.ConversionStatusSameCurrency))
	}

	return sanitizeInlineText(string(reportmodel.ConversionStatusConverted))
}

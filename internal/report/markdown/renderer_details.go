// Package markdown defines Markdown document rendering for calculated yearly
// gains-and-losses reports.
// Authored by: OpenCode
package markdown

import (
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/report/presentation"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
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
	fmt.Fprintf(builder, "## Asset Detail: %s\n\n", renderDisplayLabel(section.DisplayLabel, section.AssetIdentityKey))
	if len(section.ActivityRows) == 0 {
		if err := writePositionBlock(builder, "Historical Position", section.ClosingQuantity, section.ClosingCostBasis, section.CalculationCurrency, calculationCurrency); err != nil {
			return fmt.Errorf("render historical position for %q: %w", section.AssetIdentityKey, err)
		}

		return nil
	}
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
	var quantityText, err = decimalsupport.CanonicalString(quantity)
	if err != nil {
		return fmt.Errorf("render quantity: %w", err)
	}
	var basisText string
	basisText, err = presentation.FormatFinancialValue(basis)
	if err != nil {
		return fmt.Errorf("render cost basis: %w", err)
	}

	fmt.Fprintf(builder, "### %s\n\n", heading)
	fmt.Fprintf(builder, "- **Quantity:** %s\n", quantityText)
	fmt.Fprintf(builder, "- **Cost Basis:** %s\n", basisText)
	fmt.Fprintf(builder, "- **Calculation Currency:** %s\n\n", calculationCurrencyLabelWithFallback(sectionCurrency, fallbackCurrency))
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

	builder.WriteString("| Date | Source ID | Type | Quantity | Unit Price | Gross Value | Fee | Quantity After Row | Basis After Row | Original Activity Currency | Calculation Currency | Conversion Status | Note |\n")
	builder.WriteString("|------|-----------|------|----------|------------|-------------|-----|--------------------|-----------------|----------------------------|----------------------|-------------------|------|\n")
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
	var rendered, err = presentation.BuildActivityRow(row)
	if err != nil {
		return err
	}

	fmt.Fprintf(builder,
		"| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
		rendered.Date, sanitizeInlineText(rendered.SourceID), sanitizeInlineText(rendered.ActivityType), rendered.Quantity, rendered.UnitPrice, rendered.GrossValue, rendered.Fee, rendered.QuantityAfterRow, rendered.BasisAfterRow, sanitizeInlineText(rendered.ActivityCurrency), sanitizeInlineText(rendered.CalculationCurrency), sanitizeInlineText(rendered.ConversionStatus), sanitizeInlineText(rendered.Note),
	)
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
	builder.WriteString("| Date | Source ID | Disposed Quantity | Allocated Basis | Net Liquidation Proceeds | Gain Or Loss | Calculation Currency |\n")
	builder.WriteString("|------|-----------|-------------------|-----------------|--------------------------|--------------|----------------------|\n")
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
	var rendered, err = presentation.BuildLiquidationRow(liquidation, fallbackCurrency)
	if err != nil {
		return err
	}

	fmt.Fprintf(builder,
		"| %s | %s | %s | %s | %s | %s | %s |\n",
		rendered.Date, sanitizeInlineText(rendered.SourceID), rendered.DisposedQuantity, rendered.AllocatedBasis, rendered.NetProceeds, rendered.GainOrLoss, sanitizeInlineText(rendered.CalculationCurrency),
	)
	return nil
}

// conversionStatusColumn exposes the shared conversion-status derivation to
// existing Markdown renderer tests while preserving Markdown-specific escaping.
// Authored by: OpenCode
func conversionStatusColumn(row reportmodel.AssetActivityRow) (string, error) {
	var label, err = presentation.ActivityConversionStatus(row)
	if err != nil {
		return "", err
	}
	return sanitizeInlineText(label), nil
}

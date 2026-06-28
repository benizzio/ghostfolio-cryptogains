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
		var netGainOrLoss, err = decimalsupport.CanonicalString(entry.NetGainOrLoss)
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

	var yearlyNetTotal, err = decimalsupport.CanonicalString(report.YearlyNetTotal)
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

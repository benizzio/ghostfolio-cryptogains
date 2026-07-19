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

// writeSummarySection renders the summary heading, optional empty state, and
// yearly summary table.
// Authored by: OpenCode
func writeSummarySection(builder *strings.Builder, report reportmodel.CapitalGainsReport, calculationCurrency string) error {
	return writeSummarySectionWithFinancialFormatting(builder, report, calculationCurrency, presentation.DefaultFinancialFormattingOptions())
}

// writeSummarySectionWithFinancialFormatting renders summary monetary values
// with one renderer-scoped immutable policy.
// Authored by: OpenCode
func writeSummarySectionWithFinancialFormatting(builder *strings.Builder, report reportmodel.CapitalGainsReport, calculationCurrency string, options presentation.FinancialFormattingOptions) error {
	builder.WriteString("## Gains-And-Losses Summary\n\n")
	var renderedEntries []struct {
		entry         reportmodel.AssetSummaryEntry
		netGainOrLoss string
	}
	for index, entry := range report.SummaryEntries {
		var netGainOrLoss, err = options.Format(entry.NetGainOrLoss)
		if err != nil {
			return fmt.Errorf("render summary entry %d net gain or loss: %w", index+1, err)
		}
		if entry.NetGainOrLoss.Sign() == 0 {
			continue
		}
		renderedEntries = append(renderedEntries, struct {
			entry         reportmodel.AssetSummaryEntry
			netGainOrLoss string
		}{entry: entry, netGainOrLoss: netGainOrLoss})
	}
	if len(renderedEntries) == 0 {
		builder.WriteString("No assets had a non-zero net gain or loss in the selected year.\n\n")
	}

	builder.WriteString("| Asset | Net Gain Or Loss | Report Calculation Currency |\n")
	builder.WriteString("|-------|------------------|-----------------------------|\n")
	for _, renderedEntry := range renderedEntries {
		var entry = renderedEntry.entry
		fmt.Fprintf(builder,
			"| %s | %s | %s |\n",
			renderDisplayLabel(entry.DisplayLabel, entry.AssetIdentityKey),
			renderedEntry.netGainOrLoss,
			calculationCurrencyLabelWithFallback(entry.ReportCalculationCurrency, calculationCurrency),
		)
	}

	var yearlyNetTotal, err = options.Format(report.YearlyNetTotal)
	if err != nil {
		return fmt.Errorf("render yearly net total: %w", err)
	}

	fmt.Fprintf(builder, "| Overall Yearly Net Total | %s | %s |\n\n", yearlyNetTotal, calculationCurrency)
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

	builder.WriteString("| Asset | Historical Full Liquidation Count | Main Section Status |\n")
	builder.WriteString("|-------|-----------------------------------|---------------------|\n")
	for _, entry := range report.ReferenceEntries {
		fmt.Fprintf(builder,
			"| %s | %d | %s |\n",
			renderDisplayLabel(entry.DisplayLabel, entry.AssetIdentityKey),
			entry.FullLiquidationCountThroughYearEnd,
			sanitizeInlineText(string(entry.MainSectionStatus)),
		)
	}

	builder.WriteString("\n")
	return nil
}

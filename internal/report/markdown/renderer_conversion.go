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

// writeRateSourceSummary renders the official rate-provider summary disclosed
// by the report model.
// Authored by: OpenCode
func writeRateSourceSummary(builder *strings.Builder, report reportmodel.CapitalGainsReport) error {
	builder.WriteString("## Rate Source Summary\n\n")
	fmt.Fprintf(builder, "- **Report Base Currency:** %s\n", calculationCurrencyLabel(report.ReportCalculationCurrency))
	if len(report.RateSources) == 0 {
		builder.WriteString("- **Exchange Rate Use:** No activity required exchange-rate conversion.\n\n")
		return nil
	}

	var rendered = make(map[string]bool)
	for _, source := range report.RateSources {
		var key = strings.Join([]string{string(source.Authority), string(source.ProviderID), source.RateKind}, "|")
		if rendered[key] {
			continue
		}
		rendered[key] = true
		fmt.Fprintf(builder, "- **Authority:** %s\n", rateAuthorityLabel(source.Authority))
		fmt.Fprintf(builder, "- **Provider:** %s\n", rateProviderLabel(source.ProviderID))
		fmt.Fprintf(builder, "- **Rate Kind:** %s\n", sanitizeInlineText(source.RateKind))
		fmt.Fprintf(builder, "- **Unavailable-Date Rule:** %s\n", unavailableDateRule(source.ProviderID))
	}

	builder.WriteString("\n")
	return nil
}

// writeConversionAuditSection renders one grouped audit row per converted source
// activity disclosed by the report model.
// Authored by: OpenCode
func writeConversionAuditSection(builder *strings.Builder, report reportmodel.CapitalGainsReport) error {
	if len(report.AuditAnnex.ConversionAuditEntries) == 0 {
		return nil
	}

	builder.WriteString("## Currency Conversion Audit\n\n")
	builder.WriteString("| Date | Source ID | Asset | Rate Date | Source Currency | Report Base Currency | Converted Amounts | Quote Direction | Rate Value |\n")
	builder.WriteString("|------|-----------|-------|-----------|-----------------|----------------------|-------------------|-----------------|------------|\n")
	for entryIndex, entry := range report.AuditAnnex.ConversionAuditEntries {
		if err := writeConversionAuditRow(builder, entryIndex, entry); err != nil {
			return err
		}
	}

	builder.WriteString("\n")
	return nil
}

// writeConversionAuditRow renders one grouped activity-level conversion audit row.
// Authored by: OpenCode
func writeConversionAuditRow(builder *strings.Builder, entryIndex int, entry reportmodel.ConversionAuditEntry) error {
	var row, err = presentation.BuildConversionAuditRow(entryIndex, entry)
	if err != nil {
		return err
	}

	fmt.Fprintf(builder,
		"| %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
		sanitizeInlineText(row.Date), sanitizeInlineText(row.SourceID), sanitizeInlineText(row.Asset), sanitizeInlineText(row.RateDate), sanitizeInlineText(row.SourceCurrency), sanitizeInlineText(row.ReportBaseCurrency), sanitizeInlineText(row.ConvertedAmounts), sanitizeInlineText(row.QuoteDirection), sanitizeInlineText(row.RateValue),
	)
	return nil
}

// rateAuthorityLabel returns report-facing authority labels for canonical rate
// evidence.
// Authored by: OpenCode
func rateAuthorityLabel(authority reportmodel.RateAuthority) string {
	return sanitizeInlineText(reportmodel.RateAuthorityDisplayLabel(authority))
}

// rateProviderLabel returns report-facing provider labels for canonical rate
// evidence.
// Authored by: OpenCode
func rateProviderLabel(provider reportmodel.RateProviderID) string {
	if provider == reportmodel.RateProviderIDECBEXR {
		return sanitizeInlineText("ECB Data Portal `EXR`")
	}

	return sanitizeInlineText(reportmodel.RateProviderDisplayLabel(provider))
}

// unavailableDateRule returns the report-facing prior-observation rule for one
// canonical provider.
// Authored by: OpenCode
func unavailableDateRule(provider reportmodel.RateProviderID) string {
	return sanitizeInlineText(reportmodel.RateProviderUnavailableDateRule(provider))
}

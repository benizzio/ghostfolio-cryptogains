// Package markdown defines Markdown document rendering for calculated yearly
// gains-and-losses reports.
// Authored by: OpenCode
package markdown

import (
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

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
		var key = strings.Join([]string{string(source.Authority), string(source.ProviderID), source.RateKind}, "|")
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

// writeConversionAuditSection renders one grouped audit row per converted source
// activity disclosed by the report model.
// Authored by: OpenCode
func writeConversionAuditSection(builder *strings.Builder, report reportmodel.CapitalGainsReport) error {
	if len(report.ConversionAuditEntries) == 0 {
		return nil
	}

	builder.WriteString("## Currency Conversion Audit\n\n")
	builder.WriteString("| Date | Source ID | Asset | Rate Date | Source Currency | Report Base Currency | Converted Amounts | Quote Direction | Rate Value |\n")
	builder.WriteString("|------|-----------|-------|-----------|-----------------|----------------------|-------------------|-----------------|------------|\n")
	for entryIndex, entry := range report.ConversionAuditEntries {
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
	var rateValue, err = canonicalDecimal(entry.RateValue)
	if err != nil {
		return fmt.Errorf("render conversion audit entry %d rate value: %w", entryIndex, err)
	}
	var convertedAmounts string
	convertedAmounts, err = renderGroupedConvertedAmounts(entryIndex, entry.Amounts)
	if err != nil {
		return err
	}

	builder.WriteString(fmt.Sprintf(
		"| %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
		entry.ActivityDate.Local().Format("2006-01-02"),
		sanitizeInlineText(entry.SourceID),
		sanitizeInlineText(entry.AssetLabel),
		entry.RateDate.Local().Format("2006-01-02"),
		sanitizeInlineText(entry.SourceCurrency),
		sanitizeInlineText(entry.ReportBaseCurrency.Label()),
		convertedAmounts,
		sanitizeInlineText(string(entry.QuoteDirection)),
		rateValue,
	))
	return nil
}

// renderGroupedConvertedAmounts formats non-zero conversion amount slots for one
// source activity and omits zero-to-zero slots from report-visible audit output.
// Authored by: OpenCode
func renderGroupedConvertedAmounts(entryIndex int, amounts []reportmodel.ConvertedActivityAmount) (string, error) {
	var rendered []string
	for amountIndex, amount := range amounts {
		if amount.OriginalAmount.Sign() == 0 && amount.ConvertedAmount.Sign() == 0 {
			continue
		}
		var originalAmount, err = canonicalDecimal(amount.OriginalAmount)
		if err != nil {
			return "", fmt.Errorf("render conversion audit entry %d amount %d original amount: %w", entryIndex, amountIndex, err)
		}
		var convertedAmount string
		convertedAmount, err = canonicalDecimal(amount.ConvertedAmount)
		if err != nil {
			return "", fmt.Errorf("render conversion audit entry %d amount %d converted amount: %w", entryIndex, amountIndex, err)
		}
		rendered = append(rendered, fmt.Sprintf("%s: %s -> %s", sanitizeInlineText(string(amount.AmountKind)), originalAmount, convertedAmount))
	}

	return strings.Join(rendered, "; "), nil
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
	return sanitizeInlineText(reportmodel.RateProviderDisplayLabel(provider))
}

// unavailableDateRule returns the report-facing prior-observation rule for one
// canonical provider.
// Authored by: OpenCode
func unavailableDateRule(provider reportmodel.RateProviderID) string {
	return sanitizeInlineText(reportmodel.RateProviderUnavailableDateRule(provider))
}

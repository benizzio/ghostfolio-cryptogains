// Package markdown defines Markdown document rendering for calculated yearly
// gains-and-losses reports.
// Authored by: OpenCode
package markdown

import (
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/benizzio/ghostfolio-cryptogains/internal/support/redact"
	"github.com/cockroachdb/apd/v3"
)

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

// renderDisplayLabel returns the safe display label for one asset row or section heading.
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
// blank for rows without one selected activity monetary context.
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

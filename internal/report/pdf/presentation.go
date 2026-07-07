package pdf

import (
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	textsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/text"
)

// calculationCurrencyLabel normalizes a report calculation-currency label.
// Authored by: OpenCode
func calculationCurrencyLabel(raw string) string {
	var normalized = sanitizeText(raw)
	if normalized == "" {
		return "NOT APPLICABLE"
	}
	return normalized
}

// calculationCurrencyLabelWithFallback returns an explicit currency or fallback.
// Authored by: OpenCode
func calculationCurrencyLabelWithFallback(raw string, fallback string) string {
	var normalized = sanitizeText(raw)
	if normalized == "" {
		return calculationCurrencyLabel(fallback)
	}
	return normalized
}

// renderDisplayLabel returns a safe display label for an asset.
// Authored by: OpenCode
func renderDisplayLabel(displayLabel string, assetIdentityKey string) string {
	var normalized = sanitizeText(displayLabel)
	if normalized != "" {
		return normalized
	}
	normalized = sanitizeText(assetIdentityKey)
	if normalized != "" {
		return normalized
	}
	return "Unknown Asset"
}

// activityCurrencyColumn renders activity currency only for monetary rows.
// Authored by: OpenCode
func activityCurrencyColumn(row reportmodel.AssetActivityRow) string {
	if strings.TrimSpace(row.ActivityCurrency) == "" {
		return ""
	}
	if row.GrossValue == nil && row.FeeAmount == nil && row.UnitPrice == nil {
		return ""
	}
	return sanitizeText(row.ActivityCurrency)
}

// conversionStatusColumn returns the report-facing conversion status label.
// Authored by: OpenCode
func conversionStatusColumn(row reportmodel.AssetActivityRow) (string, error) {
	if activityCurrencyColumn(row) == "" {
		return "", nil
	}
	if strings.TrimSpace(string(row.ConversionStatus)) != "" {
		var label, err = reportmodel.RenderConversionStatusLabel(row.ConversionStatus)
		if err != nil {
			return "", err
		}
		return sanitizeText(label), nil
	}
	if strings.TrimSpace(row.ActivityCurrency) == strings.TrimSpace(row.CalculationCurrency) {
		var label, _ = reportmodel.RenderConversionStatusLabel(reportmodel.ConversionStatusSameCurrency)
		return sanitizeText(label), nil
	}
	var label, _ = reportmodel.RenderConversionStatusLabel(reportmodel.ConversionStatusConverted)
	return sanitizeText(label), nil
}

// rateProviderLabel returns a report-facing provider label.
// Authored by: OpenCode
func rateProviderLabel(provider reportmodel.RateProviderID) string {
	if provider == reportmodel.RateProviderIDECBEXR {
		return sanitizeText("ECB Data Portal EXR")
	}
	return sanitizeText(reportmodel.RateProviderDisplayLabel(provider))
}

// sanitizeText redacts obvious secret-shaped fragments and normalizes one line.
// Authored by: OpenCode
func sanitizeText(raw string) string {
	var sanitized = textsupport.RedactedSingleLine(raw)
	sanitized = strings.ReplaceAll(sanitized, "|", "/")
	return sanitized
}

package pdf

import (
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/report/presentation"
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

// rateProviderLabel returns a report-facing provider label.
// Authored by: OpenCode
func rateProviderLabel(provider reportmodel.RateProviderID) string {
	if provider == reportmodel.RateProviderIDECBEXR {
		return sanitizeText("ECB Data Portal EXR")
	}
	return sanitizeText(reportmodel.RateProviderDisplayLabel(provider))
}

// conversionStatusColumn exposes shared conversion-status derivation to the
// PDF renderer tests while retaining PDF text sanitization.
// Authored by: OpenCode
func conversionStatusColumn(row reportmodel.AssetActivityRow) (string, error) {
	var label, err = presentation.ActivityConversionStatus(row)
	if err != nil {
		return "", err
	}
	return sanitizeText(label), nil
}

// sanitizeText redacts obvious secret-shaped fragments and normalizes one line.
// Authored by: OpenCode
func sanitizeText(raw string) string {
	var sanitized = textsupport.RedactedSingleLine(raw)
	sanitized = strings.ReplaceAll(sanitized, "|", "/")
	return sanitized
}

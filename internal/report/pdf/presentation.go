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

// sanitizeConvertedCell sanitizes each logical converted entry before inserting
// the PDF-only semicolon and line-break boundary.
// Authored by: OpenCode
func sanitizeConvertedCell(entries []string) string {
	var sanitized = make([]string, 0, len(entries))
	// gopdf's vertically centered MultiCell draws explicit lines bottom-first;
	// reverse the input so the visible top-to-bottom order matches the model.
	for index := len(entries) - 1; index >= 0; index-- {
		var entry = entries[index]
		sanitized = append(sanitized, sanitizeText(entry))
	}
	return strings.Join(sanitized, ";\n")
}

// sanitizeText redacts obvious secret-shaped fragments and normalizes one line.
// Authored by: OpenCode
func sanitizeText(raw string) string {
	var sanitized = textsupport.RedactedSingleLine(raw)
	sanitized = strings.ReplaceAll(sanitized, "|", "/")
	return sanitized
}

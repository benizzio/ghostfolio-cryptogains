// Package presentation builds format-neutral report table values.
// Authored by: OpenCode
package presentation

import (
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	datesupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/date"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	textsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/text"
)

// ActivityRow contains format-neutral values for one in-year activity table
// row. Renderers apply their own table escaping and layout rules to its fields.
// Authored by: OpenCode
type ActivityRow struct {
	Date                string
	SourceID            string
	ActivityType        string
	Quantity            string
	UnitPrice           string
	GrossValue          string
	Fee                 string
	QuantityAfterRow    string
	BasisAfterRow       string
	ActivityCurrency    string
	CalculationCurrency string
	ConversionStatus    string
	Note                string
}

// LiquidationRow contains format-neutral values for one liquidation table row.
// Authored by: OpenCode
type LiquidationRow struct {
	Date                string
	SourceID            string
	DisposedQuantity    string
	AllocatedBasis      string
	NetProceeds         string
	GainOrLoss          string
	CalculationCurrency string
}

// AnnexActivityRow contains format-neutral values for one Annex 1 activity row.
// Authored by: OpenCode
type AnnexActivityRow struct {
	Date                 string
	SourceID             string
	ActivityType         string
	Quantity             string
	UnitPrice            string
	GrossValue           string
	Fee                  string
	ActivityCurrency     string
	CalculationCurrency  string
	QuantityAfter        string
	BasisAfter           string
	FullLiquidationEvent string
	AllocatedBasis       string
	NetProceeds          string
	GainOrLoss           string
	ConversionStatus     string
	Note                 string
}

// ConversionAuditRow contains format-neutral values for one conversion-audit
// table row.
// Authored by: OpenCode
type ConversionAuditRow struct {
	Date               string
	SourceID           string
	Asset              string
	RateDate           string
	SourceCurrency     string
	ReportBaseCurrency string
	ConvertedAmounts   string
	QuoteDirection     string
	RateValue          string
}

// BuildActivityRow canonicalizes report-domain activity fields for either
// renderer without applying Markdown escaping or PDF layout sanitization.
// Authored by: OpenCode
func BuildActivityRow(row reportmodel.AssetActivityRow) (ActivityRow, error) {
	quantity, err := decimalsupport.CanonicalString(row.Quantity)
	if err != nil {
		return ActivityRow{}, fmt.Errorf("render activity row %q quantity: %w", row.SourceID, err)
	}
	unitPrice, err := decimalsupport.CanonicalStringPointer(row.UnitPrice)
	if err != nil {
		return ActivityRow{}, fmt.Errorf("render activity row %q unit price: %w", row.SourceID, err)
	}
	grossValue, err := decimalsupport.CanonicalStringPointer(row.GrossValue)
	if err != nil {
		return ActivityRow{}, fmt.Errorf("render activity row %q gross value: %w", row.SourceID, err)
	}
	fee, err := decimalsupport.CanonicalStringPointer(row.FeeAmount)
	if err != nil {
		return ActivityRow{}, fmt.Errorf("render activity row %q fee: %w", row.SourceID, err)
	}
	basisAfterRow, err := decimalsupport.CanonicalString(row.BasisAfterRow)
	if err != nil {
		return ActivityRow{}, fmt.Errorf("render activity row %q basis after row: %w", row.SourceID, err)
	}
	quantityAfterRow, err := decimalsupport.CanonicalString(row.QuantityAfterRow)
	if err != nil {
		return ActivityRow{}, fmt.Errorf("render activity row %q quantity after row: %w", row.SourceID, err)
	}
	activityType, err := reportmodel.RenderActivityTypeLabel(row)
	if err != nil {
		return ActivityRow{}, fmt.Errorf("render activity row %q type label: %w", row.SourceID, err)
	}
	conversionStatus, err := activityConversionStatus(row)
	if err != nil {
		return ActivityRow{}, fmt.Errorf("render activity row %q conversion status label: %w", row.SourceID, err)
	}

	return ActivityRow{
		Date:                row.OccurredAt.UTC().Format("2006-01-02 15:04:05"),
		SourceID:            sanitize(row.SourceID),
		ActivityType:        sanitize(activityType),
		Quantity:            quantity,
		UnitPrice:           unitPrice,
		GrossValue:          grossValue,
		Fee:                 fee,
		QuantityAfterRow:    quantityAfterRow,
		BasisAfterRow:       basisAfterRow,
		ActivityCurrency:    activityCurrency(row),
		CalculationCurrency: CalculationCurrencyLabel(row.CalculationCurrency),
		ConversionStatus:    sanitize(conversionStatus),
		Note:                sanitize(row.HoldingReductionExplanation),
	}, nil
}

// ActivityConversionStatus derives the visible conversion status for a monetary
// activity row without applying renderer-specific escaping.
// Authored by: OpenCode
func ActivityConversionStatus(row reportmodel.AssetActivityRow) (string, error) {
	return activityConversionStatus(row)
}

// BuildLiquidationRow canonicalizes report-domain liquidation fields for either
// renderer without applying Markdown escaping or PDF layout sanitization.
// Authored by: OpenCode
func BuildLiquidationRow(liquidation reportmodel.LiquidationCalculation, fallbackCurrency string) (LiquidationRow, error) {
	disposedQuantity, err := decimalsupport.CanonicalString(liquidation.DisposedQuantity)
	if err != nil {
		return LiquidationRow{}, fmt.Errorf("render liquidation %q disposed quantity: %w", liquidation.SourceID, err)
	}
	allocatedBasis, err := decimalsupport.CanonicalString(liquidation.AllocatedBasis)
	if err != nil {
		return LiquidationRow{}, fmt.Errorf("render liquidation %q allocated basis: %w", liquidation.SourceID, err)
	}
	netProceeds, err := decimalsupport.CanonicalString(liquidation.NetLiquidationProceeds)
	if err != nil {
		return LiquidationRow{}, fmt.Errorf("render liquidation %q net proceeds: %w", liquidation.SourceID, err)
	}
	gainOrLoss, err := decimalsupport.CanonicalString(liquidation.GainOrLoss)
	if err != nil {
		return LiquidationRow{}, fmt.Errorf("render liquidation %q gain or loss: %w", liquidation.SourceID, err)
	}

	return LiquidationRow{Date: liquidation.OccurredAt.UTC().Format("2006-01-02 15:04:05"), SourceID: sanitize(liquidation.SourceID), DisposedQuantity: disposedQuantity, AllocatedBasis: allocatedBasis, NetProceeds: netProceeds, GainOrLoss: gainOrLoss, CalculationCurrency: CalculationCurrencyLabelWithFallback(liquidation.CalculationCurrency, fallbackCurrency)}, nil
}

// BuildAnnexActivityRow canonicalizes detailed audit activity fields for either
// renderer without applying Markdown escaping or PDF layout sanitization.
// Authored by: OpenCode
func BuildAnnexActivityRow(entry reportmodel.AuditActivityEntry) (AnnexActivityRow, error) {
	quantity, err := decimalsupport.CanonicalString(entry.Quantity)
	if err != nil {
		return AnnexActivityRow{}, fmt.Errorf("render annex activity row %q quantity: %w", entry.SourceID, err)
	}
	unitPrice, err := decimalsupport.CanonicalStringPointer(entry.UnitPrice)
	if err != nil {
		return AnnexActivityRow{}, fmt.Errorf("render annex activity row %q unit price: %w", entry.SourceID, err)
	}
	grossValue, err := decimalsupport.CanonicalStringPointer(entry.GrossValue)
	if err != nil {
		return AnnexActivityRow{}, fmt.Errorf("render annex activity row %q gross value: %w", entry.SourceID, err)
	}
	fee, err := decimalsupport.CanonicalStringPointer(entry.FeeAmount)
	if err != nil {
		return AnnexActivityRow{}, fmt.Errorf("render annex activity row %q fee: %w", entry.SourceID, err)
	}
	quantityAfter, err := decimalsupport.CanonicalString(entry.QuantityAfterActivity)
	if err != nil {
		return AnnexActivityRow{}, fmt.Errorf("render annex activity row %q quantity after activity: %w", entry.SourceID, err)
	}
	basisAfter, err := decimalsupport.CanonicalString(entry.BasisAfterActivity)
	if err != nil {
		return AnnexActivityRow{}, fmt.Errorf("render annex activity row %q basis after activity: %w", entry.SourceID, err)
	}
	allocatedBasis, err := decimalsupport.CanonicalStringPointer(entry.AllocatedBasis)
	if err != nil {
		return AnnexActivityRow{}, fmt.Errorf("render annex activity row %q allocated basis: %w", entry.SourceID, err)
	}
	netProceeds, err := decimalsupport.CanonicalStringPointer(entry.NetLiquidationProceeds)
	if err != nil {
		return AnnexActivityRow{}, fmt.Errorf("render annex activity row %q net liquidation proceeds: %w", entry.SourceID, err)
	}
	gainOrLoss, err := decimalsupport.CanonicalStringPointer(entry.GainOrLoss)
	if err != nil {
		return AnnexActivityRow{}, fmt.Errorf("render annex activity row %q gain or loss: %w", entry.SourceID, err)
	}
	activityType, err := reportmodel.RenderAuditActivityTypeLabel(entry)
	if err != nil {
		return AnnexActivityRow{}, fmt.Errorf("render annex activity row %q activity type label: %w", entry.SourceID, err)
	}
	conversionStatus := ""
	if strings.TrimSpace(string(entry.ConversionStatus)) != "" {
		conversionStatus, err = reportmodel.RenderConversionStatusLabel(entry.ConversionStatus)
		if err != nil {
			return AnnexActivityRow{}, fmt.Errorf("render annex activity row %q conversion status label: %w", entry.SourceID, err)
		}
	}

	return AnnexActivityRow{Date: entry.OccurredAt.UTC().Format("2006-01-02 15:04:05"), SourceID: sanitize(entry.SourceID), ActivityType: sanitize(activityType), Quantity: quantity, UnitPrice: unitPrice, GrossValue: grossValue, Fee: fee, ActivityCurrency: sanitize(entry.ActivityCurrency), CalculationCurrency: sanitize(entry.CalculationCurrency), QuantityAfter: quantityAfter, BasisAfter: basisAfter, FullLiquidationEvent: fmt.Sprintf("%t", entry.FullLiquidationEvent), AllocatedBasis: allocatedBasis, NetProceeds: netProceeds, GainOrLoss: gainOrLoss, ConversionStatus: sanitize(conversionStatus), Note: sanitize(entry.Note)}, nil
}

// BuildConversionAuditRow builds visible conversion-audit values without
// applying Markdown escaping or PDF layout sanitization.
// Authored by: OpenCode
func BuildConversionAuditRow(index int, entry reportmodel.ConversionAuditEntry) (ConversionAuditRow, error) {
	rateValue, err := decimalsupport.CanonicalString(entry.RateValue)
	if err != nil {
		return ConversionAuditRow{}, fmt.Errorf("render conversion audit entry %d rate value: %w", index, err)
	}
	amounts, err := ConvertedAmounts(index, entry.Amounts)
	if err != nil {
		return ConversionAuditRow{}, err
	}
	quoteDirection, err := reportmodel.RenderQuoteDirectionLabel(entry.QuoteDirection)
	if err != nil {
		return ConversionAuditRow{}, fmt.Errorf("render conversion audit entry %d quote direction: %w", index, err)
	}
	return ConversionAuditRow{Date: datesupport.FormatCalendarDate(entry.ActivityDate), SourceID: sanitize(entry.SourceID), Asset: sanitize(entry.AssetLabel), RateDate: datesupport.FormatCalendarDate(entry.RateDate), SourceCurrency: sanitize(entry.SourceCurrency), ReportBaseCurrency: sanitize(entry.ReportBaseCurrency.Label()), ConvertedAmounts: sanitize(amounts), QuoteDirection: sanitize(quoteDirection), RateValue: rateValue}, nil
}

// ConvertedAmounts formats non-zero converted amount evidence in report order.
// Authored by: OpenCode
func ConvertedAmounts(entryIndex int, amounts []reportmodel.ConvertedActivityAmount) (string, error) {
	var rendered []string
	for amountIndex, amount := range amounts {
		if amount.OriginalAmount.Sign() == 0 && amount.ConvertedAmount.Sign() == 0 {
			continue
		}
		original, err := decimalsupport.CanonicalString(amount.OriginalAmount)
		if err != nil {
			return "", fmt.Errorf("render conversion audit entry %d amount %d original amount: %w", entryIndex, amountIndex, err)
		}
		converted, err := decimalsupport.CanonicalString(amount.ConvertedAmount)
		if err != nil {
			return "", fmt.Errorf("render conversion audit entry %d amount %d converted amount: %w", entryIndex, amountIndex, err)
		}
		rendered = append(rendered, fmt.Sprintf("%s: %s -> %s", sanitize(string(amount.AmountKind)), original, converted))
	}
	return strings.Join(rendered, "; "), nil
}

// CalculationCurrencyLabel returns the report-visible fallback for an absent
// calculation currency without applying renderer-specific escaping.
// Authored by: OpenCode
func CalculationCurrencyLabel(raw string) string {
	if normalized := sanitize(raw); normalized != "" {
		return normalized
	}
	return "NOT APPLICABLE"
}

// CalculationCurrencyLabelWithFallback returns a report currency or its
// report-wide fallback without applying renderer-specific escaping.
// Authored by: OpenCode
func CalculationCurrencyLabelWithFallback(raw string, fallback string) string {
	if normalized := sanitize(raw); normalized != "" {
		return normalized
	}
	return CalculationCurrencyLabel(fallback)
}

// activityCurrency keeps explanatory rows without monetary context blank.
// Authored by: OpenCode
func activityCurrency(row reportmodel.AssetActivityRow) string {
	if strings.TrimSpace(row.ActivityCurrency) == "" || row.GrossValue == nil && row.FeeAmount == nil && row.UnitPrice == nil {
		return ""
	}
	return sanitize(row.ActivityCurrency)
}

// activityConversionStatus derives the visible conversion label for monetary rows.
// Authored by: OpenCode
func activityConversionStatus(row reportmodel.AssetActivityRow) (string, error) {
	if activityCurrency(row) == "" {
		return "", nil
	}
	if strings.TrimSpace(string(row.ConversionStatus)) != "" {
		return reportmodel.RenderConversionStatusLabel(row.ConversionStatus)
	}
	if strings.TrimSpace(row.ActivityCurrency) == strings.TrimSpace(row.CalculationCurrency) {
		return reportmodel.RenderConversionStatusLabel(reportmodel.ConversionStatusSameCurrency)
	}
	return reportmodel.RenderConversionStatusLabel(reportmodel.ConversionStatusConverted)
}

// sanitize redacts and flattens report data while leaving format delimiters intact.
// Authored by: OpenCode
func sanitize(value string) string { return textsupport.RedactedSingleLine(value) }

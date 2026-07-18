// Package presentation converts calculated report-domain values into
// format-neutral report table values. It applies presentation-only financial
// rounding, canonical quantity and rate formatting, boolean labels, and
// applicability rules without mutating the source report model or applying
// Markdown escaping or PDF layout sanitization. Renderers consume these values
// and remain responsible for their own syntax and layout policy.
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
// row. Monetary fields are presentation strings at the report's two-place
// display scale when present, quantities retain canonical decimal formatting,
// and dynamic text remains free of renderer-specific syntax. Renderers apply
// their own table escaping and layout rules to the fields.
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
// Disposed quantity retains canonical decimal formatting, while allocated
// basis, net proceeds, and gain or loss use the report's two-place financial
// display policy. The calculation currency is always represented by a visible
// label, using the supplied report fallback when the row has none.
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
// Optional monetary values are blank when absent and otherwise use the
// two-place financial display policy. A classified zero-priced holding
// reduction keeps its audit classification in the source model but exposes a
// blank activity-currency cell because that currency is not applicable to the
// visible row.
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
// table row. ConvertedAmountEntries contains explicit logical entries in the
// received order, with exact zero-to-zero pairs already omitted by the
// presentation boundary. RateValue retains canonical normalized-rate
// formatting rather than the two-place financial display scale.
// Authored by: OpenCode
type ConversionAuditRow struct {
	Date               string
	SourceID           string
	Asset              string
	RateDate           string
	SourceCurrency     string
	ReportBaseCurrency string
	// ConvertedAmountEntries contains one explicit logical entry per included
	// converted amount, in the received order.
	// Authored by: OpenCode
	ConvertedAmountEntries []ConvertedAmountEntry
	QuoteDirection         string
	RateValue              string
}

// BuildActivityRow canonicalizes one report-domain activity for either
// renderer. It formats quantities and normalized labels, applies presentation-
// only two-place financial rounding to monetary values, leaves absent optional
// amounts blank, and does not mutate row or apply Markdown escaping or PDF
// layout sanitization. An error identifies the source row and field that could
// not be presented.
//
// Example usage:
//
//	activityRow, err := presentation.BuildActivityRow(activity)
//	if err != nil {
//		return err
//	}
//	markdownCell := escapeMarkdown(activityRow.GrossValue)
//
// Authored by: OpenCode
func BuildActivityRow(row reportmodel.AssetActivityRow) (ActivityRow, error) {
	quantity, err := decimalsupport.CanonicalString(row.Quantity)
	if err != nil {
		return ActivityRow{}, fmt.Errorf("render activity row %q quantity: %w", row.SourceID, err)
	}
	unitPrice, err := formatOptionalFinancialValue(row.UnitPrice)
	if err != nil {
		return ActivityRow{}, fmt.Errorf("render activity row %q unit price: %w", row.SourceID, err)
	}
	grossValue, err := formatOptionalFinancialValue(row.GrossValue)
	if err != nil {
		return ActivityRow{}, fmt.Errorf("render activity row %q gross value: %w", row.SourceID, err)
	}
	fee, err := formatOptionalFinancialValue(row.FeeAmount)
	if err != nil {
		return ActivityRow{}, fmt.Errorf("render activity row %q fee: %w", row.SourceID, err)
	}
	basisAfterRow, err := formatFinancialValue(row.BasisAfterRow)
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

// ActivityConversionStatus derives the visible conversion status for a
// monetary activity row without applying renderer-specific escaping. It
// returns an empty label for rows without monetary context, otherwise honoring
// an explicit status or deriving same-currency versus converted status from the
// row currencies.
//
// Example usage:
//
//	status, err := presentation.ActivityConversionStatus(activity)
//	if err != nil {
//		return err
//	}
//	columnValue := sanitizeForRenderer(status)
//
// Authored by: OpenCode
func ActivityConversionStatus(row reportmodel.AssetActivityRow) (string, error) {
	return activityConversionStatus(row)
}

// BuildLiquidationRow canonicalizes one report-domain liquidation for either
// renderer. It preserves canonical disposed-quantity formatting, applies
// presentation-only two-place financial rounding to monetary fields, uses the
// fallback currency only when the calculation has no currency, and leaves
// renderer escaping to the caller. The input calculation is not mutated.
//
// Example usage:
//
//	liquidationRow, err := presentation.BuildLiquidationRow(calculation, "USD")
//	if err != nil {
//		return err
//	}
//	pdfCells := []string{liquidationRow.DisposedQuantity, liquidationRow.GainOrLoss}
//
// Authored by: OpenCode
func BuildLiquidationRow(liquidation reportmodel.LiquidationCalculation, fallbackCurrency string) (LiquidationRow, error) {
	disposedQuantity, err := decimalsupport.CanonicalString(liquidation.DisposedQuantity)
	if err != nil {
		return LiquidationRow{}, fmt.Errorf("render liquidation %q disposed quantity: %w", liquidation.SourceID, err)
	}
	allocatedBasis, err := formatFinancialValue(liquidation.AllocatedBasis)
	if err != nil {
		return LiquidationRow{}, fmt.Errorf("render liquidation %q allocated basis: %w", liquidation.SourceID, err)
	}
	netProceeds, err := formatFinancialValue(liquidation.NetLiquidationProceeds)
	if err != nil {
		return LiquidationRow{}, fmt.Errorf("render liquidation %q net proceeds: %w", liquidation.SourceID, err)
	}
	gainOrLoss, err := formatFinancialValue(liquidation.GainOrLoss)
	if err != nil {
		return LiquidationRow{}, fmt.Errorf("render liquidation %q gain or loss: %w", liquidation.SourceID, err)
	}

	return LiquidationRow{Date: liquidation.OccurredAt.UTC().Format("2006-01-02 15:04:05"), SourceID: sanitize(liquidation.SourceID), DisposedQuantity: disposedQuantity, AllocatedBasis: allocatedBasis, NetProceeds: netProceeds, GainOrLoss: gainOrLoss, CalculationCurrency: CalculationCurrencyLabelWithFallback(liquidation.CalculationCurrency, fallbackCurrency)}, nil
}

// BuildAnnexActivityRow canonicalizes one detailed audit activity for either
// renderer. It preserves canonical quantities, formats present monetary values
// at two places, maps structured booleans to Yes or No, and suppresses only the
// visible activity currency for a classified zero-priced holding reduction.
// Source audit values remain unchanged and renderer-specific escaping is left
// to the caller.
//
// Example usage:
//
//	annexRow, err := presentation.BuildAnnexActivityRow(entry)
//	if err != nil {
//		return err
//	}
//	row := []string{annexRow.ActivityType, annexRow.FullLiquidationEvent, annexRow.GainOrLoss}
//
// Authored by: OpenCode
func BuildAnnexActivityRow(entry reportmodel.AuditActivityEntry) (AnnexActivityRow, error) {
	quantity, err := decimalsupport.CanonicalString(entry.Quantity)
	if err != nil {
		return AnnexActivityRow{}, fmt.Errorf("render annex activity row %q quantity: %w", entry.SourceID, err)
	}
	unitPrice, err := formatOptionalFinancialValue(entry.UnitPrice)
	if err != nil {
		return AnnexActivityRow{}, fmt.Errorf("render annex activity row %q unit price: %w", entry.SourceID, err)
	}
	grossValue, err := formatOptionalFinancialValue(entry.GrossValue)
	if err != nil {
		return AnnexActivityRow{}, fmt.Errorf("render annex activity row %q gross value: %w", entry.SourceID, err)
	}
	fee, err := formatOptionalFinancialValue(entry.FeeAmount)
	if err != nil {
		return AnnexActivityRow{}, fmt.Errorf("render annex activity row %q fee: %w", entry.SourceID, err)
	}
	quantityAfter, err := decimalsupport.CanonicalString(entry.QuantityAfterActivity)
	if err != nil {
		return AnnexActivityRow{}, fmt.Errorf("render annex activity row %q quantity after activity: %w", entry.SourceID, err)
	}
	basisAfter, err := formatFinancialValue(entry.BasisAfterActivity)
	if err != nil {
		return AnnexActivityRow{}, fmt.Errorf("render annex activity row %q basis after activity: %w", entry.SourceID, err)
	}
	allocatedBasis, err := formatOptionalFinancialValue(entry.AllocatedBasis)
	if err != nil {
		return AnnexActivityRow{}, fmt.Errorf("render annex activity row %q allocated basis: %w", entry.SourceID, err)
	}
	netProceeds, err := formatOptionalFinancialValue(entry.NetLiquidationProceeds)
	if err != nil {
		return AnnexActivityRow{}, fmt.Errorf("render annex activity row %q net liquidation proceeds: %w", entry.SourceID, err)
	}
	gainOrLoss, err := formatOptionalFinancialValue(entry.GainOrLoss)
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

	return AnnexActivityRow{Date: entry.OccurredAt.UTC().Format("2006-01-02 15:04:05"), SourceID: sanitize(entry.SourceID), ActivityType: sanitize(activityType), Quantity: quantity, UnitPrice: unitPrice, GrossValue: grossValue, Fee: fee, ActivityCurrency: annexActivityCurrency(entry), CalculationCurrency: sanitize(entry.CalculationCurrency), QuantityAfter: quantityAfter, BasisAfter: basisAfter, FullLiquidationEvent: booleanLabel(entry.FullLiquidationEvent), AllocatedBasis: allocatedBasis, NetProceeds: netProceeds, GainOrLoss: gainOrLoss, ConversionStatus: sanitize(conversionStatus), Note: sanitize(entry.Note)}, nil
}

// BuildConversionAuditRow builds one visible conversion-audit row without
// applying Markdown escaping or PDF layout sanitization. It retains the
// received order of non-zero converted amount entries, applies two-place
// financial formatting only to their amounts, and keeps the exchange-rate
// value in canonical normalized form. Errors include the audit-entry index and
// failing field context.
//
// Example usage:
//
//	auditRow, err := presentation.BuildConversionAuditRow(0, conversion)
//	if err != nil {
//		return err
//	}
//	for _, amount := range auditRow.ConvertedAmountEntries {
//		renderConversion(amount.Label, amount.OriginalAmount, amount.ConvertedAmount)
//	}
//
// Authored by: OpenCode
func BuildConversionAuditRow(index int, entry reportmodel.ConversionAuditEntry) (ConversionAuditRow, error) {
	rateValue, err := decimalsupport.CanonicalString(entry.RateValue)
	if err != nil {
		return ConversionAuditRow{}, fmt.Errorf("render conversion audit entry %d rate value: %w", index, err)
	}
	amountEntries, err := ConvertedAmounts(index, entry.Amounts)
	if err != nil {
		return ConversionAuditRow{}, err
	}
	quoteDirection, err := reportmodel.RenderQuoteDirectionLabel(entry.QuoteDirection)
	if err != nil {
		return ConversionAuditRow{}, fmt.Errorf("render conversion audit entry %d quote direction: %w", index, err)
	}
	return ConversionAuditRow{
		Date:                   datesupport.FormatCalendarDate(entry.ActivityDate),
		SourceID:               sanitize(entry.SourceID),
		Asset:                  sanitize(entry.AssetLabel),
		RateDate:               datesupport.FormatCalendarDate(entry.RateDate),
		SourceCurrency:         sanitize(entry.SourceCurrency),
		ReportBaseCurrency:     sanitize(entry.ReportBaseCurrency.Label()),
		ConvertedAmountEntries: amountEntries,
		QuoteDirection:         sanitize(quoteDirection),
		RateValue:              rateValue,
	}, nil
}

// CalculationCurrencyLabel returns a sanitized report currency or the exact
// NOT APPLICABLE fallback for an absent calculation currency. The result is a
// single-line, renderer-neutral label and still requires renderer-specific
// escaping before output.
//
// Example usage:
//
//	currency := presentation.CalculationCurrencyLabel(report.CalculationCurrency)
//	markdownValue := escapeMarkdown(currency)
//
// Authored by: OpenCode
func CalculationCurrencyLabel(raw string) string {
	if normalized := sanitize(raw); normalized != "" {
		return normalized
	}
	return "NOT APPLICABLE"
}

// CalculationCurrencyLabelWithFallback returns a sanitized row currency when
// available, otherwise applying the same report-visible fallback policy to the
// supplied report-wide currency. The returned label is renderer-neutral.
//
// Example usage:
//
//	currency := presentation.CalculationCurrencyLabelWithFallback(row.Currency, "USD")
//	pdfCell := sanitizePDFText(currency)
//
// Authored by: OpenCode
func CalculationCurrencyLabelWithFallback(raw string, fallback string) string {
	if normalized := sanitize(raw); normalized != "" {
		return normalized
	}
	return CalculationCurrencyLabel(fallback)
}

// booleanLabel maps a structured report boolean to its reader-facing label.
// Authored by: OpenCode
func booleanLabel(value bool) string {
	if value {
		return "Yes"
	}
	return "No"
}

// annexActivityCurrency derives the visible Annex activity currency from the
// inherited classification without changing the retained audit entry.
// Authored by: OpenCode
func annexActivityCurrency(entry reportmodel.AuditActivityEntry) string {
	if entry.IsZeroPricedHoldingReduction {
		return ""
	}
	return sanitize(entry.ActivityCurrency)
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

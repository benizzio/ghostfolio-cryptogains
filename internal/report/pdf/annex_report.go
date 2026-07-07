package pdf

import (
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	datesupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/date"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
)

// renderAnnex renders Annex 1 after the required PDF page break.
// Authored by: OpenCode
func renderAnnex(document pdfLayoutDocument, annex reportmodel.AuditAnnex) error {
	if document == nil {
		return fmt.Errorf("pdf layout document is required")
	}
	if annex.Title == "" && len(annex.SectionOrder) == 0 {
		annex = reportmodel.DefaultAuditAnnex()
	}
	if err := annex.Validate(); err != nil {
		return err
	}
	if err := document.AddTitle(annex.Title); err != nil {
		return err
	}
	if err := renderAnnexPerAssetAudit(document, annex); err != nil {
		return err
	}
	return renderAnnexConversionAudit(document, annex)
}

// renderAnnexPerAssetAudit renders the Detailed Per-Asset Audit Report section.
// Authored by: OpenCode
func renderAnnexPerAssetAudit(document pdfLayoutDocument, annex reportmodel.AuditAnnex) error {
	if err := document.AddSectionHeading("Detailed Per-Asset Audit Report"); err != nil {
		return err
	}
	if len(annex.PerAssetAuditSections) == 0 {
		return document.AddParagraph("No per-asset audit activity is available for this report.")
	}
	for _, section := range annex.PerAssetAuditSections {
		if err := document.AddSubsectionHeading("Asset: " + renderDisplayLabel(section.DisplayLabel, section.AssetIdentityKey)); err != nil {
			return err
		}
		var rows [][]string
		for _, entry := range section.Entries {
			var row, err = renderAnnexActivityRow(entry)
			if err != nil {
				return fmt.Errorf("render annex audit entry %q: %w", entry.SourceID, err)
			}
			rows = append(rows, row)
		}
		if err := document.AddTable(pdfTable{
			Title:             "Per-Asset Audit Activity",
			ContinuationTitle: "Per-Asset Audit Activity (continued)",
			Columns: []pdfColumn{
				{Header: "Date/Time", Width: 42, Align: "left"},
				{Header: "Source ID", Width: 38, Align: "left"},
				{Header: "Activity Type", Width: 38, Align: "left"},
				{Header: "Quantity", Width: 34, Align: "right"},
				{Header: "Unit Price", Width: 34, Align: "right"},
				{Header: "Gross", Width: 32, Align: "right"},
				{Header: "Fee", Width: 30, Align: "right"},
				{Header: "Activity Currency", Width: 34, Align: "left"},
				{Header: "Calc Currency", Width: 34, Align: "left"},
				{Header: "Qty After", Width: 38, Align: "right"},
				{Header: "Basis After", Width: 40, Align: "right"},
				{Header: "Full Liquidation", Width: 34, Align: "left"},
				{Header: "Allocated Basis", Width: 34, Align: "right"},
				{Header: "Net Proceeds", Width: 34, Align: "right"},
				{Header: "Gain/Loss", Width: 32, Align: "right"},
				{Header: "Conversion", Width: 38, Align: "left"},
				{Header: "Note", Width: 38, Align: "left"},
			},
			Rows:      rows,
			RowHeight: 36,
		}); err != nil {
			return err
		}
	}
	return nil
}

// renderAnnexActivityRow formats one detailed audit activity row for a PDF table.
// Authored by: OpenCode
func renderAnnexActivityRow(entry reportmodel.AuditActivityEntry) ([]string, error) {
	var quantity, err = decimalsupport.CanonicalString(entry.Quantity)
	if err != nil {
		return nil, fmt.Errorf("quantity: %w", err)
	}
	var unitPrice string
	unitPrice, err = decimalsupport.CanonicalStringPointer(entry.UnitPrice)
	if err != nil {
		return nil, fmt.Errorf("unit price: %w", err)
	}
	var grossValue string
	grossValue, err = decimalsupport.CanonicalStringPointer(entry.GrossValue)
	if err != nil {
		return nil, fmt.Errorf("gross value: %w", err)
	}
	var fee string
	fee, err = decimalsupport.CanonicalStringPointer(entry.FeeAmount)
	if err != nil {
		return nil, fmt.Errorf("fee: %w", err)
	}
	var quantityAfter string
	quantityAfter, err = decimalsupport.CanonicalString(entry.QuantityAfterActivity)
	if err != nil {
		return nil, fmt.Errorf("quantity after activity: %w", err)
	}
	var basisAfter string
	basisAfter, err = decimalsupport.CanonicalString(entry.BasisAfterActivity)
	if err != nil {
		return nil, fmt.Errorf("basis after activity: %w", err)
	}
	var allocatedBasis string
	allocatedBasis, err = decimalsupport.CanonicalStringPointer(entry.AllocatedBasis)
	if err != nil {
		return nil, fmt.Errorf("allocated basis: %w", err)
	}
	var proceeds string
	proceeds, err = decimalsupport.CanonicalStringPointer(entry.NetLiquidationProceeds)
	if err != nil {
		return nil, fmt.Errorf("net liquidation proceeds: %w", err)
	}
	var gainOrLoss string
	gainOrLoss, err = decimalsupport.CanonicalStringPointer(entry.GainOrLoss)
	if err != nil {
		return nil, fmt.Errorf("gain or loss: %w", err)
	}
	var activityTypeLabel string
	activityTypeLabel, err = reportmodel.RenderAuditActivityTypeLabel(entry)
	if err != nil {
		return nil, fmt.Errorf("activity type label: %w", err)
	}
	var conversionStatus string
	if strings.TrimSpace(string(entry.ConversionStatus)) != "" {
		conversionStatus, err = reportmodel.RenderConversionStatusLabel(entry.ConversionStatus)
		if err != nil {
			return nil, fmt.Errorf("conversion status label: %w", err)
		}
	}
	return []string{
		entry.OccurredAt.UTC().Format("2006-01-02 15:04:05"),
		sanitizeText(entry.SourceID),
		sanitizeText(activityTypeLabel),
		quantity,
		unitPrice,
		grossValue,
		fee,
		sanitizeText(entry.ActivityCurrency),
		sanitizeText(entry.CalculationCurrency),
		quantityAfter,
		basisAfter,
		fmt.Sprintf("%t", entry.FullLiquidationEvent),
		allocatedBasis,
		proceeds,
		gainOrLoss,
		sanitizeText(conversionStatus),
		sanitizeText(entry.Note),
	}, nil
}

// renderAnnexConversionAudit renders Annex 1 currency conversion evidence.
// Authored by: OpenCode
func renderAnnexConversionAudit(document pdfLayoutDocument, annex reportmodel.AuditAnnex) error {
	if err := document.AddSectionHeading("Currency Conversion Audit"); err != nil {
		return err
	}
	if len(annex.ConversionAuditEntries) == 0 {
		return document.AddParagraph("No converted activity was present for this report.")
	}

	var rows [][]string
	for index, entry := range annex.ConversionAuditEntries {
		var row, err = renderConversionAuditRow(index, entry)
		if err != nil {
			return err
		}
		rows = append(rows, row)
	}
	return document.AddTable(pdfTable{
		Title:             "Currency Conversion Audit Table",
		ContinuationTitle: "Currency Conversion Audit Table (continued)",
		Columns: []pdfColumn{
			{Header: "Date", Width: 50, Align: "left"},
			{Header: "Source ID", Width: 55, Align: "left"},
			{Header: "Asset", Width: 45, Align: "left"},
			{Header: "Rate Date", Width: 50, Align: "left"},
			{Header: "Source Currency", Width: 55, Align: "left"},
			{Header: "Report Base Currency", Width: 62, Align: "left"},
			{Header: "Converted Amounts", Width: 105, Align: "left"},
			{Header: "Quote Direction", Width: 95, Align: "left"},
			{Header: "Rate Value", Width: 50, Align: "right"},
		},
		Rows:      rows,
		RowHeight: 36,
	})
}

// renderConversionAuditRow formats one conversion audit row.
// Authored by: OpenCode
func renderConversionAuditRow(index int, entry reportmodel.ConversionAuditEntry) ([]string, error) {
	var rateValue, err = decimalsupport.CanonicalString(entry.RateValue)
	if err != nil {
		return nil, fmt.Errorf("render conversion audit entry %d rate value: %w", index, err)
	}
	var convertedAmounts string
	convertedAmounts, err = renderGroupedConvertedAmounts(index, entry.Amounts)
	if err != nil {
		return nil, err
	}
	var quoteDirection string
	quoteDirection, err = reportmodel.RenderQuoteDirectionLabel(entry.QuoteDirection)
	if err != nil {
		return nil, fmt.Errorf("render conversion audit entry %d quote direction: %w", index, err)
	}
	return []string{
		datesupport.FormatCalendarDate(entry.ActivityDate),
		sanitizeText(entry.SourceID),
		sanitizeText(entry.AssetLabel),
		datesupport.FormatCalendarDate(entry.RateDate),
		sanitizeText(entry.SourceCurrency),
		sanitizeText(entry.ReportBaseCurrency.Label()),
		convertedAmounts,
		sanitizeText(quoteDirection),
		rateValue,
	}, nil
}

// renderGroupedConvertedAmounts formats non-zero converted amount evidence.
// Authored by: OpenCode
func renderGroupedConvertedAmounts(entryIndex int, amounts []reportmodel.ConvertedActivityAmount) (string, error) {
	var rendered []string
	for amountIndex, amount := range amounts {
		if amount.OriginalAmount.Sign() == 0 && amount.ConvertedAmount.Sign() == 0 {
			continue
		}
		var originalAmount, err = decimalsupport.CanonicalString(amount.OriginalAmount)
		if err != nil {
			return "", fmt.Errorf("render conversion audit entry %d amount %d original amount: %w", entryIndex, amountIndex, err)
		}
		var convertedAmount string
		convertedAmount, err = decimalsupport.CanonicalString(amount.ConvertedAmount)
		if err != nil {
			return "", fmt.Errorf("render conversion audit entry %d amount %d converted amount: %w", entryIndex, amountIndex, err)
		}
		rendered = append(rendered, fmt.Sprintf("%s: %s -> %s", sanitizeText(string(amount.AmountKind)), originalAmount, convertedAmount))
	}
	return strings.Join(rendered, "; "), nil
}

package pdf

import (
	"fmt"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/report/presentation"
)

// renderAnnex renders Annex 1 after the required PDF page break.
// Authored by: OpenCode
func renderAnnex(document pdfContentLayout, annex reportmodel.AuditAnnex) error {
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
func renderAnnexPerAssetAudit(document pdfContentLayout, annex reportmodel.AuditAnnex) error {
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
	var rendered, err = presentation.BuildAnnexActivityRow(entry)
	if err != nil {
		return nil, err
	}
	return []string{
		rendered.Date, sanitizeText(rendered.SourceID), sanitizeText(rendered.ActivityType), rendered.Quantity, rendered.UnitPrice, rendered.GrossValue, rendered.Fee, sanitizeText(rendered.ActivityCurrency), sanitizeText(rendered.CalculationCurrency), rendered.QuantityAfter, rendered.BasisAfter, rendered.FullLiquidationEvent, rendered.AllocatedBasis, rendered.NetProceeds, rendered.GainOrLoss, sanitizeText(rendered.ConversionStatus), sanitizeText(rendered.Note),
	}, nil
}

// renderAnnexConversionAudit renders Annex 1 currency conversion evidence.
// Authored by: OpenCode
func renderAnnexConversionAudit(document pdfContentLayout, annex reportmodel.AuditAnnex) error {
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
	var row, err = presentation.BuildConversionAuditRow(index, entry)
	if err != nil {
		return nil, err
	}
	return sanitizeRow([]string{row.Date, row.SourceID, row.Asset, row.RateDate, row.SourceCurrency, row.ReportBaseCurrency, row.ConvertedAmounts, row.QuoteDirection, row.RateValue}), nil
}

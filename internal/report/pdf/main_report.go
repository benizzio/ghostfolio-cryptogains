package pdf

import (
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// renderMainReport renders the main report through structured PDF operations.
// Authored by: OpenCode
func renderMainReport(document pdfLayoutDocument, report reportmodel.CapitalGainsReport) error {
	if document == nil {
		return fmt.Errorf("pdf layout document is required")
	}
	if err := report.Validate(); err != nil {
		return err
	}

	var calculationCurrency = calculationCurrencyLabel(report.ReportCalculationCurrency)
	if err := document.AddTitle(MainReportTitle); err != nil {
		return err
	}
	if err := document.AddKeyValue("Year", fmt.Sprintf("%d", report.Year)); err != nil {
		return err
	}
	if err := document.AddKeyValue("Cost Basis Method", report.CostBasisMethod.Label()); err != nil {
		return err
	}
	if err := document.AddKeyValue("Generated At", report.GeneratedAt.Local().Format("2006-01-02 15:04:05 MST")); err != nil {
		return err
	}
	if err := document.AddKeyValue("Report Calculation Currency", calculationCurrency); err != nil {
		return err
	}
	if err := renderSummarySection(document, report, calculationCurrency); err != nil {
		return err
	}
	if err := renderRateSourceSection(document, report); err != nil {
		return err
	}
	if err := renderReferenceSection(document, report); err != nil {
		return err
	}
	return renderDetailSections(document, report, calculationCurrency)
}

// renderSummarySection renders non-zero summary rows and the yearly total.
// Authored by: OpenCode
func renderSummarySection(document pdfLayoutDocument, report reportmodel.CapitalGainsReport, calculationCurrency string) error {
	if err := document.AddSectionHeading("Gains-And-Losses Summary"); err != nil {
		return err
	}
	var yearlyNetTotal, err = decimalsupport.CanonicalString(report.YearlyNetTotal)
	if err != nil {
		return fmt.Errorf("render yearly net total: %w", err)
	}
	var rows [][]string
	for _, entry := range report.SummaryEntries {
		if entry.NetGainOrLoss.Sign() == 0 {
			continue
		}
		var netGainOrLoss string
		netGainOrLoss, err = decimalsupport.CanonicalString(entry.NetGainOrLoss)
		if err != nil {
			return fmt.Errorf("render summary entry %q net gain or loss: %w", entry.AssetIdentityKey, err)
		}
		rows = append(rows, []string{
			renderDisplayLabel(entry.DisplayLabel, entry.AssetIdentityKey),
			netGainOrLoss,
			calculationCurrencyLabelWithFallback(entry.ReportCalculationCurrency, calculationCurrency),
		})
	}
	if len(rows) == 0 {
		if err := document.AddParagraph("No assets had a non-zero net gain or loss in the selected year."); err != nil {
			return err
		}
		return document.AddKeyValue("Overall Yearly Net Total", yearlyNetTotal+" "+calculationCurrency)
	} else if err := document.AddTable(pdfTable{
		Title:             "Gains-And-Losses Summary Table",
		ContinuationTitle: "Gains-And-Losses Summary Table (continued)",
		Columns: []pdfColumn{
			{Header: "Asset", Width: 220, Align: "left"},
			{Header: "Net Gain Or Loss", Width: 150, Align: "right"},
			{Header: "Report Calculation Currency", Width: 150, Align: "left"},
		},
		Rows:          append(rows, []string{"Overall Yearly Net Total", yearlyNetTotal, calculationCurrency}),
		StyledLastRow: true,
	}); err != nil {
		return err
	}
	return nil
}

// renderRateSourceSection renders provider-level rate source evidence.
// Authored by: OpenCode
func renderRateSourceSection(document pdfLayoutDocument, report reportmodel.CapitalGainsReport) error {
	if err := document.AddSectionHeading("Rate Source Summary"); err != nil {
		return err
	}
	if err := document.AddKeyValue("Report Base Currency", calculationCurrencyLabel(report.ReportCalculationCurrency)); err != nil {
		return err
	}
	if len(report.RateSources) == 0 {
		return document.AddParagraph("Exchange Rate Use: No activity required exchange-rate conversion.")
	}

	var rendered = make(map[string]bool)
	for _, source := range report.RateSources {
		var key = strings.Join([]string{string(source.Authority), string(source.ProviderID), source.RateKind}, "|")
		if rendered[key] {
			continue
		}
		rendered[key] = true
		if err := document.AddKeyValue("Authority", sanitizeText(reportmodel.RateAuthorityDisplayLabel(source.Authority))); err != nil {
			return err
		}
		if err := document.AddKeyValue("Provider", rateProviderLabel(source.ProviderID)); err != nil {
			return err
		}
		if err := document.AddKeyValue("Rate Kind", sanitizeText(source.RateKind)); err != nil {
			return err
		}
		if err := document.AddKeyValue("Unavailable-Date Rule", sanitizeText(reportmodel.RateProviderUnavailableDateRule(source.ProviderID))); err != nil {
			return err
		}
	}
	return nil
}

// renderReferenceSection renders the historical full-liquidation reference rows.
// Authored by: OpenCode
func renderReferenceSection(document pdfLayoutDocument, report reportmodel.CapitalGainsReport) error {
	if err := document.AddSectionHeading("Reference Section"); err != nil {
		return err
	}
	if len(report.ReferenceEntries) == 0 {
		return document.AddParagraph("No assets reached full liquidation by year end.")
	}

	var rows [][]string
	for _, entry := range report.ReferenceEntries {
		rows = append(rows, []string{
			renderDisplayLabel(entry.DisplayLabel, entry.AssetIdentityKey),
			fmt.Sprintf("%d", entry.FullLiquidationCountThroughYearEnd),
			sanitizeText(string(entry.MainSectionStatus)),
		})
	}
	return document.AddTable(pdfTable{
		ContinuationTitle: "Reference Section (continued)",
		Columns: []pdfColumn{
			{Header: "Asset", Width: 170, Align: "left"},
			{Header: "Historical Full Liquidation Count", Width: 190, Align: "right"},
			{Header: "Main Section Status", Width: 160, Align: "left"},
		},
		Rows: rows,
	})
}

// renderDetailSections renders asset detail sections from report-domain rows.
// Authored by: OpenCode
func renderDetailSections(document pdfLayoutDocument, report reportmodel.CapitalGainsReport, calculationCurrency string) error {
	for _, section := range report.DetailSections {
		if err := document.AddSectionHeading("Asset Detail: " + renderDisplayLabel(section.DisplayLabel, section.AssetIdentityKey)); err != nil {
			return err
		}
		if len(section.ActivityRows) == 0 {
			if err := renderPositionBlock(document, "Historical Position", section.ClosingQuantity, section.ClosingCostBasis, section.CalculationCurrency, calculationCurrency); err != nil {
				return fmt.Errorf("render historical position for %q: %w", section.AssetIdentityKey, err)
			}
			continue
		}
		if err := renderPositionBlock(document, "Opening Position", section.OpeningQuantity, section.OpeningCostBasis, section.CalculationCurrency, calculationCurrency); err != nil {
			return fmt.Errorf("render opening position for %q: %w", section.AssetIdentityKey, err)
		}
		if err := renderActivityRows(document, section); err != nil {
			return fmt.Errorf("render in-year activity for %q: %w", section.AssetIdentityKey, err)
		}
		if err := renderLiquidationRows(document, section, calculationCurrency); err != nil {
			return fmt.Errorf("render liquidation calculations for %q: %w", section.AssetIdentityKey, err)
		}
		if err := renderPositionBlock(document, "Closing Position", section.ClosingQuantity, section.ClosingCostBasis, section.CalculationCurrency, calculationCurrency); err != nil {
			return fmt.Errorf("render closing position for %q: %w", section.AssetIdentityKey, err)
		}
	}
	return nil
}

// renderPositionBlock renders one asset position block with styled labels.
// Authored by: OpenCode
func renderPositionBlock(document pdfLayoutDocument, heading string, quantity apd.Decimal, basis apd.Decimal, sectionCurrency string, fallbackCurrency string) error {
	if err := document.AddSubsectionHeading(heading); err != nil {
		return err
	}
	var quantityText, err = decimalsupport.CanonicalString(quantity)
	if err != nil {
		return fmt.Errorf("render quantity: %w", err)
	}
	var basisText string
	basisText, err = decimalsupport.CanonicalString(basis)
	if err != nil {
		return fmt.Errorf("render cost basis: %w", err)
	}
	if err := document.AddKeyValue("Quantity", quantityText); err != nil {
		return err
	}
	if err := document.AddKeyValue("Cost Basis", basisText); err != nil {
		return err
	}
	return document.AddKeyValue("Calculation Currency", calculationCurrencyLabelWithFallback(sectionCurrency, fallbackCurrency))
}

// renderActivityRows renders in-year asset activity as a table.
// Authored by: OpenCode
func renderActivityRows(document pdfLayoutDocument, section reportmodel.AssetDetailSection) error {
	var rows [][]string
	for _, row := range section.ActivityRows {
		var rendered, err = renderActivityRow(row)
		if err != nil {
			return err
		}
		rows = append(rows, rendered)
	}
	return document.AddTable(pdfTable{
		Title:             "In-Year Activity",
		ContinuationTitle: "In-Year Activity (continued)",
		Columns: []pdfColumn{
			{Header: "Date", Width: 52, Align: "left"},
			{Header: "Source ID", Width: 45, Align: "left"},
			{Header: "Type", Width: 42, Align: "left"},
			{Header: "Quantity", Width: 40, Align: "right"},
			{Header: "Unit Price", Width: 40, Align: "right"},
			{Header: "Gross", Width: 38, Align: "right"},
			{Header: "Fee", Width: 34, Align: "right"},
			{Header: "Qty After", Width: 42, Align: "right"},
			{Header: "Basis After", Width: 46, Align: "right"},
			{Header: "Activity Currency", Width: 42, Align: "left"},
			{Header: "Calc Currency", Width: 42, Align: "left"},
			{Header: "Conversion", Width: 52, Align: "left"},
			{Header: "Note", Width: 50, Align: "left"},
		},
		Rows:      rows,
		RowHeight: 32,
	})
}

// renderActivityRow formats one in-year activity row for a PDF table.
// Authored by: OpenCode
func renderActivityRow(row reportmodel.AssetActivityRow) ([]string, error) {
	var quantityText, err = decimalsupport.CanonicalString(row.Quantity)
	if err != nil {
		return nil, fmt.Errorf("render activity row %q quantity: %w", row.SourceID, err)
	}
	var unitPriceText string
	unitPriceText, err = decimalsupport.CanonicalStringPointer(row.UnitPrice)
	if err != nil {
		return nil, fmt.Errorf("render activity row %q unit price: %w", row.SourceID, err)
	}
	var grossValueText string
	grossValueText, err = decimalsupport.CanonicalStringPointer(row.GrossValue)
	if err != nil {
		return nil, fmt.Errorf("render activity row %q gross value: %w", row.SourceID, err)
	}
	var feeText string
	feeText, err = decimalsupport.CanonicalStringPointer(row.FeeAmount)
	if err != nil {
		return nil, fmt.Errorf("render activity row %q fee: %w", row.SourceID, err)
	}
	var basisAfterRowText string
	basisAfterRowText, err = decimalsupport.CanonicalString(row.BasisAfterRow)
	if err != nil {
		return nil, fmt.Errorf("render activity row %q basis after row: %w", row.SourceID, err)
	}
	var quantityAfterRowText string
	quantityAfterRowText, err = decimalsupport.CanonicalString(row.QuantityAfterRow)
	if err != nil {
		return nil, fmt.Errorf("render activity row %q quantity after row: %w", row.SourceID, err)
	}
	var activityTypeLabel, labelErr = reportmodel.RenderActivityTypeLabel(row)
	if labelErr != nil {
		return nil, fmt.Errorf("render activity row %q type label: %w", row.SourceID, labelErr)
	}
	var conversionStatusText string
	conversionStatusText, labelErr = conversionStatusColumn(row)
	if labelErr != nil {
		return nil, fmt.Errorf("render activity row %q conversion status label: %w", row.SourceID, labelErr)
	}
	return []string{
		row.OccurredAt.UTC().Format("2006-01-02 15:04:05"),
		sanitizeText(row.SourceID),
		sanitizeText(activityTypeLabel),
		quantityText,
		unitPriceText,
		grossValueText,
		feeText,
		quantityAfterRowText,
		basisAfterRowText,
		activityCurrencyColumn(row),
		calculationCurrencyLabel(row.CalculationCurrency),
		conversionStatusText,
		sanitizeText(row.HoldingReductionExplanation),
	}, nil
}

// renderLiquidationRows renders priced liquidation rows when present.
// Authored by: OpenCode
func renderLiquidationRows(document pdfLayoutDocument, section reportmodel.AssetDetailSection, fallbackCurrency string) error {
	if len(section.LiquidationSummaries) == 0 {
		return nil
	}
	var rows [][]string
	for _, liquidation := range section.LiquidationSummaries {
		var row, err = renderLiquidationRow(liquidation, fallbackCurrency)
		if err != nil {
			return err
		}
		rows = append(rows, row)
	}
	return document.AddTable(pdfTable{
		Title:             "Liquidation Calculations",
		ContinuationTitle: "Liquidation Calculations (continued)",
		Columns: []pdfColumn{
			{Header: "Date", Width: 72, Align: "left"},
			{Header: "Source ID", Width: 66, Align: "left"},
			{Header: "Disposed Quantity", Width: 76, Align: "right"},
			{Header: "Allocated Basis", Width: 74, Align: "right"},
			{Header: "Net Proceeds", Width: 72, Align: "right"},
			{Header: "Gain Or Loss", Width: 70, Align: "right"},
			{Header: "Calculation Currency", Width: 88, Align: "left"},
		},
		Rows: rows,
	})
}

// renderLiquidationRow formats one liquidation calculation row.
// Authored by: OpenCode
func renderLiquidationRow(liquidation reportmodel.LiquidationCalculation, fallbackCurrency string) ([]string, error) {
	var disposedQuantityText, err = decimalsupport.CanonicalString(liquidation.DisposedQuantity)
	if err != nil {
		return nil, fmt.Errorf("render liquidation %q disposed quantity: %w", liquidation.SourceID, err)
	}
	var allocatedBasisText string
	allocatedBasisText, err = decimalsupport.CanonicalString(liquidation.AllocatedBasis)
	if err != nil {
		return nil, fmt.Errorf("render liquidation %q allocated basis: %w", liquidation.SourceID, err)
	}
	var proceedsText string
	proceedsText, err = decimalsupport.CanonicalString(liquidation.NetLiquidationProceeds)
	if err != nil {
		return nil, fmt.Errorf("render liquidation %q net proceeds: %w", liquidation.SourceID, err)
	}
	var gainOrLossText string
	gainOrLossText, err = decimalsupport.CanonicalString(liquidation.GainOrLoss)
	if err != nil {
		return nil, fmt.Errorf("render liquidation %q gain or loss: %w", liquidation.SourceID, err)
	}
	return []string{
		liquidation.OccurredAt.UTC().Format("2006-01-02 15:04:05"),
		sanitizeText(liquidation.SourceID),
		disposedQuantityText,
		allocatedBasisText,
		proceedsText,
		gainOrLossText,
		calculationCurrencyLabelWithFallback(liquidation.CalculationCurrency, fallbackCurrency),
	}, nil
}

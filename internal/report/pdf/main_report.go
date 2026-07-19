package pdf

import (
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/report/presentation"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// renderMainReport renders the main report through structured PDF operations.
// Authored by: OpenCode
func renderMainReport(document pdfContentLayout, report reportmodel.CapitalGainsReport) error {
	return renderMainReportWithFinancialFormatting(document, report, presentation.DefaultFinancialFormattingOptions())
}

// renderMainReportWithFinancialFormatting renders the main report with one
// renderer-scoped immutable financial policy.
// Authored by: OpenCode
func renderMainReportWithFinancialFormatting(document pdfContentLayout, report reportmodel.CapitalGainsReport, options presentation.FinancialFormattingOptions) error {
	if document == nil {
		return fmt.Errorf("pdf layout document is required")
	}
	if err := report.Validate(); err != nil {
		return err
	}

	var calculationCurrency = calculationCurrencyLabel(report.ReportCalculationCurrency)
	if err := document.AddTitle(MainReportTitle); err != nil {
		return fmt.Errorf("add main report title: %w", err)
	}
	if err := document.AddKeyValue("Year", fmt.Sprintf("%d", report.Year)); err != nil {
		return fmt.Errorf("add year metadata: %w", err)
	}
	if err := document.AddKeyValue("Cost Basis Method", report.CostBasisMethod.Label()); err != nil {
		return fmt.Errorf("add cost basis method metadata: %w", err)
	}
	if err := document.AddKeyValue("Generated At", report.GeneratedAt.Local().Format("2006-01-02 15:04:05 MST")); err != nil {
		return fmt.Errorf("add generated-at metadata: %w", err)
	}
	if err := document.AddKeyValue("Report Calculation Currency", calculationCurrency); err != nil {
		return fmt.Errorf("add report calculation currency metadata: %w", err)
	}
	if err := document.AddBoldParagraph(presentation.LegalWarningText); err != nil {
		return fmt.Errorf("add legal-use warning: %w", err)
	}
	if err := renderSummarySectionWithFinancialFormatting(document, report, calculationCurrency, options); err != nil {
		return err
	}
	if err := renderRateSourceSection(document, report); err != nil {
		return err
	}
	if err := renderReferenceSection(document, report); err != nil {
		return err
	}
	return renderDetailSectionsWithFinancialFormatting(document, report, calculationCurrency, options)
}

// renderSummarySection renders non-zero summary rows and the yearly total.
// Authored by: OpenCode
//
//nolint:unparam // The currency argument is part of the direct renderer seam.
func renderSummarySection(document pdfContentLayout, report reportmodel.CapitalGainsReport, calculationCurrency string) error {
	return renderSummarySectionWithFinancialFormatting(document, report, calculationCurrency, presentation.DefaultFinancialFormattingOptions())
}

// renderSummarySectionWithFinancialFormatting renders summary monetary values
// with one renderer-scoped immutable policy.
// Authored by: OpenCode
func renderSummarySectionWithFinancialFormatting(document pdfContentLayout, report reportmodel.CapitalGainsReport, calculationCurrency string, options presentation.FinancialFormattingOptions) error {
	if err := document.AddSectionHeading("Gains-And-Losses Summary"); err != nil {
		return fmt.Errorf("add gains-and-losses summary heading: %w", err)
	}
	var yearlyNetTotal, err = options.Format(report.YearlyNetTotal)
	if err != nil {
		return fmt.Errorf("render yearly net total: %w", err)
	}
	var rows [][]string
	for index, entry := range report.SummaryEntries {
		if entry.NetGainOrLoss.Sign() == 0 {
			continue
		}
		var netGainOrLoss string
		netGainOrLoss, err = options.Format(entry.NetGainOrLoss)
		if err != nil {
			return fmt.Errorf("render summary entry %d net gain or loss: %w", index+1, err)
		}
		rows = append(rows, []string{
			renderDisplayLabel(entry.DisplayLabel, entry.AssetIdentityKey),
			netGainOrLoss,
			calculationCurrencyLabelWithFallback(entry.ReportCalculationCurrency, calculationCurrency),
		})
	}
	if len(rows) == 0 {
		if err := document.AddParagraph("No assets had a non-zero net gain or loss in the selected year."); err != nil {
			return fmt.Errorf("add empty summary paragraph: %w", err)
		}
		if err := document.AddKeyValue("Overall Yearly Net Total", yearlyNetTotal+" "+calculationCurrency); err != nil {
			return fmt.Errorf("add overall yearly net total: %w", err)
		}
		return nil
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
		return fmt.Errorf("add gains-and-losses summary table: %w", err)
	}
	return nil
}

// renderRateSourceSection renders provider-level rate source evidence.
// Authored by: OpenCode
func renderRateSourceSection(document pdfContentLayout, report reportmodel.CapitalGainsReport) error {
	if err := document.AddSectionHeading("Rate Source Summary"); err != nil {
		return fmt.Errorf("add rate source summary heading: %w", err)
	}
	if err := document.AddKeyValue("Report Base Currency", calculationCurrencyLabel(report.ReportCalculationCurrency)); err != nil {
		return fmt.Errorf("add report base currency: %w", err)
	}
	if len(report.RateSources) == 0 {
		return renderEmptyRateSourceParagraph(document)
	}

	var rendered = make(map[string]bool)
	for _, source := range report.RateSources {
		var key = strings.Join([]string{string(source.Authority), string(source.ProviderID), source.RateKind}, "|")
		if rendered[key] {
			continue
		}
		rendered[key] = true
		if err := document.AddKeyValue("Authority", sanitizeText(reportmodel.RateAuthorityDisplayLabel(source.Authority))); err != nil {
			return fmt.Errorf("add rate source authority: %w", err)
		}
		if err := document.AddKeyValue("Provider", rateProviderLabel(source.ProviderID)); err != nil {
			return fmt.Errorf("add rate source provider: %w", err)
		}
		if err := document.AddKeyValue("Rate Kind", sanitizeText(source.RateKind)); err != nil {
			return fmt.Errorf("add rate source kind: %w", err)
		}
		if err := document.AddKeyValue("Unavailable-Date Rule", sanitizeText(reportmodel.RateProviderUnavailableDateRule(source.ProviderID))); err != nil {
			return fmt.Errorf("add rate source unavailable-date rule: %w", err)
		}
	}
	return nil
}

// renderEmptyRateSourceParagraph renders the rate-source summary when no conversion was required.
// Authored by: OpenCode
func renderEmptyRateSourceParagraph(document pdfContentLayout) error {
	if err := document.AddParagraph("Exchange Rate Use: No activity required exchange-rate conversion."); err != nil {
		return fmt.Errorf("add empty rate source paragraph: %w", err)
	}
	return nil
}

// renderReferenceSection renders the historical full-liquidation reference rows.
// Authored by: OpenCode
func renderReferenceSection(document pdfContentLayout, report reportmodel.CapitalGainsReport) error {
	if err := document.AddSectionHeading("Reference Section"); err != nil {
		return fmt.Errorf("add reference section heading: %w", err)
	}
	if len(report.ReferenceEntries) == 0 {
		if err := document.AddParagraph("No assets reached full liquidation by year end."); err != nil {
			return fmt.Errorf("add empty reference paragraph: %w", err)
		}
		return nil
	}

	var rows [][]string
	for _, entry := range report.ReferenceEntries {
		rows = append(rows, []string{
			renderDisplayLabel(entry.DisplayLabel, entry.AssetIdentityKey),
			fmt.Sprintf("%d", entry.FullLiquidationCountThroughYearEnd),
			sanitizeText(string(entry.MainSectionStatus)),
		})
	}
	if err := document.AddTable(pdfTable{
		ContinuationTitle: "Reference Section (continued)",
		Columns: []pdfColumn{
			{Header: "Asset", Width: 170, Align: "left"},
			{Header: "Historical Full Liquidation Count", Width: 190, Align: "right"},
			{Header: "Main Section Status", Width: 160, Align: "left"},
		},
		Rows: rows,
	}); err != nil {
		return fmt.Errorf("add reference table: %w", err)
	}
	return nil
}

// renderDetailSections renders asset detail sections from report-domain rows.
// Authored by: OpenCode
func renderDetailSections(document pdfContentLayout, report reportmodel.CapitalGainsReport, calculationCurrency string) error {
	return renderDetailSectionsWithFinancialFormatting(document, report, calculationCurrency, presentation.DefaultFinancialFormattingOptions())
}

// renderDetailSectionsWithFinancialFormatting renders all detail sections with
// one renderer-scoped immutable financial policy.
// Authored by: OpenCode
func renderDetailSectionsWithFinancialFormatting(document pdfContentLayout, report reportmodel.CapitalGainsReport, calculationCurrency string, options presentation.FinancialFormattingOptions) error {
	for index, section := range report.DetailSections {
		if err := renderDetailSectionWithFinancialFormatting(document, section, calculationCurrency, options); err != nil {
			return fmt.Errorf("render asset detail section %d: %w", index+1, err)
		}
	}
	return nil
}

// renderDetailSectionWithFinancialFormatting renders one detail section with a
// renderer-scoped immutable financial policy.
// Authored by: OpenCode
func renderDetailSectionWithFinancialFormatting(document pdfContentLayout, section reportmodel.AssetDetailSection, calculationCurrency string, options presentation.FinancialFormattingOptions) error {
	if err := document.AddSectionHeading("Asset Detail: " + renderDisplayLabel(section.DisplayLabel, section.AssetIdentityKey)); err != nil {
		return fmt.Errorf("add asset detail heading: %w", err)
	}
	if len(section.ActivityRows) == 0 {
		return renderHistoricalDetailSectionWithFinancialFormatting(document, section, calculationCurrency, options)
	}
	return renderActiveDetailSectionWithFinancialFormatting(document, section, calculationCurrency, options)
}

// renderHistoricalDetailSectionWithFinancialFormatting renders a historical
// position with a renderer-scoped immutable financial policy.
// Authored by: OpenCode
func renderHistoricalDetailSectionWithFinancialFormatting(document pdfContentLayout, section reportmodel.AssetDetailSection, currency string, options presentation.FinancialFormattingOptions) error {
	if err := renderPositionBlockWithFinancialFormatting(document, "Historical Position", section.ClosingQuantity, section.ClosingCostBasis, section.CalculationCurrency, currency, options); err != nil {
		return fmt.Errorf("render historical position: %w", err)
	}
	return nil
}

// renderActiveDetailSectionWithFinancialFormatting renders active detail blocks
// with a renderer-scoped immutable financial policy.
// Authored by: OpenCode
func renderActiveDetailSectionWithFinancialFormatting(document pdfContentLayout, section reportmodel.AssetDetailSection, currency string, options presentation.FinancialFormattingOptions) error {
	if err := renderPositionBlockWithFinancialFormatting(document, "Opening Position", section.OpeningQuantity, section.OpeningCostBasis, section.CalculationCurrency, currency, options); err != nil {
		return fmt.Errorf("render opening position: %w", err)
	}
	if err := renderActivityRowsWithFinancialFormatting(document, section, options); err != nil {
		return fmt.Errorf("render in-year activity: %w", err)
	}
	if err := renderLiquidationRowsWithFinancialFormatting(document, section, currency, options); err != nil {
		return fmt.Errorf("render liquidation calculations: %w", err)
	}
	if err := renderPositionBlockWithFinancialFormatting(document, "Closing Position", section.ClosingQuantity, section.ClosingCostBasis, section.CalculationCurrency, currency, options); err != nil {
		return fmt.Errorf("render closing position: %w", err)
	}
	return nil
}

// renderPositionBlock renders one asset position block with styled labels.
// Authored by: OpenCode
//
//nolint:unparam // Both currency arguments remain part of the direct renderer seam.
func renderPositionBlock(document pdfContentLayout, heading string, quantity apd.Decimal, basis apd.Decimal, sectionCurrency string, fallbackCurrency string) error {
	return renderPositionBlockWithFinancialFormatting(document, heading, quantity, basis, sectionCurrency, fallbackCurrency, presentation.DefaultFinancialFormattingOptions())
}

// renderPositionBlockWithFinancialFormatting renders one position block with a
// renderer-scoped immutable financial policy.
// Authored by: OpenCode
func renderPositionBlockWithFinancialFormatting(document pdfContentLayout, heading string, quantity apd.Decimal, basis apd.Decimal, sectionCurrency string, fallbackCurrency string, options presentation.FinancialFormattingOptions) error {
	if err := document.AddSubsectionHeading(heading); err != nil {
		return fmt.Errorf("add %s heading: %w", heading, err)
	}
	var quantityText, err = decimalsupport.CanonicalString(quantity)
	if err != nil {
		return fmt.Errorf("render quantity: %w", err)
	}
	var basisText string
	basisText, err = options.Format(basis)
	if err != nil {
		return fmt.Errorf("render cost basis: %w", err)
	}
	if err := document.AddKeyValue("Quantity", quantityText); err != nil {
		return fmt.Errorf("add %s quantity: %w", heading, err)
	}
	if err := document.AddKeyValue("Cost Basis", basisText); err != nil {
		return fmt.Errorf("add %s cost basis: %w", heading, err)
	}
	if err := document.AddKeyValue("Calculation Currency", calculationCurrencyLabelWithFallback(sectionCurrency, fallbackCurrency)); err != nil {
		return fmt.Errorf("add %s calculation currency: %w", heading, err)
	}
	return nil
}

// renderActivityRows renders in-year asset activity as a table.
// Authored by: OpenCode
func renderActivityRows(document pdfContentLayout, section reportmodel.AssetDetailSection) error {
	return renderActivityRowsWithFinancialFormatting(document, section, presentation.DefaultFinancialFormattingOptions())
}

// renderActivityRowsWithFinancialFormatting renders activity rows with one
// renderer-scoped immutable financial policy.
// Authored by: OpenCode
func renderActivityRowsWithFinancialFormatting(document pdfContentLayout, section reportmodel.AssetDetailSection, options presentation.FinancialFormattingOptions) error {
	var rows [][]string
	for index, row := range section.ActivityRows {
		var rendered, err = renderActivityRowWithFinancialFormatting(row, options)
		if err != nil {
			return fmt.Errorf("render activity row %d: %w", index+1, err)
		}
		rows = append(rows, rendered)
	}
	if err := document.AddTable(pdfTable{
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
	}); err != nil {
		return fmt.Errorf("add in-year activity table: %w", err)
	}
	return nil
}

// renderActivityRow formats one in-year activity row for a PDF table.
// Authored by: OpenCode
func renderActivityRow(row reportmodel.AssetActivityRow) ([]string, error) {
	return renderActivityRowWithFinancialFormatting(row, presentation.DefaultFinancialFormattingOptions())
}

// renderActivityRowWithFinancialFormatting renders one activity row with a
// renderer-scoped immutable financial policy.
// Authored by: OpenCode
func renderActivityRowWithFinancialFormatting(row reportmodel.AssetActivityRow, options presentation.FinancialFormattingOptions) ([]string, error) {
	var rendered, err = presentation.BuildActivityRowWithFinancialFormatting(row, options)
	if err != nil {
		return nil, err
	}
	return []string{
		rendered.Date, sanitizeText(rendered.SourceID), sanitizeText(rendered.ActivityType), rendered.Quantity, rendered.UnitPrice, rendered.GrossValue, rendered.Fee, rendered.QuantityAfterRow, rendered.BasisAfterRow, sanitizeText(rendered.ActivityCurrency), sanitizeText(rendered.CalculationCurrency), sanitizeText(rendered.ConversionStatus), sanitizeText(rendered.Note),
	}, nil
}

// renderLiquidationRows renders priced liquidation rows when present.
// Authored by: OpenCode
func renderLiquidationRows(document pdfContentLayout, section reportmodel.AssetDetailSection, fallbackCurrency string) error {
	return renderLiquidationRowsWithFinancialFormatting(document, section, fallbackCurrency, presentation.DefaultFinancialFormattingOptions())
}

// renderLiquidationRowsWithFinancialFormatting renders liquidation rows with a
// renderer-scoped immutable financial policy.
// Authored by: OpenCode
func renderLiquidationRowsWithFinancialFormatting(document pdfContentLayout, section reportmodel.AssetDetailSection, fallbackCurrency string, options presentation.FinancialFormattingOptions) error {
	if len(section.LiquidationSummaries) == 0 {
		return nil
	}
	var rows [][]string
	for index, liquidation := range section.LiquidationSummaries {
		var row, err = renderLiquidationRowWithFinancialFormatting(liquidation, fallbackCurrency, options)
		if err != nil {
			return fmt.Errorf("render liquidation row %d: %w", index+1, err)
		}
		rows = append(rows, row)
	}
	if err := document.AddTable(pdfTable{
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
	}); err != nil {
		return fmt.Errorf("add liquidation calculations table: %w", err)
	}
	return nil
}

// renderLiquidationRow formats one liquidation calculation row.
// Authored by: OpenCode
func renderLiquidationRow(liquidation reportmodel.LiquidationCalculation, fallbackCurrency string) ([]string, error) {
	return renderLiquidationRowWithFinancialFormatting(liquidation, fallbackCurrency, presentation.DefaultFinancialFormattingOptions())
}

// renderLiquidationRowWithFinancialFormatting renders one liquidation row with
// a renderer-scoped immutable financial policy.
// Authored by: OpenCode
func renderLiquidationRowWithFinancialFormatting(liquidation reportmodel.LiquidationCalculation, fallbackCurrency string, options presentation.FinancialFormattingOptions) ([]string, error) {
	var rendered, err = presentation.BuildLiquidationRowWithFinancialFormatting(liquidation, fallbackCurrency, options)
	if err != nil {
		return nil, err
	}
	return []string{
		rendered.Date, sanitizeText(rendered.SourceID), rendered.DisposedQuantity, rendered.AllocatedBasis, rendered.NetProceeds, rendered.GainOrLoss, sanitizeText(rendered.CalculationCurrency),
	}, nil
}

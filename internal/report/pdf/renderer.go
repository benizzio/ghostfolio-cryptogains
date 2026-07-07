// Package pdf defines the local PDF rendering boundary for calculated yearly
// gains-and-losses reports.
//
// The renderer is intentionally scoped to in-process, local-only PDF generation
// under internal/report/pdf. It renders A4, text-based report output from report
// domain models through gopdf layout primitives so generated report text remains
// searchable and selectable in PDF readers that support text selection. The
// package accepts application-supplied font bytes and must not read platform font
// paths, call browser services, use external PDF binaries, contact remote
// rendering services, emit telemetry, or persist report state.
// Authored by: OpenCode
package pdf

import (
	"bytes"
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	datesupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/date"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	textsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/text"
	"github.com/cockroachdb/apd/v3"
	"github.com/signintech/gopdf"
)

const (
	// PageSizeA4 identifies the only supported page size for report PDF output.
	PageSizeA4 = "A4"

	// MainReportTitle identifies the required first-page PDF report title.
	MainReportTitle = "Ghostfolio Capital Gains And Losses Report"

	// AnnexTitle identifies the required Annex 1 PDF page title.
	AnnexTitle = "Annex 1 - Audit"

	fontRegular = "regular"
	fontBold    = "bold"

	pageMargin  = 36.0
	pageBottom  = 806.0
	contentWide = 523.0
)

// FontData stores application-supplied font bytes used by the PDF renderer.
// Authored by: OpenCode
type FontData struct {
	Regular []byte
	Bold    []byte
}

// Validate verifies that the renderer has the application-supplied fonts needed
// for deterministic local PDF text output.
//
// Example:
//
//	fonts := pdf.FontData{Regular: regularTTF, Bold: boldTTF}
//	if err := fonts.Validate(); err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func (fonts FontData) Validate() error {
	if len(fonts.Regular) == 0 {
		return fmt.Errorf("regular font data is required")
	}
	if len(fonts.Bold) == 0 {
		return fmt.Errorf("bold font data is required")
	}

	return nil
}

// RenderOptions stores local PDF renderer configuration.
// Authored by: OpenCode
type RenderOptions struct {
	Fonts FontData
}

// Validate verifies local PDF renderer options before a render attempt.
//
// Example:
//
//	options := pdf.RenderOptions{Fonts: pdf.FontData{Regular: regularTTF, Bold: boldTTF}}
//	if err := options.Validate(); err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func (options RenderOptions) Validate() error {
	if err := options.Fonts.Validate(); err != nil {
		return fmt.Errorf("font data: %w", err)
	}

	return nil
}

// Renderer renders one calculated report into a local A4 PDF byte payload.
// Authored by: OpenCode
type Renderer struct {
	options RenderOptions
}

// pdfDocumentStarter is the minimal seam used to verify A4 document startup.
// Authored by: OpenCode
type pdfDocumentStarter interface {
	StartPDF(pageSize string) error
}

// fontLoader is the minimal seam used to verify application-supplied font
// registration without platform font paths.
// Authored by: OpenCode
type fontLoader interface {
	AddTTFFont(name string, data []byte) error
}

// pdfLayoutDocument is the complete structured document seam used by Render.
// Authored by: OpenCode
type pdfLayoutDocument interface {
	pdfDocumentStarter
	fontLoader
	AddTitle(text string) error
	AddSectionHeading(text string) error
	AddSubsectionHeading(text string) error
	AddKeyValue(label string, value string) error
	AddParagraph(text string) error
	AddTable(table pdfTable) error
	AddAnnexPageBreak() error
	Bytes() []byte
}

// pdfColumn describes one PDF table column.
// Authored by: OpenCode
type pdfColumn struct {
	Header string
	Width  float64
	Align  string
}

// pdfTable describes one structured PDF table rendered through gopdf layout APIs.
// Authored by: OpenCode
type pdfTable struct {
	Title             string
	ContinuationTitle string
	Columns           []pdfColumn
	Rows              [][]string
	StyledLastRow     bool
	RowHeight         float64
}

// newPDFDocumentForRenderer keeps concrete PDF adapter startup failures
// testable without involving external files or platform fonts.
// Authored by: OpenCode
var newPDFDocumentForRenderer = func() pdfLayoutDocument {
	return newGopdfDocument()
}

// writeTextForGopdfDocument keeps concrete gopdf text failures testable.
// Authored by: OpenCode
var writeTextForGopdfDocument = func(document *gopdfDocument, text string) error {
	return document.pdf.Text(text)
}

// writeCellForGopdfDocument keeps concrete gopdf cell failures testable.
// Authored by: OpenCode
var writeCellForGopdfDocument = func(document *gopdfDocument, rectangle *gopdf.Rect, text string) error {
	return document.pdf.Cell(rectangle, text)
}

// writeMultiCellForGopdfDocument keeps concrete gopdf wrapped-text failures
// testable.
// Authored by: OpenCode
var writeMultiCellForGopdfDocument = func(document *gopdfDocument, rectangle *gopdf.Rect, text string) error {
	return document.pdf.MultiCell(rectangle, text)
}

// drawTableForGopdfDocument keeps concrete gopdf table failures testable.
// Authored by: OpenCode
var drawTableForGopdfDocument = func(table gopdf.TableLayout) error {
	return table.DrawTable()
}

// NewRenderer creates one validated local PDF renderer.
//
// Example:
//
//	renderer, err := pdf.NewRenderer(pdf.RenderOptions{
//		Fonts: pdf.FontData{Regular: regularTTF, Bold: boldTTF},
//	})
//	if err != nil {
//		panic(err)
//	}
//	_ = renderer
//
// Authored by: OpenCode
func NewRenderer(options RenderOptions) (Renderer, error) {
	if err := options.Validate(); err != nil {
		return Renderer{}, err
	}

	return Renderer{options: options}, nil
}

// Render validates the calculated report and returns rendered PDF bytes.
//
// Example:
//
//	renderer, err := pdf.NewRenderer(pdf.RenderOptions{
//		Fonts: pdf.FontData{Regular: regularTTF, Bold: boldTTF},
//	})
//	if err != nil {
//		panic(err)
//	}
//	payload, err := renderer.Render(report)
//	if err != nil {
//		panic(err)
//	}
//	_ = payload
//
// Authored by: OpenCode
func (renderer Renderer) Render(report reportmodel.CapitalGainsReport) ([]byte, error) {
	if err := renderer.options.Validate(); err != nil {
		return nil, err
	}
	if err := report.Validate(); err != nil {
		return nil, err
	}

	var document = newPDFDocumentForRenderer()
	if err := startPDFDocument(document); err != nil {
		return nil, err
	}
	if err := loadApplicationFonts(document, renderer.options.Fonts); err != nil {
		return nil, err
	}
	if err := renderMainReport(document, report); err != nil {
		return nil, err
	}
	if err := document.AddAnnexPageBreak(); err != nil {
		return nil, err
	}
	if err := renderAnnex(document, report.AuditAnnex); err != nil {
		return nil, err
	}

	return document.Bytes(), nil
}

// startPDFDocument starts one A4 PDF document through the renderer seam.
// Authored by: OpenCode
func startPDFDocument(document pdfDocumentStarter) error {
	if document == nil {
		return fmt.Errorf("pdf document starter is required")
	}

	return document.StartPDF(PageSizeA4)
}

// loadApplicationFonts registers regular and bold application-supplied fonts.
// Authored by: OpenCode
func loadApplicationFonts(loader fontLoader, fonts FontData) error {
	if loader == nil {
		return fmt.Errorf("pdf font loader is required")
	}
	if err := fonts.Validate(); err != nil {
		return err
	}
	if err := loader.AddTTFFont(fontRegular, fonts.Regular); err != nil {
		return fmt.Errorf("load regular font: %w", err)
	}
	if err := loader.AddTTFFont(fontBold, fonts.Bold); err != nil {
		return fmt.Errorf("load bold font: %w", err)
	}

	return nil
}

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
	var rows [][]string
	for _, entry := range report.SummaryEntries {
		if entry.NetGainOrLoss.Sign() == 0 {
			continue
		}
		var netGainOrLoss, err = decimalsupport.CanonicalString(entry.NetGainOrLoss)
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
	} else if err := document.AddTable(pdfTable{
		Title:             "Gains-And-Losses Summary Table",
		ContinuationTitle: "Gains-And-Losses Summary Table (continued)",
		Columns: []pdfColumn{
			{Header: "Asset", Width: 220, Align: "left"},
			{Header: "Net Gain Or Loss", Width: 150, Align: "right"},
			{Header: "Report Calculation Currency", Width: 150, Align: "left"},
		},
		Rows: rows,
	}); err != nil {
		return err
	}

	var yearlyNetTotal, err = decimalsupport.CanonicalString(report.YearlyNetTotal)
	if err != nil {
		return fmt.Errorf("render yearly net total: %w", err)
	}
	return document.AddKeyValue("Overall Yearly Net Total", yearlyNetTotal+" "+calculationCurrency)
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
	var rows [][]string
	for _, source := range report.RateSources {
		var key = strings.Join([]string{string(source.Authority), string(source.ProviderID), source.RateKind}, "|")
		if rendered[key] {
			continue
		}
		rendered[key] = true
		rows = append(rows, []string{
			sanitizeText(reportmodel.RateAuthorityDisplayLabel(source.Authority)),
			rateProviderLabel(source.ProviderID),
			sanitizeText(source.RateKind),
			sanitizeText(reportmodel.RateProviderUnavailableDateRule(source.ProviderID)),
		})
	}
	return document.AddTable(pdfTable{
		Title:             "Rate Source Summary Table",
		ContinuationTitle: "Rate Source Summary Table (continued)",
		Columns: []pdfColumn{
			{Header: "Authority", Width: 115, Align: "left"},
			{Header: "Provider", Width: 125, Align: "left"},
			{Header: "Rate Kind", Width: 140, Align: "left"},
			{Header: "Unavailable-Date Rule", Width: 140, Align: "left"},
		},
		Rows: rows,
	})
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
		Title:             "Reference Table",
		ContinuationTitle: "Reference Table (continued)",
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

// gopdfDocument renders selectable text through gopdf while retaining extracted
// report text comments for deterministic automated assertions.
// Authored by: OpenCode
type gopdfDocument struct {
	pdf     gopdf.GoPdf
	y       float64
	texts   []string
	started bool
}

// newGopdfDocument creates one local PDF document adapter.
// Authored by: OpenCode
func newGopdfDocument() *gopdfDocument {
	return &gopdfDocument{y: pageMargin}
}

// StartPDF starts one A4 PDF document.
// Authored by: OpenCode
func (document *gopdfDocument) StartPDF(pageSize string) error {
	if pageSize != PageSizeA4 {
		return fmt.Errorf("unsupported PDF page size %q", pageSize)
	}
	document.pdf = gopdf.GoPdf{}
	document.pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	document.pdf.AddPage()
	document.started = true
	return nil
}

// AddTTFFont registers one application-supplied font through gopdf.
// Authored by: OpenCode
func (document *gopdfDocument) AddTTFFont(name string, data []byte) error {
	if !document.started {
		return fmt.Errorf("PDF document must be started before loading fonts")
	}
	return document.pdf.AddTTFFontByReader(name, bytes.NewReader(data))
}

// AddTitle emits a top-level PDF heading with bold font styling.
// Authored by: OpenCode
func (document *gopdfDocument) AddTitle(text string) error {
	return document.addTextBlock(text, fontBold, 16, 24)
}

// AddSectionHeading emits a section heading with bold font styling.
// Authored by: OpenCode
func (document *gopdfDocument) AddSectionHeading(text string) error {
	return document.addTextBlock(text, fontBold, 12, 18)
}

// AddSubsectionHeading emits a subsection heading with bold font styling.
// Authored by: OpenCode
func (document *gopdfDocument) AddSubsectionHeading(text string) error {
	return document.addTextBlock(text, fontBold, 10, 16)
}

// AddKeyValue emits one styled label/value row using Cell and Text operations.
// Authored by: OpenCode
func (document *gopdfDocument) AddKeyValue(label string, value string) error {
	if err := document.ensureSpace(16); err != nil {
		return err
	}
	var labelText = sanitizeText(label) + ":"
	var valueText = sanitizeText(value)
	document.pdf.SetXY(pageMargin, document.y)
	if err := document.pdf.SetFont(fontBold, "", 9); err != nil {
		return err
	}
	if err := writeCellForGopdfDocument(document, &gopdf.Rect{W: 150, H: 12}, labelText); err != nil {
		return err
	}
	if err := document.pdf.SetFont(fontRegular, "", 9); err != nil {
		return err
	}
	document.pdf.SetXY(pageMargin+154, document.y)
	if err := writeTextForGopdfDocument(document, valueText); err != nil {
		return err
	}
	document.recordText(labelText + " " + valueText)
	document.y += 14
	return nil
}

// AddParagraph emits wrapped paragraph text through MultiCell.
// Authored by: OpenCode
func (document *gopdfDocument) AddParagraph(text string) error {
	if err := document.ensureSpace(34); err != nil {
		return err
	}
	var sanitized = sanitizeText(text)
	if err := document.pdf.SetFont(fontRegular, "", 9); err != nil {
		return err
	}
	document.pdf.SetXY(pageMargin, document.y)
	if err := writeMultiCellForGopdfDocument(document, &gopdf.Rect{W: contentWide, H: 30}, sanitized); err != nil {
		return err
	}
	document.recordText(sanitized)
	document.y += 34
	return nil
}

// AddTable emits one structured table through gopdf table layout primitives.
// Authored by: OpenCode
func (document *gopdfDocument) AddTable(table pdfTable) error {
	if len(table.Columns) == 0 {
		return fmt.Errorf("pdf table columns are required")
	}
	if len(table.Rows) == 0 {
		return nil
	}

	var rowHeight = table.RowHeight
	if rowHeight <= 0 {
		rowHeight = 24
	}
	var remainingRows = table.Rows
	var firstChunk = true
	for len(remainingRows) > 0 {
		var capacity = document.tableRowCapacity(rowHeight)
		if capacity < 1 {
			document.addContinuationPage(table.ContinuationTitle)
			capacity = document.tableRowCapacity(rowHeight)
		}
		if capacity > len(remainingRows) {
			capacity = len(remainingRows)
		}
		var chunk = remainingRows[:capacity]
		if err := document.drawTableChunk(table, chunk, rowHeight, firstChunk && len(remainingRows) == len(chunk)); err != nil {
			return err
		}
		remainingRows = remainingRows[capacity:]
		firstChunk = false
		if len(remainingRows) > 0 {
			document.addContinuationPage(table.ContinuationTitle)
		}
	}
	return nil
}

// AddAnnexPageBreak starts Annex 1 on a new page.
// Authored by: OpenCode
func (document *gopdfDocument) AddAnnexPageBreak() error {
	document.pdf.AddPage()
	document.y = pageMargin
	document.recordText("PAGE BREAK: Annex 1")
	return nil
}

// addTextBlock emits a title or heading through gopdf Text.
// Authored by: OpenCode
func (document *gopdfDocument) addTextBlock(text string, font string, size float64, verticalAdvance float64) error {
	if err := document.ensureSpace(verticalAdvance); err != nil {
		return err
	}
	var sanitized = sanitizeText(text)
	if err := document.pdf.SetFont(font, "", size); err != nil {
		return err
	}
	document.pdf.SetXY(pageMargin, document.y)
	if err := writeTextForGopdfDocument(document, sanitized); err != nil {
		return err
	}
	document.recordText(sanitized)
	document.y += verticalAdvance
	return nil
}

// ensureSpace adds a continuation page before content would leave the A4 area.
// Authored by: OpenCode
func (document *gopdfDocument) ensureSpace(height float64) error {
	if !document.started {
		return fmt.Errorf("PDF document must be started before adding content")
	}
	if document.y+height <= pageBottom {
		return nil
	}
	document.addContinuationPage("Continued")
	return nil
}

// tableRowCapacity returns the number of data rows that fit on the current page.
// Authored by: OpenCode
func (document *gopdfDocument) tableRowCapacity(rowHeight float64) int {
	var available = pageBottom - document.y - rowHeight
	if available <= 0 {
		return 0
	}
	if available < rowHeight {
		return 1
	}
	return int(available / rowHeight)
}

// drawTableChunk draws one page-local table chunk and records its text extract.
// Authored by: OpenCode
func (document *gopdfDocument) drawTableChunk(table pdfTable, rows [][]string, rowHeight float64, includeStyledLastRow bool) error {
	if table.Title != "" {
		if err := document.AddSubsectionHeading(table.Title); err != nil {
			return err
		}
	}
	var layout = document.pdf.NewTableLayout(pageMargin, document.y, rowHeight, len(rows))
	for _, column := range table.Columns {
		layout.AddColumn(sanitizeText(column.Header), column.Width, column.Align)
	}
	layout.SetTableStyle(tableStyle())
	layout.SetHeaderStyle(headerStyle())
	layout.SetCellStyle(cellStyle())
	for rowIndex, row := range rows {
		var sanitizedRow = sanitizeRow(row)
		if includeStyledLastRow && table.StyledLastRow && rowIndex == len(rows)-1 {
			layout.AddStyledRow(styledRowCells(sanitizedRow))
		} else {
			layout.AddRow(sanitizedRow)
		}
	}
	if err := drawTableForGopdfDocument(layout); err != nil {
		return err
	}
	document.recordTable(table.Columns, rows)
	document.y += rowHeight*float64(len(rows)+1) + 12
	return nil
}

// addContinuationPage starts a new page with repeated context.
// Authored by: OpenCode
func (document *gopdfDocument) addContinuationPage(context string) {
	document.pdf.AddPage()
	document.y = pageMargin
	var label = sanitizeText(context)
	if label == "" {
		label = "Continued"
	}
	document.recordText("CONTINUED: " + label)
}

// recordTable records table headers and rows for deterministic test assertions.
// Authored by: OpenCode
func (document *gopdfDocument) recordTable(columns []pdfColumn, rows [][]string) {
	var headers []string
	for _, column := range columns {
		headers = append(headers, sanitizeText(column.Header))
	}
	document.recordText(strings.Join(headers, "\t"))
	for _, row := range rows {
		document.recordText(strings.Join(sanitizeRow(row), "\t"))
	}
}

// recordText appends one sanitized extract line.
// Authored by: OpenCode
func (document *gopdfDocument) recordText(text string) {
	document.texts = append(document.texts, sanitizeText(text))
}

// Bytes returns the PDF byte payload with deterministic text comments for tests.
// Authored by: OpenCode
func (document *gopdfDocument) Bytes() []byte {
	var payload = append([]byte(nil), document.pdf.GetBytesPdf()...)
	var comments bytes.Buffer
	comments.WriteString("\n% ghostfolio-cryptogains text extract\n")
	for _, text := range document.texts {
		comments.WriteString("% ")
		comments.WriteString(strings.ReplaceAll(text, "\n", " "))
		comments.WriteByte('\n')
	}
	return append(payload, comments.Bytes()...)
}

// tableStyle returns the base table border style.
// Authored by: OpenCode
func tableStyle() gopdf.CellStyle {
	return gopdf.CellStyle{BorderStyle: gopdf.BorderStyle{Top: true, Left: true, Right: true, Bottom: true, Width: 0.4, RGBColor: gopdf.RGBColor{R: 90, G: 90, B: 90}}}
}

// headerStyle returns the table header style.
// Authored by: OpenCode
func headerStyle() gopdf.CellStyle {
	return gopdf.CellStyle{BorderStyle: gopdf.BorderStyle{Top: true, Left: true, Right: true, Bottom: true, Width: 0.4, RGBColor: gopdf.RGBColor{R: 70, G: 70, B: 70}}, FillColor: gopdf.RGBColor{R: 225, G: 230, B: 236}, TextColor: gopdf.RGBColor{R: 0, G: 0, B: 0}, Font: fontBold, FontSize: 7}
}

// cellStyle returns the default table cell style.
// Authored by: OpenCode
func cellStyle() gopdf.CellStyle {
	return gopdf.CellStyle{BorderStyle: gopdf.BorderStyle{Top: true, Left: true, Right: true, Bottom: true, Width: 0.3, RGBColor: gopdf.RGBColor{R: 120, G: 120, B: 120}}, FillColor: gopdf.RGBColor{R: 255, G: 255, B: 255}, TextColor: gopdf.RGBColor{R: 0, G: 0, B: 0}, Font: fontRegular, FontSize: 6.5}
}

// highlightedCellStyle returns the emphasized row style.
// Authored by: OpenCode
func highlightedCellStyle() gopdf.CellStyle {
	return gopdf.CellStyle{BorderStyle: gopdf.BorderStyle{Top: true, Left: true, Right: true, Bottom: true, Width: 0.4, RGBColor: gopdf.RGBColor{R: 80, G: 80, B: 80}}, FillColor: gopdf.RGBColor{R: 245, G: 247, B: 250}, TextColor: gopdf.RGBColor{R: 0, G: 0, B: 0}, Font: fontBold, FontSize: 6.5}
}

// styledRowCells converts strings into highlighted gopdf row cells.
// Authored by: OpenCode
func styledRowCells(row []string) []gopdf.RowCell {
	var cells = make([]gopdf.RowCell, 0, len(row))
	var style = highlightedCellStyle()
	for _, cell := range row {
		cells = append(cells, gopdf.NewRowCell(cell, style))
	}
	return cells
}

// sanitizeRow returns a sanitized copy of one table row.
// Authored by: OpenCode
func sanitizeRow(row []string) []string {
	var sanitized = make([]string, 0, len(row))
	for _, cell := range row {
		sanitized = append(sanitized, sanitizeText(cell))
	}
	return sanitized
}

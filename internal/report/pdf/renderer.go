// Package pdf defines the local PDF rendering boundary for calculated yearly
// gains-and-losses reports.
//
// The renderer is intentionally scoped to in-process, local-only PDF generation
// under internal/report/pdf. It is reserved for A4, text-based report output so
// generated report text can remain searchable and selectable in PDF readers that
// support text selection. The package accepts application-supplied font bytes and
// must not read platform font paths, call browser services, use external PDF
// binaries, contact remote rendering services, emit telemetry, or persist report
// state.
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
)

// FontData stores application-supplied font bytes used by the PDF renderer.
//
// The final renderer will load these bytes from deterministic in-application font
// data instead of platform font paths or user-installed fonts.
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
//
// The package currently supports only A4 output and application-supplied fonts.
// More layout controls should remain private until a report contract requires a
// caller-visible option.
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
//
// Renderer instances are configured with application-supplied font bytes. They
// do not own file writing, output filename selection, post-save opening, or any
// persisted report state.
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

// selectableTextEmitter is the minimal seam used to verify selectable report
// text and the Annex 1 page break.
// Authored by: OpenCode
type selectableTextEmitter interface {
	AddText(text string) error
	AddAnnexPageBreak() error
}

// newPDFDocumentForRenderer keeps concrete PDF adapter startup failures
// testable without involving external files or platform fonts.
// Authored by: OpenCode
var newPDFDocumentForRenderer = func() pdfDocument {
	return newGopdfDocument()
}

// writeTextForGopdfDocument keeps concrete gopdf text failures testable while
// preserving one adapter method as the single call site for selectable text.
// Authored by: OpenCode
var writeTextForGopdfDocument = func(document *gopdfDocument, text string) error {
	return document.pdf.Text(text)
}

// buildMainReportLinesForPDF keeps main-report PDF layout failures testable at
// the render emission boundary.
// Authored by: OpenCode
var buildMainReportLinesForPDF = buildMainReportLines

// buildAnnexLinesForPDF keeps Annex 1 PDF layout failures testable at the render
// emission boundary.
// Authored by: OpenCode
var buildAnnexLinesForPDF = buildAnnexLines

// pdfDocument is the complete concrete document seam used by Renderer.Render.
// Authored by: OpenCode
type pdfDocument interface {
	pdfDocumentStarter
	fontLoader
	selectableTextEmitter
	Bytes() []byte
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

// Render validates the calculated report and returns the rendered PDF bytes.
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
// The setup skeleton validates the existing report boundary and then returns
// ErrRendererNotImplemented until the local A4 text renderer is implemented.
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
	if err := emitMainAndAnnexShell(document, report); err != nil {
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

// loadApplicationFonts registers the application-supplied regular and bold font
// bytes through the renderer seam.
// Authored by: OpenCode
func loadApplicationFonts(loader fontLoader, fonts FontData) error {
	if loader == nil {
		return fmt.Errorf("pdf font loader is required")
	}
	if err := fonts.Validate(); err != nil {
		return err
	}
	if err := loader.AddTTFFont("regular", fonts.Regular); err != nil {
		return fmt.Errorf("load regular font: %w", err)
	}
	if err := loader.AddTTFFont("bold", fonts.Bold); err != nil {
		return fmt.Errorf("load bold font: %w", err)
	}

	return nil
}

// emitMainAndAnnexShell emits selectable PDF presentation text from report-domain
// values without passing Markdown-rendered source into the PDF body.
// Authored by: OpenCode
func emitMainAndAnnexShell(emitter selectableTextEmitter, report reportmodel.CapitalGainsReport) error {
	if emitter == nil {
		return fmt.Errorf("pdf text emitter is required")
	}
	if err := report.Validate(); err != nil {
		return err
	}

	var lines, err = buildMainReportLinesForPDF(report)
	if err != nil {
		return err
	}
	for _, line := range lines {
		if err := emitter.AddText(line); err != nil {
			return err
		}
	}
	if err := emitter.AddAnnexPageBreak(); err != nil {
		return err
	}
	var annexLines []string
	annexLines, err = buildAnnexLinesForPDF(report.AuditAnnex)
	if err != nil {
		return err
	}
	for _, line := range annexLines {
		if err := emitter.AddText(line); err != nil {
			return err
		}
	}

	return nil
}

// buildMainReportLines formats the PDF main report directly from report-domain
// fields using line-oriented PDF presentation text.
// Authored by: OpenCode
func buildMainReportLines(report reportmodel.CapitalGainsReport) ([]string, error) {
	var calculationCurrency = calculationCurrencyLabel(report.ReportCalculationCurrency)
	var lines = []string{
		MainReportTitle,
		fmt.Sprintf("Year: %d", report.Year),
		fmt.Sprintf("Cost Basis Method: %s", report.CostBasisMethod.Label()),
		fmt.Sprintf("Generated At: %s", report.GeneratedAt.Local().Format("2006-01-02 15:04:05 MST")),
		fmt.Sprintf("Report Calculation Currency: %s", calculationCurrency),
		"Gains-And-Losses Summary",
	}

	var summaryLines, err = buildSummaryLines(report, calculationCurrency)
	if err != nil {
		return nil, err
	}
	lines = append(lines, summaryLines...)

	var rateLines = buildRateSourceLines(report)
	lines = append(lines, rateLines...)
	lines = append(lines, buildReferenceLines(report)...)

	var detailLines []string
	detailLines, err = buildDetailLines(report, calculationCurrency)
	if err != nil {
		return nil, err
	}
	lines = append(lines, detailLines...)

	return lines, nil
}

// buildSummaryLines formats the non-zero summary rows and yearly total for PDF.
// Authored by: OpenCode
func buildSummaryLines(report reportmodel.CapitalGainsReport, calculationCurrency string) ([]string, error) {
	var lines []string
	var renderedEntries []reportmodel.AssetSummaryEntry
	for _, entry := range report.SummaryEntries {
		if entry.NetGainOrLoss.Sign() == 0 {
			continue
		}
		renderedEntries = append(renderedEntries, entry)
	}
	if len(renderedEntries) == 0 {
		lines = append(lines, "No assets had a non-zero net gain or loss in the selected year.")
	}
	lines = append(lines, "Summary columns: Asset, Net Gain Or Loss, Report Calculation Currency")
	for _, entry := range renderedEntries {
		var netGainOrLoss, err = decimalsupport.CanonicalString(entry.NetGainOrLoss)
		if err != nil {
			return nil, fmt.Errorf("render summary entry %q net gain or loss: %w", entry.AssetIdentityKey, err)
		}
		lines = append(lines, fmt.Sprintf(
			"Summary row: %s, %s, %s",
			renderDisplayLabel(entry.DisplayLabel, entry.AssetIdentityKey),
			netGainOrLoss,
			calculationCurrencyLabelWithFallback(entry.ReportCalculationCurrency, calculationCurrency),
		))
	}

	var yearlyNetTotal, err = decimalsupport.CanonicalString(report.YearlyNetTotal)
	if err != nil {
		return nil, fmt.Errorf("render yearly net total: %w", err)
	}
	lines = append(lines, fmt.Sprintf("Overall Yearly Net Total: %s %s", yearlyNetTotal, calculationCurrency))
	return lines, nil
}

// buildRateSourceLines formats provider-level rate source evidence for PDF.
// Authored by: OpenCode
func buildRateSourceLines(report reportmodel.CapitalGainsReport) []string {
	var lines = []string{"Rate Source Summary", fmt.Sprintf("Report Base Currency: %s", calculationCurrencyLabel(report.ReportCalculationCurrency))}
	if len(report.RateSources) == 0 {
		return append(lines, "Exchange Rate Use: No activity required exchange-rate conversion.")
	}

	var rendered = make(map[string]bool)
	for _, source := range report.RateSources {
		var key = strings.Join([]string{string(source.Authority), string(source.ProviderID), source.RateKind}, "|")
		if rendered[key] {
			continue
		}
		rendered[key] = true
		lines = append(lines,
			fmt.Sprintf("Authority: %s", sanitizeText(reportmodel.RateAuthorityDisplayLabel(source.Authority))),
			fmt.Sprintf("Provider: %s", rateProviderLabel(source.ProviderID)),
			fmt.Sprintf("Rate Kind: %s", sanitizeText(source.RateKind)),
			fmt.Sprintf("Unavailable-Date Rule: %s", sanitizeText(reportmodel.RateProviderUnavailableDateRule(source.ProviderID))),
		)
	}
	return lines
}

// buildReferenceLines formats the historical full-liquidation reference rows.
// Authored by: OpenCode
func buildReferenceLines(report reportmodel.CapitalGainsReport) []string {
	var lines = []string{"Reference Section"}
	if len(report.ReferenceEntries) == 0 {
		return append(lines, "No assets reached full liquidation by year end.")
	}
	lines = append(lines, "Reference columns: Asset, Historical Full Liquidation Count, Main Section Status")
	for _, entry := range report.ReferenceEntries {
		lines = append(lines, fmt.Sprintf(
			"Reference row: %s, %d, %s",
			renderDisplayLabel(entry.DisplayLabel, entry.AssetIdentityKey),
			entry.FullLiquidationCountThroughYearEnd,
			sanitizeText(string(entry.MainSectionStatus)),
		))
	}
	return lines
}

// buildDetailLines formats asset detail sections using PDF-specific text lines.
// Authored by: OpenCode
func buildDetailLines(report reportmodel.CapitalGainsReport, calculationCurrency string) ([]string, error) {
	var lines []string
	for _, section := range report.DetailSections {
		lines = append(lines, fmt.Sprintf("Asset Detail: %s", renderDisplayLabel(section.DisplayLabel, section.AssetIdentityKey)))
		if len(section.ActivityRows) == 0 {
			var positionLines, err = buildPositionLines("Historical Position", section.ClosingQuantity, section.ClosingCostBasis, section.CalculationCurrency, calculationCurrency)
			if err != nil {
				return nil, fmt.Errorf("render historical position for %q: %w", section.AssetIdentityKey, err)
			}
			lines = append(lines, positionLines...)
			continue
		}
		var positionLines, err = buildPositionLines("Opening Position", section.OpeningQuantity, section.OpeningCostBasis, section.CalculationCurrency, calculationCurrency)
		if err != nil {
			return nil, fmt.Errorf("render opening position for %q: %w", section.AssetIdentityKey, err)
		}
		lines = append(lines, positionLines...)
		var activityLines []string
		activityLines, err = buildActivityLines(section)
		if err != nil {
			return nil, fmt.Errorf("render in-year activity for %q: %w", section.AssetIdentityKey, err)
		}
		lines = append(lines, activityLines...)
		var liquidationLines []string
		liquidationLines, err = buildLiquidationLines(section, calculationCurrency)
		if err != nil {
			return nil, fmt.Errorf("render liquidation calculations for %q: %w", section.AssetIdentityKey, err)
		}
		lines = append(lines, liquidationLines...)
		positionLines, err = buildPositionLines("Closing Position", section.ClosingQuantity, section.ClosingCostBasis, section.CalculationCurrency, calculationCurrency)
		if err != nil {
			return nil, fmt.Errorf("render closing position for %q: %w", section.AssetIdentityKey, err)
		}
		lines = append(lines, positionLines...)
	}
	return lines, nil
}

// buildPositionLines formats one asset position block.
// Authored by: OpenCode
func buildPositionLines(heading string, quantity apd.Decimal, basis apd.Decimal, sectionCurrency string, fallbackCurrency string) ([]string, error) {
	var quantityText, err = decimalsupport.CanonicalString(quantity)
	if err != nil {
		return nil, fmt.Errorf("render quantity: %w", err)
	}
	var basisText string
	basisText, err = decimalsupport.CanonicalString(basis)
	if err != nil {
		return nil, fmt.Errorf("render cost basis: %w", err)
	}
	return []string{
		heading,
		fmt.Sprintf("Quantity: %s", quantityText),
		fmt.Sprintf("Cost Basis: %s", basisText),
		fmt.Sprintf("Calculation Currency: %s", calculationCurrencyLabelWithFallback(sectionCurrency, fallbackCurrency)),
	}, nil
}

// buildActivityLines formats the in-year activity rows for one asset section.
// Authored by: OpenCode
func buildActivityLines(section reportmodel.AssetDetailSection) ([]string, error) {
	var lines = []string{"In-Year Activity", "Activity columns: Date, Source ID, Type, Quantity, Unit Price, Gross Value, Fee, Quantity After Row, Basis After Row, Original Activity Currency, Calculation Currency, Conversion Status, Note"}
	for _, row := range section.ActivityRows {
		var rowText, err = buildActivityLine(row)
		if err != nil {
			return nil, err
		}
		lines = append(lines, rowText)
	}
	return lines, nil
}

// buildActivityLine formats one in-year activity row.
// Authored by: OpenCode
func buildActivityLine(row reportmodel.AssetActivityRow) (string, error) {
	var quantityText, err = decimalsupport.CanonicalString(row.Quantity)
	if err != nil {
		return "", fmt.Errorf("render activity row %q quantity: %w", row.SourceID, err)
	}
	var unitPriceText string
	unitPriceText, err = decimalsupport.CanonicalStringPointer(row.UnitPrice)
	if err != nil {
		return "", fmt.Errorf("render activity row %q unit price: %w", row.SourceID, err)
	}
	var grossValueText string
	grossValueText, err = decimalsupport.CanonicalStringPointer(row.GrossValue)
	if err != nil {
		return "", fmt.Errorf("render activity row %q gross value: %w", row.SourceID, err)
	}
	var feeText string
	feeText, err = decimalsupport.CanonicalStringPointer(row.FeeAmount)
	if err != nil {
		return "", fmt.Errorf("render activity row %q fee: %w", row.SourceID, err)
	}
	var basisAfterRowText string
	basisAfterRowText, err = decimalsupport.CanonicalString(row.BasisAfterRow)
	if err != nil {
		return "", fmt.Errorf("render activity row %q basis after row: %w", row.SourceID, err)
	}
	var quantityAfterRowText string
	quantityAfterRowText, err = decimalsupport.CanonicalString(row.QuantityAfterRow)
	if err != nil {
		return "", fmt.Errorf("render activity row %q quantity after row: %w", row.SourceID, err)
	}
	var activityTypeLabel, labelErr = reportmodel.RenderActivityTypeLabel(row)
	if labelErr != nil {
		return "", fmt.Errorf("render activity row %q type label: %w", row.SourceID, labelErr)
	}
	var conversionStatusText string
	conversionStatusText, labelErr = conversionStatusColumn(row)
	if labelErr != nil {
		return "", fmt.Errorf("render activity row %q conversion status label: %w", row.SourceID, labelErr)
	}
	return fmt.Sprintf(
		"Activity row: %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s",
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
	), nil
}

// buildLiquidationLines formats priced liquidation rows when present.
// Authored by: OpenCode
func buildLiquidationLines(section reportmodel.AssetDetailSection, fallbackCurrency string) ([]string, error) {
	if len(section.LiquidationSummaries) == 0 {
		return nil, nil
	}
	var lines = []string{"Liquidation Calculations", "Liquidation columns: Date, Source ID, Disposed Quantity, Allocated Basis, Net Liquidation Proceeds, Gain Or Loss, Calculation Currency"}
	for _, liquidation := range section.LiquidationSummaries {
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
		lines = append(lines, fmt.Sprintf(
			"Liquidation row: %s, %s, %s, %s, %s, %s, %s",
			liquidation.OccurredAt.UTC().Format("2006-01-02 15:04:05"),
			sanitizeText(liquidation.SourceID),
			disposedQuantityText,
			allocatedBasisText,
			proceedsText,
			gainOrLossText,
			calculationCurrencyLabelWithFallback(liquidation.CalculationCurrency, fallbackCurrency),
		))
	}
	return lines, nil
}

// buildAnnexLines formats Annex 1 directly from the calculated audit annex.
// Authored by: OpenCode
func buildAnnexLines(annex reportmodel.AuditAnnex) ([]string, error) {
	if annex.Title == "" && len(annex.SectionOrder) == 0 {
		annex = reportmodel.DefaultAuditAnnex()
	}
	var lines = []string{annex.Title, "Detailed Per-Asset Audit Report"}
	if len(annex.PerAssetAuditSections) == 0 {
		lines = append(lines, "No per-asset audit activity is available for this report.")
	} else {
		for _, section := range annex.PerAssetAuditSections {
			lines = append(lines, fmt.Sprintf("Asset: %s", renderDisplayLabel(section.DisplayLabel, section.AssetIdentityKey)))
			lines = append(lines, "Audit columns: Date/Time, Source ID, Activity Type, Quantity, Unit Price, Gross Value, Fee, Original Activity Currency, Calculation Currency, Quantity After Activity, Basis After Activity, Full Liquidation Event, Allocated Basis, Net Liquidation Proceeds, Gain/Loss, Conversion Status, Sanitized Note")
			for _, entry := range section.Entries {
				var row, err = buildAnnexActivityLine(entry)
				if err != nil {
					return nil, fmt.Errorf("render annex audit entry %q: %w", entry.SourceID, err)
				}
				lines = append(lines, row)
			}
		}
	}
	var conversionLines, err = buildAnnexConversionLines(annex)
	if err != nil {
		return nil, err
	}
	lines = append(lines, conversionLines...)
	return lines, nil
}

// buildAnnexActivityLine formats one detailed audit activity row for PDF.
// Authored by: OpenCode
func buildAnnexActivityLine(entry reportmodel.AuditActivityEntry) (string, error) {
	var quantity, err = decimalsupport.CanonicalString(entry.Quantity)
	if err != nil {
		return "", fmt.Errorf("quantity: %w", err)
	}
	var unitPrice string
	unitPrice, err = decimalsupport.CanonicalStringPointer(entry.UnitPrice)
	if err != nil {
		return "", fmt.Errorf("unit price: %w", err)
	}
	var grossValue string
	grossValue, err = decimalsupport.CanonicalStringPointer(entry.GrossValue)
	if err != nil {
		return "", fmt.Errorf("gross value: %w", err)
	}
	var fee string
	fee, err = decimalsupport.CanonicalStringPointer(entry.FeeAmount)
	if err != nil {
		return "", fmt.Errorf("fee: %w", err)
	}
	var quantityAfter string
	quantityAfter, err = decimalsupport.CanonicalString(entry.QuantityAfterActivity)
	if err != nil {
		return "", fmt.Errorf("quantity after activity: %w", err)
	}
	var basisAfter string
	basisAfter, err = decimalsupport.CanonicalString(entry.BasisAfterActivity)
	if err != nil {
		return "", fmt.Errorf("basis after activity: %w", err)
	}
	var allocatedBasis string
	allocatedBasis, err = decimalsupport.CanonicalStringPointer(entry.AllocatedBasis)
	if err != nil {
		return "", fmt.Errorf("allocated basis: %w", err)
	}
	var proceeds string
	proceeds, err = decimalsupport.CanonicalStringPointer(entry.NetLiquidationProceeds)
	if err != nil {
		return "", fmt.Errorf("net liquidation proceeds: %w", err)
	}
	var gainOrLoss string
	gainOrLoss, err = decimalsupport.CanonicalStringPointer(entry.GainOrLoss)
	if err != nil {
		return "", fmt.Errorf("gain or loss: %w", err)
	}
	var activityTypeLabel string
	activityTypeLabel, err = reportmodel.RenderAuditActivityTypeLabel(entry)
	if err != nil {
		return "", fmt.Errorf("activity type label: %w", err)
	}
	var conversionStatus string
	if strings.TrimSpace(string(entry.ConversionStatus)) != "" {
		conversionStatus, err = reportmodel.RenderConversionStatusLabel(entry.ConversionStatus)
		if err != nil {
			return "", fmt.Errorf("conversion status label: %w", err)
		}
	}
	return fmt.Sprintf(
		"Audit row: %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %t, %s, %s, %s, %s, %s",
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
		entry.FullLiquidationEvent,
		allocatedBasis,
		proceeds,
		gainOrLoss,
		sanitizeText(conversionStatus),
		sanitizeText(entry.Note),
	), nil
}

// buildAnnexConversionLines formats the Annex 1 currency conversion audit.
// Authored by: OpenCode
func buildAnnexConversionLines(annex reportmodel.AuditAnnex) ([]string, error) {
	var lines = []string{"Currency Conversion Audit"}
	if len(annex.ConversionAuditEntries) == 0 {
		return append(lines, "No converted activity was present for this report."), nil
	}
	lines = append(lines, "Conversion columns: Date, Source ID, Asset, Rate Date, Source Currency, Report Base Currency, Converted Amounts, Quote Direction, Rate Value")
	for index, entry := range annex.ConversionAuditEntries {
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
		lines = append(lines, fmt.Sprintf(
			"Conversion row: %s, %s, %s, %s, %s, %s, %s, %s, %s",
			datesupport.FormatCalendarDate(entry.ActivityDate),
			sanitizeText(entry.SourceID),
			sanitizeText(entry.AssetLabel),
			datesupport.FormatCalendarDate(entry.RateDate),
			sanitizeText(entry.SourceCurrency),
			sanitizeText(entry.ReportBaseCurrency.Label()),
			convertedAmounts,
			sanitizeText(quoteDirection),
			rateValue,
		))
	}
	return lines, nil
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
// report lines in comments for deterministic automated assertions.
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
	return &gopdfDocument{y: 36}
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

// AddText emits one selectable report text line.
// Authored by: OpenCode
func (document *gopdfDocument) AddText(text string) error {
	if err := document.ensureWritableLine(); err != nil {
		return err
	}
	if err := document.pdf.SetFont("regular", "", 9); err != nil {
		return err
	}
	document.pdf.SetXY(36, document.y)
	if err := writeTextForGopdfDocument(document, text); err != nil {
		return err
	}
	document.texts = append(document.texts, text)
	document.y += 12
	return nil
}

// AddAnnexPageBreak starts Annex 1 on a new page.
// Authored by: OpenCode
func (document *gopdfDocument) AddAnnexPageBreak() error {
	document.pdf.AddPage()
	document.y = 36
	document.texts = append(document.texts, "--- page break ---")
	return nil
}

// ensureWritableLine adds continuation pages before text would leave the A4
// printable area.
// Authored by: OpenCode
func (document *gopdfDocument) ensureWritableLine() error {
	if !document.started {
		return fmt.Errorf("PDF document must be started before adding text")
	}
	if document.y <= 800 {
		return nil
	}
	document.pdf.AddPage()
	document.y = 36
	return nil
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

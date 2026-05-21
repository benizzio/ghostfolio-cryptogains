// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"
	"time"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

// ReportDocumentType identifies one supported rendered report document format.
// Authored by: OpenCode
type ReportDocumentType string

const (
	// ReportDocumentTypeMarkdown identifies the Markdown report document format.
	ReportDocumentTypeMarkdown ReportDocumentType = "markdown"
)

// ReferenceSectionStatus identifies whether one reference entry also appears in
// the main report sections.
// Authored by: OpenCode
type ReferenceSectionStatus string

const (
	// ReferenceSectionStatusIncludedInMainSections indicates that the asset also
	// appears in the main report sections.
	ReferenceSectionStatusIncludedInMainSections ReferenceSectionStatus = "included in main sections"

	// ReferenceSectionStatusReferenceOnly indicates that the asset appears only in
	// the reference section.
	ReferenceSectionStatusReferenceOnly ReferenceSectionStatus = "reference only"
)

// ReportRequest stores the user-selected inputs for one report-generation run.
// Authored by: OpenCode
type ReportRequest struct {
	Year            int
	CostBasisMethod CostBasisMethod
	RequestedAt     time.Time
}

// CapitalGainsReport stores the fully calculated yearly report before Markdown
// rendering.
// Authored by: OpenCode
type CapitalGainsReport struct {
	Year                      int
	CostBasisMethod           CostBasisMethod
	GeneratedAt               time.Time
	ReportCalculationCurrency string
	SummaryEntries            []AssetSummaryEntry
	YearlyNetTotal            apd.Decimal
	ReferenceEntries          []ReferenceLiquidationEntry
	DetailSections            []AssetDetailSection
}

// ReportDocument stores the rendered report content before the final save.
// Authored by: OpenCode
type ReportDocument struct {
	DocumentType    ReportDocumentType
	Content         string
	Year            int
	CostBasisMethod CostBasisMethod
	GeneratedAt     time.Time
}

// ReportOutputFile stores the final cleartext report file details returned to
// the runtime workflow.
// Authored by: OpenCode
type ReportOutputFile struct {
	DocumentsDirectory string
	Filename           string
	Path               string
	SavedAt            time.Time
	OpenRequested      bool
	OpenError          string
}

// AssetSummaryEntry stores one row in the summary section of the report.
// Authored by: OpenCode
type AssetSummaryEntry struct {
	AssetIdentityKey          string
	DisplayLabel              string
	NetGainOrLoss             apd.Decimal
	ReportCalculationCurrency string
}

// ReferenceLiquidationEntry stores one row in the report reference section.
// Authored by: OpenCode
type ReferenceLiquidationEntry struct {
	AssetIdentityKey                   string
	DisplayLabel                       string
	FullLiquidationCountThroughYearEnd int
	MainSectionStatus                  ReferenceSectionStatus
}

// AssetDetailSection stores one per-asset detail section in the calculated
// report.
// Authored by: OpenCode
type AssetDetailSection struct {
	AssetIdentityKey     string
	DisplayLabel         string
	OpeningQuantity      apd.Decimal
	OpeningCostBasis     apd.Decimal
	ClosingQuantity      apd.Decimal
	ClosingCostBasis     apd.Decimal
	CalculationCurrency  string
	ActivityRows         []AssetActivityRow
	LiquidationSummaries []LiquidationCalculation
}

// AssetActivityRow stores one in-year activity row for an included asset.
// Authored by: OpenCode
type AssetActivityRow struct {
	SourceID                    string
	OccurredAt                  time.Time
	ActivityType                syncmodel.ActivityType
	Quantity                    apd.Decimal
	GrossValue                  *apd.Decimal
	FeeAmount                   *apd.Decimal
	ActivityCurrency            string
	BasisAfterRow               apd.Decimal
	CalculationCurrency         string
	QuantityAfterRow            apd.Decimal
	HoldingReductionExplanation string
	LiquidationCalculation      *LiquidationCalculation
}

// LiquidationCalculation stores one priced liquidation calculation rendered in
// an asset detail section.
// Authored by: OpenCode
type LiquidationCalculation struct {
	SourceID               string
	OccurredAt             time.Time
	DisposedQuantity       apd.Decimal
	AllocatedBasis         apd.Decimal
	NetLiquidationProceeds apd.Decimal
	GainOrLoss             apd.Decimal
	ActivityCurrency       string
	CalculationCurrency    string
}

// NewReportRequest creates one validated report-generation request.
//
// Example:
//
//	request, err := model.NewReportRequest(2024, model.CostBasisMethodFIFO, time.Now())
//	if err != nil {
//		panic(err)
//	}
//	_ = request.Year
//
// Authored by: OpenCode
func NewReportRequest(year int, method CostBasisMethod, requestedAt time.Time) (ReportRequest, error) {
	var request = ReportRequest{
		Year:            year,
		CostBasisMethod: method,
		RequestedAt:     requestedAt,
	}

	if err := request.Validate(); err != nil {
		return ReportRequest{}, err
	}

	return request, nil
}

// Validate verifies that the report request is complete and reusable by later
// report-generation stages.
// Authored by: OpenCode
func (request ReportRequest) Validate() error {
	if request.Year <= 0 {
		return fmt.Errorf("report request year must be greater than zero")
	}
	if err := validateCostBasisMethod(request.CostBasisMethod); err != nil {
		return fmt.Errorf("report request cost basis method: %w", err)
	}
	if request.RequestedAt.IsZero() {
		return fmt.Errorf("report request requested-at timestamp is required")
	}

	return nil
}

// NewAssetSummaryEntry creates one validated summary-section row.
//
// Example:
//
//	entry, err := model.NewAssetSummaryEntry("asset-btc", "BTC", net, "USD")
//	if err != nil {
//		panic(err)
//	}
//	_ = entry.DisplayLabel
//
// Authored by: OpenCode
func NewAssetSummaryEntry(assetIdentityKey string, displayLabel string, netGainOrLoss apd.Decimal, reportCalculationCurrency string) (AssetSummaryEntry, error) {
	var entry = AssetSummaryEntry{
		AssetIdentityKey:          strings.TrimSpace(assetIdentityKey),
		DisplayLabel:              strings.TrimSpace(displayLabel),
		NetGainOrLoss:             netGainOrLoss,
		ReportCalculationCurrency: strings.TrimSpace(reportCalculationCurrency),
	}

	if err := entry.Validate(); err != nil {
		return AssetSummaryEntry{}, err
	}

	return entry, nil
}

// Validate verifies one summary-section row.
// Authored by: OpenCode
func (entry AssetSummaryEntry) Validate() error {
	if strings.TrimSpace(entry.AssetIdentityKey) == "" {
		return fmt.Errorf("asset summary entry asset identity key is required")
	}
	if err := validateFiniteDecimal(entry.NetGainOrLoss, "asset summary entry net gain or loss"); err != nil {
		return err
	}

	return nil
}

// NewReferenceLiquidationEntry creates one validated reference-section row.
//
// Example:
//
//	entry, err := model.NewReferenceLiquidationEntry("asset-btc", "BTC", 1, model.ReferenceSectionStatusReferenceOnly)
//	if err != nil {
//		panic(err)
//	}
//	_ = entry.MainSectionStatus
//
// Authored by: OpenCode
func NewReferenceLiquidationEntry(assetIdentityKey string, displayLabel string, fullLiquidationCountThroughYearEnd int, mainSectionStatus ReferenceSectionStatus) (ReferenceLiquidationEntry, error) {
	var entry = ReferenceLiquidationEntry{
		AssetIdentityKey:                   strings.TrimSpace(assetIdentityKey),
		DisplayLabel:                       strings.TrimSpace(displayLabel),
		FullLiquidationCountThroughYearEnd: fullLiquidationCountThroughYearEnd,
		MainSectionStatus:                  mainSectionStatus,
	}

	if err := entry.Validate(); err != nil {
		return ReferenceLiquidationEntry{}, err
	}

	return entry, nil
}

// Validate verifies one reference-section row.
// Authored by: OpenCode
func (entry ReferenceLiquidationEntry) Validate() error {
	if strings.TrimSpace(entry.AssetIdentityKey) == "" {
		return fmt.Errorf("reference entry asset identity key is required")
	}
	if entry.FullLiquidationCountThroughYearEnd < 0 {
		return fmt.Errorf("reference entry full liquidation count must not be negative")
	}
	if err := validateReferenceSectionStatus(entry.MainSectionStatus); err != nil {
		return fmt.Errorf("reference entry main section status: %w", err)
	}

	return nil
}

// NewAssetDetailSection creates one validated per-asset detail section.
//
// Example:
//
//	section, err := model.NewAssetDetailSection("asset-btc", "BTC", openingQty, openingBasis, closingQty, closingBasis, "USD", nil, nil)
//	if err != nil {
//		panic(err)
//	}
//	_ = section.AssetIdentityKey
//
// Authored by: OpenCode
func NewAssetDetailSection(
	assetIdentityKey string,
	displayLabel string,
	openingQuantity apd.Decimal,
	openingCostBasis apd.Decimal,
	closingQuantity apd.Decimal,
	closingCostBasis apd.Decimal,
	calculationCurrency string,
	activityRows []AssetActivityRow,
	liquidationSummaries []LiquidationCalculation,
) (AssetDetailSection, error) {
	var section = AssetDetailSection{
		AssetIdentityKey:     strings.TrimSpace(assetIdentityKey),
		DisplayLabel:         strings.TrimSpace(displayLabel),
		OpeningQuantity:      openingQuantity,
		OpeningCostBasis:     openingCostBasis,
		ClosingQuantity:      closingQuantity,
		ClosingCostBasis:     closingCostBasis,
		CalculationCurrency:  strings.TrimSpace(calculationCurrency),
		ActivityRows:         append([]AssetActivityRow(nil), activityRows...),
		LiquidationSummaries: append([]LiquidationCalculation(nil), liquidationSummaries...),
	}

	if err := section.Validate(); err != nil {
		return AssetDetailSection{}, err
	}

	return section, nil
}

// Validate verifies one per-asset detail section and its nested rows.
// Authored by: OpenCode
func (section AssetDetailSection) Validate() error {
	if strings.TrimSpace(section.AssetIdentityKey) == "" {
		return fmt.Errorf("asset detail section asset identity key is required")
	}
	if err := validateNonNegativeDecimal(section.OpeningQuantity, "asset detail section opening quantity"); err != nil {
		return err
	}
	if err := validateNonNegativeDecimal(section.OpeningCostBasis, "asset detail section opening cost basis"); err != nil {
		return err
	}
	if err := validateNonNegativeDecimal(section.ClosingQuantity, "asset detail section closing quantity"); err != nil {
		return err
	}
	if err := validateNonNegativeDecimal(section.ClosingCostBasis, "asset detail section closing cost basis"); err != nil {
		return err
	}

	for index, row := range section.ActivityRows {
		if err := row.Validate(); err != nil {
			return fmt.Errorf("asset detail section activity row %d: %w", index, err)
		}
	}
	for index, liquidation := range section.LiquidationSummaries {
		if err := liquidation.Validate(); err != nil {
			return fmt.Errorf("asset detail section liquidation summary %d: %w", index, err)
		}
	}

	return nil
}

// Validate verifies one in-year asset activity row.
// Authored by: OpenCode
func (row AssetActivityRow) Validate() error {
	if strings.TrimSpace(row.SourceID) == "" {
		return fmt.Errorf("asset activity row source ID is required")
	}
	if row.OccurredAt.IsZero() {
		return fmt.Errorf("asset activity row occurred-at timestamp is required")
	}
	if err := validateActivityType(row.ActivityType); err != nil {
		return fmt.Errorf("asset activity row activity type: %w", err)
	}
	if err := validatePositiveDecimal(row.Quantity, "asset activity row quantity"); err != nil {
		return err
	}
	if err := validateOptionalDecimal(row.GrossValue, "asset activity row gross value"); err != nil {
		return err
	}
	if err := validateOptionalDecimal(row.FeeAmount, "asset activity row fee amount"); err != nil {
		return err
	}
	if err := validateNonNegativeDecimal(row.BasisAfterRow, "asset activity row basis after row"); err != nil {
		return err
	}
	if err := validateNonNegativeDecimal(row.QuantityAfterRow, "asset activity row quantity after row"); err != nil {
		return err
	}
	if row.LiquidationCalculation != nil {
		if err := row.LiquidationCalculation.Validate(); err != nil {
			return fmt.Errorf("asset activity row liquidation calculation: %w", err)
		}
	}

	return nil
}

// Validate verifies one priced liquidation calculation row.
// Authored by: OpenCode
func (calculation LiquidationCalculation) Validate() error {
	if strings.TrimSpace(calculation.SourceID) == "" {
		return fmt.Errorf("liquidation calculation source ID is required")
	}
	if calculation.OccurredAt.IsZero() {
		return fmt.Errorf("liquidation calculation occurred-at timestamp is required")
	}
	if err := validatePositiveDecimal(calculation.DisposedQuantity, "liquidation calculation disposed quantity"); err != nil {
		return err
	}
	if err := validateNonNegativeDecimal(calculation.AllocatedBasis, "liquidation calculation allocated basis"); err != nil {
		return err
	}
	if err := validateFiniteDecimal(calculation.NetLiquidationProceeds, "liquidation calculation net liquidation proceeds"); err != nil {
		return err
	}
	if err := validateFiniteDecimal(calculation.GainOrLoss, "liquidation calculation gain or loss"); err != nil {
		return err
	}
	if strings.TrimSpace(calculation.ActivityCurrency) == "" {
		return fmt.Errorf("liquidation calculation activity currency is required")
	}

	return nil
}

// NewCapitalGainsReport creates one validated calculated report model.
//
// Example:
//
//	report, err := model.NewCapitalGainsReport(request, time.Now(), "USD", nil, total, nil, nil)
//	if err != nil {
//		panic(err)
//	}
//	_ = report.GeneratedAt
//
// Authored by: OpenCode
func NewCapitalGainsReport(
	request ReportRequest,
	generatedAt time.Time,
	reportCalculationCurrency string,
	summaryEntries []AssetSummaryEntry,
	yearlyNetTotal apd.Decimal,
	referenceEntries []ReferenceLiquidationEntry,
	detailSections []AssetDetailSection,
) (CapitalGainsReport, error) {
	if err := request.Validate(); err != nil {
		return CapitalGainsReport{}, fmt.Errorf("capital gains report request: %w", err)
	}

	var report = CapitalGainsReport{
		Year:                      request.Year,
		CostBasisMethod:           request.CostBasisMethod,
		GeneratedAt:               generatedAt,
		ReportCalculationCurrency: strings.TrimSpace(reportCalculationCurrency),
		SummaryEntries:            append([]AssetSummaryEntry(nil), summaryEntries...),
		YearlyNetTotal:            yearlyNetTotal,
		ReferenceEntries:          append([]ReferenceLiquidationEntry(nil), referenceEntries...),
		DetailSections:            append([]AssetDetailSection(nil), detailSections...),
	}

	if err := report.Validate(); err != nil {
		return CapitalGainsReport{}, err
	}

	return report, nil
}

// Validate verifies one fully calculated report and its nested sections.
// Authored by: OpenCode
func (report CapitalGainsReport) Validate() error {
	if report.Year <= 0 {
		return fmt.Errorf("capital gains report year must be greater than zero")
	}
	if err := validateCostBasisMethod(report.CostBasisMethod); err != nil {
		return fmt.Errorf("capital gains report cost basis method: %w", err)
	}
	if report.GeneratedAt.IsZero() {
		return fmt.Errorf("capital gains report generated-at timestamp is required")
	}
	if err := validateFiniteDecimal(report.YearlyNetTotal, "capital gains report yearly net total"); err != nil {
		return err
	}

	for index, entry := range report.SummaryEntries {
		if err := entry.Validate(); err != nil {
			return fmt.Errorf("capital gains report summary entry %d: %w", index, err)
		}
	}
	for index, entry := range report.ReferenceEntries {
		if err := entry.Validate(); err != nil {
			return fmt.Errorf("capital gains report reference entry %d: %w", index, err)
		}
	}
	for index, section := range report.DetailSections {
		if err := section.Validate(); err != nil {
			return fmt.Errorf("capital gains report detail section %d: %w", index, err)
		}
	}

	return nil
}

// NewReportDocument creates one validated rendered report document.
//
// Example:
//
//	document, err := model.NewReportDocument(model.ReportDocumentTypeMarkdown, "# Report\n", 2024, model.CostBasisMethodFIFO, time.Now())
//	if err != nil {
//		panic(err)
//	}
//	_ = document.Content
//
// Authored by: OpenCode
func NewReportDocument(documentType ReportDocumentType, content string, year int, method CostBasisMethod, generatedAt time.Time) (ReportDocument, error) {
	var document = ReportDocument{
		DocumentType:    documentType,
		Content:         content,
		Year:            year,
		CostBasisMethod: method,
		GeneratedAt:     generatedAt,
	}

	if err := document.Validate(); err != nil {
		return ReportDocument{}, err
	}

	return document, nil
}

// Validate verifies one rendered report document before save.
// Authored by: OpenCode
func (document ReportDocument) Validate() error {
	if err := validateReportDocumentType(document.DocumentType); err != nil {
		return fmt.Errorf("report document type: %w", err)
	}
	if strings.TrimSpace(document.Content) == "" {
		return fmt.Errorf("report document content is required")
	}
	if document.Year <= 0 {
		return fmt.Errorf("report document year must be greater than zero")
	}
	if err := validateCostBasisMethod(document.CostBasisMethod); err != nil {
		return fmt.Errorf("report document cost basis method: %w", err)
	}
	if document.GeneratedAt.IsZero() {
		return fmt.Errorf("report document generated-at timestamp is required")
	}

	return nil
}

// NewReportOutputFile creates one validated report save outcome.
//
// Example:
//
//	outputFile, err := model.NewReportOutputFile("/tmp/Documents", "report.md", "/tmp/Documents/report.md", time.Now(), true, "")
//	if err != nil {
//		panic(err)
//	}
//	_ = outputFile.Path
//
// Authored by: OpenCode
func NewReportOutputFile(documentsDirectory string, filename string, path string, savedAt time.Time, openRequested bool, openError string) (ReportOutputFile, error) {
	var outputFile = ReportOutputFile{
		DocumentsDirectory: strings.TrimSpace(documentsDirectory),
		Filename:           strings.TrimSpace(filename),
		Path:               strings.TrimSpace(path),
		SavedAt:            savedAt,
		OpenRequested:      openRequested,
		OpenError:          strings.TrimSpace(openError),
	}

	if err := outputFile.Validate(); err != nil {
		return ReportOutputFile{}, err
	}

	return outputFile, nil
}

// Validate verifies one persisted-output outcome returned to runtime code.
// Authored by: OpenCode
func (outputFile ReportOutputFile) Validate() error {
	if outputFile.DocumentsDirectory == "" {
		return fmt.Errorf("report output documents directory is required")
	}
	if outputFile.Filename == "" {
		return fmt.Errorf("report output filename is required")
	}
	if outputFile.Path == "" {
		return fmt.Errorf("report output path is required")
	}
	if outputFile.SavedAt.IsZero() {
		return fmt.Errorf("report output saved-at timestamp is required")
	}
	if !outputFile.OpenRequested && outputFile.OpenError != "" {
		return fmt.Errorf("report output open error requires an open request")
	}

	return nil
}

// validateCostBasisMethod rejects unsupported report cost-basis method values.
// Authored by: OpenCode
func validateCostBasisMethod(method CostBasisMethod) error {
	for _, supportedMethod := range SupportedCostBasisMethods() {
		if method == supportedMethod {
			return nil
		}
	}

	return fmt.Errorf("unsupported cost basis method %q", method)
}

// validateReferenceSectionStatus rejects unsupported reference-entry statuses.
// Authored by: OpenCode
func validateReferenceSectionStatus(status ReferenceSectionStatus) error {
	switch status {
	case ReferenceSectionStatusIncludedInMainSections, ReferenceSectionStatusReferenceOnly:
		return nil
	default:
		return fmt.Errorf("unsupported reference section status %q", status)
	}
}

// validateReportDocumentType rejects unsupported rendered-document types.
// Authored by: OpenCode
func validateReportDocumentType(documentType ReportDocumentType) error {
	switch documentType {
	case ReportDocumentTypeMarkdown:
		return nil
	default:
		return fmt.Errorf("unsupported report document type %q", documentType)
	}
}

// validateActivityType rejects unsupported activity-row activity types.
// Authored by: OpenCode
func validateActivityType(activityType syncmodel.ActivityType) error {
	switch activityType {
	case syncmodel.ActivityTypeBuy, syncmodel.ActivityTypeSell:
		return nil
	default:
		return fmt.Errorf("unsupported activity type %q", activityType)
	}
}

// validateOptionalDecimal verifies one optional exact decimal value.
// Authored by: OpenCode
func validateOptionalDecimal(value *apd.Decimal, label string) error {
	if value == nil {
		return nil
	}

	return validateFiniteDecimal(*value, label)
}

// validatePositiveDecimal verifies one positive exact decimal value.
// Authored by: OpenCode
func validatePositiveDecimal(value apd.Decimal, label string) error {
	if err := validateFiniteDecimal(value, label); err != nil {
		return err
	}
	if value.Sign() <= 0 {
		return fmt.Errorf("%s must be greater than zero", label)
	}

	return nil
}

// validateNonNegativeDecimal verifies one non-negative exact decimal value.
// Authored by: OpenCode
func validateNonNegativeDecimal(value apd.Decimal, label string) error {
	if err := validateFiniteDecimal(value, label); err != nil {
		return err
	}
	if value.Sign() < 0 {
		return fmt.Errorf("%s must not be negative", label)
	}

	return nil
}

// validateFiniteDecimal verifies one finite exact decimal value.
// Authored by: OpenCode
func validateFiniteDecimal(value apd.Decimal, label string) error {
	if _, err := decimalsupport.CanonicalString(value); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}

	return nil
}

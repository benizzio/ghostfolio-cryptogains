// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"time"

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
	ActivityType                ActivityType
	Quantity                    apd.Decimal
	UnitPrice                   *apd.Decimal
	GrossValue                  *apd.Decimal
	FeeAmount                   *apd.Decimal
	ActivityCurrency            string
	BasisAfterRow               apd.Decimal
	CalculationCurrency         string
	QuantityAfterRow            apd.Decimal
	HoldingReductionExplanation string
	LiquidationCalculation      *LiquidationCalculation
}

// BasisMatch stores one acquisition fragment consumed by one liquidation.
// Authored by: OpenCode
type BasisMatch struct {
	AcquisitionSourceID string
	MatchedQuantity     apd.Decimal
	MatchedBasis        apd.Decimal
	MatchedProceeds     *apd.Decimal
	MatchedGainOrLoss   *apd.Decimal
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
	Matches                []BasisMatch
}

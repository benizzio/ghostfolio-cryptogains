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

	// ReportDocumentTypePDF identifies the PDF report document format.
	ReportDocumentTypePDF ReportDocumentType = "pdf"
)

// ReportDocumentRole identifies one rendered or persisted document's role in a
// report output bundle.
// Authored by: OpenCode
type ReportDocumentRole string

const (
	// ReportDocumentRoleMain identifies the main capital gains report document.
	ReportDocumentRoleMain ReportDocumentRole = "main"

	// ReportDocumentRoleAnnex identifies the separate Annex 1 document.
	ReportDocumentRoleAnnex ReportDocumentRole = "annex"

	// ReportDocumentRoleCombined identifies a combined main-plus-annex document.
	ReportDocumentRoleCombined ReportDocumentRole = "combined"
)

const (
	// ReportMediaTypeMarkdown identifies Markdown report output bytes.
	ReportMediaTypeMarkdown = "text/markdown"

	// ReportMediaTypePDF identifies PDF report output bytes.
	ReportMediaTypePDF = "application/pdf"
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
	ConversionAuditEntries    []ConversionAuditEntry
	RateSources               []ExchangeRateEvidence
	AuditAnnex                AuditAnnex
}

// AssetSummaryEntry stores one row in the summary section of the report.
// Authored by: OpenCode
type AssetSummaryEntry struct {
	AssetIdentityKey          string
	DisplayLabel              string
	NetGainOrLoss             apd.Decimal
	ReportCalculationCurrency string
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

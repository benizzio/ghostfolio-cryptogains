// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/cockroachdb/apd/v3"
)

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
		DetailSections:            cloneAssetDetailSections(detailSections),
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

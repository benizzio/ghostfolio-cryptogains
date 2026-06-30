// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"
	"time"

	datesupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/date"
	"github.com/cockroachdb/apd/v3"
)

// NewCapitalGainsReport creates one validated calculated report model without
// conversion artifacts.
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
	return NewCapitalGainsReportWithConversionArtifacts(
		request,
		generatedAt,
		reportCalculationCurrency,
		summaryEntries,
		yearlyNetTotal,
		referenceEntries,
		detailSections,
		nil,
		nil,
	)
}

// NewCapitalGainsReportWithConversionArtifacts creates one validated calculated
// report model including conversion audit entries and retained rate evidence.
//
// Example:
//
//	report, err := model.NewCapitalGainsReportWithConversionArtifacts(request, time.Now(), "USD", nil, total, nil, nil, nil, nil)
//	if err != nil {
//		panic(err)
//	}
//	_ = report.ReportCalculationCurrency
//
// Authored by: OpenCode
func NewCapitalGainsReportWithConversionArtifacts(
	request ReportRequest,
	generatedAt time.Time,
	reportCalculationCurrency string,
	summaryEntries []AssetSummaryEntry,
	yearlyNetTotal apd.Decimal,
	referenceEntries []ReferenceLiquidationEntry,
	detailSections []AssetDetailSection,
	conversionAuditEntries []ConversionAuditEntry,
	rateSources []ExchangeRateEvidence,
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
		ConversionAuditEntries:    cloneConversionAuditEntries(conversionAuditEntries),
		RateSources:               cloneExchangeRateEvidence(rateSources),
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
	if err := validateReportBaseCurrency(ReportBaseCurrency(strings.TrimSpace(report.ReportCalculationCurrency))); err != nil {
		return fmt.Errorf("capital gains report calculation currency: %w", err)
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
	if err := report.validateConversionArtifacts(); err != nil {
		return err
	}

	return nil
}

// validateConversionArtifacts verifies report-visible conversion audit entries
// and their canonical rate source evidence.
// Authored by: OpenCode
func (report CapitalGainsReport) validateConversionArtifacts() error {
	for index, source := range report.RateSources {
		if err := source.Validate(); err != nil {
			return fmt.Errorf("capital gains report rate source %d: %w", index, err)
		}
		if err := report.validateRateSourceCurrency(index, source); err != nil {
			return err
		}
	}

	for index, entry := range report.ConversionAuditEntries {
		if err := entry.Validate(); err != nil {
			return fmt.Errorf("capital gains report conversion audit entry %d: %w", index, err)
		}
		if !report.hasMatchingRateSource(entry) {
			return fmt.Errorf("capital gains report conversion audit entry %d: matching rate source is required", index)
		}
		if report.hasContradictingSameCurrencyDetailRow(entry.SourceID) {
			return fmt.Errorf("capital gains report conversion audit entry %d: matching detail row must not be same-currency", index)
		}
	}

	return nil
}

// hasContradictingSameCurrencyDetailRow reports whether an audited converted
// source activity is contradicted by a same-currency asset detail row.
// Authored by: OpenCode
func (report CapitalGainsReport) hasContradictingSameCurrencyDetailRow(sourceID string) bool {
	var wanted = strings.TrimSpace(sourceID)
	for _, section := range report.DetailSections {
		for _, row := range section.ActivityRows {
			if strings.TrimSpace(row.SourceID) == wanted && row.ConversionStatus == ConversionStatusSameCurrency {
				return true
			}
		}
	}

	return false
}

// validateRateSourceCurrency verifies that rate evidence belongs to the report
// calculation currency.
// Authored by: OpenCode
func (report CapitalGainsReport) validateRateSourceCurrency(index int, source ExchangeRateEvidence) error {
	var reportCurrency = strings.TrimSpace(report.ReportCalculationCurrency)
	if source.BaseCurrency.Label() != reportCurrency {
		return fmt.Errorf("capital gains report rate source %d: base currency must match report calculation currency", index)
	}

	return nil
}

// hasMatchingRateSource reports whether one audit entry is backed by canonical
// rate source evidence from the report model.
// Authored by: OpenCode
func (report CapitalGainsReport) hasMatchingRateSource(entry ConversionAuditEntry) bool {
	for _, source := range report.RateSources {
		if strings.TrimSpace(source.SourceCurrency) == strings.TrimSpace(entry.SourceCurrency) &&
			source.BaseCurrency == entry.ReportBaseCurrency &&
			datesupport.CalendarDate(source.ActivityDate).Equal(datesupport.CalendarDate(entry.ActivityDate)) &&
			datesupport.CalendarDate(source.RateDate).Equal(datesupport.CalendarDate(entry.RateDate)) &&
			source.Authority == entry.RateAuthority &&
			strings.TrimSpace(source.RateKind) == strings.TrimSpace(entry.RateKind) &&
			source.QuoteDirection == entry.QuoteDirection &&
			source.RateValue.Cmp(&entry.RateValue) == 0 {
			return true
		}
	}

	return false
}

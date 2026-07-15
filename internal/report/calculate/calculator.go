// Package calculate defines yearly gains-and-losses report calculation
// services built on normalized protected activity history.
// Authored by: OpenCode
package calculate

import (
	"context"
	"fmt"

	reportbasis "github.com/benizzio/ghostfolio-cryptogains/internal/report/basis"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

const reportCalculationCurrencyLabel = "NOT APPLICABLE"

// Test seams keep calculator wrapper branches directly coverable without
// weakening the production validation flow.
// Authored by: OpenCode
var (
	calculateAssetGroupFunc   = calculateAssetGroup
	newCapitalGainsReport     = reportmodel.NewCapitalGainsReportWithConversionArtifacts
	newLotMethodState         = reportbasis.NewLotMethodState
	newAssetBasisStateFunc    = newAssetBasisState
	resolveScopedInputsFunc   = resolveScopedAssetInputs
	replayAssetInputFunc      = replayAssetInput
	lotStateTotalOpenQuantity = func(state *reportbasis.LotMethodState) (apd.Decimal, error) { return state.TotalOpenQuantity() }
	reportDivideRoundHalfUp   = supportmath.DivideFiniteRoundHalfUp
)

// reportCalculationAggregation stores calculated report sections while asset
// groups are replayed.
// Authored by: OpenCode
type reportCalculationAggregation struct {
	SummaryEntries   []reportmodel.AssetSummaryEntry
	ReferenceEntries []reportmodel.ReferenceLiquidationEntry
	DetailSections   []reportmodel.AssetDetailSection
	AuditSections    []reportmodel.PerAssetAuditSection
	YearlyNetTotal   apd.Decimal
}

// Calculate replays the protected synced activity cache through one selected
// source-calendar year and cost-basis method to build the final calculated
// report model consumed by Markdown rendering.
//
// Example:
//
//	request, err := reportmodel.NewReportRequest(2024, reportmodel.CostBasisMethodFIFO, reportmodel.ReportBaseCurrencyUSD, time.Now())
//	if err != nil {
//		panic(err)
//	}
//	report, err := calculate.Calculate(request, cache)
//	if err != nil {
//		panic(err)
//	}
//	_ = report.YearlyNetTotal
//
// Authored by: OpenCode
func Calculate(request reportmodel.ReportRequest, cache syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
	return calculateReport(context.Background(), nil, request, cache)
}

// calculateReport replays synced activity history after applying the selected
// report base-currency boundary.
// Authored by: OpenCode
func calculateReport(
	ctx context.Context,
	currencyRates CurrencyRateService,
	request reportmodel.ReportRequest,
	cache syncmodel.ProtectedActivityCache,
) (reportmodel.CapitalGainsReport, error) {
	if err := validateReportCalculationRequest(request, cache.AvailableReportYears); err != nil {
		return reportmodel.CapitalGainsReport{}, err
	}

	var groups, err = selectAssetInputGroupsThroughYear(cache.Activities, request.Year)
	if err != nil {
		return reportmodel.CapitalGainsReport{}, err
	}
	var currencyBoundaryResult reportCurrencyBoundaryResult
	currencyBoundaryResult, err = applyReportCurrencyBoundaryWithRecords(ctx, currencyRates, request.ReportBaseCurrency, groups, cache.Activities)
	if err != nil {
		return reportmodel.CapitalGainsReport{}, err
	}
	groups = currencyBoundaryResult.Groups

	var aggregation reportCalculationAggregation
	aggregation, err = calculateReportAssetGroups(request, groups)
	if err != nil {
		return reportmodel.CapitalGainsReport{}, err
	}

	return buildCalculatedReport(request, currencyBoundaryResult, aggregation)
}

// validateReportCalculationRequest verifies report request validity and synced
// data year availability before calculation begins.
// Authored by: OpenCode
func validateReportCalculationRequest(request reportmodel.ReportRequest, availableYears []int) error {
	if err := request.Validate(); err != nil {
		return reportmodel.NewCalculationError(
			reportmodel.CalculationErrorKindInvalidRequest,
			err.Error(),
			"",
			"",
			err,
		)
	}
	if !reportYearAvailable(availableYears, request.Year) {
		return reportmodel.NewCalculationError(
			reportmodel.CalculationErrorKindUnavailableReportYear,
			fmt.Sprintf("report year %d is not available in synced data", request.Year),
			"",
			"",
			nil,
		)
	}

	return nil
}

// calculateReportAssetGroups replays every scoped asset group and accumulates
// report sections for the final model.
// Authored by: OpenCode
func calculateReportAssetGroups(
	request reportmodel.ReportRequest,
	groups []assetInputGroup,
) (reportCalculationAggregation, error) {
	var aggregation reportCalculationAggregation
	for _, group := range groups {
		var assetResult assetCalculationResult
		var err error
		assetResult, err = calculateAssetGroupFunc(request.CostBasisMethod, request.Year, group)
		if err != nil {
			return reportCalculationAggregation{}, err
		}
		if err = aggregation.addAssetResult(group, assetResult); err != nil {
			return reportCalculationAggregation{}, err
		}
	}

	return aggregation, nil
}

// addAssetResult appends one calculated asset contribution to the report
// aggregation.
// Authored by: OpenCode
func (aggregation *reportCalculationAggregation) addAssetResult(group assetInputGroup, assetResult assetCalculationResult) error {
	if assetResult.ReferenceEntry != nil {
		aggregation.ReferenceEntries = append(aggregation.ReferenceEntries, *assetResult.ReferenceEntry)
	}
	if assetResult.IncludeInAudit {
		aggregation.AuditSections = append(aggregation.AuditSections, assetResult.AuditSection)
	}
	if !assetResult.IncludeInMain {
		return nil
	}

	aggregation.SummaryEntries = append(aggregation.SummaryEntries, assetResult.SummaryEntry)
	aggregation.DetailSections = append(aggregation.DetailSections, assetResult.DetailSection)

	var err error
	aggregation.YearlyNetTotal, err = supportmath.Add(aggregation.YearlyNetTotal, assetResult.YearlyNet)
	if err != nil {
		return reportmodel.NewCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			"could not accumulate the yearly report total",
			"",
			group.DisplayLabel,
			err,
		)
	}

	return nil
}

// buildCalculatedReport validates the final report and annex models.
// Authored by: OpenCode
func buildCalculatedReport(
	request reportmodel.ReportRequest,
	currencyBoundaryResult reportCurrencyBoundaryResult,
	aggregation reportCalculationAggregation,
) (reportmodel.CapitalGainsReport, error) {
	var report, reportErr = newCapitalGainsReport(
		request,
		request.RequestedAt,
		request.ReportBaseCurrency.Label(),
		aggregation.SummaryEntries,
		aggregation.YearlyNetTotal,
		aggregation.ReferenceEntries,
		aggregation.DetailSections,
		nil,
		currencyBoundaryResult.RateSources,
	)
	if reportErr != nil {
		return reportmodel.CapitalGainsReport{}, reportmodel.NewCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			"calculated report validation failed",
			"",
			"",
			reportErr,
		)
	}
	var annex reportmodel.AuditAnnex
	annex, reportErr = reportmodel.NewDetailedAuditAnnex(aggregation.AuditSections, currencyBoundaryResult.ConversionAuditEntries)
	if reportErr != nil {
		return reportmodel.CapitalGainsReport{}, reportmodel.NewCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			"calculated report audit annex validation failed",
			"",
			"",
			reportErr,
		)
	}
	report.AuditAnnex = annex
	if reportErr = report.Validate(); reportErr != nil {
		return reportmodel.CapitalGainsReport{}, reportmodel.NewCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			"calculated report validation failed",
			"",
			"",
			reportErr,
		)
	}
	return report, nil
}

// reportYearAvailable verifies that the selected year is present in the synced
// protected-cache metadata.
// Authored by: OpenCode
func reportYearAvailable(years []int, year int) bool {
	for _, availableYear := range years {
		if availableYear == year {
			return true
		}
	}

	return false
}

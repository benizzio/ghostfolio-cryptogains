// Package calculate defines yearly gains-and-losses report calculation
// services built on normalized protected activity history.
// Authored by: OpenCode
package calculate

import (
	"fmt"

	reportbasis "github.com/benizzio/ghostfolio-cryptogains/internal/report/basis"
	reportdecimal "github.com/benizzio/ghostfolio-cryptogains/internal/report/decimal"
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
	newCapitalGainsReport     = reportmodel.NewCapitalGainsReport
	newLotMethodState         = reportbasis.NewLotMethodState
	newAssetBasisStateFunc    = newAssetBasisState
	resolveScopedInputsFunc   = resolveScopedAssetInputs
	replayAssetInputFunc      = replayAssetInput
	lotStateTotalOpenQuantity = func(state *reportbasis.LotMethodState) (apd.Decimal, error) { return state.TotalOpenQuantity() }
	reportDivideRoundHalfUp   = reportdecimal.DivideRoundHalfUp
)

// Calculate replays the protected synced activity cache through one selected
// source-calendar year and cost-basis method to build the final calculated
// report model consumed by Markdown rendering.
//
// Example:
//
//	request, err := reportmodel.NewReportRequest(2024, reportmodel.CostBasisMethodFIFO, time.Now())
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
	if err := request.Validate(); err != nil {
		return reportmodel.CapitalGainsReport{}, reportmodel.NewCalculationError(
			reportmodel.CalculationErrorKindInvalidRequest,
			err.Error(),
			"",
			"",
			err,
		)
	}
	if !reportYearAvailable(cache.AvailableReportYears, request.Year) {
		return reportmodel.CapitalGainsReport{}, reportmodel.NewCalculationError(
			reportmodel.CalculationErrorKindUnavailableReportYear,
			fmt.Sprintf("report year %d is not available in synced data", request.Year),
			"",
			"",
			nil,
		)
	}

	var groups, err = selectAssetInputGroupsThroughYear(cache.Activities, request.Year)
	if err != nil {
		return reportmodel.CapitalGainsReport{}, err
	}

	var summaryEntries []reportmodel.AssetSummaryEntry
	var referenceEntries []reportmodel.ReferenceLiquidationEntry
	var detailSections []reportmodel.AssetDetailSection
	var yearlyNetTotal apd.Decimal

	for _, group := range groups {
		var assetResult assetCalculationResult
		assetResult, err = calculateAssetGroupFunc(request.CostBasisMethod, request.Year, cache, group)
		if err != nil {
			return reportmodel.CapitalGainsReport{}, err
		}
		if assetResult.ReferenceEntry != nil {
			referenceEntries = append(referenceEntries, *assetResult.ReferenceEntry)
		}
		if !assetResult.IncludeInMain {
			continue
		}

		summaryEntries = append(summaryEntries, assetResult.SummaryEntry)
		detailSections = append(detailSections, assetResult.DetailSection)

		yearlyNetTotal, err = supportmath.Add(yearlyNetTotal, assetResult.YearlyNet, "left calculation decimal", "right calculation decimal", "add calculation decimals")
		if err != nil {
			return reportmodel.CapitalGainsReport{}, reportmodel.NewCalculationError(
				reportmodel.CalculationErrorKindBasisAllocation,
				"could not accumulate the yearly report total",
				"",
				group.DisplayLabel,
				err,
			)
		}
	}

	var report, reportErr = newCapitalGainsReport(
		request,
		request.RequestedAt,
		reportCalculationCurrencyLabel,
		summaryEntries,
		yearlyNetTotal,
		referenceEntries,
		detailSections,
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

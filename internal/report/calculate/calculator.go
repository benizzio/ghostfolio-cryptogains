// Package calculate defines yearly gains-and-losses report calculation
// services built on normalized protected activity history.
// Authored by: OpenCode
package calculate

import (
	"fmt"
	"strings"
	"time"

	reportbasis "github.com/benizzio/ghostfolio-cryptogains/internal/report/basis"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
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
	addCalculationOperation   = func(sum *apd.Decimal, left *apd.Decimal, right *apd.Decimal) (apd.Condition, error) {
		return apd.BaseContext.Add(sum, left, right)
	}
	subtractCalculationOperation = func(difference *apd.Decimal, left *apd.Decimal, right *apd.Decimal) (apd.Condition, error) {
		return apd.BaseContext.Sub(difference, left, right)
	}
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

		yearlyNetTotal, err = addCalculationDecimal(yearlyNetTotal, assetResult.YearlyNet)
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

// assetInputGroup stores one asset's selected calculation inputs in synced
// deterministic replay order.
// Authored by: OpenCode
type assetInputGroup struct {
	AssetIdentityKey string
	DisplayLabel     string
	Inputs           []reportmodel.ActivityCalculationInput
}

// assetCalculationResult stores one asset's calculated report contributions.
// Authored by: OpenCode
type assetCalculationResult struct {
	IncludeInMain  bool
	SummaryEntry   reportmodel.AssetSummaryEntry
	ReferenceEntry *reportmodel.ReferenceLiquidationEntry
	DetailSection  reportmodel.AssetDetailSection
	YearlyNet      apd.Decimal
}

// assetReplayState stores one asset's evolving holdings and report artifacts
// while the selected-year cutoff is replayed.
// Authored by: OpenCode
type assetReplayState struct {
	openingCaptured          bool
	openingQuantity          apd.Decimal
	openingBasis             apd.Decimal
	closingQuantity          apd.Decimal
	closingBasis             apd.Decimal
	yearlyNet                apd.Decimal
	fullLiquidationCount     int
	hadInYearFullLiquidation bool
	activityRows             []reportmodel.AssetActivityRow
	liquidationSummaries     []reportmodel.LiquidationCalculation
}

// assetInputReplayResult stores one replayed activity's contribution to the
// per-asset report state.
// Authored by: OpenCode
type assetInputReplayResult struct {
	reachedZero        bool
	liquidationSummary *reportmodel.LiquidationCalculation
	activityRow        *reportmodel.AssetActivityRow
	yearlyNetDelta     apd.Decimal
}

// basisApplicationResult stores one activity's basis and proceeds effects after
// the selected cost-basis method has been applied.
// Authored by: OpenCode
type basisApplicationResult struct {
	allocatedBasis *apd.Decimal
	netProceeds    *apd.Decimal
	gainOrLoss     *apd.Decimal
	reachedZero    bool
}

// basisAcquisitionInput stores the exact acquisition values forwarded into one
// method-specific basis state.
// Authored by: OpenCode
type basisAcquisitionInput struct {
	SourceID           string
	AcquiredAt         time.Time
	DeterministicOrder int
	Quantity           apd.Decimal
	Basis              apd.Decimal
	ApplicableScopeKey string
}

// assetBasisState adapts one method-specific open-position state behind a
// minimal calculator-local interface.
// Authored by: OpenCode
type assetBasisState interface {
	AddAcquisition(basisAcquisitionInput) error
	Dispose(basisDisposalInput) (basisDisposalResult, error)
	OpenQuantity() (apd.Decimal, error)
	OpenBasis() (apd.Decimal, error)
}

// lotBasisState adapts FIFO, LIFO, and HIFO lot tracking.
// Authored by: OpenCode
type lotBasisState struct {
	state *reportbasis.LotMethodState
}

// averageCostBasisState adapts the moving average-cost pool.
// Authored by: OpenCode
type averageCostBasisState struct {
	state *reportbasis.AverageCostState
}

// scopeLocalHybridBasisState adapts the scope-local hybrid method state.
// Authored by: OpenCode
type scopeLocalHybridBasisState struct {
	state *reportbasis.ScopeLocalHybridState
}

// basisDisposalInput stores one disposal routed through the active basis state.
// Authored by: OpenCode
type basisDisposalInput struct {
	Quantity           apd.Decimal
	ApplicableScopeKey string
}

// basisDisposalResult stores one basis allocation and whether the relevant
// asset or scope transitioned to zero.
// Authored by: OpenCode
type basisDisposalResult struct {
	AllocatedBasis apd.Decimal
	ReachedZero    bool
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

// selectAssetInputGroupsThroughYear converts selected-year-relevant protected
// activity rows into grouped calculation inputs while preserving synced replay
// order.
// Authored by: OpenCode
func selectAssetInputGroupsThroughYear(records []syncmodel.ActivityRecord, selectedYear int) ([]assetInputGroup, error) {
	var orderedKeys []string
	var groupsByKey = make(map[string]*assetInputGroup)

	for _, record := range records {
		var occurredAt, err = parseActivityOccurredAt(record)
		if err != nil {
			return nil, newRecordCalculationError(
				reportmodel.CalculationErrorKindActivityInput,
				record,
				"could not read the activity timestamp",
				err,
			)
		}
		if occurredAt.Year() > selectedYear {
			continue
		}
		if strings.TrimSpace(record.AssetIdentityKey) == "" {
			return nil, newRecordCalculationError(
				reportmodel.CalculationErrorKindActivityInput,
				record,
				"activity is missing the stored asset identity key required for reporting",
				nil,
			)
		}

		var input reportmodel.ActivityCalculationInput
		input, err = SelectActivityCalculationInput(record)
		if err != nil {
			return nil, newRecordCalculationError(
				reportmodel.CalculationErrorKindActivityInput,
				record,
				err.Error(),
				err,
			)
		}

		var group = groupsByKey[input.AssetIdentityKey]
		if group == nil {
			group = &assetInputGroup{
				AssetIdentityKey: input.AssetIdentityKey,
				DisplayLabel:     input.DisplayLabel,
			}
			groupsByKey[input.AssetIdentityKey] = group
			orderedKeys = append(orderedKeys, input.AssetIdentityKey)
		}
		if group.DisplayLabel == "" && strings.TrimSpace(input.DisplayLabel) != "" {
			group.DisplayLabel = input.DisplayLabel
		}
		group.Inputs = append(group.Inputs, input)
	}

	var groups = make([]assetInputGroup, 0, len(orderedKeys))
	for _, key := range orderedKeys {
		groups = append(groups, *groupsByKey[key])
	}

	return groups, nil
}

// calculateAssetGroup replays one grouped asset history through the selected
// year cutoff and derives its summary, reference, and detail contributions.
// Authored by: OpenCode
func calculateAssetGroup(method reportmodel.CostBasisMethod, selectedYear int, cache syncmodel.ProtectedActivityCache, group assetInputGroup) (assetCalculationResult, error) {
	var basisState, err = newAssetBasisStateFunc(method)
	if err != nil {
		return assetCalculationResult{}, reportmodel.NewCalculationError(
			reportmodel.CalculationErrorKindUnsupportedCostBasisMethod,
			err.Error(),
			"",
			group.DisplayLabel,
			err,
		)
	}

	var replayState assetReplayState
	var scopedInputs []scopedActivityInput
	scopedInputs, err = resolveScopedInputsFunc(method, group)
	if err != nil {
		return assetCalculationResult{}, err
	}

	for index, scopedInput := range scopedInputs {
		var input = scopedInput.Input
		if err = captureOpeningPositionIfNeeded(&replayState, basisState, input.SourceYear, selectedYear); err != nil {
			return assetCalculationResult{}, newInputCalculationError(
				reportmodel.CalculationErrorKindBasisAllocation,
				input,
				"could not determine the opening position carried into the selected year",
				err,
			)
		}

		var replayResult assetInputReplayResult
		replayResult, err = replayAssetInputFunc(basisState, scopedInput, index+1, selectedYear)
		if err != nil {
			return assetCalculationResult{}, err
		}

		if replayResult.reachedZero {
			replayState.fullLiquidationCount++
			if input.SourceYear == selectedYear {
				replayState.hadInYearFullLiquidation = true
			}
		}
		if replayResult.activityRow != nil {
			replayState.activityRows = append(replayState.activityRows, *replayResult.activityRow)
		}
		if replayResult.liquidationSummary != nil {
			replayState.liquidationSummaries = append(replayState.liquidationSummaries, *replayResult.liquidationSummary)
		}

		replayState.yearlyNet, err = addCalculationDecimal(replayState.yearlyNet, replayResult.yearlyNetDelta)
		if err != nil {
			return assetCalculationResult{}, newInputCalculationError(
				reportmodel.CalculationErrorKindBasisAllocation,
				input,
				"could not accumulate the asset yearly gain or loss",
				err,
			)
		}
	}

	replayState.closingQuantity, err = basisState.OpenQuantity()
	if err != nil {
		return assetCalculationResult{}, reportmodel.NewCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			"could not determine the asset closing quantity",
			"",
			group.DisplayLabel,
			err,
		)
	}
	replayState.closingBasis, err = basisState.OpenBasis()
	if err != nil {
		return assetCalculationResult{}, reportmodel.NewCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			"could not determine the asset closing basis",
			"",
			group.DisplayLabel,
			err,
		)
	}
	if !replayState.openingCaptured {
		replayState.openingQuantity = replayState.closingQuantity
		replayState.openingBasis = replayState.closingBasis
	}

	return buildAssetCalculationResult(group, replayState)
}

// captureOpeningPositionIfNeeded snapshots the carried holdings state at the
// first in-year activity boundary.
// Authored by: OpenCode
func captureOpeningPositionIfNeeded(state *assetReplayState, basisState assetBasisState, sourceYear int, selectedYear int) error {
	if state == nil || state.openingCaptured || sourceYear != selectedYear {
		return nil
	}

	var openingQuantity, err = basisState.OpenQuantity()
	if err != nil {
		return err
	}
	var openingBasis apd.Decimal
	openingBasis, err = basisState.OpenBasis()
	if err != nil {
		return err
	}

	state.openingCaptured = true
	state.openingQuantity = openingQuantity
	state.openingBasis = openingBasis
	return nil
}

// replayAssetInput applies one activity to the method-specific basis state and
// produces any selected-year report artifacts.
// Authored by: OpenCode
func replayAssetInput(basisState assetBasisState, scopedInput scopedActivityInput, deterministicOrder int, selectedYear int) (assetInputReplayResult, error) {
	var input = scopedInput.Input
	var _, err = basisState.OpenQuantity()
	if err != nil {
		return assetInputReplayResult{}, newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			"could not determine the asset quantity before applying the activity",
			err,
		)
	}

	var application basisApplicationResult
	application, err = applyBasisInput(basisState, scopedInput, deterministicOrder)
	if err != nil {
		return assetInputReplayResult{}, err
	}

	var quantityAfter apd.Decimal
	quantityAfter, err = basisState.OpenQuantity()
	if err != nil {
		return assetInputReplayResult{}, newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			"could not determine the asset quantity after applying the activity",
			err,
		)
	}
	var basisAfter apd.Decimal
	basisAfter, err = basisState.OpenBasis()
	if err != nil {
		return assetInputReplayResult{}, newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			"could not determine the asset basis after applying the activity",
			err,
		)
	}

	var replayResult = assetInputReplayResult{reachedZero: application.reachedZero}
	if input.SourceYear != selectedYear {
		return replayResult, nil
	}

	replayResult.activityRow, replayResult.liquidationSummary, replayResult.yearlyNetDelta, err = buildInYearArtifacts(input, basisAfter, quantityAfter, application)
	if err != nil {
		return assetInputReplayResult{}, err
	}

	return replayResult, nil
}

// applyBasisInput routes one activity through the selected cost-basis state.
// Authored by: OpenCode
func applyBasisInput(basisState assetBasisState, scopedInput scopedActivityInput, deterministicOrder int) (basisApplicationResult, error) {
	var input = scopedInput.Input
	switch input.ActivityType {
	case syncmodel.ActivityTypeBuy:
		return applyAcquisition(basisState, scopedInput, deterministicOrder)
	case syncmodel.ActivityTypeSell:
		if input.IsZeroPricedHoldingReduction {
			return applyZeroPricedHoldingReduction(basisState, scopedInput)
		}
		return applyPricedLiquidation(basisState, scopedInput)
	default:
		return basisApplicationResult{}, newInputCalculationError(
			reportmodel.CalculationErrorKindActivityInput,
			input,
			fmt.Sprintf("unsupported activity type %q", input.ActivityType),
			nil,
		)
	}
}

// applyAcquisition adds one priced BUY row into the active basis state.
// Authored by: OpenCode
func applyAcquisition(basisState assetBasisState, scopedInput scopedActivityInput, deterministicOrder int) (basisApplicationResult, error) {
	var input = scopedInput.Input
	if input.GrossValue == nil || input.FeeAmount == nil {
		return basisApplicationResult{}, newInputCalculationError(
			reportmodel.CalculationErrorKindActivityInput,
			input,
			"priced BUY activity requires gross value and fee amounts",
			nil,
		)
	}

	var acquisitionBasis, err = addCalculationDecimal(*input.GrossValue, *input.FeeAmount)
	if err != nil {
		return basisApplicationResult{}, newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			"could not calculate acquisition basis",
			err,
		)
	}

	err = basisState.AddAcquisition(basisAcquisitionInput{
		SourceID:           input.SourceID,
		AcquiredAt:         sourceCalendarDate(input.OccurredAt),
		DeterministicOrder: deterministicOrder,
		Quantity:           input.Quantity,
		Basis:              acquisitionBasis,
		ApplicableScopeKey: scopedInput.ApplicableScope.ScopeKey,
	})
	if err != nil {
		return basisApplicationResult{}, newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			"could not add the acquisition into the active basis state",
			err,
		)
	}

	return basisApplicationResult{}, nil
}

// applyZeroPricedHoldingReduction removes quantity and basis without proceeds or
// realized gain or loss.
// Authored by: OpenCode
func applyZeroPricedHoldingReduction(basisState assetBasisState, scopedInput scopedActivityInput) (basisApplicationResult, error) {
	var input = scopedInput.Input
	var disposal, err = basisState.Dispose(basisDisposalInput{Quantity: input.Quantity, ApplicableScopeKey: scopedInput.ApplicableScope.ScopeKey})
	if err != nil {
		return basisApplicationResult{}, newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			"could not remove basis for the zero-priced holding reduction",
			err,
		)
	}

	return basisApplicationResult{allocatedBasis: &disposal.AllocatedBasis, reachedZero: disposal.ReachedZero}, nil
}

// applyPricedLiquidation removes basis and calculates net proceeds and realized
// result for one priced SELL row.
// Authored by: OpenCode
func applyPricedLiquidation(basisState assetBasisState, scopedInput scopedActivityInput) (basisApplicationResult, error) {
	var input = scopedInput.Input
	if input.GrossValue == nil || input.FeeAmount == nil {
		return basisApplicationResult{}, newInputCalculationError(
			reportmodel.CalculationErrorKindActivityInput,
			input,
			"priced SELL activity requires gross value and fee amounts",
			nil,
		)
	}

	var netProceeds, err = subtractCalculationDecimal(*input.GrossValue, *input.FeeAmount)
	if err != nil {
		return basisApplicationResult{}, newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			"could not calculate net liquidation proceeds",
			err,
		)
	}

	var disposal basisDisposalResult
	disposal, err = basisState.Dispose(basisDisposalInput{Quantity: input.Quantity, ApplicableScopeKey: scopedInput.ApplicableScope.ScopeKey})
	if err != nil {
		return basisApplicationResult{}, newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			"could not allocate basis for the priced liquidation",
			err,
		)
	}

	var gainOrLoss apd.Decimal
	gainOrLoss, err = subtractCalculationDecimal(netProceeds, disposal.AllocatedBasis)
	if err != nil {
		return basisApplicationResult{}, newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			"could not calculate the liquidation gain or loss",
			err,
		)
	}

	return basisApplicationResult{
		allocatedBasis: &disposal.AllocatedBasis,
		netProceeds:    &netProceeds,
		gainOrLoss:     &gainOrLoss,
		reachedZero:    disposal.ReachedZero,
	}, nil
}

// buildInYearArtifacts creates the detail-row and liquidation-summary values
// rendered for one selected-year activity.
// Authored by: OpenCode
func buildInYearArtifacts(
	input reportmodel.ActivityCalculationInput,
	basisAfter apd.Decimal,
	quantityAfter apd.Decimal,
	application basisApplicationResult,
) (*reportmodel.AssetActivityRow, *reportmodel.LiquidationCalculation, apd.Decimal, error) {
	var row = &reportmodel.AssetActivityRow{
		SourceID:                    input.SourceID,
		OccurredAt:                  input.OccurredAt,
		ActivityType:                input.ActivityType,
		Quantity:                    input.Quantity,
		UnitPrice:                   input.UnitPrice,
		GrossValue:                  input.GrossValue,
		FeeAmount:                   input.FeeAmount,
		BasisAfterRow:               basisAfter,
		CalculationCurrency:         reportCalculationCurrencyLabel,
		QuantityAfterRow:            quantityAfter,
		HoldingReductionExplanation: strings.TrimSpace(input.Comment),
	}

	if !input.IsZeroPricedHoldingReduction {
		row.ActivityCurrency = input.SelectedCurrencyCode
		row.HoldingReductionExplanation = ""
	}

	if err := row.Validate(); err != nil {
		return nil, nil, apd.Decimal{}, newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			"could not build the in-year activity row",
			err,
		)
	}

	if application.gainOrLoss == nil || application.netProceeds == nil || application.allocatedBasis == nil {
		return row, nil, apd.Decimal{}, nil
	}

	var liquidation = &reportmodel.LiquidationCalculation{
		SourceID:               input.SourceID,
		OccurredAt:             input.OccurredAt,
		DisposedQuantity:       input.Quantity,
		AllocatedBasis:         *application.allocatedBasis,
		NetLiquidationProceeds: *application.netProceeds,
		GainOrLoss:             *application.gainOrLoss,
		ActivityCurrency:       input.SelectedCurrencyCode,
		CalculationCurrency:    reportCalculationCurrencyLabel,
	}
	if err := liquidation.Validate(); err != nil {
		return nil, nil, apd.Decimal{}, newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			"could not build the liquidation summary",
			err,
		)
	}

	row.LiquidationCalculation = liquidation
	return row, liquidation, *application.gainOrLoss, nil
}

// buildAssetCalculationResult converts one replayed asset timeline into the
// top-level report-section models.
// Authored by: OpenCode
func buildAssetCalculationResult(group assetInputGroup, replayState assetReplayState) (assetCalculationResult, error) {
	var result = assetCalculationResult{
		IncludeInMain: replayState.closingQuantity.Sign() > 0 || replayState.hadInYearFullLiquidation,
		YearlyNet:     replayState.yearlyNet,
	}

	if replayState.fullLiquidationCount > 0 {
		var mainSectionStatus = reportmodel.ReferenceSectionStatusReferenceOnly
		if result.IncludeInMain {
			mainSectionStatus = reportmodel.ReferenceSectionStatusIncludedInMainSections
		}

		var referenceEntry, err = reportmodel.NewReferenceLiquidationEntry(
			group.AssetIdentityKey,
			group.DisplayLabel,
			replayState.fullLiquidationCount,
			mainSectionStatus,
		)
		if err != nil {
			return assetCalculationResult{}, reportmodel.NewCalculationError(
				reportmodel.CalculationErrorKindBasisAllocation,
				"could not build the reference-section entry",
				"",
				group.DisplayLabel,
				err,
			)
		}
		result.ReferenceEntry = &referenceEntry
	}

	if !result.IncludeInMain {
		return result, nil
	}

	var summaryEntry, err = reportmodel.NewAssetSummaryEntry(
		group.AssetIdentityKey,
		group.DisplayLabel,
		replayState.yearlyNet,
		reportCalculationCurrencyLabel,
	)
	if err != nil {
		return assetCalculationResult{}, reportmodel.NewCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			"could not build the summary entry",
			"",
			group.DisplayLabel,
			err,
		)
	}

	var detailSection reportmodel.AssetDetailSection
	detailSection, err = reportmodel.NewAssetDetailSection(
		group.AssetIdentityKey,
		group.DisplayLabel,
		replayState.openingQuantity,
		replayState.openingBasis,
		replayState.closingQuantity,
		replayState.closingBasis,
		reportCalculationCurrencyLabel,
		replayState.activityRows,
		replayState.liquidationSummaries,
	)
	if err != nil {
		return assetCalculationResult{}, reportmodel.NewCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			"could not build the detail section",
			"",
			group.DisplayLabel,
			err,
		)
	}

	result.SummaryEntry = summaryEntry
	result.DetailSection = detailSection
	return result, nil
}

// newAssetBasisState creates one method-specific open-position state for the
// requested cost-basis method.
// Authored by: OpenCode
func newAssetBasisState(method reportmodel.CostBasisMethod) (assetBasisState, error) {
	switch method {
	case reportmodel.CostBasisMethodFIFO:
		var state, err = newLotMethodState(reportbasis.LotMethodFIFO)
		if err != nil {
			return nil, err
		}
		return lotBasisState{state: state}, nil
	case reportmodel.CostBasisMethodLIFO:
		var state, err = newLotMethodState(reportbasis.LotMethodLIFO)
		if err != nil {
			return nil, err
		}
		return lotBasisState{state: state}, nil
	case reportmodel.CostBasisMethodHIFO:
		var state, err = newLotMethodState(reportbasis.LotMethodHIFO)
		if err != nil {
			return nil, err
		}
		return lotBasisState{state: state}, nil
	case reportmodel.CostBasisMethodAverageCost:
		return averageCostBasisState{state: reportbasis.NewAverageCostState()}, nil
	case reportmodel.CostBasisMethodScopeLocalHybrid:
		return scopeLocalHybridBasisState{state: reportbasis.NewScopeLocalHybridState()}, nil
	default:
		return nil, fmt.Errorf("unsupported cost basis method %q", method)
	}
}

// AddAcquisition adds one acquisition lot to a lot-based method state.
// Authored by: OpenCode
func (state lotBasisState) AddAcquisition(input basisAcquisitionInput) error {
	return state.state.AddAcquisition(reportbasis.LotAcquisition{
		SourceID:           input.SourceID,
		AcquiredAt:         input.AcquiredAt,
		DeterministicOrder: input.DeterministicOrder,
		RemainingQuantity:  input.Quantity,
		RemainingBasis:     input.Basis,
	})
}

// Dispose removes one quantity from a lot-based method state and returns the
// exact allocated basis.
// Authored by: OpenCode
func (state lotBasisState) Dispose(input basisDisposalInput) (basisDisposalResult, error) {
	var result, err = state.state.Dispose(input.Quantity)
	if err != nil {
		return basisDisposalResult{}, err
	}
	var remainingQuantity apd.Decimal
	remainingQuantity, err = lotStateTotalOpenQuantity(state.state)
	if err != nil {
		return basisDisposalResult{}, err
	}

	return basisDisposalResult{AllocatedBasis: result.AllocatedBasis, ReachedZero: remainingQuantity.Sign() == 0}, nil
}

// OpenQuantity returns the exact remaining lot quantity.
// Authored by: OpenCode
func (state lotBasisState) OpenQuantity() (apd.Decimal, error) {
	return state.state.TotalOpenQuantity()
}

// OpenBasis returns the exact remaining lot basis.
// Authored by: OpenCode
func (state lotBasisState) OpenBasis() (apd.Decimal, error) {
	return state.state.TotalOpenBasis()
}

// AddAcquisition adds one acquisition into the moving average-cost pool.
// Authored by: OpenCode
func (state averageCostBasisState) AddAcquisition(input basisAcquisitionInput) error {
	return state.state.AddAcquisition(input.Quantity, input.Basis)
}

// Dispose removes one quantity from the moving average-cost pool and returns the
// exact allocated basis.
// Authored by: OpenCode
func (state averageCostBasisState) Dispose(input basisDisposalInput) (basisDisposalResult, error) {
	var result, err = state.state.Dispose(input.Quantity)
	if err != nil {
		return basisDisposalResult{}, err
	}

	return basisDisposalResult{AllocatedBasis: result.AllocatedBasis, ReachedZero: result.RemainingQuantity.Sign() == 0}, nil
}

// AddAcquisition adds one acquisition into one scope-local scope partition.
// Authored by: OpenCode
func (state scopeLocalHybridBasisState) AddAcquisition(input basisAcquisitionInput) error {
	return state.state.AddAcquisition(reportbasis.ScopeLocalHybridAcquisition{
		SourceID:           input.SourceID,
		ScopeKey:           input.ApplicableScopeKey,
		AcquiredAt:         input.AcquiredAt,
		DeterministicOrder: input.DeterministicOrder,
		Quantity:           input.Quantity,
		Basis:              input.Basis,
	})
}

// Dispose removes one quantity from one scope-local scope partition.
// Authored by: OpenCode
func (state scopeLocalHybridBasisState) Dispose(input basisDisposalInput) (basisDisposalResult, error) {
	var result, err = state.state.Dispose(input.ApplicableScopeKey, input.Quantity)
	if err != nil {
		return basisDisposalResult{}, err
	}

	return basisDisposalResult{AllocatedBasis: result.AllocatedBasis, ReachedZero: result.ReachedZero}, nil
}

// OpenQuantity returns the exact remaining quantity across all open scopes.
// Authored by: OpenCode
func (state scopeLocalHybridBasisState) OpenQuantity() (apd.Decimal, error) {
	return state.state.TotalOpenQuantity()
}

// OpenBasis returns the exact remaining basis across all open scopes.
// Authored by: OpenCode
func (state scopeLocalHybridBasisState) OpenBasis() (apd.Decimal, error) {
	return state.state.TotalOpenBasis()
}

// OpenQuantity returns the exact remaining moving-pool quantity.
// Authored by: OpenCode
func (state averageCostBasisState) OpenQuantity() (apd.Decimal, error) {
	return state.state.Quantity(), nil
}

// OpenBasis returns the exact remaining moving-pool basis.
// Authored by: OpenCode
func (state averageCostBasisState) OpenBasis() (apd.Decimal, error) {
	return state.state.Basis(), nil
}

// sourceCalendarDate normalizes one parsed activity timestamp down to its
// source-calendar date so lot chronology ignores time-of-day precision.
// Authored by: OpenCode
func sourceCalendarDate(occurredAt time.Time) time.Time {
	var year, month, day = occurredAt.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// addCalculationDecimal adds two exact calculation decimals.
// Authored by: OpenCode
func addCalculationDecimal(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
	if err := requireFiniteDecimal(left, "left calculation decimal"); err != nil {
		return apd.Decimal{}, err
	}
	if err := requireFiniteDecimal(right, "right calculation decimal"); err != nil {
		return apd.Decimal{}, err
	}

	var sum apd.Decimal
	if _, err := addCalculationOperation(&sum, &left, &right); err != nil {
		return apd.Decimal{}, fmt.Errorf("add calculation decimals: %w", err)
	}

	return sum, nil
}

// subtractCalculationDecimal subtracts one exact calculation decimal from
// another.
// Authored by: OpenCode
func subtractCalculationDecimal(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
	if err := requireFiniteDecimal(left, "left calculation decimal"); err != nil {
		return apd.Decimal{}, err
	}
	if err := requireFiniteDecimal(right, "right calculation decimal"); err != nil {
		return apd.Decimal{}, err
	}

	var difference apd.Decimal
	if _, err := subtractCalculationOperation(&difference, &left, &right); err != nil {
		return apd.Decimal{}, fmt.Errorf("subtract calculation decimals: %w", err)
	}

	return difference, nil
}

// newRecordCalculationError creates one structured calculation error from a
// normalized synced activity record.
// Authored by: OpenCode
func newRecordCalculationError(kind reportmodel.CalculationErrorKind, record syncmodel.ActivityRecord, message string, cause error) error {
	return reportmodel.NewCalculationError(kind, message, strings.TrimSpace(record.SourceID), activityDisplayLabel(record), cause).WithPersistedActivityRecord(&record)
}

// newInputCalculationError creates one structured calculation error from a
// selected activity calculation input.
// Authored by: OpenCode
func newInputCalculationError(kind reportmodel.CalculationErrorKind, input reportmodel.ActivityCalculationInput, message string, cause error) error {
	return reportmodel.NewCalculationError(kind, message, strings.TrimSpace(input.SourceID), strings.TrimSpace(input.DisplayLabel), cause).WithPersistedActivityRecord(input.PersistedActivityRecord)
}

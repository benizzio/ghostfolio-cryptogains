// Package calculate defines single-activity replay and basis application for
// yearly report calculation.
// Authored by: OpenCode
package calculate

import (
	"fmt"
	"strings"
	"time"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

// basisApplicationResult stores one activity's basis and proceeds effects after
// the selected cost-basis method has been applied.
// Authored by: OpenCode
type basisApplicationResult struct {
	allocatedBasis *apd.Decimal
	netProceeds    *apd.Decimal
	gainOrLoss     *apd.Decimal
	basisMatches   []reportmodel.BasisMatch
	reachedZero    bool
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

	var acquisitionBasis, err = supportmath.Add(*input.GrossValue, *input.FeeAmount, "left calculation decimal", "right calculation decimal", "add calculation decimals")
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

	return basisApplicationResult{allocatedBasis: &disposal.AllocatedBasis, basisMatches: disposal.Matches, reachedZero: disposal.ReachedZero}, nil
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

	var netProceeds, err = supportmath.Subtract(*input.GrossValue, *input.FeeAmount, "left calculation decimal", "right calculation decimal", "subtract calculation decimals")
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
	gainOrLoss, err = supportmath.Subtract(netProceeds, disposal.AllocatedBasis, "left calculation decimal", "right calculation decimal", "subtract calculation decimals")
	if err != nil {
		return basisApplicationResult{}, newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			"could not calculate the liquidation gain or loss",
			err,
		)
	}

	var basisMatches []reportmodel.BasisMatch
	basisMatches, err = buildPricedLiquidationMatches(disposal.Matches, input.Quantity, netProceeds)
	if err != nil {
		return basisApplicationResult{}, newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			"could not calculate fragment-level priced liquidation matches",
			err,
		)
	}

	return basisApplicationResult{
		allocatedBasis: &disposal.AllocatedBasis,
		netProceeds:    &netProceeds,
		gainOrLoss:     &gainOrLoss,
		basisMatches:   basisMatches,
		reachedZero:    disposal.ReachedZero,
	}, nil
}

// buildPricedLiquidationMatches allocates one liquidation's proceeds through the
// rounded proceeds-per-unit intermediate defined by the report rules.
// Authored by: OpenCode
func buildPricedLiquidationMatches(matches []reportmodel.BasisMatch, disposedQuantity apd.Decimal, netProceeds apd.Decimal) ([]reportmodel.BasisMatch, error) {
	if len(matches) == 0 {
		return nil, nil
	}
	if err := supportmath.RequireFinite(disposedQuantity, "disposed quantity"); err != nil {
		return nil, err
	}
	if disposedQuantity.Sign() <= 0 {
		return nil, fmt.Errorf("disposed quantity must be greater than zero")
	}
	if err := supportmath.RequireFinite(netProceeds, "net proceeds"); err != nil {
		return nil, err
	}

	var proceedsPerUnit, err = reportDivideRoundHalfUp(netProceeds, disposedQuantity)
	if err != nil {
		return nil, fmt.Errorf("calculate proceeds per unit: %w", err)
	}

	var enriched = make([]reportmodel.BasisMatch, 0, len(matches))
	for _, match := range matches {
		var matchCopy = match
		var matchedProceeds apd.Decimal
		matchedProceeds, err = supportmath.Multiply(proceedsPerUnit, match.MatchedQuantity, "left report decimal", "right report decimal", "multiply report decimals")
		if err != nil {
			return nil, fmt.Errorf("calculate matched proceeds for acquisition %q: %w", strings.TrimSpace(match.AcquisitionSourceID), err)
		}
		var matchedGainOrLoss apd.Decimal
		matchedGainOrLoss, err = supportmath.Subtract(matchedProceeds, match.MatchedBasis, "left calculation decimal", "right calculation decimal", "subtract calculation decimals")
		if err != nil {
			return nil, fmt.Errorf("calculate matched gain or loss for acquisition %q: %w", strings.TrimSpace(match.AcquisitionSourceID), err)
		}

		matchCopy.MatchedProceeds = &matchedProceeds
		matchCopy.MatchedGainOrLoss = &matchedGainOrLoss
		enriched = append(enriched, matchCopy)
	}

	return enriched, nil
}

// sourceCalendarDate normalizes one parsed activity timestamp down to its
// source-calendar date so lot chronology ignores time-of-day precision.
// Authored by: OpenCode
func sourceCalendarDate(occurredAt time.Time) time.Time {
	var year, month, day = occurredAt.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

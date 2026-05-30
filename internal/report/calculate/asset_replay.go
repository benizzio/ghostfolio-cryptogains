// Package calculate defines per-asset replay state handling for yearly report
// calculation.
// Authored by: OpenCode
package calculate

import (
	"errors"
	"fmt"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	"github.com/cockroachdb/apd/v3"
)

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

// replayAssetGroup replays one asset input history through the selected-year cutoff.
// Authored by: OpenCode
func replayAssetGroup(basisState assetBasisState, scopedInputs []scopedActivityInput, selectedYear int) (assetReplayState, error) {
	var replayState assetReplayState
	var err error

	for index, scopedInput := range scopedInputs {
		var input = scopedInput.Input
		if err = captureOpeningPositionIfNeeded(&replayState, basisState, input.SourceYear, selectedYear); err != nil {
			return assetReplayState{}, newInputCalculationError(
				reportmodel.CalculationErrorKindBasisAllocation,
				input,
				"could not determine the opening position carried into the selected year",
				err,
			)
		}

		var replayResult assetInputReplayResult
		replayResult, err = replayAssetInputFunc(basisState, scopedInput, index+1, selectedYear)
		if err != nil {
			return assetReplayState{}, err
		}

		err = applyReplayResult(&replayState, input, replayResult, selectedYear)
		if err != nil {
			return assetReplayState{}, err
		}
	}

	return finalizeReplayState(basisState, replayState)
}

// applyReplayResult accumulates one replayed activity into the asset replay state.
// Authored by: OpenCode
func applyReplayResult(state *assetReplayState, input reportmodel.ActivityCalculationInput, replayResult assetInputReplayResult, selectedYear int) error {
	if replayResult.reachedZero {
		state.fullLiquidationCount++
		if input.SourceYear == selectedYear {
			state.hadInYearFullLiquidation = true
		}
	}
	if replayResult.activityRow != nil {
		state.activityRows = append(state.activityRows, *replayResult.activityRow)
	}
	if replayResult.liquidationSummary != nil {
		state.liquidationSummaries = append(state.liquidationSummaries, *replayResult.liquidationSummary)
	}

	var nextYearlyNet, err = supportmath.Add(state.yearlyNet, replayResult.yearlyNetDelta)
	if err != nil {
		return newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			"could not accumulate the asset yearly gain or loss",
			err,
		)
	}
	state.yearlyNet = nextYearlyNet
	return nil
}

// finalizeReplayState snapshots one asset's closing state after replay.
// Authored by: OpenCode
func finalizeReplayState(basisState assetBasisState, replayState assetReplayState) (assetReplayState, error) {
	var err error
	replayState.closingQuantity, err = basisState.OpenQuantity()
	if err != nil {
		return assetReplayState{}, fmt.Errorf("could not determine the asset closing quantity: %w", err)
	}
	replayState.closingBasis, err = basisState.OpenBasis()
	if err != nil {
		return assetReplayState{}, fmt.Errorf("could not determine the asset closing basis: %w", err)
	}
	if !replayState.openingCaptured {
		replayState.openingQuantity = replayState.closingQuantity
		replayState.openingBasis = replayState.closingBasis
	}

	return replayState, nil
}

// wrapAssetGroupReplayError preserves group context for replay-finalization failures.
// Authored by: OpenCode
func wrapAssetGroupReplayError(group assetInputGroup, err error) error {
	var calculationErr *reportmodel.CalculationError
	if errors.As(err, &calculationErr) {
		return err
	}

	return newGroupCalculationError(reportmodel.CalculationErrorKindBasisAllocation, group, err.Error(), err)
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

// Package calculate defines report-model artifact construction for calculated
// asset timelines.
// Authored by: OpenCode
package calculate

import (
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/cockroachdb/apd/v3"
)

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
		Matches:                append([]reportmodel.BasisMatch(nil), application.basisMatches...),
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

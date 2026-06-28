// Package calculate defines report-model artifact construction for calculated
// asset timelines.
// Authored by: OpenCode
package calculate

import (
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	datesupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/date"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
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
	var calculationCurrency = inputCalculationCurrency(input)
	var activityCurrency = inputActivityCurrency(input)
	var row = &reportmodel.AssetActivityRow{
		SourceID:                    input.SourceID,
		OccurredAt:                  input.OccurredAt,
		ActivityType:                input.ActivityType,
		Quantity:                    input.Quantity,
		UnitPrice:                   input.UnitPrice,
		GrossValue:                  input.GrossValue,
		FeeAmount:                   input.FeeAmount,
		BasisAfterRow:               basisAfter,
		CalculationCurrency:         calculationCurrency,
		QuantityAfterRow:            quantityAfter,
		ConversionStatus:            input.ConversionStatus,
		HoldingReductionExplanation: strings.TrimSpace(input.Comment),
	}

	if !input.IsZeroPricedHoldingReduction {
		row.ActivityCurrency = activityCurrency
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
		CalculationCurrency:    calculationCurrency,
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

// buildConversionAuditEntry creates the report-visible conversion evidence for
// one converted priced activity.
// Authored by: OpenCode
func buildConversionAuditEntry(
	group assetInputGroup,
	input reportmodel.ActivityCalculationInput,
	evidence reportmodel.ExchangeRateEvidence,
	amounts []reportmodel.ConvertedActivityAmount,
) (reportmodel.ConversionAuditEntry, error) {
	var entry = reportmodel.ConversionAuditEntry{
		SourceID:           input.SourceID,
		AssetLabel:         conversionAuditAssetLabel(group, input),
		ActivityDate:       datesupport.CalendarDate(input.OccurredAt),
		SourceCurrency:     evidence.SourceCurrency,
		ReportBaseCurrency: evidence.BaseCurrency,
		RateDate:           evidence.RateDate,
		RateAuthority:      evidence.Authority,
		RateKind:           evidence.RateKind,
		RateValue:          decimalsupport.Clone(evidence.RateValue),
		QuoteDirection:     evidence.QuoteDirection,
		Amounts:            append([]reportmodel.ConvertedActivityAmount(nil), amounts...),
	}
	if err := entry.Validate(); err != nil {
		return reportmodel.ConversionAuditEntry{}, newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			"could not build the conversion audit entry",
			err,
		)
	}

	return entry, nil
}

// conversionAuditAssetLabel returns the stable report label used in conversion
// audit entries.
// Authored by: OpenCode
func conversionAuditAssetLabel(group assetInputGroup, input reportmodel.ActivityCalculationInput) string {
	var label = strings.TrimSpace(group.DisplayLabel)
	if label != "" {
		return label
	}

	label = strings.TrimSpace(input.DisplayLabel)
	if label != "" {
		return label
	}

	return fmt.Sprintf("asset %s", strings.TrimSpace(group.AssetIdentityKey))
}

// buildAssetCalculationResult converts one replayed asset timeline into the
// top-level report-section models.
// Authored by: OpenCode
func buildAssetCalculationResult(group assetInputGroup, replayState assetReplayState) (assetCalculationResult, error) {
	var calculationCurrency, currencyErr = groupCalculationCurrency(group)
	if currencyErr != nil {
		return assetCalculationResult{}, reportmodel.NewCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			"could not determine the asset calculation currency",
			"",
			group.DisplayLabel,
			currencyErr,
		)
	}
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
		calculationCurrency,
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
		calculationCurrency,
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

// groupCalculationCurrency returns the report calculation currency prepared on
// the group's inputs.
// Authored by: OpenCode
func groupCalculationCurrency(group assetInputGroup) (string, error) {
	var selectedCurrency string
	for _, input := range group.Inputs {
		var currency = inputCalculationCurrency(input)
		if currency != reportCalculationCurrencyLabel {
			if selectedCurrency == "" {
				selectedCurrency = currency
				continue
			}
			if selectedCurrency != currency {
				return "", fmt.Errorf("mixed calculation currencies %q and %q", selectedCurrency, currency)
			}
		}
	}
	if selectedCurrency != "" {
		return selectedCurrency, nil
	}

	return reportCalculationCurrencyLabel, nil
}

// inputCalculationCurrency returns the selected report currency prepared by the
// conversion boundary.
// Authored by: OpenCode
func inputCalculationCurrency(input reportmodel.ActivityCalculationInput) string {
	var currency = strings.TrimSpace(input.SelectedCurrencyCode)
	if currency == "" {
		return reportCalculationCurrencyLabel
	}

	return currency
}

// inputActivityCurrency returns the original selected activity currency when the
// conversion boundary preserved it separately from report calculation currency.
// Authored by: OpenCode
func inputActivityCurrency(input reportmodel.ActivityCalculationInput) string {
	var currency = strings.TrimSpace(input.ActivityCurrencyCode)
	if currency != "" {
		return currency
	}

	return strings.TrimSpace(input.SelectedCurrencyCode)
}

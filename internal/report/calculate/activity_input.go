// Package calculate defines yearly gains-and-losses report calculation
// services built on normalized protected activity history.
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

// SelectActivityCalculationInput converts one normalized activity record into
// one report calculation input using the report slice's strict single-tier
// currency-context rules.
//
// Example:
//
//	input, err := calculate.SelectActivityCalculationInput(record)
//	if err != nil {
//		panic(err)
//	}
//	_ = input.SelectedCurrencyCode
//
// The selector applies the `order -> asset_profile -> base` priority, skips
// tiers that do not carry an explicit currency code, continues past explicit-
// currency tiers that still cannot safely supply one complete same-tier input,
// preserves the chosen explicit currency code, and leaves explained zero-
// priced `SELL` records without a selected currency context while still
// preserving any explicit zero-valued source details already stored for that
// row.
// Authored by: OpenCode
func SelectActivityCalculationInput(record syncmodel.ActivityRecord) (reportmodel.ActivityCalculationInput, error) {
	var occurredAt, err = parseActivityOccurredAt(record)
	if err != nil {
		return reportmodel.ActivityCalculationInput{}, err
	}

	var input = reportmodel.ActivityCalculationInput{
		SourceID:         strings.TrimSpace(record.SourceID),
		OccurredAt:       occurredAt,
		SourceYear:       occurredAt.Year(),
		ActivityType:     reportActivityType(record.ActivityType),
		AssetIdentityKey: strings.TrimSpace(record.AssetIdentityKey),
		DisplayLabel:     activityDisplayLabel(record),
		Quantity:         record.Quantity,
		SourceScope:      reportSourceScope(record.SourceScope),
		Comment:          strings.TrimSpace(record.Comment),
	}

	var zeroPricedValues zeroPricedHoldingReductionValues
	var isZeroPriced bool
	zeroPricedValues, isZeroPriced, err = selectZeroPricedHoldingReductionValues(record)
	if err != nil {
		return reportmodel.ActivityCalculationInput{}, err
	}
	if isZeroPriced {
		input.IsZeroPricedHoldingReduction = true
		input.GrossValue = zeroPricedValues.grossValue
		input.FeeAmount = zeroPricedValues.feeAmount
		input.UnitPrice = zeroPricedValues.unitPrice
		return input, nil
	}

	if err = requirePositivePricedQuantity(record); err != nil {
		return reportmodel.ActivityCalculationInput{}, err
	}

	var tierInput reportmodel.ActivityCalculationInput
	tierInput, err = selectPricedActivityTier(record)
	if err != nil {
		return reportmodel.ActivityCalculationInput{}, err
	}

	input.GrossValue = tierInput.GrossValue
	input.FeeAmount = tierInput.FeeAmount
	input.UnitPrice = tierInput.UnitPrice
	input.SelectedCurrencyContext = tierInput.SelectedCurrencyContext
	input.SelectedCurrencyCode = tierInput.SelectedCurrencyCode

	return input, nil
}

// reportActivityType maps the normalized synced activity type into the report
// model's owned activity type value.
// Authored by: OpenCode
func reportActivityType(activityType syncmodel.ActivityType) reportmodel.ActivityType {
	return reportmodel.ActivityType(activityType)
}

// reportSourceScope maps normalized synced source scope into the report model's
// owned source scope value.
// Authored by: OpenCode
func reportSourceScope(scope *syncmodel.SourceScope) *reportmodel.SourceScope {
	if scope == nil {
		return nil
	}

	return &reportmodel.SourceScope{
		ID:          scope.ID,
		Name:        scope.Name,
		Kind:        reportmodel.SourceScopeKind(scope.Kind),
		Reliability: reportmodel.ScopeReliability(scope.Reliability),
	}
}

// activityMoneyTier stores one candidate activity currency context before it is
// copied into the report activity input.
// Authored by: OpenCode
type activityMoneyTier struct {
	name         reportmodel.SelectedCurrencyContext
	currencyCode string
	unitPrice    *apd.Decimal
	grossValue   *apd.Decimal
	feeAmount    *apd.Decimal
	hasAnyValue  bool
}

// zeroPricedHoldingReductionValues stores optional preserved source-field zeros
// that remain available on explained zero-priced holding reductions without
// creating a selected currency context.
// Authored by: OpenCode
type zeroPricedHoldingReductionValues struct {
	unitPrice  *apd.Decimal
	grossValue *apd.Decimal
	feeAmount  *apd.Decimal
}

// parseActivityOccurredAt parses one normalized activity timestamp using the
// stored source offset.
// Authored by: OpenCode
func parseActivityOccurredAt(record syncmodel.ActivityRecord) (time.Time, error) {
	var occurredAtText = strings.TrimSpace(record.OccurredAt)
	if occurredAtText == "" {
		return time.Time{}, fmt.Errorf("activity %q occurred_at is required", strings.TrimSpace(record.SourceID))
	}

	var occurredAt, err = time.Parse(time.RFC3339Nano, occurredAtText)
	if err != nil {
		return time.Time{}, fmt.Errorf("activity %q occurred_at is invalid: %w", strings.TrimSpace(record.SourceID), err)
	}

	return occurredAt, nil
}

// activityDisplayLabel prefers the symbol label and falls back to the asset
// name when needed.
// Authored by: OpenCode
func activityDisplayLabel(record syncmodel.ActivityRecord) string {
	var symbol = strings.TrimSpace(record.AssetSymbol)
	if symbol != "" {
		return symbol
	}

	return strings.TrimSpace(record.AssetName)
}

// selectZeroPricedHoldingReductionValues identifies the explained zero-priced
// SELL shape and preserves explicit zero-valued source fields without creating
// a selected activity currency context.
// Authored by: OpenCode
func selectZeroPricedHoldingReductionValues(record syncmodel.ActivityRecord) (zeroPricedHoldingReductionValues, bool, error) {
	if record.ActivityType != syncmodel.ActivityTypeSell {
		return zeroPricedHoldingReductionValues{}, false, nil
	}
	if strings.TrimSpace(record.Comment) == "" {
		return zeroPricedHoldingReductionValues{}, false, nil
	}

	var explicitValues = []*apd.Decimal{
		record.OrderUnitPrice,
		record.OrderGrossValue,
		record.OrderFeeAmount,
		record.AssetProfileUnitPrice,
		record.AssetProfileFeeAmount,
		record.BaseGrossValue,
		record.BaseFeeAmount,
	}
	var allZero, err = allPresentDecimalsAreZero(explicitValues)
	if err != nil {
		return zeroPricedHoldingReductionValues{}, false, fmt.Errorf(
			"activity %q zero-priced holding reduction values are invalid: %w",
			strings.TrimSpace(record.SourceID),
			err,
		)
	}
	if !allZero {
		return zeroPricedHoldingReductionValues{}, false, nil
	}

	return zeroPricedHoldingReductionValues{
		unitPrice:  firstExplicitZeroValue(record.OrderUnitPrice, record.AssetProfileUnitPrice),
		grossValue: firstExplicitZeroValue(record.OrderGrossValue, record.BaseGrossValue),
		feeAmount:  firstExplicitZeroValue(record.OrderFeeAmount, record.AssetProfileFeeAmount, record.BaseFeeAmount),
	}, true, nil
}

// allPresentDecimalsAreZero reports whether every provided decimal pointer is
// either missing or numerically zero.
// Authored by: OpenCode
func allPresentDecimalsAreZero(values []*apd.Decimal) (bool, error) {
	for _, value := range values {
		if value == nil {
			continue
		}

		var isZero, err = supportmath.IsZero(*value)
		if err != nil {
			return false, fmt.Errorf("report decimal is invalid: %w", err)
		}
		if !isZero {
			return false, nil
		}
	}

	return true, nil
}

// firstExplicitZeroValue returns the first explicitly stored prevalidated zero-
// valued source field from the provided priority list.
// Authored by: OpenCode
func firstExplicitZeroValue(values ...*apd.Decimal) *apd.Decimal {
	for _, value := range values {
		if value == nil {
			continue
		}

		return value
	}

	return nil
}

// requirePositivePricedQuantity enforces the priced-activity quantity
// precondition before tier selection.
// Authored by: OpenCode
func requirePositivePricedQuantity(record syncmodel.ActivityRecord) error {
	var comparison, err = supportmath.Compare(record.Quantity, apd.Decimal{})
	if err != nil {
		return fmt.Errorf("activity %q quantity is invalid: %w", strings.TrimSpace(record.SourceID), err)
	}
	if comparison <= 0 {
		return fmt.Errorf("activity %q priced activity quantity must be greater than zero", strings.TrimSpace(record.SourceID))
	}

	return nil
}

// selectPricedActivityTier applies the `order -> asset_profile -> base`
// priority without mixing tiers.
// Authored by: OpenCode
func selectPricedActivityTier(record syncmodel.ActivityRecord) (reportmodel.ActivityCalculationInput, error) {
	var tiers = []activityMoneyTier{
		buildOrderTier(record),
		buildAssetProfileTier(record),
		buildBaseTier(record),
	}
	var lastExplicitTierErr error

	for _, tier := range tiers {
		if !tier.hasAnyValue {
			continue
		}
		if strings.TrimSpace(tier.currencyCode) == "" {
			continue
		}

		var tierInput reportmodel.ActivityCalculationInput
		var err error
		tierInput, err = selectExplicitCurrencyTier(record, tier)
		if err != nil {
			lastExplicitTierErr = err
			continue
		}

		return tierInput, nil
	}
	if lastExplicitTierErr != nil {
		return reportmodel.ActivityCalculationInput{}, lastExplicitTierErr
	}

	return reportmodel.ActivityCalculationInput{}, fmt.Errorf(
		"activity %q priced activity requires one complete order, asset_profile, or base currency context",
		strings.TrimSpace(record.SourceID),
	)
}

// selectExplicitCurrencyTier validates and derives one complete priced-activity
// input from a single explicit-currency tier.
// Authored by: OpenCode
func selectExplicitCurrencyTier(record syncmodel.ActivityRecord, tier activityMoneyTier) (reportmodel.ActivityCalculationInput, error) {
	var grossValue, err = selectTierGrossValue(record, tier)
	if err != nil {
		return reportmodel.ActivityCalculationInput{}, err
	}
	if grossValue == nil || tier.feeAmount == nil {
		return reportmodel.ActivityCalculationInput{}, fmt.Errorf(
			"activity %q %s currency context is incomplete; provide or derive gross value and fee from that tier only",
			strings.TrimSpace(record.SourceID),
			tier.name,
		)
	}

	var unitPrice *apd.Decimal
	unitPrice, err = selectTierUnitPrice(record, tier, grossValue)
	if err != nil {
		return reportmodel.ActivityCalculationInput{}, err
	}

	return reportmodel.ActivityCalculationInput{
		GrossValue:              grossValue,
		FeeAmount:               tier.feeAmount,
		UnitPrice:               unitPrice,
		SelectedCurrencyContext: tier.name,
		SelectedCurrencyCode:    tier.currencyCode,
	}, nil
}

// selectTierGrossValue returns one tier gross value, deriving it by same-tier
// multiplication before any division-based fallback is considered.
// Authored by: OpenCode
func selectTierGrossValue(record syncmodel.ActivityRecord, tier activityMoneyTier) (*apd.Decimal, error) {
	if tier.grossValue != nil {
		return tier.grossValue, nil
	}
	if tier.unitPrice == nil {
		return nil, nil
	}

	var derivedGrossValue, err = supportmath.Multiply(record.Quantity, *tier.unitPrice)
	if err != nil {
		return nil, fmt.Errorf(
			"activity %q %s gross value cannot be derived from quantity and unit price: %w",
			strings.TrimSpace(record.SourceID),
			tier.name,
			err,
		)
	}

	return &derivedGrossValue, nil
}

// selectTierUnitPrice returns one tier unit price, deriving it only from same-
// tier inputs under the shared report-calculation decimal policy.
// Authored by: OpenCode
func selectTierUnitPrice(record syncmodel.ActivityRecord, tier activityMoneyTier, grossValue *apd.Decimal) (*apd.Decimal, error) {
	if tier.unitPrice != nil {
		return tier.unitPrice, nil
	}
	if grossValue == nil {
		return nil, nil
	}

	var derivedUnitPrice apd.Decimal
	var err error
	derivedUnitPrice, err = supportmath.DivideFiniteRoundHalfUp(*grossValue, record.Quantity)
	if err != nil {
		return nil, fmt.Errorf(
			"activity %q %s unit price cannot be derived from gross value and quantity: %w",
			strings.TrimSpace(record.SourceID),
			tier.name,
			err,
		)
	}

	return &derivedUnitPrice, nil
}

// buildOrderTier collects the order-tier priced-activity values.
// Authored by: OpenCode
func buildOrderTier(record syncmodel.ActivityRecord) activityMoneyTier {
	var currency = strings.TrimSpace(record.OrderCurrency)
	var hasAnyValue = record.OrderUnitPrice != nil || record.OrderGrossValue != nil || record.OrderFeeAmount != nil

	return activityMoneyTier{
		name:         reportmodel.SelectedCurrencyContextOrder,
		currencyCode: currency,
		unitPrice:    informedTierValue(record.OrderUnitPrice, currency),
		grossValue:   informedTierValue(record.OrderGrossValue, currency),
		feeAmount:    informedTierValue(record.OrderFeeAmount, currency),
		hasAnyValue:  hasAnyValue,
	}
}

// buildAssetProfileTier collects the asset-profile-tier priced-activity values.
// Authored by: OpenCode
func buildAssetProfileTier(record syncmodel.ActivityRecord) activityMoneyTier {
	var currency = strings.TrimSpace(record.AssetProfileCurrency)
	var hasAnyValue = record.AssetProfileUnitPrice != nil || record.AssetProfileFeeAmount != nil

	return activityMoneyTier{
		name:         reportmodel.SelectedCurrencyContextAssetProfile,
		currencyCode: currency,
		unitPrice:    informedTierValue(record.AssetProfileUnitPrice, currency),
		feeAmount:    informedTierValue(record.AssetProfileFeeAmount, currency),
		hasAnyValue:  hasAnyValue,
	}
}

// buildBaseTier collects the base-tier priced-activity values.
// Authored by: OpenCode
func buildBaseTier(record syncmodel.ActivityRecord) activityMoneyTier {
	var currency = strings.TrimSpace(record.BaseCurrency)
	var hasAnyValue = record.BaseGrossValue != nil || record.BaseFeeAmount != nil

	return activityMoneyTier{
		name:         reportmodel.SelectedCurrencyContextBase,
		currencyCode: currency,
		grossValue:   informedTierValue(record.BaseGrossValue, currency),
		feeAmount:    informedTierValue(record.BaseFeeAmount, currency),
		hasAnyValue:  hasAnyValue,
	}
}

// informedTierValue returns one tier value only when the tier keeps an explicit
// currency code.
// Authored by: OpenCode
func informedTierValue(value *apd.Decimal, currency string) *apd.Decimal {
	if value == nil || strings.TrimSpace(currency) == "" {
		return nil
	}

	return value
}

// informedGrossValue derives a tier gross value from unit price only when the
// tier keeps an explicit currency code.
// Authored by: OpenCode
func informedGrossValue(unitPrice *apd.Decimal, currency string, quantity apd.Decimal) *apd.Decimal {
	if unitPrice == nil || strings.TrimSpace(currency) == "" {
		return nil
	}

	var grossValue, err = supportmath.Multiply(quantity, *unitPrice)
	if err != nil {
		return nil
	}

	return &grossValue
}

package validate

import (
	"fmt"
	"strings"

	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

const activityAmountResolutionScale int32 = 16

// resolvedActivityAmounts stores the transient current-slice money view derived
// from one normalized activity record during validation.
// Authored by: OpenCode
type resolvedActivityAmounts struct {
	UnitPrice          *apd.Decimal
	UnitPriceCurrency  string
	GrossValue         *apd.Decimal
	GrossValueCurrency string
	FeeAmount          *apd.Decimal
	FeeAmountCurrency  string
}

// resolveActivityAmounts derives the transient unit-price, gross-value, and fee
// view required by the current validation slice.
// Authored by: OpenCode
func resolveActivityAmounts(record syncmodel.ActivityRecord) (resolvedActivityAmounts, error) {
	var grossValue *apd.Decimal
	var grossValueCurrency string
	var err error

	grossValue, grossValueCurrency, err = resolveGrossValue(record)
	if err != nil {
		return resolvedActivityAmounts{}, err
	}

	var unitPrice *apd.Decimal
	var unitPriceCurrency string
	unitPrice, unitPriceCurrency, err = resolveUnitPrice(record, grossValue, grossValueCurrency)
	if err != nil {
		return resolvedActivityAmounts{}, err
	}

	var feeAmount *apd.Decimal
	var feeAmountCurrency string
	feeAmount, feeAmountCurrency, err = resolveFeeAmount(record)
	if err != nil {
		return resolvedActivityAmounts{}, err
	}

	return resolvedActivityAmounts{
		UnitPrice:          unitPrice,
		UnitPriceCurrency:  unitPriceCurrency,
		GrossValue:         grossValue,
		GrossValueCurrency: grossValueCurrency,
		FeeAmount:          feeAmount,
		FeeAmountCurrency:  feeAmountCurrency,
	}, nil
}

// resolveUnitPrice derives the current-slice unit price view without
// persisting it on the normalized activity record.
// Authored by: OpenCode
func resolveUnitPrice(
	record syncmodel.ActivityRecord,
	grossValue *apd.Decimal,
	grossValueCurrency string,
) (*apd.Decimal, string, error) {
	var value *apd.Decimal
	var currency string
	var ok bool
	var err error

	value, currency, ok = informedActivityAmount(record.OrderUnitPrice, record.OrderCurrency)
	if ok {
		return value, currency, nil
	}

	value, currency, ok = informedActivityAmount(record.AssetProfileUnitPrice, record.AssetProfileCurrency)
	if ok {
		return value, currency, nil
	}

	if grossValue != nil && strings.TrimSpace(grossValueCurrency) != "" {
		var unitPrice apd.Decimal
		unitPrice, err = divideActivityAmountRoundHalfUp(*grossValue, record.Quantity)
		if err != nil {
			return nil, "", fmt.Errorf(
				"activity %q unit price basis input is invalid: %w",
				strings.TrimSpace(record.SourceID),
				err,
			)
		}

		return &unitPrice, strings.TrimSpace(grossValueCurrency), nil
	}

	if hasUnitPriceBasis(record) || hasGrossValueBasis(record) {
		return nil, "", allTierUninformedCurrencyError(record.SourceID, "unit price")
	}

	return nil, "", fmt.Errorf("activity %q unit price basis input is required", strings.TrimSpace(record.SourceID))
}

// resolveGrossValue derives the current-slice gross value view without
// persisting it on the normalized activity record.
// Authored by: OpenCode and benizzio
func resolveGrossValue(record syncmodel.ActivityRecord) (*apd.Decimal, string, error) {
	var value *apd.Decimal
	var currency string
	var ok bool

	value, currency, ok = informedActivityAmount(record.OrderGrossValue, record.OrderCurrency)
	if ok {
		return value, currency, nil
	}

	value, currency, ok = informedActivityAmount(record.OrderUnitPrice, record.OrderCurrency)
	if ok {
		var grossValue apd.Decimal
		var err error
		grossValue, err = multiplyActivityAmount(record.Quantity, *value)
		if err != nil {
			return nil, "", fmt.Errorf(
				"activity %q gross value basis input is invalid: %w",
				strings.TrimSpace(record.SourceID),
				err,
			)
		}
		return &grossValue, currency, nil
	}

	value, currency, ok = informedActivityAmount(record.AssetProfileUnitPrice, record.AssetProfileCurrency)
	if ok {
		var grossValue apd.Decimal
		var err error
		grossValue, err = multiplyActivityAmount(record.Quantity, *value)
		if err != nil {
			return nil, "", fmt.Errorf(
				"activity %q gross value basis input is invalid: %w",
				strings.TrimSpace(record.SourceID),
				err,
			)
		}
		return &grossValue, currency, nil
	}

	value, currency, ok = informedActivityAmount(record.BaseGrossValue, record.BaseCurrency)
	if ok {
		return value, currency, nil
	}

	if hasGrossValueBasis(record) || hasUnitPriceBasis(record) {
		return nil, "", allTierUninformedCurrencyError(record.SourceID, "gross value")
	}

	return nil, "", fmt.Errorf("activity %q gross value basis input is required", strings.TrimSpace(record.SourceID))
}

// resolveFeeAmount selects the current-slice fee view without persisting it on
// the normalized activity record.
// Authored by: OpenCode
func resolveFeeAmount(record syncmodel.ActivityRecord) (*apd.Decimal, string, error) {
	var value *apd.Decimal
	var currency string
	var ok bool

	value, currency, ok = informedActivityAmount(record.OrderFeeAmount, record.OrderCurrency)
	if ok {
		return value, currency, nil
	}

	value, currency, ok = informedActivityAmount(record.AssetProfileFeeAmount, record.AssetProfileCurrency)
	if ok {
		return value, currency, nil
	}

	value, currency, ok = informedActivityAmount(record.BaseFeeAmount, record.BaseCurrency)
	if ok {
		return value, currency, nil
	}

	if hasFeeBasis(record) {
		return nil, "", allTierUninformedCurrencyError(record.SourceID, "fee amount")
	}

	return nil, "", nil
}

// informedActivityAmount returns one preserved amount only when its currency
// tier is informed.
// Authored by: OpenCode
func informedActivityAmount(value *apd.Decimal, currency string) (*apd.Decimal, string, bool) {
	if value == nil {
		return nil, "", false
	}
	if strings.TrimSpace(currency) == "" {
		return nil, "", false
	}

	return value, strings.TrimSpace(currency), true
}

// hasUnitPriceBasis reports whether the record preserves any unit-price input,
// informed or not.
// Authored by: OpenCode
func hasUnitPriceBasis(record syncmodel.ActivityRecord) bool {
	return record.OrderUnitPrice != nil || record.AssetProfileUnitPrice != nil
}

// hasGrossValueBasis reports whether the record preserves any gross-value
// input, informed or not.
// Authored by: OpenCode
func hasGrossValueBasis(record syncmodel.ActivityRecord) bool {
	return record.OrderGrossValue != nil || record.BaseGrossValue != nil
}

// hasFeeBasis reports whether the record preserves any fee input, informed or
// not.
// Authored by: OpenCode
func hasFeeBasis(record syncmodel.ActivityRecord) bool {
	return record.OrderFeeAmount != nil || record.AssetProfileFeeAmount != nil || record.BaseFeeAmount != nil
}

// allTierUninformedCurrencyError describes the BUG-004 validation rule for
// preserved money concepts.
// Authored by: OpenCode
func allTierUninformedCurrencyError(sourceID string, amountName string) error {
	return fmt.Errorf(
		"activity %q %s currency context is uninformed across order, asset-profile, and base tiers",
		strings.TrimSpace(sourceID),
		amountName,
	)
}

// multiplyActivityAmount preserves exact-decimal precision when one transient
// current-slice amount must be derived from quantity and unit price.
// Authored by: OpenCode
func multiplyActivityAmount(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
	return supportmath.ApplyBinaryOperation(
		left,
		right,
		"left activity amount",
		"right activity amount",
		"derive activity amount from quantity and unit price",
		func(result *apd.Decimal, left *apd.Decimal, right *apd.Decimal) (apd.Condition, error) {
			return apd.BaseContext.Mul(result, left, right)
		},
	)
}

// divideActivityAmountRoundHalfUp derives one transient amount using the shared
// 16-decimal round-half-up policy for repeating divisions.
// Authored by: OpenCode
func divideActivityAmountRoundHalfUp(dividend apd.Decimal, divisor apd.Decimal) (apd.Decimal, error) {
	if err := supportmath.RequireFinite(dividend, "derive activity amount from gross value and quantity"); err != nil {
		return apd.Decimal{}, err
	}
	if err := supportmath.RequireFinite(divisor, "derive activity amount from gross value and quantity"); err != nil {
		return apd.Decimal{}, err
	}
	if divisor.Sign() == 0 {
		return apd.Decimal{}, fmt.Errorf("derive activity amount from gross value and quantity: non-zero divisor is required")
	}

	return supportmath.DivideFiniteRoundHalfUp(dividend, divisor, activityAmountResolutionScale)
}

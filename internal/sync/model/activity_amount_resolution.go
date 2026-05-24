// Package model defines normalized sync data structures shared across sync,
// snapshot, and runtime packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"math/big"
	"strings"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

const activityAmountResolutionScale int32 = 16

// ResolvedActivityAmounts stores the transient current-slice money view derived
// from one persisted activity record.
//
// Example:
//
//	amounts, err := model.ResolveActivityAmounts(record)
//	if err != nil {
//		panic(err)
//	}
//	_ = amounts.UnitPrice
//
// The resolved values are not persisted. They exist only so validation code can
// apply the current slice's basis-input rules without forcing the mapper to
// store one selected cross-currency view.
// Diagnostic records preserve source fields instead of using these resolved
// values.
// Authored by: OpenCode
type ResolvedActivityAmounts struct {
	UnitPrice          *apd.Decimal
	UnitPriceCurrency  string
	GrossValue         *apd.Decimal
	GrossValueCurrency string
	FeeAmount          *apd.Decimal
	FeeAmountCurrency  string
}

// ResolveActivityAmounts derives the transient unit-price, gross-value, and fee
// view required by the current slice from one explicit-currency activity record.
//
// Example:
//
//	amounts, err := model.ResolveActivityAmounts(record)
//	if err != nil {
//		panic(err)
//	}
//	_ = amounts.GrossValue
//
// The selection rules stay local to validation. The persisted activity record
// itself keeps only the explicit order-currency,
// asset-profile-currency, and base-currency groups that Ghostfolio exposes.
// Authored by: OpenCode
func ResolveActivityAmounts(record ActivityRecord) (ResolvedActivityAmounts, error) {
	var grossValue *apd.Decimal
	var grossValueCurrency string
	var err error

	grossValue, grossValueCurrency, err = resolveGrossValue(record)
	if err != nil {
		return ResolvedActivityAmounts{}, err
	}

	var unitPrice *apd.Decimal
	var unitPriceCurrency string
	unitPrice, unitPriceCurrency, err = resolveUnitPrice(record, grossValue, grossValueCurrency)
	if err != nil {
		return ResolvedActivityAmounts{}, err
	}

	var feeAmount *apd.Decimal
	var feeAmountCurrency string
	feeAmount, feeAmountCurrency, err = resolveFeeAmount(record)
	if err != nil {
		return ResolvedActivityAmounts{}, err
	}

	return ResolvedActivityAmounts{
		UnitPrice:          unitPrice,
		UnitPriceCurrency:  unitPriceCurrency,
		GrossValue:         grossValue,
		GrossValueCurrency: grossValueCurrency,
		FeeAmount:          feeAmount,
		FeeAmountCurrency:  feeAmountCurrency,
	}, nil
}

// resolveUnitPrice derives the current-slice unit price view without persisting
// it on the activity record.
//
// Authored by: OpenCode
func resolveUnitPrice(
	record ActivityRecord,
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

	if grossValue == nil {
		return nil, "", fmt.Errorf("activity %q unit price basis input is required", strings.TrimSpace(record.SourceID))
	}

	return nil, "", fmt.Errorf("activity %q unit price basis input is required", strings.TrimSpace(record.SourceID))
}

// resolveGrossValue derives the current-slice gross value view without
// persisting it on the activity record.
//
// Authored by: OpenCode and benizzio
func resolveGrossValue(record ActivityRecord) (*apd.Decimal, string, error) {
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
// the activity record.
//
// Authored by: OpenCode
func resolveFeeAmount(record ActivityRecord) (*apd.Decimal, string, error) {
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

// informedActivityAmount returns one preserved amount only when its currency tier is informed.
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

// hasUnitPriceBasis reports whether the record preserves any unit-price input, informed or not.
// Authored by: OpenCode
func hasUnitPriceBasis(record ActivityRecord) bool {
	return record.OrderUnitPrice != nil || record.AssetProfileUnitPrice != nil
}

// hasGrossValueBasis reports whether the record preserves any gross-value input, informed or not.
// Authored by: OpenCode
func hasGrossValueBasis(record ActivityRecord) bool {
	return record.OrderGrossValue != nil || record.BaseGrossValue != nil
}

// hasFeeBasis reports whether the record preserves any fee input, informed or not.
// Authored by: OpenCode
func hasFeeBasis(record ActivityRecord) bool {
	return record.OrderFeeAmount != nil || record.AssetProfileFeeAmount != nil || record.BaseFeeAmount != nil
}

// allTierUninformedCurrencyError describes the BUG-004 validation rule for preserved money concepts.
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
	var product apd.Decimal
	var err error
	_, err = apd.BaseContext.Mul(&product, &left, &right)
	if err != nil {
		return apd.Decimal{}, fmt.Errorf("derive activity amount from quantity and unit price: %w", err)
	}

	return product, nil
}

// divideActivityAmountRoundHalfUp derives one transient amount using the shared
// 16-decimal round-half-up policy for repeating divisions.
// Authored by: OpenCode
func divideActivityAmountRoundHalfUp(dividend apd.Decimal, divisor apd.Decimal) (apd.Decimal, error) {
	if _, err := decimalsupport.CanonicalString(dividend); err != nil {
		return apd.Decimal{}, fmt.Errorf("derive activity amount from gross value and quantity: %w", err)
	}
	if _, err := decimalsupport.CanonicalString(divisor); err != nil {
		return apd.Decimal{}, fmt.Errorf("derive activity amount from gross value and quantity: %w", err)
	}
	if divisor.Sign() == 0 {
		return apd.Decimal{}, fmt.Errorf("derive activity amount from gross value and quantity: non-zero divisor is required")
	}

	var scaledQuotient = scaledRoundedActivityQuotient(dividend, divisor, activityAmountResolutionScale)
	var magnitude = new(big.Int).Set(scaledQuotient)
	var negative = magnitude.Sign() < 0
	if negative {
		magnitude.Abs(magnitude)
	}

	var coefficient apd.BigInt
	coefficient.SetMathBigInt(magnitude)

	return apd.Decimal{
		Form:     apd.Finite,
		Negative: negative,
		Exponent: -activityAmountResolutionScale,
		Coeff:    coefficient,
	}, nil
}

// scaledRoundedActivityQuotient computes one fixed-scale quotient rounded half up.
// Authored by: OpenCode
func scaledRoundedActivityQuotient(dividend apd.Decimal, divisor apd.Decimal, scale int32) *big.Int {
	var dividendNumerator, dividendDenominator = finiteActivityDecimalFraction(dividend)
	var divisorNumerator, divisorDenominator = finiteActivityDecimalFraction(divisor)

	var signNegative = (dividendNumerator.Sign() < 0) != (divisorNumerator.Sign() < 0)
	var numerator = new(big.Int).Mul(new(big.Int).Abs(dividendNumerator), divisorDenominator)
	numerator.Mul(numerator, activityPowerOfTen(scale))

	var denominator = new(big.Int).Mul(dividendDenominator, new(big.Int).Abs(divisorNumerator))
	var quotient = new(big.Int)
	var remainder = new(big.Int)
	quotient.QuoRem(numerator, denominator, remainder)

	var doubledRemainder = new(big.Int).Lsh(new(big.Int).Set(remainder), 1)
	if doubledRemainder.Cmp(denominator) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if signNegative && quotient.Sign() != 0 {
		quotient.Neg(quotient)
	}

	return quotient
}

// finiteActivityDecimalFraction converts one finite decimal into an exact fraction.
// Authored by: OpenCode
func finiteActivityDecimalFraction(value apd.Decimal) (*big.Int, *big.Int) {
	var numerator = new(big.Int).Set(value.Coeff.MathBigInt())
	if value.Negative {
		numerator.Neg(numerator)
	}
	if value.Exponent >= 0 {
		numerator.Mul(numerator, activityPowerOfTen(value.Exponent))
		return numerator, big.NewInt(1)
	}

	return numerator, activityPowerOfTen(-value.Exponent)
}

// activityPowerOfTen returns 10 raised to one non-negative exponent.
// Authored by: OpenCode
func activityPowerOfTen(exponent int32) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponent)), nil)
}

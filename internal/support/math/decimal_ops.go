package math

import (
	"fmt"

	"github.com/cockroachdb/apd/v3"
)

// ApplyBinaryOperation validates two finite decimal operands and then applies
// one injected exact-decimal operation.
//
// Example:
//
//	sum, err := math.ApplyBinaryOperation(
//		left,
//		right,
//		"left calculation decimal",
//		"right calculation decimal",
//		"add calculation decimals",
//		func(result *apd.Decimal, left *apd.Decimal, right *apd.Decimal) (apd.Condition, error) {
//			return apd.BaseContext.Add(result, left, right)
//		},
//	)
//	if err != nil {
//		panic(err)
//	}
//	_ = sum
//
// Authored by: OpenCode
func ApplyBinaryOperation(
	left apd.Decimal,
	right apd.Decimal,
	leftLabel string,
	rightLabel string,
	errorPrefix string,
	operation func(result *apd.Decimal, left *apd.Decimal, right *apd.Decimal) (apd.Condition, error),
) (apd.Decimal, error) {
	if operation == nil {
		return apd.Decimal{}, fmt.Errorf("decimal operation is required")
	}
	if err := RequireFinite(left, leftLabel); err != nil {
		return apd.Decimal{}, err
	}
	if err := RequireFinite(right, rightLabel); err != nil {
		return apd.Decimal{}, err
	}

	var result apd.Decimal
	if _, err := operation(&result, &left, &right); err != nil {
		return apd.Decimal{}, fmt.Errorf("%s: %w", errorPrefix, err)
	}

	return result, nil
}

// Compare validates two finite decimal operands and returns their ordering.
//
// Example:
//
//	comparison, err := math.Compare(left, right, "left report decimal", "right report decimal")
//	if err != nil {
//		panic(err)
//	}
//	_ = comparison
//
// Authored by: OpenCode
func Compare(left apd.Decimal, right apd.Decimal, leftLabel string, rightLabel string) (int, error) {
	if err := RequireFinite(left, leftLabel); err != nil {
		return 0, err
	}
	if err := RequireFinite(right, rightLabel); err != nil {
		return 0, err
	}

	return left.Cmp(&right), nil
}

// IsZero validates one finite decimal operand and reports whether it is
// numerically zero.
//
// Example:
//
//	isZero, err := math.IsZero(value, "report decimal")
//	if err != nil {
//		panic(err)
//	}
//	_ = isZero
//
// Authored by: OpenCode
func IsZero(value apd.Decimal, label string) (bool, error) {
	if err := RequireFinite(value, label); err != nil {
		return false, err
	}

	return value.Cmp(&apd.Decimal{}) == 0, nil
}

// Minimum returns the smaller of two exact decimal values.
//
// Example:
//
//	minimum := math.Minimum(left, right)
//	_ = minimum
//
// Authored by: OpenCode
func Minimum(left apd.Decimal, right apd.Decimal) apd.Decimal {
	if left.Cmp(&right) <= 0 {
		return Clone(left)
	}

	return Clone(right)
}

// Zero returns one finite zero decimal value.
//
// Example:
//
//	zero := math.Zero()
//	_ = zero
//
// Authored by: OpenCode
func Zero() apd.Decimal {
	return apd.Decimal{}
}

// Clone returns a defensive copy of one exact decimal value.
//
// Example:
//
//	copy := math.Clone(value)
//	_ = copy
//
// Authored by: OpenCode
func Clone(value apd.Decimal) apd.Decimal {
	return value
}

// AllocateProportional allocates one amount proportionally across a partial
// quantity using injected multiply and divide helpers.
//
// Example:
//
//	allocated, err := math.AllocateProportional(
//		totalBasis,
//		totalQuantity,
//		matchedQuantity,
//		"total basis",
//		"total quantity",
//		"matched quantity",
//		"allocate basis proportionally",
//		multiply,
//		divide,
//	)
//	if err != nil {
//		panic(err)
//	}
//	_ = allocated
//
// Authored by: OpenCode
func AllocateProportional(
	totalAmount apd.Decimal,
	totalQuantity apd.Decimal,
	portionQuantity apd.Decimal,
	totalAmountLabel string,
	totalQuantityLabel string,
	portionQuantityLabel string,
	errorPrefix string,
	multiply func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error),
	divide func(dividend apd.Decimal, divisor apd.Decimal) (apd.Decimal, error),
) (apd.Decimal, error) {
	if multiply == nil {
		return apd.Decimal{}, fmt.Errorf("decimal multiplication helper is required")
	}
	if divide == nil {
		return apd.Decimal{}, fmt.Errorf("decimal division helper is required")
	}
	if err := RequireFinite(totalAmount, totalAmountLabel); err != nil {
		return apd.Decimal{}, err
	}
	if totalAmount.Sign() < 0 {
		return apd.Decimal{}, fmt.Errorf("%s must not be negative", totalAmountLabel)
	}
	if err := RequireFinite(totalQuantity, totalQuantityLabel); err != nil {
		return apd.Decimal{}, err
	}
	if totalQuantity.Sign() <= 0 {
		return apd.Decimal{}, fmt.Errorf("%s must be greater than zero", totalQuantityLabel)
	}
	if err := RequireFinite(portionQuantity, portionQuantityLabel); err != nil {
		return apd.Decimal{}, err
	}
	if portionQuantity.Sign() <= 0 {
		return apd.Decimal{}, fmt.Errorf("%s must be greater than zero", portionQuantityLabel)
	}
	if portionQuantity.Cmp(&totalQuantity) > 0 {
		return apd.Decimal{}, fmt.Errorf("%s exceeds %s", portionQuantityLabel, totalQuantityLabel)
	}
	if portionQuantity.Cmp(&totalQuantity) == 0 {
		return Clone(totalAmount), nil
	}

	var numerator, err = multiply(totalAmount, portionQuantity)
	if err != nil {
		return apd.Decimal{}, err
	}

	var quotient apd.Decimal
	quotient, err = divide(numerator, totalQuantity)
	if err != nil {
		return apd.Decimal{}, fmt.Errorf("%s: %w", errorPrefix, err)
	}

	return quotient, nil
}

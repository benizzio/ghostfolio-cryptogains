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

// Add validates two finite decimal operands and returns their exact sum.
//
// Example:
//
//	sum, err := math.Add(left, right, "left calculation decimal", "right calculation decimal", "add calculation decimals")
//	if err != nil {
//		panic(err)
//	}
//	_ = sum
//
// Authored by: OpenCode
func Add(left apd.Decimal, right apd.Decimal, leftLabel string, rightLabel string, errorPrefix string) (apd.Decimal, error) {
	return ApplyBinaryOperation(left, right, leftLabel, rightLabel, errorPrefix, func(result *apd.Decimal, left *apd.Decimal, right *apd.Decimal) (apd.Condition, error) {
		return apd.BaseContext.Add(result, left, right)
	})
}

// Subtract validates two finite decimal operands and returns their exact
// difference.
//
// Example:
//
//	difference, err := math.Subtract(left, right, "left calculation decimal", "right calculation decimal", "subtract calculation decimals")
//	if err != nil {
//		panic(err)
//	}
//	_ = difference
//
// Authored by: OpenCode
func Subtract(left apd.Decimal, right apd.Decimal, leftLabel string, rightLabel string, errorPrefix string) (apd.Decimal, error) {
	return ApplyBinaryOperation(left, right, leftLabel, rightLabel, errorPrefix, func(result *apd.Decimal, left *apd.Decimal, right *apd.Decimal) (apd.Condition, error) {
		return apd.BaseContext.Sub(result, left, right)
	})
}

// Multiply validates two finite decimal operands and returns their exact
// product.
//
// Example:
//
//	product, err := math.Multiply(left, right, "left report decimal", "right report decimal", "multiply report decimals")
//	if err != nil {
//		panic(err)
//	}
//	_ = product
//
// Authored by: OpenCode
func Multiply(left apd.Decimal, right apd.Decimal, leftLabel string, rightLabel string, errorPrefix string) (apd.Decimal, error) {
	return ApplyBinaryOperation(left, right, leftLabel, rightLabel, errorPrefix, func(result *apd.Decimal, left *apd.Decimal, right *apd.Decimal) (apd.Condition, error) {
		return apd.BaseContext.Mul(result, left, right)
	})
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

// RequirePositive rejects non-finite or non-positive decimal values.
//
// Example:
//
//	err := math.RequirePositive(quantity, "disposal quantity")
//	if err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func RequirePositive(value apd.Decimal, label string) error {
	if err := RequireFinite(value, label); err != nil {
		return err
	}
	if value.Sign() <= 0 {
		return fmt.Errorf("%s must be greater than zero", label)
	}

	return nil
}

// RequireNonNegative rejects non-finite or negative decimal values.
//
// Example:
//
//	err := math.RequireNonNegative(basis, "remaining basis")
//	if err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func RequireNonNegative(value apd.Decimal, label string) error {
	if err := RequireFinite(value, label); err != nil {
		return err
	}
	if value.Sign() < 0 {
		return fmt.Errorf("%s must not be negative", label)
	}

	return nil
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

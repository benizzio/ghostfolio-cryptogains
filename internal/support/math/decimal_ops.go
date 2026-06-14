package math

import (
	"fmt"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// addOperation applies exact decimal addition for shared binary operation
// validation.
// Authored by: OpenCode
func addOperation(result *apd.Decimal, left *apd.Decimal, right *apd.Decimal) (apd.Condition, error) {
	return apd.BaseContext.Add(result, left, right)
}

// subtractOperation applies exact decimal subtraction for shared binary
// operation validation.
// Authored by: OpenCode
func subtractOperation(result *apd.Decimal, left *apd.Decimal, right *apd.Decimal) (apd.Condition, error) {
	return apd.BaseContext.Sub(result, left, right)
}

// multiplyOperation applies exact decimal multiplication for shared binary
// operation validation.
// Authored by: OpenCode
func multiplyOperation(result *apd.Decimal, left *apd.Decimal, right *apd.Decimal) (apd.Condition, error) {
	return apd.BaseContext.Mul(result, left, right)
}

// ApplyBinaryOperation validates two finite decimal operands and then applies
// one injected exact-decimal operation.
//
// Example:
//
//	sum, err := math.ApplyBinaryOperation(left, right, func(result *apd.Decimal, left *apd.Decimal, right *apd.Decimal) (apd.Condition, error) {
//		return apd.BaseContext.Add(result, left, right)
//	})
//	if err != nil {
//		panic(err)
//	}
//	_ = sum
//
// Authored by: OpenCode
func ApplyBinaryOperation(
	left apd.Decimal,
	right apd.Decimal,
	operation func(result *apd.Decimal, left *apd.Decimal, right *apd.Decimal) (apd.Condition, error),
) (apd.Decimal, error) {
	if operation == nil {
		return apd.Decimal{}, fmt.Errorf("decimal operation is required")
	}
	if err := RequireFinite(left); err != nil {
		return apd.Decimal{}, fmt.Errorf("left decimal operand is invalid: %w", err)
	}
	if err := RequireFinite(right); err != nil {
		return apd.Decimal{}, fmt.Errorf("right decimal operand is invalid: %w", err)
	}

	var result apd.Decimal
	if _, err := operation(&result, &left, &right); err != nil {
		return apd.Decimal{}, fmt.Errorf("decimal operation failed: %w", err)
	}

	return result, nil
}

// Add validates two finite decimal operands and returns their exact sum.
//
// Example:
//
//	sum, err := math.Add(left, right)
//	if err != nil {
//		panic(err)
//	}
//	_ = sum
//
// Authored by: OpenCode
func Add(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
	return ApplyBinaryOperation(left, right, addOperation)
}

// Subtract validates two finite decimal operands and returns their exact
// difference.
//
// Example:
//
//	difference, err := math.Subtract(left, right)
//	if err != nil {
//		panic(err)
//	}
//	_ = difference
//
// Authored by: OpenCode
func Subtract(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
	return ApplyBinaryOperation(left, right, subtractOperation)
}

// Multiply validates two finite decimal operands and returns their exact
// product.
//
// Example:
//
//	product, err := math.Multiply(left, right)
//	if err != nil {
//		panic(err)
//	}
//	_ = product
//
// Authored by: OpenCode
func Multiply(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
	return ApplyBinaryOperation(left, right, multiplyOperation)
}

// RequireFinite rejects non-finite decimal values before shared arithmetic or
// comparison helpers are used.
//
// Example:
//
//	value, _, err := decimalsupport.ParseString("10.5")
//	if err != nil {
//		panic(err)
//	}
//	err = math.RequireFinite(value)
//	if err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func RequireFinite(value apd.Decimal) error {
	if _, err := decimalsupport.CanonicalString(value); err != nil {
		return fmt.Errorf("decimal value must be finite: %w", err)
	}

	return nil
}

// DivideFiniteRoundHalfUp divides one finite decimal by another and rounds the
// quotient half up to the active process decimal policy.
//
// Example:
//
//	dividend, _, err := decimalsupport.ParseString("1")
//	if err != nil {
//		panic(err)
//	}
//	divisor, _, err := decimalsupport.ParseString("3")
//	if err != nil {
//		panic(err)
//	}
//	quotient, err := math.DivideFiniteRoundHalfUp(dividend, divisor)
//	if err != nil {
//		panic(err)
//	}
//	_ = quotient
//
// Callers that need package-specific validation messages should validate their
// operands before calling this helper.
// Authored by: OpenCode
func DivideFiniteRoundHalfUp(dividend apd.Decimal, divisor apd.Decimal) (apd.Decimal, error) {
	return DivideFiniteRoundHalfUpWithPolicy(dividend, divisor, ActiveDecimalPolicy())
}

// DivideFiniteRoundHalfUpWithPolicy divides one finite decimal by another and
// rounds the quotient half up to an explicitly selected supported decimal
// policy. Use this helper only at boundaries that have already selected and
// validated a policy.
//
// Example:
//
//	policy, err := math.ParseDecimalPolicy("scale=16,rounding=half_up")
//	if err != nil {
//		panic(err)
//	}
//	quotient, err := math.DivideFiniteRoundHalfUpWithPolicy(dividend, divisor, policy)
//	if err != nil {
//		panic(err)
//	}
//	_ = quotient
//
// Authored by: OpenCode
func DivideFiniteRoundHalfUpWithPolicy(dividend apd.Decimal, divisor apd.Decimal, policy DecimalPolicy) (apd.Decimal, error) {
	if err := validateDecimalPolicy(policy); err != nil {
		return apd.Decimal{}, fmt.Errorf("division decimal policy is invalid: %w", err)
	}
	if err := RequireFinite(dividend); err != nil {
		return apd.Decimal{}, fmt.Errorf("division dividend is invalid: %w", err)
	}
	if err := RequireFinite(divisor); err != nil {
		return apd.Decimal{}, fmt.Errorf("division divisor is invalid: %w", err)
	}
	if divisor.Sign() == 0 {
		return apd.Decimal{}, fmt.Errorf("division requires a non-zero divisor")
	}

	var scaledQuotient = scaledRoundedQuotient(dividend, divisor, policy.scale)
	return scaledCoefficientDecimal(scaledQuotient, policy.scale), nil
}

// Compare validates two finite decimal operands and returns their ordering.
//
// Example:
//
//	comparison, err := math.Compare(left, right)
//	if err != nil {
//		panic(err)
//	}
//	_ = comparison
//
// Authored by: OpenCode
func Compare(left apd.Decimal, right apd.Decimal) (int, error) {
	if err := RequireFinite(left); err != nil {
		return 0, fmt.Errorf("left decimal operand is invalid: %w", err)
	}
	if err := RequireFinite(right); err != nil {
		return 0, fmt.Errorf("right decimal operand is invalid: %w", err)
	}

	return left.Cmp(&right), nil
}

// IsZero validates one finite decimal operand and reports whether it is
// numerically zero.
//
// Example:
//
//	isZero, err := math.IsZero(value)
//	if err != nil {
//		panic(err)
//	}
//	_ = isZero
//
// Authored by: OpenCode
func IsZero(value apd.Decimal) (bool, error) {
	if err := RequireFinite(value); err != nil {
		return false, fmt.Errorf("decimal operand is invalid: %w", err)
	}

	return value.Cmp(&apd.Decimal{}) == 0, nil
}

// RequirePositive rejects non-finite or non-positive decimal values.
//
// Example:
//
//	err := math.RequirePositive(quantity)
//	if err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func RequirePositive(value apd.Decimal) error {
	if err := RequireFinite(value); err != nil {
		return fmt.Errorf("decimal operand is invalid: %w", err)
	}
	if value.Sign() <= 0 {
		return fmt.Errorf("decimal operand must be greater than zero")
	}

	return nil
}

// RequireNonNegative rejects non-finite or negative decimal values.
//
// Example:
//
//	err := math.RequireNonNegative(basis)
//	if err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func RequireNonNegative(value apd.Decimal) error {
	if err := RequireFinite(value); err != nil {
		return fmt.Errorf("decimal operand is invalid: %w", err)
	}
	if value.Sign() < 0 {
		return fmt.Errorf("decimal operand must not be negative")
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
	multiply func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error),
	divide func(dividend apd.Decimal, divisor apd.Decimal) (apd.Decimal, error),
) (apd.Decimal, error) {
	if multiply == nil {
		return apd.Decimal{}, fmt.Errorf("decimal multiplication helper is required")
	}
	if divide == nil {
		return apd.Decimal{}, fmt.Errorf("decimal division helper is required")
	}
	if err := RequireFinite(totalAmount); err != nil {
		return apd.Decimal{}, fmt.Errorf("total amount is invalid: %w", err)
	}
	if totalAmount.Sign() < 0 {
		return apd.Decimal{}, fmt.Errorf("total amount must not be negative")
	}
	if err := RequireFinite(totalQuantity); err != nil {
		return apd.Decimal{}, fmt.Errorf("total quantity is invalid: %w", err)
	}
	if totalQuantity.Sign() <= 0 {
		return apd.Decimal{}, fmt.Errorf("total quantity must be greater than zero")
	}
	if err := RequireFinite(portionQuantity); err != nil {
		return apd.Decimal{}, fmt.Errorf("portion quantity is invalid: %w", err)
	}
	if portionQuantity.Sign() <= 0 {
		return apd.Decimal{}, fmt.Errorf("portion quantity must be greater than zero")
	}
	if portionQuantity.Cmp(&totalQuantity) > 0 {
		return apd.Decimal{}, fmt.Errorf("portion quantity exceeds total quantity")
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
		return apd.Decimal{}, fmt.Errorf("allocate proportional amount: %w", err)
	}

	return quotient, nil
}

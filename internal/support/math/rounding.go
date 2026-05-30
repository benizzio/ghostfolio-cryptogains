// Package math centralizes reusable exact-decimal arithmetic helpers shared by
// validation and report-calculation packages.
// Authored by: OpenCode
package math

import (
	"fmt"
	"math/big"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// InternalCalculationScale is the shared fixed scale for internal financial
// divisions that use round-half-up handling before report rendering.
// Authored by: OpenCode
const InternalCalculationScale int32 = 16

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
// quotient half up to the shared internal fixed scale.
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
	if err := RequireFinite(dividend); err != nil {
		return apd.Decimal{}, fmt.Errorf("division dividend is invalid: %w", err)
	}
	if err := RequireFinite(divisor); err != nil {
		return apd.Decimal{}, fmt.Errorf("division divisor is invalid: %w", err)
	}
	if divisor.Sign() == 0 {
		return apd.Decimal{}, fmt.Errorf("division requires a non-zero divisor")
	}

	var scaledQuotient = scaledRoundedQuotient(dividend, divisor, InternalCalculationScale)
	return scaledCoefficientDecimal(scaledQuotient, InternalCalculationScale), nil
}

// scaledRoundedQuotient computes one fixed-scale quotient rounded half up.
// Authored by: OpenCode
func scaledRoundedQuotient(dividend apd.Decimal, divisor apd.Decimal, scale int32) *big.Int {
	var dividendNumerator, dividendDenominator = finiteDecimalFraction(dividend)
	var divisorNumerator, divisorDenominator = finiteDecimalFraction(divisor)

	var signNegative = (dividendNumerator.Sign() < 0) != (divisorNumerator.Sign() < 0)
	var numerator = new(big.Int).Mul(new(big.Int).Abs(dividendNumerator), divisorDenominator)
	numerator.Mul(numerator, powerOfTen(scale))

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

// finiteDecimalFraction converts one finite decimal into an exact fraction.
// Authored by: OpenCode
func finiteDecimalFraction(value apd.Decimal) (*big.Int, *big.Int) {
	var numerator = new(big.Int).Set(value.Coeff.MathBigInt())
	if value.Negative {
		numerator.Neg(numerator)
	}
	if value.Exponent >= 0 {
		numerator.Mul(numerator, powerOfTen(value.Exponent))
		return numerator, big.NewInt(1)
	}

	return numerator, powerOfTen(-value.Exponent)
}

// scaledCoefficientDecimal converts one signed scaled integer into an apd
// decimal value.
// Authored by: OpenCode
func scaledCoefficientDecimal(value *big.Int, scale int32) apd.Decimal {
	var magnitude = new(big.Int).Set(value)
	var negative = magnitude.Sign() < 0
	if negative {
		magnitude.Abs(magnitude)
	}

	var coefficient apd.BigInt
	coefficient.SetMathBigInt(magnitude)

	return apd.Decimal{
		Form:     apd.Finite,
		Negative: negative,
		Exponent: -scale,
		Coeff:    coefficient,
	}
}

// powerOfTen returns 10 raised to one non-negative exponent.
// Authored by: OpenCode
func powerOfTen(exponent int32) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponent)), nil)
}

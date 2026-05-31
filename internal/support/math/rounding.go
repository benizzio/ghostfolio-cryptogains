// Package math centralizes reusable exact-decimal arithmetic helpers shared by
// validation and report-calculation packages.
// Authored by: OpenCode
package math

import (
	"math/big"

	"github.com/cockroachdb/apd/v3"
)

// InternalCalculationScale is the shared fixed scale for internal financial
// divisions that use round-half-up handling before report rendering.
// Authored by: OpenCode
const InternalCalculationScale int32 = 16

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

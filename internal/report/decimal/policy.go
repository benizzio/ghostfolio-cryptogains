// Package decimal defines report-local decimal helpers for the shared
// 16-decimal internal calculation policy.
// Authored by: OpenCode
package decimal

import (
	"fmt"
	"math/big"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

const reportCalculationScale int32 = 16

// DivideRoundHalfUp divides one finite decimal by another using the shared
// 16-decimal internal report-calculation precision.
// Authored by: OpenCode
func DivideRoundHalfUp(dividend apd.Decimal, divisor apd.Decimal) (apd.Decimal, error) {
	if err := RequireFinite(dividend, "report dividend"); err != nil {
		return apd.Decimal{}, err
	}
	if err := RequireFinite(divisor, "report divisor"); err != nil {
		return apd.Decimal{}, err
	}
	if divisor.Sign() == 0 {
		return apd.Decimal{}, fmt.Errorf("report division requires a non-zero divisor")
	}

	var scaledQuotient = scaledRoundedQuotient(dividend, divisor, reportCalculationScale)
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
		Exponent: -reportCalculationScale,
		Coeff:    coefficient,
	}, nil
}

// RequireFinite rejects non-finite decimal inputs before report-local
// arithmetic or comparisons.
// Authored by: OpenCode
func RequireFinite(value apd.Decimal, label string) error {
	if _, err := decimalsupport.CanonicalString(value); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}

	return nil
}

// scaledRoundedQuotient computes one quotient rounded half up to the requested
// fixed number of decimal places.
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

// finiteDecimalFraction converts one finite decimal into an exact numerator and
// denominator pair.
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

// powerOfTen returns 10 raised to one non-negative exponent.
// Authored by: OpenCode
func powerOfTen(exponent int32) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponent)), nil)
}

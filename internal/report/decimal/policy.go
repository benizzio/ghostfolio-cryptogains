// Package decimal defines report-local decimal helpers for the shared
// 16-decimal internal calculation policy.
// Authored by: OpenCode
package decimal

import (
	"fmt"

	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	"github.com/cockroachdb/apd/v3"
)

const reportCalculationScale int32 = 16

// DivideRoundHalfUp divides one finite decimal by another using the shared
// 16-decimal internal report-calculation precision.
// Authored by: OpenCode
func DivideRoundHalfUp(dividend apd.Decimal, divisor apd.Decimal) (apd.Decimal, error) {
	if err := supportmath.RequireFinite(dividend, "report dividend"); err != nil {
		return apd.Decimal{}, err
	}
	if err := supportmath.RequireFinite(divisor, "report divisor"); err != nil {
		return apd.Decimal{}, err
	}
	if divisor.Sign() == 0 {
		return apd.Decimal{}, fmt.Errorf("report division requires a non-zero divisor")
	}

	return supportmath.DivideFiniteRoundHalfUp(dividend, divisor, reportCalculationScale)
}

// RequireFinite rejects non-finite decimal inputs before report-local
// arithmetic or comparisons.
// Authored by: OpenCode
func RequireFinite(value apd.Decimal, label string) error {
	return supportmath.RequireFinite(value, label)
}

package calculate

import (
	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	"github.com/cockroachdb/apd/v3"
)

// Test seams keep report-local multiply wrapper branches directly coverable.
// Authored by: OpenCode
var reportMultiplyOperation = func(product *apd.Decimal, left *apd.Decimal, right *apd.Decimal) (apd.Condition, error) {
	return apd.BaseContext.Mul(product, left, right)
}

// multiplyDecimal preserves exact-decimal precision for report-local
// multiplication steps.
// Authored by: OpenCode
func multiplyDecimal(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
	return supportmath.ApplyBinaryOperation(left, right, "left report decimal", "right report decimal", "multiply report decimals", reportMultiplyOperation)
}

// decimalIsZero reports whether one finite report decimal is numerically zero.
// Authored by: OpenCode
func decimalIsZero(value apd.Decimal) (bool, error) {
	return supportmath.IsZero(value, "report decimal")
}

// compareDecimals orders two finite report decimals for report-local decisions.
// Authored by: OpenCode
func compareDecimals(left apd.Decimal, right apd.Decimal) (int, error) {
	return supportmath.Compare(left, right, "left report decimal", "right report decimal")
}

// requireFiniteDecimal rejects non-finite decimal inputs before report-local
// arithmetic or comparisons.
// Authored by: OpenCode
func requireFiniteDecimal(value apd.Decimal, label string) error {
	return supportmath.RequireFinite(value, label)
}

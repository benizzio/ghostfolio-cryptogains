package calculate

import (
	"fmt"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// reportDecimalZero provides one shared finite zero for report-local decimal comparisons.
// Exact division and canonical formatting intentionally reuse decimalsupport
// package helpers directly.
// Authored by: OpenCode
var reportDecimalZero apd.Decimal

// multiplyDecimal preserves exact-decimal precision for report-local
// multiplication steps.
// Authored by: OpenCode
func multiplyDecimal(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
	if err := requireFiniteDecimal(left, "left report decimal"); err != nil {
		return apd.Decimal{}, err
	}
	if err := requireFiniteDecimal(right, "right report decimal"); err != nil {
		return apd.Decimal{}, err
	}

	var product apd.Decimal
	var err error
	_, err = apd.BaseContext.Mul(&product, &left, &right)
	if err != nil {
		return apd.Decimal{}, fmt.Errorf("multiply report decimals: %w", err)
	}

	return product, nil
}

// decimalIsZero reports whether one finite report decimal is numerically zero.
// Authored by: OpenCode
func decimalIsZero(value apd.Decimal) (bool, error) {
	if err := requireFiniteDecimal(value, "report decimal"); err != nil {
		return false, err
	}

	return value.Cmp(&reportDecimalZero) == 0, nil
}

// compareDecimals orders two finite report decimals for report-local decisions.
// Authored by: OpenCode
func compareDecimals(left apd.Decimal, right apd.Decimal) (int, error) {
	if err := requireFiniteDecimal(left, "left report decimal"); err != nil {
		return 0, err
	}
	if err := requireFiniteDecimal(right, "right report decimal"); err != nil {
		return 0, err
	}

	return left.Cmp(&right), nil
}

// requireFiniteDecimal rejects non-finite decimal inputs before report-local
// arithmetic or comparisons.
// Authored by: OpenCode
func requireFiniteDecimal(value apd.Decimal, label string) error {
	if _, err := decimalsupport.CanonicalString(value); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}

	return nil
}

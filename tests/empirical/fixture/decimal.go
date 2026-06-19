// Package fixture provides shared parsing and validation helpers for empirical
// dataset and oracle fixture code.
// Authored by: OpenCode
package fixture

import (
	"fmt"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// ParseDecimalString parses one empirical fixture decimal field from its stored
// canonical string form and returns both the parsed value and the verified
// canonical representation.
//
// Example:
//
//	value, canonical, err := fixture.ParseDecimalString("10.5")
//	if err != nil {
//		panic(err)
//	}
//	_, _ = value, canonical
//
// ParseDecimalString rejects empty text, invalid numbers, non-finite values,
// and any representation that is not already the canonical fixed-point string
// required by the empirical dataset and oracle fixture contracts.
// Authored by: OpenCode
func ParseDecimalString(raw string) (apd.Decimal, string, error) {
	var value, canonical, err = decimalsupport.ParseCanonicalString(raw)
	if err != nil {
		return apd.Decimal{}, "", fmt.Errorf("parse fixture decimal string: %w", err)
	}

	return value, canonical, nil
}

// CanonicalDecimalString returns one finite decimal value in the canonical
// string form used by the empirical dataset and oracle fixtures.
//
// Example:
//
//	value, _, err := fixture.ParseDecimalString("10.5")
//	if err != nil {
//		panic(err)
//	}
//	canonical, err := fixture.CanonicalDecimalString(value)
//	if err != nil {
//		panic(err)
//	}
//	_ = canonical
//
// Authored by: OpenCode
func CanonicalDecimalString(value apd.Decimal) (string, error) {
	var canonical, err = decimalsupport.CanonicalString(value)
	if err != nil {
		return "", fmt.Errorf("canonicalize fixture decimal string: %w", err)
	}

	return canonical, nil
}

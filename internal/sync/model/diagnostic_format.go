// Package model defines normalized sync data structures shared across sync,
// snapshot, and runtime packages.
// Authored by: OpenCode
package model

import (
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// canonicalDiagnosticDecimal returns a stable decimal string for diagnostic
// context.
// Authored by: OpenCode
func canonicalDiagnosticDecimal(value apd.Decimal) string {
	var canonical, err = decimalsupport.CanonicalString(value)
	if err == nil {
		return canonical
	}

	return value.String()
}

// canonicalDiagnosticDecimalPointer returns a stable optional decimal string
// for diagnostic context.
// Authored by: OpenCode
func canonicalDiagnosticDecimalPointer(value *apd.Decimal) string {
	if value == nil {
		return ""
	}

	var canonical, err = decimalsupport.CanonicalStringPointer(value)
	if err == nil {
		return canonical
	}

	return value.String()
}

// diagnosticStringPointer converts one optional string field into a nullable
// JSON-ready pointer.
// Authored by: OpenCode
func diagnosticStringPointer(value string) *string {
	if value == "" {
		return nil
	}

	var copied = value
	return &copied
}

// diagnosticStringValue converts one nullable decoded string back into the
// runtime diagnostic record representation.
// Authored by: OpenCode
func diagnosticStringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

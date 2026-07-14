// Package math verifies shared exact-decimal operation helpers.
// Authored by: OpenCode
package math

import (
	"strings"
	"testing"

	"github.com/cockroachdb/apd/v3"
)

// TestIsZeroPointerVerifiesOptionalFiniteDecimals checks nil, zero, and
// non-finite optional decimal handling.
// Authored by: OpenCode
func TestIsZeroPointerVerifiesOptionalFiniteDecimals(t *testing.T) {
	t.Parallel()

	var isZero, err = IsZeroPointer(nil)
	if err != nil || isZero {
		t.Fatalf("expected nil decimal pointer not to be zero, got %t err=%v", isZero, err)
	}

	var zero = apd.New(0, 0)
	isZero, err = IsZeroPointer(zero)
	if err != nil || !isZero {
		t.Fatalf("expected zero decimal pointer to be zero, got %t err=%v", isZero, err)
	}

	var invalid apd.Decimal
	invalid.Form = apd.NaNSignaling
	if _, err = IsZeroPointer(&invalid); err == nil || !strings.Contains(err.Error(), "decimal operand") {
		t.Fatalf("expected invalid decimal pointer to fail, got %v", err)
	}
}

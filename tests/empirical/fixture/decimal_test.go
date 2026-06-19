package fixture

import (
	"strings"
	"testing"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// TestParseDecimalStringAcceptsCanonicalFixedPointText verifies canonical
// fixture decimals round-trip without mutation.
// Authored by: OpenCode
func TestParseDecimalStringAcceptsCanonicalFixedPointText(t *testing.T) {
	t.Parallel()

	var testCases = []struct {
		name string
		raw  string
	}{
		{name: "zero", raw: "0"},
		{name: "positive_fraction", raw: "10.5"},
		{name: "negative_fraction", raw: "-1.25"},
	}

	for _, testCase := range testCases {
		var testCase = testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var value, canonical, err = ParseDecimalString(testCase.raw)
			if err != nil {
				t.Fatalf("parse decimal string %q: %v", testCase.raw, err)
			}
			if canonical != testCase.raw {
				t.Fatalf("unexpected canonical value for %q: got %q want %q", testCase.raw, canonical, testCase.raw)
			}

			canonical, err = CanonicalDecimalString(value)
			if err != nil {
				t.Fatalf("canonicalize decimal string %q: %v", testCase.raw, err)
			}
			if canonical != testCase.raw {
				t.Fatalf("unexpected round-trip canonical value for %q: got %q want %q", testCase.raw, canonical, testCase.raw)
			}
		})
	}
}

// TestParseDecimalStringRejectsEmptyText verifies fixture decimal fields cannot
// be empty or whitespace-only.
// Authored by: OpenCode
func TestParseDecimalStringRejectsEmptyText(t *testing.T) {
	t.Parallel()

	var testCases = []string{"", "   "}

	for _, raw := range testCases {
		var raw = raw

		t.Run(raw, func(t *testing.T) {
			t.Parallel()

			_, _, err := ParseDecimalString(raw)
			if err == nil {
				t.Fatalf("expected empty decimal string %q to fail", raw)
			}
		})
	}
}

// TestParseDecimalStringRejectsInvalidText verifies unreadable decimal strings
// are rejected.
// Authored by: OpenCode
func TestParseDecimalStringRejectsInvalidText(t *testing.T) {
	t.Parallel()

	var testCases = []string{"not-a-number", "1_000"}

	for _, raw := range testCases {
		var raw = raw

		t.Run(raw, func(t *testing.T) {
			t.Parallel()

			_, _, err := ParseDecimalString(raw)
			if err == nil {
				t.Fatalf("expected invalid decimal string %q to fail", raw)
			}
		})
	}
}

// TestParseDecimalStringRejectsNonFiniteText verifies fixture decimal fields
// cannot use non-finite numeric text.
// Authored by: OpenCode
func TestParseDecimalStringRejectsNonFiniteText(t *testing.T) {
	t.Parallel()

	var testCases = []string{"Infinity", "-Infinity", "NaN"}

	for _, raw := range testCases {
		var raw = raw

		t.Run(raw, func(t *testing.T) {
			t.Parallel()

			_, _, err := ParseDecimalString(raw)
			if err == nil {
				t.Fatalf("expected non-finite decimal string %q to fail", raw)
			}
		})
	}
}

// TestParseDecimalStringRejectsNonCanonicalText verifies persisted fixture
// decimals must already use canonical fixed-point text.
// Authored by: OpenCode
func TestParseDecimalStringRejectsNonCanonicalText(t *testing.T) {
	t.Parallel()

	var testCases = []struct {
		name              string
		raw               string
		expectedCanonical string
	}{
		{name: "leading_zeros", raw: "001.2300", expectedCanonical: "1.23"},
		{name: "trailing_fractional_zero", raw: "0.0", expectedCanonical: "0"},
		{name: "surrounding_whitespace", raw: " 42 ", expectedCanonical: "42"},
		{name: "positive_sign", raw: "+1", expectedCanonical: "1"},
		{name: "negative_zero", raw: "-0", expectedCanonical: "0"},
		{name: "exponent_notation", raw: "1e3", expectedCanonical: "1000"},
		{name: "missing_leading_zero", raw: ".5", expectedCanonical: "0.5"},
		{name: "trailing_decimal_point", raw: "5.", expectedCanonical: "5"},
	}

	for _, testCase := range testCases {
		var testCase = testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := ParseDecimalString(testCase.raw)
			if err == nil {
				t.Fatalf("expected non-canonical decimal string %q to fail", testCase.raw)
			}
			if !strings.Contains(err.Error(), "canonical fixed-point representation") {
				t.Fatalf("expected canonicalization error for %q, got %v", testCase.raw, err)
			}
			if !strings.Contains(err.Error(), testCase.expectedCanonical) {
				t.Fatalf("expected error for %q to mention canonical form %q, got %v", testCase.raw, testCase.expectedCanonical, err)
			}
		})
	}
}

// TestCanonicalDecimalStringCanonicalizesFiniteValues verifies the helper emits
// the shared canonical decimal format for valid finite values.
// Authored by: OpenCode
func TestCanonicalDecimalStringCanonicalizesFiniteValues(t *testing.T) {
	t.Parallel()

	var value, _, err = decimalsupport.ParseString("001.2300")
	if err != nil {
		t.Fatalf("parse source decimal: %v", err)
	}

	var canonical string
	canonical, err = CanonicalDecimalString(value)
	if err != nil {
		t.Fatalf("canonicalize finite decimal: %v", err)
	}
	if canonical != "1.23" {
		t.Fatalf("unexpected canonical finite decimal: got %q want %q", canonical, "1.23")
	}
}

// TestCanonicalDecimalStringRejectsNonFiniteValues verifies only finite decimal
// values can be rendered into persisted fixture text.
// Authored by: OpenCode
func TestCanonicalDecimalStringRejectsNonFiniteValues(t *testing.T) {
	t.Parallel()

	var invalid apd.Decimal
	invalid.Form = apd.Infinite

	if _, err := CanonicalDecimalString(invalid); err == nil {
		t.Fatalf("expected non-finite decimal canonicalization to fail")
	}
}

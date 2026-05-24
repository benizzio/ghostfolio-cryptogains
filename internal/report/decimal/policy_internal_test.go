// Package decimal verifies package-local report decimal policy behavior.
// Authored by: OpenCode
package decimal

import (
	"strings"
	"testing"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// TestDivideRoundHalfUp verifies fixed-scale round-half-up division behavior for
// terminating and repeating report decimals.
// Authored by: OpenCode
func TestDivideRoundHalfUp(t *testing.T) {
	t.Parallel()

	var exact, err = DivideRoundHalfUp(mustReportPolicyDecimal(t, "5"), mustReportPolicyDecimal(t, "4"))
	if err != nil {
		t.Fatalf("divide exact report decimals: %v", err)
	}
	assertReportPolicyDecimalString(t, exact, "1.25", "exact quotient")

	var repeating, repeatingErr = DivideRoundHalfUp(mustReportPolicyDecimal(t, "1"), mustReportPolicyDecimal(t, "3"))
	if repeatingErr != nil {
		t.Fatalf("divide repeating report decimals: %v", repeatingErr)
	}
	assertReportPolicyDecimalString(t, repeating, "0.3333333333333333", "repeating quotient")

	var halfUp, halfUpErr = DivideRoundHalfUp(mustReportPolicyDecimal(t, "1"), mustReportPolicyDecimal(t, "6"))
	if halfUpErr != nil {
		t.Fatalf("divide half-up report decimals: %v", halfUpErr)
	}
	assertReportPolicyDecimalString(t, halfUp, "0.1666666666666667", "half-up quotient")

	var negative, negativeErr = DivideRoundHalfUp(mustReportPolicyDecimal(t, "-1"), mustReportPolicyDecimal(t, "6"))
	if negativeErr != nil {
		t.Fatalf("divide negative report decimals: %v", negativeErr)
	}
	assertReportPolicyDecimalString(t, negative, "-0.1666666666666667", "negative half-up quotient")

	var roundedZero, zeroErr = DivideRoundHalfUp(mustReportPolicyDecimal(t, "1"), mustReportPolicyDecimal(t, "100000000000000000"))
	if zeroErr != nil {
		t.Fatalf("divide to rounded zero report decimals: %v", zeroErr)
	}
	assertReportPolicyDecimalString(t, roundedZero, "0", "rounded zero quotient")

	if _, err = DivideRoundHalfUp(mustReportPolicyDecimal(t, "1"), apd.Decimal{}); err == nil || !strings.Contains(err.Error(), "non-zero divisor") {
		t.Fatalf("expected zero divisor failure, got %v", err)
	}

	var invalid apd.Decimal
	invalid.Form = apd.Infinite
	if _, err = DivideRoundHalfUp(invalid, mustReportPolicyDecimal(t, "1")); err == nil || !strings.Contains(err.Error(), "report dividend") {
		t.Fatalf("expected invalid dividend failure, got %v", err)
	}
	if _, err = DivideRoundHalfUp(mustReportPolicyDecimal(t, "1"), invalid); err == nil || !strings.Contains(err.Error(), "report divisor") {
		t.Fatalf("expected invalid divisor failure, got %v", err)
	}
}

// mustReportPolicyDecimal parses one exact decimal fixture for report policy
// tests.
// Authored by: OpenCode
func mustReportPolicyDecimal(t *testing.T, raw string) apd.Decimal {
	t.Helper()

	var value, _, err = decimalsupport.ParseString(raw)
	if err != nil {
		t.Fatalf("parse report policy decimal %q: %v", raw, err)
	}

	return value
}

// assertReportPolicyDecimalString compares one report-policy decimal against its
// canonical string form.
// Authored by: OpenCode
func assertReportPolicyDecimalString(t *testing.T, value apd.Decimal, want string, label string) {
	t.Helper()

	var canonical, err = decimalsupport.CanonicalString(value)
	if err != nil {
		t.Fatalf("canonicalize %s: %v", label, err)
	}
	if canonical != want {
		t.Fatalf("unexpected %s: got %q want %q", label, canonical, want)
	}
}

package calculate

import (
	"errors"
	"strings"
	"testing"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// TestReportDecimalMathReusesSharedExactDivisionAndFormatting verifies that
// report-local multiplication works alongside shared exact division and
// canonical formatting helpers.
// Authored by: OpenCode
func TestReportDecimalMathReusesSharedExactDivisionAndFormatting(t *testing.T) {
	t.Parallel()

	var left = mustReportDecimal(t, "1.25")
	var right = mustReportDecimal(t, "4")

	product, err := multiplyDecimal(left, right)
	if err != nil {
		t.Fatalf("multiply decimals: %v", err)
	}

	canonical, err := decimalsupport.CanonicalString(product)
	if err != nil {
		t.Fatalf("canonicalize product: %v", err)
	}
	if canonical != "5" {
		t.Fatalf("unexpected canonical product: %q", canonical)
	}

	quotient, canonical, err := decimalsupport.DivideExact(product, right)
	if err != nil {
		t.Fatalf("divide exact: %v", err)
	}
	if canonical != "1.25" {
		t.Fatalf("unexpected exact quotient canonical string: %q", canonical)
	}

	comparison, err := compareDecimals(quotient, left)
	if err != nil {
		t.Fatalf("compare quotient and left operand: %v", err)
	}
	if comparison != 0 {
		t.Fatalf("expected exact quotient to equal original left operand, got %d", comparison)
	}
}

// TestReportDecimalMathZeroChecksAndOrdering verifies zero checks and shared
// ordering behavior for finite report decimals.
// Authored by: OpenCode
func TestReportDecimalMathZeroChecksAndOrdering(t *testing.T) {
	t.Parallel()

	zero, err := decimalIsZero(mustReportDecimal(t, "0.000"))
	if err != nil {
		t.Fatalf("zero check: %v", err)
	}
	if !zero {
		t.Fatalf("expected canonical zero to compare as zero")
	}

	zero, err = decimalIsZero(mustReportDecimal(t, "0.01"))
	if err != nil {
		t.Fatalf("non-zero check: %v", err)
	}
	if zero {
		t.Fatalf("expected non-zero decimal not to compare as zero")
	}

	if got, err := compareDecimals(mustReportDecimal(t, "2"), mustReportDecimal(t, "10")); err != nil || got >= 0 {
		t.Fatalf("expected 2 to sort before 10, got %d err=%v", got, err)
	}
	if got, err := compareDecimals(mustReportDecimal(t, "10"), mustReportDecimal(t, "2")); err != nil || got <= 0 {
		t.Fatalf("expected 10 to sort after 2, got %d err=%v", got, err)
	}
}

// TestReportDecimalMathRejectsNonFiniteInputs verifies that report-local math
// helpers reject non-finite decimal values.
// Authored by: OpenCode
func TestReportDecimalMathRejectsNonFiniteInputs(t *testing.T) {
	t.Parallel()

	var invalid apd.Decimal
	invalid.Form = apd.Infinite

	_, err := multiplyDecimal(invalid, mustReportDecimal(t, "1"))
	if err == nil {
		t.Fatalf("expected multiply to reject non-finite decimal")
	}
	if !strings.Contains(err.Error(), "decimal value must be finite") {
		t.Fatalf("unexpected multiply error: %v", err)
	}

	if _, err := decimalIsZero(invalid); err == nil {
		t.Fatalf("expected zero check to reject non-finite decimal")
	}
	if _, err := compareDecimals(mustReportDecimal(t, "1"), invalid); err == nil {
		t.Fatalf("expected comparison to reject non-finite decimal")
	}
	if _, err := compareDecimals(invalid, mustReportDecimal(t, "1")); err == nil {
		t.Fatalf("expected comparison to reject non-finite left decimal")
	}
	if _, err := multiplyDecimal(mustReportDecimal(t, "1"), invalid); err == nil {
		t.Fatalf("expected multiply to reject non-finite right decimal")
	}
}

// TestReportDecimalMathWrapsMultiplyOperationFailure verifies the direct
// multiply wrapper branch around the decimal operation seam.
// Authored by: OpenCode
func TestReportDecimalMathWrapsMultiplyOperationFailure(t *testing.T) {
	t.Parallel()

	var previous = reportMultiplyOperation
	defer func() {
		reportMultiplyOperation = previous
	}()

	reportMultiplyOperation = func(*apd.Decimal, *apd.Decimal, *apd.Decimal) (apd.Condition, error) {
		return 0, errors.New("multiply boom")
	}

	if _, err := multiplyDecimal(mustReportDecimal(t, "2"), mustReportDecimal(t, "3")); err == nil || !strings.Contains(err.Error(), "multiply report decimals") {
		t.Fatalf("expected wrapped multiply failure, got %v", err)
	}
}

// mustReportDecimal parses one exact decimal for report calculation tests.
// Authored by: OpenCode
func mustReportDecimal(t *testing.T, raw string) apd.Decimal {
	t.Helper()

	value, _, err := decimalsupport.ParseString(raw)
	if err != nil {
		t.Fatalf("parse report decimal %q: %v", raw, err)
	}

	return value
}

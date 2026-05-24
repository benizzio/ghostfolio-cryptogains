package math

import (
	"errors"
	"strings"
	"testing"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// TestRoundHalfUpDivisionAndFiniteValidation verifies the shared fixed-scale
// division policy and finite-decimal guards.
// Authored by: OpenCode
func TestRoundHalfUpDivisionAndFiniteValidation(t *testing.T) {
	t.Parallel()

	var exact, err = DivideFiniteRoundHalfUp(mustMathDecimal(t, "5"), mustMathDecimal(t, "4"), 16)
	if err != nil {
		t.Fatalf("divide exact decimals: %v", err)
	}
	if got, canonicalErr := decimalsupport.CanonicalString(exact); canonicalErr != nil || got != "1.25" {
		t.Fatalf("unexpected exact quotient: %q err=%v", got, canonicalErr)
	}

	var repeating apd.Decimal
	repeating, err = DivideFiniteRoundHalfUp(mustMathDecimal(t, "1"), mustMathDecimal(t, "3"), 16)
	if err != nil {
		t.Fatalf("divide repeating decimals: %v", err)
	}
	if got, canonicalErr := decimalsupport.CanonicalString(repeating); canonicalErr != nil || got != "0.3333333333333333" {
		t.Fatalf("unexpected repeating quotient: %q err=%v", got, canonicalErr)
	}

	var halfUp apd.Decimal
	halfUp, err = DivideFiniteRoundHalfUp(mustMathDecimal(t, "1"), mustMathDecimal(t, "6"), 16)
	if err != nil {
		t.Fatalf("divide half-up decimals: %v", err)
	}
	if got, canonicalErr := decimalsupport.CanonicalString(halfUp); canonicalErr != nil || got != "0.1666666666666667" {
		t.Fatalf("unexpected half-up quotient: %q err=%v", got, canonicalErr)
	}

	var negative apd.Decimal
	negative, err = DivideFiniteRoundHalfUp(mustMathDecimal(t, "-1"), mustMathDecimal(t, "3"), 16)
	if err != nil {
		t.Fatalf("divide negative decimals: %v", err)
	}
	if got, canonicalErr := decimalsupport.CanonicalString(negative); canonicalErr != nil || got != "-0.3333333333333333" {
		t.Fatalf("unexpected negative quotient: %q err=%v", got, canonicalErr)
	}

	if err = RequireFinite(mustMathDecimal(t, "10"), "test decimal"); err != nil {
		t.Fatalf("require finite decimal: %v", err)
	}

	var invalid apd.Decimal
	invalid.Form = apd.Infinite
	if _, err = DivideFiniteRoundHalfUp(invalid, mustMathDecimal(t, "1"), 16); err == nil || !strings.Contains(err.Error(), "division dividend") {
		t.Fatalf("expected invalid dividend to fail, got %v", err)
	}
	if _, err = DivideFiniteRoundHalfUp(mustMathDecimal(t, "1"), invalid, 16); err == nil || !strings.Contains(err.Error(), "division divisor") {
		t.Fatalf("expected invalid divisor to fail, got %v", err)
	}
	if _, err = DivideFiniteRoundHalfUp(mustMathDecimal(t, "1"), apd.Decimal{}, 16); err == nil || !strings.Contains(err.Error(), "non-zero divisor") {
		t.Fatalf("expected zero divisor to fail, got %v", err)
	}
	if _, err = DivideFiniteRoundHalfUp(mustMathDecimal(t, "1"), mustMathDecimal(t, "1"), -1); err == nil || !strings.Contains(err.Error(), "scale") {
		t.Fatalf("expected negative scale to fail, got %v", err)
	}
	if err = RequireFinite(invalid, "invalid decimal"); err == nil || !strings.Contains(err.Error(), "invalid decimal") {
		t.Fatalf("expected invalid finite check to fail, got %v", err)
	}

	var numerator, denominator = finiteDecimalFraction(mustMathDecimal(t, "-1.25"))
	if numerator.String() != "-125" || denominator.String() != "100" {
		t.Fatalf("unexpected finite fraction: %s/%s", numerator.String(), denominator.String())
	}
	if got := powerOfTen(3).String(); got != "1000" {
		t.Fatalf("unexpected power-of-ten result: %s", got)
	}
}

// TestDecimalOpsAndAllocationHelpers verifies shared arithmetic, comparison,
// zero-check, and proportional-allocation behavior.
// Authored by: OpenCode
func TestDecimalOpsAndAllocationHelpers(t *testing.T) {
	t.Parallel()

	var sum, err = ApplyBinaryOperation(
		mustMathDecimal(t, "2"),
		mustMathDecimal(t, "3"),
		"left decimal",
		"right decimal",
		"add decimals",
		func(result *apd.Decimal, left *apd.Decimal, right *apd.Decimal) (apd.Condition, error) {
			return apd.BaseContext.Add(result, left, right)
		},
	)
	if err != nil {
		t.Fatalf("apply add operation: %v", err)
	}
	if got, canonicalErr := decimalsupport.CanonicalString(sum); canonicalErr != nil || got != "5" {
		t.Fatalf("unexpected sum: %q err=%v", got, canonicalErr)
	}

	var comparison int
	comparison, err = Compare(mustMathDecimal(t, "2"), mustMathDecimal(t, "10"), "left decimal", "right decimal")
	if err != nil || comparison >= 0 {
		t.Fatalf("expected 2 to compare before 10, got %d err=%v", comparison, err)
	}

	var isZero bool
	isZero, err = IsZero(mustMathDecimal(t, "0.000"), "test decimal")
	if err != nil || !isZero {
		t.Fatalf("expected canonical zero to compare as zero, got %t err=%v", isZero, err)
	}
	if got, canonicalErr := decimalsupport.CanonicalString(Minimum(mustMathDecimal(t, "2"), mustMathDecimal(t, "1"))); canonicalErr != nil || got != "1" {
		t.Fatalf("unexpected minimum decimal: %q err=%v", got, canonicalErr)
	}
	var cloned = Clone(mustMathDecimal(t, "7"))
	if cloned.Cmp(apd.New(7, 0)) != 0 {
		t.Fatalf("expected clone to preserve decimal value")
	}
	var zero = Zero()
	if zero.Sign() != 0 {
		t.Fatalf("expected zero helper to return zero")
	}

	var allocated apd.Decimal
	allocated, err = AllocateProportional(
		mustMathDecimal(t, "1"),
		mustMathDecimal(t, "3"),
		mustMathDecimal(t, "1"),
		"total basis",
		"total quantity",
		"matched quantity",
		"allocate basis proportionally",
		func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
			return ApplyBinaryOperation(left, right, "left decimal", "right decimal", "multiply decimals", func(result *apd.Decimal, left *apd.Decimal, right *apd.Decimal) (apd.Condition, error) {
				return apd.BaseContext.Mul(result, left, right)
			})
		},
		func(dividend apd.Decimal, divisor apd.Decimal) (apd.Decimal, error) {
			return DivideFiniteRoundHalfUp(dividend, divisor, 16)
		},
	)
	if err != nil {
		t.Fatalf("allocate proportional amount: %v", err)
	}
	if got, canonicalErr := decimalsupport.CanonicalString(allocated); canonicalErr != nil || got != "0.3333333333333333" {
		t.Fatalf("unexpected allocated amount: %q err=%v", got, canonicalErr)
	}

	allocated, err = AllocateProportional(
		mustMathDecimal(t, "10"),
		mustMathDecimal(t, "2"),
		mustMathDecimal(t, "2"),
		"total basis",
		"total quantity",
		"matched quantity",
		"allocate basis proportionally",
		func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) { return apd.Decimal{}, nil },
		func(dividend apd.Decimal, divisor apd.Decimal) (apd.Decimal, error) { return apd.Decimal{}, nil },
	)
	if err != nil {
		t.Fatalf("allocate full proportional amount: %v", err)
	}
	if got, canonicalErr := decimalsupport.CanonicalString(allocated); canonicalErr != nil || got != "10" {
		t.Fatalf("unexpected full allocation: %q err=%v", got, canonicalErr)
	}

	var invalid apd.Decimal
	invalid.Form = apd.NaNSignaling
	if _, err = ApplyBinaryOperation(invalid, mustMathDecimal(t, "1"), "left decimal", "right decimal", "add decimals", func(result *apd.Decimal, left *apd.Decimal, right *apd.Decimal) (apd.Condition, error) {
		return apd.BaseContext.Add(result, left, right)
	}); err == nil || !strings.Contains(err.Error(), "left decimal") {
		t.Fatalf("expected invalid left operand to fail, got %v", err)
	}
	if _, err = Compare(mustMathDecimal(t, "1"), invalid, "left decimal", "right decimal"); err == nil || !strings.Contains(err.Error(), "right decimal") {
		t.Fatalf("expected invalid right comparison operand to fail, got %v", err)
	}
	if _, err = IsZero(invalid, "test decimal"); err == nil || !strings.Contains(err.Error(), "test decimal") {
		t.Fatalf("expected invalid zero operand to fail, got %v", err)
	}
	if _, err = ApplyBinaryOperation(mustMathDecimal(t, "1"), mustMathDecimal(t, "1"), "left decimal", "right decimal", "add decimals", nil); err == nil || !strings.Contains(err.Error(), "operation is required") {
		t.Fatalf("expected missing operation to fail, got %v", err)
	}
	if _, err = ApplyBinaryOperation(mustMathDecimal(t, "1"), mustMathDecimal(t, "1"), "left decimal", "right decimal", "add decimals", func(*apd.Decimal, *apd.Decimal, *apd.Decimal) (apd.Condition, error) {
		return 0, errors.New("add boom")
	}); err == nil || !strings.Contains(err.Error(), "add decimals") {
		t.Fatalf("expected wrapped operation failure, got %v", err)
	}
	if _, err = AllocateProportional(mustMathDecimal(t, "1"), mustMathDecimal(t, "2"), mustMathDecimal(t, "3"), "total basis", "total quantity", "matched quantity", "allocate basis proportionally", nil, nil); err == nil || !strings.Contains(err.Error(), "multiplication helper") {
		t.Fatalf("expected missing multiply helper to fail, got %v", err)
	}
	if _, err = AllocateProportional(mustMathDecimal(t, "1"), mustMathDecimal(t, "2"), mustMathDecimal(t, "1"), "total basis", "total quantity", "matched quantity", "allocate basis proportionally", func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
		return left, nil
	}, nil); err == nil || !strings.Contains(err.Error(), "division helper") {
		t.Fatalf("expected missing divide helper to fail, got %v", err)
	}
	if _, err = AllocateProportional(mustMathDecimal(t, "-1"), mustMathDecimal(t, "2"), mustMathDecimal(t, "1"), "total basis", "total quantity", "matched quantity", "allocate basis proportionally", func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
		return left, nil
	}, func(dividend apd.Decimal, divisor apd.Decimal) (apd.Decimal, error) {
		return dividend, nil
	}); err == nil || !strings.Contains(err.Error(), "total basis") {
		t.Fatalf("expected negative total amount to fail, got %v", err)
	}
	if _, err = AllocateProportional(mustMathDecimal(t, "1"), mustMathDecimal(t, "0"), mustMathDecimal(t, "1"), "total basis", "total quantity", "matched quantity", "allocate basis proportionally", func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
		return left, nil
	}, func(dividend apd.Decimal, divisor apd.Decimal) (apd.Decimal, error) {
		return dividend, nil
	}); err == nil || !strings.Contains(err.Error(), "total quantity") {
		t.Fatalf("expected non-positive total quantity to fail, got %v", err)
	}
	if _, err = AllocateProportional(mustMathDecimal(t, "1"), mustMathDecimal(t, "2"), mustMathDecimal(t, "0"), "total basis", "total quantity", "matched quantity", "allocate basis proportionally", func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
		return left, nil
	}, func(dividend apd.Decimal, divisor apd.Decimal) (apd.Decimal, error) {
		return dividend, nil
	}); err == nil || !strings.Contains(err.Error(), "matched quantity") {
		t.Fatalf("expected non-positive matched quantity to fail, got %v", err)
	}
	if _, err = AllocateProportional(mustMathDecimal(t, "1"), mustMathDecimal(t, "2"), mustMathDecimal(t, "1"), "total basis", "total quantity", "matched quantity", "allocate basis proportionally", func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
		return apd.Decimal{}, errors.New("multiply boom")
	}, func(dividend apd.Decimal, divisor apd.Decimal) (apd.Decimal, error) {
		return dividend, nil
	}); err == nil || !strings.Contains(err.Error(), "multiply boom") {
		t.Fatalf("expected multiply failure to propagate, got %v", err)
	}
	if _, err = AllocateProportional(mustMathDecimal(t, "1"), mustMathDecimal(t, "2"), mustMathDecimal(t, "1"), "total basis", "total quantity", "matched quantity", "allocate basis proportionally", func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
		return left, nil
	}, func(dividend apd.Decimal, divisor apd.Decimal) (apd.Decimal, error) {
		return apd.Decimal{}, errors.New("divide boom")
	}); err == nil || !strings.Contains(err.Error(), "allocate basis proportionally") {
		t.Fatalf("expected wrapped divide failure, got %v", err)
	}
}

// mustMathDecimal parses one exact decimal fixture for shared-math tests.
// Authored by: OpenCode
func mustMathDecimal(t *testing.T, raw string) apd.Decimal {
	t.Helper()

	value, _, err := decimalsupport.ParseString(raw)
	if err != nil {
		t.Fatalf("parse shared-math decimal %q: %v", raw, err)
	}

	return value
}

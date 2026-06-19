package math

import (
	"errors"
	"strings"
	"testing"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// TestDecimalPolicyConfiguration verifies the production default policy and the
// documented accepted decimal-policy environment-variable values.
// Authored by: OpenCode
func TestDecimalPolicyConfiguration(t *testing.T) {
	const expectedPolicy = "scale=16,rounding=half_up"

	if got := DefaultDecimalPolicy().CanonicalString(); got != expectedPolicy {
		t.Fatalf("unexpected production default decimal policy: %q", got)
	}

	t.Run("default uses production policy", func(t *testing.T) {
		if err := SetActiveDecimalPolicy(DefaultDecimalPolicy()); err != nil {
			t.Fatalf("reset active decimal policy: %v", err)
		}

		var policy = DefaultDecimalPolicy()
		if policy.scale != InternalCalculationScale {
			t.Fatalf("unexpected default decimal policy scale: %d", policy.scale)
		}
		if got := policy.CanonicalString(); got != expectedPolicy {
			t.Fatalf("unexpected default decimal policy value: %q", got)
		}
		if got := ActiveDecimalPolicy().CanonicalString(); got != expectedPolicy {
			t.Fatalf("unexpected active decimal policy value: %q", got)
		}

		var quotient, err = DivideFiniteRoundHalfUp(mustMathDecimal(t, "1"), mustMathDecimal(t, "6"))
		if err != nil {
			t.Fatalf("divide decimals with default decimal policy: %v", err)
		}
		if got, canonicalErr := decimalsupport.CanonicalString(quotient); canonicalErr != nil || got != "0.1666666666666667" {
			t.Fatalf("unexpected default-policy quotient: %q err=%v", got, canonicalErr)
		}
	})

	for _, acceptedValue := range []string{expectedPolicy, "scale=4,rounding=half_up", "scale=64,rounding=half_up"} {
		var acceptedValue = acceptedValue
		t.Run("accepts "+acceptedValue, func(t *testing.T) {
			var policy, err = ParseDecimalPolicy(acceptedValue)
			if err != nil {
				t.Fatalf("select configured decimal policy %q: %v", acceptedValue, err)
			}
			if got := policy.CanonicalString(); got != acceptedValue {
				t.Fatalf("unexpected configured decimal policy value: %q", got)
			}

			if err = SetActiveDecimalPolicy(policy); err != nil {
				t.Fatalf("set active decimal policy: %v", err)
			}
			defer func() {
				if resetErr := SetActiveDecimalPolicy(DefaultDecimalPolicy()); resetErr != nil {
					t.Fatalf("reset active decimal policy: %v", resetErr)
				}
			}()

			if got := ActiveDecimalPolicy().CanonicalString(); got != acceptedValue {
				t.Fatalf("unexpected active configured decimal policy value: %q", got)
			}

			var quotient apd.Decimal
			quotient, err = DivideFiniteRoundHalfUpWithPolicy(mustMathDecimal(t, "1"), mustMathDecimal(t, "3"), policy)
			if err != nil {
				t.Fatalf("divide decimals with configured decimal policy: %v", err)
			}
			if got, canonicalErr := decimalsupport.CanonicalString(quotient); canonicalErr != nil || got != quotientForPolicyScale(policy.scale) {
				t.Fatalf("unexpected configured-policy quotient: %q err=%v", got, canonicalErr)
			}

			quotient, err = DivideFiniteRoundHalfUp(mustMathDecimal(t, "1"), mustMathDecimal(t, "6"))
			if err != nil {
				t.Fatalf("divide decimals with active decimal policy: %v", err)
			}
			if got, canonicalErr := decimalsupport.CanonicalString(quotient); canonicalErr != nil || got != quotientOneSixForPolicyScale(policy.scale) {
				t.Fatalf("unexpected active-policy quotient: %q err=%v", got, canonicalErr)
			}
		})
	}
}

// TestDecimalPolicyConfigurationErrors verifies invalid decimal-policy
// configuration forms and wrapped division-selection failures.
// Authored by: OpenCode
func TestDecimalPolicyConfigurationErrors(t *testing.T) {
	for _, fixture := range []struct {
		name        string
		configured  string
		wantMessage string
	}{
		{
			name:        "missing comma separator",
			configured:  "scale=16",
			wantMessage: "must use the form",
		},
		{
			name:        "empty scale",
			configured:  "scale=,rounding=half_up",
			wantMessage: "must use the form",
		},
		{
			name:        "non digit scale",
			configured:  "scale=1x,rounding=half_up",
			wantMessage: "must use the form",
		},
		{
			name:        "parse overflow",
			configured:  "scale=999999999999999999999999999999999999,rounding=half_up",
			wantMessage: "parse decimal policy",
		},
		{
			name:        "negative scale",
			configured:  "scale=-1,rounding=half_up",
			wantMessage: "must use the form",
		},
		{
			name:        "scale exceeds maximum",
			configured:  "scale=65,rounding=half_up",
			wantMessage: "exceeds maximum supported scale 64",
		},
	} {
		var fixture = fixture
		t.Run(fixture.name, func(t *testing.T) {
			if _, err := ParseDecimalPolicy(fixture.configured); err == nil || !strings.Contains(err.Error(), fixture.wantMessage) {
				t.Fatalf("expected decimal policy %q to fail with %q, got %v", fixture.configured, fixture.wantMessage, err)
			}
		})
	}

	if _, err := DivideFiniteRoundHalfUpWithPolicy(mustMathDecimal(t, "1"), mustMathDecimal(t, "2"), DecimalPolicy{scale: 65}); err == nil || !strings.Contains(err.Error(), "division decimal policy is invalid") {
		t.Fatalf("expected invalid explicit division policy to fail, got %v", err)
	}
	if err := SetActiveDecimalPolicy(DecimalPolicy{scale: -1}); err == nil || !strings.Contains(err.Error(), "scale must not be negative") {
		t.Fatalf("expected negative active decimal policy selection to fail, got %v", err)
	}
}

// quotientForPolicyScale returns the expected 1/3 quotient for one supported
// scale.
// Authored by: OpenCode
func quotientForPolicyScale(scale int32) string {
	switch scale {
	case 4:
		return "0.3333"
	case 16:
		return "0.3333333333333333"
	case 64:
		return "0.3333333333333333333333333333333333333333333333333333333333333333"
	default:
		return ""
	}
}

// quotientOneSixForPolicyScale returns the expected 1/6 quotient for one
// supported scale.
// Authored by: OpenCode
func quotientOneSixForPolicyScale(scale int32) string {
	switch scale {
	case 4:
		return "0.1667"
	case 16:
		return "0.1666666666666667"
	case 64:
		return "0.1666666666666666666666666666666666666666666666666666666666666667"
	default:
		return ""
	}
}

// TestRoundHalfUpDivisionAndFiniteValidation verifies the shared fixed-scale
// division policy and finite-decimal guards.
// Authored by: OpenCode
func TestRoundHalfUpDivisionAndFiniteValidation(t *testing.T) {
	t.Parallel()

	var exact, err = DivideFiniteRoundHalfUp(mustMathDecimal(t, "5"), mustMathDecimal(t, "4"))
	if err != nil {
		t.Fatalf("divide exact decimals: %v", err)
	}
	if got, canonicalErr := decimalsupport.CanonicalString(exact); canonicalErr != nil || got != "1.25" {
		t.Fatalf("unexpected exact quotient: %q err=%v", got, canonicalErr)
	}

	var repeating apd.Decimal
	repeating, err = DivideFiniteRoundHalfUp(mustMathDecimal(t, "1"), mustMathDecimal(t, "3"))
	if err != nil {
		t.Fatalf("divide repeating decimals: %v", err)
	}
	if got, canonicalErr := decimalsupport.CanonicalString(repeating); canonicalErr != nil || got != "0.3333333333333333" {
		t.Fatalf("unexpected repeating quotient: %q err=%v", got, canonicalErr)
	}

	var halfUp apd.Decimal
	halfUp, err = DivideFiniteRoundHalfUp(mustMathDecimal(t, "1"), mustMathDecimal(t, "6"))
	if err != nil {
		t.Fatalf("divide half-up decimals: %v", err)
	}
	if got, canonicalErr := decimalsupport.CanonicalString(halfUp); canonicalErr != nil || got != "0.1666666666666667" {
		t.Fatalf("unexpected half-up quotient: %q err=%v", got, canonicalErr)
	}

	var negative apd.Decimal
	negative, err = DivideFiniteRoundHalfUp(mustMathDecimal(t, "-1"), mustMathDecimal(t, "3"))
	if err != nil {
		t.Fatalf("divide negative decimals: %v", err)
	}
	if got, canonicalErr := decimalsupport.CanonicalString(negative); canonicalErr != nil || got != "-0.3333333333333333" {
		t.Fatalf("unexpected negative quotient: %q err=%v", got, canonicalErr)
	}

	if err = RequireFinite(mustMathDecimal(t, "10")); err != nil {
		t.Fatalf("require finite decimal: %v", err)
	}

	var invalid apd.Decimal
	invalid.Form = apd.Infinite
	if _, err = DivideFiniteRoundHalfUp(invalid, mustMathDecimal(t, "1")); err == nil || !strings.Contains(err.Error(), "division dividend") {
		t.Fatalf("expected invalid dividend to fail, got %v", err)
	}
	if _, err = DivideFiniteRoundHalfUp(mustMathDecimal(t, "1"), invalid); err == nil || !strings.Contains(err.Error(), "division divisor") {
		t.Fatalf("expected invalid divisor to fail, got %v", err)
	}
	if _, err = DivideFiniteRoundHalfUp(mustMathDecimal(t, "1"), apd.Decimal{}); err == nil || !strings.Contains(err.Error(), "non-zero divisor") {
		t.Fatalf("expected zero divisor to fail, got %v", err)
	}
	if err = RequireFinite(invalid); err == nil || !strings.Contains(err.Error(), "finite") {
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

	var difference apd.Decimal
	difference, err = Subtract(mustMathDecimal(t, "5"), mustMathDecimal(t, "3"))
	if err != nil {
		t.Fatalf("subtract decimals: %v", err)
	}
	if got, canonicalErr := decimalsupport.CanonicalString(difference); canonicalErr != nil || got != "2" {
		t.Fatalf("unexpected difference: %q err=%v", got, canonicalErr)
	}

	var product apd.Decimal
	product, err = Multiply(mustMathDecimal(t, "1.25"), mustMathDecimal(t, "4"))
	if err != nil {
		t.Fatalf("multiply decimals: %v", err)
	}
	if got, canonicalErr := decimalsupport.CanonicalString(product); canonicalErr != nil || got != "5" {
		t.Fatalf("unexpected product: %q err=%v", got, canonicalErr)
	}

	var comparison int
	comparison, err = Compare(mustMathDecimal(t, "2"), mustMathDecimal(t, "10"))
	if err != nil || comparison >= 0 {
		t.Fatalf("expected 2 to compare before 10, got %d err=%v", comparison, err)
	}

	var isZero bool
	isZero, err = IsZero(mustMathDecimal(t, "0.000"))
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

	if err = RequirePositive(mustMathDecimal(t, "1")); err != nil {
		t.Fatalf("require positive decimal: %v", err)
	}
	if err = RequireNonNegative(mustMathDecimal(t, "0")); err != nil {
		t.Fatalf("require non-negative decimal: %v", err)
	}

	var allocated apd.Decimal
	allocated, err = AllocateProportional(
		mustMathDecimal(t, "1"),
		mustMathDecimal(t, "3"),
		mustMathDecimal(t, "1"),
		func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
			return ApplyBinaryOperation(left, right, func(result *apd.Decimal, left *apd.Decimal, right *apd.Decimal) (apd.Condition, error) {
				return apd.BaseContext.Mul(result, left, right)
			})
		},
		DivideFiniteRoundHalfUp,
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
	if _, err = ApplyBinaryOperation(invalid, mustMathDecimal(t, "1"), func(result *apd.Decimal, left *apd.Decimal, right *apd.Decimal) (apd.Condition, error) {
		return apd.BaseContext.Add(result, left, right)
	}); err == nil || !strings.Contains(err.Error(), "left decimal operand") {
		t.Fatalf("expected invalid left operand to fail, got %v", err)
	}
	if _, err = Compare(mustMathDecimal(t, "1"), invalid); err == nil || !strings.Contains(err.Error(), "right decimal operand") {
		t.Fatalf("expected invalid right comparison operand to fail, got %v", err)
	}
	if _, err = IsZero(invalid); err == nil || !strings.Contains(err.Error(), "decimal operand") {
		t.Fatalf("expected invalid zero operand to fail, got %v", err)
	}
	if _, err = Add(invalid, mustMathDecimal(t, "1")); err == nil || !strings.Contains(err.Error(), "left decimal operand") {
		t.Fatalf("expected invalid add operand to fail, got %v", err)
	}
	if _, err = Subtract(mustMathDecimal(t, "1"), invalid); err == nil || !strings.Contains(err.Error(), "right decimal operand") {
		t.Fatalf("expected invalid subtract operand to fail, got %v", err)
	}
	if _, err = Multiply(mustMathDecimal(t, "1"), invalid); err == nil || !strings.Contains(err.Error(), "right decimal operand") {
		t.Fatalf("expected invalid multiply operand to fail, got %v", err)
	}
	if err = RequirePositive(mustMathDecimal(t, "0")); err == nil || !strings.Contains(err.Error(), "decimal operand must be greater than zero") {
		t.Fatalf("expected non-positive decimal to fail, got %v", err)
	}
	if err = RequireNonNegative(mustMathDecimal(t, "-1")); err == nil || !strings.Contains(err.Error(), "decimal operand must not be negative") {
		t.Fatalf("expected negative decimal to fail, got %v", err)
	}
	if _, err = ApplyBinaryOperation(mustMathDecimal(t, "1"), mustMathDecimal(t, "1"), nil); err == nil || !strings.Contains(err.Error(), "operation is required") {
		t.Fatalf("expected missing operation to fail, got %v", err)
	}
	if _, err = ApplyBinaryOperation(mustMathDecimal(t, "1"), mustMathDecimal(t, "1"), func(*apd.Decimal, *apd.Decimal, *apd.Decimal) (apd.Condition, error) {
		return 0, errors.New("add boom")
	}); err == nil || !strings.Contains(err.Error(), "decimal operation failed") {
		t.Fatalf("expected wrapped operation failure, got %v", err)
	}
	if _, err = AllocateProportional(mustMathDecimal(t, "1"), mustMathDecimal(t, "2"), mustMathDecimal(t, "3"), nil, nil); err == nil || !strings.Contains(err.Error(), "multiplication helper") {
		t.Fatalf("expected missing multiply helper to fail, got %v", err)
	}
	if _, err = AllocateProportional(mustMathDecimal(t, "1"), mustMathDecimal(t, "2"), mustMathDecimal(t, "1"), func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
		return left, nil
	}, nil); err == nil || !strings.Contains(err.Error(), "division helper") {
		t.Fatalf("expected missing divide helper to fail, got %v", err)
	}
	if _, err = AllocateProportional(mustMathDecimal(t, "-1"), mustMathDecimal(t, "2"), mustMathDecimal(t, "1"), func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
		return left, nil
	}, func(dividend apd.Decimal, divisor apd.Decimal) (apd.Decimal, error) {
		return dividend, nil
	}); err == nil || !strings.Contains(err.Error(), "total amount") {
		t.Fatalf("expected negative total amount to fail, got %v", err)
	}
	if _, err = AllocateProportional(mustMathDecimal(t, "1"), mustMathDecimal(t, "0"), mustMathDecimal(t, "1"), func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
		return left, nil
	}, func(dividend apd.Decimal, divisor apd.Decimal) (apd.Decimal, error) {
		return dividend, nil
	}); err == nil || !strings.Contains(err.Error(), "total quantity") {
		t.Fatalf("expected non-positive total quantity to fail, got %v", err)
	}
	if _, err = AllocateProportional(mustMathDecimal(t, "1"), mustMathDecimal(t, "2"), mustMathDecimal(t, "0"), func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
		return left, nil
	}, func(dividend apd.Decimal, divisor apd.Decimal) (apd.Decimal, error) {
		return dividend, nil
	}); err == nil || !strings.Contains(err.Error(), "portion quantity") {
		t.Fatalf("expected non-positive matched quantity to fail, got %v", err)
	}
	if _, err = AllocateProportional(mustMathDecimal(t, "1"), mustMathDecimal(t, "2"), mustMathDecimal(t, "1"), func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
		return apd.Decimal{}, errors.New("multiply boom")
	}, func(dividend apd.Decimal, divisor apd.Decimal) (apd.Decimal, error) {
		return dividend, nil
	}); err == nil || !strings.Contains(err.Error(), "multiply boom") {
		t.Fatalf("expected multiply failure to propagate, got %v", err)
	}
	if _, err = AllocateProportional(mustMathDecimal(t, "1"), mustMathDecimal(t, "2"), mustMathDecimal(t, "1"), func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
		return left, nil
	}, func(dividend apd.Decimal, divisor apd.Decimal) (apd.Decimal, error) {
		return apd.Decimal{}, errors.New("divide boom")
	}); err == nil || !strings.Contains(err.Error(), "allocate proportional amount") {
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

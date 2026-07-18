// Package presentation tests the exact report-visible financial formatter.
// Authored by: OpenCode
package presentation

import (
	"errors"
	"math"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/cockroachdb/apd/v3"
)

// TestFormatFinancialValueUsesScaleTwoHalfUp verifies fixed-point scale-2
// output and symmetric HALF UP rounding for positive, negative, zero, and
// non-negative values.
// Authored by: OpenCode
func TestFormatFinancialValueUsesScaleTwoHalfUp(t *testing.T) {
	var testCases = []struct {
		name  string
		input string
		want  string
	}{
		{name: "positive whole", input: "1", want: "1.00"},
		{name: "positive one place", input: "1.2", want: "1.20"},
		{name: "positive exact zero", input: "0", want: "0.00"},
		{name: "positive below half", input: "1.004", want: "1.00"},
		{name: "positive exact half", input: "1.005", want: "1.01"},
		{name: "positive above half", input: "1.006", want: "1.01"},
		{name: "negative whole", input: "-1", want: "-1.00"},
		{name: "negative one place", input: "-1.2", want: "-1.20"},
		{name: "negative below half", input: "-1.004", want: "-1.00"},
		{name: "negative exact half", input: "-1.005", want: "-1.01"},
		{name: "negative above half", input: "-1.006", want: "-1.01"},
		{name: "positive small non-zero", input: "0.004", want: "0.00"},
		{name: "negative small non-zero", input: "-0.004", want: "0.00"},
	}

	for _, testCase := range testCases {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			var value = mustFinancialDecimal(t, testCase.input)
			var got, err = formatFinancialValue(value)
			if err != nil {
				t.Fatalf("format financial value: %v", err)
			}
			if got != testCase.want {
				t.Fatalf("formatted value = %q, want %q", got, testCase.want)
			}

			var matched, matchErr = regexp.MatchString(`^-?[0-9]+\.[0-9]{2}$`, got)
			if matchErr != nil || !matched {
				t.Fatalf("formatted value %q does not match the fixed-point grammar", got)
			}
		})
	}
}

// TestFormatOptionalFinancialValuePreservesNil verifies that an absent
// optional amount remains blank instead of becoming a visible zero.
// Authored by: OpenCode
func TestFormatOptionalFinancialValuePreservesNil(t *testing.T) {
	var got, err = formatOptionalFinancialValue(nil)
	if err != nil {
		t.Fatalf("format nil optional financial value: %v", err)
	}
	if got != "" {
		t.Fatalf("formatted nil optional value = %q, want blank", got)
	}
}

// TestFormatOptionalFinancialValueUsesTheExportedBoundary verifies the public
// optional-value wrapper preserves nil and delegates present values exactly.
// Authored by: OpenCode
func TestFormatOptionalFinancialValueUsesTheExportedBoundary(t *testing.T) {
	if got, err := FormatOptionalFinancialValue(nil); err != nil || got != "" {
		t.Fatalf("exported nil optional value = %q, %v; want blank without error", got, err)
	}

	var value = mustFinancialDecimal(t, "1.005")
	var before = value
	var got, err = FormatOptionalFinancialValue(&value)
	if err != nil {
		t.Fatalf("format exported optional value: %v", err)
	}
	if got != "1.01" {
		t.Fatalf("exported optional value = %q, want %q", got, "1.01")
	}
	if !reflect.DeepEqual(value, before) {
		t.Fatalf("exported optional formatter mutated source: before=%#v after=%#v", before, value)
	}
}

// TestFormatFinancialValueNormalizesNegativeZero verifies that exact and
// rounded negative zero use the neutral visible representation.
// Authored by: OpenCode
func TestFormatFinancialValueNormalizesNegativeZero(t *testing.T) {
	for _, input := range []string{"-0", "-0.000", "-0.0049"} {
		var value = mustFinancialDecimal(t, input)
		var got, err = formatFinancialValue(value)
		if err != nil {
			t.Fatalf("format %q: %v", input, err)
		}
		if got != "0.00" {
			t.Errorf("formatted %q = %q, want %q", input, got, "0.00")
		}
	}
}

// TestFormatFinancialValueDoesNotMutateSource verifies that quantization uses
// a defensive decimal copy, including coefficient storage that is large enough
// to escape apd's inline representation.
// Authored by: OpenCode
func TestFormatFinancialValueDoesNotMutateSource(t *testing.T) {
	var source = mustFinancialDecimal(t, "1234567890123456789012345678901234567890.125")
	var before = source

	var got, err = formatFinancialValue(source)
	if err != nil {
		t.Fatalf("format source value: %v", err)
	}
	if got != "1234567890123456789012345678901234567890.13" {
		t.Fatalf("formatted source value = %q", got)
	}
	if !reflect.DeepEqual(source, before) {
		t.Fatalf("formatter mutated source decimal: before=%#v after=%#v", before, source)
	}
}

// TestFormatFinancialValueDelegatesOrdinaryExpansionToAPD verifies ordinary
// values reach Quantize unchanged through a fully initialized package context.
// Authored by: OpenCode
func TestFormatFinancialValueDelegatesOrdinaryExpansionToAPD(t *testing.T) {
	var previousQuantize = quantizeFinancialValue
	t.Cleanup(func() { quantizeFinancialValue = previousQuantize })
	quantizeFinancialValue = func(context *apd.Context, result *apd.Decimal, source *apd.Decimal, exponent int32) (apd.Condition, error) {
		if context.MaxExponent != apd.MaxExponent || context.MinExponent != apd.MinExponent {
			t.Fatalf("quantize context exponent bounds = [%d, %d]", context.MinExponent, context.MaxExponent)
		}
		if context.Traps != apd.DefaultTraps || context.Rounding != apd.RoundHalfUp {
			t.Fatalf("quantize context traps = %v, rounding = %q", context.Traps, context.Rounding)
		}
		if source.Exponent != 0 {
			t.Fatalf("ordinary source exponent = %d, want unchanged exponent 0", source.Exponent)
		}
		if exponent != financialDisplayExponent {
			t.Fatalf("quantize exponent = %d, want %d", exponent, financialDisplayExponent)
		}
		return previousQuantize(context, result, source, exponent)
	}

	var got, err = formatFinancialValue(mustFinancialDecimal(t, "1"))
	if err != nil {
		t.Fatalf("format ordinary value: %v", err)
	}
	if got != "1.00" {
		t.Fatalf("formatted ordinary value = %q, want %q", got, "1.00")
	}
}

// TestFormatFinancialValueAcceptsAdjustedExponentBounds verifies both
// inclusive adjusted-exponent endpoints and preserves the full upper-bound
// value without exponent notation.
// Authored by: OpenCode
func TestFormatFinancialValueAcceptsAdjustedExponentBounds(t *testing.T) {
	var testCases = []struct {
		name  string
		input string
		want  string
	}{
		{name: "lower bound", input: "1e-100000", want: "0.00"},
		{name: "lower bound negative", input: "-1e-100000", want: "0.00"},
		{name: "upper bound", input: "1e100000", want: "1" + strings.Repeat("0", 100000) + ".00"},
		{name: "upper bound negative", input: "-1e100000", want: "-1" + strings.Repeat("0", 100000) + ".00"},
	}

	for _, testCase := range testCases {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			var value = mustFinancialDecimal(t, testCase.input)
			var got, err = formatFinancialValue(value)
			if err != nil {
				t.Fatalf("format adjusted-exponent boundary: %v", err)
			}
			if got != testCase.want {
				t.Fatalf("formatted boundary value length=%d, want length=%d", len(got), len(testCase.want))
			}
		})
	}
}

// TestFormatFinancialValueRejectsAdjustedExponentBounds verifies that finite
// values immediately outside either inclusive endpoint fail before visible
// output is produced.
// Authored by: OpenCode
func TestFormatFinancialValueRejectsAdjustedExponentBounds(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		exponent int32
		negative bool
	}{
		{name: "positive lower", exponent: -100001},
		{name: "negative lower", exponent: -100001, negative: true},
		{name: "positive upper", exponent: 100001},
		{name: "negative upper", exponent: 100001, negative: true},
	} {
		var value = decimalWithExponent(testCase.exponent, testCase.negative)
		var input = testCase.name
		assertFinancialFormattingError(t, input, value)
	}
}

// TestFormatFinancialValueRejectsUpperBoundCarry verifies that HALF UP carry
// cannot move an accepted source at adjusted exponent 100000 to 100001.
// Authored by: OpenCode
func TestFormatFinancialValueRejectsUpperBoundCarry(t *testing.T) {
	var integerPart = strings.Repeat("9", 100001)
	for _, input := range []string{integerPart + ".995", "-" + integerPart + ".995"} {
		var value = mustFinancialDecimal(t, input)
		assertFinancialFormattingError(t, input, value)
	}
}

// TestFormatFinancialValueRejectsExtremeSourceExponent verifies that source
// metadata near the signed exponent limit is rejected without arithmetic wrap.
// Authored by: OpenCode
func TestFormatFinancialValueRejectsExtremeSourceExponent(t *testing.T) {
	var value apd.Decimal
	value.Form = apd.Finite
	value.Coeff.SetInt64(1)
	value.Exponent = math.MinInt32

	assertFinancialFormattingError(t, "minimum int32 exponent", value)
}

// TestCheckedFinancialPrecisionAcceptsOnlyAPDSupportedResults verifies the
// carry-inclusive calculation at apd's safe signed exponent-arithmetic limit.
// Authored by: OpenCode
func TestCheckedFinancialPrecisionAcceptsOnlyAPDSupportedResults(t *testing.T) {
	var testCases = []struct {
		name         string
		sourceDigits int64
		expansion    int64
		want         uint32
		wantError    bool
	}{
		{name: "smallest valid precision", sourceDigits: 1, expansion: 0, want: 2},
		{name: "maximum precision", sourceDigits: 2147383648, expansion: 0, want: 2147383649},
		{name: "maximum precision with expansion", sourceDigits: 2147383647, expansion: 1, want: 2147383649},
		{name: "source digit overflow", sourceDigits: 2147383649, expansion: 0, wantError: true},
		{name: "expansion overflow", sourceDigits: 2147383648, expansion: 1, wantError: true},
		{name: "expansion exceeds apd limit", sourceDigits: 1, expansion: 2147383650, wantError: true},
		{name: "zero source digits", sourceDigits: 0, expansion: 0, wantError: true},
		{name: "negative expansion", sourceDigits: 1, expansion: -1, wantError: true},
	}

	for _, testCase := range testCases {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			var got, err = checkedFinancialPrecision(testCase.sourceDigits, testCase.expansion)
			if testCase.wantError {
				if err == nil {
					t.Fatalf("precision result = %d without an error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("check precision: %v", err)
			}
			if got != testCase.want {
				t.Fatalf("precision = %d, want %d", got, testCase.want)
			}
		})
	}
}

// TestFormatFinancialValueRejectsNonFiniteValues verifies that infinity and
// both NaN forms cannot become report-visible financial strings.
// Authored by: OpenCode
func TestFormatFinancialValueRejectsNonFiniteValues(t *testing.T) {
	for _, form := range []apd.Form{apd.Infinite, apd.NaN, apd.NaNSignaling} {
		var value apd.Decimal
		value.Form = form
		assertFinancialFormattingError(t, form.String(), value)
	}
}

// TestFormatFinancialValueRejectsNegativeCoefficient verifies malformed finite
// decimals cannot cross the report-visible financial boundary.
// Authored by: OpenCode
func TestFormatFinancialValueRejectsNegativeCoefficient(t *testing.T) {
	var value apd.Decimal
	value.Form = apd.Finite
	value.Coeff.SetInt64(-1)
	assertFinancialFormattingError(t, "negative coefficient", value)
}

// TestCheckedFinancialAdjustedExponentRejectsInvalidInputs verifies the direct
// checked-arithmetic guards independently of decimal construction limits.
// Authored by: OpenCode
func TestCheckedFinancialAdjustedExponentRejectsInvalidInputs(t *testing.T) {
	var maxInt64 = int64(^uint64(0) >> 1)
	for _, testCase := range []struct {
		name          string
		exponent      int64
		sourceDigits  int64
		wantErrorText string
	}{
		{name: "no coefficient digits", exponent: 0, sourceDigits: 0, wantErrorText: "has no digits"},
		{name: "adjusted exponent overflow", exponent: maxInt64, sourceDigits: 2, wantErrorText: "overflows"},
	} {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			var got, err = checkedFinancialAdjustedExponent(testCase.exponent, testCase.sourceDigits)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErrorText) {
				t.Fatalf("adjusted exponent = %d, error = %v, want %q", got, err, testCase.wantErrorText)
			}
		})
	}
}

// TestFormatFinancialValueReturnsQuantizationErrors verifies that a library
// operation failure is returned before any formatted value is exposed.
// Authored by: OpenCode
func TestFormatFinancialValueReturnsQuantizationErrors(t *testing.T) {
	var value = mustFinancialDecimal(t, "1.23")
	var previousQuantize = quantizeFinancialValue
	t.Cleanup(func() { quantizeFinancialValue = previousQuantize })
	quantizeFinancialValue = func(_ *apd.Context, _ *apd.Decimal, _ *apd.Decimal, _ int32) (apd.Condition, error) {
		return 0, errors.New("quantize failure")
	}

	assertFinancialFormattingError(t, "quantization context error", value)
}

// TestFormatFinancialValueRejectsUnexpectedQuantizationConditions verifies
// decimal conditions outside the rounding contract are rejected explicitly.
// Authored by: OpenCode
func TestFormatFinancialValueRejectsUnexpectedQuantizationConditions(t *testing.T) {
	var value = mustFinancialDecimal(t, "1.23")
	var previousQuantize = quantizeFinancialValue
	t.Cleanup(func() { quantizeFinancialValue = previousQuantize })
	quantizeFinancialValue = func(_ *apd.Context, _ *apd.Decimal, _ *apd.Decimal, _ int32) (apd.Condition, error) {
		return apd.Clamped, nil
	}

	assertFinancialFormattingError(t, "unexpected quantization condition", value)
}

// TestFormatFinancialValueRejectsUnexpectedDecimalConditions verifies that an
// invalid finite decimal state is rejected instead of allowing an unexpected
// apd condition to become visible output.
// Authored by: OpenCode
func TestFormatFinancialValueRejectsUnexpectedDecimalConditions(t *testing.T) {
	var value apd.Decimal
	value.Form = apd.Finite
	value.Coeff.SetInt64(1)
	value.Exponent = apd.MaxExponent + 1

	assertFinancialFormattingError(t, "unexpected exponent condition", value)
}

// TestFormatFinancialValuePropagatesCheckedMetadataErrors verifies formatter
// guard propagation for checked arithmetic failures.
// Authored by: OpenCode
func TestFormatFinancialValuePropagatesCheckedMetadataErrors(t *testing.T) {
	var value = mustFinancialDecimal(t, "1.23")
	var previousAdjusted = checkedFinancialAdjustedExponentForFormatting
	var previousPrecision = checkedFinancialPrecisionForFormatting
	t.Cleanup(func() {
		checkedFinancialAdjustedExponentForFormatting = previousAdjusted
		checkedFinancialPrecisionForFormatting = previousPrecision
	})

	checkedFinancialAdjustedExponentForFormatting = func(int64, int64) (int64, error) {
		return 0, errors.New("adjusted exponent seam failure")
	}
	assertFinancialFormattingError(t, "adjusted exponent seam", value)

	checkedFinancialAdjustedExponentForFormatting = previousAdjusted
	checkedFinancialPrecisionForFormatting = func(int64, int64) (uint32, error) {
		return 0, errors.New("precision seam failure")
	}
	assertFinancialFormattingError(t, "precision seam", value)
}

// TestRequiresFinancialCoefficientPreExpansion verifies that custom scaling is
// reserved for shifts that exceed apd's internal exponent range.
// Authored by: OpenCode
func TestRequiresFinancialCoefficientPreExpansion(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		exponent int32
		want     bool
	}{
		{name: "ordinary integer", exponent: 0},
		{name: "largest direct shift", exponent: 99998},
		{name: "first excessive shift", exponent: 99999, want: true},
		{name: "upper adjusted exponent", exponent: 100000, want: true},
	} {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			var got = requiresFinancialCoefficientPreExpansion(testCase.exponent)
			if got != testCase.want {
				t.Fatalf("pre-expansion for exponent %d = %t, want %t", testCase.exponent, got, testCase.want)
			}
		})
	}
}

// TestValidateFinancialQuantizeConditionsRejectsUnexpectedFlags verifies that
// only the expected apd rounding flags are accepted after quantization.
// Authored by: OpenCode
func TestValidateFinancialQuantizeConditionsRejectsUnexpectedFlags(t *testing.T) {
	for _, condition := range []apd.Condition{0, apd.Rounded, apd.Inexact, apd.Rounded | apd.Inexact} {
		if err := validateFinancialQuantizeConditions(condition); err != nil {
			t.Errorf("condition %v rejected: %v", condition, err)
		}
	}

	for _, condition := range []apd.Condition{
		apd.SystemOverflow,
		apd.SystemUnderflow,
		apd.Overflow,
		apd.Underflow,
		apd.Subnormal,
		apd.DivisionUndefined,
		apd.DivisionByZero,
		apd.DivisionImpossible,
		apd.InvalidOperation,
		apd.Clamped,
	} {
		if err := validateFinancialQuantizeConditions(condition); err == nil {
			t.Errorf("unexpected condition %v was accepted", condition)
		}
	}
}

// mustFinancialDecimal parses one synthetic exact decimal for formatter tests.
// Authored by: OpenCode
func mustFinancialDecimal(t *testing.T, raw string) apd.Decimal {
	t.Helper()

	var value apd.Decimal
	if _, _, err := value.SetString(raw); err != nil {
		t.Fatalf("parse test decimal %q: %v", raw, err)
	}
	return value
}

// decimalWithExponent constructs a finite boundary value that apd's parser
// intentionally refuses before the formatter can validate its domain.
// Authored by: OpenCode
func decimalWithExponent(exponent int32, negative bool) apd.Decimal {
	var value apd.Decimal
	value.Form = apd.Finite
	value.Coeff.SetInt64(1)
	value.Exponent = exponent
	value.Negative = negative
	return value
}

// assertFinancialFormattingError verifies that one invalid formatter input
// returns no visible value and a non-nil error.
// Authored by: OpenCode
func assertFinancialFormattingError(t *testing.T, name string, value apd.Decimal) {
	t.Helper()

	var got, err = formatFinancialValue(value)
	if err == nil {
		t.Fatalf("format %s returned %q without an error", name, got)
	}
	if got != "" {
		t.Fatalf("format %s returned visible value %q with error %v", name, got, err)
	}
}

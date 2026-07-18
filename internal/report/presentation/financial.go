package presentation

import (
	"errors"
	"fmt"
	"math"

	"github.com/cockroachdb/apd/v3"
)

// Financial formatting constants define the report scale and the largest
// precision supported by apd's signed exponent arithmetic.
// Authored by: OpenCode
const (
	financialDisplayExponent int32 = -2
	maxFinancialPrecision    int64 = int64(math.MaxInt32) + int64(apd.MinExponent) + 2
)

// Financial operation seams keep defensive decimal branches testable without
// weakening the normal apd-backed formatter path.
// Authored by: OpenCode
var (
	checkedFinancialAdjustedExponentForFormatting = checkedFinancialAdjustedExponent
	checkedFinancialPrecisionForFormatting        = checkedFinancialPrecision
	quantizeFinancialValue                        = (*apd.Context).Quantize
)

// RequiredPrecisionChecker verifies the precision needed for one financial
// value before apd quantization. It exists to let renderer-scoped tests model
// the pinned library's precision ceiling without allocating an unsafe
// coefficient.
// Authored by: OpenCode
type RequiredPrecisionChecker func(sourceDigits int64, coefficientExpansion int64) (uint32, error)

// FinancialFormattingOptions is an immutable financial-formatting policy for
// one renderer instance. A zero value uses the concrete production precision
// checker; use NewFinancialFormattingTestOptions only for scoped failure tests.
// Authored by: OpenCode
type FinancialFormattingOptions struct {
	precisionChecker RequiredPrecisionChecker
	quantizer        func(*apd.Context, *apd.Decimal, *apd.Decimal, int32) (apd.Condition, error)
}

// DefaultFinancialFormattingOptions returns the concrete production financial
// formatting policy. The returned value is safe to copy into separate renderer
// instances and uses HALF UP quantization with the pinned apd limits.
//
// Example:
//
//	options := presentation.DefaultFinancialFormattingOptions()
//	formatted, err := options.Format(value)
//	_ = formatted
//	_ = err
//
// Authored by: OpenCode
func DefaultFinancialFormattingOptions() FinancialFormattingOptions {
	return FinancialFormattingOptions{precisionChecker: checkedFinancialPrecision}
}

// NewFinancialFormattingTestOptions creates a renderer-scoped policy for
// deterministic formatting-failure tests. The supplied failure hook is called
// with the exact source metadata calculated by the production formatter; when
// it does not fail, the concrete production precision checker is used.
//
// Example:
//
//	options := presentation.NewFinancialFormattingTestOptions(func(int64, int64) error {
//		return errors.New("required precision exceeds apd operational limit")
//	})
//	_ = options
//
// Authored by: OpenCode
func NewFinancialFormattingTestOptions(failureHook func(sourceDigits int64, coefficientExpansion int64) error) FinancialFormattingOptions {
	if failureHook == nil {
		return DefaultFinancialFormattingOptions()
	}
	return FinancialFormattingOptions{
		precisionChecker: func(sourceDigits int64, coefficientExpansion int64) (uint32, error) {
			if err := failureHook(sourceDigits, coefficientExpansion); err != nil {
				return 0, err
			}
			return checkedFinancialPrecision(sourceDigits, coefficientExpansion)
		},
	}
}

// Format applies this renderer-scoped policy to one report-visible financial
// value without mutating the supplied decimal.
//
// Example:
//
//	formatted, err := options.Format(value)
//	if err != nil {
//		return err
//	}
//	_ = formatted
//
// Authored by: OpenCode
func (options FinancialFormattingOptions) Format(value apd.Decimal) (string, error) {
	if options.precisionChecker == nil {
		options = DefaultFinancialFormattingOptions()
	}
	return formatFinancialValueWithOptions(value, options)
}

// FormatOptional applies this renderer-scoped policy while preserving an
// absent optional amount as a blank string.
// Authored by: OpenCode
func (options FinancialFormattingOptions) FormatOptional(value *apd.Decimal) (string, error) {
	if value == nil {
		return "", nil
	}
	return options.Format(*value)
}

// FormatFinancialValue formats one report-visible financial value at scale two
// using HALF UP rounding. It protects the supplied decimal from mutation and
// returns fixed-point ASCII text suitable for report output.
//
// Example:
//
//	var value = *apd.New(1005, -3)
//	formatted, err := presentation.FormatFinancialValue(value)
//	if err != nil {
//		// Handle a value outside the report presentation domain.
//	}
//	_ = formatted
//
// Authored by: OpenCode
func FormatFinancialValue(value apd.Decimal) (string, error) {
	return formatFinancialValue(value)
}

// FormatOptionalFinancialValue formats a present optional financial value at
// scale two using HALF UP rounding and preserves nil as an empty string. The
// supplied decimal is not mutated.
//
// Example:
//
//	var value *apd.Decimal
//	formatted, err := presentation.FormatOptionalFinancialValue(value)
//	if err != nil {
//		// Handle a value outside the report presentation domain.
//	}
//	_ = formatted
//
// Authored by: OpenCode
func FormatOptionalFinancialValue(value *apd.Decimal) (string, error) {
	return formatOptionalFinancialValue(value)
}

// formatFinancialValue returns a report-visible financial value at scale two.
// It rounds a defensive copy with HALF UP semantics and emits fixed-point ASCII
// text without changing the supplied decimal.
// Authored by: OpenCode
func formatFinancialValue(value apd.Decimal) (string, error) {
	return formatFinancialValueWithOptions(value, FinancialFormattingOptions{
		precisionChecker: checkedFinancialPrecisionForFormatting,
		quantizer:        quantizeFinancialValue,
	})
}

// formatFinancialValueWithOptions formats one value with a renderer-owned
// precision policy while keeping the apd quantization path concrete.
// Authored by: OpenCode
func formatFinancialValueWithOptions(value apd.Decimal, options FinancialFormattingOptions) (string, error) {
	if value.Form != apd.Finite {
		return "", errors.New("financial value is not finite")
	}
	if value.Coeff.Sign() < 0 {
		return "", errors.New("financial value has an invalid coefficient")
	}

	var sourceDigits = value.NumDigits()
	adjustedExponent, err := checkedFinancialAdjustedExponentForFormatting(int64(value.Exponent), sourceDigits)
	if err != nil {
		return "", err
	}
	if adjustedExponent < int64(apd.MinExponent) || adjustedExponent > int64(apd.MaxExponent) {
		return "", errors.New("financial value adjusted exponent is out of range")
	}

	var coefficientExpansion int64
	if value.Exponent > financialDisplayExponent {
		coefficientExpansion = int64(value.Exponent) - int64(financialDisplayExponent)
	}
	var precisionChecker = options.precisionChecker
	if precisionChecker == nil {
		precisionChecker = checkedFinancialPrecision
	}
	precision, err := precisionChecker(sourceDigits, coefficientExpansion)
	if err != nil {
		return "", err
	}

	if value.Coeff.Sign() == 0 {
		return "0.00", nil
	}

	var quantized apd.Decimal
	var quantizer = options.quantizer
	if quantizer == nil {
		quantizer = (*apd.Context).Quantize
	}
	quantized, err = quantizeFinancialValueForFormattingWithOptions(value, coefficientExpansion, precision, quantizer)
	if err != nil {
		if adjustedExponent == int64(apd.MaxExponent) {
			return "", errors.New("financial value adjusted exponent exceeds range after rounding")
		}
		return "", err
	}

	return quantized.Text('f'), nil
}

// quantizeFinancialValueForFormattingWithOptions performs the defensive
// coefficient handling and concrete apd quantization used by each renderer.
// Authored by: OpenCode
func quantizeFinancialValueForFormattingWithOptions(value apd.Decimal, coefficientExpansion int64, precision uint32, quantize func(*apd.Context, *apd.Decimal, *apd.Decimal, int32) (apd.Condition, error)) (apd.Decimal, error) {
	var source = &value
	var expandedSource apd.Decimal
	if requiresFinancialCoefficientPreExpansion(value.Exponent) {
		expandedSource.Set(&value)
		var ten apd.BigInt
		ten.SetInt64(10)
		var expansionExponent apd.BigInt
		expansionExponent.SetInt64(coefficientExpansion)
		var scaleFactor apd.BigInt
		scaleFactor.Exp(&ten, &expansionExponent, nil)
		expandedSource.Coeff.Mul(&expandedSource.Coeff, &scaleFactor)
		expandedSource.Exponent = financialDisplayExponent
		source = &expandedSource
	}

	var quantized apd.Decimal
	var context = apd.Context{
		Precision:   precision,
		MaxExponent: apd.MaxExponent,
		MinExponent: apd.MinExponent,
		Traps:       apd.DefaultTraps,
		Rounding:    apd.RoundHalfUp,
	}
	condition, err := quantize(&context, &quantized, source, financialDisplayExponent)
	if err != nil {
		return apd.Decimal{}, fmt.Errorf("financial value quantization failed: %w", err)
	}
	if err := validateFinancialQuantizeConditions(condition); err != nil {
		return apd.Decimal{}, err
	}
	if quantized.IsZero() {
		quantized.Negative = false
	}
	return quantized, nil
}

// requiresFinancialCoefficientPreExpansion reports whether apd would reject
// the source-to-display exponent shift before performing quantization.
// Authored by: OpenCode
func requiresFinancialCoefficientPreExpansion(sourceExponent int32) bool {
	var exponentShift = int64(financialDisplayExponent) - int64(sourceExponent)
	return exponentShift < int64(apd.MinExponent)
}

// formatOptionalFinancialValue preserves an absent optional financial value as
// blank and delegates present values to the exact financial formatter.
// Authored by: OpenCode
func formatOptionalFinancialValue(value *apd.Decimal) (string, error) {
	if value == nil {
		return "", nil
	}
	return formatFinancialValue(*value)
}

// checkedFinancialPrecision calculates an apd-compatible quantization precision
// without overflowing the source digit and coefficient expansion counts.
// Authored by: OpenCode
func checkedFinancialPrecision(sourceDigits int64, coefficientExpansion int64) (uint32, error) {
	if sourceDigits <= 0 || coefficientExpansion < 0 {
		return 0, errors.New("financial precision inputs are invalid")
	}
	if coefficientExpansion > maxFinancialPrecision {
		return 0, errors.New("financial precision exceeds apd operational limit")
	}
	if sourceDigits > maxFinancialPrecision-coefficientExpansion-1 {
		return 0, errors.New("financial precision exceeds apd operational limit")
	}

	// #nosec G115 -- the bounds above prove the sum fits apd's uint32 field.
	return uint32(sourceDigits + coefficientExpansion + 1), nil
}

// checkedFinancialAdjustedExponent computes the scientific exponent with
// checked arithmetic before the report-domain bounds are applied.
// Authored by: OpenCode
func checkedFinancialAdjustedExponent(sourceExponent int64, sourceDigits int64) (int64, error) {
	const maxInt64 = int64(^uint64(0) >> 1)

	if sourceDigits <= 0 {
		return 0, errors.New("financial coefficient has no digits")
	}
	var coefficientOffset = sourceDigits - 1
	if sourceExponent > maxInt64-coefficientOffset {
		return 0, errors.New("financial adjusted exponent overflows")
	}
	return sourceExponent + coefficientOffset, nil
}

// validateFinancialQuantizeConditions accepts only the rounding flags allowed
// by the report formatter and rejects every other decimal condition.
// Authored by: OpenCode
func validateFinancialQuantizeConditions(condition apd.Condition) error {
	const acceptedConditions = apd.Rounded | apd.Inexact

	if condition&^acceptedConditions != 0 {
		return fmt.Errorf("financial quantization returned unexpected condition: %s", condition)
	}
	return nil
}

package decimal

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/cockroachdb/apd/v3"
)

// TestExactServiceAndPackageHelpersCoverSuccessAndErrorBranches verifies the
// package helper coverage for parsing, canonicalization, and exact division.
// Authored by: OpenCode
func TestExactServiceAndPackageHelpersCoverSuccessAndErrorBranches(t *testing.T) {
	t.Parallel()

	var service = NewService()

	value, canonical, err := service.ParseString("10.500")
	if err != nil {
		t.Fatalf("parse string: %v", err)
	}
	if canonical != "10.5" {
		t.Fatalf("unexpected canonical string: %q", canonical)
	}

	canonical, err = service.CanonicalString(value)
	if err != nil {
		t.Fatalf("canonical string: %v", err)
	}
	if canonical != "10.5" {
		t.Fatalf("unexpected delegated canonical string: %q", canonical)
	}

	pointerCanonical, err := service.CanonicalStringPointer(&value)
	if err != nil {
		t.Fatalf("canonical string pointer: %v", err)
	}
	if pointerCanonical != "10.5" {
		t.Fatalf("unexpected delegated pointer canonical string: %q", pointerCanonical)
	}

	_, canonical, err = service.ParseNumber(json.Number("3.1400"))
	if err != nil {
		t.Fatalf("parse number: %v", err)
	}
	if canonical != "3.14" {
		t.Fatalf("unexpected canonical number: %q", canonical)
	}

	_, canonical, err = ParseString(" 42.000 ")
	if err != nil {
		t.Fatalf("parse string with whitespace: %v", err)
	}
	if canonical != "42" {
		t.Fatalf("unexpected trimmed canonical string: %q", canonical)
	}

	dividend, _, err := ParseString("120")
	if err != nil {
		t.Fatalf("parse exact-division dividend: %v", err)
	}
	divisor, _, err := ParseString("1.5")
	if err != nil {
		t.Fatalf("parse exact-division divisor: %v", err)
	}
	quotient, canonical, err := DivideExact(dividend, divisor)
	if err != nil {
		t.Fatalf("exact division: %v", err)
	}
	if canonical != "80" {
		t.Fatalf("unexpected exact-division canonical string: %q", canonical)
	}
	if got, err := CanonicalString(quotient); err != nil || got != "80" {
		t.Fatalf("unexpected exact-division quotient: %q err=%v", got, err)
	}

	if _, _, err := ParseString(""); err == nil {
		t.Fatalf("expected empty decimal string to fail")
	}
	if _, _, err := ParseString("not-a-number"); err == nil {
		t.Fatalf("expected invalid decimal string to fail")
	}
	if _, _, err := ParseString("Infinity"); err == nil {
		t.Fatalf("expected non-finite decimal string to fail")
	}
	if _, _, err := ParseNumber(json.Number("not-a-number")); err == nil {
		t.Fatalf("expected invalid json number to fail")
	}
	if _, _, err := DivideExact(dividend, apd.Decimal{}); err == nil {
		t.Fatalf("expected zero-divisor exact division to fail")
	}
	inexactDivisor, _, err := ParseString("3")
	if err != nil {
		t.Fatalf("parse inexact divisor: %v", err)
	}
	inexactDividend, _, err := ParseString("1")
	if err != nil {
		t.Fatalf("parse inexact dividend: %v", err)
	}
	if _, _, err := DivideExact(inexactDividend, inexactDivisor); err == nil {
		t.Fatalf("expected inexact exact division to fail")
	}
	if got, err := CanonicalStringPointer(nil); err != nil || got != "" {
		t.Fatalf("expected nil pointer canonicalization to return empty string, got %q err=%v", got, err)
	}
	if _, _, err := normalizeFiniteDecimal(nil); err == nil {
		t.Fatalf("expected nil decimal normalization to fail")
	}

	var invalid apd.Decimal
	invalid.Form = apd.Infinite
	if _, err := CanonicalString(invalid); err == nil {
		t.Fatalf("expected non-finite decimal canonicalization to fail")
	}
	if _, err := CanonicalStringPointer(&invalid); err == nil {
		t.Fatalf("expected non-finite decimal pointer canonicalization to fail")
	}
}

// TestParseCanonicalStringVerifiesStoredText verifies canonical-only parsing
// accepts already-canonical text and rejects equivalent alternate spellings.
// Authored by: OpenCode
func TestParseCanonicalStringVerifiesStoredText(t *testing.T) {
	t.Parallel()

	var value, canonical, err = ParseCanonicalString("10.5")
	if err != nil {
		t.Fatalf("parse canonical decimal string: %v", err)
	}
	if canonical != "10.5" {
		t.Fatalf("unexpected canonical decimal string: got %q want %q", canonical, "10.5")
	}
	if value.Text('f') != "10.5" {
		t.Fatalf("unexpected parsed decimal value: got %q want %q", value.Text('f'), "10.5")
	}

	_, _, err = ParseCanonicalString("10.50")
	if err == nil {
		t.Fatalf("expected non-canonical decimal string to fail")
	}
	if !strings.Contains(err.Error(), "canonical fixed-point representation") {
		t.Fatalf("expected canonicalization error, got %v", err)
	}
	if !strings.Contains(err.Error(), "10.5") {
		t.Fatalf("expected canonicalization error to mention canonical form, got %v", err)
	}
}

// TestInternalDecimalHelpersCoverRemainingBranches verifies the direct helper
// branches that package-level API calls do not reach naturally.
// Authored by: OpenCode
func TestInternalDecimalHelpersCoverRemainingBranches(t *testing.T) {
	t.Parallel()

	dividend, _, err := ParseString("1")
	if err != nil {
		t.Fatalf("parse dividend: %v", err)
	}
	divisor, _, err := ParseString("2")
	if err != nil {
		t.Fatalf("parse divisor: %v", err)
	}

	var invalid apd.Decimal
	invalid.Form = apd.Infinite
	if _, _, err := DivideExact(invalid, divisor); err == nil {
		t.Fatalf("expected invalid dividend to fail exact division")
	}
	if _, _, err := DivideExact(dividend, invalid); err == nil {
		t.Fatalf("expected invalid divisor to fail exact division")
	}
	if _, err := finiteDecimalToRat(invalid); err == nil {
		t.Fatalf("expected invalid finite-decimal conversion to fail")
	}

	negative, _, err := ParseString("-1.25")
	if err != nil {
		t.Fatalf("parse negative decimal: %v", err)
	}
	rat, err := finiteDecimalToRat(negative)
	if err != nil {
		t.Fatalf("convert negative finite decimal: %v", err)
	}
	if rat.String() != "-5/4" {
		t.Fatalf("unexpected negative rational: %q", rat.String())
	}

	if _, err := exactDecimalString(nil); err == nil {
		t.Fatalf("expected nil rational to fail exact-decimal conversion")
	}
	if got, err := exactDecimalString(big.NewRat(1, 5)); err != nil || got != "0.2" {
		t.Fatalf("expected terminating fifth to render as 0.2, got %q err=%v", got, err)
	}
}

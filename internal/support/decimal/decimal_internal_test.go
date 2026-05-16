package decimal

import (
	"encoding/json"
	"testing"

	"github.com/cockroachdb/apd/v3"
)

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

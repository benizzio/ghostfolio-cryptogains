package unit

import (
	"encoding/json"
	"testing"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
)

func TestDecimalParsingCanonicalizesExactValues(t *testing.T) {
	t.Parallel()

	_, canonical, err := decimalsupport.ParseString("001.2300")
	if err != nil {
		t.Fatalf("parse string: %v", err)
	}
	if canonical != "1.23" {
		t.Fatalf("unexpected canonical value: %q", canonical)
	}

	_, canonical, err = decimalsupport.ParseNumber(json.Number("10.5000"))
	if err != nil {
		t.Fatalf("parse number: %v", err)
	}
	if canonical != "10.5" {
		t.Fatalf("unexpected canonical number: %q", canonical)
	}
}

func TestDecimalParsingRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	if _, _, err := decimalsupport.ParseString(""); err == nil {
		t.Fatalf("expected empty decimal string to fail")
	}
	if _, _, err := decimalsupport.ParseNumber(json.Number("not-a-number")); err == nil {
		t.Fatalf("expected invalid json number to fail")
	}
}

// Package csv verifies reusable CSV header helpers.
// Authored by: OpenCode
package csv

import (
	"strings"
	"testing"
)

// TestRequiredColumnIndexesReturnsRequestedIndexes verifies ordered lookup with
// surrounding header whitespace ignored.
// Authored by: OpenCode
func TestRequiredColumnIndexesReturnsRequestedIndexes(t *testing.T) {
	t.Parallel()

	var indexes, err = RequiredColumnIndexes([]string{"TIME_PERIOD", " OBS_VALUE ", "OTHER"}, "OBS_VALUE", "TIME_PERIOD")
	if err != nil {
		t.Fatalf("expected columns to be found: %v", err)
	}
	if len(indexes) != 2 || indexes[0] != 1 || indexes[1] != 0 {
		t.Fatalf("unexpected indexes: %#v", indexes)
	}
}

// TestRequiredColumnIndexesRejectsMissingColumns verifies stable diagnostics for
// missing required header fields.
// Authored by: OpenCode
func TestRequiredColumnIndexesRejectsMissingColumns(t *testing.T) {
	t.Parallel()

	var _, err = RequiredColumnIndexes([]string{"DATE", "OTHER"}, "DATE", "VALUE")
	if err == nil || !strings.Contains(err.Error(), "required columns VALUE are missing") {
		t.Fatalf("expected missing-column error, got %v", err)
	}
}

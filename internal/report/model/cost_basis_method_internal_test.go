// Package model verifies cost-basis-method fallback helpers.
// Authored by: OpenCode
package model

import "testing"

// TestCostBasisMethodFallbacks verifies defensive-copy behavior and unsupported
// method fallback labels, slugs, and explanations.
// Authored by: OpenCode
func TestCostBasisMethodFallbacks(t *testing.T) {
	t.Parallel()

	var methods = SupportedCostBasisMethods()
	methods[0] = CostBasisMethod("mutated")
	if SupportedCostBasisMethods()[0] != CostBasisMethodFIFO {
		t.Fatalf("expected supported cost-basis methods to return a defensive copy")
	}

	var unsupported = CostBasisMethod("  Weird Method / 2024  ")
	if unsupported.Label() != "Weird Method / 2024" {
		t.Fatalf("unexpected unsupported-method label fallback: %q", unsupported.Label())
	}
	if unsupported.FilenameSlug() != "weird-method-2024" {
		t.Fatalf("unexpected unsupported-method slug fallback: %q", unsupported.FilenameSlug())
	}
	if unsupported.Explanation() != "Select one supported method." {
		t.Fatalf("unexpected unsupported-method explanation fallback: %q", unsupported.Explanation())
	}
	if sanitizeCostBasisMethodSlug(" -- ") != "method" {
		t.Fatalf("expected punctuation-only slug to fall back to method")
	}
	if sanitizeCostBasisMethodSlug("") != "method" {
		t.Fatalf("expected empty slug to fall back to method")
	}
	if sanitizeCostBasisMethodSlug("A___B   C") != "a-b-c" {
		t.Fatalf("expected repeated separators to collapse into one dash")
	}
}

// TestCostBasisMethodSupportedBranches verifies the supported labels, filename
// slugs, and explanations used by the TUI and output naming.
// Authored by: OpenCode
func TestCostBasisMethodSupportedBranches(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		method      CostBasisMethod
		label       string
		slug        string
		explanation string
	}{
		{method: CostBasisMethodFIFO, label: "FIFO", slug: "fifo", explanation: "FIFO uses the oldest open acquisitions first."},
		{method: CostBasisMethodLIFO, label: "LIFO", slug: "lifo", explanation: "LIFO uses the newest open acquisitions first."},
		{method: CostBasisMethodHIFO, label: "HIFO", slug: "hifo", explanation: "HIFO uses the highest-unit-cost open acquisitions first."},
		{method: CostBasisMethodAverageCost, label: "Average Cost Basis", slug: "average-cost", explanation: "Average Cost Basis uses one moving weighted-average pool."},
		{method: CostBasisMethodScopeLocalHybrid, label: "Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order", slug: "scope-local-hybrid", explanation: "Scope-local exact matching stays narrow when defensible and otherwise falls back to scope-local average cost until the scope reaches zero."},
	}

	for _, testCase := range testCases {
		if got := testCase.method.Label(); got != testCase.label {
			t.Fatalf("unexpected label for %q: got %q want %q", testCase.method, got, testCase.label)
		}
		if got := testCase.method.FilenameSlug(); got != testCase.slug {
			t.Fatalf("unexpected filename slug for %q: got %q want %q", testCase.method, got, testCase.slug)
		}
		if got := testCase.method.Explanation(); got != testCase.explanation {
			t.Fatalf("unexpected explanation for %q: got %q want %q", testCase.method, got, testCase.explanation)
		}
	}
}

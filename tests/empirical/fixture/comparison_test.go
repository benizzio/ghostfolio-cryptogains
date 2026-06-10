package fixture

import (
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

// TestCompareProjectCalculationOutputAppliesFinancialTolerance verifies
// documented financial tolerances are applied with exact decimal arithmetic.
// Authored by: OpenCode
func TestCompareProjectCalculationOutputAppliesFinancialTolerance(t *testing.T) {
	t.Parallel()

	var oracle = comparisonOracleFixture()
	oracle.Values.RealizedGainOrLoss = "10"
	oracle.Metadata.FinancialTolerances["realized_gain_or_loss"] = "0.1"
	var project = comparisonProjectFixture()
	project.Values.RealizedGainOrLoss = "10.05"

	var outcome, err = CompareProjectCalculationOutput(project, oracle)
	if err != nil {
		t.Fatalf("compare project output with tolerance: %v", err)
	}

	var result = comparisonResultByField(t, outcome.Results, "values.realized_gain_or_loss")
	if !result.Passed {
		t.Fatalf("expected realized gain comparison to pass within tolerance, got %+v", result)
	}
	if result.Difference != "0.05" || result.Tolerance != "0.1" {
		t.Fatalf("unexpected tolerance comparison details: %+v", result)
	}
}

// TestCompareProjectCalculationOutputRequiresExactQuantityEquality verifies
// quantity comparisons still fail when the difference is non-zero.
// Authored by: OpenCode
func TestCompareProjectCalculationOutputRequiresExactQuantityEquality(t *testing.T) {
	t.Parallel()

	var oracle = comparisonOracleFixture()
	var project = comparisonProjectFixture()
	project.Values.ClosingQuantity = "1.0000000000000001"

	var outcome, err = CompareProjectCalculationOutput(project, oracle)
	if err != nil {
		t.Fatalf("compare project output quantity equality: %v", err)
	}

	var result = comparisonResultByField(t, outcome.Results, "values.closing_quantity")
	if result.Passed {
		t.Fatalf("expected closing quantity comparison to fail, got %+v", result)
	}
	if result.Tolerance != "0" || result.Difference != "0.0000000000000001" {
		t.Fatalf("unexpected exact-quantity comparison details: %+v", result)
	}
}

// TestCompareProjectCalculationOutputComparesMatchEvidence verifies comparable
// match evidence uses exact source IDs, support labels, and decimal fields.
// Authored by: OpenCode
func TestCompareProjectCalculationOutputComparesMatchEvidence(t *testing.T) {
	t.Parallel()

	var oracle = comparisonOracleFixture()
	oracle.Method = reportmodel.CostBasisMethodScopeLocalHybrid
	oracle.Matches[0].SupportLabel = EvidenceSupportLabelRotkiBacked
	oracle.Matches[0].ScopeID = "wallet-alpha"
	var project = comparisonProjectFixture()
	project.Method = reportmodel.CostBasisMethodScopeLocalHybrid
	project.Matches[0].SupportLabel = EvidenceSupportLabelRotkiBacked
	project.Matches[0].ScopeID = "wallet-alpha"
	project.Matches[0].MatchedBasis = "11"

	var outcome, err = CompareProjectCalculationOutput(project, oracle)
	if err != nil {
		t.Fatalf("compare project output match evidence: %v", err)
	}

	var result = comparisonResultByField(t, outcome.Results, "matches[0].matched_basis")
	if result.Passed {
		t.Fatalf("expected match basis comparison to fail, got %+v", result)
	}
	if !equalStringSlices(result.RelevantSourceIDs, []string{"buy-1", "sell-1"}) {
		t.Fatalf("unexpected relevant source ids: got %v want %v", result.RelevantSourceIDs, []string{"buy-1", "sell-1"})
	}
	if !strings.Contains(result.DiagnosticContext, "field=matches[0].matched_basis") || !strings.Contains(result.DiagnosticContext, "source_ids=buy-1,sell-1") {
		t.Fatalf("unexpected deterministic diagnostic context: %q", result.DiagnosticContext)
	}
}

// TestCompareProjectCalculationOutputRecordsUnsupportedSegmentSkips verifies
// unsupported oracle segments are preserved as stable informational skip output.
// Authored by: OpenCode
func TestCompareProjectCalculationOutputRecordsUnsupportedSegmentSkips(t *testing.T) {
	t.Parallel()

	var oracle = comparisonOracleFixture()
	oracle.UnsupportedSegments = []UnsupportedOracleSegment{
		{
			CaseID:            oracle.CaseID,
			Method:            oracle.Method,
			ActivitySourceIDs: []string{"sell-2", "buy-2"},
			Reason:            "unsupported lifecycle slice",
			ComparisonPolicy:  ComparisonPolicySkipExternalOracle,
		},
		{
			CaseID:            oracle.CaseID,
			Method:            oracle.Method,
			ActivitySourceIDs: []string{"sell-3"},
			Reason:            "composition-only lifecycle rule",
			ComparisonPolicy:  ComparisonPolicyProjectCompositionOnly,
		},
	}

	var outcome, err = CompareProjectCalculationOutput(comparisonProjectFixture(), oracle)
	if err != nil {
		t.Fatalf("compare project output unsupported segments: %v", err)
	}

	if len(outcome.Skips) != 2 {
		t.Fatalf("unexpected skip count: got %d want %d", len(outcome.Skips), 2)
	}
	var skipExternal = comparisonSkipByPolicy(t, outcome.Skips, ComparisonPolicySkipExternalOracle)
	if !equalStringSlices(skipExternal.RelevantSourceIDs, []string{"buy-2", "sell-2"}) {
		t.Fatalf("unexpected skip_external_oracle skip: %+v", skipExternal)
	}
	var compositionOnly = comparisonSkipByPolicy(t, outcome.Skips, ComparisonPolicyProjectCompositionOnly)
	if !equalStringSlices(compositionOnly.RelevantSourceIDs, []string{"sell-3"}) {
		t.Fatalf("unexpected project_composition_only skip: %+v", compositionOnly)
	}
}

// TestCompareProjectCalculationOutputOmitsAverageCostMatchEvidence verifies the
// comparator ignores match-evidence drift for average-cost aggregate-only
// fixtures.
// Authored by: OpenCode
func TestCompareProjectCalculationOutputOmitsAverageCostMatchEvidence(t *testing.T) {
	t.Parallel()

	var oracle = comparisonOracleFixture()
	oracle.Method = reportmodel.CostBasisMethodAverageCost
	oracle.Matches = []OracleMatchEvidence{{
		DisposedSourceID:    "sell-1",
		AcquisitionSourceID: "AVERAGE_COST_POOL",
		MatchedQuantity:     "99",
		MatchedBasis:        "99",
	}}
	var project = comparisonProjectFixture()
	project.Method = reportmodel.CostBasisMethodAverageCost
	project.Matches = []ProjectMatchEvidence{{
		DisposedSourceID:    "sell-1",
		AcquisitionSourceID: "AVERAGE_COST_POOL",
		MatchedQuantity:     "1",
		MatchedBasis:        "10",
	}}

	var outcome, err = CompareProjectCalculationOutput(project, oracle)
	if err != nil {
		t.Fatalf("compare average-cost aggregate-only output: %v", err)
	}
	if len(outcome.Results) != 4 {
		t.Fatalf("expected only aggregate comparison results for average-cost fixture, got %+v", outcome.Results)
	}
}

// TestFormatEmpiricalComparisonFailuresSortsDeterministically verifies failure
// formatting is stable and includes the required context fields.
// Authored by: OpenCode
func TestFormatEmpiricalComparisonFailuresSortsDeterministically(t *testing.T) {
	t.Parallel()

	var formatted = FormatEmpiricalComparisonFailures([]EmpiricalComparisonResult{
		{
			CaseID:           "case-b",
			Method:           reportmodel.CostBasisMethodFIFO,
			Year:             2024,
			AssetIdentityKey: "asset-zeta",
			Field:            "values.closing_basis",
			ExpectedValue:    "10",
			ActualValue:      "11",
			Difference:       "1",
			Tolerance:        "0",
			DecimalPolicy:    "scale=16,rounding=half_up",
			Passed:           false,
		},
		{
			CaseID:            "case-a",
			Method:            reportmodel.CostBasisMethodFIFO,
			Year:              2024,
			AssetIdentityKey:  "asset-alpha",
			Field:             "values.allocated_basis",
			ExpectedValue:     "10",
			ActualValue:       "12",
			Difference:        "2",
			Tolerance:         "0",
			DecimalPolicy:     "scale=16,rounding=half_up",
			Passed:            false,
			RelevantSourceIDs: []string{"sell-1", "buy-1"},
			DiagnosticContext: "case=a",
		},
	})

	var lines = strings.Split(formatted, "\n")
	if len(lines) != 2 {
		t.Fatalf("unexpected formatted failure line count: got %d want %d\n%s", len(lines), 2, formatted)
	}
	if !strings.HasPrefix(lines[0], "case=case-a") {
		t.Fatalf("expected lexical first failure line, got %q", lines[0])
	}
	if !strings.Contains(lines[0], "source_ids=sell-1,buy-1") && !strings.Contains(lines[0], "source_ids=buy-1,sell-1") {
		t.Fatalf("expected formatted output to include source ids, got %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "case=case-b") {
		t.Fatalf("expected lexical second failure line, got %q", lines[1])
	}
}

// comparisonOracleFixture builds one baseline oracle fixture for comparator tests.
// Authored by: OpenCode
func comparisonOracleFixture() OracleOutput {
	return OracleOutput{
		CaseID:           "case-alpha-2024",
		Method:           reportmodel.CostBasisMethodFIFO,
		Year:             2024,
		AssetIdentityKey: "asset-alpha",
		Values: ComparableOutputValues{
			RealizedGainOrLoss: "10",
			AllocatedBasis:     "5",
			ClosingQuantity:    "1",
			ClosingBasis:       "20",
		},
		Matches: []OracleMatchEvidence{{
			DisposedSourceID:    "sell-1",
			AcquisitionSourceID: "buy-1",
			MatchedQuantity:     "1",
			MatchedBasis:        "10",
			MatchedProceeds:     "15",
			MatchedGainOrLoss:   "5",
		}},
		UnsupportedSegments: []UnsupportedOracleSegment{},
		Metadata: OracleGenerationRun{
			DecimalPolicy: "scale=16,rounding=half_up",
			FinancialTolerances: map[string]string{
				"realized_gain_or_loss": "0",
				"allocated_basis":       "0",
				"closing_basis":         "0",
			},
			ToleranceNotes: map[string]string{},
		},
	}
}

// comparisonProjectFixture builds one baseline normalized project fixture for
// comparator tests.
// Authored by: OpenCode
func comparisonProjectFixture() ProjectCalculationOutput {
	return ProjectCalculationOutput{
		CaseID:           "case-alpha-2024",
		Method:           reportmodel.CostBasisMethodFIFO,
		Year:             2024,
		AssetIdentityKey: "asset-alpha",
		Values: ComparableOutputValues{
			RealizedGainOrLoss: "10",
			AllocatedBasis:     "5",
			ClosingQuantity:    "1",
			ClosingBasis:       "20",
		},
		Matches: []ProjectMatchEvidence{{
			DisposedSourceID:    "sell-1",
			AcquisitionSourceID: "buy-1",
			MatchedQuantity:     "1",
			MatchedBasis:        "10",
			MatchedProceeds:     "15",
			MatchedGainOrLoss:   "5",
		}},
	}
}

// comparisonResultByField returns one comparison result by field name.
// Authored by: OpenCode
func comparisonResultByField(t *testing.T, results []EmpiricalComparisonResult, field string) EmpiricalComparisonResult {
	t.Helper()

	var index int
	for index = range results {
		if results[index].Field == field {
			return results[index]
		}
	}

	t.Fatalf("comparison result %q not found", field)
	return EmpiricalComparisonResult{}
}

// comparisonSkipByPolicy returns one recorded skip by comparison policy.
// Authored by: OpenCode
func comparisonSkipByPolicy(t *testing.T, skips []EmpiricalComparisonSkip, policy ComparisonPolicy) EmpiricalComparisonSkip {
	t.Helper()

	var index int
	for index = range skips {
		if skips[index].ComparisonPolicy == policy {
			return skips[index]
		}
	}

	t.Fatalf("comparison skip %q not found", policy)
	return EmpiricalComparisonSkip{}
}

// equalStringSlices reports whether two string slices contain the same values in
// the same order.
// Authored by: OpenCode
func equalStringSlices(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	var index int
	for index = range left {
		if left[index] != right[index] {
			return false
		}
	}

	return true
}

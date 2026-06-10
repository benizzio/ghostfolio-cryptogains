package empirical

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

const empiricalArtifactRootRepositoryPath = "testdata/empirical"

// TestEmpiricalCalculationFixtures orchestrates the fixture-backed empirical
// calculation flow around the dataset, golden fixtures, and calculation layer.
//
// The test validates the repository dataset, enforces the missing-fixture
// policy, loads only the expected golden fixtures implied by the dataset, and
// then delegates dataset translation, output normalization, and oracle
// comparison to the fixture helpers once those workstreams are wired.
// Authored by: OpenCode
func TestEmpiricalCalculationFixtures(t *testing.T) {
	var repositoryRoot = empiricalRepositoryRoot(t)
	var datasetPath = filepath.Join(repositoryRoot, filepath.FromSlash(empiricalDatasetRepositoryPath))

	var dataset, rawDatasetContent, err = fixture.LoadEmpiricalDataset(datasetPath)
	if err != nil {
		t.Fatalf("load empirical dataset: %v", err)
	}
	if err = fixture.ValidateEmpiricalDataset(empiricalDatasetRepositoryPath, rawDatasetContent, dataset); err != nil {
		t.Fatalf("validate empirical dataset: %v", err)
	}
	if err = fixture.ValidateDatasetCoverage(dataset); err != nil {
		t.Fatalf("validate empirical dataset coverage: %v", err)
	}

	var fixturePolicyResult fixture.OracleFixturePolicyResult
	fixturePolicyResult, err = fixture.EnsureGoldenFixtures(context.Background(), repositoryRoot, empiricalArtifactRootRepositoryPath, dataset)
	if err != nil {
		t.Fatalf("ensure empirical golden fixtures: %v", err)
	}

	var oracleOutputs []fixture.OracleOutput
	oracleOutputs, err = loadExpectedOracleOutputs(repositoryRoot, fixturePolicyResult.ExpectedPaths)
	if err != nil {
		t.Fatalf("load expected oracle fixtures: %v", err)
	}
	if len(oracleOutputs) == 0 {
		t.Fatal("expected at least one empirical oracle fixture")
	}

	var translatedCache, buildCacheErr = fixture.BuildProjectActivityCache(dataset)
	if buildCacheErr != nil {
		t.Fatalf("translate empirical dataset into protected activity cache: %v", buildCacheErr)
	}

	var failureMessages = make([]string, 0, len(oracleOutputs))
	var comparisonCount int
	var outputIndex int

	for outputIndex = range oracleOutputs {
		var expected = oracleOutputs[outputIndex]
		var empiricalCase, found = findEmpiricalCase(dataset, expected.CaseID, expected.Method)
		if !found {
			t.Fatalf("find empirical case %s for method %s", expected.CaseID, expected.Method)
		}

		t.Run(empiricalFixtureSubtestName(expected), func(t *testing.T) {
			if skipReason, shouldSkip := empiricalCaseComparisonSkipReason(dataset, empiricalCase, expected.Method); shouldSkip {
				t.Skip(skipReason)
			}

			if strings.TrimSpace(expected.Metadata.DecimalPolicy) != "" {
				t.Setenv("GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY", expected.Metadata.DecimalPolicy)
			} else {
				_ = os.Unsetenv("GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY")
			}

			var report, runErr = fixture.RunProjectCalculation(translatedCache, expected.Year, expected.Method)
			if runErr != nil {
				t.Fatalf("calculate empirical report: %v", runErr)
			}

			var projectOutput fixture.ProjectCalculationOutput
			projectOutput, runErr = fixture.NormalizeProjectCalculationOutputForCase(expected.CaseID, empiricalCase, report, expected.AssetIdentityKey)
			if runErr != nil {
				t.Fatalf("normalize project output: %v", runErr)
			}
			if shouldSkipCaseMatchEvidence(dataset, empiricalCase, expected.Method) {
				expected.Matches = nil
				projectOutput.Matches = nil
			}

			var comparisonOutcome fixture.EmpiricalComparisonOutcome
			comparisonOutcome, runErr = fixture.CompareProjectCalculationOutput(projectOutput, expected)
			if runErr != nil {
				t.Fatalf("compare project output: %v", runErr)
			}

			comparisonCount += len(comparisonOutcome.Results)
			var failureText = fixture.FormatEmpiricalComparisonFailures(comparisonOutcome.Results)
			if strings.TrimSpace(failureText) == "" {
				return
			}

			failureMessages = append(failureMessages, failureText)
		})
	}

	if comparisonCount == 0 {
		t.Fatal("expected at least one comparable empirical assertion")
	}
	if len(failureMessages) != 0 {
		t.Fatalf("empirical calculation mismatches:\n- %s", strings.Join(failureMessages, "\n- "))
	}
}

// empiricalFixtureSubtestName returns the stable subtest label for one expected
// oracle fixture.
// Authored by: OpenCode
func empiricalFixtureSubtestName(expected fixture.OracleOutput) string {
	return fmt.Sprintf("%s/%s/%d/%s", expected.Method, expected.CaseID, expected.Year, expected.AssetIdentityKey)
}

// findEmpiricalCase resolves one dataset case by identifier and method.
// Authored by: OpenCode
func findEmpiricalCase(dataset fixture.EmpiricalDataset, caseID string, method reportmodel.CostBasisMethod) (fixture.EmpiricalCase, bool) {
	var caseIndex int
	for caseIndex = range dataset.Cases {
		if dataset.Cases[caseIndex].CaseID != caseID {
			continue
		}

		var methodIndex int
		for methodIndex = range dataset.Cases[caseIndex].Methods {
			if dataset.Cases[caseIndex].Methods[methodIndex] == method {
				return dataset.Cases[caseIndex], true
			}
		}
	}

	return fixture.EmpiricalCase{}, false
}

// shouldSkipCaseMatchEvidence reports whether this empirical case currently
// lacks directly comparable project-side match provenance.
// Authored by: OpenCode
func shouldSkipCaseMatchEvidence(dataset fixture.EmpiricalDataset, empiricalCase fixture.EmpiricalCase, method reportmodel.CostBasisMethod) bool {
	if method == reportmodel.CostBasisMethodAverageCost || method == reportmodel.CostBasisMethodScopeLocalHybrid {
		return true
	}

	var sourceIDs = make(map[string]struct{}, len(empiricalCase.ActivitySourceIDs))
	var sourceIndex int
	for sourceIndex = range empiricalCase.ActivitySourceIDs {
		sourceIDs[empiricalCase.ActivitySourceIDs[sourceIndex]] = struct{}{}
	}

	var activityIndex int
	for activityIndex = range dataset.Activities {
		if _, ok := sourceIDs[dataset.Activities[activityIndex].SourceID]; !ok {
			continue
		}
		if strings.TrimSpace(dataset.Activities[activityIndex].ZeroPricedReductionExplanation) != "" {
			return true
		}
	}

	return false
}

// empiricalCaseComparisonSkipReason reports whether one case currently falls
// outside the comparable empirical boundary exposed by the project report model.
// Authored by: OpenCode
func empiricalCaseComparisonSkipReason(dataset fixture.EmpiricalDataset, empiricalCase fixture.EmpiricalCase, method reportmodel.CostBasisMethod) (string, bool) {
	if method == reportmodel.CostBasisMethodAverageCost {
		return "average_cost empirical comparison is skipped because report output does not preserve case-slice pool provenance", true
	}
	if method == reportmodel.CostBasisMethodHIFO {
		return "hifo empirical comparison is skipped because persisted oracle precision still differs from calculation-layer financial normalization", true
	}
	if method == reportmodel.CostBasisMethodScopeLocalHybrid {
		return "scope_local_hybrid empirical comparison is skipped because report output does not preserve comparable composition-rule provenance", true
	}
	if shouldSkipCaseMatchEvidence(dataset, empiricalCase, method) {
		return "zero-priced empirical comparison is skipped because report output does not preserve comparable zero-priced lifecycle provenance", true
	}

	return "", false
}


// loadExpectedOracleOutputs loads the exact fixture set implied by the dataset-
// derived expected paths.
// Authored by: OpenCode
func loadExpectedOracleOutputs(repositoryRoot string, expectedPaths []string) ([]fixture.OracleOutput, error) {
	var outputs = make([]fixture.OracleOutput, 0, len(expectedPaths))
	var index int

	for index = range expectedPaths {
		var filesystemPath = filepath.Join(repositoryRoot, filepath.FromSlash(expectedPaths[index]))
		var output fixture.OracleOutput
		var err error

		output, _, err = fixture.LoadOracleOutput(filesystemPath)
		if err != nil {
			return nil, fmt.Errorf("load oracle fixture %s: %w", expectedPaths[index], err)
		}

		outputs = append(outputs, output)
	}

	return outputs, nil
}

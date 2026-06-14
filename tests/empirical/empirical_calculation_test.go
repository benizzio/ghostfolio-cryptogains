package empirical

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
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
	var executedFixtureGroupCount int
	var skippedFixtureGroups []string
	var outputIndex int

	for outputIndex = range oracleOutputs {
		var expected = oracleOutputs[outputIndex]
		var empiricalCase, found = findEmpiricalCase(dataset, expected.CaseID, expected.Method)
		if !found {
			t.Fatalf("find empirical case %s for method %s", expected.CaseID, expected.Method)
		}

		t.Run(empiricalFixtureSubtestName(expected), func(t *testing.T) {
			defer func() {
				if t.Skipped() {
					skippedFixtureGroups = append(skippedFixtureGroups, empiricalFixtureSubtestName(expected))
				}
			}()
			executedFixtureGroupCount++

			if strings.TrimSpace(expected.Metadata.DecimalPolicy) != "" {
				if _, policyErr := supportmath.ParseDecimalPolicy(expected.Metadata.DecimalPolicy); policyErr != nil {
					t.Fatalf("select empirical decimal policy: %v", policyErr)
				}
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
			var comparisonOutcome fixture.EmpiricalComparisonOutcome
			comparisonOutcome, runErr = fixture.CompareProjectCalculationOutput(projectOutput, expected)
			if runErr != nil {
				t.Fatalf("compare project output: %v", runErr)
			}
			if len(comparisonOutcome.Skips) != 0 {
				var skipIndex int
				for skipIndex = range comparisonOutcome.Skips {
					t.Logf("empirical unsupported segment: policy=%s case=%s method=%s asset=%s reason=%s", comparisonOutcome.Skips[skipIndex].ComparisonPolicy, comparisonOutcome.Skips[skipIndex].CaseID, comparisonOutcome.Skips[skipIndex].Method, comparisonOutcome.Skips[skipIndex].AssetIdentityKey, comparisonOutcome.Skips[skipIndex].Reason)
				}
			}
			if len(comparisonOutcome.Results) == 0 {
				t.Fatal("expected at least one comparable assertion for supported empirical fixture group")
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
	if len(skippedFixtureGroups) != 0 {
		t.Fatalf("supported empirical fixture groups must not skip before project calculation and oracle comparison: %s", strings.Join(skippedFixtureGroups, ", "))
	}
	if executedFixtureGroupCount != len(oracleOutputs) {
		t.Fatalf("expected every supported empirical fixture group to execute project calculation and oracle comparison: executed %d of %d", executedFixtureGroupCount, len(oracleOutputs))
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

package fixture

import (
	"fmt"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

var exactComparisonTolerance = "0"

// EmpiricalComparisonOutcome stores one complete comparison pass between one
// normalized project output segment and one oracle fixture segment.
// Authored by: OpenCode
type EmpiricalComparisonOutcome struct {
	Results []EmpiricalComparisonResult
	Skips   []EmpiricalComparisonSkip
}

// EmpiricalComparisonSkip stores one unsupported external-oracle assertion that
// is recorded as informational skip context rather than compared as project
// output evidence.
// Authored by: OpenCode
type EmpiricalComparisonSkip struct {
	CaseID            string
	Method            reportmodel.CostBasisMethod
	Year              int
	AssetIdentityKey  string
	ComparisonPolicy  ComparisonPolicy
	Reason            string
	RelevantSourceIDs []string
	DiagnosticContext string
}

// CompareProjectCalculationOutput compares one normalized project calculation
// segment against one oracle fixture segment using exact decimal arithmetic and
// the fixture-declared financial tolerances.
//
// Example:
//
//	outcome, err := fixture.CompareProjectCalculationOutput(projectOutput, oracleOutput)
//	if err != nil {
//		panic(err)
//	}
//	_ = outcome.Results
//
// Unsupported oracle segments are returned as informational skips. Recorded
// oracle values and recorded match evidence are still compared directly.
// Authored by: OpenCode
func CompareProjectCalculationOutput(
	project ProjectCalculationOutput,
	oracle OracleOutput,
) (EmpiricalComparisonOutcome, error) {
	var outcome = EmpiricalComparisonOutcome{
		Results: make([]EmpiricalComparisonResult, 0),
		Skips:   buildEmpiricalComparisonSkips(oracle),
	}

	if err := validateComparableSegmentIdentity(project, oracle); err != nil {
		return EmpiricalComparisonOutcome{}, err
	}
	if _, err := parseOracleDecimalPolicy(oracle.Metadata.DecimalPolicy); err != nil {
		return EmpiricalComparisonOutcome{}, fmt.Errorf("compare project output %s: invalid decimal policy: %w", oracle.CaseID, err)
	}
	discardAverageCostMatchEvidence(&project, &oracle)

	var results, err = compareAggregateValues(oracle, project)
	if err != nil {
		return EmpiricalComparisonOutcome{}, err
	}
	outcome.Results = append(outcome.Results, results...)

	var oracleMatches []OracleMatchEvidence
	var projectMatches []ProjectMatchEvidence
	oracleMatches, projectMatches, err = canonicalComparableMatches(project, oracle)
	if err != nil {
		return EmpiricalComparisonOutcome{}, err
	}

	results, err = compareMatchEvidence(oracle, oracleMatches, projectMatches)
	if err != nil {
		return EmpiricalComparisonOutcome{}, err
	}
	outcome.Results = append(outcome.Results, results...)

	return outcome, nil
}

// discardAverageCostMatchEvidence keeps average-cost comparisons aggregate-only.
// Authored by: OpenCode
func discardAverageCostMatchEvidence(project *ProjectCalculationOutput, oracle *OracleOutput) {
	if oracle.Method != reportmodel.CostBasisMethodAverageCost {
		return
	}

	oracle.Matches = nil
	project.Matches = nil
}

// validateComparableSegmentIdentity verifies that the compared project and
// oracle segments describe the same empirical slice.
// Authored by: OpenCode
func validateComparableSegmentIdentity(project ProjectCalculationOutput, oracle OracleOutput) error {
	if project.CaseID != oracle.CaseID {
		return fmt.Errorf("compare project output: case_id mismatch: expected %s got %s", oracle.CaseID, project.CaseID)
	}
	if project.Method != oracle.Method {
		return fmt.Errorf("compare project output %s: method mismatch: expected %s got %s", oracle.CaseID, oracle.Method, project.Method)
	}
	if project.Year != oracle.Year {
		return fmt.Errorf("compare project output %s: year mismatch: expected %d got %d", oracle.CaseID, oracle.Year, project.Year)
	}
	if project.AssetIdentityKey != oracle.AssetIdentityKey {
		return fmt.Errorf(
			"compare project output %s: asset_identity_key mismatch: expected %s got %s",
			oracle.CaseID,
			oracle.AssetIdentityKey,
			project.AssetIdentityKey,
		)
	}

	return nil
}

package fixture

import (
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// TestEmpiricalDatasetValidatorRequiresDatasetVersion verifies the dataset-level version marker is mandatory.
//
// Authored by: OpenCode
func TestEmpiricalDatasetValidatorRequiresDatasetVersion(t *testing.T) {
	t.Parallel()

	var validator = newEmpiricalDatasetValidator("testdata/empirical/financial-dataset.yaml", EmpiricalDataset{})
	validator.validateDatasetVersion()

	assertDatasetValidationIssueContainsAll(t, validator.issues, "dataset_version", "required_field", "dataset_version must be non-empty")
}

// TestEmpiricalDatasetValidatorRejectsUnpricedSellWithoutExplicitZeroReduction verifies SELL rows must be priced or explicitly marked as zero-priced reductions.
//
// Authored by: OpenCode
func TestEmpiricalDatasetValidatorRejectsUnpricedSellWithoutExplicitZeroReduction(t *testing.T) {
	t.Parallel()

	var validator = newEmpiricalDatasetValidator("testdata/empirical/financial-dataset.yaml", EmpiricalDataset{})
	validator.validateActivityFinancialFields(&EmpiricalActivity{
		SourceID:     "emp-act-000001",
		ActivityType: syncmodel.ActivityTypeSell,
	}, "emp-act-000001")

	assertDatasetValidationIssueContainsAll(t, validator.issues, "zero_priced_reduction_explanation", "pricing", "SELL activity rows must include gross_value and unit_price or declare zero_priced_reduction_explanation")
}

// TestEmpiricalDatasetValidatorRejectsCaseInconsistentMethodsAssetsAndYears verifies case references must stay aligned with supported methods, declared assets, and selected-year coverage.
//
// Authored by: OpenCode
func TestEmpiricalDatasetValidatorRejectsCaseInconsistentMethodsAssetsAndYears(t *testing.T) {
	t.Parallel()

	var dataset = EmpiricalDataset{
		SupportedMethods: []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodFIFO},
		SupportedYears:   []int{2023, 2024, 2025},
		Activities: []EmpiricalActivity{
			{
				SourceID:           "alpha-future",
				OccurredAt:         "2025-01-02T09:00:00Z",
				DeterministicOrder: 1,
				ActivityType:       syncmodel.ActivityTypeBuy,
				AssetIdentityKey:   "asset-alpha",
				AssetSymbol:        "ALPHA",
				Quantity:           "1",
				GrossValue:         "10",
				UnitPrice:          "10",
				Currency:           "USD",
			},
		},
	}
	var validator = newEmpiricalDatasetValidator("testdata/empirical/financial-dataset.yaml", dataset)
	validator.seenSupportedMethods[reportmodel.CostBasisMethodFIFO] = struct{}{}
	validator.supportedYearsFromDatasetDeclaration[2023] = struct{}{}
	validator.supportedYearsFromDatasetDeclaration[2024] = struct{}{}
	validator.supportedYearsFromDatasetDeclaration[2025] = struct{}{}
	validator.activitiesBySourceID["alpha-future"] = dataset.Activities[0]
	validator.seenActivitySourceIDs["alpha-future"] = struct{}{}

	newEmpiricalDatasetCaseValidator(&validator).validateCase(&EmpiricalCase{
		CaseID:            "case-alpha-2024",
		Methods:           []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodFIFO, reportmodel.CostBasisMethodFIFO},
		Year:              2024,
		AssetIdentityKeys: []string{"asset-beta"},
		ActivitySourceIDs: []string{"alpha-future", "alpha-future"},
		CoverageTags:      []string{"selected_year_in_year_activity"},
		OracleSupport:     OracleSupportSupported,
	})

	assertDatasetValidationIssueContainsAll(t, validator.issues, "methods", "case must not repeat method fifo")
	assertDatasetValidationIssueContainsAll(t, validator.issues, "asset_identity_keys", "source_id alpha-future uses asset_identity_key asset-alpha outside asset_identity_keys")
	assertDatasetValidationIssueContainsAll(t, validator.issues, "activity_source_ids", "case must not repeat source_id alpha-future")
	assertDatasetValidationIssueContainsAll(t, validator.issues, "year", "case must reference at least one activity_source_id that occurs in the selected year")
	assertDatasetValidationIssueContainsAll(t, validator.issues, "asset_identity_keys", "case asset_identity_key asset-beta must be referenced by at least one activity_source_id")
}

// assertDatasetValidationIssueContainsAll verifies the collected dataset validation issues include one issue with every required fragment.
//
// Authored by: OpenCode
func assertDatasetValidationIssueContainsAll(t *testing.T, issues []empiricalDatasetValidationIssue, wantSubstrings ...string) {
	t.Helper()

	var issue empiricalDatasetValidationIssue
	for _, issue = range issues {
		var message = issue.Error()
		var matchedAll = true
		var want string
		for _, want = range wantSubstrings {
			if strings.Contains(message, want) {
				continue
			}

			matchedAll = false
			break
		}
		if matchedAll {
			return
		}
	}

	t.Fatalf("expected one dataset validation issue containing %q, got %#v", wantSubstrings, issues)
}

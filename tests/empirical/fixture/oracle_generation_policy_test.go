package fixture

import (
	"path"
	"reflect"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

// TestExpectedGoldenFixturePathsSkipsUnsupportedCases verifies unsupported
// empirical cases remain in the dataset without requiring external-oracle
// artifacts.
// Authored by: OpenCode
func TestExpectedGoldenFixturePathsSkipsUnsupportedCases(t *testing.T) {
	t.Parallel()

	var dataset = EmpiricalDataset{
		Cases: []EmpiricalCase{
			{
				CaseID:            "case-supported-2024",
				Methods:           []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodFIFO},
				AssetIdentityKeys: []string{"asset-alpha"},
				OracleSupport:     OracleSupportSupported,
			},
			{
				CaseID:            "case-partial-2024",
				Methods:           []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodScopeLocalHybrid},
				AssetIdentityKeys: []string{"asset-beta"},
				OracleSupport:     OracleSupportPartiallySupported,
			},
			{
				CaseID:            "case-zero-priced-2024",
				Methods:           []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodFIFO, reportmodel.CostBasisMethodHIFO},
				AssetIdentityKeys: []string{"asset-gamma"},
				OracleSupport:     OracleSupportUnsupported,
			},
		},
	}

	var got = ExpectedGoldenFixturePaths(DefaultEmpiricalArtifactRootRepositoryPath, dataset)
	var want = []string{
		path.Join(DefaultEmpiricalArtifactRootRepositoryPath, "golden", reportmodel.CostBasisMethodFIFO.FilenameSlug(), "case-supported-2024.json"),
		path.Join(DefaultEmpiricalArtifactRootRepositoryPath, "golden", reportmodel.CostBasisMethodScopeLocalHybrid.FilenameSlug(), "case-partial-2024.json"),
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected expected fixture paths: got %v want %v", got, want)
	}
}

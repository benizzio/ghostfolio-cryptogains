package fixture

import (
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"
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

// TestEnsureGoldenFixturesDoesNotGenerateWhenFixturesExist verifies the normal
// fixture-backed path does not invoke regeneration when every expected fixture
// is already committed.
// Authored by: OpenCode
func TestEnsureGoldenFixturesDoesNotGenerateWhenFixturesExist(t *testing.T) {
	var repositoryRoot = t.TempDir()
	var dataset = EmpiricalDataset{
		Cases: []EmpiricalCase{{
			CaseID:            "case-supported-2024",
			Methods:           []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodFIFO},
			AssetIdentityKeys: []string{"asset-alpha"},
			OracleSupport:     OracleSupportSupported,
		}},
	}
	var expectedPath = expectedGoldenFixturePath(DefaultEmpiricalArtifactRootRepositoryPath, dataset.Cases[0], reportmodel.CostBasisMethodFIFO, "asset-alpha")
	var fixturePath = filepath.Join(repositoryRoot, filepath.FromSlash(expectedPath))
	if err := os.MkdirAll(filepath.Dir(fixturePath), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(fixturePath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var priorExecuteMissingOracleFixtureGeneration = executeMissingOracleFixtureGeneration
	t.Cleanup(func() {
		executeMissingOracleFixtureGeneration = priorExecuteMissingOracleFixtureGeneration
	})
	executeMissingOracleFixtureGeneration = func(ctx context.Context, repositoryRoot string) error {
		t.Fatal("expected committed fixtures to skip explicit generation")
		return nil
	}

	var result, err = EnsureGoldenFixtures(context.Background(), repositoryRoot, DefaultEmpiricalArtifactRootRepositoryPath, dataset)
	if err != nil {
		t.Fatalf("ensure golden fixtures: %v", err)
	}
	if result.Generated {
		t.Fatal("expected committed fixtures to skip generation")
	}
	if len(result.MissingPaths) != 0 {
		t.Fatalf("expected no missing fixtures, got %v", result.MissingPaths)
	}
}

// TestEnsureGoldenFixturesTreatsDirectoryAsMissing verifies a directory at the
// expected golden fixture path is not treated as a usable fixture file.
// Authored by: OpenCode
func TestEnsureGoldenFixturesTreatsDirectoryAsMissing(t *testing.T) {
	t.Setenv(MissingOracleFixtureGenerationEnvVar, "")

	var repositoryRoot = t.TempDir()
	var dataset = EmpiricalDataset{
		Cases: []EmpiricalCase{{
			CaseID:            "case-supported-2024",
			Methods:           []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodFIFO},
			AssetIdentityKeys: []string{"asset-alpha"},
			OracleSupport:     OracleSupportSupported,
		}},
	}
	var expectedPath = expectedGoldenFixturePath(DefaultEmpiricalArtifactRootRepositoryPath, dataset.Cases[0], reportmodel.CostBasisMethodFIFO, "asset-alpha")
	var fixturePath = filepath.Join(repositoryRoot, filepath.FromSlash(expectedPath))
	var err = os.MkdirAll(fixturePath, 0o755)
	if err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}

	var result OracleFixturePolicyResult
	result, err = EnsureGoldenFixtures(context.Background(), repositoryRoot, DefaultEmpiricalArtifactRootRepositoryPath, dataset)
	if err == nil {
		t.Fatal("expected missing fixture setup error for fixture directory")
	}
	var missingErr missingOracleFixturesError
	if !errors.As(err, &missingErr) {
		t.Fatalf("expected missingOracleFixturesError, got %T: %v", err, err)
	}
	if !reflect.DeepEqual(result.MissingPaths, []string{expectedPath}) {
		t.Fatalf("unexpected result missing paths: got %v want %v", result.MissingPaths, []string{expectedPath})
	}
	if !reflect.DeepEqual(missingErr.MissingPaths, []string{expectedPath}) {
		t.Fatalf("unexpected error missing paths: got %v want %v", missingErr.MissingPaths, []string{expectedPath})
	}
}

// TestEnsureGoldenFixturesGeneratesOnlyWhenExplicitlyEnabled verifies the
// missing-fixture path stays opt-in and delegates to the repository-owned
// regeneration command seam only when explicitly allowed.
// Authored by: OpenCode
func TestEnsureGoldenFixturesGeneratesOnlyWhenExplicitlyEnabled(t *testing.T) {
	var repositoryRoot = t.TempDir()
	var dataset = EmpiricalDataset{
		Cases: []EmpiricalCase{{
			CaseID:            "case-supported-2024",
			Methods:           []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodFIFO},
			AssetIdentityKeys: []string{"asset-alpha"},
			OracleSupport:     OracleSupportSupported,
		}},
	}

	var result, err = EnsureGoldenFixtures(context.Background(), repositoryRoot, DefaultEmpiricalArtifactRootRepositoryPath, dataset)
	if err == nil {
		t.Fatal("expected missing fixture setup error without explicit opt-in")
	}
	var missingErr missingOracleFixturesError
	if !errors.As(err, &missingErr) {
		t.Fatalf("expected missingOracleFixturesError, got %T: %v", err, err)
	}
	if len(result.MissingPaths) != 1 {
		t.Fatalf("expected one missing fixture path, got %v", result.MissingPaths)
	}

	var priorExecuteMissingOracleFixtureGeneration = executeMissingOracleFixtureGeneration
	t.Cleanup(func() {
		executeMissingOracleFixtureGeneration = priorExecuteMissingOracleFixtureGeneration
		_ = os.Unsetenv(MissingOracleFixtureGenerationEnvVar)
	})
	var generationCalls int
	executeMissingOracleFixtureGeneration = func(ctx context.Context, gotRepositoryRoot string) error {
		generationCalls++
		if gotRepositoryRoot != repositoryRoot {
			t.Fatalf("unexpected repository root: got %s want %s", gotRepositoryRoot, repositoryRoot)
		}
		var fixturePath = filepath.Join(repositoryRoot, filepath.FromSlash(result.MissingPaths[0]))
		if err := os.MkdirAll(filepath.Dir(fixturePath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(fixturePath, []byte("{}\n"), 0o644)
	}
	if err = os.Setenv(MissingOracleFixtureGenerationEnvVar, "true"); err != nil {
		t.Fatalf("set env: %v", err)
	}

	result, err = EnsureGoldenFixtures(context.Background(), repositoryRoot, DefaultEmpiricalArtifactRootRepositoryPath, dataset)
	if err != nil {
		t.Fatalf("ensure golden fixtures with opt-in generation: %v", err)
	}
	if !result.Generated {
		t.Fatal("expected opt-in generation to report Generated=true")
	}
	if generationCalls != 1 {
		t.Fatalf("expected one generation call, got %d", generationCalls)
	}
}

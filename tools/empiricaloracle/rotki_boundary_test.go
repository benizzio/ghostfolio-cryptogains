package main

import (
	"path/filepath"
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

// TestVerifyRotkiBoundaryMaterialsPassesForRepositoryFixtures verifies the
// repository-controlled rotki boundary is complete enough for fixture
// regeneration checks.
// Authored by: OpenCode
func TestVerifyRotkiBoundaryMaterialsPassesForRepositoryFixtures(t *testing.T) {
	t.Parallel()

	var repositoryRoot, err = resolveEmpiricalRepositoryRoot()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	var datasetPath = filepath.Join(repositoryRoot, "testdata/empirical/financial-dataset.yaml")
	var dataset, _, loadErr = fixture.LoadEmpiricalDataset(datasetPath)
	if loadErr != nil {
		t.Fatalf("load empirical dataset: %v", loadErr)
	}

	if err = verifyRotkiBoundaryMaterials(repositoryRoot, dataset); err != nil {
		t.Fatalf("verify rotki boundary materials: %v", err)
	}
}

// TestLoadBoundaryOracleInputRejectsMissingFile verifies boundary input loading
// fails with an actionable missing-file error.
// Authored by: OpenCode
func TestLoadBoundaryOracleInputRejectsMissingFile(t *testing.T) {
	t.Parallel()

	var repositoryRoot, err = resolveEmpiricalRepositoryRoot()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}

	var empiricalCase = fixture.EmpiricalCase{
		CaseID:            "case-missing-2024",
		AssetIdentityKeys: []string{"asset-alpha"},
	}

	_, _, _, err = loadBoundaryOracleInput(repositoryRoot, empiricalCase, reportmodel.CostBasisMethodFIFO, "asset-alpha")
	if err == nil {
		t.Fatal("expected missing boundary input error, got nil")
	}
	if !strings.Contains(err.Error(), "read repository-controlled boundary input") {
		t.Fatalf("unexpected missing boundary input error: %v", err)
	}
}

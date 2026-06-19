package main

import (
	"path/filepath"
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

// TestSelectRotkiOracleActivitiesKeepsHistoricalStateAndDropsPostYearRows
// verifies generated adapter inputs keep prior history while excluding
// post-selected-year activities.
// Authored by: OpenCode
func TestSelectRotkiOracleActivitiesKeepsHistoricalStateAndDropsPostYearRows(t *testing.T) {
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

	var empiricalCase, findErr = findEmpiricalCase(dataset, "case-scope-local-reset-epsilon-2024", reportmodel.CostBasisMethodScopeLocalHybrid)
	if findErr != nil {
		t.Fatalf("find empirical case: %v", findErr)
	}

	var activities = selectRotkiOracleActivities(dataset, empiricalCase, "asset-epsilon")
	if len(activities) == 0 {
		t.Fatal("expected generated rotki adapter activities")
	}
	if strings.TrimSpace(activities[0].SourceID) != "emp-act-000041" {
		t.Fatalf("expected historical opening state to be retained, got first source_id %s", activities[0].SourceID)
	}
	if strings.TrimSpace(activities[len(activities)-1].SourceID) != "emp-act-000100" {
		t.Fatalf("expected post-year activities to be excluded, got last source_id %s", activities[len(activities)-1].SourceID)
	}
}

// TestBuildRotkiOracleInputUsesUntrackedCachePath verifies generated adapter
// inputs are written below the repository-local ignored cache path.
// Authored by: OpenCode
func TestBuildRotkiOracleInputUsesUntrackedCachePath(t *testing.T) {
	t.Parallel()

	var empiricalCase = fixture.EmpiricalCase{
		CaseID:            "case-hifo-gamma-2024",
		AssetIdentityKeys: []string{"asset-gamma"},
		Year:              2024,
		ActivitySourceIDs: []string{"emp-act-000073", "emp-act-000075"},
	}
	var activities = []fixture.EmpiricalActivity{{SourceID: "emp-act-000021", OccurredAt: "2023-01-05T09:00:00Z", DeterministicOrder: 1, ActivityType: "BUY", AssetIdentityKey: "asset-gamma", AssetSymbol: "GAMMA", Quantity: "1", GrossValue: "18"}}

	_, relativePath, rawInput, err := buildRotkiOracleInput(empiricalCase, reportmodel.CostBasisMethodHIFO, "asset-gamma", activities, false)
	if err != nil {
		t.Fatalf("build rotki oracle input: %v", err)
	}
	if !strings.HasPrefix(relativePath, rotkiOracleInputRootRepositoryPath+"/") {
		t.Fatalf("expected generated rotki input below %s, got %s", rotkiOracleInputRootRepositoryPath, relativePath)
	}
	if len(rawInput) == 0 {
		t.Fatal("expected generated rotki adapter input content")
	}
	if !strings.Contains(string(rawInput), "\"rotki_method\": \"hifo\"") {
		t.Fatalf("expected generated rotki input to record rotki_method hifo, got %s", string(rawInput))
	}
}

// TestBuildAverageCostUnsupportedSegmentsUsesRelevantSellRows verifies the
// average-cost aggregate fixtures keep their explicit pool-provenance skip
// metadata on the selected sell rows.
// Authored by: OpenCode
func TestBuildAverageCostUnsupportedSegmentsUsesRelevantSellRows(t *testing.T) {
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

	var empiricalCase, findErr = findEmpiricalCase(dataset, "case-average-cost-reset-delta-2024", reportmodel.CostBasisMethodAverageCost)
	if findErr != nil {
		t.Fatalf("find empirical case: %v", findErr)
	}

	var segments = buildAverageCostUnsupportedSegments(dataset, empiricalCase, "asset-delta")
	if len(segments) != 1 {
		t.Fatalf("expected one unsupported segment, got %d", len(segments))
	}
	if strings.Join(segments[0].ActivitySourceIDs, ",") != "emp-act-000083,emp-act-000085,emp-act-000087" {
		t.Fatalf("unexpected average-cost unsupported source ids: %#v", segments[0].ActivitySourceIDs)
	}
	if segments[0].ComparisonPolicy != fixture.ComparisonPolicySkipExternalOracle {
		t.Fatalf("unexpected average-cost comparison policy: %s", segments[0].ComparisonPolicy)
	}
}

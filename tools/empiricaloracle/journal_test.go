package main

import (
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

// TestJournalRenderJournalRendersPricedActivitiesAndHistoricalContext verifies
// priced acquisitions, priced liquidations, fee handling, same-date ordering,
// and the required pre-case historical slice.
// Authored by: OpenCode
func TestJournalRenderJournalRendersPricedActivitiesAndHistoricalContext(t *testing.T) {
	t.Parallel()

	var rawDatasetContent = "synthetic-priced-history-dataset"
	var dataset = fixture.EmpiricalDataset{
		DatasetVersion: "1",
		Currency:       "USD",
		Activities: []fixture.EmpiricalActivity{
			{
				SourceID:           "buy-old",
				OccurredAt:         "2023-12-31T09:00:00Z",
				DeterministicOrder: 1,
				ActivityType:       syncmodel.ActivityTypeBuy,
				AssetIdentityKey:   "asset-alpha",
				AssetSymbol:        "ALPHA",
				Quantity:           "1",
				GrossValue:         "10",
				UnitPrice:          "10",
				Currency:           "USD",
			},
			{
				SourceID:           "buy-1",
				OccurredAt:         "2024-01-05T09:00:00Z",
				DeterministicOrder: 1,
				ActivityType:       syncmodel.ActivityTypeBuy,
				AssetIdentityKey:   "asset-alpha",
				AssetSymbol:        "ALPHA",
				Quantity:           "2",
				GrossValue:         "20",
				FeeAmount:          "1",
				UnitPrice:          "10",
				Currency:           "USD",
			},
			{
				SourceID:           "sell-1",
				OccurredAt:         "2024-01-05T09:00:00Z",
				DeterministicOrder: 2,
				ActivityType:       syncmodel.ActivityTypeSell,
				AssetIdentityKey:   "asset-alpha",
				AssetSymbol:        "ALPHA",
				Quantity:           "1",
				GrossValue:         "15",
				FeeAmount:          "1",
				UnitPrice:          "15",
				Currency:           "USD",
			},
			{
				SourceID:           "buy-after",
				OccurredAt:         "2024-02-01T09:00:00Z",
				DeterministicOrder: 1,
				ActivityType:       syncmodel.ActivityTypeBuy,
				AssetIdentityKey:   "asset-alpha",
				AssetSymbol:        "ALPHA",
				Quantity:           "1",
				GrossValue:         "30",
				UnitPrice:          "30",
				Currency:           "USD",
			},
		},
		Cases: []fixture.EmpiricalCase{{
			CaseID:            "case-fifo-alpha-2024",
			Methods:           []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodFIFO},
			Year:              2024,
			AssetIdentityKeys: []string{"asset-alpha"},
			ActivitySourceIDs: []string{"buy-1", "sell-1"},
			OracleSupport:     fixture.OracleSupportSupported,
		}},
	}

	var output, err = renderJournal(dataset, rawDatasetContent, dataset.Cases[0], reportmodel.CostBasisMethodFIFO)
	if err != nil {
		t.Fatalf("render journal: %v", err)
	}

	if !strings.Contains(output.content, "commodity ALPHA  ; lots: FIFO") {
		t.Fatalf("expected FIFO commodity directive, got:\n%s", output.content)
	}
	if strings.Contains(output.content, "buy-after") {
		t.Fatalf("expected post-case activity to be excluded, got:\n%s", output.content)
	}
	assertJournalStringOrder(t, output.content, "2023-12-31 buy buy-old", "2024-01-05 buy buy-1", "2024-01-05 sell sell-1")

	if !strings.Contains(output.content, `assets:empirical:fifo:asset-alpha  2 ALPHA {2024-01-05, "buy-1", $10.5}`) {
		t.Fatalf("expected priced BUY selector with fee-adjusted basis, got:\n%s", output.content)
	}
	if !strings.Contains(output.content, `assets:cash:USD  -$21`) {
		t.Fatalf("expected BUY cash posting to balance gross plus fee, got:\n%s", output.content)
	}
	if !strings.Contains(output.content, `assets:empirical:fifo:asset-alpha  -1 ALPHA {} @ $14  ; posting_source_id: sell-1`) {
		t.Fatalf("expected priced SELL posting with fee-adjusted net proceeds, got:\n%s", output.content)
	}
	if !strings.Contains(output.content, `assets:cash:USD  $14`) {
		t.Fatalf("expected SELL cash posting to balance net proceeds, got:\n%s", output.content)
	}
}

// TestJournalRenderJournalUsesReliableHybridScopeAccounts verifies the special
// reliable hybrid case uses FIFO lot mode, scoped reliable accounts, and the
// fallback account for non-reliable rows.
// Authored by: OpenCode
func TestJournalRenderJournalUsesReliableHybridScopeAccounts(t *testing.T) {
	t.Parallel()

	var dataset = fixture.EmpiricalDataset{
		DatasetVersion: "1",
		Currency:       "USD",
		Activities: []fixture.EmpiricalActivity{
			{
				SourceID:           "scope-buy",
				OccurredAt:         "2024-01-05T09:00:00Z",
				DeterministicOrder: 1,
				ActivityType:       syncmodel.ActivityTypeBuy,
				AssetIdentityKey:   "asset-epsilon",
				AssetSymbol:        "EPS",
				Quantity:           "1",
				GrossValue:         "29",
				UnitPrice:          "29",
				Currency:           "USD",
				SourceScope: &fixture.EmpiricalScope{
					ScopeID:     "wallet-1",
					ScopeKind:   syncmodel.SourceScopeKindWallet,
					Reliability: syncmodel.ScopeReliabilityReliable,
				},
			},
			{
				SourceID:           "fallback-buy",
				OccurredAt:         "2024-01-06T09:00:00Z",
				DeterministicOrder: 1,
				ActivityType:       syncmodel.ActivityTypeBuy,
				AssetIdentityKey:   "asset-epsilon",
				AssetSymbol:        "EPS",
				Quantity:           "1",
				GrossValue:         "30",
				UnitPrice:          "30",
				Currency:           "USD",
			},
			{
				SourceID:           "scope-sell",
				OccurredAt:         "2024-01-07T09:00:00Z",
				DeterministicOrder: 1,
				ActivityType:       syncmodel.ActivityTypeSell,
				AssetIdentityKey:   "asset-epsilon",
				AssetSymbol:        "EPS",
				Quantity:           "1",
				GrossValue:         "32",
				UnitPrice:          "32",
				Currency:           "USD",
				SourceScope: &fixture.EmpiricalScope{
					ScopeID:     "wallet-1",
					ScopeKind:   syncmodel.SourceScopeKindWallet,
					Reliability: syncmodel.ScopeReliabilityReliable,
				},
			},
		},
		Cases: []fixture.EmpiricalCase{{
			CaseID:            reliableHybridFIFOCaseID,
			Methods:           []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodScopeLocalHybrid},
			Year:              2024,
			AssetIdentityKeys: []string{"asset-epsilon"},
			ActivitySourceIDs: []string{"scope-buy", "fallback-buy", "scope-sell"},
			OracleSupport:     fixture.OracleSupportPartiallySupported,
		}},
	}

	var output, err = renderJournal(dataset, "synthetic-hybrid-scope-dataset", dataset.Cases[0], reportmodel.CostBasisMethodScopeLocalHybrid)
	if err != nil {
		t.Fatalf("render scope-local-hybrid journal: %v", err)
	}

	if !strings.Contains(output.content, "commodity EPS  ; lots: FIFO") {
		t.Fatalf("expected hybrid reliable case to use FIFO lot mode, got:\n%s", output.content)
	}
	if !strings.Contains(output.content, `assets:empirical:scope-local-hybrid:wallet-1:asset-epsilon  1 EPS {2024-01-05, "scope-buy", $29}`) {
		t.Fatalf("expected reliable scoped account path, got:\n%s", output.content)
	}
	if !strings.Contains(output.content, `assets:empirical:scope-local-hybrid:fallback:asset-epsilon  1 EPS {2024-01-06, "fallback-buy", $30}`) {
		t.Fatalf("expected fallback account path for non-reliable row, got:\n%s", output.content)
	}
	if !strings.Contains(output.content, `assets:empirical:scope-local-hybrid:wallet-1:asset-epsilon  -1 EPS {} @ $32  ; posting_source_id: scope-sell`) {
		t.Fatalf("expected reliable scoped SELL posting, got:\n%s", output.content)
	}
}

// TestJournalRenderJournalsAreCaseScopedAndPopulateMetadata verifies one
// journal is produced per method and case, with stable paths and hashes.
// Authored by: OpenCode
func TestJournalRenderJournalsAreCaseScopedAndPopulateMetadata(t *testing.T) {
	t.Parallel()

	var rawDatasetContent = "synthetic-metadata-dataset"
	var dataset = fixture.EmpiricalDataset{
		DatasetVersion: "1",
		Currency:       "USD",
		Activities: []fixture.EmpiricalActivity{
			{
				SourceID:           "alpha-buy",
				OccurredAt:         "2024-01-01T09:00:00Z",
				DeterministicOrder: 1,
				ActivityType:       syncmodel.ActivityTypeBuy,
				AssetIdentityKey:   "asset-alpha",
				AssetSymbol:        "ALPHA",
				Quantity:           "1",
				GrossValue:         "11",
				UnitPrice:          "11",
				Currency:           "USD",
			},
			{
				SourceID:           "beta-buy",
				OccurredAt:         "2024-01-01T09:00:00Z",
				DeterministicOrder: 1,
				ActivityType:       syncmodel.ActivityTypeBuy,
				AssetIdentityKey:   "asset-beta",
				AssetSymbol:        "BETA",
				Quantity:           "1",
				GrossValue:         "22",
				UnitPrice:          "22",
				Currency:           "USD",
			},
		},
		Cases: []fixture.EmpiricalCase{
			{
				CaseID:            "case-alpha",
				Methods:           []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodFIFO},
				Year:              2024,
				AssetIdentityKeys: []string{"asset-alpha"},
				ActivitySourceIDs: []string{"alpha-buy"},
				OracleSupport:     fixture.OracleSupportSupported,
			},
			{
				CaseID:            "case-beta",
				Methods:           []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodFIFO},
				Year:              2024,
				AssetIdentityKeys: []string{"asset-beta"},
				ActivitySourceIDs: []string{"beta-buy"},
				OracleSupport:     fixture.OracleSupportSupported,
			},
		},
	}

	var outputs, err = renderJournals(dataset, rawDatasetContent)
	if err != nil {
		t.Fatalf("render case-scoped journals: %v", err)
	}
	if len(outputs) != 2 {
		t.Fatalf("expected one journal per case, got %d", len(outputs))
	}

	var alpha = findJournalByPath(t, outputs, "testdata/empirical/hledger/fifo/case-alpha.journal")
	var beta = findJournalByPath(t, outputs, "testdata/empirical/hledger/fifo/case-beta.journal")
	var datasetHash = stablePrefixedSHA256Hash([]byte(rawDatasetContent))

	if alpha.ledger.LedgerID != "empirical-journal:fifo:case-alpha" {
		t.Fatalf("unexpected alpha ledger id: %q", alpha.ledger.LedgerID)
	}
	if len(alpha.ledger.CaseIDs) != 1 || alpha.ledger.CaseIDs[0] != "case-alpha" {
		t.Fatalf("unexpected alpha case ids: %#v", alpha.ledger.CaseIDs)
	}
	if alpha.ledger.DatasetInputHash != datasetHash {
		t.Fatalf("unexpected alpha dataset hash: got %q want %q", alpha.ledger.DatasetInputHash, datasetHash)
	}
	if alpha.ledger.ExternalOracleInputHash != stablePrefixedSHA256Hash([]byte(alpha.content)) {
		t.Fatalf("unexpected alpha journal hash metadata: got %q want %q", alpha.ledger.ExternalOracleInputHash, stablePrefixedSHA256Hash([]byte(alpha.content)))
	}
	if strings.Contains(alpha.content, "beta-buy") {
		t.Fatalf("expected alpha journal to stay case-scoped, got:\n%s", alpha.content)
	}

	if beta.ledger.LedgerID != "empirical-journal:fifo:case-beta" {
		t.Fatalf("unexpected beta ledger id: %q", beta.ledger.LedgerID)
	}
	if len(beta.ledger.CaseIDs) != 1 || beta.ledger.CaseIDs[0] != "case-beta" {
		t.Fatalf("unexpected beta case ids: %#v", beta.ledger.CaseIDs)
	}
	if beta.ledger.DatasetInputHash != datasetHash {
		t.Fatalf("unexpected beta dataset hash: got %q want %q", beta.ledger.DatasetInputHash, datasetHash)
	}
	if beta.ledger.ExternalOracleInputHash != stablePrefixedSHA256Hash([]byte(beta.content)) {
		t.Fatalf("unexpected beta journal hash metadata: got %q want %q", beta.ledger.ExternalOracleInputHash, stablePrefixedSHA256Hash([]byte(beta.content)))
	}
	if strings.Contains(beta.content, "alpha-buy") {
		t.Fatalf("expected beta journal to stay case-scoped, got:\n%s", beta.content)
	}
}

// TestJournalRenderJournalSplitsNativeZeroPricedReduction verifies a native
// exact-lot journal pre-splits one zero-priced reduction into matched sink
// transfer transactions.
// Authored by: OpenCode
func TestJournalRenderJournalSplitsNativeZeroPricedReduction(t *testing.T) {
	t.Parallel()

	var dataset = fixture.EmpiricalDataset{
		DatasetVersion: "1",
		Currency:       "USD",
		Activities: []fixture.EmpiricalActivity{
			{
				SourceID:           "low-buy",
				OccurredAt:         "2024-01-01T09:00:00Z",
				DeterministicOrder: 1,
				ActivityType:       syncmodel.ActivityTypeBuy,
				AssetIdentityKey:   "asset-gamma",
				AssetSymbol:        "GAMMA",
				Quantity:           "1",
				GrossValue:         "10",
				UnitPrice:          "10",
				Currency:           "USD",
			},
			{
				SourceID:           "high-buy",
				OccurredAt:         "2024-01-02T09:00:00Z",
				DeterministicOrder: 1,
				ActivityType:       syncmodel.ActivityTypeBuy,
				AssetIdentityKey:   "asset-gamma",
				AssetSymbol:        "GAMMA",
				Quantity:           "2",
				GrossValue:         "30",
				UnitPrice:          "15",
				Currency:           "USD",
			},
			{
				SourceID:           "sell-high",
				OccurredAt:         "2024-01-03T09:00:00Z",
				DeterministicOrder: 1,
				ActivityType:       syncmodel.ActivityTypeSell,
				AssetIdentityKey:   "asset-gamma",
				AssetSymbol:        "GAMMA",
				Quantity:           "1",
				GrossValue:         "20",
				UnitPrice:          "20",
				Currency:           "USD",
			},
			{
				SourceID:                       "zero-1",
				OccurredAt:                     "2024-01-04T09:00:00Z",
				DeterministicOrder:             1,
				ActivityType:                   syncmodel.ActivityTypeSell,
				AssetIdentityKey:               "asset-gamma",
				AssetSymbol:                    "GAMMA",
				Quantity:                       "2",
				ZeroPricedReductionExplanation: "native exact-lot reduction",
			},
		},
		Cases: []fixture.EmpiricalCase{{
			CaseID:            "case-zero-hifo-gamma-2024",
			Methods:           []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodHIFO},
			Year:              2024,
			AssetIdentityKeys: []string{"asset-gamma"},
			ActivitySourceIDs: []string{"high-buy", "sell-high", "zero-1"},
			OracleSupport:     fixture.OracleSupportSupported,
		}},
	}

	var output, err = renderJournal(dataset, "synthetic-native-zero-dataset", dataset.Cases[0], reportmodel.CostBasisMethodHIFO)
	if err != nil {
		t.Fatalf("render HIFO zero-priced journal: %v", err)
	}
	if len(output.ledger.GenerationNotes) != 0 {
		t.Fatalf("expected native exact-lot handling without omission notes, got %#v", output.ledger.GenerationNotes)
	}

	if !strings.Contains(output.content, "commodity GAMMA  ; lots: HIFO") {
		t.Fatalf("expected HIFO commodity directive, got:\n%s", output.content)
	}
	assertJournalStringOrder(
		t,
		output.content,
		"2024-01-04 zero-priced reduction zero-1 from high-buy",
		"2024-01-04 zero-priced reduction zero-1 from low-buy",
	)
	if !strings.Contains(output.content, `assets:empirical:hifo:asset-gamma  -1 GAMMA {2024-01-02, "high-buy", $15}`) {
		t.Fatalf("expected first zero-priced segment to use the remaining higher-cost lot, got:\n%s", output.content)
	}
	if !strings.Contains(output.content, `assets:empirical:hifo:asset-gamma  -1 GAMMA {2024-01-01, "low-buy", $10}`) {
		t.Fatalf("expected second zero-priced segment to use the remaining lower-cost lot, got:\n%s", output.content)
	}
	if strings.Count(output.content, "equity:zero-priced-reduction  1 GAMMA") != 2 {
		t.Fatalf("expected one sink transfer per matched lot segment, got:\n%s", output.content)
	}
}

// TestJournalRenderJournalOmitsNonNativeZeroPricedReductionWithNote verifies a
// non-native lot mode skips zero-priced reductions and records a generation
// note instead.
// Authored by: OpenCode
func TestJournalRenderJournalOmitsNonNativeZeroPricedReductionWithNote(t *testing.T) {
	t.Parallel()

	var dataset = fixture.EmpiricalDataset{
		DatasetVersion: "1",
		Currency:       "USD",
		Activities: []fixture.EmpiricalActivity{
			{
				SourceID:           "avg-buy",
				OccurredAt:         "2025-01-01T09:00:00Z",
				DeterministicOrder: 1,
				ActivityType:       syncmodel.ActivityTypeBuy,
				AssetIdentityKey:   "asset-delta",
				AssetSymbol:        "DELTA",
				Quantity:           "2",
				GrossValue:         "20",
				UnitPrice:          "10",
				Currency:           "USD",
			},
			{
				SourceID:                       "avg-zero",
				OccurredAt:                     "2025-01-02T09:00:00Z",
				DeterministicOrder:             1,
				ActivityType:                   syncmodel.ActivityTypeSell,
				AssetIdentityKey:               "asset-delta",
				AssetSymbol:                    "DELTA",
				Quantity:                       "1",
				ZeroPricedReductionExplanation: "average-cost omission",
			},
		},
		Cases: []fixture.EmpiricalCase{{
			CaseID:            "case-average-zero-2025",
			Methods:           []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodAverageCost},
			Year:              2025,
			AssetIdentityKeys: []string{"asset-delta"},
			ActivitySourceIDs: []string{"avg-buy", "avg-zero"},
			OracleSupport:     fixture.OracleSupportSupported,
		}},
	}

	var output, err = renderJournal(dataset, "synthetic-average-zero-dataset", dataset.Cases[0], reportmodel.CostBasisMethodAverageCost)
	if err != nil {
		t.Fatalf("render average-cost zero-priced journal: %v", err)
	}

	if strings.Contains(output.content, "avg-zero") {
		t.Fatalf("expected omitted zero-priced reduction to stay out of the journal body, got:\n%s", output.content)
	}
	if len(output.ledger.GenerationNotes) != 1 {
		t.Fatalf("expected one omission note, got %#v", output.ledger.GenerationNotes)
	}
	if !strings.Contains(output.ledger.GenerationNotes[0], "avg-zero") || !strings.Contains(output.ledger.GenerationNotes[0], "AVERAGE") {
		t.Fatalf("expected omission note to record source id and lot mode, got %q", output.ledger.GenerationNotes[0])
	}
}

// findJournalByPath returns one rendered journal selected by its persisted
// relative journal path.
// Authored by: OpenCode
func findJournalByPath(t *testing.T, outputs []journal, wantPath string) journal {
	t.Helper()

	var index int
	for index = range outputs {
		if outputs[index].ledger.ExternalOracleInputPath == wantPath {
			return outputs[index]
		}
	}

	t.Fatalf("expected rendered journal path %q, got %#v", wantPath, outputs)
	return journal{}
}

// assertJournalStringOrder verifies a set of marker substrings appear in the
// rendered journal content in the expected order.
// Authored by: OpenCode
func assertJournalStringOrder(t *testing.T, content string, wantInOrder ...string) {
	t.Helper()

	var previousIndex = -1
	var want string
	for _, want = range wantInOrder {
		var currentIndex = strings.Index(content, want)
		if currentIndex < 0 {
			t.Fatalf("expected rendered journal to contain %q, got:\n%s", want, content)
		}
		if currentIndex <= previousIndex {
			t.Fatalf("expected %q to appear after the previous marker in:\n%s", want, content)
		}

		previousIndex = currentIndex
	}
}

package main

import (
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

// TestHledgerBuildOracleComparableValuesDerivesMatchesAndClosingBalances
// verifies hledger-derived comparable values, matched fragments, and closing
// balances stay aligned with the parsed print and balance inputs.
// Authored by: OpenCode
func TestHledgerBuildOracleComparableValuesDerivesMatchesAndClosingBalances(t *testing.T) {
	t.Parallel()

	var oracleData = hledgerJournalOracleData{
		printTransactions: []hledgerPrintTransaction{{
			Date:        "2024-01-03",
			Description: "sell sell-1",
			Postings: []hledgerPosting{
				{
					Account: "assets:empirical:fifo:asset-alpha:{2024-01-01, \"buy-1\", $10}",
					Amounts: []hledgerAmount{{
						Commodity: "ALPHA",
						Quantity:  hledgerDecimal{Mantissa: -1, Places: 0},
						Cost: &hledgerTaggedAmount{
							Tag: "UnitCost",
							Contents: hledgerRawAmount{
								Commodity: "$",
								Quantity:  hledgerDecimal{Mantissa: 15, Places: 0},
							},
						},
						CostBasis: &hledgerCostBasis{
							Label: "buy-1",
							Cost: hledgerRawAmount{
								Commodity: "$",
								Quantity:  hledgerDecimal{Mantissa: 10, Places: 0},
							},
						},
					}},
					Tags: [][]string{{"posting_source_id", "sell-1"}, {"_ptype", "dispose"}},
				},
				{
					Account: "revenues:gain",
					Amounts: []hledgerAmount{{
						Commodity: "$",
						Quantity:  hledgerDecimal{Mantissa: -5, Places: 0},
					}},
					Tags: [][]string{{"_ptype", "rgain"}},
				},
			},
		}},
		balanceRows: []hledgerBalanceAccountRow{{
			Account: "assets:empirical:fifo:asset-alpha:{2024-01-01, \"buy-1\", $10}",
			Amounts: []hledgerAmount{
				{
					Commodity: "ALPHA",
					Quantity:  hledgerDecimal{Mantissa: 2, Places: 0},
					CostBasis: &hledgerCostBasis{
						Label: "buy-1",
						Cost: hledgerRawAmount{
							Commodity: "$",
							Quantity:  hledgerDecimal{Mantissa: 10, Places: 0},
						},
					},
				},
				{
					Commodity: "ALPHA",
					Quantity:  hledgerDecimal{Mantissa: -1, Places: 0},
					CostBasis: &hledgerCostBasis{
						Label: "buy-1",
						Cost: hledgerRawAmount{
							Commodity: "$",
							Quantity:  hledgerDecimal{Mantissa: 10, Places: 0},
						},
					},
				},
			},
		}},
	}

	var values, matches, err = buildOracleComparableValues("asset-alpha", oracleData)
	if err != nil {
		t.Fatalf("build oracle comparable values: %v", err)
	}

	if values.RealizedGainOrLoss != "5" || values.AllocatedBasis != "10" || values.ClosingQuantity != "1" || values.ClosingBasis != "10" {
		t.Fatalf("unexpected comparable values: %+v", values)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one matched fragment, got %+v", matches)
	}
	if matches[0].DisposedSourceID != "sell-1" || matches[0].AcquisitionSourceID != "buy-1" {
		t.Fatalf("unexpected matched source ids: %+v", matches[0])
	}
	if matches[0].MatchedQuantity != "1" || matches[0].MatchedBasis != "10" || matches[0].MatchedProceeds != "15" || matches[0].MatchedGainOrLoss != "5" {
		t.Fatalf("unexpected matched values: %+v", matches[0])
	}
	if matches[0].SupportLabel != fixture.EvidenceSupportLabelHledgerBacked {
		t.Fatalf("expected hledger-backed support label, got %+v", matches[0])
	}
}

// TestOracleBuildUnsupportedSegmentsFiltersByAssetAndOmissions verifies
// partially supported cases stay asset-filtered and zero-priced omissions add a
// separate project-composition-only unsupported segment.
// Authored by: OpenCode
func TestOracleBuildUnsupportedSegmentsFiltersByAssetAndOmissions(t *testing.T) {
	t.Parallel()

	var empiricalCase = fixture.EmpiricalCase{
		CaseID:            "case-scope-local-broadening-gamma-2024",
		Methods:           []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodScopeLocalHybrid},
		Year:              2024,
		AssetIdentityKeys: []string{"asset-gamma", "asset-delta"},
		ActivitySourceIDs: []string{"gamma-buy", "gamma-zero", "delta-buy"},
		OracleSupport:     fixture.OracleSupportPartiallySupported,
		UnsupportedReason: "Hybrid broadening remains partly project-owned composition",
	}
	var dataset = fixture.EmpiricalDataset{
		Activities: []fixture.EmpiricalActivity{
			{
				SourceID:         "gamma-buy",
				AssetIdentityKey: "asset-gamma",
				ActivityType:     syncmodel.ActivityTypeBuy,
			},
			{
				SourceID:                       "gamma-zero",
				AssetIdentityKey:               "asset-gamma",
				ActivityType:                   syncmodel.ActivityTypeSell,
				ZeroPricedReductionExplanation: "omitted average-cost style reduction",
			},
			{
				SourceID:         "delta-buy",
				AssetIdentityKey: "asset-delta",
				ActivityType:     syncmodel.ActivityTypeBuy,
			},
		},
	}

	var omissionNote = zeroPricedReductionOmissionNote(dataset.Activities[1], empiricalCase, reportmodel.CostBasisMethodScopeLocalHybrid)
	var segments = buildUnsupportedSegments(
		dataset,
		empiricalCase,
		reportmodel.CostBasisMethodScopeLocalHybrid,
		"asset-gamma",
		[]string{omissionNote},
		[]oracleMatchEvidenceInput{{
			DisposedSourceID: "gamma-buy",
			SupportLabel:     fixture.EvidenceSupportLabelHledgerBacked,
		}},
	)

	if len(segments) != 1 {
		t.Fatalf("expected only omission unsupported segment after hledger-backed filtering, got %+v", segments)
	}
	if len(segments[0].ActivitySourceIDs) != 1 || segments[0].ActivitySourceIDs[0] != "gamma-zero" {
		t.Fatalf("expected omitted zero-priced activity ids, got %+v", segments[0])
	}
	if segments[0].ComparisonPolicy != fixture.ComparisonPolicyProjectCompositionOnly {
		t.Fatalf("expected project-composition-only omission policy, got %+v", segments[0])
	}
}

// TestOracleCommandProvenanceArgumentsIncludePrintAndBalance verifies metadata records both hledger inputs needed to derive one oracle fixture.
//
// Authored by: OpenCode
func TestOracleCommandProvenanceArgumentsIncludePrintAndBalance(t *testing.T) {
	t.Parallel()

	var arguments = oracleCommandProvenanceArguments(hledgerJournalOracleData{
		printCommandArguments:   oraclePrintCommandArguments("testdata/empirical/hledger/fifo/case-alpha.journal", 2024),
		balanceCommandArguments: oracleClosingBalanceCommandArguments("testdata/empirical/hledger/fifo/case-alpha.journal", 2024),
	})

	if len(arguments) == 0 {
		t.Fatal("expected command provenance arguments, got none")
	}
	if !containsString(arguments, "print") || !containsString(arguments, "balance") {
		t.Fatalf("expected print and balance commands in provenance, got %#v", arguments)
	}
	if !containsString(arguments, "--next-command--") {
		t.Fatalf("expected command separator in provenance, got %#v", arguments)
	}
	if !containsString(arguments, "testdata/empirical/hledger/fifo/case-alpha.journal") {
		t.Fatalf("expected journal path in provenance, got %#v", arguments)
	}
}

// containsString reports whether one string slice contains the expected exact value.
//
// Authored by: OpenCode
func containsString(values []string, want string) bool {
	var value string
	for _, value = range values {
		if value == want {
			return true
		}
	}

	return false
}

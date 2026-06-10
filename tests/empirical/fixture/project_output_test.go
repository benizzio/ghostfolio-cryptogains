package fixture

import (
	"testing"
	"time"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// TestNormalizeProjectCalculationOutputAggregatesComparableValues verifies
// realized gain or loss, allocated basis, closing quantity, closing basis, and
// flattened match evidence all normalize into the empirical comparison shape.
// Authored by: OpenCode
func TestNormalizeProjectCalculationOutputAggregatesComparableValues(t *testing.T) {
	t.Parallel()

	var matchedProceeds = fixtureDecimalPointer(t, "15")
	var matchedGainOrLoss = fixtureDecimalPointer(t, "5")
	var secondaryGainOrLoss = fixtureDecimalPointer(t, "-1")
	var report = reportmodel.CapitalGainsReport{
		Year:            2024,
		CostBasisMethod: reportmodel.CostBasisMethodFIFO,
		DetailSections: []reportmodel.AssetDetailSection{{
			AssetIdentityKey: "asset-alpha",
			ClosingQuantity:  fixtureDecimal(t, "2"),
			ClosingCostBasis: fixtureDecimal(t, "20"),
			LiquidationSummaries: []reportmodel.LiquidationCalculation{
				{
					SourceID:       "sell-2",
					AllocatedBasis: fixtureDecimal(t, "10"),
					GainOrLoss:     fixtureDecimal(t, "5"),
					Matches: []reportmodel.BasisMatch{{
						AcquisitionSourceID: "buy-2",
						MatchedQuantity:     fixtureDecimal(t, "1"),
						MatchedBasis:        fixtureDecimal(t, "10"),
						MatchedProceeds:     matchedProceeds,
						MatchedGainOrLoss:   matchedGainOrLoss,
					}},
				},
				{
					SourceID:       "sell-1",
					AllocatedBasis: fixtureDecimal(t, "3"),
					GainOrLoss:     fixtureDecimal(t, "-1"),
					Matches: []reportmodel.BasisMatch{{
						AcquisitionSourceID: "buy-1",
						MatchedQuantity:     fixtureDecimal(t, "0.5"),
						MatchedBasis:        fixtureDecimal(t, "3"),
						MatchedGainOrLoss:   secondaryGainOrLoss,
					}},
				},
			},
		}},
	}

	var output, err = NormalizeProjectCalculationOutput("case-alpha-2024", report, "asset-alpha")
	if err != nil {
		t.Fatalf("normalize project output: %v", err)
	}

	if output.CaseID != "case-alpha-2024" || output.Method != reportmodel.CostBasisMethodFIFO || output.Year != 2024 {
		t.Fatalf("unexpected output identity: %+v", output)
	}
	if output.Values.RealizedGainOrLoss != "4" {
		t.Fatalf("unexpected realized gain or loss: got %q want %q", output.Values.RealizedGainOrLoss, "4")
	}
	if output.Values.AllocatedBasis != "13" {
		t.Fatalf("unexpected allocated basis: got %q want %q", output.Values.AllocatedBasis, "13")
	}
	if output.Values.ClosingQuantity != "2" {
		t.Fatalf("unexpected closing quantity: got %q want %q", output.Values.ClosingQuantity, "2")
	}
	if output.Values.ClosingBasis != "20" {
		t.Fatalf("unexpected closing basis: got %q want %q", output.Values.ClosingBasis, "20")
	}
	if len(output.Matches) != 2 {
		t.Fatalf("unexpected match count: got %d want %d", len(output.Matches), 2)
	}
	if output.Matches[0].DisposedSourceID != "sell-1" || output.Matches[0].AcquisitionSourceID != "buy-1" {
		t.Fatalf("unexpected first sorted match: %+v", output.Matches[0])
	}
	if output.Matches[0].MatchedQuantity != "0.5" || output.Matches[0].MatchedBasis != "3" || output.Matches[0].MatchedGainOrLoss != "-1" {
		t.Fatalf("unexpected first match values: %+v", output.Matches[0])
	}
	if output.Matches[1].DisposedSourceID != "sell-2" || output.Matches[1].AcquisitionSourceID != "buy-2" {
		t.Fatalf("unexpected second sorted match: %+v", output.Matches[1])
	}
	if output.Matches[1].MatchedProceeds != "15" || output.Matches[1].MatchedGainOrLoss != "5" {
		t.Fatalf("unexpected second match optional values: %+v", output.Matches[1])
	}
}

// TestNormalizeProjectCalculationOutputHandlesReferenceOnlyAssets verifies that
// assets excluded from the selected-year main sections normalize deterministically.
// Authored by: OpenCode
func TestNormalizeProjectCalculationOutputHandlesReferenceOnlyAssets(t *testing.T) {
	t.Parallel()

	var report = reportmodel.CapitalGainsReport{
		Year:            2024,
		CostBasisMethod: reportmodel.CostBasisMethodFIFO,
		ReferenceEntries: []reportmodel.ReferenceLiquidationEntry{{
			AssetIdentityKey:                   "asset-eth-001",
			FullLiquidationCountThroughYearEnd: 1,
			MainSectionStatus:                  reportmodel.ReferenceSectionStatusReferenceOnly,
		}},
	}

	var output, err = NormalizeProjectCalculationOutput("case-reference-only-2024", report, "asset-eth-001")
	if err != nil {
		t.Fatalf("normalize reference-only project output: %v", err)
	}

	if output.Values != (ComparableOutputValues{
		RealizedGainOrLoss: "0",
		AllocatedBasis:     "0",
		ClosingQuantity:    "0",
		ClosingBasis:       "0",
	}) {
		t.Fatalf("unexpected reference-only comparable values: %+v", output.Values)
	}
	if len(output.Matches) != 0 {
		t.Fatalf("expected no reference-only matches, got %+v", output.Matches)
	}
}

// TestNormalizeProjectCalculationOutputLabelsScopeLocalHybridEvidence verifies
// scope-local hybrid match evidence uses the hledger-backed label when the
// project report exposes comparable match rows.
// Authored by: OpenCode
func TestNormalizeProjectCalculationOutputLabelsScopeLocalHybridEvidence(t *testing.T) {
	t.Parallel()

	var report = reportmodel.CapitalGainsReport{
		Year:            2024,
		CostBasisMethod: reportmodel.CostBasisMethodScopeLocalHybrid,
		DetailSections: []reportmodel.AssetDetailSection{{
			AssetIdentityKey: "asset-atom-001",
			ClosingQuantity:  fixtureDecimal(t, "1"),
			ClosingCostBasis: fixtureDecimal(t, "20"),
			LiquidationSummaries: []reportmodel.LiquidationCalculation{{
				SourceID:       "atom-sell-1",
				OccurredAt:     time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				AllocatedBasis: fixtureDecimal(t, "100"),
				GainOrLoss:     fixtureDecimal(t, "150"),
				Matches: []reportmodel.BasisMatch{{
					AcquisitionSourceID: "atom-buy-1",
					MatchedQuantity:     fixtureDecimal(t, "1"),
					MatchedBasis:        fixtureDecimal(t, "100"),
				}},
			}},
		}},
	}

	var output, err = NormalizeProjectCalculationOutput("case-scope-local-2024", report, "asset-atom-001")
	if err != nil {
		t.Fatalf("normalize scope-local project output: %v", err)
	}

	if len(output.Matches) != 1 {
		t.Fatalf("unexpected scope-local match count: %d", len(output.Matches))
	}
	if output.Matches[0].SupportLabel != EvidenceSupportLabelHledgerBacked {
		t.Fatalf("unexpected scope-local support label: got %q want %q", output.Matches[0].SupportLabel, EvidenceSupportLabelHledgerBacked)
	}
	if output.Matches[0].ScopeID != "" {
		t.Fatalf("expected scope id to remain empty when unavailable in report output, got %q", output.Matches[0].ScopeID)
	}
}

// fixtureDecimal parses one decimal fixture for project-output tests.
// Authored by: OpenCode
func fixtureDecimal(t *testing.T, raw string) apd.Decimal {
	t.Helper()

	var value, _, err = decimalsupport.ParseString(raw)
	if err != nil {
		t.Fatalf("parse decimal %q: %v", raw, err)
	}

	return value
}

// fixtureDecimalPointer parses one optional decimal fixture for project-output
// tests.
// Authored by: OpenCode
func fixtureDecimalPointer(t *testing.T, raw string) *apd.Decimal {
	t.Helper()

	var value = fixtureDecimal(t, raw)
	return &value
}

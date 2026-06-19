package main

import (
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

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
			SupportLabel:     fixture.EvidenceSupportLabelRotkiBacked,
		}},
	)

	if len(segments) != 1 {
		t.Fatalf("expected only omission unsupported segment after rotki-backed filtering, got %+v", segments)
	}
	if len(segments[0].ActivitySourceIDs) != 1 || segments[0].ActivitySourceIDs[0] != "gamma-zero" {
		t.Fatalf("expected omitted zero-priced activity ids, got %+v", segments[0])
	}
	if segments[0].ComparisonPolicy != fixture.ComparisonPolicyProjectCompositionOnly {
		t.Fatalf("expected project-composition-only omission policy, got %+v", segments[0])
	}
}

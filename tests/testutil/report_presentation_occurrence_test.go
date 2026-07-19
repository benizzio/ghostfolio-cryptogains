package testutil

import "testing"

// TestPresentationAbsentNullableFieldIsExcludedFromVisibleFinancial proves a
// blank nullable field remains available to parity without earning V credit.
// Authored by: OpenCode
func TestPresentationAbsentNullableFieldIsExcludedFromVisibleFinancial(t *testing.T) {
	var manifest = DeterministicReportPresentationAcceptanceFixture()
	for _, acceptanceCase := range manifest.Cases {
		if acceptanceCase.ID != "financial/in-year-activity/absent" {
			continue
		}
		var nullableParity int
		for _, occurrence := range acceptanceCase.OccurrenceKeys {
			if occurrence.FieldName != "unit_price" {
				continue
			}
			if occurrence.Population == ReportPresentationPopulationVisibleFinancial {
				t.Fatal("blank nullable unit_price earned visible-financial credit")
			}
			if occurrence.Population == ReportPresentationPopulationParity {
				nullableParity++
			}
		}
		if nullableParity != 1 {
			t.Fatalf("blank nullable unit_price parity controls = %d, want 1", nullableParity)
		}
		return
	}
	t.Fatal("in-year activity absent control is missing")
}

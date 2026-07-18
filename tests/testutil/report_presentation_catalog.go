package testutil

// DeterministicReportPresentationAcceptanceFixture returns the finalized closed
// acceptance manifest required by the final report rendering contract. The
// manifest contains the complete ordered case catalog, two renderer attempts
// per case, semantic occurrence keys, and counters derived from those keys.
//
// Every case has exactly one Markdown attempt and one PDF attempt. The returned
// slices are newly allocated on each call, so a test may annotate its copy
// without changing another test's fixture.
//
// Example usage:
//
//	manifest := testutil.DeterministicReportPresentationAcceptanceFixture()
//	for _, acceptanceCase := range manifest.Cases {
//		if len(acceptanceCase.Attempts) != 2 {
//			t.Fatalf("case %q has %d attempts", acceptanceCase.ID, len(acceptanceCase.Attempts))
//		}
//	}
//	if manifest.Counters.CaseCount != len(manifest.Cases) {
//		t.Fatal("acceptance counters are inconsistent")
//	}
//
// Authored by: OpenCode
func DeterministicReportPresentationAcceptanceFixture() ReportPresentationAcceptanceManifest {
	var cases = make([]ReportPresentationAcceptanceCase, 0, 148)
	cases = append(cases, newPresentationWarningCase())
	cases = append(cases, newPresentationFinancialCases()...)
	cases = append(cases, newPresentationQuantityCases()...)
	cases = append(cases, newPresentationRateCases()...)
	cases = append(cases, newPresentationBooleanCases()...)
	cases = append(cases, newPresentationCurrencyCases()...)
	cases = append(cases, newPresentationConvertedCases()...)

	var manifest = ReportPresentationAcceptanceManifest{Cases: cases}
	manifest.Counters = countPresentationOccurrences(manifest.Cases)
	return manifest
}

// newPresentationWarningCase creates the sole wrapped-warning acceptance case.
// Authored by: OpenCode
func newPresentationWarningCase() ReportPresentationAcceptanceCase {
	var acceptanceCase = newPresentationCase(ReportPresentationAcceptanceCase{
		ID:         "warning/wrapped",
		Kind:       ReportPresentationCaseKindWarning,
		Section:    "warning",
		VectorCase: "wrapped",
	})
	acceptanceCase.ExpectedText = ReportPresentationLegalWarningText
	return acceptanceCase
}

// newPresentationFinancialCases expands the two closed financial vectors over
// every matrix row, nullable absent control, and inherited summary omission.
// Authored by: OpenCode
func newPresentationFinancialCases() []ReportPresentationAcceptanceCase {
	var cases = make([]ReportPresentationAcceptanceCase, 0, 124)
	for _, row := range presentationFinancialRows() {
		var vectors = presentationNonNegativeVectors()
		if row.Signed {
			vectors = presentationSignedVectors()
		}
		for _, vector := range vectors {
			var acceptanceCase = newPresentationCase(ReportPresentationAcceptanceCase{
				ID:                   "financial/" + row.ID + "/" + vector.ID,
				Kind:                 ReportPresentationCaseKindFinancial,
				Section:              row.ID,
				VectorCase:           vector.ID,
				ExactValue:           vector.ExactValue,
				ExpectedVisibleValue: vector.ExpectedValue,
				FinancialFields:      row.Fields,
			})
			acceptanceCase.FinancialFieldClass = row.ID
			appendPresentationFinancialOccurrences(&acceptanceCase, row)
			cases = append(cases, acceptanceCase)
		}
		if row.Nullable {
			var absentCase = newPresentationCase(ReportPresentationAcceptanceCase{
				ID:              "financial/" + row.ID + "/absent",
				Kind:            ReportPresentationCaseKindFinancial,
				Section:         row.ID,
				VectorCase:      "absent",
				Absent:          true,
				FinancialFields: row.Fields,
			})
			absentCase.FinancialFieldClass = row.ID
			appendPresentationFinancialOccurrences(&absentCase, row)
			cases = append(cases, absentCase)
		}
		if row.SummaryOmission {
			var omittedCase = newPresentationCase(ReportPresentationAcceptanceCase{
				ID:              "financial/" + row.ID + "/exact-zero-omitted",
				Kind:            ReportPresentationCaseKindFinancial,
				Section:         row.ID,
				VectorCase:      "exact-zero-omitted",
				ExactValue:      "0",
				Omitted:         true,
				FinancialFields: row.Fields,
			})
			omittedCase.FinancialFieldClass = row.ID
			omittedCase.OmittedFieldNames = []string{"per_asset_net_gain_or_loss"}
			appendPresentationFinancialOccurrences(&omittedCase, row)
			cases = append(cases, omittedCase)
		}
	}
	return cases
}

// newPresentationQuantityCases creates the five FR-009 quantity controls.
// Authored by: OpenCode
func newPresentationQuantityCases() []ReportPresentationAcceptanceCase {
	var values = []presentationScalarCase{
		{ID: "zero", ExactValue: "0", ExpectedValue: "0"},
		{ID: "whole-trailing-zero", ExactValue: "2.000", ExpectedValue: "2"},
		{ID: "fraction-trailing-zero", ExactValue: "0.1000", ExpectedValue: "0.1"},
		{ID: "small", ExactValue: "0.00000001", ExpectedValue: "0.00000001"},
		{ID: "large", ExactValue: "12345678901234567890.123456789", ExpectedValue: "12345678901234567890.123456789"},
	}
	var cases = make([]ReportPresentationAcceptanceCase, 0, len(values))
	for _, value := range values {
		cases = append(cases, newPresentationScalarCase("quantity/"+value.ID, ReportPresentationCaseKindQuantity, "quantity", value))
	}
	return cases
}

// newPresentationRateCases creates the normalized-rate lexical-scale controls.
// Authored by: OpenCode
func newPresentationRateCases() []ReportPresentationAcceptanceCase {
	var values = []presentationScalarCase{
		{ID: "0.86010", ExactValue: "0.86010", ExpectedValue: "0.8601"},
		{ID: "16.9140", ExactValue: "16.9140", ExpectedValue: "16.914"},
		{ID: "1.094600", ExactValue: "1.094600", ExpectedValue: "1.0946"},
		{ID: "1.0900", ExactValue: "1.0900", ExpectedValue: "1.09"},
		{ID: "2.00", ExactValue: "2.00", ExpectedValue: "2"},
	}
	var cases = make([]ReportPresentationAcceptanceCase, 0, len(values))
	for _, value := range values {
		cases = append(cases, newPresentationScalarCase("rate/"+value.ID, ReportPresentationCaseKindRate, "rate", value))
	}
	return cases
}

// newPresentationBooleanCases creates both structured boolean states.
// Authored by: OpenCode
func newPresentationBooleanCases() []ReportPresentationAcceptanceCase {
	var cases = make([]ReportPresentationAcceptanceCase, 0, 2)
	for _, value := range []struct {
		id    string
		value bool
	}{
		{id: "true", value: true},
		{id: "false", value: false},
	} {
		var acceptanceCase = newPresentationScalarCase(
			"boolean/"+value.id,
			ReportPresentationCaseKindBoolean,
			"full_liquidation_event",
			presentationScalarCase{ID: value.id},
		)
		acceptanceCase.BooleanValue = value.value
		acceptanceCase.HasBooleanValue = true
		if value.value {
			acceptanceCase.ExpectedVisibleValue = "Yes"
		} else {
			acceptanceCase.ExpectedVisibleValue = "No"
		}
		cases = append(cases, acceptanceCase)
	}
	return cases
}

// newPresentationCurrencyCases creates classified and unclassified Annex
// currency controls, including a non-zero value that displays as 0.00.
// Authored by: OpenCode
func newPresentationCurrencyCases() []ReportPresentationAcceptanceCase {
	var cases = make([]ReportPresentationAcceptanceCase, 0, 3)
	var classified = newPresentationScalarCase(
		"currency/classified-zero-priced",
		ReportPresentationCaseKindCurrency,
		"original_activity_currency",
		presentationScalarCase{ID: "classified-zero-priced", ExactValue: "0", ExpectedValue: "0.00"},
	)
	classified.IsZeroPricedHoldingReduction = true
	classified.HasZeroPricedClassification = true
	classified.PreFormatActivityCurrency = "USD"
	classified.VisibleOriginalActivityCurrency = ""
	appendPresentationCurrencyOccurrences(&classified)
	cases = append(cases, classified)

	var priced = newPresentationScalarCase(
		"currency/unclassified-priced",
		ReportPresentationCaseKindCurrency,
		"original_activity_currency",
		presentationScalarCase{ID: "unclassified-priced", ExactValue: "1", ExpectedValue: "1.00"},
	)
	priced.HasZeroPricedClassification = true
	priced.PreFormatActivityCurrency = "EUR"
	priced.VisibleOriginalActivityCurrency = "EUR"
	appendPresentationCurrencyOccurrences(&priced)
	cases = append(cases, priced)

	var tinyPositive = newPresentationScalarCase(
		"currency/unclassified-tiny-positive",
		ReportPresentationCaseKindCurrency,
		"original_activity_currency",
		presentationScalarCase{ID: "unclassified-tiny-positive", ExactValue: "0.004", ExpectedValue: "0.00"},
	)
	tinyPositive.HasZeroPricedClassification = true
	tinyPositive.PreFormatActivityCurrency = "GBP"
	tinyPositive.VisibleOriginalActivityCurrency = "GBP"
	appendPresentationCurrencyOccurrences(&tinyPositive)
	cases = append(cases, tinyPositive)
	return cases
}

// newPresentationConvertedCases creates the eight order-preserving conversion
// subsequences from FR-019.
// Authored by: OpenCode
func newPresentationConvertedCases() []ReportPresentationAcceptanceCase {
	var values = []struct {
		id    string
		kinds []string
	}{
		{id: "empty"},
		{id: "unit-price", kinds: []string{"unit_price"}},
		{id: "gross-value", kinds: []string{"gross_value"}},
		{id: "fee-amount", kinds: []string{"fee_amount"}},
		{id: "unit-price-gross-value", kinds: []string{"unit_price", "gross_value"}},
		{id: "unit-price-fee-amount", kinds: []string{"unit_price", "fee_amount"}},
		{id: "gross-value-fee-amount", kinds: []string{"gross_value", "fee_amount"}},
		{id: "all", kinds: []string{"unit_price", "gross_value", "fee_amount"}},
	}
	var cases = make([]ReportPresentationAcceptanceCase, 0, len(values))
	for _, value := range values {
		var acceptanceCase = newPresentationCase(ReportPresentationAcceptanceCase{
			ID:         "converted/" + value.id,
			Kind:       ReportPresentationCaseKindConverted,
			Section:    "currency_conversion_audit",
			VectorCase: value.id,
		})
		acceptanceCase.ConvertedAmountKinds = append([]string(nil), value.kinds...)
		appendPresentationConvertedOccurrences(&acceptanceCase)
		cases = append(cases, acceptanceCase)
	}
	return cases
}

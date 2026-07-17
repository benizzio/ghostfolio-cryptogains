// Package testutil contains deterministic fixtures for report presentation
// acceptance tests.
// Authored by: OpenCode
package testutil

// ReportPresentationFormat identifies one supported report-rendering attempt.
// Authored by: OpenCode
type ReportPresentationFormat string

const (
	// ReportPresentationLegalWarningText is the exact standalone warning used
	// by both main-report formats.
	// Authored by: OpenCode
	ReportPresentationLegalWarningText = "The data in this report does not follow any legally required rules for any country's tax returns and is for reference only."
	// ReportPresentationFormatMarkdown identifies the Markdown bundle attempt.
	// Authored by: OpenCode
	ReportPresentationFormatMarkdown ReportPresentationFormat = "markdown"
	// ReportPresentationFormatPDF identifies the combined PDF attempt.
	// Authored by: OpenCode
	ReportPresentationFormatPDF ReportPresentationFormat = "pdf"
	// reportPresentationFormatCrossFormat identifies a parity comparison.
	// Authored by: OpenCode
	reportPresentationFormatCrossFormat ReportPresentationFormat = "cross-format"
)

// ReportPresentationDocumentRole identifies the logical report section used by
// a semantic occurrence key.
// Authored by: OpenCode
type ReportPresentationDocumentRole string

const (
	// ReportPresentationDocumentRoleMain identifies the main report section.
	// Authored by: OpenCode
	ReportPresentationDocumentRoleMain ReportPresentationDocumentRole = "main"
	// ReportPresentationDocumentRoleAnnex identifies the Annex 1 section.
	// Authored by: OpenCode
	ReportPresentationDocumentRoleAnnex ReportPresentationDocumentRole = "annex"
	// ReportPresentationDocumentRoleCombined identifies the combined PDF document.
	// Authored by: OpenCode
	ReportPresentationDocumentRoleCombined ReportPresentationDocumentRole = "combined"
	// reportPresentationDocumentRoleModel identifies a model comparison.
	// Authored by: OpenCode
	reportPresentationDocumentRoleModel ReportPresentationDocumentRole = "model"
)

// ReportPresentationCaseKind identifies one closed acceptance-case family.
// Authored by: OpenCode
type ReportPresentationCaseKind string

const (
	// ReportPresentationCaseKindWarning identifies the wrapped-warning case.
	// Authored by: OpenCode
	ReportPresentationCaseKindWarning ReportPresentationCaseKind = "warning"
	// ReportPresentationCaseKindFinancial identifies a matrix financial case.
	// Authored by: OpenCode
	ReportPresentationCaseKindFinancial ReportPresentationCaseKind = "financial"
	// ReportPresentationCaseKindQuantity identifies a quantity case.
	// Authored by: OpenCode
	ReportPresentationCaseKindQuantity ReportPresentationCaseKind = "quantity"
	// ReportPresentationCaseKindRate identifies a normalized-rate case.
	// Authored by: OpenCode
	ReportPresentationCaseKindRate ReportPresentationCaseKind = "rate"
	// ReportPresentationCaseKindBoolean identifies a structured-boolean case.
	// Authored by: OpenCode
	ReportPresentationCaseKindBoolean ReportPresentationCaseKind = "boolean"
	// ReportPresentationCaseKindCurrency identifies an audit-currency case.
	// Authored by: OpenCode
	ReportPresentationCaseKindCurrency ReportPresentationCaseKind = "currency"
	// ReportPresentationCaseKindConverted identifies a conversion-sequence case.
	// Authored by: OpenCode
	ReportPresentationCaseKindConverted ReportPresentationCaseKind = "converted"
)

// ReportPresentationPopulation identifies an acceptance denominator.
// Authored by: OpenCode
type ReportPresentationPopulation string

const (
	// ReportPresentationPopulationWarning identifies warning occurrences.
	// Authored by: OpenCode
	ReportPresentationPopulationWarning ReportPresentationPopulation = "W"
	// ReportPresentationPopulationVisibleFinancial identifies present financial fields.
	// Authored by: OpenCode
	ReportPresentationPopulationVisibleFinancial ReportPresentationPopulation = "V"
	// ReportPresentationPopulationModelIntegrity identifies model comparisons.
	// Authored by: OpenCode
	ReportPresentationPopulationModelIntegrity ReportPresentationPopulation = "M"
	// ReportPresentationPopulationQuantity identifies quantity occurrences.
	// Authored by: OpenCode
	ReportPresentationPopulationQuantity ReportPresentationPopulation = "Q"
	// ReportPresentationPopulationBoolean identifies structured boolean occurrences.
	// Authored by: OpenCode
	ReportPresentationPopulationBoolean ReportPresentationPopulation = "B"
	// ReportPresentationPopulationClassifiedCurrency identifies classified currency controls.
	// Authored by: OpenCode
	ReportPresentationPopulationClassifiedCurrency ReportPresentationPopulation = "Z"
	// ReportPresentationPopulationUnclassified identifies unclassified currency controls.
	// Authored by: OpenCode
	ReportPresentationPopulationUnclassified ReportPresentationPopulation = "N"
	// ReportPresentationPopulationConversionRow identifies conversion-row occurrences.
	// Authored by: OpenCode
	ReportPresentationPopulationConversionRow ReportPresentationPopulation = "C"
	// ReportPresentationPopulationParity identifies cross-format parity items.
	// Authored by: OpenCode
	ReportPresentationPopulationParity ReportPresentationPopulation = "P"
	// ReportPresentationPopulationConvertedEntry identifies included conversion entries.
	// Authored by: OpenCode
	ReportPresentationPopulationConvertedEntry ReportPresentationPopulation = "E"
)

// ReportPresentationFinancialField describes one semantic financial field in a
// matrix row, including its amount kind and ordinal within a repeated group.
// Authored by: OpenCode
type ReportPresentationFinancialField struct {
	Name          string
	AmountKind    string
	AmountOrdinal int
}

// ReportPresentationFormatAttempt describes one format attempt for an
// acceptance case. Markdown has a main document and a separate Annex; PDF has
// one combined document with both logical sections.
// Authored by: OpenCode
type ReportPresentationFormatAttempt struct {
	Format        ReportPresentationFormat
	DocumentRoles []ReportPresentationDocumentRole
}

// ReportPresentationOccurrenceKey identifies one semantic occurrence without
// relying on substring counts in generated document text.
// Authored by: OpenCode
type ReportPresentationOccurrenceKey struct {
	Population          ReportPresentationPopulation
	CaseID              string
	Format              ReportPresentationFormat
	DocumentRole        ReportPresentationDocumentRole
	Section             string
	AssetIdentity       string
	SourceOrRowIdentity string
	FieldName           string
	AmountKind          string
	AmountOrdinal       int
}

// ReportPresentationAcceptanceCase stores one closed case, its exact source
// control, both format attempts, and all semantic occurrence keys expected from
// those attempts.
// Authored by: OpenCode
type ReportPresentationAcceptanceCase struct {
	ID                              string
	Kind                            ReportPresentationCaseKind
	Section                         string
	FinancialFieldClass             string
	VectorCase                      string
	ExactValue                      string
	ExpectedVisibleValue            string
	ExpectedText                    string
	Absent                          bool
	Omitted                         bool
	BooleanValue                    bool
	HasBooleanValue                 bool
	IsZeroPricedHoldingReduction    bool
	HasZeroPricedClassification     bool
	PreFormatActivityCurrency       string
	VisibleOriginalActivityCurrency string
	FinancialFields                 []ReportPresentationFinancialField
	OmittedFieldNames               []string
	ConvertedAmountKinds            []string
	Attempts                        []ReportPresentationFormatAttempt
	OccurrenceKeys                  []ReportPresentationOccurrenceKey
}

// ReportPresentationAcceptanceCounters reports the derived denominators for
// the closed acceptance manifest. A is the case count; the remaining fields are
// counted from semantic occurrence keys.
// Authored by: OpenCode
type ReportPresentationAcceptanceCounters struct {
	A int
	W int
	V int
	M int
	Q int
	B int
	Z int
	N int
	C int
	P int
	E int
}

// ReportPresentationAcceptanceManifest contains the immutable-shape closed
// case set and its semantic population counters.
// Authored by: OpenCode
type ReportPresentationAcceptanceManifest struct {
	Cases    []ReportPresentationAcceptanceCase
	Counters ReportPresentationAcceptanceCounters
}

// DeterministicReportPresentationAcceptanceFixture returns the finalized closed
// acceptance manifest required by the final report rendering contract.
//
// Every case has exactly one Markdown attempt and one PDF attempt. The returned
// slices are newly allocated on each call, so a test may annotate its copy
// without changing another test's fixture.
//
// Example usage:
//
//	manifest := testutil.DeterministicReportPresentationAcceptanceFixture()
//	if manifest.Counters.A == 0 {
//		t.Fatal("acceptance manifest is empty")
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
	var acceptanceCase = newPresentationCase(
		"warning/wrapped",
		ReportPresentationCaseKindWarning,
		"warning",
		"wrapped",
		"",
		"",
		false,
		false,
		nil,
	)
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
			var acceptanceCase = newPresentationCase(
				"financial/"+row.ID+"/"+vector.ID,
				ReportPresentationCaseKindFinancial,
				row.ID,
				vector.ID,
				vector.ExactValue,
				vector.ExpectedValue,
				false,
				false,
				row.Fields,
			)
			acceptanceCase.FinancialFieldClass = row.ID
			appendPresentationFinancialOccurrences(&acceptanceCase, row)
			cases = append(cases, acceptanceCase)
		}
		if row.Nullable {
			var absentCase = newPresentationCase(
				"financial/"+row.ID+"/absent",
				ReportPresentationCaseKindFinancial,
				row.ID,
				"absent",
				"",
				"",
				true,
				false,
				row.Fields,
			)
			absentCase.FinancialFieldClass = row.ID
			appendPresentationFinancialOccurrences(&absentCase, row)
			cases = append(cases, absentCase)
		}
		if row.SummaryOmission {
			var omittedCase = newPresentationCase(
				"financial/"+row.ID+"/exact-zero-omitted",
				ReportPresentationCaseKindFinancial,
				row.ID,
				"exact-zero-omitted",
				"0",
				"",
				false,
				true,
				row.Fields,
			)
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
		var acceptanceCase = newPresentationCase(
			"converted/"+value.id,
			ReportPresentationCaseKindConverted,
			"currency_conversion_audit",
			value.id,
			"",
			"",
			false,
			false,
			nil,
		)
		acceptanceCase.ConvertedAmountKinds = append([]string(nil), value.kinds...)
		appendPresentationConvertedOccurrences(&acceptanceCase)
		cases = append(cases, acceptanceCase)
	}
	return cases
}

// newPresentationScalarCase builds a non-matrix acceptance case and its base
// warning, model, and parity occurrences.
// Authored by: OpenCode
func newPresentationScalarCase(id string, kind ReportPresentationCaseKind, fieldName string, value presentationScalarCase) ReportPresentationAcceptanceCase {
	var acceptanceCase = newPresentationCase(id, kind, fieldName, value.ID, value.ExactValue, value.ExpectedValue, false, false, nil)
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
		ReportPresentationPopulationParity,
		acceptanceCase.ID,
		reportPresentationFormatCrossFormat,
		reportPresentationDocumentRoleModel,
		"acceptance_control",
		"acceptance",
		fieldName,
		"",
		"",
		0,
	)...)
	if kind == ReportPresentationCaseKindQuantity {
		acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
			ReportPresentationPopulationQuantity,
			acceptanceCase.ID,
			ReportPresentationFormatMarkdown,
			ReportPresentationDocumentRoleMain,
			"quantity_controls",
			"acceptance",
			acceptanceCase.ID,
			fieldName,
			"",
			0,
		)...)
		acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
			ReportPresentationPopulationQuantity,
			acceptanceCase.ID,
			ReportPresentationFormatPDF,
			ReportPresentationDocumentRoleMain,
			"quantity_controls",
			"acceptance",
			acceptanceCase.ID,
			fieldName,
			"",
			0,
		)...)
	}
	if kind == ReportPresentationCaseKindBoolean {
		acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
			ReportPresentationPopulationBoolean,
			acceptanceCase.ID,
			ReportPresentationFormatMarkdown,
			ReportPresentationDocumentRoleAnnex,
			"detailed_per_asset_audit",
			"acceptance",
			acceptanceCase.ID,
			fieldName,
			"",
			0,
		)...)
		acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
			ReportPresentationPopulationBoolean,
			acceptanceCase.ID,
			ReportPresentationFormatPDF,
			ReportPresentationDocumentRoleAnnex,
			"detailed_per_asset_audit",
			"acceptance",
			acceptanceCase.ID,
			fieldName,
			"",
			0,
		)...)
	}
	return acceptanceCase
}

// appendPresentationFinancialOccurrences adds visible-field and parity keys for
// one matrix case, excluding absent and inherited omitted values from V.
// Authored by: OpenCode
func appendPresentationFinancialOccurrences(acceptanceCase *ReportPresentationAcceptanceCase, row presentationFinancialRow) {
	var documentRole = ReportPresentationDocumentRoleMain
	if row.DocumentRole != "" {
		documentRole = row.DocumentRole
	}
	for _, field := range row.Fields {
		if !acceptanceCase.Absent && !presentationFieldIsOmitted(*acceptanceCase, field.Name) {
			acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
				ReportPresentationPopulationVisibleFinancial,
				acceptanceCase.ID,
				ReportPresentationFormatMarkdown,
				documentRole,
				row.Section,
				"acceptance",
				acceptanceCase.ID,
				field.Name,
				field.AmountKind,
				field.AmountOrdinal,
			)...)
			acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
				ReportPresentationPopulationVisibleFinancial,
				acceptanceCase.ID,
				ReportPresentationFormatPDF,
				documentRole,
				row.Section,
				"acceptance",
				acceptanceCase.ID,
				field.Name,
				field.AmountKind,
				field.AmountOrdinal,
			)...)
		}
		acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, ReportPresentationOccurrenceKey{
			Population:          ReportPresentationPopulationParity,
			CaseID:              acceptanceCase.ID,
			Format:              reportPresentationFormatCrossFormat,
			DocumentRole:        documentRole,
			Section:             row.Section,
			AssetIdentity:       "acceptance",
			SourceOrRowIdentity: acceptanceCase.ID,
			FieldName:           field.Name,
			AmountKind:          field.AmountKind,
			AmountOrdinal:       field.AmountOrdinal,
		})
	}
}

// presentationFieldIsOmitted reports whether a field is the inherited exact
// zero summary row omitted from visible presentation.
// Authored by: OpenCode
func presentationFieldIsOmitted(acceptanceCase ReportPresentationAcceptanceCase, fieldName string) bool {
	for _, omittedFieldName := range acceptanceCase.OmittedFieldNames {
		if omittedFieldName == fieldName {
			return true
		}
	}
	return false
}

// appendPresentationCurrencyOccurrences adds currency applicability and the
// monetary source-price control for one Annex case.
// Authored by: OpenCode
func appendPresentationCurrencyOccurrences(acceptanceCase *ReportPresentationAcceptanceCase) {
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
		ReportPresentationPopulationVisibleFinancial,
		acceptanceCase.ID,
		ReportPresentationFormatMarkdown,
		ReportPresentationDocumentRoleAnnex,
		"detailed_per_asset_audit",
		"acceptance",
		acceptanceCase.ID,
		"unit_price",
		"unit_price",
		0,
	)...)
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
		ReportPresentationPopulationVisibleFinancial,
		acceptanceCase.ID,
		ReportPresentationFormatPDF,
		ReportPresentationDocumentRoleAnnex,
		"detailed_per_asset_audit",
		"acceptance",
		acceptanceCase.ID,
		"unit_price",
		"unit_price",
		0,
	)...)
	var population = ReportPresentationPopulationUnclassified
	if acceptanceCase.IsZeroPricedHoldingReduction {
		population = ReportPresentationPopulationClassifiedCurrency
	}
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
		population,
		acceptanceCase.ID,
		ReportPresentationFormatMarkdown,
		ReportPresentationDocumentRoleAnnex,
		"detailed_per_asset_audit",
		"acceptance",
		acceptanceCase.ID,
		"original_activity_currency",
		"",
		0,
	)...)
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
		population,
		acceptanceCase.ID,
		ReportPresentationFormatPDF,
		ReportPresentationDocumentRoleAnnex,
		"detailed_per_asset_audit",
		"acceptance",
		acceptanceCase.ID,
		"original_activity_currency",
		"",
		0,
	)...)
}

// newPresentationCase builds the common case shape and warning/model evidence.
// Authored by: OpenCode
func newPresentationCase(
	id string,
	kind ReportPresentationCaseKind,
	section string,
	vectorCase string,
	exactValue string,
	expectedValue string,
	absent bool,
	omitted bool,
	financialFields []ReportPresentationFinancialField,
) ReportPresentationAcceptanceCase {
	var acceptanceCase = ReportPresentationAcceptanceCase{
		ID:                   id,
		Kind:                 kind,
		Section:              section,
		VectorCase:           vectorCase,
		ExactValue:           exactValue,
		ExpectedVisibleValue: expectedValue,
		Absent:               absent,
		Omitted:              omitted,
		FinancialFields:      append([]ReportPresentationFinancialField(nil), financialFields...),
		Attempts: []ReportPresentationFormatAttempt{
			{Format: ReportPresentationFormatMarkdown, DocumentRoles: []ReportPresentationDocumentRole{ReportPresentationDocumentRoleMain, ReportPresentationDocumentRoleAnnex}},
			{Format: ReportPresentationFormatPDF, DocumentRoles: []ReportPresentationDocumentRole{ReportPresentationDocumentRoleCombined}},
		},
	}
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
		ReportPresentationPopulationWarning,
		id,
		ReportPresentationFormatMarkdown,
		ReportPresentationDocumentRoleMain,
		"report_header",
		"acceptance",
		"legal_use_warning",
		"",
		"",
		0,
	)...)
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
		ReportPresentationPopulationWarning,
		id,
		ReportPresentationFormatPDF,
		ReportPresentationDocumentRoleMain,
		"report_header",
		"acceptance",
		"legal_use_warning",
		"",
		"",
		0,
	)...)
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
		ReportPresentationPopulationModelIntegrity,
		id,
		ReportPresentationFormatMarkdown,
		reportPresentationDocumentRoleModel,
		"AUD-001",
		"acceptance",
		"exact_model_before_after_render",
		"",
		"",
		0,
	)...)
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
		ReportPresentationPopulationModelIntegrity,
		id,
		ReportPresentationFormatPDF,
		reportPresentationDocumentRoleModel,
		"AUD-001",
		"acceptance",
		"exact_model_before_after_render",
		"",
		"",
		0,
	)...)
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, ReportPresentationOccurrenceKey{
		Population:          ReportPresentationPopulationParity,
		CaseID:              id,
		Format:              reportPresentationFormatCrossFormat,
		DocumentRole:        reportPresentationDocumentRoleModel,
		Section:             "acceptance_control",
		AssetIdentity:       "acceptance",
		SourceOrRowIdentity: "acceptance",
		FieldName:           "warning_and_metadata",
	})
	return acceptanceCase
}

// appendPresentationConvertedOccurrences adds conversion-row, entry, financial,
// and parity keys while preserving the received amount-kind order.
// Authored by: OpenCode
func appendPresentationConvertedOccurrences(acceptanceCase *ReportPresentationAcceptanceCase) {
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
		ReportPresentationPopulationConversionRow,
		acceptanceCase.ID,
		ReportPresentationFormatMarkdown,
		ReportPresentationDocumentRoleAnnex,
		"currency_conversion_audit",
		"conversion-row",
		"converted_amounts",
		"",
		"",
		0,
	)...)
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
		ReportPresentationPopulationConversionRow,
		acceptanceCase.ID,
		ReportPresentationFormatPDF,
		ReportPresentationDocumentRoleAnnex,
		"currency_conversion_audit",
		"conversion-row",
		"converted_amounts",
		"",
		"",
		0,
	)...)
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, ReportPresentationOccurrenceKey{
		Population:          ReportPresentationPopulationParity,
		CaseID:              acceptanceCase.ID,
		Format:              reportPresentationFormatCrossFormat,
		DocumentRole:        ReportPresentationDocumentRoleAnnex,
		Section:             "currency_conversion_audit",
		AssetIdentity:       "conversion-row",
		SourceOrRowIdentity: acceptanceCase.ID,
		FieldName:           "converted_amount_sequence",
	})
	for amountOrdinal, amountKind := range acceptanceCase.ConvertedAmountKinds {
		var fieldName = "original_" + amountKind
		acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
			ReportPresentationPopulationVisibleFinancial,
			acceptanceCase.ID,
			ReportPresentationFormatMarkdown,
			ReportPresentationDocumentRoleAnnex,
			"currency_conversion_audit",
			"conversion-row",
			acceptanceCase.ID,
			fieldName,
			amountKind,
			amountOrdinal*2,
		)...)
		var convertedFieldName = "converted_" + amountKind
		acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
			ReportPresentationPopulationVisibleFinancial,
			acceptanceCase.ID,
			ReportPresentationFormatMarkdown,
			ReportPresentationDocumentRoleAnnex,
			"currency_conversion_audit",
			"conversion-row",
			acceptanceCase.ID,
			convertedFieldName,
			amountKind,
			amountOrdinal*2+1,
		)...)
		acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
			ReportPresentationPopulationVisibleFinancial,
			acceptanceCase.ID,
			ReportPresentationFormatPDF,
			ReportPresentationDocumentRoleAnnex,
			"currency_conversion_audit",
			"conversion-row",
			acceptanceCase.ID,
			convertedFieldName,
			amountKind,
			amountOrdinal*2+1,
		)...)
		acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
			ReportPresentationPopulationVisibleFinancial,
			acceptanceCase.ID,
			ReportPresentationFormatPDF,
			ReportPresentationDocumentRoleAnnex,
			"currency_conversion_audit",
			"conversion-row",
			acceptanceCase.ID,
			fieldName,
			amountKind,
			amountOrdinal*2,
		)...)
		acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
			ReportPresentationPopulationConvertedEntry,
			acceptanceCase.ID,
			ReportPresentationFormatMarkdown,
			ReportPresentationDocumentRoleAnnex,
			"currency_conversion_audit",
			"conversion-row",
			acceptanceCase.ID,
			amountKind,
			amountKind,
			amountOrdinal,
		)...)
		acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, formatOccurrenceKeys(
			ReportPresentationPopulationConvertedEntry,
			acceptanceCase.ID,
			ReportPresentationFormatPDF,
			ReportPresentationDocumentRoleAnnex,
			"currency_conversion_audit",
			"conversion-row",
			acceptanceCase.ID,
			amountKind,
			amountKind,
			amountOrdinal,
		)...)
	}
}

// formatOccurrenceKeys creates one key for each requested format while keeping
// the logical document role stable between Markdown and PDF.
// Authored by: OpenCode
func formatOccurrenceKeys(
	population ReportPresentationPopulation,
	caseID string,
	format ReportPresentationFormat,
	documentRole ReportPresentationDocumentRole,
	section string,
	assetIdentity string,
	sourceOrRowIdentity string,
	fieldName string,
	amountKind string,
	amountOrdinal int,
) []ReportPresentationOccurrenceKey {
	if format == reportPresentationFormatCrossFormat {
		return []ReportPresentationOccurrenceKey{{
			Population:          population,
			CaseID:              caseID,
			Format:              format,
			DocumentRole:        documentRole,
			Section:             section,
			AssetIdentity:       assetIdentity,
			SourceOrRowIdentity: sourceOrRowIdentity,
			FieldName:           fieldName,
			AmountKind:          amountKind,
			AmountOrdinal:       amountOrdinal,
		}}
	}
	return []ReportPresentationOccurrenceKey{
		{
			Population:          population,
			CaseID:              caseID,
			Format:              format,
			DocumentRole:        documentRole,
			Section:             section,
			AssetIdentity:       assetIdentity,
			SourceOrRowIdentity: sourceOrRowIdentity,
			FieldName:           fieldName,
			AmountKind:          amountKind,
			AmountOrdinal:       amountOrdinal,
		},
	}
}

// countPresentationOccurrences derives every requested counter from the case
// manifest, retaining failed attempts in the denominator represented by keys.
// Authored by: OpenCode
func countPresentationOccurrences(cases []ReportPresentationAcceptanceCase) ReportPresentationAcceptanceCounters {
	var counters = ReportPresentationAcceptanceCounters{A: len(cases)}
	for _, acceptanceCase := range cases {
		for _, occurrence := range acceptanceCase.OccurrenceKeys {
			switch occurrence.Population {
			case ReportPresentationPopulationWarning:
				counters.W++
			case ReportPresentationPopulationVisibleFinancial:
				counters.V++
			case ReportPresentationPopulationModelIntegrity:
				counters.M++
			case ReportPresentationPopulationQuantity:
				counters.Q++
			case ReportPresentationPopulationBoolean:
				counters.B++
			case ReportPresentationPopulationClassifiedCurrency:
				counters.Z++
			case ReportPresentationPopulationUnclassified:
				counters.N++
			case ReportPresentationPopulationConversionRow:
				counters.C++
			case ReportPresentationPopulationParity:
				counters.P++
			case ReportPresentationPopulationConvertedEntry:
				counters.E++
			}
		}
	}
	return counters
}

// presentationFinancialRow describes one row in the closed financial matrix.
// Authored by: OpenCode
type presentationFinancialRow struct {
	ID              string
	Section         string
	DocumentRole    ReportPresentationDocumentRole
	Signed          bool
	Nullable        bool
	SummaryOmission bool
	Fields          []ReportPresentationFinancialField
}

// presentationFinancialRows returns all nine financial field classes from the
// specification's matrix.
// Authored by: OpenCode
func presentationFinancialRows() []presentationFinancialRow {
	return []presentationFinancialRow{
		{
			ID:              "summary-net-gain-or-loss",
			Section:         "gains_and_losses_summary",
			DocumentRole:    ReportPresentationDocumentRoleMain,
			Signed:          true,
			SummaryOmission: true,
			Fields: []ReportPresentationFinancialField{
				{Name: "per_asset_net_gain_or_loss", AmountKind: "gain_or_loss", AmountOrdinal: 0},
				{Name: "overall_yearly_net_total", AmountKind: "gain_or_loss", AmountOrdinal: 1},
			},
		},
		{
			ID:           "position-cost-basis",
			Section:      "position",
			DocumentRole: ReportPresentationDocumentRoleMain,
			Fields: []ReportPresentationFinancialField{
				{Name: "opening_cost_basis", AmountKind: "cost_basis", AmountOrdinal: 0},
				{Name: "closing_cost_basis", AmountKind: "cost_basis", AmountOrdinal: 1},
				{Name: "historical_cost_basis", AmountKind: "cost_basis", AmountOrdinal: 2},
			},
		},
		{
			ID:           "in-year-activity",
			Section:      "in_year_activity",
			DocumentRole: ReportPresentationDocumentRoleMain,
			Nullable:     true,
			Fields: []ReportPresentationFinancialField{
				{Name: "unit_price", AmountKind: "unit_price", AmountOrdinal: 0},
				{Name: "gross_value", AmountKind: "gross_value", AmountOrdinal: 1},
				{Name: "fee_amount", AmountKind: "fee_amount", AmountOrdinal: 2},
				{Name: "basis_after_row", AmountKind: "cost_basis", AmountOrdinal: 3},
			},
		},
		{
			ID:           "liquidation-allocated-basis",
			Section:      "liquidation_calculations",
			DocumentRole: ReportPresentationDocumentRoleMain,
			Nullable:     true,
			Fields:       []ReportPresentationFinancialField{{Name: "allocated_basis", AmountKind: "cost_basis", AmountOrdinal: 0}},
		},
		{
			ID:           "liquidation-net-proceeds-gain-or-loss",
			Section:      "liquidation_calculations",
			DocumentRole: ReportPresentationDocumentRoleMain,
			Signed:       true,
			Nullable:     true,
			Fields: []ReportPresentationFinancialField{
				{Name: "net_proceeds", AmountKind: "proceeds", AmountOrdinal: 0},
				{Name: "gain_or_loss", AmountKind: "gain_or_loss", AmountOrdinal: 1},
			},
		},
		{
			ID:           "audit-activity",
			Section:      "detailed_per_asset_audit",
			DocumentRole: ReportPresentationDocumentRoleAnnex,
			Nullable:     true,
			Fields: []ReportPresentationFinancialField{
				{Name: "unit_price", AmountKind: "unit_price", AmountOrdinal: 0},
				{Name: "gross_value", AmountKind: "gross_value", AmountOrdinal: 1},
				{Name: "fee_amount", AmountKind: "fee_amount", AmountOrdinal: 2},
				{Name: "basis_after_activity", AmountKind: "cost_basis", AmountOrdinal: 3},
			},
		},
		{
			ID:           "audit-allocated-basis",
			Section:      "detailed_per_asset_audit",
			DocumentRole: ReportPresentationDocumentRoleAnnex,
			Nullable:     true,
			Fields:       []ReportPresentationFinancialField{{Name: "allocated_basis", AmountKind: "cost_basis", AmountOrdinal: 0}},
		},
		{
			ID:           "audit-net-proceeds-gain-or-loss",
			Section:      "detailed_per_asset_audit",
			DocumentRole: ReportPresentationDocumentRoleAnnex,
			Signed:       true,
			Nullable:     true,
			Fields: []ReportPresentationFinancialField{
				{Name: "net_proceeds", AmountKind: "proceeds", AmountOrdinal: 0},
				{Name: "gain_or_loss", AmountKind: "gain_or_loss", AmountOrdinal: 1},
			},
		},
		{
			ID:           "conversion-amount",
			Section:      "currency_conversion_audit",
			DocumentRole: ReportPresentationDocumentRoleAnnex,
			Fields: []ReportPresentationFinancialField{
				{Name: "original_unit_price", AmountKind: "unit_price", AmountOrdinal: 0},
				{Name: "converted_unit_price", AmountKind: "unit_price", AmountOrdinal: 1},
				{Name: "original_gross_value", AmountKind: "gross_value", AmountOrdinal: 2},
				{Name: "converted_gross_value", AmountKind: "gross_value", AmountOrdinal: 3},
				{Name: "original_fee_amount", AmountKind: "fee_amount", AmountOrdinal: 4},
				{Name: "converted_fee_amount", AmountKind: "fee_amount", AmountOrdinal: 5},
			},
		},
	}
}

// presentationScalarCase stores one exact-value control used outside the
// financial matrix.
// Authored by: OpenCode
type presentationScalarCase struct {
	ID            string
	ExactValue    string
	ExpectedValue string
}

// presentationNonNegativeVectors returns the complete non-negative financial
// rounding vector in specification order.
// Authored by: OpenCode
func presentationNonNegativeVectors() []presentationScalarCase {
	return []presentationScalarCase{
		{ID: "zero", ExactValue: "0", ExpectedValue: "0.00"},
		{ID: "tiny-positive", ExactValue: "0.00000001", ExpectedValue: "0.00"},
		{ID: "whole", ExactValue: "1", ExpectedValue: "1.00"},
		{ID: "one-place", ExactValue: "1.2", ExpectedValue: "1.20"},
		{ID: "two-place", ExactValue: "1.23", ExpectedValue: "1.23"},
		{ID: "below-positive-tie", ExactValue: "1.004", ExpectedValue: "1.00"},
		{ID: "positive-tie", ExactValue: "1.005", ExpectedValue: "1.01"},
		{ID: "above-positive-tie", ExactValue: "1.006", ExpectedValue: "1.01"},
		{ID: "positive-carry", ExactValue: "9.995", ExpectedValue: "10.00"},
		{ID: "large-positive", ExactValue: "12345678901234567890.123456789", ExpectedValue: "12345678901234567890.12"},
	}
}

// presentationSignedVectors extends the non-negative vector with every signed
// boundary required by the financial display contract.
// Authored by: OpenCode
func presentationSignedVectors() []presentationScalarCase {
	var vectors = append([]presentationScalarCase(nil), presentationNonNegativeVectors()...)
	vectors = append(vectors,
		presentationScalarCase{ID: "negative-whole", ExactValue: "-1", ExpectedValue: "-1.00"},
		presentationScalarCase{ID: "negative-below-tie", ExactValue: "-1.004", ExpectedValue: "-1.00"},
		presentationScalarCase{ID: "negative-tie", ExactValue: "-1.005", ExpectedValue: "-1.01"},
		presentationScalarCase{ID: "negative-above-tie", ExactValue: "-1.006", ExpectedValue: "-1.01"},
		presentationScalarCase{ID: "negative-carry", ExactValue: "-9.995", ExpectedValue: "-10.00"},
		presentationScalarCase{ID: "signed-zero", ExactValue: "-0", ExpectedValue: "0.00"},
		presentationScalarCase{ID: "negative-tiny", ExactValue: "-0.004", ExpectedValue: "0.00"},
		presentationScalarCase{ID: "negative-zero-adjacent-tie", ExactValue: "-0.005", ExpectedValue: "-0.01"},
		presentationScalarCase{ID: "large-negative", ExactValue: "-12345678901234567890.123456789", ExpectedValue: "-12345678901234567890.12"},
	)
	return vectors
}

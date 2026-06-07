package fixture

import (
	"encoding/json"
	"reflect"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// TestEmpiricalModelEnumValuesMatchSpecification verifies the foundational
// shared model enums keep the exact string values defined by the empirical
// specification.
// Authored by: OpenCode
func TestEmpiricalModelEnumValuesMatchSpecification(t *testing.T) {
	t.Parallel()

	var testCases = []struct {
		name string
		got  string
		want string
	}{
		{name: "oracle_support_supported", got: string(OracleSupportSupported), want: "supported"},
		{name: "oracle_support_partially_supported", got: string(OracleSupportPartiallySupported), want: "partially_supported"},
		{name: "oracle_support_unsupported", got: string(OracleSupportUnsupported), want: "unsupported"},
		{name: "evidence_support_label_hledger_backed", got: string(EvidenceSupportLabelHledgerBacked), want: "hledger_backed"},
		{name: "evidence_support_label_project_composition_rule", got: string(EvidenceSupportLabelProjectCompositionRule), want: "project_composition_rule"},
		{name: "comparison_policy_skip_external_oracle", got: string(ComparisonPolicySkipExternalOracle), want: "skip_external_oracle"},
		{name: "comparison_policy_project_composition_only", got: string(ComparisonPolicyProjectCompositionOnly), want: "project_composition_only"},
		{name: "comparison_policy_fail_if_selected", got: string(ComparisonPolicyFailIfSelected), want: "fail_if_selected"},
	}

	for _, testCase := range testCases {
		var testCase = testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if testCase.got != testCase.want {
				t.Fatalf("unexpected enum value: got %q want %q", testCase.got, testCase.want)
			}
		})
	}
}

// TestEmpiricalDatasetActivityCaseModelContracts verifies the shared dataset,
// activity, scope, and case structs keep the exact foundational field types and
// serialization tags required by the empirical dataset contract.
// Authored by: OpenCode
func TestEmpiricalDatasetActivityCaseModelContracts(t *testing.T) {
	t.Parallel()

	var datasetType = reflect.TypeOf(EmpiricalDataset{})
	assertStructFieldTypeAndTags(t, datasetType, "DatasetVersion", reflect.TypeOf(""), "dataset_version", "dataset_version")
	assertStructFieldTypeAndTags(t, datasetType, "Currency", reflect.TypeOf(""), "currency", "currency")
	assertStructFieldTypeAndTags(t, datasetType, "SupportedYears", reflect.TypeOf([]int{}), "supported_years", "supported_years")
	assertStructFieldTypeAndTags(t, datasetType, "SupportedMethods", reflect.TypeOf([]reportmodel.CostBasisMethod{}), "supported_methods", "supported_methods")
	assertStructFieldTypeAndTags(t, datasetType, "Activities", reflect.TypeOf([]EmpiricalActivity{}), "activities", "activities")
	assertStructFieldTypeAndTags(t, datasetType, "Cases", reflect.TypeOf([]EmpiricalCase{}), "cases", "cases")

	var activityType = reflect.TypeOf(EmpiricalActivity{})
	assertStructFieldTypeAndTags(t, activityType, "SourceID", reflect.TypeOf(""), "source_id", "source_id")
	assertStructFieldTypeAndTags(t, activityType, "OccurredAt", reflect.TypeOf(""), "occurred_at", "occurred_at")
	assertStructFieldTypeAndTags(t, activityType, "DeterministicOrder", reflect.TypeOf(0), "deterministic_order", "deterministic_order")
	assertStructFieldTypeAndTags(t, activityType, "ActivityType", reflect.TypeOf(syncmodel.ActivityType("")), "activity_type", "activity_type")
	assertStructFieldTypeAndTags(t, activityType, "Quantity", reflect.TypeOf(""), "quantity", "quantity")
	assertStructFieldTypeAndTags(t, activityType, "GrossValue", reflect.TypeOf(""), "gross_value,omitempty", "gross_value,omitempty")
	assertStructFieldTypeAndTags(t, activityType, "UnitPrice", reflect.TypeOf(""), "unit_price,omitempty", "unit_price,omitempty")
	assertStructFieldTypeAndTags(t, activityType, "FeeAmount", reflect.TypeOf(""), "fee_amount,omitempty", "fee_amount,omitempty")
	assertStructFieldTypeAndTags(t, activityType, "Currency", reflect.TypeOf(""), "currency,omitempty", "currency,omitempty")
	assertStructFieldTypeAndTags(t, activityType, "SourceScope", reflect.TypeOf(&EmpiricalScope{}), "source_scope,omitempty", "source_scope,omitempty")
	assertStructFieldTypeAndTags(t, activityType, "ZeroPricedReductionExplanation", reflect.TypeOf(""), "zero_priced_reduction_explanation,omitempty", "zero_priced_reduction_explanation,omitempty")

	var scopeType = reflect.TypeOf(EmpiricalScope{})
	assertStructFieldTypeAndTags(t, scopeType, "ScopeID", reflect.TypeOf(""), "scope_id", "scope_id")
	assertStructFieldTypeAndTags(t, scopeType, "ScopeKind", reflect.TypeOf(syncmodel.SourceScopeKind("")), "scope_kind", "scope_kind")
	assertStructFieldTypeAndTags(t, scopeType, "Reliability", reflect.TypeOf(syncmodel.ScopeReliability("")), "reliability", "reliability")
	assertStructFieldTypeAndTags(t, scopeType, "DisplayName", reflect.TypeOf(""), "display_name,omitempty", "display_name,omitempty")

	var caseType = reflect.TypeOf(EmpiricalCase{})
	assertStructFieldTypeAndTags(t, caseType, "Methods", reflect.TypeOf([]reportmodel.CostBasisMethod{}), "methods", "methods")
	assertStructFieldTypeAndTags(t, caseType, "Year", reflect.TypeOf(0), "year", "year")
	assertStructFieldTypeAndTags(t, caseType, "AssetIdentityKeys", reflect.TypeOf([]string{}), "asset_identity_keys", "asset_identity_keys")
	assertStructFieldTypeAndTags(t, caseType, "ActivitySourceIDs", reflect.TypeOf([]string{}), "activity_source_ids", "activity_source_ids")
	assertStructFieldTypeAndTags(t, caseType, "OracleSupport", reflect.TypeOf(OracleSupport("")), "oracle_support", "oracle_support")
	assertStructFieldTypeAndTags(t, caseType, "UnsupportedReason", reflect.TypeOf(""), "unsupported_reason,omitempty", "unsupported_reason,omitempty")
}

// TestOracleAndComparisonModelContracts verifies the shared oracle, comparable
// output, project output, and comparison structs keep the field shapes needed by
// later parser, normalizer, and comparator work.
// Authored by: OpenCode
func TestOracleAndComparisonModelContracts(t *testing.T) {
	t.Parallel()

	var valuesType = reflect.TypeOf(ComparableOutputValues{})
	assertStructFieldTypeAndTags(t, valuesType, "RealizedGainOrLoss", reflect.TypeOf(""), "realized_gain_or_loss", "realized_gain_or_loss")
	assertStructFieldTypeAndTags(t, valuesType, "AllocatedBasis", reflect.TypeOf(""), "allocated_basis", "allocated_basis")
	assertStructFieldTypeAndTags(t, valuesType, "ClosingQuantity", reflect.TypeOf(""), "closing_quantity", "closing_quantity")
	assertStructFieldTypeAndTags(t, valuesType, "ClosingBasis", reflect.TypeOf(""), "closing_basis", "closing_basis")

	var metadataType = reflect.TypeOf(OracleGenerationRun{})
	assertStructFieldTypeAndTags(t, metadataType, "HledgerVersion", reflect.TypeOf(""), "hledger_version", "hledger_version")
	assertStructFieldTypeAndTags(t, metadataType, "CommandArguments", reflect.TypeOf([]string{}), "command_arguments", "command_arguments")
	assertStructFieldTypeAndTags(t, metadataType, "DecimalPolicy", reflect.TypeOf(""), "decimal_policy", "decimal_policy")
	assertStructFieldTypeAndTags(t, metadataType, "FinancialTolerances", reflect.TypeOf(map[string]string{}), "financial_tolerances", "financial_tolerances")
	assertStructFieldTypeAndTags(t, metadataType, "ToleranceNotes", reflect.TypeOf(map[string]string{}), "tolerance_notes", "tolerance_notes")

	var oracleType = reflect.TypeOf(OracleOutput{})
	assertStructFieldTypeAndTags(t, oracleType, "Method", reflect.TypeOf(reportmodel.CostBasisMethod("")), "method", "method")
	assertStructFieldTypeAndTags(t, oracleType, "Values", valuesType, "values", "values")
	assertStructFieldTypeAndTags(t, oracleType, "Matches", reflect.TypeOf([]OracleMatchEvidence{}), "matches", "matches")
	assertStructFieldTypeAndTags(t, oracleType, "UnsupportedSegments", reflect.TypeOf([]UnsupportedOracleSegment{}), "unsupported_segments", "unsupported_segments")
	assertStructFieldTypeAndTags(t, oracleType, "Metadata", metadataType, "metadata", "metadata")

	var oracleMatchType = reflect.TypeOf(OracleMatchEvidence{})
	assertStructFieldTypeAndTags(t, oracleMatchType, "DisposedSourceID", reflect.TypeOf(""), "disposed_source_id", "disposed_source_id")
	assertStructFieldTypeAndTags(t, oracleMatchType, "MatchedQuantity", reflect.TypeOf(""), "matched_quantity", "matched_quantity")
	assertStructFieldTypeAndTags(t, oracleMatchType, "MatchedBasis", reflect.TypeOf(""), "matched_basis", "matched_basis")
	assertStructFieldTypeAndTags(t, oracleMatchType, "SupportLabel", reflect.TypeOf(EvidenceSupportLabel("")), "support_label,omitempty", "support_label,omitempty")
	assertStructFieldTypeAndTags(t, oracleMatchType, "CompositionRuleID", reflect.TypeOf(""), "composition_rule_id,omitempty", "composition_rule_id,omitempty")

	var unsupportedType = reflect.TypeOf(UnsupportedOracleSegment{})
	assertStructFieldTypeAndTags(t, unsupportedType, "Method", reflect.TypeOf(reportmodel.CostBasisMethod("")), "method", "method")
	assertStructFieldTypeAndTags(t, unsupportedType, "ComparisonPolicy", reflect.TypeOf(ComparisonPolicy("")), "comparison_policy", "comparison_policy")

	var projectOutputType = reflect.TypeOf(ProjectCalculationOutput{})
	assertStructFieldTypeAndTags(t, projectOutputType, "Values", valuesType, "values", "values")
	assertStructFieldTypeAndTags(t, projectOutputType, "Matches", reflect.TypeOf([]ProjectMatchEvidence{}), "matches", "matches")

	var projectMatchType = reflect.TypeOf(ProjectMatchEvidence{})
	assertStructFieldTypeAndTags(t, projectMatchType, "MatchedQuantity", reflect.TypeOf(""), "matched_quantity", "matched_quantity")
	assertStructFieldTypeAndTags(t, projectMatchType, "MatchedBasis", reflect.TypeOf(""), "matched_basis", "matched_basis")
	assertStructFieldTypeAndTags(t, projectMatchType, "SupportLabel", reflect.TypeOf(EvidenceSupportLabel("")), "support_label,omitempty", "support_label,omitempty")

	var comparisonType = reflect.TypeOf(EmpiricalComparisonResult{})
	assertStructFieldTypeAndTags(t, comparisonType, "Method", reflect.TypeOf(reportmodel.CostBasisMethod("")), "method", "method")
	assertStructFieldTypeAndTags(t, comparisonType, "ExpectedValue", reflect.TypeOf(""), "expected_value", "expected_value")
	assertStructFieldTypeAndTags(t, comparisonType, "ActualValue", reflect.TypeOf(""), "actual_value", "actual_value")
	assertStructFieldTypeAndTags(t, comparisonType, "Difference", reflect.TypeOf(""), "difference", "difference")
	assertStructFieldTypeAndTags(t, comparisonType, "DecimalPolicy", reflect.TypeOf(""), "decimal_policy", "decimal_policy")
	assertStructFieldTypeAndTags(t, comparisonType, "Tolerance", reflect.TypeOf(""), "tolerance", "tolerance")
	assertStructFieldTypeAndTags(t, comparisonType, "Passed", reflect.TypeOf(true), "passed", "passed")
	assertStructFieldTypeAndTags(t, comparisonType, "RelevantSourceIDs", reflect.TypeOf([]string{}), "relevant_source_ids,omitempty", "relevant_source_ids,omitempty")
}

// TestEmpiricalActivityJSONContractKeepsDecimalFieldsAsStrings verifies the
// dataset activity model persists decimal fields as JSON strings and omits empty
// optional fields.
// Authored by: OpenCode
func TestEmpiricalActivityJSONContractKeepsDecimalFieldsAsStrings(t *testing.T) {
	t.Parallel()

	var activity = EmpiricalActivity{
		SourceID:           "emp-act-000001",
		OccurredAt:         "2024-01-02T09:00:00Z",
		DeterministicOrder: 1,
		ActivityType:       syncmodel.ActivityTypeBuy,
		AssetIdentityKey:   "asset-alpha",
		AssetSymbol:        "ALPHA",
		Quantity:           "1.25",
		FeeAmount:          "0",
		CoverageTags:       []string{"fifo"},
	}

	var payload = marshalToMap(t, activity)
	assertMapStringValue(t, payload, "quantity", "1.25")
	assertMapStringValue(t, payload, "fee_amount", "0")
	assertMapOmitted(t, payload, "gross_value")
	assertMapOmitted(t, payload, "unit_price")
	assertMapOmitted(t, payload, "currency")
	assertMapOmitted(t, payload, "source_scope")
	assertMapOmitted(t, payload, "zero_priced_reduction_explanation")
}

// TestOracleOutputAndComparisonJSONContracts verifies the shared oracle and
// comparison models persist nested comparable values, tolerances, and diagnostic
// fields in the string-based shape required by the contracts.
// Authored by: OpenCode
func TestOracleOutputAndComparisonJSONContracts(t *testing.T) {
	t.Parallel()

	var oracleOutput = OracleOutput{
		FixtureVersion:   "1",
		DatasetVersion:   "1",
		CaseID:           "case-fifo-basic-2024",
		Method:           reportmodel.CostBasisMethodFIFO,
		Year:             2024,
		AssetIdentityKey: "asset-alpha",
		Values: ComparableOutputValues{
			RealizedGainOrLoss: "5",
			AllocatedBasis:     "10",
			ClosingQuantity:    "0",
			ClosingBasis:       "0",
		},
		Matches: []OracleMatchEvidence{
			{
				DisposedSourceID:    "emp-act-000010",
				AcquisitionSourceID: "emp-act-000001",
				MatchedQuantity:     "1",
				MatchedBasis:        "10",
				MatchedProceeds:     "15",
				MatchedGainOrLoss:   "5",
				SupportLabel:        EvidenceSupportLabelHledgerBacked,
			},
		},
		Metadata: OracleGenerationRun{
			HledgerVersion:       "1.52.1",
			CommandArguments:     []string{"-f", "testdata/empirical/hledger/fifo.journal", "print"},
			DatasetInputHash:     "sha256:dataset",
			HledgerInputHash:     "sha256:ledger",
			DecimalPolicy:        "scale=16,rounding=half_up",
			NormalizationVersion: "1",
			FinancialTolerances: map[string]string{
				"allocated_basis": "0.0000000000000001",
			},
			ToleranceNotes: map[string]string{
				"allocated_basis": "One-unit residual after decimal-policy alignment",
			},
			OracleOutputHash: "sha256:oracle",
		},
	}

	var oraclePayload = marshalToMap(t, oracleOutput)
	var values = requireNestedMap(t, oraclePayload, "values")
	assertMapStringValue(t, values, "realized_gain_or_loss", "5")
	assertMapStringValue(t, values, "allocated_basis", "10")
	assertMapStringValue(t, values, "closing_quantity", "0")
	assertMapStringValue(t, values, "closing_basis", "0")

	var metadata = requireNestedMap(t, oraclePayload, "metadata")
	assertMapStringValue(t, metadata, "decimal_policy", "scale=16,rounding=half_up")
	var financialTolerances = requireNestedMap(t, metadata, "financial_tolerances")
	assertMapStringValue(t, financialTolerances, "allocated_basis", "0.0000000000000001")

	var comparison = EmpiricalComparisonResult{
		CaseID:            "case-fifo-basic-2024",
		Method:            reportmodel.CostBasisMethodFIFO,
		Year:              2024,
		AssetIdentityKey:  "asset-alpha",
		Field:             "values.allocated_basis",
		ExpectedValue:     "10",
		ActualValue:       "10.0000000000000001",
		Difference:        "0.0000000000000001",
		DecimalPolicy:     "scale=16,rounding=half_up",
		Tolerance:         "0.0000000000000001",
		Passed:            false,
		DiagnosticContext: "case-fifo-basic-2024 allocated basis mismatch",
	}

	var comparisonPayload = marshalToMap(t, comparison)
	assertMapStringValue(t, comparisonPayload, "expected_value", "10")
	assertMapStringValue(t, comparisonPayload, "actual_value", "10.0000000000000001")
	assertMapStringValue(t, comparisonPayload, "difference", "0.0000000000000001")
	assertMapStringValue(t, comparisonPayload, "tolerance", "0.0000000000000001")
	assertMapBoolValue(t, comparisonPayload, "passed", false)
	assertMapStringValue(t, comparisonPayload, "diagnostic_context", "case-fifo-basic-2024 allocated basis mismatch")
	assertMapOmitted(t, comparisonPayload, "relevant_source_ids")
}

// assertStructFieldTypeAndTags verifies one struct field exists with the exact
// reflected type and JSON and YAML tags required by the contracts.
// Authored by: OpenCode
func assertStructFieldTypeAndTags(t *testing.T, structType reflect.Type, fieldName string, wantType reflect.Type, wantJSON string, wantYAML string) {
	t.Helper()

	var field = requireStructField(t, structType, fieldName)
	if field.Type != wantType {
		t.Fatalf("unexpected %s.%s type: got %v want %v", structType.Name(), fieldName, field.Type, wantType)
	}
	if field.Tag.Get("json") != wantJSON {
		t.Fatalf("unexpected %s.%s json tag: got %q want %q", structType.Name(), fieldName, field.Tag.Get("json"), wantJSON)
	}
	if field.Tag.Get("yaml") != wantYAML {
		t.Fatalf("unexpected %s.%s yaml tag: got %q want %q", structType.Name(), fieldName, field.Tag.Get("yaml"), wantYAML)
	}
}

// requireStructField returns one exported struct field by name or fails the
// current test immediately.
// Authored by: OpenCode
func requireStructField(t *testing.T, structType reflect.Type, fieldName string) reflect.StructField {
	t.Helper()

	var field, ok = structType.FieldByName(fieldName)
	if !ok {
		t.Fatalf("missing field %s.%s", structType.Name(), fieldName)
	}

	return field
}

// marshalToMap marshals one fixture value to JSON and decodes it into a generic
// map for contract-shape assertions.
// Authored by: OpenCode
func marshalToMap(t *testing.T, value any) map[string]any {
	t.Helper()

	var raw, err = json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal JSON: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal JSON payload: %v", err)
	}

	return payload
}

// requireNestedMap returns one nested JSON object field from a decoded payload
// or fails the current test.
// Authored by: OpenCode
func requireNestedMap(t *testing.T, payload map[string]any, key string) map[string]any {
	t.Helper()

	var value, ok = payload[key]
	if !ok {
		t.Fatalf("missing JSON field %q", key)
	}

	var nested, nestedOK = value.(map[string]any)
	if !nestedOK {
		t.Fatalf("JSON field %q has unexpected type %T", key, value)
	}

	return nested
}

// assertMapStringValue verifies one decoded JSON field is present as the exact
// expected string value.
// Authored by: OpenCode
func assertMapStringValue(t *testing.T, payload map[string]any, key string, want string) {
	t.Helper()

	var value, ok = payload[key]
	if !ok {
		t.Fatalf("missing JSON field %q", key)
	}

	var got, stringOK = value.(string)
	if !stringOK {
		t.Fatalf("JSON field %q has unexpected type %T", key, value)
	}
	if got != want {
		t.Fatalf("unexpected JSON field %q: got %q want %q", key, got, want)
	}
}

// assertMapBoolValue verifies one decoded JSON field is present as the exact
// expected boolean value.
// Authored by: OpenCode
func assertMapBoolValue(t *testing.T, payload map[string]any, key string, want bool) {
	t.Helper()

	var value, ok = payload[key]
	if !ok {
		t.Fatalf("missing JSON field %q", key)
	}

	var got, boolOK = value.(bool)
	if !boolOK {
		t.Fatalf("JSON field %q has unexpected type %T", key, value)
	}
	if got != want {
		t.Fatalf("unexpected JSON field %q: got %t want %t", key, got, want)
	}
}

// assertMapOmitted verifies one decoded JSON field is absent from the payload.
// Authored by: OpenCode
func assertMapOmitted(t *testing.T, payload map[string]any, key string) {
	t.Helper()

	if _, ok := payload[key]; ok {
		t.Fatalf("expected JSON field %q to be omitted", key)
	}
}

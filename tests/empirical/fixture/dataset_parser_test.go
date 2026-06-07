package fixture

import (
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

const empiricalDatasetPath = "testdata/empirical/financial-dataset.yaml"

// TestParseEmpiricalDatasetParsesTopLevelActivityAndCaseFields verifies the
// constrained dataset parser decodes the documented top-level, activity, and
// case fields into the shared empirical models.
// Authored by: OpenCode
func TestParseEmpiricalDatasetParsesTopLevelActivityAndCaseFields(t *testing.T) {
	t.Parallel()

	var yamlFixture = strings.Join([]string{
		`dataset_version: "1"`,
		`description: "Synthetic empirical financial validation dataset"`,
		`currency: "USD"`,
		`supported_years:`,
		`  - 2023`,
		`  - 2024`,
		`  - 2025`,
		`supported_methods:`,
		`  - fifo`,
		`  - lifo`,
		`  - hifo`,
		`  - average_cost`,
		`  - scope_local_hybrid`,
		`coverage_tags:`,
		`  - multi_year_opening_history`,
		`  - zero_priced_reduction`,
		`activities:`,
		`  - source_id: emp-act-000001`,
		`    occurred_at: "2023-01-02T09:00:00Z"`,
		`    deterministic_order: 1`,
		`    activity_type: BUY`,
		`    asset_identity_key: asset-alpha`,
		`    asset_symbol: ALPHA`,
		`    quantity: "1.5"`,
		`    gross_value: "15"`,
		`    unit_price: "10"`,
		`    fee_amount: "0"`,
		`    currency: "USD"`,
		`    coverage_tags:`,
		`      - fifo`,
		`      - gain_case`,
		`  - source_id: emp-act-000002`,
		`    occurred_at: "2024-02-03T12:30:00Z"`,
		`    deterministic_order: 2`,
		`    activity_type: SELL`,
		`    asset_identity_key: asset-alpha`,
		`    asset_symbol: ALPHA`,
		`    quantity: "0.5"`,
		`    gross_value: "7.5"`,
		`    unit_price: "15"`,
		`    fee_amount: "0.25"`,
		`    currency: "USD"`,
		`    coverage_tags:`,
		`      - fifo`,
		`cases:`,
		`  - case_id: case-alpha-disposal-2024`,
		`    description: "Synthetic disposal case"`,
		`    methods:`,
		`      - fifo`,
		`      - scope_local_hybrid`,
		`    year: 2024`,
		`    asset_identity_keys:`,
		`      - asset-alpha`,
		`    activity_source_ids:`,
		`      - emp-act-000001`,
		`      - emp-act-000002`,
		`    coverage_tags:`,
		`      - fifo`,
		`      - gain_case`,
		`    oracle_support: partially_supported`,
		`    unsupported_reason: "Scope-local composition remains project-owned"`,
	}, "\n")

	var dataset = mustParseEmpiricalDataset(t, empiricalDatasetPath, yamlFixture)

	if dataset.DatasetVersion != "1" {
		t.Fatalf("unexpected dataset version: got %q want %q", dataset.DatasetVersion, "1")
	}
	if dataset.Description != "Synthetic empirical financial validation dataset" {
		t.Fatalf("unexpected dataset description: got %q", dataset.Description)
	}
	if dataset.Currency != "USD" {
		t.Fatalf("unexpected dataset currency: got %q want %q", dataset.Currency, "USD")
	}
	if len(dataset.SupportedYears) != 3 || dataset.SupportedYears[0] != 2023 || dataset.SupportedYears[1] != 2024 || dataset.SupportedYears[2] != 2025 {
		t.Fatalf("unexpected supported years: %#v", dataset.SupportedYears)
	}
	if len(dataset.SupportedMethods) != 5 ||
		dataset.SupportedMethods[0] != reportmodel.CostBasisMethodFIFO ||
		dataset.SupportedMethods[1] != reportmodel.CostBasisMethodLIFO ||
		dataset.SupportedMethods[2] != reportmodel.CostBasisMethodHIFO ||
		dataset.SupportedMethods[3] != reportmodel.CostBasisMethodAverageCost ||
		dataset.SupportedMethods[4] != reportmodel.CostBasisMethodScopeLocalHybrid {
		t.Fatalf("unexpected supported methods: %#v", dataset.SupportedMethods)
	}
	if len(dataset.CoverageTags) != 2 || dataset.CoverageTags[0] != "multi_year_opening_history" || dataset.CoverageTags[1] != "zero_priced_reduction" {
		t.Fatalf("unexpected dataset coverage tags: %#v", dataset.CoverageTags)
	}

	if len(dataset.Activities) != 2 {
		t.Fatalf("unexpected activity count: got %d want %d", len(dataset.Activities), 2)
	}

	var firstActivity = dataset.Activities[0]
	if firstActivity.SourceID != "emp-act-000001" {
		t.Fatalf("unexpected first activity source id: got %q", firstActivity.SourceID)
	}
	if firstActivity.OccurredAt != "2023-01-02T09:00:00Z" {
		t.Fatalf("unexpected first activity occurred_at: got %q", firstActivity.OccurredAt)
	}
	if firstActivity.DeterministicOrder != 1 {
		t.Fatalf("unexpected first activity deterministic order: got %d want %d", firstActivity.DeterministicOrder, 1)
	}
	if firstActivity.ActivityType != syncmodel.ActivityTypeBuy {
		t.Fatalf("unexpected first activity type: got %q want %q", firstActivity.ActivityType, syncmodel.ActivityTypeBuy)
	}
	if firstActivity.AssetIdentityKey != "asset-alpha" {
		t.Fatalf("unexpected first activity asset identity key: got %q", firstActivity.AssetIdentityKey)
	}
	if firstActivity.AssetSymbol != "ALPHA" {
		t.Fatalf("unexpected first activity asset symbol: got %q", firstActivity.AssetSymbol)
	}
	if firstActivity.Quantity != "1.5" || firstActivity.GrossValue != "15" || firstActivity.UnitPrice != "10" || firstActivity.FeeAmount != "0" || firstActivity.Currency != "USD" {
		t.Fatalf("unexpected first activity financial fields: %#v", firstActivity)
	}
	if len(firstActivity.CoverageTags) != 2 || firstActivity.CoverageTags[0] != "fifo" || firstActivity.CoverageTags[1] != "gain_case" {
		t.Fatalf("unexpected first activity coverage tags: %#v", firstActivity.CoverageTags)
	}

	var secondActivity = dataset.Activities[1]
	if secondActivity.SourceID != "emp-act-000002" {
		t.Fatalf("unexpected second activity source id: got %q", secondActivity.SourceID)
	}
	if secondActivity.ActivityType != syncmodel.ActivityTypeSell {
		t.Fatalf("unexpected second activity type: got %q want %q", secondActivity.ActivityType, syncmodel.ActivityTypeSell)
	}
	if secondActivity.Quantity != "0.5" || secondActivity.GrossValue != "7.5" || secondActivity.UnitPrice != "15" || secondActivity.FeeAmount != "0.25" || secondActivity.Currency != "USD" {
		t.Fatalf("unexpected second activity financial fields: %#v", secondActivity)
	}

	if len(dataset.Cases) != 1 {
		t.Fatalf("unexpected case count: got %d want %d", len(dataset.Cases), 1)
	}

	var parsedCase = dataset.Cases[0]
	if parsedCase.CaseID != "case-alpha-disposal-2024" {
		t.Fatalf("unexpected case id: got %q", parsedCase.CaseID)
	}
	if parsedCase.Description != "Synthetic disposal case" {
		t.Fatalf("unexpected case description: got %q", parsedCase.Description)
	}
	if len(parsedCase.Methods) != 2 || parsedCase.Methods[0] != reportmodel.CostBasisMethodFIFO || parsedCase.Methods[1] != reportmodel.CostBasisMethodScopeLocalHybrid {
		t.Fatalf("unexpected case methods: %#v", parsedCase.Methods)
	}
	if parsedCase.Year != 2024 {
		t.Fatalf("unexpected case year: got %d want %d", parsedCase.Year, 2024)
	}
	if len(parsedCase.AssetIdentityKeys) != 1 || parsedCase.AssetIdentityKeys[0] != "asset-alpha" {
		t.Fatalf("unexpected case asset identity keys: %#v", parsedCase.AssetIdentityKeys)
	}
	if len(parsedCase.ActivitySourceIDs) != 2 || parsedCase.ActivitySourceIDs[0] != "emp-act-000001" || parsedCase.ActivitySourceIDs[1] != "emp-act-000002" {
		t.Fatalf("unexpected case activity source ids: %#v", parsedCase.ActivitySourceIDs)
	}
	if len(parsedCase.CoverageTags) != 2 || parsedCase.CoverageTags[0] != "fifo" || parsedCase.CoverageTags[1] != "gain_case" {
		t.Fatalf("unexpected case coverage tags: %#v", parsedCase.CoverageTags)
	}
	if parsedCase.OracleSupport != OracleSupportPartiallySupported {
		t.Fatalf("unexpected case oracle support: got %q want %q", parsedCase.OracleSupport, OracleSupportPartiallySupported)
	}
	if parsedCase.UnsupportedReason != "Scope-local composition remains project-owned" {
		t.Fatalf("unexpected case unsupported reason: got %q", parsedCase.UnsupportedReason)
	}
}

// TestParseEmpiricalDatasetParsesScopesAndZeroPricedReductions verifies the
// parser preserves scoped activity metadata and accepts zero-priced reduction
// rows without priced monetary fields.
// Authored by: OpenCode
func TestParseEmpiricalDatasetParsesScopesAndZeroPricedReductions(t *testing.T) {
	t.Parallel()

	var yamlFixture = strings.Join([]string{
		`dataset_version: "1"`,
		`description: "Synthetic scope and zero-priced reduction dataset"`,
		`currency: "USD"`,
		`supported_years:`,
		`  - 2024`,
		`supported_methods:`,
		`  - scope_local_hybrid`,
		`coverage_tags:`,
		`  - scope_local_hybrid`,
		`  - zero_priced_reduction`,
		`activities:`,
		`  - source_id: emp-act-100001`,
		`    occurred_at: "2024-01-10T10:00:00Z"`,
		`    deterministic_order: 1`,
		`    activity_type: BUY`,
		`    asset_identity_key: asset-beta`,
		`    asset_symbol: BETA`,
		`    quantity: "2"`,
		`    gross_value: "40"`,
		`    unit_price: "20"`,
		`    fee_amount: "0"`,
		`    currency: "USD"`,
		`    source_scope:`,
		`      scope_id: wallet-beta`,
		`      scope_kind: wallet`,
		`      reliability: reliable`,
		`      display_name: "Synthetic Wallet Beta"`,
		`    coverage_tags:`,
		`      - scope_local_hybrid`,
		`  - source_id: emp-act-100002`,
		`    occurred_at: "2024-03-11T10:00:00Z"`,
		`    deterministic_order: 2`,
		`    activity_type: SELL`,
		`    asset_identity_key: asset-beta`,
		`    asset_symbol: BETA`,
		`    quantity: "0.5"`,
		`    zero_priced_reduction_explanation: "Synthetic protocol burn with no proceeds"`,
		`    source_scope:`,
		`      reliability: unavailable`,
		`    coverage_tags:`,
		`      - zero_priced_reduction`,
		`cases: []`,
	}, "\n")

	var dataset = mustParseEmpiricalDataset(t, empiricalDatasetPath, yamlFixture)

	if len(dataset.Activities) != 2 {
		t.Fatalf("unexpected activity count: got %d want %d", len(dataset.Activities), 2)
	}

	var scopedBuy = dataset.Activities[0]
	if scopedBuy.SourceScope == nil {
		t.Fatal("expected reliable scoped activity to keep source scope")
	}
	if scopedBuy.SourceScope.ScopeID != "wallet-beta" {
		t.Fatalf("unexpected scope id: got %q want %q", scopedBuy.SourceScope.ScopeID, "wallet-beta")
	}
	if scopedBuy.SourceScope.ScopeKind != syncmodel.SourceScopeKindWallet {
		t.Fatalf("unexpected scope kind: got %q want %q", scopedBuy.SourceScope.ScopeKind, syncmodel.SourceScopeKindWallet)
	}
	if scopedBuy.SourceScope.Reliability != syncmodel.ScopeReliabilityReliable {
		t.Fatalf("unexpected scope reliability: got %q want %q", scopedBuy.SourceScope.Reliability, syncmodel.ScopeReliabilityReliable)
	}
	if scopedBuy.SourceScope.DisplayName != "Synthetic Wallet Beta" {
		t.Fatalf("unexpected scope display name: got %q", scopedBuy.SourceScope.DisplayName)
	}

	var zeroPricedReduction = dataset.Activities[1]
	if zeroPricedReduction.ActivityType != syncmodel.ActivityTypeSell {
		t.Fatalf("unexpected zero-priced reduction activity type: got %q want %q", zeroPricedReduction.ActivityType, syncmodel.ActivityTypeSell)
	}
	if zeroPricedReduction.ZeroPricedReductionExplanation != "Synthetic protocol burn with no proceeds" {
		t.Fatalf("unexpected zero-priced reduction explanation: got %q", zeroPricedReduction.ZeroPricedReductionExplanation)
	}
	if zeroPricedReduction.GrossValue != "" || zeroPricedReduction.UnitPrice != "" || zeroPricedReduction.FeeAmount != "" || zeroPricedReduction.Currency != "" {
		t.Fatalf("expected zero-priced reduction to keep priced fields empty, got %#v", zeroPricedReduction)
	}
	if zeroPricedReduction.SourceScope == nil {
		t.Fatal("expected zero-priced reduction to keep explicit unavailable scope metadata")
	}
	if zeroPricedReduction.SourceScope.ScopeID != "" {
		t.Fatalf("unexpected unavailable scope id: got %q want empty string", zeroPricedReduction.SourceScope.ScopeID)
	}
	if zeroPricedReduction.SourceScope.ScopeKind != "" {
		t.Fatalf("unexpected unavailable scope kind: got %q want empty string", zeroPricedReduction.SourceScope.ScopeKind)
	}
	if zeroPricedReduction.SourceScope.Reliability != syncmodel.ScopeReliabilityUnavailable {
		t.Fatalf("unexpected unavailable scope reliability: got %q want %q", zeroPricedReduction.SourceScope.Reliability, syncmodel.ScopeReliabilityUnavailable)
	}
}

// TestParseEmpiricalDatasetRejectsNonStringDecimalFields verifies the parser
// rejects YAML numeric scalars for financial decimal fields instead of silently
// coercing them into strings.
// Authored by: OpenCode
func TestParseEmpiricalDatasetRejectsNonStringDecimalFields(t *testing.T) {
	t.Parallel()

	var testCases = []struct {
		name  string
		field string
		line  string
	}{
		{name: "quantity", field: "quantity", line: `    quantity: 1`},
		{name: "gross_value", field: "gross_value", line: `    gross_value: 10`},
		{name: "unit_price", field: "unit_price", line: `    unit_price: 10`},
		{name: "fee_amount", field: "fee_amount", line: `    fee_amount: 0`},
	}

	for _, testCase := range testCases {
		var testCase = testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var _, err = ParseEmpiricalDataset(empiricalDatasetPath, []byte(buildSingleActivityDatasetYAML(testCase.field, testCase.line)))
			if err == nil {
				t.Fatalf("expected non-string decimal field %q to fail parsing", testCase.field)
			}

			assertErrorContainsFragments(t, err, empiricalDatasetPath, "emp-act-000001", testCase.field, "string")
		})
	}
}

// mustParseEmpiricalDataset parses one inline YAML dataset fixture through the
// public empirical dataset parser and fails the current test if parsing does not
// succeed.
// Authored by: OpenCode
func mustParseEmpiricalDataset(t *testing.T, path string, yamlFixture string) EmpiricalDataset {
	t.Helper()

	var dataset, err = ParseEmpiricalDataset(path, []byte(yamlFixture))
	if err != nil {
		t.Fatalf("parse empirical dataset: %v", err)
	}

	return dataset
}

// buildSingleActivityDatasetYAML returns one minimal dataset fixture with one
// caller-controlled decimal field line for string-only decimal parser tests.
// Authored by: OpenCode
func buildSingleActivityDatasetYAML(decimalFieldName string, decimalFieldLine string) string {
	var quantityLine = `    quantity: "1"`
	var grossValueLine = `    gross_value: "10"`
	var unitPriceLine = `    unit_price: "10"`
	var feeAmountLine = `    fee_amount: "0"`

	switch decimalFieldName {
	case "quantity":
		quantityLine = decimalFieldLine
	case "gross_value":
		grossValueLine = decimalFieldLine
	case "unit_price":
		unitPriceLine = decimalFieldLine
	case "fee_amount":
		feeAmountLine = decimalFieldLine
	}

	return strings.Join([]string{
		`dataset_version: "1"`,
		`description: "Synthetic decimal parser contract dataset"`,
		`currency: "USD"`,
		`supported_years:`,
		`  - 2024`,
		`supported_methods:`,
		`  - fifo`,
		`coverage_tags:`,
		`  - fifo`,
		`activities:`,
		`  - source_id: emp-act-000001`,
		`    occurred_at: "2024-01-02T09:00:00Z"`,
		`    deterministic_order: 1`,
		`    activity_type: BUY`,
		`    asset_identity_key: asset-alpha`,
		`    asset_symbol: ALPHA`,
		quantityLine,
		grossValueLine,
		unitPriceLine,
		feeAmountLine,
		`    currency: "USD"`,
		`    coverage_tags:`,
		`      - fifo`,
		`cases: []`,
	}, "\n")
}

// assertErrorContainsFragments verifies one parser error remains actionable by
// including the expected path and field context fragments.
// Authored by: OpenCode
func assertErrorContainsFragments(t *testing.T, err error, fragments ...string) {
	t.Helper()

	var message = err.Error()

	for _, fragment := range fragments {
		if !strings.Contains(message, fragment) {
			t.Fatalf("expected parser error %q to contain fragment %q", message, fragment)
		}
	}
}

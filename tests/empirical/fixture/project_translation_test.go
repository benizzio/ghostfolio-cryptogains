package fixture

import (
	"strings"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/report/calculate"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

// TestBuildProjectActivityCachePreservesDeterministicOrderingAndYears verifies
// dataset activity ordering is translated into stable replay order and keeps the
// declared report years.
// Authored by: OpenCode
func TestBuildProjectActivityCachePreservesDeterministicOrderingAndYears(t *testing.T) {
	t.Parallel()

	var dataset = mustParseEmpiricalDataset(t, empiricalDatasetPath, strings.Join([]string{
		`dataset_version: "1"`,
		`description: "Synthetic ordering translation dataset"`,
		`currency: "USD"`,
		`supported_years:`,
		`  - 2023`,
		`  - 2024`,
		`  - 2025`,
		`supported_methods:`,
		`  - fifo`,
		`coverage_tags:`,
		`  - ordering`,
		`activities:`,
		`  - source_id: emp-act-000003`,
		`    occurred_at: "2024-01-02T09:00:00Z"`,
		`    deterministic_order: 2`,
		`    activity_type: SELL`,
		`    asset_identity_key: asset-alpha`,
		`    asset_symbol: ALPHA`,
		`    quantity: "0.25"`,
		`    gross_value: "5"`,
		`    unit_price: "20"`,
		`    fee_amount: "0"`,
		`    currency: "USD"`,
		`    coverage_tags:`,
		`      - ordering`,
		`  - source_id: emp-act-000001`,
		`    occurred_at: "2023-12-31T18:00:00Z"`,
		`    deterministic_order: 1`,
		`    activity_type: BUY`,
		`    asset_identity_key: asset-alpha`,
		`    asset_symbol: ALPHA`,
		`    quantity: "1"`,
		`    gross_value: "10"`,
		`    unit_price: "10"`,
		`    fee_amount: "0"`,
		`    currency: "USD"`,
		`    coverage_tags:`,
		`      - ordering`,
		`  - source_id: emp-act-000004`,
		`    occurred_at: "2025-01-01T00:00:00Z"`,
		`    deterministic_order: 1`,
		`    activity_type: BUY`,
		`    asset_identity_key: asset-beta`,
		`    asset_symbol: BETA`,
		`    quantity: "2"`,
		`    gross_value: "8"`,
		`    unit_price: "4"`,
		`    fee_amount: "0"`,
		`    currency: "USD"`,
		`    coverage_tags:`,
		`      - ordering`,
		`  - source_id: emp-act-000002`,
		`    occurred_at: "2024-01-02T09:00:00Z"`,
		`    deterministic_order: 1`,
		`    activity_type: BUY`,
		`    asset_identity_key: asset-alpha`,
		`    asset_symbol: ALPHA`,
		`    quantity: "0.5"`,
		`    gross_value: "9"`,
		`    unit_price: "18"`,
		`    fee_amount: "0"`,
		`    currency: "USD"`,
		`    coverage_tags:`,
		`      - ordering`,
		`cases: []`,
	}, "\n"))

	var cache, err = BuildProjectActivityCache(dataset)
	if err != nil {
		t.Fatalf("build project activity cache: %v", err)
	}

	if cache.RetrievedCount != 4 || cache.ActivityCount != 4 {
		t.Fatalf("unexpected cache counts: %#v", cache)
	}
	if len(cache.AvailableReportYears) != 3 || cache.AvailableReportYears[0] != 2023 || cache.AvailableReportYears[1] != 2024 || cache.AvailableReportYears[2] != 2025 {
		t.Fatalf("unexpected available report years: %#v", cache.AvailableReportYears)
	}
	if len(cache.Activities) != 4 {
		t.Fatalf("unexpected translated activity count: got %d want %d", len(cache.Activities), 4)
	}

	var orderedSourceIDs = []string{
		cache.Activities[0].SourceID,
		cache.Activities[1].SourceID,
		cache.Activities[2].SourceID,
		cache.Activities[3].SourceID,
	}
	if orderedSourceIDs[0] != "emp-act-000001" || orderedSourceIDs[1] != "emp-act-000002" || orderedSourceIDs[2] != "emp-act-000003" || orderedSourceIDs[3] != "emp-act-000004" {
		t.Fatalf("unexpected translated activity order: %#v", orderedSourceIDs)
	}
	if cache.Activities[1].OccurredAt != "2024-01-02T09:00:00Z" || cache.Activities[2].OccurredAt != "2024-01-02T09:00:00Z" {
		t.Fatalf("expected same-timestamp rows to remain adjacent in deterministic order: %#v", cache.Activities)
	}
}

// TestBuildProjectActivityCacheDerivesScopeReliability verifies the translated
// cache preserves unavailable, reliable, and partial scope-reliability states.
// Authored by: OpenCode
func TestBuildProjectActivityCacheDerivesScopeReliability(t *testing.T) {
	t.Parallel()

	var testCases = []struct {
		name        string
		yamlFixture string
		expected    syncmodel.ScopeReliability
	}{
		{
			name:     "unavailable when no usable scope exists",
			expected: syncmodel.ScopeReliabilityUnavailable,
			yamlFixture: strings.Join([]string{
				`dataset_version: "1"`,
				`description: "Synthetic unavailable scope dataset"`,
				`currency: "USD"`,
				`supported_years:`,
				`  - 2024`,
				`supported_methods:`,
				`  - fifo`,
				`coverage_tags:`,
				`  - scopes`,
				`activities:`,
				`  - source_id: emp-act-100001`,
				`    occurred_at: "2024-01-02T09:00:00Z"`,
				`    deterministic_order: 1`,
				`    activity_type: BUY`,
				`    asset_identity_key: asset-alpha`,
				`    asset_symbol: ALPHA`,
				`    quantity: "1"`,
				`    gross_value: "10"`,
				`    unit_price: "10"`,
				`    fee_amount: "0"`,
				`    currency: "USD"`,
				`    source_scope:`,
				`      reliability: unavailable`,
				`    coverage_tags:`,
				`      - scopes`,
				`cases: []`,
			}, "\n"),
		},
		{
			name:     "reliable when one timeline keeps the same usable scope",
			expected: syncmodel.ScopeReliabilityReliable,
			yamlFixture: strings.Join([]string{
				`dataset_version: "1"`,
				`description: "Synthetic reliable scope dataset"`,
				`currency: "USD"`,
				`supported_years:`,
				`  - 2024`,
				`supported_methods:`,
				`  - fifo`,
				`coverage_tags:`,
				`  - scopes`,
				`activities:`,
				`  - source_id: emp-act-100101`,
				`    occurred_at: "2024-01-02T09:00:00Z"`,
				`    deterministic_order: 1`,
				`    activity_type: BUY`,
				`    asset_identity_key: asset-alpha`,
				`    asset_symbol: ALPHA`,
				`    quantity: "1"`,
				`    gross_value: "10"`,
				`    unit_price: "10"`,
				`    fee_amount: "0"`,
				`    currency: "USD"`,
				`    source_scope:`,
				`      scope_id: account-alpha`,
				`      scope_kind: account`,
				`      reliability: reliable`,
				`      display_name: "Synthetic Account Alpha"`,
				`    coverage_tags:`,
				`      - scopes`,
				`  - source_id: emp-act-100102`,
				`    occurred_at: "2024-02-02T09:00:00Z"`,
				`    deterministic_order: 2`,
				`    activity_type: SELL`,
				`    asset_identity_key: asset-alpha`,
				`    asset_symbol: ALPHA`,
				`    quantity: "0.5"`,
				`    gross_value: "6"`,
				`    unit_price: "12"`,
				`    fee_amount: "0"`,
				`    currency: "USD"`,
				`    source_scope:`,
				`      scope_id: account-alpha`,
				`      scope_kind: account`,
				`      reliability: reliable`,
				`      display_name: "Synthetic Account Alpha"`,
				`    coverage_tags:`,
				`      - scopes`,
				`cases: []`,
			}, "\n"),
		},
		{
			name:     "partial when a usable scope disappears within one asset timeline",
			expected: syncmodel.ScopeReliabilityPartial,
			yamlFixture: strings.Join([]string{
				`dataset_version: "1"`,
				`description: "Synthetic partial scope dataset"`,
				`currency: "USD"`,
				`supported_years:`,
				`  - 2024`,
				`supported_methods:`,
				`  - fifo`,
				`coverage_tags:`,
				`  - scopes`,
				`activities:`,
				`  - source_id: emp-act-100201`,
				`    occurred_at: "2024-01-02T09:00:00Z"`,
				`    deterministic_order: 1`,
				`    activity_type: BUY`,
				`    asset_identity_key: asset-alpha`,
				`    asset_symbol: ALPHA`,
				`    quantity: "1"`,
				`    gross_value: "10"`,
				`    unit_price: "10"`,
				`    fee_amount: "0"`,
				`    currency: "USD"`,
				`    source_scope:`,
				`      scope_id: wallet-alpha`,
				`      scope_kind: wallet`,
				`      reliability: reliable`,
				`      display_name: "Synthetic Wallet Alpha"`,
				`    coverage_tags:`,
				`      - scopes`,
				`  - source_id: emp-act-100202`,
				`    occurred_at: "2024-02-02T09:00:00Z"`,
				`    deterministic_order: 2`,
				`    activity_type: SELL`,
				`    asset_identity_key: asset-alpha`,
				`    asset_symbol: ALPHA`,
				`    quantity: "0.5"`,
				`    gross_value: "6"`,
				`    unit_price: "12"`,
				`    fee_amount: "0"`,
				`    currency: "USD"`,
				`    source_scope:`,
				`      reliability: unavailable`,
				`    coverage_tags:`,
				`      - scopes`,
				`cases: []`,
			}, "\n"),
		},
	}

	for _, testCase := range testCases {
		var testCase = testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var dataset = mustParseEmpiricalDataset(t, empiricalDatasetPath, testCase.yamlFixture)
			var cache, err = BuildProjectActivityCache(dataset)
			if err != nil {
				t.Fatalf("build project activity cache: %v", err)
			}
			if cache.ScopeReliability != testCase.expected {
				t.Fatalf("unexpected scope reliability: got %q want %q", cache.ScopeReliability, testCase.expected)
			}
		})
	}
}

// TestBuildProjectActivityCachePreservesOrderTierCurrencySelection verifies the
// translated priced activity remains selectable through the order currency tier.
// Authored by: OpenCode
func TestBuildProjectActivityCachePreservesOrderTierCurrencySelection(t *testing.T) {
	t.Parallel()

	var dataset = mustParseEmpiricalDataset(t, empiricalDatasetPath, strings.Join([]string{
		`dataset_version: "1"`,
		`description: "Synthetic order-tier selection dataset"`,
		`currency: "USD"`,
		`supported_years:`,
		`  - 2024`,
		`supported_methods:`,
		`  - fifo`,
		`coverage_tags:`,
		`  - order_tier`,
		`activities:`,
		`  - source_id: emp-act-200001`,
		`    occurred_at: "2024-01-02T09:00:00Z"`,
		`    deterministic_order: 1`,
		`    activity_type: BUY`,
		`    asset_identity_key: asset-alpha`,
		`    asset_symbol: ALPHA`,
		`    quantity: "1.5"`,
		`    gross_value: "15"`,
		`    unit_price: "10"`,
		`    fee_amount: "0.5"`,
		`    currency: "USD"`,
		`    coverage_tags:`,
		`      - order_tier`,
		`cases: []`,
	}, "\n"))

	var cache, err = BuildProjectActivityCache(dataset)
	if err != nil {
		t.Fatalf("build project activity cache: %v", err)
	}

	var input reportmodel.ActivityCalculationInput
	input, err = calculate.SelectActivityCalculationInput(cache.Activities[0])
	if err != nil {
		t.Fatalf("select activity calculation input: %v", err)
	}

	if input.SelectedCurrencyContext != reportmodel.SelectedCurrencyContextOrder {
		t.Fatalf("unexpected selected currency context: got %q want %q", input.SelectedCurrencyContext, reportmodel.SelectedCurrencyContextOrder)
	}
	if input.SelectedCurrencyCode != "USD" {
		t.Fatalf("unexpected selected currency code: got %q want %q", input.SelectedCurrencyCode, "USD")
	}
	if input.GrossValue == nil || input.GrossValue.Cmp(apd.New(15, 0)) != 0 {
		t.Fatalf("unexpected selected gross value: %#v", input.GrossValue)
	}
	if input.UnitPrice == nil || input.UnitPrice.Cmp(apd.New(10, 0)) != 0 {
		t.Fatalf("unexpected selected unit price: %#v", input.UnitPrice)
	}
	if input.FeeAmount == nil || input.FeeAmount.Cmp(apd.New(5, -1)) != 0 {
		t.Fatalf("unexpected selected fee amount: %#v", input.FeeAmount)
	}
}

// TestBuildProjectActivityCacheMapsZeroPricedHoldingReduction verifies the
// translated zero-priced reduction becomes a valid zero-priced calculation input
// and can be replayed through the pure calculator.
// Authored by: OpenCode
func TestBuildProjectActivityCacheMapsZeroPricedHoldingReduction(t *testing.T) {
	t.Parallel()

	var dataset = mustParseEmpiricalDataset(t, empiricalDatasetPath, strings.Join([]string{
		`dataset_version: "1"`,
		`description: "Synthetic zero-priced reduction translation dataset"`,
		`currency: "USD"`,
		`supported_years:`,
		`  - 2024`,
		`supported_methods:`,
		`  - fifo`,
		`coverage_tags:`,
		`  - zero_priced_reduction`,
		`activities:`,
		`  - source_id: emp-act-300001`,
		`    occurred_at: "2024-01-02T09:00:00Z"`,
		`    deterministic_order: 1`,
		`    activity_type: BUY`,
		`    asset_identity_key: asset-alpha`,
		`    asset_symbol: ALPHA`,
		`    quantity: "2"`,
		`    gross_value: "20"`,
		`    unit_price: "10"`,
		`    fee_amount: "0"`,
		`    currency: "USD"`,
		`    coverage_tags:`,
		`      - zero_priced_reduction`,
		`  - source_id: emp-act-300002`,
		`    occurred_at: "2024-02-02T09:00:00Z"`,
		`    deterministic_order: 2`,
		`    activity_type: SELL`,
		`    asset_identity_key: asset-alpha`,
		`    asset_symbol: ALPHA`,
		`    quantity: "0.5"`,
		`    zero_priced_reduction_explanation: "Synthetic protocol burn with no proceeds"`,
		`    coverage_tags:`,
		`      - zero_priced_reduction`,
		`cases: []`,
	}, "\n"))

	var cache, err = BuildProjectActivityCache(dataset)
	if err != nil {
		t.Fatalf("build project activity cache: %v", err)
	}

	var input reportmodel.ActivityCalculationInput
	input, err = calculate.SelectActivityCalculationInput(cache.Activities[1])
	if err != nil {
		t.Fatalf("select zero-priced calculation input: %v", err)
	}

	if !input.IsZeroPricedHoldingReduction {
		t.Fatal("expected zero-priced holding reduction input")
	}
	if input.SelectedCurrencyCode != "" || input.SelectedCurrencyContext != "" {
		t.Fatalf("expected zero-priced input to keep no selected currency context, got %q %q", input.SelectedCurrencyContext, input.SelectedCurrencyCode)
	}
	if input.Comment != "Synthetic protocol burn with no proceeds" {
		t.Fatalf("unexpected zero-priced explanation: got %q", input.Comment)
	}
	if input.GrossValue == nil || input.GrossValue.Sign() != 0 || input.UnitPrice == nil || input.UnitPrice.Sign() != 0 || input.FeeAmount == nil || input.FeeAmount.Sign() != 0 {
		t.Fatalf("expected explicit zero-valued reduction inputs, got %#v", input)
	}

	var report reportmodel.CapitalGainsReport
	report, err = RunProjectCalculation(cache, 2024, reportmodel.CostBasisMethodFIFO)
	if err != nil {
		t.Fatalf("run project calculation: %v", err)
	}

	if report.Year != 2024 || report.CostBasisMethod != reportmodel.CostBasisMethodFIFO {
		t.Fatalf("unexpected calculated report identity: %#v", report)
	}
	if !report.GeneratedAt.Equal(time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected helper-generated report timestamp: %v", report.GeneratedAt)
	}
	if report.YearlyNetTotal.Sign() != 0 {
		t.Fatalf("expected zero-priced holding reduction to keep yearly net total at zero, got %v", report.YearlyNetTotal)
	}
	if len(report.DetailSections) != 1 || len(report.DetailSections[0].ActivityRows) != 2 {
		t.Fatalf("unexpected detail sections: %#v", report.DetailSections)
	}
	if len(report.DetailSections[0].LiquidationSummaries) != 0 {
		t.Fatalf("expected zero-priced holding reduction to avoid liquidation summaries, got %#v", report.DetailSections[0].LiquidationSummaries)
	}

	var reductionRow = report.DetailSections[0].ActivityRows[1]
	if reductionRow.HoldingReductionExplanation != "Synthetic protocol burn with no proceeds" {
		t.Fatalf("unexpected holding reduction explanation: got %q", reductionRow.HoldingReductionExplanation)
	}
	if reductionRow.ActivityCurrency != "" || reductionRow.LiquidationCalculation != nil {
		t.Fatalf("expected zero-priced holding reduction row without activity currency or liquidation summary, got %#v", reductionRow)
	}
	if reductionRow.QuantityAfterRow.Cmp(apd.New(15, -1)) != 0 {
		t.Fatalf("unexpected post-reduction quantity: got %v want %v", reductionRow.QuantityAfterRow, apd.New(15, -1))
	}
	if reductionRow.BasisAfterRow.Cmp(apd.New(15, 0)) != 0 {
		t.Fatalf("unexpected post-reduction basis: got %v want %v", reductionRow.BasisAfterRow, apd.New(15, 0))
	}
}

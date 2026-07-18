// Package unit verifies focused report-Markdown rendering seams without the
// full yearly report runtime.
// Authored by: OpenCode
package unit

import (
	"strings"
	"testing"
	"time"

	reportmarkdown "github.com/benizzio/ghostfolio-cryptogains/internal/report/markdown"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// TestRenderMarkdownIncludesHeaderSectionOrderAndRequiredTables verifies the
// required header block, section order, and table headings.
// Authored by: OpenCode
func TestRenderMarkdownIncludesHeaderSectionOrderAndRequiredTables(t *testing.T) {
	t.Parallel()

	document, err := reportmarkdown.Render(populatedMarkdownReportFixture())
	if err != nil {
		t.Fatalf("render markdown report: %v", err)
	}

	assertContainsInOrder(
		t,
		document.Content,
		"# Ghostfolio Capital Gains And Losses Report",
		"- **Year:** 2024",
		"- **Cost Basis Method:** Average Cost Basis",
		"- **Generated At:**",
		"- **Report Calculation Currency:** USD",
		"## Gains-And-Losses Summary",
		"## Reference Section",
		"## Asset Detail: BTC",
		"## Asset Detail: XRP",
	)
	assertContainsInOrder(
		t,
		document.Content,
		"| Asset | Net Gain Or Loss | Report Calculation Currency |",
		"| Asset | Historical Full Liquidation Count | Main Section Status |",
		"### In-Year Activity",
		"| Date | Source ID | Type | Quantity | Unit Price | Gross Value | Fee | Quantity After Row | Basis After Row | Original Activity Currency | Calculation Currency | Conversion Status | Note |",
		"### Liquidation Calculations",
		"| Date | Source ID | Disposed Quantity | Allocated Basis | Net Liquidation Proceeds | Gain Or Loss | Calculation Currency |",
	)
}

// TestRenderMarkdownRendersEmptyStates verifies the summary and reference
// empty-state contract and the absence of detail sections when no main assets
// qualify.
// Authored by: OpenCode
func TestRenderMarkdownRendersEmptyStates(t *testing.T) {
	t.Parallel()

	document, err := reportmarkdown.Render(emptyMarkdownReportFixture())
	if err != nil {
		t.Fatalf("render empty markdown report: %v", err)
	}

	assertContainsString(t, document.Content, "No assets had a non-zero net gain or loss in the selected year.")
	assertContainsString(t, document.Content, "| Overall Yearly Net Total | 0.00 | USD |")
	assertContainsString(t, document.Content, "No assets reached full liquidation by year end.")
	assertNotContainsString(t, document.Content, "## Asset Detail:")
}

// TestRenderMarkdownRendersNoInYearActivityAndOmitsLiquidationTable verifies
// the per-asset no-activity contract.
// Authored by: OpenCode
func TestRenderMarkdownRendersNoInYearActivityAndOmitsLiquidationTable(t *testing.T) {
	t.Parallel()

	var report = emptyMarkdownReportFixture()
	report.SummaryEntries = []reportmodel.AssetSummaryEntry{{
		AssetIdentityKey:          "asset-btc",
		DisplayLabel:              "BTC",
		NetGainOrLoss:             mustMarkdownDecimal(t, "0"),
		ReportCalculationCurrency: "USD",
	}}
	report.DetailSections = []reportmodel.AssetDetailSection{{
		AssetIdentityKey:    "asset-btc",
		DisplayLabel:        "BTC",
		OpeningQuantity:     mustMarkdownDecimal(t, "1.25"),
		OpeningCostBasis:    mustMarkdownDecimal(t, "1000"),
		ClosingQuantity:     mustMarkdownDecimal(t, "1.25"),
		ClosingCostBasis:    mustMarkdownDecimal(t, "1000"),
		CalculationCurrency: "USD",
	}}

	document, err := reportmarkdown.Render(report)
	if err != nil {
		t.Fatalf("render no-activity markdown report: %v", err)
	}

	assertContainsString(t, document.Content, "### Historical Position")
	assertContainsString(t, document.Content, "- **Quantity:** 1.25")
	assertNotContainsString(t, document.Content, "No in-year activity for the selected year.")
	assertNotContainsString(t, document.Content, "### Opening Position")
	assertNotContainsString(t, document.Content, "### Closing Position")
	assertNotContainsString(t, document.Content, "### Liquidation Calculations")
}

// TestRenderMarkdownCanonicalDecimalsCurrenciesAndSecretExclusion verifies
// canonical decimal formatting, currency-column rendering, and basic secret
// redaction.
// Authored by: OpenCode
func TestRenderMarkdownCanonicalDecimalsCurrenciesAndSecretExclusion(t *testing.T) {
	t.Parallel()

	document, err := reportmarkdown.Render(populatedMarkdownReportFixture())
	if err != nil {
		t.Fatalf("render markdown report: %v", err)
	}

	assertContainsString(t, document.Content, "| BTC | 1250.50 | USD |")
	assertContainsString(t, document.Content, "| ETH | -10.00 | USD |")
	assertContainsString(t, document.Content, "| Overall Yearly Net Total | 1240.50 | USD |")
	assertContainsString(t, document.Content, "| btc-sell-2024-001 | SELL | 1 | 25000.00 | 25000.00 | 0.00 | 1 | 22009.00 | USD | USD |")
	assertContainsString(t, document.Content, "| xrp-reduction-2024-001 | BLOCKCHAIN OP | 200 | 0.00 | 0.00 | 0.00 | 800 | 400.00 |  | USD |  | manual custody transfer token=[REDACTED] jwt=[REDACTED] payload=[REDACTED] |")
	assertContainsString(t, document.Content, "| btc-sell-2024-001 | 1 | 22009.00 | 25000.00 | 2991.00 | USD |")
	assertNotContainsString(t, document.Content, "secret-token")
	assertNotContainsString(t, document.Content, "secret-jwt")
	assertNotContainsString(t, document.Content, "opaque-payload")
}

// TestRenderMarkdownRejectsInvalidFinancialValueWithoutDocument verifies an
// invalid financial input returns an error and no successful document payload.
// Authored by: OpenCode
func TestRenderMarkdownRejectsInvalidFinancialValueWithoutDocument(t *testing.T) {
	var report = populatedMarkdownReportFixture()
	var invalid apd.Decimal
	invalid.Form = apd.Infinite
	report.YearlyNetTotal = invalid

	var document, err = reportmarkdown.Render(report)
	if err == nil {
		t.Fatal("expected invalid financial value to fail rendering")
	}
	if len(document.Content) != 0 {
		t.Fatalf("expected invalid financial value to return no document, got %q", document.Content)
	}
}

// populatedMarkdownReportFixture returns one deterministic calculated report for
// Markdown rendering assertions.
// Authored by: OpenCode
func populatedMarkdownReportFixture() reportmodel.CapitalGainsReport {
	return reportmodel.CapitalGainsReport{
		Year:                      2024,
		CostBasisMethod:           reportmodel.CostBasisMethodAverageCost,
		GeneratedAt:               time.Date(2026, time.May, 21, 12, 34, 56, 0, time.Local),
		ReportCalculationCurrency: "USD",
		SummaryEntries: []reportmodel.AssetSummaryEntry{
			{
				AssetIdentityKey:          "asset-btc",
				DisplayLabel:              "BTC",
				NetGainOrLoss:             mustMarkdownDecimal(nil, "1250.500"),
				ReportCalculationCurrency: "USD",
			},
			{
				AssetIdentityKey:          "asset-eth",
				DisplayLabel:              "ETH",
				NetGainOrLoss:             mustMarkdownDecimal(nil, "-10.000"),
				ReportCalculationCurrency: "USD",
			},
		},
		YearlyNetTotal: mustMarkdownDecimal(nil, "1240.500"),
		AuditAnnex:     reportmodel.DefaultAuditAnnex(),
		ReferenceEntries: []reportmodel.ReferenceLiquidationEntry{
			{
				AssetIdentityKey:                   "asset-eth",
				DisplayLabel:                       "ETH",
				FullLiquidationCountThroughYearEnd: 1,
				MainSectionStatus:                  reportmodel.ReferenceSectionStatusReferenceOnly,
			},
			{
				AssetIdentityKey:                   "asset-btc",
				DisplayLabel:                       "BTC",
				FullLiquidationCountThroughYearEnd: 1,
				MainSectionStatus:                  reportmodel.ReferenceSectionStatusIncludedInMainSections,
			},
		},
		DetailSections: []reportmodel.AssetDetailSection{
			{
				AssetIdentityKey:    "asset-btc",
				DisplayLabel:        "BTC",
				OpeningQuantity:     mustMarkdownDecimal(nil, "2.000"),
				OpeningCostBasis:    mustMarkdownDecimal(nil, "44018.000"),
				ClosingQuantity:     mustMarkdownDecimal(nil, "1.000"),
				ClosingCostBasis:    mustMarkdownDecimal(nil, "22009.000"),
				CalculationCurrency: "USD",
				ActivityRows: []reportmodel.AssetActivityRow{{
					SourceID:            "btc-sell-2024-001",
					OccurredAt:          time.Date(2024, time.January, 1, 0, 15, 0, 0, time.Local),
					ActivityType:        reportmodel.ActivityTypeSell,
					Quantity:            mustMarkdownDecimal(nil, "1.000"),
					UnitPrice:           markdownDecimalPointer("25000.000"),
					GrossValue:          markdownDecimalPointer("25000.000"),
					FeeAmount:           markdownDecimalPointer("0.000"),
					ActivityCurrency:    "USD",
					BasisAfterRow:       mustMarkdownDecimal(nil, "22009.000"),
					CalculationCurrency: "USD",
					QuantityAfterRow:    mustMarkdownDecimal(nil, "1.000"),
				}},
				LiquidationSummaries: []reportmodel.LiquidationCalculation{{
					SourceID:               "btc-sell-2024-001",
					OccurredAt:             time.Date(2024, time.January, 1, 0, 15, 0, 0, time.Local),
					DisposedQuantity:       mustMarkdownDecimal(nil, "1.000"),
					AllocatedBasis:         mustMarkdownDecimal(nil, "22009.000"),
					NetLiquidationProceeds: mustMarkdownDecimal(nil, "25000.000"),
					GainOrLoss:             mustMarkdownDecimal(nil, "2991.000"),
					ActivityCurrency:       "USD",
					CalculationCurrency:    "USD",
				}},
			},
			{
				AssetIdentityKey:    "asset-xrp",
				DisplayLabel:        "XRP",
				OpeningQuantity:     mustMarkdownDecimal(nil, "1000.000"),
				OpeningCostBasis:    mustMarkdownDecimal(nil, "500.000"),
				ClosingQuantity:     mustMarkdownDecimal(nil, "800.000"),
				ClosingCostBasis:    mustMarkdownDecimal(nil, "400.000"),
				CalculationCurrency: "USD",
				ActivityRows: []reportmodel.AssetActivityRow{{
					SourceID:                    "xrp-reduction-2024-001",
					OccurredAt:                  time.Date(2024, time.April, 1, 12, 0, 0, 0, time.Local),
					ActivityType:                reportmodel.ActivityTypeSell,
					Quantity:                    mustMarkdownDecimal(nil, "200.000"),
					UnitPrice:                   markdownDecimalPointer("0.000"),
					GrossValue:                  markdownDecimalPointer("0.000"),
					FeeAmount:                   markdownDecimalPointer("0.000"),
					BasisAfterRow:               mustMarkdownDecimal(nil, "400.000"),
					CalculationCurrency:         "USD",
					QuantityAfterRow:            mustMarkdownDecimal(nil, "800.000"),
					HoldingReductionExplanation: "manual custody transfer token=secret-token jwt=secret-jwt payload=opaque-payload",
				}},
			},
		},
	}
}

// emptyMarkdownReportFixture returns one calculated report with no main-section
// assets so empty-state rendering can be asserted directly.
// Authored by: OpenCode
func emptyMarkdownReportFixture() reportmodel.CapitalGainsReport {
	return reportmodel.CapitalGainsReport{
		Year:                      2024,
		CostBasisMethod:           reportmodel.CostBasisMethodFIFO,
		GeneratedAt:               time.Date(2026, time.May, 21, 12, 34, 56, 0, time.Local),
		ReportCalculationCurrency: "USD",
		YearlyNetTotal:            mustMarkdownDecimal(nil, "0.000"),
		AuditAnnex:                reportmodel.DefaultAuditAnnex(),
	}
}

// mustMarkdownDecimal parses one test decimal and fails the current test when
// the fixture value is malformed.
// Authored by: OpenCode
func mustMarkdownDecimal(t *testing.T, raw string) apd.Decimal {
	if raw == "" {
		return apd.Decimal{}
	}

	var value, _, err = decimalsupport.ParseString(raw)
	if err != nil {
		if t != nil {
			t.Helper()
			t.Fatalf("parse decimal %q: %v", raw, err)
		}
		panic(err)
	}

	return value
}

// markdownDecimalPointer returns one parsed decimal pointer for test fixtures.
// Authored by: OpenCode
func markdownDecimalPointer(raw string) *apd.Decimal {
	var value = mustMarkdownDecimal(nil, raw)
	return &value
}

// assertContainsInOrder verifies that all expected substrings appear in the
// rendered Markdown in the required order.
// Authored by: OpenCode
func assertContainsInOrder(t *testing.T, content any, expected ...string) {
	t.Helper()

	var rendered = string(reportDocumentContent(content))
	var offset int
	for _, current := range expected {
		var index = strings.Index(rendered[offset:], current)
		if index < 0 {
			t.Fatalf("expected %q after offset %d in content %q", current, offset, rendered)
		}
		offset += index + len(current)
	}
}

// assertContainsString verifies that the rendered Markdown includes one required
// substring.
// Authored by: OpenCode
func assertContainsString(t *testing.T, content any, expected string) {
	t.Helper()
	if !strings.Contains(string(reportDocumentContent(content)), expected) {
		t.Fatalf("expected content to contain %q", expected)
	}
}

// assertNotContainsString verifies that the rendered Markdown excludes one
// forbidden substring.
// Authored by: OpenCode
func assertNotContainsString(t *testing.T, content any, unexpected string) {
	t.Helper()
	if strings.Contains(string(reportDocumentContent(content)), unexpected) {
		t.Fatalf("expected content not to contain %q", unexpected)
	}
}

// reportDocumentContent normalizes Markdown payloads for test assertions.
// Authored by: OpenCode
func reportDocumentContent(content any) []byte {
	switch value := content.(type) {
	case string:
		return []byte(value)
	case []byte:
		return value
	default:
		panic("unsupported report document content")
	}
}

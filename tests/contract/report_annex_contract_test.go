// Package contract verifies Annex 1 rendering contracts for report output.
// Authored by: OpenCode
package contract

import (
	"strings"
	"testing"
	"time"

	reportmarkdown "github.com/benizzio/ghostfolio-cryptogains/internal/report/markdown"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

// TestReportAnnexRenderingContract verifies Annex 1 title, section order,
// detailed per-asset audit fields, conversion audit labels, and redaction.
// Authored by: OpenCode
func TestReportAnnexRenderingContract(t *testing.T) {
	t.Parallel()

	var report = contractMarkdownReportFixture(reportmodel.ReportBaseCurrencyEUR.Label())
	report.AuditAnnex = contractDetailedAuditAnnex()

	var document, err = reportmarkdown.RenderAnnex(report)
	if err != nil {
		t.Fatalf("render annex: %v", err)
	}

	assertContains(t, document.Content, "# Annex 1 - Audit")
	assertSectionOrder(t, document.Content, "## Detailed Per-Asset Audit Report", "## Currency Conversion Audit")
	assertContains(t, document.Content, "### Asset: BTC")
	assertContains(t, document.Content, "### Asset: ETH")
	assertContains(t, document.Content, "| Date/Time | Source ID | Activity Type | Quantity | Unit Price | Gross Value | Fee | Original Activity Currency | Calculation Currency | Quantity After Activity | Basis After Activity | Full Liquidation Event | Allocated Basis | Net Liquidation Proceeds | Gain/Loss | Conversion Status | Sanitized Note |")
	assertContains(t, document.Content, "| 2023-12-31 23:15:00 | btc-sell-2024-001 | SELL | 1 | 25000 | 25000 | 0 | USD | EUR | 1 | 22009 | true | 22009 | 25000 | 1240.5 | Converted | note token=[REDACTED] |")
	assertContains(t, document.Content, "| 2024-04-01 10:00:00 | xrp-reduction-2024-001 | BLOCKCHAIN OP | 200 | 0 | 0 | 0 | USD | EUR | 800 | 400 | false |  |  |  | Same currency | custody transfer |")
	assertContains(t, document.Content, "| 2023-01-01 10:00:00 | eth-reference-buy | BUY | 1 | 50 | 50 | 0 | EUR | EUR | 1 | 50 | false |  |  |  | Same currency | reference-only acquisition |")
	assertContains(t, document.Content, "## Currency Conversion Audit")
	assertContains(t, document.Content, "| 2024-01-01 | btc-sell-2024-001 | BTC | 2023-12-29 | USD | EUR | unit_price: 27000 -> 25000; gross_value: 27000 -> 25000 | Source currency per base currency | 1.08 |")
	assertNotContains(t, document.Content, "source_per_base")
	assertNotContains(t, document.Content, "base_per_source")
	assertNotContains(t, document.Content, "secret-token")
	assertNotContains(t, document.Content, "post-year")
}

// contractDetailedAuditAnnex returns a deterministic detailed Annex 1 fixture.
// Authored by: OpenCode
func contractDetailedAuditAnnex() reportmodel.AuditAnnex {
	var annex, err = reportmodel.NewDetailedAuditAnnex([]reportmodel.PerAssetAuditSection{
		{
			AssetIdentityKey: "asset-btc",
			DisplayLabel:     "BTC",
			Entries: []reportmodel.AuditActivityEntry{contractAuditEntry(reportmodel.AuditActivityEntry{
				SourceID:               "btc-sell-2024-001",
				OccurredAt:             time.Date(2024, time.January, 1, 0, 15, 0, 0, contractMarkdownFixtureLocation),
				ActivityType:           reportmodel.ActivityTypeSell,
				Quantity:               mustContractDecimal("1"),
				UnitPrice:              contractDecimalPointer("25000"),
				GrossValue:             contractDecimalPointer("25000"),
				FeeAmount:              contractDecimalPointer("0"),
				ActivityCurrency:       "USD",
				CalculationCurrency:    "EUR",
				QuantityAfterActivity:  mustContractDecimal("1"),
				BasisAfterActivity:     mustContractDecimal("22009"),
				FullLiquidationEvent:   true,
				AllocatedBasis:         contractDecimalPointer("22009"),
				NetLiquidationProceeds: contractDecimalPointer("25000"),
				GainOrLoss:             contractDecimalPointer("1240.5"),
				ConversionStatus:       reportmodel.ConversionStatusConverted,
				Note:                   "note token=secret-token",
			})},
		},
		{
			AssetIdentityKey: "asset-xrp",
			DisplayLabel:     "XRP",
			Entries: []reportmodel.AuditActivityEntry{contractAuditEntry(reportmodel.AuditActivityEntry{
				SourceID:              "xrp-reduction-2024-001",
				OccurredAt:            time.Date(2024, time.April, 1, 12, 0, 0, 0, contractMarkdownFixtureSummerLocation),
				ActivityType:          reportmodel.ActivityTypeSell,
				Quantity:              mustContractDecimal("200"),
				UnitPrice:             contractDecimalPointer("0"),
				GrossValue:            contractDecimalPointer("0"),
				FeeAmount:             contractDecimalPointer("0"),
				ActivityCurrency:      "USD",
				CalculationCurrency:   "EUR",
				QuantityAfterActivity: mustContractDecimal("800"),
				BasisAfterActivity:    mustContractDecimal("400"),
				ConversionStatus:      reportmodel.ConversionStatusSameCurrency,
				Note:                  "custody transfer",
			})},
		},
		{
			AssetIdentityKey: "asset-eth",
			DisplayLabel:     "ETH",
			Entries: []reportmodel.AuditActivityEntry{contractAuditEntry(reportmodel.AuditActivityEntry{
				SourceID:              "eth-reference-buy",
				OccurredAt:            time.Date(2023, time.January, 1, 10, 0, 0, 0, time.UTC),
				ActivityType:          reportmodel.ActivityTypeBuy,
				Quantity:              mustContractDecimal("1"),
				UnitPrice:             contractDecimalPointer("50"),
				GrossValue:            contractDecimalPointer("50"),
				FeeAmount:             contractDecimalPointer("0"),
				ActivityCurrency:      "EUR",
				CalculationCurrency:   "EUR",
				QuantityAfterActivity: mustContractDecimal("1"),
				BasisAfterActivity:    mustContractDecimal("50"),
				ConversionStatus:      reportmodel.ConversionStatusSameCurrency,
				Note:                  "reference-only acquisition",
			})},
		},
	}, contractMarkdownReportFixture(reportmodel.ReportBaseCurrencyEUR.Label()).AuditAnnex.ConversionAuditEntries)
	if err != nil {
		panic(err)
	}

	return annex
}

// contractAuditEntry returns its input so composite literals remain readable.
// Authored by: OpenCode
func contractAuditEntry(entry reportmodel.AuditActivityEntry) reportmodel.AuditActivityEntry {
	return entry
}

// assertSectionOrder verifies that the first section appears before the second.
// Authored by: OpenCode
func assertSectionOrder(t *testing.T, content any, first string, second string) {
	t.Helper()

	var rendered = string(reportDocumentContent(content))

	var firstIndex = strings.Index(rendered, first)
	var secondIndex = strings.Index(rendered, second)
	if firstIndex < 0 || secondIndex < 0 || firstIndex >= secondIndex {
		t.Fatalf("expected %q before %q in %q", first, second, content)
	}
}

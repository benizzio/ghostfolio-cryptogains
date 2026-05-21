// Package contract verifies rendered workflow and Ghostfolio-boundary contracts
// for the sync-and-storage slice.
// Authored by: OpenCode
package contract

import (
	"testing"
	"time"

	reportmarkdown "github.com/benizzio/ghostfolio-cryptogains/internal/report/markdown"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	reportoutput "github.com/benizzio/ghostfolio-cryptogains/internal/report/output"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/cockroachdb/apd/v3"
)

// TestMarkdownReportDocumentContract verifies the required Markdown document
// shape from the contract.
// Authored by: OpenCode
func TestMarkdownReportDocumentContract(t *testing.T) {
	t.Parallel()

	document, err := reportmarkdown.Render(contractMarkdownReportFixture())
	if err != nil {
		t.Fatalf("render markdown report: %v", err)
	}

	assertContains(t, document.Content, "# Ghostfolio Capital Gains And Losses Report")
	assertContains(t, document.Content, "- Year: 2024")
	assertContains(t, document.Content, "- Cost Basis Method: FIFO")
	assertContains(t, document.Content, "- Generated At:")
	assertContains(t, document.Content, "- Report Calculation Currency: NOT APPLICABLE")
	assertContains(t, document.Content, "## Gains-And-Losses Summary")
	assertContains(t, document.Content, "## Reference Section")
	assertContains(t, document.Content, "## Asset Detail: BTC")
	assertContains(t, document.Content, "| Asset | Net Gain Or Loss | Report Calculation Currency |")
	assertContains(t, document.Content, "| Overall Yearly Net Total | 1240.5 | NOT APPLICABLE |")
	assertContains(t, document.Content, "| Asset | Full Liquidation Count Through Year End | Main Section Status |")
	assertContains(t, document.Content, "### Opening Position")
	assertContains(t, document.Content, "### In-Year Activity")
	assertContains(t, document.Content, "### Liquidation Calculations")
	assertContains(t, document.Content, "### Closing Position")
	assertContains(t, document.Content, "| 2024-01-01 00:15:00 | btc-sell-2024-001 | SELL | 1 | 25000 | 0 | USD | 22009 | NOT APPLICABLE | 1 | note token=[REDACTED] |")
	assertContains(t, document.Content, "| 2024-01-01 00:15:00 | btc-sell-2024-001 | 1 | USD | 22009 | 25000 | 1240.5 | NOT APPLICABLE |")
	assertNotContains(t, document.Content, "secret-token")
}

// TestMarkdownReportOutputFileContract verifies the visible output-file
// contract points that are direct consequences of successful Markdown
// rendering.
// Authored by: OpenCode
func TestMarkdownReportOutputFileContract(t *testing.T) {
	var fixture = testutil.NewReportIOFixture(t)
	var report = contractMarkdownReportFixture()
	var document, err = reportmarkdown.Render(report)
	if err != nil {
		t.Fatalf("render markdown report: %v", err)
	}

	var outputFile reportmodel.ReportOutputFile
	outputFile, err = reportoutput.WriteReportDocument(document)
	if err != nil {
		t.Fatalf("write markdown report document: %v", err)
	}

	if outputFile.Filename != "ghostfolio-capital-gains-2024-fifo-2026-05-21_12-34-56.md" {
		t.Fatalf("unexpected output filename: %q", outputFile.Filename)
	}
	testutil.AssertPathWithin(t, outputFile.Path, fixture.DocumentsDir)
	testutil.AssertRegularFile(t, outputFile.Path)
	testutil.AssertFileContent(t, outputFile.Path, document.Content)
	assertNotContains(t, document.Content, "secret-token")
	assertNotContains(t, outputFile.Path, "secret-token")
}

// contractMarkdownReportFixture returns one deterministic calculated report used
// by the Markdown contract tests.
// Authored by: OpenCode
func contractMarkdownReportFixture() reportmodel.CapitalGainsReport {
	return reportmodel.CapitalGainsReport{
		Year:                      2024,
		CostBasisMethod:           reportmodel.CostBasisMethodFIFO,
		GeneratedAt:               time.Date(2026, time.May, 21, 12, 34, 56, 0, time.Local),
		ReportCalculationCurrency: "NOT APPLICABLE",
		SummaryEntries: []reportmodel.AssetSummaryEntry{{
			AssetIdentityKey:          "asset-btc",
			DisplayLabel:              "BTC",
			NetGainOrLoss:             mustContractDecimal("1240.5"),
			ReportCalculationCurrency: "NOT APPLICABLE",
		}},
		YearlyNetTotal: mustContractDecimal("1240.5"),
		ReferenceEntries: []reportmodel.ReferenceLiquidationEntry{{
			AssetIdentityKey:                   "asset-btc",
			DisplayLabel:                       "BTC",
			FullLiquidationCountThroughYearEnd: 1,
			MainSectionStatus:                  reportmodel.ReferenceSectionStatusIncludedInMainSections,
		}},
		DetailSections: []reportmodel.AssetDetailSection{{
			AssetIdentityKey:    "asset-btc",
			DisplayLabel:        "BTC",
			OpeningQuantity:     mustContractDecimal("2"),
			OpeningCostBasis:    mustContractDecimal("44018"),
			ClosingQuantity:     mustContractDecimal("1"),
			ClosingCostBasis:    mustContractDecimal("22009"),
			CalculationCurrency: "NOT APPLICABLE",
			ActivityRows: []reportmodel.AssetActivityRow{{
				SourceID:                    "btc-sell-2024-001",
				OccurredAt:                  time.Date(2024, time.January, 1, 0, 15, 0, 0, time.Local),
				ActivityType:                syncmodel.ActivityTypeSell,
				Quantity:                    mustContractDecimal("1"),
				GrossValue:                  contractDecimalPointer("25000"),
				FeeAmount:                   contractDecimalPointer("0"),
				ActivityCurrency:            "USD",
				BasisAfterRow:               mustContractDecimal("22009"),
				CalculationCurrency:         "NOT APPLICABLE",
				QuantityAfterRow:            mustContractDecimal("1"),
				HoldingReductionExplanation: "note token=secret-token",
			}},
			LiquidationSummaries: []reportmodel.LiquidationCalculation{{
				SourceID:               "btc-sell-2024-001",
				OccurredAt:             time.Date(2024, time.January, 1, 0, 15, 0, 0, time.Local),
				DisposedQuantity:       mustContractDecimal("1"),
				AllocatedBasis:         mustContractDecimal("22009"),
				NetLiquidationProceeds: mustContractDecimal("25000"),
				GainOrLoss:             mustContractDecimal("1240.5"),
				ActivityCurrency:       "USD",
				CalculationCurrency:    "NOT APPLICABLE",
			}},
		}},
	}
}

// mustContractDecimal parses one deterministic contract-fixture decimal.
// Authored by: OpenCode
func mustContractDecimal(raw string) apd.Decimal {
	var value, _, err = decimalsupport.ParseString(raw)
	if err != nil {
		panic(err)
	}
	return value
}

// contractDecimalPointer returns one optional decimal pointer for the contract
// fixtures.
// Authored by: OpenCode
func contractDecimalPointer(raw string) *apd.Decimal {
	var value = mustContractDecimal(raw)
	return &value
}

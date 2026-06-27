// Package contract verifies rendered workflow and Ghostfolio-boundary contracts
// for the sync-and-storage slice.
// Authored by: OpenCode
package contract

import (
	"strings"
	"testing"
	"time"

	reportmarkdown "github.com/benizzio/ghostfolio-cryptogains/internal/report/markdown"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	reportoutput "github.com/benizzio/ghostfolio-cryptogains/internal/report/output"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/cockroachdb/apd/v3"
)

// TestMarkdownReportDocumentContract verifies the required Markdown document
// shape from the contract.
// Authored by: OpenCode
func TestMarkdownReportDocumentContract(t *testing.T) {
	t.Parallel()

	var reportCalculationCurrency = reportmodel.ReportBaseCurrencyEUR.Label()
	document, err := reportmarkdown.Render(contractMarkdownReportFixture(reportCalculationCurrency))
	if err != nil {
		t.Fatalf("render markdown report: %v", err)
	}

	assertContains(t, document.Content, "# Ghostfolio Capital Gains And Losses Report")
	assertContains(t, document.Content, "- Year: 2024")
	assertContains(t, document.Content, "- Cost Basis Method: FIFO")
	assertContains(t, document.Content, "- Generated At:")
	assertContains(t, document.Content, "- Report Calculation Currency: EUR")
	assertNotContains(t, document.Content, "- Report Calculation Currency: NOT APPLICABLE")
	assertContains(t, document.Content, "## Gains-And-Losses Summary")
	assertContains(t, document.Content, "## Reference Section")
	assertContains(t, document.Content, "## Asset Detail: BTC")
	assertContains(t, document.Content, "## Asset Detail: ETH")
	assertContains(t, document.Content, "## Asset Detail: XRP")
	assertContains(t, document.Content, "| Asset | Net Gain Or Loss | Report Calculation Currency |")
	assertContains(t, document.Content, "| Overall Yearly Net Total | 2240.5 | EUR |")
	assertContains(t, document.Content, "| Asset | Full Liquidation Count Through Year End | Main Section Status |")
	assertContains(t, document.Content, "### Opening Position")
	assertContains(t, document.Content, "### In-Year Activity")
	assertContains(t, document.Content, "| Date | Source ID | Type | Quantity | Unit Price | Gross Value | Fee | Activity Currency | Basis After Row | Calculation Currency | Quantity After Row | Conversion Status | Note |")
	assertContains(t, document.Content, "### Liquidation Calculations")
	assertContains(t, document.Content, "### Closing Position")
	assertContains(t, document.Content, "| 2024-01-01 00:15:00 | btc-sell-2024-001 | SELL | 1 | 25000 | 25000 | 0 | USD | 22009 | EUR | 1 | converted | note token=[REDACTED] |")
	assertContains(t, document.Content, "| 2024-04-01 12:00:00 | xrp-reduction-2024-001 | SELL | 200 | 0 | 0 | 0 |  | 400 | EUR | 800 |  | custody transfer |")
	assertContains(t, document.Content, "| 2024-01-01 00:15:00 | btc-sell-2024-001 | 1 | USD | 22009 | 25000 | 1240.5 | EUR |")
	assertNotContains(t, document.Content, "NOT APPLICABLE")
	assertNotContains(t, document.Content, "secret-token")
}

// TestMarkdownReportConversionAuditContract verifies the report-visible audit
// fields required for converted priced activities and their rate source summary.
// Authored by: OpenCode
func TestMarkdownReportConversionAuditContract(t *testing.T) {
	t.Parallel()

	var reportCalculationCurrency = reportmodel.ReportBaseCurrencyEUR.Label()
	document, err := reportmarkdown.Render(contractMarkdownReportFixture(reportCalculationCurrency))
	if err != nil {
		t.Fatalf("render markdown report: %v", err)
	}

	assertContains(t, document.Content, "## Rate Source Summary")
	assertContains(t, document.Content, "European Central Bank")
	assertContains(t, document.Content, "ECB Data Portal `EXR`")
	assertContains(t, document.Content, "daily euro foreign exchange reference rate")
	assertContains(t, document.Content, "most recent previous available ECB observation")
	assertNotContains(t, rateSourceSummarySection(document.Content), "Quote Direction")
	assertNotContains(t, rateSourceSummarySection(document.Content), "Rate Value")
	assertNotContains(t, rateSourceSummarySection(document.Content), "source_per_base")
	assertNotContains(t, rateSourceSummarySection(document.Content), "1.08")
	assertContains(t, document.Content, "## Currency Conversion Audit")
	var audit = currencyConversionAuditSection(document.Content)
	assertContains(t, audit, "| Date | Source ID | Asset | Rate Date | Source Currency | Report Base Currency | Converted Amounts | Quote Direction | Rate Value |")
	assertNotContains(t, audit, "Rate Authority")
	assertNotContains(t, audit, "Rate Kind")
	assertContains(t, audit, "| 2024-01-01 | btc-sell-2024-001 | BTC | 2023-12-29 | USD | EUR | unit_price: 27000 -> 25000; gross_value: 27000 -> 25000 | source_per_base | 1.08 |")
	assertNotContains(t, audit, "fee_amount")
	assertNotContains(t, audit, "0 -> 0")
	if got := countOccurrences(audit, "| 2024-01-01 | btc-sell-2024-001 |"); got != 1 {
		t.Fatalf("expected one grouped audit row for btc-sell-2024-001, got %d in %q", got, audit)
	}
	assertContains(t, document.Content, "btc-sell-2024-001")
	assertContains(t, document.Content, "2024-01-01")
	assertContains(t, document.Content, "2023-12-29")
	assertNotContains(t, document.Content, "secret-token")
}

// TestMarkdownReportRateSourceSummaryAggregatesByProvider verifies that the
// provider source summary is rendered once for a selected base-currency provider,
// while rate-specific evidence remains in the conversion audit table.
// Authored by: OpenCode
func TestMarkdownReportRateSourceSummaryAggregatesByProvider(t *testing.T) {
	t.Parallel()

	var report = contractMarkdownReportFixture(reportmodel.ReportBaseCurrencyEUR.Label())
	var secondEvidence = report.RateSources[0]
	secondEvidence.SourceCurrency = "GBP"
	secondEvidence.ActivityDate = time.Date(2024, time.January, 2, 0, 15, 0, 0, time.Local)
	secondEvidence.RateDate = time.Date(2024, time.January, 2, 0, 0, 0, 0, time.Local)
	secondEvidence.RateValue = mustContractDecimal("1.16")
	report.RateSources = append(report.RateSources, secondEvidence)
	report.ConversionAuditEntries = append(report.ConversionAuditEntries, contractConversionAuditEntry(
		"gbp-buy-2024-001",
		"ETH",
		"GBP",
		"1.16",
		reportmodel.QuoteDirectionSourcePerBase,
	))

	var document, err = reportmarkdown.Render(report)
	if err != nil {
		t.Fatalf("render markdown report: %v", err)
	}

	var summary = rateSourceSummarySection(document.Content)
	if got := countOccurrences(summary, "- Authority: European Central Bank"); got != 1 {
		t.Fatalf("expected one provider-level ECB summary, got %d in %q", got, summary)
	}
	assertNotContains(t, summary, "Quote Direction")
	assertNotContains(t, summary, "Rate Value")
	assertNotContains(t, summary, "1.08")
	assertNotContains(t, summary, "1.16")
	var audit = currencyConversionAuditSection(document.Content)
	assertNotContains(t, audit, "Rate Authority")
	assertNotContains(t, audit, "Rate Kind")
	assertContains(t, audit, "| 2024-01-01 | btc-sell-2024-001 | BTC | 2023-12-29 | USD | EUR | unit_price: 27000 -> 25000; gross_value: 27000 -> 25000 | source_per_base | 1.08 |")
	assertContains(t, audit, "| 2024-01-02 | gbp-buy-2024-001 | ETH | 2024-01-02 | GBP | EUR | gross_value: 116 -> 100 | source_per_base | 1.16 |")
}

// TestMarkdownReportCurrencyConversionAuditGroupedHeaderOrder verifies the
// BUG-005 grouped audit table order from the Markdown report contract.
// Authored by: OpenCode
func TestMarkdownReportCurrencyConversionAuditGroupedHeaderOrder(t *testing.T) {
	t.Parallel()

	var document, err = reportmarkdown.Render(contractMarkdownReportFixture(reportmodel.ReportBaseCurrencyEUR.Label()))
	if err != nil {
		t.Fatalf("render markdown report: %v", err)
	}

	var audit = currencyConversionAuditSection(document.Content)
	var expectedHeader = "| Date | Source ID | Asset | Rate Date | Source Currency | Report Base Currency | Converted Amounts | Quote Direction | Rate Value |"
	var expectedSeparator = "|------|-----------|-------|-----------|-----------------|----------------------|-------------------|-----------------|------------|"
	assertContains(t, audit, expectedHeader+"\n"+expectedSeparator)
	assertContains(t, audit, "Converted Amounts")
	assertContains(t, audit, "unit_price: 27000 -> 25000; gross_value: 27000 -> 25000")
	assertNotContains(t, audit, "Amount Kind")
	assertNotContains(t, audit, "Original Amount")
	assertNotContains(t, audit, "Converted Amount |")
	assertNotContains(t, audit, "Rate Authority")
	assertNotContains(t, audit, "Rate Kind")
	assertContains(t, rateSourceSummarySection(document.Content), "- Authority: European Central Bank")
	assertContains(t, rateSourceSummarySection(document.Content), "- Rate Kind: daily euro foreign exchange reference rate")
}

// TestMarkdownReportConversionAuditGroupsAmountsAndSuppressesZeroSlots verifies
// BUG-005 row cardinality and zero-to-zero amount omission.
// Authored by: OpenCode
func TestMarkdownReportConversionAuditGroupsAmountsAndSuppressesZeroSlots(t *testing.T) {
	t.Parallel()

	var document, err = reportmarkdown.Render(contractMarkdownReportFixture(reportmodel.ReportBaseCurrencyEUR.Label()))
	if err != nil {
		t.Fatalf("render markdown report: %v", err)
	}

	var audit = currencyConversionAuditSection(document.Content)
	if got := countOccurrences(audit, "| 2024-01-01 | btc-sell-2024-001 |"); got != 1 {
		t.Fatalf("expected one audit row for the converted source activity, got %d in %q", got, audit)
	}
	assertContains(t, audit, "unit_price: 27000 -> 25000; gross_value: 27000 -> 25000")
	assertNotContains(t, audit, "| fee_amount |")
	assertNotContains(t, audit, "fee_amount: 0 -> 0")
	assertNotContains(t, audit, "0 | EUR | 0")
}

// TestMarkdownReportDistinguishesSameCurrencyAndConvertedRows verifies that
// priced activity rows expose whether conversion changed their monetary values.
// Authored by: OpenCode
func TestMarkdownReportDistinguishesSameCurrencyAndConvertedRows(t *testing.T) {
	t.Parallel()

	var reportCalculationCurrency = reportmodel.ReportBaseCurrencyEUR.Label()
	document, err := reportmarkdown.Render(contractMarkdownReportFixture(reportCalculationCurrency))
	if err != nil {
		t.Fatalf("render markdown report: %v", err)
	}

	assertContains(t, document.Content, "| Date | Source ID | Type | Quantity | Unit Price | Gross Value | Fee | Activity Currency | Basis After Row | Calculation Currency | Quantity After Row | Conversion Status | Note |")
	assertContains(t, document.Content, "| 2024-01-01 00:15:00 | btc-sell-2024-001 | SELL | 1 | 25000 | 25000 | 0 | USD | 22009 | EUR | 1 | converted | note token=[REDACTED] |")
	assertContains(t, document.Content, "| 2024-02-01 09:30:00 | eth-sell-2024-001 | SELL | 2 | 1000 | 2000 | 0 | EUR | 1000 | EUR | 3 | same_currency | same-currency priced sale |")
	assertNotContains(t, document.Content, "| 2024-02-01 | eth-sell-2024-001 | ETH | EUR | EUR |")
	assertNotContains(t, document.Content, "secret-token")
}

// TestMarkdownReportOutputFileContract verifies the visible output-file
// contract points that are direct consequences of successful Markdown
// rendering.
// Authored by: OpenCode
func TestMarkdownReportOutputFileContract(t *testing.T) {
	var fixture = testutil.NewReportIOFixture(t)
	var report = contractMarkdownReportFixture(reportmodel.ReportBaseCurrencyEUR.Label())
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

// contractMarkdownReportFixture returns one deterministic calculated report with
// the selected calculation currency used by the Markdown contract tests.
// Authored by: OpenCode
func contractMarkdownReportFixture(reportCalculationCurrency string) reportmodel.CapitalGainsReport {
	return reportmodel.CapitalGainsReport{
		Year:                      2024,
		CostBasisMethod:           reportmodel.CostBasisMethodFIFO,
		GeneratedAt:               time.Date(2026, time.May, 21, 12, 34, 56, 0, time.Local),
		ReportCalculationCurrency: reportCalculationCurrency,
		SummaryEntries: []reportmodel.AssetSummaryEntry{{
			AssetIdentityKey:          "asset-btc",
			DisplayLabel:              "BTC",
			NetGainOrLoss:             mustContractDecimal("1240.5"),
			ReportCalculationCurrency: reportCalculationCurrency,
		}, {
			AssetIdentityKey:          "asset-eth",
			DisplayLabel:              "ETH",
			NetGainOrLoss:             mustContractDecimal("1000"),
			ReportCalculationCurrency: reportCalculationCurrency,
		}, {
			AssetIdentityKey:          "asset-xrp",
			DisplayLabel:              "XRP",
			NetGainOrLoss:             mustContractDecimal("0"),
			ReportCalculationCurrency: reportCalculationCurrency,
		}},
		YearlyNetTotal: mustContractDecimal("2240.5"),
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
			CalculationCurrency: reportCalculationCurrency,
			ActivityRows: []reportmodel.AssetActivityRow{{
				SourceID:                    "btc-sell-2024-001",
				OccurredAt:                  time.Date(2024, time.January, 1, 0, 15, 0, 0, time.Local),
				ActivityType:                reportmodel.ActivityTypeSell,
				Quantity:                    mustContractDecimal("1"),
				UnitPrice:                   contractDecimalPointer("25000"),
				GrossValue:                  contractDecimalPointer("25000"),
				FeeAmount:                   contractDecimalPointer("0"),
				ActivityCurrency:            "USD",
				BasisAfterRow:               mustContractDecimal("22009"),
				CalculationCurrency:         reportCalculationCurrency,
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
				CalculationCurrency:    reportCalculationCurrency,
			}},
		}, {
			AssetIdentityKey:    "asset-eth",
			DisplayLabel:        "ETH",
			OpeningQuantity:     mustContractDecimal("5"),
			OpeningCostBasis:    mustContractDecimal("2500"),
			ClosingQuantity:     mustContractDecimal("3"),
			ClosingCostBasis:    mustContractDecimal("1500"),
			CalculationCurrency: reportCalculationCurrency,
			ActivityRows: []reportmodel.AssetActivityRow{{
				SourceID:                    "eth-sell-2024-001",
				OccurredAt:                  time.Date(2024, time.February, 1, 9, 30, 0, 0, time.Local),
				ActivityType:                reportmodel.ActivityTypeSell,
				Quantity:                    mustContractDecimal("2"),
				UnitPrice:                   contractDecimalPointer("1000"),
				GrossValue:                  contractDecimalPointer("2000"),
				FeeAmount:                   contractDecimalPointer("0"),
				ActivityCurrency:            "EUR",
				BasisAfterRow:               mustContractDecimal("1000"),
				CalculationCurrency:         reportCalculationCurrency,
				QuantityAfterRow:            mustContractDecimal("3"),
				HoldingReductionExplanation: "same-currency priced sale",
			}},
			LiquidationSummaries: []reportmodel.LiquidationCalculation{{
				SourceID:               "eth-sell-2024-001",
				OccurredAt:             time.Date(2024, time.February, 1, 9, 30, 0, 0, time.Local),
				DisposedQuantity:       mustContractDecimal("2"),
				AllocatedBasis:         mustContractDecimal("1000"),
				NetLiquidationProceeds: mustContractDecimal("2000"),
				GainOrLoss:             mustContractDecimal("1000"),
				ActivityCurrency:       "EUR",
				CalculationCurrency:    reportCalculationCurrency,
			}},
		}, {
			AssetIdentityKey:    "asset-xrp",
			DisplayLabel:        "XRP",
			OpeningQuantity:     mustContractDecimal("1000"),
			OpeningCostBasis:    mustContractDecimal("500"),
			ClosingQuantity:     mustContractDecimal("800"),
			ClosingCostBasis:    mustContractDecimal("400"),
			CalculationCurrency: reportCalculationCurrency,
			ActivityRows: []reportmodel.AssetActivityRow{{
				SourceID:                    "xrp-reduction-2024-001",
				OccurredAt:                  time.Date(2024, time.April, 1, 12, 0, 0, 0, time.Local),
				ActivityType:                reportmodel.ActivityTypeSell,
				Quantity:                    mustContractDecimal("200"),
				UnitPrice:                   contractDecimalPointer("0"),
				GrossValue:                  contractDecimalPointer("0"),
				FeeAmount:                   contractDecimalPointer("0"),
				BasisAfterRow:               mustContractDecimal("400"),
				CalculationCurrency:         reportCalculationCurrency,
				QuantityAfterRow:            mustContractDecimal("800"),
				HoldingReductionExplanation: "custody transfer",
			}},
		}},
		ConversionAuditEntries: []reportmodel.ConversionAuditEntry{{
			SourceID:           "btc-sell-2024-001",
			AssetLabel:         "BTC",
			ActivityDate:       time.Date(2024, time.January, 1, 0, 15, 0, 0, time.Local),
			SourceCurrency:     "USD",
			ReportBaseCurrency: reportmodel.ReportBaseCurrencyEUR,
			RateDate:           time.Date(2023, time.December, 29, 0, 0, 0, 0, time.Local),
			RateAuthority:      reportmodel.RateAuthorityEuropeanCentralBank,
			RateKind:           "daily euro foreign exchange reference rate",
			RateValue:          mustContractDecimal("1.08"),
			QuoteDirection:     reportmodel.QuoteDirectionSourcePerBase,
			Amounts: []reportmodel.ConvertedActivityAmount{
				contractConvertedActivityAmount(reportmodel.ConvertedAmountKindUnitPrice, "27000", "25000"),
				contractConvertedActivityAmount(reportmodel.ConvertedAmountKindGrossValue, "27000", "25000"),
				contractConvertedActivityAmount(reportmodel.ConvertedAmountKindFeeAmount, "0", "0"),
			},
		}},
		RateSources: []reportmodel.ExchangeRateEvidence{{
			SourceCurrency:   "USD",
			BaseCurrency:     reportmodel.ReportBaseCurrencyEUR,
			ActivityDate:     time.Date(2024, time.January, 1, 0, 15, 0, 0, time.Local),
			RateDate:         time.Date(2023, time.December, 29, 0, 0, 0, 0, time.Local),
			Authority:        reportmodel.RateAuthorityEuropeanCentralBank,
			ProviderID:       reportmodel.RateProviderIDECBEXR,
			RateKind:         "daily euro foreign exchange reference rate",
			QuoteDirection:   reportmodel.QuoteDirectionSourcePerBase,
			RateValue:        mustContractDecimal("1.08"),
			DatasetReference: "ECB Data Portal `EXR`",
		}},
	}
}

// contractConversionAuditEntry returns one conversion audit fixture with
// matching amount-level exchange-rate evidence.
// Authored by: OpenCode
func contractConversionAuditEntry(sourceID string, assetLabel string, sourceCurrency string, rateValue string, quoteDirection reportmodel.QuoteDirection) reportmodel.ConversionAuditEntry {
	var activityDate = time.Date(2024, time.January, 2, 0, 15, 0, 0, time.Local)
	var evidence = reportmodel.ExchangeRateEvidence{
		SourceCurrency:   sourceCurrency,
		BaseCurrency:     reportmodel.ReportBaseCurrencyEUR,
		ActivityDate:     activityDate,
		RateDate:         time.Date(2024, time.January, 2, 0, 0, 0, 0, time.Local),
		Authority:        reportmodel.RateAuthorityEuropeanCentralBank,
		ProviderID:       reportmodel.RateProviderIDECBEXR,
		RateKind:         "daily euro foreign exchange reference rate",
		QuoteDirection:   quoteDirection,
		RateValue:        mustContractDecimal(rateValue),
		DatasetReference: "ECB Data Portal `EXR`",
	}

	return reportmodel.ConversionAuditEntry{
		SourceID:           sourceID,
		AssetLabel:         assetLabel,
		ActivityDate:       activityDate,
		SourceCurrency:     sourceCurrency,
		ReportBaseCurrency: reportmodel.ReportBaseCurrencyEUR,
		RateDate:           evidence.RateDate,
		RateAuthority:      evidence.Authority,
		RateKind:           evidence.RateKind,
		RateValue:          evidence.RateValue,
		QuoteDirection:     evidence.QuoteDirection,
		Amounts: []reportmodel.ConvertedActivityAmount{{
			SourceID:             sourceID,
			AmountKind:           reportmodel.ConvertedAmountKindGrossValue,
			OriginalCurrency:     sourceCurrency,
			OriginalAmount:       mustContractDecimal("116"),
			ReportBaseCurrency:   reportmodel.ReportBaseCurrencyEUR,
			ConvertedAmount:      mustContractDecimal("100"),
			ExchangeRateEvidence: &evidence,
			ConversionStatus:     reportmodel.ConversionStatusConverted,
		}},
	}
}

// rateSourceSummarySection extracts the Rate Source Summary block from a
// rendered Markdown document.
// Authored by: OpenCode
func rateSourceSummarySection(content string) string {
	var start = strings.Index(content, "## Rate Source Summary")
	if start < 0 {
		return ""
	}
	var rest = content[start:]
	var next = strings.Index(rest[len("## Rate Source Summary"):], "\n## ")
	if next < 0 {
		return rest
	}
	return rest[:len("## Rate Source Summary")+next]
}

// currencyConversionAuditSection extracts the Currency Conversion Audit block
// from a rendered Markdown document.
// Authored by: OpenCode
func currencyConversionAuditSection(content string) string {
	var start = strings.Index(content, "## Currency Conversion Audit")
	if start < 0 {
		return ""
	}
	var rest = content[start:]
	var next = strings.Index(rest[len("## Currency Conversion Audit"):], "\n## ")
	if next < 0 {
		return rest
	}
	return rest[:len("## Currency Conversion Audit")+next]
}

// countOccurrences counts non-overlapping occurrences of a substring.
// Authored by: OpenCode
func countOccurrences(content string, needle string) int {
	return strings.Count(content, needle)
}

// contractConvertedActivityAmount returns one converted amount tied to the
// canonical BTC conversion audit fixture.
// Authored by: OpenCode
func contractConvertedActivityAmount(kind reportmodel.ConvertedAmountKind, original string, converted string) reportmodel.ConvertedActivityAmount {
	var evidence = reportmodel.ExchangeRateEvidence{
		SourceCurrency:   "USD",
		BaseCurrency:     reportmodel.ReportBaseCurrencyEUR,
		ActivityDate:     time.Date(2024, time.January, 1, 0, 15, 0, 0, time.Local),
		RateDate:         time.Date(2023, time.December, 29, 0, 0, 0, 0, time.Local),
		Authority:        reportmodel.RateAuthorityEuropeanCentralBank,
		ProviderID:       reportmodel.RateProviderIDECBEXR,
		RateKind:         "daily euro foreign exchange reference rate",
		QuoteDirection:   reportmodel.QuoteDirectionSourcePerBase,
		RateValue:        mustContractDecimal("1.08"),
		DatasetReference: "ECB Data Portal `EXR`",
	}

	return reportmodel.ConvertedActivityAmount{
		SourceID:             "btc-sell-2024-001",
		AmountKind:           kind,
		OriginalCurrency:     "USD",
		OriginalAmount:       mustContractDecimal(original),
		ReportBaseCurrency:   reportmodel.ReportBaseCurrencyEUR,
		ConvertedAmount:      mustContractDecimal(converted),
		ExchangeRateEvidence: &evidence,
		ConversionStatus:     reportmodel.ConversionStatusConverted,
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

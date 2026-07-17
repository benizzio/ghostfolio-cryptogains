// Package unit verifies focused report-Markdown rendering seams without the
// full yearly report runtime.
// Authored by: OpenCode
package unit

import (
	"reflect"
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

// TestRenderMarkdownEmitsExactStandaloneWarning verifies the fully bold warning
// appears once between metadata and the summary and not in Annex 1.
// Authored by: OpenCode
func TestRenderMarkdownEmitsExactStandaloneWarning(t *testing.T) {
	var report = populatedMarkdownReportFixture()
	var warning = "The data in this report does not follow any legally required rules for any country's tax returns and is for reference only."
	var expectedLine = "**" + warning + "**"

	var document, err = reportmarkdown.Render(report)
	if err != nil {
		t.Fatalf("render markdown report: %v", err)
	}
	var content = string(document.Content)
	if strings.Count(content, expectedLine) != 1 {
		t.Fatalf("expected one fully bold warning line, got %d in %q", strings.Count(content, expectedLine), content)
	}
	if strings.Count(content, warning) != 1 {
		t.Fatalf("expected one warning sentence, got %d in %q", strings.Count(content, warning), content)
	}
	if !strings.Contains(content, "\n"+expectedLine+"\n\n") {
		t.Fatalf("expected warning to occupy one standalone Markdown line, got %q", content)
	}
	assertContainsInOrder(t, content, "- **Report Calculation Currency:** USD", expectedLine, "## Gains-And-Losses Summary")

	var annex, annexErr = reportmarkdown.RenderAnnex(report)
	if annexErr != nil {
		t.Fatalf("render Annex 1: %v", annexErr)
	}
	assertNotContainsString(t, annex.Content, warning)
}

// TestRenderMarkdownFormatsEveryFinancialField verifies the complete direct and
// row-built financial field set in a validated main report and Annex 1.
// Authored by: OpenCode
func TestRenderMarkdownFormatsEveryFinancialField(t *testing.T) {
	var report = markdownFinancialMatrixReportFixture(t, "1.005")
	var before = report

	var document, err = reportmarkdown.Render(report)
	if err != nil {
		t.Fatalf("render main financial matrix: %v", err)
	}
	var annex, annexErr = reportmarkdown.RenderAnnex(report)
	if annexErr != nil {
		t.Fatalf("render Annex financial matrix: %v", annexErr)
	}
	var mainContent = string(document.Content)
	var annexContent = string(annex.Content)
	for _, expected := range []string{
		"| BTC | 1.01 | USD |",
		"| Overall Yearly Net Total | 1.01 | USD |",
		"- **Cost Basis:** 1.01",
		"| btc-sell-2024-001 | SELL | 0.00000001 | 1.01 | 1.01 | 1.01 | 0.1 | 1.01 | USD | USD |",
		"| btc-sell-2024-001 | 0.00000001 | 1.01 | 1.01 | 1.01 | USD |",
	} {
		assertContainsString(t, mainContent, expected)
	}
	if strings.Count(mainContent, "- **Cost Basis:** 1.01") != 3 {
		t.Fatalf("expected opening, closing, and historical cost bases to be formatted, got %q", mainContent)
	}
	for _, expected := range []string{
		"| matrix-annex | BUY | 0.00000001 | 1.01 | 1.01 | 1.01 | USD | USD | 0.1 | 2.00 |",
		"| 1.01 | 1.01 | 1.01 | Same currency |",
		"unit_price: 1.01 -> 1.01",
		"gross_value: 1.01 -> 1.01",
		"fee_amount: 1.01 -> 1.01",
		"| 0.8601 |",
	} {
		assertContainsString(t, annexContent, expected)
	}
	if !strings.Contains(mainContent, "| btc-sell-2024-001 | SELL | 0.00000001 |") || !strings.Contains(annexContent, "| matrix-annex | BUY | 0.00000001 |") {
		t.Fatalf("expected canonical quantity values in both documents, main=%q annex=%q", mainContent, annexContent)
	}
	if !reflect.DeepEqual(report, before) {
		t.Fatalf("rendering changed the source report: before=%#v after=%#v", before, report)
	}
}

// TestRenderMarkdownAppliesFinancialMatrixVectors verifies representative
// non-negative matrix vectors at every visible financial field boundary.
// Authored by: OpenCode
func TestRenderMarkdownAppliesFinancialMatrixVectors(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		raw      string
		expected string
	}{
		{name: "zero", raw: "0", expected: "0.00"},
		{name: "tiny-positive", raw: "0.00000001", expected: "0.00"},
		{name: "whole", raw: "1", expected: "1.00"},
		{name: "one-place", raw: "1.2", expected: "1.20"},
		{name: "below-tie", raw: "1.004", expected: "1.00"},
		{name: "exact-tie", raw: "1.005", expected: "1.01"},
		{name: "carry", raw: "9.995", expected: "10.00"},
	} {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			var report = markdownFinancialMatrixReportFixture(t, testCase.raw)
			var document, err = reportmarkdown.Render(report)
			if err != nil {
				t.Fatalf("render main matrix vector: %v", err)
			}
			var annex, annexErr = reportmarkdown.RenderAnnex(report)
			if annexErr != nil {
				t.Fatalf("render Annex matrix vector: %v", annexErr)
			}
			var mainContent = string(document.Content)
			var annexContent = string(annex.Content)
			assertContainsString(t, mainContent, "| Overall Yearly Net Total | "+testCase.expected+" | USD |")
			var activityType = "SELL"
			if testCase.raw == "0" {
				activityType = "BLOCKCHAIN OP"
			}
			assertContainsString(t, mainContent, "| btc-sell-2024-001 | "+activityType+" | 0.00000001 | "+testCase.expected+" | "+testCase.expected+" | "+testCase.expected+" | 0.1 | "+testCase.expected+" |")
			assertContainsString(t, annexContent, "| matrix-annex | BUY | 0.00000001 | "+testCase.expected+" | "+testCase.expected+" | "+testCase.expected+" |")
		})
	}
}

// TestRenderMarkdownLeavesNilFinancialOptionalsBlank verifies nil activity and
// Annex amounts remain empty cells rather than visible zero values.
// Authored by: OpenCode
func TestRenderMarkdownLeavesNilFinancialOptionalsBlank(t *testing.T) {
	var report = markdownFinancialMatrixReportFixture(t, "1")
	report.DetailSections[0].ActivityRows[0].UnitPrice = nil
	report.DetailSections[0].ActivityRows[0].GrossValue = nil
	report.DetailSections[0].ActivityRows[0].FeeAmount = nil
	report.AuditAnnex.PerAssetAuditSections[0].Entries[0].UnitPrice = nil
	report.AuditAnnex.PerAssetAuditSections[0].Entries[0].GrossValue = nil
	report.AuditAnnex.PerAssetAuditSections[0].Entries[0].FeeAmount = nil
	report.AuditAnnex.PerAssetAuditSections[0].Entries[0].AllocatedBasis = nil
	report.AuditAnnex.PerAssetAuditSections[0].Entries[0].NetLiquidationProceeds = nil
	report.AuditAnnex.PerAssetAuditSections[0].Entries[0].GainOrLoss = nil

	var document, err = reportmarkdown.Render(report)
	if err != nil {
		t.Fatalf("render nil-optionals main report: %v", err)
	}
	var annex, annexErr = reportmarkdown.RenderAnnex(report)
	if annexErr != nil {
		t.Fatalf("render nil-optionals Annex: %v", annexErr)
	}
	assertContainsString(t, document.Content, "| btc-sell-2024-001 | SELL | 0.00000001 |  |  |  | 0.1 | 1.00 |  | USD |")
	assertContainsString(t, annex.Content, "| matrix-annex | BUY | 0.00000001 |  |  |  | USD | USD | 0.1 | 2.00 |")
}

// TestRenderMarkdownPreservesCanonicalQuantitiesRatesAndSourceValues verifies
// quantity and normalized-rate text remain canonical and the input model stays
// unchanged after both Markdown render operations.
// Authored by: OpenCode
func TestRenderMarkdownPreservesCanonicalQuantitiesRatesAndSourceValues(t *testing.T) {
	var report = markdownFinancialMatrixReportFixture(t, "1.005")
	var before = report
	var document, err = reportmarkdown.Render(report)
	if err != nil {
		t.Fatalf("render canonical-value main report: %v", err)
	}
	var annex, annexErr = reportmarkdown.RenderAnnex(report)
	if annexErr != nil {
		t.Fatalf("render canonical-value Annex: %v", annexErr)
	}
	assertContainsString(t, document.Content, "| btc-sell-2024-001 | SELL | 0.00000001 | 1.01 |")
	assertContainsString(t, document.Content, "| 0.1 | 1.01 | USD | USD |")
	assertContainsString(t, annex.Content, "| matrix-annex | BUY | 0.00000001 | 1.01 |")
	assertContainsString(t, annex.Content, "| 0.8601 |")
	assertNotContainsString(t, annex.Content, "| 0.86 |")
	if !reflect.DeepEqual(report, before) {
		t.Fatalf("rendering changed quantities, rates, or financial source values: before=%#v after=%#v", before, report)
	}
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

// markdownFinancialMatrixReportFixture returns a validated report with one
// exact financial value copied into every main and Annex matrix field.
// Authored by: OpenCode
func markdownFinancialMatrixReportFixture(t *testing.T, raw string) reportmodel.CapitalGainsReport {
	var report = populatedMarkdownReportFixture()
	var value = mustMarkdownDecimal(t, raw)
	report.SummaryEntries[0].NetGainOrLoss = value
	report.YearlyNetTotal = value
	report.DetailSections[0].OpeningQuantity = mustMarkdownDecimal(t, "2.000")
	report.DetailSections[0].OpeningCostBasis = value
	report.DetailSections[0].ClosingQuantity = mustMarkdownDecimal(t, "0.00000001")
	report.DetailSections[0].ClosingCostBasis = value
	report.DetailSections[0].ActivityRows[0].Quantity = mustMarkdownDecimal(t, "0.00000001")
	report.DetailSections[0].ActivityRows[0].QuantityAfterRow = mustMarkdownDecimal(t, "0.1000")
	report.DetailSections[0].ActivityRows[0].UnitPrice = markdownDecimalPointer(raw)
	report.DetailSections[0].ActivityRows[0].GrossValue = markdownDecimalPointer(raw)
	report.DetailSections[0].ActivityRows[0].FeeAmount = markdownDecimalPointer(raw)
	report.DetailSections[0].ActivityRows[0].BasisAfterRow = value
	report.DetailSections[0].LiquidationSummaries[0].DisposedQuantity = mustMarkdownDecimal(t, "0.00000001")
	report.DetailSections[0].LiquidationSummaries[0].AllocatedBasis = value
	report.DetailSections[0].LiquidationSummaries[0].NetLiquidationProceeds = value
	report.DetailSections[0].LiquidationSummaries[0].GainOrLoss = value
	report.DetailSections = append(report.DetailSections, reportmodel.AssetDetailSection{
		AssetIdentityKey:    "matrix-historical",
		DisplayLabel:        "Matrix Historical",
		ClosingQuantity:     mustMarkdownDecimal(t, "0.00000001"),
		ClosingCostBasis:    value,
		CalculationCurrency: "USD",
	})

	var conversion = markdownFinancialConversionEntry(t, raw)
	report.AuditAnnex = reportmodel.DefaultAuditAnnex()
	report.AuditAnnex.PerAssetAuditSections = []reportmodel.PerAssetAuditSection{{
		AssetIdentityKey: "matrix-asset",
		DisplayLabel:     "Matrix Asset",
		Entries:          []reportmodel.AuditActivityEntry{markdownFinancialAuditEntry(t, raw)},
	}}
	report.AuditAnnex.ConversionAuditEntries = []reportmodel.ConversionAuditEntry{conversion}
	report.RateSources = []reportmodel.ExchangeRateEvidence{*conversion.Amounts[0].ExchangeRateEvidence}
	return report
}

// markdownFinancialAuditEntry creates one valid Annex activity containing every
// optional monetary field for Markdown matrix assertions.
// Authored by: OpenCode
func markdownFinancialAuditEntry(t *testing.T, raw string) reportmodel.AuditActivityEntry {
	var value = mustMarkdownDecimal(t, raw)
	return reportmodel.AuditActivityEntry{
		SourceID:               "matrix-annex",
		OccurredAt:             time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
		ActivityType:           reportmodel.ActivityTypeBuy,
		Quantity:               mustMarkdownDecimal(t, "0.00000001"),
		UnitPrice:              markdownDecimalPointer(raw),
		GrossValue:             markdownDecimalPointer(raw),
		FeeAmount:              markdownDecimalPointer(raw),
		ActivityCurrency:       "USD",
		CalculationCurrency:    "USD",
		QuantityAfterActivity:  mustMarkdownDecimal(t, "0.1000"),
		BasisAfterActivity:     mustMarkdownDecimal(t, "2.000"),
		FullLiquidationEvent:   true,
		AllocatedBasis:         &value,
		NetLiquidationProceeds: markdownDecimalPointer(raw),
		GainOrLoss:             markdownDecimalPointer(raw),
		ConversionStatus:       reportmodel.ConversionStatusSameCurrency,
	}
}

// markdownFinancialConversionEntry creates one valid three-component conversion
// audit row with a normalized rate and the requested financial value.
// Authored by: OpenCode
func markdownFinancialConversionEntry(t *testing.T, raw string) reportmodel.ConversionAuditEntry {
	var activityDate = time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC)
	var rate = mustMarkdownDecimal(t, "0.86010")
	var evidence = reportmodel.ExchangeRateEvidence{
		SourceCurrency:   "EUR",
		BaseCurrency:     reportmodel.ReportBaseCurrencyUSD,
		ActivityDate:     activityDate,
		RateDate:         activityDate,
		Authority:        reportmodel.RateAuthorityFederalReserve,
		ProviderID:       reportmodel.RateProviderIDFederalReserveH10,
		RateKind:         "daily noon buying rate",
		QuoteDirection:   reportmodel.QuoteDirectionSourcePerBase,
		RateValue:        rate,
		DatasetReference: "synthetic Markdown matrix fixture",
	}
	var amounts = []reportmodel.ConvertedActivityAmount{
		markdownFinancialConvertedAmount(t, raw, reportmodel.ConvertedAmountKindUnitPrice, &evidence),
		markdownFinancialConvertedAmount(t, raw, reportmodel.ConvertedAmountKindGrossValue, &evidence),
		markdownFinancialConvertedAmount(t, raw, reportmodel.ConvertedAmountKindFeeAmount, &evidence),
	}
	return reportmodel.ConversionAuditEntry{
		SourceID:           "matrix-conversion",
		AssetLabel:         "Matrix Asset",
		ActivityDate:       activityDate,
		SourceCurrency:     "EUR",
		ReportBaseCurrency: reportmodel.ReportBaseCurrencyUSD,
		RateDate:           activityDate,
		RateAuthority:      reportmodel.RateAuthorityFederalReserve,
		RateKind:           "daily noon buying rate",
		RateValue:          rate,
		QuoteDirection:     reportmodel.QuoteDirectionSourcePerBase,
		Amounts:            amounts,
	}
}

// markdownFinancialConvertedAmount creates one converted amount tied to the
// shared synthetic exchange-rate evidence.
// Authored by: OpenCode
func markdownFinancialConvertedAmount(t *testing.T, raw string, kind reportmodel.ConvertedAmountKind, evidence *reportmodel.ExchangeRateEvidence) reportmodel.ConvertedActivityAmount {
	return reportmodel.ConvertedActivityAmount{
		SourceID:             "matrix-conversion",
		AmountKind:           kind,
		OriginalCurrency:     "EUR",
		OriginalAmount:       mustMarkdownDecimal(t, raw),
		ReportBaseCurrency:   reportmodel.ReportBaseCurrencyUSD,
		ConvertedAmount:      mustMarkdownDecimal(t, raw),
		ExchangeRateEvidence: evidence,
		ConversionStatus:     reportmodel.ConversionStatusConverted,
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

// TestRenderMarkdownAnnexUsesInheritedClassificationForVisibleCurrency
// verifies classified rows blank only visible original currency while an
// unclassified tiny-positive row retains EUR and displays its price as 0.00.
// Authored by: OpenCode
func TestRenderMarkdownAnnexUsesInheritedClassificationForVisibleCurrency(t *testing.T) {
	var report = markdownClassificationAnnexReportFixture(t)
	var beforeEntries = cloneMarkdownAuditEntries(report.AuditAnnex.PerAssetAuditSections[0].Entries)

	var annex, err = reportmarkdown.RenderAnnex(report)
	if err != nil {
		t.Fatalf("render classification Annex: %v", err)
	}

	var content = string(annex.Content)
	assertContainsString(t, content, "| annex-classified | BLOCKCHAIN OP | 1 | 0.00 | 0.00 | 0.00 |  | USD |")
	assertContainsString(t, content, "| annex-unclassified-tiny | SELL | 1 | 0.00 | 0.00 | 0.00 | EUR | USD |")
	if !reflect.DeepEqual(report.AuditAnnex.PerAssetAuditSections[0].Entries, beforeEntries) {
		t.Fatalf("rendering changed audit source entries: before=%#v after=%#v", beforeEntries, report.AuditAnnex.PerAssetAuditSections[0].Entries)
	}
}

// markdownClassificationAnnexReportFixture creates a report with classified
// and unclassified Annex controls without testing their upstream derivation.
// Authored by: OpenCode
func markdownClassificationAnnexReportFixture(t *testing.T) reportmodel.CapitalGainsReport {
	var report = populatedMarkdownReportFixture()
	report.AuditAnnex = reportmodel.DefaultAuditAnnex()
	report.AuditAnnex.PerAssetAuditSections = []reportmodel.PerAssetAuditSection{{
		AssetIdentityKey: "asset-classification",
		DisplayLabel:     "Classification Asset",
		Entries: []reportmodel.AuditActivityEntry{
			markdownClassificationAuditEntry(t, "annex-classified", true, "0"),
			markdownClassificationAuditEntry(t, "annex-unclassified-tiny", false, "0.00000001"),
		},
	}}
	return report
}

// markdownClassificationAuditEntry creates one synthetic Annex control with
// distinct source and calculation currencies for presentation assertions.
// Authored by: OpenCode
func markdownClassificationAuditEntry(t *testing.T, sourceID string, classified bool, unitPrice string) reportmodel.AuditActivityEntry {
	var value = mustMarkdownDecimal(t, unitPrice)
	return reportmodel.AuditActivityEntry{
		SourceID:                     sourceID,
		OccurredAt:                   time.Date(2024, time.August, 9, 0, 0, 0, 0, time.UTC),
		ActivityType:                 reportmodel.ActivityTypeSell,
		Quantity:                     mustMarkdownDecimal(t, "1"),
		UnitPrice:                    &value,
		GrossValue:                   markdownDecimalPointer(unitPrice),
		FeeAmount:                    markdownDecimalPointer("0"),
		ActivityCurrency:             "EUR",
		CalculationCurrency:          "USD",
		QuantityAfterActivity:        mustMarkdownDecimal(t, "0"),
		BasisAfterActivity:           mustMarkdownDecimal(t, "1"),
		FullLiquidationEvent:         false,
		AllocatedBasis:               markdownDecimalPointer("1"),
		NetLiquidationProceeds:       markdownDecimalPointer(unitPrice),
		GainOrLoss:                   markdownDecimalPointer(unitPrice),
		IsZeroPricedHoldingReduction: classified,
	}
}

// cloneMarkdownAuditEntries makes a deep copy of Annex decimal fields for
// source-immutability assertions around Markdown rendering.
// Authored by: OpenCode
func cloneMarkdownAuditEntries(entries []reportmodel.AuditActivityEntry) []reportmodel.AuditActivityEntry {
	var cloned = append([]reportmodel.AuditActivityEntry(nil), entries...)
	for index := range cloned {
		cloned[index].Quantity = cloneMarkdownDecimal(cloned[index].Quantity)
		cloned[index].UnitPrice = cloneMarkdownDecimalPointer(cloned[index].UnitPrice)
		cloned[index].GrossValue = cloneMarkdownDecimalPointer(cloned[index].GrossValue)
		cloned[index].FeeAmount = cloneMarkdownDecimalPointer(cloned[index].FeeAmount)
		cloned[index].QuantityAfterActivity = cloneMarkdownDecimal(cloned[index].QuantityAfterActivity)
		cloned[index].BasisAfterActivity = cloneMarkdownDecimal(cloned[index].BasisAfterActivity)
		cloned[index].AllocatedBasis = cloneMarkdownDecimalPointer(cloned[index].AllocatedBasis)
		cloned[index].NetLiquidationProceeds = cloneMarkdownDecimalPointer(cloned[index].NetLiquidationProceeds)
		cloned[index].GainOrLoss = cloneMarkdownDecimalPointer(cloned[index].GainOrLoss)
	}
	return cloned
}

// cloneMarkdownDecimal makes a deep decimal copy for source snapshots.
// Authored by: OpenCode
func cloneMarkdownDecimal(value apd.Decimal) apd.Decimal {
	var clone apd.Decimal
	clone.Set(&value)
	return clone
}

// cloneMarkdownDecimalPointer makes a deep optional decimal copy for source
// snapshots.
// Authored by: OpenCode
func cloneMarkdownDecimalPointer(value *apd.Decimal) *apd.Decimal {
	if value == nil {
		return nil
	}
	var clone = cloneMarkdownDecimal(*value)
	return &clone
}

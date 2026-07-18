// Package contract verifies Annex 1 rendering contracts for report output.
// Authored by: OpenCode
package contract

import (
	"math"
	"strings"
	"testing"
	"time"

	reportmarkdown "github.com/benizzio/ghostfolio-cryptogains/internal/report/markdown"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	testreportpdf "github.com/benizzio/ghostfolio-cryptogains/tests/testutil/reportpdf"
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
	assertContains(t, document.Content, "| 2023-12-31 23:15:00 | btc-sell-2024-001 | SELL | 1 | 25000.00 | 25000.00 | 0.00 | USD | EUR | 1 | 22009.00 | Yes | 22009.00 | 25000.00 | 1240.50 | Converted | note token=[REDACTED] |")
	assertContains(t, document.Content, "| 2024-04-01 10:00:00 | xrp-reduction-2024-001 | BLOCKCHAIN OP | 200 | 0.00 | 0.00 | 0.00 |  | EUR | 800 | 400.00 | No |  |  |  | Same currency | custody transfer |")
	assertContains(t, document.Content, "| 2023-01-01 10:00:00 | eth-reference-buy | BUY | 1 | 50.00 | 50.00 | 0.00 | EUR | EUR | 1 | 50.00 | No |  |  |  | Same currency | reference-only acquisition |")
	assertContains(t, document.Content, "| 2024-05-01 10:00:00 | tiny-positive-unclassified | SELL | 1 | 0.00 | 0.00 | 0.00 | GBP | USD | 1 | 0.00 | No |  |  |  | Converted | tiny positive unclassified control |")
	assertNotContains(t, document.Content, " | true |")
	assertNotContains(t, document.Content, " | false |")
	assertContains(t, document.Content, "## Currency Conversion Audit")
	assertContains(t, document.Content, "| 2024-01-01 | btc-sell-2024-001 | BTC | 2023-12-29 | USD | EUR | unit_price: 27000.00 -> 25000.00;<br>gross_value: 27000.00 -> 25000.00 | Source currency per base currency | 1.08 |")
	assertNotContains(t, document.Content, "source_per_base")
	assertNotContains(t, document.Content, "base_per_source")
	assertNotContains(t, document.Content, "secret-token")
	assertNotContains(t, document.Content, "post-year")
}

// TestReportAnnexUS2PopulationContract verifies that both boolean states and
// every classified/unclassified currency control remain non-empty in the
// closed acceptance manifest.
// Authored by: OpenCode
func TestReportAnnexUS2PopulationContract(t *testing.T) {
	t.Parallel()

	var manifest = testutil.DeterministicReportPresentationAcceptanceFixture()
	var expected = map[testutil.ReportPresentationPopulation]int{
		testutil.ReportPresentationPopulationBoolean:            16,
		testutil.ReportPresentationPopulationClassifiedCurrency: 2,
		testutil.ReportPresentationPopulationUnclassified:       4,
	}
	for population, want := range expected {
		var numerator = reportRenderingPopulationCount(manifest.Cases, population)
		var denominator = reportRenderingPopulationCounter(t, manifest.Counters, population)
		if numerator == 0 || denominator == 0 {
			t.Fatalf("Annex population %s must be non-empty: %d/%d", population, numerator, denominator)
		}
		if numerator != want || denominator != want {
			t.Fatalf("Annex population %s numerator/denominator = %d/%d, want %d/%d", population, numerator, denominator, want, want)
		}
	}
}

// TestReportAnnexConcretePDFContract verifies the concrete combined PDF exposes
// both boolean labels and the exact classified/unclassified currency controls
// in selectable Annex text.
// Authored by: OpenCode
func TestReportAnnexConcretePDFContract(t *testing.T) {
	t.Parallel()

	var report = contractMarkdownReportFixture(reportmodel.ReportBaseCurrencyEUR.Label())
	report.AuditAnnex = contractDetailedAuditAnnex()
	var inspection, err = testreportpdf.RenderAndInspect(report)
	if err != nil {
		t.Fatalf("render PDF report: %v", err)
	}
	assertLandscapeA4PDF(t, inspection)
	if !inspection.ContainsSearchableText("Annex 1 - Audit") {
		t.Fatal("generated PDF Annex title is not searchable")
	}
	assertContractPDFAuditRow(t, inspection, "btc-sell-2024-001", "Yes", map[string]int{"USD": 1, "EUR": 1})
	assertContractPDFAuditRow(t, inspection, "xrp-reduction-2024-001", "No", map[string]int{"USD": 0, "EUR": 1})
	assertContractPDFAuditRow(t, inspection, "eth-reference-buy", "No", map[string]int{"EUR": 2})
	assertContractPDFAuditRow(t, inspection, "tiny-positive-unclassified", "No", map[string]int{"GBP": 1, "USD": 1})
	var searchable = strings.ToLower(inspection.SearchableText)
	if strings.Contains(searchable, "true") || strings.Contains(searchable, "false") {
		t.Fatalf("generated PDF exposes lowercase boolean value: %q", inspection.SearchableText)
	}
}

// assertContractPDFAuditRow verifies one concrete PDF row's boolean and currency
// tokens at the row baseline, so a blank classified cell cannot pass by using a
// value found in another Annex row.
// Authored by: OpenCode
func assertContractPDFAuditRow(t *testing.T, inspection testutil.GeneratedPDF, sourceID string, wantBoolean string, currencyCounts map[string]int) {
	t.Helper()

	var sourceRuns, found = contractPDFAuditSourceRuns(inspection, sourceID)
	if !found {
		t.Fatalf("generated PDF does not contain Annex source ID %q", sourceID)
	}

	var rowRuns = contractPDFAuditRowRuns(inspection, sourceRuns)
	var rowText = contractPDFAuditRunTexts(rowRuns)
	var rowTokens = strings.Fields(strings.Join(rowText, " "))
	if !containsContractPDFToken(rowTokens, wantBoolean) {
		t.Fatalf("PDF Annex row %q lacks boolean %q: %q", sourceID, wantBoolean, rowText)
	}
	for currency, want := range currencyCounts {
		var got = countContractPDFToken(rowTokens, currency)
		if got != want {
			t.Fatalf("PDF Annex row %q currency %q count = %d, want %d: %q", sourceID, currency, got, want, rowText)
		}
	}
}

// contractPDFAuditSourceRuns locates a detailed Annex source ID from its X/Y
// neighborhood, avoiding the repeated source ID in the conversion table.
// Authored by: OpenCode
func contractPDFAuditSourceRuns(inspection testutil.GeneratedPDF, sourceID string) ([]testutil.PDFTextRun, bool) {
	var annexPage, conversionPage int
	var conversionY float64
	var foundAnnex, foundConversion bool
	for _, run := range inspection.TextRuns {
		if run.Text == "Annex 1 - Audit" && !foundAnnex {
			annexPage = run.Page
			foundAnnex = true
		}
		if run.Text == "Currency Conversion Audit" && foundAnnex {
			conversionPage = run.Page
			conversionY = run.Y
			foundConversion = true
			break
		}
	}
	if !foundAnnex || !foundConversion {
		return nil, false
	}

	var target = contractPDFAuditSourceText(sourceID)
	for index, run := range inspection.TextRuns {
		if !contractPDFAuditDetailedRun(run, annexPage, conversionPage, conversionY) || contractPDFAuditSourceText(run.Text) == "" {
			continue
		}
		var candidate []testutil.PDFTextRun
		var normalized strings.Builder
		for next := index; next < len(inspection.TextRuns); next++ {
			var fragment = inspection.TextRuns[next]
			if !contractPDFAuditDetailedRun(fragment, annexPage, conversionPage, conversionY) || math.Abs(fragment.X-run.X) > 0.1 {
				break
			}
			if len(candidate) > 0 && math.Abs(fragment.Y-candidate[len(candidate)-1].Y) > 16 {
				break
			}
			candidate = append(candidate, fragment)
			normalized.WriteString(contractPDFAuditSourceText(fragment.Text))
			if strings.Contains(normalized.String(), target) {
				return candidate, true
			}
		}
	}
	return nil, false
}

// contractPDFAuditDetailedRun limits source-ID lookup to the detailed Annex
// section, including rows before a conversion heading on the same page.
// Authored by: OpenCode
func contractPDFAuditDetailedRun(run testutil.PDFTextRun, annexPage int, conversionPage int, conversionY float64) bool {
	return run.Page >= annexPage && (run.Page < conversionPage || run.Y > conversionY)
}

// contractPDFAuditSourceText removes PDF line whitespace without changing the
// source ID's semantic punctuation.
// Authored by: OpenCode
func contractPDFAuditSourceText(value string) string {
	return strings.Join(strings.Fields(value), "")
}

// contractPDFAuditRowRuns expands a detailed source cell to its adjacent row
// baselines while excluding neighboring rows and section headings.
// Authored by: OpenCode
func contractPDFAuditRowRuns(inspection testutil.GeneratedPDF, sourceRuns []testutil.PDFTextRun) []testutil.PDFTextRun {
	if len(sourceRuns) == 0 {
		return nil
	}
	var minimumY = sourceRuns[0].Y
	var maximumY = sourceRuns[0].Y
	for _, run := range sourceRuns[1:] {
		minimumY = math.Min(minimumY, run.Y)
		maximumY = math.Max(maximumY, run.Y)
	}
	for {
		var expanded bool
		for _, run := range inspection.TextRuns {
			if run.Page != sourceRuns[0].Page {
				continue
			}
			if run.Y > maximumY && run.Y-maximumY <= 16 {
				maximumY = run.Y
				expanded = true
			}
			if run.Y < minimumY && minimumY-run.Y <= 16 {
				minimumY = run.Y
				expanded = true
			}
		}
		if !expanded {
			break
		}
	}

	var rowRuns []testutil.PDFTextRun
	for _, run := range inspection.TextRuns {
		if run.Page == sourceRuns[0].Page && run.Y >= minimumY-0.01 && run.Y <= maximumY+0.01 {
			rowRuns = append(rowRuns, run)
		}
	}
	return rowRuns
}

// contractPDFAuditRunTexts returns the decoded text fragments from one local
// detailed Annex row.
// Authored by: OpenCode
func contractPDFAuditRunTexts(runs []testutil.PDFTextRun) []string {
	var texts = make([]string, 0, len(runs))
	for _, run := range runs {
		texts = append(texts, run.Text)
	}
	return texts
}

// containsContractPDFToken reports whether a recovered PDF row contains one
// complete semantic token.
// Authored by: OpenCode
func containsContractPDFToken(tokens []string, want string) bool {
	return countContractPDFToken(tokens, want) > 0
}

// countContractPDFToken counts complete semantic tokens in one recovered row.
// Authored by: OpenCode
func countContractPDFToken(tokens []string, want string) int {
	var count int
	for _, token := range tokens {
		if token == want {
			count++
		}
	}
	return count
}

// contractDetailedAuditAnnex returns a deterministic detailed Annex 1 fixture.
// Authored by: OpenCode
func contractDetailedAuditAnnex() reportmodel.AuditAnnex {
	var annex, err = reportmodel.NewDetailedAuditAnnex([]reportmodel.PerAssetAuditSection{
		{
			AssetIdentityKey: "asset-btc",
			DisplayLabel:     "BTC",
			Entries: []reportmodel.AuditActivityEntry{contractAuditEntry(reportmodel.AuditActivityEntry{
				SourceID:                     "btc-sell-2024-001",
				OccurredAt:                   time.Date(2024, time.January, 1, 0, 15, 0, 0, contractMarkdownFixtureLocation),
				ActivityType:                 reportmodel.ActivityTypeSell,
				Quantity:                     mustContractDecimal("1"),
				UnitPrice:                    contractDecimalPointer("25000"),
				GrossValue:                   contractDecimalPointer("25000"),
				FeeAmount:                    contractDecimalPointer("0"),
				ActivityCurrency:             "USD",
				CalculationCurrency:          "EUR",
				QuantityAfterActivity:        mustContractDecimal("1"),
				BasisAfterActivity:           mustContractDecimal("22009"),
				FullLiquidationEvent:         true,
				IsZeroPricedHoldingReduction: false,
				AllocatedBasis:               contractDecimalPointer("22009"),
				NetLiquidationProceeds:       contractDecimalPointer("25000"),
				GainOrLoss:                   contractDecimalPointer("1240.5"),
				ConversionStatus:             reportmodel.ConversionStatusConverted,
				Note:                         "note token=secret-token",
			})},
		},
		{
			AssetIdentityKey: "asset-xrp",
			DisplayLabel:     "XRP",
			Entries: []reportmodel.AuditActivityEntry{contractAuditEntry(reportmodel.AuditActivityEntry{
				SourceID:                     "xrp-reduction-2024-001",
				OccurredAt:                   time.Date(2024, time.April, 1, 12, 0, 0, 0, contractMarkdownFixtureSummerLocation),
				ActivityType:                 reportmodel.ActivityTypeSell,
				Quantity:                     mustContractDecimal("200"),
				UnitPrice:                    contractDecimalPointer("0"),
				GrossValue:                   contractDecimalPointer("0"),
				FeeAmount:                    contractDecimalPointer("0"),
				ActivityCurrency:             "USD",
				CalculationCurrency:          "EUR",
				QuantityAfterActivity:        mustContractDecimal("800"),
				BasisAfterActivity:           mustContractDecimal("400"),
				IsZeroPricedHoldingReduction: true,
				ConversionStatus:             reportmodel.ConversionStatusSameCurrency,
				Note:                         "custody transfer",
			})},
		},
		{
			AssetIdentityKey: "asset-eth",
			DisplayLabel:     "ETH",
			Entries: []reportmodel.AuditActivityEntry{contractAuditEntry(reportmodel.AuditActivityEntry{
				SourceID:                     "eth-reference-buy",
				OccurredAt:                   time.Date(2023, time.January, 1, 10, 0, 0, 0, time.UTC),
				ActivityType:                 reportmodel.ActivityTypeBuy,
				Quantity:                     mustContractDecimal("1"),
				UnitPrice:                    contractDecimalPointer("50"),
				GrossValue:                   contractDecimalPointer("50"),
				FeeAmount:                    contractDecimalPointer("0"),
				ActivityCurrency:             "EUR",
				CalculationCurrency:          "EUR",
				QuantityAfterActivity:        mustContractDecimal("1"),
				BasisAfterActivity:           mustContractDecimal("50"),
				IsZeroPricedHoldingReduction: false,
				ConversionStatus:             reportmodel.ConversionStatusSameCurrency,
				Note:                         "reference-only acquisition",
			}), contractAuditEntry(reportmodel.AuditActivityEntry{
				SourceID:                     "tiny-positive-unclassified",
				OccurredAt:                   time.Date(2024, time.May, 1, 10, 0, 0, 0, time.UTC),
				ActivityType:                 reportmodel.ActivityTypeSell,
				Quantity:                     mustContractDecimal("1"),
				UnitPrice:                    contractDecimalPointer("0.004"),
				GrossValue:                   contractDecimalPointer("0.004"),
				FeeAmount:                    contractDecimalPointer("0"),
				ActivityCurrency:             "GBP",
				CalculationCurrency:          "USD",
				QuantityAfterActivity:        mustContractDecimal("1"),
				BasisAfterActivity:           mustContractDecimal("0.004"),
				IsZeroPricedHoldingReduction: false,
				ConversionStatus:             reportmodel.ConversionStatusConverted,
				Note:                         "tiny positive unclassified control",
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

// Package contract verifies the closed report-rendering acceptance contract.
// Authored by: OpenCode
package contract

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	reportmarkdown "github.com/benizzio/ghostfolio-cryptogains/internal/report/markdown"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	reportpdf "github.com/benizzio/ghostfolio-cryptogains/internal/report/pdf"
	"github.com/benizzio/ghostfolio-cryptogains/internal/report/presentation"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/cockroachdb/apd/v3"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

// TestReportRenderingClosedManifestContract verifies the immutable case families,
// exact financial field schema, paired US1 occurrence keys, and population
// numerator/denominator reporting required by the closed acceptance manifest.
// Authored by: OpenCode
func TestReportRenderingClosedManifestContract(t *testing.T) {
	t.Parallel()

	var manifest = testutil.DeterministicReportPresentationAcceptanceFixture()
	var expectedCaseKinds = map[testutil.ReportPresentationCaseKind]int{
		testutil.ReportPresentationCaseKindWarning:   1,
		testutil.ReportPresentationCaseKindFinancial: 124,
		testutil.ReportPresentationCaseKindQuantity:  5,
		testutil.ReportPresentationCaseKindRate:      5,
		testutil.ReportPresentationCaseKindBoolean:   2,
		testutil.ReportPresentationCaseKindCurrency:  3,
		testutil.ReportPresentationCaseKindConverted: 8,
	}
	if len(manifest.Cases) != 148 {
		t.Fatalf("closed acceptance case count = %d, want 148", len(manifest.Cases))
	}

	var caseIDs = make(map[string]bool, len(manifest.Cases))
	var caseKinds = make(map[testutil.ReportPresentationCaseKind]int)
	for _, acceptanceCase := range manifest.Cases {
		if caseIDs[acceptanceCase.ID] {
			t.Fatalf("duplicate closed acceptance case ID %q", acceptanceCase.ID)
		}
		caseIDs[acceptanceCase.ID] = true
		caseKinds[acceptanceCase.Kind]++
		assertReportPresentationAttempts(t, acceptanceCase)
		assertReportPresentationOccurrenceShape(t, acceptanceCase)
		if acceptanceCase.Kind == testutil.ReportPresentationCaseKindFinancial {
			assertReportPresentationFinancialFields(t, acceptanceCase)
		}
	}
	for kind, want := range expectedCaseKinds {
		if caseKinds[kind] != want {
			t.Fatalf("closed %s case count = %d, want %d", kind, caseKinds[kind], want)
		}
	}

	var expectedCounters = testutil.ReportPresentationAcceptanceCounters{
		CaseCount: 148,
		Populations: map[testutil.ReportPresentationPopulation]int{
			testutil.ReportPresentationPopulationWarning:            296,
			testutil.ReportPresentationPopulationVisibleFinancial:   688,
			testutil.ReportPresentationPopulationModelIntegrity:     296,
			testutil.ReportPresentationPopulationQuantity:           80,
			testutil.ReportPresentationPopulationBoolean:            16,
			testutil.ReportPresentationPopulationClassifiedCurrency: 2,
			testutil.ReportPresentationPopulationUnclassified:       4,
			testutil.ReportPresentationPopulationConversionRow:      16,
			testutil.ReportPresentationPopulationParity:             601,
			testutil.ReportPresentationPopulationConvertedEntry:     24,
		},
	}
	if manifest.Counters.CaseCount != expectedCounters.CaseCount || !reflect.DeepEqual(manifest.Counters.Populations, expectedCounters.Populations) {
		t.Fatalf("closed acceptance counters = %#v, want %#v", manifest.Counters, expectedCounters)
	}
	reportRenderingPopulationEvidence(t, manifest)
	assertReportPresentationPairedPopulation(t, manifest, testutil.ReportPresentationPopulationWarning)
	assertReportPresentationPairedPopulation(t, manifest, testutil.ReportPresentationPopulationVisibleFinancial)
	assertReportPresentationPairedPopulation(t, manifest, testutil.ReportPresentationPopulationQuantity)
	assertReportPresentationParityPopulation(t, manifest)
}

// TestReportRenderingUS1WarningValuesParityAndSearchability verifies the main
// Markdown warning, exact fields, and quantity boundaries for one report.
// Authored by: OpenCode
func TestReportRenderingUS1WarningValuesParityAndSearchability(t *testing.T) {
	t.Parallel()

	var report = contractMarkdownReportFixture(reportmodel.ReportBaseCurrencyEUR.Label())
	var documents, err = reportmarkdown.RenderDocuments(report)
	if err != nil {
		t.Fatalf("render Markdown report documents: %v", err)
	}
	if len(documents) != 2 {
		t.Fatalf("Markdown document count = %d, want main and Annex", len(documents))
	}
	var mainMarkdown = string(documents[0].Content)
	var annexMarkdown = string(documents[1].Content)
	var warning = testutil.ReportPresentationLegalWarningText
	var boldWarning = "**" + warning + "**"
	if countOccurrences(mainMarkdown, boldWarning) != 1 {
		t.Fatalf("Markdown warning occurrence count = %d, want 1", countOccurrences(mainMarkdown, boldWarning))
	}
	var currencyIndex = strings.Index(mainMarkdown, "- **Report Calculation Currency:** EUR")
	var warningIndex = strings.Index(mainMarkdown, boldWarning)
	var summaryIndex = strings.Index(mainMarkdown, "## Gains-And-Losses Summary")
	if currencyIndex < 0 || warningIndex <= currencyIndex || summaryIndex <= warningIndex {
		t.Fatalf("Markdown warning placement is invalid: currency=%d warning=%d summary=%d", currencyIndex, warningIndex, summaryIndex)
	}
	if !strings.Contains(mainMarkdown, "\n\n"+boldWarning+"\n\n") {
		t.Fatalf("Markdown warning is not one standalone paragraph")
	}
	assertNotContains(t, annexMarkdown, warning)

	for _, expected := range []string{
		"| BTC | 1240.50 | EUR |",
		"| Overall Yearly Net Total | 2240.50 | EUR |",
		"- **Cost Basis:** 44018.00",
		"| 2023-12-31 23:15:00 | btc-sell-2024-001 | SELL | 1 | 25000.00 | 25000.00 | 0.00 | 1 | 22009.00 | USD | EUR | Converted | note token=[REDACTED] |",
		"| 2023-12-31 23:15:00 | btc-sell-2024-001 | 1 | 22009.00 | 25000.00 | 1240.50 | EUR |",
	} {
		assertContains(t, mainMarkdown, expected)
	}
	for _, expected := range []string{"1", "2", "800", "200"} {
		if !strings.Contains(mainMarkdown, "| "+expected+" |") && !strings.Contains(mainMarkdown, "**Quantity:** "+expected) {
			t.Fatalf("Markdown quantity %q is not present at a semantic field boundary", expected)
		}
	}
}

// TestReportRenderingUS1PDFWarningTextRunsAndSearchability verifies the PDF
// warning font evidence, semantic placement, exact values, and searchable text.
// Authored by: OpenCode
func TestReportRenderingUS1PDFWarningTextRunsAndSearchability(t *testing.T) {
	t.Parallel()

	var report = contractMarkdownReportFixture(reportmodel.ReportBaseCurrencyEUR.Label())
	var renderer reportpdf.Renderer
	var err error
	renderer, err = reportpdf.NewRenderer(reportpdf.RenderOptions{Fonts: reportpdf.FontData{Regular: goregular.TTF, Bold: gobold.TTF}})
	if err != nil {
		t.Fatalf("create PDF renderer: %v", err)
	}
	var payload []byte
	payload, err = renderer.Render(report)
	if err != nil {
		t.Fatalf("render PDF report: %v", err)
	}
	var inspection testutil.GeneratedPDF
	inspection, err = testutil.InspectGeneratedPDF(payload)
	if err != nil {
		t.Fatalf("inspect generated PDF: %v", err)
	}
	assertLandscapeA4PDF(t, inspection)
	if len(inspection.TextRuns) == 0 {
		t.Fatal("generated PDF has no selectable text runs")
	}
	var warning = testutil.ReportPresentationLegalWarningText
	if !inspection.ContainsSearchableText(warning) {
		t.Fatalf("generated PDF warning is not searchable: %q", inspection.SearchableText)
	}
	reportRenderingAssertBoldWarningRuns(t, inspection, warning)
	var searchableText = normalizeReportRenderingText(inspection.SearchableText)
	var searchableBefore = strings.Index(searchableText, normalizeReportRenderingText("Report Calculation Currency"))
	var searchableWarning = strings.Index(searchableText, normalizeReportRenderingText(warning))
	var searchableSummary = strings.Index(searchableText, normalizeReportRenderingText("Gains-And-Losses Summary"))
	if searchableBefore < 0 || searchableWarning <= searchableBefore || searchableSummary <= searchableWarning {
		t.Fatalf("PDF warning placement is invalid: currency=%d warning=%d summary=%d", searchableBefore, searchableWarning, searchableSummary)
	}
	for _, expected := range []string{"1240.50", "2240.50", "44018.00", "25000.00", "22009.00", "1.08", "800", "200"} {
		if !inspection.ContainsSearchableText(expected) {
			t.Fatalf("generated PDF does not contain searchable value %q", expected)
		}
	}
}

// TestReportRenderingRejectsFR004aOutOfDomainInBothFormats verifies every
// FR-004a rejection class through the actual Markdown and PDF renderer objects.
// Authored by: OpenCode
func TestReportRenderingRejectsFR004aOutOfDomainInBothFormats(t *testing.T) {
	t.Parallel()

	var failureCases = []struct {
		name       string
		value      apd.Decimal
		formatOpts presentation.FinancialFormattingOptions
		want       string
	}{
		{name: "adjusted exponent below lower bound", value: reportRenderingDecimalWithExponent(-100001), want: "adjusted exponent"},
		{name: "adjusted exponent above upper bound", value: reportRenderingDecimalWithExponent(100001), want: "adjusted exponent"},
		{name: "upper bound carry", value: reportRenderingUpperBoundCarry(), want: "adjusted exponent"},
		{name: "required precision above apd limit", value: reportRenderingDecimal("1.23"), formatOpts: presentation.NewFinancialFormattingTestOptions(func(int64, int64) error {
			return fmt.Errorf("required precision %d exceeds apd operational limit", int64(2147383650))
		}), want: "required precision"},
	}

	for _, failureCase := range failureCases {
		var failureCase = failureCase
		t.Run(failureCase.name, func(t *testing.T) {
			var report = contractMarkdownReportFixture(reportmodel.ReportBaseCurrencyEUR.Label())
			report.YearlyNetTotal = failureCase.value
			var markdownRenderer = reportmarkdown.NewRenderer(reportmarkdown.RenderOptions{FinancialFormatting: failureCase.formatOpts})
			var err error
			var markdownDocument reportmodel.ReportDocument
			markdownDocument, err = markdownRenderer.Render(report)
			assertFR004aRendererFailure(t, "Markdown", markdownDocument.Content, err, failureCase.want)

			var pdfRenderer reportpdf.Renderer
			pdfRenderer, err = reportpdf.NewRenderer(reportpdf.RenderOptions{
				Fonts:               reportpdf.FontData{Regular: goregular.TTF, Bold: gobold.TTF},
				FinancialFormatting: failureCase.formatOpts,
			})
			if err != nil {
				t.Fatalf("create PDF renderer: %v", err)
			}
			var payload []byte
			payload, err = pdfRenderer.Render(report)
			assertFR004aRendererFailure(t, "PDF", payload, err, failureCase.want)
		})
	}
}

// assertFR004aRendererFailure verifies that a selected renderer returns no
// visible document after a contextual financial-formatting rejection.
// Authored by: OpenCode
func assertFR004aRendererFailure(t *testing.T, format string, payload []byte, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), want) {
		t.Fatalf("%s FR-004a rejection = %v, want context %q", format, err, want)
	}
	if len(payload) != 0 {
		t.Fatalf("%s returned visible output after FR-004a rejection: %d bytes", format, len(payload))
	}
}

// assertReportPresentationAttempts verifies that every closed case has the two
// required output attempts and the inherited Markdown/PDF document roles.
// Authored by: OpenCode
func assertReportPresentationAttempts(t *testing.T, acceptanceCase testutil.ReportPresentationAcceptanceCase) {
	t.Helper()
	if len(acceptanceCase.Attempts) != 2 {
		t.Fatalf("case %q attempts = %d, want Markdown and PDF", acceptanceCase.ID, len(acceptanceCase.Attempts))
	}
	var expected = []testutil.ReportPresentationFormatAttempt{
		{Format: testutil.ReportPresentationFormatMarkdown, DocumentRoles: []testutil.ReportPresentationDocumentRole{testutil.ReportPresentationDocumentRoleMain, testutil.ReportPresentationDocumentRoleAnnex}},
		{Format: testutil.ReportPresentationFormatPDF, DocumentRoles: []testutil.ReportPresentationDocumentRole{testutil.ReportPresentationDocumentRoleCombined}},
	}
	for index := range expected {
		if acceptanceCase.Attempts[index].Format != expected[index].Format || strings.Join(stringRoles(acceptanceCase.Attempts[index].DocumentRoles), ",") != strings.Join(stringRoles(expected[index].DocumentRoles), ",") {
			t.Fatalf("case %q attempt %d = %#v, want %#v", acceptanceCase.ID, index, acceptanceCase.Attempts[index], expected[index])
		}
	}
}

// assertReportPresentationOccurrenceShape verifies warning/model/parity keys
// exist for every case and that all semantic keys carry their closed identity.
// Authored by: OpenCode
func assertReportPresentationOccurrenceShape(t *testing.T, acceptanceCase testutil.ReportPresentationAcceptanceCase) {
	t.Helper()
	var warningKeys int
	var modelKeys int
	var parityKeys int
	for _, occurrence := range acceptanceCase.OccurrenceKeys {
		if occurrence.CaseID != acceptanceCase.ID || occurrence.SourceOrRowIdentity == "" || occurrence.Section == "" {
			t.Fatalf("case %q has incomplete occurrence key %#v", acceptanceCase.ID, occurrence)
		}
		switch occurrence.Population {
		case testutil.ReportPresentationPopulationWarning:
			warningKeys++
		case testutil.ReportPresentationPopulationModelIntegrity:
			modelKeys++
		case testutil.ReportPresentationPopulationParity:
			parityKeys++
		}
	}
	if warningKeys != 2 || modelKeys != 2 || parityKeys == 0 {
		t.Fatalf("case %q occurrence shape = warning %d model %d parity %d", acceptanceCase.ID, warningKeys, modelKeys, parityKeys)
	}
}

// assertReportPresentationFinancialFields verifies every closed matrix row uses
// the exact field names, amount kinds, and ordinals from the report contract.
// Authored by: OpenCode
func assertReportPresentationFinancialFields(t *testing.T, acceptanceCase testutil.ReportPresentationAcceptanceCase) {
	t.Helper()
	var expected = map[string][]string{
		"summary-net-gain-or-loss":              {"per_asset_net_gain_or_loss:gain_or_loss", "overall_yearly_net_total:gain_or_loss"},
		"position-cost-basis":                   {"opening_cost_basis:cost_basis", "closing_cost_basis:cost_basis", "historical_cost_basis:cost_basis"},
		"in-year-activity":                      {"unit_price:unit_price", "gross_value:gross_value", "fee_amount:fee_amount", "basis_after_row:cost_basis"},
		"liquidation-allocated-basis":           {"allocated_basis:cost_basis"},
		"liquidation-net-proceeds-gain-or-loss": {"net_proceeds:proceeds", "gain_or_loss:gain_or_loss"},
		"audit-activity":                        {"unit_price:unit_price", "gross_value:gross_value", "fee_amount:fee_amount", "basis_after_activity:cost_basis"},
		"audit-allocated-basis":                 {"allocated_basis:cost_basis"},
		"audit-net-proceeds-gain-or-loss":       {"net_proceeds:proceeds", "gain_or_loss:gain_or_loss"},
		"conversion-amount":                     {"original_unit_price:unit_price", "converted_unit_price:unit_price", "original_gross_value:gross_value", "converted_gross_value:gross_value", "original_fee_amount:fee_amount", "converted_fee_amount:fee_amount"},
	}
	var got []string
	for _, field := range acceptanceCase.FinancialFields {
		got = append(got, fmt.Sprintf("%s:%s", field.Name, field.AmountKind))
	}
	if strings.Join(got, ",") != strings.Join(expected[acceptanceCase.FinancialFieldClass], ",") {
		t.Fatalf("case %q financial fields = %v, want %v", acceptanceCase.ID, got, expected[acceptanceCase.FinancialFieldClass])
	}
	for ordinal, field := range acceptanceCase.FinancialFields {
		if field.AmountOrdinal != ordinal {
			t.Fatalf("case %q field %q ordinal = %d, want %d", acceptanceCase.ID, field.Name, field.AmountOrdinal, ordinal)
		}
	}
}

// reportRenderingPopulationEvidence logs and verifies every manifest-derived
// numerator and denominator without dropping failed future format attempts.
// Authored by: OpenCode
func reportRenderingPopulationEvidence(t *testing.T, manifest testutil.ReportPresentationAcceptanceManifest) {
	t.Helper()
	var populations = []testutil.ReportPresentationPopulation{
		testutil.ReportPresentationPopulationWarning,
		testutil.ReportPresentationPopulationVisibleFinancial,
		testutil.ReportPresentationPopulationModelIntegrity,
		testutil.ReportPresentationPopulationQuantity,
		testutil.ReportPresentationPopulationBoolean,
		testutil.ReportPresentationPopulationClassifiedCurrency,
		testutil.ReportPresentationPopulationUnclassified,
		testutil.ReportPresentationPopulationConversionRow,
		testutil.ReportPresentationPopulationParity,
		testutil.ReportPresentationPopulationConvertedEntry,
	}
	for _, population := range populations {
		var numerator = reportRenderingPopulationCount(manifest.Cases, population)
		var denominator = reportRenderingPopulationCounter(t, manifest.Counters, population)
		t.Logf("population %s numerator/denominator: %d/%d", population, numerator, denominator)
		if denominator == 0 || numerator != denominator {
			t.Fatalf("population %s numerator/denominator = %d/%d", population, numerator, denominator)
		}
	}
	t.Logf("population A numerator/denominator: %d/%d", len(manifest.Cases), manifest.Counters.CaseCount)
}

// reportRenderingPopulationCount counts occurrence keys for one closed
// acceptance population.
// Authored by: OpenCode
func reportRenderingPopulationCount(cases []testutil.ReportPresentationAcceptanceCase, population testutil.ReportPresentationPopulation) int {
	var count int
	for _, acceptanceCase := range cases {
		for _, occurrence := range acceptanceCase.OccurrenceKeys {
			if occurrence.Population == population {
				count++
			}
		}
	}
	return count
}

// reportRenderingPopulationCounter selects one manifest denominator.
// Authored by: OpenCode
func reportRenderingPopulationCounter(t *testing.T, counters testutil.ReportPresentationAcceptanceCounters, population testutil.ReportPresentationPopulation) int {
	t.Helper()
	var count, ok = counters.Populations[population]
	if !ok {
		t.Fatalf("population %s is missing from acceptance counters", population)
	}
	return count
}

// assertReportPresentationPairedPopulation verifies Markdown/PDF parity keys
// exist as a pair for each semantic occurrence in W, V, and Q.
// Authored by: OpenCode
func assertReportPresentationPairedPopulation(t *testing.T, manifest testutil.ReportPresentationAcceptanceManifest, population testutil.ReportPresentationPopulation) {
	t.Helper()
	var pairs = make(map[string]map[testutil.ReportPresentationFormat]bool)
	for _, acceptanceCase := range manifest.Cases {
		for _, occurrence := range acceptanceCase.OccurrenceKeys {
			if occurrence.Population != population {
				continue
			}
			var role = occurrence.DocumentRole
			if occurrence.Format == testutil.ReportPresentationFormatPDF && role == testutil.ReportPresentationDocumentRoleCombined {
				if occurrence.Section == "detailed_per_asset_audit" || occurrence.Section == "currency_conversion_audit" {
					role = testutil.ReportPresentationDocumentRoleAnnex
				} else {
					role = testutil.ReportPresentationDocumentRoleMain
				}
			}
			var identity = fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%d", occurrence.CaseID, role, occurrence.Section, occurrence.AssetIdentity, occurrence.SourceOrRowIdentity, occurrence.FieldName, occurrence.AmountKind, occurrence.AmountOrdinal)
			if pairs[identity] == nil {
				pairs[identity] = make(map[testutil.ReportPresentationFormat]bool)
			}
			pairs[identity][occurrence.Format] = true
		}
	}
	for identity, formats := range pairs {
		if !formats[testutil.ReportPresentationFormatMarkdown] || !formats[testutil.ReportPresentationFormatPDF] {
			t.Fatalf("population %s identity %q is not paired: %#v", population, identity, formats)
		}
	}
}

// assertReportPresentationParityPopulation verifies P consists of explicit
// cross-format semantic parity identities rather than substring-count evidence.
// Authored by: OpenCode
func assertReportPresentationParityPopulation(t *testing.T, manifest testutil.ReportPresentationAcceptanceManifest) {
	t.Helper()
	var identities = make(map[string]bool)
	for _, acceptanceCase := range manifest.Cases {
		for _, occurrence := range acceptanceCase.OccurrenceKeys {
			if occurrence.Population != testutil.ReportPresentationPopulationParity {
				continue
			}
			if occurrence.Format != "cross-format" || occurrence.CaseID != acceptanceCase.ID {
				t.Fatalf("invalid cross-format parity occurrence: %#v", occurrence)
			}
			var identity = fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%d", occurrence.CaseID, occurrence.DocumentRole, occurrence.Section, occurrence.AssetIdentity, occurrence.SourceOrRowIdentity, occurrence.FieldName, occurrence.AmountKind, occurrence.AmountOrdinal)
			if identities[identity] {
				t.Fatalf("duplicate parity identity %q", identity)
			}
			identities[identity] = true
		}
	}
}

// reportRenderingAssertBoldWarningRuns proves the ordered warning fragments all
// use the same embedded font resource as the known bold report title.
// Authored by: OpenCode
func reportRenderingAssertBoldWarningRuns(t *testing.T, inspection testutil.GeneratedPDF, warning string) {
	t.Helper()
	var boldResource string
	for _, run := range inspection.TextRuns {
		if strings.Contains(normalizeReportRenderingText(run.Text), normalizeReportRenderingText("Ghostfolio Capital Gains And Losses Report")) {
			boldResource = run.FontResource
			break
		}
	}
	if boldResource == "" {
		t.Fatal("could not identify the embedded bold PDF font resource from the report title")
	}
	var target = normalizeReportRenderingText(warning)
	for start := range inspection.TextRuns {
		var joined string
		var fragments []testutil.PDFTextRun
		for end := start; end < len(inspection.TextRuns); end++ {
			fragments = append(fragments, inspection.TextRuns[end])
			joined = normalizeReportRenderingText(joined + " " + inspection.TextRuns[end].Text)
			if joined == target {
				for _, fragment := range fragments {
					if fragment.FontResource != boldResource {
						t.Fatalf("warning fragment %q uses font %q, want bold font %q", fragment.Text, fragment.FontResource, boldResource)
					}
				}
				if !strings.HasSuffix(strings.TrimSpace(joined), ".") {
					t.Fatal("bold warning text-run evidence does not include the final period")
				}
				return
			}
			if len(joined) > len(target) || !strings.HasPrefix(target, joined) {
				break
			}
		}
	}
	t.Fatalf("could not find complete warning text-run sequence in generated PDF")
}

// normalizeReportRenderingText makes PDF text-run and searchable-text ordering
// comparable without changing punctuation or semantic values.
// Authored by: OpenCode
func normalizeReportRenderingText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

// stringRoles converts document-role values to a stable comparison slice.
// Authored by: OpenCode
func stringRoles(roles []testutil.ReportPresentationDocumentRole) []string {
	var result = make([]string, 0, len(roles))
	for _, role := range roles {
		result = append(result, string(role))
	}
	return result
}

// reportRenderingDecimalWithExponent constructs a finite synthetic decimal for
// the immediate FR-004a adjusted-exponent rejection boundary.
// Authored by: OpenCode
func reportRenderingDecimalWithExponent(exponent int32) apd.Decimal {
	var value apd.Decimal
	value.Form = apd.Finite
	value.Coeff.SetInt64(1)
	value.Exponent = exponent
	return value
}

// reportRenderingDecimal parses one small synthetic value for renderer tests.
// Authored by: OpenCode
func reportRenderingDecimal(raw string) apd.Decimal {
	var value apd.Decimal
	if _, _, err := value.SetString(raw); err != nil {
		panic(err)
	}
	return value
}

// reportRenderingUpperBoundCarry builds the accepted adjusted-exponent upper
// boundary whose HALF UP result would carry into adjusted exponent 100001.
// Authored by: OpenCode
func reportRenderingUpperBoundCarry() apd.Decimal {
	return reportRenderingDecimal(strings.Repeat("9", 100001) + ".995")
}

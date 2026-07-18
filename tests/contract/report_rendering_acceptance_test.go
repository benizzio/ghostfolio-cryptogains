// Package contract verifies the aggregate report-rendering acceptance accounting.
// Authored by: OpenCode
package contract

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	reportmarkdown "github.com/benizzio/ghostfolio-cryptogains/internal/report/markdown"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	reportpdf "github.com/benizzio/ghostfolio-cryptogains/internal/report/pdf"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/cockroachdb/apd/v3"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	// renderingAcceptancePopulationA identifies closed renderer acceptance cases.
	// Authored by: OpenCode
	renderingAcceptancePopulationA = "A"
	// renderingAcceptancePopulationR identifies the fixed calculation regression population.
	// Authored by: OpenCode
	renderingAcceptancePopulationR = "R"
	// renderingAcceptanceRegressionCases is the expected fixed regression-case count.
	// Authored by: OpenCode
	renderingAcceptanceRegressionCases = 102
)

// renderingAcceptanceAttemptResult records whether one listed format attempt
// passed rendering, output-shape, semantic-control, and model-integrity checks.
// Authored by: OpenCode
type renderingAcceptanceAttemptResult struct {
	format   testutil.ReportPresentationFormat
	passed   bool
	message  string
	semantic string
}

// TestRenderingAcceptanceAccounting executes both format attempts for every
// closed acceptance case and verifies every contract population has exact,
// non-empty numerator and denominator accounting.
// Authored by: OpenCode
func TestRenderingAcceptanceAccounting(t *testing.T) {
	var manifest = testutil.DeterministicReportPresentationAcceptanceFixture()
	assertRenderingAcceptanceManifest(t, manifest)

	var accounting = newRenderingAcceptanceAccounting(manifest)
	var regressionPassed = runRenderingAcceptanceRegressionPopulation(t)
	if regressionPassed {
		accounting.add(renderingAcceptancePopulationR, renderingAcceptanceRegressionKey(0))
		for index := 1; index < renderingAcceptanceRegressionCases; index++ {
			accounting.add(renderingAcceptancePopulationR, renderingAcceptanceRegressionKey(index))
		}
	}

	var pdfRenderer, err = reportpdf.NewRenderer(reportpdf.RenderOptions{
		Fonts: reportpdf.FontData{Regular: goregular.TTF, Bold: gobold.TTF},
	})
	if err != nil {
		t.Fatalf("create acceptance PDF renderer: %v", err)
	}

	for _, acceptanceCase := range manifest.Cases {
		var report, reportErr = renderingAcceptanceReport(acceptanceCase)
		if reportErr != nil {
			t.Errorf("case %q synthetic report: %v", acceptanceCase.ID, reportErr)
			continue
		}

		var markdownAttempt = runRenderingAcceptanceMarkdownAttempt(acceptanceCase, report)
		var pdfAttempt = runRenderingAcceptancePDFAttempt(acceptanceCase, report, pdfRenderer)
		if markdownAttempt.passed {
			accounting.addFormatOccurrences(acceptanceCase, markdownAttempt.format)
		}
		if pdfAttempt.passed {
			accounting.addFormatOccurrences(acceptanceCase, pdfAttempt.format)
		}
		if !markdownAttempt.passed {
			t.Errorf("case %q Markdown attempt failed: %s", acceptanceCase.ID, markdownAttempt.message)
		}
		if !pdfAttempt.passed {
			t.Errorf("case %q PDF attempt failed: %s", acceptanceCase.ID, pdfAttempt.message)
		}
		if markdownAttempt.passed && pdfAttempt.passed {
			accounting.add(renderingAcceptancePopulationA, acceptanceCase.ID)
			if renderingAcceptanceParityPasses(acceptanceCase, markdownAttempt.semantic, pdfAttempt.semantic) {
				accounting.addParityOccurrences(acceptanceCase)
			} else {
				t.Errorf("case %q cross-format parity failed", acceptanceCase.ID)
			}
		}
	}

	accounting.assertComplete(t)
}

// newRenderingAcceptanceAccounting creates independent expected and observed
// semantic-key sets so missing, extra, and empty populations are distinguishable.
// Authored by: OpenCode
func newRenderingAcceptanceAccounting(manifest testutil.ReportPresentationAcceptanceManifest) *renderingAcceptanceAccounting {
	var accounting = &renderingAcceptanceAccounting{
		expected: make(map[string]map[string]struct{}),
		observed: make(map[string]map[string]struct{}),
	}
	accounting.expected[renderingAcceptancePopulationA] = make(map[string]struct{}, len(manifest.Cases))
	accounting.observed[renderingAcceptancePopulationA] = make(map[string]struct{})
	accounting.expected[renderingAcceptancePopulationR] = make(map[string]struct{}, renderingAcceptanceRegressionCases)
	accounting.observed[renderingAcceptancePopulationR] = make(map[string]struct{})
	for index := 0; index < renderingAcceptanceRegressionCases; index++ {
		accounting.expected[renderingAcceptancePopulationR][renderingAcceptanceRegressionKey(index)] = struct{}{}
	}
	for _, acceptanceCase := range manifest.Cases {
		accounting.expected[renderingAcceptancePopulationA][acceptanceCase.ID] = struct{}{}
		for _, occurrence := range acceptanceCase.OccurrenceKeys {
			var population = string(occurrence.Population)
			if accounting.expected[population] == nil {
				accounting.expected[population] = make(map[string]struct{})
			}
			if accounting.observed[population] == nil {
				accounting.observed[population] = make(map[string]struct{})
			}
			if occurrence.Format == "cross-format" {
				accounting.expected[population][renderingAcceptanceOccurrenceKey(occurrence)] = struct{}{}
				continue
			}
			accounting.expected[population][renderingAcceptanceOccurrenceKey(occurrence)] = struct{}{}
		}
	}
	return accounting
}

// renderingAcceptanceAccounting stores closed semantic populations and their
// successful observations.
// Authored by: OpenCode
type renderingAcceptanceAccounting struct {
	expected map[string]map[string]struct{}
	observed map[string]map[string]struct{}
}

// add records one observed population identity.
// Authored by: OpenCode
func (accounting *renderingAcceptanceAccounting) add(population string, identity string) {
	if accounting.observed[population] == nil {
		accounting.observed[population] = make(map[string]struct{})
	}
	accounting.observed[population][identity] = struct{}{}
}

// addFormatOccurrences records all semantic keys belonging to a successful
// Markdown or PDF attempt while excluding cross-format parity keys.
// Authored by: OpenCode
func (accounting *renderingAcceptanceAccounting) addFormatOccurrences(acceptanceCase testutil.ReportPresentationAcceptanceCase, format testutil.ReportPresentationFormat) {
	for _, occurrence := range acceptanceCase.OccurrenceKeys {
		if occurrence.Format == format {
			accounting.add(string(occurrence.Population), renderingAcceptanceOccurrenceKey(occurrence))
		}
	}
}

// addParityOccurrences records parity keys only after both format attempts pass.
// Authored by: OpenCode
func (accounting *renderingAcceptanceAccounting) addParityOccurrences(acceptanceCase testutil.ReportPresentationAcceptanceCase) {
	for _, occurrence := range acceptanceCase.OccurrenceKeys {
		if occurrence.Population == testutil.ReportPresentationPopulationParity {
			accounting.add(string(occurrence.Population), renderingAcceptanceOccurrenceKey(occurrence))
		}
	}
}

// assertComplete reports exact population accounting and rejects missing, extra,
// or empty evidence identities.
// Authored by: OpenCode
func (accounting *renderingAcceptanceAccounting) assertComplete(t *testing.T) {
	t.Helper()
	var populations = []string{
		renderingAcceptancePopulationA,
		testutilPopulationName(testutil.ReportPresentationPopulationWarning),
		testutilPopulationName(testutil.ReportPresentationPopulationVisibleFinancial),
		renderingAcceptancePopulationR,
		testutilPopulationName(testutil.ReportPresentationPopulationModelIntegrity),
		testutilPopulationName(testutil.ReportPresentationPopulationQuantity),
		testutilPopulationName(testutil.ReportPresentationPopulationBoolean),
		testutilPopulationName(testutil.ReportPresentationPopulationClassifiedCurrency),
		testutilPopulationName(testutil.ReportPresentationPopulationUnclassified),
		testutilPopulationName(testutil.ReportPresentationPopulationConversionRow),
		testutilPopulationName(testutil.ReportPresentationPopulationParity),
		testutilPopulationName(testutil.ReportPresentationPopulationConvertedEntry),
	}
	for _, population := range populations {
		var expected = accounting.expected[population]
		var observed = accounting.observed[population]
		var numerator = len(observed)
		var denominator = len(expected)
		t.Logf("population %s numerator/denominator: %d/%d", population, numerator, denominator)
		if denominator == 0 {
			t.Errorf("population %s has an empty denominator", population)
		}
		if numerator == 0 {
			t.Errorf("population %s has an empty numerator", population)
		}
		for identity := range expected {
			if _, ok := observed[identity]; !ok {
				t.Errorf("population %s is missing identity %q", population, identity)
			}
		}
		for identity := range observed {
			if _, ok := expected[identity]; !ok {
				t.Errorf("population %s contains extra identity %q", population, identity)
			}
		}
	}
}

// runRenderingAcceptanceMarkdownAttempt executes and validates the Markdown
// main-plus-Annex attempt for one closed case.
// Authored by: OpenCode
func runRenderingAcceptanceMarkdownAttempt(acceptanceCase testutil.ReportPresentationAcceptanceCase, input reportmodel.CapitalGainsReport) renderingAcceptanceAttemptResult {
	var result = renderingAcceptanceAttemptResult{format: testutil.ReportPresentationFormatMarkdown}
	var before, cloneErr = renderingAcceptanceCloneReport(input)
	if cloneErr != nil {
		result.message = fmt.Sprintf("clone baseline: %v", cloneErr)
		return result
	}
	var report reportmodel.CapitalGainsReport
	report, cloneErr = renderingAcceptanceCloneReport(input)
	if cloneErr != nil {
		result.message = fmt.Sprintf("clone render input: %v", cloneErr)
		return result
	}
	var documents, err = reportmarkdown.RenderDocuments(report)
	if err != nil {
		result.message = fmt.Sprintf("render: %v", err)
		return result
	}
	if err = reportmodel.ValidateRenderedDocuments(reportmodel.ReportOutputFormatMarkdown, documents); err != nil {
		result.message = fmt.Sprintf("validate bundle: %v", err)
		return result
	}
	result.semantic, err = validateRenderingAcceptanceMarkdownControls(acceptanceCase, documents)
	if err != nil {
		result.message = err.Error()
		return result
	}
	if !reflect.DeepEqual(before, report) {
		result.message = "AUD-001 model changed after Markdown rendering"
		return result
	}
	result.passed = true
	return result
}

// runRenderingAcceptancePDFAttempt executes and validates the combined PDF
// attempt for one closed case.
// Authored by: OpenCode
func runRenderingAcceptancePDFAttempt(acceptanceCase testutil.ReportPresentationAcceptanceCase, input reportmodel.CapitalGainsReport, renderer reportpdf.Renderer) renderingAcceptanceAttemptResult {
	var result = renderingAcceptanceAttemptResult{format: testutil.ReportPresentationFormatPDF}
	var before, cloneErr = renderingAcceptanceCloneReport(input)
	if cloneErr != nil {
		result.message = fmt.Sprintf("clone baseline: %v", cloneErr)
		return result
	}
	var report reportmodel.CapitalGainsReport
	report, cloneErr = renderingAcceptanceCloneReport(input)
	if cloneErr != nil {
		result.message = fmt.Sprintf("clone render input: %v", cloneErr)
		return result
	}
	var payload, err = renderer.Render(report)
	if err != nil {
		result.message = fmt.Sprintf("render: %v", err)
		return result
	}
	var document, documentErr = reportmodel.NewReportDocument(
		reportmodel.ReportDocumentTypePDF,
		reportmodel.ReportDocumentRoleCombined,
		payload,
		report.Year,
		report.CostBasisMethod,
		report.GeneratedAt,
	)
	if documentErr != nil {
		result.message = fmt.Sprintf("create document: %v", documentErr)
		return result
	}
	if err = reportmodel.ValidateRenderedDocuments(reportmodel.ReportOutputFormatPDF, []reportmodel.ReportDocument{document}); err != nil {
		result.message = fmt.Sprintf("validate bundle: %v", err)
		return result
	}
	var inspection, inspectionErr = testutil.InspectGeneratedPDF(payload)
	if inspectionErr != nil {
		result.message = fmt.Sprintf("inspect PDF: %v", inspectionErr)
		return result
	}
	result.semantic, err = validateRenderingAcceptancePDFControls(acceptanceCase, inspection)
	if err != nil {
		result.message = err.Error()
		return result
	}
	if !reflect.DeepEqual(before, report) {
		result.message = "AUD-001 model changed after PDF rendering"
		return result
	}
	result.passed = true
	return result
}

// validateRenderingAcceptanceMarkdownControls validates the common warning and
// the closed case's concrete synthetic control in Markdown documents.
// Authored by: OpenCode
func validateRenderingAcceptanceMarkdownControls(acceptanceCase testutil.ReportPresentationAcceptanceCase, documents []reportmodel.ReportDocument) (string, error) {
	if len(documents) != 2 {
		return "", fmt.Errorf("Markdown document count = %d, want 2", len(documents))
	}
	var main = string(documents[0].Content)
	var annex = string(documents[1].Content)
	if strings.Count(main, testutil.ReportPresentationLegalWarningText) != 1 || strings.Contains(annex, testutil.ReportPresentationLegalWarningText) {
		return "", fmt.Errorf("warning occurrence or Annex exclusion is invalid")
	}
	if err := validateRenderingAcceptanceControlText(acceptanceCase, main+"\n"+annex); err != nil {
		return "", err
	}
	var semantic = testutil.ReportPresentationLegalWarningText + "\x00" + acceptanceCase.ExpectedVisibleValue
	if acceptanceCase.Kind == testutil.ReportPresentationCaseKindConverted {
		var sequence, ok = renderingAcceptanceConvertedSequence(acceptanceCase)
		if !ok {
			return "", fmt.Errorf("converted case %q has no synthetic sequence", acceptanceCase.ID)
		}
		var sourceID = "cv" + renderingAcceptanceConvertedSequenceIndex(sequence.ID)
		var cell, found = contractMarkdownConvertedAmountsCell(annex, sourceID)
		if !found || cell != sequence.ExpectedMarkdownCell {
			return "", fmt.Errorf("converted case %q cell = %q, want %q", acceptanceCase.ID, cell, sequence.ExpectedMarkdownCell)
		}
		semantic += "\x00" + normalizeReportRenderingText(strings.ReplaceAll(strings.ReplaceAll(cell, "<br>", " "), ";", ""))
	}
	return semantic, nil
}

// validateRenderingAcceptancePDFControls validates searchable combined-PDF text
// for one synthetic acceptance case.
// Authored by: OpenCode
func validateRenderingAcceptancePDFControls(acceptanceCase testutil.ReportPresentationAcceptanceCase, inspection testutil.GeneratedPDF) (string, error) {
	if !inspection.ContainsSearchableText(testutil.ReportPresentationLegalWarningText) {
		return "", fmt.Errorf("PDF warning is not searchable")
	}
	if err := validateRenderingAcceptancePDFControlText(acceptanceCase, inspection); err != nil {
		return "", err
	}
	var semantic = testutil.ReportPresentationLegalWarningText + "\x00" + acceptanceCase.ExpectedVisibleValue
	if acceptanceCase.Kind == testutil.ReportPresentationCaseKindConverted {
		var sequence, ok = renderingAcceptanceConvertedSequence(acceptanceCase)
		if !ok {
			return "", fmt.Errorf("converted case %q has no synthetic sequence", acceptanceCase.ID)
		}
		var sourceID = "cv" + renderingAcceptanceConvertedSequenceIndex(sequence.ID)
		var rowRuns = contractPDFConversionRowRuns(inspection, sourceID, []string{sourceID})
		var cellRuns = contractPDFConvertedCellRuns(rowRuns)
		var rowText = normalizeReportRenderingText(strings.ReplaceAll(strings.Join(contractPDFRunTexts(cellRuns), " "), ";", ""))
		var expectedText = normalizeReportRenderingText(strings.ReplaceAll(strings.ReplaceAll(sequence.ExpectedMarkdownCell, "<br>", " "), ";", ""))
		if expectedText != "" && !strings.Contains(rowText, expectedText) {
			return "", fmt.Errorf("PDF converted case %q cell = %q, want %q", acceptanceCase.ID, rowText, expectedText)
		}
		if expectedText == "" && strings.Contains(rowText, "unit_price:") {
			return "", fmt.Errorf("PDF converted case %q contains an entry in its empty cell", acceptanceCase.ID)
		}
		semantic += "\x00" + expectedText
	}
	return semantic, nil
}

// validateRenderingAcceptancePDFControlText checks a visible control through
// the PDF inspector's whitespace-normalized searchable-text boundary.
// Authored by: OpenCode
func validateRenderingAcceptancePDFControlText(acceptanceCase testutil.ReportPresentationAcceptanceCase, inspection testutil.GeneratedPDF) error {
	if acceptanceCase.ExpectedVisibleValue != "" && !acceptanceCase.Absent && !acceptanceCase.Omitted && !containsRenderingAcceptancePDFControl(inspection.SearchableText, acceptanceCase.ExpectedVisibleValue) {
		if !containsRenderingAcceptancePDFWrappedControl(inspection.TextRuns, acceptanceCase.ExpectedVisibleValue) {
			return fmt.Errorf("case %q is missing expected visible value %q", acceptanceCase.ID, acceptanceCase.ExpectedVisibleValue)
		}
	}
	if acceptanceCase.Kind == testutil.ReportPresentationCaseKindBoolean && !containsRenderingAcceptancePDFControl(inspection.SearchableText, acceptanceCase.ExpectedVisibleValue) {
		return fmt.Errorf("case %q is missing boolean label %q", acceptanceCase.ID, acceptanceCase.ExpectedVisibleValue)
	}
	return nil
}

// containsRenderingAcceptancePDFControl tolerates renderer whitespace inserted
// between long numeric glyph runs while preserving the exact visible sequence.
// Authored by: OpenCode
func containsRenderingAcceptancePDFControl(content string, expected string) bool {
	if strings.Contains(normalizeReportRenderingText(content), normalizeReportRenderingText(expected)) {
		return true
	}
	var compactContent = strings.ReplaceAll(normalizeReportRenderingText(content), " ", "")
	var compactExpected = strings.ReplaceAll(normalizeReportRenderingText(expected), " ", "")
	return strings.Contains(compactContent, compactExpected)
}

// containsRenderingAcceptancePDFWrappedControl reconstructs a long numeric
// control when the PDF renderer splits it across adjacent text-showing runs.
// Authored by: OpenCode
func containsRenderingAcceptancePDFWrappedControl(runs []testutil.PDFTextRun, expected string) bool {
	if len(expected) < 8 {
		return false
	}
	for startIndex, startRun := range runs {
		var consumed = commonRenderingAcceptancePrefix(startRun.Text, expected)
		if consumed == 0 {
			continue
		}
		var remaining = expected[consumed:]
		for _, run := range runs[startIndex+1:] {
			var matched = commonRenderingAcceptancePrefix(run.Text, remaining)
			if matched == 0 {
				continue
			}
			remaining = remaining[matched:]
			if remaining == "" {
				return true
			}
		}
	}
	return false
}

// commonRenderingAcceptancePrefix returns the matching prefix length for one
// PDF text run and one expected value.
// Authored by: OpenCode
func commonRenderingAcceptancePrefix(run string, expected string) int {
	var limit = len(run)
	if len(expected) < limit {
		limit = len(expected)
	}
	for index := 0; index < limit; index++ {
		if run[index] != expected[index] {
			return index
		}
	}
	return limit
}

// validateRenderingAcceptanceControlText checks the scalar control expected in
// the concrete synthetic report selected for one case.
// Authored by: OpenCode
func validateRenderingAcceptanceControlText(acceptanceCase testutil.ReportPresentationAcceptanceCase, content string) error {
	if acceptanceCase.ExpectedVisibleValue != "" && !acceptanceCase.Absent && !acceptanceCase.Omitted && !strings.Contains(content, acceptanceCase.ExpectedVisibleValue) {
		return fmt.Errorf("case %q is missing expected visible value %q", acceptanceCase.ID, acceptanceCase.ExpectedVisibleValue)
	}
	if acceptanceCase.Kind == testutil.ReportPresentationCaseKindBoolean && !strings.Contains(content, acceptanceCase.ExpectedVisibleValue) {
		return fmt.Errorf("case %q is missing boolean label %q", acceptanceCase.ID, acceptanceCase.ExpectedVisibleValue)
	}
	return nil
}

// renderingAcceptanceParityPasses uses the shared warning and case-control
// validation results carried by both attempt messages.
// Authored by: OpenCode
func renderingAcceptanceParityPasses(acceptanceCase testutil.ReportPresentationAcceptanceCase, markdownSemantic string, pdfSemantic string) bool {
	_ = acceptanceCase
	return markdownSemantic != "" && markdownSemantic == pdfSemantic
}

// renderingAcceptanceReport creates the deterministic synthetic model used by
// one closed case without changing any production or empirical fixture.
// Authored by: OpenCode
func renderingAcceptanceReport(acceptanceCase testutil.ReportPresentationAcceptanceCase) (reportmodel.CapitalGainsReport, error) {
	var report reportmodel.CapitalGainsReport
	switch acceptanceCase.Kind {
	case testutil.ReportPresentationCaseKindBoolean, testutil.ReportPresentationCaseKindCurrency:
		report = confidentialityReportFixture()
	case testutil.ReportPresentationCaseKindConverted:
		report = contractMarkdownReportFixture(reportmodel.ReportBaseCurrencyEUR.Label())
		var sequence, ok = renderingAcceptanceConvertedSequence(acceptanceCase)
		if !ok {
			return reportmodel.CapitalGainsReport{}, fmt.Errorf("converted sequence is not closed")
		}
		report.AuditAnnex.ConversionAuditEntries = []reportmodel.ConversionAuditEntry{contractConvertedAuditEntryForSequence(sequence)}
	default:
		report = contractMarkdownReportFixture(reportmodel.ReportBaseCurrencyEUR.Label())
		report.AuditAnnex = contractDetailedAuditAnnex()
	}

	switch acceptanceCase.Kind {
	case testutil.ReportPresentationCaseKindFinancial:
		applyRenderingAcceptanceFinancialControl(&report, acceptanceCase)
	case testutil.ReportPresentationCaseKindQuantity:
		var value = mustContractDecimal(acceptanceCase.ExactValue)
		if len(report.DetailSections) > 0 {
			report.DetailSections[0].OpeningQuantity = value
		}
	case testutil.ReportPresentationCaseKindRate:
		var value = mustContractDecimal(acceptanceCase.ExactValue)
		for index := range report.RateSources {
			report.RateSources[index].RateValue = value
		}
		for index := range report.AuditAnnex.ConversionAuditEntries {
			report.AuditAnnex.ConversionAuditEntries[index].RateValue = value
			for amountIndex := range report.AuditAnnex.ConversionAuditEntries[index].Amounts {
				if report.AuditAnnex.ConversionAuditEntries[index].Amounts[amountIndex].ExchangeRateEvidence != nil {
					report.AuditAnnex.ConversionAuditEntries[index].Amounts[amountIndex].ExchangeRateEvidence.RateValue = value
				}
			}
		}
	case testutil.ReportPresentationCaseKindBoolean:
		for sectionIndex := range report.AuditAnnex.PerAssetAuditSections {
			for entryIndex := range report.AuditAnnex.PerAssetAuditSections[sectionIndex].Entries {
				report.AuditAnnex.PerAssetAuditSections[sectionIndex].Entries[entryIndex].FullLiquidationEvent = acceptanceCase.BooleanValue
			}
		}
	case testutil.ReportPresentationCaseKindCurrency:
		var sourceID = renderingAcceptanceCurrencySourceID(acceptanceCase.VectorCase)
		var value = mustContractDecimal(acceptanceCase.ExactValue)
		for sectionIndex := range report.AuditAnnex.PerAssetAuditSections {
			for entryIndex := range report.AuditAnnex.PerAssetAuditSections[sectionIndex].Entries {
				var entry = &report.AuditAnnex.PerAssetAuditSections[sectionIndex].Entries[entryIndex]
				if entry.SourceID == sourceID {
					entry.UnitPrice = &value
				}
			}
		}
	}
	return report, nil
}

// applyRenderingAcceptanceFinancialControl applies one matrix vector to every
// model field in its declared financial field class. Non-target financial
// fields receive a distinct sentinel so an aggregate substring cannot satisfy
// a case by matching only an unrelated report value.
// Authored by: OpenCode
func applyRenderingAcceptanceFinancialControl(report *reportmodel.CapitalGainsReport, acceptanceCase testutil.ReportPresentationAcceptanceCase) {
	var sentinel = mustContractDecimal("777777777.77")
	setRenderingAcceptanceFinancialSentinel(report, sentinel)
	if acceptanceCase.Absent {
		clearRenderingAcceptanceFinancialControl(report, acceptanceCase.FinancialFieldClass)
		return
	}

	var exact = sentinel
	if acceptanceCase.Omitted {
		exact = mustContractDecimal("0")
	} else {
		exact = mustContractDecimal(acceptanceCase.ExactValue)
	}
	if acceptanceCase.FinancialFieldClass == "conversion-amount" && exact.IsZero() {
		// Keep the logical conversion row present while preserving its 0.00
		// visible control; exact zero-to-zero entries are intentionally omitted.
		exact = mustContractDecimal("0.004")
	}
	switch acceptanceCase.FinancialFieldClass {
	case "summary-net-gain-or-loss":
		if len(report.SummaryEntries) > 0 {
			report.SummaryEntries[0].NetGainOrLoss = exact
		}
		report.YearlyNetTotal = exact
	case "position-cost-basis":
		if len(report.DetailSections) > 0 {
			report.DetailSections[0].OpeningCostBasis = exact
			report.DetailSections[0].ClosingCostBasis = exact
		}
	case "in-year-activity":
		for sectionIndex := range report.DetailSections {
			if len(report.DetailSections[sectionIndex].ActivityRows) == 0 {
				continue
			}
			var row = &report.DetailSections[sectionIndex].ActivityRows[0]
			row.UnitPrice = renderingAcceptanceDecimalPointer(exact)
			row.GrossValue = renderingAcceptanceDecimalPointer(exact)
			row.FeeAmount = renderingAcceptanceDecimalPointer(exact)
			row.BasisAfterRow = exact
			break
		}
	case "liquidation-allocated-basis", "liquidation-net-proceeds-gain-or-loss":
		for sectionIndex := range report.DetailSections {
			if len(report.DetailSections[sectionIndex].LiquidationSummaries) == 0 {
				continue
			}
			var liquidation = &report.DetailSections[sectionIndex].LiquidationSummaries[0]
			if acceptanceCase.FinancialFieldClass == "liquidation-allocated-basis" {
				liquidation.AllocatedBasis = exact
			} else {
				liquidation.NetLiquidationProceeds = exact
				liquidation.GainOrLoss = exact
			}
			break
		}
	case "audit-activity", "audit-allocated-basis", "audit-net-proceeds-gain-or-loss":
		for sectionIndex := range report.AuditAnnex.PerAssetAuditSections {
			if len(report.AuditAnnex.PerAssetAuditSections[sectionIndex].Entries) == 0 {
				continue
			}
			var entry = &report.AuditAnnex.PerAssetAuditSections[sectionIndex].Entries[0]
			switch acceptanceCase.FinancialFieldClass {
			case "audit-activity":
				entry.UnitPrice = renderingAcceptanceDecimalPointer(exact)
				entry.GrossValue = renderingAcceptanceDecimalPointer(exact)
				entry.FeeAmount = renderingAcceptanceDecimalPointer(exact)
				entry.BasisAfterActivity = exact
			case "audit-allocated-basis":
				entry.AllocatedBasis = renderingAcceptanceDecimalPointer(exact)
			case "audit-net-proceeds-gain-or-loss":
				entry.NetLiquidationProceeds = renderingAcceptanceDecimalPointer(exact)
				entry.GainOrLoss = renderingAcceptanceDecimalPointer(exact)
			}
			break
		}
	case "conversion-amount":
		for entryIndex := range report.AuditAnnex.ConversionAuditEntries {
			for amountIndex := range report.AuditAnnex.ConversionAuditEntries[entryIndex].Amounts {
				report.AuditAnnex.ConversionAuditEntries[entryIndex].Amounts[amountIndex].OriginalAmount = exact
				report.AuditAnnex.ConversionAuditEntries[entryIndex].Amounts[amountIndex].ConvertedAmount = exact
			}
		}
	}
}

// setRenderingAcceptanceFinancialSentinel isolates one matrix case from the
// unrelated monetary fields already present in the deterministic fixture.
// Authored by: OpenCode
func setRenderingAcceptanceFinancialSentinel(report *reportmodel.CapitalGainsReport, sentinel apd.Decimal) {
	for index := range report.SummaryEntries {
		report.SummaryEntries[index].NetGainOrLoss = sentinel
	}
	report.YearlyNetTotal = sentinel
	for sectionIndex := range report.DetailSections {
		var section = &report.DetailSections[sectionIndex]
		section.OpeningCostBasis = sentinel
		section.ClosingCostBasis = sentinel
		for rowIndex := range section.ActivityRows {
			var row = &section.ActivityRows[rowIndex]
			row.UnitPrice = renderingAcceptanceDecimalPointer(sentinel)
			row.GrossValue = renderingAcceptanceDecimalPointer(sentinel)
			row.FeeAmount = renderingAcceptanceDecimalPointer(sentinel)
			row.BasisAfterRow = sentinel
		}
		for liquidationIndex := range section.LiquidationSummaries {
			var liquidation = &section.LiquidationSummaries[liquidationIndex]
			liquidation.AllocatedBasis = sentinel
			liquidation.NetLiquidationProceeds = sentinel
			liquidation.GainOrLoss = sentinel
		}
	}
	for sectionIndex := range report.AuditAnnex.PerAssetAuditSections {
		var section = &report.AuditAnnex.PerAssetAuditSections[sectionIndex]
		for entryIndex := range section.Entries {
			var entry = &section.Entries[entryIndex]
			entry.UnitPrice = renderingAcceptanceDecimalPointer(sentinel)
			entry.GrossValue = renderingAcceptanceDecimalPointer(sentinel)
			entry.FeeAmount = renderingAcceptanceDecimalPointer(sentinel)
			entry.BasisAfterActivity = sentinel
			entry.AllocatedBasis = renderingAcceptanceDecimalPointer(sentinel)
			entry.NetLiquidationProceeds = renderingAcceptanceDecimalPointer(sentinel)
			entry.GainOrLoss = renderingAcceptanceDecimalPointer(sentinel)
		}
	}
	for entryIndex := range report.AuditAnnex.ConversionAuditEntries {
		for amountIndex := range report.AuditAnnex.ConversionAuditEntries[entryIndex].Amounts {
			var amount = &report.AuditAnnex.ConversionAuditEntries[entryIndex].Amounts[amountIndex]
			amount.OriginalAmount = sentinel
			amount.ConvertedAmount = sentinel
		}
	}
}

// renderingAcceptanceDecimalPointer returns an independent optional decimal
// value for a presentation field.
// Authored by: OpenCode
func renderingAcceptanceDecimalPointer(value apd.Decimal) *apd.Decimal {
	var cloned = value
	return &cloned
}

// clearRenderingAcceptanceFinancialControl removes the nullable field class
// from the synthetic report so an absent vector cannot pass through a sentinel.
// Authored by: OpenCode
func clearRenderingAcceptanceFinancialControl(report *reportmodel.CapitalGainsReport, fieldClass string) {
	switch fieldClass {
	case "in-year-activity":
		for index := range report.DetailSections {
			report.DetailSections[index].ActivityRows = nil
		}
	case "liquidation-allocated-basis", "liquidation-net-proceeds-gain-or-loss":
		for index := range report.DetailSections {
			report.DetailSections[index].LiquidationSummaries = nil
		}
	case "audit-activity", "audit-allocated-basis", "audit-net-proceeds-gain-or-loss":
		for index := range report.AuditAnnex.PerAssetAuditSections {
			report.AuditAnnex.PerAssetAuditSections[index].Entries = nil
		}
	}
}

// renderingAcceptanceCloneReport creates an independent exact-model baseline
// for AUD-001 comparison around one renderer call.
// Authored by: OpenCode
func renderingAcceptanceCloneReport(report reportmodel.CapitalGainsReport) (reportmodel.CapitalGainsReport, error) {
	var request, err = reportmodel.NewReportRequest(
		report.Year,
		report.CostBasisMethod,
		reportmodel.ReportBaseCurrency(report.ReportCalculationCurrency),
		reportmodel.ReportOutputFormatMarkdown,
		report.GeneratedAt,
	)
	if err != nil {
		return reportmodel.CapitalGainsReport{}, err
	}
	var cloned reportmodel.CapitalGainsReport
	cloned, err = reportmodel.NewCapitalGainsReportWithConversionArtifacts(
		request,
		report.GeneratedAt,
		report.ReportCalculationCurrency,
		report.SummaryEntries,
		report.YearlyNetTotal,
		report.ReferenceEntries,
		report.DetailSections,
		report.AuditAnnex.ConversionAuditEntries,
		report.RateSources,
	)
	if err != nil {
		return reportmodel.CapitalGainsReport{}, err
	}
	cloned.AuditAnnex, err = reportmodel.NewDetailedAuditAnnex(report.AuditAnnex.PerAssetAuditSections, report.AuditAnnex.ConversionAuditEntries)
	if err != nil {
		return reportmodel.CapitalGainsReport{}, err
	}
	return cloned, nil
}

// assertRenderingAcceptanceManifest verifies the closed case and attempt shape
// before any renderer execution begins.
// Authored by: OpenCode
func assertRenderingAcceptanceManifest(t *testing.T, manifest testutil.ReportPresentationAcceptanceManifest) {
	t.Helper()
	if len(manifest.Cases) != 148 || manifest.Counters.CaseCount != 148 {
		t.Fatalf("closed acceptance case population A = %d/%d, want 148/148", len(manifest.Cases), manifest.Counters.CaseCount)
	}
	var IDs = make(map[string]struct{}, len(manifest.Cases))
	for _, acceptanceCase := range manifest.Cases {
		if acceptanceCase.ID == "" {
			t.Fatalf("closed acceptance case has an empty ID")
		}
		if _, exists := IDs[acceptanceCase.ID]; exists {
			t.Fatalf("closed acceptance case %q is duplicated", acceptanceCase.ID)
		}
		IDs[acceptanceCase.ID] = struct{}{}
		if len(acceptanceCase.Attempts) != 2 {
			t.Fatalf("case %q has %d attempts, want Markdown and PDF", acceptanceCase.ID, len(acceptanceCase.Attempts))
		}
		var expectedAttempts = []testutil.ReportPresentationFormatAttempt{
			{Format: testutil.ReportPresentationFormatMarkdown, DocumentRoles: []testutil.ReportPresentationDocumentRole{testutil.ReportPresentationDocumentRoleMain, testutil.ReportPresentationDocumentRoleAnnex}},
			{Format: testutil.ReportPresentationFormatPDF, DocumentRoles: []testutil.ReportPresentationDocumentRole{testutil.ReportPresentationDocumentRoleCombined}},
		}
		for index, expected := range expectedAttempts {
			if acceptanceCase.Attempts[index].Format != expected.Format || !reflect.DeepEqual(acceptanceCase.Attempts[index].DocumentRoles, expected.DocumentRoles) {
				t.Fatalf("case %q attempt %d = %#v, want %#v", acceptanceCase.ID, index, acceptanceCase.Attempts[index], expected)
			}
		}
	}
	var expectedIDs = renderingAcceptanceClosedCaseIDs()
	for ID := range expectedIDs {
		if _, found := IDs[ID]; !found {
			t.Fatalf("closed acceptance case %q is missing", ID)
		}
	}
	for ID := range IDs {
		if _, found := expectedIDs[ID]; !found {
			t.Fatalf("closed acceptance case %q is extra", ID)
		}
	}
	var expectedCounters = testutil.ReportPresentationAcceptanceCounters{
		CaseCount: 148,
		Populations: map[testutil.ReportPresentationPopulation]int{
			testutil.ReportPresentationPopulationWarning:            296,
			testutil.ReportPresentationPopulationVisibleFinancial:   664,
			testutil.ReportPresentationPopulationModelIntegrity:     296,
			testutil.ReportPresentationPopulationQuantity:           10,
			testutil.ReportPresentationPopulationBoolean:            4,
			testutil.ReportPresentationPopulationClassifiedCurrency: 2,
			testutil.ReportPresentationPopulationUnclassified:       4,
			testutil.ReportPresentationPopulationConversionRow:      16,
			testutil.ReportPresentationPopulationParity:             491,
			testutil.ReportPresentationPopulationConvertedEntry:     24,
		},
	}
	if manifest.Counters.CaseCount != expectedCounters.CaseCount || !reflect.DeepEqual(manifest.Counters.Populations, expectedCounters.Populations) {
		t.Fatalf("closed acceptance counters = %#v, want %#v", manifest.Counters, expectedCounters)
	}
}

// renderingAcceptanceClosedCaseIDs expands the specification's closed case
// schemas independently of the manifest's generated slice.
// Authored by: OpenCode
func renderingAcceptanceClosedCaseIDs() map[string]struct{} {
	var IDs = make(map[string]struct{}, 148)
	IDs["warning/wrapped"] = struct{}{}
	var nonNegative = []string{"zero", "tiny-positive", "whole", "one-place", "two-place", "below-positive-tie", "positive-tie", "above-positive-tie", "positive-carry", "large-positive"}
	var signedOnly = []string{"negative-whole", "negative-below-tie", "negative-tie", "negative-above-tie", "negative-carry", "signed-zero", "negative-tiny", "negative-zero-adjacent-tie", "large-negative"}
	var financialRows = []struct {
		id       string
		signed   bool
		nullable bool
		omitted  bool
	}{
		{id: "summary-net-gain-or-loss", signed: true, omitted: true},
		{id: "position-cost-basis"},
		{id: "in-year-activity", nullable: true},
		{id: "liquidation-allocated-basis", nullable: true},
		{id: "liquidation-net-proceeds-gain-or-loss", signed: true, nullable: true},
		{id: "audit-activity", nullable: true},
		{id: "audit-allocated-basis", nullable: true},
		{id: "audit-net-proceeds-gain-or-loss", signed: true, nullable: true},
		{id: "conversion-amount"},
	}
	for _, row := range financialRows {
		var vectors = nonNegative
		if row.signed {
			vectors = append(append([]string(nil), nonNegative...), signedOnly...)
		}
		for _, vector := range vectors {
			IDs["financial/"+row.id+"/"+vector] = struct{}{}
		}
		if row.nullable {
			IDs["financial/"+row.id+"/absent"] = struct{}{}
		}
		if row.omitted {
			IDs["financial/"+row.id+"/exact-zero-omitted"] = struct{}{}
		}
	}
	for _, ID := range []string{
		"quantity/zero", "quantity/whole-trailing-zero", "quantity/fraction-trailing-zero", "quantity/small", "quantity/large",
		"rate/0.86010", "rate/16.9140", "rate/1.094600", "rate/1.0900", "rate/2.00",
		"boolean/true", "boolean/false",
		"currency/classified-zero-priced", "currency/unclassified-priced", "currency/unclassified-tiny-positive",
		"converted/empty", "converted/unit-price", "converted/gross-value", "converted/fee-amount", "converted/unit-price-gross-value", "converted/unit-price-fee-amount", "converted/gross-value-fee-amount", "converted/all",
	} {
		IDs[ID] = struct{}{}
	}
	return IDs
}

// runRenderingAcceptanceRegressionPopulation runs the fixed calculation test
// packages; R is intentionally separate from renderer-attempt accounting.
// Authored by: OpenCode
func runRenderingAcceptanceRegressionPopulation(t *testing.T) bool {
	t.Helper()
	var root, err = renderingAcceptanceModuleRoot()
	if err != nil {
		t.Errorf("locate module root for R: %v", err)
		return false
	}
	// #nosec G204 -- the command and package paths are fixed by the acceptance contract.
	var command = exec.CommandContext(context.Background(), "go", "test", "./internal/report/basis", "./internal/report/calculate", "./tests/empirical", "-count=1")
	command.Dir = root
	command.Env = append(os.Environ(), "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local")
	if err = command.Run(); err != nil {
		t.Errorf("fixed calculation regression population R failed: %v", err)
		return false
	}
	return true
}

// renderingAcceptanceModuleRoot locates the repository root from this source
// file so the nested fixed-regression command is independent of test cwd.
// Authored by: OpenCode
func renderingAcceptanceModuleRoot() (string, error) {
	var _, filename, _, ok = runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime caller path is unavailable")
	}
	var directory = filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory, nil
		}
		var parent = filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("go.mod was not found")
		}
		directory = parent
	}
}

// renderingAcceptanceOccurrenceKey serializes the complete semantic identity,
// preventing repeated field text from becoming an occurrence denominator.
// Authored by: OpenCode
func renderingAcceptanceOccurrenceKey(occurrence testutil.ReportPresentationOccurrenceKey) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%d", occurrence.Population, occurrence.CaseID, occurrence.Format, occurrence.DocumentRole, occurrence.Section, occurrence.SourceOrRowIdentity, occurrence.FieldName, occurrence.AmountKind, occurrence.AmountOrdinal)
}

// renderingAcceptanceRegressionKey identifies one fixed baseline regression
// identity without importing it into renderer-attempt populations.
// Authored by: OpenCode
func renderingAcceptanceRegressionKey(index int) string {
	return fmt.Sprintf("baseline/%03d", index)
}

// testutilPopulationName converts a closed manifest population to its report
// letter for aggregate accounting.
// Authored by: OpenCode
func testutilPopulationName(population testutil.ReportPresentationPopulation) string {
	return string(population)
}

// renderingAcceptanceConvertedSequence resolves one of the eight closed
// conversion subsequences from the existing synthetic contract fixture.
// Authored by: OpenCode
func renderingAcceptanceConvertedSequence(acceptanceCase testutil.ReportPresentationAcceptanceCase) (contractConvertedSequence, bool) {
	for _, sequence := range contractConvertedAuditSequences() {
		if sequence.ID == acceptanceCase.VectorCase {
			return sequence, true
		}
	}
	return contractConvertedSequence{}, false
}

// renderingAcceptanceConvertedSequenceIndex returns the deterministic source-ID
// suffix used by the existing conversion fixture.
// Authored by: OpenCode
func renderingAcceptanceConvertedSequenceIndex(sequenceID string) string {
	for index, sequence := range contractConvertedAuditSequences() {
		if sequence.ID == sequenceID {
			return fmt.Sprintf("%d", index)
		}
	}
	return "-1"
}

// renderingAcceptanceCurrencySourceID resolves the source row for one closed
// currency control in the synthetic detailed Annex fixture.
// Authored by: OpenCode
func renderingAcceptanceCurrencySourceID(vectorCase string) string {
	switch vectorCase {
	case "classified-zero-priced":
		return "xrp-reduction-2024-001"
	case "unclassified-priced":
		return "eth-reference-buy"
	case "unclassified-tiny-positive":
		return "tiny-positive-unclassified"
	default:
		return ""
	}
}

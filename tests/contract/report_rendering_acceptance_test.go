// Package contract verifies the aggregate report-rendering acceptance accounting.
// Authored by: OpenCode
package contract

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	reportmarkdown "github.com/benizzio/ghostfolio-cryptogains/internal/report/markdown"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	reportpdf "github.com/benizzio/ghostfolio-cryptogains/internal/report/pdf"
	"github.com/benizzio/ghostfolio-cryptogains/internal/report/presentation"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil/runtimeflow"
	"github.com/cockroachdb/apd/v3"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	// renderingAcceptancePopulationA identifies closed renderer acceptance cases.
	// Authored by: OpenCode
	renderingAcceptancePopulationA = "A"
)

// renderingAcceptanceAttemptResult records independently asserted semantic
// observations for one listed format attempt.
// Authored by: OpenCode
type renderingAcceptanceAttemptResult struct {
	format   testutil.ReportPresentationFormat
	passed   bool
	message  string
	observed map[string]string
}

// TestRenderingAcceptanceAccounting executes both format attempts for every
// closed acceptance case and verifies every contract population has exact,
// non-empty numerator and denominator accounting.
// Authored by: OpenCode
func TestRenderingAcceptanceAccounting(t *testing.T) {
	var manifest = testutil.DeterministicReportPresentationAcceptanceFixture()
	assertRenderingAcceptanceManifest(t, manifest)
	assertStaticCalculationRegressionBaseline(t)

	var accounting = newRenderingAcceptanceAccounting(manifest)

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
		accounting.addObservedOccurrences(markdownAttempt.observed)
		accounting.addObservedOccurrences(pdfAttempt.observed)
		if !markdownAttempt.passed {
			t.Errorf("case %q Markdown attempt failed: %s", acceptanceCase.ID, markdownAttempt.message)
		}
		if !pdfAttempt.passed {
			t.Errorf("case %q PDF attempt failed: %s", acceptanceCase.ID, pdfAttempt.message)
		}
		if markdownAttempt.passed && pdfAttempt.passed {
			accounting.add(renderingAcceptancePopulationA, acceptanceCase.ID)
			if err = accounting.addParityOccurrences(acceptanceCase, markdownAttempt.observed, pdfAttempt.observed); err != nil {
				t.Errorf("case %q cross-format parity failed: %v", acceptanceCase.ID, err)
			}
		}
	}

	accounting.assertComplete(t)
}

// TestRenderingAcceptanceAccountingRejectsUnverifiedOccurrences proves that
// accounting cannot be satisfied by a missing, misplaced, blank, or mismatched
// semantic field observation.
// Authored by: OpenCode
func TestRenderingAcceptanceAccountingRejectsUnverifiedOccurrences(t *testing.T) {
	var manifest = testutil.DeterministicReportPresentationAcceptanceFixture()
	var acceptanceCase = manifest.Cases[1]
	var expectedOccurrence testutil.ReportPresentationOccurrenceKey
	for _, occurrence := range acceptanceCase.OccurrenceKeys {
		if occurrence.Population == testutil.ReportPresentationPopulationVisibleFinancial && occurrence.Format == testutil.ReportPresentationFormatMarkdown {
			expectedOccurrence = occurrence
			break
		}
	}
	if expectedOccurrence.CaseID == "" {
		t.Fatal("test fixture has no financial occurrence")
	}
	var accounting = newRenderingAcceptanceAccounting(manifest)
	accounting.addObservedOccurrences(map[string]string{})
	if len(accounting.observed[string(testutil.ReportPresentationPopulationVisibleFinancial)]) != 0 {
		t.Fatal("missing occurrence received credit")
	}
	var misplaced = expectedOccurrence
	misplaced.SourceOrRowIdentity = "wrong-row"
	accounting.addObservedOccurrences(map[string]string{renderingAcceptanceOccurrenceKey(misplaced): "1.00"})
	if len(accounting.observed[string(testutil.ReportPresentationPopulationVisibleFinancial)]) != 0 {
		t.Fatal("misplaced occurrence received credit")
	}
	var observed = make(map[string]string)
	if err := recordRenderingAcceptanceObservation(observed, acceptanceCase, expectedOccurrence, "", "1.00"); err == nil {
		t.Fatal("blank occurrence was accepted")
	}
	if err := recordRenderingAcceptanceObservation(observed, acceptanceCase, expectedOccurrence, "2.00", "1.00"); err == nil {
		t.Fatal("mismatched occurrence was accepted")
	}
	if len(observed) != 0 {
		t.Fatal("rejected occurrence received observation credit")
	}
}

// TestRenderingAcceptancePDFWarningRejectsInvalidSequences proves duplicate,
// partially regular, and misplaced complete warnings cannot earn W credit.
// Authored by: OpenCode
func TestRenderingAcceptancePDFWarningRejectsInvalidSequences(t *testing.T) {
	var acceptanceCase = testutil.DeterministicReportPresentationAcceptanceFixture().Cases[0]
	var occurrence = reportPresentationOccurrenceForFormat(acceptanceCase, testutil.ReportPresentationFormatPDF, testutil.ReportPresentationPopulationWarning)
	var warningFragments = []testutil.PDFTextRun{
		{Text: "The data in this report does not follow any legally required rules", FontResource: "F2"},
		{Text: "for any country's tax returns and is for reference only.", FontResource: "F2"},
	}
	var valid = []testutil.PDFTextRun{
		{Text: "Ghostfolio Capital Gains And Losses Report", FontResource: "F2"},
		{Text: "Report Calculation Currency:", FontResource: "F1"},
		{Text: "EUR", FontResource: "F1"},
	}
	valid = append(valid, warningFragments...)
	valid = append(valid, testutil.PDFTextRun{Text: "Gains-And-Losses Summary", FontResource: "F2"})

	var duplicate = append([]testutil.PDFTextRun(nil), valid[:len(valid)-1]...)
	duplicate = append(duplicate, warningFragments...)
	duplicate = append(duplicate, valid[len(valid)-1])
	var nonBold = append([]testutil.PDFTextRun(nil), valid...)
	nonBold[4].FontResource = "F1"
	var misplaced = append([]testutil.PDFTextRun(nil), valid...)
	misplaced[1], misplaced[3] = misplaced[3], misplaced[1]
	misplaced[2], misplaced[4] = misplaced[4], misplaced[2]

	for name, runs := range map[string][]testutil.PDFTextRun{
		"duplicate": duplicate,
		"non-bold":  nonBold,
		"misplaced": misplaced,
	} {
		t.Run(name, func(t *testing.T) {
			var observed = make(map[string]string)
			var err = recordRenderingAcceptancePDFWarningObservation(observed, acceptanceCase, occurrence, "EUR", testutil.GeneratedPDF{TextRuns: runs})
			if err == nil {
				t.Fatal("invalid PDF warning sequence was accepted")
			}
			if len(observed) != 0 {
				t.Fatal("invalid PDF warning sequence earned W credit")
			}
		})
	}
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

// addObservedOccurrences credits only observations whose complete semantic key
// was independently asserted by a renderer owner.
// Authored by: OpenCode
func (accounting *renderingAcceptanceAccounting) addObservedOccurrences(observed map[string]string) {
	for identity := range observed {
		var occurrence, ok = parseRenderingAcceptanceOccurrenceKey(identity)
		if !ok || occurrence.Format == "cross-format" {
			continue
		}
		var population = string(occurrence.Population)
		if _, expected := accounting.expected[population][identity]; expected {
			accounting.add(population, identity)
		}
	}
}

// addParityOccurrences records a parity key only after both independently
// observed semantic values exist and compare equal.
// Authored by: OpenCode
func (accounting *renderingAcceptanceAccounting) addParityOccurrences(acceptanceCase testutil.ReportPresentationAcceptanceCase, markdown map[string]string, pdf map[string]string) error {
	for _, occurrence := range acceptanceCase.OccurrenceKeys {
		if occurrence.Population != testutil.ReportPresentationPopulationParity {
			continue
		}
		var markdownValue, markdownOK = renderingAcceptanceParityValue(occurrence, testutil.ReportPresentationFormatMarkdown, markdown)
		var pdfValue, pdfOK = renderingAcceptanceParityValue(occurrence, testutil.ReportPresentationFormatPDF, pdf)
		if markdownOK && pdfOK && markdownValue == pdfValue {
			accounting.add(string(occurrence.Population), renderingAcceptanceOccurrenceKey(occurrence))
			continue
		}
		return fmt.Errorf("missing or mismatched observation %q: Markdown=%q PDF=%q", renderingAcceptanceOccurrenceKey(occurrence), markdownValue, pdfValue)
	}
	return nil
}

// renderingAcceptanceParityValue finds an observed format-specific value for a
// cross-format key, preserving population-independent semantic identity.
// Authored by: OpenCode
func renderingAcceptanceParityValue(parity testutil.ReportPresentationOccurrenceKey, format testutil.ReportPresentationFormat, observed map[string]string) (string, bool) {
	for identity, value := range observed {
		var occurrence, ok = parseRenderingAcceptanceOccurrenceKey(identity)
		var expectedRole = reportPresentationDocumentRoleForParity(parity.DocumentRole, format)
		if !ok || occurrence.Format != format || occurrence.CaseID != parity.CaseID || occurrence.DocumentRole != expectedRole || occurrence.Section != parity.Section || occurrence.AssetIdentity != parity.AssetIdentity || occurrence.SourceOrRowIdentity != parity.SourceOrRowIdentity || occurrence.FieldName != parity.FieldName || occurrence.AmountKind != parity.AmountKind || occurrence.AmountOrdinal != parity.AmountOrdinal {
			continue
		}
		return value, true
	}
	return "", false
}

// parseRenderingAcceptanceOccurrenceKey reverses the closed identity encoding
// used by acceptance accounting.
// Authored by: OpenCode
func parseRenderingAcceptanceOccurrenceKey(identity string) (testutil.ReportPresentationOccurrenceKey, bool) {
	var parts = strings.Split(identity, "|")
	if len(parts) != 10 {
		return testutil.ReportPresentationOccurrenceKey{}, false
	}
	var ordinal, err = strconv.Atoi(parts[9])
	if err != nil {
		return testutil.ReportPresentationOccurrenceKey{}, false
	}
	return testutil.ReportPresentationOccurrenceKey{Population: testutil.ReportPresentationPopulation(parts[0]), CaseID: parts[1], Format: testutil.ReportPresentationFormat(parts[2]), DocumentRole: testutil.ReportPresentationDocumentRole(parts[3]), Section: parts[4], AssetIdentity: parts[5], SourceOrRowIdentity: parts[6], FieldName: parts[7], AmountKind: parts[8], AmountOrdinal: ordinal}, true
}

// addRenderingAcceptanceModelObservation credits the AUD-001 comparison only
// after the exact pre/post model assertion succeeds.
// Authored by: OpenCode
func addRenderingAcceptanceModelObservation(observed map[string]string, acceptanceCase testutil.ReportPresentationAcceptanceCase, format testutil.ReportPresentationFormat) {
	for _, occurrence := range acceptanceCase.OccurrenceKeys {
		if occurrence.Population == testutil.ReportPresentationPopulationModelIntegrity && occurrence.Format == format {
			observed[renderingAcceptanceOccurrenceKey(occurrence)] = "equal"
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
	result.observed, err = validateRenderingAcceptanceMarkdownControls(acceptanceCase, report, documents)
	if err != nil {
		result.message = err.Error()
		return result
	}
	if !reflect.DeepEqual(before, report) {
		result.message = "AUD-001 model changed after Markdown rendering"
		return result
	}
	addRenderingAcceptanceModelObservation(result.observed, acceptanceCase, testutil.ReportPresentationFormatMarkdown)
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
	result.observed, err = validateRenderingAcceptancePDFControls(acceptanceCase, report, inspection)
	if err != nil {
		result.message = err.Error()
		return result
	}
	if !reflect.DeepEqual(before, report) {
		result.message = "AUD-001 model changed after PDF rendering"
		return result
	}
	addRenderingAcceptanceModelObservation(result.observed, acceptanceCase, testutil.ReportPresentationFormatPDF)
	result.passed = true
	return result
}

// validateRenderingAcceptanceMarkdownControls validates each Markdown semantic
// occurrence and returns only values that passed their own field assertion.
// Authored by: OpenCode
func validateRenderingAcceptanceMarkdownControls(acceptanceCase testutil.ReportPresentationAcceptanceCase, report reportmodel.CapitalGainsReport, documents []reportmodel.ReportDocument) (map[string]string, error) {
	var observed = make(map[string]string)
	if len(documents) != 2 {
		return observed, fmt.Errorf("Markdown document count = %d, want 2", len(documents))
	}
	var main = string(documents[0].Content)
	var annex = string(documents[1].Content)
	var boldWarning = "**" + testutil.ReportPresentationLegalWarningText + "**"
	var metadataIndex = strings.Index(main, "- **Report Calculation Currency:** "+report.ReportCalculationCurrency)
	var warningIndex = strings.Index(main, "\n"+boldWarning+"\n")
	var summaryIndex = strings.Index(main, "## Gains-And-Losses Summary")
	if strings.Count(main, testutil.ReportPresentationLegalWarningText) != 1 || strings.Count(main, boldWarning) != 1 || strings.Contains(annex, testutil.ReportPresentationLegalWarningText) || metadataIndex < 0 || warningIndex <= metadataIndex || summaryIndex <= warningIndex {
		return observed, fmt.Errorf("warning occurrence or Annex exclusion is invalid")
	}
	if err := recordRenderingAcceptanceObservation(observed, acceptanceCase, reportPresentationOccurrenceForFormat(acceptanceCase, testutil.ReportPresentationFormatMarkdown, testutil.ReportPresentationPopulationWarning), testutil.ReportPresentationLegalWarningText, testutil.ReportPresentationLegalWarningText); err != nil {
		return observed, err
	}
	for _, occurrence := range acceptanceCase.OccurrenceKeys {
		var parityControl = occurrence.Population == testutil.ReportPresentationPopulationParity && (isRenderingAcceptanceRateMetadataField(occurrence.FieldName) || isRenderingAcceptanceAbsentNullableField(acceptanceCase, occurrence.FieldName))
		if (occurrence.Format != testutil.ReportPresentationFormatMarkdown && !parityControl) || occurrence.Population == testutil.ReportPresentationPopulationWarning || occurrence.Population == testutil.ReportPresentationPopulationModelIntegrity || (occurrence.Population == testutil.ReportPresentationPopulationParity && !parityControl) {
			continue
		}
		var expected, err = renderingAcceptanceExpectedValue(acceptanceCase, report, occurrence)
		if err != nil {
			return observed, err
		}
		var lookup = occurrence
		if occurrence.Population == testutil.ReportPresentationPopulationParity && isRenderingAcceptanceAbsentNullableField(acceptanceCase, occurrence.FieldName) {
			lookup.Population = testutil.ReportPresentationPopulationVisibleFinancial
		}
		var actual, found = renderingAcceptanceMarkdownValue(acceptanceCase, report, lookup, main, annex)
		if !found {
			return observed, fmt.Errorf("Markdown occurrence %q is missing", renderingAcceptanceOccurrenceKey(occurrence))
		}
		var concrete = occurrence
		if occurrence.Population == testutil.ReportPresentationPopulationParity {
			concrete.Format = testutil.ReportPresentationFormatMarkdown
			concrete.DocumentRole = reportPresentationDocumentRoleForParity(occurrence.DocumentRole, testutil.ReportPresentationFormatMarkdown)
		}
		if err = recordRenderingAcceptanceObservation(observed, acceptanceCase, concrete, actual, expected); err != nil {
			return observed, err
		}
	}
	addRenderingAcceptanceFormatParityObservations(observed, acceptanceCase, testutil.ReportPresentationFormatMarkdown)
	return observed, nil
}

// validateRenderingAcceptancePDFControls validates each PDF semantic occurrence
// through row-local text runs and returns only asserted observations.
// Authored by: OpenCode
func validateRenderingAcceptancePDFControls(acceptanceCase testutil.ReportPresentationAcceptanceCase, report reportmodel.CapitalGainsReport, inspection testutil.GeneratedPDF) (map[string]string, error) {
	var observed = make(map[string]string)
	var warning = reportPresentationOccurrenceForFormat(acceptanceCase, testutil.ReportPresentationFormatPDF, testutil.ReportPresentationPopulationWarning)
	if err := recordRenderingAcceptancePDFWarningObservation(observed, acceptanceCase, warning, report.ReportCalculationCurrency, inspection); err != nil {
		return observed, err
	}
	for _, occurrence := range acceptanceCase.OccurrenceKeys {
		var parityControl = occurrence.Population == testutil.ReportPresentationPopulationParity && (isRenderingAcceptanceRateMetadataField(occurrence.FieldName) || isRenderingAcceptanceAbsentNullableField(acceptanceCase, occurrence.FieldName))
		if (occurrence.Format != testutil.ReportPresentationFormatPDF && !parityControl) || occurrence.Population == testutil.ReportPresentationPopulationWarning || occurrence.Population == testutil.ReportPresentationPopulationModelIntegrity || (occurrence.Population == testutil.ReportPresentationPopulationParity && !parityControl) {
			continue
		}
		var expected, err = renderingAcceptanceExpectedValue(acceptanceCase, report, occurrence)
		if err != nil {
			return observed, err
		}
		var lookup = occurrence
		if occurrence.Population == testutil.ReportPresentationPopulationParity && isRenderingAcceptanceAbsentNullableField(acceptanceCase, occurrence.FieldName) {
			lookup.Population = testutil.ReportPresentationPopulationVisibleFinancial
		}
		var actual, found = renderingAcceptancePDFValue(acceptanceCase, report, lookup, inspection)
		if !found {
			return observed, fmt.Errorf("PDF occurrence %q is missing", renderingAcceptanceOccurrenceKey(occurrence))
		}
		var concrete = occurrence
		if occurrence.Population == testutil.ReportPresentationPopulationParity {
			concrete.Format = testutil.ReportPresentationFormatPDF
			concrete.DocumentRole = reportPresentationDocumentRoleForParity(occurrence.DocumentRole, testutil.ReportPresentationFormatPDF)
		}
		if err = recordRenderingAcceptanceObservation(observed, acceptanceCase, concrete, actual, expected); err != nil {
			return observed, err
		}
	}
	addRenderingAcceptanceFormatParityObservations(observed, acceptanceCase, testutil.ReportPresentationFormatPDF)
	return observed, nil
}

// renderingAcceptancePDFRunSequence identifies one complete logical phrase in
// ordered PDF text runs, including all width-wrapped fragments.
// Authored by: OpenCode
type renderingAcceptancePDFRunSequence struct {
	start int
	end   int
	runs  []testutil.PDFTextRun
}

// recordRenderingAcceptancePDFWarningObservation credits W only after the sole
// complete sequence is fully bold and lies between complete metadata and summary
// sequences.
// Authored by: OpenCode
func recordRenderingAcceptancePDFWarningObservation(observed map[string]string, acceptanceCase testutil.ReportPresentationAcceptanceCase, occurrence testutil.ReportPresentationOccurrenceKey, currency string, inspection testutil.GeneratedPDF) error {
	var warningSequences = renderingAcceptancePDFRunSequences(inspection.TextRuns, testutil.ReportPresentationLegalWarningText)
	if len(warningSequences) != 1 {
		return fmt.Errorf("PDF complete warning sequence count = %d, want 1", len(warningSequences))
	}
	var boldResource string
	for _, run := range inspection.TextRuns {
		if renderingAcceptancePDFText(run.Text) == "Ghostfolio Capital Gains And Losses Report" {
			boldResource = run.FontResource
			break
		}
	}
	if boldResource == "" {
		return fmt.Errorf("PDF embedded bold font resource is not identifiable")
	}
	var warningSequence = warningSequences[0]
	for _, fragment := range warningSequence.runs {
		if fragment.FontResource != boldResource {
			return fmt.Errorf("PDF warning fragment %q uses font %q, want %q", fragment.Text, fragment.FontResource, boldResource)
		}
	}
	var metadataSequences = renderingAcceptancePDFRunSequences(inspection.TextRuns, "Report Calculation Currency: "+currency)
	var summarySequences = renderingAcceptancePDFRunSequences(inspection.TextRuns, "Gains-And-Losses Summary")
	if len(metadataSequences) != 1 || len(summarySequences) == 0 || metadataSequences[0].end >= warningSequence.start || summarySequences[0].start <= warningSequence.end {
		return fmt.Errorf("PDF warning is not after metadata and before summary")
	}
	return recordRenderingAcceptanceObservation(observed, acceptanceCase, occurrence, testutil.ReportPresentationLegalWarningText, testutil.ReportPresentationLegalWarningText)
}

// renderingAcceptancePDFRunSequences returns every complete contiguous logical
// phrase from the ordered run stream rather than stopping at the first match.
// Authored by: OpenCode
func renderingAcceptancePDFRunSequences(runs []testutil.PDFTextRun, target string) []renderingAcceptancePDFRunSequence {
	var normalizedTarget = renderingAcceptancePDFText(target)
	var sequences []renderingAcceptancePDFRunSequence
	for start := range runs {
		var joined string
		for end := start; end < len(runs); end++ {
			joined = renderingAcceptancePDFText(joined + " " + runs[end].Text)
			if joined == normalizedTarget {
				sequences = append(sequences, renderingAcceptancePDFRunSequence{start: start, end: end, runs: append([]testutil.PDFTextRun(nil), runs[start:end+1]...)})
				break
			}
			if !strings.HasPrefix(normalizedTarget, joined) {
				break
			}
		}
	}
	return sequences
}

// renderingAcceptancePDFText normalizes only layout whitespace while retaining
// warning punctuation and complete words.
// Authored by: OpenCode
func renderingAcceptancePDFText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

// reportPresentationOccurrenceForFormat finds the single warning or model key
// for a case and format.
// Authored by: OpenCode
func reportPresentationOccurrenceForFormat(acceptanceCase testutil.ReportPresentationAcceptanceCase, format testutil.ReportPresentationFormat, population testutil.ReportPresentationPopulation) testutil.ReportPresentationOccurrenceKey {
	for _, occurrence := range acceptanceCase.OccurrenceKeys {
		if occurrence.Format == format && occurrence.Population == population {
			return occurrence
		}
	}
	return testutil.ReportPresentationOccurrenceKey{Population: population, CaseID: acceptanceCase.ID, Format: format}
}

// recordRenderingAcceptanceObservation records a value only after its complete
// semantic identity and exact expected text have both passed.
// Authored by: OpenCode
func recordRenderingAcceptanceObservation(observed map[string]string, acceptanceCase testutil.ReportPresentationAcceptanceCase, occurrence testutil.ReportPresentationOccurrenceKey, actual string, expected string) error {
	if occurrence.CaseID != acceptanceCase.ID {
		return fmt.Errorf("observation case mismatch for %q", renderingAcceptanceOccurrenceKey(occurrence))
	}
	if actual != expected {
		return fmt.Errorf("occurrence %q = %q, want %q", renderingAcceptanceOccurrenceKey(occurrence), actual, expected)
	}
	observed[renderingAcceptanceOccurrenceKey(occurrence)] = actual
	return nil
}

// addRenderingAcceptanceFormatParityObservations records format-local evidence
// for parity-only rate metadata and exact omission controls.
// Authored by: OpenCode
func addRenderingAcceptanceFormatParityObservations(observed map[string]string, acceptanceCase testutil.ReportPresentationAcceptanceCase, format testutil.ReportPresentationFormat) {
	for _, occurrence := range acceptanceCase.OccurrenceKeys {
		if occurrence.Population != testutil.ReportPresentationPopulationParity {
			continue
		}
		if (acceptanceCase.Omitted || (acceptanceCase.FinancialFieldClass == "summary-net-gain-or-loss" && (acceptanceCase.ExactValue == "0" || acceptanceCase.ExactValue == "-0"))) && occurrence.FieldName == "per_asset_net_gain_or_loss" {
			var formatOccurrence = occurrence
			formatOccurrence.Format = format
			formatOccurrence.DocumentRole = reportPresentationDocumentRoleForParity(occurrence.DocumentRole, format)
			observed[renderingAcceptanceOccurrenceKey(formatOccurrence)] = "<omitted>"
		}
	}
}

// reportPresentationDocumentRoleForParity converts a logical section role to
// the concrete combined-PDF role used by format-local observations.
// Authored by: OpenCode
func reportPresentationDocumentRoleForParity(role testutil.ReportPresentationDocumentRole, format testutil.ReportPresentationFormat) testutil.ReportPresentationDocumentRole {
	if format == testutil.ReportPresentationFormatPDF && role != "model" {
		return testutil.ReportPresentationDocumentRoleCombined
	}
	return role
}

// renderingAcceptanceExpectedValue derives the exact expected value for one
// occurrence from the unformatted synthetic report model.
// Authored by: OpenCode
func renderingAcceptanceExpectedValue(acceptanceCase testutil.ReportPresentationAcceptanceCase, report reportmodel.CapitalGainsReport, occurrence testutil.ReportPresentationOccurrenceKey) (string, error) {
	if occurrence.Population == testutil.ReportPresentationPopulationParity {
		if isRenderingAcceptanceRateMetadataField(occurrence.FieldName) {
			return renderingAcceptanceRateMetadataValue(report, occurrence.FieldName)
		}
		if acceptanceCase.Omitted && occurrence.FieldName == "per_asset_net_gain_or_loss" {
			return "<omitted>", nil
		}
		if isRenderingAcceptanceAbsentNullableField(acceptanceCase, occurrence.FieldName) {
			return "", nil
		}
	}
	switch occurrence.Population {
	case testutil.ReportPresentationPopulationWarning:
		return testutil.ReportPresentationLegalWarningText, nil
	case testutil.ReportPresentationPopulationQuantity:
		return renderingAcceptanceQuantityExpectedValue(report, occurrence), nil
	case testutil.ReportPresentationPopulationBoolean:
		return acceptanceCase.ExpectedVisibleValue, nil
	case testutil.ReportPresentationPopulationClassifiedCurrency, testutil.ReportPresentationPopulationUnclassified:
		return renderingAcceptanceCurrencyValue(report, occurrence.SourceOrRowIdentity), nil
	case testutil.ReportPresentationPopulationConversionRow:
		var sequence, ok = renderingAcceptanceConvertedSequence(acceptanceCase)
		if !ok {
			return "", fmt.Errorf("converted sequence is not closed for %q", acceptanceCase.ID)
		}
		return normalizeAcceptanceConvertedCell(sequence.ExpectedMarkdownCell), nil
	case testutil.ReportPresentationPopulationConvertedEntry:
		var amount, ok = renderingAcceptanceConversionAmount(report, occurrence.SourceOrRowIdentity, occurrence.AmountKind)
		if !ok {
			return "", fmt.Errorf("conversion amount %q is missing", occurrence.AmountKind)
		}
		return renderingAcceptanceConvertedEntryText(amount), nil
	case testutil.ReportPresentationPopulationVisibleFinancial:
		if acceptanceCase.Absent && presentationFinancialFieldIsNullableForAcceptance(acceptanceCase, occurrence.FieldName) {
			return "", nil
		}
		var value, ok = renderingAcceptanceFinancialValue(report, occurrence)
		if !ok {
			return "", fmt.Errorf("financial occurrence %q has no model value", renderingAcceptanceOccurrenceKey(occurrence))
		}
		return presentation.FormatFinancialValue(value)
	}
	return "", nil
}

// isRenderingAcceptanceRateMetadataField identifies parity-only conversion-row
// metadata controls.
// Authored by: OpenCode
func isRenderingAcceptanceRateMetadataField(fieldName string) bool {
	switch fieldName {
	case "source_id", "asset", "rate_date", "source_currency", "report_base_currency", "quote_direction", "rate_value":
		return true
	default:
		return false
	}
}

// renderingAcceptanceRateMetadataValue returns one canonical visible rate-row
// metadata value from the validated conversion evidence.
// Authored by: OpenCode
func renderingAcceptanceRateMetadataValue(report reportmodel.CapitalGainsReport, fieldName string) (string, error) {
	if len(report.AuditAnnex.ConversionAuditEntries) == 0 {
		return "", fmt.Errorf("rate metadata has no conversion row")
	}
	var entry = report.AuditAnnex.ConversionAuditEntries[0]
	switch fieldName {
	case "source_id":
		return entry.SourceID, nil
	case "asset":
		return entry.AssetLabel, nil
	case "rate_date":
		return entry.RateDate.Format("2006-01-02"), nil
	case "source_currency":
		return entry.SourceCurrency, nil
	case "report_base_currency":
		return entry.ReportBaseCurrency.Label(), nil
	case "quote_direction":
		return reportmodel.RenderQuoteDirectionLabel(entry.QuoteDirection)
	case "rate_value":
		return canonicalAcceptanceDecimal(entry.RateValue), nil
	default:
		return "", fmt.Errorf("unknown rate metadata field %q", fieldName)
	}
}

// canonicalAcceptanceDecimal returns the report's canonical fixed-point text.
// Authored by: OpenCode
func canonicalAcceptanceDecimal(value apd.Decimal) string {
	var rendered, err = decimalsupport.CanonicalString(value)
	if err != nil {
		return ""
	}
	return rendered
}

// presentationFinancialFieldIsNullableForAcceptance limits blank controls to
// optional model fields while retaining required decimal fields in their rows.
// Authored by: OpenCode
func presentationFinancialFieldIsNullableForAcceptance(acceptanceCase testutil.ReportPresentationAcceptanceCase, fieldName string) bool {
	for _, field := range acceptanceCase.FinancialFields {
		if field.Name == fieldName {
			return field.Nullable
		}
	}
	return false
}

// isRenderingAcceptanceAbsentNullableField identifies blank fields retained for
// applicability and parity evidence but excluded from V.
// Authored by: OpenCode
func isRenderingAcceptanceAbsentNullableField(acceptanceCase testutil.ReportPresentationAcceptanceCase, fieldName string) bool {
	return acceptanceCase.Absent && presentationFinancialFieldIsNullableForAcceptance(acceptanceCase, fieldName)
}

// renderingAcceptanceFinancialValue returns one exact model amount for a
// field-local financial occurrence.
// Authored by: OpenCode
func renderingAcceptanceFinancialValue(report reportmodel.CapitalGainsReport, occurrence testutil.ReportPresentationOccurrenceKey) (apd.Decimal, bool) {
	if occurrence.Section == "detailed_per_asset_audit" {
		for _, section := range report.AuditAnnex.PerAssetAuditSections {
			for _, entry := range section.Entries {
				if entry.SourceID != occurrence.SourceOrRowIdentity {
					continue
				}
				switch occurrence.FieldName {
				case "unit_price":
					if entry.UnitPrice == nil {
						return apd.Decimal{}, false
					}
					return *entry.UnitPrice, true
				case "gross_value":
					if entry.GrossValue == nil {
						return apd.Decimal{}, false
					}
					return *entry.GrossValue, true
				case "fee_amount":
					if entry.FeeAmount == nil {
						return apd.Decimal{}, false
					}
					return *entry.FeeAmount, true
				case "basis_after_activity":
					return entry.BasisAfterActivity, true
				case "allocated_basis":
					if entry.AllocatedBasis == nil {
						return apd.Decimal{}, false
					}
					return *entry.AllocatedBasis, true
				case "net_proceeds":
					if entry.NetLiquidationProceeds == nil {
						return apd.Decimal{}, false
					}
					return *entry.NetLiquidationProceeds, true
				case "gain_or_loss":
					if entry.GainOrLoss == nil {
						return apd.Decimal{}, false
					}
					return *entry.GainOrLoss, true
				}
			}
		}
		return apd.Decimal{}, false
	}
	switch occurrence.FieldName {
	case "per_asset_net_gain_or_loss":
		if len(report.SummaryEntries) == 0 {
			return apd.Decimal{}, false
		}
		return report.SummaryEntries[0].NetGainOrLoss, true
	case "overall_yearly_net_total":
		return report.YearlyNetTotal, true
	case "opening_cost_basis":
		if len(report.DetailSections) == 0 {
			return apd.Decimal{}, false
		}
		return report.DetailSections[0].OpeningCostBasis, true
	case "closing_cost_basis":
		if len(report.DetailSections) == 0 {
			return apd.Decimal{}, false
		}
		return report.DetailSections[0].ClosingCostBasis, true
	case "historical_cost_basis":
		for _, section := range report.DetailSections {
			if section.AssetIdentityKey == occurrence.AssetIdentity {
				return section.ClosingCostBasis, true
			}
		}
	case "unit_price", "gross_value", "fee_amount", "basis_after_row":
		for _, section := range report.DetailSections {
			for _, row := range section.ActivityRows {
				if row.SourceID != occurrence.SourceOrRowIdentity {
					continue
				}
				switch occurrence.FieldName {
				case "unit_price":
					if row.UnitPrice == nil {
						return apd.Decimal{}, false
					}
					return *row.UnitPrice, true
				case "gross_value":
					if row.GrossValue == nil {
						return apd.Decimal{}, false
					}
					return *row.GrossValue, true
				case "fee_amount":
					if row.FeeAmount == nil {
						return apd.Decimal{}, false
					}
					return *row.FeeAmount, true
				default:
					return row.BasisAfterRow, true
				}
			}
		}
	case "allocated_basis", "net_proceeds", "gain_or_loss":
		for _, section := range report.DetailSections {
			for _, liquidation := range section.LiquidationSummaries {
				if liquidation.SourceID != occurrence.SourceOrRowIdentity {
					continue
				}
				switch occurrence.FieldName {
				case "allocated_basis":
					return liquidation.AllocatedBasis, true
				case "net_proceeds":
					return liquidation.NetLiquidationProceeds, true
				default:
					return liquidation.GainOrLoss, true
				}
			}
			for _, section := range report.AuditAnnex.PerAssetAuditSections {
				for _, entry := range section.Entries {
					if entry.SourceID != occurrence.SourceOrRowIdentity {
						continue
					}
					switch occurrence.FieldName {
					case "allocated_basis":
						if entry.AllocatedBasis == nil {
							return apd.Decimal{}, false
						}
						return *entry.AllocatedBasis, true
					case "net_proceeds":
						if entry.NetLiquidationProceeds == nil {
							return apd.Decimal{}, false
						}
						return *entry.NetLiquidationProceeds, true
					default:
						if entry.GainOrLoss == nil {
							return apd.Decimal{}, false
						}
						return *entry.GainOrLoss, true
					}
				}
			}
		}
	case "original_unit_price", "converted_unit_price", "original_gross_value", "converted_gross_value", "original_fee_amount", "converted_fee_amount":
		var amountKind = strings.TrimPrefix(strings.TrimPrefix(occurrence.FieldName, "original_"), "converted_")
		var amount, ok = renderingAcceptanceConversionAmount(report, occurrence.SourceOrRowIdentity, amountKind)
		if !ok {
			return apd.Decimal{}, false
		}
		if strings.HasPrefix(occurrence.FieldName, "original_") {
			return amount.OriginalAmount, true
		}
		return amount.ConvertedAmount, true
	}
	return apd.Decimal{}, false
}

// renderingAcceptanceConversionAmount returns one received conversion amount
// without changing its order or exact values.
// Authored by: OpenCode
func renderingAcceptanceConversionAmount(report reportmodel.CapitalGainsReport, sourceID string, amountKind string) (reportmodel.ConvertedActivityAmount, bool) {
	for _, entry := range report.AuditAnnex.ConversionAuditEntries {
		if entry.SourceID != sourceID {
			continue
		}
		for _, amount := range entry.Amounts {
			if string(amount.AmountKind) == amountKind {
				return amount, true
			}
		}
	}
	return reportmodel.ConvertedActivityAmount{}, false
}

// renderingAcceptanceConvertedEntryText creates the exact logical conversion
// entry expected in either renderer's local cell.
// Authored by: OpenCode
func renderingAcceptanceConvertedEntryText(amount reportmodel.ConvertedActivityAmount) string {
	var original, originalErr = presentation.FormatFinancialValue(amount.OriginalAmount)
	if originalErr != nil {
		original = canonicalAcceptanceDecimal(amount.OriginalAmount)
	}
	var converted, convertedErr = presentation.FormatFinancialValue(amount.ConvertedAmount)
	if convertedErr != nil {
		converted = canonicalAcceptanceDecimal(amount.ConvertedAmount)
	}
	return string(amount.AmountKind) + ": " + original + " -> " + converted
}

// renderingAcceptanceCurrencyValue returns the visible original currency for a
// concrete Annex source row.
// Authored by: OpenCode
func renderingAcceptanceCurrencyValue(report reportmodel.CapitalGainsReport, sourceID string) string {
	for _, section := range report.AuditAnnex.PerAssetAuditSections {
		for _, entry := range section.Entries {
			if entry.SourceID == sourceID && entry.IsZeroPricedHoldingReduction {
				return ""
			}
			if entry.SourceID == sourceID {
				return entry.ActivityCurrency
			}
		}
	}
	return ""
}

// renderingAcceptanceQuantityExpectedValue returns the exact visible quantity
// for one occurrence. Positive-only activity quantities retain valid positive
// controls when the scalar case is the zero quantity boundary.
// Authored by: OpenCode
func renderingAcceptanceQuantityExpectedValue(report reportmodel.CapitalGainsReport, occurrence testutil.ReportPresentationOccurrenceKey) string {
	if occurrence.SourceOrRowIdentity == "opening-position" {
		return canonicalAcceptanceDecimal(report.DetailSections[0].OpeningQuantity)
	}
	if occurrence.SourceOrRowIdentity == "closing-position" {
		return canonicalAcceptanceDecimal(report.DetailSections[0].ClosingQuantity)
	}
	if occurrence.SourceOrRowIdentity == "historical-position" {
		for _, section := range report.DetailSections {
			if section.AssetIdentityKey == "asset-historical" {
				return canonicalAcceptanceDecimal(section.ClosingQuantity)
			}
		}
	}
	for _, section := range report.DetailSections {
		for _, row := range section.ActivityRows {
			if row.SourceID == occurrence.SourceOrRowIdentity {
				if occurrence.FieldName == "activity_quantity" {
					return canonicalAcceptanceDecimal(row.Quantity)
				}
				if occurrence.FieldName == "quantity_after_row" {
					return canonicalAcceptanceDecimal(row.QuantityAfterRow)
				}
			}
		}
		for _, liquidation := range section.LiquidationSummaries {
			if liquidation.SourceID == occurrence.SourceOrRowIdentity && occurrence.FieldName == "disposed_quantity" {
				return canonicalAcceptanceDecimal(liquidation.DisposedQuantity)
			}
		}
	}
	for _, section := range report.AuditAnnex.PerAssetAuditSections {
		for _, entry := range section.Entries {
			if entry.SourceID == occurrence.SourceOrRowIdentity {
				if occurrence.FieldName == "audit_quantity" {
					return canonicalAcceptanceDecimal(entry.Quantity)
				}
				if occurrence.FieldName == "quantity_after_activity" {
					return canonicalAcceptanceDecimal(entry.QuantityAfterActivity)
				}
			}
		}
	}
	return ""
}

// renderingAcceptanceMarkdownValue extracts one field from its owning Markdown
// row or position block, preserving blank table cells as empty strings.
// Authored by: OpenCode
func renderingAcceptanceMarkdownValue(acceptanceCase testutil.ReportPresentationAcceptanceCase, report reportmodel.CapitalGainsReport, occurrence testutil.ReportPresentationOccurrenceKey, main string, annex string) (string, bool) {
	if occurrence.Population == testutil.ReportPresentationPopulationParity && isRenderingAcceptanceRateMetadataField(occurrence.FieldName) {
		var row, found = markdownAcceptanceRow(annex, occurrence.SourceOrRowIdentity, 9)
		if !found {
			return "", false
		}
		var columns = map[string]int{"source_id": 1, "asset": 2, "rate_date": 3, "source_currency": 4, "report_base_currency": 5, "quote_direction": 7, "rate_value": 8}
		column, ok := columns[occurrence.FieldName]
		if !ok {
			return "", false
		}
		return row[column], true
	}
	switch occurrence.Population {
	case testutil.ReportPresentationPopulationConversionRow:
		var row, found = markdownAcceptanceRow(annex, occurrence.SourceOrRowIdentity, 9)
		if !found {
			return "", false
		}
		return normalizeAcceptanceConvertedCell(row[6]), true
	case testutil.ReportPresentationPopulationConvertedEntry:
		var row, found = markdownAcceptanceRow(annex, occurrence.SourceOrRowIdentity, 9)
		if !found {
			return "", false
		}
		var expected, err = renderingAcceptanceExpectedValue(acceptanceCase, report, occurrence)
		if err != nil {
			return "", false
		}
		if strings.Contains(row[6], expected) {
			return expected, true
		}
		return "", false
	case testutil.ReportPresentationPopulationClassifiedCurrency, testutil.ReportPresentationPopulationUnclassified:
		var row, found = markdownAcceptanceRow(annex, occurrence.SourceOrRowIdentity, 17)
		if !found {
			return "", false
		}
		return row[7], true
	case testutil.ReportPresentationPopulationBoolean:
		var row, found = markdownAcceptanceRow(annex, occurrence.SourceOrRowIdentity, 17)
		if !found {
			return "", false
		}
		return row[11], true
	case testutil.ReportPresentationPopulationQuantity:
		return markdownAcceptanceQuantityValue(main, annex, occurrence)
	case testutil.ReportPresentationPopulationVisibleFinancial:
		if strings.HasPrefix(occurrence.FieldName, "original_") || strings.HasPrefix(occurrence.FieldName, "converted_") {
			var row, found = markdownAcceptanceRow(annex, occurrence.SourceOrRowIdentity, 9)
			if !found {
				return "", false
			}
			var amount, ok = renderingAcceptanceConversionAmount(report, occurrence.SourceOrRowIdentity, occurrence.AmountKind)
			if !ok {
				return "", false
			}
			var expectedEntry = renderingAcceptanceConvertedEntryText(amount)
			if !strings.Contains(row[6], expectedEntry) {
				return "", false
			}
			if strings.HasPrefix(occurrence.FieldName, "original_") {
				var value, err = presentation.FormatFinancialValue(amount.OriginalAmount)
				if err != nil {
					return canonicalAcceptanceDecimal(amount.OriginalAmount), true
				}
				return value, true
			}
			var value, err = presentation.FormatFinancialValue(amount.ConvertedAmount)
			if err != nil {
				return canonicalAcceptanceDecimal(amount.ConvertedAmount), true
			}
			return value, true
		}
		if occurrence.Section == "gains_and_losses_summary" {
			var label = "Overall Yearly Net Total"
			if occurrence.FieldName == "per_asset_net_gain_or_loss" {
				label = "BTC"
			}
			var row, found = markdownAcceptanceRowByFirstCell(main, label)
			if !found {
				return "", false
			}
			return row[1], true
		}
		if occurrence.Section == "position" {
			var heading = "Opening Position"
			if occurrence.SourceOrRowIdentity == "closing-position" {
				heading = "Closing Position"
			}
			if occurrence.SourceOrRowIdentity == "historical-position" {
				heading = "Historical Position"
			}
			var label = "Cost Basis"
			return markdownAcceptancePositionValue(main, occurrence.AssetIdentity, heading, label)
		}
		if occurrence.Section == "in_year_activity" {
			var row, found = markdownAcceptanceRow(main, occurrence.SourceOrRowIdentity, 13)
			if !found {
				return "", false
			}
			var columns = map[string]int{"unit_price": 4, "gross_value": 5, "fee_amount": 6, "basis_after_row": 8}
			column, ok := columns[occurrence.FieldName]
			if !ok {
				return "", false
			}
			return row[column], true
		}
		if occurrence.Section == "liquidation_calculations" {
			var row, found = markdownAcceptanceRow(main, occurrence.SourceOrRowIdentity, 7)
			if !found {
				return "", false
			}
			var columns = map[string]int{"allocated_basis": 3, "net_proceeds": 4, "gain_or_loss": 5}
			column, ok := columns[occurrence.FieldName]
			if !ok {
				return "", false
			}
			return row[column], true
		}
		if occurrence.Section == "detailed_per_asset_audit" {
			var row, found = markdownAcceptanceRow(annex, occurrence.SourceOrRowIdentity, 17)
			if !found {
				return "", false
			}
			var columns = map[string]int{"unit_price": 4, "gross_value": 5, "fee_amount": 6, "basis_after_activity": 10, "allocated_basis": 12, "net_proceeds": 13, "gain_or_loss": 14}
			column, ok := columns[occurrence.FieldName]
			if !ok {
				return "", false
			}
			return row[column], true
		}
	}
	return "", false
}

// markdownAcceptanceRow returns one pipe-table row with leading/trailing pipes
// removed and blank cells retained.
// Authored by: OpenCode
func markdownAcceptanceRow(content string, sourceID string, width int) ([]string, bool) {
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
			continue
		}
		var cells = splitMarkdownAcceptanceCells(line)
		if len(cells) == width && len(cells) > 1 && cells[1] == sourceID {
			return cells, true
		}
	}
	return nil, false
}

// markdownAcceptanceRowByFirstCell locates summary rows by their unique first
// semantic cell.
// Authored by: OpenCode
func markdownAcceptanceRowByFirstCell(content string, firstCell string) ([]string, bool) {
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
			continue
		}
		var cells = splitMarkdownAcceptanceCells(line)
		if len(cells) >= 2 && cells[0] == firstCell {
			return cells, true
		}
	}
	return nil, false
}

// splitMarkdownAcceptanceCells parses a pipe-table row without losing blank
// semantic cells.
// Authored by: OpenCode
func splitMarkdownAcceptanceCells(line string) []string {
	var raw = strings.Split(line[1:len(line)-1], "|")
	for index := range raw {
		raw[index] = strings.TrimSpace(raw[index])
	}
	return raw
}

// markdownAcceptancePositionValue locates a labeled position value inside one
// asset detail block.
// Authored by: OpenCode
func markdownAcceptancePositionValue(content string, asset string, heading string, label string) (string, bool) {
	var inAsset, inHeading bool
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "## Asset Detail: ") {
			inAsset = strings.TrimSpace(strings.TrimPrefix(line, "## Asset Detail: ")) == renderingAcceptanceAssetLabel(asset)
			inHeading = false
		}
		if !inAsset {
			continue
		}
		if strings.TrimSpace(line) == "### "+heading {
			inHeading = true
			continue
		}
		if inHeading && strings.HasPrefix(line, "### ") {
			inHeading = false
		}
		if inHeading {
			var prefix = "- **" + label + ":** "
			if strings.HasPrefix(line, prefix) {
				return strings.TrimSpace(strings.TrimPrefix(line, prefix)), true
			}
		}
	}
	return "", false
}

// renderingAcceptanceAssetLabel maps a fixture asset identity to its visible
// report label.
// Authored by: OpenCode
func renderingAcceptanceAssetLabel(asset string) string {
	if asset == "asset-historical" {
		return "HIST"
	}
	return strings.ToUpper(strings.TrimPrefix(asset, "asset-"))
}

// markdownAcceptanceQuantityValue maps each quantity occurrence to its exact
// visible Markdown field.
// Authored by: OpenCode
func markdownAcceptanceQuantityValue(main string, annex string, occurrence testutil.ReportPresentationOccurrenceKey) (string, bool) {
	switch occurrence.FieldName {
	case "opening_quantity":
		return markdownAcceptancePositionValue(main, occurrence.AssetIdentity, "Opening Position", "Quantity")
	case "closing_quantity":
		return markdownAcceptancePositionValue(main, occurrence.AssetIdentity, "Closing Position", "Quantity")
	case "historical_position_quantity":
		return markdownAcceptancePositionValue(main, occurrence.AssetIdentity, "Historical Position", "Quantity")
	case "activity_quantity", "quantity_after_row":
		var row, found = markdownAcceptanceRow(main, occurrence.SourceOrRowIdentity, 13)
		if !found {
			return "", false
		}
		if occurrence.FieldName == "activity_quantity" {
			return row[3], true
		}
		return row[7], true
	case "disposed_quantity":
		var row, found = markdownAcceptanceRow(main, occurrence.SourceOrRowIdentity, 7)
		if !found {
			return "", false
		}
		return row[2], true
	case "audit_quantity", "quantity_after_activity":
		var row, found = markdownAcceptanceRow(annex, occurrence.SourceOrRowIdentity, 17)
		if !found {
			return "", false
		}
		if occurrence.FieldName == "audit_quantity" {
			return row[3], true
		}
		return row[9], true
	}
	return "", false
}

// renderingAcceptancePDFValue extracts one field from a row-local PDF text-run
// set. Searchable document text is used only for the warning; all other values
// are constrained to their semantic column and source row.
// Authored by: OpenCode
func renderingAcceptancePDFValue(acceptanceCase testutil.ReportPresentationAcceptanceCase, report reportmodel.CapitalGainsReport, occurrence testutil.ReportPresentationOccurrenceKey, inspection testutil.GeneratedPDF) (string, bool) {
	if occurrence.Population == testutil.ReportPresentationPopulationParity && isRenderingAcceptanceRateMetadataField(occurrence.FieldName) {
		var rowRuns = contractPDFConversionRowRuns(inspection, occurrence.SourceOrRowIdentity, []string{occurrence.SourceOrRowIdentity})
		if len(rowRuns) == 0 {
			return "", false
		}
		var bounds = map[string][2]float64{"source_id": {100, 175}, "asset": {175, 235}, "rate_date": {235, 305}, "source_currency": {305, 380}, "report_base_currency": {380, 465}, "quote_direction": {610, 745}, "rate_value": {745, 830}}
		column, ok := bounds[occurrence.FieldName]
		if !ok {
			return "", false
		}
		return renderingAcceptancePDFCell(rowRuns, column[0], column[1], false), true
	}
	switch occurrence.Population {
	case testutil.ReportPresentationPopulationConversionRow:
		var rowRuns = renderingAcceptancePDFConversionRow(inspection, occurrence.SourceOrRowIdentity)
		if len(rowRuns) == 0 {
			return "", false
		}
		var cellRuns = contractPDFConvertedCellRuns(rowRuns)
		if len(cellRuns) == 0 {
			return "", true
		}
		return normalizeAcceptanceConvertedCell(strings.Join(contractPDFRunTexts(cellRuns), " ")), true
	case testutil.ReportPresentationPopulationConvertedEntry:
		var rowRuns = renderingAcceptancePDFConversionRow(inspection, occurrence.SourceOrRowIdentity)
		if len(rowRuns) == 0 {
			return "", false
		}
		var expected, err = renderingAcceptanceExpectedValue(acceptanceCase, report, occurrence)
		if err != nil {
			return "", false
		}
		var cellText = normalizeAcceptanceConvertedCell(strings.Join(contractPDFRunTexts(contractPDFConvertedCellRuns(rowRuns)), " "))
		if strings.Contains(cellText, expected) {
			return expected, true
		}
		return "", false
	case testutil.ReportPresentationPopulationClassifiedCurrency, testutil.ReportPresentationPopulationUnclassified, testutil.ReportPresentationPopulationBoolean:
		var cells, found = renderingAcceptancePDFAuditCells(inspection, occurrence.SourceOrRowIdentity)
		if !found {
			return "", false
		}
		if occurrence.Population == testutil.ReportPresentationPopulationBoolean {
			return cells[11], true
		}
		return cells[7], true
	case testutil.ReportPresentationPopulationQuantity:
		return renderingAcceptancePDFQuantityValue(inspection, occurrence)
	case testutil.ReportPresentationPopulationVisibleFinancial:
		if strings.HasPrefix(occurrence.FieldName, "original_") || strings.HasPrefix(occurrence.FieldName, "converted_") {
			var rowRuns = renderingAcceptancePDFConversionRow(inspection, occurrence.SourceOrRowIdentity)
			if len(rowRuns) == 0 {
				return "", false
			}
			var amount, ok = renderingAcceptanceConversionAmount(report, occurrence.SourceOrRowIdentity, occurrence.AmountKind)
			if !ok {
				return "", false
			}
			if !strings.Contains(normalizeAcceptanceConvertedCell(renderingAcceptancePDFCell(rowRuns, 465, 610, false)), renderingAcceptanceConvertedEntryText(amount)) {
				return "", false
			}
			if strings.HasPrefix(occurrence.FieldName, "original_") {
				var value, err = presentation.FormatFinancialValue(amount.OriginalAmount)
				if err != nil {
					return canonicalAcceptanceDecimal(amount.OriginalAmount), true
				}
				return value, true
			}
			var value, err = presentation.FormatFinancialValue(amount.ConvertedAmount)
			if err != nil {
				return canonicalAcceptanceDecimal(amount.ConvertedAmount), true
			}
			return value, true
		}
		if occurrence.Section == "gains_and_losses_summary" {
			var rowRuns, found = renderingAcceptancePDFSummaryRowRuns(inspection, occurrence.FieldName == "per_asset_net_gain_or_loss")
			if !found {
				return "", false
			}
			var bounds = renderingAcceptancePDFTableColumnBounds([]float64{220, 150, 150}, 1)
			return renderingAcceptancePDFCell(rowRuns, bounds[0], bounds[1], true), true
		}
		if occurrence.Section == "position" {
			var heading = "Opening Position"
			if occurrence.SourceOrRowIdentity == "closing-position" {
				heading = "Closing Position"
			}
			if occurrence.SourceOrRowIdentity == "historical-position" {
				heading = "Historical Position"
			}
			return renderingAcceptancePDFPositionValue(inspection, occurrence.AssetIdentity, heading, "Cost Basis")
		}
		if occurrence.Section == "in_year_activity" {
			var rowRuns, found = renderingAcceptancePDFMainSourceRowInRange(inspection, occurrence.SourceOrRowIdentity, 100, 170)
			if !found {
				return "", false
			}
			var columns = map[string]int{"unit_price": 4, "gross_value": 5, "fee_amount": 6, "basis_after_row": 8}
			column, ok := columns[occurrence.FieldName]
			if !ok {
				return "", false
			}
			var bounds = renderingAcceptancePDFTableColumnBounds([]float64{52, 45, 42, 40, 40, 38, 34, 42, 46, 42, 42, 52, 50}, column)
			return renderingAcceptancePDFCell(rowRuns, bounds[0], bounds[1], true), true
		}
		if occurrence.Section == "liquidation_calculations" {
			var rowRuns, found = renderingAcceptancePDFMainSourceRowInRange(inspection, occurrence.SourceOrRowIdentity, 140, 260)
			if !found {
				return "", false
			}
			var columns = map[string]int{"allocated_basis": 3, "net_proceeds": 4, "gain_or_loss": 5}
			column, ok := columns[occurrence.FieldName]
			if !ok {
				return "", false
			}
			var bounds = renderingAcceptancePDFTableColumnBounds([]float64{72, 66, 76, 74, 72, 70, 88}, column)
			return renderingAcceptancePDFCell(rowRuns, bounds[0], bounds[1], true), true
		}
		if occurrence.Section == "detailed_per_asset_audit" {
			var cells, found = renderingAcceptancePDFAuditCells(inspection, occurrence.SourceOrRowIdentity)
			if !found {
				return "", false
			}
			var columns = map[string]int{"unit_price": 4, "gross_value": 5, "fee_amount": 6, "basis_after_activity": 10, "allocated_basis": 12, "net_proceeds": 13, "gain_or_loss": 14}
			column, ok := columns[occurrence.FieldName]
			if !ok {
				return "", false
			}
			return strings.ReplaceAll(cells[column], " ", ""), true
		}
	}
	return "", false
}

// normalizeAcceptanceConvertedCell makes Markdown and PDF controlled line
// boundaries comparable without removing semantic punctuation.
// Authored by: OpenCode
func normalizeAcceptanceConvertedCell(value string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(strings.ReplaceAll(value, "<br>", " "), ";", "")), " ")
}

// renderingAcceptancePDFAuditCells returns all fixed Annex columns for one
// detailed source row, including blank cells.
// Authored by: OpenCode
func renderingAcceptancePDFAuditCells(inspection testutil.GeneratedPDF, sourceID string) ([]string, bool) {
	var firstPage int
	for _, run := range inspection.TextRuns {
		if run.Text == "Annex 1 - Audit" {
			firstPage = run.Page
			break
		}
	}
	var sourceRuns, found = runtimeflow.FindAnnexPDFSourceRuns(inspection.TextRuns, firstPage, sourceID)
	if !found {
		return nil, false
	}
	var cells = runtimeflow.AnnexPDFSemanticCells(runtimeflow.AnnexPDFRowRuns(inspection.TextRuns, sourceRuns))
	return cells, len(cells) == runtimeflow.AnnexPDFColumnCount
}

// renderingAcceptancePDFQuantityValue maps every quantity occurrence to its
// field-local PDF value.
// Authored by: OpenCode
func renderingAcceptancePDFQuantityValue(inspection testutil.GeneratedPDF, occurrence testutil.ReportPresentationOccurrenceKey) (string, bool) {
	switch occurrence.FieldName {
	case "opening_quantity":
		return renderingAcceptancePDFPositionValue(inspection, occurrence.AssetIdentity, "Opening Position", "Quantity")
	case "closing_quantity":
		return renderingAcceptancePDFPositionValue(inspection, occurrence.AssetIdentity, "Closing Position", "Quantity")
	case "historical_position_quantity":
		return renderingAcceptancePDFPositionValue(inspection, occurrence.AssetIdentity, "Historical Position", "Quantity")
	case "activity_quantity", "quantity_after_row", "disposed_quantity":
		var rowRuns, found = renderingAcceptancePDFMainSourceRow(inspection, occurrence.SourceOrRowIdentity)
		var widths = []float64{52, 45, 42, 40, 40, 38, 34, 42, 46, 42, 42, 52, 50}
		var column = map[string]int{"activity_quantity": 3, "quantity_after_row": 7}[occurrence.FieldName]
		if occurrence.FieldName == "disposed_quantity" {
			rowRuns, found = renderingAcceptancePDFMainSourceRowInRange(inspection, occurrence.SourceOrRowIdentity, 140, 260)
			widths = []float64{72, 66, 76, 74, 72, 70, 88}
			column = 2
		}
		if !found {
			return "", false
		}
		var bounds = renderingAcceptancePDFTableColumnBounds(widths, column)
		return renderingAcceptancePDFCell(rowRuns, bounds[0], bounds[1], true), true
	case "audit_quantity", "quantity_after_activity":
		var cells, found = renderingAcceptancePDFAuditCells(inspection, occurrence.SourceOrRowIdentity)
		if !found {
			return "", false
		}
		if occurrence.FieldName == "audit_quantity" {
			return strings.ReplaceAll(cells[3], " ", ""), true
		}
		return strings.ReplaceAll(cells[9], " ", ""), true
	}
	return "", false
}

// renderingAcceptancePDFPositionValue locates a key/value within the selected
// asset detail block by heading and label coordinates.
// Authored by: OpenCode
func renderingAcceptancePDFPositionValue(inspection testutil.GeneratedPDF, asset string, heading string, label string) (string, bool) {
	var inAsset, inHeading bool
	for _, run := range inspection.TextRuns {
		if run.Text == "Asset Detail: "+renderingAcceptanceAssetLabel(asset) {
			inAsset = true
			inHeading = false
		}
		if !inAsset {
			continue
		}
		if run.Text == heading {
			inHeading = true
			continue
		}
		if strings.HasPrefix(run.Text, "Asset Detail: ") && run.Text != "Asset Detail: "+renderingAcceptanceAssetLabel(asset) {
			inAsset = false
			inHeading = false
		}
		if inHeading && run.Text == label+":" {
			for _, value := range inspection.TextRuns {
				if value.Page == run.Page && value.X >= 150 && value.X <= 260 && value.Y > run.Y-1 && value.Y < run.Y+12 {
					return strings.TrimSpace(value.Text), true
				}
			}
		}
	}
	return "", false
}

// renderingAcceptancePDFConversionRow expands the local conversion row enough
// to include width-wrapped fragments of large financial entries.
// Authored by: OpenCode
func renderingAcceptancePDFConversionRow(inspection testutil.GeneratedPDF, sourceID string) []testutil.PDFTextRun {
	var normalizedSource = strings.ReplaceAll(strings.ToLower(sourceID), " ", "")
	var conversionPage int
	var conversionY float64
	for _, run := range inspection.TextRuns {
		if run.Text == "Currency Conversion Audit" {
			conversionPage = run.Page
			conversionY = run.Y
			break
		}
	}
	for _, sourceRun := range inspection.TextRuns {
		if sourceRun.Page < conversionPage || (sourceRun.Page == conversionPage && sourceRun.Y >= conversionY) || sourceRun.X < 95 || sourceRun.X > 180 || strings.ReplaceAll(strings.ToLower(sourceRun.Text), " ", "") != normalizedSource {
			continue
		}
		var row []testutil.PDFTextRun
		for _, run := range inspection.TextRuns {
			if run.Page == sourceRun.Page && run.Y >= sourceRun.Y-35 && run.Y <= sourceRun.Y+35 {
				row = append(row, run)
			}
		}
		return row
	}
	return nil
}

// renderingAcceptancePDFSummaryRowRuns isolates one summary row by its first
// semantic cell and keeps the amount column local.
// Authored by: OpenCode
func renderingAcceptancePDFSummaryRowRuns(inspection testutil.GeneratedPDF, assetRow bool) ([]testutil.PDFTextRun, bool) {
	var target = "Overall Yearly Net Total"
	if assetRow {
		target = "BTC"
	}
	var summaryPage, summaryHeaderY float64
	var foundSummary bool
	for _, run := range inspection.TextRuns {
		if run.Text == "Gains-And-Losses Summary Table" {
			summaryPage = float64(run.Page)
			summaryHeaderY = run.Y
			foundSummary = true
			break
		}
	}
	if !foundSummary {
		return nil, false
	}
	for _, run := range inspection.TextRuns {
		if run.Text != target || run.X > 200 || float64(run.Page) != summaryPage || run.Y >= summaryHeaderY {
			continue
		}
		var row []testutil.PDFTextRun
		for _, candidate := range inspection.TextRuns {
			if candidate.Page == run.Page && candidate.Y >= run.Y-17 && candidate.Y <= run.Y+17 {
				row = append(row, candidate)
			}
		}
		return row, true
	}
	return nil, false
}

// renderingAcceptancePDFMainSourceRow isolates one main-report table row by
// its Source ID column, including wrapped source fragments.
// Authored by: OpenCode
func renderingAcceptancePDFMainSourceRow(inspection testutil.GeneratedPDF, sourceID string) ([]testutil.PDFTextRun, bool) {
	return renderingAcceptancePDFMainSourceRowInRange(inspection, sourceID, 100, 170)
}

// renderingAcceptancePDFMainSourceRowInRange isolates one repeated main table
// row using the source-ID column bounds of its owning table.
// Authored by: OpenCode
func renderingAcceptancePDFMainSourceRowInRange(inspection testutil.GeneratedPDF, sourceID string, minimumX float64, maximumX float64) ([]testutil.PDFTextRun, bool) {
	var normalizedSource = strings.ReplaceAll(strings.ToLower(sourceID), " ", "")
	for _, run := range inspection.TextRuns {
		if run.X < minimumX || run.X > maximumX || strings.ReplaceAll(strings.ToLower(run.Text), " ", "") == "" {
			continue
		}
		var sourceText string
		for _, candidate := range inspection.TextRuns {
			if candidate.Page == run.Page && candidate.X >= minimumX && candidate.X <= maximumX && candidate.Y >= run.Y-1 && candidate.Y <= run.Y+1 {
				sourceText += strings.ReplaceAll(candidate.Text, " ", "")
			}
		}
		if !strings.Contains(strings.ToLower(sourceText), normalizedSource) {
			continue
		}
		var row []testutil.PDFTextRun
		for _, candidate := range inspection.TextRuns {
			if candidate.Page == run.Page && candidate.Y >= run.Y-17 && candidate.Y <= run.Y+17 {
				row = append(row, candidate)
			}
		}
		return row, true
	}
	return nil, false
}

// renderingAcceptancePDFCell joins one row-local column. Numeric columns may
// be right-aligned or split across physical PDF runs.
// Authored by: OpenCode
func renderingAcceptancePDFCell(runs []testutil.PDFTextRun, minimumX float64, maximumX float64, compact bool) string {
	var cellRuns []testutil.PDFTextRun
	for _, run := range runs {
		if run.X >= minimumX && run.X < maximumX {
			cellRuns = append(cellRuns, run)
		}
	}
	sort.SliceStable(cellRuns, func(left int, right int) bool {
		return cellRuns[left].Y > cellRuns[right].Y
	})
	var values = make([]string, 0, len(cellRuns))
	for _, run := range cellRuns {
		values = append(values, run.Text)
	}
	var value = strings.Join(values, " ")
	if compact {
		return strings.Join(strings.Fields(value), "")
	}
	return strings.Join(strings.Fields(value), " ")
}

// renderingAcceptancePDFTableColumnBounds scales one declared table column to
// the production printable-width geometry used by gopdf.
// Authored by: OpenCode
func renderingAcceptancePDFTableColumnBounds(widths []float64, target int) [2]float64 {
	const tableStartX = 36.0
	const tableWidth = 770.0
	var totalWidth float64
	for _, width := range widths {
		totalWidth += width
	}
	var sourceStart float64
	for column := 0; column < target; column++ {
		sourceStart += widths[column]
	}
	return [2]float64{
		tableStartX + sourceStart*tableWidth/totalWidth,
		tableStartX + (sourceStart+widths[target])*tableWidth/totalWidth,
	}
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
	report.DetailSections = append(report.DetailSections, reportmodel.AssetDetailSection{
		AssetIdentityKey:    "asset-historical",
		DisplayLabel:        "HIST",
		ClosingQuantity:     mustContractDecimal("7"),
		ClosingCostBasis:    mustContractDecimal("70"),
		CalculationCurrency: report.ReportCalculationCurrency,
	})

	switch acceptanceCase.Kind {
	case testutil.ReportPresentationCaseKindFinancial:
		applyRenderingAcceptanceFinancialControl(&report, acceptanceCase)
	case testutil.ReportPresentationCaseKindQuantity:
		var value = mustContractDecimal(acceptanceCase.ExactValue)
		for sectionIndex := range report.DetailSections {
			section := &report.DetailSections[sectionIndex]
			if section.AssetIdentityKey == "asset-historical" {
				section.ClosingQuantity = value
				continue
			}
			section.OpeningQuantity = value
			section.ClosingQuantity = value
			for rowIndex := range section.ActivityRows {
				if value.Sign() > 0 {
					section.ActivityRows[rowIndex].Quantity = value
				}
				section.ActivityRows[rowIndex].QuantityAfterRow = value
			}
			for liquidationIndex := range section.LiquidationSummaries {
				if value.Sign() > 0 {
					section.LiquidationSummaries[liquidationIndex].DisposedQuantity = value
				}
			}
		}
		for sectionIndex := range report.AuditAnnex.PerAssetAuditSections {
			for entryIndex := range report.AuditAnnex.PerAssetAuditSections[sectionIndex].Entries {
				entry := &report.AuditAnnex.PerAssetAuditSections[sectionIndex].Entries[entryIndex]
				if value.Sign() > 0 {
					entry.Quantity = value
				}
				entry.QuantityAfterActivity = value
			}
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
		for index := range report.DetailSections {
			if report.DetailSections[index].AssetIdentityKey == "asset-historical" {
				report.DetailSections[index].ClosingCostBasis = exact
			}
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
			for rowIndex := range report.DetailSections[index].ActivityRows {
				report.DetailSections[index].ActivityRows[rowIndex].UnitPrice = nil
				report.DetailSections[index].ActivityRows[rowIndex].GrossValue = nil
				report.DetailSections[index].ActivityRows[rowIndex].FeeAmount = nil
			}
		}
	case "liquidation-allocated-basis", "liquidation-net-proceeds-gain-or-loss":
		return
	case "audit-activity", "audit-allocated-basis", "audit-net-proceeds-gain-or-loss":
		for index := range report.AuditAnnex.PerAssetAuditSections {
			for entryIndex := range report.AuditAnnex.PerAssetAuditSections[index].Entries {
				var entry = &report.AuditAnnex.PerAssetAuditSections[index].Entries[entryIndex]
				entry.UnitPrice = nil
				entry.GrossValue = nil
				entry.FeeAmount = nil
				entry.AllocatedBasis = nil
				entry.NetLiquidationProceeds = nil
				entry.GainOrLoss = nil
			}
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
	if len(manifest.Cases) != 146 || manifest.Counters.CaseCount != 146 {
		t.Fatalf("closed acceptance case population A = %d/%d, want 146/146", len(manifest.Cases), manifest.Counters.CaseCount)
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
		CaseCount: 146,
		Populations: map[testutil.ReportPresentationPopulation]int{
			testutil.ReportPresentationPopulationWarning:            292,
			testutil.ReportPresentationPopulationVisibleFinancial:   664,
			testutil.ReportPresentationPopulationModelIntegrity:     292,
			testutil.ReportPresentationPopulationQuantity:           80,
			testutil.ReportPresentationPopulationBoolean:            16,
			testutil.ReportPresentationPopulationClassifiedCurrency: 2,
			testutil.ReportPresentationPopulationUnclassified:       4,
			testutil.ReportPresentationPopulationConversionRow:      16,
			testutil.ReportPresentationPopulationParity:             596,
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
	var IDs = make(map[string]struct{}, 146)
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
		{id: "liquidation-allocated-basis"},
		{id: "liquidation-net-proceeds-gain-or-loss", signed: true},
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

// assertStaticCalculationRegressionBaseline validates R without executing its
// direct owner packages from the acceptance contract.
// Authored by: OpenCode
func assertStaticCalculationRegressionBaseline(t *testing.T) {
	t.Helper()
	var root = calculationRegressionRepositoryRoot(t)
	var baseline, err = loadCalculationRegressionBaseline(root)
	if err != nil {
		t.Fatal(err)
	}
	var numerator, mismatches = validateCalculationRegressionBaseline(root, baseline)
	t.Logf("R=%d/%d", numerator, len(baseline.cases))
	if len(mismatches) != 0 {
		sort.Strings(mismatches)
		t.Fatalf("static calculation regression validation failed (R=%d/%d):\n- %s", numerator, len(baseline.cases), strings.Join(mismatches, "\n- "))
	}
}

// renderingAcceptanceOccurrenceKey serializes the complete semantic identity,
// preventing repeated field text from becoming an occurrence denominator.
// Authored by: OpenCode
func renderingAcceptanceOccurrenceKey(occurrence testutil.ReportPresentationOccurrenceKey) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%d", occurrence.Population, occurrence.CaseID, occurrence.Format, occurrence.DocumentRole, occurrence.Section, occurrence.AssetIdentity, occurrence.SourceOrRowIdentity, occurrence.FieldName, occurrence.AmountKind, occurrence.AmountOrdinal)
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

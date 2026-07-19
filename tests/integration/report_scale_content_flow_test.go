// Package integration verifies deterministic report content at the named
// 10,000-activity scale without assigning document assertions to performance
// timing tests.
//
// Authored by: OpenCode
package integration

import (
	"context"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	runtimeapp "github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil/runtimeflow"
)

// TestReportScaleContentFlow verifies every non-USD conversion row and its
// three controlled entries in both generated formats. It also verifies that
// the PDF Annex continues with repeated context and headers while all
// searchable conversion content remains inside the printable A4 bounds.
// Authored by: OpenCode
func TestReportScaleContentFlow(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = runtimeflow.InstallOpenCommandRecorder(t, 0)
	var fixture = runtimeflow.LargeReportFixture(t)
	var harness = runtimeflow.NewRuntimeBackedFlowHarnessWithCurrencyRateService(t, t.TempDir(), runtimeflow.MustCloudSetupConfig(t), false, runtimeflow.DeterministicCurrencyRates{})
	var token = "scale-content-token"
	var expectedSources = scaleExpectedConversionSources(fixture.ProtectedActivityCache.Activities)

	if fixture.ActivityCount != 10000 {
		t.Fatalf("expected named fixture to contain 10000 activities, got %d", fixture.ActivityCount)
	}
	if len(expectedSources) != 6666 {
		t.Fatalf("expected named fixture to contain 6666 non-USD activities, got %d", len(expectedSources))
	}
	runtimeflow.SeedProtectedSnapshot(t, harness, token, fixture.ProtectedActivityCache)
	var unlockResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !unlockResult.ProtectedData.HasReadableSnapshot {
		t.Fatalf("expected readable scale-content snapshot, got %#v", unlockResult)
	}

	var requestedAt = time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC)
	var markdownRequest, err = reportmodel.NewReportRequest(fixture.ReportYear, reportmodel.CostBasisMethodHIFO, reportmodel.ReportBaseCurrencyUSD, reportmodel.ReportOutputFormatMarkdown, requestedAt)
	if err != nil {
		t.Fatalf("new scale-content Markdown request: %v", err)
	}
	var markdownOutcome = harness.App.ReportService.Generate(context.Background(), runtimeapp.ReportGenerationRequest{Request: markdownRequest})
	if !markdownOutcome.Success {
		t.Fatalf("expected scale-content Markdown generation success, got %#v", markdownOutcome)
	}
	var markdownFiles = runtimeflow.ReportOutputPaths(t, reportIO.DocumentsDir, reportmodel.ReportOutputFormatMarkdown)
	var _, annexPath = runtimeflow.MarkdownBundlePaths(t, markdownFiles)
	// #nosec G304 -- the Markdown path is created in the test-owned Documents fixture.
	var rawMarkdown, readErr = os.ReadFile(annexPath)
	if readErr != nil {
		t.Fatalf("read scale-content Markdown Annex %q: %v", annexPath, readErr)
	}
	var markdownRows = assertScaleMarkdownRows(t, string(rawMarkdown), expectedSources)

	var pdfRequest reportmodel.ReportRequest
	pdfRequest, err = reportmodel.NewReportRequest(fixture.ReportYear, reportmodel.CostBasisMethodHIFO, reportmodel.ReportBaseCurrencyUSD, reportmodel.ReportOutputFormatPDF, requestedAt)
	if err != nil {
		t.Fatalf("new scale-content PDF request: %v", err)
	}
	var pdfOutcome = harness.App.ReportService.Generate(context.Background(), runtimeapp.ReportGenerationRequest{Request: pdfRequest})
	if !pdfOutcome.Success {
		t.Fatalf("expected scale-content PDF generation success, got %#v", pdfOutcome)
	}
	var pdfFiles = runtimeflow.ReportOutputPaths(t, reportIO.DocumentsDir, reportmodel.ReportOutputFormatPDF)
	// #nosec G304 -- the PDF path is created in the test-owned Documents fixture.
	var rawPDF []byte
	rawPDF, readErr = os.ReadFile(pdfFiles[0])
	if readErr != nil {
		t.Fatalf("read scale-content PDF %q: %v", pdfFiles[0], readErr)
	}
	var inspection, inspectErr = testutil.InspectGeneratedPDF(rawPDF)
	if inspectErr != nil {
		t.Fatalf("inspect scale-content PDF: %v", inspectErr)
	}
	runtimeflow.AssertLandscapeA4PDF(t, inspection)
	assertScalePDFRows(t, inspection, expectedSources, markdownRows)

	var openRequests = runtimeflow.ReadOpenCommandRequests(t, openLogPath)
	if len(openRequests) != 2 {
		t.Fatalf("expected one opener request per selected format, got %#v", openRequests)
	}
	runtimeflow.AssertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// scaleExpectedConversionSources returns the exact source-ID set for the
// non-USD activities that must produce conversion-audit rows.
// Authored by: OpenCode
func scaleExpectedConversionSources(activities []syncmodel.ActivityRecord) map[string]struct{} {
	var sources = make(map[string]struct{})
	for _, activity := range activities {
		if activity.OrderCurrency != "USD" {
			sources[activity.SourceID] = struct{}{}
		}
	}
	return sources
}

// scaleMarkdownRows stores the three visible entries for one conversion row.
// Authored by: OpenCode
type scaleMarkdownRows map[string][]string

// assertScaleMarkdownRows verifies controlled boundaries and exact row
// cardinality in the generated Markdown Annex.
// Authored by: OpenCode
func assertScaleMarkdownRows(t *testing.T, content string, expectedSources map[string]struct{}) scaleMarkdownRows {
	t.Helper()
	var sections = strings.Split(content, "## Currency Conversion Audit")
	if len(sections) != 2 {
		t.Fatalf("expected one Currency Conversion Audit section, got %d", len(sections)-1)
	}
	var rows = make(scaleMarkdownRows, len(expectedSources))
	for _, line := range strings.Split(sections[1], "\n") {
		if !strings.HasPrefix(line, "|") || strings.HasPrefix(line, "|------") {
			continue
		}
		var cells = strings.Split(line, "|")
		if len(cells) != 11 {
			t.Fatalf("expected nine Markdown conversion cells, got %d in %q", len(cells)-2, line)
		}
		var sourceID = strings.TrimSpace(cells[2])
		if sourceID == "Source ID" {
			continue
		}
		if _, expected := expectedSources[sourceID]; !expected {
			t.Fatalf("unexpected or duplicated Markdown conversion source %q", sourceID)
		}
		if _, duplicate := rows[sourceID]; duplicate {
			t.Fatalf("duplicated Markdown conversion source %q", sourceID)
		}
		var entries = strings.Split(strings.TrimSpace(cells[7]), ";<br>")
		if len(entries) != 3 || strings.Contains(cells[7], "<br><br>") || strings.HasSuffix(strings.TrimSpace(cells[7]), ";") {
			t.Fatalf("Markdown conversion row %q does not have three controlled entries: %q", sourceID, cells[7])
		}
		assertScaleEntryLabelsAndSpacing(t, sourceID, entries)
		rows[sourceID] = entries
	}
	if len(rows) != len(expectedSources) {
		t.Fatalf("Markdown conversion row count = %d, want %d", len(rows), len(expectedSources))
	}
	for sourceID := range expectedSources {
		if _, present := rows[sourceID]; !present {
			t.Fatalf("Markdown conversion source %q was omitted", sourceID)
		}
	}
	return rows
}

// assertScaleEntryLabelsAndSpacing verifies the fixed three-entry conversion
// sequence without relying on a delimiter-stripped aggregate string.
// Authored by: OpenCode
func assertScaleEntryLabelsAndSpacing(t *testing.T, sourceID string, entries []string) {
	t.Helper()
	var labels = []string{"unit_price", "gross_value", "fee_amount"}
	for index, entry := range entries {
		if !strings.HasPrefix(entry, labels[index]+": ") || strings.Count(entry, ": ") != 1 || strings.Count(entry, " -> ") != 1 {
			t.Fatalf("Markdown conversion source %q entry %d has uncontrolled syntax: %q", sourceID, index, entry)
		}
		if strings.Contains(entry, "<br>") || strings.Contains(entry, ";") {
			t.Fatalf("Markdown conversion source %q entry %d contains an uncontrolled boundary: %q", sourceID, index, entry)
		}
	}
}

// assertScalePDFRows verifies the conversion section's source rows, logical
// entries, continuation context, searchable text, and printable coordinates.
// Authored by: OpenCode
func assertScalePDFRows(t *testing.T, inspection testutil.GeneratedPDF, expectedSources map[string]struct{}, markdownRows scaleMarkdownRows) {
	t.Helper()
	var conversionPage, conversionY = scalePDFConversionHeading(t, inspection)
	var sourceRuns = make(map[string][]testutil.PDFTextRun, len(expectedSources))
	var sourceOccurrences = make(map[string]int, len(expectedSources))
	var entryCounts = make(map[string]int)
	var conversionPages = make(map[int]struct{})
	var conversionRuns []testutil.PDFTextRun
	for _, run := range inspection.TextRuns {
		if !scalePDFRunInConversionSection(run, conversionPage, conversionY) {
			continue
		}
		if run.X < 36 || run.X > 806 || run.Y < 36 || run.Y > 559 {
			t.Fatalf("conversion text run is outside printable A4 bounds: %#v", run)
		}
		conversionRuns = append(conversionRuns, run)
		conversionPages[run.Page] = struct{}{}
		for _, label := range []string{"unit_price:", "gross_value:", "fee_amount:"} {
			if strings.Contains(run.Text, label) {
				entryCounts[label]++
			}
		}
	}
	var unexpectedSources = collectScalePDFSourceGroups(sourceRuns, sourceOccurrences, conversionRuns, expectedSources)
	if len(conversionPages) < 2 {
		t.Fatalf("expected multi-page PDF conversion Annex, got pages %#v", conversionPages)
	}
	if len(unexpectedSources) != 0 {
		t.Fatalf("PDF conversion contained unexpected source rows: %#v", unexpectedSources)
	}
	if len(sourceRuns) != len(expectedSources) {
		t.Fatalf("PDF conversion source row count = %d, want %d", len(sourceRuns), len(expectedSources))
	}
	var rowEntryCounts = scalePDFRowEntryCounts(t, sourceRuns, conversionRuns)
	var searchableText = compactScalePDFText(inspection.SearchableText)
	for sourceID := range expectedSources {
		if sourceOccurrences[sourceID] != 1 {
			t.Fatalf("PDF conversion source %q occurrence count = %d, want exactly one", sourceID, sourceOccurrences[sourceID])
		}
		if !strings.Contains(searchableText, compactScalePDFText(sourceID)) {
			t.Fatalf("PDF conversion source %q is not searchable", sourceID)
		}
		for _, entry := range markdownRows[sourceID] {
			if !strings.Contains(searchableText, compactScalePDFText(entry)) {
				t.Fatalf("PDF conversion entry %q for source %q is not searchable", entry, sourceID)
			}
		}
		for _, label := range []string{"unit_price:", "gross_value:", "fee_amount:"} {
			if rowEntryCounts[sourceID][label] != 1 {
				t.Fatalf("PDF conversion source %q entry %s count = %d, want one", sourceID, label, rowEntryCounts[sourceID][label])
			}
		}
	}
	for _, label := range []string{"unit_price:", "gross_value:", "fee_amount:"} {
		if entryCounts[label] != len(expectedSources) {
			t.Fatalf("PDF conversion entry %s count = %d, want %d", label, entryCounts[label], len(expectedSources))
		}
	}
	var repeatedHeaders = 0
	var repeatedContinuation = 0
	for _, run := range conversionRuns {
		if run.Text == "Source ID" {
			repeatedHeaders++
		}
		if run.Text == "Currency Conversion Audit Table (continued)" {
			repeatedContinuation++
		}
	}
	if repeatedHeaders < 2 || repeatedContinuation == 0 {
		t.Fatalf("expected repeated conversion headers and continuation context, headers=%d continuation=%d", repeatedHeaders, repeatedContinuation)
	}
	assertScalePDFRepresentativeRows(t, inspection, sourceRuns, markdownRows)
}

// scalePDFRowEntryCounts associates every conversion entry run with its
// nearest source row on the same page, proving each source row has one of each
// controlled entry rather than only proving a document-wide total.
// Authored by: OpenCode
func scalePDFRowEntryCounts(t *testing.T, sourceRuns map[string][]testutil.PDFTextRun, conversionRuns []testutil.PDFTextRun) map[string]map[string]int {
	t.Helper()
	type sourceRow struct {
		sourceID string
		y        float64
	}
	var rowsByPage = make(map[int][]sourceRow)
	for sourceID, runs := range sourceRuns {
		var minimumY = runs[0].Y
		var maximumY = runs[0].Y
		for _, run := range runs[1:] {
			minimumY = math.Min(minimumY, run.Y)
			maximumY = math.Max(maximumY, run.Y)
		}
		rowsByPage[runs[0].Page] = append(rowsByPage[runs[0].Page], sourceRow{sourceID: sourceID, y: (minimumY + maximumY) / 2})
	}
	var counts = make(map[string]map[string]int, len(sourceRuns))
	for _, run := range conversionRuns {
		var label string
		for _, candidate := range []string{"unit_price:", "gross_value:", "fee_amount:"} {
			if strings.Contains(run.Text, candidate) {
				label = candidate
				break
			}
		}
		if label == "" {
			continue
		}
		var closest sourceRow
		var closestDistance = math.MaxFloat64
		for _, row := range rowsByPage[run.Page] {
			var distance = math.Abs(row.y - run.Y)
			if distance < closestDistance {
				closest = row
				closestDistance = distance
			}
		}
		if closest.sourceID == "" || closestDistance > 20 {
			t.Fatalf("could not associate PDF conversion entry %#v with a source row", run)
		}
		if counts[closest.sourceID] == nil {
			counts[closest.sourceID] = make(map[string]int)
		}
		counts[closest.sourceID][label]++
	}
	return counts
}

// collectScalePDFSourceGroups reconstructs wrapped source identifiers from
// adjacent same-column PDF text runs before checking exact row cardinality.
// Authored by: OpenCode
func collectScalePDFSourceGroups(sourceRuns map[string][]testutil.PDFTextRun, sourceOccurrences map[string]int, conversionRuns []testutil.PDFTextRun, expectedSources map[string]struct{}) []string {
	var normalizedSources = make(map[string]string, len(expectedSources))
	for sourceID := range expectedSources {
		normalizedSources[runtimeflow.NormalizePDFSourceID(sourceID)] = sourceID
	}
	var sourceX float64
	var foundSourceColumn bool
	for _, run := range conversionRuns {
		if strings.Contains(run.Text, "-performance-") {
			sourceX = run.X
			foundSourceColumn = true
			break
		}
	}
	if !foundSourceColumn {
		return nil
	}
	var sourceColumnRuns []testutil.PDFTextRun
	for _, run := range conversionRuns {
		if math.Abs(run.X-sourceX) <= 0.1 {
			sourceColumnRuns = append(sourceColumnRuns, run)
		}
	}
	var group []testutil.PDFTextRun
	var unexpectedSources []string
	flush := func() {
		if len(group) == 0 {
			return
		}
		var normalized strings.Builder
		for _, run := range group {
			normalized.WriteString(runtimeflow.NormalizePDFSourceID(run.Text))
		}
		var normalizedSourceID = normalized.String()
		if sourceID, expected := normalizedSources[normalizedSourceID]; expected {
			sourceOccurrences[sourceID]++
			if sourceOccurrences[sourceID] == 1 {
				sourceRuns[sourceID] = append(sourceRuns[sourceID], group...)
			}
		} else if normalizedSourceID != "" {
			unexpectedSources = append(unexpectedSources, normalizedSourceID)
		}
		group = nil
	}
	var previous testutil.PDFTextRun
	for _, run := range sourceColumnRuns {
		if len(group) > 0 && (run.Page != previous.Page || previous.Y-run.Y > 16) {
			flush()
		}
		group = append(group, run)
		previous = run
	}
	flush()
	return unexpectedSources
}

// scalePDFConversionHeading locates the conversion table heading and its
// section boundary in ordered PDF text runs.
// Authored by: OpenCode
func scalePDFConversionHeading(t *testing.T, inspection testutil.GeneratedPDF) (int, float64) {
	t.Helper()
	var foundAnnex bool
	for _, run := range inspection.TextRuns {
		if run.Text == "Annex 1 - Audit" {
			foundAnnex = true
		}
		if foundAnnex && run.Text == "Currency Conversion Audit" {
			return run.Page, run.Y
		}
	}
	t.Fatal("expected PDF Currency Conversion Audit heading")
	return 0, 0
}

// scalePDFRunInConversionSection limits checks to text after the conversion
// heading, including all continuation pages.
// Authored by: OpenCode
func scalePDFRunInConversionSection(run testutil.PDFTextRun, page int, headingY float64) bool {
	return run.Page > page || run.Page == page && run.Y < headingY
}

// assertScalePDFRepresentativeRows checks row-local entry order and vertical
// line starts on rows at the beginning, middle, and end of the workload.
// Authored by: OpenCode
func assertScalePDFRepresentativeRows(t *testing.T, inspection testutil.GeneratedPDF, sourceRuns map[string][]testutil.PDFTextRun, markdownRows scaleMarkdownRows) {
	t.Helper()
	var sourceIDs []string
	for sourceID := range sourceRuns {
		sourceIDs = append(sourceIDs, sourceID)
	}
	var selected = []string{sourceIDs[0], sourceIDs[len(sourceIDs)/2], sourceIDs[len(sourceIDs)-1]}
	for _, sourceID := range selected {
		var rowRuns, found = runtimeflow.FindPDFConversionRowRuns(inspection, sourceID)
		if !found {
			t.Fatalf("could not isolate PDF conversion row %q", sourceID)
		}
		var previousY float64
		for index, entry := range markdownRows[sourceID] {
			var label = strings.SplitN(entry, ":", 2)[0]
			var y, ok = runtimeflow.PDFConversionStartY(inspection, sourceID, label, 0)
			if !ok || (index > 0 && y >= previousY) {
				t.Fatalf("PDF conversion row %q entry %q lacks a lower logical line start", sourceID, entry)
			}
			previousY = y
		}
		if len(rowRuns) == 0 {
			t.Fatalf("PDF conversion row %q has no searchable row runs", sourceID)
		}
	}
}

// compactScalePDFText normalizes PDF layout whitespace for searchable-content
// comparisons while retaining all alphanumeric report content.
// Authored by: OpenCode
func compactScalePDFText(value string) string {
	var compact strings.Builder
	for _, character := range strings.ToUpper(value) {
		if character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' {
			compact.WriteRune(character)
		}
	}
	return compact.String()
}

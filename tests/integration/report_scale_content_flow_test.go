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
	"sort"
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
	var convertedCells = scalePDFConvertedCells(t, sourceRuns, conversionRuns)
	var searchableText = compactScalePDFText(inspection.SearchableText)
	var sourceIDs = make([]string, 0, len(expectedSources))
	for sourceID := range expectedSources {
		sourceIDs = append(sourceIDs, sourceID)
	}
	sort.Strings(sourceIDs)
	for _, sourceID := range sourceIDs {
		if sourceOccurrences[sourceID] != 1 {
			t.Fatalf("PDF conversion source %q occurrence count = %d, want exactly one", sourceID, sourceOccurrences[sourceID])
		}
		if !strings.Contains(searchableText, compactScalePDFText(sourceID)) {
			t.Fatalf("PDF conversion source %q is not searchable", sourceID)
		}
		assertScalePDFConvertedCell(t, sourceID, convertedCells[sourceID], markdownRows[sourceID])
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
}

// scalePDFConvertedCells indexes converted-column runs by exact source-row
// boundaries so every cell can be checked without repeated document scans.
// Authored by: OpenCode
func scalePDFConvertedCells(t *testing.T, sourceRuns map[string][]testutil.PDFTextRun, conversionRuns []testutil.PDFTextRun) map[string][]testutil.PDFTextRun {
	t.Helper()
	type sourceRow struct {
		sourceID string
		centerY  float64
	}
	var rowsByPage = make(map[int][]sourceRow)
	for sourceID, runs := range sourceRuns {
		var minimumY = runs[0].Y
		var maximumY = runs[0].Y
		for _, run := range runs[1:] {
			minimumY = math.Min(minimumY, run.Y)
			maximumY = math.Max(maximumY, run.Y)
		}
		rowsByPage[runs[0].Page] = append(rowsByPage[runs[0].Page], sourceRow{sourceID: sourceID, centerY: (minimumY + maximumY) / 2})
	}
	for page := range rowsByPage {
		sort.SliceStable(rowsByPage[page], func(left, right int) bool {
			return rowsByPage[page][left].centerY > rowsByPage[page][right].centerY
		})
	}

	var convertedX float64
	var foundConvertedColumn bool
	for _, run := range conversionRuns {
		if strings.Contains(run.Text, "unit_price:") {
			convertedX = run.X
			foundConvertedColumn = true
			break
		}
	}
	if !foundConvertedColumn {
		t.Fatal("could not locate PDF Converted Amounts column")
	}

	var convertedRunsByPage = make(map[int][]testutil.PDFTextRun)
	for _, run := range conversionRuns {
		if math.Abs(run.X-convertedX) <= 0.01 {
			convertedRunsByPage[run.Page] = append(convertedRunsByPage[run.Page], run)
		}
	}
	var cells = make(map[string][]testutil.PDFTextRun, len(sourceRuns))
	for page, rows := range rowsByPage {
		var runs = convertedRunsByPage[page]
		sort.SliceStable(runs, func(left, right int) bool {
			return runs[left].Y > runs[right].Y
		})
		var rowIndex int
		for _, run := range runs {
			var upperBoundary = rows[0].centerY + 18
			if len(rows) > 1 {
				upperBoundary = rows[0].centerY + (rows[0].centerY-rows[1].centerY)/2
			}
			if run.Y > upperBoundary {
				continue
			}
			for rowIndex+1 < len(rows) && run.Y <= (rows[rowIndex].centerY+rows[rowIndex+1].centerY)/2 {
				rowIndex++
			}
			cells[rows[rowIndex].sourceID] = append(cells[rows[rowIndex].sourceID], run)
		}
	}
	return cells
}

// assertScalePDFConvertedCell reconstructs one row's Converted Amounts cell
// and compares its exact entries, delimiters, order, and logical line starts.
// Authored by: OpenCode
func assertScalePDFConvertedCell(t *testing.T, sourceID string, runs []testutil.PDFTextRun, expectedEntries []string) {
	t.Helper()
	var actualEntries []string
	var startYs []float64
	for _, run := range runs {
		var line = strings.Join(strings.Fields(run.Text), " ")
		if line == "" {
			continue
		}
		var startsEntry bool
		for _, label := range []string{"unit_price: ", "gross_value: ", "fee_amount: "} {
			if strings.HasPrefix(line, label) {
				startsEntry = true
				break
			}
		}
		if startsEntry {
			actualEntries = append(actualEntries, line)
			startYs = append(startYs, run.Y)
			continue
		}
		if len(actualEntries) == 0 {
			t.Fatalf("PDF conversion source %q has unexpected text before its first entry: %q", sourceID, line)
		}
		actualEntries[len(actualEntries)-1] += " " + line
	}
	if len(actualEntries) != len(expectedEntries) {
		t.Fatalf("PDF conversion source %q entries = %#v, want %d exact entries", sourceID, actualEntries, len(expectedEntries))
	}
	for index, expected := range expectedEntries {
		if index < len(expectedEntries)-1 {
			expected += ";"
		}
		if actualEntries[index] != expected {
			t.Fatalf("PDF conversion source %q entry %d = %q, want %q", sourceID, index, actualEntries[index], expected)
		}
		if index > 0 && startYs[index] >= startYs[index-1] {
			t.Fatalf("PDF conversion source %q entry %q starts at Y %.2f after previous Y %.2f; want a lower logical line", sourceID, actualEntries[index], startYs[index], startYs[index-1])
		}
	}
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

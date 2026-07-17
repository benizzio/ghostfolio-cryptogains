// Package integration verifies runtime-backed converted-amount presentation,
// failure isolation, and retry behavior.
// Authored by: OpenCode
package integration

import (
	"context"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	reportcalculate "github.com/benizzio/ghostfolio-cryptogains/internal/report/calculate"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/report/presentation"
	"github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil/runtimeflow"
)

// TestConvertedAmountsRuntimeParityPreservesAUD001 verifies that the same
// runtime-calculated conversion evidence reaches Markdown and PDF with the
// same ordered logical entries and without changing the calculated model.
// Authored by: OpenCode
func TestConvertedAmountsRuntimeParityPreservesAUD001(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = runtimeflow.InstallOpenCommandRecorder(t, 0)
	var fixture = mixedCurrencyConversionProtectedActivityCache(t, 6)
	var harness = runtimeflow.NewRuntimeBackedFlowHarness(t, t.TempDir(), runtimeflow.MustCloudSetupConfig(t), false)
	var token = "converted-amounts-parity-token"

	runtimeflow.SeedProtectedSnapshot(t, harness, token, fixture)
	var unlockResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !unlockResult.ProtectedData.HasReadableSnapshot {
		t.Fatalf("expected readable snapshot for converted-amount parity, got %#v", unlockResult)
	}

	var calculator = reportcalculate.NewCalculator(runtimeflow.DeterministicCurrencyRates{})
	var baseline, err = calculator.Calculate(context.Background(), mustIntegrationReportRequestForFormat(t, 2024, reportmodel.ReportOutputFormatMarkdown), fixture)
	if err != nil {
		t.Fatalf("calculate converted-amount AUD-001 baseline: %v", err)
	}
	if len(baseline.AuditAnnex.ConversionAuditEntries) == 0 {
		t.Fatal("expected converted-amount audit entries in the baseline")
	}

	var rendererProbe = &convertedAmountsRendererProbe{delegate: harness.App.ReportService}
	var observations = make(map[reportmodel.ReportOutputFormat]convertedAmountsOutputObservation)
	for _, outputFormat := range []reportmodel.ReportOutputFormat{
		reportmodel.ReportOutputFormatMarkdown,
		reportmodel.ReportOutputFormatPDF,
	} {
		var request = mustIntegrationReportRequestForFormat(t, 2024, outputFormat)
		var before, calculateErr = calculator.Calculate(context.Background(), request, fixture)
		if calculateErr != nil {
			t.Fatalf("calculate %s converted-amount baseline: %v", outputFormat, calculateErr)
		}

		var outcome = rendererProbe.Generate(context.Background(), runtime.ReportGenerationRequest{Request: request})
		if !outcome.Success {
			t.Fatalf("expected %s converted-amount report success, got %#v", outputFormat, outcome)
		}
		if outcome.OutputFormat != outputFormat {
			t.Fatalf("expected selected %s renderer, got %q", outputFormat, outcome.OutputFormat)
		}

		var after reportmodel.CapitalGainsReport
		after, calculateErr = calculator.Calculate(context.Background(), request, fixture)
		if calculateErr != nil {
			t.Fatalf("calculate %s converted-amount post-render model: %v", outputFormat, calculateErr)
		}
		assertAUD001ReportEqual(t, outputFormat, before, after)
		observations[outputFormat] = readConvertedAmountsOutput(t, reportIO.DocumentsDir, outcome)
	}

	if len(rendererProbe.formats) != 2 || rendererProbe.formats[0] != reportmodel.ReportOutputFormatMarkdown || rendererProbe.formats[1] != reportmodel.ReportOutputFormatPDF {
		t.Fatalf("expected one renderer call per selected format and no alternate format, got %#v", rendererProbe.formats)
	}
	assertConvertedAmountsOutputsMatch(t, baseline, observations[reportmodel.ReportOutputFormatMarkdown], observations[reportmodel.ReportOutputFormatPDF])

	var openerRequests = runtimeflow.ReadOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != 2 {
		t.Fatalf("expected one opener request per successful selected format, got %#v", openerRequests)
	}
	runtimeflow.AssertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// TestConvertedAmountsPDFLayoutFailureLeavesBoundaryAndAllowsRetry verifies
// that an oversized multiline conversion row fails before output and opening,
// then succeeds after the invalid synthetic snapshot is replaced.
// Authored by: OpenCode
func TestConvertedAmountsPDFLayoutFailureLeavesBoundaryAndAllowsRetry(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = runtimeflow.InstallOpenCommandRecorder(t, 0)
	var harness = runtimeflow.NewRuntimeBackedFlowHarness(t, t.TempDir(), runtimeflow.MustCloudSetupConfig(t), false)
	var token = "converted-amounts-retry-token"
	var failureCache = convertedAmountsMultilineLayoutFailureCache(t)
	var failureCandidate = runtimeflow.SeedProtectedSnapshot(t, harness, token, failureCache)
	var unlockResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !unlockResult.ProtectedData.HasReadableSnapshot {
		t.Fatalf("expected readable failure-test snapshot, got %#v", unlockResult)
	}

	var calculator = reportcalculate.NewCalculator(runtimeflow.DeterministicCurrencyRates{})
	var failureRequest = mustIntegrationReportRequestForFormat(t, 2024, reportmodel.ReportOutputFormatPDF)
	var failureBaseline, err = calculator.Calculate(context.Background(), failureRequest, failureCache)
	if err != nil {
		t.Fatalf("calculate multiline failure AUD-001 baseline: %v", err)
	}

	var rendererProbe = &convertedAmountsRendererProbe{delegate: harness.App.ReportService}
	var failed = rendererProbe.Generate(context.Background(), runtime.ReportGenerationRequest{Request: failureRequest})
	if failed.Success {
		t.Fatalf("expected multiline PDF layout failure, got %#v", failed)
	}
	if failed.OutputFormat != "" || failed.OutputFile.Path != "" || len(failed.OutputBundle.Files) != 0 {
		t.Fatalf("expected failed layout attempt to return no successful output, got %#v", failed)
	}
	if !strings.Contains(failed.Message, "No report file was saved") {
		t.Fatalf("expected no-output failure guidance, got %q", failed.Message)
	}
	if !strings.Contains(strings.ToLower(failed.Message), "render") && !strings.Contains(strings.ToLower(failed.Message), "layout") {
		t.Fatalf("expected contextual render/layout failure, got %q", failed.Message)
	}
	if files := runtimeflow.AllMarkdownFiles(t, reportIO.DocumentsDir); len(files) != 0 {
		t.Fatalf("expected no Markdown output after PDF layout failure, got %#v", files)
	}
	if files := mustPDFFiles(t, reportIO.DocumentsDir); len(files) != 0 {
		t.Fatalf("expected no PDF output after PDF layout failure, got %#v", files)
	}
	if openerRequests := runtimeflow.ReadOpenCommandRequests(t, openLogPath); len(openerRequests) != 0 {
		t.Fatalf("expected no opener request after PDF layout failure, got %#v", openerRequests)
	}
	if len(rendererProbe.formats) != 1 || rendererProbe.formats[0] != reportmodel.ReportOutputFormatPDF {
		t.Fatalf("expected one selected PDF renderer call and no alternate format, got %#v", rendererProbe.formats)
	}

	var failureAfter, calculateErr = calculator.Calculate(context.Background(), failureRequest, failureCache)
	if calculateErr != nil {
		t.Fatalf("calculate multiline failure AUD-001 post-render model: %v", calculateErr)
	}
	assertAUD001ReportEqual(t, reportmodel.ReportOutputFormatPDF, failureBaseline, failureAfter)

	var retryCache = mixedCurrencyConversionProtectedActivityCache(t, 6)
	runtimeflow.SeedProtectedSnapshot(t, harness, token, retryCache)
	if err = os.Remove(failureCandidate.Path); err != nil {
		t.Fatalf("remove failed-attempt snapshot fixture: %v", err)
	}
	var retryUnlockResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !retryUnlockResult.ProtectedData.HasReadableSnapshot {
		t.Fatalf("expected readable retry snapshot, got %#v", retryUnlockResult)
	}

	var retryBaseline, retryErr = calculator.Calculate(context.Background(), failureRequest, retryCache)
	if retryErr != nil {
		t.Fatalf("calculate retry AUD-001 baseline: %v", retryErr)
	}
	var retried = rendererProbe.Generate(context.Background(), runtime.ReportGenerationRequest{Request: failureRequest})
	if !retried.Success || retried.OutputFormat != reportmodel.ReportOutputFormatPDF {
		t.Fatalf("expected successful PDF retry, got %#v", retried)
	}
	var retryAfter reportmodel.CapitalGainsReport
	retryAfter, retryErr = calculator.Calculate(context.Background(), failureRequest, retryCache)
	if retryErr != nil {
		t.Fatalf("calculate retry AUD-001 post-render model: %v", retryErr)
	}
	assertAUD001ReportEqual(t, reportmodel.ReportOutputFormatPDF, retryBaseline, retryAfter)
	if files := runtimeflow.AllMarkdownFiles(t, reportIO.DocumentsDir); len(files) != 0 {
		t.Fatalf("expected retry to save only PDF output, got Markdown files %#v", files)
	}
	if files := mustPDFFiles(t, reportIO.DocumentsDir); len(files) != 1 {
		t.Fatalf("expected one PDF after successful retry, got %#v", files)
	}
	if len(rendererProbe.formats) != 2 || rendererProbe.formats[1] != reportmodel.ReportOutputFormatPDF {
		t.Fatalf("expected failed PDF followed by successful PDF without fallback, got %#v", rendererProbe.formats)
	}
	if openerRequests := runtimeflow.ReadOpenCommandRequests(t, openLogPath); len(openerRequests) != 1 {
		t.Fatalf("expected one opener request only for the successful retry, got %#v", openerRequests)
	}
	runtimeflow.AssertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// convertedAmountsRendererProbe records the selected runtime renderer boundary
// while delegating generation to the real application report service.
// Authored by: OpenCode
type convertedAmountsRendererProbe struct {
	delegate runtime.ReportService
	formats  []reportmodel.ReportOutputFormat
}

// Generate records one selected-format runtime render request and delegates it
// to the production report service.
// Authored by: OpenCode
func (probe *convertedAmountsRendererProbe) Generate(ctx context.Context, request runtime.ReportGenerationRequest) runtime.ReportOutcome {
	probe.formats = append(probe.formats, request.Request.OutputFormat)
	return probe.delegate.Generate(ctx, request)
}

// convertedAmountsOutputObservation stores format-neutral output evidence for
// one selected report format.
// Authored by: OpenCode
type convertedAmountsOutputObservation struct {
	markdownCell string
	pdf          testutil.GeneratedPDF
}

// readConvertedAmountsOutput extracts the conversion cell or PDF inspection
// result from one successful runtime output bundle.
// Authored by: OpenCode
func readConvertedAmountsOutput(t *testing.T, documentsDir string, outcome runtime.ReportOutcome) convertedAmountsOutputObservation {
	t.Helper()
	if outcome.OutputFormat == reportmodel.ReportOutputFormatMarkdown {
		var files = runtimeflow.AllMarkdownFiles(t, documentsDir)
		var _, annexPath = markdownBundlePaths(t, files)
		// #nosec G304 -- the Annex path is created in the test-owned Documents fixture.
		var raw, err = os.ReadFile(annexPath)
		if err != nil {
			t.Fatalf("read converted-amount Markdown Annex %q: %v", annexPath, err)
		}
		return convertedAmountsOutputObservation{markdownCell: convertedAmountsMarkdownCell(t, string(raw))}
	}

	if outcome.OutputFormat == reportmodel.ReportOutputFormatPDF {
		// #nosec G304 -- the PDF path is returned by the controlled runtime output fixture.
		var raw, err = os.ReadFile(outcome.OutputBundle.Files[0].Path)
		if err != nil {
			t.Fatalf("read converted-amount PDF %q: %v", outcome.OutputBundle.Files[0].Path, err)
		}
		var inspection, inspectErr = testutil.InspectGeneratedPDF(raw)
		if inspectErr != nil {
			t.Fatalf("inspect converted-amount PDF: %v", inspectErr)
		}
		return convertedAmountsOutputObservation{pdf: inspection}
	}

	t.Fatalf("unsupported converted-amount output format %q", outcome.OutputFormat)
	return convertedAmountsOutputObservation{}
}

// convertedAmountsMarkdownCell returns the Converted Amounts cell from the
// synthetic mixed-currency Annex row.
// Authored by: OpenCode
func convertedAmountsMarkdownCell(t *testing.T, content string) string {
	t.Helper()
	var sections = strings.Split(content, "## Currency Conversion Audit")
	if len(sections) < 2 {
		t.Fatalf("expected Currency Conversion Audit section, got %q", content)
	}
	for _, line := range strings.Split(sections[1], "\n") {
		if !strings.Contains(line, "mixed-") || !strings.HasPrefix(line, "|") {
			continue
		}
		var cells = strings.Split(line, "|")
		if len(cells) < 8 {
			t.Fatalf("expected converted-amount Markdown row columns, got %q", line)
		}
		return strings.TrimSpace(cells[7])
	}
	t.Fatalf("expected a synthetic mixed-currency conversion row, got %q", content)
	return ""
}

// assertConvertedAmountsOutputsMatch compares both output formats against the
// exact calculated converted-entry sequence and requires renderer line starts
// for later entries in PDF text runs.
// Authored by: OpenCode
func assertConvertedAmountsOutputsMatch(t *testing.T, baseline reportmodel.CapitalGainsReport, markdown convertedAmountsOutputObservation, pdf convertedAmountsOutputObservation) {
	t.Helper()
	var checkedRows int
	for _, entry := range baseline.AuditAnnex.ConversionAuditEntries {
		if checkedRows > 0 {
			break
		}
		var expectedEntries = convertedAmountEntryText(t, entry)
		if len(expectedEntries) == 0 {
			continue
		}
		checkedRows++
		var expectedMarkdown = strings.Join(expectedEntries, ";<br>")
		if markdown.markdownCell != expectedMarkdown {
			t.Fatalf("converted amounts for %q = %q, want controlled lines %q", entry.SourceID, markdown.markdownCell, expectedMarkdown)
		}

		var labelOccurrences = make(map[string]int)
		var previousY float64
		for index, expected := range expectedEntries {
			if !pdf.pdf.ContainsSearchableText(expected) {
				t.Fatalf("PDF converted amounts for %q omit %q from searchable text %q", entry.SourceID, expected, pdf.pdf.SearchableText)
			}
			var label = strings.SplitN(expected, ":", 2)[0]
			var rowRuns, rowFound = convertedAmountPDFRowRuns(pdf.pdf, entry.SourceID)
			if !rowFound || !convertedAmountPDFContainsEntry(rowRuns, expected) {
				t.Fatalf("PDF converted amounts for %q omit row-local entry %q", entry.SourceID, expected)
			}
			var occurrence = labelOccurrences[label]
			var y, ok = convertedAmountPDFStartY(pdf.pdf, entry.SourceID, label, occurrence)
			if !ok {
				t.Fatalf("PDF converted amounts for %q omit logical start %q from text runs", entry.SourceID, expected)
			}
			labelOccurrences[label]++
			if index > 0 && y <= previousY {
				t.Fatalf("PDF converted amount %q starts at Y %.2f after previous Y %.2f; want a later logical line", expected, y, previousY)
			}
			previousY = y
		}
	}
	if checkedRows == 0 {
		t.Fatal("expected at least one non-empty converted-amount row")
	}
}

// convertedAmountEntryText formats the baseline converted amounts without
// introducing renderer-specific line delimiters.
// Authored by: OpenCode
func convertedAmountEntryText(t *testing.T, entry reportmodel.ConversionAuditEntry) []string {
	t.Helper()
	var rendered []string
	for _, amount := range entry.Amounts {
		if amount.OriginalAmount.Sign() == 0 && amount.ConvertedAmount.Sign() == 0 {
			continue
		}
		var original, err = presentation.FormatFinancialValue(amount.OriginalAmount)
		if err != nil {
			t.Fatalf("format original converted amount %q: %v", amount.AmountKind, err)
		}
		var converted, convertedErr = presentation.FormatFinancialValue(amount.ConvertedAmount)
		if convertedErr != nil {
			t.Fatalf("format converted amount %q: %v", amount.AmountKind, convertedErr)
		}
		rendered = append(rendered, fmt.Sprintf("%s: %s -> %s", amount.AmountKind, original, converted))
	}
	return rendered
}

// convertedAmountPDFStartY locates one row-local converted-amount logical label
// in semantic coordinate order.
// Authored by: OpenCode
func convertedAmountPDFStartY(inspection testutil.GeneratedPDF, sourceID string, label string, occurrence int) (float64, bool) {
	var rowRuns, found = convertedAmountPDFRowRuns(inspection, sourceID)
	if !found {
		return 0, false
	}
	var cellRuns = convertedAmountPDFCellRuns(rowRuns)
	var matches []testutil.PDFTextRun
	for _, run := range cellRuns {
		if strings.Contains(run.Text, label+":") {
			matches = append(matches, run)
		}
	}
	if occurrence < 0 || occurrence >= len(matches) {
		return 0, false
	}
	return matches[occurrence].Y, true
}

// convertedAmountPDFContainsEntry verifies a complete converted entry within
// the selected row rather than relying on searchable text from another row.
// Authored by: OpenCode
func convertedAmountPDFContainsEntry(rowRuns []testutil.PDFTextRun, expected string) bool {
	var normalizedExpected = strings.Join(strings.Fields(expected), " ")
	for _, run := range convertedAmountPDFCellRuns(rowRuns) {
		var normalizedRun = strings.Join(strings.Fields(strings.ReplaceAll(run.Text, ";", "")), " ")
		if strings.Contains(normalizedRun, normalizedExpected) {
			return true
		}
	}
	return false
}

// convertedAmountPDFRowRuns isolates one conversion row using Annex section
// boundaries, the source-ID X/Y neighborhood, and nearest source baselines.
// Authored by: OpenCode
func convertedAmountPDFRowRuns(inspection testutil.GeneratedPDF, sourceID string) ([]testutil.PDFTextRun, bool) {
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

	var sourceRuns, found = convertedAmountPDFSourceRuns(inspection, sourceID, annexPage, conversionPage, conversionY)
	if !found {
		return nil, false
	}
	var sourceY = convertedAmountPDFSourceCenterY(sourceRuns)
	var sourceYs = convertedAmountPDFSourceRowYs(inspection, sourceRuns[0].Page, sourceRuns[0].X, annexPage, conversionPage, conversionY)
	var neighborhood = convertedAmountPDFRowNeighborhood(sourceY, sourceYs)
	var rowRuns []testutil.PDFTextRun
	for _, run := range inspection.TextRuns {
		if run.Page == sourceRuns[0].Page && math.Abs(run.Y-sourceY) <= neighborhood+0.01 {
			rowRuns = append(rowRuns, run)
		}
	}
	return rowRuns, len(rowRuns) > 0
}

// convertedAmountPDFSourceRuns locates a possibly wrapped source ID in the
// conversion table after the Annex conversion heading.
// Authored by: OpenCode
func convertedAmountPDFSourceRuns(inspection testutil.GeneratedPDF, sourceID string, annexPage int, conversionPage int, conversionY float64) ([]testutil.PDFTextRun, bool) {
	var target = convertedAmountPDFSourceText(sourceID)
	if target == "" {
		return nil, false
	}
	for index, run := range inspection.TextRuns {
		if !convertedAmountPDFRunInSection(run, annexPage, conversionPage, conversionY) || convertedAmountPDFSourceText(run.Text) == "" {
			continue
		}
		var candidate []testutil.PDFTextRun
		var normalized strings.Builder
		for next := index; next < len(inspection.TextRuns); next++ {
			var fragment = inspection.TextRuns[next]
			if !convertedAmountPDFRunInSection(fragment, annexPage, conversionPage, conversionY) || math.Abs(fragment.X-run.X) > 0.01 {
				break
			}
			if len(candidate) > 0 && math.Abs(fragment.Y-candidate[len(candidate)-1].Y) > 16 {
				break
			}
			candidate = append(candidate, fragment)
			normalized.WriteString(convertedAmountPDFSourceText(fragment.Text))
			if strings.Contains(normalized.String(), target) {
				return candidate, true
			}
		}
	}
	return nil, false
}

// convertedAmountPDFSourceRowYs groups source-column text fragments into row
// centers used to reject neighboring conversion rows.
// Authored by: OpenCode
func convertedAmountPDFSourceRowYs(inspection testutil.GeneratedPDF, page int, sourceX float64, annexPage int, conversionPage int, conversionY float64) []float64 {
	var ys []float64
	for _, run := range inspection.TextRuns {
		if run.Page == page && math.Abs(run.X-sourceX) <= 0.01 && convertedAmountPDFRunInSection(run, annexPage, conversionPage, conversionY) {
			ys = append(ys, run.Y)
		}
	}
	sort.Float64s(ys)
	var centers []float64
	for _, y := range ys {
		if len(centers) == 0 || y-centers[len(centers)-1] > 16 {
			centers = append(centers, y)
			continue
		}
		centers[len(centers)-1] = (centers[len(centers)-1] + y) / 2
	}
	return centers
}

// convertedAmountPDFSourceCenterY returns the center of a wrapped source cell.
// Authored by: OpenCode
func convertedAmountPDFSourceCenterY(sourceRuns []testutil.PDFTextRun) float64 {
	var minimumY = sourceRuns[0].Y
	var maximumY = sourceRuns[0].Y
	for _, run := range sourceRuns[1:] {
		minimumY = math.Min(minimumY, run.Y)
		maximumY = math.Max(maximumY, run.Y)
	}
	return (minimumY + maximumY) / 2
}

// convertedAmountPDFRowNeighborhood returns the midpoint to the nearest source
// row, with the known 36-point table row as the single-row fallback.
// Authored by: OpenCode
func convertedAmountPDFRowNeighborhood(sourceY float64, sourceYs []float64) float64 {
	var neighborhood = 18.0
	for _, otherY := range sourceYs {
		if math.Abs(otherY-sourceY) <= 0.01 {
			continue
		}
		neighborhood = math.Min(neighborhood, math.Abs(otherY-sourceY)/2)
	}
	return neighborhood
}

// convertedAmountPDFCellRuns keeps only the converted-amount column and orders
// its physical lines by the expected semantic coordinate order.
// Authored by: OpenCode
func convertedAmountPDFCellRuns(rowRuns []testutil.PDFTextRun) []testutil.PDFTextRun {
	var anchorX float64
	var found bool
	for _, run := range rowRuns {
		if strings.Contains(run.Text, string(reportmodel.ConvertedAmountKindUnitPrice)+":") || strings.Contains(run.Text, string(reportmodel.ConvertedAmountKindGrossValue)+":") || strings.Contains(run.Text, string(reportmodel.ConvertedAmountKindFeeAmount)+":") {
			anchorX = run.X
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	var result []testutil.PDFTextRun
	for _, run := range rowRuns {
		if math.Abs(run.X-anchorX) <= 0.01 {
			result = append(result, run)
		}
	}
	sort.SliceStable(result, func(left int, right int) bool {
		return result[left].Y < result[right].Y
	})
	return result
}

// convertedAmountPDFRunInSection reports whether a run is below the conversion
// heading and inside the Annex page range.
// Authored by: OpenCode
func convertedAmountPDFRunInSection(run testutil.PDFTextRun, annexPage int, conversionPage int, conversionY float64) bool {
	return run.Page >= annexPage && (run.Page > conversionPage || run.Page == conversionPage && run.Y < conversionY)
}

// convertedAmountPDFSourceText removes line whitespace without changing source
// ID punctuation.
// Authored by: OpenCode
func convertedAmountPDFSourceText(value string) string {
	return strings.Join(strings.Fields(value), "")
}

// convertedAmountsMultilineLayoutFailureCache creates a synthetic conversion
// row whose asset cell cannot fit on a fresh PDF page.
// Authored by: OpenCode
func convertedAmountsMultilineLayoutFailureCache(t *testing.T) model.ProtectedActivityCache {
	t.Helper()
	var cache = mixedCurrencyConversionProtectedActivityCache(t, 6)
	var longAssetLabel = strings.Repeat("multiline-layout-word ", 700)
	for index := range cache.Activities {
		cache.Activities[index].AssetName = longAssetLabel
		cache.Activities[index].AssetSymbol = longAssetLabel
	}
	return cache
}

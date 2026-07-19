// Package integration verifies runtime-backed converted-amount presentation,
// failure isolation, and retry behavior.
// Authored by: OpenCode
package integration

import (
	"context"
	"fmt"
	"os"
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
	var fixture = runtimeflow.MixedCurrencyConversionProtectedActivityCache(t, 6)
	var harness = runtimeflow.NewRuntimeBackedFlowHarness(t, t.TempDir(), runtimeflow.MustCloudSetupConfig(t), false)
	var token = "converted-amounts-parity-token"

	runtimeflow.SeedProtectedSnapshot(t, harness, token, fixture)
	var unlockResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !unlockResult.ProtectedData.HasReadableSnapshot {
		t.Fatalf("expected readable snapshot for converted-amount parity, got %#v", unlockResult)
	}

	var calculator = reportcalculate.NewCalculator(runtimeflow.DeterministicCurrencyRates{})
	var baseline, err = calculator.Calculate(context.Background(), runtimeflow.MustIntegrationReportRequestForFormat(t, 2024, reportmodel.ReportOutputFormatMarkdown), fixture)
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
		var request = runtimeflow.MustIntegrationReportRequestForFormat(t, 2024, outputFormat)
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
	var failureRequest = runtimeflow.MustIntegrationReportRequestForFormat(t, 2024, reportmodel.ReportOutputFormatPDF)
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
	var failedMarkdownFiles = runtimeflow.AllMarkdownFiles(t, reportIO.DocumentsDir)
	if len(failedMarkdownFiles) != 0 {
		t.Fatalf("expected no Markdown output after PDF layout failure, got %#v", failedMarkdownFiles)
	}
	var failedPDFFiles = runtimeflow.PDFFiles(t, reportIO.DocumentsDir)
	if len(failedPDFFiles) != 0 {
		t.Fatalf("expected no PDF output after PDF layout failure, got %#v", failedPDFFiles)
	}
	var failedOpenerRequests = runtimeflow.ReadOpenCommandRequests(t, openLogPath)
	if len(failedOpenerRequests) != 0 {
		t.Fatalf("expected no opener request after PDF layout failure, got %#v", failedOpenerRequests)
	}
	if len(rendererProbe.formats) != 1 || rendererProbe.formats[0] != reportmodel.ReportOutputFormatPDF {
		t.Fatalf("expected one selected PDF renderer call and no alternate format, got %#v", rendererProbe.formats)
	}

	var failureAfter, calculateErr = calculator.Calculate(context.Background(), failureRequest, failureCache)
	if calculateErr != nil {
		t.Fatalf("calculate multiline failure AUD-001 post-render model: %v", calculateErr)
	}
	assertAUD001ReportEqual(t, reportmodel.ReportOutputFormatPDF, failureBaseline, failureAfter)

	var retryCache = runtimeflow.MixedCurrencyConversionProtectedActivityCache(t, 6)
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
	var retryMarkdownFiles = runtimeflow.AllMarkdownFiles(t, reportIO.DocumentsDir)
	if len(retryMarkdownFiles) != 0 {
		t.Fatalf("expected retry to save only PDF output, got Markdown files %#v", retryMarkdownFiles)
	}
	var retryPDFFiles = runtimeflow.PDFFiles(t, reportIO.DocumentsDir)
	if len(retryPDFFiles) != 1 {
		t.Fatalf("expected one PDF after successful retry, got %#v", retryPDFFiles)
	}
	if len(rendererProbe.formats) != 2 || rendererProbe.formats[1] != reportmodel.ReportOutputFormatPDF {
		t.Fatalf("expected failed PDF followed by successful PDF without fallback, got %#v", rendererProbe.formats)
	}
	var retryOpenerRequests = runtimeflow.ReadOpenCommandRequests(t, openLogPath)
	if len(retryOpenerRequests) != 1 {
		t.Fatalf("expected one opener request only for the successful retry, got %#v", retryOpenerRequests)
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
		var _, annexPath = runtimeflow.MarkdownBundlePaths(t, files)
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
			var rowRuns, rowFound = runtimeflow.FindPDFConversionRowRuns(pdf.pdf, entry.SourceID)
			if !rowFound || !runtimeflow.PDFConversionContainsEntry(rowRuns, expected) {
				t.Fatalf("PDF converted amounts for %q omit row-local entry %q", entry.SourceID, expected)
			}
			var occurrence = labelOccurrences[label]
			var y, ok = runtimeflow.PDFConversionStartY(pdf.pdf, entry.SourceID, label, occurrence)
			if !ok {
				t.Fatalf("PDF converted amounts for %q omit logical start %q from text runs", entry.SourceID, expected)
			}
			labelOccurrences[label]++
			if index > 0 && y >= previousY {
				t.Fatalf("PDF converted amount %q starts at Y %.2f after previous Y %.2f; want a lower logical line", expected, y, previousY)
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

// convertedAmountsMultilineLayoutFailureCache creates a synthetic conversion
// row whose asset cell cannot fit on a fresh PDF page.
// Authored by: OpenCode
func convertedAmountsMultilineLayoutFailureCache(t *testing.T) model.ProtectedActivityCache {
	t.Helper()
	var cache = runtimeflow.MixedCurrencyConversionProtectedActivityCache(t, 6)
	var longAssetLabel = strings.Repeat("multiline-layout-word ", 700)
	for index := range cache.Activities {
		cache.Activities[index].AssetName = longAssetLabel
		cache.Activities[index].AssetSymbol = longAssetLabel
	}
	return cache
}

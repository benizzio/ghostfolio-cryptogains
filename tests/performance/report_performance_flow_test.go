//go:build performance

// Authored by: OpenCode
package performance

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil/runtimeflow"
)

// TestReportPerformanceFlowLargeHistoryFixture verifies the 10,000-activity
// Markdown and PDF report path with deterministic conversion evidence.
// Authored by: OpenCode
func TestReportPerformanceFlowLargeHistoryFixture(t *testing.T) {
	const minimumActivityCount = 10000
	const minimumCalendarYearSpan = 5
	const threshold = 2 * time.Minute
	var fixture = largeReportFixture(t)
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = runtimeflow.InstallOpenCommandRecorder(t, 0)
	var harness = runtimeflow.NewRuntimeBackedFlowHarness(t, t.TempDir(), runtimeflow.MustCloudSetupConfig(t), false)
	var token = "performance-token"
	if fixture.ActivityCount != minimumActivityCount {
		t.Fatalf("expected %d activities, got %d", minimumActivityCount, fixture.ActivityCount)
	}
	if fixture.CalendarYearSpan < minimumCalendarYearSpan {
		t.Fatalf("expected at least %d calendar years, got %d", minimumCalendarYearSpan, fixture.CalendarYearSpan)
	}
	assertLargeHistoryCrossCurrencyActivity(t, fixture.ProtectedActivityCache.Activities)
	runtimeflow.SeedProtectedSnapshot(t, harness, token, fixture.ProtectedActivityCache)
	var contextResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !contextResult.ProtectedData.HasReadableSnapshot || contextResult.ReportUnavailableReason != runtime.ReportFailureNone {
		t.Fatalf("expected report availability after unlock, got %#v", contextResult)
	}
	var outputFormats = []reportmodel.ReportOutputFormat{reportmodel.ReportOutputFormatMarkdown, reportmodel.ReportOutputFormatPDF}
	var outcomes = make(map[reportmodel.ReportOutputFormat]runtime.ReportOutcome)
	var elapsedByFormat = make(map[reportmodel.ReportOutputFormat]time.Duration)
	for _, outputFormat := range outputFormats {
		var request, err = reportmodel.NewReportRequest(fixture.ReportYear, reportmodel.CostBasisMethodHIFO, reportmodel.ReportBaseCurrencyUSD, outputFormat, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("new %s report request: %v", outputFormat, err)
		}
		var startedAt = time.Now()
		var outcome = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{Request: request})
		var elapsed = time.Since(startedAt)
		if elapsed >= threshold {
			t.Fatalf("expected %s large-history report generation under %s, got %s", outputFormat, threshold, elapsed)
		}
		if !outcome.Success || outcome.FailureReason != runtime.ReportFailureNone {
			t.Fatalf("expected successful %s large-history report generation, got %#v", outputFormat, outcome)
		}
		if !outcome.OutputBundle.OpenRequested {
			t.Fatalf("expected %s opener request, got %#v", outputFormat, outcome.OutputBundle)
		}
		for _, outputFile := range outcome.OutputBundle.Files {
			testutil.AssertPathWithin(t, outputFile.Path, reportIO.DocumentsDir)
			testutil.AssertRegularFile(t, outputFile.Path)
		}
		outcomes[outputFormat] = outcome
		elapsedByFormat[outputFormat] = elapsed
		t.Logf("%s large-history report generation completed in %s", outputFormat, elapsed)
	}
	var requests = runtimeflow.ReadOpenCommandRequests(t, openLogPath)
	if len(requests) != len(outcomes) {
		t.Fatalf("expected one opener request per generated output, got %#v", requests)
	}
	for index, outputFormat := range outputFormats {
		var outcome = outcomes[outputFormat]
		if requests[index] != outcome.OutputBundle.Files[0].Path {
			t.Fatalf("expected opener request %d for %q, got %#v", index, outcome.OutputBundle.Files[0].Path, requests)
		}
	}

	var markdownOutcome = outcomes[reportmodel.ReportOutputFormatMarkdown]
	if len(markdownOutcome.OutputBundle.Files) != 2 {
		t.Fatalf("expected main-plus-annex Markdown bundle, got %#v", markdownOutcome.OutputBundle)
	}
	var reportBytes, readErr = os.ReadFile(markdownOutcome.OutputBundle.Files[0].Path)
	if readErr != nil {
		t.Fatalf("read saved report: %v", readErr)
	}
	for _, expected := range []string{"# Ghostfolio Capital Gains And Losses Report", "- **Year:** 2025", "- **Cost Basis Method:** HIFO", "## Gains-And-Losses Summary", "## Reference Section"} {
		if !strings.Contains(string(reportBytes), expected) {
			t.Fatalf("expected saved report to contain %q", expected)
		}
	}
	var annexBytes, annexReadErr = os.ReadFile(markdownOutcome.OutputBundle.Files[1].Path)
	if annexReadErr != nil {
		t.Fatalf("read saved annex: %v", annexReadErr)
	}
	for _, expected := range []string{"# Annex 1 - Audit", "## Detailed Per-Asset Audit Report", "## Currency Conversion Audit"} {
		if !strings.Contains(string(annexBytes), expected) {
			t.Fatalf("expected saved annex to contain %q", expected)
		}
	}

	var pdfOutcome = outcomes[reportmodel.ReportOutputFormatPDF]
	if len(pdfOutcome.OutputBundle.Files) != 1 {
		t.Fatalf("expected one combined PDF file, got %#v", pdfOutcome.OutputBundle)
	}
	var pdfBytes, pdfReadErr = os.ReadFile(pdfOutcome.OutputBundle.Files[0].Path)
	if pdfReadErr != nil {
		t.Fatalf("read saved PDF: %v", pdfReadErr)
	}
	assertGeneratedLargeHistoryPDFContract(t, pdfBytes)
	runtimeflow.AssertNoCleartextReportInAppStorage(t, harness.BaseDir)
	t.Logf("10,000-activity verification completed for Markdown in %s and PDF in %s across %d calendar years", elapsedByFormat[reportmodel.ReportOutputFormatMarkdown], elapsedByFormat[reportmodel.ReportOutputFormatPDF], fixture.CalendarYearSpan)
}

// assertLargeHistoryCrossCurrencyActivity verifies the fixture exercises all
// currencies required by the report conversion scenario.
// Authored by: OpenCode
func assertLargeHistoryCrossCurrencyActivity(t *testing.T, activities []syncmodel.ActivityRecord) {
	t.Helper()
	var currencies = make(map[string]bool)
	for _, activity := range activities {
		currencies[activity.OrderCurrency] = true
	}
	for _, currency := range []string{"USD", "EUR", "GBP"} {
		if !currencies[currency] {
			t.Fatalf("expected large-history fixture to include %s activity currency, got %#v", currency, currencies)
		}
	}
}

// assertGeneratedLargeHistoryPDFContract verifies the combined PDF envelope,
// page metadata, and pagination.
// Authored by: OpenCode
func assertGeneratedLargeHistoryPDFContract(t *testing.T, pdfBytes []byte) {
	t.Helper()
	var pdfText = string(pdfBytes)
	for _, expected := range []string{"%PDF-", "%%EOF", "/MediaBox"} {
		if !strings.Contains(pdfText, expected) {
			t.Fatalf("expected generated PDF to contain %q", expected)
		}
	}
	if len(regexp.MustCompile(`/Type\s*/Page\b`).FindAll(pdfBytes, -1)) < 2 {
		t.Fatalf("expected generated PDF to contain multiple report pages")
	}
}

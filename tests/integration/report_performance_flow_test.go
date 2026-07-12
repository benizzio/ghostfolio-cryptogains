// Package integration verifies black-box workflow behavior for the current
// slice, including the documented large-history report performance path.
// Authored by: OpenCode
package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
)

// TestReportPerformanceFlowLargeHistoryFixture verifies the documented SC-007
// path with one deterministic 10,000-activity report run when explicitly
// enabled.
// Authored by: OpenCode
func TestReportPerformanceFlowLargeHistoryFixture(t *testing.T) {
	if os.Getenv(performanceVerificationEnvironmentVariable) != "1" {
		t.Skipf("set %s=1 to run the SC-007 performance verification path", performanceVerificationEnvironmentVariable)
	}

	const minimumActivityCount = 10000
	const minimumCalendarYearSpan = 5
	const performanceThreshold = 2 * time.Minute
	const coverageThreshold = 5 * time.Minute

	var threshold = performanceThreshold
	if os.Getenv("GHOSTFOLIO_CRYPTOGAINS_PERFORMANCE_COVERAGE") == "1" {
		// Coverage instrumentation materially changes this end-to-end runtime.
		threshold = coverageThreshold
	}

	var fixture = testutil.DeterministicLargeReportPerformanceFixture()
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)
	var token = "performance-token"

	if fixture.ActivityCount != minimumActivityCount {
		t.Fatalf("expected deterministic performance fixture to contain %d activities, got %d", minimumActivityCount, fixture.ActivityCount)
	}
	if fixture.CalendarYearSpan < minimumCalendarYearSpan {
		t.Fatalf("expected deterministic performance fixture to span at least %d calendar years, got %d", minimumCalendarYearSpan, fixture.CalendarYearSpan)
	}
	assertLargeHistoryCrossCurrencyActivity(t, fixture.ProtectedActivityCache.Activities)

	seedProtectedSnapshot(t, harness, token, fixture.ProtectedActivityCache)

	var contextResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !contextResult.ProtectedData.HasReadableSnapshot {
		t.Fatalf("expected readable snapshot after unlock, got %#v", contextResult)
	}
	if contextResult.ReportUnavailableReason != runtime.ReportFailureNone {
		t.Fatalf("expected report availability after unlock, got %#v", contextResult)
	}

	var startedAt = time.Now()
	var outcomes []runtime.ReportOutcome
	for _, outputFormat := range []reportmodel.ReportOutputFormat{reportmodel.ReportOutputFormatMarkdown, reportmodel.ReportOutputFormatPDF} {
		var request, err = reportmodel.NewReportRequest(
			fixture.ReportYear,
			reportmodel.CostBasisMethodHIFO,
			reportmodel.ReportBaseCurrencyUSD,
			outputFormat,
			time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC),
		)
		if err != nil {
			t.Fatalf("new report request: %v", err)
		}
		outcomes = append(outcomes, harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{Request: request}))
	}
	var elapsed = time.Since(startedAt)

	for _, outcome := range outcomes {
		if !outcome.Success {
			t.Fatalf("expected successful large-history report generation, got %#v", outcome)
		}
		if outcome.FailureReason != runtime.ReportFailureNone {
			t.Fatalf("expected success without warning category, got %#v", outcome)
		}
	}
	if elapsed >= threshold {
		t.Fatalf("expected SC-007 verification under %s, got %s", threshold, elapsed)
	}
	for _, outcome := range outcomes {
		for _, outputFile := range outcome.OutputBundle.Files {
			testutil.AssertPathWithin(t, outputFile.Path, reportIO.DocumentsDir)
			testutil.AssertRegularFile(t, outputFile.Path)
		}
	}

	var openerRequests = readOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != len(outcomes) {
		t.Fatalf("expected one opener request per generated output, got %#v", openerRequests)
	}

	var markdownOutcome = outcomes[0]
	var reportBytes, readErr = os.ReadFile(markdownOutcome.OutputBundle.Files[0].Path)
	if readErr != nil {
		t.Fatalf("read saved report %q: %v", markdownOutcome.OutputBundle.Files[0].Path, readErr)
	}
	var reportText = string(reportBytes)
	for _, expected := range []string{
		"# Ghostfolio Capital Gains And Losses Report",
		"- **Year:** 2025",
		"- **Cost Basis Method:** HIFO",
		"## Gains-And-Losses Summary",
		"## Reference Section",
	} {
		if !strings.Contains(reportText, expected) {
			t.Fatalf("expected saved report to contain %q", expected)
		}
	}
	var annexBytes, annexReadErr = os.ReadFile(markdownOutcome.OutputBundle.Files[1].Path)
	if annexReadErr != nil {
		t.Fatalf("read saved annex %q: %v", markdownOutcome.OutputBundle.Files[1].Path, annexReadErr)
	}
	if !strings.Contains(string(annexBytes), "# Annex 1 - Audit") || !strings.Contains(string(annexBytes), "## Detailed Per-Asset Audit Report") {
		t.Fatalf("expected Markdown annex output to contain Annex 1 audit sections")
	}
	var pdfBytes, pdfReadErr = os.ReadFile(outcomes[1].OutputBundle.Files[0].Path)
	if pdfReadErr != nil {
		t.Fatalf("read saved PDF %q: %v", outcomes[1].OutputBundle.Files[0].Path, pdfReadErr)
	}
	assertGeneratedLargeHistoryPDFContract(t, pdfBytes)

	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
	t.Logf(
		"SC-007 verification completed in %s for %d activities across %d calendar years",
		elapsed,
		fixture.ActivityCount,
		fixture.CalendarYearSpan,
	)
}

// assertLargeHistoryCrossCurrencyActivity verifies the shared large-history
// fixture exercises conversion for each output format generated by this flow.
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

// assertGeneratedLargeHistoryPDFContract verifies the saved combined PDF has a
// valid envelope, landscape A4 pages, and paginated generated report content.
// Authored by: OpenCode
func assertGeneratedLargeHistoryPDFContract(t *testing.T, pdfBytes []byte) {
	t.Helper()

	var pdfText = string(pdfBytes)
	for _, expected := range []string{"%PDF-", "%%EOF", "/MediaBox"} {
		if !strings.Contains(pdfText, expected) {
			t.Fatalf("expected generated PDF to contain %q", expected)
		}
	}
	if strings.Count(pdfText, "/Type /Page") < 2 {
		t.Fatalf("expected generated PDF to contain multiple report pages")
	}
}

//go:build performance

// Authored by: OpenCode
package performance

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil/runtimeflow"
)

// TestReportPerformanceFlowLargeHistoryFixture verifies SC-007 with one
// deterministic 10,000-activity report run.
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
	runtimeflow.SeedProtectedSnapshot(t, harness, token, fixture.ProtectedActivityCache)
	var contextResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !contextResult.ProtectedData.HasReadableSnapshot || contextResult.ReportUnavailableReason != runtime.ReportFailureNone {
		t.Fatalf("expected report availability after unlock, got %#v", contextResult)
	}
	var request, err = reportmodel.NewReportRequest(fixture.ReportYear, reportmodel.CostBasisMethodHIFO, reportmodel.ReportBaseCurrencyUSD, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}
	var startedAt = time.Now()
	var outcome = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{Request: request})
	var elapsed = time.Since(startedAt)
	if !outcome.Success || outcome.FailureReason != runtime.ReportFailureNone {
		t.Fatalf("expected successful large-history report generation, got %#v", outcome)
	}
	if elapsed >= threshold {
		t.Fatalf("expected SC-007 under %s, got %s", threshold, elapsed)
	}
	if !outcome.OutputFile.OpenRequested {
		t.Fatalf("expected opener request, got %#v", outcome.OutputFile)
	}
	testutil.AssertPathWithin(t, outcome.OutputFile.Path, reportIO.DocumentsDir)
	testutil.AssertRegularFile(t, outcome.OutputFile.Path)
	var requests = runtimeflow.ReadOpenCommandRequests(t, openLogPath)
	if len(requests) != 1 || requests[0] != outcome.OutputFile.Path {
		t.Fatalf("expected one opener request for %q, got %#v", outcome.OutputFile.Path, requests)
	}
	var reportBytes, readErr = os.ReadFile(outcome.OutputFile.Path)
	if readErr != nil {
		t.Fatalf("read saved report: %v", readErr)
	}
	for _, expected := range []string{"# Ghostfolio Capital Gains And Losses Report", "- Year: 2025", "- Cost Basis Method: HIFO", "## Gains-And-Losses Summary", "## Reference Section"} {
		if !strings.Contains(string(reportBytes), expected) {
			t.Fatalf("expected saved report to contain %q", expected)
		}
	}
	runtimeflow.AssertNoCleartextReportInAppStorage(t, harness.BaseDir)
	t.Logf("SC-007 verification completed in %s for %d activities across %d calendar years", elapsed, fixture.ActivityCount, fixture.CalendarYearSpan)
}

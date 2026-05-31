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
	const threshold = 2 * time.Minute

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

	seedProtectedSnapshot(t, harness, token, fixture.ProtectedActivityCache)

	var contextResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !contextResult.ProtectedData.HasReadableSnapshot {
		t.Fatalf("expected readable snapshot after unlock, got %#v", contextResult)
	}
	if contextResult.ReportUnavailableReason != runtime.ReportFailureNone {
		t.Fatalf("expected report availability after unlock, got %#v", contextResult)
	}

	var request, err = reportmodel.NewReportRequest(
		fixture.ReportYear,
		reportmodel.CostBasisMethodHIFO,
		time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}

	var startedAt = time.Now()
	var outcome = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{Request: request})
	var elapsed = time.Since(startedAt)

	if !outcome.Success {
		t.Fatalf("expected successful large-history report generation, got %#v", outcome)
	}
	if outcome.FailureReason != runtime.ReportFailureNone {
		t.Fatalf("expected success without warning category, got %#v", outcome)
	}
	if elapsed >= threshold {
		t.Fatalf("expected SC-007 verification under %s, got %s", threshold, elapsed)
	}
	if !outcome.OutputFile.OpenRequested {
		t.Fatalf("expected successful run to request automatic opening, got %#v", outcome.OutputFile)
	}

	testutil.AssertPathWithin(t, outcome.OutputFile.Path, reportIO.DocumentsDir)
	testutil.AssertRegularFile(t, outcome.OutputFile.Path)

	var openerRequests = readOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != 1 || openerRequests[0] != outcome.OutputFile.Path {
		t.Fatalf("expected one opener request for %q, got %#v", outcome.OutputFile.Path, openerRequests)
	}

	var reportBytes, readErr = os.ReadFile(outcome.OutputFile.Path)
	if readErr != nil {
		t.Fatalf("read saved report %q: %v", outcome.OutputFile.Path, readErr)
	}
	var reportText = string(reportBytes)
	for _, expected := range []string{
		"# Ghostfolio Capital Gains And Losses Report",
		"- Year: 2025",
		"- Cost Basis Method: HIFO",
		"## Gains-And-Losses Summary",
		"## Reference Section",
	} {
		if !strings.Contains(reportText, expected) {
			t.Fatalf("expected saved report to contain %q", expected)
		}
	}

	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
	t.Logf(
		"SC-007 verification completed in %s for %d activities across %d calendar years",
		elapsed,
		fixture.ActivityCount,
		fixture.CalendarYearSpan,
	)
}

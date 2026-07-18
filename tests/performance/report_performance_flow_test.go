//go:build performance

// Authored by: OpenCode
package performance

import (
	"context"
	"os"
	stdruntime "runtime"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil/runtimeflow"
)

// TestReportPerformanceFlowLargeHistoryFixture verifies the exact 10,000-activity
// Markdown and PDF report generation path using local deterministic inputs.
// Authored by: OpenCode
func TestReportPerformanceFlowLargeHistoryFixture(t *testing.T) {
	const expectedActivityCount = 10000
	const threshold = 2 * time.Minute
	var requestedAt = time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC)
	logPerformanceEnvironment(t)
	var fixture = largeReportFixture(t)
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = runtimeflow.InstallOpenCommandRecorder(t, 0)
	var harness = runtimeflow.NewRuntimeBackedFlowHarnessWithCurrencyRateService(t, t.TempDir(), runtimeflow.MustCloudSetupConfig(t), false, runtimeflow.DeterministicCurrencyRates{})
	var token = "performance-token"
	if fixture.ActivityCount != expectedActivityCount {
		t.Fatalf("expected %d activities, got %d", expectedActivityCount, fixture.ActivityCount)
	}
	if fixture.CalendarYearSpan != 6 {
		t.Fatalf("expected six calendar years, got %d", fixture.CalendarYearSpan)
	}
	assertLargeHistoryActivityComposition(t, fixture.ProtectedActivityCache.Activities)
	runtimeflow.SeedProtectedSnapshot(t, harness, token, fixture.ProtectedActivityCache)
	var contextResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !contextResult.ProtectedData.HasReadableSnapshot || contextResult.ReportUnavailableReason != runtime.ReportFailureNone {
		t.Fatalf("expected report availability after unlock, got %#v", contextResult)
	}
	var outputFormats = []reportmodel.ReportOutputFormat{reportmodel.ReportOutputFormatMarkdown, reportmodel.ReportOutputFormatPDF}
	var outcomes = make(map[reportmodel.ReportOutputFormat]runtime.ReportOutcome)
	for _, outputFormat := range outputFormats {
		var request, err = reportmodel.NewReportRequest(fixture.ReportYear, reportmodel.CostBasisMethodHIFO, reportmodel.ReportBaseCurrencyUSD, outputFormat, requestedAt)
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
		if outcome.Request.Year != 2025 || outcome.Request.CostBasisMethod != reportmodel.CostBasisMethodHIFO || outcome.Request.ReportBaseCurrency != reportmodel.ReportBaseCurrencyUSD || !outcome.Request.RequestedAt.Equal(requestedAt) {
			t.Fatalf("unexpected %s report request: %#v", outputFormat, outcome.Request)
		}
		if !outcome.OutputBundle.OpenRequested {
			t.Fatalf("expected %s opener request, got %#v", outputFormat, outcome.OutputBundle)
		}
		if len(outcome.OutputBundle.Files) == 0 {
			t.Fatalf("expected at least one generated %s output file", outputFormat)
		}
		for _, outputFile := range outcome.OutputBundle.Files {
			testutil.AssertPathWithin(t, outputFile.Path, reportIO.DocumentsDir)
			testutil.AssertRegularFile(t, outputFile.Path)
			var fileInfo, statErr = os.Stat(outputFile.Path)
			if statErr != nil {
				t.Fatalf("stat generated %s output: %v", outputFormat, statErr)
			}
			if fileInfo.Size() == 0 {
				t.Fatalf("expected non-empty generated %s output at %q", outputFormat, outputFile.Path)
			}
		}
		outcomes[outputFormat] = outcome
		t.Logf("performance format=%s elapsed=%s threshold=%s", outputFormat, elapsed, threshold)
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

	runtimeflow.AssertNoCleartextReportInAppStorage(t, harness.BaseDir)
	t.Logf("10,000-activity verification completed across %d calendar years; performance coverage mode=none", fixture.CalendarYearSpan)
}

// logPerformanceEnvironment records the runner and Go conditions for local or
// authoritative isolated performance evidence.
// Authored by: OpenCode
func logPerformanceEnvironment(t *testing.T) {
	t.Helper()
	var runnerImage = os.Getenv("ImageOS")
	if runnerImage == "" {
		runnerImage = os.Getenv("RUNNER_OS")
	}
	if runnerImage == "" {
		runnerImage = stdruntime.GOOS
	}
	var runnerImageVersion = os.Getenv("ImageVersion")
	if runnerImageVersion == "" {
		runnerImageVersion = "local"
	}
	t.Logf("performance environment: runner_image=%s runner_image_version=%s os=%s arch=%s cpus=%d go=%s rate_source=local exact_rate=1.1 network=disabled coverage_mode=none", runnerImage, runnerImageVersion, stdruntime.GOOS, stdruntime.GOARCH, stdruntime.NumCPU(), stdruntime.Version())
}

// assertLargeHistoryActivityComposition verifies every scale-fixture dimension
// before protected snapshot setup and report generation begin.
// Authored by: OpenCode
func assertLargeHistoryActivityComposition(t *testing.T, activities []syncmodel.ActivityRecord) {
	t.Helper()
	if len(activities) != 10000 {
		t.Fatalf("expected 10000 activities, got %d", len(activities))
	}
	var assetCounts = make(map[string]int)
	var currencyCounts = make(map[string]int)
	var activityTypeCounts = make(map[string]map[syncmodel.ActivityType]int)
	var buyYearCounts = make(map[string]map[int]int)
	var quantityOne = mustDecimalPointer(t, "1")
	for _, activity := range activities {
		assetCounts[activity.AssetIdentityKey]++
		currencyCounts[activity.OrderCurrency]++
		if activityTypeCounts[activity.AssetIdentityKey] == nil {
			activityTypeCounts[activity.AssetIdentityKey] = make(map[syncmodel.ActivityType]int)
		}
		activityTypeCounts[activity.AssetIdentityKey][activity.ActivityType]++
		if activity.Quantity.Cmp(quantityOne) != 0 {
			t.Fatalf("expected quantity-one activity, got %s for %q", activity.Quantity.String(), activity.SourceID)
		}
		if activity.OrderUnitPrice != nil || activity.OrderGrossValue == nil || activity.OrderFeeAmount == nil {
			t.Fatalf("expected priced activity fields for %q, got %#v", activity.SourceID, activity)
		}
		if activity.OrderGrossValue.Sign() <= 0 || activity.OrderFeeAmount.Sign() <= 0 {
			t.Fatalf("expected positive same-tier priced values for %q, got gross=%s fee=%s", activity.SourceID, activity.OrderGrossValue, activity.OrderFeeAmount)
		}
		var occurredAt, err = time.Parse(time.RFC3339, activity.OccurredAt)
		if err != nil {
			t.Fatalf("parse activity date for %q: %v", activity.SourceID, err)
		}
		if activity.ActivityType == syncmodel.ActivityTypeBuy && (occurredAt.Year() < 2020 || occurredAt.Year() > 2024) {
			t.Fatalf("expected BUY activity between 2020 and 2024, got %s for %q", occurredAt.Format(time.DateOnly), activity.SourceID)
		}
		if activity.ActivityType == syncmodel.ActivityTypeBuy {
			if buyYearCounts[activity.AssetIdentityKey] == nil {
				buyYearCounts[activity.AssetIdentityKey] = make(map[int]int)
			}
			buyYearCounts[activity.AssetIdentityKey][occurredAt.Year()]++
		}
		if activity.ActivityType == syncmodel.ActivityTypeSell && occurredAt.Year() != 2025 {
			t.Fatalf("expected SELL activity in 2025, got %s for %q", occurredAt.Format(time.DateOnly), activity.SourceID)
		}
	}
	for assetKey, count := range map[string]int{"asset-btc-performance-001": 5000, "asset-eth-performance-001": 5000} {
		if assetCounts[assetKey] != count {
			t.Fatalf("expected %d activities for %s, got %d", count, assetKey, assetCounts[assetKey])
		}
		if activityTypeCounts[assetKey][syncmodel.ActivityTypeBuy] != 2500 || activityTypeCounts[assetKey][syncmodel.ActivityTypeSell] != 2500 {
			t.Fatalf("expected 2500 BUY and 2500 SELL activities for %s, got %#v", assetKey, activityTypeCounts[assetKey])
		}
		for year := 2020; year <= 2024; year++ {
			if buyYearCounts[assetKey][year] != 500 {
				t.Fatalf("expected 500 BUY activities for %s in %d, got %d", assetKey, year, buyYearCounts[assetKey][year])
			}
		}
	}
	for currency, count := range map[string]int{"USD": 3334, "EUR": 3333, "GBP": 3333} {
		if currencyCounts[currency] != count {
			t.Fatalf("expected %d %s activities, got %d", count, currency, currencyCounts[currency])
		}
	}
}

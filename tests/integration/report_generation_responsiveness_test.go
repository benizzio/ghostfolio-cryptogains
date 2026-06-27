// Package integration verifies black-box report generation responsiveness for
// large cross-currency histories.
// Authored by: OpenCode
package integration

import (
	"os"
	"strings"
	"testing"
	"time"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
)

// TestReportGenerationResponsivenessLargeCrossCurrencyFixture verifies that the
// US2 10,000-activity conversion path remains asynchronous and bounds provider
// lookups by unique source-calendar rate keys rather than monetary field count.
// Authored by: OpenCode
func TestReportGenerationResponsivenessLargeCrossCurrencyFixture(t *testing.T) {
	const activityCount = 10000
	const expectedUniqueRateLookupUpperBound = 92

	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)
	var token = "responsiveness-token"
	var cache = responsivenessCurrencyProtectedActivityCache(t, activityCount)

	seedProtectedSnapshot(t, harness, token, cache)

	var model = unlockSyncReportsContext(t, harness.Model, token)
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, 2025)
	model = selectReportBaseCurrency(t, model, reportmodel.ReportBaseCurrencyUSD)
	model, cmd := startReportGenerationAfterBaseCurrencySelection(t, model)
	if model.ActiveScreen() != "report_busy" {
		t.Fatalf("expected asynchronous report busy screen before delayed provider work completes, got %s", model.ActiveScreen())
	}

	model = applyBatchCmd(t, model, cmd)
	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result after large cross-currency generation, got %s", model.ActiveScreen())
	}

	var content = normalizeRenderedText(model.View().Content)
	if !strings.Contains(content, "Saved Markdown Path:") {
		t.Fatalf("expected successful large cross-currency report result, got %q", content)
	}
	if strings.Contains(content, "lookups: 30000") || strings.Contains(content, "lookups: 10000") {
		t.Fatalf("expected lookup count bounded by unique rate keys, got %q", content)
	}

	var files = mustMarkdownFiles(t, reportIO.DocumentsDir)
	if len(files) != 1 {
		t.Fatalf("expected one saved responsiveness Markdown report, got %#v", files)
	}
	var openerRequests = readOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != 1 || openerRequests[0] != files[0] {
		t.Fatalf("expected one opener request for %q, got %#v", files[0], openerRequests)
	}

	var rawReport, err = os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("read saved report %q: %v", files[0], err)
	}
	var reportText = string(rawReport)
	for _, expected := range []string{
		"- Report Calculation Currency: USD",
		"Federal Reserve Board H.10/Data Download Program",
		"responsiveness-eur-buy-00000",
		"responsiveness-gbp-buy-00001",
	} {
		if !strings.Contains(reportText, expected) {
			t.Fatalf("expected responsiveness report to contain %q", expected)
		}
	}
	if count := strings.Count(reportText, "Federal Reserve Board H.10/Data Download Program"); count > expectedUniqueRateLookupUpperBound {
		t.Fatalf("expected at most %d unique H.10 evidence rows, got %d", expectedUniqueRateLookupUpperBound, count)
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// responsivenessCurrencyProtectedActivityCache builds 10,000 priced activities
// sharing a small deterministic set of source-calendar conversion keys.
// Authored by: OpenCode
func responsivenessCurrencyProtectedActivityCache(t *testing.T, activityCount int) syncmodel.ProtectedActivityCache {
	t.Helper()

	var activities = make([]syncmodel.ActivityRecord, 0, activityCount)
	var currencies = []string{"EUR", "GBP", "USD"}
	for index := 0; index < activityCount; index++ {
		var currency = currencies[index%len(currencies)]
		var day = 1 + index%28
		var month = time.Month(1 + index%3)
		activities = append(activities, roundedReportActivity(t, roundedReportActivityInput{
			SourceID:         responsivenessSourceID(currency, index),
			OccurredAt:       time.Date(2025, month, day, 9, 0, 0, 0, time.FixedZone("source", 2*60*60)).Format(time.RFC3339),
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-responsive-001",
			AssetSymbol:      "RSP",
			AssetName:        "Responsive Asset",
			Quantity:         "1",
			OrderCurrency:    currency,
			OrderUnitPrice:   "10",
			OrderGrossValue:  "10",
			OrderFeeAmount:   "1",
		}))
	}

	return syncmodel.ProtectedActivityCache{
		SyncedAt:             mustReportFixtureTime(t, "2026-05-20T15:04:05Z"),
		RetrievedCount:       len(activities),
		ActivityCount:        len(activities),
		AvailableReportYears: []int{2025},
		Activities:           activities,
	}
}

// responsivenessSourceID returns a deterministic large-fixture activity ID.
// Authored by: OpenCode
func responsivenessSourceID(currency string, index int) string {
	var raw = leftPadFive(index)
	return strings.ToLower("responsiveness-"+currency+"-buy-") + raw
}

// leftPadFive renders one fixed-width large-fixture index.
// Authored by: OpenCode
func leftPadFive(value int) string {
	var raw = leftPadThree(value)
	for len(raw) < 5 {
		raw = "0" + raw
	}

	return raw
}

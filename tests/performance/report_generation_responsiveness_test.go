//go:build performance

// Authored by: OpenCode
package performance

import (
	"os"
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil/runtimeflow"
)

// TestReportGenerationResponsivenessLargeCrossCurrencyFixture verifies US2
// asynchronous generation and bounded source-calendar rate lookups.
// Authored by: OpenCode
func TestReportGenerationResponsivenessLargeCrossCurrencyFixture(t *testing.T) {
	const activityCount = 10000
	const expectedUniqueRateLookupUpperBound = 92
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = runtimeflow.InstallOpenCommandRecorder(t, 0)
	var harness = runtimeflow.NewRuntimeBackedFlowHarness(t, t.TempDir(), runtimeflow.MustCloudSetupConfig(t), false)
	runtimeflow.SeedProtectedSnapshot(t, harness, "responsiveness-token", largeCrossCurrencyCache(t, activityCount))
	var model = runtimeflow.UnlockSyncReportsContext(t, harness.Model, "responsiveness-token")
	model = runtimeflow.OpenReportSelection(t, model)
	model = runtimeflow.SelectReportYear(t, model, 2025)
	model = runtimeflow.SelectReportBaseCurrency(t, model, reportmodel.ReportBaseCurrencyUSD)
	model, cmd := runtimeflow.StartReportGeneration(t, model)
	if model.ActiveScreen() != "report_busy" {
		t.Fatalf("expected asynchronous report busy screen, got %s", model.ActiveScreen())
	}
	model = runtimeflow.ApplyBatchCmd(t, model, cmd)
	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result, got %s", model.ActiveScreen())
	}
	var content = runtimeflow.NormalizeRenderedText(model.View().Content)
	if !strings.Contains(content, "Saved Markdown Path:") || strings.Contains(content, "lookups: 30000") || strings.Contains(content, "lookups: 10000") {
		t.Fatalf("unexpected result content %q", content)
	}
	var files = runtimeflow.MarkdownFiles(t, reportIO.DocumentsDir)
	if len(files) != 1 {
		t.Fatalf("expected one saved Markdown report, got %#v", files)
	}
	var requests = runtimeflow.ReadOpenCommandRequests(t, openLogPath)
	if len(requests) != 1 || requests[0] != files[0] {
		t.Fatalf("expected one opener request for %q, got %#v", files[0], requests)
	}
	var reportBytes, err = os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("read saved report: %v", err)
	}
	var reportText = string(reportBytes)
	for _, expected := range []string{"- **Report Calculation Currency:** USD", "Federal Reserve Board H.10/Data Download Program", "responsiveness-eur-buy-00000", "responsiveness-gbp-buy-00001"} {
		if !strings.Contains(reportText, expected) {
			t.Fatalf("expected report to contain %q", expected)
		}
	}
	if count := strings.Count(reportText, "Federal Reserve Board H.10/Data Download Program"); count > expectedUniqueRateLookupUpperBound {
		t.Fatalf("expected at most %d evidence rows, got %d", expectedUniqueRateLookupUpperBound, count)
	}
	runtimeflow.AssertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

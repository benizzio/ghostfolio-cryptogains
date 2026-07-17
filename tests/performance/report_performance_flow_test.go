//go:build performance

// Authored by: OpenCode
package performance

import (
	"context"
	"os"
	"regexp"
	stdruntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil/runtimeflow"
)

// TestReportPerformanceFlowLargeHistoryFixture verifies the exact 10,000-activity
// Markdown and PDF report path with deterministic conversion evidence.
// Authored by: OpenCode
func TestReportPerformanceFlowLargeHistoryFixture(t *testing.T) {
	const expectedActivityCount = 10000
	const expectedConversionRows = 6666
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
		for _, outputFile := range outcome.OutputBundle.Files {
			testutil.AssertPathWithin(t, outputFile.Path, reportIO.DocumentsDir)
			testutil.AssertRegularFile(t, outputFile.Path)
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
	assertGeneratedLargeHistoryMarkdownContract(t, reportBytes, annexBytes, expectedConversionRows)

	var pdfOutcome = outcomes[reportmodel.ReportOutputFormatPDF]
	if len(pdfOutcome.OutputBundle.Files) != 1 {
		t.Fatalf("expected one combined PDF file, got %#v", pdfOutcome.OutputBundle)
	}
	var pdfBytes, pdfReadErr = os.ReadFile(pdfOutcome.OutputBundle.Files[0].Path)
	if pdfReadErr != nil {
		t.Fatalf("read saved PDF: %v", pdfReadErr)
	}
	assertGeneratedLargeHistoryPDFContract(t, pdfBytes, expectedConversionRows)
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

// assertGeneratedLargeHistoryMarkdownContract verifies exact conversion-row
// cardinality, three-entry completeness, and local rate evidence in Markdown.
// Authored by: OpenCode
func assertGeneratedLargeHistoryMarkdownContract(t *testing.T, reportBytes []byte, annexBytes []byte, expectedConversionRows int) {
	t.Helper()
	var reportText = string(reportBytes)
	var annexText = string(annexBytes)
	for _, expected := range []string{"- **Report Calculation Currency:** USD", "The data in this report does not follow any legally required rules for any country's tax returns and is for reference only."} {
		if !strings.Contains(reportText, expected) {
			t.Fatalf("expected Markdown main report to contain %q", expected)
		}
	}
	var conversionRows []string
	for _, line := range strings.Split(annexText, "\n") {
		if strings.Contains(line, "unit_price:") {
			conversionRows = append(conversionRows, line)
		}
	}
	if len(conversionRows) != expectedConversionRows {
		t.Fatalf("expected %d Markdown conversion rows, got %d", expectedConversionRows, len(conversionRows))
	}
	var currencyCounts = make(map[string]int)
	for _, line := range conversionRows {
		for _, entryLabel := range []string{"unit_price:", "gross_value:", "fee_amount:"} {
			if strings.Count(line, entryLabel) != 1 {
				t.Fatalf("expected one %s entry in Markdown conversion row %q", entryLabel, line)
			}
		}
		if strings.Count(line, "<br>") != 2 || strings.Count(line, " -> ") != 3 {
			t.Fatalf("expected three controlled Markdown conversion entries in %q", line)
		}
		if strings.Contains(line, "| EUR |") {
			currencyCounts["EUR"]++
		}
		if strings.Contains(line, "| GBP |") {
			currencyCounts["GBP"]++
		}
		if strings.Contains(line, "| USD | USD |") {
			t.Fatalf("expected no same-currency conversion row in %q", line)
		}
		if !strings.Contains(line, "| 1.1 |") {
			t.Fatalf("expected exact local 1.1 rate evidence in %q", line)
		}
	}
	for currency, count := range map[string]int{"EUR": 3333, "GBP": 3333} {
		if currencyCounts[currency] != count {
			t.Fatalf("expected %d Markdown %s conversion rows, got %d", count, currency, currencyCounts[currency])
		}
	}
}

// assertGeneratedLargeHistoryPDFContract verifies the combined PDF envelope,
// complete conversion entries, repeated table context, and pagination bounds.
// Authored by: OpenCode
func assertGeneratedLargeHistoryPDFContract(t *testing.T, pdfBytes []byte, expectedConversionRows int) {
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
	var inspection, err = testutil.InspectGeneratedPDF(pdfBytes)
	if err != nil {
		t.Fatalf("inspect generated performance PDF: %v", err)
	}
	var searchableText = normalizePerformancePDFText(inspection.SearchableText)
	for _, entryLabel := range []string{"unit_price:", "gross_value:", "fee_amount:"} {
		if strings.Count(searchableText, entryLabel) != expectedConversionRows {
			t.Fatalf("expected %d PDF %s entries, got %d", expectedConversionRows, entryLabel, strings.Count(searchableText, entryLabel))
		}
	}
	if strings.Count(searchableText, " -> ") < expectedConversionRows*3 || strings.Count(searchableText, ";") < expectedConversionRows*2 {
		t.Fatalf("expected complete three-entry PDF conversion rows, searchable text is incomplete")
	}
	if strings.Count(searchableText, "1.1") < expectedConversionRows {
		t.Fatalf("expected exact local 1.1 rate evidence in every PDF conversion row")
	}
	var pageText = make(map[int]string)
	var conversionPages = make(map[int]bool)
	for _, run := range inspection.TextRuns {
		pageText[run.Page] += " " + run.Text
		if strings.Contains(run.Text, "unit_price:") {
			conversionPages[run.Page] = true
			if run.Page < 1 || run.Page > len(inspection.PageBoxes) {
				t.Fatalf("conversion entry has invalid PDF page %d", run.Page)
			}
			var page = inspection.PageBoxes[run.Page-1]
			if run.X < 0 || run.X >= page.Width || run.Y < 0 || run.Y >= page.Height {
				t.Fatalf("conversion entry is outside PDF page bounds: run=%#v page=%#v", run, page)
			}
		}
	}
	var headerPages = make(map[int]bool)
	var continuationPages = make(map[int]bool)
	for page, text := range pageText {
		var normalizedPageText = normalizePerformancePDFText(text)
		if strings.Contains(normalizedPageText, "Currency Conversion Audit Table") {
			headerPages[page] = true
		}
		if strings.Contains(normalizedPageText, "Currency Conversion Audit Table (continued)") {
			continuationPages[page] = true
		}
		if conversionPages[page] && !headerPages[page] {
			t.Fatalf("expected conversion page %d to repeat the audit table header", page)
		}
	}
	if len(conversionPages) < 2 {
		t.Fatalf("expected PDF conversion audit to span continuation pages, got %d pages", len(conversionPages))
	}
	if len(headerPages) < 2 || len(continuationPages) == 0 {
		t.Fatalf("expected repeated PDF audit headers and continuation context, headers=%d continuation=%d", len(headerPages), len(continuationPages))
	}
}

// normalizePerformancePDFText makes searchable PDF text suitable for exact
// cardinality and continuation-context assertions.
// Authored by: OpenCode
func normalizePerformancePDFText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

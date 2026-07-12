// Package integration verifies black-box workflow behavior for the current
// slice, including runtime-backed report generation flows.
// Authored by: OpenCode
package integration

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	syncvalidate "github.com/benizzio/ghostfolio-cryptogains/internal/sync/validate"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/cockroachdb/apd/v3"
)

// TestReportGenerationSuccessWritesMarkdownAndReturnsToUnlockedContext verifies
// one end-to-end report save through the real runtime-backed workflow.
// Authored by: OpenCode
func TestReportGenerationSuccessWritesMarkdownAndReturnsToUnlockedContext(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var fixture = testutil.DeterministicReportLedgerFixture()
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)

	seedProtectedSnapshot(t, harness, "token-123", fixture.ProtectedActivityCache)

	var model = unlockSyncReportsContext(t, harness.Model, "token-123")
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, fixture.PrimaryReportYear)
	model = selectReportBaseCurrency(t, model, reportmodel.ReportBaseCurrencyUSD)
	model, cmd := startReportGenerationAfterBaseCurrencySelection(t, model)
	model = applyBatchCmd(t, model, cmd)

	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
	}

	var files = mustAllMarkdownFiles(t, reportIO.DocumentsDir)
	if len(files) != 2 {
		t.Fatalf("expected main-plus-annex Markdown output, got %#v", files)
	}

	var reportPath, annexPath = markdownBundlePaths(t, files)
	testutil.AssertPathWithin(t, reportPath, reportIO.DocumentsDir)
	testutil.AssertRegularFile(t, reportPath)
	testutil.AssertPathWithin(t, annexPath, reportIO.DocumentsDir)
	testutil.AssertRegularFile(t, annexPath)
	if !strings.HasPrefix(filepath.Base(reportPath), "ghostfolio-capital-gains-2024-fifo-") {
		t.Fatalf("expected FIFO report filename slug, got %q", filepath.Base(reportPath))
	}
	if !strings.HasPrefix(filepath.Base(annexPath), "ghostfolio-capital-gains-2024-fifo-annex-1-") {
		t.Fatalf("expected FIFO annex filename slug, got %q", filepath.Base(annexPath))
	}

	var content = normalizeRenderedText(model.View().Content)
	assertSavedMarkdownBundlePaths(t, content, reportPath, annexPath)
	if !strings.Contains(content, "Selected Year: 2024") || !strings.Contains(content, "Cost Basis Method: FIFO") || !strings.Contains(content, "Report Base Currency: USD") {
		t.Fatalf("expected selected year, method, and report base currency in result view, got %q", content)
	}

	var openerRequests = readOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != 1 || openerRequests[0] != reportPath {
		t.Fatalf("expected one opener request for %q, got %#v", reportPath, openerRequests)
	}

	var rawReport, err = os.ReadFile(reportPath) // #nosec G304 -- test reads the report path returned by the controlled output fixture.
	if err != nil {
		t.Fatalf("read saved report %q: %v", reportPath, err)
	}
	var reportText = string(rawReport)
	for _, expected := range []string{
		"# Ghostfolio Capital Gains And Losses Report",
		"- **Year:** 2024",
		"- **Cost Basis Method:** FIFO",
		"- **Report Calculation Currency:** USD",
		"## Gains-And-Losses Summary",
		"## Reference Section",
		"| Overall Yearly Net Total |",
	} {
		if !strings.Contains(reportText, expected) {
			t.Fatalf("expected saved report to contain %q, got %q", expected, reportText)
		}
	}
	assertTextOmitted(t, reportText, "token-123", reportPath)
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)

	var updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "sync_reports_menu" {
		t.Fatalf("expected result dismissal to return to sync and reports menu, got %s", model.ActiveScreen())
	}
	var syncReportsContent = normalizeRenderedText(model.View().Content)
	if strings.Contains(syncReportsContent, reportPath) {
		t.Fatalf("expected no report history after result dismissal, got %q", syncReportsContent)
	}

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "report_selection" {
		t.Fatalf("expected unlocked context to reopen report selection without another token prompt, got %s", model.ActiveScreen())
	}
}

// TestReportGenerationWritesSelectedMarkdownAndPDFBundles verifies that the
// same deterministic runtime fixture can be generated as the selected Markdown
// main-plus-annex bundle and as one combined PDF file.
// Authored by: OpenCode
func TestReportGenerationWritesSelectedMarkdownAndPDFBundles(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var fixture = testutil.DeterministicReportLedgerFixture()
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)
	var token = "token-123"

	seedProtectedSnapshot(t, harness, token, fixture.ProtectedActivityCache)
	var contextResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !contextResult.ProtectedData.HasReadableSnapshot {
		t.Fatalf("expected readable snapshot after unlock, got %#v", contextResult)
	}

	var markdownRequest = mustIntegrationReportRequestForFormat(t, fixture.PrimaryReportYear, reportmodel.ReportBaseCurrencyUSD, reportmodel.ReportOutputFormatMarkdown)
	var markdownOutcome = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{Request: markdownRequest})
	if !markdownOutcome.Success {
		t.Fatalf("expected Markdown report generation success, got %#v", markdownOutcome)
	}
	var markdownFiles = mustAllMarkdownFiles(t, reportIO.DocumentsDir)
	if len(markdownFiles) != 2 {
		t.Fatalf("expected main-plus-annex Markdown output, got %#v", markdownFiles)
	}
	var markdownMainPath = selectedMainReportPath(t, markdownFiles, nil, reportmodel.ReportOutputFormatMarkdown)
	//nolint:gosec // Test reads the report path returned by the controlled output fixture.
	var rawMarkdown, readErr = os.ReadFile(markdownMainPath)
	if readErr != nil {
		t.Fatalf("read generated Markdown report %q: %v", markdownMainPath, readErr)
	}

	var pdfRequest = mustIntegrationReportRequestForFormat(t, fixture.PrimaryReportYear, reportmodel.ReportBaseCurrencyUSD, reportmodel.ReportOutputFormatPDF)
	var pdfOutcome = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{Request: pdfRequest})
	if !pdfOutcome.Success {
		t.Fatalf("expected PDF report generation success, got %#v", pdfOutcome)
	}
	var pdfFiles = mustPDFFiles(t, reportIO.DocumentsDir)
	if len(pdfFiles) != 1 {
		t.Fatalf("expected one combined PDF output, got %#v", pdfFiles)
	}
	var rawPDF, pdfReadErr = os.ReadFile(pdfFiles[0])
	if pdfReadErr != nil {
		t.Fatalf("read generated PDF %q: %v", pdfFiles[0], pdfReadErr)
	}
	var inspection, inspectErr = testutil.InspectGeneratedPDF(rawPDF)
	if inspectErr != nil {
		t.Fatalf("inspect generated PDF: %v", inspectErr)
	}
	assertIntegrationLandscapeA4PDF(t, inspection)

	var markdownText = string(rawMarkdown)
	for _, sharedValue := range []string{"Ghostfolio Capital Gains And Losses Report", "Gains-And-Losses Summary", "Overall Yearly Net Total", "ADA", "Same currency"} {
		if !strings.Contains(markdownText, sharedValue) {
			t.Fatalf("expected Markdown from shared protected cache to contain %q, got %q", sharedValue, markdownText)
		}
		if !inspection.ContainsSearchableText(sharedValue) {
			t.Fatalf("expected PDF from the same protected cache to contain shared value %q, got %q", sharedValue, inspection.SearchableText)
		}
	}
	var openerRequests = readOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != 2 {
		t.Fatalf("expected one opener request for each successful output, got %#v", openerRequests)
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// assertIntegrationLandscapeA4PDF verifies every generated integration PDF page
// has the production landscape A4 dimensions.
// Authored by: OpenCode
func assertIntegrationLandscapeA4PDF(t *testing.T, inspection testutil.GeneratedPDF) {
	t.Helper()

	for index, page := range inspection.PageBoxes {
		if page.Width != 842 || page.Height != 595 {
			t.Fatalf("page %d dimensions = %.0fx%.0f, want landscape A4 842x595", index+1, page.Width, page.Height)
		}
	}
}

// TestReportGenerationOpenWarningPreservesSavedReportAndAllowsAnotherRun
// verifies that an opener failure stays non-fatal and keeps the workflow in the
// unlocked report context.
// Authored by: OpenCode
func TestReportGenerationOpenWarningPreservesSavedReportAndAllowsAnotherRun(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 7)
	var fixture = testutil.DeterministicReportLedgerFixture()
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)

	seedProtectedSnapshot(t, harness, "token-123", fixture.ProtectedActivityCache)

	var model = unlockSyncReportsContext(t, harness.Model, "token-123")
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, fixture.PrimaryReportYear)
	model = selectReportBaseCurrency(t, model, reportmodel.ReportBaseCurrencyUSD)
	model, cmd := startReportGenerationAfterBaseCurrencySelection(t, model)
	model = applyBatchCmd(t, model, cmd)

	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
	}

	var files = mustMarkdownFiles(t, reportIO.DocumentsDir)
	if len(files) != 1 {
		t.Fatalf("expected one saved Markdown file after opener warning, got %#v", files)
	}
	var reportPath = files[0]
	testutil.AssertRegularFile(t, reportPath)

	var openerRequests = readOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != 1 || openerRequests[0] != reportPath {
		t.Fatalf("expected one opener request for %q, got %#v", reportPath, openerRequests)
	}

	var content = normalizeRenderedText(model.View().Content)
	if !strings.Contains(content, "Success With Warning: automatic open failed after save") {
		t.Fatalf("expected opener warning headline, got %q", content)
	}
	if !strings.Contains(content, "automatic opening failed") || !strings.Contains(content, "Open the file manually") {
		t.Fatalf("expected actionable opener warning, got %q", content)
	}

	var updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "report_selection" {
		t.Fatalf("expected Generate Another Report to return to report selection, got %s", model.ActiveScreen())
	}
}

// TestReportGenerationSkipsCurrencylessOrderTier verifies the BUG-007
// regression path where later explicit-currency values remain usable when a
// higher-priority order tier carries financial values but no currency label.
// Authored by: OpenCode
func TestReportGenerationSkipsCurrencylessOrderTier(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var fixture = testutil.DeterministicReportLedgerFixture()
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)
	var cache = fixture.ProtectedActivityCache
	var filteredActivities = cache.Activities[:0]
	for _, activity := range cache.Activities {
		if activity.SourceID == "doge-buy-2025-incomplete-001" {
			continue
		}
		filteredActivities = append(filteredActivities, activity)
	}
	cache.Activities = filteredActivities
	cache.ActivityCount = len(filteredActivities)
	cache.RetrievedCount = len(filteredActivities)

	seedProtectedSnapshot(t, harness, "token-123", cache)

	var model = unlockSyncReportsContext(t, harness.Model, "token-123")
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, fixture.CurrencylessOrderReportYear)
	model = selectReportBaseCurrency(t, model, reportmodel.ReportBaseCurrencyEUR)
	model, cmd := startReportGenerationAfterBaseCurrencySelection(t, model)
	model = applyBatchCmd(t, model, cmd)

	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
	}
	var content = normalizeRenderedText(model.View().Content)
	if !strings.Contains(content, "Saved Markdown Path:") {
		t.Fatalf("expected successful report result, got %q", content)
	}

	var files = mustMarkdownFiles(t, reportIO.DocumentsDir)
	if len(files) != 1 {
		t.Fatalf("expected one saved Markdown file, got %#v", files)
	}
	var reportPath = files[0]
	testutil.AssertRegularFile(t, reportPath)

	var openerRequests = readOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != 1 || openerRequests[0] != reportPath {
		t.Fatalf("expected one opener request for %q, got %#v", reportPath, openerRequests)
	}

	var rawReport, err = os.ReadFile(reportPath) // #nosec G304 -- test reads the report path returned by the controlled output fixture.
	if err != nil {
		t.Fatalf("read saved report %q: %v", reportPath, err)
	}
	var reportText = string(rawReport)
	if !strings.Contains(reportText, "| sol-buy-2026-asset-tier-001 | BUY | 50 | 80 | 4000 | 0.5 | 50 | 4000.5 | EUR | EUR |") {
		t.Fatalf("expected saved report to show the later explicit-currency asset tier, got %q", reportText)
	}
	if strings.Contains(reportText, "| sol-buy-2026-asset-tier-001 | BUY | 50 | 81 | 4050 | 1 |") {
		t.Fatalf("expected saved report to skip the currencyless order-tier monetary values, got %q", reportText)
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// TestReportGenerationRoundsSameTierUnitPriceDerivation verifies that
// repeating same-tier division succeeds and the rendered report reuses the
// rounded 16-decimal internal result without extra boundary rounding.
// Authored by: OpenCode
func TestReportGenerationRoundsSameTierUnitPriceDerivation(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)
	var cache = roundedUnitPriceProtectedActivityCache(t)

	seedProtectedSnapshot(t, harness, "token-123", cache)

	var model = unlockSyncReportsContext(t, harness.Model, "token-123")
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, 2024)
	model = selectReportBaseCurrency(t, model, reportmodel.ReportBaseCurrencyUSD)
	model, cmd := startReportGenerationAfterBaseCurrencySelection(t, model)
	model = applyBatchCmd(t, model, cmd)

	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
	}

	var files = mustMarkdownFiles(t, reportIO.DocumentsDir)
	if len(files) != 1 {
		t.Fatalf("expected one saved Markdown file, got %#v", files)
	}
	var reportPath = files[0]
	var openerRequests = readOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != 1 || openerRequests[0] != reportPath {
		t.Fatalf("expected one opener request for %q, got %#v", reportPath, openerRequests)
	}

	//nolint:gosec // Test reads the report path returned by the controlled output fixture.
	var rawReport, err = os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read saved report %q: %v", reportPath, err)
	}
	var reportText = string(rawReport)
	for _, expected := range []string{
		"| unit-buy-2024-001 | BUY | 3 | 0.3333333333333333 | 1 | 0 | 3 | 1 | USD | USD |",
		"| unit-sell-2024-001 | SELL | 1 | 1 | 1 | 0 | 2 | 0.6666666666666667 | USD | USD |",
		"| unit-sell-2024-001 | 1 | 0.3333333333333333 | 1 | 0.6666666666666667 | USD |",
	} {
		if !strings.Contains(reportText, expected) {
			t.Fatalf("expected rounded report to contain %q, got %q", expected, reportText)
		}
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// TestReportGenerationAfterSyncAllowsRepeatingGrossValueOnlyUnitPriceDerivation
// verifies that a raw Ghostfolio activity shape with repeating same-tier
// gross-value-only unit-price derivation survives sync-boundary validation and
// remains reportable through the runtime-backed report workflow.
// Authored by: OpenCode
func TestReportGenerationAfterSyncAllowsRepeatingGrossValueOnlyUnitPriceDerivation(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var baseDir = t.TempDir()
	var server = newTokenAwareStorageServer(t)
	server.SetTokenPages("token-123", []storagePageFixture{{
		Count: 2,
		ActivitiesJSON: `[
			{"id":"repeat-buy-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":3,"valueInBaseCurrency":1,"feeInBaseCurrency":0,"baseCurrency":"USD","SymbolProfile":{"id":"asset-repeat-report-flow-001","symbol":"RPT","name":"Repeat Asset","currency":"USD"}},
			{"id":"repeat-sell-1","date":"2024-02-01T10:00:00Z","type":"SELL","quantity":1,"valueInBaseCurrency":1,"unitPriceInAssetProfileCurrency":1,"feeInBaseCurrency":0,"baseCurrency":"USD","SymbolProfile":{"id":"asset-repeat-report-flow-001","symbol":"RPT","name":"Repeat Asset","currency":"USD"}}
		]`,
	}})

	var syncConfig = mustReportGenerationSyncConfig(t, server.URL())
	var syncService = runtime.NewSyncService(
		ghostfolioclient.New(server.Client()),
		time.Second,
		baseDir,
		true,
		decimalsupport.NewService(),
		syncnormalize.NewNormalizer(),
		syncvalidate.NewValidator(),
		snapshotstore.NewEncryptedStore(baseDir, nil),
	)
	var syncOutcome = syncService.Run(context.Background(), runtime.SyncRequest{Config: syncConfig, SecurityToken: "token-123"})
	if !syncOutcome.Success {
		t.Fatalf("expected sync success for repeating gross-value-only report fixture, got %#v", syncOutcome)
	}

	var harness = newRuntimeBackedFlowHarness(t, baseDir, syncConfig, true)
	var model = unlockSyncReportsContext(t, harness.Model, "token-123")
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, 2024)
	model = selectReportBaseCurrency(t, model, reportmodel.ReportBaseCurrencyUSD)
	model, cmd := startReportGenerationAfterBaseCurrencySelection(t, model)
	model = applyBatchCmd(t, model, cmd)

	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
	}

	var files = mustMarkdownFiles(t, reportIO.DocumentsDir)
	if len(files) != 1 {
		t.Fatalf("expected one saved Markdown file, got %#v", files)
	}
	var reportPath = files[0]
	var openerRequests = readOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != 1 || openerRequests[0] != reportPath {
		t.Fatalf("expected one opener request for %q, got %#v", reportPath, openerRequests)
	}

	var rawReport, err = os.ReadFile(reportPath) // #nosec G304 -- test reads the report path returned by the controlled output fixture.
	if err != nil {
		t.Fatalf("read saved report %q: %v", reportPath, err)
	}
	var reportText = string(rawReport)
	for _, expected := range []string{
		"| repeat-buy-1 | BUY | 3 | 0.3333333333333333 | 1 | 0 | 3 | 1 | USD | USD |",
		"| repeat-sell-1 | SELL | 1 | 1 | 1 | 0 | 2 | 0.6666666666666667 | USD | USD |",
		"| repeat-sell-1 | 1 | 0.3333333333333333 | 1 | 0.6666666666666667 | USD |",
	} {
		if !strings.Contains(reportText, expected) {
			t.Fatalf("expected synced repeating-derivation report to contain %q, got %q", expected, reportText)
		}
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// TestReportGenerationFlowSendsSelectedBaseCurrencyToRuntime verifies that the
// selected USD and EUR report base currencies reach the runtime report service.
// Authored by: OpenCode
func TestReportGenerationFlowSendsSelectedBaseCurrencyToRuntime(t *testing.T) {
	var cases = []reportmodel.ReportBaseCurrency{
		reportmodel.ReportBaseCurrencyUSD,
		reportmodel.ReportBaseCurrencyEUR,
	}

	for _, reportBaseCurrency := range cases {
		var reportBaseCurrency = reportBaseCurrency
		t.Run(reportBaseCurrency.Label(), func(t *testing.T) {
			var capture = &capturingReportService{}
			var harness = newReportCaptureFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false, capture)

			seedProtectedSnapshot(t, harness, "token-123", sameCurrencyRoundedUnitPriceProtectedActivityCache(t, reportBaseCurrency))

			var model = unlockSyncReportsContext(t, harness.Model, "token-123")
			model = openReportSelectionFromContext(t, model)
			model = selectReportYear(t, model, 2024)
			model = selectReportBaseCurrency(t, model, reportBaseCurrency)
			model, cmd := startReportGenerationAfterBaseCurrencySelection(t, model)
			model = applyBatchCmd(t, model, cmd)

			if model.ActiveScreen() != "report_result" {
				t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
			}
			if len(capture.requests) != 1 {
				t.Fatalf("expected one runtime report-generation request, got %#v", capture.requests)
			}
			var request = capture.requests[0]
			if request.Request.ReportBaseCurrency != reportBaseCurrency {
				t.Fatalf("expected runtime request base currency %q, got %#v", reportBaseCurrency, request.Request)
			}
			if request.Request.Year != 2024 || request.Request.CostBasisMethod != reportmodel.CostBasisMethodFIFO {
				t.Fatalf("expected runtime request to preserve year and method, got %#v", request.Request)
			}
		})
	}
}

// TestSameCurrencyReportPreservesPriorMonetaryResults verifies the single-
// currency no-conversion regression path for both supported report base
// currencies.
// Authored by: OpenCode
func TestSameCurrencyReportPreservesPriorMonetaryResults(t *testing.T) {
	var cases = []reportmodel.ReportBaseCurrency{
		reportmodel.ReportBaseCurrencyUSD,
		reportmodel.ReportBaseCurrencyEUR,
	}

	for _, reportBaseCurrency := range cases {
		var reportBaseCurrency = reportBaseCurrency
		t.Run(reportBaseCurrency.Label(), func(t *testing.T) {
			var reportIO = testutil.NewReportIOFixture(t)
			var openLogPath = installOpenCommandRecorder(t, 0)
			var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)
			var token = "token-123"

			seedProtectedSnapshot(t, harness, token, sameCurrencyRoundedUnitPriceProtectedActivityCache(t, reportBaseCurrency))
			var contextResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
			if !contextResult.ProtectedData.HasReadableSnapshot {
				t.Fatalf("expected readable snapshot after unlock, got %#v", contextResult)
			}

			var request, err = reportmodel.NewReportRequest(2024, reportmodel.CostBasisMethodFIFO, reportBaseCurrency, reportmodel.ReportOutputFormatMarkdown, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
			if err != nil {
				t.Fatalf("new report request: %v", err)
			}

			var outcome = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{Request: request})
			if !outcome.Success {
				t.Fatalf("expected same-currency report generation success, got %#v", outcome)
			}

			var files = mustMarkdownFiles(t, reportIO.DocumentsDir)
			if len(files) != 1 {
				t.Fatalf("expected one saved Markdown file, got %#v", files)
			}
			var reportPath = files[0]
			var openerRequests = readOpenCommandRequests(t, openLogPath)
			if len(openerRequests) != 1 || openerRequests[0] != reportPath {
				t.Fatalf("expected one opener request for %q, got %#v", reportPath, openerRequests)
			}

			//nolint:gosec // Test reads the report path returned by the controlled output fixture.
			var rawReport, readErr = os.ReadFile(reportPath)
			if readErr != nil {
				t.Fatalf("read saved report %q: %v", reportPath, readErr)
			}
			var reportText = string(rawReport)
			var label = reportBaseCurrency.Label()
			for _, expected := range []string{
				"- **Report Calculation Currency:** " + label,
				"| unit-buy-2024-001 | BUY | 3 | 0.3333333333333333 | 1 | 0 | 3 | 1 | " + label + " | " + label + " |",
				"| unit-sell-2024-001 | SELL | 1 | 1 | 1 | 0 | 2 | 0.6666666666666667 | " + label + " | " + label + " |",
				"| unit-sell-2024-001 | 1 | 0.3333333333333333 | 1 | 0.6666666666666667 | " + label + " |",
			} {
				if !strings.Contains(reportText, expected) {
					t.Fatalf("expected same-currency report to contain %q, got %q", expected, reportText)
				}
			}
			assertNoCleartextReportInAppStorage(t, harness.BaseDir)
		})
	}
}

// TestReportGenerationConvertsDeterministicMixedCurrencyFixture verifies the US2
// report-generation path for a mixed official-rate fixture that spans both
// supported base-currency providers and previous-available-rate fallback cases.
// Authored by: OpenCode
func TestReportGenerationConvertsDeterministicMixedCurrencyFixture(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)
	var token = "mixed-currency-token"
	var cache = mixedCurrencyConversionProtectedActivityCache(t, 54)

	if cache.ActivityCount < 50 {
		t.Fatalf("expected at least 50 priced activities, got %d", cache.ActivityCount)
	}
	seedProtectedSnapshot(t, harness, token, cache)

	var contextResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !contextResult.ProtectedData.HasReadableSnapshot {
		t.Fatalf("expected readable snapshot after unlock, got %#v", contextResult)
	}

	var eurRequest = mustIntegrationReportRequest(t, 2024, reportmodel.ReportBaseCurrencyEUR)
	var eurOutcome = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{Request: eurRequest})
	if !eurOutcome.Success {
		t.Fatalf("expected EUR report conversion success for ECB division fixture, got %#v", eurOutcome)
	}

	var usdRequest = mustIntegrationReportRequest(t, 2025, reportmodel.ReportBaseCurrencyUSD)
	var usdOutcome = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{Request: usdRequest})
	if !usdOutcome.Success {
		t.Fatalf("expected USD report conversion success for H.10 division and multiplication fixture, got %#v", usdOutcome)
	}

	var files = mustMarkdownFiles(t, reportIO.DocumentsDir)
	if len(files) != 2 {
		t.Fatalf("expected two saved mixed-currency Markdown reports, got %#v", files)
	}
	var openerRequests = readOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != 2 {
		t.Fatalf("expected two opener requests, got %#v", openerRequests)
	}

	var combinedReportText strings.Builder
	for _, reportPath := range files {
		//nolint:gosec // Test reads report paths returned by the controlled output fixture.
		var rawReport, err = os.ReadFile(reportPath)
		if err != nil {
			t.Fatalf("read saved report %q: %v", reportPath, err)
		}
		combinedReportText.Write(rawReport)
		combinedReportText.WriteString("\n")
	}
	var reportText = combinedReportText.String()
	for _, expected := range []string{
		"- **Report Calculation Currency:** EUR",
		"- **Report Calculation Currency:** USD",
		"ECB Data Portal `EXR`",
		"Federal Reserve Board H.10/Data Download Program",
		"mixed-usd-buy-2024-000",
		"mixed-eur-buy-2025-001",
		"mixed-gbp-buy-2025-005",
		"2024-01-06",
	} {
		if !strings.Contains(reportText, expected) {
			t.Fatalf("expected mixed-currency reports to contain %q, got %q", expected, reportText)
		}
	}
	for _, forbidden := range []string{"## Currency Conversion Audit", "source_per_base", "base_per_source"} {
		if strings.Contains(reportText, forbidden) {
			t.Fatalf("expected mixed-currency main reports to omit %q, got %q", forbidden, reportText)
		}
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// TestReportGenerationUsesPreservedOffsetSourceCalendarDateForRateSelection
// verifies source-calendar rate lookup dates when the preserved activity offset
// date differs from the corresponding UTC date.
// Authored by: OpenCode
func TestReportGenerationUsesPreservedOffsetSourceCalendarDateForRateSelection(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)
	var token = "offset-calendar-token"

	seedProtectedSnapshot(t, harness, token, offsetSensitiveCurrencyProtectedActivityCache(t))

	var contextResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !contextResult.ProtectedData.HasReadableSnapshot {
		t.Fatalf("expected readable snapshot after unlock, got %#v", contextResult)
	}

	var request = mustIntegrationReportRequest(t, 2024, reportmodel.ReportBaseCurrencyUSD)
	var outcome = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{Request: request})
	if !outcome.Success {
		t.Fatalf("expected offset-calendar report conversion success, got %#v", outcome)
	}

	var files = mustMarkdownFiles(t, reportIO.DocumentsDir)
	if len(files) != 1 {
		t.Fatalf("expected one saved offset-calendar Markdown report, got %#v", files)
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
		"offset-before-utc-buy",
		"offset-after-utc-buy",
		"2024-01-01",
		"2024-01-02",
		"Federal Reserve Board H.10/Data Download Program",
	} {
		if !strings.Contains(reportText, expected) {
			t.Fatalf("expected offset-calendar report to contain %q, got %q", expected, reportText)
		}
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// roundedUnitPriceProtectedActivityCache builds one deterministic cache where a
// same-tier unit-price derivation requires repeating decimal rounding.
// Authored by: OpenCode
func roundedUnitPriceProtectedActivityCache(t *testing.T) syncmodel.ProtectedActivityCache {
	t.Helper()

	return syncmodel.ProtectedActivityCache{
		SyncedAt:             mustReportFixtureTime(t),
		RetrievedCount:       2,
		ActivityCount:        2,
		AvailableReportYears: []int{2024},
		Activities: []syncmodel.ActivityRecord{
			roundedReportActivity(t, roundedReportActivityInput{
				SourceID:         "unit-buy-2024-001",
				OccurredAt:       "2024-01-01T10:00:00Z",
				ActivityType:     syncmodel.ActivityTypeBuy,
				AssetIdentityKey: "asset-unit-001",
				AssetSymbol:      "UNIT",
				AssetName:        "Unit Asset",
				Quantity:         "3",
				OrderCurrency:    "USD",
				OrderGrossValue:  "1",
				OrderFeeAmount:   "0",
			}),
			roundedReportActivity(t, roundedReportActivityInput{
				SourceID:         "unit-sell-2024-001",
				OccurredAt:       "2024-03-01T10:00:00Z",
				ActivityType:     syncmodel.ActivityTypeSell,
				AssetIdentityKey: "asset-unit-001",
				AssetSymbol:      "UNIT",
				AssetName:        "Unit Asset",
				Quantity:         "1",
				OrderCurrency:    "USD",
				OrderGrossValue:  "1",
				OrderFeeAmount:   "0",
				OrderUnitPrice:   "1",
			}),
		},
	}
}

// sameCurrencyRoundedUnitPriceProtectedActivityCache returns the rounded
// regression fixture denominated entirely in the selected report base currency.
// Authored by: OpenCode
func sameCurrencyRoundedUnitPriceProtectedActivityCache(t *testing.T, reportBaseCurrency reportmodel.ReportBaseCurrency) syncmodel.ProtectedActivityCache {
	t.Helper()

	var cache = roundedUnitPriceProtectedActivityCache(t)
	for index := range cache.Activities {
		cache.Activities[index].OrderCurrency = reportBaseCurrency.Label()
	}

	return cache
}

// mixedCurrencyConversionProtectedActivityCache builds a deterministic priced
// activity fixture with USD, EUR, and GBP source currencies across two report
// years.
// Authored by: OpenCode
func mixedCurrencyConversionProtectedActivityCache(t *testing.T, activityCount int) syncmodel.ProtectedActivityCache {
	t.Helper()

	var activities = make([]syncmodel.ActivityRecord, 0, activityCount)
	var currencies = []string{"USD", "EUR", "GBP"}
	for index := 0; index < activityCount; index++ {
		var year = 2024
		if index%2 == 1 {
			year = 2025
		}
		var currency = currencies[index%len(currencies)]
		var date = time.Date(year, time.January, 2+(index%24), 10, 0, 0, 0, time.FixedZone("source", (index%5-2)*60*60))
		if index == 6 {
			date = time.Date(2024, time.January, 6, 11, 0, 0, 0, time.UTC)
		}

		activities = append(activities, roundedReportActivity(t, roundedReportActivityInput{
			SourceID:         mixedCurrencySourceID(currency, year, index),
			OccurredAt:       date.Format(time.RFC3339),
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-mixed-001",
			AssetSymbol:      "MIX",
			AssetName:        "Mixed Currency Asset",
			Quantity:         "1",
			OrderCurrency:    currency,
			OrderUnitPrice:   "10",
			OrderGrossValue:  "10",
			OrderFeeAmount:   "1",
		}))
	}

	return syncmodel.ProtectedActivityCache{
		SyncedAt:             mustReportFixtureTime(t),
		RetrievedCount:       len(activities),
		ActivityCount:        len(activities),
		AvailableReportYears: []int{2024, 2025},
		Activities:           activities,
	}
}

// offsetSensitiveCurrencyProtectedActivityCache builds two activities whose UTC
// dates differ from their preserved source-offset calendar dates.
// Authored by: OpenCode
func offsetSensitiveCurrencyProtectedActivityCache(t *testing.T) syncmodel.ProtectedActivityCache {
	t.Helper()

	var activities = []syncmodel.ActivityRecord{
		roundedReportActivity(t, roundedReportActivityInput{
			SourceID:         "offset-before-utc-buy",
			OccurredAt:       "2024-01-01T23:30:00-02:00",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-offset-001",
			AssetSymbol:      "OFF",
			AssetName:        "Offset Asset",
			Quantity:         "1",
			OrderCurrency:    "EUR",
			OrderUnitPrice:   "10",
			OrderGrossValue:  "10",
			OrderFeeAmount:   "1",
		}),
		roundedReportActivity(t, roundedReportActivityInput{
			SourceID:         "offset-after-utc-buy",
			OccurredAt:       "2024-01-02T00:30:00+02:00",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-offset-001",
			AssetSymbol:      "OFF",
			AssetName:        "Offset Asset",
			Quantity:         "1",
			OrderCurrency:    "GBP",
			OrderUnitPrice:   "10",
			OrderGrossValue:  "10",
			OrderFeeAmount:   "1",
		}),
	}

	return syncmodel.ProtectedActivityCache{
		SyncedAt:             mustReportFixtureTime(t),
		RetrievedCount:       len(activities),
		ActivityCount:        len(activities),
		AvailableReportYears: []int{2024},
		Activities:           activities,
	}
}

// mixedCurrencySourceID returns a deterministic activity reference for the
// mixed-currency conversion fixture.
// Authored by: OpenCode
func mixedCurrencySourceID(currency string, year int, index int) string {
	return strings.ToLower("mixed-"+currency+"-buy-") + strconv.Itoa(year) + "-" + leftPadThree(index)
}

// leftPadThree renders one small deterministic fixture index.
// Authored by: OpenCode
func leftPadThree(value int) string {
	var raw = strconv.Itoa(value)
	for len(raw) < 3 {
		raw = "0" + raw
	}

	return raw
}

// mustIntegrationReportRequest creates one validated report request for
// integration conversion tests.
// Authored by: OpenCode
func mustIntegrationReportRequest(t *testing.T, year int, reportBaseCurrency reportmodel.ReportBaseCurrency) reportmodel.ReportRequest {
	t.Helper()

	var request, err = reportmodel.NewReportRequest(
		year,
		reportmodel.CostBasisMethodFIFO,
		reportBaseCurrency,
		reportmodel.ReportOutputFormatMarkdown,
		time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("new integration report request: %v", err)
	}

	return request
}

// mustIntegrationReportRequestForFormat creates one validated report request for
// integration tests that exercise a specific output format.
// Authored by: OpenCode
func mustIntegrationReportRequestForFormat(t *testing.T, year int, _ reportmodel.ReportBaseCurrency, outputFormat reportmodel.ReportOutputFormat) reportmodel.ReportRequest {
	t.Helper()

	var request, err = reportmodel.NewReportRequest(
		year,
		reportmodel.CostBasisMethodFIFO,
		reportmodel.ReportBaseCurrencyUSD,
		outputFormat,
		time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("new integration report request for %s: %v", outputFormat, err)
	}

	return request
}

// selectedMainReportPath returns the main output path for the selected report
// format in integration assertions.
// Authored by: OpenCode
func selectedMainReportPath(t *testing.T, markdownFiles []string, pdfFiles []string, outputFormat reportmodel.ReportOutputFormat) string {
	t.Helper()

	switch outputFormat {
	case reportmodel.ReportOutputFormatMarkdown:
		if len(markdownFiles) == 0 {
			t.Fatalf("expected at least one Markdown main report")
		}
		return markdownFiles[0]
	case reportmodel.ReportOutputFormatPDF:
		if len(pdfFiles) == 0 {
			t.Fatalf("expected at least one PDF report")
		}
		return pdfFiles[0]
	default:
		t.Fatalf("unsupported report output format %q", outputFormat)
		return ""
	}
}

// mustPDFFiles returns all generated PDF files in one directory.
// Authored by: OpenCode
func mustPDFFiles(t *testing.T, dir string) []string {
	t.Helper()

	var entries, err = os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %q: %v", dir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".pdf") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}

	return files
}

// capturingReportService records runtime report-generation requests received
// from the TUI flow.
// Authored by: OpenCode
type capturingReportService struct {
	requests []runtime.ReportGenerationRequest
}

// Generate records one runtime report-generation request and returns a successful
// transient outcome for request-propagation assertions.
// Authored by: OpenCode
func (service *capturingReportService) Generate(_ context.Context, request runtime.ReportGenerationRequest) runtime.ReportOutcome {
	service.requests = append(service.requests, request)
	return runtime.ReportOutcome{
		Success:       true,
		Message:       "captured report generation request",
		FailureReason: runtime.ReportFailureNone,
		Request:       request.Request,
	}
}

// newReportCaptureFlowHarness creates a runtime-backed flow harness with a
// capturing report service replacing final report generation.
// Authored by: OpenCode
func newReportCaptureFlowHarness(
	t *testing.T,
	baseDir string,
	config configmodel.AppSetupConfig,
	allowDevHTTP bool,
	reportService runtime.ReportService,
) runtimeBackedFlowHarness {
	t.Helper()

	var options = bootstrap.DefaultOptions()
	options.ConfigDir = baseDir
	options.AllowDevHTTP = allowDevHTTP

	var app, err = runtime.New(options)
	if err != nil {
		t.Fatalf("runtime new: %v", err)
	}

	var store = configstore.NewJSONStore(baseDir)
	if err := store.Save(context.Background(), config); err != nil {
		t.Fatalf("save setup config: %v", err)
	}

	var model = flow.NewModel(flow.Dependencies{
		Options:       options,
		Startup:       bootstrap.StartupState{ActiveConfig: &config},
		SetupService:  app.SetupService,
		SyncService:   app.SyncService,
		ReportService: reportService,
	})

	return runtimeBackedFlowHarness{
		BaseDir: baseDir,
		App:     app,
		Config:  config,
		Store:   store,
		Model:   model,
	}
}

// selectReportBaseCurrency moves focus to the report base-currency list and
// selects the requested currency.
// Authored by: OpenCode
func selectReportBaseCurrency(t *testing.T, model *flow.Model, reportBaseCurrency reportmodel.ReportBaseCurrency) *flow.Model {
	t.Helper()

	for focusStep := 0; focusStep < 2; focusStep++ {
		var updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
		model = assertFlowModel(t, updated)
	}

	var marker = "> " + reportBaseCurrency.Label()
	for attempt := 0; attempt < len(reportmodel.SupportedReportBaseCurrencies())+1; attempt++ {
		var content = normalizeRenderedText(model.View().Content)
		if strings.Contains(content, "Report Base Currency") && strings.Contains(content, marker) {
			return model
		}

		var updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		model = assertFlowModel(t, updated)
	}

	t.Fatalf("expected report base currency %q to be selected, got %q", reportBaseCurrency.Label(), model.View().Content)
	return model
}

// selectReportBaseCurrencyFromMethodFocus moves from method focus to the report
// base-currency list and selects the requested currency.
// Authored by: OpenCode
func selectReportBaseCurrencyFromMethodFocus(t *testing.T, model *flow.Model, reportBaseCurrency reportmodel.ReportBaseCurrency) *flow.Model {
	t.Helper()

	var updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	model = assertFlowModel(t, updated)

	var marker = "> " + reportBaseCurrency.Label()
	for attempt := 0; attempt < len(reportmodel.SupportedReportBaseCurrencies())+1; attempt++ {
		var content = normalizeRenderedText(model.View().Content)
		if strings.Contains(content, "Report Base Currency") && strings.Contains(content, marker) {
			return model
		}

		updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		model = assertFlowModel(t, updated)
	}

	t.Fatalf("expected report base currency %q to be selected from method focus, got %q", reportBaseCurrency.Label(), model.View().Content)
	return model
}

// startReportGenerationAfterBaseCurrencySelection advances from base-currency
// focus to generation and returns the asynchronous report command.
// Authored by: OpenCode
func startReportGenerationAfterBaseCurrencySelection(t *testing.T, model *flow.Model) (*flow.Model, tea.Cmd) {
	t.Helper()

	for attempt := 0; attempt < 4; attempt++ {
		var updated tea.Model
		var cmd tea.Cmd
		updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
		model = assertFlowModel(t, updated)

		if model.ActiveScreen() == "report_busy" {
			return model, cmd
		}
	}

	t.Fatalf("expected report busy screen after base-currency selection, got %s", model.ActiveScreen())
	return model, nil
}

// roundedReportActivityInput stores one compact rounded-division integration
// fixture before conversion into a normalized activity record.
// Authored by: OpenCode
type roundedReportActivityInput struct {
	SourceID         string
	OccurredAt       string
	ActivityType     syncmodel.ActivityType
	AssetIdentityKey string
	AssetSymbol      string
	AssetName        string
	Quantity         string
	OrderCurrency    string
	OrderUnitPrice   string
	OrderGrossValue  string
	OrderFeeAmount   string
}

// roundedReportActivity converts one compact rounded-division integration
// fixture into the normalized activity record used by the runtime flow.
// Authored by: OpenCode
func roundedReportActivity(t *testing.T, input roundedReportActivityInput) syncmodel.ActivityRecord {
	t.Helper()

	return syncmodel.ActivityRecord{
		SourceID:         input.SourceID,
		OccurredAt:       input.OccurredAt,
		ActivityType:     input.ActivityType,
		AssetIdentityKey: input.AssetIdentityKey,
		AssetSymbol:      input.AssetSymbol,
		AssetName:        input.AssetName,
		Quantity:         mustRoundedIntegrationDecimal(t, input.Quantity),
		OrderCurrency:    input.OrderCurrency,
		OrderUnitPrice:   roundedIntegrationDecimalPointer(t, input.OrderUnitPrice),
		OrderGrossValue:  roundedIntegrationDecimalPointer(t, input.OrderGrossValue),
		OrderFeeAmount:   roundedIntegrationDecimalPointer(t, input.OrderFeeAmount),
	}
}

// mustReportGenerationSyncConfig returns one custom-origin config for sync then
// runtime-backed report generation within the same base directory.
// Authored by: OpenCode
func mustReportGenerationSyncConfig(t *testing.T, origin string) configmodel.AppSetupConfig {
	t.Helper()

	config, err := configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, origin, true, time.Now())
	if err != nil {
		t.Fatalf("new report-generation sync config: %v", err)
	}

	return config
}

// mustReportFixtureTime parses one RFC3339 fixture timestamp for integration
// caches.
// Authored by: OpenCode
func mustReportFixtureTime(t *testing.T) time.Time {
	t.Helper()

	const raw = "2026-05-20T15:04:05Z"

	var parsed, err = time.Parse(time.RFC3339, raw)
	if err != nil {
		t.Fatalf("parse report fixture time %q: %v", raw, err)
	}

	return parsed
}

// roundedIntegrationDecimalPointer parses one optional decimal fixture for
// rounded-division integration tests.
// Authored by: OpenCode
func roundedIntegrationDecimalPointer(t *testing.T, raw string) *apd.Decimal {
	t.Helper()

	if raw == "" {
		return nil
	}

	var value = mustRoundedIntegrationDecimal(t, raw)
	return &value
}

// mustRoundedIntegrationDecimal parses one decimal fixture for rounded-
// division integration tests.
// Authored by: OpenCode
func mustRoundedIntegrationDecimal(t *testing.T, raw string) apd.Decimal {
	t.Helper()

	var value, _, err = decimalsupport.ParseString(raw)
	if err != nil {
		t.Fatalf("parse rounded integration decimal %q: %v", raw, err)
	}

	return value
}

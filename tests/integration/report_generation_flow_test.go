// Package integration verifies black-box workflow behavior for the current
// slice, including runtime-backed report generation flows.
// Authored by: OpenCode
package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	syncvalidate "github.com/benizzio/ghostfolio-cryptogains/internal/sync/validate"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil/runtimeflow"
)

// TestReportGenerationSuccessWritesMarkdownAndReturnsToUnlockedContext verifies
// normal end-to-end Markdown and PDF saves through the real runtime-backed workflow.
// Authored by: OpenCode
func TestReportGenerationSuccessWritesMarkdownAndReturnsToUnlockedContext(t *testing.T) {
	for _, selectedFormat := range reportmodel.SupportedReportOutputFormats() {
		var selectedFormat = selectedFormat
		t.Run(selectedFormat.Label(), func(t *testing.T) {
			var reportIO = testutil.NewReportIOFixture(t)
			var openLogPath = runtimeflow.InstallOpenCommandRecorder(t, 0)
			var fixture = testutil.DeterministicReportLedgerFixture()
			var harness = runtimeflow.NewRuntimeBackedFlowHarness(t, t.TempDir(), runtimeflow.MustCloudSetupConfig(t), false)

			runtimeflow.SeedProtectedSnapshot(t, harness, "token-123", fixture.ProtectedActivityCache)

			var model = runtimeflow.UnlockSyncReportsContext(t, harness.Model, "token-123")
			model = runtimeflow.OpenReportSelection(t, model)
			model = runtimeflow.SelectReportYear(t, model, fixture.PrimaryReportYear)
			model = runtimeflow.SelectReportBaseCurrency(t, model, reportmodel.ReportBaseCurrencyUSD)
			model = runtimeflow.SelectReportOutputFormat(t, model, selectedFormat)
			model, cmd := runtimeflow.StartReportGeneration(t, model)
			model = runtimeflow.ApplyBatchCmd(t, model, cmd)

			if model.ActiveScreen() != "report_result" {
				t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
			}

			var files = runtimeflow.ReportOutputPaths(t, reportIO.DocumentsDir, selectedFormat)
			for _, path := range files {
				testutil.AssertPathWithin(t, path, reportIO.DocumentsDir)
				testutil.AssertRegularFile(t, path)
			}
			var reportPath = files[0]
			if selectedFormat == reportmodel.ReportOutputFormatMarkdown {
				var annexPath string
				reportPath, annexPath = runtimeflow.MarkdownBundlePaths(t, files)
				if !strings.HasPrefix(filepath.Base(reportPath), "ghostfolio-capital-gains-2024-fifo-") {
					t.Fatalf("expected FIFO report filename slug, got %q", filepath.Base(reportPath))
				}
				if !strings.HasPrefix(filepath.Base(annexPath), "ghostfolio-capital-gains-2024-fifo-annex-1-") {
					t.Fatalf("expected FIFO annex filename slug, got %q", filepath.Base(annexPath))
				}

				// #nosec G304 -- reportPath is created in the test-owned Documents fixture.
				var rawReport, err = os.ReadFile(reportPath)
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
			} else {
				// #nosec G304 -- reportPath is created in the test-owned Documents fixture.
				var rawPDF, err = os.ReadFile(reportPath)
				if err != nil {
					t.Fatalf("read saved PDF %q: %v", reportPath, err)
				}
				var inspection, inspectErr = testutil.InspectGeneratedPDF(rawPDF)
				if inspectErr != nil {
					t.Fatalf("inspect generated PDF: %v", inspectErr)
				}
				runtimeflow.AssertLandscapeA4PDF(t, inspection)
				if !inspection.ContainsSearchableText("Gains-And-Losses Summary") {
					t.Fatalf("expected generated PDF to contain the report summary")
				}
			}

			var content = runtimeflow.ReportResultText(t, model)
			runtimeflow.AssertReportResultDisclosure(t, content, selectedFormat, files)
			if !strings.Contains(content, "Selected Year: 2024") || !strings.Contains(content, "Cost Basis Method: FIFO") || !strings.Contains(content, "Report Base Currency: USD") || !strings.Contains(content, "Output Format: "+selectedFormat.Label()) {
				t.Fatalf("expected selected report settings in result view, got %q", content)
			}
			if strings.Count(content, "Report saved successfully and automatic opening was requested.") != 1 {
				t.Fatalf("expected one runtime operational success message, got %q", content)
			}

			var openerRequests = runtimeflow.ReadOpenCommandRequests(t, openLogPath)
			if len(openerRequests) != 1 || openerRequests[0] != reportPath {
				t.Fatalf("expected one opener request for %q, got %#v", reportPath, openerRequests)
			}
			runtimeflow.AssertNoCleartextReportInAppStorage(t, harness.BaseDir)

			var updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
			model = runtimeflow.AssertFlowModel(t, updated)
			if model.ActiveScreen() != "sync_reports_menu" {
				t.Fatalf("expected result dismissal to return to sync and reports menu, got %s", model.ActiveScreen())
			}
			runtimeflow.AssertReportResultCleared(t, runtimeflow.NormalizeRenderedText(model.View().Content), files)

			updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
			model = runtimeflow.AssertFlowModel(t, updated)
			if model.ActiveScreen() != "report_selection" {
				t.Fatalf("expected unlocked context to reopen report selection without another token prompt, got %s", model.ActiveScreen())
			}
		})
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

	var markdownRequest = runtimeflow.MustIntegrationReportRequestForFormat(t, fixture.PrimaryReportYear, reportmodel.ReportOutputFormatMarkdown)
	var markdownOutcome = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{Request: markdownRequest})
	if !markdownOutcome.Success {
		t.Fatalf("expected Markdown report generation success, got %#v", markdownOutcome)
	}
	var markdownFiles = runtimeflow.AllMarkdownFiles(t, reportIO.DocumentsDir)
	if len(markdownFiles) != 2 {
		t.Fatalf("expected main-plus-annex Markdown output, got %#v", markdownFiles)
	}
	var markdownMainPath = runtimeflow.SelectedMainReportPath(t, markdownFiles, nil, reportmodel.ReportOutputFormatMarkdown)
	//nolint:gosec // Test reads the report path returned by the controlled output fixture.
	var rawMarkdown, readErr = os.ReadFile(markdownMainPath)
	if readErr != nil {
		t.Fatalf("read generated Markdown report %q: %v", markdownMainPath, readErr)
	}

	var pdfRequest = runtimeflow.MustIntegrationReportRequestForFormat(t, fixture.PrimaryReportYear, reportmodel.ReportOutputFormatPDF)
	var pdfOutcome = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{Request: pdfRequest})
	if !pdfOutcome.Success {
		t.Fatalf("expected PDF report generation success, got %#v", pdfOutcome)
	}
	var pdfFiles = runtimeflow.PDFFiles(t, reportIO.DocumentsDir)
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
	runtimeflow.AssertLandscapeA4PDF(t, inspection)

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

// TestReportGenerationOpenWarningPreservesSavedReportAndAllowsAnotherRun
// verifies that Markdown and PDF opener failures stay non-fatal, retain files,
// and keep the workflow in the unlocked report context.
// Authored by: OpenCode
func TestReportGenerationOpenWarningPreservesSavedReportAndAllowsAnotherRun(t *testing.T) {
	for _, selectedFormat := range reportmodel.SupportedReportOutputFormats() {
		var selectedFormat = selectedFormat
		t.Run(selectedFormat.Label(), func(t *testing.T) {
			var reportIO = testutil.NewReportIOFixture(t)
			var openLogPath = runtimeflow.InstallOpenCommandRecorder(t, 7)
			var fixture = testutil.DeterministicReportLedgerFixture()
			var harness = runtimeflow.NewRuntimeBackedFlowHarness(t, t.TempDir(), runtimeflow.MustCloudSetupConfig(t), false)

			runtimeflow.SeedProtectedSnapshot(t, harness, "token-123", fixture.ProtectedActivityCache)

			var model = runtimeflow.UnlockSyncReportsContext(t, harness.Model, "token-123")
			model = runtimeflow.OpenReportSelection(t, model)
			model = runtimeflow.SelectReportYear(t, model, fixture.PrimaryReportYear)
			model = runtimeflow.SelectReportBaseCurrency(t, model, reportmodel.ReportBaseCurrencyUSD)
			model = runtimeflow.SelectReportOutputFormat(t, model, selectedFormat)
			model, cmd := runtimeflow.StartReportGeneration(t, model)
			model = runtimeflow.ApplyBatchCmd(t, model, cmd)

			if model.ActiveScreen() != "report_result" {
				t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
			}

			var files = runtimeflow.ReportOutputPaths(t, reportIO.DocumentsDir, selectedFormat)
			for _, path := range files {
				testutil.AssertRegularFile(t, path)
			}
			var reportPath = files[0]
			if selectedFormat == reportmodel.ReportOutputFormatMarkdown {
				reportPath, _ = runtimeflow.MarkdownBundlePaths(t, files)
			}

			var content = runtimeflow.ReportResultText(t, model)
			runtimeflow.AssertReportResultDisclosure(t, content, selectedFormat, files)
			if !strings.Contains(content, "Output Format: "+selectedFormat.Label()) {
				t.Fatalf("expected selected output format in result view, got %q", content)
			}
			if !strings.Contains(content, "Success With Warning: automatic open failed after save") || !strings.Contains(content, "automatic opening failed") || !strings.Contains(content, "Open the file manually") {
				t.Fatalf("expected actionable opener warning, got %q", content)
			}
			if strings.Count(content, "Report saved successfully, but automatic opening failed:") != 1 {
				t.Fatalf("expected one runtime operational opener warning, got %q", content)
			}

			var openerRequests = runtimeflow.ReadOpenCommandRequests(t, openLogPath)
			if len(openerRequests) != 1 || openerRequests[0] != reportPath {
				t.Fatalf("expected one opener request for %q, got %#v", reportPath, openerRequests)
			}

			var updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
			model = runtimeflow.AssertFlowModel(t, updated)
			updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
			model = runtimeflow.AssertFlowModel(t, updated)
			if model.ActiveScreen() != "report_selection" {
				t.Fatalf("expected Generate Another Report to return to report selection, got %s", model.ActiveScreen())
			}
			runtimeflow.AssertReportResultCleared(t, runtimeflow.NormalizeRenderedText(model.View().Content), files)
			for _, path := range files {
				testutil.AssertRegularFile(t, path)
			}
			runtimeflow.AssertNoCleartextReportInAppStorage(t, harness.BaseDir)
		})
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
	model, cmd := runtimeflow.StartReportGeneration(t, model)
	model = applyBatchCmd(t, model, cmd)

	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
	}
	var content = runtimeflow.ReportResultText(t, model)
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

	// #nosec G304 -- reportPath is created in the test-owned Documents fixture.
	var rawReport, err = os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read saved report %q: %v", reportPath, err)
	}
	var reportText = string(rawReport)
	if !strings.Contains(reportText, "| sol-buy-2026-asset-tier-001 | BUY | 50 | 80.00 | 4000.00 | 0.50 | 50 | 4000.50 | EUR | EUR |") {
		t.Fatalf("expected saved report to show the later explicit-currency asset tier, got %q", reportText)
	}
	if strings.Contains(reportText, "| sol-buy-2026-asset-tier-001 | BUY | 50 | 81.00 | 4050.00 | 1.00 |") {
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
	var cache = runtimeflow.RoundedUnitPriceProtectedActivityCache(t)

	seedProtectedSnapshot(t, harness, "token-123", cache)

	var model = unlockSyncReportsContext(t, harness.Model, "token-123")
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, 2024)
	model = selectReportBaseCurrency(t, model, reportmodel.ReportBaseCurrencyUSD)
	model, cmd := runtimeflow.StartReportGeneration(t, model)
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

	// #nosec G304 -- reportPath is created in the test-owned Documents fixture.
	var rawReport, err = os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read saved report %q: %v", reportPath, err)
	}
	var reportText = string(rawReport)
	for _, expected := range []string{
		"| unit-buy-2024-001 | BUY | 3 | 0.33 | 1.00 | 0.00 | 3 | 1.00 | USD | USD |",
		"| unit-sell-2024-001 | SELL | 1 | 1.00 | 1.00 | 0.00 | 2 | 0.67 | USD | USD |",
		"| unit-sell-2024-001 | 1 | 0.33 | 1.00 | 0.67 | USD |",
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

	var syncConfig = runtimeflow.MustReportGenerationSyncConfig(t, server.URL())
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
	model, cmd := runtimeflow.StartReportGeneration(t, model)
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

	// #nosec G304 -- reportPath is created in the test-owned Documents fixture.
	var rawReport, err = os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read saved report %q: %v", reportPath, err)
	}
	var reportText = string(rawReport)
	for _, expected := range []string{
		"| repeat-buy-1 | BUY | 3 | 0.33 | 1.00 | 0.00 | 3 | 1.00 | USD | USD |",
		"| repeat-sell-1 | SELL | 1 | 1.00 | 1.00 | 0.00 | 2 | 0.67 | USD | USD |",
		"| repeat-sell-1 | 1 | 0.33 | 1.00 | 0.67 | USD |",
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

			seedProtectedSnapshot(t, harness, "token-123", runtimeflow.SameCurrencyRoundedUnitPriceProtectedActivityCache(t, reportBaseCurrency))

			var model = unlockSyncReportsContext(t, harness.Model, "token-123")
			model = openReportSelectionFromContext(t, model)
			model = selectReportYear(t, model, 2024)
			model = selectReportBaseCurrency(t, model, reportBaseCurrency)
			model, cmd := runtimeflow.StartReportGeneration(t, model)
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

			seedProtectedSnapshot(t, harness, token, runtimeflow.SameCurrencyRoundedUnitPriceProtectedActivityCache(t, reportBaseCurrency))
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

			// #nosec G304 -- reportPath is created in the test-owned Documents fixture.
			var rawReport, readErr = os.ReadFile(reportPath)
			if readErr != nil {
				t.Fatalf("read saved report %q: %v", reportPath, readErr)
			}
			var reportText = string(rawReport)
			var label = reportBaseCurrency.Label()
			for _, expected := range []string{
				"- **Report Calculation Currency:** " + label,
				"| unit-buy-2024-001 | BUY | 3 | 0.33 | 1.00 | 0.00 | 3 | 1.00 | " + label + " | " + label + " |",
				"| unit-sell-2024-001 | SELL | 1 | 1.00 | 1.00 | 0.00 | 2 | 0.67 | " + label + " | " + label + " |",
				"| unit-sell-2024-001 | 1 | 0.33 | 1.00 | 0.67 | " + label + " |",
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
	var cache = runtimeflow.MixedCurrencyConversionProtectedActivityCache(t, 54)

	if cache.ActivityCount < 50 {
		t.Fatalf("expected at least 50 priced activities, got %d", cache.ActivityCount)
	}
	seedProtectedSnapshot(t, harness, token, cache)

	var contextResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !contextResult.ProtectedData.HasReadableSnapshot {
		t.Fatalf("expected readable snapshot after unlock, got %#v", contextResult)
	}

	var eurRequest = runtimeflow.MustIntegrationReportRequest(t, 2024, reportmodel.ReportBaseCurrencyEUR)
	var eurOutcome = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{Request: eurRequest})
	if !eurOutcome.Success {
		t.Fatalf("expected EUR report conversion success for ECB division fixture, got %#v", eurOutcome)
	}

	var usdRequest = runtimeflow.MustIntegrationReportRequest(t, 2025, reportmodel.ReportBaseCurrencyUSD)
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
		// #nosec G304 -- reportPath is created in the test-owned Documents fixture.
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

	seedProtectedSnapshot(t, harness, token, runtimeflow.OffsetSensitiveCurrencyProtectedActivityCache(t))

	var contextResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !contextResult.ProtectedData.HasReadableSnapshot {
		t.Fatalf("expected readable snapshot after unlock, got %#v", contextResult)
	}

	var request = runtimeflow.MustIntegrationReportRequest(t, 2024, reportmodel.ReportBaseCurrencyUSD)
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
) runtimeflow.RuntimeBackedFlowHarness {
	t.Helper()

	var harness = runtimeflow.NewRuntimeBackedFlowHarness(t, baseDir, config, allowDevHTTP)
	harness.Model = flow.NewModel(flow.Dependencies{
		Options:       harness.App.Options,
		Startup:       bootstrap.StartupState{ActiveConfig: &config},
		SetupService:  harness.App.SetupService,
		SyncService:   harness.App.SyncService,
		ReportService: reportService,
	})
	return harness
}

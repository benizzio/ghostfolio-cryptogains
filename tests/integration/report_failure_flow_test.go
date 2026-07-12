// Package integration verifies black-box workflow behavior for the current
// slice, including runtime-backed report failure flows.
// Authored by: OpenCode
package integration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	currencyintegration "github.com/benizzio/ghostfolio-cryptogains/internal/integration/currency"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil/runtimeflow"
	"github.com/cockroachdb/apd/v3"
)

// TestReportGenerationEmptyMainSectionWritesEmptyMarkdownReport verifies that a
// valid empty-main-section report still saves a Markdown document with the
// required empty-state and selected report-currency contract.
// Authored by: OpenCode
func TestReportGenerationEmptyMainSectionWritesEmptyMarkdownReport(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)

	seedProtectedSnapshot(t, harness, "token-123", referenceOnlyProtectedActivityCache(t))

	var model = unlockSyncReportsContext(t, harness.Model, "token-123")
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, 2024)
	model = selectReportBaseCurrency(t, model, reportmodel.ReportBaseCurrencyUSD)
	model, cmd := startReportGenerationFromSelection(t, model)
	model = applyBatchCmd(t, model, cmd)

	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
	}

	var files = mustMarkdownFiles(t, reportIO.DocumentsDir)
	if len(files) != 1 {
		t.Fatalf("expected one saved Markdown file, got %#v", files)
	}
	var reportPath = files[0]
	testutil.AssertRegularFile(t, reportPath)

	// #nosec G304 -- reportPath is created in the test-owned Documents fixture.
	var reportBytes, err = os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read saved report %q: %v", reportPath, err)
	}
	var reportText = string(reportBytes)
	for _, expected := range []string{
		"- **Report Calculation Currency:** USD",
		"No assets had a non-zero net gain or loss in the selected year.",
		"| Overall Yearly Net Total | 0 | USD |",
		"## Reference Section",
		"reference only",
	} {
		if !strings.Contains(reportText, expected) {
			t.Fatalf("expected empty report to contain %q, got %q", expected, reportText)
		}
	}
	if strings.Contains(reportText, "## Asset Detail:") {
		t.Fatalf("expected empty-main-section report to omit detail sections, got %q", reportText)
	}

	var openerRequests = readOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != 1 || openerRequests[0] != reportPath {
		t.Fatalf("expected one opener request for %q, got %#v", reportPath, openerRequests)
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// TestReportGenerationWriteFailureGeneratesWrappedDiagnosticCauseChain verifies
// that output-preparation diagnostics preserve both the actionable outer
// failure and the wrapped inner write cause.
// Authored by: OpenCode
func TestReportGenerationWriteFailureGeneratesWrappedDiagnosticCauseChain(t *testing.T) {
	if os.Getenv("GHOSTFOLIO_CRYPTOGAINS_HELPER_WRITE_FAILURE") == "2" {
		runReportGenerationWriteFailureDiagnosticScenario(t)
		return
	}

	// #nosec G204,G702 -- os.Args[0] re-executes the current test binary with a fixed test name.
	var command = exec.CommandContext(context.Background(), os.Args[0], "-test.run=TestReportGenerationWriteFailureGeneratesWrappedDiagnosticCauseChain$")
	command.Env = append(
		os.Environ(),
		"GHOSTFOLIO_CRYPTOGAINS_HELPER_WRITE_FAILURE=2",
		"GHOSTFOLIO_CRYPTOGAINS_OUTPUT_FAIL_WRITE_AFTER_CREATE=forced write failure",
	)
	var output, err = command.CombinedOutput()
	if err != nil {
		t.Fatalf("run write-failure diagnostic helper process: %v\n%s", err, string(output))
	}
}

// TestReportGenerationIncompleteMonetaryContextShowsFailure verifies the
// runtime-backed unsupported-calculation outcome for incomplete priced activity
// currency data.
// Authored by: OpenCode
func TestReportGenerationIncompleteMonetaryContextShowsFailure(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var fixture = testutil.DeterministicReportLedgerFixture()
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)

	seedProtectedSnapshot(t, harness, "token-123", fixture.ProtectedActivityCache)

	var model = unlockSyncReportsContext(t, harness.Model, "token-123")
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, fixture.IncompleteContextReportYear)
	model = selectReportBaseCurrency(t, model, reportmodel.ReportBaseCurrencyUSD)
	model, cmd := startReportGenerationFromSelection(t, model)
	model = applyBatchCmd(t, model, cmd)

	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
	}

	var content = normalizeRenderedText(model.View().Content)
	for _, expected := range []string{
		"Failure Category: unsupported report calculation",
		"DOGE",
		"doge-buy-2025-incomplete-001",
		"incomplete",
		"No report file was saved.",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected failure result to contain %q, got %q", expected, content)
		}
	}

	if files := mustMarkdownFiles(t, reportIO.DocumentsDir); len(files) != 0 {
		t.Fatalf("expected no saved Markdown report after calculation failure, got %#v", files)
	}
	if openerRequests := readOpenCommandRequests(t, openLogPath); len(openerRequests) != 0 {
		t.Fatalf("expected no opener request after calculation failure, got %#v", openerRequests)
	}
	if files := mustDiagnosticFiles(t, harness.BaseDir); len(files) != 0 {
		t.Fatalf("expected production mode to defer report diagnostics until explicit choice, got %#v", files)
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)

	var updated tea.Model
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if !strings.Contains(normalizeRenderedText(model.View().Content), "Generating diagnostic report...") {
		t.Fatalf("expected report result diagnostics busy state, got %q", model.View().Content)
	}
	updated, _ = model.Update(testutil.RunCmd(cmd))
	model = assertFlowModel(t, updated)
	content = normalizeRenderedText(model.View().Content)
	if !strings.Contains(content, "Diagnostic Report Path:") {
		t.Fatalf("expected report diagnostics path disclosure, got %q", content)
	}
	var diagnosticFiles = mustDiagnosticFiles(t, harness.BaseDir)
	if len(diagnosticFiles) != 1 {
		t.Fatalf("expected one report diagnostics artifact after explicit choice, got %#v", diagnosticFiles)
	}
	assertReportFailureDiagnosticArtifact(t, diagnosticFiles[0], false)

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "sync_reports_menu" {
		t.Fatalf("expected failure dismissal to return to sync and reports menu, got %s", model.ActiveScreen())
	}
}

// TestReportGenerationIncompleteMonetaryContextAutoGeneratesDiagnosticsInExplicitDevelopmentMode
// verifies the explicit-development diagnostics path for activity-specific
// report failures.
// Authored by: OpenCode
func TestReportGenerationIncompleteMonetaryContextAutoGeneratesDiagnosticsInExplicitDevelopmentMode(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var fixture = testutil.DeterministicReportLedgerFixture()
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), true)

	seedProtectedSnapshot(t, harness, "token-123", fixture.ProtectedActivityCache)

	var model = unlockSyncReportsContext(t, harness.Model, "token-123")
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, fixture.IncompleteContextReportYear)
	model = selectReportBaseCurrency(t, model, reportmodel.ReportBaseCurrencyUSD)
	model, cmd := startReportGenerationFromSelection(t, model)
	model = applyBatchCmd(t, model, cmd)

	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
	}

	var content = normalizeRenderedText(model.View().Content)
	if strings.Contains(content, "Generate Diagnostic Report") {
		t.Fatalf("expected explicit development mode to skip the prompt, got %q", content)
	}
	if !strings.Contains(content, "Diagnostic Report Path:") {
		t.Fatalf("expected explicit-development diagnostics path disclosure, got %q", content)
	}
	if files := mustMarkdownFiles(t, reportIO.DocumentsDir); len(files) != 0 {
		t.Fatalf("expected no Markdown report after calculation failure, got %#v", files)
	}
	if openerRequests := readOpenCommandRequests(t, openLogPath); len(openerRequests) != 0 {
		t.Fatalf("expected no opener request after calculation failure, got %#v", openerRequests)
	}
	var diagnosticFiles = mustDiagnosticFiles(t, harness.BaseDir)
	if len(diagnosticFiles) != 1 {
		t.Fatalf("expected one auto-generated report diagnostics artifact, got %#v", diagnosticFiles)
	}
	assertReportFailureDiagnosticArtifact(t, diagnosticFiles[0], true)
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// TestReportGenerationIncompleteMonetaryContextFailsAfterAllExplicitCurrencyTiers
// verifies that a currencyless higher-priority tier is skipped and failure
// occurs only after the remaining explicit-currency tiers are exhausted.
// Authored by: OpenCode
func TestReportGenerationIncompleteMonetaryContextFailsAfterAllExplicitCurrencyTiers(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)

	var cache = syncmodel.ProtectedActivityCache{
		SyncedAt:             time.Date(2026, time.May, 20, 15, 4, 5, 0, time.UTC),
		RetrievedCount:       1,
		ActivityCount:        1,
		AvailableReportYears: []int{2026},
		Activities: []syncmodel.ActivityRecord{{
			SourceID:              "ada-buy-2026-currencyless-order-incomplete-001",
			OccurredAt:            "2026-02-01T12:00:00Z",
			ActivityType:          syncmodel.ActivityTypeBuy,
			AssetIdentityKey:      "asset-ada-002",
			AssetSymbol:           "ADA",
			AssetName:             "Cardano",
			Quantity:              mustReportFlowDecimal(t, "10"),
			OrderUnitPrice:        reportFlowDecimalPointer(t, "1"),
			OrderGrossValue:       reportFlowDecimalPointer(t, "10"),
			OrderFeeAmount:        reportFlowDecimalPointer(t, "0"),
			AssetProfileCurrency:  "EUR",
			AssetProfileUnitPrice: reportFlowDecimalPointer(t, "2"),
			BaseCurrency:          "USD",
			BaseGrossValue:        reportFlowDecimalPointer(t, "30"),
			DataSource:            "integration-report-fixture",
			RawHash:               "ada-buy-2026-currencyless-order-incomplete-001",
		}},
	}

	seedProtectedSnapshot(t, harness, "token-123", cache)

	var model = unlockSyncReportsContext(t, harness.Model, "token-123")
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, 2026)
	model = selectReportBaseCurrency(t, model, reportmodel.ReportBaseCurrencyUSD)
	model, cmd := startReportGenerationFromSelection(t, model)
	model = applyBatchCmd(t, model, cmd)

	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
	}

	var content = normalizeRenderedText(model.View().Content)
	for _, expected := range []string{
		"Failure Category: unsupported report calculation",
		"ADA",
		"incomplete",
		"No report file was saved.",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected failure result to contain %q, got %q", expected, content)
		}
	}
	if !strings.Contains(strings.ReplaceAll(content, " ", ""), "ada-buy-2026-currencyless-order-incomplete-001") {
		t.Fatalf("expected failure result to reference the offending source ID, got %q", content)
	}

	if files := mustMarkdownFiles(t, reportIO.DocumentsDir); len(files) != 0 {
		t.Fatalf("expected no saved Markdown report after exhausted explicit-currency failure, got %#v", files)
	}
	if openerRequests := readOpenCommandRequests(t, openLogPath); len(openerRequests) != 0 {
		t.Fatalf("expected no opener request after exhausted explicit-currency failure, got %#v", openerRequests)
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// TestReportGenerationConversionFailureMatrixShowsSafeFailure verifies US3
// conversion failures stop before final save and disclose only non-secret lookup
// context in the report-result screen.
// Authored by: OpenCode
func TestReportGenerationConversionFailureMatrixShowsSafeFailure(t *testing.T) {
	var cases = []struct {
		name                string
		reportBaseCurrency  reportmodel.ReportBaseCurrency
		cache               syncmodel.ProtectedActivityCache
		failures            map[string]error
		expectedSnippets    []string
		unexpectedSnippets  []string
		expectedLookupCount int
	}{
		{
			name:               "unsupported source currency",
			reportBaseCurrency: reportmodel.ReportBaseCurrencyUSD,
			cache:              conversionFailureProtectedActivityCache(t, "unsupported-rub-buy", "RUB", "2024-02-05T10:00:00Z"),
			failures: map[string]error{
				"RUB|USD|2024-02-05": mustIntegrationConversionFailure(t, "RUB|USD|2024-02-05", currencyintegration.ProviderIDFederalReserveH10, currencyintegration.ConversionFailureReasonUnsupportedCurrency, "source currency RUB is not supported by federal_reserve_h10"),
			},
			expectedSnippets:    []string{"Failure Category: unsupported report calculation", "unsupported-rub-buy", "RUB", "USD", "2024-02-05", "unsupported_currency", "federal_reserve_h10", "No report file was saved."},
			expectedLookupCount: 1,
		},
		{
			name:               "missing authoritative rate",
			reportBaseCurrency: reportmodel.ReportBaseCurrencyEUR,
			cache:              conversionFailureProtectedActivityCache(t, "missing-usd-rate-buy", "USD", "2024-03-09T10:00:00Z"),
			failures: map[string]error{
				"USD|EUR|2024-03-09": mustIntegrationConversionFailure(t, "USD|EUR|2024-03-09", currencyintegration.ProviderIDECBEXR, currencyintegration.ConversionFailureReasonMissingRate, "no current or prior ECB EXR observation for USD/EUR on 2024-03-09"),
			},
			expectedSnippets:    []string{"Failure Category: unsupported report calculation", "missing-usd-rate-buy", "USD", "EUR", "2024-03-09", "missing_rate", "ecb_exr", "No report file was saved."},
			expectedLookupCount: 1,
		},
		{
			name:               "provider unavailable without cache",
			reportBaseCurrency: reportmodel.ReportBaseCurrencyUSD,
			cache:              conversionFailureProtectedActivityCache(t, "provider-down-eur-buy", "EUR", "2024-04-10T10:00:00Z"),
			failures: map[string]error{
				"EUR|USD|2024-04-10": mustIntegrationConversionFailure(t, "EUR|USD|2024-04-10", currencyintegration.ProviderIDFederalReserveH10, currencyintegration.ConversionFailureReasonProviderUnavailable, "federal_reserve_h10 request failed without cached evidence"),
			},
			expectedSnippets:    []string{"Failure Category: unsupported report calculation", "provider-down-eur-buy", "EUR", "USD", "2024-04-10", "provider_unavailable", "federal_reserve_h10", "No report file was saved."},
			expectedLookupCount: 1,
		},
		{
			name:                "malformed selected activity currency",
			reportBaseCurrency:  reportmodel.ReportBaseCurrencyUSD,
			cache:               conversionFailureProtectedActivityCache(t, "malformed-currency-buy", "EU", "2024-05-11T10:00:00Z"),
			expectedSnippets:    []string{"Failure Category: unsupported report calculation", "malformed-currency-buy", "EU", "USD", "2024-05-11", "invalid_activity_currency", "No report file was saved."},
			expectedLookupCount: 0,
		},
		{
			name:               "late failure after earlier conversion",
			reportBaseCurrency: reportmodel.ReportBaseCurrencyUSD,
			cache:              lateConversionFailureProtectedActivityCache(t),
			failures: map[string]error{
				"GBP|USD|2024-06-13": mustIntegrationConversionFailure(t, "GBP|USD|2024-06-13", currencyintegration.ProviderIDFederalReserveH10, currencyintegration.ConversionFailureReasonMalformedRate, "Federal Reserve H.10 observation for GBP on 2024-06-13 is not exact-decimal parseable"),
			},
			expectedSnippets:    []string{"Failure Category: unsupported report calculation", "late-gbp-buy", "GBP", "USD", "2024-06-13", "malformed_rate", "federal_reserve_h10", "No report file was saved."},
			expectedLookupCount: 2,
		},
	}

	for _, testCase := range cases {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			var reportIO = testutil.NewReportIOFixture(t)
			var openLogPath = installOpenCommandRecorder(t, 0)
			var rateService = &failingIntegrationCurrencyRates{failures: testCase.failures}
			var harness = runtimeflow.NewRuntimeBackedFlowHarnessWithCurrencyRateService(t, t.TempDir(), runtimeflow.MustCloudSetupConfig(t), false, rateService)

			seedProtectedSnapshot(t, harness, "token-123", testCase.cache)

			var model = unlockSyncReportsContext(t, harness.Model, "token-123")
			model = openReportSelectionFromContext(t, model)
			model = selectReportYear(t, model, 2024)
			model = selectReportBaseCurrency(t, model, testCase.reportBaseCurrency)
			model, cmd := runtimeflow.StartReportGeneration(t, model)
			model = applyBatchCmd(t, model, cmd)

			if model.ActiveScreen() != "report_result" {
				t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
			}
			var content = normalizeRenderedText(model.View().Content)
			for _, expected := range testCase.expectedSnippets {
				if !strings.Contains(content, expected) {
					t.Fatalf("expected conversion failure result to contain %q, got %q", expected, content)
				}
			}
			for _, unexpected := range append(testCase.unexpectedSnippets, "token-123", "Bearer") {
				if strings.Contains(content, unexpected) {
					t.Fatalf("expected conversion failure result to exclude %q, got %q", unexpected, content)
				}
			}
			if len(rateService.requests) != testCase.expectedLookupCount {
				t.Fatalf("expected %d lookup requests, got %#v", testCase.expectedLookupCount, rateService.requests)
			}
			if files := mustMarkdownFiles(t, reportIO.DocumentsDir); len(files) != 0 {
				t.Fatalf("expected no saved Markdown report after conversion failure, got %#v", files)
			}
			if openerRequests := readOpenCommandRequests(t, openLogPath); len(openerRequests) != 0 {
				t.Fatalf("expected no opener request after conversion failure, got %#v", openerRequests)
			}
			assertNoCleartextReportInAppStorage(t, harness.BaseDir)
		})
	}
}

// TestReportGenerationDocumentsUnavailableShowsFailure verifies the save
// failure path when the resolved Documents directory is missing.
// Authored by: OpenCode
func TestReportGenerationDocumentsUnavailableShowsFailure(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	if err := os.RemoveAll(reportIO.DocumentsDir); err != nil {
		t.Fatalf("remove Documents directory: %v", err)
	}
	var openLogPath = installOpenCommandRecorder(t, 0)
	var fixture = testutil.DeterministicReportLedgerFixture()
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)

	seedProtectedSnapshot(t, harness, "token-123", fixture.ProtectedActivityCache)

	var model = unlockSyncReportsContext(t, harness.Model, "token-123")
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, fixture.PrimaryReportYear)
	model = selectReportBaseCurrency(t, model, reportmodel.ReportBaseCurrencyUSD)
	model, cmd := startReportGenerationFromSelection(t, model)
	model = applyBatchCmd(t, model, cmd)

	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
	}

	var content = normalizeRenderedText(model.View().Content)
	if !strings.Contains(content, "Failure Category: documents folder unavailable") {
		t.Fatalf("expected documents-folder failure category, got %q", content)
	}
	if !strings.Contains(content, "Documents folder is unavailable") || !strings.Contains(content, "No report file was saved.") {
		t.Fatalf("expected actionable documents-folder failure message, got %q", content)
	}

	if files := readOpenCommandRequests(t, openLogPath); len(files) != 0 {
		t.Fatalf("expected no opener request when save failed, got %#v", files)
	}
	if _, err := os.Stat(reportIO.DocumentsDir); !os.IsNotExist(err) {
		t.Fatalf("expected Documents directory to remain absent after failed save, got %v", err)
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// TestReportGenerationWriteFailureRemovesPartialFileAndShowsFailure verifies
// that a save failure after file creation removes the partial report artifact
// and reports the write failure clearly.
// Authored by: OpenCode
func TestReportGenerationWriteFailureRemovesPartialFileAndShowsFailure(t *testing.T) {
	if os.Getenv("GHOSTFOLIO_CRYPTOGAINS_HELPER_WRITE_FAILURE") == "1" {
		runReportGenerationWriteFailureScenario(t)
		return
	}

	// #nosec G204,G702 -- os.Args[0] re-executes the current test binary with a fixed test name.
	var command = exec.CommandContext(context.Background(), os.Args[0], "-test.run=TestReportGenerationWriteFailureRemovesPartialFileAndShowsFailure$")
	command.Env = append(
		os.Environ(),
		"GHOSTFOLIO_CRYPTOGAINS_HELPER_WRITE_FAILURE=1",
		"GHOSTFOLIO_CRYPTOGAINS_OUTPUT_FAIL_WRITE_AFTER_CREATE=forced write failure",
	)
	var output, err = command.CombinedOutput()
	if err != nil {
		t.Fatalf("run write-failure helper process: %v\n%s", err, string(output))
	}
}

// TestReportGenerationPDFOutputUsesTheConcreteRenderer verifies that PDF output
// uses the production renderer and writes only its combined PDF bundle.
// Authored by: OpenCode
func TestReportGenerationPDFOutputUsesTheConcreteRenderer(t *testing.T) {
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

	var request = mustIntegrationReportRequestForFormat(t, fixture.PrimaryReportYear, reportmodel.ReportBaseCurrencyUSD, reportmodel.ReportOutputFormatPDF)
	var outcome = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{Request: request})
	if !outcome.Success {
		t.Fatalf("expected concrete PDF report generation success, got %#v", outcome)
	}
	if outcome.FailureReason != runtime.ReportFailureNone {
		t.Fatalf("expected no PDF generation failure, got %#v", outcome)
	}
	if files := mustPDFFiles(t, reportIO.DocumentsDir); len(files) != 1 {
		t.Fatalf("expected one combined PDF file, got %#v", files)
	}
	if files := runtimeflow.AllMarkdownFiles(t, reportIO.DocumentsDir); len(files) != 0 {
		t.Fatalf("expected no Markdown files for PDF output, got %#v", files)
	}
	if openerRequests := readOpenCommandRequests(t, openLogPath); len(openerRequests) != 1 {
		t.Fatalf("expected one opener request after PDF generation, got %#v", openerRequests)
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// TestReportGenerationBundleWriteFailureLeavesNoPartialBundleFiles verifies that
// a write failure after output reservation removes every bundle file created by
// the failed attempt.
// Authored by: OpenCode
func TestReportGenerationBundleWriteFailureLeavesNoPartialBundleFiles(t *testing.T) {
	if os.Getenv("GHOSTFOLIO_CRYPTOGAINS_HELPER_BUNDLE_WRITE_FAILURE") == "1" {
		runReportGenerationBundleWriteFailureScenario(t)
		return
	}

	//nolint:gosec // This test intentionally re-executes the current test binary as a helper process.
	var command = exec.CommandContext(context.Background(), os.Args[0], "-test.run=TestReportGenerationBundleWriteFailureLeavesNoPartialBundleFiles$")
	command.Env = append(
		os.Environ(),
		"GHOSTFOLIO_CRYPTOGAINS_HELPER_BUNDLE_WRITE_FAILURE=1",
		"GHOSTFOLIO_CRYPTOGAINS_OUTPUT_FAIL_WRITE_AFTER_CREATE=forced bundle write failure",
	)
	var output, err = command.CombinedOutput()
	if err != nil {
		t.Fatalf("run bundle write-failure helper process: %v\n%s", err, string(output))
	}
}

// TestReportGenerationRendererFailureLeavesNoOutputFiles verifies a runtime
// renderer failure occurs before output writing or automatic opening.
// Authored by: OpenCode
func TestReportGenerationRendererFailureLeavesNoOutputFiles(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var fixture = testutil.DeterministicReportLedgerFixture()
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)
	var token = "token-123"

	if err := harness.App.SetReportBundleRendererForTesting(func(reportmodel.ReportOutputFormat, reportmodel.CapitalGainsReport) ([]reportmodel.ReportDocument, error) {
		return nil, errors.New("forced renderer failure")
	}); err != nil {
		t.Fatalf("inject renderer failure: %v", err)
	}
	seedProtectedSnapshot(t, harness, token, fixture.ProtectedActivityCache)

	var contextResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !contextResult.ProtectedData.HasReadableSnapshot {
		t.Fatalf("expected readable snapshot after unlock, got %#v", contextResult)
	}

	var request = mustIntegrationReportRequestForFormat(t, fixture.PrimaryReportYear, reportmodel.ReportBaseCurrencyUSD, reportmodel.ReportOutputFormatMarkdown)
	var outcome = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{Request: request})
	if outcome.Success {
		t.Fatalf("expected renderer failure outcome, got %#v", outcome)
	}
	if outcome.FailureReason != runtime.ReportFailureUnsupportedReportCalculation {
		t.Fatalf("expected renderer failure category %q, got %#v", runtime.ReportFailureUnsupportedReportCalculation, outcome)
	}
	if !strings.Contains(outcome.Message, "Could not render the") || !strings.Contains(outcome.Message, "forced renderer failure") || !strings.Contains(outcome.Message, "No report file was saved.") {
		t.Fatalf("expected renderer failure result, got %q", outcome.Message)
	}
	assertNoReportBundleFiles(t, reportIO.DocumentsDir)
	if openerRequests := readOpenCommandRequests(t, openLogPath); len(openerRequests) != 0 {
		t.Fatalf("expected no opener request after renderer failure, got %#v", openerRequests)
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// runReportGenerationWriteFailureScenario executes the runtime-backed write
// failure assertions in a helper subprocess so report-output test seams stay
// isolated from other parallel integration tests.
// Authored by: OpenCode
func runReportGenerationWriteFailureScenario(t *testing.T) {
	t.Helper()

	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var fixture = testutil.DeterministicReportLedgerFixture()
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)

	seedProtectedSnapshot(t, harness, "token-123", fixture.ProtectedActivityCache)

	var model = unlockSyncReportsContext(t, harness.Model, "token-123")
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, fixture.PrimaryReportYear)
	model = selectReportBaseCurrency(t, model, reportmodel.ReportBaseCurrencyUSD)
	model, cmd := startReportGenerationFromSelection(t, model)
	model = applyBatchCmd(t, model, cmd)

	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
	}

	var content = normalizeRenderedText(model.View().Content)
	for _, expected := range []string{
		"Failure Category: report file write failed",
		"Could not save the report file:",
		"forced write failure",
		"Any partial file created during this attempt was removed.",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected write-failure result to contain %q, got %q", expected, content)
		}
	}

	if files := mustMarkdownFiles(t, reportIO.DocumentsDir); len(files) != 0 {
		t.Fatalf("expected no saved Markdown report after write failure, got %#v", files)
	}
	var documentsEntries, err = os.ReadDir(reportIO.DocumentsDir)
	if err != nil {
		t.Fatalf("read Documents directory: %v", err)
	}
	if len(documentsEntries) != 0 {
		t.Fatalf("expected partial-file cleanup to leave Documents empty, got %#v", documentsEntries)
	}
	if openerRequests := readOpenCommandRequests(t, openLogPath); len(openerRequests) != 0 {
		t.Fatalf("expected no opener request when save failed after create, got %#v", openerRequests)
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)

	var updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "sync_reports_menu" {
		t.Fatalf("expected write-failure dismissal to return to sync and reports menu, got %s", model.ActiveScreen())
	}
}

// runReportGenerationBundleWriteFailureScenario executes bundle-level write
// failure cleanup assertions in a helper subprocess so output test seams stay
// isolated from other integration tests.
// Authored by: OpenCode
func runReportGenerationBundleWriteFailureScenario(t *testing.T) {
	t.Helper()

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

	var request = mustIntegrationReportRequestForFormat(t, fixture.PrimaryReportYear, reportmodel.ReportBaseCurrencyUSD, reportmodel.ReportOutputFormatMarkdown)
	var outcome = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{Request: request})
	if outcome.Success {
		t.Fatalf("expected bundle write failure, got %#v", outcome)
	}
	if outcome.FailureReason != runtime.ReportFailureReportFileWriteFailed {
		t.Fatalf("expected bundle write failure category %q, got %#v", runtime.ReportFailureReportFileWriteFailed, outcome)
	}
	if !strings.Contains(outcome.Message, "forced bundle write failure") {
		t.Fatalf("expected bundle write failure detail, got %q", outcome.Message)
	}
	assertNoReportBundleFiles(t, reportIO.DocumentsDir)
	if openerRequests := readOpenCommandRequests(t, openLogPath); len(openerRequests) != 0 {
		t.Fatalf("expected no opener request when bundle save failed, got %#v", openerRequests)
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// runReportGenerationWriteFailureDiagnosticScenario exercises the report-result
// diagnostic path for a wrapped output-preparation failure.
// Authored by: OpenCode
func runReportGenerationWriteFailureDiagnosticScenario(t *testing.T) {
	t.Helper()

	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var fixture = testutil.DeterministicReportLedgerFixture()
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)

	seedProtectedSnapshot(t, harness, "token-123", fixture.ProtectedActivityCache)

	var model = unlockSyncReportsContext(t, harness.Model, "token-123")
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, fixture.PrimaryReportYear)
	model = selectReportBaseCurrency(t, model, reportmodel.ReportBaseCurrencyUSD)
	model, cmd := startReportGenerationFromSelection(t, model)
	model = applyBatchCmd(t, model, cmd)

	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
	}
	if files := mustMarkdownFiles(t, reportIO.DocumentsDir); len(files) != 0 {
		t.Fatalf("expected no saved Markdown report after write failure, got %#v", files)
	}
	if openerRequests := readOpenCommandRequests(t, openLogPath); len(openerRequests) != 0 {
		t.Fatalf("expected no opener request when save failed after create, got %#v", openerRequests)
	}

	var updated tea.Model
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(testutil.RunCmd(cmd))
	_ = assertFlowModel(t, updated)

	var diagnosticFiles = mustDiagnosticFiles(t, harness.BaseDir)
	if len(diagnosticFiles) != 1 {
		t.Fatalf("expected one report diagnostics artifact after explicit choice, got %#v", diagnosticFiles)
	}
	assertReportFailureDiagnosticArtifact(t, diagnosticFiles[0], false)

	var raw, err = os.ReadFile(diagnosticFiles[0])
	if err != nil {
		t.Fatalf("read report diagnostics artifact: %v", err)
	}
	var text = string(raw)
	for _, expected := range []string{
		`"failure_detail": "could not save the report file"`,
		`"failure_cause_chain": [`,
		`"could not save the report file"`,
		`"write report file`,
		`forced write failure`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected wrapped output-preparation diagnostics to contain %q, got %q", expected, text)
		}
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// referenceOnlyProtectedActivityCache returns one deterministic protected cache
// that produces a valid empty-main-section report with one reference-only row.
// Authored by: OpenCode
func referenceOnlyProtectedActivityCache(t *testing.T) syncmodel.ProtectedActivityCache {
	t.Helper()

	var activities = []syncmodel.ActivityRecord{
		{
			SourceID:         "eth-buy-2023-001",
			OccurredAt:       "2023-01-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-eth-001",
			AssetSymbol:      "ETH",
			AssetName:        "Ethereum",
			Quantity:         mustReportFlowDecimal(t, "1"),
			OrderCurrency:    "USD",
			OrderUnitPrice:   reportFlowDecimalPointer(t, "10"),
			OrderGrossValue:  reportFlowDecimalPointer(t, "10"),
			OrderFeeAmount:   reportFlowDecimalPointer(t, "0"),
			DataSource:       "integration-report-fixture",
			RawHash:          "eth-buy-2023-001",
		},
		{
			SourceID:         "eth-sell-2023-001",
			OccurredAt:       "2023-06-01T09:00:00Z",
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: "asset-eth-001",
			AssetSymbol:      "ETH",
			AssetName:        "Ethereum",
			Quantity:         mustReportFlowDecimal(t, "1"),
			OrderCurrency:    "USD",
			OrderUnitPrice:   reportFlowDecimalPointer(t, "12"),
			OrderGrossValue:  reportFlowDecimalPointer(t, "12"),
			OrderFeeAmount:   reportFlowDecimalPointer(t, "0"),
			DataSource:       "integration-report-fixture",
			RawHash:          "eth-sell-2023-001",
		},
	}

	return syncmodel.ProtectedActivityCache{
		SyncedAt:             time.Date(2026, time.May, 20, 15, 4, 5, 0, time.UTC),
		RetrievedCount:       len(activities),
		ActivityCount:        len(activities),
		AvailableReportYears: []int{2024},
		Activities:           activities,
	}
}

// failingIntegrationCurrencyRates records lookup requests and returns configured
// conversion failures for fail-first US3 integration fixtures.
// Authored by: OpenCode
type failingIntegrationCurrencyRates struct {
	failures map[string]error
	requests []currencyintegration.RateLookupRequest
}

// LookupRate returns configured failure evidence or deterministic successful
// evidence for conversion attempts before a later matrix failure.
// Authored by: OpenCode
func (service *failingIntegrationCurrencyRates) LookupRate(_ context.Context, request currencyintegration.RateLookupRequest) (currencyintegration.ExchangeRateEvidence, error) {
	service.requests = append(service.requests, request)
	if service.failures != nil {
		var failure, ok = service.failures[integrationFailureRateKey(request)]
		if ok {
			return currencyintegration.ExchangeRateEvidence{}, failure
		}
	}

	return runtimeflow.DeterministicCurrencyRates{}.LookupRate(context.Background(), request)
}

// ProviderCategoryForBaseCurrency returns deterministic provider metadata for
// failing conversion integration tests.
// Authored by: OpenCode
func (service *failingIntegrationCurrencyRates) ProviderCategoryForBaseCurrency(baseCurrency string) string {
	return runtimeflow.DeterministicCurrencyRates{}.ProviderCategoryForBaseCurrency(baseCurrency)
}

// conversionFailureProtectedActivityCache returns one priced cross-currency row
// that must request a conversion rate for the selected report base currency.
// Authored by: OpenCode
func conversionFailureProtectedActivityCache(t *testing.T, sourceID string, sourceCurrency string, occurredAt string) syncmodel.ProtectedActivityCache {
	t.Helper()

	return syncmodel.ProtectedActivityCache{
		SyncedAt:             time.Date(2026, time.May, 20, 15, 4, 5, 0, time.UTC),
		RetrievedCount:       1,
		ActivityCount:        1,
		AvailableReportYears: []int{2024},
		Activities: []syncmodel.ActivityRecord{conversionFailureActivityRecord(
			t,
			sourceID,
			sourceCurrency,
			occurredAt,
			"1",
			"1000.25",
		)},
	}
}

// lateConversionFailureProtectedActivityCache returns a deterministic history
// where one conversion succeeds before a later conversion fails.
// Authored by: OpenCode
func lateConversionFailureProtectedActivityCache(t *testing.T) syncmodel.ProtectedActivityCache {
	t.Helper()

	return syncmodel.ProtectedActivityCache{
		SyncedAt:             time.Date(2026, time.May, 20, 15, 4, 5, 0, time.UTC),
		RetrievedCount:       2,
		ActivityCount:        2,
		AvailableReportYears: []int{2024},
		Activities: []syncmodel.ActivityRecord{
			conversionFailureActivityRecord(t, "late-eur-buy", "EUR", "2024-06-12T10:00:00Z", "1", "100"),
			conversionFailureActivityRecord(t, "late-gbp-buy", "GBP", "2024-06-13T10:00:00Z", "1", "200"),
		},
	}
}

// conversionFailureActivityRecord builds one priced buy fixture for conversion
// failure tests.
// Authored by: OpenCode
func conversionFailureActivityRecord(t *testing.T, sourceID string, sourceCurrency string, occurredAt string, quantity string, grossValue string) syncmodel.ActivityRecord {
	t.Helper()

	return syncmodel.ActivityRecord{
		SourceID:         sourceID,
		OccurredAt:       occurredAt,
		ActivityType:     syncmodel.ActivityTypeBuy,
		AssetIdentityKey: "asset-conversion-failure-001",
		AssetSymbol:      "FAIL",
		AssetName:        "Failure Fixture",
		Quantity:         mustReportFlowDecimal(t, quantity),
		OrderCurrency:    sourceCurrency,
		OrderUnitPrice:   reportFlowDecimalPointer(t, grossValue),
		OrderGrossValue:  reportFlowDecimalPointer(t, grossValue),
		OrderFeeAmount:   reportFlowDecimalPointer(t, "0"),
		DataSource:       "integration-report-failure-fixture",
		RawHash:          sourceID,
		Comment:          "non-secret conversion failure fixture",
	}
}

// integrationFailureRateKey returns the deterministic lookup key used by the
// failure matrix rate-service stub.
// Authored by: OpenCode
func integrationFailureRateKey(request currencyintegration.RateLookupRequest) string {
	return request.SourceCurrency + "|" + request.BaseCurrency + "|" + request.ActivityDate.Format(time.DateOnly)
}

// mustIntegrationConversionFailure creates one structured matrix failure from a
// deterministic source|base|date lookup key.
// Authored by: OpenCode
func mustIntegrationConversionFailure(t *testing.T, key string, providerID currencyintegration.ProviderID, reason currencyintegration.ConversionFailureReason, detail string) error {
	t.Helper()

	var parts = strings.Split(key, "|")
	if len(parts) != 3 {
		t.Fatalf("expected conversion failure key source|base|date, got %q", key)
	}
	var activityDate, err = time.Parse(time.DateOnly, parts[2])
	if err != nil {
		t.Fatalf("parse conversion failure date %q: %v", parts[2], err)
	}
	var request currencyintegration.RateLookupRequest
	request, err = currencyintegration.NewRateLookupRequest(parts[0], parts[1], activityDate)
	if err != nil {
		t.Fatalf("create conversion failure lookup request from %q: %v", key, err)
	}

	return currencyintegration.NewConversionFailure(request, providerID, reason, detail)
}

// mustReportFlowDecimal parses one integration fixture decimal value.
// Authored by: OpenCode
func mustReportFlowDecimal(t *testing.T, raw string) apd.Decimal {
	t.Helper()

	var value, _, err = decimalsupport.ParseString(raw)
	if err != nil {
		t.Fatalf("parse decimal %q: %v", raw, err)
	}

	return value
}

// reportFlowDecimalPointer returns one parsed decimal pointer for integration
// report fixtures.
// Authored by: OpenCode
func reportFlowDecimalPointer(t *testing.T, raw string) *apd.Decimal {
	t.Helper()

	var value = mustReportFlowDecimal(t, raw)
	return &value
}

// assertNoReportBundleFiles verifies that a failed generation attempt left no
// cleartext report bundle artifacts in the Documents directory.
// Authored by: OpenCode
func assertNoReportBundleFiles(t *testing.T, documentsDir string) {
	t.Helper()

	if markdownFiles := runtimeflow.AllMarkdownFiles(t, documentsDir); len(markdownFiles) != 0 {
		t.Fatalf("expected no Markdown bundle files after failed generation, got %#v", markdownFiles)
	}
	if pdfFiles := mustPDFFiles(t, documentsDir); len(pdfFiles) != 0 {
		t.Fatalf("expected no PDF bundle files after failed generation, got %#v", pdfFiles)
	}
	var documentsEntries, err = os.ReadDir(documentsDir)
	if err != nil {
		t.Fatalf("read Documents directory: %v", err)
	}
	if len(documentsEntries) != 0 {
		t.Fatalf("expected failed generation cleanup to leave Documents empty, got %#v", documentsEntries)
	}
}

// assertReportFailureDiagnosticArtifact verifies one report-failure diagnostic
// artifact for BUG-006 source-faithful persisted-record behavior.
// Authored by: OpenCode
func assertReportFailureDiagnosticArtifact(t *testing.T, path string, expectFinancialValues bool) {
	t.Helper()

	testutil.AssertRegularFile(t, path)
	if filepath.Ext(path) != ".json" {
		t.Fatalf("expected diagnostic artifact path, got %q", path)
	}
	// #nosec G304 -- path is selected from the test-owned diagnostics fixture.
	var raw, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report diagnostics artifact: %v", err)
	}
	var text = string(raw)
	if strings.Contains(text, "token-123") || strings.Contains(text, "jwt") || strings.Contains(text, "Ghostfolio Capital Gains And Losses Report") {
		t.Fatalf("expected report diagnostics artifact to remain secret-safe and report-free, got %q", text)
	}
	for _, expected := range []string{
		`"failure_detail":`,
		`"failure_cause_chain": [`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected diagnostics artifact to contain %q, got %q", expected, text)
		}
	}
	if !strings.Contains(text, `"failure_category": "unsupported report calculation"`) && !strings.Contains(text, `"failure_category": "report file write failed"`) {
		t.Fatalf("expected diagnostics artifact to contain a supported report failure category, got %q", text)
	}
	if strings.Contains(text, `"failure_category": "unsupported report calculation"`) {
		for _, expected := range []string{
			`"offending_activity_record"`,
			`"source_id": "doge-buy-2025-incomplete-001"`,
			`"asset_identity_key": "asset-doge-001"`,
			`"order_currency": "USD"`,
			`"asset_profile_currency": null`,
			`"base_currency": null`,
			`"source_scope": {`,
			`"id": "wallet-speculative"`,
		} {
			if !strings.Contains(text, expected) {
				t.Fatalf("expected calculation diagnostics artifact to contain %q, got %q", expected, text)
			}
		}
	}
	if strings.Contains(text, `"selected_currency_context"`) || strings.Contains(text, `"activity_currency"`) {
		t.Fatalf("expected diagnostics artifact to omit derived report fields, got %q", text)
	}

	var payload struct {
		FailureCategory         string   `json:"failure_category"`
		FailureDetail           string   `json:"failure_detail"`
		FailureCauseChain       []string `json:"failure_cause_chain"`
		OffendingActivityRecord *struct {
			SourceID             *string `json:"source_id"`
			AssetIdentityKey     *string `json:"asset_identity_key"`
			Quantity             *string `json:"quantity"`
			OrderCurrency        *string `json:"order_currency"`
			AssetProfileCurrency *string `json:"asset_profile_currency"`
			BaseCurrency         *string `json:"base_currency"`
			SourceScope          *struct {
				ID *string `json:"id"`
			} `json:"source_scope"`
		} `json:"offending_activity_record"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal report diagnostics artifact: %v", err)
	}
	if payload.FailureDetail == "" || len(payload.FailureCauseChain) == 0 {
		t.Fatalf("expected diagnostics artifact to preserve failure detail and cause chain, got %#v", payload)
	}
	if payload.FailureCauseChain[0] != payload.FailureDetail {
		t.Fatalf("expected cause chain to start with the actionable outer failure, got %#v", payload)
	}
	if payload.FailureCategory == "unsupported report calculation" {
		if len(payload.FailureCauseChain) < 2 || !strings.Contains(payload.FailureCauseChain[1], `activity "doge-buy-2025-incomplete-001" order currency context is incomplete; provide`) || !strings.Contains(payload.FailureCauseChain[1], `derive gross value and fee from that tier only`) {
			t.Fatalf("expected wrapped calculation diagnostics cause chain, got %#v", payload.FailureCauseChain)
		}
		if payload.OffendingActivityRecord == nil || payload.OffendingActivityRecord.SourceID == nil || *payload.OffendingActivityRecord.SourceID != "doge-buy-2025-incomplete-001" {
			t.Fatalf("expected calculation diagnostics artifact to preserve offending activity context, got %#v", payload)
		}
	}
	if payload.FailureCategory == "report file write failed" {
		if len(payload.FailureCauseChain) < 3 || payload.FailureCauseChain[0] != "could not save the report file" || !strings.Contains(payload.FailureCauseChain[1], "write report file") || payload.FailureCauseChain[2] != "forced write failure" {
			t.Fatalf("expected wrapped write diagnostics cause chain, got %#v", payload.FailureCauseChain)
		}
		if payload.OffendingActivityRecord != nil {
			t.Fatalf("expected write-failure diagnostics to omit activity-specific context, got %#v", payload.OffendingActivityRecord)
		}
	}
	if expectFinancialValues {
		if payload.OffendingActivityRecord == nil || payload.OffendingActivityRecord.Quantity == nil || *payload.OffendingActivityRecord.Quantity != "10000" {
			t.Fatalf("expected explicit-development diagnostics to retain quantity, got %q", text)
		}
		return
	}
	if payload.OffendingActivityRecord != nil && payload.OffendingActivityRecord.Quantity != nil {
		t.Fatalf("expected production diagnostics to redact quantity, got %q", text)
	}
}

// Package integration verifies black-box workflow behavior for the current
// slice, including runtime-backed report failure flows.
// Authored by: OpenCode
package integration

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	reportoutput "github.com/benizzio/ghostfolio-cryptogains/internal/report/output"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/cockroachdb/apd/v3"
)

// TestReportGenerationEmptyMainSectionWritesEmptyMarkdownReport verifies that a
// valid empty-main-section report still saves a Markdown document with the
// required empty-state and NOT APPLICABLE currency contract.
// Authored by: OpenCode
func TestReportGenerationEmptyMainSectionWritesEmptyMarkdownReport(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)

	seedProtectedSnapshot(t, harness, "token-123", referenceOnlyProtectedActivityCache(t))

	var model = unlockSyncReportsContext(t, harness.Model, "token-123")
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, 2024)
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

	var reportBytes, err = os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read saved report %q: %v", reportPath, err)
	}
	var reportText = string(reportBytes)
	for _, expected := range []string{
		"- Report Calculation Currency: NOT APPLICABLE",
		"No assets qualified for the main report sections in the selected year.",
		"| Overall Yearly Net Total | 0 | NOT APPLICABLE |",
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
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)

	var updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "sync_reports_menu" {
		t.Fatalf("expected failure dismissal to return to sync and reports menu, got %s", model.ActiveScreen())
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
		reportoutput.InstallWriteFailureAfterCreateForTesting(errors.New("forced write failure"))
		runReportGenerationWriteFailureScenario(t)
		return
	}

	var command = exec.Command(os.Args[0], "-test.run=TestReportGenerationWriteFailureRemovesPartialFileAndShowsFailure$")
	command.Env = append(os.Environ(), "GHOSTFOLIO_CRYPTOGAINS_HELPER_WRITE_FAILURE=1")
	var output, err = command.CombinedOutput()
	if err != nil {
		t.Fatalf("run write-failure helper process: %v\n%s", err, string(output))
	}
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

	var updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "sync_reports_menu" {
		t.Fatalf("expected write-failure dismissal to return to sync and reports menu, got %s", model.ActiveScreen())
	}
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

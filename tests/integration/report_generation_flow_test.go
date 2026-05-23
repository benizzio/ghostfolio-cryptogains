// Package integration verifies black-box workflow behavior for the current
// slice, including runtime-backed report generation flows.
// Authored by: OpenCode
package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
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
	model, cmd := startReportGenerationFromSelection(t, model)
	model = applyBatchCmd(t, model, cmd)

	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
	}

	var content = normalizeRenderedText(model.View().Content)
	if !strings.Contains(content, "Saved Markdown Path: ") {
		t.Fatalf("expected saved Markdown path in result view, got %q", content)
	}
	if !strings.Contains(content, "Selected Year: 2024") || !strings.Contains(content, "Cost Basis Method: FIFO") {
		t.Fatalf("expected selected year and method in result view, got %q", content)
	}

	var files = mustMarkdownFiles(t, reportIO.DocumentsDir)
	if len(files) != 1 {
		t.Fatalf("expected one saved Markdown file, got %#v", files)
	}

	var reportPath = files[0]
	testutil.AssertPathWithin(t, reportPath, reportIO.DocumentsDir)
	testutil.AssertRegularFile(t, reportPath)
	if !strings.HasPrefix(filepath.Base(reportPath), "ghostfolio-capital-gains-2024-fifo-") {
		t.Fatalf("expected FIFO report filename slug, got %q", filepath.Base(reportPath))
	}

	var openerRequests = readOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != 1 || openerRequests[0] != reportPath {
		t.Fatalf("expected one opener request for %q, got %#v", reportPath, openerRequests)
	}

	var rawReport, err = os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read saved report %q: %v", reportPath, err)
	}
	var reportText = string(rawReport)
	for _, expected := range []string{
		"# Ghostfolio Capital Gains And Losses Report",
		"- Year: 2024",
		"- Cost Basis Method: FIFO",
		"- Report Calculation Currency: NOT APPLICABLE",
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
	model, cmd := startReportGenerationFromSelection(t, model)
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
	model, cmd := startReportGenerationFromSelection(t, model)
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

	var rawReport, err = os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read saved report %q: %v", reportPath, err)
	}
	var reportText = string(rawReport)
	if !strings.Contains(reportText, "| sol-buy-2026-asset-tier-001 | BUY | 50 | 4000 | 0.5 | EUR |") {
		t.Fatalf("expected saved report to show the later explicit-currency asset tier, got %q", reportText)
	}
	if strings.Contains(reportText, "| sol-buy-2026-asset-tier-001 | BUY | 50 | 4050 | 1 |") {
		t.Fatalf("expected saved report to skip the currencyless order-tier monetary values, got %q", reportText)
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

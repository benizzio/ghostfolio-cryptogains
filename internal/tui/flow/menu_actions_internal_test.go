// Package flow tests menu-action helpers that keep screen navigation stable
// even when rendered menu indexes change.
// Authored by: OpenCode
package flow

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
)

// TestMenuIndexForActionReturnsExpectedIndex verifies both the matching and
// fallback branches for stable action lookup.
// Authored by: OpenCode
func TestMenuIndexForActionReturnsExpectedIndex(t *testing.T) {
	t.Parallel()

	var actions = []menuActionID{
		syncReportsMenuActionSyncData,
		syncReportsMenuActionGenerateReport,
		syncReportsMenuActionBackToMainMenu,
	}

	if got := menuIndexForAction(actions, syncReportsMenuActionGenerateReport); got != 1 {
		t.Fatalf("expected Generate Report index 1, got %d", got)
	}
	if got := menuIndexForAction(actions, resultMenuActionGenerateDiagnostic); got != 0 {
		t.Fatalf("expected missing action to fall back to index 0, got %d", got)
	}
}

// TestMainMenuActionsAndSelection verifies the stable main-menu action list and
// the selected action helper.
// Authored by: OpenCode
func TestMainMenuActionsAndSelection(t *testing.T) {
	t.Parallel()

	var model = &Model{}
	if got := model.mainMenuActions(); len(got) != 1 || got[0] != mainMenuActionSyncReports {
		t.Fatalf("expected main menu to expose only Sync and Reports, got %#v", got)
	}
	if got := model.selectedMainMenuAction(); got != mainMenuActionSyncReports {
		t.Fatalf("expected selected main-menu action to be Sync and Reports, got %q", got)
	}
}

// TestResultMenuActionsCoversDiagnosticStates verifies sync-result menu
// ordering for busy, pending-diagnostic, and settled outcomes.
// Authored by: OpenCode
func TestResultMenuActionsCoversDiagnosticStates(t *testing.T) {
	t.Parallel()

	var model = &Model{}

	model.result.Busy = true
	if got := model.resultMenuActions(); len(got) != 3 || got[0] != resultMenuActionGenerateDiagnostic || got[1] != resultMenuActionSyncAgain || got[2] != resultMenuActionBackToMainMenu {
		t.Fatalf("expected busy result menu to expose diagnostic, sync again, and back actions, got %#v", got)
	}

	model.result.Busy = false
	model.result.Outcome.Diagnostic = runtime.DiagnosticReportState{Eligible: true}
	if got := model.resultMenuActions(); len(got) != 3 || got[0] != resultMenuActionGenerateDiagnostic || got[1] != resultMenuActionSyncAgain || got[2] != resultMenuActionBackToMainMenu {
		t.Fatalf("expected pending diagnostic result menu to expose diagnostic, sync again, and back actions, got %#v", got)
	}

	model.result.Outcome.Diagnostic.Path = "/tmp/report.diagnostic.json"
	if got := model.resultMenuActions(); len(got) != 2 || got[0] != resultMenuActionSyncAgain || got[1] != resultMenuActionBackToMainMenu {
		t.Fatalf("expected settled result menu to expose sync again and back actions, got %#v", got)
	}
}

// TestReportResultActionsCoversBusyDiagnosticAndRetryOptions verifies the
// report-result action ordering for each supported state combination.
// Authored by: OpenCode
func TestReportResultActionsCoversBusyDiagnosticAndRetryOptions(t *testing.T) {
	t.Parallel()

	var model = &Model{}

	model.report.Busy = true
	if got := model.reportResultActions(); len(got) != 2 || got[0] != reportResultActionGenerateDiagnostic || got[1] != reportResultActionBackToSyncReports {
		t.Fatalf("expected busy report result actions to expose diagnostic and back, got %#v", got)
	}

	model.report.Busy = false
	model.syncReports.ReportResult.Diagnostic = runtime.DiagnosticReportState{Eligible: true}
	if got := model.reportResultActions(); len(got) != 2 || got[0] != reportResultActionGenerateDiagnostic || got[1] != reportResultActionBackToSyncReports {
		t.Fatalf("expected pending report diagnostic actions to expose diagnostic and back, got %#v", got)
	}

	model.syncReports.ReportResult.Diagnostic.Path = "/tmp/report.diagnostic.json"
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024}}
	if got := model.reportResultActions(); len(got) != 2 || got[0] != reportResultActionBackToSyncReports || got[1] != reportResultActionGenerateAnother {
		t.Fatalf("expected readable-snapshot report result actions to expose back and generate another, got %#v", got)
	}

	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: false, AvailableReportYears: []int{2024}}
	if got := model.reportResultActions(); len(got) != 1 || got[0] != reportResultActionBackToSyncReports {
		t.Fatalf("expected unreadable snapshot report result actions to expose only back, got %#v", got)
	}
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true}
	if got := model.reportResultActions(); len(got) != 1 || got[0] != reportResultActionBackToSyncReports {
		t.Fatalf("expected missing-year report result actions to expose only back, got %#v", got)
	}
}

// TestSyncReportsReportAvailabilityHelpers verifies the unlocked-context report
// availability helpers and their user-visible failure reason mapping.
// Authored by: OpenCode
func TestSyncReportsReportAvailabilityHelpers(t *testing.T) {
	t.Parallel()

	var model = &Model{}

	model.syncReports.ReportUnavailable = runtime.ReportFailureNoSyncedDataAvailable
	if !model.reportUnavailable() {
		t.Fatalf("expected report generation to be unavailable without a readable snapshot")
	}
	if got := model.syncReportsReportUnavailableReason(); got != runtime.ReportFailureNoSyncedDataAvailable {
		t.Fatalf("expected unavailable reason to surface stored failure, got %q", got)
	}
	if got := model.syncReportsDefaultMenuAction(); got != syncReportsMenuActionSyncData {
		t.Fatalf("expected no pending diagnostic to keep Sync Data as default action, got %q", got)
	}

	model.syncReports.SyncResult.Outcome.Diagnostic = runtime.DiagnosticReportState{Eligible: true}
	if got := model.syncReportsDefaultMenuAction(); got != syncReportsMenuActionGenerateDiagnostic {
		t.Fatalf("expected pending diagnostic to become the default action, got %q", got)
	}

	model.syncReports.SyncResult.Outcome.Diagnostic.Path = "/tmp/report.diagnostic.json"
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024, 2025}}
	if model.reportUnavailable() {
		t.Fatalf("expected report generation to be available with a readable snapshot and years")
	}
	if got := model.syncReportsReportUnavailableReason(); got != runtime.ReportFailureNone {
		t.Fatalf("expected available report state to suppress unavailable reason, got %q", got)
	}
	if got := model.syncReportsDefaultMenuAction(); got != syncReportsMenuActionSyncData {
		t.Fatalf("expected written diagnostic path to restore Sync Data as default action, got %q", got)
	}
}
